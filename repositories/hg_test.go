package repositories_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/reviewboard/rb-gateway/helpers"
)

func TestHgGetName(t *testing.T) {
	repo, client := helpers.CreateHgRepo(t, "hg-repo")
	defer helpers.CleanupHgRepo(t, client)

	assert.Equal(t, "hg-repo", repo.GetName())
}

func TestHgGetPath(t *testing.T) {
	repo, client := helpers.CreateHgRepo(t, "hg-repo")
	defer helpers.CleanupHgRepo(t, client)

	assert.Equal(t, repo.GetPath(), client.RepoRoot())
}

func TestHgGetFile(t *testing.T) {
	assert := assert.New(t)

	repo, client := helpers.CreateHgRepo(t, "hg-repo")
	defer helpers.CleanupHgRepo(t, client)

	helpers.SeedHgRepo(t, repo, client)

	fileContent := helpers.GetRepoFiles()["README"]

	result, err := repo.GetFile("README")
	assert.Nil(err)

	assert.Equal(fileContent, result[:], "Expected file contents to match.")
}

func TestHgGetFileByCommit(t *testing.T) {
	assert := assert.New(t)

	repo, client := helpers.CreateHgRepo(t, "hg-repo")
	defer helpers.CleanupHgRepo(t, client)

	commitID := helpers.SeedHgRepo(t, repo, client)

	result, err := repo.GetFileByCommit(commitID, "README")
	assert.Nil(err)

	fileContent := helpers.GetRepoFiles()["README"]

	assert.Equal(fileContent, result[:], "Expected file contents to match.")
}

func TestHgFileExists(t *testing.T) {
	assert := assert.New(t)

	repo, client := helpers.CreateHgRepo(t, "hg-repo")
	defer helpers.CleanupHgRepo(t, client)

	helpers.SeedHgRepo(t, repo, client)

	fileExists, err := repo.FileExists("AUTHORS")
	assert.Nil(err)
	assert.False(fileExists, "File 'AUTHORS' should not exist.")

	fileExists, err = repo.FileExists("README")
	assert.Nil(err)
	assert.True(fileExists, "File 'README' should exist.")
}

func TestHgFileExistsByCommit(t *testing.T) {
	assert := assert.New(t)

	repo, client := helpers.CreateHgRepo(t, "hg-repo")
	defer helpers.CleanupHgRepo(t, client)

	commitID := helpers.SeedHgRepo(t, repo, client)
	bookmarkCommitID := helpers.SeedHgBookmark(t, repo, client)

	fileExists, err := repo.FileExistsByCommit(commitID, "AUTHORS")
	assert.Nil(err)
	assert.False(fileExists, "File 'AUTHORS' should not exist at first commit.")

	fileExists, err = repo.FileExistsByCommit(bookmarkCommitID, "AUTHORS")
	assert.Nil(err)
	assert.True(fileExists, "File 'AUTHORS' should exist at bookmark.")
}

func TestHgGetBranches(t *testing.T) {
	assert := assert.New(t)

	repo, client := helpers.CreateHgRepo(t, "hg-repo")
	defer helpers.CleanupHgRepo(t, client)

	helpers.SeedHgRepo(t, repo, client)
	helpers.SeedHgBookmark(t, repo, client)

	branches, err := repo.GetBranches()
	assert.Nil(err)

	assert.Equal(2, len(branches))
	assert.Equal("default", branches[0].Name)
	assert.Equal("test-bookmark", branches[1].Name)

	for i := 0; i < len(branches); i++ {
		output, err := client.ExecCmd([]string{"log", "-r", branches[i].Name, "--template", "{node}"})
		assert.Nil(err)
		assert.Equal(string(output), branches[i].Id)
	}
}

func TestHgGetCommits(t *testing.T) {
	assert := assert.New(t)

	repo, client := helpers.CreateHgRepo(t, "hg-repo")
	defer helpers.CleanupHgRepo(t, client)

	commitID := helpers.SeedHgRepo(t, repo, client)
	bookmarkCommitID := helpers.SeedHgBookmark(t, repo, client)

	commits, err := repo.GetCommits("test-bookmark", "")
	assert.Nil(err)

	assert.Equal(2, len(commits))
	assert.Equal(bookmarkCommitID, commits[0].Id)
	assert.Equal(commitID, commits[1].Id)

	// 0x1e is the ASCII record separator. 0x1f is the field separator.
	revisions := make([]string, 0, len(commits))
	for _, commit := range commits {
		revisions = append(revisions, commit.Id)
	}

	records, err := repo.Log(client,
		[]string{
			"{author}",
			"{node}",
			"{date|rfc3339date}",
			"{desc}",
			"{parents}",
		},
		revisions,
	)

	assert.Equal(len(commits), len(records))
	for i, record := range records {
		commit := commits[i]

		assert.Equal(commit.Author, record[0])
		assert.Equal(commit.Id, record[1])
		assert.Equal(commit.Date, record[2])
	}
}

func TestHgGetCommit(t *testing.T) {
	assert := assert.New(t)

	repo, client := helpers.CreateHgRepo(t, "hg-repo")
	defer helpers.CleanupHgRepo(t, client)

	helpers.SeedHgRepo(t, repo, client)
	helpers.SeedHgBookmark(t, repo, client)

	commit, err := repo.GetCommit("1")
	assert.Nil(err)

	output, err := client.ExecCmd([]string{
		"diff", "--git", "-r", fmt.Sprintf("%s^:%s", commit.Id, commit.Id),
	})
	assert.Nil(err)

	assert.Equal(commit.Diff, string(output))
}
