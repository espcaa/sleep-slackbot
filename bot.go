package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"time"

	_ "modernc.org/sqlite"

	googlehealthgo "github.com/espcaa/google-health-go"
	"github.com/robfig/cron/v3"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type Bot struct {
	Config        Config
	Db            *sql.DB
	GHealthClient *googlehealthgo.Client
}

func NewBot(config Config) (*Bot, error) {

	db, err := sql.Open("sqlite", "./data.db")
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS sleep_logs_sent (
			data_point_name TEXT PRIMARY KEY
		);`)
	if err != nil {
		return nil, err
	}

	return &Bot{
		Config:        config,
		Db:            db,
		GHealthClient: nil,
	}, nil
}

func (b *Bot) Start() {
	log.Println("sleepbot started!")

	// authenticate with ghealth api

	httpClient := GetAuthenticatedHttpClient(b.Config)
	gHealthClient := googlehealthgo.NewClient(httpClient)
	b.GHealthClient = gHealthClient

	// start the loop
	c := cron.New()

	c.AddFunc("0 * * * *", func() {
		b.Tick()
	})

	c.Run()
	defer c.Stop()

	// res, err := gHealthClient.Reconcile(googlehealthgo.DataTypeSleep, googlehealthgo.QueryOptions{
	// 	StartTime: time.Now().Add(-48 * time.Hour).UTC(),
	// 	EndTime:   time.Now().UTC(),
	// })

	// if err != nil {
	// 	log.Fatalf("Failed to fetch sleep data: %v", err)
	// }

	// for _, pt := range res.DataPoints {
	// 	log.Printf("Sleep chunk source: %v\n", pt.Sleep)
	// }
}

func GetAuthenticatedHttpClient(config Config) *http.Client {
	ctx := context.Background()

	oauthConfig := &oauth2.Config{
		ClientID:     config.GoogleClientID,
		ClientSecret: config.GoogleClientSecret,
		Endpoint:     google.Endpoint,
	}

	token := &oauth2.Token{
		RefreshToken: config.GoogleHealthRefreshToken,
	}

	httpClient := oauthConfig.Client(ctx, token)

	return httpClient
}

func (b *Bot) Tick() {
	// get the sleep info of the last 24 hours
	log.Println("tick!")

	res, err := b.GHealthClient.Reconcile(googlehealthgo.DataTypeSleep, googlehealthgo.QueryOptions{
		StartTime: time.Now().Add(-24 * time.Hour).UTC(),
		EndTime:   time.Now().UTC(),
	})

	if err != nil {
		log.Printf("Failed to fetch sleep data: %v", err)
		return
	}

	if len(res.DataPoints) == 0 {
		log.Println("No sleep data found in the last 24 hours.")
		return
	}

	for _, pt := range res.DataPoints {
		log.Printf("found sleep source with id: %v\n", pt.DataSource)

		// check if it has been sent already
		var exists bool
		err := b.Db.QueryRow("SELECT EXISTS(SELECT 1 FROM sleep_logs_sent WHERE data_point_name = ?)", pt.DataSource).Scan(&exists)
		if err != nil {
			log.Printf("failed to check if sleep log has been sent: %v", err)
			continue
		}

		if exists {
			log.Printf("sleep log with id %v has already been sent. skipping.", pt.DataSource)
			continue
		}

		// else send the sleep log to slack!
	}
}
