//go:build integration

package executor

import (
	"context"
	"database/sql"
	"errors"
)

// readRootCompletionsOriginal is the report traversal from ef38ca4.
// Keep it independent of report grouping and grouped charge aggregation.
func readRootCompletionsOriginal(ctx context.Context, q queryer) ([]RootCompletion, error) {
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
	// Callers provide a stable ownership/transaction cut. Reuse membership,
	// including successful absence, only within this call.
	var membershipRoots map[RunID]string
	out := []RootCompletion{}
	for _, id := range roots {
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
		r.Charges.RootID = id
		if err = q.QueryRowContext(ctx, "SELECT coalesce(sum(json_extract(descriptor,'$.Invocations')),0),coalesce(sum(json_extract(descriptor,'$.CreatedRuns')),0),coalesce(sum(json_extract(descriptor,'$.Effects')),0) FROM composition_charges WHERE root_id=?", id).Scan(&r.Charges.Invocations, &r.Charges.CreatedRuns, &r.Charges.Effects); err != nil {
			return nil, err
		}
		prepared, err := prepareRoot(ctx, q, id)
		if err != nil {
			return nil, err
		}
		r.Pending = prepared.Pending
		if prepared.Outcome.ID != "" {
			o := prepared.Outcome
			r.Outcome = &o
		}
		for _, operation := range sortedKeys(intents) {
			i := intents[operation]
			memberRoot, cached := membershipRoots[i.Run]
			var e error
			if !cached {
				memberRoot, e = compositionMembershipRoot(ctx, q, i.Run)
				if e != nil {
					return nil, e
				}
				if membershipRoots == nil {
					membershipRoots = make(map[RunID]string)
				}
				membershipRoots[i.Run] = memberRoot
			}
			if memberRoot != id {
				continue
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
