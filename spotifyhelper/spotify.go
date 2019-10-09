package spotifyhelper

import (
	"context"
	"fmt"
	"net/http"

	"github.com/heptiolabs/healthcheck"
	"github.com/rs/xid"
	"github.com/sirupsen/logrus"
	"github.com/zmb3/spotify"
)

type Session struct {
	auth       spotify.Authenticator
	url        string
	handler    http.HandlerFunc
	clientChan <-chan *spotify.Client
}

func NewSession(ctx context.Context, clientID, secretKey, redirectURL string) *Session {
	auth := spotify.NewAuthenticator(redirectURL, spotify.ScopeUserReadCurrentlyPlaying, spotify.ScopeUserReadPlaybackState, spotify.ScopeUserModifyPlaybackState)
	auth.SetAuthInfo(clientID, secretKey)

	sessionID := xid.New().String()
	url := auth.AuthURL(sessionID)
	c := make(chan *spotify.Client)

	handler := func(w http.ResponseWriter, r *http.Request) {
		token, err := auth.Token(sessionID, r)
		if err != nil {
			http.Error(w, "Was not able to get token", http.StatusNotFound)
		}
		client := auth.NewClient(token)
		client.AutoRetry = true
		logrus.Info("New client token received %s", token.AccessToken)
		//Send token
		c <- &client
	}
	return &Session{
		auth:       auth,
		url:        url,
		handler:    handler,
		clientChan: c,
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
	cmd    chan int
	health healthcheck.Handler
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

func New(ctx context.Context, s *Session, health healthcheck.Handler) Controller {
	temp := Controller{
		cmd:    make(chan int),
		health: health,
	}

	go run(ctx, s.clientChan, temp)
	return temp
}

func run(ctx context.Context, clientChan <-chan *spotify.Client, c Controller) {
	var client *spotify.Client
	c.health.AddReadinessCheck("spotify client ready", func() error {
		if client == nil {
			return fmt.Errorf("No spotify client received")
		}
		return nil
	})
	client = <-clientChan
	for {
		select {
		case <-ctx.Done():
			return
		case client = <-clientChan:
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
		}
	}
}

func (s *Session) Handler() http.HandlerFunc {
	return s.handler
}

func (s *Session) LoginURL() string {
	return s.url
}
