package main

import (
	"log"
	"net/http"
	"strconv"
)

type logHTTPHandler struct {
	handler http.Handler
}

type loggedResponse struct {
	http.ResponseWriter
	status  int
	content []byte
}

func (l *loggedResponse) writeHeader(status int) {
	l.status = status
	l.ResponseWriter.WriteHeader(status)
}

func (l *loggedResponse) write(content []byte) (int, error) {
	l.content = content
	return l.ResponseWriter.Write(content)
}

// ServeHTTP intercepts the default http.Handler implementation in order to
// handle HTTP request and response logging. It provides a default response
// containing a 200 OK status, and an empty byte array as the content, if not
// specified in the Responsewriter.
//
// It logs the request status, method, and URL, and the response status and
// content length.
func (h *logHTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	lw := &loggedResponse{ResponseWriter: w, status: 200, content: []byte{}}
	h.handler.ServeHTTP(lw, r)
	log.Printf("%s %s %s status:%d content-length:%d",
		r.RemoteAddr, r.Method, r.URL, lw.status, len(lw.content))
}

func main() {
	LoadConfig()

	Route()

	log.Println("Starting rb-gateway server at port", GetPort())
	log.Println("Quit the server with CONTROL-C.")
	err := http.ListenAndServe(":"+strconv.Itoa(GetPort()),
		&logHTTPHandler{http.DefaultServeMux})
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
