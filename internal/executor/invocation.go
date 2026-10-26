package executor

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
)

// Only the invocation profile creates these tables. Foundation stores and their
// retained evidence keep the original layout and validator behavior.
const invocationSchema = `
CREATE TABLE composition_publications(operation_id TEXT PRIMARY KEY REFERENCES composition_admission_decisions(operation_id), descriptor BLOB NOT NULL);
CREATE TABLE composition_outcome_deliveries(operation_id TEXT PRIMARY KEY REFERENCES composition_publications(operation_id), observation_id TEXT NOT NULL UNIQUE REFERENCES observations(id));
CREATE TABLE composition_admission_decisions(operation_id TEXT PRIMARY KEY REFERENCES intents(id), root_id TEXT NOT NULL REFERENCES composition_roots(id), descriptor BLOB NOT NULL);
CREATE TABLE composition_requests(invocation_id TEXT PRIMARY KEY REFERENCES composition_invocations(id), observation_id TEXT NOT NULL UNIQUE REFERENCES observations(id), descriptor BLOB NOT NULL);
CREATE TABLE composition_replies(operation_id TEXT PRIMARY KEY REFERENCES intents(id), invocation_id TEXT NOT NULL REFERENCES composition_invocations(id), descriptor BLOB NOT NULL);
CREATE TABLE composition_freezes(scope_id TEXT PRIMARY KEY, root_id TEXT NOT NULL REFERENCES composition_roots(id), descriptor BLOB NOT NULL);
`

type compositionMemberOrigin struct {
	Run                                     RunID
	InvocationID, RootID, Member, ProgramID string
	Initial                                 Projection
	Owner                                   string
}
type compositionReply struct {
	InvocationID, RootID string
	Intent               Intent
	Authorization        Authorization
	Accepted             bool
	Reason               string
}

// InvocationPreparation is not a committed outcome. Ready means the known
// outcome can be published; unfinished descendant/external joins stay pending.
type InvocationPreparation struct {
	Ready   bool
	Pending string
	Freeze  compositionFreeze
	Outcome compositionOutcome
}

func loadCompositionAdmission(ctx context.Context, q queryer, operation string) (compositionAdmission, error) {
	var a compositionAdmission
	var raw []byte
	err := q.QueryRowContext(ctx, "SELECT descriptor FROM composition_admission_decisions WHERE operation_id=?", operation).Scan(&raw)
	if err != nil {
		return a, err
	}
	if err = decode(raw, &a); err != nil {
		return a, err
	}
	if a.Invocation.OperationID != operation || !sameJSON(raw, a) {
		return a, ErrCorrupt
	}
	return a, nil
}
func loadCompositionBinding(ctx context.Context, q queryer, run RunID) (compositionBinding, string, error) {
	var b compositionBinding
	var raw []byte
	var member string
	err := q.QueryRowContext(ctx,
		"SELECT b.descriptor,m.member FROM composition_members m JOIN composition_bindings b ON b.invocation_id=m.invocation_id WHERE m.subject=? AND m.generation=?",
		run.Subject,
		run.Generation).Scan(&raw,
		&member)
	if err != nil {
		return b, member, err
	}
	err = decode(raw, &b)
	if err == nil && !sameJSON(raw, b) {
		err = ErrCorrupt
	}
	return b, member, err
}
func loadCompositionTemplate(ctx context.Context, q queryer, id string) (compositionTemplate, error) {
	var t compositionTemplate
	var raw []byte
	err := q.QueryRowContext(ctx, "SELECT descriptor FROM composition_templates WHERE id=?", id).Scan(&raw)
	if err != nil {
		return t, err
	}
	err = decode(raw, &t)
	if err == nil && (t.ID != id || !sameJSON(raw, t)) {
		err = ErrCorrupt
	}
	return t, err
}
func boundCompositionProgram(ctx context.Context, q queryer, p Program, run RunID, l Limits) (Program, error) {
	n, err := schemaNumber(ctx, q)
	if err != nil {
		return p, err
	}
	if n != compositionSchemaVersion {
		return p, nil
	}
	b, member, err := loadCompositionBinding(ctx, q, run)
	if errors.Is(err, sql.ErrNoRows) {
		return p, nil
	}
	if err != nil {
		return p, err
	}
	t, err := loadCompositionTemplate(ctx, q, b.TemplateID)
	if err != nil {
		return p, err
	}
	env, err := retainCompositionBinding(b.Namespace, b.RootID, t, b)
	if err != nil {
		return p, err
	}
	return resolveCompositionProgram(p, env, member)
}
func compositionReplyBinding(ctx context.Context, q queryer, intent Intent) (compositionBinding, bool, error) {
	b, member, err := loadCompositionBinding(ctx, q, intent.Run)
	if errors.Is(err, sql.ErrNoRows) {
		return b, false, nil
	}
	if err != nil {
		return b, false, err
	}
	t, err := loadCompositionTemplate(ctx, q, b.TemplateID)
	if err != nil {
		return b, false, err
	}
	p, err := loadProgram(ctx, q, t.Members[t.EntryMember].ProgramID, DefaultLimits())
	if err != nil {
		return b, false, err
	}
	port := p.Author.Ports[t.EntryPort]
	return b, b.Protocol == compositionProtocolVersion && intent.Kind == "message" && member == t.EntryMember && intent.Target == b.Caller.Subject && intent.Generation == b.Caller.Generation && intent.Obligation == port.Obligation, nil
}
func compositionRootFor(ctx context.Context, q queryer, run RunID) (compositionRoot, error) {
	var r compositionRoot
	id, err := compositionMembershipRoot(ctx, q, run)
	if err != nil {
		return r, err
	}
	if id == "" {
		return r, fmt.Errorf("invocation source has no resource root")
	}
	var raw []byte
	err = q.QueryRowContext(ctx, "SELECT descriptor FROM composition_roots WHERE id=?", id).Scan(&raw)
	if err != nil {
		return r, err
	}
	err = decode(raw, &r)
	if err == nil && (r.ID != id || !sameJSON(raw, r)) {
		err = ErrCorrupt
	}
	return r, err
}

// compositionScopeClosed is the final admission/reservation boundary shared with
// Task2c. It does not rewrite historical source authorization.
func compositionScopeClosed(ctx context.Context, q queryer, run RunID) (bool, error) {
	root, err := compositionRootFor(ctx, q, run)
	if err != nil {
		return false, err
	}
	r, err := loadRun(ctx, q, root.Run)
	if err != nil {
		return false, err
	}
	if r.Projection.Status != "open" {
		return true, nil
	}
	var n int
	if err = q.QueryRowContext(ctx,
		"SELECT count(*) FROM composition_freezes WHERE scope_id=? AND json_extract(descriptor,'$.Status')!='succeeded'",
		"root:"+root.ID).Scan(&n); err != nil {
		return false, err
	}
	if n != 0 {
		return true, nil
	}
	b, _, err := loadCompositionBinding(ctx, q, run)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	seen := map[string]bool{}
	for b.InvocationID != "" {
		if seen[b.InvocationID] {
			return false, ErrCorrupt
		}
		seen[b.InvocationID] = true
		if err = q.QueryRowContext(ctx,
			"SELECT count(*) FROM composition_freezes WHERE scope_id=? AND json_extract(descriptor,'$.Status')!='succeeded'",
			b.InvocationID).Scan(&n); err != nil {
			return false, err
		}
		if n != 0 {
			return true, nil
		}
		b, _, err = loadCompositionBinding(ctx, q, b.Caller)
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
	}
	return false, nil
}

// AdmitInvocation authorizes its immutable committed source separately, then
// atomically retains the original admission, every child, binding and root debit.
func (s *Store) AdmitInvocation(ctx context.Context, operation string) (compositionAdmission, error) {
	if !s.invocationEnabled() {
		return compositionAdmission{}, ErrVersion
	}
	auth, err := s.AuthorizeIntent(ctx, operation)
	if err != nil {
		return compositionAdmission{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return compositionAdmission{}, sql.ErrConnDone
	}
	tx, err := beginOwnedTx(ctx, s.conn)
	if err != nil {
		return compositionAdmission{}, err
	}
	defer tx.Rollback()
	old, err := loadCompositionAdmission(ctx, tx, operation)
	if err == nil {
		return old, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return old, err
	}
	intent, err := loadIntent(ctx, tx, operation)
	if err != nil {
		return old, err
	}
	if intent.Kind != "invocation" {
		return old, fmt.Errorf("not an invocation intent")
	}
	root, err := compositionRootFor(ctx, tx, intent.Run)
	if err != nil {
		return old, err
	}
	invocation := compositionInvocation{ID: compositionInvocationID(s.namespace,
		operation),
		OperationID:    operation,
		TemplateID:     intent.Target,
		RootID:         root.ID,
		Caller:         intent.Run,
		SourceRevision: intent.revision,
		SourceSequence: intent.Sequence,
		Depth:          1,
		Inputs:         intent.Payload}
	parent, _, err := loadCompositionBinding(ctx, tx, intent.Run)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return old, err
	}
	if err == nil {
		invocation.ParentInvocationID = parent.InvocationID
		invocation.Depth = parent.Depth + 1
	}
	a := compositionAdmission{Invocation: invocation, Reason: "accepted"}
	source, err := loadRun(ctx, tx, intent.Run)
	if err != nil {
		return a, err
	}
	program, err := loadProgram(ctx, tx, source.ProgramID, s.limits)
	if err != nil {
		return a, err
	}
	var declaration InvocationDeclaration
	declared := false
	if program.Author != nil {
		declaration, declared = program.Author.Invocations[intent.Obligation]
	}
	closed, err := compositionScopeClosed(ctx, tx, intent.Run)
	if err != nil {
		return a, err
	}
	t, templateErr := loadCompositionTemplate(ctx, tx, intent.Target)
	if templateErr != nil && !errors.Is(templateErr, sql.ErrNoRows) {
		return a, templateErr
	}
	switch {
	case !auth.Allowed:
		a.Reason = "source_denied:" + auth.Reason
	case source.Projection.Status != "open" || source.Revision != intent.revision:
		a.Reason = "source_closed"
	case closed:
		a.Reason = "scope_closed"
	case !declared || declaration.TemplateID != intent.Target || intent.Generation != 1:
		a.Reason = "declaration_denied"
	case templateErr != nil:
		a.Reason = "template_missing"
	case !slices.Contains(root.Templates, intent.Target):
		a.Reason = "template_denied"
	case invocation.Depth > root.Limits.MaxDepth:
		a.Reason = "depth_limit"
	case declaration.Input.Record.Validate(intent.Payload) != nil:
		a.Reason = "schema_invalid"
	}
	if a.Reason == "accepted" && parent.InvocationID != "" {
		pt, e := loadCompositionTemplate(ctx, tx, parent.TemplateID)
		if e != nil {
			return a, e
		}
		if !slices.Contains(pt.Templates, t.ID) {
			a.Reason = "template_denied"
		}
	}
	if a.Reason == "accepted" {
		for _, capability := range t.Capabilities {
			if !slices.Contains(root.Capabilities, capability) {
				a.Reason = "capability_denied"
			}
		}
	}
	var invocations, runs int64
	if err = tx.QueryRowContext(ctx,
		"SELECT count(*),coalesce(sum(json_extract(descriptor,'$.CreatedRuns')),0) FROM composition_charges WHERE root_id=? AND kind='invocation'",
		root.ID).Scan(&invocations,
		&runs); err != nil {
		return a, err
	}
	if a.Reason == "accepted" {
		switch {
		case invocations >= root.Limits.MaxInvocations:
			a.Reason = "invocation_limit"
		case int64(len(t.Members)) > root.Limits.MaxCreatedRuns-runs:
			a.Reason = "created_run_limit"
		}
	}
	a.Accepted = a.Reason == "accepted"
	var binding compositionBinding
	if a.Accepted {
		binding = compositionBinding{Protocol: compositionProtocolVersion,
			Version:      compositionBindingVersion,
			Namespace:    s.namespace,
			InvocationID: invocation.ID,
			OperationID:  operation,
			TemplateID:   t.ID,
			RootID:       root.ID,
			Caller:       intent.Run,
			Depth:        invocation.Depth,
			Members:      map[string]RunID{},
			Programs:     map[string]string{}}
		for name, m := range t.Members {
			binding.Members[name] = compositionMemberID(invocation.ID, name)
			binding.Programs[name] = m.ProgramID
		}
		env, e := retainCompositionBinding(s.namespace, root.ID, t, binding)
		if e != nil {
			return a, e
		}
		if e = decode([]byte(env.binding), &binding); e != nil {
			return a, e
		}
		a.BindingID = binding.ID
		for _, name := range sortedKeys(t.Members) {
			a.Members = append(a.Members, binding.Members[name])
		}
	}
	raw, err := json.Marshal(a)
	if err != nil {
		return a, err
	}
	if err = checkSize(a, s.limits.MaxOutputBytes); err != nil {
		return a, err
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO composition_admission_decisions VALUES(?,?,?)", operation, root.ID, raw); err != nil {
		return a, err
	}
	if a.Accepted {
		invraw, err := json.Marshal(invocation)
		if err != nil {
			return a, err
		}
		braw, err := json.Marshal(binding)
		if err != nil {
			return a, err
		}
		if _, err = tx.ExecContext(ctx,
			"INSERT INTO composition_invocations VALUES(?,?,?,?,?,?)",
			invocation.ID,
			operation,
			t.ID,
			root.ID,
			invraw,
			raw); err != nil {
			return a, err
		}
		if _, err = tx.ExecContext(ctx, "INSERT INTO composition_bindings VALUES(?,?,?)", binding.ID, invocation.ID, braw); err != nil {
			return a, err
		}
		for _, name := range sortedKeys(t.Members) {
			m := t.Members[name]
			run := binding.Members[name]
			owner := "invocation:" + invocation.ID
			initial, err := json.Marshal(m.Initial)
			if err != nil {
				return a, err
			}
			if _, err = tx.ExecContext(ctx, "INSERT INTO subjects VALUES(?,?,?,1,0,0)", run.Subject, 1, owner); err != nil {
				return a, err
			}
			if _, err = tx.ExecContext(ctx, "INSERT INTO runs VALUES(?,?,?,?,1,0,?)", run.Subject, 1, m.ProgramID, initial, initial); err != nil {
				return a, err
			}
			origin, err := json.Marshal(compositionMemberOrigin{Run: run,
				InvocationID: invocation.ID,
				RootID:       root.ID,
				Member:       name,
				ProgramID:    m.ProgramID,
				Initial:      m.Initial,
				Owner:        owner})
			if err != nil {
				return a, err
			}
			if _, err = tx.ExecContext(ctx,
				"INSERT INTO composition_members VALUES(?,?,?,?,?,?)",
				run.Subject,
				1,
				invocation.ID,
				root.ID,
				name,
				origin); err != nil {
				return a, err
			}
			p, e := loadProgram(ctx, tx, m.ProgramID, s.limits)
			if e != nil {
				return a, e
			}
			p, e = boundCompositionProgram(ctx, tx, p, run, s.limits)
			if e != nil {
				return a, e
			}
			for _, port := range p.Author.Ports {
				if port.SendUsed && port.Peer != intent.Run.Subject {
					if _, err = tx.ExecContext(ctx,
						"INSERT OR IGNORE INTO permissions VALUES('message',?,?,'message',1,?)",
						run.Subject,
						port.Peer,
						port.Obligation); err != nil {
						return a, err
					}
				}
			}
			for effect, effectDecl := range p.Author.Effects {
				if _, err = tx.ExecContext(ctx,
					"INSERT INTO permissions VALUES('adapter',?,?,?,1,?)",
					run.Subject,
					effectDecl.Capability,
					effect,
					effect); err != nil {
					return a, err
				}
			}
		}
		entry := binding.Members[t.EntryMember]
		p, e := loadProgram(ctx, tx, t.Members[t.EntryMember].ProgramID, s.limits)
		if e != nil {
			return a, e
		}
		port := p.Author.Ports[t.EntryPort]
		if err = port.Receive.Record.Validate(intent.Payload); err != nil {
			return a, err
		}
		request := Observation{ID: compositionIdentity("alan/invocation-request/v1",
			invocation.ID),
			Run:              entry,
			Kind:             "message",
			Provenance:       "runtime-invocation-request",
			Source:           intent.Run.Subject,
			SourceGeneration: intent.Run.Generation,
			Obligation:       port.Obligation,
			Operation:        operation,
			Payload:          intent.Payload}
		if err = insertObservation(ctx, tx, request, s.limits); err != nil {
			return a, err
		}
		requestRaw, err := json.Marshal(request)
		if err != nil {
			return a, err
		}
		if _, err = tx.ExecContext(ctx, "INSERT INTO composition_requests VALUES(?,?,?)", invocation.ID, request.ID, requestRaw); err != nil {
			return a, err
		}
	}
	fact, err := compositionProtocolEvidence(ctx, tx, compositionAdmissionFact, operation)
	if err != nil {
		return a, err
	}
	order, err := appendCompositionFact(ctx, tx, fact)
	if err != nil {
		return a, err
	}
	if a.Accepted {
		charge := compositionCharge{RootID: root.ID,
			OperationID:   operation,
			Kind:          "invocation",
			Invocations:   1,
			CreatedRuns:   int64(len(t.Members)),
			DecisionOrder: order}
		raw, err := json.Marshal(charge)
		if err != nil {
			return a, err
		}
		if _, err = tx.ExecContext(ctx,
			"INSERT INTO composition_charges VALUES(?,?,?,?,?)",
			operation,
			"invocation",
			root.ID,
			order,
			raw); err != nil {
			return a, err
		}
	}
	return a, s.commitOwnedTx(ctx, tx)
}
func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}

// Named public views of retained admission and outcome evidence.
type InvocationAdmission = compositionAdmission
type InvocationDescriptor = compositionInvocation
type InvocationOutcome = compositionOutcome
type InvocationFreeze = compositionFreeze
