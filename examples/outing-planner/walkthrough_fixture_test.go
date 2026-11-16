//go:build integration

package main

import (
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"
)

// The walkthrough launches this fixture-only entry point once and reuses its
// listener across the ordinary planner commands.
func TestOutingWalkthroughFixtureServer(t *testing.T) {
	ready, serve := os.LookupEnv("ALAN_OUTING_WALKTHROUGH_READY")
	if !serve {
		return
	}
	fixture, _, err := readFixture(os.Getenv("ALAN_OUTING_WALKTHROUGH_FIXTURE"))
	if err != nil {
		t.Fatal(err)
	}
	listener, err := listenForFixture()
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	ctx, stop := signal.NotifyContext(t.Context(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	handler := newFixtureHandler(fixture, "celsius")
	server := &http.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second}
	done := make(chan error, 1)
	go func() { done <- server.Serve(listener) }()
	t.Cleanup(func() { _ = server.Close() })

	endpoint := "http://" + listener.Addr().String() + fixturePath + "\n"
	if err := os.WriteFile(ready+".tmp", []byte(endpoint), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(ready+".tmp", ready); err != nil {
		t.Fatal(err)
	}
	select {
	case <-ctx.Done():
		if err := server.Close(); err != nil {
			t.Fatal(err)
		}
		err = <-done
	case err = <-done:
	}
	if !errors.Is(err, http.ErrServerClosed) {
		t.Fatalf("fixture server stopped: %v", err)
	}
	if calls, unexpected := handler.results(); unexpected != nil {
		t.Fatalf("fixture received an unexpected request: %v", unexpected)
	} else {
		t.Logf("fixture accepted %d requests", calls)
	}
}
