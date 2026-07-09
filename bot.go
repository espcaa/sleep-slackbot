package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"time"

	_ "modernc.org/sqlite"

	googlehealthgo "github.com/espcaa/google-health-go"
	"github.com/robfig/cron/v3"
	"github.com/slack-go/slack"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type Bot struct {
	Config        Config
	Db            *sql.DB
	GHealthClient *googlehealthgo.Client
	AiClient      *AiClient
	SlackClient   *slack.Client
}

func NewBot(config Config) (*Bot, error) {

	db, err := sql.Open("sqlite", "./data.db")
	if err != nil {
		return nil, err
	}

	// create the table with the data point names + simplified sleep log data
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS sleep_logs_sent (
			data_point_name TEXT PRIMARY KEY,
			date TEXT,
			efficiency INTEGER,
			duration INTEGER
		);`)
	if err != nil {
		return nil, err
	}

	return &Bot{
		Config:        config,
		Db:            db,
		GHealthClient: nil,
		AiClient:      nil,
		SlackClient:   nil,
	}, nil
}

func (b *Bot) Start() {
	log.Println("sleepbot started!")

	// authenticate with ghealth api

	httpClient := GetAuthenticatedHttpClient(b.Config)
	gHealthClient := googlehealthgo.NewClient(httpClient)
	b.GHealthClient = gHealthClient

	// get env files for the ai client
	var aiApiKey string
	var aiModel string
	var aiBaseUrl string

	aiApiKey = os.Getenv("AI_API_KEY")
	aiModel = os.Getenv("AI_MODEL")
	aiBaseUrl = os.Getenv("AI_BASE_URL")

	b.AiClient = NewAiClient(aiBaseUrl, aiApiKey, aiModel)

	// slack client

	slackToken := os.Getenv("SLACK_BOT_TOKEN")
	b.SlackClient = slack.New(slackToken)

	// start the loop
	c := cron.New()

	// start a first tick
	b.Tick()

	c.AddFunc("*/10 * * * *", func() {
		b.Tick()
	})

	c.AddFunc("0 0 1 * *", func() {
		// clear any item with a date older than 30 days
		_, err := b.Db.Exec("DELETE FROM sleep_logs_sent WHERE date < date('now', '-30 days')")
		if err != nil {
			log.Printf("failed to clear old sleep logs: %v", err)
		}
	})

	c.Run()
	defer c.Stop()
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
		StartTime: time.Now().Add(-48 * time.Hour).UTC(),
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
		log.Printf("found sleep source with id: %v\n", pt.Name)

		// check if it has been sent already
		var exists bool
		err := b.Db.QueryRow("SELECT EXISTS(SELECT 1 FROM sleep_logs_sent WHERE data_point_name = ?)", pt.Name).Scan(&exists)
		if err != nil {
			log.Printf("failed to check if sleep log has been sent: %v", err)
			continue
		}

		if exists {
			log.Printf("sleep log with id %v has already been sent. skipping.", pt.Name)
			continue
		}

		// prepare the ai roast

		sleepLog := MakeSleepLogData(pt.Sleep, *b)
		sleepLogString, err := FormatSleepLog(sleepLog)
		if err != nil {
			sleepLogString = "failed to format sleep log"
		}

		systemPrompt := GetSystemPrompt()

		messages := []AiMessage{
			{
				Role:    "system",
				Content: systemPrompt,
			},
			{
				Role:    "system",
				Content: sleepLogString,
			},
		}

		roast, err := b.AiClient.Complete(messages)

		if err != nil {
			log.Printf("failed to get ai roast: %v", err)
			continue
		}

		roastContent := roast.GetContent()

		log.Printf("roast for sleep log %v: %v", pt.Name, roastContent)

		// send the roast to slack

		err = b.SendMessageToSlack(*pt.Sleep, roastContent)
		if err != nil {
			log.Printf("failed to send message to slack: %v", err)
			continue
		}

		log.Printf("sent roast for sleep log %v to slack", pt.Name)

		efficiency, err := pt.Sleep.Summary.CalculateEfficiency()

		if err != nil {
			log.Printf("failed to calculate efficiency for sleep log %v: %v", pt.Name, err)
			efficiency = 0
		}

		duration := pt.Sleep.Interval.EndTime.Sub(pt.Sleep.Interval.StartTime).Minutes()

		// save the record in the database
		record := DbSleepRecord{
			DataPointName: pt.Name,
			Date:          pt.Sleep.Interval.StartTime.Format("2006-01-02"),
			Efficiency:    int(efficiency),
			Duration:      int(duration),
		}

		err = b.SaveMinimalSleepLog(record)
		if err != nil {
			log.Printf("failed to save sleep log record: %v", err)
			continue
		}

		log.Printf("saved sleep log record for %v", pt.Name)
	}
}
