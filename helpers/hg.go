package helpers

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/reviewboard/rb-gateway/repositories"
)

const (
	hgBin = "hg"
)

// HgClient is a thin wrapper around the hg command-line tool for testing.
type HgClient struct {
	// Path is the root directory of the repository.
	Path string
}

// RunHg executes an hg command in the repository directory.
func (c *HgClient) RunHg(args ...string) ([]byte, error) {
	cmd := exec.Command(hgBin, args...)
	cmd.Dir = c.Path
	return cmd.CombinedOutput()
}

// Create a Mercurial repository for testing.
//
// The caller is responsible for cleaning up the client and filesystem afterwards.
//
// Example:
//
// ```go
//
//	func Test(t *testing.T) {
//	    repo, hgClient := testing.CreateHgRepo(t, "repo-name")
//	    defer testing.CleanupHgRepo(t, hgClient)
//
//	    // ...
//	}
//
// ```
func CreateHgRepo(t *testing.T, name string) (*repositories.HgRepository, *HgClient) {
	t.Helper()
	assert := assert.New(t)

	path, err := os.MkdirTemp("", "rb-gateway-hg-repo-")
	assert.Nil(err)

	path, err = filepath.EvalSymlinks(path)
	assert.Nil(err)

	client := &HgClient{Path: path}
	_, err = client.RunHg("init", path)
	assert.Nil(err)

	repo := repositories.HgRepository{
		RepositoryInfo: repositories.RepositoryInfo{
			Name: name,
			Path: path,
		},
	}

	return &repo, client
}

// Clean up a created Mercurial repository.
func CleanupHgRepo(t *testing.T, client *HgClient) {
	t.Helper()

	err := os.RemoveAll(client.Path)
	assert.Nil(t, err)
}

// Create a new commit with some files, returning the commit ID.
//
// Callers can compare committed file contents with the result of `helpers.GetRepoFiles`.
func SeedHgRepo(t *testing.T, repo *repositories.HgRepository, client *HgClient) string {
	t.Helper()

	CreateAndAddFilesHg(t, repo.Path, client, repoFiles)

	return CommitHg(t, client, "Commit message", DefaultAuthor)
}

// Create a new bookmark with some test files, returning the commit ID.
//
// Callers can compare committed file contents with the result of `helpers.GetRepoFiles`.
func SeedHgBookmark(t *testing.T, repo *repositories.HgRepository, client *HgClient) string {
	t.Helper()
	assert := assert.New(t)

	_, err := client.RunHg("bookmark", "test-bookmark")
	assert.Nil(err)

	CreateAndAddFilesHg(t, repo.Path, client, branchFiles)

	return CommitHg(t, client, "Branch commit message", DefaultAuthor)
}

func CreateAndAddFilesHg(t *testing.T, repoPath string, client *HgClient, files map[string][]byte) {
	t.Helper()
	assert := assert.New(t)

	for filename, content := range files {
		path := filepath.Join(repoPath, filename)
		err := os.WriteFile(path, content, 0644)
		assert.Nil(err)

		_, err = client.RunHg("add", filename)
		assert.Nil(err)
	}
}

func CommitHg(t *testing.T, client *HgClient, message, author string) string {
	t.Helper()

	_, err := client.RunHg(
		"commit",
		"-m", message,
		"-u", author,
	)
	assert.Nil(t, err)

	return GetHgHead(t, client)
}

func GetHgHead(t *testing.T, client *HgClient) string {
	t.Helper()

	id, err := client.RunHg(
		"log",
		"--rev", ".",
		"--template", "{node}",
	)
	assert.Nil(t, err)

	return strings.TrimSpace(string(id))
}

func CreateHgTag(t *testing.T, client *HgClient, node, tag, message, author string) string {
	t.Helper()

	_, err := client.RunHg(
		"tag",
		"--rev", node,
		"-m", message,
		"-u", author,
		tag,
	)
	assert.Nil(t, err)

	return GetHgHead(t, client)

}

func CreateHgBookmark(t *testing.T, client *HgClient, node, bookmark string) {
	t.Helper()

	_, err := client.RunHg(
		"bookmark",
		"--rev", node,
		bookmark,
	)
	assert.Nil(t, err)
}
