package config

import (
	"encoding/json"
	"io/ioutil"
	"log"

	"github.com/reviewboard/rb-gateway/repositories"
)

const DefaultConfigPath = "config.json"

type serializedRepository struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Scm  string `json:"scm"`
}

type serializedConfig struct {
	Port         uint16                 `json:"port"`
	Username     string                 `json:"username"`
	Password     string                 `json:"password"`
	Repositories []serializedRepository `json:"repositories"`
}

type Config struct {
	Port         uint16
	Username     string
	Password     string
	Repositories map[string]repositories.Repository
}

func (c Config) Serialize() ([]byte, error) {
	rawRepos := make([]serializedRepository, 0, len(c.Repositories))

	for _, repo := range c.Repositories {
		rawRepos = append(rawRepos, serializedRepository{
			Name: repo.GetName(),
			Path: repo.GetPath(),
			Scm:  repo.GetScm(),
		})
	}

	rawConfig := serializedConfig{
		Port:         c.Port,
		Username:     c.Username,
		Password:     c.Password,
		Repositories: rawRepos,
	}

	return json.Marshal(rawConfig)
}

func Load(path string) (*Config, error) {
	var rawConfig serializedConfig

	content, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	if err = json.Unmarshal(content, &rawConfig); err != nil {
		return nil, err
	}

	config := Config{
		Port:         rawConfig.Port,
		Username:     rawConfig.Username,
		Password:     rawConfig.Password,
		Repositories: make(map[string]repositories.Repository),
	}

	for _, repo := range rawConfig.Repositories {
		switch repo.Scm {
		case "git":
			config.Repositories[repo.Name] = &repositories.GitRepository{
				repositories.RepositoryInfo{
					Name: repo.Name,
					Path: repo.Path,
				},
			}

		default:
			log.Printf("Unknown SCM '%s' while loading configuration '%s'; ignoring.", repo.Scm, path)
		}
	}

	return &config, nil
}
