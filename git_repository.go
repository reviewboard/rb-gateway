package main

import (
	"gopkg.in/libgit2/git2go.v22"
)

// GitRepository extends RepositoryInfo and is meant to represent a Git
// Repository.
type GitRepository struct {
	RepositoryInfo
}

// GetName is a Repository implementation that returns the name of the
// GitRepository.
func (repo *GitRepository) GetName() string {
	return repo.Name
}

// GetPath is a Repository implementation that returns the path of the
// GitRepository.
func (repo *GitRepository) GetPath() string {
	return repo.Path
}

// GetFile is a Repository implementation that returns the contents of a file
// in the GitRepository based on the file revision sha. On success, it returns
// the file contents in a byte array. On failure, the error will be returned.
func (repo *GitRepository) GetFile(id string) ([]byte, error) {
	gitRepo, err := git.OpenRepository(repo.Path)
	if err != nil {
		return nil, err
	}

	oid, err := git.NewOid(id)
	if err != nil {
		return nil, err
	}

	blob, err := gitRepo.LookupBlob(oid)
	if err != nil {
		return nil, err
	}

	return blob.Contents(), nil
}

// GetFileByCommit is a Repository implementation that returns the contents of
// a file in the GitRepository based on a commit sha and the file path. On
// success, it returns the file contents in a byte array. On failure, the error
// will be returned.
func (repo *GitRepository) GetFileByCommit(commit,
	filepath string) ([]byte, error) {
	gitRepo, err := git.OpenRepository(repo.Path)
	if err != nil {
		return nil, err
	}

	oid, err := git.NewOid(commit)
	if err != nil {
		return nil, err
	}

	c, err := gitRepo.LookupCommit(oid)
	if err != nil {
		return nil, err
	}

	tree, err := c.Tree()
	if err != nil {
		return nil, err
	}

	file, err := tree.EntryByPath(filepath)
	if err != nil {
		return nil, err
	}

	blob, err := gitRepo.LookupBlob(file.Id)
	if err != nil {
		return nil, err
	}

	return blob.Contents(), nil
}

// FileExists is a Repository implementation that returns whether a file exists
// in the GitRepository based on the file revision sha. It returns true if the
// file exists, false otherwise. On failure, the error will also be returned.
func (repo *GitRepository) FileExists(id string) (bool, error) {
	gitRepo, err := git.OpenRepository(repo.Path)
	if err != nil {
		return false, err
	}

	oid, err := git.NewOid(id)
	if err != nil {
		return false, err
	}

	if _, err := gitRepo.Lookup(oid); err != nil {
		return false, nil
	}

	return true, nil
}

// FileExistsByCommit is a Repository implementation that returns whether a
// file exists in the GitRepository based on a commit sha and the file path.
// It returns true if the file exists, false otherwise. On failure, the error
// will also be returned.
func (repo *GitRepository) FileExistsByCommit(commit,
	filepath string) (bool, error) {
	gitRepo, err := git.OpenRepository(repo.Path)
	if err != nil {
		return false, err
	}

	oid, err := git.NewOid(commit)
	if err != nil {
		return false, err
	}

	c, err := gitRepo.LookupCommit(oid)
	if err != nil {
		return false, err
	}

	tree, err := c.Tree()
	if err != nil {
		return false, err
	}

	file, err := tree.EntryByPath(filepath)
	if err != nil {
		return false, err
	}

	if _, err := gitRepo.Lookup(file.Id); err != nil {
		return false, nil
	}

	return true, nil
}
