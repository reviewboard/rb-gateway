package commands

import (
	"log"
	"os"

	"github.com/reviewboard/rb-gateway/config"
)

// Reinstall hooks in all repositories.
func ReinstallHooks(configPath string) {
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatal("Could not parse configuration: ", err.Error())
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
			log.Printf(
				`An error occurred while installing hooks for repository "%s": %s`,
				repository.GetName(), err.Error())
		}
	}

	if len(errors) == 0 {
		errors = nil
	}

	return errors
}
