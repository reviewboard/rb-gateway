package config_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/reviewboard/rb-gateway/config"
	"github.com/reviewboard/rb-gateway/helpers"
)

// writeValidConfig writes a valid config file to the given path.
func writeValidConfig(t *testing.T, path string, port uint16) {
	t.Helper()
	assert := assert.New(t)

	repo, _ := helpers.CreateGitRepo(t, "watch-repo")
	t.Cleanup(func() { helpers.CleanupRepository(t, repo.Path) })

	content := fmt.Sprintf(`{
		"htpasswdPath": "htpasswd",
		"port": %d,
		"tokenStorePath": ":memory:",
		"repositories": [
			{
				"name": "%s",
				"path": "%s",
				"scm": "git"
			}
		]
	}`, port, repo.Name, repo.Path)

	err := os.WriteFile(path, []byte(content), 0600)
	assert.Nil(err)
}

// Test that Watch emits the initial config.
func TestWatchInitialConfig(t *testing.T) {
	assert := assert.New(t)

	tmpfile, err := os.CreateTemp("", "rb-gateway-watch-")
	assert.Nil(err)
	path := tmpfile.Name()
	tmpfile.Close()
	defer os.Remove(path)

	writeValidConfig(t, path, 9000)

	watcher := config.Watch(path)

	select {
	case cfg := <-watcher.NewConfig:
		assert.NotNil(cfg)
		assert.Equal(uint16(9000), cfg.Port)
	case err := <-watcher.Errors:
		t.Fatalf("Unexpected error: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("Timed out waiting for initial config")
	}
}

// Test that modifying the config file emits a new config.
func TestWatchConfigChange(t *testing.T) {
	assert := assert.New(t)

	tmpfile, err := os.CreateTemp("", "rb-gateway-watch-")
	assert.Nil(err)
	path := tmpfile.Name()
	tmpfile.Close()
	defer os.Remove(path)

	writeValidConfig(t, path, 9000)

	watcher := config.Watch(path)

	// Consume the initial config.
	select {
	case <-watcher.NewConfig:
	case err := <-watcher.Errors:
		t.Fatalf("Unexpected error: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("Timed out waiting for initial config")
	}

	// Modify the file.
	writeValidConfig(t, path, 9001)

	// Should get a new config with the updated port.
	select {
	case cfg := <-watcher.NewConfig:
		assert.NotNil(cfg)
		assert.Equal(uint16(9001), cfg.Port)
	case err := <-watcher.Errors:
		t.Fatalf("Unexpected error: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("Timed out waiting for updated config")
	}
}

// Test ForceReload.
func TestWatchForceReload(t *testing.T) {
	assert := assert.New(t)

	tmpfile, err := os.CreateTemp("", "rb-gateway-watch-")
	assert.Nil(err)
	path := tmpfile.Name()
	tmpfile.Close()
	defer os.Remove(path)

	writeValidConfig(t, path, 9000)

	watcher := config.Watch(path)

	// Consume the initial config.
	select {
	case <-watcher.NewConfig:
	case err := <-watcher.Errors:
		t.Fatalf("Unexpected error: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("Timed out waiting for initial config")
	}

	// Force a reload - should re-read and emit the same config.
	cfg, err := watcher.ForceReload()
	assert.Nil(err)
	assert.NotNil(cfg)
	assert.Equal(uint16(9000), cfg.Port)
}

// Test that watching a non-existent file returns an error.
func TestWatchNonExistentFile(t *testing.T) {
	watcher := config.Watch("/nonexistent/config.json")

	select {
	case err := <-watcher.Errors:
		assert.NotNil(t, err)
	case <-watcher.NewConfig:
		t.Fatal("Should not receive config for non-existent file")
	case <-time.After(2 * time.Second):
		t.Fatal("Timed out waiting for error")
	}
}

// Test that an invalid config file returns an error.
func TestWatchInvalidConfig(t *testing.T) {
	assert := assert.New(t)

	tmpfile, err := os.CreateTemp("", "rb-gateway-watch-")
	assert.Nil(err)
	path := tmpfile.Name()
	defer os.Remove(path)

	_, err = tmpfile.WriteString("not valid json")
	assert.Nil(err)
	assert.Nil(tmpfile.Close())

	watcher := config.Watch(path)

	select {
	case err := <-watcher.Errors:
		assert.NotNil(err)
	case <-watcher.NewConfig:
		t.Fatal("Should not receive config for invalid file")
	case <-time.After(2 * time.Second):
		t.Fatal("Timed out waiting for error")
	}
}
