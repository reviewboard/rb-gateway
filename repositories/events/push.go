package events

// A payload for a push event.
type PushPayload struct {
	// The repository where the event occurred.
	Repository string `json:"repository"`

	// The commits that were pushed.
	Commits []PushPayloadCommit `json:"commits"`
}

// A commit that is part of the push.
type PushPayloadCommit struct {
	// The commit ID.
	Id string `json:"id"`

	// The commit message.
	Message string `json:"message"`

	// The targets the commit was pushed to.
	Target PushPayloadCommitTarget `json:"target"`
}

// A target for a push.
type PushPayloadCommitTarget struct {
	// The branch the commit was pushed to, if any.
	Branch string `json:"branch,omitempty"`

	// The bookmarks that point at the commit, if any.
	//
	// This can only be non-empty for Mercurial.
	Bookmarks []string `json:"bookmarks,omitempty"`

	// The tags that point at the commit, if any.
	Tags []string `json:"tags,omitempty"`
}

// Return the event the payload corresponds to.
func (_ PushPayload) GetEvent() string {
	return PushEvent
}

// Return the repository where the event occurred.
func (p PushPayload) GetRepository() string {
	return p.Repository
}

// Return the contents of the payload.
func (p PushPayload) GetContent() (string, interface{}) {
	return "commits", p.Commits
}
