package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	googlehealthgo "github.com/espcaa/google-health-go"
	"github.com/slack-go/slack"
)

func (b *Bot) SendMessageToSlack(sleep googlehealthgo.Sleep, roast string) error {
	// prepare the msg

	startTime := sleep.Interval.StartTime.Add(time.Duration(sleep.Interval.StartUTCOffset) * time.Second)
	endTime := sleep.Interval.EndTime.Add(time.Duration(sleep.Interval.EndUTCOffset) * time.Second)

	hours := endTime.Sub(startTime).Hours()
	hoursStr := strconv.FormatFloat(hours, 'f', 2, 64)

	// generate the progress bar
	goal := os.Getenv("GOAL_HOURS")
	goalFloat, err := strconv.ParseFloat(goal, 64)
	if err != nil {
		return fmt.Errorf("failed to parse GOAL_HOURS: %w", err)
	}
	progressBar := ProgressBar(hours, goalFloat)

	blocks := []slack.Block{
		slack.NewSectionBlock(
			slack.NewTextBlockObject(
				"mrkdwn",
				fmt.Sprintf(
					"_I slept from %s -> %s for a total of %s hours!_\n",
					startTime.Format("15:04"),
					endTime.Format("15:04"),
					hoursStr,
				),
				false,
				false,
			),
			nil,
			nil,
		),
		slack.NewSectionBlock(
			slack.NewTextBlockObject(
				"mrkdwn",
				roast,
				false,
				false,
			),
			nil,
			nil,
		),
		slack.NewDividerBlock(),
		slack.NewSectionBlock(
			slack.NewTextBlockObject(
				"mrkdwn",
				fmt.Sprintf("`%s` (%.1fh/%.1fh)", progressBar, hours, goalFloat),
				false,
				false,
			),
			nil,
			nil,
		),
	}

	channelID := os.Getenv("SLACK_CHANNEL_ID")

	_, _, _, err = b.SlackClient.SendMessage(channelID, slack.MsgOptionBlocks(blocks...))

	if err != nil {
		return fmt.Errorf("failed to send message to Slack: %w", err)
	}

	return nil
}
