package hooks_test

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/reviewboard/rb-gateway/repositories/hooks"
)

func TestSignPayload(t *testing.T) {
	assert := assert.New(t)

	secret := strings.Repeat("s", 20)
	hook := hooks.Webhook{
		Secret: secret,
	}

	payload := []byte(`{"event":"push","repository":"repo"}`)

	signature := hook.SignPayload(payload)

	// Verify against a known-good HMAC-SHA1 computation.
	mac := hmac.New(sha1.New, []byte(secret))
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))

	assert.Equal(expected, signature)
	assert.Equal(40, len(signature), "HMAC-SHA1 hex digest should be 40 characters")
}

func TestSignPayloadEmptyPayload(t *testing.T) {
	assert := assert.New(t)

	hook := hooks.Webhook{
		Secret: strings.Repeat("s", 20),
	}

	signature := hook.SignPayload([]byte{})
	assert.NotEmpty(signature, "Signature of empty payload should still produce a digest")
	assert.Equal(40, len(signature))
}

func TestSignPayloadDifferentSecrets(t *testing.T) {
	assert := assert.New(t)

	payload := []byte("test payload")

	hook1 := hooks.Webhook{Secret: strings.Repeat("a", 20)}
	hook2 := hooks.Webhook{Secret: strings.Repeat("b", 20)}

	sig1 := hook1.SignPayload(payload)
	sig2 := hook2.SignPayload(payload)

	assert.NotEqual(sig1, sig2, "Different secrets should produce different signatures")
}

func TestValidateValid(t *testing.T) {
	assert := assert.New(t)

	repos := map[string]struct{}{
		"repo-1": {},
	}

	hook := hooks.Webhook{
		Id:      "hook-1",
		Url:     "https://example.com/webhook",
		Secret:  strings.Repeat("s", 20),
		Enabled: true,
		Events:  []string{"push"},
		Repos:   []string{"repo-1"},
	}

	assert.Nil(hook.Validate(repos))
}

func TestValidateNoEvents(t *testing.T) {
	assert := assert.New(t)

	repos := map[string]struct{}{"repo-1": {}}

	hook := hooks.Webhook{
		Url:    "https://example.com/webhook",
		Secret: strings.Repeat("s", 20),
		Events: []string{},
		Repos:  []string{"repo-1"},
	}

	err := hook.Validate(repos)
	assert.NotNil(err)
	assert.Contains(err.Error(), "no events")
}

func TestValidateInvalidEvent(t *testing.T) {
	assert := assert.New(t)

	repos := map[string]struct{}{"repo-1": {}}

	hook := hooks.Webhook{
		Url:    "https://example.com/webhook",
		Secret: strings.Repeat("s", 20),
		Events: []string{"nonexistent-event"},
		Repos:  []string{"repo-1"},
	}

	err := hook.Validate(repos)
	assert.NotNil(err)
	assert.Contains(err.Error(), "Invalid event")
}

func TestValidateNoRepos(t *testing.T) {
	assert := assert.New(t)

	repos := map[string]struct{}{"repo-1": {}}

	hook := hooks.Webhook{
		Url:    "https://example.com/webhook",
		Secret: strings.Repeat("s", 20),
		Events: []string{"push"},
		Repos:  []string{},
	}

	err := hook.Validate(repos)
	assert.NotNil(err)
	assert.Contains(err.Error(), "no repositories")
}

func TestValidateInvalidRepo(t *testing.T) {
	assert := assert.New(t)

	repos := map[string]struct{}{"repo-1": {}}

	hook := hooks.Webhook{
		Url:    "https://example.com/webhook",
		Secret: strings.Repeat("s", 20),
		Events: []string{"push"},
		Repos:  []string{"nonexistent-repo"},
	}

	err := hook.Validate(repos)
	assert.NotNil(err)
	assert.Contains(err.Error(), "Invalid repository")
}

func TestValidateInvalidURL(t *testing.T) {
	assert := assert.New(t)

	repos := map[string]struct{}{"repo-1": {}}

	hook := hooks.Webhook{
		Url:    "ftp://example.com/webhook",
		Secret: strings.Repeat("s", 20),
		Events: []string{"push"},
		Repos:  []string{"repo-1"},
	}

	err := hook.Validate(repos)
	assert.NotNil(err)
	assert.Contains(err.Error(), "Invalid URL scheme")
}

func TestValidateSecretTooShort(t *testing.T) {
	assert := assert.New(t)

	repos := map[string]struct{}{"repo-1": {}}

	hook := hooks.Webhook{
		Url:    "https://example.com/webhook",
		Secret: "short",
		Events: []string{"push"},
		Repos:  []string{"repo-1"},
	}

	err := hook.Validate(repos)
	assert.NotNil(err)
	assert.Contains(err.Error(), "Secret is too short")
}
