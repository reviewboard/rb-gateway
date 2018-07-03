package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	auth "github.com/abbot/go-http-auth"
	"github.com/gorilla/mux"

	"github.com/reviewboard/rb-gateway/api/tokens"
	"github.com/reviewboard/rb-gateway/config"
	"github.com/reviewboard/rb-gateway/repositories"
	"github.com/reviewboard/rb-gateway/repositories/hooks"
)

const (
	PrivateTokenHeader = "PRIVATE-TOKEN"
)

type routingEntry struct {
	methods []string
	path    string
	handler http.Handler
}

func addRoutes(router *mux.Router, routes []routingEntry) {
	for _, route := range routes {
		router.Path(route.path).
			Methods(route.methods...).
			Handler(route.handler)
	}
}

type API struct {
	// A lock for reading/writing `config`.
	configLock sync.RWMutex

	// The server configuration.
	config *config.Config

	// The server router.
	router *mux.Router

	// A lock for reading from/writing to the hook store.
	hookStoreLock sync.RWMutex

	// The webhook store.
	hookStore hooks.WebhookStore

	// The token store.
	tokenStore tokens.TokenStore

	// The authenticator for requesting tokens.
	authenticator *auth.BasicAuth
}

// Return a new router for the API.
func New(cfg *config.Config) (*API, error) {
	api := API{
		config:        &config.Config{},
		router:        mux.NewRouter(),
		authenticator: auth.NewBasicAuthenticator("RB Gateway", nil),
	}

	if err := api.setConfigUnsafe(cfg); err != nil {
		return nil, err
	}

	api.router.Path("/session").
		Methods("GET", "POST").
		HandlerFunc(api.authenticator.Wrap(api.getSession))

	// The following routes all require token authorization.
	repoRouter := api.router.PathPrefix("/repos/{repo}").Subrouter()
	repoRouter.Use(api.withAuthorizationRequired)
	repoRouter.Use(api.withRepository)

	addRoutes(repoRouter, []routingEntry{
		{[]string{"GET"}, "/branches", http.HandlerFunc(api.getBranches)},
		{[]string{"GET"}, "/branches/{branch}/commits", http.HandlerFunc(api.getCommits)},
		{[]string{"GET"}, "/commits/{commit-id}", http.HandlerFunc(api.getCommit)},
		{[]string{"GET"}, "/commits/{commit-id}/path/{path}", http.HandlerFunc(api.getFileByCommit)},
		{[]string{"HEAD"}, "/commits/{commit-id}/path/{path}", http.HandlerFunc(api.getFileExistsByCommit)},
		{[]string{"GET"}, "/file/{file-id}", http.HandlerFunc(api.getFile)},
		{[]string{"HEAD"}, "/file/{file-id}", http.HandlerFunc(api.getFileExists)},
		{[]string{"GET"}, "/path", http.HandlerFunc(api.getPath)},
	})

	hookRouter := api.router.PathPrefix("/webhooks").Subrouter()
	hookRouter.Use(api.withAuthorizationRequired)

	addRoutes(hookRouter, []routingEntry{
		{[]string{"GET"}, "", http.HandlerFunc(api.getHooks)},
		{[]string{"POST"}, "", http.HandlerFunc(api.createHook)},
		{[]string{"GET"}, "/{hook-id}", http.HandlerFunc(api.getHook)},
		{[]string{"DELETE"}, "/{hook-id}", http.HandlerFunc(api.deleteHook)},
		{[]string{"PATCH"}, "/{hook-id}", http.HandlerFunc(api.updateHook)},
	})

	return &api, nil
}

// Update the configuration.
//
// If there is an error setting the configuration (e.g., from attempting to
// load a new token store), that error will be returned and the configuration
// will not bet set.
func (api *API) SetConfig(newConfig *config.Config) error {
	api.configLock.Lock()
	defer api.configLock.Unlock()

	return api.setConfigUnsafe(newConfig)
}

// Unsafely set the configuration.
//
// If there is an error setting the configuration (e.g., from attempting to
// load a new token store), that error will be returned and the configuration
// will not bet set.
//
// This is called internally by SetConfig (because we have acquired the lock)
// and in New (because the API object is still internal at that point). It is
// used in the latter case to avoid the overhead of unnecessary
// locking/unlocking.
func (api *API) setConfigUnsafe(newConfig *config.Config) error {
	tokenStore, err := tokens.NewStore(newConfig.TokenStorePath)
	if err != nil {
		return err
	}

	provider, err := newHtpasswdSecretProvider(newConfig.HtpasswdPath)
	if err != nil {
		return err
	}

	hookStore, err := hooks.LoadStore(newConfig.WebhookStorePath, newConfig.RepositorySet())
	if err != nil {
		return err
	}

	api.tokenStore = tokenStore
	api.authenticator.Secrets = provider
	api.config = newConfig
	api.hookStore = hookStore
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
	 * We have to acquire the lock here because the goroutine in api.Serve()
	 * acquires the read portion of the lock.
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
func (api *API) withRepository(next http.Handler) http.Handler {
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

func (api *API) withAuthorizationRequired(next http.Handler) http.Handler {
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

// Return the token store.
//
// This is intended for use only by unit tests.
func (api *API) GetTokenStore() *tokens.TokenStore {
	return &api.tokenStore
}
