package repositories

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnsureDirCreatesNew(t *testing.T) {
	assert := assert.New(t)

	dir := filepath.Join(t.TempDir(), "newdir")

	created, err := ensureDir(dir)
	assert.Nil(err)
	assert.True(created, "Should report that the directory was created")

	info, err := os.Stat(dir)
	assert.Nil(err)
	assert.True(info.IsDir())
}

func TestEnsureDirExisting(t *testing.T) {
	assert := assert.New(t)

	dir := t.TempDir() // Already exists.

	created, err := ensureDir(dir)
	assert.Nil(err)
	assert.False(created, "Should report that the directory already existed")
}

func TestEnsureDirInvalidParent(t *testing.T) {
	assert := assert.New(t)

	dir := filepath.Join("/nonexistent", "parent", "child")

	created, err := ensureDir(dir)
	assert.NotNil(err)
	assert.False(created)
}

func TestGetExePath(t *testing.T) {
	assert := assert.New(t)

	exePath, err := getExePath()
	assert.Nil(err)
	assert.True(filepath.IsAbs(exePath), "Exe path should be absolute")
	assert.NotEmpty(exePath)
}
