package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/JakeFAU/alan/internal/boundary"
	"github.com/JakeFAU/alan/internal/fixture"
	"io"
	"net"
	"net/http"
	"os"
	"time"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	flags := flag.NewFlagSet("alan-resource", flag.ContinueOnError)
	path := flags.String("db", "", "independent SQLite store")
	token := flags.String("token", "", "process-start identity")
	address := flags.String("listen", "127.0.0.1:0", "loopback listen address")
	if err := flags.Parse(os.Args[1:]); err != nil {
		return err
	}
	if *path == "" || *token == "" || flags.NArg() != 0 {
		return fmt.Errorf("db and token required")
	}
	host, _, err := net.SplitHostPort(*address)
	if err != nil || net.ParseIP(host) == nil || !net.ParseIP(host).IsLoopback() {
		return fmt.Errorf("literal loopback address required")
	}
	ctx, err := boundary.FromEnvironment(context.Background(), *path)
	if err != nil {
		return err
	}
	resource, err := fixture.Open(ctx, *path, *token)
	if err != nil {
		return err
	}
	defer resource.Close()
	listener, err := net.Listen("tcp", *address)
	if err != nil {
		return err
	}
	defer listener.Close()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/apply" {
			http.Error(w, "unsupported protocol", http.StatusNotFound)
			return
		}
		defer r.Body.Close()
		decoder := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
		decoder.DisallowUnknownFields()
		var request fixture.Request
		if err := decoder.Decode(&request); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		var extra any
		if decoder.Decode(&extra) != io.EOF {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		receipt, err := resource.Apply(r.Context(), request)
		if err != nil {
			status := http.StatusBadRequest
			if errors.Is(err, fixture.ErrConflict) {
				status = http.StatusConflict
			}
			http.Error(w, err.Error(), status)
			return
		}
		event := boundary.Event{Boundary: "resource.applied", Phase: "post-commit", Subject: request.Subject, Generation: request.Generation, Epoch: request.Epoch, Sequence: request.Sequence, Intent: request.Key, Attempt: request.Authorization}
		if err = boundary.Reached(boundary.Inherit(r.Context(), ctx), event); err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(receipt)
	})
	server := &http.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second}
	if err = json.NewEncoder(os.Stdout).Encode(map[string]any{"URL": "http://" + listener.Addr().String() + "/apply", "Token": *token, "Store": *path, "PID": os.Getpid()}); err != nil {
		return err
	}
	return server.Serve(listener)
}
