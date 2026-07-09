package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"html/template"
	"log"
	"time"

	googlehealthgo "github.com/espcaa/google-health-go"
)

//go:embed template.tmpl
var sleepLogTemplateString string

//go:embed system_prompt.txt
var systemPromptString string

type SleepLogData struct {
	Date                string
	Duration            string
	Efficiency          string
	StartTime           string
	EndTime             string
	MinutesAfterWakeup  string
	MinutesAwake        string
	MinutesAsleep       string
	MinutesToFallAsleep string
	TimeInBed           string
	Stages              SleepStages
	StageDetails        []StageDetail
	History             []HistoryItem
}

type SleepStages struct {
	Deep  string
	Light string
	Rem   string
	Wake  string
}

type StageDetail struct {
	Name            string
	StartTime       string
	EndTime         string
	DurationSeconds int
}

type HistoryItem struct {
	Date       string
	Duration   string
	Efficiency int
}

func FormatSleepLog(data SleepLogData) (string, error) {

	tmpl, err := template.New("sleepLog").Parse(sleepLogTemplateString)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

func GetSystemPrompt() string {
	return systemPromptString
}

func MakeSleepLogData(sleep *googlehealthgo.Sleep, bot Bot) SleepLogData {

	// build the complex stage info

	var stageDetails []StageDetail
	for _, s := range sleep.Stages {
		stageDetails = append(stageDetails, StageDetail{
			Name:            s.Type,
			StartTime:       s.StartTime.Add(time.Duration(s.StartUTCOffset) * time.Second).Format("15:04"),
			EndTime:         s.EndTime.Add(time.Duration(s.EndUTCOffset) * time.Second).Format("15:04"),
			DurationSeconds: int(s.EndTime.Sub(s.StartTime).Seconds()),
		})
	}

	// get history of the last 7 days

	sevenDaysAgo := sleep.Interval.StartTime.Add(-7 * 24 * time.Hour)
	records, err := bot.GetRecordsAfter(sevenDaysAgo.Format("2006-01-02"))

	if err != nil {
		log.Printf("failed to get history records: %v", err)
	}

	var history []HistoryItem
	for _, r := range records {
		history = append(history, HistoryItem{
			Date:       r.Date,
			Duration:   fmt.Sprintf("%d", r.Duration),
			Efficiency: r.Efficiency,
		})
	}

	// calculate efficiency

	efficiency, err := sleep.Summary.CalculateEfficiency()
	if err != nil {
		efficiency = 0
	}

	// build the stages summary
	var deepMinutes, lightMinutes, remMinutes, wakeMinutes int

	for _, s := range sleep.Summary.StagesSummary {
		switch s.Type {
		case "AWAKE":
			wakeMinutes = int(s.Minutes)
		case "LIGHT":
			lightMinutes = int(s.Minutes)
		case "DEEP":
			deepMinutes = int(s.Minutes)
		case "REM":
			remMinutes = int(s.Minutes)
		}
	}

	totalDuration := sleep.Interval.EndTime.Sub(sleep.Interval.StartTime)
	hours := int(totalDuration.Hours())
	minutes := int(totalDuration.Minutes()) % 60

	return SleepLogData{
		Date:                sleep.Interval.StartTime.Format("2006-01-02"),
		Duration:            fmt.Sprintf("%d hours and %d minutes", hours, minutes),
		Efficiency:          fmt.Sprintf("%.0f%%", efficiency),
		StartTime:           sleep.Interval.StartTime.Add(time.Duration(sleep.Interval.StartUTCOffset) * time.Second).Format("15:04"),
		EndTime:             sleep.Interval.EndTime.Add(time.Duration(sleep.Interval.EndUTCOffset) * time.Second).Format("15:04"),
		MinutesAfterWakeup:  MinutesToHoursAndMinutes(int(sleep.Summary.MinutesAfterWakeUp)),
		MinutesAwake:        MinutesToHoursAndMinutes(int(sleep.Summary.MinutesAwake)),
		MinutesAsleep:       MinutesToHoursAndMinutes(int(sleep.Summary.MinutesAsleep)),
		MinutesToFallAsleep: MinutesToHoursAndMinutes(int(sleep.Summary.MinutesToFallAsleep)),
		TimeInBed:           MinutesToHoursAndMinutes(int(sleep.Summary.MinutesInSleepPeriod)),
		Stages: SleepStages{
			Wake:  MinutesToHoursAndMinutes(wakeMinutes),
			Light: MinutesToHoursAndMinutes(lightMinutes),
			Deep:  MinutesToHoursAndMinutes(deepMinutes),
			Rem:   MinutesToHoursAndMinutes(remMinutes),
		},
		StageDetails: stageDetails,
		History:      history,
	}
}

func MinutesToHoursAndMinutes(minutes int) string {
	hours := minutes / 60
	remainingMinutes := minutes % 60
	return fmt.Sprintf("%d hours and %d minutes", hours, remainingMinutes)
}
