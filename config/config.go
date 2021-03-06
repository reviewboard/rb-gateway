package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"strings"

	"github.com/reviewboard/rb-gateway/repositories"
)

const DefaultConfigPath = "config.json"

const (
	defaultPort uint16 = 8888
)

type RawRepository struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Scm  string `json:"scm"`
}

type Config struct {
	HtpasswdPath     string          `json:"htpasswdPath"`
	Port             uint16          `json:"port"`
	RepositoryData   []RawRepository `json:"repositories"`
	SSLCertificate   string          `json:"sslCertificate"`
	SSLKey           string          `json:"sslKey"`
	TokenStorePath   string          `json:"tokenStorePath"`
	UseTLS           bool            `json:"useTLS"`
	WebhookStorePath string          `json:"webhookStorePath"`

	Repositories map[string]repositories.Repository `json:"-"`
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

	var cfgDir string
	if cfgDir, err = filepath.Abs(path); err != nil {
		return nil, err
	} else {
		cfgDir = filepath.Dir(cfgDir)
	}

	if err = validate(cfgDir, &config); err != nil {
		return nil, err
	}

	config.Repositories = make(map[string]repositories.Repository)

	for _, repo := range config.RepositoryData {
		info := repositories.RepositoryInfo{
			Name: repo.Name,
			Path: repo.Path,
		}

		switch repo.Scm {
		case "git":
			config.Repositories[repo.Name] = &repositories.GitRepository{
				RepositoryInfo: info,
			}

		case "hg":
			config.Repositories[repo.Name] = &repositories.HgRepository{
				RepositoryInfo: info,
			}

		default:
			log.Printf("Unknown SCM '%s' while loading configuration '%s'; ignoring.", repo.Scm, path)
		}
	}

	return &config, nil
}

// Return the set of repository names.
//
// See `hooks.LoadStore()`.
func (cfg *Config) RepositorySet() (repos map[string]struct{}) {
	repos = make(map[string]struct{})
	for name := range cfg.Repositories {
		repos[name] = struct{}{}
	}
	return
}

func validate(cfgDir string, config *Config) (err error) {
	missingFields := []string{}

	if config.Port == 0 {
		log.Printf("WARNING: Port missing from config, defaulting to %d.", defaultPort)
		config.Port = defaultPort
	}

	if len(config.RepositoryData) == 0 {
		missingFields = append(missingFields, "repositories")
	}

	if config.UseTLS {
		if config.SSLCertificate == "" {
			missingFields = append(missingFields, "ssl_certificate")
		} else {
			config.SSLCertificate = resolvePath(cfgDir, config.SSLCertificate)
		}

		if config.SSLKey == "" {
			missingFields = append(missingFields, "ssl_key")
		} else {
			config.SSLKey = resolvePath(cfgDir, config.SSLKey)
		}
	}

	optionalPathFields := []struct {
		field        *string
		name         string
		defaultValue string
	}{
		{&config.TokenStorePath, "tokenStorePath", "tokens.dat"},
		{&config.HtpasswdPath, "htpasswdPath", "htpasswd"},
		{&config.WebhookStorePath, "webhookStorePath", "webhooks.json"},
	}

	for _, fieldInfo := range optionalPathFields {
		if *fieldInfo.field == "" {
			log.Printf(`Warning: %s missing from config, defaulting to "%v".`, fieldInfo.name, fieldInfo.defaultValue)
			*fieldInfo.field = fieldInfo.defaultValue
		}
	}

	if config.TokenStorePath != ":memory:" {
		config.TokenStorePath = resolvePath(cfgDir, config.TokenStorePath)
	}

	config.HtpasswdPath = resolvePath(cfgDir, config.HtpasswdPath)
	config.WebhookStorePath = resolvePath(cfgDir, config.WebhookStorePath)

	if len(missingFields) != 0 {
		err = fmt.Errorf("Some required fields were missing from the configuration: %s.", strings.Join(missingFields, ","))
	}

	return
}

// Resolve a path so that . is treated as cfgDir
func resolvePath(cfgDir string, path string) string {
	if !filepath.IsAbs(path) {
		path = filepath.Join(cfgDir, path)
	}
	return path
}
