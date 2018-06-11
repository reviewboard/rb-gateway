package helpers

import (
	"testing"

	"github.com/reviewboard/rb-gateway/config"
	"github.com/reviewboard/rb-gateway/repositories"
)

func CreateTestConfig(t *testing.T, repos ...repositories.Repository) config.Config {
	t.Helper()

	cfg := config.Config{
		Port:           8888,
		Username:       "username",
		Password:       "password",
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
