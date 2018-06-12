package repositories

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
)

const (
	commitsPageSize        = 20 // The max page size for commits.
	branchesAllocationSize = 10 // The initial allocation size for branches.
	patchIndexLength       = 40 // The patch index length.
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

func (repo *GitRepository) GetScm() string {
	return "git"
}

// GetFile is a Repository implementation that returns the contents of a file
// in the GitRepository based on the file revision sha.
//
// On success, it returns the file contents in a byte array. On failure, the
// error will be returned.
func (repo *GitRepository) GetFile(id string) ([]byte, error) {
	gitRepo, err := git.PlainOpen(repo.Path)
	if err != nil {
		return nil, err
	}

	blob, err := gitRepo.BlobObject(plumbing.NewHash(id))
	if err != nil {
		return nil, err
	}

	reader, err := blob.Reader()
	if err != nil {
		return nil, err
	}

	defer reader.Close()

	buf := new(bytes.Buffer)
	buf.ReadFrom(reader)

	return buf.Bytes(), nil
}

// GetFileByCommit is a Repository implementation that returns the contents of
// a file in the GitRepository based on a commit sha and the file path.
//
// On success, it returns the file contents in a byte array. On failure, the
// error will be returned.
func (repo *GitRepository) GetFileByCommit(commitId, filepath string) ([]byte, error) {
	gitRepo, err := git.PlainOpen(repo.Path)
	if err != nil {
		return nil, err
	}

	commit, err := gitRepo.CommitObject(plumbing.NewHash(commitId))
	if err != nil {
		return nil, err
	}

	tree, err := gitRepo.TreeObject(commit.TreeHash)
	if err != nil {
		return nil, err
	}

	file, err := tree.FindEntry(filepath)
	if err != nil {
		return nil, err
	}

	blob, err := gitRepo.BlobObject(file.Hash)
	if err != nil {
		return nil, err
	}

	reader, err := blob.Reader()
	if err != nil {
		return nil, err
	}

	defer reader.Close()

	buf := new(bytes.Buffer)
	buf.ReadFrom(reader)

	return buf.Bytes(), nil
}

// FileExists is a Repository implementation that returns whether a file exists
// in the GitRepository based on the file revision sha.
//
// It returns true if the file exists, false otherwise. On failure, the error
// will also be returned.
func (repo *GitRepository) FileExists(id string) (bool, error) {
	gitRepo, err := git.PlainOpen(repo.Path)
	if err != nil {
		return false, err
	}

	_, err = gitRepo.BlobObject(plumbing.NewHash(id))
	if err != nil {
		if err.Error() == "object not found" {
			return false, nil
		} else {
			return false, err
		}
	}

	return true, nil
}

// FileExistsByCommit is a Repository implementation that returns whether a
// file exists in the GitRepository based on a commit sha and the file path.
//
// It returns true if the file exists, false otherwise. On failure, the error
// will also be returned.
func (repo *GitRepository) FileExistsByCommit(commitId, filepath string) (bool, error) {
	gitRepo, err := git.PlainOpen(repo.Path)
	if err != nil {
		return false, err
	}

	commit, err := gitRepo.CommitObject(plumbing.NewHash(commitId))
	if err != nil {
		return false, err
	}

	tree, err := gitRepo.TreeObject(commit.TreeHash)
	if err != nil {
		return false, err
	}

	file, err := tree.FindEntry(filepath)
	if err != nil {
		return false, err
	}

	_, err = gitRepo.BlobObject(file.Hash)
	if err != nil {
		return false, err
	}

	return true, nil
}

// GetBranches is a Repository implementation that returns all the branches in
// the repository.
//
// On failure, the error will also be returned.
func (repo *GitRepository) GetBranches() ([]Branch, error) {
	var branches []Branch = make([]Branch, 0, branchesAllocationSize)

	gitRepo, err := git.PlainOpen(repo.Path)
	if err != nil {
		return nil, err
	}

	iter, err := gitRepo.Branches()
	if err != nil {
		return nil, err
	}

	err = iter.ForEach(func(ref *plumbing.Reference) error {
		name := strings.Split(ref.Name().String(), "refs/heads/")[1]
		id := ref.Hash().String()

		branches = append(branches, Branch{
			Name: name,
			Id:   id,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	return branches, nil
}

// GetCommits is a Repository implementation that returns all the commits in
// the repository for the specified branch. It also takes an optional start
// commit sha, which will return all commits starting from the start commit
// sha instead.
//
// On failure, the error will also be returned.
func (repo *GitRepository) GetCommits(branch string, start string) ([]CommitInfo, error) {
	var commits []CommitInfo = make([]CommitInfo, 0, commitsPageSize)

	gitRepo, err := git.PlainOpen(repo.Path)
	if err != nil {
		return nil, err
	}

	var startCommit plumbing.Hash
	if len(start) != 0 {
		startCommit = plumbing.NewHash(start)
	} else {
		ref, err := gitRepo.Reference(plumbing.ReferenceName(fmt.Sprintf("refs/heads/%s", branch)), true)
		if err != nil {
			return nil, err
		}

		startCommit = ref.Hash()
	}

	iter, err := gitRepo.Log(&git.LogOptions{
		From:  startCommit,
		Order: git.LogOrderCommitterTime,
	})
	if err != nil {
		return nil, err
	}

	commit, err := iter.Next()
	for err == nil {
		if len(commits) == commitsPageSize {
			// We only want to return at max one page of commits.
			break
		}

		var parent string
		if commit.NumParents() > 0 {
			parent = commit.ParentHashes[0].String()
		}

		commitInfo := CommitInfo{
			Author:   commit.Author.Name,
			Id:       commit.Hash.String(),
			Date:     commit.Author.When.Format("2006-01-02T15:04:05-0700"),
			Message:  commit.Message,
			ParentId: parent,
		}

		commits = append(commits, commitInfo)

		commit, err = iter.Next()
	}

	return commits, nil
}

// GetCommit is a Repository implementation that returns the commit information
// in the repository for the specified commit id.
//
// On failure, the error will also be returned.
func (repo *GitRepository) GetCommit(commitId string) (*Commit, error) {
	gitRepo, err := git.PlainOpen(repo.Path)
	if err != nil {
		return nil, err
	}

	commit, err := gitRepo.CommitObject(plumbing.NewHash(commitId))
	if err != nil {
		if err.Error() == "object not found" {
			return nil, nil
		} else {
			return nil, err
		}
	}

	if commit.NumParents() == 0 {
		return nil, errors.New("Commit has no parents.")
	}

	parent, err := commit.Parent(0)
	if err != nil {
		return nil, err
	}

	patch, err := parent.Patch(commit)
	if err != nil {
		return nil, err
	}

	change := Commit{
		CommitInfo: CommitInfo{
			Author:   commit.Author.Name,
			Id:       commit.Hash.String(),
			Date:     commit.Author.When.Format("2006-01-02T15:04:05-0700"),
			Message:  commit.Message,
			ParentId: commit.ParentHashes[0].String(),
		},
		Diff: patch.String(),
	}

	return &change, nil
}
