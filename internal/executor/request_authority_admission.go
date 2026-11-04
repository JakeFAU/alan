package executor

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"reflect"
	"slices"
)

const compositionAuthorityAdmissionFact compositionFactKind = "authority_admission"

// AuthorityAdmission records a local opportunity, not proof of endpoint contact.
type AuthorityAdmission struct {
	Version, Namespace string
	Ordinal            int64
	Source             Authority
	AttemptDigest      string
	Allowed            bool
	Reason             string
}

func authorityAttempt(ctx context.Context, q queryer, i Intent) (any, *EffectAuthority, bool, error) {
	switch i.Kind {
	case "external":
		a, e := loadExternalAttempt(ctx, q, i.ID)
		if e == nil && (!reflect.DeepEqual(a.Intent, i) || a.AuthorizationID != identity("authorization", i.ID)) {
			e = ErrCorrupt
		}
		return a, a.Authority, a.Allowed, e
	case "adapter":
		a, e := loadAdapterAttempt(ctx, q, i.ID)
		if e == nil && (!reflect.DeepEqual(a.Intent, i) || a.AuthorizationID != identity("authorization", i.ID)) {
			e = ErrCorrupt
		}
		return a, a.Authority, a.Allowed, e
	default:
		return nil, nil, false, ErrVersion
	}
}
func expectedAuthorityAdmission(ctx context.Context, q queryer, id string, source Authority, prefix int64) (compositionFact, error) {
	i, err := loadIntent(ctx, q, id)
	if err != nil {
		return compositionFact{}, err
	}
	x, err := readAuthorityPrefix(ctx, q, prefix)
	if err != nil {
		return compositionFact{}, err
	}
	a, w, allowed, err := authorityAttempt(ctx, q, i.Intent)
	if err != nil {
		return compositionFact{}, err
	}
	if !allowed || w == nil || !w.Allowed || !x.has(compositionReservationFact, id) {
		return compositionFact{}, corrupt("gate without allowed original reservation")
	}
	var reservationRoot string
	var reservationRaw []byte
	if err = q.QueryRowContext(ctx, "SELECT root_id,descriptor FROM composition_reservations WHERE operation_id=?", id).Scan(&reservationRoot, &reservationRaw); err != nil {
		return compositionFact{}, err
	}
	reservation := compositionReservation{OperationID: id, RootID: w.ServingRootID, Kind: i.Kind, Run: i.Run, Sequence: i.Sequence, Allowed: true, Reason: "authorized", AttemptDigest: compositionIdentity("alan/attempt-row/v1", a)}
	if reservationRoot != w.ServingRootID || !sameJSON(reservationRaw, reservation) {
		return compositionFact{}, corrupt("gate original reservation mismatch")
	}
	reservationOrder, err := authorityFactOrder(ctx, q, compositionReservationFact, id)
	if err != nil {
		return compositionFact{}, err
	}
	var chargeRoot string
	var chargeOrder int64
	var chargeRaw []byte
	if err = q.QueryRowContext(ctx, "SELECT root_id,decision_order,descriptor FROM composition_charges WHERE operation_id=? AND kind='effect'", id).Scan(&chargeRoot, &chargeOrder, &chargeRaw); err != nil {
		return compositionFact{}, err
	}
	charge := compositionCharge{RootID: w.ServingRootID, OperationID: id, Kind: "effect", Effects: 1, DecisionOrder: reservationOrder}
	if chargeRoot != w.ServingRootID || chargeOrder != reservationOrder || !sameJSON(chargeRaw, charge) {
		return compositionFact{}, corrupt("gate original debit mismatch")
	}
	auth, err := loadAuthorization(ctx, q, id)
	if err != nil {
		return compositionFact{}, err
	}
	if !auth.Allowed || !x.has(compositionAuthorizationFact, id) {
		return compositionFact{}, corrupt("gate without original source allowance")
	}
	if _, err = retainedEffectAuthority(ctx, q, i, w); err != nil {
		return compositionFact{}, err
	}
	st, ok := x.states[i.Run]
	if !ok || source.Run != i.Run || source.Sequence != st.Sequence || source.Revision != st.Revision {
		return compositionFact{}, corrupt("gate source prefix mismatch")
	}
	run, err := loadRun(ctx, q, i.Run)
	if err != nil {
		return compositionFact{}, err
	}
	if err = validateWitnessEpoch(ctx, q, run, source.Sequence, source.Epoch); err != nil {
		return compositionFact{}, err
	}
	ordinal := int64(1)
	for _, f := range x.facts {
		if f.Kind == compositionAuthorityAdmissionFact && f.OperationID == id {
			if f.AuthorityAdmission == nil || f.AuthorityAdmission.Ordinal != ordinal {
				return compositionFact{}, ErrCorrupt
			}
			ordinal++
		}
	}
	d := &AuthorityAdmission{Version: RequestAuthorityVersion, Namespace: x.namespace, Ordinal: ordinal, Source: source, AttemptDigest: compositionIdentity("alan/attempt-row/v1", a), Allowed: true, Reason: "authorized"}
	switch {
	case source.Revision != i.revision:
		d.Allowed = false
		d.Reason = "source_revision_changed"
	case st.Retired:
		d.Allowed = false
		d.Reason = "source_retired"
	case st.Projection.Status != "open":
		d.Allowed = false
		d.Reason = "source_closed"
	default:
		for _, r := range x.profile.Roots {
			if r.ID == w.ServingRootID && x.states[r.Run].Projection.Status != "open" {
				d.Allowed = false
				d.Reason = "scope_closed"
			}
		}
		for _, f := range x.facts {
			if f.Kind == compositionFreezeFact && f.OperationID == "root:"+w.ServingRootID {
				var raw []byte
				var freeze compositionFreeze
				if err = q.QueryRowContext(ctx, "SELECT descriptor FROM composition_freezes WHERE scope_id=?", f.OperationID).Scan(&raw); err != nil {
					return compositionFact{}, err
				}
				if err = decode(raw, &freeze); err != nil {
					return compositionFact{}, err
				}
				if freeze.Status != "succeeded" {
					d.Allowed = false
					d.Reason = "scope_closed"
				}
			}
		}
		var closures int
		if err = q.QueryRowContext(ctx, "SELECT count(*) FROM composition_closures WHERE scope_id=? AND decision_order<?", "root:"+w.ServingRootID, prefix).Scan(&closures); err != nil {
			return compositionFact{}, err
		}
		if closures > 0 {
			d.Allowed = false
			d.Reason = "scope_closed"
		}
	}
	if d.Allowed {
		current, e := evaluateEffectAuthority(ctx, q, i, prefix)
		if e != nil {
			return compositionFact{}, e
		}
		d.Allowed = current.Allowed
		d.Reason = current.Reason
	}
	roots := []string{w.ServingRootID}
	if w.Use.Path == "delegated" {
		issuer := x.memberRoot(w.Use.Root.Run)
		if issuer == "" || issuer != w.RootID {
			return compositionFact{}, ErrCorrupt
		}
		if !slices.Contains(roots, issuer) {
			roots = append(roots, issuer)
		}
	}
	slices.Sort(roots)
	return compositionFact{Version: compositionFactVersion, Kind: compositionAuthorityAdmissionFact, Run: source.Run, Sequence: source.Sequence, OperationID: id, RootIDs: roots, ID: compositionIdentity("alan/fact/authority-admission/v1", []any{x.namespace, id, ordinal}), RecordDigest: compositionIdentity("alan/authority-admission/v1", d), AuthorityAdmission: d}, nil
}
func validateAuthorityAdmission(ctx context.Context, q queryer, f compositionFact) error {
	if f.Kind != compositionAuthorityAdmissionFact || f.AuthorityAdmission == nil {
		return ErrVersion
	}
	want, err := expectedAuthorityAdmission(ctx, q, f.OperationID, f.AuthorityAdmission.Source, f.Order)
	if err != nil {
		return err
	}
	want.Order = f.Order
	if !reflect.DeepEqual(want, f) {
		return corrupt("authority admission differs from historical policy")
	}
	return nil
}

// Caller holds both Store locks. This is the sole primary-fact writer; it derives
// the decision itself and commits before a volatile opportunity is released.
func (s *Store) recordAuthorityAdmission(ctx context.Context, id string) (*AuthorityAdmission, error) {
	tx, err := beginOwnedTx(ctx, s.conn)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	i, err := loadIntent(ctx, tx, id)
	if err != nil {
		return nil, err
	}
	run, err := loadRun(ctx, tx, i.Run)
	if err != nil {
		return nil, err
	}
	// Agreement between the attempt and its reservation digest is insufficient:
	// reconstruct the checkable request from the immutable original before a
	// fresh opportunity can escape. Adapter Wire remains trusted ABI output.
	switch i.Kind {
	case "external":
		a, err := loadExternalAttempt(ctx, tx, id)
		if err != nil {
			return nil, err
		}
		if err = validateExternalAttempt(ctx, tx, a, s.namespace); err != nil {
			return nil, err
		}
	case "adapter":
		a, err := loadAdapterAttempt(ctx, tx, id)
		if err != nil {
			return nil, err
		}
		if err = validateAdapterAttempt(ctx, tx, a, s.namespace, s.limits); err != nil {
			return nil, err
		}
	default:
		return nil, ErrVersion
	}
	f, err := expectedAuthorityAdmission(ctx, tx, id, run.Authority(), authorityCurrentPrefix)
	if err != nil {
		return nil, err
	}
	if err = tx.QueryRowContext(ctx, "SELECT coalesce(max(decision_order),0)+1 FROM composition_facts").Scan(&f.Order); err != nil {
		return nil, err
	}
	if f.Order <= 0 {
		return nil, ErrLimit
	}
	raw, err := json.Marshal(f)
	if err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO composition_facts VALUES(?,?,?,?)", f.Order, f.ID, f.Kind, raw); err != nil {
		return nil, err
	}
	var retained []byte
	if err = tx.QueryRowContext(ctx, "SELECT descriptor FROM composition_facts WHERE id=? AND decision_order=? AND kind=?", f.ID, f.Order, f.Kind).Scan(&retained); err != nil {
		return nil, err
	}
	if !sameJSON(retained, f) {
		return nil, corrupt("authority admission not retained exactly")
	}
	if err = ctx.Err(); err != nil {
		return nil, err
	}
	// ownerMu is already held: commitOwnedTx would recursively acquire it.
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return f.AuthorityAdmission, nil
}
func requireAuthorityExposure(ctx context.Context, q queryer, id string, before int64) error {
	enabled, err := requestAuthorityEnabled(ctx, q)
	if err != nil || !enabled {
		return err
	}
	facts, err := readCompositionFacts(ctx, q)
	if err != nil {
		return err
	}
	for _, f := range facts {
		if f.Order >= before {
			break
		}
		if f.Kind == compositionAuthorityAdmissionFact && f.OperationID == id && f.AuthorityAdmission != nil && f.AuthorityAdmission.Allowed {
			if err = validateAuthorityAdmission(ctx, q, f); err != nil {
				return err
			}
			return nil
		}
	}
	return corrupt("result lacks earlier allowed authority admission")
}
func validateRequestAuthorityHistory(ctx context.Context, q queryer) error {
	enabled, err := requestAuthorityEnabled(ctx, q)
	if err != nil || !enabled {
		return err
	}
	facts, err := readCompositionFacts(ctx, q)
	if err != nil {
		return err
	}
	for _, f := range facts {
		if f.Kind != compositionAuthorityAdmissionFact && f.AuthorityAdmission != nil {
			return ErrVersion
		}
		switch f.Kind {
		case compositionLifecycleFact:
			r, e := loadLifecycleResult(ctx, q, f.OperationID)
			if e != nil {
				return e
			}
			if !authorityOperation(r.Request.Operation) {
				if r.Authority != nil {
					return ErrVersion
				}
				continue
			}
			req, e := loadLifecycleRequest(ctx, q, f.OperationID)
			if e != nil {
				return e
			}
			if !reflect.DeepEqual(req, r.Request) {
				return ErrCorrupt
			}
			want, e := decideRequestAuthority(ctx, q, req, r.DecisionSequence, r.Before, f.Order)
			if e != nil {
				return e
			}
			if !reflect.DeepEqual(want, r) {
				return corrupt("authority lifecycle prefix mismatch")
			}
		case compositionReservationFact:
			i, e := loadIntent(ctx, q, f.OperationID)
			if e != nil {
				return e
			}
			_, w, _, e := authorityAttempt(ctx, q, i.Intent)
			if e != nil {
				return e
			}
			if _, e = retainedEffectAuthority(ctx, q, i, w); e != nil {
				return e
			}
		case compositionAuthorityAdmissionFact:
			if e := validateAuthorityAdmission(ctx, q, f); e != nil {
				return e
			}
		case compositionExternalResultFact:
			i, e := loadIntent(ctx, q, f.OperationID)
			if e != nil {
				return e
			}
			_, _, allowed, e := authorityAttempt(ctx, q, i.Intent)
			if e != nil && !errors.Is(e, sql.ErrNoRows) {
				return e
			}
			if allowed {
				if e = requireAuthorityExposure(ctx, q, f.OperationID, f.Order); e != nil {
					return e
				}
			}
		}
	}
	return nil
}
