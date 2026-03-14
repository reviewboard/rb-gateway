package commands

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/reviewboard/rb-gateway/config"
	"github.com/reviewboard/rb-gateway/repositories"
	"github.com/reviewboard/rb-gateway/repositories/events"
	"github.com/reviewboard/rb-gateway/repositories/hooks"
)

// Trigger all webhooks that match the repository and event.
func TriggerWebhooks(configPath, repoName, event string) {
	cfg, err := config.Load(configPath)
	if err != nil {
		slog.Error("could not parse configuration", "err", err)
		os.Exit(1)
	}

	var repository repositories.Repository
	var exists bool

	if repository, exists = cfg.Repositories[repoName]; !exists {
		slog.Error("unknown repository", "repo", repoName)
		os.Exit(1)
	}

	if !events.IsValidEvent(event) {
		slog.Error("unknown event", "event", event)
		os.Exit(1)
	}

	store, err := hooks.LoadStore(cfg.WebhookStorePath, cfg.RepositorySet())

	if err != nil {
		slog.Error("could not load webhook store", "err", err)
		os.Exit(1)
	}

	payload, err := repository.ParseEventPayload(event, os.Stdin)
	if err != nil {
		slog.Error("could not parse event payload", "err", err)
		os.Exit(1)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	err = repositories.InvokeAllHooks(client, store, event, repository, payload)
	if err != nil {
		slog.Error("error invoking hooks", "err", err)
		os.Exit(1)
	}
}
