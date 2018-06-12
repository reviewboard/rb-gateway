package api

import (
	"encoding/csv"
	"errors"
	"os"

	auth "github.com/abbot/go-http-auth"
)

// Create a new secret provider for htpasswd files.
//
// Unlike `auth.HtpasswdFileProvider`, this loads the entire file up front and
// (therefore) does not handle reloads. The one provided by `go-http-auth` does
// not do any I/O upfront and will trigger a `panic` if we attempt to
// authenticate and the file does not exist.
//
// If the user wants to reload the `htpasswd` file, they need to trigger a full
// config reload (e.g., with `SIGHUP`).
func newHtpasswdSecretProvider(path string) (auth.SecretProvider, error) {
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

	provider := func(user, realm string) string {
		secret, ok := secrets[user]

		if ok {
			return secret
		} else {
			return ""
		}
	}

	return provider, nil
}
