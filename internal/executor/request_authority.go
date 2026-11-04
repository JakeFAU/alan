package executor

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"reflect"
	"slices"
	"unicode/utf8"
)

const RequestAuthorityVersion = "alan/request-authority/v1"
const requestAuthoritySchema = 13
const requestAuthorityConflictSchema = 14

// AuthorityRoot names one original holder, scope and complete permission row.
type AuthorityRoot struct {
	Run        RunID
	RootRun    RunID
	Permission Permission
}
type DelegationClaim struct {
	Version                               string
	Root                                  AuthorityRoot
	Recipient                             RunID
	RecipientRoot                         RunID
	RecipientRevision                     int64
	OriginalRequestID, ServiceOperationID string
}
type RevocationClaim struct {
	Version string
	Root    AuthorityRoot
}
type AuthorityUse struct {
	Version, Path string
	Root          AuthorityRoot
	GrantID       string
}
type AuthorityDecision struct {
	Version, Namespace      string
	Root                    AuthorityRoot
	RootID, RecipientRootID string
}
type EffectAuthority struct {
	Version, Namespace                                          string
	Use                                                         *AuthorityUse
	RootID, ServingRootID                                       string
	Allowed                                                     bool
	Reason                                                      string
	OriginalRequestID, ServiceOperationID, RequestObservationID string
	RequestConsumedSequence, GrantDecisionSequence              int64
}

func hasRequestAuthoritySchema(v int) bool {
	return v == requestAuthoritySchema || v == requestAuthorityConflictSchema
}
func requestAuthorityEnabled(ctx context.Context, q queryer) (bool, error) {
	v, e := storedSchemaNumber(ctx, q)
	return hasRequestAuthoritySchema(v), e
}
func authorityOperation(op string) bool { return op == "DelegateAuthority" || op == "RevokeAuthority" }
func validAuthorityRun(r RunID) bool    { return r.Subject != "" && r.Generation > 0 }
func validateAuthorityRoot(r AuthorityRoot) error {
	p := r.Permission
	for _, v := range []string{r.Run.Subject, r.RootRun.Subject, p.Kind, p.Source, p.Target, p.Operation, p.Obligation} {
		if !utf8.ValidString(v) {
			return fmt.Errorf("invalid UTF-8 authority root")
		}
	}
	if !validAuthorityRun(r.Run) || !validAuthorityRun(r.RootRun) || p.Source == "" || p.Target == "" || p.Operation == "" || p.Obligation == "" || p.Revision <= 0 || (p.Kind != "external" && p.Kind != "adapter") {
		return fmt.Errorf("invalid authority root encoding")
	}
	return nil
}
func validateAuthorityUse(u *AuthorityUse) error {
	if u == nil {
		return nil
	}
	if !utf8.ValidString(u.GrantID) {
		return fmt.Errorf("invalid UTF-8 grant identity")
	}
	if u.Version != RequestAuthorityVersion || (u.Path != "own" && u.Path != "delegated") || (u.Path == "own" && u.GrantID != "") || (u.Path == "delegated" && u.GrantID == "") {
		return ErrVersion
	}
	return validateAuthorityRoot(u.Root)
}
func parseAuthorityClaim(r LifecycleRequest) (DelegationClaim, RevocationClaim, error) {
	var d DelegationClaim
	var v RevocationClaim
	c, ok := r.Intent.Payload["claim"]
	if !ok || c.Kind != "string" || len(r.Intent.Payload) != 3 {
		return d, v, fmt.Errorf("invalid authority control payload")
	}
	raw := []byte(c.Text)
	if r.Operation == "DelegateAuthority" {
		if err := decode(raw, &d); err != nil {
			return d, v, err
		}
		if !sameJSON(raw, d) || d.Version != RequestAuthorityVersion || !validAuthorityRun(d.Recipient) || !validAuthorityRun(d.RecipientRoot) || d.RecipientRevision <= 0 || d.OriginalRequestID == "" {
			return d, v, ErrCorrupt
		}
		if _, _, err := authorityOperationLocator(d.ServiceOperationID); err != nil {
			return d, v, err
		}
		return d, v, validateAuthorityRoot(d.Root)
	}
	if r.Operation != "RevokeAuthority" {
		return d, v, ErrVersion
	}
	if err := decode(raw, &v); err != nil {
		return d, v, err
	}
	if !sameJSON(raw, v) || v.Version != RequestAuthorityVersion {
		return d, v, ErrCorrupt
	}
	return d, v, validateAuthorityRoot(v.Root)
}
func authorityOperationLocator(id string) (RunID, int64, error) {
	var fields []json.RawMessage
	if err := decode([]byte(id), &fields); err != nil || len(fields) != 4 {
		return RunID{}, 0, fmt.Errorf("invalid authority operation locator")
	}
	var run RunID
	var seq int64
	var ordinal int
	if json.Unmarshal(fields[0], &run.Subject) != nil || json.Unmarshal(fields[1], &run.Generation) != nil || json.Unmarshal(fields[2], &seq) != nil || json.Unmarshal(fields[3], &ordinal) != nil || !validAuthorityRun(run) || seq <= 0 || ordinal != 0 || identity(run.Subject, run.Generation, seq, 0) != id {
		return run, seq, fmt.Errorf("noncanonical authority operation locator")
	}
	return run, seq, nil
}

// authorityPrefix reconstructs authority from immutable origins and the earlier
// shared journal. Current projections/revocations are never historical oracles.
type authorityPrefix struct {
	namespace string
	profile   compositionProvision
	facts     []compositionFact
	states    map[RunID]TargetState
	revoked   map[AuthorityRoot]bool
}

func readAuthorityPrefix(ctx context.Context, q queryer, before int64) (*authorityPrefix, error) {
	enabled, err := requestAuthorityEnabled(ctx, q)
	if err != nil {
		return nil, err
	}
	if !enabled {
		return nil, ErrVersion
	}
	x := &authorityPrefix{states: map[RunID]TargetState{}, revoked: map[AuthorityRoot]bool{}}
	x.namespace, err = readNamespace(ctx, q)
	if err != nil {
		return nil, err
	}
	var raw []byte
	if err = q.QueryRowContext(ctx, "SELECT descriptor FROM composition_profile WHERE singleton=1").Scan(&raw); err != nil {
		return nil, err
	}
	if err = decode(raw, &x.profile); err != nil {
		return nil, err
	}
	if x.profile.RequestAuthority != RequestAuthorityVersion {
		return nil, ErrVersion
	}
	for _, root := range x.profile.Roots {
		for _, id := range root.Members {
			origin, _, e := loadRoot(ctx, q, id.Subject)
			if e != nil {
				return nil, e
			}
			x.states[id] = TargetState{Run: id, ProgramID: origin.ProgramID, Owner: origin.Subject.Owner, Revision: origin.Subject.Revision, Epoch: 1, Retired: origin.Subject.Retired, RetryCancelled: origin.Subject.RetryCancelled, Projection: origin.Initial}
		}
	}
	facts, err := readCompositionFacts(ctx, q)
	if err != nil {
		return nil, err
	}
	for _, f := range facts {
		if f.Order >= before {
			break
		}
		x.facts = append(x.facts, f)
		switch f.Kind {
		case compositionExecutionFact:
			st, ok := x.states[f.Run]
			if !ok {
				return nil, ErrCorrupt
			}
			st.Sequence = f.Sequence
			st.Projection, err = projectionAt(ctx, q, f.Run, f.Sequence)
			if err != nil {
				return nil, err
			}
			if err = q.QueryRowContext(ctx, "SELECT epoch FROM history WHERE subject=? AND generation=? AND sequence=?", f.Run.Subject, f.Run.Generation, f.Sequence).Scan(&st.Epoch); err != nil {
				return nil, err
			}
			x.states[f.Run] = st
		case compositionLifecycleFact:
			r, e := loadLifecycleResult(ctx, q, f.OperationID)
			if e != nil {
				return nil, e
			}
			if r.Accepted && r.After != nil {
				x.states[r.After.Run] = *r.After
			}
			if r.Accepted && r.Request.Operation == "RevokeAuthority" {
				_, v, e := parseAuthorityClaim(r.Request)
				if e != nil {
					return nil, e
				}
				x.revoked[v.Root] = true
			}
		}
	}
	return x, nil
}
func (x *authorityPrefix) has(kind compositionFactKind, id string) bool {
	for _, f := range x.facts {
		if f.Kind == kind && f.OperationID == id {
			return true
		}
	}
	return false
}
func (x *authorityPrefix) committed(i Intent) bool {
	if i.ID != identity(i.Run.Subject, i.Run.Generation, i.Sequence, i.Ordinal) || i.Ordinal != 0 || i.Sequence <= 0 {
		return false
	}
	for _, f := range x.facts {
		if f.Kind == compositionExecutionFact && f.Run == i.Run && f.Sequence == i.Sequence {
			return true
		}
	}
	return false
}
func (x *authorityPrefix) memberRoot(run RunID) string {
	for _, r := range x.profile.Roots {
		if slices.Contains(r.Members, run) {
			return r.ID
		}
	}
	return ""
}
func (x *authorityPrefix) rootMember(run, locator RunID) bool {
	for _, r := range x.profile.Roots {
		if r.Run == locator && r.ID == compositionRootID(x.namespace, locator) && slices.Contains(r.Members, run) {
			return true
		}
	}
	return false
}
func (x *authorityPrefix) rootReason(ctx context.Context, q queryer, r AuthorityRoot, allowRevoked bool) (string, error) {
	if r.Permission.Source != r.Run.Subject {
		return "authority_holder_mismatch", nil
	}
	if !x.rootMember(r.Run, r.RootRun) {
		return "authority_root_mismatch", nil
	}
	st, ok := x.states[r.Run]
	if !ok || st.Run != r.Run || r.Run.Generation != 1 {
		return "authority_generation_changed", nil
	}
	if st.Revision != r.Permission.Revision {
		return "authority_revision_changed", nil
	}
	if st.Retired {
		return "authority_retired", nil
	}
	var raw []byte
	var permissions []Permission
	if err := q.QueryRowContext(ctx, "SELECT descriptor FROM composition_permissions WHERE singleton=1").Scan(&raw); err != nil {
		return "", err
	}
	if err := decode(raw, &permissions); err != nil {
		return "", err
	}
	if !slices.Contains(permissions, r.Permission) {
		return "authority_permission_missing", nil
	}
	if !allowRevoked && x.revoked[r] {
		return "authority_revoked", nil
	}
	return "", nil
}
func authorityFactOrder(ctx context.Context, q queryer, kind compositionFactKind, id string) (int64, error) {
	facts, err := readCompositionFacts(ctx, q)
	if err != nil {
		return 0, err
	}
	for _, f := range facts {
		if f.Kind == kind && f.OperationID == id {
			return f.Order, nil
		}
	}
	return 0, corrupt("authority decision lacks ordered fact")
}
func decideRequestAuthority(ctx context.Context, q queryer, r LifecycleRequest, order int64, before *TargetState, prefix int64) (LifecycleResult, error) {
	d, v, err := parseAuthorityClaim(r)
	if err != nil {
		return LifecycleResult{}, err
	}
	root := v.Root
	if r.Operation == "DelegateAuthority" {
		root = d.Root
	}
	x, err := readAuthorityPrefix(ctx, q, prefix)
	if err != nil {
		return LifecycleResult{}, err
	}
	out := LifecycleResult{Request: r, DecisionSequence: order, Before: before, After: before, Authority: &AuthorityDecision{Version: RequestAuthorityVersion, Namespace: x.namespace, Root: root, RootID: compositionRootID(x.namespace, root.RootRun)}}
	if r.Operation == "DelegateAuthority" {
		out.Authority.RecipientRootID = compositionRootID(x.namespace, d.RecipientRoot)
	}
	deny := func(reason string) (LifecycleResult, error) { out.Reason = reason; return out, nil }
	if before == nil {
		return deny("target_missing")
	}
	source, ok := x.states[r.Intent.Run]
	if !x.committed(r.Intent) || !x.has(compositionAuthorizationFact, r.Intent.ID) {
		return out, corrupt("authority control before committed authorization")
	}
	if r.Intent.Run != root.Run || r.SourceRevision != root.Permission.Revision {
		return deny("authority_origin_mismatch")
	}
	if !ok || source.Retired || source.Revision != r.SourceRevision || source.Projection.Status != "open" {
		return deny("authority_source_closed")
	}
	reason, err := x.rootReason(ctx, q, root, r.Operation == "RevokeAuthority")
	if err != nil {
		return out, err
	}
	if reason != "" {
		return deny(reason)
	}
	if r.Operation == "RevokeAuthority" {
		if r.Intent.Target != root.Run.Subject || r.Intent.Generation != root.Run.Generation || r.ExpectedRevision != root.Permission.Revision {
			return deny("authority_target_mismatch")
		}
		out.Accepted = true
		out.Reason = "revoked"
		if x.revoked[root] {
			out.Reason = "already_revoked"
		}
		return out, nil
	}
	if r.Intent.Target != d.Recipient.Subject || r.Intent.Generation != d.Recipient.Generation || r.ExpectedRevision != d.RecipientRevision {
		return deny("authority_recipient_mismatch")
	}
	recipient, ok := x.states[d.Recipient]
	if !ok || recipient.Retired || recipient.Revision != d.RecipientRevision || recipient.Projection.Status != "open" {
		return deny("authority_recipient_closed")
	}
	if !x.rootMember(d.Recipient, d.RecipientRoot) {
		return deny("authority_recipient_root_mismatch")
	}
	request, err := loadIntent(ctx, q, d.OriginalRequestID)
	if errors.Is(err, sql.ErrNoRows) {
		return deny("authority_request_missing")
	}
	if err != nil {
		return out, err
	}
	if !x.committed(request.Intent) || request.Kind != "message" || request.Run != root.Run || request.revision != root.Permission.Revision || request.Target != d.Recipient.Subject || request.Generation != d.Recipient.Generation {
		return deny("authority_request_mismatch")
	}
	run, _, err := authorityOperationLocator(d.ServiceOperationID)
	if err != nil {
		return out, err
	}
	if run != d.Recipient {
		return deny("authority_service_mismatch")
	}
	out.Accepted = true
	out.Reason = "delegated"
	return out, nil
}
func evaluateEffectAuthority(ctx context.Context, q queryer, i retainedIntent, prefix int64) (*EffectAuthority, error) {
	x, err := readAuthorityPrefix(ctx, q, prefix)
	if err != nil {
		return nil, err
	}
	w := &EffectAuthority{Version: RequestAuthorityVersion, Namespace: x.namespace, Use: i.Authority, ServingRootID: x.memberRoot(i.Run)}
	if w.ServingRootID == "" {
		return nil, ErrCorrupt
	}
	deny := func(reason string) (*EffectAuthority, error) { w.Reason = reason; return w, nil }
	if i.Authority == nil {
		return deny("authority_missing")
	}
	u := i.Authority
	if err = validateAuthorityUse(u); err != nil {
		return nil, err
	}
	w.RootID = compositionRootID(x.namespace, u.Root.RootRun)
	if !x.committed(i.Intent) {
		return nil, corrupt("effect before original commit")
	}
	reason, err := x.rootReason(ctx, q, u.Root, false)
	if err != nil {
		return nil, err
	}
	if reason != "" {
		return deny(reason)
	}
	p := u.Root.Permission
	op := "apply"
	if i.Kind == "adapter" {
		op = i.Obligation
	}
	if p.Kind != i.Kind || p.Target != i.Target || p.Operation != op || p.Obligation != i.Obligation {
		return deny("authority_action_mismatch")
	}
	if u.Path == "own" {
		if u.Root.Run != i.Run || p.Revision != i.revision {
			return deny("authority_owner_mismatch")
		}
	} else {
		if !x.has(compositionLifecycleFact, u.GrantID) {
			return deny("authority_grant_missing")
		}
		grant, e := loadLifecycleResult(ctx, q, u.GrantID)
		if e != nil {
			return nil, e
		}
		if grant.Request.Operation != "DelegateAuthority" || !grant.Accepted || grant.Authority == nil {
			return deny("authority_grant_refused")
		}
		d, _, e := parseAuthorityClaim(grant.Request)
		if e != nil {
			return nil, e
		}
		if d.Root != u.Root || d.Recipient != i.Run || d.RecipientRevision != i.revision || !x.rootMember(i.Run, d.RecipientRoot) || d.ServiceOperationID != i.ID || i.Ordinal != 0 {
			return deny("authority_service_mismatch")
		}
		w.OriginalRequestID = d.OriginalRequestID
		w.ServiceOperationID = d.ServiceOperationID
		w.GrantDecisionSequence = grant.DecisionSequence
		if i.Sequence <= 1 {
			return deny("authority_request_not_selected")
		}
		var selected sql.NullString
		if e = q.QueryRowContext(ctx, "SELECT selected_observation FROM history WHERE subject=? AND generation=? AND sequence=?", i.Run.Subject, i.Run.Generation, i.Sequence-1).Scan(&selected); e != nil {
			return nil, e
		}
		if !selected.Valid {
			return deny("authority_request_not_selected")
		}
		var raw []byte
		var consumed sql.NullInt64
		var observation Observation
		var observedRun RunID
		if e = q.QueryRowContext(ctx, "SELECT subject,generation,descriptor,consumed_sequence FROM observations WHERE id=?", selected.String).Scan(&observedRun.Subject, &observedRun.Generation, &raw, &consumed); e != nil {
			return nil, e
		}
		if e = decode(raw, &observation); e != nil {
			return nil, e
		}
		if observedRun != i.Run || !sameJSON(raw, observation) || observation.ID != selected.String || observation.Run != i.Run || observation.Provenance != "ordinary-message" || observation.Source != d.Root.Run.Subject || observation.SourceGeneration != d.Root.Run.Generation || !consumed.Valid || consumed.Int64 != i.Sequence-1 {
			return deny("authority_request_not_selected")
		}
		ids, e := stringList(ctx, q, "SELECT id FROM admissions")
		if e != nil {
			return nil, e
		}
		admitted := false
		for _, id := range ids {
			desc, a, e := loadAdmissionKey(ctx, q, id)
			if e != nil {
				return nil, e
			}
			if a.Accepted && a.Bound != nil && *a.Bound == i.Run && desc.Kind == "message" && desc.IntentID == d.OriginalRequestID && a.ObservationID == selected.String && reflect.DeepEqual(observationFor(desc, i.Run), observation) {
				admitted = true
			}
		}
		if !admitted || !x.has(compositionAuthorizationFact, d.OriginalRequestID) {
			return deny("authority_request_not_admitted")
		}
		auth, e := loadAuthorization(ctx, q, d.OriginalRequestID)
		if e != nil {
			return nil, e
		}
		if !auth.Allowed {
			return deny("authority_request_not_admitted")
		}
		w.RequestObservationID = selected.String
		w.RequestConsumedSequence = consumed.Int64
	}
	w.Allowed = true
	w.Reason = "authorized"
	return w, nil
}
func retainedEffectAuthority(ctx context.Context, q queryer, i retainedIntent, actual *EffectAuthority) (*EffectAuthority, error) {
	enabled, err := requestAuthorityEnabled(ctx, q)
	if err != nil {
		return nil, err
	}
	if !enabled {
		if actual != nil || i.Authority != nil {
			return nil, ErrVersion
		}
		return nil, nil
	}
	order, err := authorityFactOrder(ctx, q, compositionReservationFact, i.ID)
	if err != nil {
		return nil, err
	}
	expected, err := evaluateEffectAuthority(ctx, q, i, order)
	if err != nil {
		return nil, err
	}
	if !reflect.DeepEqual(actual, expected) {
		return nil, corrupt("effect authority differs from original prefix")
	}
	return expected, nil
}
func validateRequestAuthorityProgram(p Program, enabled bool) error {
	for _, in := range p.Code {
		if in.Effect == nil {
			continue
		}
		e := in.Effect
		if e.Authority != nil {
			if !enabled {
				return ErrVersion
			}
			if e.Kind != "external" && e.Kind != "adapter" {
				return ErrVersion
			}
			if err := validateAuthorityUse(e.Authority); err != nil {
				return err
			}
		}
	}
	return nil
}

const authorityCurrentPrefix = math.MaxInt64

func validateRequestAuthorityIntent(ctx context.Context, q queryer, i Intent, revision int64) error {
	enabled, err := requestAuthorityEnabled(ctx, q)
	if err != nil {
		return err
	}
	if i.Authority != nil {
		if !enabled || (i.Kind != "external" && i.Kind != "adapter") {
			return ErrVersion
		}
		if err = validateAuthorityUse(i.Authority); err != nil {
			return err
		}
	}
	// Outside the feature these names are historically inert payload text.
	// The lifecycle loader retains its unknown-operation refusal there.
	if enabled && i.Kind == "lifecycle" && authorityOperation(i.Payload["operation"].Text) {
		if i.Payload["operation"].Kind != "string" || i.Payload["revision"].Kind != "int" || i.Payload["revision"].Number <= 0 || i.Target == "" || i.Generation <= 0 || i.Obligation == "" {
			return ErrCorrupt
		}
		_, _, err = parseAuthorityClaim(LifecycleRequest{Intent: i, SourceRevision: revision, Operation: i.Payload["operation"].Text, ExpectedRevision: i.Payload["revision"].Number})
		return err
	}
	return nil
}
