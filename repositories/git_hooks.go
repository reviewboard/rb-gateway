package repositories

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"text/template"

	"github.com/kballard/go-shellquote"

	"github.com/reviewboard/rb-gateway/repositories/events"
)

const (
	gitHookDispatchScriptTemplate = (`#!/bin/bash
# Run hooks in .git/hooks/{{ .HookName }}.d
# This file was installed by rb-gateway.

HOOK_DIR=$(dirname $0)/{{ .HookName }}.d

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
`)

	gitHookScriptTemplate = (`#!/bin/bash
exec {{ .ExePath }} --config {{ .ConfigPath }} trigger-webhooks {{ .Repository }} {{ .Event }}
`)
)

var (
	gitEvents = map[string]string{
		events.PushEvent: "post-receive",
	}
)

type gitHookData struct {
	ConfigPath string
	Event      string
	ExePath    string
	HookDir    string
	HookName   string
	Repository string
}

// Install all hooks for the given repository.
func (repo *GitRepository) InstallHooks(cfgPath string, force bool) (err error) {
	var commonDir string
	if commonDir, err = repo.commonDir(); err != nil {
		return
	}

	hookDir := filepath.Join(commonDir, "hooks")

	if _, err = ensureDir(hookDir); err != nil {
		return
	}

	var exePath string
	if exePath, err = filepath.Abs(os.Args[0]); err != nil {
		return
	}

	hookData := gitHookData{
		ConfigPath: shellquote.Join(cfgPath),
		ExePath:    shellquote.Join(exePath),
		HookDir:    shellquote.Join(hookDir),
		Repository: shellquote.Join(repo.Name),
	}

	for event, hookName := range gitEvents {
		hookData.Event = shellquote.Join(event)
		hookData.HookName = shellquote.Join(hookName)

		err = repo.installHook(hookDir, &hookData, force)
		if err != nil {
			return
		}
	}

	return
}

// Install a single repository hook.
//
// This function installs (1) a hook dispatch script to run any number of hooks
// per event and (2) a RB Gateway-specific script to call `trigger-webhook`. If
// a hook with the specified name already exists, it will be renamed and moved
// into the `hookname.d` directory.
//
// If `force` is `true`, the hooks will be installed over existing hooks if
// they already exist.
func (repo *GitRepository) installHook(hookDir string, hookData *gitHookData, force bool) (err error) {
	dispatchPath := filepath.Join(hookDir, hookData.HookName)
	scriptDir := filepath.Join(hookDir, fmt.Sprintf("%s.d", hookData.HookName))
	scriptPath := filepath.Join(scriptDir, fmt.Sprintf("99-rbgateway-%s-event.sh", hookData.Event))

	var created bool

	if created, err = ensureDir(scriptDir); err != nil {
		return
	}

	if created {
		renamedPath := filepath.Join(scriptDir, fmt.Sprintf("00-original-%s", hookData.HookName))

		// If there is an existing hook in .git/hooks/, we copy it into the
		// script dir so that it will still be executed after we install our
		// dispatcher.
		if _, err = os.Stat(dispatchPath); err != nil && !os.IsNotExist(err) {
			return
		} else if err == nil {
			if err = os.Rename(dispatchPath, renamedPath); err != nil {
				return
			}

			defer func() {
				// Something went wrong so we are going to try to restore the
				// filesystem to near its original state.
				log.Printf(`Restoring filesystem to original state for hook "%s"`, hookData.HookName)
				if err != nil {
					if err = os.Rename(renamedPath, dispatchPath); err != nil {
						log.Println("Could not restore filesystem after error: ", err.Error())
					}

				}
			}()
		}
	}

	// If the script to trigger `rbgateway trigger-webhooks` does not exist, create it.
	if _, err = os.Stat(scriptPath); force || os.IsNotExist(err) {
		t := template.Must(template.New(scriptPath).Parse(gitHookScriptTemplate))

		var f *os.File
		if f, err = os.OpenFile(scriptPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0700); err != nil {
			return
		}

		defer f.Close()

		if err = t.Execute(f, hookData); err != nil {
			return
		}
	} else if err != nil {
		return
	}

	// If the dispatch script does not exist, create it.
	if _, err = os.Stat(dispatchPath); force || os.IsNotExist(err) {
		t := template.Must(template.New(dispatchPath).Parse(gitHookDispatchScriptTemplate))

		var f *os.File
		if f, err = os.OpenFile(dispatchPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0700); err != nil {
			return
		}

		defer f.Close()

		if err = t.Execute(f, hookData); err != nil {
			return
		}
	} else if err != nil {
		return
	}

	return
}
