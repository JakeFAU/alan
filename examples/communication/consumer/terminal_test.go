//go:build integration

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

// Real terminal delivery and selected consumption through a separate module.
// The fixture contains no ordinary report/ACK join or private runtime imports.
func TestTerminalObservationPublicConsumer(t *testing.T) {
	result, err := runTerminalProbe(t.Context(), filepath.Join(t.TempDir(), "terminal.db"))
	if err != nil {
		t.Fatal(err)
	}
	if result.Source.Projection.Status != "succeeded" || result.Source.Sequence != 1 {
		t.Fatal("peer did not genuinely terminate", result.Source)
	}
	before := result.Snapshots["before-delivery"]
	if len(before.Tables["terminal_obligations"]) != 1 {
		t.Fatal("terminal route not retained")
	}
	terminal := before.Tables["terminal_obligations"][0]
	var destinations []struct{ Target, Obligation string }
	if err = json.Unmarshal([]byte(fmt.Sprint(terminal["destinations"])), &destinations); err != nil {
		t.Fatal(err)
	}
	sourceTerminal := terminal["subject"] == "peer" && fmt.Sprint(terminal["sequence"]) == "1"
	routeMatches := len(destinations) == 1 && destinations[0].Target == "receiver" && destinations[0].Obligation == "done"
	if !sourceTerminal || !routeMatches {
		t.Fatal("wrong frozen route", terminal)
	}
	admitted := result.Admission
	receiverBound := admitted.Bound != nil && *admitted.Bound == (runtime.RunID{Subject: "receiver", Generation: 1})
	if !admitted.Accepted || admitted.ObservationID == "" || !receiverBound {
		t.Fatal("terminal admission refused or unbound", admitted)
	}
	delivered := result.Snapshots["after-delivery"]
	if len(delivered.Tables["observations"]) != 1 {
		t.Fatal("missing authentic observation")
	}
	row := delivered.Tables["observations"][0]
	var o struct {
		ID                       string
		Run                      runtime.RunID
		Kind, Provenance, Source string
		SourceGeneration         int64
		Obligation, Operation    string
		Payload                  map[string]schema.Value
	}
	if err = json.Unmarshal([]byte(fmt.Sprint(row["descriptor"])), &o); err != nil {
		t.Fatal(err)
	}
	sameObservation := row["id"] == admitted.ObservationID && o.ID == admitted.ObservationID
	authenticSource := o.Kind == "terminal" && o.Provenance == "runtime-terminal" &&
		o.Source == "peer" && o.SourceGeneration == 1
	exactRoute := o.Run == (runtime.RunID{Subject: "receiver", Generation: 1}) &&
		o.Obligation == "done" && o.Operation == ""
	exactOutcome := o.Payload["outcome"].Equal(schema.Text(result.Source.Projection.Status))
	if !sameObservation || row["consumed_sequence"] != nil || !authenticSource || !exactRoute || !exactOutcome {
		t.Fatal("terminal metadata not authentic/unconsumed", row)
	}
	consumed := result.Consumed
	boundOutcome := consumed.Projection.Data["peer_outcome"].Equal(schema.Text("succeeded"))
	if consumed.Sequence != 1 || consumed.Projection.PC != 1 || !boundOutcome {
		t.Fatal("receiver did not bind terminal outcome", consumed)
	}
	after := result.Snapshots["after-consumption"]
	if fmt.Sprint(after.Tables["observations"][0]["consumed_sequence"]) != "1" {
		t.Fatal("consumption not retained")
	}
	selected := false
	for _, h := range after.Tables["history"] {
		receiverStep := h["subject"] == "receiver" && fmt.Sprint(h["sequence"]) == "1"
		if receiverStep && h["selected_observation"] == admitted.ObservationID {
			selected = true
		}
	}
	if !selected {
		t.Fatal("consumption lacks genuine selected history")
	}
	if result.Receiver.Projection.Status != "succeeded" || result.Receiver.Sequence != 2 {
		t.Fatal("receiver End did not execute", result.Receiver)
	}
	final := result.Snapshots["final"]
	if len(final.Tables["intents"]) != 0 || len(final.Roots) != 0 || len(final.Tables["runs"]) != 2 {
		t.Fatal("probe added protocol work")
	}
	if dir := os.Getenv("ALAN_TERMINAL_EXPORT"); dir != "" {
		if err = exportProbe(dir, result); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("external terminal path succeeded: peer seq=%d receiver selected=%s seq=%d",
		result.Source.Sequence, admitted.ObservationID, consumed.Sequence)
}
