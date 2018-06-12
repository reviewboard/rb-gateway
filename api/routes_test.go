package api_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/reviewboard/rb-gateway/api"
	"github.com/reviewboard/rb-gateway/api/tokens"
	"github.com/reviewboard/rb-gateway/config"
	"github.com/reviewboard/rb-gateway/helpers"
)

const (
	routesTestInvalidId = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
)

func testRoute(t *testing.T, cfg config.Config, url, method string) *httptest.ResponseRecorder {
	t.Helper()
	assert := assert.New(t)

	handler, err := api.New(cfg)
	assert.Nil(err)

	tokenStore := handler.GetTokenStore()
	token, err := (*tokenStore).New()

	response := httptest.NewRecorder()

	request, err := http.NewRequest(method, url, nil)
	assert.Nil(err)

	request.Header.Set(api.PrivateTokenHeader, *token)

	handler.ServeHTTP(response, request)

	return response
}

func TestGetFileAPI(t *testing.T) {
	assert := assert.New(t)

	repo, rawRepo := helpers.CreateGitRepo(t, "repo")
	defer helpers.CleanupRepository(t, repo.Path)

	helpers.SeedGitRepo(t, repo, rawRepo)
	helpers.CreateGitBranch(t, repo, rawRepo)

	cfg := helpers.CreateTestConfig(t, repo)
	helpers.CreateTestHtpasswd(t, "username", "password", &cfg)

	fileId := helpers.GetRepositoryFileId(t, rawRepo, "README").String()

	// Testing valid file id
	url := fmt.Sprintf("/repos/%s/file/%s", repo.Name, fileId)
	assert.Equal(http.StatusOK, testRoute(t, cfg, url, "GET").Code)

	// Testing invalid file id
	url = fmt.Sprintf("/repos/%s/file/%s", repo.Name, routesTestInvalidId)
	assert.Equal(http.StatusNotFound, testRoute(t, cfg, url, "GET").Code)

}

func TestFileExistsAPI(t *testing.T) {
	assert := assert.New(t)

	repo, rawRepo := helpers.CreateGitRepo(t, "repo")
	defer helpers.CleanupRepository(t, repo.Path)

	helpers.SeedGitRepo(t, repo, rawRepo)
	helpers.CreateGitBranch(t, repo, rawRepo)

	cfg := helpers.CreateTestConfig(t, repo)
	helpers.CreateTestHtpasswd(t, "username", "password", &cfg)

	fileId := helpers.GetRepositoryFileId(t, rawRepo, "README").String()

	// Testing valid file
	url := fmt.Sprintf("/repos/%s/file/%s", "repo", fileId)
	assert.Equal(http.StatusOK, testRoute(t, cfg, url, "HEAD").Code)

	// Testing invalid file id
	url = fmt.Sprintf("/repos/%s/file/%s", "repo", routesTestInvalidId)
	assert.Equal(http.StatusNotFound, testRoute(t, cfg, url, "HEAD").Code)

}

func testGetFileByCommitAPI(t *testing.T) {
	assert := assert.New(t)

	repo, rawRepo := helpers.CreateGitRepo(t, "repo")
	defer helpers.CleanupRepository(t, repo.Path)

	helpers.SeedGitRepo(t, repo, rawRepo)
	helpers.CreateGitBranch(t, repo, rawRepo)

	cfg := helpers.CreateTestConfig(t, repo)
	helpers.CreateTestHtpasswd(t, "username", "password", &cfg)

	head := helpers.GetRepoHead(t, rawRepo).String()

	// Testing valid commit and file path
	url := fmt.Sprintf("/repos/%s/commits/%s/path/%s", "repo", head, "README")
	assert.Equal(http.StatusOK, testRoute(t, cfg, url, "GET").Code)

	// Testing invalid file path
	url = fmt.Sprintf("/repos/%s/commits/%s/path/%s", "repo", head, "bad-file-path")
	assert.Equal(http.StatusBadRequest, testRoute(t, cfg, url, "GET").Code)

	// Testing invalid commit
	url = fmt.Sprintf("/repos/%s/commits/%s/path/%s", "repo", "bad-commit", "README")
	assert.Equal(http.StatusBadRequest, testRoute(t, cfg, url, "GET"))
}

func TestFileExistsByCommitAPI(t *testing.T) {
	assert := assert.New(t)

	repo, rawRepo := helpers.CreateGitRepo(t, "repo")
	defer helpers.CleanupRepository(t, repo.Path)

	helpers.SeedGitRepo(t, repo, rawRepo)
	helpers.CreateGitBranch(t, repo, rawRepo)

	cfg := helpers.CreateTestConfig(t, repo)
	helpers.CreateTestHtpasswd(t, "username", "password", &cfg)

	head := helpers.GetRepoHead(t, rawRepo).String()

	// Testing valid commit and file path
	url := fmt.Sprintf("/repos/%s/commits/%s/path/%s", "repo", head, "README")
	assert.Equal(http.StatusOK, testRoute(t, cfg, url, "HEAD").Code)

	// Testing invalid file path
	url = fmt.Sprintf("/repos/%s/commits/%s/path/%s", "repo", head, "bad-file-path")
	assert.Equal(http.StatusBadRequest, testRoute(t, cfg, url, "HEAD").Code)
}

func TestGetBranchesAPI(t *testing.T) {
	repo, rawRepo := helpers.CreateGitRepo(t, "repo")
	defer helpers.CleanupRepository(t, repo.Path)

	helpers.SeedGitRepo(t, repo, rawRepo)
	helpers.CreateGitBranch(t, repo, rawRepo)

	cfg := helpers.CreateTestConfig(t, repo)
	helpers.CreateTestHtpasswd(t, "username", "password", &cfg)

	url := fmt.Sprintf("/repos/%s/branches", "repo")
	assert.Equal(t, http.StatusOK, testRoute(t, cfg, url, "GET").Code)
}

func TestGetCommitsAPI(t *testing.T) {
	repo, rawRepo := helpers.CreateGitRepo(t, "repo")
	defer helpers.CleanupRepository(t, repo.Path)

	helpers.SeedGitRepo(t, repo, rawRepo)
	branch := helpers.CreateGitBranch(t, repo, rawRepo)
	branchName := branch.Name().Short()

	cfg := helpers.CreateTestConfig(t, repo)
	helpers.CreateTestHtpasswd(t, "username", "password", &cfg)

	url := fmt.Sprintf("/repos/%s/branches/%s/commits", "repo", branchName)
	assert.Equal(t, http.StatusOK, testRoute(t, cfg, url, "GET").Code)
}

func TestGetCommitAPI(t *testing.T) {
	assert := assert.New(t)
	repo, rawRepo := helpers.CreateGitRepo(t, "repo")
	defer helpers.CleanupRepository(t, repo.Path)

	helpers.SeedGitRepo(t, repo, rawRepo)
	helpers.CreateGitBranch(t, repo, rawRepo)

	cfg := helpers.CreateTestConfig(t, repo)
	helpers.CreateTestHtpasswd(t, "username", "password", &cfg)

	head := helpers.GetRepoHead(t, rawRepo).String()

	// Testing valid commit id
	url := fmt.Sprintf("/repos/%s/commits/%s", "repo", head)
	assert.Equal(http.StatusOK, testRoute(t, cfg, url, "GET").Code)

	// Testing invalid commit id
	url = fmt.Sprintf("/repos/%s/commits/%s", "repo", routesTestInvalidId)
	assert.Equal(http.StatusNotFound, testRoute(t, cfg, url, "GET").Code)

}

func TestGetSessionAPI(t *testing.T) {
	assert := assert.New(t)

	repo, rawRepo := helpers.CreateGitRepo(t, "repo")
	defer helpers.CleanupRepository(t, repo.Path)

	helpers.SeedGitRepo(t, repo, rawRepo)
	helpers.CreateGitBranch(t, repo, rawRepo)

	cfg := helpers.CreateTestConfig(t, repo)
	helpers.CreateTestHtpasswd(t, "username", "password", &cfg)
	defer helpers.CleanupConfig(t, cfg)

	request, err := http.NewRequest("GET", "/session", nil)
	assert.Nil(err)

	request.SetBasicAuth("username", "password")

	response := httptest.NewRecorder()
	handler, err := api.New(cfg)
	assert.Nil(err)

	handler.ServeHTTP(response, request)

	var session api.Session

	assert.Nil(json.Unmarshal(response.Body.Bytes(), &session))

	assert.Equal(len(session.PrivateToken), tokens.TokenSize, "Private token is not provided in the response")
}
