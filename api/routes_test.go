package api_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"

	"github.com/reviewboard/rb-gateway/api"
	"github.com/reviewboard/rb-gateway/api/tokens"
	"github.com/reviewboard/rb-gateway/config"
	"github.com/reviewboard/rb-gateway/helpers"
	"github.com/reviewboard/rb-gateway/repositories"
	"github.com/reviewboard/rb-gateway/repositories/events"
	"github.com/reviewboard/rb-gateway/repositories/hooks"
)

const (
	routesTestInvalidId = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
)

// Make a request to the given URL with a new API instance using the given config.
func testRoute(t *testing.T, cfg *config.Config, url, method string, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	assert := assert.New(t)

	handler, err := api.New(cfg)
	assert.NotNil(handler)
	assert.Nil(err)

	tokenStore := handler.GetTokenStore()
	token, err := (*tokenStore).New()
	assert.Nil(err)

	response := httptest.NewRecorder()

	request, err := http.NewRequest(method, url, nil)
	assert.Nil(err)

	request.Header.Set(api.PrivateTokenHeader, *token)

	if body != nil {
		request.Body = ioutil.NopCloser(bytes.NewReader(body))
	}

	handler.ServeHTTP(response, request)

	return response
}

// Common data for routes tests.
type routeTestSetup struct {
	api     *api.API
	server  *httptest.Server
	repo    *repositories.GitRepository
	rawRepo *git.Repository
	config  *config.Config
	branch  *plumbing.Reference
	hooks   hooks.WebhookStore
}

// Cleanup temporary files created for the test.
func (setup *routeTestSetup) cleanup(t *testing.T) {
	helpers.CleanupRepository(t, setup.repo.Path)
	helpers.CleanupConfig(t, setup.config)
}

func setupRoutesTest(t *testing.T) routeTestSetup {
	t.Helper()

	repo, rawRepo := helpers.CreateGitRepo(t, "repo")

	helpers.SeedGitRepo(t, repo, rawRepo)
	branch := helpers.CreateGitBranch(t, repo, rawRepo)

	cfg := helpers.CreateTestConfig(t, repo)
	helpers.CreateTestHtpasswd(t, "username", "password", &cfg)

	hookStore := hooks.WebhookStore{
		"test-hook-1": &hooks.Webhook{
			Id:      "test-hook-1",
			Url:     "http://example.com/1/",
			Secret:  strings.Repeat("a", 20),
			Enabled: true,
			Events:  []string{events.PushEvent},
			Repos:   []string{repo.Name},
		},
		"test-hook-2": &hooks.Webhook{
			Id:      "test-hook-2",
			Url:     "http://example.com/2/",
			Secret:  strings.Repeat("a", 20),
			Enabled: false,
			Events:  []string{events.PushEvent},
			Repos:   []string{repo.Name},
		},
	}

	helpers.WriteTestWebhookStore(t, hookStore, &cfg)

	return routeTestSetup{
		repo:    repo,
		rawRepo: rawRepo,
		config:  &cfg,
		branch:  branch,
		hooks:   hookStore,
	}
}

func TestGetFileAPI(t *testing.T) {
	assert := assert.New(t)

	testSetup := setupRoutesTest(t)
	defer testSetup.cleanup(t)

	fileId := helpers.GetRepositoryFileId(t, testSetup.rawRepo, "README").String()

	// Testing valid file id
	url := fmt.Sprintf("/repos/%s/file/%s", testSetup.repo.Name, fileId)
	assert.Equal(
		http.StatusOK,
		testRoute(t, testSetup.config, url, "GET", nil).Code,
	)

	// Testing invalid file id
	url = fmt.Sprintf("/repos/%s/file/%s", testSetup.repo.Name, routesTestInvalidId)
	assert.Equal(
		http.StatusNotFound,
		testRoute(t, testSetup.config, url, "GET", nil).Code,
	)

}

func TestFileExistsAPI(t *testing.T) {
	assert := assert.New(t)

	testSetup := setupRoutesTest(t)
	defer testSetup.cleanup(t)

	fileId := helpers.GetRepositoryFileId(t, testSetup.rawRepo, "README").String()

	// Testing valid file
	url := fmt.Sprintf("/repos/%s/file/%s", "repo", fileId)
	assert.Equal(
		http.StatusOK,
		testRoute(t, testSetup.config, url, "HEAD", nil).Code,
	)

	// Testing invalid file id
	url = fmt.Sprintf("/repos/%s/file/%s", "repo", routesTestInvalidId)
	assert.Equal(
		http.StatusNotFound,
		testRoute(t, testSetup.config, url, "HEAD", nil).Code,
	)

}

func testGetFileByCommitAPI(t *testing.T) {
	assert := assert.New(t)

	testSetup := setupRoutesTest(t)
	defer testSetup.cleanup(t)

	head := helpers.GetRepoHead(t, testSetup.rawRepo).String()

	// Testing valid commit and file path
	url := fmt.Sprintf("/repos/%s/commits/%s/path/%s", "repo", head, "README")
	assert.Equal(
		http.StatusOK,
		testRoute(t, testSetup.config, url, "GET", nil).Code,
	)

	// Testing invalid file path
	url = fmt.Sprintf("/repos/%s/commits/%s/path/%s", "repo", head, "bad-file-path")
	assert.Equal(
		http.StatusBadRequest,
		testRoute(t, testSetup.config, url, "GET", nil).Code,
	)

	// Testing invalid commit
	url = fmt.Sprintf("/repos/%s/commits/%s/path/%s", "repo", "bad-commit", "README")
	assert.Equal(
		http.StatusBadRequest,
		testRoute(t, testSetup.config, url, "GET", nil).Code,
	)
}

func TestFileExistsByCommitAPI(t *testing.T) {
	assert := assert.New(t)

	testSetup := setupRoutesTest(t)
	defer testSetup.cleanup(t)

	head := helpers.GetRepoHead(t, testSetup.rawRepo).String()

	// Testing valid commit and file path
	url := fmt.Sprintf("/repos/%s/commits/%s/path/%s", "repo", head, "README")
	assert.Equal(
		http.StatusOK,
		testRoute(t, testSetup.config, url, "HEAD", nil).Code,
	)

	// Testing invalid file path
	url = fmt.Sprintf("/repos/%s/commits/%s/path/%s", "repo", head, "bad-file-path")
	assert.Equal(
		http.StatusBadRequest,
		testRoute(t, testSetup.config, url, "HEAD", nil).Code,
	)
}

func TestGetBranchesAPI(t *testing.T) {
	testSetup := setupRoutesTest(t)
	defer testSetup.cleanup(t)

	url := fmt.Sprintf("/repos/%s/branches", "repo")
	assert.Equal(t,
		http.StatusOK,
		testRoute(t, testSetup.config, url, "GET", nil).Code,
	)
}

func TestGetCommitsAPI(t *testing.T) {
	testSetup := setupRoutesTest(t)
	defer testSetup.cleanup(t)

	branchName := testSetup.branch.Name().Short()

	url := fmt.Sprintf("/repos/%s/branches/%s/commits", "repo", branchName)
	assert.Equal(t,
		http.StatusOK,
		testRoute(t, testSetup.config, url, "GET", nil).Code,
	)
}

func TestGetCommitAPI(t *testing.T) {
	assert := assert.New(t)

	testSetup := setupRoutesTest(t)
	defer testSetup.cleanup(t)

	head := helpers.GetRepoHead(t, testSetup.rawRepo).String()

	// Testing valid commit id
	url := fmt.Sprintf("/repos/%s/commits/%s", "repo", head)
	assert.Equal(
		http.StatusOK,
		testRoute(t, testSetup.config, url, "GET", nil).Code,
	)

	// Testing invalid commit id
	url = fmt.Sprintf("/repos/%s/commits/%s", "repo", routesTestInvalidId)
	assert.Equal(
		http.StatusNotFound,
		testRoute(t, testSetup.config, url, "GET", nil).Code,
	)

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

	assert.Equal(len(session.PrivateToken), tokens.TokenSize,
		"Private token is not provided in the response")
}

func TestGetHooksAPI(t *testing.T) {
	assert := assert.New(t)

	testSetup := setupRoutesTest(t)
	defer testSetup.cleanup(t)

	rsp := testRoute(t, testSetup.config, "/webhooks", "GET", nil)
	assert.Equal(http.StatusOK, rsp.Code)

	var parsedRsp struct {
		Webhooks []hooks.Webhook `json:"webhooks"`
	}

	assert.Nil(json.Unmarshal(rsp.Body.Bytes(), &parsedRsp))

	parsedWebhooks := make(hooks.WebhookStore)
	for hookId := range parsedRsp.Webhooks {
		hook := &parsedRsp.Webhooks[hookId]
		parsedWebhooks[hook.Id] = hook
	}

	assert.Equal(testSetup.hooks, parsedWebhooks)
}

func TestGetHookAPI(t *testing.T) {
	assert := assert.New(t)

	testSetup := setupRoutesTest(t)
	defer testSetup.cleanup(t)

	rsp := testRoute(t, testSetup.config, "/webhooks/test-hook-1", "GET", nil)
	assert.Equal(http.StatusOK, rsp.Code)

	var parsedHook hooks.Webhook
	fmt.Println(string(rsp.Body.Bytes()))

	assert.Nil(json.Unmarshal(rsp.Body.Bytes(), &parsedHook))
	assert.Equal(*testSetup.hooks["test-hook-1"], parsedHook)
}

func TestDeleteGetHookAPI(t *testing.T) {
	assert := assert.New(t)

	testSetup := setupRoutesTest(t)
	defer testSetup.cleanup(t)

	assert.Equal(
		http.StatusNoContent,
		testRoute(t, testSetup.config, "/webhooks/test-hook-1", "DELETE", nil).Code,
	)

	rsp := testRoute(t, testSetup.config, "/webhooks", "GET", nil)
	assert.Equal(http.StatusOK, rsp.Code)

	var parsedRsp struct {
		Webhooks []hooks.Webhook `json:"webhooks"`
	}

	assert.Nil(json.Unmarshal(rsp.Body.Bytes(), &parsedRsp))
	assert.Equal(1, len(parsedRsp.Webhooks))
	assert.Equal(*testSetup.hooks["test-hook-2"], parsedRsp.Webhooks[0])
}

func TestCreateHookAPI(t *testing.T) {
	assert := assert.New(t)

	testSetup := setupRoutesTest(t)
	defer testSetup.cleanup(t)

	hook := hooks.Webhook{
		Id:      "test-hook-3",
		Url:     "http://example.com/3/",
		Secret:  "a very very secret thing",
		Enabled: true,
		Events:  []string{events.PushEvent},
		Repos:   []string{testSetup.repo.Name},
	}

	body, err := json.Marshal(hook)
	assert.Nil(err)

	assert.Equal(
		http.StatusCreated,
		testRoute(t, testSetup.config, "/webhooks", "POST", body).Code,
	)

	rsp := testRoute(t, testSetup.config, "/webhooks/test-hook-3", "GET", nil)
	assert.Equal(http.StatusOK, rsp.Code)

	var parsedHook hooks.Webhook
	assert.Nil(json.Unmarshal(rsp.Body.Bytes(), &parsedHook))

	assert.Equal(hook, parsedHook)
}

func TestCreateHookAPIValidate(t *testing.T) {
	assert := assert.New(t)

	testSetup := setupRoutesTest(t)
	defer testSetup.cleanup(t)

	testCases := []struct {
		hook     hooks.Webhook
		errorMsg string
	}{
		{
			hook:     *testSetup.hooks["test-hook-1"],
			errorMsg: "A webhook with ID \"test-hook-1\" already exists.\n",
		},
		{
			hook: hooks.Webhook{
				Id:      "test-hook-3",
				Url:     "http://example.com",
				Secret:  strings.Repeat("a", 20),
				Enabled: true,
				Events:  []string{},
				Repos:   []string{"repo"},
			},
			errorMsg: "Hook has no events.\n",
		},
		{
			hook: hooks.Webhook{
				Id:      "test-hook-3",
				Url:     "http://example.com",
				Secret:  strings.Repeat("a", 20),
				Enabled: true,
				Events:  []string{"foo"},
				Repos:   []string{"repo"},
			},
			errorMsg: "Invalid event: \"foo\".\n",
		},
		{
			hook: hooks.Webhook{
				Id:      "test-hook-3",
				Url:     "http://example.com",
				Secret:  strings.Repeat("a", 20),
				Enabled: true,
				Events:  []string{events.PushEvent},
				Repos:   []string{},
			},
			errorMsg: "Hook has no repositories.\n",
		},
		{
			hook: hooks.Webhook{
				Id:      "test-hook-3",
				Url:     "http://example.com",
				Secret:  strings.Repeat("a", 20),
				Enabled: true,
				Events:  []string{events.PushEvent},
				Repos:   []string{"foo"},
			},
			errorMsg: "Invalid repository: \"foo\".\n",
		},
		{
			hook: hooks.Webhook{
				Id:      "test-hook-3",
				Url:     "ftp://example.com",
				Secret:  strings.Repeat("a", 20),
				Enabled: true,
				Events:  []string{events.PushEvent},
				Repos:   []string{"repo"},
			},
			errorMsg: "Invalid URL scheme \"ftp\": only HTTP and HTTPS are supported.\n",
		},
		{
			hook: hooks.Webhook{
				Id:      "test-hook-3",
				Url:     "http://example.com",
				Secret:  "a",
				Enabled: true,
				Events:  []string{events.PushEvent},
				Repos:   []string{"repo"},
			},
			errorMsg: "Secret is too short (1 bytes); secrets must be at least 20 bytes.\n",
		},
	}

	for _, testCase := range testCases {
		body, err := json.Marshal(testCase.hook)
		assert.Nil(err)

		rsp := testRoute(t, testSetup.config, "/webhooks", "POST", body)
		assert.Equal(http.StatusBadRequest, rsp.Code)
		assert.Equal(testCase.errorMsg, string(rsp.Body.Bytes()))
	}
}

func TestUpdateHook(t *testing.T) {
	assert := assert.New(t)

	testSetup := setupRoutesTest(t)
	defer testSetup.cleanup(t)

	hook := testSetup.hooks["test-hook-1"]

	testCases := []struct {
		body       map[string]interface{}
		statusCode int
		errorMsg   string
		expected   *hooks.Webhook
	}{
		{
			body: map[string]interface{}{
				"id": "foo-bar",
			},
			statusCode: 400,
			errorMsg:   "Hook ID cannot be updated.\n",
		},
		{
			body: map[string]interface{}{
				"url": "https://example.com/some-path/?foo",
			},
			statusCode: 200,
			expected: &hooks.Webhook{
				Id:      hook.Id,
				Url:     "https://example.com/some-path/?foo",
				Secret:  hook.Secret,
				Enabled: true,
				Events:  hook.Events,
				Repos:   hook.Repos,
			},
		},
		{
			body: map[string]interface{}{
				"secret": "abcd",
			},
			statusCode: 400,
			errorMsg:   "Secret is too short (4 bytes); secrets must be at least 20 bytes.\n",
		},
		{
			body: map[string]interface{}{
				"secret": strings.Repeat("b", 20),
			},
			statusCode: 200,
			expected: &hooks.Webhook{
				Id:      hook.Id,
				Url:     "https://example.com/some-path/?foo",
				Secret:  strings.Repeat("b", 20),
				Enabled: true,
				Events:  hook.Events,
				Repos:   hook.Repos,
			},
		},
		{
			body: map[string]interface{}{
				"enabled": false,
			},
			statusCode: 200,
			expected: &hooks.Webhook{
				Id:      hook.Id,
				Url:     "https://example.com/some-path/?foo",
				Secret:  strings.Repeat("b", 20),
				Enabled: false,
				Events:  hook.Events,
				Repos:   hook.Repos,
			},
		},
		{
			body: map[string]interface{}{
				"events": []string{events.PushEvent, "pull"},
			},
			statusCode: 400,
			errorMsg:   "Invalid event: \"pull\".\n",
		},
		{
			body: map[string]interface{}{
				"events": []string{},
			},
			statusCode: 400,
			errorMsg:   "Hook has no events.\n",
		},
		{
			body: map[string]interface{}{
				"repos": []string{"asdf"},
			},
			statusCode: 400,
			errorMsg:   "Invalid repository: \"asdf\".\n",
		},
		{
			body: map[string]interface{}{
				"repos": []string{},
			},
			statusCode: 400,
			errorMsg:   "Hook has no repositories.\n",
		},
	}

	for _, testCase := range testCases {
		body, err := json.Marshal(testCase.body)
		assert.Nil(err)

		rsp := testRoute(t, testSetup.config, "/webhooks/test-hook-1", "PATCH", body)
		assert.Equal(testCase.statusCode, rsp.Code)
		if testCase.statusCode < 200 || testCase.statusCode > 299 {
			assert.Equal(testCase.errorMsg, string(rsp.Body.Bytes()))
		} else {
			assert.NotNil(testCase.expected)

			body := rsp.Body.Bytes()
			fmt.Printf("Body = %s\n", string(body))

			var parsedHook hooks.Webhook
			assert.Nil(json.Unmarshal(body, &parsedHook))
			assert.Equal(*testCase.expected, parsedHook)
		}
	}
}
