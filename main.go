package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
)

const (
	config = "config.json"
)

type logHTTPHandler struct {
	handler http.Handler
}

type loggedResponse struct {
	http.ResponseWriter
	status  int
	content []byte
}

func (l *loggedResponse) WriteHeader(status int) {
	l.status = status
	l.ResponseWriter.WriteHeader(status)
}

func (l *loggedResponse) Write(content []byte) (int, error) {
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
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal("Could not watch configuration: ", err)
	}

	if err = watcher.Add(config); err != nil {
		log.Fatal("Could not watch configuration: ", err)
	}

	LoadConfig(config)

	server := &http.Server{
		Addr:    ":" + strconv.Itoa(GetPort()),
		Handler: &logHTTPHandler{Route()},
	}

	hup := make(chan os.Signal, 1)
	signal.Notify(hup, syscall.SIGHUP)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	for {
		shouldExit := false
		log.Println("Starting rb-gateway server on port", GetPort())
		log.Println("Quit the server with CONTROL-C.")

		go func() {
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Fatal("listenAndServe: ", err)
			}
		}()

		select {
		case <-watcher.Events:
			log.Println("Detected configuration change, reloading...")

		case watchErr := <-watcher.Errors:
			log.Fatal("Unexpected error: ", watchErr)

		case <-hup:
			log.Println("Received SIGHUP, reloading configuration...")

		case <-interrupt:
			shouldExit = true
			signal.Reset(os.Interrupt)
			log.Println("Received SIGINT, shutting down...")
			log.Println("CONTROL-C again to force quit.")
		}

		/*
		 * This allows us to give the server a grace period for finishing
		 * in-progress requests before it closes all connections.
		 */
		ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
		server.Shutdown(ctx)
		log.Println("Server shut down.")

		if shouldExit {
			os.Exit(0)
		}

		LoadConfig(config)
		server.Addr = ":" + strconv.Itoa(GetPort())
	}
}
