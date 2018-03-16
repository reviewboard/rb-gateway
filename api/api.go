package api

import (
	"net/http"

	"github.com/gorilla/mux"
)

// Return a new router for the API.
func New() http.Handler {
	routes := mux.NewRouter()
	routes.Use(loggingMiddleware)

	routes.Path("/session").
		Methods("GET").
		HandlerFunc(getSession)

	// The following routes all require authentication.
	repoRoutes := routes.PathPrefix("/repos/{repo}").
		Subrouter()
	repoRoutes.Use(authenticationMiddleware)

	repoRoutes.Path("/branches").
		Methods("GET").
		HandlerFunc(getBranches)

	repoRoutes.Path("/branches/{branch}/commits").
		Methods("GET").
		HandlerFunc(getCommits)

	repoRoutes.Path("/commits/{id}").
		Methods("GET").
		HandlerFunc(getCommit)

	repoRoutes.Path("/file/{id}").
		Methods("GET", "HEAD").
		HandlerFunc(getFile)

	repoRoutes.Path("/commits/{commit}/path/{path}").
		Methods("GET", "HEAD").
		HandlerFunc(getFileByCommit)

	repoRoutes.Path("/path").
		Methods("GET").
		HandlerFunc(getPath)

	return routes
}
