package integration_tests

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	git_config "github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/stretchr/testify/assert"

	"github.com/reviewboard/rb-gateway/helpers"
	"github.com/reviewboard/rb-gateway/repositories"
	"github.com/reviewboard/rb-gateway/repositories/events"
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

	upstream := setupBareGitRepo(t)
	defer helpers.CleanupRepository(t, upstream.Path)

	cfgDir, cfg := setupConfig(t, upstream)
	defer os.RemoveAll(cfgDir)

	hook := setupStore(t, server.URL, &cfg)

	assert.Nil(upstream.InstallHooks(filepath.Join(cfgDir, "config.json"), false))

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

	cases := []testCase{
		{
			recorded: &requests[0],
			message:  "Initial commit",
			commitId: origHead.String(),
			target: events.PushPayloadCommitTarget{
				Branch: "master",
			},
		},
		{
			recorded: &requests[1],
			message:  "New commit",
			commitId: newHead.String(),
			target: events.PushPayloadCommitTarget{
				Branch: "master",
			},
		},
	}

	runTests(t, cases, upstream, hook)
}

// Create a bare repository that we can push to.
func setupBareGitRepo(t *testing.T) *repositories.GitRepository {
	t.Helper()
	assert := assert.New(t)

	repoDir, err := ioutil.TempDir("", "rb-gateway-bare-repo-")
	assert.Nil(err)

	_, err = git.PlainInit(repoDir, true)
	assert.Nil(err)

	return &repositories.GitRepository{
		RepositoryInfo: repositories.RepositoryInfo{
			Name: "upstream",
			Path: repoDir,
		},
	}
}
