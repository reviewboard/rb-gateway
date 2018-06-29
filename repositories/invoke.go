package repositories

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
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
			log.Printf(`Error ocurred while processing hook "%s" for URL "%s": %s`,
				hook.Id, hook.Url, err.Error())
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

	log.Printf(`Dispatching webhook "%s" for event "%s" for repository "%s" to URL "%s"`,
		hook.Id, event, repository.GetName(), hook.Url)

	rsp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer rsp.Body.Close()
	if rsp.StatusCode < 200 || rsp.StatusCode > 299 {
		log.Printf("Expected status 2XX, received %s.", rsp.Status)
		body, err := ioutil.ReadAll(rsp.Body)
		if err != nil {
			return err
		}

		log.Printf("Response body: %s", body)
	}

	return nil
}
