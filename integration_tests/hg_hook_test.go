package integration_tests

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	hg "bitbucket.org/gohg/gohg"
	"github.com/stretchr/testify/assert"

	"github.com/reviewboard/rb-gateway/helpers"
	"github.com/reviewboard/rb-gateway/repositories"
	"github.com/reviewboard/rb-gateway/repositories/events"
)

func TestIntegrationForHgHooks(t *testing.T) {
	assert := assert.New(t)

	server, requestsChan := helpers.CreateRequestRecorder(t)

	upstream, upstreamClient := setupBareHgRepository(t)
	defer helpers.CleanupHgRepo(t, upstreamClient)

	cfgDir, cfg := setupConfig(t, upstream)
	defer os.RemoveAll(cfgDir)

	hook := setupStore(t, server.URL, &cfg)

	assert.Nil(upstream.InstallHooks(filepath.Join(cfgDir, "config.json")))

	repo, client := cloneHgUpstream(t, upstreamClient)
	defer helpers.CleanupHgRepo(t, client)

	fmt.Printf("Upstream: %s\ncfgDir: %s\nrepo: %s\n", upstream.Path, cfgDir, repo.Path)

	helpers.CreateAndAddFilesHg(t, repo.Path, client, map[string][]byte{"foo": []byte("foo")})
	origHead := helpers.CommitHg(t, client, "Initial commit", helpers.DefaultAuthor)

	rsp, err := client.Push(nil, []string{"default"})
	assert.Nil(err)

	fmt.Printf("Response after first push:\n%s", string(rsp))

	helpers.CreateAndAddFilesHg(t, repo.Path, client, map[string][]byte{"bar": []byte("bar")})
	newHead := helpers.CommitHg(t, client, "New commit", helpers.DefaultAuthor)

	rsp, err = client.Push(nil, []string{"default"})
	assert.Nil(err)

	fmt.Printf("Response after second push:\n%s", string(rsp))

	requests := helpers.AssertNumRequests(t, 2, requestsChan)

	cases := []testCase{
		{
			recorded: &requests[0],
			message:  "Initial commit",
			commitId: origHead,
			target: events.PushPayloadCommitTarget{
				Branch: "default",
				Tags:   []string{"tip"},
			},
		},
		{
			recorded: &requests[1],
			message:  "New commit",
			commitId: newHead,
			target: events.PushPayloadCommitTarget{
				Branch: "default",
				Tags:   []string{"tip"},
			},
		},
	}

	runTests(t, cases, upstream, hook)
}

func setupBareHgRepository(t *testing.T) (*repositories.HgRepository, *hg.HgClient) {
	t.Helper()
	assert := assert.New(t)

	repoDir, err := ioutil.TempDir("", "rb-gateway-bare-repo-")
	assert.Nil(err)

	client := hg.NewHgClient()
	assert.Nil(client.Connect("hg", repoDir, nil, true))

	repo := repositories.HgRepository{
		RepositoryInfo: repositories.RepositoryInfo{
			Path: repoDir,
			Name: "upstream",
		},
	}

	return &repo, client
}

func cloneHgUpstream(t *testing.T, upstream *hg.HgClient) (*repositories.HgRepository, *hg.HgClient) {
	t.Helper()
	assert := assert.New(t)

	cloneDir, err := ioutil.TempDir("", "rb-gateway-clone-repo-")
	assert.Nil(err)

	assert.Nil(upstream.Clone(nil, []string{upstream.RepoRoot(), cloneDir}))

	client := hg.NewHgClient()
	assert.Nil(client.Connect("hg", cloneDir, nil, false))

	repo := repositories.HgRepository{
		RepositoryInfo: repositories.RepositoryInfo{
			Path: cloneDir,
			Name: "clone",
		},
	}

	return &repo, client
}
