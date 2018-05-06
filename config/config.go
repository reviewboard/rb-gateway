package config

import (
	"encoding/json"
	"io/ioutil"
	"log"

	"github.com/reviewboard/rb-gateway/repositories"
)

const DefaultConfigPath = "config.json"

type repositoryData struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Scm  string `json:"scm"`
}

type Config struct {
	Port           uint16           `json:"port"`
	Username       string           `json:"username"`
	Password       string           `json:"password"`
	UseTLS         bool             `json:"useTLS"`
	SSLCertificate string           `json:"sslCertificate"`
	SSLKey         string           `json:"sslKey"`
	RepositoryData []repositoryData `json:"repositories"`
	Repositories   map[string]repositories.Repository
}

func Load(path string) (*Config, error) {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	if err = json.Unmarshal(content, &config); err != nil {
		return nil, err
	}

	config.Repositories = make(map[string]repositories.Repository)

	for _, repo := range config.RepositoryData {
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
