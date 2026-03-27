package auth

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

func GetTokens() error {
	filepath := "./secret/client_secret.json"
	data, err := os.ReadFile(filepath)
	if err != nil {
		return err
	}

	state := rand.Text()
	config, err := google.ConfigFromJSON(data, "https://www.googleapis.com/auth/gmail.readonly")
	if err != nil {
		return err
	}
	url := config.AuthCodeURL(state, oauth2.AccessTypeOffline)

	fmt.Printf("Access this URl to enable GmailAPI: %v\n", url)
	err = getAccessToken(*config)
	if err != nil {
		return err
	}
	return nil
}

func getAccessToken(conf oauth2.Config) error {
	serverMux := http.NewServeMux()
	server := http.Server{Addr: ":8080", Handler: serverMux}
	serverMux.HandleFunc("/", func(respW http.ResponseWriter, req *http.Request) {
		code := req.URL.Query().Get("code")
		if code != "" {
			file, err := os.OpenFile("./secret/token.json", os.O_RDWR|os.O_CREATE, 0600)
			if err != nil {
				http.Error(respW, "Internal error", 500)
				return
			}
			fmt.Fprintf(respW, "Authorized! You can close this tab.")
			tok, err := conf.Exchange(context.Background(), code)
			encoder := json.NewEncoder(file)
			encoder.Encode(tok)
			defer file.Close()
			defer server.Shutdown(req.Context())
		}
	})
	server.ListenAndServe()
	// if err != nil {
	// 	return err
	// }
	return nil
}
