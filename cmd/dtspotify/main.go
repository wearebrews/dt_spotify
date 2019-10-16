package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"time"

	"io/ioutil"

	"github.com/go-redis/redis"
	"github.com/heptiolabs/healthcheck"
	"github.com/sirupsen/logrus"
	"github.com/wearebrews/dt_spotify/spotifyhelper"
	"golang.org/x/oauth2"
)

type SlackMessage struct {
	Text string `json:"text"`
}

type DTEvent struct {
	Event struct {
		EventID    string `json:"eventId"`
		TargetName string `json:"targetName"`
		EventType  string `json:"eventType"`
		Data       struct {
			ObjectPresent *struct {
				State string `json:"state"`
			} `json:"objectPresent"`
			Temperature *struct {
				Value float32 `json:"value"`
			} `json:"temperature"`
		} `json:"data"`
	} `json:"event"`
	Labels struct {
		Action   *string `json:"spotify_action"`
		Song     *string `json:"spotify_song"`
		Playlist *string `json:"spotify_playlist"`
	} `json:"labels"`
}

var dtEvents chan DTEvent

func postLoginURL(slackURL, loginURL string) {
	//Ask user to login
	jsonBytes, err := json.Marshal(SlackMessage{loginURL})
	if err != nil {
		logrus.Panic(err)
	}
	http.Post(slackURL, "application/json", bytes.NewBuffer(jsonBytes))
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
	hostPort, ok := os.LookupEnv("HOST_PORT")
	if !ok {
		hostPort = "8080"
	}
	slackURL, ok := os.LookupEnv("SLACK_URL")
	if !ok {
		logrus.Panic("Missing SLACK URL")
	}

	health := healthcheck.NewHandler()
	logrus.Info(slackURL)

	loginPath := "/login"
	redirectURL := baseURL + loginPath
	//Create new session
	session := spotifyhelper.NewSession(context.TODO(), spotifyClientID, spotifyClientSecret, redirectURL)
	spotify := spotifyhelper.New(context.TODO(), session, health)
	//Set up HTTP handler for login session
	http.HandleFunc(loginPath, session.Handler())
	//Send login url to start authentication

	//Create DT handler
	http.HandleFunc("/dtconn", handleDTEvents)

	redisClient := redis.NewClient(&redis.Options{
		Addr:     "redis-master:6379",
		Password: "",
		DB:       0,
	})

	var token oauth2.Token
	rBytes, err := redisClient.Get("spotify_token").Bytes()
	if err == nil {
		//Load token from redis
		err := json.Unmarshal(rBytes, &token)
		if err != nil {
			logrus.Panic(err)
		}
		session.SetToken(token)
	} else {
		postLoginURL(slackURL, session.LoginURL())
	}

	//Refresh token frequently
	go writeSpotifyTokenPeriodically(redisClient, spotify.CurrentToken, 5*time.Minute)

	//Create channel for DT events
	dtEvents = make(chan DTEvent)

	//Start
	ctx := context.Background()
	go run(ctx, dtEvents, spotify)
	go http.ListenAndServe("0.0.0.0:8086", health)
	http.ListenAndServe(":"+hostPort, nil)
}

func writeSpotifyTokenPeriodically(client *redis.Client, tokenChan <-chan oauth2.Token, period time.Duration) {
	for {
		token := <-tokenChan
		logrus.WithField("token", token).Info("Refreshing token in redis")
		bytes, err := json.Marshal(token)
		if err != nil {
			logrus.Panic(err)
		}
		if err := client.Set("spotify_token", bytes, token.Expiry.Sub(time.Now())).Err(); err != nil {
			logrus.Panic(err)
		}
		<-time.After(period)
	}
}

func handleDTEvents(w http.ResponseWriter, r *http.Request) {
	if dtEvents == nil {
		http.Error(w, "Application not ready", http.StatusInternalServerError)
		return
	}
	bytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		logrus.Panic(err)
	}
	event := &DTEvent{}
	err = json.Unmarshal(bytes, event)
	if err != nil {
		logrus.Panic(err)
	}
	dtEvents <- *event
}

func run(ctx context.Context, dtEvents chan DTEvent, c spotifyhelper.Controller) {
	//Wait until spotify is READY
	select {
	case <-c.Ready:
		break
	case <-time.After(5 * time.Minute):
		logrus.Panic("Spotify is not ready after 5 minutes")
	}
	logrus.Info("Application is ready for events")
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-dtEvents:
			if event.Labels.Action != nil {
				logrus.WithField("event", event).Info("Processing event")
				switch *event.Labels.Action {
				case "play":
					c.Play()
				case "pause":
					c.Pause()
				case "toggle":
					c.Toggle()
				case "next_song":
					c.NextSong()
				case "prev_song":
					c.PrevSong()
				case "play_song":
					if event.Labels.Song != nil {
						c.PlaySong(*event.Labels.Song)
					}
				case "play_playlist":
					if event.Labels.Playlist != nil {
						c.PlayPlaylist(*event.Labels.Playlist)
					}
				}

			}

		}
	}
}
