package main

import "strings"

func ProgressBar(filled float64, max float64) string {

	percentage := filled / max
	barLength := 10
	filledLength := int(percentage * float64(barLength))

	return strings.Repeat("█", filledLength) + strings.Repeat("░", barLength-filledLength)
}
