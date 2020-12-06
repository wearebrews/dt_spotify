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

// Session contains a spotify session with oauth handler
type Session struct {
	auth      spotify.Authenticator
	url       string
	handler   http.HandlerFunc
	tokenChan chan oauth2.Token
	Init      <-chan struct{}
}

// Handler returns http handler for oauth tokens
func (s *Session) Handler() http.HandlerFunc {
	return s.handler
}

// LoginURL returns a login url for spotify user login
func (s *Session) LoginURL() string {
	return s.url
}

// SetToken allows an external token to be loaded
func (s *Session) SetToken(token oauth2.Token) {
	s.tokenChan <- token
}

// NewSession creates a new session
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

const (
	playCMD = iota
	pauseCMD
	toggleCMD
	nextSongCMD
	prevSongCMD
)

// Controller contains valid spotify actions
type Controller interface {
	Play()
	Pause()
	Toggle()
	NextSong()
	PrevSong()
	PlaySong(song string)
	PlayPlaylist(playlist string)
	Token() oauth2.Token
	Ready() <-chan struct{}
}

type controller struct {
	cmd          chan int
	song         chan string
	playlist     chan string
	health       healthcheck.Handler
	ready        chan struct{}
	currentToken chan oauth2.Token
}

func (c controller) Token() oauth2.Token {
	return <-c.currentToken
}

func (c controller) Play() {
	c.cmd <- playCMD
}
func (c controller) Pause() {
	c.cmd <- pauseCMD
}
func (c controller) Toggle() {
	c.cmd <- toggleCMD
}
func (c controller) NextSong() {
	c.cmd <- nextSongCMD
}

func (c controller) PrevSong() {
	c.cmd <- prevSongCMD
}
func (c controller) PlaySong(song string) {
	c.song <- song
}

func (c controller) PlayPlaylist(playlist string) {
	c.playlist <- playlist
}

func (c controller) Ready() <-chan struct{} {
	return c.ready
}

// New creates a new spotify controller from a session
func New(ctx context.Context, s *Session, health healthcheck.Handler) Controller {
	temp := controller{
		cmd:          make(chan int),
		song:         make(chan string),
		playlist:     make(chan string),
		ready:        make(chan struct{}),
		currentToken: make(chan oauth2.Token),
		health:       health,
	}

	go run(ctx, s, temp)
	return temp
}

func run(ctx context.Context, session *Session, c controller) {

	initToken := <-session.tokenChan
	client := session.auth.NewClient(&initToken)
	logrus.Info("Initial token received, starting spotifyhelper")
	//Signal ready
	close(c.ready)
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
		case c.currentToken <- *token:
			//Send token when ready. NB: Might be OLD (unlikely)!
		case cmd := <-c.cmd:
			switch cmd {
			case playCMD:
				if err := client.Play(); err != nil {
					logrus.Error(err)
				}
			case pauseCMD:
				if err := client.Pause(); err != nil {
					logrus.Error(err)
				}
			case toggleCMD:
				state, err := client.PlayerState()
				if err != nil {
					logrus.Error(err)
					continue
				}
				if state.Playing {
					if err := client.Pause(); err != nil {
						logrus.Error(err)
					}
				} else {
					if err := client.Play(); err != nil {
						logrus.Error(err)
					}

				}
			case nextSongCMD:
				if err := client.Next(); err != nil {
					logrus.Error(err)
				}
			case prevSongCMD:
				client.Previous()
			}
		case song := <-c.song:
			if err := client.PlayOpt(&spotify.PlayOptions{URIs: []spotify.URI{spotify.URI(song)}}); err != nil {
				logrus.Error(err)
			}
		case playlist := <-c.playlist:
			playlistURI := spotify.URI(playlist)
			if err := client.PlayOpt(&spotify.PlayOptions{PlaybackContext: &playlistURI}); err != nil {
				logrus.Error(err)
			}
		}
	}
}
