package repositories

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"

	"github.com/reviewboard/rb-gateway/repositories/events"
)

const (
	commitsPageSize        = 20 // The max page size for commits.
	branchesAllocationSize = 10 // The initial allocation size for branches.
	patchIndexLength       = 40 // The patch index length.

	refsHeadsPrefix = "refs/heads/"
)

var (
	nullRevision = plumbing.NewHash(strings.Repeat("0", 40))
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

func (repo *GitRepository) ParseEventPayload(event string, input io.Reader) (events.Payload, error) {
	if !events.IsValidEvent(event) {
		return nil, events.InvalidEventErr
	}

	gitRepo, err := git.PlainOpen(repo.Path)
	if err != nil {
		return nil, err
	}

	switch event {
	case events.PushEvent: // post-receive
		return repo.parsePushEvent(gitRepo, event, input)

	default:
		return nil, fmt.Errorf(`Event "%s" unsupported by Git.`, event)
	}
}

// Parse a post-receive hook and turn it into a PushPayload
func (repo *GitRepository) parsePushEvent(
	gitRepo *git.Repository,
	event string,
	input io.Reader,
) (events.Payload, error) {
	data, err := ioutil.ReadAll(input)

	if err != nil {
		return nil, err
	} else if len(data) == 0 {
		return nil, errors.New("No input")
	}

	records := strings.Split(strings.TrimRight(string(data), "\n"), "\n")

	// A set of all commit hashes we have processed. A commit may appear in
	// more than one set of updated refs.
	seen := make(map[plumbing.Hash]bool)

	payload := events.PushPayload{
		Repository: repo.Name,
		Commits:    []events.PushPayloadCommit{},
	}

	for _, record := range records {
		fields := strings.Split(record, " ")
		if len(fields) != 3 {
			return nil, errors.New("Invalid input format")
		}

		oldRevision := plumbing.NewHash(fields[0])
		newRevision := plumbing.NewHash(fields[1])
		refName := fields[2]

		// This revision was deleted and therefore doesn't correspond to new
		// changes being pushed.
		if newRevision == nullRevision {
			continue
		}

		if !strings.HasPrefix(refName, refsHeadsPrefix) {
			// This is some ref type we don't care about.
			continue
		}
		branchName := strings.TrimPrefix(refName, refsHeadsPrefix)

		ignore := []plumbing.Hash{}

		if oldRevision == nullRevision {
			// A new branch was created. We only want refs belonging to this
			// branch, so we are going to ignore all refs belonging to other
			// branches.
			iter, err := gitRepo.Branches()
			if err != nil {
				return nil, err
			}

			err = iter.ForEach(func(r *plumbing.Reference) error {
				if r.Name().String() != refName {
					ignore = append(ignore, r.Hash())
				}
				return nil
			})
			if err != nil {
				return nil, err
			}
		} else {
			base, err := mergeBase(gitRepo, newRevision, oldRevision)

			if err != nil {
				return nil, err
			} else if base != nil {
				ignore = append(ignore, *base)
			}
		}

		startCommit, err := object.GetCommit(gitRepo.Storer, newRevision)
		if err != nil {
			return nil, err
		}

		commits := []*object.Commit{}
		err = object.NewCommitPreorderIter(startCommit, seen, ignore).
			ForEach(func(c *object.Commit) error {
				seen[c.Hash] = true
				commits = append(commits, c)
				return nil
			})

		if err != nil {
			return nil, err
		}

		// The commits will be yielded in DAG order, i.e.,
		// reverse-chronological order. We want to go through them in
		// chronological order, so we traverse this slice in reverse.
		for i := len(commits) - 1; i >= 0; i-- {
			commit := commits[i]
			payload.Commits = append(payload.Commits, events.PushPayloadCommit{
				Id:      commit.Hash.String(),
				Message: commit.Message,
				Target: events.PushPayloadCommitTarget{
					Branch: branchName,
				},
			})
		}
	}

	return payload, nil
}

func setDefault(m map[plumbing.Hash]struct{}, h plumbing.Hash) map[plumbing.Hash]struct{} {
	if m == nil {
		m = make(map[plumbing.Hash]struct{})
	}
	m[h] = struct{}{}
	return m
}

// Return a merge base between the two commits.
//
// This algorithm is biased to return the closest ancestor of `a` that is
// common to `a` and `b`.
//
// Other bases may exist.
func mergeBase(gitRepo *git.Repository, a, b plumbing.Hash) (*plumbing.Hash, error) {
	commitA, err := object.GetCommit(gitRepo.Storer, a)
	if err != nil {
		return nil, err
	}

	commitB, err := object.GetCommit(gitRepo.Storer, b)
	if err != nil {
		return nil, err
	}

	// The direct descendants of each commit.
	directDescendants := make(map[plumbing.Hash]map[plumbing.Hash]struct{})

	// Ancestors of commit A.
	ancestors := make(map[plumbing.Hash]int)
	maxDistance := 0

	err = object.NewCommitPreorderIter(commitA, nil, nil).
		ForEach(func(c *object.Commit) error {
			maxDistance++
			ancestors[c.Hash] = maxDistance
			for _, p := range c.ParentHashes {
				directDescendants[p] = setDefault(directDescendants[p], c.Hash)
			}
			return nil
		})

	if err != nil {
		return nil, err
	}

	// Find all common ancestors of A and B.
	commonAncestors := make(map[plumbing.Hash]struct{})
	err = object.NewCommitPreorderIter(commitB, nil, nil).
		ForEach(func(c *object.Commit) error {
			if _, exists := ancestors[c.Hash]; exists {
				commonAncestors[c.Hash] = struct{}{}
			} else {
				for _, p := range c.ParentHashes {
					directDescendants[p] = setDefault(directDescendants[p], c.Hash)
				}
			}
			return nil
		})
	if err != nil {
		return nil, err
	}

	if len(commonAncestors) == 0 {
		return nil, nil
	}

	var best plumbing.Hash
	minDistance := -1

	// We want to select the commit the closest distance to a.
outer:
	for h := range commonAncestors {
		// Any commit with a child that is a common ancestor of `a` and `b`
		// cannot be the closest to `a` by definition.
		for child := range directDescendants[h] {
			if _, exists := commonAncestors[child]; exists {
				continue outer
			}
		}

		distance := ancestors[h]
		if distance < minDistance || minDistance == -1 {
			minDistance = distance
			best = h
		}
	}

	return &best, nil
}
