// Command consumer exercises the Phase 2C construction using public imports.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/JakeFAU/alan/examples/reconciliation"
	"github.com/JakeFAU/alan/runtime"
	"os"
)

// static schedules only ordinary commits and delivery of the two real intents.
func static(ctx context.Context, path string) (map[string]any, error) {
	p, err := reconciliation.StaticHandoff()
	if err != nil {
		return nil, err
	}
	s, err := runtime.Create(ctx, path, p, runtime.DefaultLimits())
	if err != nil {
		return nil, err
	}
	defer s.Close()
	src := runtime.RunID{Subject: "source", Generation: 1}
	parent := runtime.RunID{Subject: "parent", Generation: 1}
	step := func(id runtime.RunID, o string) (runtime.RunState, error) {
		r, e := s.Snapshot(ctx, id)
		if e != nil {
			return r, e
		}
		return s.Commit(ctx, r.Authority(), o)
	}
	original, err := step(src, "")
	if err != nil {
		return nil, err
	}
	send, err := step(src, "")
	if err != nil {
		return nil, err
	}
	deliver := func(r runtime.RunState) (runtime.AdmissionResult, error) {
		raw, _ := json.Marshal([]any{r.ID.Subject, r.ID.Generation, r.Sequence, 0})
		return s.DeliverMessage(ctx, string(raw), "public-handoff:"+string(raw))
	}
	msg, err := deliver(send)
	if err != nil || !msg.Accepted {
		return nil, fmt.Errorf("handoff %+v: %w", msg, err)
	}
	rec, err := step(parent, msg.ObservationID)
	if err != nil {
		return nil, err
	}
	ack, err := step(parent, "")
	if err != nil {
		return nil, err
	}
	echo, err := deliver(ack)
	if err != nil || !echo.Accepted {
		return nil, fmt.Errorf("ACK %+v: %w", echo, err)
	}
	consume, err := step(src, echo.ObservationID)
	if err != nil {
		return nil, err
	}
	end, err := step(src, "")
	if err != nil {
		return nil, err
	}
	id := original.Projection.Data["original"].Text
	if id == "" || rec.Projection.Data["received"].Text != id || consume.Projection.Data["echo"].Text != id || end.Projection.Status != "succeeded" || ack.Projection.Status != "open" {
		return nil, errors.New("handoff mismatch")
	}
	if err = s.AwaitDispatchQuiescence(ctx, runtime.DispatchAuthority{Namespace: s.Namespace(), Run: src}); err != nil {
		return nil, err
	}
	replay, err := runtime.Replay(ctx, path, runtime.DefaultLimits())
	if err != nil {
		return nil, err
	}
	_, closed := reconciliation.ClosedTemplate()
	if closed == nil || closed.Error() != "extra caller boundary" {
		return nil, errors.New("legacy constructor accepted an extra caller boundary")
	}
	return map[string]any{"original": original, "sent": send, "message": msg, "received": rec, "ack": ack, "delivery": echo, "consumed": consume, "end": end, "replay": replay, "closed_template_error": closed.Error()}, nil
}
func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: consumer NEW_DATABASE")
		os.Exit(2)
	}
	e, err := static(context.Background(), os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err = json.NewEncoder(os.Stdout).Encode(e); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
