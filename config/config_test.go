package config_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/reviewboard/rb-gateway/config"
	"github.com/reviewboard/rb-gateway/helpers"
)

func TestLoadConfig(t *testing.T) {
	file, err := ioutil.TempFile("", "rb-gateway-config-")
	assert.Nil(t, err)

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
			]
		}
		`,
		port, username, password,
		repo.GetName(), repo.GetPath(), repo.GetScm()))

	path := file.Name()
	file.Close()

	defer os.Remove(path)

	loaded, err := config.Load(path)
	assert.Nil(t, err)

	assert.Equal(t, loaded.Port, port)
	assert.Equal(t, loaded.Username, username)
	assert.Equal(t, loaded.Password, password)

	assert.Equal(t, len(loaded.Repositories), 1)
	assert.Contains(t, loaded.Repositories, repo.Name)

	loadedRepo := loaded.Repositories[repo.Name]
	assert.Equal(t, loadedRepo.GetName(), repo.Name)
	assert.Equal(t, loadedRepo.GetPath(), repo.Path)
	assert.Equal(t, loadedRepo.GetScm(), repo.GetScm())
}
