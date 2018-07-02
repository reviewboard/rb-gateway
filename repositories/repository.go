package repositories

import (
	"io"

	"github.com/reviewboard/rb-gateway/repositories/events"
)

// RepositoryInfo is a generic representation of a repository, containing
// a name and a path to the repository.
type RepositoryInfo struct {
	Name string
	Path string
}

// Repository is an interface that contains functions to perform actions on
// repositories, such as getting the file contents or checking if a file
// exists.
type Repository interface {
	// GetName returns the name of the repository.
	GetName() string

	// GetPath returns the path of the repository.
	GetPath() string

	// Return the name of the SCM.
	GetScm() string

	// GetFile takes a file ID and returns the file contents as a byte array.
	// If an error occurs, it will also be returned.
	GetFile(id string) ([]byte, error)

	// GetFileByCommit takes a commit and a file path pair, and returns the
	// file contents as a byte array. If an error occurs, it will also be
	// returned.
	GetFileByCommit(commit, filepath string) ([]byte, error)

	// FileExists takes a file ID and returns true if the file is found in the
	// repository; false otherwise. If an error occurs, it will also be
	// returned.
	FileExists(id string) (bool, error)

	// FileExistsByCommit takes a commit and file path pair, and returns true
	// if the file is found in the repository; false otherwise. If an error
	// occurs, it will also be returned.
	FileExistsByCommit(commit, filepath string) (bool, error)

	// GetBranches returns all the branches in the repository as a JSON byte
	// array. If an error occurs, it will also be returned.
	GetBranches() ([]Branch, error)

	// GetCommit returns all the commits in the repository starting at the
	// specified branch as a JSON byte array. It also takes an optional start
	// commit id, which will return all commits starting from the start commit
	// id instead. If an error occurs, it will also be returned.
	GetCommits(branch string, start string) ([]CommitInfo, error)

	// GetCommit returns the commit in the repository provided by the commit
	// id as a JSON byte array. If an error occurs, it will also be returned.
	GetCommit(commitId string) (*Commit, error)

	// Parse the raw payload from the given event.
	ParseEventPayload(event string, input io.Reader) (events.Payload, error)

	// Install scripts to trigger webhooks.
	InstallHooks(cfgPath string) error
}

// Metadata about a commit.
type CommitInfo struct {
	// The author of the commit.
	Author string `json:"author"`

	// The unique identifier of the commit.
	Id string `json:"id"`

	// The date the commit was authored.
	Date string `json:"date"`

	// The commit's message.
	Message string `json:"message"`

	// The unique identifier of the parent commit.
	ParentId string `json:"parent_id"`
}

// A commit with metadata and a diff.
type Commit struct {
	// Commit metadata.
	CommitInfo

	// The contents of the diff.
	Diff string `json:"diff"`
}

// Information about a branch in an SCM.
type Branch struct {
	// The name of the branch.
	Name string `json:"name"`

	// The commit ID the branch points to.
	Id string `json:"id"`
}
