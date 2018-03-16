package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	git "github.com/libgit2/git2go"
	"github.com/stretchr/testify/assert"

	"github.com/reviewboard/rb-gateway/helpers"
)

const (
	routesTestInvalidId = "a30e45f1dadd788d9c0f8b0fe829329cce0e31d1"
)

var (
	routesTestRepoPath   string
	routesTestFileId     string
	routesTestCommitId   string
	routesTestFilename   string
	routesTestBranchName string
)

func routesTestSetup(t *testing.T, rawRepo *git.Repository) string {
	t.Helper()

	file, err := ioutil.TempFile("", "rb-gateway-config")
	defer file.Close()
	assert.Nil(t, err)

	commitId := helpers.SeedTestRepo(t, rawRepo)
	branch := helpers.CreateTestBranch(t, rawRepo)
	content := fmt.Sprintf(
		`{
			"port": 8888,
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
		`, testUser, testPass, testRepoName, rawRepo.Workdir())

	_, err = file.Write([]byte(content))
	assert.Nil(t, err)

	routesTestRepoPath = rawRepo.Workdir()

	commit, err := rawRepo.LookupCommit(branch.Target())
	assert.Nil(t, err)

	tree, err := commit.Tree()
	assert.Nil(t, err)

	var filename = ""
	for filename = range helpers.GetRepoFiles() {
		break
	}
	routesTestFilename = filename

	assert.NotEqual(t, filename, "")
	entry := tree.EntryByName("README")

	routesTestFileId = entry.Id.String()
	routesTestCommitId = commitId.String()
	routesTestBranchName, err = branch.Name()
	assert.Nil(t, err)

	LoadConfig(file.Name())

	return file.Name()
}

func routesTestTeardown(t *testing.T, configPath string) {
	t.Helper()

	os.RemoveAll(routesTestRepoPath)
	os.Remove(configPath)
}

func testRoute(url, method string, t *testing.T) *httptest.ResponseRecorder {
	t.Helper()

	mux := Route()

	response := httptest.NewRecorder()

	request, err := http.NewRequest(method, url, nil)
	assert.Nil(t, err)

	request.SetBasicAuth(testUser, testPass)
	session, err := CreateSession(request)
	assert.Nil(t, err)

	request.Header.Set(authPrivateToken, session.PrivateToken)

	mux.ServeHTTP(response, request)

	return response
}

func testStatusCode(expected int, response *httptest.ResponseRecorder, t *testing.T) {
	t.Helper()

	if expected != response.Code {
		t.Errorf("Expected status '%d', got '%d'", expected, response.Code)
	}
}

func TestGetFileAPI(t *testing.T) {
	_, rawRepo := helpers.CreateTestRepo(t, testRepoName)
	defer helpers.CleanupRepository(t, rawRepo)

	configPath := routesTestSetup(t, rawRepo)
	defer routesTestTeardown(t, configPath)

	// Testing valid file id
	url := fmt.Sprintf("/repos/%s/file/%s", testRepoName, routesTestFileId)
	testStatusCode(http.StatusOK, testRoute(url, "GET", t), t)

	// Testing invalid file id
	url = fmt.Sprintf("/repos/%s/file/%s", testRepoName, routesTestInvalidId)
	testStatusCode(http.StatusBadRequest, testRoute(url, "GET", t), t)

}

func TestFileExistsAPI(t *testing.T) {
	_, rawRepo := helpers.CreateTestRepo(t, testRepoName)
	defer helpers.CleanupRepository(t, rawRepo)

	configPath := routesTestSetup(t, rawRepo)
	defer routesTestTeardown(t, configPath)

	// Testing valid file
	url := fmt.Sprintf("/repos/%s/file/%s", testRepoName, routesTestFileId)
	testStatusCode(http.StatusOK, testRoute(url, "HEAD", t), t)

	// Testing invalid file id
	url = fmt.Sprintf("/repos/%s/file/%s", testRepoName, routesTestInvalidId)
	testStatusCode(http.StatusNotFound, testRoute(url, "HEAD", t), t)

	// Testing file id with bad formatroute.
	url = fmt.Sprintf("/repos/%s/file/%s", testRepoName, "bad-id")
	testStatusCode(http.StatusBadRequest, testRoute(url, "HEAD", t), t)

}

func TestGetFileByCommitAPI(t *testing.T) {
	_, rawRepo := helpers.CreateTestRepo(t, testRepoName)
	defer helpers.CleanupRepository(t, rawRepo)

	configPath := routesTestSetup(t, rawRepo)
	defer routesTestTeardown(t, configPath)

	// Testing valid commit and file path
	url := fmt.Sprintf("/repos/%s/commits/%s/path/%s", testRepoName, routesTestCommitId, routesTestFilename)
	testStatusCode(http.StatusOK, testRoute(url, "GET", t), t)

	// Testing invalid file path
	url = fmt.Sprintf("/repos/%s/commits/%s/path/%s", testRepoName, routesTestCommitId, "bad-file-path")
	testStatusCode(http.StatusBadRequest, testRoute(url, "GET", t), t)

	// Testing invalid commit
	url = fmt.Sprintf("/repos/%s/commits/%s/path/%s", testRepoName, "bad-commit", routesTestFilename)
	testStatusCode(http.StatusBadRequest, testRoute(url, "GET", t), t)
}

func TestFileExistsByCommitAPI(t *testing.T) {
	_, rawRepo := helpers.CreateTestRepo(t, testRepoName)
	defer helpers.CleanupRepository(t, rawRepo)

	configPath := routesTestSetup(t, rawRepo)
	defer routesTestTeardown(t, configPath)

	// Testing valid commit and file path
	url := fmt.Sprintf("/repos/%s/commits/%s/path/%s", testRepoName, routesTestCommitId, routesTestFilename)
	testStatusCode(http.StatusOK, testRoute(url, "HEAD", t), t)

	// Testing invalid file path
	url = fmt.Sprintf("/repos/%s/commits/%s/path/%s", testRepoName, routesTestCommitId, "bad-file-path")
	testStatusCode(http.StatusBadRequest, testRoute(url, "HEAD", t), t)
}

func TestGetPathAPI(t *testing.T) {
	_, rawRepo := helpers.CreateTestRepo(t, testRepoName)
	defer helpers.CleanupRepository(t, rawRepo)

	configPath := routesTestSetup(t, rawRepo)
	defer routesTestTeardown(t, configPath)

	url := fmt.Sprintf("/repos/%s/path", testRepoName)
	response := testRoute(url, "GET", t)

	testStatusCode(http.StatusOK, response, t)

	if string(response.Body.Bytes()) != routesTestRepoPath+"/info/refs" {
		t.Errorf("Expected repo path '%s', got '%s'",
			routesTestRepoPath+"/info/refs", string(response.Body.Bytes()))
	}
}

func TestGetBranchesAPI(t *testing.T) {
	_, rawRepo := helpers.CreateTestRepo(t, testRepoName)
	defer helpers.CleanupRepository(t, rawRepo)

	configPath := routesTestSetup(t, rawRepo)
	defer routesTestTeardown(t, configPath)

	url := fmt.Sprintf("/repos/%s/branches", testRepoName)
	testStatusCode(http.StatusOK, testRoute(url, "GET", t), t)
}

func TestGetCommitsAPI(t *testing.T) {
	_, rawRepo := helpers.CreateTestRepo(t, testRepoName)
	defer helpers.CleanupRepository(t, rawRepo)

	configPath := routesTestSetup(t, rawRepo)
	defer routesTestTeardown(t, configPath)

	url := fmt.Sprintf("/repos/%s/branches/%s/commits", testRepoName, routesTestBranchName)
	testStatusCode(http.StatusOK, testRoute(url, "GET", t), t)
}

func TestGetCommitAPI(t *testing.T) {
	_, rawRepo := helpers.CreateTestRepo(t, testRepoName)
	defer helpers.CleanupRepository(t, rawRepo)

	configPath := routesTestSetup(t, rawRepo)
	defer routesTestTeardown(t, configPath)

	// Testing valid commit id
	url := fmt.Sprintf("/repos/%s/commits/%s", testRepoName, routesTestCommitId)
	testStatusCode(http.StatusOK, testRoute(url, "GET", t), t)

	// Testing invalid commit id
	url = fmt.Sprintf("/repos/%s/commits/%s", testRepoName, routesTestInvalidId)
	testStatusCode(http.StatusNotFound, testRoute(url, "GET", t), t)

	// Testing invalid commit id with bad format
	url = fmt.Sprintf("/repos/%s/commits/%s", testRepoName, "bad-commit-format")
	testStatusCode(http.StatusBadRequest, testRoute(url, "GET", t), t)
}

func TestGetSessionAPI(t *testing.T) {
	_, rawRepo := helpers.CreateTestRepo(t, testRepoName)
	defer helpers.CleanupRepository(t, rawRepo)

	configPath := routesTestSetup(t, rawRepo)
	defer routesTestTeardown(t, configPath)

	request, err := http.NewRequest("GET", "/session", nil)
	assert.Nil(t, err)

	request.SetBasicAuth(GetUsername(), GetPassword())

	response := httptest.NewRecorder()
	Route().ServeHTTP(response, request)

	var session Session

	assert.Nil(t, json.Unmarshal(response.Body.Bytes(), &session))

	if session.PrivateToken == "" {
		t.Error("Private token is not provided in the response")
	}
}
