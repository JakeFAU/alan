package executor

import (
	"context"
	"database/sql"
	"errors"

	"github.com/mattn/go-sqlite3"
)

// RootCompletion exposes logical completion separately from local run status and
// retained external uncertainty. A non-success outcome does not erase evidence.
type RootCompletion struct {
	RootID             string
	Run                RunID
	LocalStatus        string
	Limits             ResourceLimits
	Charges            compositionCharge
	Outcome            *InvocationOutcome
	Pending            string
	PendingInvocations []string
	UnresolvedExternal []string
	UnknownExternal    []string
}

var compositionInspectionTables = []string{"composition_profile", "composition_roots", "composition_static_members", "composition_templates", "composition_facts", "composition_invocations", "composition_bindings", "composition_members", "composition_results", "composition_charges", "composition_deployment", "composition_permissions", "composition_closures", "composition_admission_decisions", "composition_requests", "composition_replies", "composition_freezes", "composition_reservations", "composition_publications", "composition_outcome_deliveries", "composition_root_publications"}

func readRootCompletions(ctx context.Context, q queryer) ([]RootCompletion, error) {
	reader := reportReader{q: q}
	return readRootCompletionsWithReader(ctx, &reader)
}

func readRootCompletionsWithReader(ctx context.Context, reader *reportReader) ([]RootCompletion, error) {
	q := reader.q
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
		root, err := reader.root(ctx, id)
		if err != nil {
			return nil, err
		}
		state, err := reader.run(ctx, root.Run)
		if err != nil {
			return nil, err
		}
		r := RootCompletion{RootID: id, Run: root.Run, LocalStatus: state.Projection.Status, Limits: root.Limits, PendingInvocations: []string{}, UnresolvedExternal: []string{}, UnknownExternal: []string{}}
		if index == 0 {
			charges, err = readRootReportCharges(ctx, q)
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
		prepared, err := reader.prepareRoot(ctx, id)
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

func readRootReportCharges(ctx context.Context, q queryer) (map[string]compositionCharge, error) {
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
