package main

import (
	"os"

	"github.com/alecthomas/kingpin"

	"github.com/reviewboard/rb-gateway/commands"
	"github.com/reviewboard/rb-gateway/config"
)

var (
	app        = kingpin.New("rb-gateway", "Repository API server.")
	configPath = app.Flag("config", "Path to configuration file.").
			Default(config.DefaultConfigPath).
			String()

	serve = app.Command("serve", "Start the API server.").Default()

	webhook  = app.Command("trigger-webhooks", "Trigger matching webhooks.")
	repoName = webhook.Arg("repository", "The name of the repository to trigger the webhook for.").
			Required().
			String()
	event = webhook.Arg("event", "The name of the event.").
		Required().
		String()

	reinstallHooks = app.Command("reinstall-hooks", "Re-install hook scripts if  the configuration path has changed.")
)

func main() {
	switch kingpin.MustParse(app.Parse(os.Args[1:])) {
	case serve.FullCommand():
		commands.Serve(*configPath)

	case webhook.FullCommand():
		commands.TriggerWebhooks(*configPath, *repoName, *event)

	case reinstallHooks.FullCommand():
		commands.ReinstallHooks(*configPath)
	}
}
