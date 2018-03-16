package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	git "github.com/libgit2/git2go"
	"github.com/stretchr/testify/assert"

	"github.com/reviewboard/rb-gateway/helpers"
	"github.com/reviewboard/rb-gateway/repositories"
)

const (
	testPort     = 8888
	testUser     = "myuser"
	testPass     = "mypass"
	testRepoName = "testrepo"
)

func createTestConfig(t *testing.T) (string, *repositories.GitRepository, *git.Repository) {
	t.Helper()

	path, err := ioutil.TempFile("", "rb-gateway-config")
	assert.Nil(t, err)

	repo, rawRepo := helpers.CreateTestRepo(t, testRepoName)

	content := fmt.Sprintf(
		`{
			"port": %d,
			"username": "%s",
			"password": "%s",
			"repositories": [
				{
					"name": "%s",
					"path": "%s",
					"scm": "git"
				}
			]
		}
		`, testPort, testUser, testPass, testRepoName, repo.Path)

	err = ioutil.WriteFile(path.Name(), []byte(content), 0644)
	assert.Nil(t, err)

	return path.Name(), repo, rawRepo
}

func TestLoadConfig(t *testing.T) {
	path, repo, rawRepo := createTestConfig(t)

	defer helpers.CleanupRepository(t, rawRepo)
	defer os.RemoveAll(path)

	LoadConfig(path)

	repository := GetRepository(testRepoName)
	if repository == nil {
		t.Fatalf("Expected repository %s does not exist", testRepoName)
	}

	repoPath := repository.GetPath()
	if repoPath != repo.GetPath() {
		t.Fatalf("Expected repository path '%s', got '%s'", repoPath, path)
	}

	configPort := GetPort()
	if configPort != testPort {
		t.Fatalf("Expected port '%d', got '%d'", configPort, testPort)
	}

	configUser := GetUsername()
	if configUser != testUser {
		t.Fatalf("Expected username '%s', got '%s'", configUser, testUser)
	}

	configPass := GetPassword()
	if configPass != testPass {
		t.Fatalf("Expected password '%s', got '%s'", configPass, testPass)
	}
}
