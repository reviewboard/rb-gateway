package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/reviewboard/rb-gateway/api"
	"github.com/reviewboard/rb-gateway/api/tokens"
	"github.com/reviewboard/rb-gateway/helpers"
)

// Test that newHtpasswdSecretProvider loads a valid htpasswd file.
func TestHtpasswdSecretProvider(t *testing.T) {
	assert := assert.New(t)

	cfg := helpers.CreateTestConfig(t)
	helpers.CreateTestHtpasswd(t, "testuser", "testpass", &cfg)
	defer os.Remove(cfg.HtpasswdPath)

	// Verify the API can be created with this config (which loads the htpasswd).
	// We test indirectly through the session endpoint.
	helpers.WriteTestWebhookStore(t, nil, &cfg)
	defer helpers.CleanupConfig(t, &cfg)

	handler, err := api.New(&cfg)
	assert.NotNil(handler)
	assert.Nil(err)

	// Valid credentials should succeed.
	request, err := http.NewRequest("GET", "/session", nil)
	assert.Nil(err)
	request.SetBasicAuth("testuser", "testpass")

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	assert.Equal(http.StatusOK, response.Code)

	var session api.Session
	assert.Nil(json.Unmarshal(response.Body.Bytes(), &session))
	assert.Equal(tokens.TokenSize, len(session.PrivateToken))
}

// Test that an unknown user gets a 401.
func TestBasicAuthUnknownUser(t *testing.T) {
	assert := assert.New(t)

	cfg := helpers.CreateTestConfig(t)
	helpers.CreateTestHtpasswd(t, "testuser", "testpass", &cfg)
	helpers.WriteTestWebhookStore(t, nil, &cfg)
	defer helpers.CleanupConfig(t, &cfg)

	handler, err := api.New(&cfg)
	assert.NotNil(handler)
	assert.Nil(err)

	request, err := http.NewRequest("GET", "/session", nil)
	assert.Nil(err)
	request.SetBasicAuth("nosuchuser", "testpass")

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	assert.Equal(http.StatusUnauthorized, response.Code)
	assert.Equal(`Basic realm="RB Gateway"`, response.Header().Get("WWW-Authenticate"))
}

// Test that a wrong password gets a 401.
func TestBasicAuthWrongPassword(t *testing.T) {
	assert := assert.New(t)

	cfg := helpers.CreateTestConfig(t)
	helpers.CreateTestHtpasswd(t, "testuser", "testpass", &cfg)
	helpers.WriteTestWebhookStore(t, nil, &cfg)
	defer helpers.CleanupConfig(t, &cfg)

	handler, err := api.New(&cfg)
	assert.NotNil(handler)
	assert.Nil(err)

	request, err := http.NewRequest("GET", "/session", nil)
	assert.Nil(err)
	request.SetBasicAuth("testuser", "wrongpass")

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	assert.Equal(http.StatusUnauthorized, response.Code)
	assert.Equal(`Basic realm="RB Gateway"`, response.Header().Get("WWW-Authenticate"))
}

// Test that a request with no credentials gets a 401.
func TestBasicAuthNoCredentials(t *testing.T) {
	assert := assert.New(t)

	cfg := helpers.CreateTestConfig(t)
	helpers.CreateTestHtpasswd(t, "testuser", "testpass", &cfg)
	helpers.WriteTestWebhookStore(t, nil, &cfg)
	defer helpers.CleanupConfig(t, &cfg)

	handler, err := api.New(&cfg)
	assert.NotNil(handler)
	assert.Nil(err)

	request, err := http.NewRequest("GET", "/session", nil)
	assert.Nil(err)
	// No SetBasicAuth call.

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	assert.Equal(http.StatusUnauthorized, response.Code)
	assert.Equal(`Basic realm="RB Gateway"`, response.Header().Get("WWW-Authenticate"))
}

// Test that a malformed htpasswd file returns an error.
func TestHtpasswdMalformedFile(t *testing.T) {
	assert := assert.New(t)

	tmpfile, err := os.CreateTemp("", "htpasswd-malformed-")
	assert.Nil(err)
	defer os.Remove(tmpfile.Name())

	// Write a line with three colon-separated fields (malformed).
	_, err = tmpfile.WriteString("user:hash:extra\n")
	assert.Nil(err)
	assert.Nil(tmpfile.Close())

	cfg := helpers.CreateTestConfig(t)
	cfg.HtpasswdPath = tmpfile.Name()
	cfg.WebhookStorePath = ""
	cfg.TokenStorePath = ":memory:"

	// Creating the API should fail because of the malformed htpasswd.
	handler, err := api.New(&cfg)
	assert.Nil(handler)
	assert.NotNil(err)
}

// Test that a non-existent htpasswd file returns an error.
func TestHtpasswdMissingFile(t *testing.T) {
	assert := assert.New(t)

	cfg := helpers.CreateTestConfig(t)
	cfg.HtpasswdPath = "/nonexistent/htpasswd"
	cfg.TokenStorePath = ":memory:"

	handler, err := api.New(&cfg)
	assert.Nil(handler)
	assert.NotNil(err)
}

// Test checkPassword with plaintext comparison.
//
// We test this indirectly through the session endpoint using a plaintext
// htpasswd file.
func TestPlaintextPassword(t *testing.T) {
	assert := assert.New(t)

	tmpfile, err := os.CreateTemp("", "htpasswd-plain-")
	assert.Nil(err)
	defer os.Remove(tmpfile.Name())

	// Write a plaintext htpasswd entry.
	_, err = tmpfile.WriteString("plainuser:plainpass\n")
	assert.Nil(err)
	assert.Nil(tmpfile.Close())

	cfg := helpers.CreateTestConfig(t)
	cfg.HtpasswdPath = tmpfile.Name()
	helpers.WriteTestWebhookStore(t, nil, &cfg)
	defer os.Remove(cfg.WebhookStorePath)

	handler, err := api.New(&cfg)
	assert.NotNil(handler)
	assert.Nil(err)

	// Correct plaintext password should succeed.
	request, err := http.NewRequest("GET", "/session", nil)
	assert.Nil(err)
	request.SetBasicAuth("plainuser", "plainpass")

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	assert.Equal(http.StatusOK, response.Code)

	// Wrong plaintext password should fail.
	request, err = http.NewRequest("GET", "/session", nil)
	assert.Nil(err)
	request.SetBasicAuth("plainuser", "wrong")

	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	assert.Equal(http.StatusUnauthorized, response.Code)
}
