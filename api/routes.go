package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	auth "github.com/abbot/go-http-auth"
	"github.com/gorilla/mux"

	"github.com/reviewboard/rb-gateway/repositories"
	"github.com/reviewboard/rb-gateway/repositories/hooks"
)

// Return a session given basic auth credentials.
//
// URL: `/session`
func (api *API) getSession(w http.ResponseWriter, r *auth.AuthenticatedRequest) {
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
func (_ *API) getBranches(w http.ResponseWriter, r *http.Request) {
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
func (_ *API) getCommits(w http.ResponseWriter, r *http.Request) {
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
func (_ *API) getCommit(w http.ResponseWriter, r *http.Request) {
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
func (_ *API) getFile(w http.ResponseWriter, r *http.Request) {
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
func (_ *API) getFileExists(w http.ResponseWriter, r *http.Request) {
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
func (_ *API) getFileByCommit(w http.ResponseWriter, r *http.Request) {
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
func (_ *API) getFileExistsByCommit(w http.ResponseWriter, r *http.Request) {
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
func (_ *API) getPath(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (api *API) getHooks(w http.ResponseWriter, r *http.Request) {
	api.hookStoreLock.RLock()
	defer api.hookStoreLock.RUnlock()

	buffer := bytes.NewBufferString(`{"webhooks":[`)
	first := true

	for _, hook := range api.hookStore {
		if first {
			first = false
		} else {
			buffer.WriteString(",")
		}

		b, err := json.Marshal(hook)
		if err != nil {
			log.Printf("Could not serialize hooks: %s", err.Error())
			http.Error(w, "An unexpected error occurred.", http.StatusInternalServerError)
			return
		}

		buffer.Write(b)
	}

	buffer.WriteString("]}")

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	w.Write(buffer.Bytes())

}

func (api *API) createHook(w http.ResponseWriter, r *http.Request) {
	api.hookStoreLock.Lock()
	defer api.hookStoreLock.Unlock()

	var hook hooks.Webhook

	if err := json.NewDecoder(r.Body).Decode(&hook); err != nil {
		http.Error(w,
			fmt.Sprintf("Could not parse request body: %s", err.Error()),
			http.StatusBadRequest)
		return
	}

	if api.hookStore[hook.Id] != nil {
		http.Error(w,
			fmt.Sprintf(`A webhook with ID "%s" already exists.`, hook.Id),
			http.StatusBadRequest)
		return
	}

	if err := hook.Validate(api.config.RepositorySet()); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	api.hookStore[hook.Id] = &hook
	if err := api.hookStore.Save(api.config.WebhookStorePath); err != nil {
		// If we cannot save the store, revert our state so that we stay
		// consistent with it.
		log.Println("Could not save webhook store: ", err.Error())
		delete(api.hookStore, hook.Id)
		http.Error(w, "An unexpected error occurred.", http.StatusInternalServerError)
	} else {
		w.WriteHeader(http.StatusCreated)
	}
}

func (api *API) getHook(w http.ResponseWriter, r *http.Request) {
	api.hookStoreLock.RLock()
	defer api.hookStoreLock.RUnlock()

	hookId := mux.Vars(r)["hook-id"]

	var hook *hooks.Webhook
	if hook = api.hookStore[hookId]; hook == nil {
		http.Error(w, "No such webhook", http.StatusNotFound)
		return
	}

	b, err := json.Marshal(hook)
	if err != nil {
		log.Printf("Could not serialize hooks: %s", err.Error())
		http.Error(w, "An unexpected error occurred.", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}

func (api *API) deleteHook(w http.ResponseWriter, r *http.Request) {
	api.hookStoreLock.Lock()
	defer api.hookStoreLock.Unlock()

	hookId := mux.Vars(r)["hook-id"]

	var hook *hooks.Webhook
	if hook = api.hookStore[hookId]; hook == nil {
		http.Error(w, "No such webhook", http.StatusNotFound)
	}

	delete(api.hookStore, hookId)
	if err := api.hookStore.Save(api.config.WebhookStorePath); err != nil {
		// If we cannot save the store, revert our state so that we stay
		// consistent with it.
		log.Println("Could not save webhook store: ", err.Error())
		api.hookStore[hookId] = hook
		http.Error(w, "An unexpected error occurred.", http.StatusInternalServerError)
	} else {
		w.WriteHeader(http.StatusNoContent)
	}
}

func (api *API) updateHook(w http.ResponseWriter, r *http.Request) {
	api.hookStoreLock.Lock()
	defer api.hookStoreLock.Unlock()

	hookId := mux.Vars(r)["hook-id"]

	var hook *hooks.Webhook
	var exists bool
	if hook, exists = api.hookStore[hookId]; !exists {
		http.Error(w, "No such webhook", http.StatusNotFound)
		return
	}

	var parsedRequest struct {
		Id      *string  `json:"id"`
		Url     *string  `json:"url,omitempty"`
		Secret  *string  `json:"secret,omitempty"`
		Enabled *bool    `json:"enabled"`
		Events  []string `json:"events"`
		Repos   []string `json:"repos"`
	}

	if err := json.NewDecoder(r.Body).Decode(&parsedRequest); err != nil {
		http.Error(w,
			fmt.Sprintf("Could not parse request body: %s", err.Error()),
			http.StatusBadRequest)
		return
	}

	updatedHook := hooks.Webhook{
		Id:      hook.Id,
		Url:     hook.Url,
		Secret:  hook.Secret,
		Enabled: hook.Enabled,
		Events:  hook.Events[:],
		Repos:   hook.Repos[:],
	}

	if parsedRequest.Id != nil {
		http.Error(w, "Hook ID cannot be updated.", http.StatusBadRequest)
		return
	}

	if parsedRequest.Url != nil {
		updatedHook.Url = *parsedRequest.Url
	}

	if parsedRequest.Secret != nil {
		updatedHook.Secret = *parsedRequest.Secret
	}

	if parsedRequest.Enabled != nil {
		updatedHook.Enabled = *parsedRequest.Enabled
	}

	if parsedRequest.Events != nil {
		updatedHook.Events = parsedRequest.Events
	}

	if parsedRequest.Repos != nil {
		updatedHook.Repos = parsedRequest.Repos
	}

	if err := updatedHook.Validate(api.config.RepositorySet()); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	api.hookStore[hook.Id] = &updatedHook
	if err := api.hookStore.Save(api.config.WebhookStorePath); err != nil {
		// If we cannot save the store, revert our state so that we stay
		// consistent with it.
		api.hookStore[hook.Id] = hook
		log.Println("Could not update hook store: ", err.Error())
		http.Error(w, "An unexpected error occurred.", http.StatusInternalServerError)
	} else {
		var b []byte
		if b, err = json.MarshalIndent(updatedHook, "", "  "); err != nil {
			http.Error(w, "An unexpected error occurred.", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		w.Write(b)
	}
}
