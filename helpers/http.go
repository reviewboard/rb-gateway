package helpers

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const (
	maxRequestBufferSize = 100
)

// A request recorded from `CreateRequestRecorder`.
type RecordedRequest struct {
	// The recorded request.
	Request *http.Request

	// The recorded request body.
	//
	// The server will call `Close()` on the body when it has finished
	// processing the request, so it will no longer be readable on the request
	// itself.
	Body []byte
}

// Create a request recorded.
//
// The caller is responsible for shutting down the server.
//
// Due to channels being bounded, there is a maximum of 100 recorded requests
// before the server will start blocking requests.
//
// ```go
// func Test(t *testing.T) {
//     assert := assert.New(t)
//
//     server, reqs := helpers.CreateRequestRecorder(t)
//     defer server.Close()
//
//     // Make a request against the server.
//     rsp, err := http.Get(server.URL + "/test-url")
//     assert.Nil(t, err)
//
//     Retrieve the recorded request.
//     recorded := <- reqs
//     assert.Equal([]byte("Hello, world!"), recorded.Body)
// }
// ```
func CreateRequestRecorder(t *testing.T) (*httptest.Server, <-chan RecordedRequest) {
	ch := make(chan RecordedRequest, maxRequestBufferSize)
	server := httptest.NewServer(requestRecorder(t, ch))

	return server, ch
}

func requestRecorder(t *testing.T, send chan<- RecordedRequest) http.Handler {
	return http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Helper()

		body, err := ioutil.ReadAll(r.Body)
		assert.Nil(t, err)
		send <- RecordedRequest{
			Request: r,
			Body:    body,
		}
	})
}

// Require that the channel has at least `num` requests recorded.
//
// There is a 5 second timeout before the function will fail.
func AssertNumRequests(t *testing.T, num int, recv <-chan RecordedRequest) []RecordedRequest {
	requests := make([]RecordedRequest, 0, num)

	for i := 0; i < num; i++ {
		select {
		case request := <-recv:
			requests = append(requests, request)

		case <-time.After(5 * time.Second):
			t.Fatalf("Timed out waiting for request %d", i)
		}
	}

	return requests
}
