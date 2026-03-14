package api_test

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/reviewboard/rb-gateway/api"
	"github.com/reviewboard/rb-gateway/helpers"
)

// Test that the logging middleware logs request details and passes through
// to the next handler. We test this indirectly through the full API since
// loggingMiddleware is unexported, but ServeHTTP wraps the router with it.
func TestLoggingMiddlewarePassthrough(t *testing.T) {
	assert := assert.New(t)

	cfg := helpers.CreateTestConfig(t)
	helpers.CreateTestHtpasswd(t, "user", "pass", &cfg)
	helpers.WriteTestWebhookStore(t, nil, &cfg)
	defer helpers.CleanupConfig(t, &cfg)

	// Capture log output.
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	slog.SetDefault(logger)
	defer slog.SetDefault(slog.Default())

	handler, err := api.New(&cfg)
	assert.NotNil(handler)
	assert.Nil(err)

	// Make a request that will go through the logging middleware.
	request, err := http.NewRequest("GET", "/session", nil)
	assert.Nil(err)
	request.SetBasicAuth("user", "pass")

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	assert.Equal(http.StatusOK, response.Code)

	logOutput := buf.String()
	assert.Contains(logOutput, "request")
	assert.Contains(logOutput, "method=GET")
	assert.Contains(logOutput, "path=/session")
	assert.Contains(logOutput, "status=200")
}

// Test that the logging middleware captures non-200 status codes.
func TestLoggingMiddlewareNon200(t *testing.T) {
	assert := assert.New(t)

	cfg := helpers.CreateTestConfig(t)
	helpers.CreateTestHtpasswd(t, "user", "pass", &cfg)
	helpers.WriteTestWebhookStore(t, nil, &cfg)
	defer helpers.CleanupConfig(t, &cfg)

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	slog.SetDefault(logger)
	defer slog.SetDefault(slog.Default())

	handler, err := api.New(&cfg)
	assert.NotNil(handler)
	assert.Nil(err)

	// Request without auth should get 401.
	request, err := http.NewRequest("GET", "/session", nil)
	assert.Nil(err)

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	assert.Equal(http.StatusUnauthorized, response.Code)

	logOutput := buf.String()
	assert.Contains(logOutput, "status=401")
}
