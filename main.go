package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/reviewboard/rb-gateway/api"
	"github.com/reviewboard/rb-gateway/config"
)

func main() {
	var cfg *config.Config
	configWatcher := config.Watch(config.DefaultConfigPath)

	select {
	case cfg = <-configWatcher.NewConfig:
		break

	case err := <-configWatcher.Errors:
		log.Fatal("Could not watch configuration file: ", err)
	}

	api := api.New(*cfg)

	hup := make(chan os.Signal, 1)
	signal.Notify(hup, syscall.SIGHUP)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

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
		}

		/*
		 * This allows us to give the server a grace period for finishing
		 * in-progress requests before it closes all connections.
		 */
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		server.Shutdown(ctx)
		cancel()
		log.Println("Server shut down.")

		if shouldExit {
			break
		}

		if newCfg != nil {
			api.SetConfig(*newCfg)
			log.Println("Configuration reloaded.")
		}
	}
}
