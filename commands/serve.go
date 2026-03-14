package commands

import (
	"log/slog"
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
		slog.Error("unable to load configuration file, see installation instructions at http://www.reviewboard.org/docs/rbgateway/latest/installation/",
			"path", configPath)
		os.Exit(1)
	}

	if cfg.TokenStorePath == ":memory:" {
		slog.Error("cannot use memory store outside of tests")
		os.Exit(1)
	}

	api, err := api.New(cfg)
	if err != nil {
		slog.Error("could not create API", "err", err)
		os.Exit(1)
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
		slog.Info("starting rb-gateway server", "port", cfg.Port)
		slog.Info("quit the server with CONTROL-C")

		server := api.Serve()

		select {
		case newCfg = <-configWatcher.NewConfig:
			slog.Info("detected configuration change, reloading...")

		case err := <-configWatcher.Errors:
			slog.Error("unexpected error", "err", err)
			os.Exit(1)

		case <-hup:
			slog.Info("received SIGHUP, reloading configuration...")

			var err error
			if newCfg, err = configWatcher.ForceReload(); err != nil {
				slog.Error("unexpected error", "err", err)
				os.Exit(1)
			}

		case <-interrupt:
			shouldExit = true
			signal.Reset(os.Interrupt)
			slog.Info("received SIGINT, shutting down...")
			slog.Info("CONTROL-C again to force quit")

		case <-terminate:
			shouldExit = true
			slog.Info("received SIGTERM, shutting down...")
		}

		err = api.Shutdown(server)
		if err != nil {
			slog.Error("error while shutting down the server", "err", err)
			os.Exit(1)
		}

		slog.Info("server shut down")

		if shouldExit {
			break
		}

		if newCfg != nil {
			if newCfg.TokenStorePath == ":memory:" {
				slog.Error("failed to reload configuration: cannot use memory store outside of tests")
				slog.Info("configuration was not reloaded")
			} else if err = api.SetConfig(newCfg); err != nil {
				slog.Error("failed to reload configuration", "err", err)
			} else {
				slog.Info("configuration reloaded")

				// If we have any new repositories, install hooks for them.
				// We do not need to force install because configPath has not changed.
				installHooks(cfg, configPath, false)
			}
		}
	}
}
