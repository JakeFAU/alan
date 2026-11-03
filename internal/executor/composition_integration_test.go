package executor

import (
	"context"
	"errors"
	"fmt"
	"github.com/JakeFAU/alan/internal/boundary"
	recordschema "github.com/JakeFAU/alan/schema"
	"reflect"
	"strings"
	"testing"
)

func TestCompositionRunComputesChildren(t *testing.T) {
	s := invocationStore(t, "succeeded", 1, nil)
	report, err := s.Run(t.Context(), RunOptions{MaxSteps: 100, MaxDispatch: 100, MaxInFlight: 1})
	if err != nil {
		t.Fatal(err)
	}
	var admitted, published, computed int
	if err := s.conn.QueryRowContext(t.Context(), "SELECT count(*) FROM composition_invocations").Scan(&admitted); err != nil {
		t.Fatal(err)
	}
	if err := s.conn.QueryRowContext(t.Context(), "SELECT count(*) FROM composition_publications").Scan(&published); err != nil {
		t.Fatal(err)
	}
	if err := s.conn.QueryRowContext(t.Context(), "SELECT count(*) FROM composition_members m JOIN runs r USING(subject,generation) WHERE json_extract(r.projection,'$.Status')='succeeded'").Scan(&computed); err != nil {
		t.Fatal(err)
	}
	if admitted != 2 || published != 2 || computed != 4 || !report.Quiescent {
		t.Fatalf("admitted=%d published=%d computed=%d report=%+v", admitted, published, computed, report)
	}
}

func TestCompositionRootJoinWaitsAndPublishes(t *testing.T) {
	s := invocationStore(t, "succeeded", 1, nil)
	a := invokeOnce(t, s)
	for range 2 {
		if _, err := s.Commit(t.Context(), current(t, s, "root").Authority(), ""); err != nil {
			t.Fatal(err)
		}
	}
	root, err := compositionRootFor(t.Context(), s.conn, RunID{"root", 1})
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := s.PrepareRoot(t.Context(), root.ID)
	if err != nil || prepared.Ready || prepared.Pending != "descendant_outcome" {
		t.Fatal(prepared, err)
	}
	if _, err = s.Run(t.Context(), RunOptions{MaxSteps: 100, MaxDispatch: 100, MaxInFlight: 1}); err != nil {
		t.Fatal(err)
	}
	prepared, err = s.PrepareRoot(t.Context(), root.ID)
	if err != nil || !prepared.Ready || prepared.Outcome.Status != "succeeded" {
		t.Fatal(prepared, err, a)
	}
	var published int
	if err = s.conn.QueryRowContext(t.Context(), "SELECT count(*) FROM composition_root_publications").Scan(&published); err != nil || published != 1 {
		t.Fatal(published, err)
	}
}

func TestCompositionFullReconstruction(t *testing.T) {
	s := invocationStore(t, "succeeded", 1, nil)
	if _, err := s.Run(t.Context(), RunOptions{MaxSteps: 100, MaxDispatch: 100, MaxInFlight: 1}); err != nil {
		t.Fatal(err)
	}
	before, err := inspectTable(t.Context(), s.conn, "composition_facts")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = validateCompositionStore(t.Context(), s.conn, DefaultLimits()); err != nil {
		t.Fatal(err)
	}
	after, err := inspectTable(t.Context(), s.conn, "composition_facts")
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatal("validation mutated facts", err)
	}
}

func TestCompositionReconstructionRejectsMissingAndConflictingRows(t *testing.T) {
	mutations := map[string]string{
		"origin":                  "DELETE FROM composition_members WHERE rowid=(SELECT min(rowid) FROM composition_members)",
		"binding":                 "DELETE FROM composition_bindings WHERE rowid=(SELECT min(rowid) FROM composition_bindings)",
		"request":                 "DELETE FROM composition_requests WHERE rowid=(SELECT min(rowid) FROM composition_requests)",
		"reply admission reverse": "DELETE FROM message_acknowledgments; DELETE FROM admissions WHERE domain='message'",
		"request reverse":         "DELETE FROM composition_requests",
		"charge":                  "DELETE FROM composition_charges WHERE rowid=(SELECT min(rowid) FROM composition_charges)",
		"candidate":               "DELETE FROM composition_replies WHERE rowid=(SELECT min(rowid) FROM composition_replies)",
		"freeze":                  "DELETE FROM composition_freezes WHERE rowid=(SELECT min(rowid) FROM composition_freezes)",
		"publication":             "DELETE FROM composition_publications WHERE rowid=(SELECT min(rowid) FROM composition_publications)",
		"result reverse":          "DELETE FROM composition_results",
		"delivery reverse":        "DELETE FROM composition_outcome_deliveries",
		"root fact":               "DELETE FROM composition_facts WHERE decision_order=(SELECT decision_order FROM composition_root_publications)",
		"authority":               "INSERT INTO permissions VALUES('adapter','root','invented','call',1,'call')",
		"origin descriptor":       "UPDATE composition_members SET descriptor=json_set(descriptor,'$.Owner','forged')",
		"binding descriptor":      "UPDATE composition_bindings SET descriptor=json_set(descriptor,'$.Depth',9)",
		"charge descriptor":       "UPDATE composition_charges SET descriptor=json_set(descriptor,'$.CreatedRuns',0)",
		"root column":             "UPDATE composition_root_publications SET decision_order=1",
		"table":                   "DROP TABLE composition_closures",
	}
	for name, mutation := range mutations {
		t.Run(name, func(t *testing.T) {
			s := invocationStore(t, "succeeded", 1, nil)
			if _, err := s.Run(t.Context(), RunOptions{MaxSteps: 100, MaxDispatch: 100, MaxInFlight: 1}); err != nil {
				t.Fatal(err)
			}
			if _, err := validateCompositionStore(t.Context(), s.conn, DefaultLimits()); err != nil {
				t.Fatal("baseline", err)
			}
			if _, err := s.conn.ExecContext(t.Context(), "PRAGMA foreign_keys=OFF"); err != nil {
				t.Fatal(err)
			}
			if _, err := s.conn.ExecContext(t.Context(), mutation); err != nil {
				t.Fatal("injection", err)
			}
			if _, err := s.conn.ExecContext(t.Context(), "PRAGMA foreign_keys=ON"); err != nil {
				t.Fatal(err)
			}
			if _, err := validateCompositionStore(t.Context(), s.conn, DefaultLimits()); err == nil {
				t.Fatal("accepted corruption", mutation)
			} else {
				t.Log(err)
			}
		})
	}
}
func TestCompositionReconstructionOutcomeVariants(t *testing.T) {
	for _, outcome := range []string{"succeeded", "failed"} {
		for _, replies := range []int{0, 1, 2} {
			t.Run(fmt.Sprintf("%s_%d", outcome, replies), func(t *testing.T) {
				s := invocationStore(t, outcome, replies, nil)
				if _, err := s.Run(t.Context(), RunOptions{MaxSteps: 100, MaxDispatch: 100, MaxInFlight: 1}); err != nil {
					t.Fatal(err)
				}
				if _, err := validateCompositionStore(t.Context(), s.conn, DefaultLimits()); err != nil {
					t.Fatal(err)
				}
			})
		}
	}
}
func TestCompositionReconstructionRejectsProtocolOrderOnly(t *testing.T) {
	for _, kind := range []compositionFactKind{compositionAuthorizationFact, compositionAdmissionFact, compositionReplyFact, compositionFreezeFact, compositionOutcomeFact} {
		t.Run(string(kind), func(t *testing.T) {
			s := invocationStore(t, "succeeded", 1, nil)
			if _, err := s.Run(t.Context(), RunOptions{MaxSteps: 100, MaxDispatch: 100, MaxInFlight: 1}); err != nil {
				t.Fatal(err)
			}
			if _, err := validateCompositionStore(t.Context(), s.conn, DefaultLimits()); err != nil {
				t.Fatal("baseline", err)
			}
			facts, err := readCompositionFacts(t.Context(), s.conn)
			if err != nil {
				t.Fatal(err)
			}
			index := -1
			for i, f := range facts {
				if f.Kind == kind {
					index = i
					break
				}
			}
			if index < 1 {
				t.Fatal("missing fixture fact", kind)
			}
			changed := append([]compositionFact{facts[index]}, append(append([]compositionFact{}, facts[:index]...), facts[index+1:]...)...)
			if _, err = s.conn.ExecContext(t.Context(), "PRAGMA foreign_keys=OFF"); err != nil {
				t.Fatal(err)
			}
			if _, err = s.conn.ExecContext(t.Context(), "DELETE FROM composition_facts"); err != nil {
				t.Fatal(err)
			}
			for i, f := range changed {
				f.Order = int64(i + 1)
				if _, err = s.conn.ExecContext(t.Context(), "INSERT INTO composition_facts VALUES(?,?,?,?)", f.Order, f.ID, f.Kind, compositionJSON(f)); err != nil {
					t.Fatal(err)
				}
			}
			if _, err = s.conn.ExecContext(t.Context(), "PRAGMA foreign_keys=ON"); err != nil {
				t.Fatal(err)
			}
			if _, err = validateCompositionStore(t.Context(), s.conn, DefaultLimits()); err == nil {
				t.Fatal("accepted order-only corruption")
			}
		})
	}
}

func TestCompositionFailedChildCannotHideUnresolvedGrandchild(t *testing.T) {
	p, c := nestedInvocationFixture(t, true)
	s, err := createInvocationStore(t.Context(), t.TempDir()+"/grandchild.sqlite", p, c, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	outer := invokeOnce(t, s)
	entry := compositionMemberID(outer.Invocation.ID, "entry")
	helper := compositionMemberID(outer.Invocation.ID, "helper")
	if _, err = s.Commit(t.Context(), mustSnapshot(t, s, entry).Authority(), compositionIdentity("alan/invocation-request/v1", outer.Invocation.ID)); err != nil {
		t.Fatal(err)
	}
	state, err := s.Commit(t.Context(), mustSnapshot(t, s, entry).Authority(), "")
	if err != nil {
		t.Fatal(err)
	}
	leaf, err := s.AdmitInvocation(t.Context(), state.Projection.Data["pending"].Text)
	if err != nil || !leaf.Accepted {
		t.Fatal(leaf, err)
	}
	if _, err = s.Commit(t.Context(), mustSnapshot(t, s, helper).Authority(), ""); err != nil {
		t.Fatal(err)
	}
	if result, err := s.SettleInvocation(t.Context(), outer.Invocation.OperationID); err != nil || !result.Accepted {
		t.Fatal(result, err)
	}
	if _, err = s.Commit(t.Context(), current(t, s, "root").Authority(), ""); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Commit(t.Context(), current(t, s, "root").Authority(), ""); err != nil {
		t.Fatal(err)
	}
	if _, err = s.AdmitInvocation(t.Context(), identity("root", 1, 2, 0)); err != nil {
		t.Fatal(err)
	}
	root, err := compositionRootFor(t.Context(), s.conn, RunID{"root", 1})
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := s.PrepareRoot(t.Context(), root.ID)
	if err != nil || prepared.Ready || prepared.Pending != "descendant_outcome" {
		t.Fatal("known failure hid unfinished grandchild", prepared, err)
	}
}

func awaitingInvocationRoot(t *testing.T, p *Provision) {
	t.Helper()
	for n, a := range p.Programs {
		if a.ID != p.Runs[0].ProgramID {
			continue
		}
		var root Program
		if err := decode(a.Bytes, &root); err != nil {
			t.Fatal(err)
		}
		d := root.Author.Invocations["child"]
		record, err := InvocationOutcomeRecord(d.Output.Record).Retain()
		if err != nil {
			t.Fatal(err)
		}
		root.Code[1] = Instruction{Op: "receive", Match: &Match{Kind: "invocation", Provenance: "runtime-invocation", Source: Lit(Text(d.TemplateID)), Generation: Lit(Int(1)), Obligation: Lit(Text("child")), Operation: Ref("pending")}, Bind: map[string]string{"status": "status", "reason": "reason", "response_present": "present", "response.result": "result"}}
		root.Author.Receives = map[int]recordschema.Retained{1: record}
		raw, id, err := EncodeProgram(root)
		if err != nil {
			t.Fatal(err)
		}
		p.Programs[n] = Artifact{ID: id, Semantics: root.Semantics, Bytes: raw}
		p.Runs[0].ProgramID = id
		return
	}
	t.Fatal("root artifact missing")
}
func TestCompositionNestedAutomaticRunAndConsumption(t *testing.T) {
	p, c := nestedInvocationFixture(t, false)
	awaitingInvocationRoot(t, &p)
	s, err := createInvocationStore(t.Context(), t.TempDir()+"/nested.sqlite", p, c, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	report, err := s.Run(t.Context(), RunOptions{MaxSteps: 100, MaxDispatch: 100, MaxInFlight: 1})
	if err != nil {
		t.Fatal(err)
	}
	root := current(t, s, "root")
	if root.Projection.Status != "succeeded" || root.Projection.Data["status"].Text != "succeeded" || root.Projection.Data["result"].Text != "ok" || !root.Projection.Data["present"].Flag || !report.Quiescent {
		t.Fatal(root, report)
	}
	if count(t, s, "composition_invocations") != 2 || count(t, s, "composition_publications") != 2 {
		t.Fatal("nested work missing")
	}
	if _, err = validateCompositionStore(t.Context(), s.conn, DefaultLimits()); err != nil {
		t.Fatal(err)
	}
}

func TestCompositionPublishedPreparationIsOriginalOutcome(t *testing.T) {
	s := invocationStore(t, "succeeded", 1, nil)
	if _, err := s.Run(t.Context(), RunOptions{MaxSteps: 100, MaxDispatch: 100, MaxInFlight: 1}); err != nil {
		t.Fatal(err)
	}
	id := identity("root", 1, 1, 0)
	prepared, err := s.PrepareInvocation(t.Context(), id)
	if err != nil || prepared.Outcome.ID == "" || prepared.Outcome.DecisionOrder == 0 {
		t.Fatal("preparation lost original published identity", prepared, err)
	}
}
func TestCompositionRejectsInventedStaticOrigin(t *testing.T) {
	s := invocationStore(t, "succeeded", 1, nil)
	var program string
	var initial []byte
	if err := s.conn.QueryRowContext(t.Context(), "SELECT program_id,initial FROM runs WHERE subject='root'").Scan(&program, &initial); err != nil {
		t.Fatal(err)
	}
	subject := Subject{ID: "invented", Generation: 1, Owner: "owner", Revision: 1}
	var projection Projection
	if err := decode(initial, &projection); err != nil {
		t.Fatal(err)
	}
	if _, err := s.conn.ExecContext(t.Context(), "INSERT INTO subjects VALUES(?,1,'owner',1,0,0)", subject.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.conn.ExecContext(t.Context(), "INSERT INTO runs VALUES(?,1,?,?,1,0,?)", subject.ID, program, initial, initial); err != nil {
		t.Fatal(err)
	}
	if _, err := s.conn.ExecContext(t.Context(), "INSERT INTO run_origins VALUES(?,1,'root',NULL,NULL,?)", subject.ID, compositionJSON(rootOrigin{Subject: subject, ProgramID: program, Initial: projection})); err != nil {
		t.Fatal(err)
	}
	if _, err := validateCompositionStore(t.Context(), s.conn, DefaultLimits()); err == nil {
		t.Fatal("invented static origin escaped immutable deployment")
	}
}

func TestCompositionPublicProfileCreateReplayInspectReopen(t *testing.T) {
	p, c := nestedInvocationFixture(t, false)
	awaitingInvocationRoot(t, &p)
	p.Composition = &c
	path := t.TempDir() + "/public.sqlite"
	s, err := Create(t.Context(), path, p, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	state, err := s.Commit(t.Context(), current(t, s, "root").Authority(), "")
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	replay, err := Replay(t.Context(), path, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	if len(replay.Roots) != 1 || len(replay.Roots[0].PendingInvocations) != 1 || replay.Roots[0].Outcome != nil {
		t.Fatal(replay.Roots)
	}
	s, err = Open(t.Context(), path, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	reopened := current(t, s, "root")
	if reopened.Epoch != state.Epoch+1 || reopened.Sequence != state.Sequence || !reflect.DeepEqual(reopened.Projection, state.Projection) {
		t.Fatal(reopened, state)
	}
	report, err := s.Run(t.Context(), RunOptions{MaxSteps: 100, MaxDispatch: 100, MaxInFlight: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Roots) != 1 || report.Roots[0].Outcome == nil || report.Roots[0].Outcome.Status != "succeeded" {
		t.Fatal(report.Roots)
	}
	before, err := inspectTable(t.Context(), s.conn, "composition_facts")
	if err != nil {
		t.Fatal(err)
	}
	inspection, err := Inspect(t.Context(), path, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	if inspection.Settings.QueryOnly != 1 || len(inspection.Tables["composition_invocations"]) != 2 || len(inspection.Roots) != 1 || inspection.Roots[0].LocalStatus != "succeeded" || inspection.Roots[0].Outcome == nil {
		t.Fatal(inspection.Roots)
	}
	if _, err = Replay(t.Context(), path, DefaultLimits()); err != nil {
		t.Fatal(err)
	}
	after, err := inspectTable(t.Context(), s.conn, "composition_facts")
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatal("read-only inspection/replay wrote facts", err)
	}
}

func TestCompositionPublicRejectsIncompleteWithoutActivation(t *testing.T) {
	for name, mutation := range map[string]string{"missing table": "DROP TABLE composition_reservations", "future profile": "UPDATE composition_profile SET version='alan/composition/future'", "future schema": "UPDATE metadata SET schema_version=99"} {
		t.Run(name, func(t *testing.T) {
			p, c := invocationFixture(t, "succeeded", 1)
			p.Composition = &c
			path := t.TempDir() + "/invalid.sqlite"
			s, err := Create(t.Context(), path, p, DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			before := current(t, s, "root")
			if _, err = s.conn.ExecContext(t.Context(), mutation); err != nil {
				t.Fatal(err)
			}
			if err = s.Close(); err != nil {
				t.Fatal(err)
			}
			if _, err = Replay(t.Context(), path, DefaultLimits()); err == nil {
				t.Fatal("replay accepted incomplete/unknown store")
			}
			if opened, err := Open(t.Context(), path, DefaultLimits()); err == nil {
				opened.Close()
				t.Fatal("open accepted incomplete/unknown store")
			}
			reader, err := connect(t.Context(), path, true)
			if err != nil {
				t.Fatal(err)
			}
			defer reader.Close()
			after, err := loadRun(t.Context(), reader.conn, before.ID)
			if err != nil || after.Epoch != before.Epoch {
				t.Fatal("rejected open advanced authority", after, err)
			}
			if name == "missing table" {
				var n int
				if err = reader.conn.QueryRowContext(t.Context(), "SELECT count(*) FROM sqlite_master WHERE name='composition_reservations'").Scan(&n); err != nil || n != 0 {
					t.Fatal("replay repaired missing table", n, err)
				}
			}
		})
	}
}
func TestCompositionFoundationReservedTableGuardValidFK(t *testing.T) {
	p, c := compositionFixture(t)
	_, artifact := bindingProgram(t, "caller", "reply")
	p.Programs = append(p.Programs, artifact)
	c.Templates = []compositionTemplate{{Members: map[string]compositionTemplateMember{"entry": {ProgramID: artifact.ID, Initial: Projection{Status: "open", Data: map[string]Value{}}}}, EntryMember: "entry", EntryPort: "public", CallerPeer: "caller"}}
	s, err := createCompositionStore(t.Context(), t.TempDir()+"/foundation.sqlite", p, c, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if _, err = s.Commit(t.Context(), current(t, s, "A").Authority(), ""); err != nil {
		t.Fatal(err)
	}
	retained, err := s.composition.descriptor()
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.conn.ExecContext(t.Context(), "INSERT INTO composition_invocations VALUES(?,?,?,?,?,?)", "reserved", identity("A", 1, 1, 0), retained.Templates[0].ID, retained.Roots[0].ID, []byte("{}"), []byte("{}")); err != nil {
		t.Fatal(err)
	}
	fk, err := s.conn.QueryContext(t.Context(), "PRAGMA foreign_key_check")
	if err != nil {
		t.Fatal(err)
	}
	violation := fk.Next()
	fk.Close()
	if violation {
		t.Fatal("fixture has FK violation")
	}
	if _, err = validateCompositionStore(t.Context(), s.conn, DefaultLimits()); !errors.Is(err, ErrVersion) || !strings.Contains(err.Error(), "unfinished composition protocol in composition_invocations") {
		t.Fatal("did not reach reserved-profile guard", err)
	}
}
func TestCompositionInvocationStaticSourceRequiresRoot(t *testing.T) {
	p, c := invocationFixture(t, "succeeded", 1)
	addProgram(t, &p, "actual-root", []Instruction{{Op: "end", Outcome: "succeeded"}}, map[string]Value{})
	c.Roots[0].Run = RunID{"actual-root", 1}
	c.Roots[0].Members = []RunID{{"actual-root", 1}}
	p.Composition = &c
	s, err := Create(t.Context(), t.TempDir()+"/unscoped.sqlite", p, DefaultLimits())
	if s != nil {
		s.Close()
	}
	if err == nil {
		t.Fatal("unscoped static Invoke escaped provisioning")
	}
}

func TestCompositionLegacyReservationsClosureAndLateEvidence(t *testing.T) {
	p := Provision{}
	for _, name := range []string{"root", "peer"} {
		addExternalClient(t, &p, name, "http://127.0.0.1:1", "non-idempotent", 5)
	}
	addInvocationController(t, &p, "CancelRun")
	c := compositionProvision{Version: compositionProtocolVersion, Roots: []compositionRoot{{Run: RunID{"root", 1}, Members: []RunID{{"root", 1}, {"peer", 1}}, Limits: compositionLimits{MaxEffects: 1, MaxInFlight: 1}}}}
	s, err := createInvocationStore(t.Context(), t.TempDir()+"/legacy-reservation.sqlite", p, c, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	for _, name := range []string{"root", "peer"} {
		if _, err = s.Commit(t.Context(), current(t, s, name).Authority(), ""); err != nil {
			t.Fatal(err)
		}
	}
	rootID, peerID := identity("root", 1, 1, 0), identity("peer", 1, 1, 0)
	allowed, fresh, err := s.prepareExternal(t.Context(), rootID)
	if err != nil || !fresh || !allowed.Allowed {
		t.Fatal(allowed, fresh, err)
	}
	if auth, err := s.AuthorizeIntent(t.Context(), peerID); err != nil || !auth.Allowed {
		t.Fatal(auth, err)
	}
	if result := authorizedDecision(t, s, committedRequest(t, s, "controller")); !result.Accepted {
		t.Fatal(result)
	}
	denied, fresh, err := s.prepareExternal(t.Context(), peerID)
	if err != nil || !fresh || denied.Allowed || denied.Reason != "scope_closed" {
		t.Fatal(denied, fresh, err)
	}
	report, err := s.Run(t.Context(), RunOptions{MaxSteps: 100, MaxDispatch: 100, MaxInFlight: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Roots) != 1 || report.Roots[0].Outcome == nil || report.Roots[0].Outcome.Status != "cancelled" || report.Roots[0].LocalStatus != "cancelled" || !reflect.DeepEqual(report.Roots[0].UnknownExternal, []string{rootID}) || report.Roots[0].Charges.Effects != 1 {
		t.Fatal(report.Roots)
	}
	if _, err = validateCompositionStore(t.Context(), s.conn, DefaultLimits()); err != nil {
		t.Fatal(err)
	}
	if err = s.recordExternalReceipt(t.Context(), allowed, receiptFor(allowed.Request)); err != nil {
		t.Fatal(err)
	}
	known, err := s.externalOutcome(t.Context(), rootID)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.settleExternal(t.Context(), known); err != nil {
		t.Fatal(err)
	}
	if _, err = validateCompositionStore(t.Context(), s.conn, DefaultLimits()); err != nil {
		t.Fatal(err)
	}
	roots, err := readRootCompletions(t.Context(), s.conn)
	if err != nil || roots[0].Outcome.Status != "cancelled" || len(roots[0].UnknownExternal) != 0 || roots[0].Charges.Effects != 1 {
		t.Fatal(roots, err)
	}
}
func TestCompositionPostCommitBarriers(t *testing.T) {
	p, c := invocationFixture(t, "succeeded", 1)
	awaitingInvocationRoot(t, &p)
	s, err := createInvocationStore(t.Context(), t.TempDir()+"/barrier.sqlite", p, c, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	admitted, published := 0, 0
	ctx := boundary.WithObserver(t.Context(), func(ctx context.Context, event boundary.Event) error {
		switch event.Boundary {
		case "invocation.admitted":
			admitted++
			var n int
			if err := s.conn.QueryRowContext(ctx, "SELECT count(*) FROM composition_admission_decisions WHERE operation_id=? AND root_id=?", event.Intent, event.Root).Scan(&n); err != nil {
				return err
			}
			if n != 1 || event.Attempt == "" {
				return fmt.Errorf("admission hook before real commit")
			}
		case "invocation.result-recorded":
			published++
			var n int
			if err := s.conn.QueryRowContext(ctx, "SELECT count(*) FROM composition_outcome_deliveries d JOIN observations o ON o.id=d.observation_id WHERE d.operation_id=? AND o.consumed_sequence IS NULL", event.Intent).Scan(&n); err != nil {
				return err
			}
			if n != 1 || event.Root == "" {
				return fmt.Errorf("outcome hook missing unconsumed committed result")
			}
		}
		return nil
	})
	if _, err = s.Run(ctx, RunOptions{MaxSteps: 100, MaxDispatch: 100, MaxInFlight: 1}); err != nil {
		t.Fatal(err)
	}
	if admitted != 1 || published != 1 {
		t.Fatal(admitted, published)
	}
}

func TestCompositionReopenPreservesAdmissionBudget(t *testing.T) {
	p, c := invocationFixture(t, "succeeded", 1)
	c.Roots[0].Limits.MaxInvocations = 1
	p.Composition = &c
	path := t.TempDir() + "/budget.sqlite"
	s, err := Create(t.Context(), path, p, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	first := invokeOnce(t, s)
	if !first.Accepted {
		t.Fatal(first)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = Open(t.Context(), path, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	second := invokeOnce(t, s)
	if second.Accepted || second.Reason != "invocation_limit" {
		t.Fatal("restart reset admitted work allowance", second)
	}
	if count(t, s, "composition_members") != 2 || count(t, s, "composition_charges") != 1 {
		t.Fatal("reopen changed original members or debit")
	}
	again, err := s.AdmitInvocation(t.Context(), first.Invocation.OperationID)
	if err != nil || !reflect.DeepEqual(first, again) {
		t.Fatal(first, again, err)
	}
	if _, err = Replay(t.Context(), path, DefaultLimits()); err != nil {
		t.Fatal(err)
	}
}
