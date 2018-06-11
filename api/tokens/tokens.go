package tokens

import (
	"net/http"
)

const (
	TokenHeader = "PRIVATE-TOKEN"
	TokenSize   = 64
)

// A generic token store.
type TokenStore interface {
	Save() error
	Get(r *http.Request) *string
	New() (*string, error)
	Exists(string) bool
}

// Create a new TokenStore
//
// If the special path ":memory:" is used, an in-memory store will be returned.
// However, in-memory stores should only be used for testing as they are not
// re-entrant.
func NewStore(path string) (store TokenStore, err error) {
	if path == ":memory:" {
		store = make(MemoryStore)
		err = nil
	} else {
		store, err = NewFileStore(path)
	}

	return
}
