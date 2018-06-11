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

	var port uint16 = 8888
	var username string = "username"
	var password string = "password"

	repo, _ := helpers.CreateTestRepo(t, "repo")
	defer helpers.CleanupRepository(t, repo.Path)

	_, err = file.WriteString(fmt.Sprintf(`
		{
			"port": %d,
			"username": "%s",
			"password": "%s",
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
		port, username, password,
		repo.GetName(), repo.GetPath(), repo.GetScm()))

	loaded, err := config.Load(path)
	assert.Nil(err)

	assert.Equal(loaded.Port, port)
	assert.Equal(loaded.Username, username)
	assert.Equal(loaded.Password, password)

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

	username := "username"
	password := "password"

	_, err = file.WriteString(fmt.Sprintf(`
		{
			"username": "%s",
			"password": "%s",
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
		username, password,
		repo.GetName(), repo.GetPath(), repo.GetScm()))

	file.Close()

	cfg, err := config.Load(path)
	assert.Nil(err)
	assert.NotNil(cfg)

	assert.Equal(uint16(8888), cfg.Port)
	assert.Equal(username, cfg.Username)
	assert.Equal(password, cfg.Password)

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

	username := "username"
	password := "password"

	_, err = file.WriteString(fmt.Sprintf(`
		{
			"username": "%s",
			"password": "%s",
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
		username, password,
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

	username := "username"
	password := "password"
	sslCertificate := "foo.pem"
	sslKey := "/etc/ssl/private/keys/foo.key"

	_, err = cfgFile.WriteString(fmt.Sprintf(`
		{
			"username": "%s",
			"password": "%s",
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
		username, password, sslCertificate, sslKey,
		repo.GetName(), repo.GetPath(), repo.GetScm()))

	cfgFile.Close()

	cfg, err := config.Load(cfgPath)
	assert.Nil(err)
	assert.NotNil(cfg)

	assert.Equal(uint16(8888), cfg.Port)
	assert.Equal(username, cfg.Username)
	assert.Equal(password, cfg.Password)
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
