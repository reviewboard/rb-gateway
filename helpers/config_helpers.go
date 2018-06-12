package helpers

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/foomo/htpasswd"
	"github.com/stretchr/testify/assert"

	"github.com/reviewboard/rb-gateway/config"
	"github.com/reviewboard/rb-gateway/repositories"
)

func CreateTestConfig(t *testing.T, repos ...repositories.Repository) config.Config {
	t.Helper()

	cfg := config.Config{
		Port:           8888,
		Repositories:   make(map[string]repositories.Repository),
		TokenStorePath: ":memory:",
	}

	for _, repo := range repos {
		name := repo.GetName()
		cfg.Repositories[name] = &repositories.GitRepository{
			repositories.RepositoryInfo{
				Name: repo.GetName(),
				Path: repo.GetPath(),
			},
		}
	}

	return cfg
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

// Cleanup any and all temp files specified in the configuration.
func CleanupConfig(t *testing.T, cfg config.Config) {
	t.Helper()
	assert := assert.New(t)
	// We use defer here so that if deleting cfg.TokenStorePath fails we can
	// still attempt to delete cfg.HtpasswdPath
	if cfg.TokenStorePath != "" && cfg.TokenStorePath != ":memory:" {
		err := os.Remove(cfg.TokenStorePath)
		defer assert.Nil(err)
	}

	if cfg.HtpasswdPath != "" {
		err := os.Remove(cfg.HtpasswdPath)
		assert.Nil(err)
	}
}
