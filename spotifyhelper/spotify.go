package spotifyhelper

import (
	"context"
	"net/http"

	"golang.org/x/oauth2"

	"github.com/heptiolabs/healthcheck"
	"github.com/rs/xid"
	"github.com/sirupsen/logrus"
	"github.com/zmb3/spotify"
)

type Session struct {
	auth      spotify.Authenticator
	url       string
	handler   http.HandlerFunc
	tokenChan chan oauth2.Token
	Init      <-chan struct{}
}

func NewSession(ctx context.Context, clientID, secretKey, redirectURL string) *Session {
	auth := spotify.NewAuthenticator(redirectURL, spotify.ScopeUserReadCurrentlyPlaying, spotify.ScopeUserReadPlaybackState, spotify.ScopeUserModifyPlaybackState)
	auth.SetAuthInfo(clientID, secretKey)

	sessionID := xid.New().String()
	url := auth.AuthURL(sessionID)
	tokenChan := make(chan oauth2.Token)
	init := make(chan struct{})

	handler := func(w http.ResponseWriter, r *http.Request) {
		token, err := auth.Token(sessionID, r)
		if err != nil {
			http.Error(w, "Was not able to get token", http.StatusNotFound)
			return
		}
		//Send token if chan available
		tokenChan <- *token
	}
	return &Session{
		auth:      auth,
		url:       url,
		handler:   handler,
		tokenChan: tokenChan,
		Init:      init,
	}
}

func (s *Session) SetToken(token oauth2.Token) {
	s.tokenChan <- token
}

const (
	playCMD = iota
	pauseCMD
	toggleCMD
	nextSongCMD
	prevSongCMD
)

type Controller struct {
	cmd          chan int
	song         chan string
	playlist     chan string
	health       healthcheck.Handler
	Ready        chan struct{}
	CurrentToken chan oauth2.Token
}

func (c Controller) Play() {
	c.cmd <- playCMD
}
func (c Controller) Pause() {
	c.cmd <- pauseCMD
}
func (c Controller) Toggle() {
	c.cmd <- toggleCMD
}
func (c Controller) NextSong() {
	c.cmd <- nextSongCMD
}

func (c Controller) PrevSong() {
	c.cmd <- prevSongCMD
}
func (c Controller) PlaySong(song string) {
	c.song <- song
}

func (c Controller) PlayPlaylist(playlist string) {
	c.playlist <- playlist
}

func New(ctx context.Context, s *Session, health healthcheck.Handler) Controller {
	temp := Controller{
		cmd:          make(chan int),
		song:         make(chan string),
		playlist:     make(chan string),
		Ready:        make(chan struct{}),
		CurrentToken: make(chan oauth2.Token),
		health:       health,
	}

	go run(ctx, s, temp)
	return temp
}

func run(ctx context.Context, session *Session, c Controller) {

	initToken := <-session.tokenChan
	client := session.auth.NewClient(&initToken)
	logrus.Info("Initial token received, starting spotifyhelper")
	//Signal ready
	close(c.Ready)
	for {
		token, err := client.Token()
		if err != nil {
			logrus.Panic(err)
		}

		select {
		case <-ctx.Done():
			return
		case token := <-session.tokenChan:
			client = session.auth.NewClient(&token)
		case c.CurrentToken <- *token:
			//Send token when ready. NB: Might be OLD (unlikely)!
		case cmd := <-c.cmd:
			switch cmd {
			case playCMD:
				if err := client.Play(); err != nil {
					logrus.Info(err)
				}
			case pauseCMD:
				if err := client.Pause(); err != nil {
					logrus.Info(err)
				}
			case toggleCMD:
				state, err := client.PlayerState()
				if err != nil {
					logrus.Panic(err)
				}
				if state.Playing {
					if err := client.Pause(); err != nil {
						logrus.Panic(err)
					}
				} else {
					client.Play()
				}
			case nextSongCMD:
				client.Next()
			case prevSongCMD:
				client.Previous()
			}
		case song := <-c.song:
			if err := client.PlayOpt(&spotify.PlayOptions{URIs: []spotify.URI{spotify.URI(song)}}); err != nil {
				logrus.Panic(err)
			}
		case playlist := <-c.playlist:
			playlistURI := spotify.URI(playlist)
			if err := client.PlayOpt(&spotify.PlayOptions{PlaybackContext: &playlistURI}); err != nil {
				logrus.Panic(err)
			}
		}
	}
}

func (s *Session) Handler() http.HandlerFunc {
	return s.handler
}

func (s *Session) LoginURL() string {
	return s.url
}
