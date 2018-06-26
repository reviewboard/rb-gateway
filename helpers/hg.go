package helpers

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	hg "bitbucket.org/gohg/gohg"
	"github.com/stretchr/testify/assert"

	"github.com/reviewboard/rb-gateway/repositories"
)

const (
	hgBin = "hg"
)

// Create a Mercurial repository for testing.
//
// The caller is responsible for cleaning up the client and filesystem afterwards.
//
// Example:
//
// ```go
// func Test(t *testing.T) {
//     repo, hgClient := testing.CreateHgtRepo(t, "repo-name")
//     defer testing.CleanupHgRepo(t, hgClient)
//
//     // ...
// }
// ```
func CreateHgRepo(t *testing.T, name string) (*repositories.HgRepository, *hg.HgClient) {
	t.Helper()
	assert := assert.New(t)

	path, err := ioutil.TempDir("", "rb-gateway-hg-repo-")
	assert.Nil(err)

	path, err = filepath.EvalSymlinks(path)
	assert.Nil(err)

	client := hg.NewHgClient()
	assert.Nil(client.Connect(hgBin, path, nil, true))

	repo := repositories.HgRepository{
		RepositoryInfo: repositories.RepositoryInfo{
			Name: name,
			Path: path,
		},
	}

	return &repo, client
}

// Clean up a created Mercurial repository.
func CleanupHgRepo(t *testing.T, client *hg.HgClient) {
	t.Helper()

	client.Disconnect()
	err := os.RemoveAll(client.RepoRoot())
	assert.Nil(t, err)
}

// Create a new commit with some files, returning the commit ID.
//
// Callers can compare committed file contents with the result of `helpers.GetRepoFiles`.
func SeedHgRepo(t *testing.T, repo *repositories.HgRepository, client *hg.HgClient) string {
	t.Helper()
	assert := assert.New(t)

	createAndAddFilesHg(t, repo.Path, client, repoFiles)

	_, err := client.ExecCmd([]string{"commit", "-m", "Commit message", "-u", "Author <author@example.com>"})
	assert.Nil(err)

	id, err := client.ExecCmd([]string{"log", "-r", ".", "--template", "{node}"})
	assert.Nil(err)

	return string(id)
}

// Create a new bookmark with some test files, returning the commit ID.
//
// Callers can compare committed file contents with the result of `helpers.GetRepoFiles`.
func SeedHgBookmark(t *testing.T, repo *repositories.HgRepository, client *hg.HgClient) string {
	t.Helper()
	assert := assert.New(t)

	_, err := client.ExecCmd([]string{"bookmark", "test-bookmark"})
	assert.Nil(err)

	createAndAddFilesHg(t, repo.Path, client, branchFiles)

	_, err = client.ExecCmd([]string{"commit", "-m", "Branch commit message", "-u", "Author <author@example.com>"})
	assert.Nil(err)

	id, err := client.ExecCmd([]string{"log", "-r", ".", "--template", "{node}"})
	assert.Nil(err)

	return string(id)
}

func createAndAddFilesHg(t *testing.T, repoPath string, client *hg.HgClient, files map[string][]byte) {
	t.Helper()
	assert := assert.New(t)

	for filename, content := range files {
		path := filepath.Join(repoPath, filename)
		err := ioutil.WriteFile(path, content, 0644)
		assert.Nil(err)

		_, err = client.ExecCmd([]string{"add", filename})
		assert.Nil(err)
	}
}
