package commands

import (
	"log/slog"
	"os"

	"github.com/reviewboard/rb-gateway/config"
)

// Reinstall hooks in all repositories.
func ReinstallHooks(configPath string) {
	cfg, err := config.Load(configPath)
	if err != nil {
		slog.Error("could not parse configuration", "err", err)
		os.Exit(1)
	}

	errors := installHooks(cfg, configPath, true)
	if len(errors) != 0 {
		os.Exit(1)
	}
}

// Install hooks for all the repositories specified by cfg.
func installHooks(cfg *config.Config, configPath string, force bool) []error {
	errors := []error{}

	for _, repository := range cfg.Repositories {
		if err := repository.InstallHooks(configPath, force); err != nil {
			errors = append(errors, err)
			slog.Error("error installing hooks for repository",
				"repo", repository.GetName(), "err", err)
		}
	}

	if len(errors) == 0 {
		errors = nil
	}

	return errors
}
