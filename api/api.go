package api

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"

	"github.com/reviewboard/rb-gateway/api/tokens"
	"github.com/reviewboard/rb-gateway/config"
	"github.com/reviewboard/rb-gateway/repositories"
)

const (
	PrivateTokenHeader = "PRIVATE-TOKEN"
)

type API struct {
	configLock sync.RWMutex
	config     config.Config
	router     *mux.Router
	tokenStore tokens.TokenStore
}

// Return a new router for the API.
func New(cfg config.Config) (*API, error) {
	store, err := tokens.NewStore(cfg.TokenStorePath)
	if err != nil {
		return nil, err
	}

	api := API{
		config:     cfg,
		router:     mux.NewRouter(),
		tokenStore: store,
	}

	api.router.Path("/session").
		Methods("GET", "POST").
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
		{[]string{"GET"}, "/path", http.HandlerFunc(api.getPath)},
	}

	for _, route := range routeTable {
		repoRoutes.Path(route.path).
			Methods(route.methods...).
			Handler(route.handler)
	}

	return &api, nil
}

func (api *API) SetConfig(newConfig config.Config) error {
	api.configLock.Lock()
	defer api.configLock.Unlock()

	if api.config.TokenStorePath != newConfig.TokenStorePath {
		store, err := tokens.NewStore(newConfig.TokenStorePath)
		if err != nil {
			return err
		}

		api.tokenStore = store
	}

	api.config = newConfig
	return nil
}

func (api *API) Shutdown(server *http.Server) error {
	/*
	 * This allows us to give the server a grace period for finishing
	 * in-progress requests before it closes all connections.
	 */
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	server.Shutdown(ctx)
	cancel()

	/*
	 * We have to acquire the lock here because the goroutine returned by
	 * api.Serve() acquires the read portion of the lock.
	 */
	api.configLock.Lock()
	defer api.configLock.Unlock()

	return api.tokenStore.Save()
}

func (api *API) Serve() *http.Server {
	server := http.Server{
		Addr:    fmt.Sprintf(":%d", api.config.Port),
		Handler: loggingMiddleware(api.router),
	}

	go func() {
		api.configLock.RLock()
		defer api.configLock.RUnlock()

		var err error
		if api.config.UseTLS {
			err = server.ListenAndServeTLS(api.config.SSLCertificate, api.config.SSLKey)
		} else {
			err = server.ListenAndServe()
		}

		if err != http.ErrServerClosed {
			log.Fatal("ListenAndServe:", err)
		}
	}()

	return &server
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
		if api.tokenStore.Get(r) == nil {
			http.Error(w, "Authorization failed.", http.StatusUnauthorized)
		} else {
			next.ServeHTTP(w, r)
		}
	})
}

// Serve a request.
//
// This is only meant for unit tests since it acquires a lock for every request
// that would normally only be acquired once in `api.Serve`
func (api *API) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	api.configLock.RLock()
	defer api.configLock.RUnlock()

	loggingMiddleware(api.router).ServeHTTP(w, r)
}

func (api *API) validateCredentials(username, password string) bool {
	validUsername := subtle.ConstantTimeCompare([]byte(username), []byte(api.config.Username))
	validPassword := subtle.ConstantTimeCompare([]byte(password), []byte(api.config.Password))

	return validUsername+validPassword == 2
}

func (api *API) CreateSession(r *http.Request) (*Session, error) {
	username, password, ok := r.BasicAuth()
	if !ok {
		return nil, errors.New("Invalid Authorization header.")
	}

	if !api.validateCredentials(username, password) {
		return nil, errors.New("Authorization failed.")
	}

	token, err := api.tokenStore.New()
	if err != nil {
		return nil, err
	}
	return &Session{PrivateToken: *token}, nil
}
