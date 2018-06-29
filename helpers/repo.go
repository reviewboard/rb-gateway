package helpers

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
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

const (
	DefaultAuthor = "Author <author@example.com>"
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
