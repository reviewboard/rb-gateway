package api

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/mux"

	"github.com/reviewboard/rb-gateway/config"
	"github.com/reviewboard/rb-gateway/repositories"
)

const (
	PrivateTokenHeader = "PRIVATE-TOKEN"
)

type API struct {
	config config.Config
	router *mux.Router
}

// Return a new router for the API.
func New(cfg config.Config) API {
	api := API{
		config: cfg,
		router: mux.NewRouter(),
	}

	api.router.Path("/session").
		Methods("GET").
		HandlerFunc(api.getSession)

	// The following routes all require authorization.
	repoRoutes := api.router.PathPrefix("/repos/{repo}").Subrouter()
	repoRoutes.Use(api.withAuthorizationRequired)
	repoRoutes.Use(api.withRepository)

	routeTable := []struct {
		methods []string
		path    string
		handler http.Handler
	}{
		{[]string{"GET"}, "/branches", http.HandlerFunc(api.getBranches)},
		{[]string{"GET"}, "/branches/{branch}/commits", http.HandlerFunc(api.getCommits)},
		{[]string{"GET"}, "/commits/{commit-id}", http.HandlerFunc(api.getCommit)},
		{[]string{"GET"}, "/commits/{commit-id}/path/{path}", http.HandlerFunc(api.getFileByCommit)},
		{[]string{"HEAD"}, "/commits/{commit-id}/path/{path}", http.HandlerFunc(api.getFileExistsByCommit)},
		{[]string{"GET"}, "/file/{file-id}", http.HandlerFunc(api.getFile)},
		{[]string{"HEAD"}, "/file/{file-id}", http.HandlerFunc(api.getFileExists)},
	}

	for _, route := range routeTable {
		repoRoutes.Path(route.path).
			Methods(route.methods...).
			Handler(route.handler)
	}

	return api
}

func (api *API) SetConfig(newConfig config.Config) {
	api.config = newConfig
}

// A middleware for wrapping routes that require a repository.
//
// If the requested repository exists, it will be provided through the context
// as `"repo"`. Otherwise, an appropriate error will be returned.
func (api API) withRepository(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		repoName := mux.Vars(r)["repo"]

		var repo repositories.Repository
		var exists bool

		if len(repoName) == 0 {
			http.Error(w, "Repository not provided.", http.StatusBadRequest)
		} else if repo, exists = api.config.Repositories[repoName]; !exists {
			http.Error(w, "Repository not found.", http.StatusNotFound)
		} else {
			ctx := context.WithValue(r.Context(), "repo", repo)
			next.ServeHTTP(w, r.WithContext(ctx))
		}
	})
}

// TODO: Replace this with actual token logic.
func (api API) withAuthorizationRequired(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		privateToken := r.Header.Get(PrivateTokenHeader)

		var payload []byte
		var err error
		var pair []string

		if len(privateToken) == 0 {
			http.Error(w, "Bad private token.", http.StatusBadRequest)
		} else if payload, err = base64.StdEncoding.DecodeString(privateToken); err != nil {
			http.Error(w, "Bad private token.", http.StatusBadRequest)
		} else if pair = strings.SplitN(string(payload), ":", 2); len(pair) != 2 {
			http.Error(w, "Bad private token.", http.StatusBadRequest)
		} else if !api.validateCredentials(pair[0], pair[1]) {
			http.Error(w, "Authorization failed.", http.StatusUnauthorized)
		} else {
			next.ServeHTTP(w, r)
		}
	})
}

// Serve the HTTP request.
func (api API) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	loggingMiddleware(api.router).ServeHTTP(w, r)
}

func (api *API) validateCredentials(username, password string) bool {
	return username == api.config.Username && password == api.config.Password
}

func (api *API) CreateSession(r *http.Request) (*Session, error) {
	username, password, ok := r.BasicAuth()
	if !ok {
		return nil, errors.New("Invalid Authorization header.")
	}

	if !api.validateCredentials(username, password) {
		return nil, errors.New("Authorization failed.")
	}

	// TODO: replace with an actual token.
	credentials := []byte(fmt.Sprintf("%s:%s", username, password))
	token := base64.StdEncoding.EncodeToString(credentials)
	return &Session{token}, nil
}
