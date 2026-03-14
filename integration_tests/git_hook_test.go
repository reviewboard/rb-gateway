//go:build integration

package integration_tests

import (
	"bytes"
	"fmt"
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
func TestIntegrationForGitHooks(t *testing.T) {
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

	commitOpts := &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Author",
			Email: "author@example.com",
			When:  time.Now(),
		},
	}

	// Create a file so the commit is non-empty.
	err = os.WriteFile(filepath.Join(repo.Path, "README"), []byte("test"), 0644)
	assert.Nil(err)
	_, err = worktree.Add("README")
	assert.Nil(err)

	origHead, err := worktree.Commit("Initial commit", commitOpts)
	assert.Nil(err)

	// Determine the branch name from HEAD (depends on git's
	// init.defaultBranch config, e.g. "main" or "master").
	headRef, err := gitRepo.Head()
	assert.Nil(err)
	branchName := headRef.Name().Short()

	refSpec := git_config.RefSpec(
		fmt.Sprintf("refs/heads/%s:refs/heads/%s", branchName, branchName),
	)

	progressBuffer := new(bytes.Buffer)
	pushOptions := &git.PushOptions{
		RemoteName: "origin",
		RefSpecs:   []git_config.RefSpec{refSpec},
		Progress:   progressBuffer,
	}

	assert.Nil(gitRepo.Push(pushOptions))
	fmt.Printf("Response from first push:\n %s\n", progressBuffer.String())

	progressBuffer.Reset()

	err = os.WriteFile(filepath.Join(repo.Path, "CHANGES"), []byte("changes"), 0644)
	assert.Nil(err)
	_, err = worktree.Add("CHANGES")
	assert.Nil(err)

	newHead, err := worktree.Commit("New commit", commitOpts)
	assert.Nil(err)

	assert.Nil(gitRepo.Push(pushOptions))
	fmt.Printf("Response from second push:\n %s\n", progressBuffer.String())

	requests := helpers.AssertNumRequests(t, 2, requestsChan)

	cases := []testCase{
		{
			recorded: &requests[0],
			message:  "Initial commit",
			commitId: origHead.String(),
			target: events.PushPayloadCommitTarget{
				Branch: branchName,
			},
		},
		{
			recorded: &requests[1],
			message:  "New commit",
			commitId: newHead.String(),
			target: events.PushPayloadCommitTarget{
				Branch: branchName,
			},
		},
	}

	runTests(t, cases, upstream, hook)
}

// Create a bare repository that we can push to.
func setupBareGitRepo(t *testing.T) *repositories.GitRepository {
	t.Helper()
	assert := assert.New(t)

	repoDir, err := os.MkdirTemp("", "rb-gateway-bare-repo-")
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
