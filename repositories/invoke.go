package repositories

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/reviewboard/rb-gateway/repositories/events"
	"github.com/reviewboard/rb-gateway/repositories/hooks"
)

// Invoke all webhooks that match the given event and repository.
func InvokeAllHooks(
	client *http.Client,
	store hooks.WebhookStore,
	event string,
	repository Repository,
	payload events.Payload,
) error {
	if !events.IsValidEvent(event) {
		return fmt.Errorf(`Unknown event type "%s"`, event)
	}

	rawPayload, err := events.MarshalPayload(payload)
	if err != nil {
		return err
	}

	errs := store.ForEach(event, repository.GetName(), func(hook hooks.Webhook) error {
		err := invokeHook(client, event, repository, hook, rawPayload)
		if err != nil {
			slog.Error("error processing hook", "hook", hook.Id, "url", hook.Url, "err", err)
		}

		return err
	})

	if errs != nil {
		return fmt.Errorf("%d errors occurred wihle processing webhooks", len(errs))
	}

	return nil
}

// Invoke a webhook.
func invokeHook(
	client *http.Client,
	event string,
	repository Repository,
	hook hooks.Webhook,
	rawPayload []byte,
) error {
	req, err := http.NewRequest("POST", hook.Url, bytes.NewBuffer(rawPayload))
	if err != nil {
		return err
	}

	signature := hook.SignPayload(rawPayload)

	req.Header.Set("X-RBG-Signature", signature)
	req.Header.Set("X-RBG-Event", event)
	req.Header.Set("Content-Type", "application/json")

	slog.Info("dispatching webhook", "hook", hook.Id, "event", event, "repo", repository.GetName(), "url", hook.Url)

	rsp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer rsp.Body.Close()
	if rsp.StatusCode < 200 || rsp.StatusCode > 299 {
		body, err := io.ReadAll(rsp.Body)
		if err != nil {
			return err
		}

		slog.Warn("unexpected webhook response", "status", rsp.Status, "body", string(body))
	}

	return nil
}
