//go:build integration

// Frozen report traversal from 5f5d382. Install as report_frozen_test.go for
// differential tests. Scheduling and authoritative leaf decoders are shared.
package executor

import (
	"context"
	"database/sql"
	"errors"
	"sort"

	"github.com/mattn/go-sqlite3"
)

func (s *Store) finishRunFrozenReport(ctx context.Context, report RunReport, quiescent bool, runErr error) (RunReport, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	report.Quiescent = quiescent
	report.OpenRuns = []RunID{}
	report.UnresolvedIntents = []Intent{}
	report.PendingDeliveries = []string{}
	report.UnknownExternal = []string{}
	report.UnknownAdapters = []string{}
	if s.closed {
		return report, errors.Join(runErr, sql.ErrConnDone)
	}
	roots, err := readRootCompletionsFrozenReport(ctx, s.conn)
	if err != nil {
		return report, errors.Join(runErr, err)
	}
	report.Roots = roots
	ids, err := readRunIDs(ctx, s.conn)
	if err != nil {
		return report, errors.Join(runErr, err)
	}
	for _, id := range ids {
		run, err := loadRun(ctx, s.conn, id)
		if err != nil {
			return report, errors.Join(runErr, err)
		}
		if run.Projection.Status == "open" {
			report.OpenRuns = append(report.OpenRuns, id)
		}
	}
	intents, err := readIntents(ctx, s.conn)
	if err != nil {
		return report, errors.Join(runErr, err)
	}
	for id, intent := range intents {
		var count int
		if err = s.conn.QueryRowContext(ctx, "SELECT count(*) FROM message_acknowledgments WHERE intent_id=?", id).Scan(&count); err != nil {
			return report, errors.Join(runErr, err)
		}
		if count == 0 && intent.Kind == "invocation" {
			if err = s.conn.QueryRowContext(ctx, "SELECT count(*) FROM composition_outcome_deliveries WHERE operation_id=?", id).Scan(&count); err != nil {
				return report, errors.Join(runErr, err)
			}
		}
		if count == 0 && intent.Kind == "lifecycle" {
			if err = s.conn.QueryRowContext(ctx, "SELECT count(*) FROM settlements WHERE id=?", identity("lifecycle-outcome", id)).Scan(&count); err != nil {
				return report, errors.Join(runErr, err)
			}
		}
		if count == 0 && intent.Kind == "external" {
			if err = s.conn.QueryRowContext(ctx, "SELECT count(*) FROM settlements WHERE id=?", identity("external-outcome", id)).Scan(&count); err != nil {
				return report, errors.Join(runErr, err)
			}
		}
		if count == 0 && intent.Kind == "external" {
			a, e := loadExternalAttempt(ctx, s.conn, id)
			if e == nil && a.Allowed {
				// A retained receipt makes the physical result known even if its
				// separate local settlement has not committed yet.
				_, outcomeErr := externalOutcome(ctx, s.conn, id)
				if errors.Is(outcomeErr, sql.ErrNoRows) {
					report.UnknownExternal = append(report.UnknownExternal, id)
				} else if outcomeErr != nil {
					return report, errors.Join(runErr, outcomeErr)
				}
			} else if e != nil && !errors.Is(e, sql.ErrNoRows) {
				return report, errors.Join(runErr, e)
			}
		}
		if count == 0 && intent.Kind == "adapter" {
			if err = s.conn.QueryRowContext(ctx, "SELECT count(*) FROM settlements WHERE id=?", identity("adapter-outcome", id)).Scan(&count); err != nil {
				return report, errors.Join(runErr, err)
			}
			if count == 0 {
				a, e := loadAdapterAttempt(ctx, s.conn, id)
				if e == nil && a.Allowed {
					_, e = loadAdapterResult(ctx, s.conn, id)
					if errors.Is(e, sql.ErrNoRows) {
						report.UnknownAdapters = append(report.UnknownAdapters, id)
					} else if e != nil {
						return report, errors.Join(runErr, e)
					}
				} else if e != nil && !errors.Is(e, sql.ErrNoRows) {
					return report, errors.Join(runErr, e)
				}
			}
		}
		if count == 0 {
			report.UnresolvedIntents = append(report.UnresolvedIntents, intent.Intent)
		}
	}
	sort.Slice(report.UnresolvedIntents, func(i, j int) bool { return report.UnresolvedIntents[i].ID < report.UnresolvedIntents[j].ID })
	sort.Strings(report.UnknownExternal)
	sort.Strings(report.UnknownAdapters)
	pending, err := pendingDeliveriesFrozenReport(ctx, s.conn)
	if err != nil {
		return report, errors.Join(runErr, err)
	}
	for _, d := range pending {
		report.PendingDeliveries = append(report.PendingDeliveries, d.key)
	}
	rows, err := s.conn.QueryContext(ctx, "SELECT result FROM admissions")
	if err != nil {
		return report, errors.Join(runErr, err)
	}
	defer rows.Close()
	for rows.Next() {
		var raw []byte
		if err = rows.Scan(&raw); err != nil {
			return report, errors.Join(runErr, err)
		}
		var result AdmissionResult
		if err = decode(raw, &result); err != nil {
			return report, errors.Join(runErr, err)
		}
		if !result.Accepted {
			report.RejectedAdmissions++
		}
	}
	return report, errors.Join(runErr, rows.Err())
}

func pendingDeliveriesFrozenReport(ctx context.Context, q queryer) ([]pendingDelivery, error) {
	pending := []pendingDelivery{}
	intents, err := readIntents(ctx, q)
	if err != nil {
		return nil, err
	}
	for id, intent := range intents {
		if intent.Kind == "invocation" {
			var n int
			if err = q.QueryRowContext(ctx, "SELECT count(*) FROM composition_outcome_deliveries WHERE operation_id=?", id).Scan(&n); err != nil {
				return nil, err
			}
			if n == 0 {
				prepared, e := prepareInvocationFrozenReport(ctx, q, id)
				if e != nil && !errors.Is(e, sql.ErrNoRows) {
					return nil, e
				}
				if errors.Is(e, sql.ErrNoRows) || prepared.Ready {
					pending = append(pending, pendingDelivery{key: identity("invocation-delivery", id), intent: id, kind: "invocation"})
				}
			}
			continue
		}
		if intent.Kind == "adapter" {
			var count int
			if err = q.QueryRowContext(ctx, "SELECT count(*) FROM settlements WHERE id=?", identity("adapter-outcome", id)).Scan(&count); err != nil {
				return nil, err
			}
			if count == 0 {
				pending = append(pending, pendingDelivery{key: identity("adapter-delivery", id), intent: id, kind: "adapter"})
			}
			continue
		}
		if intent.Kind == "external" {
			var count int
			if err = q.QueryRowContext(ctx, "SELECT count(*) FROM settlements WHERE id=?", identity("external-outcome", id)).Scan(&count); err != nil {
				return nil, err
			}
			if count == 0 {
				pending = append(pending, pendingDelivery{key: identity("external-delivery", id), intent: id, kind: "external"})
			}
			continue
		}
		if intent.Kind == "lifecycle" {
			var count int
			if err = q.QueryRowContext(ctx, "SELECT count(*) FROM settlements WHERE id=?", identity("lifecycle-outcome", id)).Scan(&count); err != nil {
				return nil, err
			}
			if count == 0 {
				pending = append(pending, pendingDelivery{key: identity("lifecycle-delivery", id), intent: id, kind: "lifecycle"})
			}
			continue
		}
		if intent.Kind != "message" {
			continue
		}
		var count int
		if err = q.QueryRowContext(ctx, "SELECT count(*) FROM message_acknowledgments WHERE intent_id=?", id).Scan(&count); err != nil {
			return nil, err
		}
		if count == 0 {
			pending = append(pending, pendingDelivery{key: identity("message-delivery", id), intent: id})
		}
	}
	rows, err := q.QueryContext(ctx, "SELECT subject,generation,sequence,destinations,delivery_status FROM terminal_obligations ORDER BY subject,generation")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var source RunID
		var sequence int64
		var raw, status []byte
		if err = rows.Scan(&source.Subject, &source.Generation, &sequence, &raw, &status); err != nil {
			return nil, err
		}
		var destinations []TerminalDestination
		var delivered map[string]bool
		if err = decode(raw, &destinations); err != nil {
			return nil, err
		}
		if err = decode(status, &delivered); err != nil {
			return nil, err
		}
		for _, destination := range destinations {
			if !delivered[destination.key()] {
				pending = append(pending, pendingDelivery{key: identity("terminal-delivery", source.Subject, source.Generation, sequence, destination.Target, destination.Obligation), source: source, destination: destination})
			}
		}
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	sort.Slice(pending, func(i, j int) bool { return pending[i].key < pending[j].key })
	return pending, nil
}

func readRootCompletionsFrozenReport(ctx context.Context, q queryer) ([]RootCompletion, error) {
	enabled, err := compositionProtocol(ctx, q)
	if err != nil || !enabled {
		return nil, err
	}
	roots, err := stringList(ctx, q, "SELECT id FROM composition_roots ORDER BY id")
	if err != nil {
		return nil, err
	}
	intents, err := readIntents(ctx, q)
	if err != nil {
		return nil, err
	}
	// Both groupings are local to the caller's ownership/transaction cut.
	var charges map[string]compositionCharge
	operationsByRoot := make(map[string][]string)
	out := []RootCompletion{}
	for index, id := range roots {
		var raw []byte
		var root ResourceRoot
		if err = q.QueryRowContext(ctx, "SELECT descriptor FROM composition_roots WHERE id=?", id).Scan(&raw); err != nil {
			return nil, err
		}
		if err = decode(raw, &root); err != nil {
			return nil, err
		}
		state, err := loadRun(ctx, q, root.Run)
		if err != nil {
			return nil, err
		}
		r := RootCompletion{RootID: id, Run: root.Run, LocalStatus: state.Projection.Status, Limits: root.Limits, PendingInvocations: []string{}, UnresolvedExternal: []string{}, UnknownExternal: []string{}}
		if index == 0 {
			charges, err = readRootReportChargesFrozenReport(ctx, q)
			if err != nil {
				return nil, err
			}
		}
		r.Charges = charges[id]
		r.Charges.RootID = id
		if charges == nil {
			// When batching cannot reproduce a total, let SQLite evaluate
			// each root's original expressions at its original boundary.
			if err = q.QueryRowContext(ctx, "SELECT coalesce(sum(json_extract(descriptor,'$.Invocations')),0),coalesce(sum(json_extract(descriptor,'$.CreatedRuns')),0),coalesce(sum(json_extract(descriptor,'$.Effects')),0) FROM composition_charges WHERE root_id=?", id).Scan(&r.Charges.Invocations, &r.Charges.CreatedRuns, &r.Charges.Effects); err != nil {
				return nil, err
			}
		}
		prepared, err := prepareRootFrozenReport(ctx, q, id)
		if err != nil {
			return nil, err
		}
		r.Pending = prepared.Pending
		if prepared.Outcome.ID != "" {
			o := prepared.Outcome
			r.Outcome = &o
		}
		operations := operationsByRoot[id]
		var membershipRoots map[RunID]string
		if index == 0 {
			// Resolve membership during the first root's sorted traversal.
			// Its result checks must still precede any later lookups.
			operations = sortedKeys(intents)
			membershipRoots = make(map[RunID]string)
		}
		for _, operation := range operations {
			i := intents[operation]
			var e error
			if index == 0 {
				memberRoot, cached := membershipRoots[i.Run]
				if !cached {
					memberRoot, e = compositionMembershipRoot(ctx, q, i.Run)
					if e != nil {
						return nil, e
					}
					membershipRoots[i.Run] = memberRoot
				}
				if memberRoot != id {
					if memberRoot != "" {
						operationsByRoot[memberRoot] = append(operationsByRoot[memberRoot], operation)
					}
					continue
				}
			}
			switch i.Kind {
			case "invocation":
				var n int
				if e = q.QueryRowContext(ctx, "SELECT count(*) FROM composition_publications WHERE operation_id=?", operation).Scan(&n); e != nil {
					return nil, e
				}
				if n == 0 {
					r.PendingInvocations = append(r.PendingInvocations, operation)
				}
			case "adapter", "external":
				known, e := compositionExternalKnown(ctx, q, operation)
				if e != nil {
					return nil, e
				}
				if known {
					continue
				}
				r.UnresolvedExternal = append(r.UnresolvedExternal, operation)
				allowed := false
				if i.Kind == "adapter" {
					a, err := loadAdapterAttempt(ctx, q, operation)
					e = err
					allowed = a.Allowed
				} else {
					a, err := loadExternalAttempt(ctx, q, operation)
					e = err
					allowed = a.Allowed
				}
				if e != nil && !errors.Is(e, sql.ErrNoRows) {
					return nil, e
				}
				if e == nil && allowed {
					r.UnknownExternal = append(r.UnknownExternal, operation)
				}
			}
		}
		out = append(out, r)
	}
	return out, nil
}

func readRootReportChargesFrozenReport(ctx context.Context, q queryer) (map[string]compositionCharge, error) {
	rows, err := q.QueryContext(ctx, `
		WITH charges AS (
			SELECT root_id, json_extract(descriptor,'$.Invocations') AS invocations,
				json_extract(descriptor,'$.CreatedRuns') AS created_runs,
				json_extract(descriptor,'$.Effects') AS effects
			FROM composition_charges
		)
		SELECT root_id, sum(invocations), sum(created_runs), sum(effects),
			max(typeof(invocations) != 'integer' OR invocations < 0
				OR typeof(created_runs) != 'integer' OR created_runs < 0
				OR typeof(effects) != 'integer' OR effects < 0)
		FROM charges GROUP BY root_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	charges := make(map[string]compositionCharge)
	for rows.Next() {
		var root, invocations, createdRuns, effects any
		var outsideDomain bool
		if err = rows.Scan(&root, &invocations, &createdRuns, &effects, &outsideDomain); err != nil {
			return nil, err
		}
		if outsideDomain {
			// Signed or non-integer inputs can make SUM sensitive to scan
			// order. Retain the original per-root SQLite behavior for them.
			return nil, nil
		}
		id, textRoot := root.(string)
		invocationTotal, integerInvocations := invocations.(int64)
		createdRunTotal, integerCreatedRuns := createdRuns.(int64)
		effectTotal, integerEffects := effects.(int64)
		if !(textRoot && integerInvocations && integerCreatedRuns && integerEffects) {
			return nil, nil
		}
		charges[id] = compositionCharge{
			Invocations: invocationTotal,
			CreatedRuns: createdRunTotal,
			Effects:     effectTotal,
		}
	}
	err = rows.Err()
	var sqliteErr sqlite3.Error
	if errors.As(err, &sqliteErr) && sqliteErr.Code == sqlite3.ErrError {
		switch sqliteErr.Error() {
		case "malformed JSON", "integer overflow":
			// The failing row can belong to a root the traversal has not
			// reached. The original query reproduces its error in order.
			return nil, nil
		}
	}
	return charges, err
}

func prepareRootFrozenReport(ctx context.Context, q queryer, rootID string) (InvocationPreparation, error) {
	var root compositionRoot
	var raw []byte
	if err := q.QueryRowContext(ctx, "SELECT descriptor FROM composition_roots WHERE id=?", rootID).Scan(&raw); err != nil {
		return InvocationPreparation{}, err
	}
	if err := decode(raw, &root); err != nil {
		return InvocationPreparation{}, err
	}
	scope := "root:" + rootID
	p := InvocationPreparation{Outcome: compositionOutcome{InvocationID: scope, OperationID: scope, Caller: root.Run}}
	err := q.QueryRowContext(ctx, "SELECT descriptor FROM composition_root_publications WHERE scope_id=?", scope).Scan(&raw)
	if err == nil {
		if err = decode(raw, &p.Outcome); err != nil {
			return p, err
		}
		p.Ready = true
		return p, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return p, err
	}
	return prepareCompositionScopeFrozenReport(ctx, q, p, scope, false)
}

func prepareInvocationFrozenReport(ctx context.Context, q queryer, operation string) (InvocationPreparation, error) {
	a, err := loadCompositionAdmission(ctx, q, operation)
	if err != nil {
		return InvocationPreparation{}, err
	}
	p := InvocationPreparation{Outcome: compositionOutcome{InvocationID: a.Invocation.ID,
		OperationID: operation,
		TemplateID:  a.Invocation.TemplateID,
		Caller:      a.Invocation.Caller}}
	var published []byte
	publicationErr := q.QueryRowContext(ctx, "SELECT descriptor FROM composition_publications WHERE operation_id=?", operation).Scan(&published)
	if publicationErr == nil {
		if err = decode(published, &p.Outcome); err != nil {
			return p, err
		}
		if !sameJSON(published, p.Outcome) {
			return p, ErrCorrupt
		}
		p.Ready = true
		return p, nil
	}
	if !errors.Is(publicationErr, sql.ErrNoRows) {
		return p, publicationErr
	}
	if !a.Accepted {
		p.Ready = true
		p.Outcome.Status = "rejected"
		p.Outcome.Reason = a.Reason
		return p, nil
	}
	return prepareCompositionScopeFrozenReport(ctx, q, p, a.Invocation.ID, true)
}

func prepareCompositionScopeFrozenReport(ctx context.Context, q queryer, p InvocationPreparation, scope string, requireReply bool) (InvocationPreparation, error) {
	var err error
	var raw []byte
	err = q.QueryRowContext(ctx, "SELECT descriptor FROM composition_freezes WHERE scope_id=?", scope).Scan(&raw)
	hasFreeze := err == nil
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return p, err
	}
	if hasFreeze {
		if err = decode(raw, &p.Freeze); err != nil {
			return p, err
		}
		if !sameJSON(raw, p.Freeze) {
			return p, ErrCorrupt
		}
	}
	closure, closureErr := loadCompositionClosure(ctx, q, scope)
	if closureErr != nil && !errors.Is(closureErr, sql.ErrNoRows) {
		return p, closureErr
	}
	closureReason := ""
	if closureErr == nil && (!hasFreeze || p.Freeze.Status == "succeeded" || closure.Freeze.DecisionOrder < p.Freeze.DecisionOrder) {
		p.Freeze = closure.Freeze
		closureReason = closure.Reason
		hasFreeze = true
	}
	if !hasFreeze {
		p.Pending = "members"
		return p, nil
	}
	responses := []map[string]Value{}
	for _, id := range p.Freeze.ReplyIntents {
		var r compositionReply
		if err = q.QueryRowContext(ctx,
			"SELECT descriptor FROM composition_replies WHERE operation_id=?",
			id).Scan(&raw); errors.Is(err,
			sql.ErrNoRows) {
			p.Pending = "reply_admission"
			return p, nil
		}
		if err != nil {
			return p, err
		}
		if err = decode(raw, &r); err != nil {
			return p, err
		}
		if r.Accepted {
			responses = append(responses, r.Intent.Payload)
		}
	}
	p.Outcome.Status = p.Freeze.Status
	p.Outcome.Reason = "member_" + p.Freeze.Status
	if closureReason != "" {
		p.Outcome.Reason = closureReason
	}
	if p.Freeze.Status == "succeeded" {
		for _, id := range p.Freeze.InvocationIntents {
			child, e := loadCompositionAdmission(ctx, q, id)
			if errors.Is(e, sql.ErrNoRows) {
				p.Pending = "invocation_admission"
				return p, nil
			}
			if e != nil {
				return p, e
			}
			if child.Accepted {
				pending, e := compositionDescendantsPending(ctx, q, child, map[string]bool{})
				if e != nil {
					return p, e
				}
				if pending != "" {
					p.Pending = pending
					return p, nil
				}
				var n int
				if e = q.QueryRowContext(ctx, "SELECT count(*) FROM composition_publications WHERE operation_id=?", id).Scan(&n); e != nil {
					return p, e
				}
				if n == 0 {
					p.Pending = "descendant_outcome"
					return p, nil
				}
			}
		}
		for _, id := range p.Freeze.ExternalIntents {
			intent, e := loadIntent(ctx, q, id)
			if e != nil {
				return p, e
			}
			if intent.Kind == "adapter" {
				_, e = loadAdapterResult(ctx, q, id)
			} else {
				_, e = externalOutcome(ctx, q, id)
			}
			if errors.Is(e, sql.ErrNoRows) {
				p.Pending = "external_outcome"
				return p, nil
			}
			if e != nil {
				return p, e
			}
		}
		if requireReply && len(responses) == 0 {
			p.Outcome.Status = "failed"
			p.Outcome.Reason = "missing_reply"
		}
	}
	if len(p.Freeze.ReplyIntents) > 1 {
		// Conflicts remain retained evidence. They cannot replace the first
		// retirement closure or select one candidate as its public response.
		if closureReason == "" {
			p.Outcome.Status = "failed"
			p.Outcome.Reason = "conflicting_replies"
		}
	} else if len(responses) == 1 {
		p.Outcome.ResponsePresent = true
		p.Outcome.Response = responses[0]
	}
	p.Ready = true
	return p, nil
}
