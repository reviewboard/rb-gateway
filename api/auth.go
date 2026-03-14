package api

import (
	"crypto/subtle"
	"encoding/csv"
	"errors"
	"net/http"
	"os"

	"golang.org/x/crypto/bcrypt"
)

// secretProvider returns the hashed password for a user, or "" if not found.
type secretProvider func(user string) string

// Create a new secret provider for htpasswd files.
//
// Unlike a lazy file provider, this loads the entire file up front.
//
// If the user wants to reload the `htpasswd` file, they need to trigger a full
// config reload (e.g., with `SIGHUP`).
func newHtpasswdSecretProvider(path string) (secretProvider, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	defer f.Close()

	csv := csv.NewReader(f)
	csv.Comma = ':'
	csv.Comment = '#'
	csv.TrimLeadingSpace = true

	records, err := csv.ReadAll()
	if err != nil {
		return nil, err
	}

	secrets := make(map[string]string)
	for _, record := range records {
		if len(record) != 2 {
			return nil, errors.New("Malformed htpasswd file")
		}

		secrets[record[0]] = record[1]
	}

	provider := func(user string) string {
		secret, ok := secrets[user]
		if ok {
			return secret
		}
		return ""
	}

	return provider, nil
}

// withBasicAuth wraps a handler that requires HTTP Basic Auth.
//
// The provided secretProvider is used to look up the hashed password for the
// given username. If the credentials are valid, the wrapped handler is called.
// Otherwise, a 401 response with a WWW-Authenticate header is returned.
func (api *API) withBasicAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if !ok {
			w.Header().Set("WWW-Authenticate", `Basic realm="RB Gateway"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		hashedPassword := api.secretProvider(username)
		if hashedPassword == "" {
			w.Header().Set("WWW-Authenticate", `Basic realm="RB Gateway"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if !checkPassword(password, hashedPassword) {
			w.Header().Set("WWW-Authenticate", `Basic realm="RB Gateway"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	}
}

// checkPassword verifies a plaintext password against a hash.
// Supports bcrypt (prefixed with $2a$, $2b$, $2y$) and plain text comparison.
func checkPassword(password, hashedPassword string) bool {
	// Try bcrypt first (most common for htpasswd)
	if len(hashedPassword) > 3 && hashedPassword[0] == '$' {
		err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
		return err == nil
	}

	// Fall back to constant-time plain text comparison
	return subtle.ConstantTimeCompare([]byte(password), []byte(hashedPassword)) == 1
}
