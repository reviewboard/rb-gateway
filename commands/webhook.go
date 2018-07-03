package commands

import (
	"log"
	"net/http"
	"os"

	"github.com/reviewboard/rb-gateway/config"
	"github.com/reviewboard/rb-gateway/repositories"
	"github.com/reviewboard/rb-gateway/repositories/events"
	"github.com/reviewboard/rb-gateway/repositories/hooks"
)

// Trigger all webhooks that match the repository and event.
func TriggerWebhooks(configPath, repoName, event string) {
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatal("Could not parse configuration: ", err.Error())
	}

	var repository repositories.Repository
	var exists bool

	if repository, exists = cfg.Repositories[repoName]; !exists {
		log.Fatalf(`Unknown repository: "%s".`, repoName)
	}

	if !events.IsValidEvent(event) {
		log.Fatalf(`Unknown event: "%s"`, event)
	}

	store, err := hooks.LoadStore(cfg.WebhookStorePath, cfg.RepositorySet())

	if err != nil {
		log.Fatal("Could not load webhook store: ", err.Error())
	}

	payload, err := repository.ParseEventPayload(event, os.Stdin)
	if err != nil {
		log.Fatal("Could not parse event payload: ", err.Error())
	}

	err = repositories.InvokeAllHooks(http.DefaultClient, store, event, repository, payload)
	if err != nil {
		log.Fatal(err.Error())
	}
}
