package repositories_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-ini/ini"
	"github.com/stretchr/testify/assert"

	"github.com/reviewboard/rb-gateway/helpers"
	"github.com/reviewboard/rb-gateway/repositories/events"
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

func TestHgGetBranchesNoBookmarks(t *testing.T) {
	assert := assert.New(t)

	repo, client := helpers.CreateHgRepo(t, "hg-repo")
	defer helpers.CleanupHgRepo(t, client)

	helpers.SeedHgRepo(t, repo, client)

	branches, err := repo.GetBranches()
	assert.Nil(err)

	assert.Equal(1, len(branches))
	assert.Equal("default", branches[0].Name)

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

func TestParsePushEvent(t *testing.T) {
	assert := assert.New(t)

	repo, client := helpers.CreateHgRepo(t, "hg-repo")
	defer helpers.CleanupHgRepo(t, client)
	fmt.Println("====== CWD: ", repo.Path)

	files := []map[string][]byte{
		{"foo": []byte("foo")},
		{"bar": []byte("bar")},
		{"baz": []byte("baz")},
		{"qux": []byte("qux")},
	}

	nodes := make([]string, 0, 5)

	for i, filesToAdd := range files[:3] {
		helpers.CreateAndAddFilesHg(t, repo.Path, client, filesToAdd)
		commitId := helpers.CommitHg(t, client, fmt.Sprintf("Commit %d", i), helpers.DefaultAuthor)
		nodes = append(nodes, commitId)
	}

	tagId := helpers.CreateHgTag(t, client, nodes[2], "commit-2", "Tag commit-2", helpers.DefaultAuthor)
	nodes = append(nodes, tagId)

	for i, filesToAdd := range files[3:] {
		helpers.CreateAndAddFilesHg(t, repo.Path, client, filesToAdd)
		commitId := helpers.CommitHg(t, client, fmt.Sprintf("Commit %d", 3+i), helpers.DefaultAuthor)
		nodes = append(nodes, commitId)
	}

	helpers.CreateHgBookmark(t, client, nodes[4], "bookmark-1")

	env := map[string]string{
		"HG_NODE":      nodes[1],
		"HG_NODE_LAST": nodes[4],
	}

	var payload events.Payload
	var err error

	helpers.WithEnv(t, env, func() {
		payload, err = repo.ParseEventPayload(events.PushEvent, nil)
	})

	assert.Nil(err)

	expected := events.PushPayload{
		Repository: repo.Name,
		Commits: []events.PushPayloadCommit{
			{
				Id:      nodes[1],
				Message: "Commit 1",
				Target: events.PushPayloadCommitTarget{
					Branch: "default",
				},
			},
			{
				Id:      nodes[2],
				Message: "Commit 2",
				Target: events.PushPayloadCommitTarget{
					Branch: "default",
					Tags:   []string{"commit-2"},
				},
			},
			{
				Id:      nodes[3],
				Message: "Tag commit-2",
				Target: events.PushPayloadCommitTarget{
					Branch: "default",
				},
			},
			{
				Id:      nodes[4],
				Message: "Commit 3",
				Target: events.PushPayloadCommitTarget{
					Branch:    "default",
					Bookmarks: []string{"bookmark-1"},
					Tags:      []string{"tip"},
				},
			},
		},
	}

	assert.Equal(expected, payload)
}

func TestInstallHgHooks(t *testing.T) {
	assert := assert.New(t)

	repo, client := helpers.CreateHgRepo(t, "hg-repo")
	defer helpers.CleanupHgRepo(t, client)

	repo.InstallHooks("/tmp/config.json", false)

	hgrc, err := ini.Load(filepath.Join(repo.Path, ".hg", "hgrc"))
	assert.Nil(err)

	exePath, err := filepath.Abs(os.Args[0])
	assert.Nil(err)

	assert.Equal(
		fmt.Sprintf("%s --config /tmp/config.json trigger-webhooks hg-repo push", exePath),
		hgrc.Section("hooks").Key("changegroup.rbgateway").String(),
	)
}

func TestInstallHgHooksQuoted(t *testing.T) {
	assert := assert.New(t)

	repo, client := helpers.CreateHgRepo(t, "hg-repo with a space")
	defer helpers.CleanupHgRepo(t, client)

	repo.InstallHooks("/tmp/config with a space.json", false)

	hgrc, err := ini.Load(filepath.Join(repo.Path, ".hg", "hgrc"))
	assert.Nil(err)

	exePath, err := filepath.Abs(os.Args[0])
	assert.Nil(err)

	assert.Equal(
		fmt.Sprintf("%s --config '/tmp/config with a space.json' trigger-webhooks 'hg-repo with a space' push", exePath),
		hgrc.Section("hooks").Key("changegroup.rbgateway").String(),
	)
}

func TestInstallHgHooksForce(t *testing.T) {
	assert := assert.New(t)

	repo, client := helpers.CreateHgRepo(t, "repo")
	defer helpers.CleanupHgRepo(t, client)

	assert.Nil(repo.InstallHooks("/tmp/config1", false))
	assert.Nil(repo.InstallHooks("/tmp/config2", true))
	assert.Nil(repo.InstallHooks("/tmp/config3", false))

	hgrc, err := ini.Load(filepath.Join(repo.Path, ".hg", "hgrc"))
	assert.Nil(err)

	exePath, err := filepath.Abs(os.Args[0])
	assert.Nil(err)

	assert.Equal(
		fmt.Sprintf("%s --config /tmp/config2 trigger-webhooks repo push", exePath),
		hgrc.Section("hooks").Key("changegroup.rbgateway").String(),
	)
}
