package hooks_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/reviewboard/rb-gateway/repositories/hooks"
)

func TestReadStore(t *testing.T) {
	assert := assert.New(t)

	reader := strings.NewReader(`[
		{
			"id": "webhook-1",
			"url": "http://example.com",
			"secret": "top-secret",
			"enabled": true,
			"events": ["push", "invalid-1"],
			"repos": ["repo-1"]
		},
		{
			"id": "webhook-2",
			"url": "http://example.com",
			"secret": "top-secret",
			"enabled": false,
			"events": ["push"],
			"repos": ["repo-2", "invalid-1", "repo-1"]
		},
		{
			"id": "webhook-3",
			"url": "http://example.com",
			"secret": "top-secret",
			"enabled": true,
			"events": ["invalid-1", "invalid-2"],
			"repos": ["invalid-1", "invalid-2"]
		}
	]`)

	repos := map[string]struct{}{
		"repo-1": struct{}{},
		"repo-2": struct{}{},
	}

	store, err := hooks.ReadStore(reader, repos)
	assert.Nil(err)
	assert.NotNil(store)

	expected := []hooks.Webhook{
		{
			Id:      "webhook-1",
			Url:     "http://example.com",
			Secret:  "top-secret",
			Enabled: true,
			Events:  []string{"push"},
			Repos:   []string{"repo-1"},
		},
		{
			Id:      "webhook-2",
			Url:     "http://example.com",
			Secret:  "top-secret",
			Enabled: false,
			Events:  []string{"push"},
			Repos:   []string{"repo-1", "repo-2"},
		},
	}

	assert.Equal(2, len(store))
	for _, hook := range expected {
		assert.Contains(store, hook.Id)

		parsed := store[hook.Id]

		assert.Equal(hook.Id, parsed.Id)
		assert.Equal(hook.Url, parsed.Url)
		assert.Equal(hook.Secret, parsed.Secret)
		assert.Equal(hook.Enabled, parsed.Enabled)
		assert.Equal(hook.Events, parsed.Events)
		assert.Equal(hook.Repos, parsed.Repos)
	}

	var buf strings.Builder
	assert.Nil(store.Write(&buf))

	reader = strings.NewReader(buf.String())

	store, err = hooks.ReadStore(reader, repos)
	assert.Nil(err)
	assert.NotNil(store)

	assert.Equal(2, len(store))
	for _, hook := range expected {
		assert.Contains(store, hook.Id)

		parsed := store[hook.Id]

		assert.Equal(hook.Id, parsed.Id)
		assert.Equal(hook.Url, parsed.Url)
		assert.Equal(hook.Secret, parsed.Secret)
		assert.Equal(hook.Enabled, parsed.Enabled)
		assert.Equal(hook.Events, parsed.Events)
		assert.Equal(hook.Repos, parsed.Repos)
	}

}
