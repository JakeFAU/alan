package main

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/JakeFAU/alan/author"
	aruntime "github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

// A measurement is usable only if identity/provenance defects fail its proof.
func TestProofRejectsChangedProvenance(t *testing.T) {
	r := report{Bound: 8, Sample: "proof-test"}
	dir := t.TempDir()
	if err := run(context.Background(), &r, dir, 0); err != nil {
		t.Fatal(err)
	}
	var window *stage
	for i := range r.Stages {
		if r.Stages[i].Name == "execute_window" {
			window = &r.Stages[i]
		}
	}
	if window == nil || window.WallNS < r.ExecuteTotals.WallNS || window.TotalAlloc < r.ExecuteTotals.TotalAlloc || window.Mallocs < r.ExecuteTotals.Mallocs {
		t.Fatal("whole execution window is absent or excludes measured execution phases")
	}
	decl, err := author.BoundedSeries(definition(), author.SeriesOptions{ID: "series", Count: 8, Inputs: map[string]schema.Value{"station": schema.Text("north")}, First: author.ApproveAtProvision, ApprovalReasons: []string{"operator"}})
	if err != nil {
		t.Fatal(err)
	}
	cut, err := aruntime.InspectExecution(context.Background(), filepath.Join(dir, "store.sqlite"), aruntime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	view, err := decl.Inspect(cut)
	if err != nil {
		t.Fatal(err)
	}
	m := &meter{contacts: append([]contact(nil), r.Contacts...)}
	if _, err = verify(cut, view, m, 8); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name   string
		change func() func()
	}{
		{"wrong contact key", func() func() {
			old := m.contacts[0].Key
			m.contacts[0].Key = "wrong"
			return func() { m.contacts[0].Key = old }
		}},
		{"wrong contact capability", func() func() {
			old := m.contacts[0].Capability
			m.contacts[0].Capability = "wrong"
			return func() { m.contacts[0].Capability = old }
		}},
		{"wrong contact effect", func() func() {
			old := m.contacts[0].Effect
			m.contacts[0].Effect = "wrong"
			return func() { m.contacts[0].Effect = old }
		}},
		{"wrong contact protocol", func() func() {
			old := m.contacts[0].Protocol
			m.contacts[0].Protocol = "wrong"
			return func() { m.contacts[0].Protocol = old }
		}},
		{"wrong contact endpoint", func() func() {
			old := m.contacts[0].Endpoint
			m.contacts[0].Endpoint = "wrong"
			return func() { m.contacts[0].Endpoint = old }
		}},
		{"duplicate original operation", func() func() {
			p := view.Positions[1].Effect
			old := p.OperationID
			p.OperationID = view.Positions[0].Effect.OperationID
			return func() { p.OperationID = old }
		}},
		{"wrong approval reason", func() func() {
			p := view.Positions[1].Approval
			old := p.Approval.Reason
			p.Approval.Reason = "foreign"
			return func() { p.Approval.Reason = old }
		}},
		{"wrong predecessor identity", func() func() {
			p := view.Positions[1].Approval
			old := p.Approval.PredecessorID
			p.Approval.PredecessorID = "foreign"
			return func() { p.Approval.PredecessorID = old }
		}},
		{"approval not admitted", func() func() {
			p := view.Positions[1].Approval
			old := p.Approval.Admitted
			p.Approval.Admitted = false
			return func() { p.Approval.Admitted = old }
		}},
		{"wrong predecessor evidence", func() func() {
			p := view.Positions[1].Approval
			old := p.Approval.ResultDigest
			p.Approval.ResultDigest = "wrong"
			return func() { p.Approval.ResultDigest = old }
		}},
		{"selection detached from reduction", func() func() {
			p := view.Positions[0].Observation
			old := p.SelectedAt
			p.SelectedAt = &aruntime.ReductionRef{Run: p.SelectedAt.Run, Sequence: 999999}
			return func() { p.SelectedAt = old }
		}},
		{"missing paid charge", func() func() {
			p := view.Positions[0].Root
			old := p.Charges.Effects
			p.Charges.Effects = 0
			return func() { p.Charges.Effects = old }
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			restore := tt.change()
			defer restore()
			if _, err := verify(cut, view, m, 8); err == nil {
				t.Fatal("proof accepted changed provenance")
			}
		})
	}
}
