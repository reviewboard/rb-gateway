package helpers

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/reviewboard/rb-gateway/config"
	"github.com/reviewboard/rb-gateway/repositories"
)

func CreateTestConfig(t *testing.T, repos ...repositories.Repository) config.Config {
	cfg := config.Config{
		Port:         8888,
		Username:     "username",
		Password:     "password",
		Repositories: make(map[string]repositories.Repository),
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

func WriteTestConfig(t *testing.T, cfg config.Config) string {
	t.Helper()

	file, err := ioutil.TempFile("", "rb-gateway-config-")
	assert.Nil(t, err)

	defer file.Close()

	content, err := cfg.Serialize()
	if err != nil {
		defer os.Remove(file.Name())
	}
	assert.Nil(t, err)

	_, err = file.Write(content)

	return file.Name()
}

func CleanupConfig(t *testing.T, configPath string) {
	os.Remove(configPath)
}
