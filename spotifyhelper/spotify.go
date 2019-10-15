package spotifyhelper

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"golang.org/x/oauth2"

	"github.com/go-redis/redis"
	"github.com/heptiolabs/healthcheck"
	"github.com/rs/xid"
	"github.com/sirupsen/logrus"
	"github.com/zmb3/spotify"
)

type Session struct {
	auth      spotify.Authenticator
	url       string
	handler   http.HandlerFunc
	UserToken <-chan oauth2.Token
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
		}
		client := redis.NewClient(&redis.Options{
			Addr:     "redis-master:6379",
			Password: "",
			DB:       0,
		})

		var bytes []byte
		bytes, err = json.Marshal(token)
		if err != nil {
			logrus.Panic(err)
		}
		_, err = client.Set("spotify_token", bytes, token.Expiry.Sub(time.Now())).Result()
		if err != nil {
			logrus.Panic(err)
		}
		//Send token if chan availabl
		select {
		case tokenChan <- *token:
		default:
		}
	}
	return &Session{
		auth:      auth,
		url:       url,
		handler:   handler,
		UserToken: tokenChan,
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

type Controller struct {
	cmd      chan int
	song     chan string
	playlist chan string
	health   healthcheck.Handler
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

func New(ctx context.Context, token *oauth2.Token, s *Session, health healthcheck.Handler) Controller {
	temp := Controller{
		cmd:      make(chan int),
		song:     make(chan string),
		playlist: make(chan string),
		health:   health,
	}

	go run(ctx, token, s.auth, temp)
	return temp
}

func run(ctx context.Context, token *oauth2.Token, auth spotify.Authenticator, c Controller) {
	client := auth.NewClient(token)
	for {
		select {
		case <-ctx.Done():
			return
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
