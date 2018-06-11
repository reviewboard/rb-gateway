package tokens

import (
	"crypto/rand"
	"fmt"
	"net/http"
)

const (
	maxAttempts  = 10
	rawTokenSize = 32
)

// A memory-ony token store for unit testing.
//
// This store is not re-entrant and should only be used for unit tests. Tokens do not
// persist between restarts.
type MemoryStore map[string]bool

// Save the store.
//
// This is intentionally a no-op.
func (MemoryStore) Save() error {
	return nil
}

// Return the token from the request, if any.
//
// If there is no token associated with this request or the token is invalid
// `nil` will be returned instead.
//
// This method is not re-entrant.
func (store MemoryStore) Get(r *http.Request) *string {
	token := r.Header.Get(TokenHeader)

	if len(token) != TokenSize {
		return nil
	}

	if !store.Exists(token) {
		return nil
	}

	return &token
}

// Create a new, unique token.
//
// This may return an error if we cannot read from the OS random device or if we
// cannot generate a unique token after a number of attempts.
//
// This method is not re-entrant.
func (store MemoryStore) New() (*string, error) {
	var raw [rawTokenSize]byte

	for i := 0; i < maxAttempts; i++ {
		if _, err := rand.Read(raw[:]); err != nil {
			return nil, fmt.Errorf("Could not generate token: %s\n", err.Error())
		}

		token := fmt.Sprintf("%X", raw)
		if _, exists := store[token]; !exists {
			store[token] = true
			return &token, nil
		}
	}

	return nil, fmt.Errorf("Could not generate token after %d attempts.\n", maxAttempts)
}

// Return whether or not a token exists in the store.
//
// This method is not re-entrant.
func (store MemoryStore) Exists(token string) bool {
	if len(token) != TokenSize {
		return false
	}

	_, exists := store[token]
	return exists
}
