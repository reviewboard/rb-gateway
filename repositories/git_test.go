package repositories_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"

	"github.com/reviewboard/rb-gateway/helpers"
	"github.com/reviewboard/rb-gateway/repositories/events"
)

func TestGetFile(t *testing.T) {
	assert := assert.New(t)

	repo, rawRepo := helpers.CreateGitRepo(t, "repo")
	defer helpers.CleanupRepository(t, repo.Path)

	helpers.SeedGitRepo(t, repo, rawRepo)
	fileId := helpers.GetRepositoryFileId(t, rawRepo, "README").String()

	fileContent, err := repo.GetFile(fileId)
	assert.Nil(err)

	expectedContent := helpers.GetRepoFiles()["README"]
	assert.Equal(string(expectedContent), string(fileContent))
}

func TestGetFileByCommit(t *testing.T) {
	assert := assert.New(t)

	repo, rawRepo := helpers.CreateGitRepo(t, "repo")
	defer helpers.CleanupRepository(t, repo.Path)

	commitId := helpers.SeedGitRepo(t, repo, rawRepo).String()

	fileContent, err := repo.GetFileByCommit(commitId, "README")
	assert.Nil(err)

	expectedContent := helpers.GetRepoFiles()["README"]
	assert.Equal(string(expectedContent), string(fileContent))
}

func TestFileExists(t *testing.T) {
	assert := assert.New(t)

	repo, rawRepo := helpers.CreateGitRepo(t, "repo")
	defer helpers.CleanupRepository(t, repo.Path)

	helpers.SeedGitRepo(t, repo, rawRepo)
	fileId := helpers.GetRepositoryFileId(t, rawRepo, "README").String()

	exists, err := repo.FileExists(fileId)
	assert.Nil(err)
	assert.True(exists)
}

func TestFileExistsByCommit(t *testing.T) {
	assert := assert.New(t)

	repo, rawRepo := helpers.CreateGitRepo(t, "repo")
	defer helpers.CleanupRepository(t, repo.Path)

	commitId := helpers.SeedGitRepo(t, repo, rawRepo).String()

	exists, err := repo.FileExistsByCommit(commitId, "README")
	assert.Nil(err)
	assert.True(exists)
}

func TestGetBranches(t *testing.T) {
	assert := assert.New(t)

	repo, rawRepo := helpers.CreateGitRepo(t, "repo")
	defer helpers.CleanupRepository(t, repo.Path)

	helpers.SeedGitRepo(t, repo, rawRepo)
	branch := helpers.CreateGitBranch(t, repo, rawRepo)
	branchName := branch.Name().Short()

	branches, err := repo.GetBranches()
	assert.Nil(err)

	assert.Equal(2, len(branches))

	for i := range branches {
		assert.Contains([]string{"master", branchName}, branches[i].Name)

		if branches[i].Name == branchName {
			assert.Equal(branch.Hash().String(), branches[i].Id)
		}
	}
}

func TestGetCommits(t *testing.T) {
	assert := assert.New(t)

	repo, rawRepo := helpers.CreateGitRepo(t, "repo")
	defer helpers.CleanupRepository(t, repo.Path)

	commitId := helpers.SeedGitRepo(t, repo, rawRepo)
	commit, err := rawRepo.CommitObject(commitId)
	assert.Nil(err)

	branch := helpers.CreateGitBranch(t, repo, rawRepo)
	branchCommit, err := rawRepo.CommitObject(branch.Hash())
	assert.Nil(err)

	branchName := branch.Name().Short()
	assert.Nil(err)

	// Testing GetCommits without a starting commit.
	commits, err := repo.GetCommits(branchName, "")
	assert.Nil(err)

	assert.Equal(len(commits), 2)

	assert.Equal(branchCommit.Message, commits[0].Message)
	assert.Equal(branch.Hash().String(), commits[0].Id)
	assert.Equal(commitId.String(), commits[0].ParentId)
	assert.Equal(commit.Message, commits[1].Message)
	assert.Equal(commitId.String(), commits[1].Id)
	assert.Equal("", commits[1].ParentId)

	// Testing GetCommits with a starting commit.
	commits, err = repo.GetCommits(branchName, commitId.String())
	assert.Nil(err)

	assert.Equal(len(commits), 1)

	assert.Equal(commit.Message, commits[0].Message)
	assert.Equal(commitId.String(), commits[0].Id)
}

func TestGetCommit(t *testing.T) {
	assert := assert.New(t)

	repo, rawRepo := helpers.CreateGitRepo(t, "repo")
	defer helpers.CleanupRepository(t, repo.Path)

	helpers.SeedGitRepo(t, repo, rawRepo)

	branch := helpers.CreateGitBranch(t, repo, rawRepo)

	expected, err := rawRepo.CommitObject(branch.Hash())
	assert.Nil(err)

	result, err := repo.GetCommit(branch.Hash().String())
	assert.Nil(err)

	fileIds := make(map[string]string)
	files := helpers.GetRepoFiles()
	for filename := range files {
		fileIds[filename] = helpers.GetRepositoryFileId(t, rawRepo, filename).String()
	}

	assert.Equal(expected.Message, result.CommitInfo.Message)

	diff := fmt.Sprintf(`diff --git a/AUTHORS b/AUTHORS
new file mode 100644
index 0000000000000000000000000000000000000000..%s
--- /dev/null
+++ b/AUTHORS
@@ -0,0 +1 @@
+%s`, fileIds["AUTHORS"], string(files["AUTHORS"]))

	assert.Equal(diff, result.Diff)
}

func TestGitParsePushEvent(t *testing.T) {
	assert := assert.New(t)

	repo, rawRepo := helpers.CreateGitRepo(t, "git-repo")
	defer helpers.CleanupRepository(t, repo.Path)

	oldHead := helpers.SeedGitRepo(t, repo, rawRepo)
	commitIds := make([]plumbing.Hash, 0, 3)

	worktree, err := rawRepo.Worktree()
	assert.Nil(err)
	for i := 1; i <= 3; i++ {
		assert.Nil(err)

		commitId, err := worktree.Commit(fmt.Sprintf("Commit %d", i), &git.CommitOptions{
			Author: &object.Signature{
				Name:  "Author",
				Email: "author@example.com",
				When:  time.Now(),
			},
		})
		assert.Nil(err)

		commitIds = append(commitIds, commitId)
	}

	input := strings.NewReader(fmt.Sprintf("%s %s refs/heads/master\n", oldHead.String(), commitIds[2].String()))

	payload, err := repo.ParseEventPayload(events.PushEvent, input)
	assert.Nil(err)
	expected := events.PushPayload{
		Repository: repo.Name,
		Commits: []events.PushPayloadCommit{
			{
				Id:      commitIds[0].String(),
				Message: "Commit 1",
				Target: events.PushPayloadCommitTarget{
					Branch: "master",
				},
			},
			{
				Id:      commitIds[1].String(),
				Message: "Commit 2",
				Target: events.PushPayloadCommitTarget{
					Branch: "master",
				},
			},
			{
				Id:      commitIds[2].String(),
				Message: "Commit 3",
				Target: events.PushPayloadCommitTarget{
					Branch: "master",
				},
			},
		},
	}
	assert.Equal(expected, payload)
}

func TestGitParsePushEventNewBranch(t *testing.T) {
	assert := assert.New(t)

	repo, rawRepo := helpers.CreateGitRepo(t, "git-repo")
	defer helpers.CleanupRepository(t, repo.Path)

	helpers.SeedGitRepo(t, repo, rawRepo)

	commitIds := make([]plumbing.Hash, 0, 2)
	worktree, err := rawRepo.Worktree()
	worktree.Checkout(&git.CheckoutOptions{
		Branch: "refs/heads/dev",
		Create: true,
	})

	for i := 1; i <= 2; i++ {
		commitId, err := worktree.Commit(fmt.Sprintf("Commit %d", i), &git.CommitOptions{
			Author: &object.Signature{
				Name:  "Author",
				Email: "author@example.com",
				When:  time.Now(),
			},
		})
		assert.Nil(err)

		commitIds = append(commitIds, commitId)
	}

	input := strings.NewReader(fmt.Sprintf("%040d %s refs/heads/dev\n", 0, commitIds[1].String()))

	payload, err := repo.ParseEventPayload(events.PushEvent, input)
	assert.Nil(err)
	expected := events.PushPayload{
		Repository: repo.Name,
		Commits: []events.PushPayloadCommit{
			{
				Id:      commitIds[0].String(),
				Message: "Commit 1",
				Target: events.PushPayloadCommitTarget{
					Branch: "dev",
				},
			},
			{
				Id:      commitIds[1].String(),
				Message: "Commit 2",
				Target: events.PushPayloadCommitTarget{
					Branch: "dev",
				},
			},
		},
	}

	assert.Equal(expected, payload)

}

// This test models a force push to a repository.
//
// It creates the following branch struture:
//
// Before force push:
// o -- A -- o -- B
//
// New:
// o -- A -- o -- B' (original B)
//       \
//        -- C -- B
//
// The payload should contain the commits C and B.
func TestGitParsePushEventRebase(t *testing.T) {
	assert := assert.New(t)

	repo, rawRepo := helpers.CreateGitRepo(t, "git-repo")
	defer helpers.CleanupRepository(t, repo.Path)

	mergeBase := helpers.SeedGitRepo(t, repo, rawRepo)

	worktree, err := rawRepo.Worktree()
	assert.Nil(err)

	var oldHead plumbing.Hash
	for i := 0; i < 2; i++ {
		oldHead, err = worktree.Commit(fmt.Sprintf("Commit %d", i+1), &git.CommitOptions{
			Author: &object.Signature{
				Name:  "Author",
				Email: "author@example.com",
				When:  time.Now(),
			},
		})
		assert.Nil(err)
	}

	err = worktree.Checkout(&git.CheckoutOptions{
		Hash:   mergeBase,
		Branch: "refs/heads/dev",
		Create: true,
	})
	assert.Nil(err)

	commitIds := make([]plumbing.Hash, 0, 2)
	for i := 1; i <= 2; i++ {
		commitId, err := worktree.Commit(fmt.Sprintf("New Commit %d", i), &git.CommitOptions{
			Author: &object.Signature{
				Name:  "Author",
				Email: "author@example.com",
				When:  time.Now(),
			},
		})
		assert.Nil(err)

		commitIds = append(commitIds, commitId)
	}

	input := strings.NewReader(fmt.Sprintf("%s %s refs/heads/dev\n", oldHead.String(), commitIds[1].String()))

	payload, err := repo.ParseEventPayload(events.PushEvent, input)
	assert.Nil(err)
	expected := events.PushPayload{
		Repository: repo.Name,
		Commits: []events.PushPayloadCommit{
			{
				Id:      commitIds[0].String(),
				Message: "New Commit 1",
				Target: events.PushPayloadCommitTarget{
					Branch: "dev",
				},
			},
			{
				Id:      commitIds[1].String(),
				Message: "New Commit 2",
				Target: events.PushPayloadCommitTarget{
					Branch: "dev",
				},
			},
		},
	}

	assert.Equal(expected, payload)
}

func TestGitParsePushEventMultiple(t *testing.T) {
	assert := assert.New(t)

	repo, rawRepo := helpers.CreateGitRepo(t, "git-repo")
	defer helpers.CleanupRepository(t, repo.Path)

	helpers.SeedGitRepo(t, repo, rawRepo)
	worktree, err := rawRepo.Worktree()
	assert.Nil(err)

	err = worktree.Checkout(&git.CheckoutOptions{
		Branch: "refs/heads/branch-1",
		Create: true,
	})

	commitIds := make([]plumbing.Hash, 0, 4)

	for i := 1; i <= 2; i++ {
		commitId, err := worktree.Commit(fmt.Sprintf("Commit %d", i), &git.CommitOptions{
			Author: &object.Signature{
				Name:  "Author",
				Email: "author@example.com",
				When:  time.Now(),
			},
		})
		assert.Nil(err)

		commitIds = append(commitIds, commitId)
	}

	err = worktree.Checkout(&git.CheckoutOptions{
		Branch: "refs/heads/branch-2",
		Create: true,
	})

	for i := 3; i <= 4; i++ {
		commitId, err := worktree.Commit(fmt.Sprintf("Commit %d", i), &git.CommitOptions{
			Author: &object.Signature{
				Name:  "Author",
				Email: "author@example.com",
				When:  time.Now(),
			},
		})
		assert.Nil(err)

		commitIds = append(commitIds, commitId)
	}

	input := strings.NewReader(fmt.Sprintf(
		"%040d %s refs/heads/branch-1\n%040d %s refs/heads/branch-2\n",
		0, commitIds[1].String(),
		0, commitIds[3].String(),
	))

	payload, err := repo.ParseEventPayload(events.PushEvent, input)
	assert.Nil(err)

	expected := events.PushPayload{
		Repository: repo.Name,
		Commits: []events.PushPayloadCommit{
			{
				Id:      commitIds[0].String(),
				Message: "Commit 1",
				Target: events.PushPayloadCommitTarget{
					Branch: "branch-1",
				},
			},
			{
				Id:      commitIds[1].String(),
				Message: "Commit 2",
				Target: events.PushPayloadCommitTarget{
					Branch: "branch-1",
				},
			},
			{
				Id:      commitIds[2].String(),
				Message: "Commit 3",
				Target: events.PushPayloadCommitTarget{
					Branch: "branch-2",
				},
			},
			{
				Id:      commitIds[3].String(),
				Message: "Commit 4",
				Target: events.PushPayloadCommitTarget{
					Branch: "branch-2",
				},
			},
		},
	}

	assert.Equal(expected, payload)
}
