package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

type compositionDeployment struct {
	Subjects []Subject
	Runs     []InitialRun
}

// Reconstruction is read-only. Expected rows are derived from immutable source
// artifacts and ordered decisions; no missing record is repaired or synthesized.
func validateCompositionProtocol(ctx context.Context, q queryer, l Limits, c compositionProvision) error {
	tables := map[string][][]any{}
	columns := map[string]string{
		"composition_invocations":        "id,operation_id,template_id,root_id,descriptor,admission",
		"composition_bindings":           "id,invocation_id,descriptor",
		"composition_members":            "subject,generation,invocation_id,root_id,member,descriptor",
		"composition_requests":           "invocation_id,observation_id,descriptor",
		"composition_results":            "invocation_id,decision_order,descriptor",
		"composition_charges":            "operation_id,kind,root_id,decision_order,descriptor",
		"composition_outcome_deliveries": "operation_id,observation_id",
		"permissions":                    "kind,source,target,operation,revision,obligation",
	}
	for table := range columns {
		tables[table] = [][]any{}
	}
	var raw []byte
	var deployment compositionDeployment
	if err := q.QueryRowContext(ctx, "SELECT descriptor FROM composition_deployment WHERE singleton=1").Scan(&raw); err != nil {
		return err
	}
	if err := decode(raw, &deployment); err != nil {
		return err
	}
	if !sameJSON(raw, deployment) {
		return corrupt("deployment encoding")
	}
	subjects := map[string]Subject{}
	for _, subject := range deployment.Subjects {
		if _, ok := subjects[subject.ID]; ok {
			return corrupt("duplicate deployment subject")
		}
		subjects[subject.ID] = subject
	}
	origins := [][]any{}
	for _, run := range deployment.Runs {
		subject, ok := subjects[run.ID.Subject]
		if !ok || subject.Generation != run.ID.Generation {
			return corrupt("deployment run subject")
		}
		delete(subjects, run.ID.Subject)
		origin := rootOrigin{Subject: subject, ProgramID: run.ProgramID, Initial: run.Projection}
		origins = append(origins, []any{run.ID.Subject, run.ID.Generation, "root", nil, nil, compositionJSON(origin)})
	}
	if len(subjects) != 0 {
		return corrupt("deployment subject without run")
	}
	if err := compositionExactRows(ctx, q, "run_origins WHERE kind='root'", "subject,generation,kind,operation_id,predecessor_generation,descriptor", origins); err != nil {
		return err
	}
	var permissions []Permission
	if err := q.QueryRowContext(ctx, "SELECT descriptor FROM composition_permissions WHERE singleton=1").Scan(&raw); err != nil {
		return err
	}
	if err := decode(raw, &permissions); err != nil {
		return err
	}
	if !sameJSON(raw, permissions) {
		return corrupt("deployment permission encoding")
	}
	addPermission := func(p Permission) {
		tables["permissions"] = append(tables["permissions"], []any{p.Kind, p.Source, p.Target, p.Operation, p.Revision, p.Obligation})
	}
	for _, p := range permissions {
		addPermission(p)
	}
	namespace, err := readNamespace(ctx, q)
	if err != nil {
		return err
	}
	admissions := map[string]compositionAdmission{}
	ids, err := stringList(ctx, q, "SELECT operation_id FROM composition_admission_decisions ORDER BY operation_id")
	if err != nil {
		return err
	}
	observations, err := readObservations(ctx, q, l)
	if err != nil {
		return err
	}
	explained := map[string]bool{}
	for _, id := range ids {
		a, err := loadCompositionAdmission(ctx, q, id)
		if err != nil {
			return err
		}
		admissions[id] = a
		var rootID string
		if err = q.QueryRowContext(ctx, "SELECT root_id FROM composition_admission_decisions WHERE operation_id=?", id).Scan(&rootID); err != nil {
			return err
		}
		if rootID != a.Invocation.RootID {
			return corrupt("admission relational root")
		}
		if !a.Accepted {
			if a.BindingID != "" || len(a.Members) != 0 {
				return corrupt("rejected invocation created members")
			}
			continue
		}
		i := a.Invocation
		t, err := loadCompositionTemplate(ctx, q, i.TemplateID)
		if err != nil {
			return err
		}
		b := compositionBinding{Protocol: compositionProtocolVersion, Version: compositionBindingVersion, Namespace: namespace, InvocationID: i.ID, OperationID: id, TemplateID: t.ID, RootID: i.RootID, Caller: i.Caller, Depth: i.Depth, Members: map[string]RunID{}, Programs: map[string]string{}}
		members := []RunID{}
		for _, name := range sortedKeys(t.Members) {
			b.Members[name] = compositionMemberID(i.ID, name)
			b.Programs[name] = t.Members[name].ProgramID
			members = append(members, b.Members[name])
		}
		env, err := retainCompositionBinding(namespace, i.RootID, t, b)
		if err != nil {
			return err
		}
		if err = decode([]byte(env.binding), &b); err != nil {
			return err
		}
		if b.ID != a.BindingID || !reflect.DeepEqual(members, a.Members) {
			return corrupt("admission binding/members")
		}
		tables["composition_invocations"] = append(tables["composition_invocations"], []any{i.ID, id, t.ID, i.RootID, compositionJSON(i), compositionJSON(a)})
		tables["composition_bindings"] = append(tables["composition_bindings"], []any{b.ID, i.ID, compositionJSON(b)})
		for _, name := range sortedKeys(t.Members) {
			m := t.Members[name]
			run := b.Members[name]
			owner := "invocation:" + i.ID
			origin := compositionMemberOrigin{Run: run, InvocationID: i.ID, RootID: i.RootID, Member: name, ProgramID: m.ProgramID, Initial: m.Initial, Owner: owner}
			tables["composition_members"] = append(tables["composition_members"], []any{run.Subject, 1, i.ID, i.RootID, name, compositionJSON(origin)})
			state, err := loadRun(ctx, q, run)
			if err != nil {
				return err
			}
			var initial []byte
			if err = q.QueryRowContext(ctx, "SELECT initial FROM runs WHERE subject=? AND generation=?", run.Subject, 1).Scan(&initial); err != nil {
				return err
			}
			subject, err := loadSubject(ctx, q, run.Subject)
			if err != nil {
				return err
			}
			var origins int
			if err = q.QueryRowContext(ctx, "SELECT count(*) FROM run_origins WHERE subject=?", run.Subject).Scan(&origins); err != nil {
				return err
			}
			if origins != 0 || state.ProgramID != m.ProgramID || !sameJSON(initial, m.Initial) || subject.Owner != owner || subject.Generation != 1 || subject.Revision != 1 || subject.RetryCancelled {
				return corrupt("dynamic immutable origin")
			}
			program, err := loadProgram(ctx, q, m.ProgramID, l)
			if err != nil {
				return err
			}
			program, err = resolveCompositionProgram(program, env, name)
			if err != nil {
				return err
			}
			for _, port := range program.Author.Ports {
				if port.SendUsed && port.Peer != i.Caller.Subject {
					addPermission(Permission{Kind: "message", Source: run.Subject, Target: port.Peer, Operation: "message", Revision: 1, Obligation: port.Obligation})
				}
			}
			for effect, d := range program.Author.Effects {
				addPermission(Permission{Kind: "adapter", Source: run.Subject, Target: d.Capability, Operation: effect, Revision: 1, Obligation: effect})
			}
		}
		program, err := loadProgram(ctx, q, t.Members[t.EntryMember].ProgramID, l)
		if err != nil {
			return err
		}
		request := Observation{ID: compositionIdentity("alan/invocation-request/v1", i.ID), Run: b.Members[t.EntryMember], Kind: "message", Provenance: "runtime-invocation-request", Source: i.Caller.Subject, SourceGeneration: i.Caller.Generation, Obligation: program.Author.Ports[t.EntryPort].Obligation, Operation: id, Payload: i.Inputs}
		tables["composition_requests"] = append(tables["composition_requests"], []any{i.ID, request.ID, compositionJSON(request)})
		if !reflect.DeepEqual(observations[request.ID].Observation, request) {
			return corrupt("invocation request observation")
		}
		explained[request.ID] = true
	}
	facts, err := readCompositionFacts(ctx, q)
	if err != nil {
		return err
	}
	orders := map[string]int64{}
	for _, f := range facts {
		orders[f.ID] = f.Order
	}
	for _, a := range admissions {
		if a.Accepted {
			order := orders[compositionIdentity("alan/fact/admission/v1", a.Invocation.OperationID)]
			charge := compositionCharge{RootID: a.Invocation.RootID, OperationID: a.Invocation.OperationID, Kind: "invocation", Invocations: 1, CreatedRuns: int64(len(a.Members)), DecisionOrder: order}
			tables["composition_charges"] = append(tables["composition_charges"], []any{charge.OperationID, charge.Kind, charge.RootID, order, compositionJSON(charge)})
		}
	}
	ids, err = stringList(ctx, q, "SELECT operation_id FROM composition_reservations")
	if err != nil {
		return err
	}
	for _, id := range ids {
		var r compositionReservation
		var root string
		if err = q.QueryRowContext(ctx, "SELECT root_id,descriptor FROM composition_reservations WHERE operation_id=?", id).Scan(&root, &raw); err != nil {
			return err
		}
		if err = decode(raw, &r); err != nil {
			return err
		}
		if !sameJSON(raw, r) || root != r.RootID || r.OperationID != id {
			return corrupt("reservation relational binding")
		}
		var digest string
		if r.Kind == "adapter" {
			a, e := loadAdapterAttempt(ctx, q, id)
			if e != nil {
				return e
			}
			if a.Allowed != r.Allowed || a.Reason != r.Reason {
				return corrupt("adapter reservation decision")
			}
			digest = compositionIdentity("alan/attempt-row/v1", a)
		} else if r.Kind == "external" {
			a, e := loadExternalAttempt(ctx, q, id)
			if e != nil {
				return e
			}
			if a.Allowed != r.Allowed || a.Reason != r.Reason {
				return corrupt("external reservation decision")
			}
			digest = compositionIdentity("alan/attempt-row/v1", a)
		} else {
			return corrupt("reservation kind")
		}
		if r.AttemptDigest != digest {
			return corrupt("reservation attempt digest")
		}
		if r.Allowed {
			order := orders[compositionIdentity("alan/fact/reservation/v1", id)]
			charge := compositionCharge{RootID: r.RootID, OperationID: id, Kind: "effect", Effects: 1, DecisionOrder: order}
			tables["composition_charges"] = append(tables["composition_charges"], []any{id, "effect", r.RootID, order, compositionJSON(charge)})
		}
	}
	ids, err = stringList(ctx, q, "SELECT operation_id FROM composition_publications")
	if err != nil {
		return err
	}
	for _, id := range ids {
		var o compositionOutcome
		if err = q.QueryRowContext(ctx, "SELECT descriptor FROM composition_publications WHERE operation_id=?", id).Scan(&raw); err != nil {
			return err
		}
		if err = decode(raw, &o); err != nil {
			return err
		}
		a, ok := admissions[id]
		if !ok || !sameJSON(raw, o) {
			return corrupt("outcome admission")
		}
		if a.Accepted {
			tables["composition_results"] = append(tables["composition_results"], []any{a.Invocation.ID, o.DecisionOrder, string(raw)})
		}
		i, err := loadIntent(ctx, q, id)
		if err != nil {
			return err
		}
		observation := invocationObservation(o, i.Intent)
		if !reflect.DeepEqual(observations[observation.ID].Observation, observation) {
			return corrupt("outcome observation")
		}
		explained[observation.ID] = true
		tables["composition_outcome_deliveries"] = append(tables["composition_outcome_deliveries"], []any{id, observation.ID})
	}
	for id, o := range observations {
		if (o.Provenance == "runtime-invocation" || o.Provenance == "runtime-invocation-request") && !explained[id] {
			return corrupt("unexplained invocation observation")
		}
	}
	for table, cols := range columns {
		if err = compositionExactRows(ctx, q, table, cols, tables[table]); err != nil {
			return err
		}
	}
	return validateCompositionPrefix(ctx, q, l, c, admissions, facts)
}
func compositionJSON(v any) string { raw, _ := json.Marshal(v); return string(raw) }
func compositionExactRows(ctx context.Context, q queryer, table, columns string, want [][]any) error {
	key := func(row []any) string {
		values := make([]string, len(row))
		for i, v := range row {
			if raw, ok := v.([]byte); ok {
				values[i] = string(raw)
			} else {
				values[i] = fmt.Sprint(v)
			}
		}
		return compositionJSON(values)
	}
	expected := map[string]bool{}
	for _, row := range want {
		expected[key(row)] = true
	}
	rows, err := q.QueryContext(ctx, "SELECT "+columns+" FROM "+table)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		row := make([]any, len(strings.Split(columns, ",")))
		dest := make([]any, len(row))
		for i := range row {
			dest[i] = &row[i]
		}
		if err = rows.Scan(dest...); err != nil {
			return err
		}
		k := key(row)
		if !expected[k] {
			return corrupt("unexpected %s row", table)
		}
		delete(expected, k)
	}
	if err = rows.Err(); err != nil {
		return err
	}
	if len(expected) != 0 {
		return corrupt("missing %s row", table)
	}
	return nil
}
func invocationObservation(outcome compositionOutcome, intent Intent) Observation {
	payload := map[string]Value{"status": Text(outcome.Status), "reason": Text(outcome.Reason), "response_present": {Kind: "bool", Flag: outcome.ResponsePresent}}
	if outcome.ResponsePresent {
		for field, value := range outcome.Response {
			payload["response."+field] = value.Clone()
		}
	}
	return Observation{ID: compositionIdentity("alan/invocation-result-observation/v1", outcome.ID), Run: outcome.Caller, Kind: "invocation", Provenance: "runtime-invocation", Source: outcome.TemplateID, SourceGeneration: 1, Obligation: intent.Obligation, Operation: intent.ID, Payload: payload}
}

type compositionPrefix struct {
	states       map[RunID]Projection
	sequences    map[RunID]int64
	retired      map[RunID]bool
	active       map[RunID]bool
	public       map[string]bool
	intents      map[string]Intent
	authorized   map[string]bool
	admitted     map[string]compositionAdmission
	replies      map[string]compositionReply
	freezes      map[string]compositionFreeze
	closures     map[string]compositionClosure
	outcomes     map[string]compositionOutcome
	reservations map[string]bool
	known        map[string]bool
	charges      map[string]compositionCharge
	scopes       map[RunID]compositionScope
	roots        map[string]compositionRoot
	bindings     map[RunID]compositionBinding
}

func validateCompositionPrefix(ctx context.Context, q queryer, l Limits, c compositionProvision, admissions map[string]compositionAdmission, facts []compositionFact) error {
	x := compositionPrefix{public: map[string]bool{}, states: map[RunID]Projection{}, sequences: map[RunID]int64{}, retired: map[RunID]bool{}, active: map[RunID]bool{}, intents: map[string]Intent{}, authorized: map[string]bool{}, admitted: map[string]compositionAdmission{}, replies: map[string]compositionReply{}, freezes: map[string]compositionFreeze{}, closures: map[string]compositionClosure{}, outcomes: map[string]compositionOutcome{}, reservations: map[string]bool{}, known: map[string]bool{}, charges: map[string]compositionCharge{}, scopes: map[RunID]compositionScope{}, roots: map[string]compositionRoot{}, bindings: map[RunID]compositionBinding{}}
	for _, r := range c.Roots {
		x.roots[r.ID] = r
		for _, m := range r.Members {
			root, _, err := loadRoot(ctx, q, m.Subject)
			if err != nil {
				return err
			}
			x.states[m] = root.Initial
			x.active[m] = true
			x.scopes[m] = compositionScope{ID: "root:" + r.ID, RootID: r.ID, Members: r.Members}
		}
	}
	expected := map[string]compositionFact{}
	runs, err := readRunIDs(ctx, q)
	if err != nil {
		return err
	}
	for _, run := range runs {
		history, err := readHistory(ctx, q, run)
		if err != nil {
			return err
		}
		for _, h := range history {
			f, err := compositionExecutionEvidence(ctx, q, run, h.seq)
			if err != nil {
				return err
			}
			if f.ID != "" {
				expected[f.ID] = f
			}
		}
	}
	ids, err := stringList(ctx, q, "SELECT id FROM lifecycle_results ORDER BY decision_sequence")
	if err != nil {
		return err
	}
	for _, id := range ids {
		r, e := loadLifecycleResult(ctx, q, id)
		if e != nil {
			return e
		}
		f, e := compositionLifecycleEvidence(ctx, q, r)
		if e != nil {
			return e
		}
		if f.ID != "" {
			expected[f.ID] = f
		}
	}
	for _, entry := range []struct {
		kind  compositionFactKind
		query string
	}{
		{compositionAuthorizationFact, "SELECT intent_id FROM attempts"}, {compositionExternalResultFact, "SELECT intent_id FROM adapter_results UNION SELECT intent_id FROM external_receipts"},
		{compositionAdmissionFact, "SELECT operation_id FROM composition_admission_decisions"}, {compositionReservationFact, "SELECT operation_id FROM composition_reservations"}, {compositionReplyFact, "SELECT operation_id FROM composition_replies"}, {compositionFreezeFact, "SELECT scope_id FROM composition_freezes"}, {compositionOutcomeFact, "SELECT operation_id FROM composition_publications UNION SELECT scope_id FROM composition_root_publications"},
	} {
		ids, e := stringList(ctx, q, entry.query)
		if e != nil {
			return e
		}
		for _, id := range ids {
			f, e := compositionProtocolEvidence(ctx, q, entry.kind, id)
			if e != nil {
				return e
			}
			if f.ID != "" {
				expected[f.ID] = f
			}
		}
	}
	if len(expected) != len(facts) {
		return corrupt("missing/invented protocol fact")
	}
	seen := map[string]bool{}
	lastLifecycle := int64(0)
	expectedClosures := [][]any{}
	expectedFreezes := [][]any{}
	for index, f := range facts {
		want, ok := expected[f.ID]
		want.Order = f.Order
		if !ok || f.Order != int64(index+1) || !reflect.DeepEqual(want, f) || seen[f.ID] {
			return corrupt("protocol fact identity/order/source")
		}
		switch f.Kind {
		case compositionExecutionFact:
			if !x.active[f.Run] || x.states[f.Run].Status != "open" || x.retired[f.Run] || f.Sequence != x.sequences[f.Run]+1 {
				return corrupt("execution prefix authority")
			}
			if err = x.validateConsumption(ctx, q, f, seen); err != nil {
				return err
			}
			var successor, raw []byte
			if err = q.QueryRowContext(ctx, "SELECT successor,intents FROM history WHERE subject=? AND generation=? AND sequence=?", f.Run.Subject, f.Run.Generation, f.Sequence).Scan(&successor, &raw); err != nil {
				return err
			}
			var p Projection
			var intents []Intent
			if err = decode(successor, &p); err != nil {
				return err
			}
			if err = decode(raw, &intents); err != nil {
				return err
			}
			x.states[f.Run] = p
			x.sequences[f.Run] = f.Sequence
			for _, i := range intents {
				x.intents[i.ID] = i
				if i.Kind == "message" {
					_, reply, e := compositionReplyBinding(ctx, q, i)
					if e != nil {
						return e
					}
					x.public[i.ID] = reply
				}
			}
			if freeze, yes := x.freeze(f.Run, f.ID, f.Order+1); yes {
				x.freezes[freeze.InvocationID] = freeze
				expectedFreezes = append(expectedFreezes, []any{freeze.InvocationID, x.scopes[f.Run].RootID, compositionJSON(freeze)})
			}
		case compositionAuthorizationFact:
			a, e := loadAuthorization(ctx, q, f.OperationID)
			if e != nil {
				return e
			}
			i, ok := x.intents[f.OperationID]
			if !ok || !x.active[i.Run] || a.Source.Sequence != x.sequences[i.Run] {
				return corrupt("authorization before source prefix")
			}
			_, reply, e := compositionReplyBinding(ctx, q, i)
			if e != nil {
				return e
			}
			reason := "authorized"
			allowed := true
			source, e := loadIntent(ctx, q, i.ID)
			if e != nil {
				return e
			}
			switch {
			case a.Source.Revision != source.revision:
				reason = "source_revision_changed"
				allowed = false
			case x.retired[i.Run] && !reply:
				reason = "source_retired"
				allowed = false
			case x.states[i.Run].Status != "open" && !reply:
				reason = "source_closed"
				allowed = false
			}
			if a.Allowed != allowed || a.Reason != reason {
				return corrupt("authorization prefix decision")
			}
			x.authorized[i.ID] = true
		case compositionAdmissionFact:
			a := admissions[f.OperationID]
			if err = x.validateAdmission(ctx, q, l, a); err != nil {
				return err
			}
			x.admitted[f.OperationID] = a
			if a.Accepted {
				b, _, e := loadCompositionBinding(ctx, q, a.Members[0])
				if e != nil {
					return e
				}
				t, e := loadCompositionTemplate(ctx, q, a.Invocation.TemplateID)
				if e != nil {
					return e
				}
				members := []RunID{}
				for _, name := range sortedKeys(b.Members) {
					members = append(members, b.Members[name])
				}
				for name, m := range b.Members {
					x.active[m] = true
					x.states[m] = t.Members[name].Initial
					x.bindings[m] = b
					x.scopes[m] = compositionScope{ID: b.InvocationID, RootID: b.RootID, Members: members}
				}
				spent := x.charges[a.Invocation.RootID]
				spent.Invocations++
				spent.CreatedRuns += int64(len(a.Members))
				x.charges[a.Invocation.RootID] = spent
			}
		case compositionReservationFact:
			var r compositionReservation
			var raw []byte
			if err = q.QueryRowContext(ctx, "SELECT descriptor FROM composition_reservations WHERE operation_id=?", f.OperationID).Scan(&raw); err != nil {
				return err
			}
			if err = decode(raw, &r); err != nil {
				return err
			}
			i, ok := x.intents[f.OperationID]
			if !ok || !x.authorized[i.ID] {
				return corrupt("reservation before source authorization")
			}
			auth, e := loadAuthorization(ctx, q, i.ID)
			if e != nil {
				return e
			}
			allowed, reason := auth.Allowed, auth.Reason
			retained, e := loadIntent(ctx, q, i.ID)
			if e != nil {
				return e
			}
			grant := false
			if i.Kind == "adapter" {
				grant, e = adapterGrant(ctx, q, retained)
			} else {
				var n int
				e = q.QueryRowContext(ctx, "SELECT count(*) FROM permissions WHERE kind='external' AND source=? AND target=? AND operation='apply' AND revision=?", i.Run.Subject, i.Target, retained.revision).Scan(&n)
				grant = n > 0
			}
			if e != nil {
				return e
			}
			if allowed && !grant {
				allowed = false
				reason = "grant_denied"
			}
			root := x.roots[x.scopes[i.Run].RootID]
			if allowed {
				if x.closed(i.Run) {
					allowed = false
					reason = "scope_closed"
				} else if x.charges[root.ID].Effects >= root.Limits.MaxEffects {
					allowed = false
					reason = "effect_limit"
				}
			}
			if r.Allowed != allowed || r.Reason != reason || r.RootID != root.ID || r.Run != i.Run || r.Sequence != i.Sequence {
				return corrupt("reservation prefix decision")
			}
			x.reservations[i.ID] = true
			if allowed {
				spent := x.charges[root.ID]
				spent.Effects++
				x.charges[root.ID] = spent
			} else if i.Kind == "external" {
				x.known[i.ID] = true
			}
		case compositionExternalResultFact:
			if !x.reservations[f.OperationID] {
				return corrupt("result before reservation")
			}
			x.known[f.OperationID] = true
		case compositionReplyFact:
			i, ok := x.intents[f.OperationID]
			if !ok || !x.authorized[i.ID] {
				return corrupt("reply before source authorization")
			}
			b, reply, e := compositionReplyBinding(ctx, q, i)
			if e != nil {
				return e
			}
			if !reply || !x.active[i.Run] {
				return corrupt("reply outside immutable endpoint")
			}
			auth, e := loadAuthorization(ctx, q, i.ID)
			if e != nil {
				return e
			}
			r := compositionReply{InvocationID: b.InvocationID, RootID: b.RootID, Intent: i, Authorization: auth, Accepted: true, Reason: "recorded_reply"}
			t, e := loadCompositionTemplate(ctx, q, b.TemplateID)
			if e != nil {
				return e
			}
			p, e := loadProgram(ctx, q, t.Members[t.EntryMember].ProgramID, l)
			if e != nil {
				return e
			}
			if !auth.Allowed {
				r.Accepted = false
				r.Reason = "source_denied:" + auth.Reason
			} else if p.Author.Ports[t.EntryPort].Send.Record.Validate(i.Payload) != nil {
				r.Accepted = false
				r.Reason = "schema_invalid"
			}
			var raw []byte
			var invocation string
			if err = q.QueryRowContext(ctx, "SELECT invocation_id,descriptor FROM composition_replies WHERE operation_id=?", i.ID).Scan(&invocation, &raw); err != nil {
				return err
			}
			if invocation != b.InvocationID || !sameJSON(raw, r) {
				return corrupt("reply candidate binding")
			}
			var transports int
			if err = q.QueryRowContext(ctx, "SELECT count(*) FROM admissions WHERE domain='message' AND json_extract(descriptor,'$.IntentID')=?", i.ID).Scan(&transports); err != nil {
				return err
			}
			if transports == 0 {
				return corrupt("public reply lacks atomic transport admission")
			}
			x.replies[i.ID] = r
		case compositionFreezeFact:
			freeze, ok := x.freezes[f.OperationID]
			if !ok || freeze.DecisionOrder != f.Order {
				return corrupt("freeze not adjacent to terminal prefix")
			}
		case compositionLifecycleFact:
			r, e := loadLifecycleResult(ctx, q, f.OperationID)
			if e != nil {
				return e
			}
			if r.DecisionSequence <= lastLifecycle {
				return corrupt("lifecycle order")
			}
			lastLifecycle = r.DecisionSequence
			if scope, ok := x.scopes[r.Request.Intent.Run]; ok && scope.ID != "" {
				if !x.authorized[f.OperationID] || x.sequences[r.Request.Intent.Run] < r.Request.Intent.Sequence {
					return corrupt("lifecycle before source")
				}
			}
			scope, scoped := x.scopes[f.Run]
			if scoped && r.Before != nil {
				if !x.active[f.Run] || x.sequences[f.Run] != r.Before.Sequence || !reflect.DeepEqual(x.states[f.Run], r.Before.Projection) || x.retired[f.Run] != r.Before.Retired {
					return corrupt("lifecycle target prefix")
				}
			}
			if scoped && r.Accepted && r.After != nil {
				x.states[f.Run] = r.After.Projection
				x.retired[f.Run] = r.After.Retired
				if r.Request.Operation == "RetireSubject" && !r.Before.Retired && r.After.Retired {
					earlier, has := x.freezes[scope.ID]
					_, closed := x.closures[scope.ID]
					_, published := x.outcomes[scope.ID]
					if !closed && !published && (!has || earlier.Status == "succeeded") {
						freeze := x.capture(scope, f.Run, f.ID, f.Order, "cancelled")
						closure := compositionClosure{RootID: scope.RootID, LifecycleOperationID: f.OperationID, Reason: "subject_retired", Freeze: freeze}
						x.closures[scope.ID] = closure
						expectedClosures = append(expectedClosures, []any{scope.ID, scope.RootID, f.Order, compositionJSON(closure)})
					}
				} else if r.Before.Projection.Status == "open" && r.After.Projection.Status != "open" {
					if freeze, yes := x.freeze(f.Run, f.ID, f.Order+1); yes {
						x.freezes[scope.ID] = freeze
						expectedFreezes = append(expectedFreezes, []any{scope.ID, scope.RootID, compositionJSON(freeze)})
					}
				}
			}
		case compositionOutcomeFact:
			outcome, e := x.expectedOutcome(f.OperationID)
			if e != nil {
				return e
			}
			outcome.DecisionOrder = f.Order
			table, column := "composition_publications", "operation_id"
			if strings.HasPrefix(f.OperationID, "root:") {
				table, column = "composition_root_publications", "scope_id"
				outcome.ID = compositionIdentity("alan/root-outcome/v1", strings.TrimPrefix(f.OperationID, "root:"))
			} else {
				outcome.ID = compositionIdentity("alan/invocation-outcome/v1", outcome.InvocationID)
			}
			var raw []byte
			if err = q.QueryRowContext(ctx, "SELECT descriptor FROM "+table+" WHERE "+column+"=?", f.OperationID).Scan(&raw); err != nil {
				return err
			}
			if !sameJSON(raw, outcome) {
				return corrupt("outcome prefix decision")
			}
			if table == "composition_root_publications" {
				var root string
				var order int64
				if err = q.QueryRowContext(ctx, "SELECT root_id,decision_order FROM composition_root_publications WHERE scope_id=?", f.OperationID).Scan(&root, &order); err != nil {
					return err
				}
				if root != strings.TrimPrefix(f.OperationID, "root:") || order != f.Order {
					return corrupt("root publication relational binding")
				}
			}
			x.outcomes[outcome.InvocationID] = outcome
		default:
			return ErrVersion
		}
		seen[f.ID] = true
	}
	if err = compositionExactRows(ctx, q, "composition_freezes", "scope_id,root_id,descriptor", expectedFreezes); err != nil {
		return err
	}
	if err = compositionExactRows(ctx, q, "composition_closures", "scope_id,root_id,decision_order,descriptor", expectedClosures); err != nil {
		return err
	}
	// Every attempt needs its reservation, including denied attempts. Results and
	// publication reverse links were checked against exact expected rows above.
	for _, table := range []string{"adapter_attempts", "external_attempts"} {
		ids, e := stringList(ctx, q, "SELECT intent_id FROM "+table)
		if e != nil {
			return e
		}
		for _, id := range ids {
			i, e := loadIntent(ctx, q, id)
			if e != nil {
				return e
			}
			root, e := compositionMembershipRoot(ctx, q, i.Run)
			if e != nil {
				return e
			}
			if root != "" && !x.reservations[id] {
				return corrupt("attempt without reservation")
			}
		}
	}
	return nil
}
