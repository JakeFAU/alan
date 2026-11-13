package executor

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"unicode/utf8"

	"github.com/JakeFAU/alan/effects"
	"github.com/JakeFAU/alan/internal/boundary"
)

// ContinuationSlot reserves one local approval route for one distinct adapter
// operation on Target. Its action must match a paid operation on Predecessor.
type ContinuationSlot struct {
	Predecessor, Target            RunID
	Effect, Obligation, ApprovalID string
}

// ContinuationBasis names the exact predecessor evidence the operator reviewed.
// An empty ResultDigest means the original has no retained adapter response.
type ContinuationBasis struct {
	Namespace                                   string
	Target                                      RunID
	PredecessorID, PreparedDigest, ResultDigest string
	Admitted                                    bool
}

// ContinuationApproval is the immutable basis, declared purpose, prepared
// action scope and target authority at the approval decision.
type ContinuationApproval struct {
	ContinuationBasis
	Reason, ScopeDigest string
	TargetRevision      int64
}

func validContinuationReason(reason string) bool {
	return reason != "" && len(reason) <= 128 && utf8.ValidString(reason)
}

func continuationPayload(a ContinuationApproval) map[string]Value {
	return map[string]Value{
		"previous_operation_id": Text(a.PredecessorID), "purpose": Text(a.Reason),
		"prepared_digest": Text(a.PreparedDigest), "result_digest": Text(a.ResultDigest),
	}
}

// Providers which embed an operation identity in the wire bytes cannot use
// this same-action approval. All deployment settings and bytes stay in scope.
func continuationScope(p effects.Prepared) string {
	p.Request.ID, p.Request.Key = "", ""
	return effects.Freeze(p).Digest
}

func (s *Store) continuationSlot(target RunID) (ContinuationSlot, bool) {
	if s.composition == nil {
		return ContinuationSlot{}, false
	}
	slot, ok := s.composition.slots[target]
	return slot, ok
}

func loadContinuationSlots(ctx context.Context, q queryer, c compositionProvision, l Limits) error {
	var raw []byte
	var d compositionDeployment
	if err := q.QueryRowContext(ctx, "SELECT descriptor FROM composition_deployment WHERE singleton=1").Scan(&raw); err != nil {
		return err
	}
	if err := decode(raw, &d); err != nil {
		return err
	}
	var grants []Permission
	if err := q.QueryRowContext(ctx, "SELECT descriptor FROM composition_permissions WHERE singleton=1").Scan(&raw); err != nil {
		return err
	}
	if err := decode(raw, &grants); err != nil {
		return err
	}
	return validateContinuationSlots(c, d, grants, func(id string) (Program, error) { return loadProgram(ctx, q, id, l) })
}

func validateContinuationSlots(c compositionProvision, deploy compositionDeployment, permissions []Permission, lookup func(string) (Program, error)) error {
	if len(c.Continuations) == 0 {
		return nil
	}
	if !c.RecordDispatchAdmissions || c.Version != compositionProtocolVersion {
		return ErrVersion
	}
	roots := map[RunID]compositionRoot{}
	for _, root := range c.Roots {
		for _, m := range root.Members {
			roots[m] = root
		}
	}
	grants := map[Permission]bool{}
	for _, p := range permissions {
		grants[p] = true
	}
	revisions := map[RunID]int64{}
	for _, s := range deploy.Subjects {
		revisions[RunID{s.ID, s.Generation}] = s.Revision
	}
	programs := map[RunID]string{}
	for _, r := range deploy.Runs {
		programs[r.ID] = r.ProgramID
	}
	parents := map[RunID]RunID{}
	ids := map[string]bool{}
	for _, slot := range c.Continuations {
		root := roots[slot.Target]
		if slot.Target.Generation != 1 || slot.Predecessor.Generation != 1 || slot.Target == slot.Predecessor ||
			slot.Target.Subject == "" || slot.Predecessor.Subject == "" || slot.Effect == "" || slot.Obligation == "" || slot.ApprovalID == "" ||
			ids[slot.ApprovalID] || parents[slot.Target].Subject != "" || roots[slot.Predecessor].ID == "" ||
			root.Run != slot.Target || len(root.Members) != 1 || root.Limits.MaxEffects != 1 || len(root.Templates) != 0 {
			return corrupt("invalid single-use continuation slot")
		}
		parents[slot.Target], ids[slot.ApprovalID] = slot.Predecessor, true
		source, err := lookup(programs[slot.Predecessor])
		if err != nil {
			return err
		}
		target, err := lookup(programs[slot.Target])
		if err != nil {
			return err
		}
		if source.Author == nil || target.Author == nil {
			return corrupt("continuation requires an authored adapter")
		}
		e, ok := source.Author.Effects[slot.Effect]
		next, has := target.Author.Effects[slot.Effect]
		if !ok || !has || !reflect.DeepEqual(e, next) || !grants[Permission{"input", "local", slot.Target.Subject, "input", 0, slot.Obligation}] {
			return corrupt("continuation lacks matching action or approval route")
		}
		for _, id := range []RunID{slot.Predecessor, slot.Target} {
			if !grants[Permission{"adapter", id.Subject, e.Capability, slot.Effect, revisions[id], slot.Effect}] {
				return corrupt("continuation lacks original action grant")
			}
		}
		portOK := false
		for _, port := range target.Author.Ports {
			if port.Local && port.ReceiveUsed && port.Peer == "local" && port.Obligation == slot.Obligation &&
				len(port.Receive.Record) == 4 && port.Receive.Record["previous_operation_id"].Kind == "string" &&
				port.Receive.Record["purpose"].Kind == "string" && port.Receive.Record["prepared_digest"].Kind == "string" &&
				port.Receive.Record["result_digest"].Kind == "string" {
				portOK = true
			}
		}
		if !portOK {
			return corrupt("continuation lacks an authored approval receiver")
		}
	}
	// Each target has one parent. Mark each chain once, including shared prefixes.
	done := map[RunID]bool{}
	for target := range parents {
		chain := map[RunID]bool{}
		for id := target; id.Subject != "" && !done[id]; id = parents[id] {
			if chain[id] {
				return corrupt("cyclic continuation slots")
			}
			chain[id] = true
		}
		for id := range chain {
			done[id] = true
		}
	}
	return nil
}

func continuationFact(slot ContinuationSlot, d AdmissionDescriptor, r AdmissionResult, root string) compositionFact {
	return compositionFact{
		Version: compositionFactVersion, ID: compositionIdentity("alan/fact/continuation/v1", slot.ApprovalID),
		Kind: compositionContinuationFact, RootIDs: []string{root}, Run: slot.Target, OperationID: slot.ApprovalID,
		RecordDigest: compositionIdentity("alan/continuation-admission/v1", []any{d, r}),
	}
}

func continuationFactEvidence(ctx context.Context, q queryer, approvalID string) (compositionFact, error) {
	d, r, err := loadAdmission(ctx, q, "input", approvalID)
	if err != nil {
		return compositionFact{}, err
	}
	if d.Continuation == nil || !r.Accepted || r.Bound == nil || *r.Bound != d.Continuation.Target || d.LogicalID != approvalID || r.ObservationID != identity("observation", "input", approvalID) {
		return compositionFact{}, corrupt("continuation fact lacks its accepted approval")
	}
	root, err := compositionMembershipRoot(ctx, q, *r.Bound)
	if err != nil {
		return compositionFact{}, err
	}
	if root == "" {
		return compositionFact{}, corrupt("unscoped continuation approval")
	}
	return continuationFact(ContinuationSlot{Target: *r.Bound, ApprovalID: approvalID}, d, r, root), nil
}

// ApproveContinuation compares the operator's expected basis with the original
// paid adapter operation in the same transaction that admits the local input.
// Repeating the identical approval returns its immutable first decision. The
// operator chooses the reason and whether to spend. This call performs no I/O
// to a provider and grants no extra resource allowance.
func (s *Store) ApproveContinuation(ctx context.Context, basis ContinuationBasis, reason string) (AdmissionResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return AdmissionResult{}, sql.ErrConnDone
	}
	slot, ok := s.continuationSlot(basis.Target)
	if !ok || basis.Namespace != s.namespace || basis.PredecessorID == "" || !digestIdentity(basis.PreparedDigest) ||
		(basis.ResultDigest != "" && !digestIdentity(basis.ResultDigest)) || !validContinuationReason(reason) {
		return AdmissionResult{}, fmt.Errorf("invalid continuation approval: %w", ErrConflict)
	}
	tx, err := s.beginOwnedTx(ctx)
	if err != nil {
		return AdmissionResult{}, err
	}
	defer tx.Rollback()
	d, r, err := loadAdmission(ctx, tx, "input", slot.ApprovalID)
	if err == nil {
		if d.Continuation == nil || d.Continuation.ContinuationBasis != basis || d.Continuation.Reason != reason {
			return AdmissionResult{}, ErrConflict
		}
		return r, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return AdmissionResult{}, err
	}
	original, err := loadAdapterAttempt(ctx, tx, basis.PredecessorID)
	if errors.Is(err, sql.ErrNoRows) {
		return AdmissionResult{}, fmt.Errorf("continuation requires the original paid reservation: %w", ErrConflict)
	}
	if err != nil {
		return AdmissionResult{}, err
	}
	if !original.Allowed || original.Intent.Run != slot.Predecessor || original.Intent.Kind != "adapter" ||
		original.Intent.Obligation != slot.Effect || original.Prepared.Digest != basis.PreparedDigest {
		return AdmissionResult{}, fmt.Errorf("continuation predecessor reservation: %w", ErrConflict)
	}
	var admitted bool
	if err = tx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM adapter_dispatch_admissions WHERE intent_id=?)", basis.PredecessorID).Scan(&admitted); err != nil {
		return AdmissionResult{}, err
	}
	result, err := loadAdapterResult(ctx, tx, basis.PredecessorID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return AdmissionResult{}, err
	}
	if basis.Admitted != admitted || result.Digest != basis.ResultDigest {
		return AdmissionResult{}, fmt.Errorf("continuation predecessor evidence changed: %w", ErrStale)
	}
	state, err := loadRun(ctx, tx, slot.Target)
	if err != nil {
		return AdmissionResult{}, err
	}
	closed, err := compositionScopeClosed(ctx, tx, slot.Target)
	if err != nil {
		return AdmissionResult{}, err
	}
	if closed || state.Projection.Status != "open" {
		return AdmissionResult{}, ErrClosed
	}
	a := ContinuationApproval{ContinuationBasis: basis, Reason: reason, ScopeDigest: continuationScope(original.Prepared), TargetRevision: state.Revision}
	d = AdmissionDescriptor{TransportID: slot.ApprovalID, LogicalID: slot.ApprovalID, Kind: "input", Source: RunID{"local", 1}, Target: slot.Target.Subject, Generation: slot.Target.Generation, Obligation: slot.Obligation, Payload: continuationPayload(a), Continuation: &a}
	r, err = admit(ctx, tx, d, s.limits, "")
	if err != nil {
		return AdmissionResult{}, err
	}
	if !r.Accepted {
		return r, fmt.Errorf("continuation approval refused (%s): %w", r.Reason, ErrConflict)
	}
	root := compositionRootID(s.namespace, slot.Target)
	fact := continuationFact(slot, d, r, root)
	order, err := appendCompositionFact(ctx, tx, fact)
	if err != nil {
		return AdmissionResult{}, err
	}
	fact.Order = order
	if err := verifyContinuationWrite(ctx, tx, fact, d, r); err != nil {
		return AdmissionResult{}, err
	}
	event := boundary.Event{Boundary: "continuation.approval", Phase: "pre-commit", Subject: slot.Target.Subject, Generation: slot.Target.Generation, Epoch: state.Epoch, Sequence: state.Sequence, Intent: basis.PredecessorID, Attempt: slot.ApprovalID}
	if err = boundary.Reached(ctx, event); err != nil {
		return AdmissionResult{}, err
	}
	if err = s.commitOwnedTx(ctx, tx, executionDelta{run: slot.Target, observations: []string{r.ObservationID}}); err != nil {
		return AdmissionResult{}, err
	}
	event.Phase = "post-commit"
	return r, boundary.Reached(ctx, event)
}

// Verify the complete transaction before making the approval selectable.
func verifyContinuationWrite(ctx context.Context, q queryer, f compositionFact, d AdmissionDescriptor, r AdmissionResult) error {
	var fact, descriptor, result, observation []byte
	err := q.QueryRowContext(ctx, `SELECT f.descriptor, a.descriptor, a.result, o.descriptor
		FROM composition_facts f JOIN admissions a ON a.id=?
		JOIN observations o ON o.id=a.observation_id
		WHERE f.id=? AND f.kind=? AND f.decision_order=? AND a.domain='input' AND a.transport_id=?
		AND o.id=? AND o.subject=? AND o.generation=? AND o.consumed_sequence IS NULL`,
		admissionKey("input", d.TransportID), f.ID, f.Kind, f.Order, d.TransportID,
		r.ObservationID, d.Target, d.Generation).Scan(&fact, &descriptor, &result, &observation)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if err != nil || !sameJSON(fact, f) || !sameJSON(descriptor, d) || !sameJSON(result, r) || !sameJSON(observation, observationFor(d, *r.Bound)) {
		return corrupt("continuation decision, input and observation were not retained together")
	}
	return nil
}

// Verify the exact consumed observation before granting the successor an
// operation identity. No fabricated input or merely admitted observation can
// pass this guard, even when the lowest public Store APIs are used directly.
func selectedContinuation(ctx context.Context, q queryer, slot ContinuationSlot, sequence, revision int64) (ContinuationApproval, error) {
	d, r, err := loadAdmission(ctx, q, "input", slot.ApprovalID)
	if errors.Is(err, sql.ErrNoRows) {
		return ContinuationApproval{}, fmt.Errorf("successor requires a selected approval: %w", ErrIneligible)
	}
	if err != nil {
		return ContinuationApproval{}, err
	}
	if d.Continuation == nil || !r.Accepted || d.Continuation.TargetRevision != revision || d.Continuation.Target != slot.Target {
		return ContinuationApproval{}, ErrStale
	}
	var selected sql.NullInt64
	if err = q.QueryRowContext(ctx, "SELECT consumed_sequence FROM observations WHERE id=? AND subject=? AND generation=?", r.ObservationID, slot.Target.Subject, slot.Target.Generation).Scan(&selected); err != nil {
		return ContinuationApproval{}, err
	}
	if !selected.Valid || selected.Int64 >= sequence {
		return ContinuationApproval{}, fmt.Errorf("successor requires a previously selected continuation: %w", ErrIneligible)
	}
	return *d.Continuation, nil
}

func checkContinuationIntents(ctx context.Context, q queryer, slot ContinuationSlot, run RunState, intents []Intent) error {
	for _, i := range intents {
		if i.Kind == "message" {
			continue
		}
		if i.Kind != "adapter" || i.Obligation != slot.Effect {
			return fmt.Errorf("operation outside continuation scope: %w", ErrConflict)
		}
		a, err := selectedContinuation(ctx, q, slot, i.Sequence, run.Revision)
		if err != nil {
			return err
		}
		original, err := loadAdapterAttempt(ctx, q, a.PredecessorID)
		if err != nil {
			return err
		}
		if original.Intent.Run != slot.Predecessor || !original.Allowed || i.Target != original.Intent.Target ||
			i.Generation != original.Intent.Generation || !reflect.DeepEqual(i.Payload, original.Intent.Payload) {
			return fmt.Errorf("successor action differs from approved predecessor: %w", ErrConflict)
		}
		var used bool
		if err = q.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM intents WHERE subject=? AND generation=?
			AND json_extract(descriptor,'$.Kind')='adapter')`, i.Run.Subject, i.Run.Generation).Scan(&used); err != nil {
			return err
		}
		if used {
			return fmt.Errorf("continuation slot already used: %w", ErrConflict)
		}
	}
	return nil
}

func checkContinuationPreparation(ctx context.Context, q queryer, slot ContinuationSlot, i retainedIntent, p effects.Prepared) error {
	a, err := selectedContinuation(ctx, q, slot, i.Sequence, i.revision)
	if err != nil {
		return err
	}
	if continuationScope(p) != a.ScopeDigest {
		return fmt.Errorf("prepared action differs from approval: %w", ErrConflict)
	}
	return nil
}
