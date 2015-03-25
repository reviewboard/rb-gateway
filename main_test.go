package main

import (
	"runtime"
	"testing"
)

func checkFatal(t *testing.T, err error) {
	if err == nil {
		return
	}

	// Determine where the failure happened.
	_, file, line, success := runtime.Caller(1)
	if !success {
		t.Fatal()
	}

	t.Fatalf("Failed at %v:%v; %v", file, line, err)
}
