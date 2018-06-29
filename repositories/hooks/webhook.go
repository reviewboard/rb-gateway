package hooks

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
