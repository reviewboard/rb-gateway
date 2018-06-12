package repositories_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/reviewboard/rb-gateway/helpers"
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
