package hooks

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
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
