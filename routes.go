package main

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"net/http"
	"strings"
)

const (
	paramRepository  = "repo"   // repo where the file is located
	paramId          = "id"     // a sha in Git, nodeid in Mercurial, etc.
	paramCommit      = "commit" // a commit
	paramPath        = "path"   // the file name
	paramBranch      = "branch" // the branch id
	paramStartCommit = "start"  // the starting commit id
)

// getFile gets the contents of a file, given the file id.
//
// If the request method is GET, this sends a response containing the file
// blob, if it exists. Otherwise, it will send a 404 Not Found. If the request
// method is HEAD, this sends a 200 OK response if the file exists. Otherwise,
// it will send a 404 Not Found.
//
// URL: /repos/:repo/file/:id
// Parameters:
//     repo - The repository name specified in config.json.
//     id - The file ID.
func getFile(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	repoName := params[paramRepository]
	id := params[paramId]

	if len(repoName) != 0 && len(id) != 0 {
		repo := GetRepository(repoName)
		if repo == nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		switch r.Method {
		case "GET":
			blob, err := repo.GetFile(id)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
			} else {
				w.Header().Set("Content-Type", "application/octet-stream")
				w.Write(blob)
			}
		case "HEAD":
			exists, err := repo.FileExists(id)
			if exists {
				w.WriteHeader(http.StatusOK)
			} else {
				if err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
				} else {
					w.WriteHeader(http.StatusNotFound)
				}
			}
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	} else {
		http.Error(w, "Repository or ID not specified", http.StatusBadRequest)
	}
}

// getFileByCommit gets the contents of a file, given the commit id and file
// path.
//
// If the request method is GET, this sends a response containing the file
// blob, if it exists. Otherwise, it will send a 404 Not Found. If the request
// method is HEAD, this sends a 200 OK response if the file exists. Otherwise,
// it will send a 404 Not Found.
//
// URL: /repos/:repo/commits/:commit/path/:path
// Parameters:
//     repo - The repository name specified in config.json.
//     commit - The commit ID to retrieve the file revision from.
//     path - The relative path of the file.
func getFileByCommit(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	repoName := params[paramRepository]
	commit := params[paramCommit]
	path := params[paramPath]

	if len(repoName) != 0 && len(commit) != 0 && len(path) != 0 {
		repo := GetRepository(repoName)
		if repo == nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		switch r.Method {
		case "GET":
			blob, err := repo.GetFileByCommit(commit, path)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
			} else {
				w.Header().Set("Content-Type", "application/octet-stream")
				w.Write(blob)
			}
		case "HEAD":
			exists, err := repo.FileExistsByCommit(commit, path)
			if exists {
				w.WriteHeader(http.StatusOK)
			} else {
				if err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
				} else {
					w.WriteHeader(http.StatusNotFound)
				}
			}
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	} else {
		http.Error(w, "Repository, Commit ID, or File path not specified",
			http.StatusBadRequest)
	}
}

// getPath retrieves the path of a valid repository.
//
// URL: /repos/:repo/path
// Parameters:
//     repo - The repository name specified in config.json.
func getPath(w http.ResponseWriter, r *http.Request) {
	repoName := mux.Vars(r)[paramRepository]

	if len(repoName) != 0 {
		repo := GetRepository(strings.Split(repoName, "/")[0])
		if repo == nil {
			http.Error(w, "Repository not found", http.StatusBadRequest)
			return
		}

		if r.Method == "GET" {
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte(repo.GetPath() + "/info/refs"))
		} else {
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	} else {
		http.Error(w, "Repository not specified", http.StatusBadRequest)
	}
}

// getBranches gets all the branches in a repository.
//
// On success this responds with a 200 OK response, and a JSON representation
// of the branches.
//
// URL: /repos/:repo/branches
// Parameters:
//     repo - The repository name specified in config.json.
func getBranches(w http.ResponseWriter, r *http.Request) {
	repoName := mux.Vars(r)[paramRepository]

	if len(repoName) != 0 {
		repo := GetRepository(repoName)
		if repo == nil {
			http.Error(w, "Repository not found", http.StatusNotFound)
			return
		}

		if r.Method == "GET" {
			branches, err := repo.GetBranches()
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
			} else {
				w.Header().Set("Content-Type", "application/json")
				w.Write(branches)
			}
		} else {
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	} else {
		http.Error(w, "Repository not specified", http.StatusBadRequest)
	}
}

// getCommits gets all the commits for the specified branch.
//
// On success this responds with a 200 OK response, and a JSON representation
// of the commits.
//
// URL: /repos/:repo/branches/:branch/commits
// Parameters:
//     repo - The repository name specified in config.json
//     branch - The branch ID.
//     start (optional) -  The commit ID to start listing commits from.
func getCommits(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	repoName := params[paramRepository]
	branch := params[paramBranch]
	start := r.URL.Query().Get(paramStartCommit)

	if len(repoName) != 0 && len(branch) != 0 {
		repo := GetRepository(repoName)
		if repo == nil {
			http.Error(w, "Repository not found", http.StatusNotFound)
			return
		}

		if r.Method == "GET" {
			commits, err := repo.GetCommits(branch, start)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
			} else {
				w.Header().Set("Content-Type", "application/json")
				w.Write(commits)
			}
		} else {
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	} else {
		http.Error(w, "Repository or branch not specified",
			http.StatusBadRequest)
	}
}

// getCommit retrieves commit information including the diff for the specified
// commit id.
//
// On success this responds with a 200 OK response, and a JSON representation
// of the commit.
//
// URL: /repos/:repo/commits/:id
// Parameters:
//     repo - The repository name specified in config.json.
//     id - The commit ID.
func getCommit(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	repoName := params[paramRepository]
	commit := params[paramId]

	if len(repoName) != 0 && len(commit) != 0 {
		repo := GetRepository(repoName)
		if repo == nil {
			http.Error(w, "Repository not found", http.StatusNotFound)
			return
		}

		if r.Method == "GET" {
			change, err := repo.GetCommit(commit)
			if err != nil {
				if strings.HasPrefix(err.Error(),
					"Object not found - failed to find pack entry") {
					http.Error(w, err.Error(), http.StatusNotFound)
				} else if strings.HasPrefix(err.Error(),
					"encoding/hex") {
					http.Error(w, err.Error(), http.StatusBadRequest)
				} else {
					http.Error(w, err.Error(), http.StatusInternalServerError)
				}
			} else {
				w.Header().Set("Content-Type", "application/json")
				w.Write(change)
			}
		} else {
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	} else {
		http.Error(w, "Repository or commit not specified",
			http.StatusBadRequest)
	}
}

// getSession uses Basic Authentication to return a private session token based
// on the authentication information provided in the request header.
//
// This responds with a JSON of the Session, in the following format:
// { "private-token":<token> }
//
// URL: /session
func getSession(w http.ResponseWriter, r *http.Request) {
	session, err := CreateSession(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	json, err := json.Marshal(session)
	if err != nil {
		http.Error(w, "Session marshalling error",
			http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(json)
}

// Route handles all the URL routing.
func Route() *mux.Router {
	router := mux.NewRouter()

	routes := map[string]handler{
		"/repos/{repo}/file/{id}":                    BasicAuth(getFile),
		"/repos/{repo}/commits/{commit}/path/{path}": BasicAuth(getFileByCommit),
		"/repos/{repo}/path":                         BasicAuth(getPath),
		"/repos/{repo}/branches":                     BasicAuth(getBranches),
		"/repos/{repo}/branches/{branch}/commits":    BasicAuth(getCommits),
		"/repos/{repo}/commits/{id}":                 BasicAuth(getCommit),
		"/session":                                   getSession,
	}

	for route, handler := range routes {
		router.HandleFunc(route, handler)
	}

	return router
}
