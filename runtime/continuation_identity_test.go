//go:build integration

package runtime_test

import (
	"database/sql"
	"encoding/json"
	"testing"

	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

func TestContinuationRejectsMaskedFormatFour(t *testing.T) {
	f := newMeterFixture(t, "north")
	if err := f.owner.Close(); err != nil {
		t.Fatal(err)
	}
	path := copyMeterLedger(t, f.path)
	changeMeterLedger(t, path, func(tx *sql.Tx) {
		meterExec(t, tx, "UPDATE metadata SET schema_version=4")
		meterExec(t, tx, "PRAGMA user_version=4")
		meterExec(t, tx, "UPDATE composition_profile SET descriptor=CAST(json_remove(descriptor,'$.Continuations','$.RecordDispatchAdmissions') AS BLOB)")
		meterExec(t, tx, "DROP TABLE adapter_dispatch_admissions")
	})
	rejectMeterLedger(t, path)
}

func TestContinuationRejectsAliasedIngress(t *testing.T) {
	f := newMeterFixture(t, "north", func(p *runtime.Provision) { p.Composition.Continuations[0].ApprovalID = "\uFFFD" })
	if err := f.run(t.Context()); err != nil {
		t.Fatal(err)
	}
	basis := meterBasis(t, f)
	raw := runtime.Input{TransportID: "\xff", LogicalID: "ordinary", Target: "original", Generation: 1, Obligation: "unused", Payload: map[string]schema.Value{}}
	if d, err := f.owner.AdmitInput(t.Context(), raw); err == nil || d.Accepted {
		t.Fatalf("invalid UTF-8 occupied a canonical transport identity: %+v %v", d, err)
	}
	if d, err := f.owner.ApproveContinuation(t.Context(), basis, "sample"); err != nil || !d.Accepted {
		t.Fatalf("valid approval after rejected input: %+v %v", d, err)
	}
	if err := f.run(t.Context()); err != nil {
		t.Fatal(err)
	}
	s := meterCut(t, f.path)
	if meterSpent(s) != 2 || f.meter.calls.Load() != 2 {
		t.Fatal("invalid input displaced or amplified the original opportunity")
	}
}

func TestContinuationRollsBackAlteredApproval(t *testing.T) {
	cases := map[string]string{
		"fact_digest":         "CREATE TRIGGER damage AFTER INSERT ON composition_facts WHEN NEW.kind='continuation' BEGIN UPDATE composition_facts SET descriptor=CAST(json_set(NEW.descriptor,'$.RecordDigest',printf('%064d',0)) AS BLOB) WHERE id=NEW.id; END",
		"observation_payload": "CREATE TRIGGER damage AFTER INSERT ON observations WHEN json_extract(NEW.descriptor,'$.Kind')='input' BEGIN UPDATE observations SET descriptor=CAST(json_set(NEW.descriptor,'$.Payload.purpose.Text','changed') AS BLOB) WHERE id=NEW.id; END",
		"admission_link":      "CREATE TRIGGER damage AFTER INSERT ON composition_facts WHEN NEW.kind='continuation' BEGIN UPDATE admissions SET observation_id=NULL WHERE domain='input'; END",
	}
	for name, ddl := range cases {
		t.Run(name, func(t *testing.T) {
			f := newMeterFixture(t, "north")
			if err := f.run(t.Context()); err != nil {
				t.Fatal(err)
			}
			basis := meterBasis(t, f)
			if err := f.owner.Close(); err != nil {
				t.Fatal(err)
			}
			changeMeterLedger(t, f.path, func(tx *sql.Tx) { meterExec(t, tx, ddl) })
			owner, err := runtime.Open(t.Context(), f.path, runtime.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			f.owner = owner
			if d, err := f.owner.ApproveContinuation(t.Context(), basis, "sample"); err == nil || d.Accepted {
				t.Fatalf("approval survived partial write: %+v %v", d, err)
			}
			if err = f.run(t.Context()); err != nil {
				t.Fatal(err)
			}
			s := meterCut(t, f.path)
			if _, ok := s.Continuation(meterNext); ok || meterSpent(s) != 1 || f.meter.calls.Load() != 1 {
				t.Fatal("bad write retained an approval or allowed a second dispatch")
			}
		})
	}
}

func TestContinuationMarkerInOrdinaryFormat(t *testing.T) {
	f := newMeterFixture(t, "north", func(p *runtime.Provision) { p.Composition = nil })
	input := runtime.Input{TransportID: meterApproval, LogicalID: meterApproval, Target: meterNext.Subject, Generation: 1, Obligation: "approve", Payload: map[string]schema.Value{"previous_operation_id": schema.Text("n"), "purpose": schema.Text("sample"), "prepared_digest": schema.Text("p"), "result_digest": schema.Text("")}}
	if r, err := f.owner.AdmitInput(t.Context(), input); err != nil || !r.Accepted {
		t.Fatalf("setup: %+v %v", r, err)
	}
	if err := f.owner.Close(); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"Continuation", "continuation", `\u0043ontinuation`} {
		t.Run(field, func(t *testing.T) {
			path := copyMeterLedger(t, f.path)
			changeMeterLedger(t, path, func(tx *sql.Tx) {
				claim, err := json.Marshal(runtime.ContinuationApproval{ContinuationBasis: runtime.ContinuationBasis{Namespace: f.owner.Namespace(), Target: meterNext, PredecessorID: "n", PreparedDigest: "p"}, Reason: "sample", TargetRevision: 1})
				if err != nil {
					t.Fatal(err)
				}
				meterExec(t, tx, `UPDATE admissions SET descriptor=CAST(replace(json_set(descriptor,'$.Continuation',json(?)),'"Continuation":',?) AS BLOB)`, string(claim), `"`+field+`":`)
			})
			rejectMeterLedger(t, path)
		})
	}
}
