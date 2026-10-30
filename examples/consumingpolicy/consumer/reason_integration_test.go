//go:build integration

package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"sort"
	"testing"

	"github.com/JakeFAU/alan/runtime"
)

// These literals derive from the behavioral contract. A mode-based decision,
// missing reason guard, early Failure or uncommitted host tag must fail here.
func TestConsumerReasonClassifiesConsumedOutcomes(t *testing.T) {
	for _, tc := range []struct {
		name, candidate, mode, sMode, status, reason, classification string
	}{
		{"a answer", "a", "answer", "answer", "succeeded", "member_succeeded", "answer"},
		{"b answer", "b", "answer", "answer", "succeeded", "member_succeeded", "answer"},
		{"a fail", "a", "fail", "answer", "failed", "member_failed", "candidate_failed"},
		{"b fail", "b", "fail", "answer", "failed", "member_failed", "candidate_failed"},
		{"a absent", "a", "absent", "answer", "failed", "missing_reply", "candidate_missing_reply"},
		{"b absent", "b", "absent", "answer", "failed", "missing_reply", "candidate_missing_reply"},
		{"absent deliberately fails", "deliberate-failure", "absent", "answer", "failed", "member_failed", "candidate_failed"},
		{"wrong id", "wrong-id", "answer", "answer", "succeeded", "member_succeeded", "invalid_outcome"},
		{"wrong complement", "weak-a", "answer", "answer", "succeeded", "member_succeeded", "invalid_outcome"},
		{"s fail", "a", "answer", "fail", "succeeded", "member_succeeded", "invalid_outcome"},
		{"s absent", "a", "answer", "absent", "succeeded", "member_succeeded", "invalid_outcome"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "reason.db")
			args := []string{"-db", path, "-policy", "reason", "-candidate", tc.candidate, "-mode", tc.mode, "-s-mode", tc.sMode}
			for _, command := range []string{"init", "run", "replay"} {
				var out bytes.Buffer
				if err := execute(t.Context(), append([]string{command}, args...), &out); err != nil {
					t.Fatal(command, err)
				}
			}
			s, err := runtime.Inspect(t.Context(), path, runtime.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			p := parentProjection(t, s)
			if p.Data["candidate_status"].Text != tc.status || p.Data["candidate_reason"].Text != tc.reason || p.Data["candidate_present"].Flag != (tc.status == "succeeded") {
				t.Fatal("original candidate envelope not preserved", p.Data)
			}
			for _, side := range []string{"candidate", "s"} {
				if !p.Data[side+"_present"].Flag {
					for _, field := range []string{"id", "value"} {
						if _, ok := p.Data[side+"_"+field]; ok {
							t.Fatal("invented absent payload", side, field)
						}
					}
				}
			}
			wantStatus, wantReason := "failed", "member_failed"
			if tc.classification == "answer" {
				wantStatus, wantReason = "succeeded", "member_succeeded"
				if p.Data["candidate_id"].Number != 1 || p.Data["candidate_value"].Number != 1 || p.Data["s_id"].Number != 1 || p.Data["s_value"].Number != 0 {
					t.Fatal("incorrect answer", p.Data)
				}
			}
			if p.Status != wantStatus || p.Data["classification"].Text != tc.classification || len(s.Roots) != 1 || s.Roots[0].Outcome == nil || s.Roots[0].Outcome.Status != wantStatus || s.Roots[0].Outcome.Reason != wantReason {
				t.Fatal("classification or root reason", p, s.Roots)
			}
			checkReasonHistory(t, s, tc.classification, wantStatus)
		})
	}
}

// Early classification or treating a completed sibling as readiness fails these
// checks. No second request is sent to pending components.
func TestConsumerReasonPendingHasNoClassification(t *testing.T) {
	for _, tc := range []struct{ name, candidate, mode, sMode string }{
		{"a candidate", "a", "pending", "answer"},
		{"b candidate", "b", "pending", "answer"},
		{"a s", "a", "answer", "pending"},
		{"b s", "b", "answer", "pending"},
		{"s fail candidate pending", "a", "pending", "fail"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "pending.db")
			args := []string{"-db", path, "-policy", "reason", "-candidate", tc.candidate, "-mode", tc.mode, "-s-mode", tc.sMode}
			for _, command := range []string{"init", "run", "replay"} {
				var out bytes.Buffer
				if err := execute(t.Context(), append([]string{command}, args...), &out); err != nil {
					t.Fatal(err)
				}
			}
			s, err := runtime.Inspect(t.Context(), path, runtime.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			p := parentProjection(t, s)
			if _, ok := p.Data["classification"]; ok {
				t.Fatal("pending has classification", p)
			}
			if p.Status != "open" || len(s.Roots) != 1 || s.Roots[0].Outcome != nil {
				t.Fatal("pending parent closed", p, s.Roots)
			}
			eSeq := consumption(t, s, p.Data["candidate_operation"].Text)
			sSeq := consumption(t, s, p.Data["s_operation"].Text)
			pendingOp := p.Data["candidate_operation"].Text
			if tc.mode == "pending" {
				if p.PC != 3 || sSeq == 0 || eSeq != 0 {
					t.Fatal("original candidate await not open after S", p, eSeq, sSeq)
				}
			} else {
				pendingOp = p.Data["s_operation"].Text
				if p.PC != 2 || eSeq != 0 || sSeq != 0 {
					t.Fatal("pending S bypassed", p, eSeq, sSeq)
				}
			}
			for _, row := range s.Tables["composition_publications"] {
				if row["operation_id"] == pendingOp {
					t.Fatal("invented pending publication", row)
				}
			}
			for _, row := range s.Tables["history"] {
				if row["subject"] == "root" {
					h := decode[runtime.Projection](t, row, "successor")
					if _, ok := h.Data["classification"]; ok || h.Status != "open" {
						t.Fatal("pending history classified or terminated", row)
					}
				}
			}
		})
	}
}

// Bind actual writes/terminal transitions to selected consumption sequences and
// compiled instructions, rather than treating the final tag as branch evidence.
func checkReasonHistory(t *testing.T, s runtime.Inspection, classification, status string) {
	t.Helper()
	p := parentProjection(t, s)
	sSeq := consumption(t, s, p.Data["s_operation"].Text)
	eSeq := consumption(t, s, p.Data["candidate_operation"].Text)
	if sSeq == 0 || eSeq <= sSeq {
		t.Fatal("both original joins required before classification", sSeq, eSeq)
	}
	var code []struct {
		Op, Dst, Outcome string
		A                struct{ Literal *runtime.Value }
	}
	for _, row := range s.Tables["programs"] {
		// The root run's program ID is authoritative; child programs also end.
		for _, run := range s.Tables["runs"] {
			if run["subject"] == "root" && run["program_id"] == row["id"] {
				var program struct{ Code json.RawMessage }
				program = decode[struct{ Code json.RawMessage }](t, row, "bytes")
				if err := json.Unmarshal(program.Code, &code); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
	if len(code) == 0 {
		t.Fatal("root compiled instructions unavailable")
	}
	var history []map[string]any
	for _, row := range s.Tables["history"] {
		if row["subject"] == "root" {
			history = append(history, row)
		}
	}
	sort.Slice(history, func(i, j int) bool { return history[i]["sequence"].(int64) < history[j]["sequence"].(int64) })
	previous := runtime.Projection{PC: 0, Status: "open"}
	var write, terminal int64
	for _, row := range history {
		h := decode[runtime.Projection](t, row, "successor")
		sequence := row["sequence"].(int64)
		in := code[previous.PC]
		if in.Op == "set" && in.Dst == "classification" {
			if write != 0 || sequence <= eSeq || in.A.Literal == nil || in.A.Literal.Text != classification || h.Data["classification"].Text != classification {
				t.Fatal("classification write not after original consumptions", row, in)
			}
			write = sequence
		}
		if in.Op == "end" {
			if terminal != 0 || write == 0 || sequence <= write || in.Outcome != status || h.Status != status {
				t.Fatal("terminal not committed after correct classification", row, in)
			}
			terminal = sequence
		}
		if sequence <= eSeq {
			if _, ok := h.Data["classification"]; ok {
				t.Fatal("early classification", row)
			}
		}
		previous = h
	}
	if write == 0 || terminal == 0 {
		t.Fatal("missing committed classification or terminal instruction", write, terminal)
	}
}
