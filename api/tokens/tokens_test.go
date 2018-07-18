package tokens_test

import (
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/reviewboard/rb-gateway/api/tokens"
)

// Test generated tokens are unique.
func TestUniqueTokens(t *testing.T) {
	assert := assert.New(t)

	store, err := tokens.NewStore(":memory:")
	assert.Nil(err)

	tok, err := store.New()
	assert.Nil(err)

	tok2, err := store.New()
	assert.Nil(err)

	assert.NotEqual(tok, tok2)
}

// Testing MemoryStore.Get
func TestGetFromRequest(t *testing.T) {
	assert := assert.New(t)

	store, err := tokens.NewStore(":memory:")
	assert.Nil(err)

	tok, err := store.New()
	assert.Nil(err)
	assert.NotNil(tok)

	body := strings.NewReader("")
	request, err := http.NewRequest("GET", "/", body)
	assert.Nil(err)

	request.Header.Set(tokens.TokenHeader, *tok)

	result := store.Get(request)
	assert.NotNil(result)
	assert.Equal(result, tok)
}

// Testing round-tripping a FileStore.
func TestFileStoreLoadSave(t *testing.T) {
	assert := assert.New(t)

	tmpdir, err := ioutil.TempDir("", "rb-gateway-test-")
	assert.Nil(err)
	defer os.RemoveAll(tmpdir)

	storePath := filepath.Join(tmpdir, "tokens.dat")

	store, err := tokens.NewStore(storePath)
	assert.Nil(err)
	assert.NotNil(store)

	_, err = os.Stat(storePath)
	assert.NotNil(err)
	assert.True(os.IsNotExist(err))

	var tokenList = make([]string, 0, 10)

	for i := 0; i < 10; i++ {
		tok, err := store.New()
		assert.Nil(err)

		tokenList = append(tokenList, *tok)
	}

	err = store.Save()
	assert.Nil(err)

	_, err = os.Stat(storePath)
	assert.Nil(err)

	store, err = tokens.NewStore(storePath)
	assert.Nil(err)
	assert.NotNil(store)

	for _, tok := range tokenList {
		assert.True(store.Exists(tok))
	}
}

func TestFileStoreExistsEmpty(t *testing.T) {
	assert := assert.New(t)

	tmpfile, err := ioutil.TempFile("", "rb-gateway-tokens.dat-")
	assert.Nil(err)

	defer tmpfile.Close()

	store, err := tokens.NewStore(tmpfile.Name())
	assert.Nil(err)
	assert.NotNil(store)
}
