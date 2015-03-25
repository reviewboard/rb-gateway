package main

import (
	"io/ioutil"
	"os"
	"strconv"
	"testing"
)

const (
	testPort = 8888
	testUser = "myuser"
	testPass = "mypass"
)

func createTestConfig(t *testing.T) (string, *GitRepository) {
	path, err := ioutil.TempFile("", "rb-gateway-config")
	checkFatal(t, err)

	repo, _ := createTestRepo(t)

	content := "{" +
		"\"port\":" + strconv.Itoa(testPort) + "," +
		"\"username\": \"" + testUser + "\"," +
		"\"password\": \"" + testPass + "\"," +
		"\"repositories\": " +
		"[{\"name\": \"testrepo\"," +
		"\"path\": \"" + repo.Path + "\"," +
		"\"scm\": \"git\"}]" +
		"}"

	err = ioutil.WriteFile(path.Name(), []byte(content), 0644)
	checkFatal(t, err)

	return path.Name(), repo
}

func TestLoadConfig(t *testing.T) {
	path, repo := createTestConfig(t)
	defer os.RemoveAll(repo.GetPath())
	defer os.RemoveAll(path)

	LoadConfig(path)

	repository := GetRepository(repoName)
	if repository == nil {
		t.Fatalf("Expected repository %s does not exist", repoName)
	}

	repoPath := repository.GetPath()
	if repoPath != repo.GetPath() {
		t.Fatalf("Expected repository path '%s', got '%s'", repoPath, path)
	}

	configPort := GetPort()
	if configPort != testPort {
		t.Fatalf("Expected port '%s', got '%s'", configPort, testPort)
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
