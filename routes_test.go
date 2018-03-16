package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/reviewboard/rb-gateway/config"
	"github.com/reviewboard/rb-gateway/helpers"
)

const (
	routesTestInvalidId = "a30e45f1dadd788d9c0f8b0fe829329cce0e31d1"
)

func testRoute(url, method string, t *testing.T) *httptest.ResponseRecorder {
	t.Helper()

	mux := Route()

	response := httptest.NewRecorder()

	request, err := http.NewRequest(method, url, nil)
	assert.Nil(t, err)

	request.SetBasicAuth(config.GetUsername(), config.GetPassword())
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
	repo, rawRepo := helpers.CreateTestRepo(t, "repo")
	defer helpers.CleanupRepository(t, rawRepo)

	helpers.SeedTestRepo(t, rawRepo)
	helpers.CreateTestBranch(t, rawRepo)

	cfg := helpers.CreateTestConfig(t, repo)
	path := helpers.WriteTestConfig(t, cfg)
	defer helpers.CleanupConfig(t, path)

	_, err := config.Load(path)
	assert.Nil(t, err)

	fileId := helpers.GetRepositoryFileId(t, rawRepo, "README").String()

	// Testing valid file id
	url := fmt.Sprintf("/repos/%s/file/%s", repo.Name, fileId)
	testStatusCode(http.StatusOK, testRoute(url, "GET", t), t)

	// Testing invalid file id
	url = fmt.Sprintf("/repos/%s/file/%s", repo.Name, routesTestInvalidId)
	testStatusCode(http.StatusBadRequest, testRoute(url, "GET", t), t)

}

func TestFileExistsAPI(t *testing.T) {
	repo, rawRepo := helpers.CreateTestRepo(t, "repo")
	defer helpers.CleanupRepository(t, rawRepo)

	helpers.SeedTestRepo(t, rawRepo)
	helpers.CreateTestBranch(t, rawRepo)

	cfg := helpers.CreateTestConfig(t, repo)
	path := helpers.WriteTestConfig(t, cfg)
	defer helpers.CleanupConfig(t, path)

	_, err := config.Load(path)
	assert.Nil(t, err)

	fileId := helpers.GetRepositoryFileId(t, rawRepo, "README").String()

	// Testing valid file
	url := fmt.Sprintf("/repos/%s/file/%s", "repo", fileId)
	testStatusCode(http.StatusOK, testRoute(url, "HEAD", t), t)

	// Testing invalid file id
	url = fmt.Sprintf("/repos/%s/file/%s", "repo", routesTestInvalidId)
	testStatusCode(http.StatusNotFound, testRoute(url, "HEAD", t), t)

	// Testing file id with bad formatroute.
	url = fmt.Sprintf("/repos/%s/file/%s", "repo", "bad-id")
	testStatusCode(http.StatusBadRequest, testRoute(url, "HEAD", t), t)

}

func testGetFileByCommitAPI(t *testing.T) {
	repo, rawRepo := helpers.CreateTestRepo(t, "repo")
	defer helpers.CleanupRepository(t, rawRepo)

	helpers.SeedTestRepo(t, rawRepo)
	helpers.CreateTestBranch(t, rawRepo)

	cfg := helpers.CreateTestConfig(t, repo)
	path := helpers.WriteTestConfig(t, cfg)
	defer helpers.CleanupConfig(t, path)

	_, err := config.Load(path)
	assert.Nil(t, err)

	head := helpers.GetRepoHead(t, rawRepo).String()

	// Testing valid commit and file path
	url := fmt.Sprintf("/repos/%s/commits/%s/path/%s", "repo", head, "README")
	testStatusCode(http.StatusOK, testRoute(url, "GET", t), t)

	// Testing invalid file path
	url = fmt.Sprintf("/repos/%s/commits/%s/path/%s", "repo", head, "bad-file-path")
	testStatusCode(http.StatusBadRequest, testRoute(url, "GET", t), t)

	// Testing invalid commit
	url = fmt.Sprintf("/repos/%s/commits/%s/path/%s", "repo", "bad-commit", "README")
	testStatusCode(http.StatusBadRequest, testRoute(url, "GET", t), t)
}

func TestFileExistsByCommitAPI(t *testing.T) {
	repo, rawRepo := helpers.CreateTestRepo(t, "repo")
	defer helpers.CleanupRepository(t, rawRepo)

	helpers.SeedTestRepo(t, rawRepo)
	helpers.CreateTestBranch(t, rawRepo)

	cfg := helpers.CreateTestConfig(t, repo)
	path := helpers.WriteTestConfig(t, cfg)
	defer helpers.CleanupConfig(t, path)

	_, err := config.Load(path)
	assert.Nil(t, err)

	head := helpers.GetRepoHead(t, rawRepo).String()

	// Testing valid commit and file path
	url := fmt.Sprintf("/repos/%s/commits/%s/path/%s", "repo", head, "README")
	testStatusCode(http.StatusOK, testRoute(url, "HEAD", t), t)

	// Testing invalid file path
	url = fmt.Sprintf("/repos/%s/commits/%s/path/%s", "repo", head, "bad-file-path")
	testStatusCode(http.StatusBadRequest, testRoute(url, "HEAD", t), t)
}

func TestGetPathAPI(t *testing.T) {
	repo, rawRepo := helpers.CreateTestRepo(t, "repo")
	defer helpers.CleanupRepository(t, rawRepo)

	helpers.SeedTestRepo(t, rawRepo)
	helpers.CreateTestBranch(t, rawRepo)

	cfg := helpers.CreateTestConfig(t, repo)
	path := helpers.WriteTestConfig(t, cfg)
	defer helpers.CleanupConfig(t, path)

	_, err := config.Load(path)
	assert.Nil(t, err)

	url := fmt.Sprintf("/repos/%s/path", "repo")
	response := testRoute(url, "GET", t)

	testStatusCode(http.StatusOK, response, t)

	assert.Equal(t, string(response.Body.Bytes()), repo.Path)
}

func TestGetBranchesAPI(t *testing.T) {
	repo, rawRepo := helpers.CreateTestRepo(t, "repo")
	defer helpers.CleanupRepository(t, rawRepo)

	helpers.SeedTestRepo(t, rawRepo)
	helpers.CreateTestBranch(t, rawRepo)

	cfg := helpers.CreateTestConfig(t, repo)
	path := helpers.WriteTestConfig(t, cfg)
	defer helpers.CleanupConfig(t, path)

	_, err := config.Load(path)
	assert.Nil(t, err)

	url := fmt.Sprintf("/repos/%s/branches", "repo")
	testStatusCode(http.StatusOK, testRoute(url, "GET", t), t)
}

func TestGetCommitsAPI(t *testing.T) {
	repo, rawRepo := helpers.CreateTestRepo(t, "repo")
	defer helpers.CleanupRepository(t, rawRepo)

	helpers.SeedTestRepo(t, rawRepo)
	branch := helpers.CreateTestBranch(t, rawRepo)

	cfg := helpers.CreateTestConfig(t, repo)
	path := helpers.WriteTestConfig(t, cfg)
	defer helpers.CleanupConfig(t, path)

	_, err := config.Load(path)
	assert.Nil(t, err)

	branchName, err := branch.Name()
	assert.Nil(t, err)

	url := fmt.Sprintf("/repos/%s/branches/%s/commits", "repo", branchName)
	testStatusCode(http.StatusOK, testRoute(url, "GET", t), t)
}

func TestGetCommitAPI(t *testing.T) {
	repo, rawRepo := helpers.CreateTestRepo(t, "repo")
	defer helpers.CleanupRepository(t, rawRepo)

	helpers.SeedTestRepo(t, rawRepo)
	helpers.CreateTestBranch(t, rawRepo)

	cfg := helpers.CreateTestConfig(t, repo)
	path := helpers.WriteTestConfig(t, cfg)
	defer helpers.CleanupConfig(t, path)

	_, err := config.Load(path)
	assert.Nil(t, err)

	head := helpers.GetRepoHead(t, rawRepo).String()

	// Testing valid commit id
	url := fmt.Sprintf("/repos/%s/commits/%s", "repo", head)
	testStatusCode(http.StatusOK, testRoute(url, "GET", t), t)

	// Testing invalid commit id
	url = fmt.Sprintf("/repos/%s/commits/%s", "repo", routesTestInvalidId)
	testStatusCode(http.StatusNotFound, testRoute(url, "GET", t), t)

	// Testing invalid commit id with bad format
	url = fmt.Sprintf("/repos/%s/commits/%s", "repo", "bad-commit-format")
	testStatusCode(http.StatusBadRequest, testRoute(url, "GET", t), t)
}

func TestGetSessionAPI(t *testing.T) {
	repo, rawRepo := helpers.CreateTestRepo(t, "repo")
	defer helpers.CleanupRepository(t, rawRepo)

	helpers.SeedTestRepo(t, rawRepo)
	helpers.CreateTestBranch(t, rawRepo)

	cfg := helpers.CreateTestConfig(t, repo)
	path := helpers.WriteTestConfig(t, cfg)
	defer helpers.CleanupConfig(t, path)

	_, err := config.Load(path)
	assert.Nil(t, err)

	request, err := http.NewRequest("GET", "/session", nil)
	assert.Nil(t, err)

	request.SetBasicAuth(config.GetUsername(), config.GetPassword())

	response := httptest.NewRecorder()
	Route().ServeHTTP(response, request)

	var session Session

	assert.Nil(t, json.Unmarshal(response.Body.Bytes(), &session))

	if session.PrivateToken == "" {
		t.Error("Private token is not provided in the response")
	}
}
