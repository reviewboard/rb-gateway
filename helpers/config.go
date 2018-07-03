package helpers

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"

	"github.com/foomo/htpasswd"
	"github.com/stretchr/testify/assert"

	"github.com/reviewboard/rb-gateway/config"
	"github.com/reviewboard/rb-gateway/repositories"
	"github.com/reviewboard/rb-gateway/repositories/hooks"
)

func CreateTestConfig(t *testing.T, repos ...repositories.Repository) config.Config {
	t.Helper()

	cfg := config.Config{
		Port:           8888,
		Repositories:   make(map[string]repositories.Repository),
		RepositoryData: make([]config.RawRepository, 0, len(repos)),
		TokenStorePath: ":memory:",
	}

	for _, repo := range repos {
		cfg.Repositories[repo.GetName()] = repo
	}

	return cfg
}

func WriteConfig(t *testing.T, path string, cfg *config.Config) {
	t.Helper()
	assert := assert.New(t)

	for _, repository := range cfg.Repositories {
		cfg.RepositoryData = append(cfg.RepositoryData, config.RawRepository{
			Name: repository.GetName(),
			Path: repository.GetPath(),
			Scm:  repository.GetScm(),
		})
	}

	data, err := json.Marshal(cfg)
	assert.Nil(err)

	err = ioutil.WriteFile(path, data, 0600)
	assert.Nil(err)
}

// Create an htpasswd file and store its path in the given Config instance.
func CreateTestHtpasswd(t *testing.T, username, password string, cfg *config.Config) {
	t.Helper()
	assert := assert.New(t)

	tmpfile, err := ioutil.TempFile("", "htpasswd-")
	assert.Nil(err)

	cfg.HtpasswdPath = tmpfile.Name()

	err = tmpfile.Close()
	assert.Nil(err)

	err = htpasswd.SetPassword(cfg.HtpasswdPath, username, password, htpasswd.HashBCrypt)
	assert.Nil(err)
}

// Write the given webhook store and save the path in the given config.
func WriteTestWebhookStore(t *testing.T, store hooks.WebhookStore, cfg *config.Config) {
	t.Helper()
	assert := assert.New(t)

	tmpfile, err := ioutil.TempFile("", "rbgateway-webhooks-")
	assert.Nil(err)

	cfg.WebhookStorePath = tmpfile.Name()

	assert.Nil(store.Write(tmpfile))
	assert.Nil(tmpfile.Close())
}

// Cleanup any and all temp files specified in the configuration.
func CleanupConfig(t *testing.T, cfg *config.Config) {
	t.Helper()
	assert := assert.New(t)

	if cfg.TokenStorePath != "" && cfg.TokenStorePath != ":memory:" {
		err := os.Remove(cfg.TokenStorePath)
		assert.Nil(err)
	}

	if cfg.WebhookStorePath != "" {
		err := os.Remove(cfg.WebhookStorePath)
		assert.Nil(err)
	}

	if cfg.HtpasswdPath != "" {
		err := os.Remove(cfg.HtpasswdPath)
		assert.Nil(err)
	}
}
