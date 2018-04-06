package repositories_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/reviewboard/rb-gateway/helpers"
)

func TestGetFile(t *testing.T) {
	repo, rawRepo := helpers.CreateTestRepo(t, "repo")
	defer helpers.CleanupRepository(t, repo.Path)

	helpers.SeedTestRepo(t, repo, rawRepo)
	fileId := helpers.GetRepositoryFileId(t, rawRepo, "README").String()

	fileContent, err := repo.GetFile(fileId)
	assert.Nil(t, err)

	expectedContent := helpers.GetRepoFiles()["README"]
	assert.Equal(t, string(expectedContent), string(fileContent))
}

func TestGetFileByCommit(t *testing.T) {
	repo, rawRepo := helpers.CreateTestRepo(t, "repo")
	defer helpers.CleanupRepository(t, repo.Path)

	commitId := helpers.SeedTestRepo(t, repo, rawRepo).String()

	fileContent, err := repo.GetFileByCommit(commitId, "README")
	assert.Nil(t, err)

	expectedContent := helpers.GetRepoFiles()["README"]
	assert.Equal(t, string(expectedContent), string(fileContent))
}

func TestFileExists(t *testing.T) {
	repo, rawRepo := helpers.CreateTestRepo(t, "repo")
	defer helpers.CleanupRepository(t, repo.Path)

	helpers.SeedTestRepo(t, repo, rawRepo)
	fileId := helpers.GetRepositoryFileId(t, rawRepo, "README").String()

	exists, err := repo.FileExists(fileId)
	assert.Nil(t, err)
	assert.True(t, exists)
}

func TestFileExistsByCommit(t *testing.T) {
	repo, rawRepo := helpers.CreateTestRepo(t, "repo")
	defer helpers.CleanupRepository(t, repo.Path)

	commitId := helpers.SeedTestRepo(t, repo, rawRepo).String()

	exists, err := repo.FileExistsByCommit(commitId, "README")
	assert.Nil(t, err)
	assert.True(t, exists)
}

func TestGetBranches(t *testing.T) {
	type Branch struct {
		Name string `json:"name"`
		Id   string `json:"id"`
	}

	repo, rawRepo := helpers.CreateTestRepo(t, "repo")
	defer helpers.CleanupRepository(t, repo.Path)

	helpers.SeedTestRepo(t, repo, rawRepo)
	branch := helpers.CreateTestBranch(t, repo, rawRepo)
	branchName := branch.Name().Short()

	result, err := repo.GetBranches()
	assert.Nil(t, err)

	var branches []Branch
	err = json.Unmarshal(result, &branches)
	assert.Nil(t, err)

	assert.Equal(t, len(branches), 2)

	for i := range branches {
		assert.Contains(t, []string{"master", branchName}, branches[i].Name)

		if branches[i].Name == branchName {
			assert.Equal(t, branches[i].Id, branch.Hash().String())
		}
	}
}

func TestGetCommits(t *testing.T) {
	type Commit struct {
		Author   string `json:"author"`
		Id       string `json:"id"`
		Date     string `json:"date"`
		Message  string `json:"message"`
		ParentId string `json:"parent_id"`
	}

	repo, rawRepo := helpers.CreateTestRepo(t, "repo")
	defer helpers.CleanupRepository(t, repo.Path)

	commitId := helpers.SeedTestRepo(t, repo, rawRepo)
	commit, err := rawRepo.CommitObject(commitId)
	assert.Nil(t, err)

	branch := helpers.CreateTestBranch(t, repo, rawRepo)
	branchCommit, err := rawRepo.CommitObject(branch.Hash())
	assert.Nil(t, err)

	branchName := branch.Name().Short()
	assert.Nil(t, err)

	// Testing GetCommits without a starting commit.
	result, err := repo.GetCommits(branchName, "")
	assert.Nil(t, err)

	var commits []Commit
	err = json.Unmarshal(result, &commits)
	assert.Nil(t, err)

	assert.Equal(t, len(commits), 2)

	assert.Equal(t, commits[0].Message, branchCommit.Message)
	assert.Equal(t, commits[0].Id, branch.Hash().String())
	assert.Equal(t, commits[0].ParentId, commitId.String())
	assert.Equal(t, commits[1].Message, commit.Message)
	assert.Equal(t, commits[1].Id, commitId.String())
	assert.Equal(t, commits[1].ParentId, "")

	// Testing GetCommits with a starting commit.
	result, err = repo.GetCommits(branchName, commitId.String())
	assert.Nil(t, err)

	err = json.Unmarshal(result, &commits)
	assert.Nil(t, err)

	assert.Equal(t, len(commits), 1)

	assert.Equal(t, commits[0].Message, commit.Message)
	assert.Equal(t, commits[0].Id, commitId.String())
}

func TestGetCommit(t *testing.T) {
	type Commit struct {
		Author   string `json:"author"`
		Id       string `json:"id"`
		Date     string `json:"date"`
		Message  string `json:"message"`
		ParentId string `json:"parent_id"`
		Diff     string `json:"diff"`
	}

	repo, rawRepo := helpers.CreateTestRepo(t, "repo")
	defer helpers.CleanupRepository(t, repo.Path)

	helpers.SeedTestRepo(t, repo, rawRepo)

	branch := helpers.CreateTestBranch(t, repo, rawRepo)

	commit, err := rawRepo.CommitObject(branch.Hash())
	assert.Nil(t, err)

	response, err := repo.GetCommit(branch.Hash().String())
	assert.Nil(t, err)

	fileIds := make(map[string]string)
	files := helpers.GetRepoFiles()
	for filename := range files {
		fileIds[filename] = helpers.GetRepositoryFileId(t, rawRepo, filename).String()
	}

	var result Commit
	err = json.Unmarshal(response, &result)
	assert.Nil(t, err)

	assert.Equal(t, result.Message, commit.Message)

	diff := fmt.Sprintf(`diff --git a/AUTHORS b/AUTHORS
new file mode 100644
index 0000000000000000000000000000000000000000..%s
--- /dev/null
+++ b/AUTHORS
@@ -0,0 +1 @@
+%s`, fileIds["AUTHORS"], string(files["AUTHORS"]))

	assert.Equal(t, diff, result.Diff)
}
