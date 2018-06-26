package main

import (
	"os"

	"github.com/alecthomas/kingpin"

	"github.com/reviewboard/rb-gateway/commands"
	"github.com/reviewboard/rb-gateway/config"
)

var (
	app        = kingpin.New("rb-gateway", "Repository API server.")
	configPath = app.Flag("config", "Path to configuration file.").Default(config.DefaultConfigPath).String()
)

func main() {
	kingpin.MustParse(app.Parse(os.Args[1:]))
	commands.Serve(*configPath)
}
