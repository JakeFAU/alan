// Package substitution is an independent, finite Phase1I specification and oracle.
// It imports no ALAN compiler, reducer, author, runtime, or storage package.
package substitution

import "fmt"

type Request struct {
	ID, Value int
	Mode      string
}

// Observation is a schema-independent retained fact. Operation and Parent must
// use one consistent identity mapping throughout a trace. Debit fields are deltas.
// Consumers must retain absence explicitly; zero values do not imply presence.
type Observation struct {
	Kind, Operation, Parent, Role string
	Request                       Request
	ID, Value                     int
	Status, Reason                string
	Present                       bool
	I, R, E                       int
}
type Violation struct {
	Index        int
	Code, Detail string
}

// CheckTrace checks safety on every prefix. It never infers completion from a
// truncated trace; use CheckContext for a complete context observation.
func CheckTrace(request Request, observations []Observation) []Violation {
	var failures []Violation
	helperCommits := map[string]bool{}
	admitted := map[string]bool{}
	replies := map[string]bool{}
	publications := map[string]Observation{}
	consumed := map[string]bool{}
	fail := func(i int, code, detail string) { failures = append(failures, Violation{i, code, detail}) }
	for i, o := range observations {
		switch o.Kind {
		case "invoke_commit":
			if o.Role == "helper" {
				helperCommits[o.Operation] = true
				if len(helperCommits) > 1 {
					fail(i, "extra_helper", "more than one candidate helper committed")
				}
			}
		case "admit":
			if o.Status == "accepted" {
				if admitted[o.Operation] {
					fail(i, "duplicate_admission", o.Operation)
				}
				admitted[o.Operation] = true
				if o.I != 1 || o.R != 1 || o.E != 0 {
					fail(i, "charge", "singleton debit must be (1,1,0)")
				}
				if o.Role == "helper" {
					helperCommits[o.Operation] = true
					if len(helperCommits) > 1 {
						fail(i, "extra_helper", "more than one candidate helper charged")
					}
				}
			} else if o.I != 0 || o.R != 0 || o.E != 0 {
				fail(i, "refusal_charge", o.Operation)
			}
		case "reply_admit":
			if o.Present || o.Status == "accepted" {
				if replies[o.Operation] {
					fail(i, "duplicate_reply", o.Operation)
				}
				replies[o.Operation] = true
				if o.Role == "candidate" || o.Role == "wrapper" || o.Role == "helper" {
					if o.ID != request.ID {
						fail(i, "application_id", fmt.Sprintf("got %d want %d", o.ID, request.ID))
					}
					if o.Value != 1-request.Value {
						fail(i, "complement", fmt.Sprintf("got %d want %d", o.Value, 1-request.Value))
					}
					if o.Role != "helper" && request.Mode != "answer" {
						fail(i, "mode_reply", request.Mode)
					}
				}
			}
		case "publish":
			if old, ok := publications[o.Operation]; ok && (old.Status != o.Status || old.Reason != o.Reason || old.Present != o.Present || old.ID != o.ID || old.Value != o.Value) {
				fail(i, "mutable_outcome", o.Operation)
			}
			publications[o.Operation] = o
			if o.Status == "succeeded" && !o.Present {
				fail(i, "missing_reply", o.Operation)
			}
		case "outcome_consume":
			if consumed[o.Operation] {
				fail(i, "duplicate_consumption", o.Operation)
			}
			consumed[o.Operation] = true
			if old, ok := publications[o.Operation]; ok && (old.Status != o.Status || old.Reason != o.Reason || old.Present != o.Present || old.ID != o.ID || old.Value != o.Value) {
				fail(i, "outcome_correlation", o.Operation)
			}
		}
	}
	return failures
}
