package repositories

import (
	"bytes"
	"encoding/json"
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
// in the GitRepository based on the file revision sha. On success, it returns
// the file contents in a byte array. On failure, the error will be returned.
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
// a file in the GitRepository based on a commit sha and the file path. On
// success, it returns the file contents in a byte array. On failure, the error
// will be returned.
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
// in the GitRepository based on the file revision sha. It returns true if the
// file exists, false otherwise. On failure, the error will also be returned.
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
// the repository. It returns a JSON representation of the branches containing
// the branch name and sha id. On failure, the error will also be returned.
//
// The JSON returned has the following format:
// [
//   {
//     "name": master,
//     "id": "1b6f00c0fe975dd12251431ed2ea561e0acc6d44"
//   }
// ]
func (repo *GitRepository) GetBranches() ([]byte, error) {
	type GitBranch struct {
		Name string `json:"name"`
		Id   string `json:"id"`
	}

	var branches []GitBranch = make([]GitBranch, 0, branchesAllocationSize)

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

		branches = append(branches, GitBranch{name, id})
		return nil
	})
	if err != nil {
		return nil, err
	}

	json, err := json.Marshal(branches)
	if err != nil {
		return nil, err
	}

	return json, nil
}

// GetCommits is a Repository implementation that returns all the commits in
// the repository for the specified branch. It also takes an optional start
// commit sha, which will return all commits starting from the start commit
// sha instead. The returned JSON representation of the commits contains the
// author's name, the sha id, the date, the commit message, and the parent sha.
// On failure, the error will also be returned.
//
// The JSON returned has the following format:
// [
//   {
//     "author": "John Doe",
//     "id": "1b6f00c0fe975dd12251431ed2ea561e0acc6d44",
//     "date": "2015-06-27T05:51:39-07:00",
//     "message": "Add README.md",
//     "parent_id": "bfdde95432b3af879af969bd2377dc3e55ee46e6"
//   }
// ]
func (repo *GitRepository) GetCommits(branch string, start string) ([]byte, error) {
	type GitCommit struct {
		Author   string `json:"author"`
		Id       string `json:"id"`
		Date     string `json:"date"`
		Message  string `json:"message"`
		ParentId string `json:"parent_id"`
	}

	var commits []GitCommit = make([]GitCommit, 0, commitsPageSize)

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
		From: startCommit,
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

		gitCommit := GitCommit{
			commit.Author.Name,
			commit.Hash.String(),
			commit.Author.When.String(),
			commit.Message,
			parent,
		}

		commits = append(commits, gitCommit)

		commit, err = iter.Next()
	}

	json, err := json.Marshal(commits)
	if err != nil {
		return nil, err
	}

	return json, nil
}

// GetCommit is a Repository implementation that returns the commit information
// in the repository for the specified commit id. The returned JSON
// representation of the commit contains the author's name, the sha id, the
// date, the commit message, the parent sha, and the diff. On failure, the
// error will also be returned.
//
// The JSON returned has the following format:
// {
//   "author": "John Doe",
//   "id": "1b6f00c0fe975dd12251431ed2ea561e0acc6d44",
//   "date": "2015-06-27T05:51:39-07:00",
//   "message": "Add README.md",
//   "parent_id": "bfdde95432b3af879af969bd2377dc3e55ee46e6",
//   "diff": "diff --git a/test b/test
//            index e69de29bb2d1d6434b8b29ae775ad8c2e48c5391..044f599c9a720fe1a7d02e694a8dab492cbda8f0 100644
//            --- a/test
//            +++ b/test
//            @@ -1 +1,3 @@
//            test
//            +
//            +test"
// }
func (repo *GitRepository) GetCommit(commitId string) ([]byte, error) {
	type GitCommit struct {
		Author   string `json:"author"`
		Id       string `json:"id"`
		Date     string `json:"date"`
		Message  string `json:"message"`
		ParentId string `json:"parent_id"`
		Diff     string `json:"diff"`
	}

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

	gitCommit := GitCommit{
		commit.Author.Name,
		commit.Hash.String(),
		commit.Author.When.String(),
		commit.Message,
		commit.ParentHashes[0].String(),
		patch.String(),
	}

	json, err := json.Marshal(gitCommit)
	if err != nil {
		return nil, err
	}

	return json, nil
}
