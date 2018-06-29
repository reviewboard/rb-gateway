package helpers

import (
	"fmt"

	"github.com/reviewboard/rb-gateway/repositories/hooks"
)

// Create a hooks.WebhookStore for testing.
//
// The target URLs will all be based off the given `baseUrl`.
func CreateTestWebhookStore(baseUrl string) hooks.WebhookStore {
	return hooks.WebhookStore{
		"webhook-1": &hooks.Webhook{
			Id:      "wehook-1",
			Url:     fmt.Sprintf("%s/webhook-1", baseUrl),
			Secret:  "top-secret-1",
			Enabled: true,
			Events:  []string{"push"},
			Repos:   []string{"git-repo"},
		},
		"webhook-2": &hooks.Webhook{
			Id:      "wehook-2",
			Url:     fmt.Sprintf("%s/webhook-2", baseUrl),
			Secret:  "top-secret-2",
			Enabled: true,
			Events:  []string{"push"},
			Repos:   []string{"hg-repo"},
		},
		"webhook-3": &hooks.Webhook{
			Id:      "wehook-3",
			Url:     fmt.Sprintf("%s/webhook-3", baseUrl),
			Secret:  "top-secret-3",
			Enabled: false,
			Events:  []string{"push"},
			Repos:   []string{"git-repo"},
		},
		"webhook-4": &hooks.Webhook{
			Id:      "wehook-3",
			Url:     fmt.Sprintf("%s/webhook-4", baseUrl),
			Secret:  "top-secret-4",
			Enabled: false,
			Events:  []string{"other-event"},
			Repos:   []string{"git-repo"},
		},
	}
}
