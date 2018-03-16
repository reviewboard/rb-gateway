package main

import (
	"encoding/base64"
	"errors"
	"net/http"
	"strings"

	"github.com/reviewboard/rb-gateway/config"
)

const (
	authPrivateToken = "PRIVATE-TOKEN"
)

type handler func(w http.ResponseWriter, r *http.Request)

// BasicAuth relays all routes that require authentication checking.
// A PRIVATE-TOKEN value is expected to be in the request header, corresponding
// to the encrypted username, password combination. If the PRIVATE-TOKEN is
// valid, this function will pass the request to the appropriate handler,
// otherwise it will raise an appropriate HTTP error.
func BasicAuth(pass handler) handler {
	return func(w http.ResponseWriter, r *http.Request) {
		privateToken := r.Header.Get(authPrivateToken)

		if privateToken == "" {
			http.Error(w, "Bad private token", http.StatusBadRequest)
			return
		}

		payload, _ := base64.StdEncoding.DecodeString(privateToken)
		pair := strings.SplitN(string(payload), ":", 2)

		if len(pair) != 2 || !validate(pair[0], pair[1]) {
			http.Error(w, "Authorization failed", http.StatusUnauthorized)
			return
		}

		pass(w, r)
	}
}

// CreateSession creates a session from the Basic Authentication information
// expected in the request. The session is constructed using the encrypted
// username, password combination.
// It returns a session and an error if Basic Authentication fails.
func CreateSession(r *http.Request) (Session, error) {
	authHeader := r.Header["Authorization"]
	if authHeader == nil {
		return Session{}, errors.New("Bad authorization syntax")
	}

	auth := strings.SplitN(authHeader[0], " ", 2)

	if len(auth) != 2 || auth[0] != "Basic" {
		return Session{}, errors.New("Bad authorization syntax")
	}

	payload, _ := base64.StdEncoding.DecodeString(auth[1])
	pair := strings.SplitN(string(payload), ":", 2)

	if len(pair) != 2 || !validate(pair[0], pair[1]) {
		return Session{}, errors.New("Authorization failed")
	}

	return Session{string(auth[1])}, nil
}

// validate against the single user stored in the config file.
// It returns true if the username and password matches what is in config.json.
func validate(username, password string) bool {
	if username == config.GetUsername() && password == config.GetPassword() {
		return true
	}
	return false
}
