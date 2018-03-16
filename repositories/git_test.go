package repositories_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/reviewboard/rb-gateway/helpers"
)

func TestGetName(t *testing.T) {
	repo, rawRepo := helpers.CreateTestRepo(t, "repo")
	defer helpers.CleanupRepository(t, rawRepo)

	name := repo.GetName()
	assert.Equal(t, name, "repo")
}

func TestGetPath(t *testing.T) {
	repo, rawRepo := helpers.CreateTestRepo(t, "repo")
	defer helpers.CleanupRepository(t, rawRepo)

	path := repo.GetPath()
	assert.Equal(t, fmt.Sprintf("%s/", path), rawRepo.Workdir())
}

func TestGetFile(t *testing.T) {
	repo, rawRepo := helpers.CreateTestRepo(t, "repo")
	defer helpers.CleanupRepository(t, rawRepo)

	helpers.SeedTestRepo(t, rawRepo)
	fileId := helpers.GetRepositoryFileId(t, rawRepo, "README").String()

	fileContent, err := repo.GetFile(fileId)
	assert.Nil(t, err)

	expectedContent := helpers.GetRepoFiles()["README"]
	assert.Equal(t, string(expectedContent), string(fileContent))
}

func TestGetFileByCommit(t *testing.T) {
	repo, rawRepo := helpers.CreateTestRepo(t, "repo")
	defer helpers.CleanupRepository(t, rawRepo)

	commitId := helpers.SeedTestRepo(t, rawRepo).String()

	fileContent, err := repo.GetFileByCommit(commitId, "README")
	assert.Nil(t, err)

	expectedContent := helpers.GetRepoFiles()["README"]
	assert.Equal(t, string(expectedContent), string(fileContent))
}

func TestFileExists(t *testing.T) {
	repo, rawRepo := helpers.CreateTestRepo(t, "repo")
	defer helpers.CleanupRepository(t, rawRepo)

	helpers.SeedTestRepo(t, rawRepo)
	fileId := helpers.GetRepositoryFileId(t, rawRepo, "README").String()

	exists, err := repo.FileExists(fileId)
	assert.Nil(t, err)
	assert.True(t, exists)
}

func TestFileExistsByCommit(t *testing.T) {
	repo, rawRepo := helpers.CreateTestRepo(t, "repo")
	defer helpers.CleanupRepository(t, rawRepo)

	commitId := helpers.SeedTestRepo(t, rawRepo).String()

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
	defer helpers.CleanupRepository(t, rawRepo)

	helpers.SeedTestRepo(t, rawRepo)
	branch := helpers.CreateTestBranch(t, rawRepo)
	branchName, err := branch.Name()
	assert.Nil(t, err)

	result, err := repo.GetBranches()
	assert.Nil(t, err)

	var branches []Branch
	err = json.Unmarshal(result, &branches)
	assert.Nil(t, err)

	assert.Equal(t, len(branches), 2)

	for i := range branches {
		assert.Contains(t, []string{"master", branchName}, branches[i].Name)

		if branches[i].Name == branchName {
			assert.Equal(t, branches[i].Id, branch.Target().String())
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
	defer helpers.CleanupRepository(t, rawRepo)

	commitId := helpers.SeedTestRepo(t, rawRepo)
	commit, err := rawRepo.LookupCommit(commitId)
	assert.Nil(t, err)

	branch := helpers.CreateTestBranch(t, rawRepo)
	branchCommit, err := rawRepo.LookupCommit(branch.Target())
	assert.Nil(t, err)

	branchName, err := branch.Name()
	assert.Nil(t, err)

	// Testing GetCommits without a starting commit.
	result, err := repo.GetCommits(branchName, "")
	assert.Nil(t, err)

	var commits []Commit
	err = json.Unmarshal(result, &commits)
	assert.Nil(t, err)

	assert.Equal(t, len(commits), 2)

	assert.Equal(t, commits[0].Message, branchCommit.Message())
	assert.Equal(t, commits[0].Id, branch.Target().String())
	assert.Equal(t, commits[0].ParentId, commitId.String())
	assert.Equal(t, commits[1].Message, commit.Message())
	assert.Equal(t, commits[1].Id, commitId.String())
	assert.Equal(t, commits[1].ParentId, "")

	// Testing GetCommits with a starting commit.
	result, err = repo.GetCommits(branchName, commitId.String())
	assert.Nil(t, err)

	err = json.Unmarshal(result, &commits)
	assert.Nil(t, err)

	assert.Equal(t, len(commits), 1)

	assert.Equal(t, commits[0].Message, commit.Message())
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
	defer helpers.CleanupRepository(t, rawRepo)

	commitId := helpers.SeedTestRepo(t, rawRepo)
	commit, err := rawRepo.LookupCommit(commitId)
	assert.Nil(t, err)

	response, err := repo.GetCommit(commitId.String())
	assert.Nil(t, err)

	fileIds := make(map[string]string)
	files := helpers.GetRepoFiles()
	for filename := range files {
		fileIds[filename] = helpers.GetRepositoryFileId(t, rawRepo, filename).String()
	}

	var result Commit
	err = json.Unmarshal(response, &result)
	assert.Nil(t, err)

	assert.Equal(t, result.Message, commit.Message())

	diff := fmt.Sprintf(`diff --git a/COPYING b/COPYING
new file mode 100644
index 0000000000000000000000000000000000000000..%s
--- /dev/null
+++ b/COPYING
@@ -0,0 +1 @@
+%sdiff --git a/README b/README
new file mode 100644
index 0000000000000000000000000000000000000000..%s
--- /dev/null
+++ b/README
@@ -0,0 +1 @@
+%s`, fileIds["COPYING"], string(files["COPYING"]), fileIds["README"], string(files["README"]))

	assert.Equal(t, diff, result.Diff)
}
