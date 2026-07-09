package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

func main() {

	// create the config

	godotenv.Load()

	clientID := os.Getenv("CLIENT_ID")
	clientSecret := os.Getenv("CLIENT_SECRET")
	refreshToken := os.Getenv("REFRESH_TOKEN")

	config := Config{
		GoogleClientID:           clientID,
		GoogleClientSecret:       clientSecret,
		GoogleHealthRefreshToken: refreshToken,
	}

	// get command line args to launch submodules
	args := os.Args[1:]
	if len(args) > 0 {
		switch args[0] {
		case "login":
			// handle login
			log.Println("[login]")
			startLogin(config)
		}
	} else {
		log.Println("[bot]")
		bot, err := NewBot(config)
		if err != nil {
			log.Fatal(err)
		}
		bot.Start()
	}
}
