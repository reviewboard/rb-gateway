package integration_tests

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/foomo/htpasswd"
	"github.com/stretchr/testify/assert"

	"github.com/reviewboard/rb-gateway/config"
	"github.com/reviewboard/rb-gateway/helpers"
	"github.com/reviewboard/rb-gateway/repositories"
	"github.com/reviewboard/rb-gateway/repositories/events"
	"github.com/reviewboard/rb-gateway/repositories/hooks"
)

// Set up the configuration and write it to disk.
func setupConfig(t *testing.T, upstream repositories.Repository) (string, config.Config) {
	t.Helper()
	assert := assert.New(t)

	cfgDir, err := ioutil.TempDir("", "rb-gateway-config-")
	assert.Nil(err)
	cfg := helpers.CreateTestConfig(t, upstream)

	hookStorePath := filepath.Join(cfgDir, "webhooks.json")
	cfg.WebhookStorePath = hookStorePath
	cfg.TokenStorePath = filepath.Join(cfgDir, "tokens.dat")
	cfg.HtpasswdPath = filepath.Join(cfgDir, "htpasswd")

	assert.Nil(htpasswd.SetPassword(cfg.HtpasswdPath, "username", "password", htpasswd.HashBCrypt))

	helpers.WriteConfig(t, filepath.Join(cfgDir, "config.json"), &cfg)

	return cfgDir, cfg
}

// Set up the webhook store and write it to disk.
func setupStore(t *testing.T, serverUrl string, cfg *config.Config) *hooks.Webhook {
	assert := assert.New(t)
	t.Helper()

	var repoName string
	for repoName, _ = range cfg.Repositories {
		break
	}

	assert.NotEqual(0, len(repoName))
	hook := &hooks.Webhook{
		Id:      "test-hook",
		Enabled: true,
		Url:     fmt.Sprintf("%s/test-hook", serverUrl),
		Secret:  "top-secret-123",
		Events:  []string{events.PushEvent},
		Repos:   []string{repoName},
	}

	store := hooks.WebhookStore{
		hook.Id: hook,
	}

    assert.Nil(store.Save(cfg.WebhookStorePath))

	return hook
}

type testCase struct {
	recorded *helpers.RecordedRequest
	message  string
	commitId string
	target   events.PushPayloadCommitTarget
}

func runTests(t *testing.T, cases []testCase, upstream repositories.Repository, hook *hooks.Webhook) {
	assert := assert.New(t)

	for i, testCase := range cases {
		request := testCase.recorded.Request
		body := testCase.recorded.Body

		assert.Equalf("/test-hook", request.URL.Path, "URL for request %d does not match", i)
		assert.Equalf(events.PushEvent, request.Header.Get("X-RBG-Event"), "X-RBG-Event header for request %d does not match", i)
		assert.Equalf(hook.SignPayload(body), request.Header.Get("X-RBG-Signature"), "Signature for request %d does not match", i)
		payload := events.PushPayload{
			Repository: upstream.GetName(),
			Commits: []events.PushPayloadCommit{
				{
					Id:      testCase.commitId,
					Message: testCase.message,
					Target:  testCase.target,
				},
			},
		}

		rawJson, err := events.MarshalPayload(payload)
		assert.Nil(err)

		assert.Equalf(string(rawJson), string(body), "Body for request %d does not match", i)
	}
}
