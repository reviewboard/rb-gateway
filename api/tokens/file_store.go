package tokens

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sync"
)

// A token store backed by a file on disk.
type FileStore struct {
	lock   sync.RWMutex
	path   string
	tokens MemoryStore
}

// Create a new store from the conents of file at the given path.
func NewFileStore(path string) (*FileStore, error) {
	if path == ":memory:" {
		panic("Cannot create FileStore in memory")
	}

	f, err := os.OpenFile(path, os.O_RDONLY, 0600)

	if err != nil && !os.IsNotExist(err) {
		log.Printf("Could not open token store at \"%s\": %s", path, err.Error())
		return nil, err
	} else {
		tokens := make(MemoryStore)

		if f != nil {
			defer f.Close()

			bytes, err := ioutil.ReadAll(f)
			if err != nil {
				return nil, err
			}

			if len(bytes) != 0 {
				var unmarshalled []string

				if err := json.Unmarshal(bytes, &unmarshalled); err != nil {
					return nil, err
				}

				for _, token := range unmarshalled {
					tokens[token] = true
				}
			}
		}

		store := FileStore{
			path:   path,
			tokens: tokens,
		}

		return &store, nil
	}
}

// Save the tokens in the store to the backing file.
func (store *FileStore) Save() error {
	store.lock.Lock()
	defer store.lock.Unlock()

	f, err := os.OpenFile(store.path, os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return err
	}

	tokens := make([]string, 0, len(store.tokens))

	for token := range store.tokens {
		tokens = append(tokens, token)
	}

	bytes, err := json.Marshal(tokens)
	if err != nil {
		return err
	}

	if _, err := f.Write(bytes); err != nil {
		return err
	}

	return nil

}

// Return the token from the request, if any.
//
// If there is no token associated with this request or the token is invalid
// `nil` will be returned instead.
func (store *FileStore) Get(r *http.Request) *string {
	store.lock.RLock()
	defer store.lock.RUnlock()

	return store.tokens.Get(r)
}

// Create a new, unique token.
//
// This may return an error if we cannot read from the OS random device or if we
// cannot generate a unique token after a number of attempts.
func (store *FileStore) New() (*string, error) {
	store.lock.Lock()
	defer store.lock.Unlock()

	tok, err := store.tokens.New()
	return tok, err
}

// Return whether or not a token exists in the store.
func (store *FileStore) Exists(token string) bool {
	store.lock.RLock()
	defer store.lock.RUnlock()

	return store.tokens.Exists(token)
}
