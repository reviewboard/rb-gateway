package helpers

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Execute the funtion `f` with the given environment variables set.
//
// This helper method temporarily merges the variables in `vars` into the
// current environment, executes `f`, and restores the original state of the
// environment.
func WithEnv(t *testing.T, vars map[string]string, f func()) {
	t.Helper()
	assert := assert.New(t)
	origVars := make(map[string]string)

	for key, val := range vars {
		origVars[key] = os.Getenv(key)
		assert.Nil(os.Setenv(key, val))
	}

	defer func() {
		for key, val := range origVars {
			if val == "" {
				assert.Nil(os.Unsetenv(key))
			} else {
				assert.Nil(os.Setenv(key, val))
			}
		}
	}()

	f()
}
