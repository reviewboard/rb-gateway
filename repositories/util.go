package repositories

import (
	"os"
	"path/filepath"
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

// Return the absolute path of the rb-gateway executable.
func getExePath() (exePath string, err error) {
	exePath, err = filepath.Abs(os.Args[0])
	return
}
