package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/wearebrews/dt_spotify/spotifyhelper"
	"io/ioutil"
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

	logrus.Info(slackURL)

	spotifyClientID = spotifyClientID[:len(spotifyClientID)-1]
	spotifyClientSecret = spotifyClientSecret[:len(spotifyClientSecret)-1]
	logrus.Info(spotifyClientID)
	logrus.Info(spotifyClientSecret)

	dtEvents = make(chan DTEvent)

	loginPath := "/login"
	redirectURL := baseURL + loginPath
	//Create new session
	session := spotifyhelper.NewSession(context.TODO(), spotifyClientID, spotifyClientSecret, redirectURL)
	//Set up HTTP handler for login session
	http.HandleFunc(loginPath, session.Handler())
	//Send login url to start authentication
	jsonBytes, err := json.Marshal(SlackMessage{session.LoginURL()})
	if err != nil {
		logrus.Panic(err)
	}
	http.Post(slackURL, "application/json", bytes.NewBuffer(jsonBytes))

	//Create DT handler
	http.HandleFunc("/dtconn", handleDTEvents)

	spotify := spotifyhelper.New(context.TODO(), session)

	//Start
	ctx := context.Background()
	go run(ctx, dtEvents, spotify)
	http.ListenAndServe(":"+hostPort, nil)
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
	logrus.Info("Application is READY!")
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-dtEvents:
			if event.Labels.Action != nil {
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
				}
			}

		}
	}
}
