package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"gopkg.in/libgit2/git2go.v22"
)

const (
	repoName         = "testrepo"
	repoFile         = "README"
	repoFile2        = "testfile"
	repoFileContent  = "foo\n"
	repoFileContent2 = "bar\n"
	repoBranch       = "im-a-branch"
	repoCommitMsg    = "A commit"
	repoCommitMsg2   = "Another commit"
)

func createTestRepo(t *testing.T) (*GitRepository, *git.Repository) {
	path, err := ioutil.TempDir("", "rb-gateway")
	checkFatal(t, err)
	path, err = filepath.EvalSymlinks(path)
	checkFatal(t, err)

	// Use git2go to initialize a git repository at the temporary directory.
	git2goRepo, err := git.InitRepository(path, false)
	checkFatal(t, err)

	// Write out a couple test files.
	dest := path + "/" + repoFile
	err = ioutil.WriteFile(dest, []byte(repoFileContent), 0644)
	checkFatal(t, err)
	dest = path + "/" + repoFile2
	err = ioutil.WriteFile(dest, []byte(repoFileContent2), 0644)
	checkFatal(t, err)

	repo := GitRepository{RepositoryInfo{repoName, path}}

	return &repo, git2goRepo
}

func seedTestRepo(t *testing.T, repo *git.Repository) (string, string) {
	loc, err := time.LoadLocation("UTC")
	checkFatal(t, err)
	sig := &git.Signature{
		Name:  "Easter Egg",
		Email: "rb-gateway@example.com",
		When:  time.Date(2014, 03, 12, 2, 15, 0, 0, loc),
	}

	idx, err := repo.Index()
	checkFatal(t, err)
	err = idx.AddByPath(repoFile)
	checkFatal(t, err)
	err = idx.AddByPath(repoFile2)
	checkFatal(t, err)
	treeId, err := idx.WriteTree()
	checkFatal(t, err)

	tree, err := repo.LookupTree(treeId)
	checkFatal(t, err)
	commitId, err := repo.CreateCommit("HEAD", sig, sig, repoCommitMsg, tree)
	checkFatal(t, err)
	file, err := tree.EntryByPath(repoFile)
	checkFatal(t, err)

	return commitId.String(), file.Id.String()
}

func branchTestRepo(t *testing.T, repo *git.Repository) (*git.Branch, string) {
	loc, err := time.LoadLocation("UTC")
	checkFatal(t, err)
	sig := &git.Signature{
		Name:  "Easter Egg",
		Email: "rb-gateway@example.com",
		When:  time.Date(2015, 03, 12, 2, 15, 0, 0, loc),
	}

	head, err := repo.Head()
	checkFatal(t, err)
	headCommit, err := repo.LookupCommit(head.Target())
	checkFatal(t, err)
	branch, err := repo.CreateBranch(repoBranch, headCommit, false,
		sig, "Add a branch")
	checkFatal(t, err)

	idx, err := repo.Index()
	checkFatal(t, err)
	err = idx.AddByPath(repoFile2)
	checkFatal(t, err)
	treeId, err := idx.WriteTree()
	checkFatal(t, err)

	tree, err := repo.LookupTree(treeId)
	checkFatal(t, err)
	commitTarget, err := repo.LookupCommit(branch.Target())
	checkFatal(t, err)
	commitId, err := repo.CreateCommit("refs/heads/"+repoBranch, sig, sig,
		repoCommitMsg2, tree, commitTarget)
	checkFatal(t, err)
	// Need to force another branch look up here, otherwise the old commit
	// target will be referenced.
	branch, err = repo.LookupBranch(repoBranch, git.BranchLocal)
	checkFatal(t, err)

	return branch, commitId.String()
}

func TestGetName(t *testing.T) {
	repo, git2goRepo := createTestRepo(t)
	defer os.RemoveAll(git2goRepo.Workdir())

	name := repo.GetName()
	if name != repoName {
		t.Fatalf("Expected '%s', got '%s'", name, repoName)
	}
}

func TestGetPath(t *testing.T) {
	repo, git2goRepo := createTestRepo(t)
	defer os.RemoveAll(git2goRepo.Workdir())

	path := repo.GetPath()
	if path+"/" != git2goRepo.Workdir() {
		t.Fatalf("Expected '%s', got '%s'", path+"/", git2goRepo.Workdir())
	}
}

func TestGetFile(t *testing.T) {
	repo, git2goRepo := createTestRepo(t)
	defer os.RemoveAll(git2goRepo.Workdir())

	_, fileId := seedTestRepo(t, git2goRepo)

	file, err := repo.GetFile(fileId)
	checkFatal(t, err)

	if repoFileContent != string(file) {
		t.Fatalf("Expected '%s', got '%s'", repoFileContent, string(file))
	}
}

func TestGetFileByCommit(t *testing.T) {
	repo, git2goRepo := createTestRepo(t)
	defer os.RemoveAll(git2goRepo.Workdir())

	commitId, _ := seedTestRepo(t, git2goRepo)

	file, err := repo.GetFileByCommit(commitId, repoFile)
	checkFatal(t, err)

	if repoFileContent != string(file) {
		t.Fatalf("Expected '%s', got '%s'", repoFileContent, string(file))
	}
}

func TestFileExists(t *testing.T) {
	repo, git2goRepo := createTestRepo(t)
	defer os.RemoveAll(git2goRepo.Workdir())

	_, fileId := seedTestRepo(t, git2goRepo)

	exists, err := repo.FileExists(fileId)
	checkFatal(t, err)

	if !exists {
		t.Fatal("Expected 'True', got 'False'")
	}
}

func TestFileExistsByCommit(t *testing.T) {
	repo, git2goRepo := createTestRepo(t)
	defer os.RemoveAll(git2goRepo.Workdir())

	commitId, _ := seedTestRepo(t, git2goRepo)

	exists, err := repo.FileExistsByCommit(commitId, repoFile)
	checkFatal(t, err)

	if !exists {
		t.Fatal("Expected 'True', got 'False'")
	}
}

func TestGetBranches(t *testing.T) {
	type Branch struct {
		Name string `json:"name"`
		Id   string `json:"id"`
	}

	repo, git2goRepo := createTestRepo(t)
	defer os.RemoveAll(git2goRepo.Workdir())

	seedTestRepo(t, git2goRepo)
	branch, _ := branchTestRepo(t, git2goRepo)

	result, err := repo.GetBranches()
	checkFatal(t, err)
	var branches []Branch
	err = json.Unmarshal(result, &branches)
	checkFatal(t, err)

	if len(branches) != 2 {
		t.Fatalf("Expected 2 branches, got %d", len(branches))
	}

	for i := range branches {
		if branches[i].Name != "master" && branches[i].Name != repoBranch {
			t.Fatalf("Expected branch name 'master' or '%s', got '%s'",
				repoBranch, branches[i].Name)
		}

		if branches[i].Name == repoBranch &&
			branches[i].Id != branch.Target().String() {
			t.Fatalf("Expected branch id '%s' for branch '%s', got '%s'",
				branch.Target().String(), repoBranch, branches[i].Id)
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

	repo, git2goRepo := createTestRepo(t)
	defer os.RemoveAll(git2goRepo.Workdir())

	commit, _ := seedTestRepo(t, git2goRepo)
	branch, _ := branchTestRepo(t, git2goRepo)

	branchName, err := branch.Name()
	checkFatal(t, err)

	// Testing GetCommits without a starting commit.
	result, err := repo.GetCommits(branchName, "")
	checkFatal(t, err)
	var commits []Commit
	err = json.Unmarshal(result, &commits)
	checkFatal(t, err)

	if len(commits) != 2 {
		t.Fatalf("Expected 2 commits, got %d", len(commits))
	}

	if commits[0].Message != repoCommitMsg2 {
		t.Fatalf("Expected commit message '%s', got '%s'",
			repoCommitMsg2, commits[0].Message)
	}

	if commits[0].ParentId != commit {
		t.Fatalf("Expected parent commit id '%s' for commit '%s', got '%s'",
			commit, commits[0].Id, commits[0].ParentId)
	}

	if commits[1].Message != repoCommitMsg {
		t.Fatalf("Expected commit message '%s', got '%s'",
			repoCommitMsg, commits[1].Message)
	}

	if commits[1].ParentId != "" {
		t.Fatalf("Expected no parent commit id for commit '%s', got '%s'",
			commits[1].Id, commits[1].ParentId)
	}

	// Testing GetCommits with a starting commit.
	result, err = repo.GetCommits(branchName, commit)
	checkFatal(t, err)
	err = json.Unmarshal(result, &commits)
	checkFatal(t, err)

	if len(commits) != 1 {
		t.Fatalf("Expected 1 commit, got %d", len(commits))
	}

	if commits[0].Message != repoCommitMsg {
		t.Fatalf("Expected commit message '%s', got '%s'",
			repoCommitMsg, commits[0].Message)
	}
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

	repo, git2goRepo := createTestRepo(t)
	defer os.RemoveAll(git2goRepo.Workdir())

	commitId, fileId := seedTestRepo(t, git2goRepo)

	result, err := repo.GetCommit(commitId)
	checkFatal(t, err)
	var commit Commit
	err = json.Unmarshal(result, &commit)
	checkFatal(t, err)

	if commit.Message != repoCommitMsg {
		t.Fatalf("Expected commit message '%s', got '%s'",
			repoCommitMsg, commit.Message)
	}

	diff := fmt.Sprintf("diff --git a/%s b/%s\n"+
		"new file mode 100644\n"+
		"index 0000000000000000000000000000000000000000..%s\n"+
		"--- /dev/null\n"+
		"+++ b/%s\n"+
		"@@ -0,0 +1 @@\n"+
		"+%s"+
		"diff --git a/%s b/%s\n"+
		"new file mode 100644\n"+
		"index 0000000000000000000000000000000000000000..5716ca5987cbf97d6bb54920bea6adde242d87e6\n"+
		"--- /dev/null\n"+
		"+++ b/%s\n"+
		"@@ -0,0 +1 @@\n"+
		"+%s",
		repoFile, repoFile, fileId, repoFile, repoFileContent,
		repoFile2, repoFile2, repoFile2, repoFileContent2)

	if commit.Diff != diff {
		t.Fatalf("Expected commit diff:\n%s\n, got:\n%s", diff, commit.Diff)
	}
}
