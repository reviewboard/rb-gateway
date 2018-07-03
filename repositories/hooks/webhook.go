package hooks

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"

	"github.com/reviewboard/rb-gateway/repositories/events"
)

type Webhook struct {
	// A unique ID for the webhook.
	Id string `json:"id"`

	// The URL that the webhook will request.
	Url string `json:"url"`

	// A secret used for generating an HMAC-SHA1 signature for the payload.
	Secret string `json:"secret"`

	// Whether or not the webhook is enabled.
	Enabled bool `json:"enabled"`

	// A sorted list of events that this webhook applies to.
	Events []string `json:"events"`

	// A sorted list of repository names that this webhook applies to.
	Repos []string `json:"repos"`
}

// Return an HMAC-SHA1 signature of the payload using the hook's secret.
func (hook Webhook) SignPayload(payload []byte) string {
	hmac := hmac.New(sha1.New, []byte(hook.Secret))
	hmac.Write(payload)

	return hex.EncodeToString(hmac.Sum(nil))
}

// Validate a hook.
func (hook Webhook) Validate(repos map[string]struct{}) error {
	if len(hook.Events) == 0 {
		return errors.New("Hook has no events.")
	} else {
		for _, event := range hook.Events {
			if !events.IsValidEvent(event) {
				return fmt.Errorf(`Invalid event: "%s".`, event)
			}
		}
	}

	if len(hook.Repos) == 0 {
		return errors.New("Hook has no repositories.")
	} else {
		for _, repo := range hook.Repos {
			if _, ok := repos[repo]; !ok {
				return fmt.Errorf(`Invalid repository: "%s".`, repo)
			}
		}
	}

	url, err := url.Parse(hook.Url)
	if err != nil {
		return fmt.Errorf("Invalid URL: %s", err.Error())
	}

	if url.Scheme != "http" && url.Scheme != "https" {
		return fmt.Errorf(`Invalid URL scheme "%s": only HTTP and HTTPS are supported.`,
			url.Scheme)
	}

	if len(hook.Secret) < 20 {
		return fmt.Errorf(`Secret is too short (%d bytes); secrets must be at least 20 bytes.`,
			len(hook.Secret))
	}

	return nil
}
