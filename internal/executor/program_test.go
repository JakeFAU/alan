package executor

import (
	"encoding/json"
	"errors"
	recordschema "github.com/JakeFAU/alan/schema"
	"strings"
	"testing"
)

func TestReducerFiniteGenericWork(t *testing.T) {
	p := Program{Semantics: SemanticVersion, Code: []Instruction{
		{Op: "set", Dst: "budget", A: Lit(Int(1000))},
		{Op: "emit", Dst: "pending", Effect: &Effect{Kind: "opaque-operation", Target: Lit(Text("worker")), Generation: Lit(Int(99)), Obligation: Lit(Text("ob-1")), Payload: map[string]Operand{"amount": Ref("budget")}}, Updates: map[string]Operand{"budget": Lit(Int(999))}},
		{Op: "jump", Target: 2},
	}}
	initial := Projection{Data: map[string]Value{}, Status: "open"}
	r, err := Reduce(p, initial, nil, RunID{"supervisor", 123}, 1, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	if r.Projection.Data["budget"] != Int(1000) || initial.PC != 0 || len(initial.Data) != 0 {
		t.Fatalf("reduction mutated input or bounded ordinary integer: %+v", r)
	}
	r, err = Reduce(p, r.Projection, nil, RunID{"supervisor", 123}, 2, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Intents) != 1 || r.Intents[0].Payload["amount"] != Int(1000) || r.Projection.Data["budget"] != Int(999) || r.Projection.Data["pending"] != Text(r.Intents[0].ID) {
		t.Fatalf("intent and ordinary reservation: %+v", r)
	}
	r, err = Reduce(p, r.Projection, nil, RunID{"supervisor", 123}, 3, DefaultLimits())
	if err != nil || r.Projection.PC != 2 {
		t.Fatalf("loop must yield after finite step: %+v %v", r, err)
	}
	limits := DefaultLimits()
	limits.MaxDataBytes = 1
	if _, err = Reduce(p, initial, nil, RunID{"supervisor", 123}, 1, limits); !errors.Is(err, ErrLimit) {
		t.Fatalf("want explicit limit, got %v", err)
	}
}

func TestReceiveExactCorrelationAndTerminal(t *testing.T) {
	p := Program{Semantics: SemanticVersion, Code: []Instruction{{Op: "receive", Match: &Match{Kind: "result", Provenance: "authority", Source: Lit(Text("worker")), Generation: Lit(Int(90)), Obligation: Lit(Text("ob")), Operation: Lit(Text("op"))}, Bind: map[string]string{"answer": "result"}}, {Op: "end", Outcome: "succeeded"}}}
	state := Projection{Data: map[string]Value{}, Status: "open"}
	obs := Observation{ID: "o1", Run: RunID{"inner", 50}, Kind: "result", Provenance: "authority", Source: "worker", SourceGeneration: 90, Obligation: "ob", Operation: "wrong", Payload: map[string]Value{"answer": Int(1000)}}
	if _, err := Reduce(p, state, &obs, obs.Run, 1, DefaultLimits()); !errors.Is(err, ErrIneligible) {
		t.Fatalf("wrong operation: %v", err)
	}
	obs.Operation = "op"
	for _, test := range []struct {
		name   string
		change func(*Observation)
	}{
		{"kind", func(o *Observation) { o.Kind = "other" }},
		{"provenance", func(o *Observation) { o.Provenance = "other" }},
		{"source", func(o *Observation) { o.Source = "other" }},
		{"source generation", func(o *Observation) { o.SourceGeneration++ }},
		{"obligation", func(o *Observation) { o.Obligation = "other" }},
		{"receiving subject", func(o *Observation) { o.Run.Subject = "other" }},
		{"receiving generation", func(o *Observation) { o.Run.Generation++ }},
	} {
		t.Run(test.name, func(t *testing.T) {
			wrong := obs
			test.change(&wrong)
			if _, err := Reduce(p, state, &wrong, obs.Run, 1, DefaultLimits()); !errors.Is(err, ErrIneligible) {
				t.Fatalf("mismatched %s accepted: %v", test.name, err)
			}
		})
	}
	r, err := Reduce(p, state, &obs, obs.Run, 1, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	if r.Projection.Data["result"] != Int(1000) {
		t.Fatal(r)
	}
	r, err = Reduce(p, r.Projection, nil, obs.Run, 2, DefaultLimits())
	if err != nil || r.Projection.Status != "succeeded" {
		t.Fatalf("end: %+v %v", r, err)
	}
	if _, err = Reduce(p, r.Projection, nil, obs.Run, 3, DefaultLimits()); !errors.Is(err, ErrClosed) {
		t.Fatalf("closed run: %v", err)
	}
}

func TestEmitReservesArithmeticUpdateAtomically(t *testing.T) {
	delta := int64(1)
	p := Program{Semantics: SemanticVersion, Code: []Instruction{{Op: "emit", Dst: "pending", Effect: &Effect{Kind: "operation", Target: Lit(Text("target")), Generation: Lit(Int(999)), Obligation: Lit(Text("ob"))}, Updates: map[string]Operand{"spent": {Ref: "spent", Offset: &delta}}}, {Op: "end", Outcome: "failed"}}}
	initial := Projection{Status: "open", Data: map[string]Value{"spent": Int(900)}}
	r, err := Reduce(p, initial, nil, RunID{"arbitrary", 999}, 1, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	if r.Projection.Data["spent"] != Int(901) || len(r.Intents) != 1 || initial.Data["spent"] != Int(900) {
		t.Fatalf("reservation %+v", r)
	}
	initial.Data["spent"] = Int(9223372036854775807)
	if _, err := Reduce(p, initial, nil, RunID{"arbitrary", 999}, 1, DefaultLimits()); !errors.Is(err, ErrLimit) {
		t.Fatalf("overflow must not commit: %v", err)
	}
}

func TestOrdinaryEndCannotClaimAuthorityOutcome(t *testing.T) {
	for _, outcome := range []string{"cancelled", "abandoned"} {
		t.Run(outcome, func(t *testing.T) {
			p := Program{Semantics: SemanticVersion, Code: []Instruction{{Op: "end", Outcome: outcome}}}
			if _, _, err := EncodeProgram(p); err == nil {
				t.Fatalf("ordinary program manufactured %s authority outcome", outcome)
			}
			if _, err := Reduce(p, Projection{Status: "open", Data: map[string]Value{}}, nil, RunID{"worker", 1}, 1, DefaultLimits()); err == nil {
				t.Fatal("reducer accepted authority outcome")
			}
		})
	}
}

func TestReceiveRejectsAmbiguousBinding(t *testing.T) {
	p := Program{Semantics: SemanticVersion, Code: []Instruction{{Op: "receive", Match: &Match{Kind: "input", Provenance: "input", Source: Lit(Text("sender")), Generation: Lit(Int(1)), Obligation: Lit(Text("")), Operation: Lit(Text(""))}, Bind: map[string]string{"left": "same", "right": "same"}}, {Op: "end", Outcome: "succeeded"}}}
	if _, _, err := EncodeProgram(p); err == nil {
		t.Fatal("multiple payload fields bind one destination with nondeterministic map order")
	}
}

// This is the exact ordinary program/data shape from independent review R1.
func amplifiedOutputFixture(chars int) (Program, Projection) {
	payload := map[string]Operand{}
	for _, key := range []string{"a", "b", "c", "d", "e", "f", "g", "h"} {
		payload[key] = Ref("text")
	}
	program := Program{Semantics: SemanticVersion, Code: []Instruction{{Op: "emit", Effect: &Effect{Kind: "mutation", Target: Lit(Text("endpoint")), Generation: Lit(Int(1)), Obligation: Lit(Text("ob")), Payload: payload}}, {Op: "end", Outcome: "succeeded"}}}
	initial := Projection{Status: "open", Data: map[string]Value{"text": Text(strings.Repeat("x", chars))}}
	return program, initial
}

func TestGeneratedOutputRejectsRepeatedReferenceAmplification(t *testing.T) {
	program, initial := amplifiedOutputFixture(500000)
	encoded, _, err := EncodeProgram(program)
	if err != nil {
		t.Fatal(err)
	}
	projection, err := json.Marshal(initial)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("review R1 original input: program=%d projection=%d, eight references to 500000-character text", len(encoded), len(projection))
	result, err := Reduce(program, initial, nil, RunID{"worker", 1}, 1, DefaultLimits())
	if !errors.Is(err, ErrLimit) {
		output, encodeErr := json.Marshal(result.Intents)
		if encodeErr != nil {
			t.Fatal(encodeErr)
		}
		t.Fatalf("generated output limit not enforced: bytes=%d err=%v", len(output), err)
	}
	if initial.PC != 0 || initial.Data["text"].Text != strings.Repeat("x", 500000) {
		t.Fatal("rejected reduction mutated initial state")
	}
}

func TestGeneratedOutputHonorsExactConfiguredJSONBoundary(t *testing.T) {
	program, initial := amplifiedOutputFixture(100000)
	// Escaping in both metadata and payload must be included in the byte accounting.
	program.Code[0].Effect.Kind = "<mutation>\u2028\""
	initial.Data["text"] = Text("<>&\"\\\x00\r\n\u2028\u2029" + initial.Data["text"].Text + string([]byte{0xff}))
	id := RunID{"worker", 1}
	result, err := Reduce(program, initial, nil, id, 1, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	output, err := json.Marshal(result.Intents)
	if err != nil {
		t.Fatal(err)
	}
	limits := DefaultLimits()
	limits.MaxOutputBytes = len(output)
	if _, err = Reduce(program, initial, nil, id, 1, limits); err != nil {
		t.Fatalf("exact byte limit rejected: %v", err)
	}
	limits.MaxOutputBytes--
	if _, err = Reduce(program, initial, nil, id, 1, limits); !errors.Is(err, ErrLimit) {
		t.Fatalf("one byte over accepted: %v", err)
	}
	limits.MaxOutputBytes = 0
	if _, err = Reduce(program, initial, nil, id, 1, limits); !errors.Is(err, ErrLimit) {
		t.Fatalf("nonpositive output limit accepted: %v", err)
	}
}

func TestLegacySemanticsRejectCollections(t *testing.T) {
	for _, sem := range []string{SemanticVersion, AuthorSemanticVersion} {
		p := Program{Semantics: sem, Code: []Instruction{{Op: "set", Dst: "items", A: Lit(recordschema.List())}, {Op: "end", Outcome: "succeeded"}}}
		if sem == AuthorSemanticVersion {
			empty, _ := (recordschema.Record{}).Retain()
			p.Author = &ProgramMetadata{Compiler: AuthorCompilerVersion, Inputs: empty, Outputs: empty, Positions: []string{"set", "end"}}
		}
		if _, _, err := EncodeProgram(p); err == nil {
			t.Fatalf("legacy %s accepted list literal", sem)
		}
	}
	p := Program{Semantics: SemanticVersion, Code: []Instruction{{Op: "end", Outcome: "succeeded"}}}
	if _, err := Reduce(p, Projection{Status: "open", Data: map[string]Value{"items": recordschema.List()}}, nil, RunID{"old", 1}, 1, DefaultLimits()); err == nil {
		t.Fatal("legacy projection accepted list")
	}
	if authorAdmissionValid(p, AdmissionDescriptor{Kind: "input", Payload: map[string]Value{"items": recordschema.List()}}) {
		t.Fatal("legacy program admitted collection")
	}
}

func TestUnambiguousSendContractsAndAuthorIntentValidation(t *testing.T) {
	empty, err := (recordschema.Record{}).Retain()
	if err != nil {
		t.Fatal(err)
	}
	strictRec, err := (recordschema.Record{
		"mode": {Kind: "string", MaxBytes: 4, Enum: []string{"fast", "slow"}},
	}).Retain()
	if err != nil {
		t.Fatal(err)
	}
	wideRec, err := (recordschema.Record{
		"mode": {Kind: "string", MaxBytes: 32},
	}).Retain()
	if err != nil {
		t.Fatal(err)
	}

	ambiguousProg := Program{
		Semantics: AuthorSemanticVersion,
		Author: &ProgramMetadata{
			Compiler: AuthorCompilerVersion,
			Inputs:   wideRec,
			Outputs:  empty,
			Ports: map[string]PortDeclaration{
				"strict_out": {Peer: "receiver", Obligation: "work", SendUsed: true, Receive: empty, Send: strictRec},
				"wide_out":   {Peer: "receiver", Obligation: "work", SendUsed: true, Receive: empty, Send: wideRec},
			},
			Positions: []string{"emit", "end"},
		},
		Code: []Instruction{
			{
				Op: "emit",
				Effect: &Effect{
					Kind:       "message",
					Target:     Lit(Text("receiver")),
					Generation: Lit(Int(1)),
					Obligation: Lit(Text("work")),
					Payload:    map[string]Operand{"mode": Ref("mode")},
				},
			},
			{Op: "end", Outcome: "succeeded"},
		},
	}

	if _, _, err := EncodeProgram(ambiguousProg); err == nil || !strings.Contains(err.Error(), "ambiguous send contract") {
		t.Fatalf("expected EncodeProgram to reject ambiguous SendUsed contract, got: %v", err)
	}
	if _, err := Reduce(ambiguousProg, Projection{Status: "open", Data: map[string]Value{"mode": Text("VIOLATES_STRICT_SCHEMA")}}, nil, RunID{"sender", 1}, 1, DefaultLimits()); err == nil || !strings.Contains(err.Error(), "ambiguous send contract") {
		t.Fatalf("expected Reduce to reject ambiguous SendUsed contract, got: %v", err)
	}

	localSendProg := ambiguousProg
	localMeta := *ambiguousProg.Author
	localMeta.Ports = map[string]PortDeclaration{
		"local_out": {Local: true, Peer: "local", Obligation: "work", SendUsed: true, Receive: empty, Send: strictRec},
	}
	localSendProg.Author = &localMeta
	if _, _, err := EncodeProgram(localSendProg); err == nil || !strings.Contains(err.Error(), "local port cannot send") {
		t.Fatalf("expected EncodeProgram to reject local port with SendUsed, got: %v", err)
	}

	aliasProg := ambiguousProg
	aliasMeta := *ambiguousProg.Author
	aliasMeta.Ports = map[string]PortDeclaration{
		"out_a": {Peer: "receiver", Obligation: "work", SendUsed: true, Receive: empty, Send: strictRec},
		"out_b": {Peer: "receiver", Obligation: "work", SendUsed: true, Receive: empty, Send: strictRec},
	}
	aliasProg.Author = &aliasMeta
	if _, _, err := EncodeProgram(aliasProg); err != nil {
		t.Fatalf("expected identical Send.ID alias ports to encode cleanly, got: %v", err)
	}
	if _, err := Reduce(aliasProg, Projection{Status: "open", Data: map[string]Value{"mode": Text("VIOLATES_STRICT_SCHEMA")}}, nil, RunID{"sender", 1}, 1, DefaultLimits()); err == nil {
		t.Fatal("expected Reduce on alias ports to reject payload violating strict schema")
	}
	if _, err := Reduce(aliasProg, Projection{Status: "open", Data: map[string]Value{"mode": Text("fast")}}, nil, RunID{"sender", 1}, 1, DefaultLimits()); err != nil {
		t.Fatalf("expected Reduce on alias ports with valid payload to succeed, got: %v", err)
	}
}
