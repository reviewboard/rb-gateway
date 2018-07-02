package integration_tests

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/foomo/htpasswd"
	"github.com/stretchr/testify/assert"
	"gopkg.in/src-d/go-git.v4"
	git_config "gopkg.in/src-d/go-git.v4/config"
	"gopkg.in/src-d/go-git.v4/plumbing/object"

	"github.com/reviewboard/rb-gateway/config"
	"github.com/reviewboard/rb-gateway/helpers"
	"github.com/reviewboard/rb-gateway/repositories"
	"github.com/reviewboard/rb-gateway/repositories/events"
	"github.com/reviewboard/rb-gateway/repositories/hooks"
)

// Integration tests for Git hooks.
//
// This test sets up a a Git repository and installs RB Gateway's hook scripts
// into it, with an RB Gateway configuration that points a webhook at a
// webserver we are running. Then we push some commits to it and monitor the
// requests made to the webserver, verifying that they are correct.
func TestIntegrtionForGitHooks(t *testing.T) {
	assert := assert.New(t)

	server, requestsChan := helpers.CreateRequestRecorder(t)

	upstream := setupBareRepo(t)
	defer helpers.CleanupRepository(t, upstream.Path)

	cfgDir, cfg := setupConfig(t, upstream)
	defer os.RemoveAll(cfgDir)

	hook := setupStore(t, server.URL, &cfg)

	assert.Nil(upstream.InstallHooks(filepath.Join(cfgDir, "config.json")))

	repo, gitRepo := helpers.CreateGitRepo(t, "clone")
	defer helpers.CleanupRepository(t, repo.Path)

	fmt.Printf("Upstream: %s\ncfgDir: %s\nrepo: %s\n", upstream.Path, cfgDir, repo.Path)

	_, err := gitRepo.CreateRemote(&git_config.RemoteConfig{
		Name: "origin",
		URLs: []string{upstream.Path},
	})
	assert.Nil(err)

	worktree, err := gitRepo.Worktree()
	assert.Nil(err)

	progressBuffer := new(bytes.Buffer)
	pushOptions := &git.PushOptions{
		RemoteName: "origin",
		RefSpecs:   []git_config.RefSpec{"refs/heads/master:refs/heads/master"},
		Progress:   progressBuffer,
	}

	origHead, err := worktree.Commit("Initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Author",
			Email: "author@example.com",
			When:  time.Now(),
		},
	})

	assert.Nil(gitRepo.Push(pushOptions))
	fmt.Printf("Response from first push:\n %s\n", progressBuffer.String())

	progressBuffer.Reset()

	newHead, err := worktree.Commit("New commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Author",
			Email: "author@example.com",
			When:  time.Now(),
		},
	})

	assert.Nil(gitRepo.Push(pushOptions))
	fmt.Printf("Response from second push:\n %s\n", progressBuffer.String())

	requests := helpers.AssertNumRequests(t, 2, requestsChan)

	testCases := []struct {
		recorded *helpers.RecordedRequest
		message  string
		commitId string
	}{
		{
			recorded: &requests[0],
			message:  "Initial commit",
			commitId: origHead.String(),
		},
		{
			recorded: &requests[1],
			message:  "New commit",
			commitId: newHead.String(),
		},
	}

	for i, testCase := range testCases {
		request := testCase.recorded.Request
		body := testCase.recorded.Body

		assert.Equalf("/test-hook", request.URL.Path, "URL for request %d does not match", i)
		assert.Equalf(events.PushEvent, request.Header.Get("X-RBG-Event"), "X-RBG-Event header for request %d does not match", i)
		assert.Equalf(hook.SignPayload(body), request.Header.Get("X-RBG-Signature"), "Signature for request %d does not match", i)
		payload := events.PushPayload{
			Repository: "bare-repo",
			Commits: []events.PushPayloadCommit{
				{
					Id:      testCase.commitId,
					Message: testCase.message,
					Target: events.PushPayloadCommitTarget{
						Branch: "master",
					},
				},
			},
		}

		rawJson, err := events.MarshalPayload(payload)
		assert.Nil(err)

		assert.Equalf(string(rawJson), string(body), "Body for request %d does not match", i)
	}
}

// Create a bare repository that we can push to.
func setupBareRepo(t *testing.T) *repositories.GitRepository {
	t.Helper()
	assert := assert.New(t)

	repoDir, err := ioutil.TempDir("", "rb-gateway-bare-repo-")
	assert.Nil(err)

	_, err = git.PlainInit(repoDir, true)
	assert.Nil(err)

	return &repositories.GitRepository{
		RepositoryInfo: repositories.RepositoryInfo{
			Name: "bare-repo",
			Path: repoDir,
		},
	}
}

// Set up the configuration and write it to disk.
func setupConfig(t *testing.T, upstream *repositories.GitRepository) (string, config.Config) {
	t.Helper()
	assert := assert.New(t)

	cfgDir, err := ioutil.TempDir("", "rb-gateway-config-")
	assert.Nil(err)
	cfg := helpers.CreateTestConfig(t, upstream)

	hookStorePath := filepath.Join(cfgDir, "webhooks.json")
	cfg.WebhookStorePath = hookStorePath
	cfg.TokenStorePath = filepath.Join(cfgDir, "tokens.dat")
	cfg.HtpasswdPath = filepath.Join(cfgDir, "htpasswd")

	assert.Nil(htpasswd.SetPassword(cfg.HtpasswdPath, "username", "password", htpasswd.HashBCrypt))

	helpers.WriteConfig(t, filepath.Join(cfgDir, "config.json"), &cfg)

	return cfgDir, cfg
}

// Set up the webhook store and write it to disk.
func setupStore(t *testing.T, serverUrl string, cfg *config.Config) *hooks.Webhook {
	assert := assert.New(t)
	t.Helper()

	var repoName string
	for repoName, _ = range cfg.Repositories {
		break
	}

	assert.NotEqual(0, len(repoName))
	hook := &hooks.Webhook{
		Id:      "test-hook",
		Enabled: true,
		Url:     fmt.Sprintf("%s/test-hook", serverUrl),
		Secret:  "top-secret-123",
		Events:  []string{events.PushEvent},
		Repos:   []string{repoName},
	}

	store := hooks.WebhookStore{
		hook.Id: hook,
	}

	f, err := os.OpenFile(cfg.WebhookStorePath, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0600)
	assert.Nil(err)

	defer f.Close()

	err = store.Save(f)
	assert.Nil(err)

	return hook
}
