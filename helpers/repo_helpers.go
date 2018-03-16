package helpers

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	git "github.com/libgit2/git2go"
	"github.com/stretchr/testify/assert"

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

// Get the files contained in the repository's initial commit.
//
// This returns a copy of the original data structure, so it may be mutated by callers.
func GetRepoFiles() (files map[string][]byte) {
	files = make(map[string][]byte)

	for key, content := range repoFiles {
		files[key] = content
	}

	return
}

// The the files contained only on the repository's alternate branch.
//
// This returns a copy of the original data structure, so it may be mutated by callers.
func GetBranchFiles() (files map[string][]byte) {
	files = make(map[string][]byte)

	for key, content := range branchFiles {
		files[key] = content
	}

	return
}

// Clean up a testing repository.
//
// This deletes the temporary files from disk.
func CleanupRepository(t *testing.T, rawRepo *git.Repository) {
	t.Helper()

	var err error
	if rawRepo.IsBare() {
		err = os.RemoveAll(rawRepo.Path())
	} else {
		err = os.RemoveAll(rawRepo.Workdir())
	}

	assert.Nil(t, err, "Could not cleanup repository.")
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
//     defer testing.CleanupRepository(t, rawRepo)
//
//     // ...
// }
// ```
func CreateTestRepo(t *testing.T, name string) (*repositories.GitRepository, *git.Repository) {
	t.Helper()

	path, err := ioutil.TempDir("", "rb-gateway-test-")
	assert.Nil(t, err, "Could not create temporary directory.")
	path, err = filepath.EvalSymlinks(path)
	assert.Nil(t, err, "Could not get absolute path.")

	rawRepo, err := git.InitRepository(path, false)
	assert.Nil(t, err, "Could not initialize repository.")

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
func SeedTestRepo(t *testing.T, rawRepo *git.Repository) *git.Oid {
	t.Helper()

	loc, err := time.LoadLocation("UTC")
	assert.Nil(t, err)

	sig := git.Signature{
		Name:  "Author",
		Email: "author@example.com",
		When:  time.Date(2015, 03, 12, 2, 15, 0, 0, loc),
	}

	index := createAndAddFiles(t, rawRepo, repoFiles)
	treeId, err := index.WriteTree()
	assert.Nil(t, err)

	tree, err := rawRepo.LookupTree(treeId)
	assert.Nil(t, err)

	commitId, err := rawRepo.CreateCommit("HEAD", &sig, &sig, "Initial commit", tree)
	assert.Nil(t, err)

	return commitId
}

// Create a new branch with some test files, returning the commit ID.
//
// Callers can compare committed file contents with the result of `testing.GetBranchFiles`.
func CreateTestBranch(t *testing.T, rawRepo *git.Repository) *git.Branch {
	t.Helper()

	loc, err := time.LoadLocation("UTC")
	assert.Nil(t, err)

	sig := git.Signature{
		Name:  "Author",
		Email: "author@example.com",
		When:  time.Date(2015, 03, 12, 2, 15, 0, 0, loc),
	}

	// Create a new branch based off HEAD.
	head, err := rawRepo.Head()
	assert.Nil(t, err)
	headCommit, err := rawRepo.LookupCommit(head.Target())
	assert.Nil(t, err)
	branch, err := rawRepo.CreateBranch("test-branch", headCommit, false)
	assert.Nil(t, err)

	index := createAndAddFiles(t, rawRepo, branchFiles)
	treeId, err := index.WriteTree()
	assert.Nil(t, err)

	tree, err := rawRepo.LookupTree(treeId)
	assert.Nil(t, err)

	_, err = rawRepo.CreateCommit("refs/heads/test-branch", &sig, &sig, "Add branch", tree, headCommit)
	assert.Nil(t, err)

	branch, err = rawRepo.LookupBranch("test-branch", git.BranchLocal)
	assert.Nil(t, err)

	return branch
}

// Return the object ID of the given file.
func GetRepositoryFileId(t *testing.T, rawRepo *git.Repository, path string) *git.Oid {
	head, err := rawRepo.Head()
	assert.Nil(t, err)

	headCommit, err := rawRepo.LookupCommit(head.Target())
	assert.Nil(t, err)

	tree, err := headCommit.Tree()
	assert.Nil(t, err)

	entry := tree.EntryByName(path)
	assert.NotNil(t, entry)

	return entry.Id
}

// Get the object ID of the repository head.
func GetRepoHead(t *testing.T, rawRepo *git.Repository) *git.Oid {
	head, err := rawRepo.Head()
	assert.Nil(t, err)

	return head.Target()
}

// Create some files and add them to to an index.
func createAndAddFiles(t *testing.T, rawRepo *git.Repository, files map[string][]byte) *git.Index {
	t.Helper()

	index, err := rawRepo.Index()
	assert.Nil(t, err)

	for filename, content := range repoFiles {
		path := filepath.Join(rawRepo.Workdir(), filename)

		err = ioutil.WriteFile(path, content, 0644)
		assert.Nil(t, err)

		err = index.AddByPath(filename)
		assert.Nil(t, err)
	}

	return index
}
