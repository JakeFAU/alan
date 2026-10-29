// Command consumer probes authentic terminal access through public APIs only.
// The frozen ordinary J2 peer does not compile; this is not a J2 join executable.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

type terminalProbe struct {
	Provision                    runtime.Provision
	PeerProgram, ReceiverProgram runtime.Artifact
	Source, Consumed, Receiver   runtime.RunState
	Admission                    runtime.AdmissionResult
	Snapshots                    map[string]runtime.Inspection
	Replay                       runtime.ReplayReport
}

func runTerminalProbe(ctx context.Context, path string) (terminalProbe, error) {
	result := terminalProbe{Snapshots: map[string]runtime.Inspection{}}
	peer, err := author.Compile(author.Definition{
		Name: "phase1k-terminal-probe-peer", Steps: []author.Step{author.Success()},
	})
	if err != nil {
		return result, fmt.Errorf("peer compile: %w", err)
	}
	literal := func(v schema.Value) map[string]any { return map[string]any{"Literal": v} }
	// Independent original-semantics JSON; no compiler output is rewritten.
	raw, err := json.Marshal(map[string]any{
		"Semantics": "alan/1d/v1",
		"Code": []any{
			map[string]any{
				"Op": "receive",
				"Match": map[string]any{
					"Kind": "terminal", "Provenance": "runtime-terminal",
					"Source": literal(schema.Text("peer")), "Generation": literal(schema.Int(1)),
					"Obligation": literal(schema.Text("done")), "Operation": literal(schema.Text("")),
				},
				"Bind": map[string]string{"outcome": "peer_outcome"},
			},
			map[string]any{"Op": "end", "Outcome": "succeeded"},
		},
	})
	if err != nil {
		return result, fmt.Errorf("raw receiver marshal: %w", err)
	}
	sum := sha256.Sum256(raw)
	var peerHeader struct{ Semantics string }
	if err = json.Unmarshal(peer.Bytes(), &peerHeader); err != nil {
		return result, err
	}
	result.PeerProgram = runtime.Artifact{ID: peer.ID(), Semantics: peerHeader.Semantics, Bytes: peer.Bytes()}
	result.ReceiverProgram = runtime.Artifact{ID: hex.EncodeToString(sum[:]), Semantics: "alan/1d/v1", Bytes: raw}
	result.Provision = runtime.Provision{
		Programs: []runtime.Artifact{result.PeerProgram, result.ReceiverProgram},
		Subjects: []runtime.Subject{
			{ID: "peer", Generation: 1, Owner: "owner:peer", Revision: 1},
			{ID: "receiver", Generation: 1, Owner: "owner:receiver", Revision: 1},
		},
		Runs: []runtime.InitialRun{
			{ID: runtime.RunID{Subject: "peer", Generation: 1}, ProgramID: result.PeerProgram.ID,
				Projection: runtime.Projection{Status: "open", Data: map[string]schema.Value{}}},
			{ID: runtime.RunID{Subject: "receiver", Generation: 1}, ProgramID: result.ReceiverProgram.ID,
				Projection: runtime.Projection{Status: "open", Data: map[string]schema.Value{}}},
		},
		Permissions: []runtime.Permission{{
			Kind: "observation", Source: "peer", Target: "receiver",
			Operation: "terminal", Revision: 1, Obligation: "done",
		}},
	}
	store, err := runtime.Create(ctx, path, result.Provision, runtime.DefaultLimits())
	if err != nil {
		return result, fmt.Errorf("public provision: %w", err)
	}
	defer store.Close()
	inspect := func(name string) error {
		snapshot, err := runtime.Inspect(ctx, path, runtime.DefaultLimits())
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		result.Snapshots[name] = snapshot
		return nil
	}
	peerState, err := store.Snapshot(ctx, runtime.RunID{Subject: "peer", Generation: 1})
	if err != nil {
		return result, err
	}
	result.Source, err = store.Commit(ctx, peerState.Authority(), "")
	if err != nil {
		return result, fmt.Errorf("peer terminal commit: %w", err)
	}
	if err = inspect("before-delivery"); err != nil {
		return result, err
	}
	// The facade does not alias TerminalDestination. This unnamed argument is
	// assignable to the method's exported, otherwise unnameable parameter type.
	destination := struct{ Target, Obligation string }{Target: "receiver", Obligation: "done"}
	result.Admission, err = store.DeliverTerminal(ctx, result.Source.ID, destination, "phase1k-terminal-probe")
	if err != nil {
		return result, fmt.Errorf("public terminal delivery: %w", err)
	}
	if !result.Admission.Accepted || result.Admission.ObservationID == "" {
		return result, fmt.Errorf("terminal admission refused: %+v", result.Admission)
	}
	if err = inspect("after-delivery"); err != nil {
		return result, err
	}
	receiver, err := store.Snapshot(ctx, runtime.RunID{Subject: "receiver", Generation: 1})
	if err != nil {
		return result, err
	}
	result.Consumed, err = store.Commit(ctx, receiver.Authority(), result.Admission.ObservationID)
	if err != nil {
		return result, fmt.Errorf("public selected terminal consumption: %w", err)
	}
	if err = inspect("after-consumption"); err != nil {
		return result, err
	}
	result.Receiver, err = store.Commit(ctx, result.Consumed.Authority(), "")
	if err != nil {
		return result, fmt.Errorf("receiver terminal commit: %w", err)
	}
	if err = inspect("final"); err != nil {
		return result, err
	}
	result.Replay, err = runtime.Replay(ctx, path, runtime.DefaultLimits())
	if err != nil {
		return result, fmt.Errorf("historical replay: %w", err)
	}
	if err = store.Close(); err != nil {
		return result, err
	}
	return result, nil
}

func exportProbe(dir string, result terminalProbe) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	hashes := map[string]string{}
	write := func(name string, raw []byte) error {
		if err := os.WriteFile(filepath.Join(dir, name), raw, 0644); err != nil {
			return err
		}
		sum := sha256.Sum256(raw)
		hashes[name] = hex.EncodeToString(sum[:])
		return nil
	}
	writeJSON := func(name string, v any) error {
		raw, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			return err
		}
		return write(name, append(raw, '\n'))
	}
	if err := write("peer-program.json", result.PeerProgram.Bytes); err != nil {
		return err
	}
	if err := write("receiver-program.json", result.ReceiverProgram.Bytes); err != nil {
		return err
	}
	if err := writeJSON("provision.json", result.Provision); err != nil {
		return err
	}
	if err := writeJSON("admission.json", result.Admission); err != nil {
		return err
	}
	if err := writeJSON("replay.json", result.Replay); err != nil {
		return err
	}
	for name, snapshot := range result.Snapshots {
		if err := writeJSON(name+".json", snapshot); err != nil {
			return err
		}
	}
	raw, err := json.MarshalIndent(hashes, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "sha256.json"), append(raw, '\n'), 0644)
}

func execute(ctx context.Context, args []string, out io.Writer) error {
	flags := flag.NewFlagSet("terminal-probe", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	path := flags.String("db", "", "fresh retained probe database")
	evidence := flags.String("evidence", "", "optional exact artifact and inspection directory")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *path == "" || flags.NArg() != 0 {
		return fmt.Errorf("-db is required; positional arguments are not accepted")
	}
	result, err := runTerminalProbe(ctx, *path)
	if err != nil {
		return err
	}
	if *evidence != "" {
		if err = exportProbe(*evidence, result); err != nil {
			return err
		}
	}
	return json.NewEncoder(out).Encode(struct{ SourceStatus, ReceiverStatus, ObservationID string }{
		SourceStatus: result.Source.Projection.Status, ReceiverStatus: result.Receiver.Projection.Status,
		ObservationID: result.Admission.ObservationID,
	})
}
func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	if err := execute(ctx, os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
