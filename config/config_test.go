package config_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/reviewboard/rb-gateway/config"
	"github.com/reviewboard/rb-gateway/helpers"
)

func TestLoadConfig(t *testing.T) {
	repo, _ := helpers.CreateTestRepo(t, "repo")
	defer helpers.CleanupRepository(t, repo.Path)

	cfg := helpers.CreateTestConfig(t, repo)
	path := helpers.WriteTestConfig(t, cfg)
	defer helpers.CleanupConfig(t, path)

	loaded, err := config.Load(path)
	assert.Nil(t, err)

	assert.Equal(t, cfg.Port, loaded.Port)
	assert.Equal(t, cfg.Username, loaded.Username)
	assert.Equal(t, cfg.Password, loaded.Password)

	assert.Equal(t, len(loaded.Repositories), 1)
	assert.Contains(t, loaded.Repositories, repo.Name)

	loadedRepo := loaded.Repositories[repo.Name]
	assert.Equal(t, loadedRepo.GetName(), repo.Name)
	assert.Equal(t, loadedRepo.GetPath(), repo.Path)
	assert.Equal(t, loadedRepo.GetScm(), repo.GetScm())
}
