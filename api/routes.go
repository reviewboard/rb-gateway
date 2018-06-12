package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	auth "github.com/abbot/go-http-auth"
	"github.com/gorilla/mux"

	"github.com/reviewboard/rb-gateway/repositories"
)

// Return a session given basic auth credentials.
//
// URL: `/session`
func (api API) getSession(w http.ResponseWriter, r *auth.AuthenticatedRequest) {
	token, err := api.tokenStore.New()

	if err != nil {
		log.Printf("Could not create session: %s", err.Error())
		http.Error(w, "Could not create session", http.StatusInternalServerError)
	}

	session := Session{
		PrivateToken: *token,
	}

	json, err := json.Marshal(&session)
	if err != nil {
		log.Printf("Could not serialize session: %s", err.Error())
		http.Error(w, "Could not create session", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(json)
}

// Return the branches in the repository.
//
// URL: `/repos/<repo>/branches`
func (api API) getBranches(w http.ResponseWriter, r *http.Request) {
	repo := r.Context().Value("repo").(repositories.Repository)

	var branches []repositories.Branch
	var response []byte
	var err error

	if branches, err = repo.GetBranches(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	} else if response, err = json.Marshal(branches); err != nil {
		log.Printf("Could not serialize branches: %s", err.Error())
		http.Error(w, "An unexpected error occurred.", http.StatusInternalServerError)
	} else {
		w.Header().Set("Content-Type", "application/json")
		w.Write(response)
	}
}

// Return the commits for a branch.
//
// URL: `/repos/<repo>/branches/<branch>/commits?start=<start>`
func (api API) getCommits(w http.ResponseWriter, r *http.Request) {
	repo := r.Context().Value("repo").(repositories.Repository)
	params := mux.Vars(r)
	branch := params["branch"]
	start := r.URL.Query().Get("start")

	var commits []repositories.CommitInfo
	var response []byte
	var err error

	if len(branch) == 0 {
		http.Error(w, "Branch not specified.", http.StatusBadRequest)
	} else if commits, err = repo.GetCommits(branch, start); err != nil {
		http.Error(w, fmt.Sprintf("Could not get branches: %s", err.Error()),
			http.StatusBadRequest)
	} else if response, err = json.Marshal(commits); err != nil {
		log.Printf("Could not serialize commits: %s", err.Error())
		http.Error(w, "An unexpected error occurred.", http.StatusInternalServerError)
	} else {
		w.Header().Set("Content-Type", "application/json")
		w.Write(response)
	}
}

// Return a commit.
//
// URL: `/repos/<repo>/commit/<commit-id>`
func (api API) getCommit(w http.ResponseWriter, r *http.Request) {
	repo := r.Context().Value("repo").(repositories.Repository)
	params := mux.Vars(r)
	commitId := params["commit-id"]

	var commit *repositories.Commit
	var response []byte
	var err error

	if len(commitId) == 0 {
		http.Error(w, "Commit ID not specified.", http.StatusBadRequest)
	} else if commit, err = repo.GetCommit(commitId); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else if commit == nil {
		http.Error(w, "Commit ID not found.", http.StatusNotFound)
	} else if response, err = json.Marshal(*commit); err != nil {
		log.Printf("Could not serialize commit \"%s\" in repo \"%s\": %s", commit.Id, repo.GetName(), err.Error())
		http.Error(w, "An unexpected error occurred.", http.StatusInternalServerError)
	} else {
		w.Header().Set("Content-Type", "application/json")
		w.Write(response)
	}
}

// Return the contents of a file (identified by an object ID) in a repository.
//
// URL: `/repos/<repo>/file/<file-id>`
func (api API) getFile(w http.ResponseWriter, r *http.Request) {
	repo := r.Context().Value("repo").(repositories.Repository)
	objectId := mux.Vars(r)["file-id"]

	var contents []byte
	var err error

	if len(objectId) == 0 {
		http.Error(w, "File ID not specified.", http.StatusBadRequest)
	} else if contents, err = repo.GetFile(objectId); err != nil {
		http.Error(w, fmt.Sprintf("Could not get file \"%s\": %s", objectId, err.Error()),
			http.StatusNotFound)
	} else {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write(contents)
	}
}

// Return whether or not a file (identified by an object ID) exists in a repository.
//
// URL: `/repos/<repo>/file/<file-id>`
func (api API) getFileExists(w http.ResponseWriter, r *http.Request) {
	repo := r.Context().Value("repo").(repositories.Repository)
	objectId := mux.Vars(r)["file-id"]

	var exists bool
	var err error

	if len(objectId) == 0 {
		http.Error(w, "File ID not specified.", http.StatusBadRequest)
	} else if exists, err = repo.FileExists(objectId); err != nil {
		http.Error(w, fmt.Sprintf("Could not find file \"%s\" in repo: %s", objectId, err.Error()),
			http.StatusBadRequest)
	} else if !exists {
		w.WriteHeader(http.StatusNotFound)
	} else {
		w.WriteHeader(http.StatusOK)
	}
}

// Return the contents of a file (at a specific commit) in a repository.
//
// URL: `/repos/<repo>/commits/<commit-id>/path/<path>`
func (api API) getFileByCommit(w http.ResponseWriter, r *http.Request) {
	repo := r.Context().Value("repo").(repositories.Repository)
	params := mux.Vars(r)

	commitId := params["commit-id"]
	path := params["path"]

	var contents []byte
	var err error

	if len(commitId) == 0 {
		http.Error(w, "Commit ID not specified.", http.StatusBadRequest)
	} else if len(path) == 0 {
		http.Error(w, "File path not specified.", http.StatusBadRequest)
	} else if contents, err = repo.GetFileByCommit(commitId, path); err != nil {
		http.Error(w,
			fmt.Sprintf("Could not get file \"%s\" at commit \"%s\": %s",
				path, commitId, err.Error()),
			http.StatusNotFound)
	} else {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write(contents)
	}
}

// Return whether or not a file (at a specific commit) exists in the repository.
//
// URL: `/repos/<repo>/commits/<commit-id>/path/<path>`
func (api API) getFileExistsByCommit(w http.ResponseWriter, r *http.Request) {
	repo := r.Context().Value("repo").(repositories.Repository)
	params := mux.Vars(r)

	commitId := params["commit-id"]
	path := params["path"]

	var exists bool
	var err error

	if len(commitId) == 0 {
		http.Error(w, "Commit ID not specified.", http.StatusBadRequest)
	} else if len(path) == 0 {
		http.Error(w, "File path not specified.", http.StatusBadRequest)
	} else if exists, err = repo.FileExistsByCommit(commitId, path); err != nil {
		http.Error(w,
			fmt.Sprintf("Could not find file \"%s\" at commit \"%s\": %s",
				path, commitId, err.Error()),
			http.StatusBadRequest)
	} else if !exists {
		w.WriteHeader(http.StatusNotFound)
	} else {
		w.WriteHeader(http.StatusOK)
	}
}

// Return an HTTP OK if the user can access the repository.
//
// Review Board has shipped with rb-gateway support requiring this endpoint to
// confirm access to the repository. However, all it does is check for a 200 OK.
//
// Since this is behind the authorization middleware, we can always just return
// 200 OK.
//
// URL: `/repos/<repo>/path`
func (api API) getPath(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}
