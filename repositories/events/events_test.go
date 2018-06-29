package events_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/reviewboard/rb-gateway/repositories/events"
)

func TestMarshalPushPayload(t *testing.T) {
	assert := assert.New(t)

	payload := events.PushPayload{
		Repository: "foo",
		Commits: []events.PushPayloadCommit{
			{
				Id:      "abababab",
				Message: "Commit message 1",
				Target: events.PushPayloadCommitTarget{
					Branch: "master",
					Tags:   []string{"v1"},
				},
			},
			{
				Id:      "cdcdcdcd",
				Message: "Commit message 2",
				Target: events.PushPayloadCommitTarget{
					Branch: "dev",
				},
			},
			{
				Id:      "efefefef",
				Message: "Commit message 3",
				Target: events.PushPayloadCommitTarget{
					Branch:    "default",
					Bookmarks: []string{"my-bookmark"},
					Tags:      []string{"dev", "foo"},
				},
			},
		},
	}

	bytes, err := events.MarshalPayload(payload)
	assert.Nil(err)
	assert.NotNil(bytes)

	expected := `{
	"event": "push",
	"repository": "foo",
	"commits": [
		{
			"id": "abababab",
			"message": "Commit message 1",
			"target": {
				"branch": "master",
				"tags": [
					"v1"
				]
			}
		},
		{
			"id": "cdcdcdcd",
			"message": "Commit message 2",
			"target": {
				"branch": "dev"
			}
		},
		{
			"id": "efefefef",
			"message": "Commit message 3",
			"target": {
				"branch": "default",
				"bookmarks": [
					"my-bookmark"
				],
				"tags": [
					"dev",
					"foo"
				]
			}
		}
	]
}
`

	assert.Equal(expected, string(bytes))
}
