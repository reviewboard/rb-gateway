package repositories_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/reviewboard/rb-gateway/helpers"
)

func TestInstallGitHooks(t *testing.T) {
	assert := assert.New(t)

	repo, _ := helpers.CreateGitRepo(t, "git-repo")
	defer helpers.CleanupRepository(t, repo.Path)

	err := repo.InstallHooks("/tmp/config.json")
	if err != nil {
		assert.Nilf(err, "%s", err.Error())
	}

	dispatchPath := filepath.Join(repo.Path, ".git", "hooks", "post-receive")
	scriptPath := filepath.Join(repo.Path, ".git", "hooks", "post-receive.d", "99-rbgateway-push-event.sh")

	assert.DirExists(filepath.Join(repo.Path, ".git", "hooks"))
	assert.DirExists(filepath.Join(repo.Path, ".git", "hooks", "post-receive.d"))
	assert.FileExists(dispatchPath)
	assert.FileExists(scriptPath)

	exePath, err := filepath.Abs(os.Args[0])
	assert.Nil(err)

	rawContent, err := ioutil.ReadFile(scriptPath)
	assert.Nil(err)

	assert.Equal(fmt.Sprintf(
		"#!/bin/bash\n"+
			"exec %s --config /tmp/config.json trigger-webhooks git-repo push\n",
		exePath),
		string(rawContent))

	rawContent, err = ioutil.ReadFile(dispatchPath)
	assert.Nil(err)
	expected := `#!/bin/bash
# Run hooks in .git/hooks/post-receive.d
# This file was installed by rb-gateway.

HOOK_DIR=$(dirname $0)/post-receive.d

EXIT=0

if [ -d "$HOOK_DIR" ]; then
	STDIN=$(cat /dev/stdin)
	for HOOK in ${HOOK_DIR}/*; do
		if [ -x "$HOOK" ]; then
			echo -n "$STDIN" | "$HOOK" "$@"
		fi
	done
	LAST_EXIT=$?
	if [ $LAST_EXIT != 0 ]; then
		EXIT=$LAST_EXIT
	fi
fi

exit $EXIT
`
	content := string(rawContent)
	// The error message that go uses by default is unreadable.
	assert.Truef(expected == content,
		"expected:\n=========\n%s\n\t\t======\n\nactual:\n=======\n%s\n\t\t======", expected, content)
}

func TestInstallGitHooksQuoted(t *testing.T) {
	assert := assert.New(t)

	repo, _ := helpers.CreateGitRepo(t, "git-repo with a space")
	defer helpers.CleanupRepository(t, repo.Path)

	err := repo.InstallHooks("/tmp/config with a space.json")
	if err != nil {
		assert.Nil(err, err.Error())
	}

	dispatchPath := filepath.Join(repo.Path, ".git", "hooks", "post-receive")
	scriptPath := filepath.Join(repo.Path, ".git", "hooks", "post-receive.d", "99-rbgateway-push-event.sh")

	assert.DirExists(filepath.Join(repo.Path, ".git", "hooks"))
	assert.DirExists(filepath.Join(repo.Path, ".git", "hooks", "post-receive.d"))
	assert.FileExists(dispatchPath)
	assert.FileExists(scriptPath)

	exePath, err := filepath.Abs(os.Args[0])
	assert.Nil(err)

	content, err := ioutil.ReadFile(scriptPath)
	assert.Nil(err)

	assert.Equal(fmt.Sprintf(
		"#!/bin/bash\n"+
			"exec %s --config '/tmp/config with a space.json' trigger-webhooks 'git-repo with a space' push\n",
		exePath),
		string(content))
}

func TestInstallGitHooksPreexisting(t *testing.T) {
	assert := assert.New(t)

	repo, _ := helpers.CreateGitRepo(t, "git-repo")
	defer helpers.CleanupRepository(t, repo.Path)

	err := os.Mkdir(filepath.Join(repo.Path, ".git", "hooks"), 0777)
	assert.Nil(err)

	dispatchPath := filepath.Join(repo.Path, ".git", "hooks", "post-receive")
	origPath := filepath.Join(repo.Path, ".git", "hooks", "post-receive.d", "00-original-post-receive")
	scriptPath := filepath.Join(repo.Path, ".git", "hooks", "post-receive.d", "99-rbgateway-push-event.sh")

	err = ioutil.WriteFile(dispatchPath, []byte("#!/bin/true\n"), 0777)
	assert.Nil(err)

	assert.FileExists(dispatchPath)

	err = repo.InstallHooks("/tmp/config")
	if err != nil {
		assert.Nil(err, err.Error())
	}

	assert.DirExists(filepath.Join(repo.Path, ".git", "hooks"))
	assert.DirExists(filepath.Join(repo.Path, ".git", "hooks", "post-receive.d"))
	assert.FileExists(dispatchPath)
	assert.FileExists(scriptPath)
	assert.FileExists(origPath)

	content, err := ioutil.ReadFile(origPath)
	assert.Nil(err)

	assert.Equal("#!/bin/true\n", string(content))
}
