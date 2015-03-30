package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

const (
	routesTestInvalidId = "a30e45f1dadd788d9c0f8b0fe829329cce0e31d1"
	routesTestUser      = "myuser"
	routesTestPass      = "mypass"
	routesTestRepo      = "myrepo"
)

var (
	routesTestConfigPath string
	routesTestRepoPath   string
	routesTestFileId     string
	routesTestCommitId   string
	routesTestBranchName string
)

func routesTestSetup(t *testing.T) {
	path, err := ioutil.TempFile("", "rb-gateway-config")
	checkFatal(t, err)

	repo, git2goRepo := createTestRepo(t)
	commitId, fileId := seedTestRepo(t, git2goRepo)
	branch, _ := branchTestRepo(t, git2goRepo)

	content := "{" +
		"\"port\": 8888," +
		"\"username\": \"" + routesTestUser + "\"," +
		"\"password\": \"" + routesTestPass + "\"," +
		"\"repositories\": " +
		"[{\"name\": \"" + routesTestRepo + "\"," +
		"\"path\": \"" + repo.Path + "\"," +
		"\"scm\": \"git\"}]" +
		"}"

	err = ioutil.WriteFile(path.Name(), []byte(content), 0644)
	checkFatal(t, err)

	routesTestConfigPath = path.Name()
	routesTestRepoPath = repo.GetPath()
	routesTestFileId = fileId
	routesTestCommitId = commitId
	routesTestBranchName, err = branch.Name()
	checkFatal(t, err)

	LoadConfig(routesTestConfigPath)
}

func routesTestTeardown() {
	os.RemoveAll(routesTestRepoPath)
	os.RemoveAll(routesTestConfigPath)
}

func testRoute(url, method string, t *testing.T) *httptest.ResponseRecorder {
	mux := Route()

	response := httptest.NewRecorder()

	request, err := http.NewRequest(method, url, nil)
	checkFatal(t, err)

	request.SetBasicAuth(routesTestUser, routesTestPass)
	session, err := CreateSession(request)
	checkFatal(t, err)

	request.Header.Set(authPrivateToken, session.PrivateToken)

	mux.ServeHTTP(response, request)

	return response
}

func testStatusCode(expected int, response *httptest.ResponseRecorder, t *testing.T) {
	if expected != response.Code {
		t.Errorf("Expected status '%d', got '%d'", expected, response.Code)
	}
}

func TestGetFileAPI(t *testing.T) {
	routesTestSetup(t)

	// Testing valid file id
	url := fmt.Sprintf("/repos/%s/file/%s", routesTestRepo, routesTestFileId)
	testStatusCode(http.StatusOK, testRoute(url, "GET", t), t)

	// Testing invalid file id
	url = fmt.Sprintf("/repos/%s/file/%s", routesTestRepo, routesTestInvalidId)
	testStatusCode(http.StatusBadRequest, testRoute(url, "GET", t), t)

	routesTestTeardown()
}

func TestFileExistsAPI(t *testing.T) {
	routesTestSetup(t)

	// Testing valid file
	url := fmt.Sprintf("/repos/%s/file/%s", routesTestRepo, routesTestFileId)
	testStatusCode(http.StatusOK, testRoute(url, "HEAD", t), t)

	// Testing invalid file id
	url = fmt.Sprintf("/repos/%s/file/%s", routesTestRepo, routesTestInvalidId)
	testStatusCode(http.StatusNotFound, testRoute(url, "HEAD", t), t)

	// Testing file id with bad formatroute.
	url = fmt.Sprintf("/repos/%s/file/%s", routesTestRepo, "bad-id")
	testStatusCode(http.StatusBadRequest, testRoute(url, "HEAD", t), t)

	routesTestTeardown()
}

func TestGetFileByCommitAPI(t *testing.T) {
	routesTestSetup(t)

	// Testing valid commit and file path
	url := fmt.Sprintf("/repos/%s/commits/%s/path/%s", routesTestRepo, routesTestCommitId, repoFile)
	testStatusCode(http.StatusOK, testRoute(url, "GET", t), t)

	// Testing invalid file path
	url = fmt.Sprintf("/repos/%s/commits/%s/path/%s", routesTestRepo, routesTestCommitId, "bad-file-path")
	testStatusCode(http.StatusBadRequest, testRoute(url, "GET", t), t)

	// Testing invalid commit
	url = fmt.Sprintf("/repos/%s/commits/%s/path/%s", routesTestRepo, "bad-commit", repoFile)
	testStatusCode(http.StatusBadRequest, testRoute(url, "GET", t), t)

	routesTestTeardown()
}

func TestFileExistsByCommitAPI(t *testing.T) {
	routesTestSetup(t)

	// Testing valid commit and file path
	url := fmt.Sprintf("/repos/%s/commits/%s/path/%s", routesTestRepo, routesTestCommitId, repoFile)
	testStatusCode(http.StatusOK, testRoute(url, "HEAD", t), t)

	// Testing invalid file path
	url = fmt.Sprintf("/repos/%s/commits/%s/path/%s", routesTestRepo, routesTestCommitId, "bad-file-path")
	testStatusCode(http.StatusBadRequest, testRoute(url, "HEAD", t), t)

	routesTestTeardown()
}

func TestGetPathAPI(t *testing.T) {
	routesTestSetup(t)

	url := fmt.Sprintf("/repos/%s/path", routesTestRepo)
	response := testRoute(url, "GET", t)

	testStatusCode(http.StatusOK, response, t)

	if string(response.Body.Bytes()) != routesTestRepoPath+"/info/refs" {
		t.Errorf("Expected repo path '%s', got '%s'",
			routesTestRepoPath+"/info/refs", string(response.Body.Bytes()))
	}

	routesTestTeardown()
}

func TestGetBranchesAPI(t *testing.T) {
	routesTestSetup(t)

	url := fmt.Sprintf("/repos/%s/branches", routesTestRepo)
	testStatusCode(http.StatusOK, testRoute(url, "GET", t), t)

	routesTestTeardown()
}

func TestGetCommitsAPI(t *testing.T) {
	routesTestSetup(t)

	url := fmt.Sprintf("/repos/%s/branches/%s/commits", routesTestRepo, routesTestBranchName)
	testStatusCode(http.StatusOK, testRoute(url, "GET", t), t)

	routesTestTeardown()
}

func TestGetCommitAPI(t *testing.T) {
	routesTestSetup(t)

	// Testing valid commit id
	url := fmt.Sprintf("/repos/%s/commits/%s", routesTestRepo, routesTestCommitId)
	testStatusCode(http.StatusOK, testRoute(url, "GET", t), t)

	// Testing invalid commit id
	url = fmt.Sprintf("/repos/%s/commits/%s", routesTestRepo, routesTestInvalidId)
	testStatusCode(http.StatusNotFound, testRoute(url, "GET", t), t)

	// Testing invalid commit id with bad format
	url = fmt.Sprintf("/repos/%s/commits/%s", routesTestRepo, "bad-commit-format")
	testStatusCode(http.StatusBadRequest, testRoute(url, "GET", t), t)

	routesTestTeardown()
}

func TestGetSessionAPI(t *testing.T) {
	routesTestSetup(t)

	request, err := http.NewRequest("GET", "/session", nil)
	checkFatal(t, err)

	request.SetBasicAuth(GetUsername(), GetPassword())

	response := httptest.NewRecorder()
	Route().ServeHTTP(response, request)

	var session Session

	checkFatal(t, json.Unmarshal(response.Body.Bytes(), &session))

	if session.PrivateToken == "" {
		t.Error("Private token is not provided in the response")
	}

	routesTestTeardown()
}
