package spotifyhelper

import (
	"context"
	"net/http"

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

func (s *Session) Handler() http.HandlerFunc {
	return s.handler
}

func (s *Session) LoginURL() string {
	return s.url
}

func (s *Session) Client() <-chan *spotify.Client {
	return s.clientChan
}
