package main

import (
	"bytes"
	"encoding/json"
	"strings"

	"gopkg.in/libgit2/git2go.v22"
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

	gitRepo, err := git.OpenRepository(repo.Path)
	if err != nil {
		return nil, err
	}

	iter, err := gitRepo.NewReferenceIterator()
	if err != nil {
		return nil, err
	}

	ref, err := iter.Next()
	for err == nil {
		if ref.IsBranch() {
			name := strings.Split(ref.Name(), "refs/heads/")[1]
			id := ref.Target().String()

			branches = append(branches, GitBranch{name, id})
		}
		ref, err = iter.Next()
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
func (repo *GitRepository) GetCommits(branch string,
	start string) ([]byte, error) {
	type GitCommit struct {
		Author   string `json:"author"`
		Id       string `json:"id"`
		Date     string `json:"date"`
		Message  string `json:"message"`
		ParentId string `json:"parent_id"`
	}

	var commits []GitCommit = make([]GitCommit, 0, commitsPageSize)

	gitRepo, err := git.OpenRepository(repo.Path)
	if err != nil {
		return nil, err
	}

	revWalk, err := gitRepo.Walk()
	if err != nil {
		return nil, err
	}

	revWalk.Sorting(git.SortTopological | git.SortTime)

	// First try to look up the branch by its sha. If this fails, attempt to
	// get the branch by name.
	oid, err := git.NewOid(branch)
	if err != nil {
		gitBranch, err := gitRepo.LookupBranch(branch, git.BranchLocal)
		if err != nil {
			return nil, err
		}
		oid = gitBranch.Target()
		branch = gitBranch.Target().String()
	}

	if len(start) == 0 {
		start = branch
	}

	startOid, err := git.NewOid(start)
	if err != nil {
		return nil, err
	}
	revWalk.Push(startOid)

	err = revWalk.HideGlob("tags/*")
	if err != nil {
		return nil, err
	}

	for revWalk.Next(oid) == nil {
		commit, err := gitRepo.LookupCommit(oid)
		if err != nil {
			return nil, err
		}

		var parent string
		if commit.ParentCount() > 0 {
			parent = commit.Parent(0).Id().String()
		}

		gitCommit := GitCommit{
			commit.Author().Name,
			commit.Id().String(),
			commit.Author().When.String(),
			commit.Message(),
			parent,
		}

		commits = append(commits, gitCommit)

		if len(commits) == commitsPageSize {
			// We only want to return at max one page of commits.
			break
		}
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

	gitRepo, err := git.OpenRepository(repo.Path)
	if err != nil {
		return nil, err
	}

	commitOid, err := git.NewOid(commitId)
	if err != nil {
		return nil, err
	}

	commit, err := gitRepo.LookupCommit(commitOid)
	if err != nil {
		return nil, err
	}

	var parent string
	var diff string

	commitTree, err := commit.Tree()
	if err != nil {
		return nil, err
	}

	options, err := git.DefaultDiffOptions()
	if err != nil {
		return nil, err
	}

	// Specifying full patch indices.
	options.IdAbbrev = patchIndexLength

	var parentTree *git.Tree
	if commit.ParentCount() > 0 {
		parent = commit.Parent(0).Id().String()
		parentTree, err = commit.Parent(0).Tree()
		if err != nil {
			return nil, err
		}
	}

	gitDiff, err := gitRepo.DiffTreeToTree(parentTree, commitTree, &options)
	if err != nil {
		return nil, err
	}

	deltas, err := gitDiff.NumDeltas()
	if err != nil {
		return nil, err
	}

	var buffer bytes.Buffer

	if deltas > 0 {
		for i := 0; i < deltas; i++ {
			patch, err := gitDiff.Patch(i)
			if err != nil {
				return nil, err
			}

			patchString, err := patch.String()
			if err != nil {
				return nil, err
			}

			buffer.WriteString(patchString)

			patch.Free()
		}
		diff = buffer.String()
	}

	gitCommit := GitCommit{
		commit.Author().Name,
		commit.Id().String(),
		commit.Author().When.String(),
		commit.Message(),
		parent,
		diff,
	}

	json, err := json.Marshal(gitCommit)
	if err != nil {
		return nil, err
	}

	return json, nil
}
