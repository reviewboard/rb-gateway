package api_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"

	"github.com/reviewboard/rb-gateway/api"
	"github.com/reviewboard/rb-gateway/api/tokens"
	"github.com/reviewboard/rb-gateway/config"
	"github.com/reviewboard/rb-gateway/helpers"
	"github.com/reviewboard/rb-gateway/repositories"
)

const (
	routesTestInvalidId = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
)

// Make a request to the given URL with a new API instance using the given config.
func testRoute(t *testing.T, cfg *config.Config, url, method string) *httptest.ResponseRecorder {
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

// Common data for routes tests.
type routeTestSetup struct {
	repo    *repositories.GitRepository
	rawRepo *git.Repository
	config  *config.Config
	branch  *plumbing.Reference
}

// Cleanup temporary files created for the test.
func (setup *routeTestSetup) cleanup(t *testing.T) {
	helpers.CleanupRepository(t, setup.repo.Path)
	helpers.CleanupConfig(t, setup.config)
}

// Do common setup for a routes test.
func setupRoutesTest(t *testing.T) routeTestSetup {
	t.Helper()

	repo, rawRepo := helpers.CreateGitRepo(t, "repo")

	helpers.SeedGitRepo(t, repo, rawRepo)
	branch := helpers.CreateGitBranch(t, repo, rawRepo)

	cfg := helpers.CreateTestConfig(t, repo)
	helpers.CreateTestHtpasswd(t, "username", "password", &cfg)

	return routeTestSetup{
		repo:    repo,
		rawRepo: rawRepo,
		config:  &cfg,
		branch:  branch,
	}
}

func TestGetFileAPI(t *testing.T) {
	assert := assert.New(t)

	testSetup := setupRoutesTest(t)
	defer testSetup.cleanup(t)

	fileId := helpers.GetRepositoryFileId(t, testSetup.rawRepo, "README").String()

	// Testing valid file id
	url := fmt.Sprintf("/repos/%s/file/%s", testSetup.repo.Name, fileId)
	assert.Equal(http.StatusOK, testRoute(t, testSetup.config, url, "GET").Code)

	// Testing invalid file id
	url = fmt.Sprintf("/repos/%s/file/%s", testSetup.repo.Name, routesTestInvalidId)
	assert.Equal(http.StatusNotFound, testRoute(t, testSetup.config, url, "GET").Code)

}

func TestFileExistsAPI(t *testing.T) {
	assert := assert.New(t)

	testSetup := setupRoutesTest(t)
	defer testSetup.cleanup(t)

	fileId := helpers.GetRepositoryFileId(t, testSetup.rawRepo, "README").String()

	// Testing valid file
	url := fmt.Sprintf("/repos/%s/file/%s", "repo", fileId)
	assert.Equal(http.StatusOK, testRoute(t, testSetup.config, url, "HEAD").Code)

	// Testing invalid file id
	url = fmt.Sprintf("/repos/%s/file/%s", "repo", routesTestInvalidId)
	assert.Equal(http.StatusNotFound, testRoute(t, testSetup.config, url, "HEAD").Code)

}

func testGetFileByCommitAPI(t *testing.T) {
	assert := assert.New(t)

	testSetup := setupRoutesTest(t)
	defer testSetup.cleanup(t)

	head := helpers.GetRepoHead(t, testSetup.rawRepo).String()

	// Testing valid commit and file path
	url := fmt.Sprintf("/repos/%s/commits/%s/path/%s", "repo", head, "README")
	assert.Equal(http.StatusOK, testRoute(t, testSetup.config, url, "GET").Code)

	// Testing invalid file path
	url = fmt.Sprintf("/repos/%s/commits/%s/path/%s", "repo", head, "bad-file-path")
	assert.Equal(http.StatusBadRequest, testRoute(t, testSetup.config, url, "GET").Code)

	// Testing invalid commit
	url = fmt.Sprintf("/repos/%s/commits/%s/path/%s", "repo", "bad-commit", "README")
	assert.Equal(http.StatusBadRequest, testRoute(t, testSetup.config, url, "GET"))
}

func TestFileExistsByCommitAPI(t *testing.T) {
	assert := assert.New(t)

	testSetup := setupRoutesTest(t)
	defer testSetup.cleanup(t)

	head := helpers.GetRepoHead(t, testSetup.rawRepo).String()

	// Testing valid commit and file path
	url := fmt.Sprintf("/repos/%s/commits/%s/path/%s", "repo", head, "README")
	assert.Equal(http.StatusOK, testRoute(t, testSetup.config, url, "HEAD").Code)

	// Testing invalid file path
	url = fmt.Sprintf("/repos/%s/commits/%s/path/%s", "repo", head, "bad-file-path")
	assert.Equal(http.StatusBadRequest, testRoute(t, testSetup.config, url, "HEAD").Code)
}

func TestGetBranchesAPI(t *testing.T) {
	testSetup := setupRoutesTest(t)
	defer testSetup.cleanup(t)

	url := fmt.Sprintf("/repos/%s/branches", "repo")
	assert.Equal(t, http.StatusOK, testRoute(t, testSetup.config, url, "GET").Code)
}

func TestGetCommitsAPI(t *testing.T) {
	testSetup := setupRoutesTest(t)
	defer testSetup.cleanup(t)

	branchName := testSetup.branch.Name().Short()

	url := fmt.Sprintf("/repos/%s/branches/%s/commits", "repo", branchName)
	assert.Equal(t, http.StatusOK, testRoute(t, testSetup.config, url, "GET").Code)
}

func TestGetCommitAPI(t *testing.T) {
	assert := assert.New(t)

	testSetup := setupRoutesTest(t)
	defer testSetup.cleanup(t)

	head := helpers.GetRepoHead(t, testSetup.rawRepo).String()

	// Testing valid commit id
	url := fmt.Sprintf("/repos/%s/commits/%s", "repo", head)
	assert.Equal(http.StatusOK, testRoute(t, testSetup.config, url, "GET").Code)

	// Testing invalid commit id
	url = fmt.Sprintf("/repos/%s/commits/%s", "repo", routesTestInvalidId)
	assert.Equal(http.StatusNotFound, testRoute(t, testSetup.config, url, "GET").Code)

}

func TestGetSessionAPI(t *testing.T) {
	assert := assert.New(t)

	testSetup := setupRoutesTest(t)
	defer testSetup.cleanup(t)

	request, err := http.NewRequest("GET", "/session", nil)
	assert.Nil(err)

	request.SetBasicAuth("username", "password")

	response := httptest.NewRecorder()
	handler, err := api.New(testSetup.config)
	assert.Nil(err)

	handler.ServeHTTP(response, request)

	var session api.Session

	assert.Nil(json.Unmarshal(response.Body.Bytes(), &session))

	assert.Equal(len(session.PrivateToken), tokens.TokenSize, "Private token is not provided in the response")
}
