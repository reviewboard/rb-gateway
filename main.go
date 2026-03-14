package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/reviewboard/rb-gateway/commands"
	"github.com/reviewboard/rb-gateway/config"
)

// version is set at build time via -ldflags.
var version = "dev"

func usage() {
	fmt.Fprintf(os.Stderr, `rb-gateway - Repository API server.

Usage:
  rb-gateway [--config PATH] <command> [args...]

Commands:
  serve               Start the API server (default).
  trigger-webhooks    Trigger matching webhooks.
  reinstall-hooks     Re-install hook scripts if the configuration path has changed.

Flags:
  --config PATH       Path to configuration file (default: %s).
`, config.DefaultConfigPath)
}

func main() {
	configPath := flag.String("config", config.DefaultConfigPath, "Path to configuration file.")
	showVersion := flag.Bool("version", false, "Print version and exit.")
	flag.Usage = usage
	flag.Parse()

	if *showVersion {
		fmt.Println(version)
		return
	}

	cmd := flag.Arg(0)
	if cmd == "" {
		cmd = "serve"
	}

	switch cmd {
	case "serve":
		commands.Serve(*configPath)

	case "trigger-webhooks":
		args := flag.Args()
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "Usage: rb-gateway trigger-webhooks <repository> <event>")
			os.Exit(1)
		}
		commands.TriggerWebhooks(*configPath, args[1], args[2])

	case "reinstall-hooks":
		commands.ReinstallHooks(*configPath)

	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", cmd)
		usage()
		os.Exit(1)
	}
}
