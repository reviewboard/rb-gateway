package api

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

	auth "github.com/abbot/go-http-auth"

	"github.com/reviewboard/rb-gateway/api/tokens"
	"github.com/reviewboard/rb-gateway/config"
	"github.com/reviewboard/rb-gateway/repositories/hooks"
)

// contextKey is a private type for context keys to avoid collisions.
type contextKey string

const (
	PrivateTokenHeader = "PRIVATE-TOKEN"

	// repoContextKey is the context key for the repository.
	repoContextKey contextKey = "repo"
)

type API struct {
	// A lock for reading/writing `config`.
	configLock sync.RWMutex

	// The server configuration.
	config *config.Config

	// The server router.
	router *http.ServeMux

	// A lock for reading from/writing to the hook store.
	hookStoreLock sync.RWMutex

	// The webhook store.
	hookStore hooks.WebhookStore

	// The token store.
	tokenStore tokens.TokenStore

	// The authenticator for requesting tokens.
	authenticator *auth.BasicAuth

	// The structured logger.
	logger *slog.Logger
}

// Return a new router for the API.
func New(cfg *config.Config) (*API, error) {
	api := API{
		config:        &config.Config{},
		router:        http.NewServeMux(),
		authenticator: auth.NewBasicAuthenticator("RB Gateway", nil),
		logger:        slog.Default(),
	}

	if err := api.setConfigUnsafe(cfg); err != nil {
		return nil, err
	}

	// Session endpoint uses basic auth.
	api.router.HandleFunc("GET /session", api.authenticator.Wrap(api.getSession))
	api.router.HandleFunc("POST /session", api.authenticator.Wrap(api.getSession))

	// Repository routes (require token auth + repo middleware).
	api.router.Handle("GET /repos/{repo}/branches", api.withAuth(api.withRepo(api.getBranches)))
	api.router.Handle("GET /repos/{repo}/branches/{branch}/commits", api.withAuth(api.withRepo(api.getCommits)))
	api.router.Handle("GET /repos/{repo}/commits/{commitID}", api.withAuth(api.withRepo(api.getCommit)))
	api.router.Handle("GET /repos/{repo}/commits/{commitID}/path/{path...}", api.withAuth(api.withRepo(api.getFileByCommit)))
	api.router.Handle("HEAD /repos/{repo}/commits/{commitID}/path/{path...}", api.withAuth(api.withRepo(api.getFileExistsByCommit)))
	api.router.Handle("GET /repos/{repo}/file/{fileID}", api.withAuth(api.withRepo(api.getFile)))
	api.router.Handle("HEAD /repos/{repo}/file/{fileID}", api.withAuth(api.withRepo(api.getFileExists)))
	api.router.Handle("GET /repos/{repo}/path", api.withAuth(api.withRepo(api.getPath)))

	// Webhook routes (require token auth).
	api.router.Handle("GET /webhooks", api.withAuth(http.HandlerFunc(api.getHooks)))
	api.router.Handle("POST /webhooks", api.withAuth(http.HandlerFunc(api.createHook)))
	api.router.Handle("GET /webhooks/{hookID}", api.withAuth(http.HandlerFunc(api.getHook)))
	api.router.Handle("DELETE /webhooks/{hookID}", api.withAuth(http.HandlerFunc(api.deleteHook)))
	api.router.Handle("PATCH /webhooks/{hookID}", api.withAuth(http.HandlerFunc(api.updateHook)))

	return &api, nil
}

// withAuth wraps a handler with token authorization.
func (api *API) withAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if api.tokenStore.Get(r) == nil {
			http.Error(w, "Authorization failed.", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// withRepo wraps a handler function with repository lookup middleware.
func (api *API) withRepo(next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		repoName := r.PathValue("repo")

		if len(repoName) == 0 {
			http.Error(w, "Repository not provided.", http.StatusBadRequest)
			return
		}

		repo, exists := api.config.Repositories[repoName]
		if !exists {
			http.Error(w, "Repository not found.", http.StatusNotFound)
			return
		}

		ctx := context.WithValue(r.Context(), repoContextKey, repo)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
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
		Handler: loggingMiddleware(api.logger, api.router),
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
			api.logger.Error("ListenAndServe failed", "err", err)
			os.Exit(1)
		}
	}()

	return &server
}

// Serve a request.
//
// This is only meant for unit tests since it acquires a lock for every request
// that would normally only be acquired once in `api.Serve`
func (api *API) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	api.configLock.RLock()
	defer api.configLock.RUnlock()

	loggingMiddleware(api.logger, api.router).ServeHTTP(w, r)
}

// Return the token store.
//
// This is intended for use only by unit tests.
func (api *API) GetTokenStore() *tokens.TokenStore {
	return &api.tokenStore
}
