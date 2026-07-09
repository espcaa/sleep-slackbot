package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var redirectURI = "http://localhost:8080/callback"

func startLogin(config Config) {

	conf := &oauth2.Config{
		ClientID:     config.GoogleClientID,
		ClientSecret: config.GoogleClientSecret,
		RedirectURL:  redirectURI,
		Endpoint:     google.Endpoint,
		Scopes: []string{
			"https://www.googleapis.com/auth/googlehealth.sleep.readonly",
			"https://www.googleapis.com/auth/googlehealth.activity_and_fitness.readonly",
		},
	}

	url := conf.AuthCodeURL("state-string",
		oauth2.AccessTypeOffline, // for the refresh token
		oauth2.ApprovalForce,
	)

	fmt.Printf("open this in url in your browser:\n\n%s\n\n", url)

	http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			w.Write([]byte("Authorization failed!"))
			return
		}

		token, err := conf.Exchange(context.Background(), code)
		if err != nil {
			log.Fatalf("Exchange failed: %v", err)
		}
		fmt.Printf("\n--- {temp} copy & save ---\n")
		fmt.Printf("access token: %s\n", token.AccessToken)
		fmt.Printf("refresh token: %s\n", token.RefreshToken)
		fmt.Printf("expiry: %s\n", token.Expiry)

		w.Write([]byte("<h1>yay it worked :3</h1>"))
	})

	log.Fatal(http.ListenAndServe(":8080", nil))

}
