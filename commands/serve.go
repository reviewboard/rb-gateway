package commands

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/reviewboard/rb-gateway/api"
	"github.com/reviewboard/rb-gateway/config"
)

func Serve(configPath string) {
	var cfg *config.Config
	configWatcher := config.Watch(configPath)

	select {
	case cfg = <-configWatcher.NewConfig:
		installHooks(cfg, configPath, false)
		break

	case <-configWatcher.Errors:
		log.Fatalf("Unable to load configuration file %s. See installation instructions at http://www.reviewboard.org/docs/rbgateway/latest/installation/",
			configPath)
	}

	if cfg.TokenStorePath == ":memory:" {
		log.Fatal("Cannot use memory store outside of tests.")
	}

	api, err := api.New(cfg)
	if err != nil {
		log.Fatalf("Could not create API: %s", err.Error())
	}

	hup := make(chan os.Signal, 1)
	signal.Notify(hup, syscall.SIGHUP)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	terminate := make(chan os.Signal, 1)
	signal.Notify(terminate, syscall.SIGTERM)

	for {
		var newCfg *config.Config = nil
		shouldExit := false
		log.Println("Starting rb-gateway server on port", cfg.Port)
		log.Println("Quit the server with CONTROL-C.")

		server := api.Serve()

		select {
		case newCfg = <-configWatcher.NewConfig:
			log.Println("Detected configuration change, reloading...")

		case err := <-configWatcher.Errors:
			log.Fatal("Unexpected error: ", err)

		case <-hup:
			log.Println("Received SIGHUP, reloading configuration...")

			var err error
			if newCfg, err = configWatcher.ForceReload(); err != nil {
				log.Fatal("Unexpected error: ", err)
			}

		case <-interrupt:
			shouldExit = true
			signal.Reset(os.Interrupt)
			log.Println("Received SIGINT, shutting down...")
			log.Println("CONTROL-C again to force quit.")

		case <-terminate:
			shouldExit = true
			log.Println("Received SIGTERM, shutting down...")
		}

		err = api.Shutdown(server)
		if err != nil {
			log.Fatalf("An error occurred while shutting down the server: %s", err.Error())
		}

		log.Println("Server shut down.")

		if shouldExit {
			break
		}

		if newCfg != nil {
			if newCfg.TokenStorePath == ":memory:" {
				log.Println("Failed to reload configuration: cannot use memory store outside of tests.")
				log.Println("Configuration was not reloaded.")
			} else if err = api.SetConfig(newCfg); err != nil {
				log.Printf("Failed to reload configuration: %s\n", err.Error())
			} else {
				log.Println("Configuration reloaded.")

				// If we have any new repositories, install hooks for them.
				// We do not need to force install because configPath has not changed.
				installHooks(cfg, configPath, false)
			}
		}
	}
}
