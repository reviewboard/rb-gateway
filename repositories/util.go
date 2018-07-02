package repositories

import (
	"os"
)

// Ensure the directory exists, creating it if it doesn't.
func ensureDir(dir string) (created bool, err error) {
	if _, err = os.Stat(dir); os.IsNotExist(err) {
		if err = os.Mkdir(dir, 0700); err == nil {
			created = true
		}
	}

	return
}
