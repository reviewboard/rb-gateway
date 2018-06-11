package helpers

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"

	"github.com/reviewboard/rb-gateway/repositories"
)

var (
	// A set of files to create in the initial commit.
	repoFiles = map[string][]byte{
		"README":  []byte("README\n"),
		"COPYING": []byte("COPYING\n"),
	}

	// A set of files to create in the branch commit.
	branchFiles = map[string][]byte{
		"AUTHORS": []byte("AUTHORS\n"),
	}
)

// Get the files contained in the repository.
//
// This returns a copy of the original data structure, so it may be mutated by callers.
func GetRepoFiles() (files map[string][]byte) {
	files = make(map[string][]byte)

	for key, content := range repoFiles {
		files[key] = content
	}
	for key, content := range branchFiles {
		files[key] = content
	}

	return
}

// Clean up a testing repository.
//
// This deletes the temporary files from disk.
func CleanupRepository(t *testing.T, path string) {
	t.Helper()

	err := os.RemoveAll(path)
	assert.Nil(t, err, "Could not cleanup repository")
}

// Create a Git Repository for testing.
//
// The caller is responsible for cleaning up the filesystem afterwards.
//
// Example:
//
// ```go
// func Test(t *testing.T) {
//     repo, rawRepo := testing.CreateTestRepo(t)
//     defer testing.CleanupRepository(t, repo.Path)
//
//     // ...
// }
// ```
func CreateTestRepo(t *testing.T, name string) (*repositories.GitRepository, *git.Repository) {
	t.Helper()
	assert := assert.New(t)

	path, err := ioutil.TempDir("", "rb-gateway-test-")
	assert.Nil(err, "Could not create temporary directory.")
	path, err = filepath.EvalSymlinks(path)
	assert.Nil(err, "Could not get absolute path.")

	rawRepo, err := git.PlainInit(path, false)
	assert.Nil(err, "Could not initialize repository.")

	repo := &repositories.GitRepository{
		repositories.RepositoryInfo{
			Name: name,
			Path: path,
		},
	}

	return repo, rawRepo
}

// Add files to a repository and commit them, returning the commit ID.
//
// Callers can compare committed file contents with the result of `testing.GetRepoFiles`.
func SeedTestRepo(t *testing.T, repo *repositories.GitRepository, rawRepo *git.Repository) plumbing.Hash {
	t.Helper()
	assert := assert.New(t)

	worktree, err := rawRepo.Worktree()
	assert.Nil(err)

	createAndAddFiles(t, repo.Path, worktree, repoFiles)

	commitId, err := worktree.Commit("Initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Author",
			Email: "author@example.com",
			When:  time.Now(),
		},
	})
	assert.Nil(err)

	return commitId
}

// Create a new branch with some test files, returning the commit ID.
//
// Callers can compare committed file contents with the result of `testing.GetRepoFiles`.
func CreateTestBranch(t *testing.T, repo *repositories.GitRepository, rawRepo *git.Repository) *plumbing.Reference {
	t.Helper()
	assert := assert.New(t)

	worktree, err := rawRepo.Worktree()
	assert.Nil(err)

	err = worktree.Checkout(&git.CheckoutOptions{
		Branch: "refs/heads/test-branch",
		Create: true,
	})
	assert.Nil(err)

	createAndAddFiles(t, repo.Path, worktree, branchFiles)

	_, err = worktree.Commit("Add branch", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Author",
			Email: "author@example.com",
			When:  time.Now(),
		},
	})
	assert.Nil(err)

	branch, err := rawRepo.Reference("refs/heads/test-branch", false)
	assert.Nil(err)

	return branch
}

// Return the object ID of the given file.
func GetRepositoryFileId(t *testing.T, rawRepo *git.Repository, path string) plumbing.Hash {
	t.Helper()
	assert := assert.New(t)

	head, err := rawRepo.Head()
	assert.Nil(err)

	headCommit, err := rawRepo.CommitObject(head.Hash())
	assert.Nil(err)

	tree, err := headCommit.Tree()
	assert.Nil(err)

	entry, err := tree.FindEntry(path)
	assert.Nil(err)

	return entry.Hash
}

// Get the object ID of the repository head.
func GetRepoHead(t *testing.T, rawRepo *git.Repository) plumbing.Hash {
	t.Helper()

	head, err := rawRepo.Head()
	assert.Nil(t, err)

	return head.Hash()
}

// Create some files and add them to to an index.
func createAndAddFiles(t *testing.T, path string, worktree *git.Worktree, files map[string][]byte) {
	t.Helper()
	assert := assert.New(t)

	for filename, content := range files {
		path := filepath.Join(path, filename)

		err := ioutil.WriteFile(path, content, 0644)
		assert.Nil(err)

		_, err = worktree.Add(filename)
		assert.Nil(err)
	}
}
