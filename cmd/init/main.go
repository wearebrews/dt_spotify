package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/go-redis/redis"
	"github.com/sirupsen/logrus"
	"github.com/wearebrews/dt_spotify/spotifyhelper"
)

const tokenKey string = "spotify_token"

type SlackMessage struct {
	Text string `json:"text"`
}

func main() {

	//Load envs
	spotifyClientID, ok := os.LookupEnv("SPOTIFY_CLIENT_ID")
	if !ok {
		logrus.Panic("Missing SPOTIFY_CLIENT_ID")
	}
	spotifyClientSecret, ok := os.LookupEnv("SPOTIFY_CLIENT_SECRET")
	if !ok {
		logrus.Panic("Missing SPOTIFY_CLIENT_SECRET")
	}

	baseURL, ok := os.LookupEnv("BASE_URL")
	if !ok {
		logrus.Panic("Missing BASE_URL")
	}

	slackURL, ok := os.LookupEnv("SLACK_URL")
	if !ok {
		logrus.Panic("Missing SLACK URL")
	}
	hostPort, ok := os.LookupEnv("HOST_PORT")
	if !ok {
		hostPort = "8080"
	}
	spotifyClientID = spotifyClientID[:len(spotifyClientID)-1]
	spotifyClientSecret = spotifyClientSecret[:len(spotifyClientSecret)-1]

	client := redis.NewClient(&redis.Options{
		Addr:     "redis-slave:6379",
		Password: "",
		DB:       0,
	})
	pong, err := client.Ping().Result()
	if err != nil {
		logrus.Panic(err)
	}
	logrus.Info("Redis responded to PING: " + pong)

	//Try to get token
	_, err = client.Get(tokenKey).Result()
	if err == nil {
		if client.TTL(tokenKey).Val() > 5*time.Minute {
			//Success, we found a live token
			os.Exit(0)
		}
	}

	loginPath := "/login"
	redirectURL := baseURL + loginPath
	session := spotifyhelper.NewSession(context.TODO(), spotifyClientID, spotifyClientSecret, redirectURL)
	//Set up HTTP handler for login session
	http.HandleFunc(loginPath, session.Handler())
	//Send login url to start authentication
	jsonBytes, err := json.Marshal(SlackMessage{session.LoginURL()})
	if err != nil {
		logrus.Panic(err)
	}
	go func() {
		<-session.Init
		os.Exit(0)
	}()
	http.Post(slackURL, "application/json", bytes.NewBuffer(jsonBytes))
	http.ListenAndServe(":"+hostPort, nil)
}
