package config_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/reviewboard/rb-gateway/config"
	"github.com/reviewboard/rb-gateway/helpers"
)

func TestLoadConfig(t *testing.T) {
	assert := assert.New(t)

	file, err := ioutil.TempFile("", "rb-gateway-config-")
	assert.Nil(err)
	defer file.Close()

	path := file.Name()
	defer os.Remove(path)

	htpasswdPath := "htpasswd"
	port := uint16(8888)
	tokenStorePath := ":memory:"

	repo, _ := helpers.CreateTestRepo(t, "repo")
	defer helpers.CleanupRepository(t, repo.Path)

	_, err = file.WriteString(fmt.Sprintf(`
		{
			"htpasswdPath": "%s",
			"port": %d,
			"repositories": [
				{
					"name": "%s",
					"path": "%s",
					"scm": "%s"
				}
			],
			"tokenStorePath": "%s"
		}
		`,
		htpasswdPath,
		port,
		repo.GetName(),
		repo.GetPath(),
		repo.GetScm(),
		tokenStorePath))

	loaded, err := config.Load(path)
	assert.Nil(err)

	assert.Equal(loaded.HtpasswdPath, filepath.Join(filepath.Dir(path), htpasswdPath))
	assert.Equal(loaded.Port, port)
	assert.Equal(loaded.TokenStorePath, tokenStorePath)

	assert.Equal(len(loaded.Repositories), 1)
	assert.Contains(loaded.Repositories, repo.Name)

	loadedRepo := loaded.Repositories[repo.Name]
	assert.Equal(loadedRepo.GetName(), repo.Name)
	assert.Equal(loadedRepo.GetPath(), repo.Path)
	assert.Equal(loadedRepo.GetScm(), repo.GetScm())

}

func TestLoadConfigAllFieldsMissing(t *testing.T) {
	assert := assert.New(t)

	file, err := ioutil.TempFile("", "rb-gateway-config-")
	assert.Nil(err)

	path := file.Name()
	defer os.Remove(path)

	_, err = file.WriteString("{}")
	assert.Nil(err)
	err = file.Close()
	assert.Nil(err)

	cfg, err := config.Load(path)
	assert.NotNil(err)
	assert.Nil(cfg)
}

func TestLoadConfigPortMissing(t *testing.T) {
	assert := assert.New(t)

	file, err := ioutil.TempFile("", "rb-gateway-config-")
	assert.Nil(err)

	path := file.Name()
	defer os.Remove(path)

	repo, _ := helpers.CreateTestRepo(t, "repo")
	defer helpers.CleanupRepository(t, repo.Path)

	_, err = file.WriteString(fmt.Sprintf(`
		{
			"htpasswdPath": "htpasswd",
			"repositories": [
				{
					"name": "%s",
					"path": "%s",
					"scm": "%s"
				}
			],
			"tokenStorePath": ":memory:"
		}
		`,
		repo.GetName(), repo.GetPath(), repo.GetScm()))

	file.Close()

	cfg, err := config.Load(path)
	assert.Nil(err)
	assert.NotNil(cfg)

	assert.Equal(uint16(8888), cfg.Port)
	assert.Equal(filepath.Join(filepath.Dir(path), "htpasswd"), cfg.HtpasswdPath)

	assert.Equal(1, len(cfg.Repositories))
	assert.Contains(cfg.Repositories, repo.Name)

	cfgRepo := cfg.Repositories[repo.Name]
	assert.Equal(repo.Name, cfgRepo.GetName())
	assert.Equal(repo.Path, cfgRepo.GetPath())
	assert.Equal(repo.GetScm(), cfgRepo.GetScm())
}

func TestLoadConfigTlsMissing(t *testing.T) {
	assert := assert.New(t)

	file, err := ioutil.TempFile("", "rb-gateway-config-")
	assert.Nil(err)

	path := file.Name()
	defer os.Remove(path)

	repo, _ := helpers.CreateTestRepo(t, "repo")
	defer helpers.CleanupRepository(t, repo.Path)

	_, err = file.WriteString(fmt.Sprintf(`
		{
			"htpasswdPath": "htpasswd",
			"useTLS": true,
			"repositories": [
				{
					"name": "%s",
					"path": "%s",
					"scm": "%s"
				}
			]
		}
		`,
		repo.GetName(), repo.GetPath(), repo.GetScm()))

	file.Close()

	cfg, err := config.Load(path)
	assert.NotNil(err)
	assert.Nil(cfg)
}

func TestLoadConfigTls(t *testing.T) {
	assert := assert.New(t)

	repo, _ := helpers.CreateTestRepo(t, "repo")
	defer helpers.CleanupRepository(t, repo.Path)

	dir, err := ioutil.TempDir("", "rb-gateway-test")
	assert.Nil(err)

	cfgPath := filepath.Join(dir, "config.json")
	cfgFile, err := os.OpenFile(cfgPath, os.O_WRONLY|os.O_CREATE, 0600)
	assert.Nil(err)

	sslCertificate := "foo.pem"
	sslKey := "/etc/ssl/private/keys/foo.key"

	_, err = cfgFile.WriteString(fmt.Sprintf(`
		{
			"htpasswdPath": "htpasswd",
			"useTLS": true,
			"sslCertificate": "%s",
			"sslKey": "%s",
			"tokenStorePath": ":memory:",
			"repositories": [
				{
					"name": "%s",
					"path": "%s",
					"scm": "%s"
				}
			]
		}
		`,
		sslCertificate, sslKey,
		repo.GetName(), repo.GetPath(), repo.GetScm()))

	cfgFile.Close()

	cfg, err := config.Load(cfgPath)
	assert.Nil(err)
	assert.NotNil(cfg)

	assert.Equal(uint16(8888), cfg.Port)
	assert.True(cfg.UseTLS)
	assert.Equal(filepath.Join(dir, sslCertificate), cfg.SSLCertificate)
	assert.Equal(sslKey, cfg.SSLKey)

	assert.Equal(1, len(cfg.Repositories))
	assert.Contains(cfg.Repositories, repo.Name)

	cfgRepo := cfg.Repositories[repo.Name]
	assert.Equal(repo.Name, cfgRepo.GetName())
	assert.Equal(repo.Path, cfgRepo.GetPath())
	assert.Equal(repo.GetScm(), cfgRepo.GetScm())

}
