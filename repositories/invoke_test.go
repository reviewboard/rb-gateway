package repositories_test

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/reviewboard/rb-gateway/helpers"
	"github.com/reviewboard/rb-gateway/repositories"
	"github.com/reviewboard/rb-gateway/repositories/events"
)

func TestInvokeAllHooks(t *testing.T) {
	assert := assert.New(t)

	repo := &repositories.GitRepository{
		RepositoryInfo: repositories.RepositoryInfo{
			Name: "git-repo",
			Path: "does-not-exist",
		},
	}

	server, requestsChan := helpers.CreateRequestRecorder(t)
	defer server.Close()

	payload := events.PushPayload{
		Repository: "git-repo",
		Commits: []events.PushPayloadCommit{
			{
				Id:      "f00f00",
				Message: "Commit message",
				Target: events.PushPayloadCommitTarget{
					Branch: "master",
				},
			},
		},
	}

	store := helpers.CreateTestWebhookStore(server.URL)

	err := repositories.InvokeAllHooks(
		server.Client(),
		store,
		events.PushEvent,
		repo,
		payload)

	assert.Nil(err)

	request := helpers.AssertNumRequests(t, 1, requestsChan)[0]

	assert.Equal("/webhook-1", request.Request.URL.Path)
	assert.Equal("push", request.Request.Header.Get("X-RBG-Event"))

	json, err := events.MarshalPayload(payload)
	assert.Nil(err)

	expectedSignature := store["webhook-1"].SignPayload(json)

	assert.Equal(expectedSignature, request.Request.Header.Get("X-RBG-Signature"))
	assert.Equal(json, request.Body)
}

func TestInvokeAllHooksMultiple(t *testing.T) {
	assert := assert.New(t)

	repo := &repositories.GitRepository{
		RepositoryInfo: repositories.RepositoryInfo{
			Name: "git-repo",
			Path: "does-not-exist",
		},
	}

	server, requestsChan := helpers.CreateRequestRecorder(t)
	defer server.Close()

	payload := events.PushPayload{
		Repository: "git-repo",
		Commits: []events.PushPayloadCommit{
			{
				Id:      "f00f00",
				Message: "Commit message",
				Target: events.PushPayloadCommitTarget{
					Branch: "master",
				},
			},
		},
	}

	store := helpers.CreateTestWebhookStore(server.URL)
	hook := store["webhook-3"]
	hook.Enabled = true

	err := repositories.InvokeAllHooks(
		server.Client(),
		store,
		events.PushEvent,
		repo,
		payload)

	assert.Nil(err)

	requests := helpers.AssertNumRequests(t, 2, requestsChan)

	// The underlying implementation for dispatching webhooks iterates over a
	// map, for which iteration order is random. Sort the requests by URL so
	// that we can compare them to what we expect.
	sort.Slice(requests, func(i, j int) bool {
		return requests[i].Request.URL.Path < requests[j].Request.URL.Path
	})

	rawJson, err := events.MarshalPayload(payload)
	assert.Nil(err)
	json := string(rawJson)

	expectedHooks := []string{"webhook-1", "webhook-3"}

	for i, r := range requests {
		request := r.Request

		expectedHookId := expectedHooks[i]

		assert.Equal("/"+expectedHookId, request.URL.Path)

		hook := store[expectedHookId]
		assert.Equal(request.Header.Get("X-RBG-Signature"), hook.SignPayload(rawJson))
		assert.Equal(json, string(r.Body))
	}

}
