package main

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
	GetBranches() ([]byte, error)

	// GetCommit returns all the commits in the repository starting at the
	// specified branch as a JSON byte array. It also takes an optional start
	// commit id, which will return all commits starting from the start commit
	// id instead. If an error occurs, it will also be returned.
	GetCommits(branch string, start string) ([]byte, error)

	// GetCommit returns the commit in the repository provided by the commit
	// id as a JSON byte array. If an error occurs, it will also be returned.
	GetCommit(commitId string) ([]byte, error)
}
