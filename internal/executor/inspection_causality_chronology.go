package executor

import "strings"

// An authorization can outlive the source state that allowed or refused it.
// Check its place among the other journals even if the target is present or
// the source refused the request. Missing transports to later-created members
// need the same test, including when there is no lifecycle request at this cut.
func (b *inspectionBuilder) checkCreationChronology() error {
	needed := len(b.snap.LifecycleDecisions) > 0
	if !needed {
		for id := range b.authorizations {
			if b.intents[id].Kind == "lifecycle" {
				needed = true
				break
			}
		}
	}
	if !needed {
		for _, a := range b.snap.Admissions {
			run := b.runs[RunID{a.Target, 1}]
			if a.Bound == nil && run != nil && run.Origin == "invocation" {
				needed = true
				break
			}
		}
	}
	if !needed {
		return nil
	}
	// Reply authorization is the sole retirement exception, even before delivery.
	publicReplies := map[string]bool{}
	if b.snap.Composition != nil {
		c := causalCut{inspectionBuilder: b, templates: map[string]compositionTemplate{}}
		for _, t := range b.snap.Composition.Templates {
			c.templates[t.ID] = t
		}
		for id := range b.authorizations {
			i := b.intents[id]
			v := b.invocations[b.runs[i.Run].OriginOperationID]
			publicReplies[id] = v != nil && c.isInvocationReply(v, i.Intent)
		}
	}
	return checkMemberOrder(b.tables, publicReplies)
}

type protocolRecord struct {
	kind compositionFactKind
	id   string
}

// Each positive index denotes a transaction with durable evidence. Zero
// denotes an absent predecessor, for example a provisioned run's first step.
// Records known to be written atomically use the same index across journals.
type memberOrder struct {
	*creationEvidence
	edges            [][]int
	incoming         []int
	records          map[protocolRecord]int
	reductions       map[ReductionRef]int
	creation         map[RunID]int
	member           map[string]int
	closure          map[RunID]int
	closedByDecision map[RunID]bool
	successor        map[RunID]int
	retirement       map[string]int
	readWitnesses    map[int]Authority
}

func checkMemberOrder(tables map[string][]map[string]any, publicReplies map[string]bool) error {
	evidence, err := indexCreationEvidence(tables, publicReplies)
	if err != nil {
		return err
	}
	g := memberOrder{
		creationEvidence: evidence,
		edges:            make([][]int, 1), incoming: make([]int, 1),
		records: map[protocolRecord]int{}, reductions: map[ReductionRef]int{},
		creation: map[RunID]int{}, member: map[string]int{},
		closure: map[RunID]int{}, closedByDecision: map[RunID]bool{},
		successor: map[RunID]int{}, retirement: map[string]int{},
		readWitnesses: map[int]Authority{},
	}
	if err := g.indexTransactions(); err != nil {
		return err
	}
	g.authorityOrder()
	if err := g.transportOrder(); err != nil {
		return err
	}
	// Kahn's algorithm checks all negative existence claims jointly. For
	// example, a path from a child's admission to an allegedly earlier
	// missing-target decision now forms a cycle.
	var ready []int
	for id := 1; id < len(g.incoming); id++ {
		if g.incoming[id] == 0 {
			ready = append(ready, id)
		}
	}
	for i := 0; i < len(ready); i++ {
		for _, next := range g.edges[ready[i]] {
			g.incoming[next]--
			if g.incoming[next] == 0 {
				ready = append(ready, next)
			}
		}
	}
	if len(ready) != len(g.incoming)-1 {
		return corrupt("lifecycle or transport observations contradict retained transaction order")
	}
	return g.checkActivationCounts()
}

func (g *memberOrder) transaction() int {
	id := len(g.edges)
	g.edges = append(g.edges, nil)
	g.incoming = append(g.incoming, 0)
	return id
}

func (g *memberOrder) record(kind compositionFactKind, id string) int {
	key := protocolRecord{kind, id}
	if g.records[key] == 0 {
		g.records[key] = g.transaction()
	}
	return g.records[key]
}

func (g *memberOrder) before(a, b int) {
	if a != 0 && b != 0 && a != b {
		g.edges[a] = append(g.edges[a], b)
		g.incoming[b]++
	}
}

func (g *memberOrder) appendRecord(previous *int, id int) {
	g.before(*previous, id)
	*previous = id
}

func (g *memberOrder) indexTransactions() error {
	// A freeze and its trigger share a transaction. An adopted original
	// result and its resolver result also become durable together.
	group := map[string]string{}
	for _, row := range g.tables["composition_freezes"] {
		f, err := inspectionDecode[InvocationFreeze](row, "descriptor")
		if err != nil {
			return err
		}
		group[compositionIdentity("alan/fact/freeze/v1", f.InvocationID)] = f.Trigger
	}
	for _, row := range g.tables["adapter_results"] {
		r, err := inspectionDecode[AdapterResult](row, "descriptor")
		if err != nil {
			return err
		}
		if r.Reconciled != nil {
			group[compositionIdentity("alan/fact/external_result/v1", inspectText(row, "intent_id"))] = compositionIdentity("alan/fact/external_result/v1", r.Reconciled.QueryID)
		}
	}
	facts := map[string]int{}
	factOrder := map[int64]int{}
	readOrder := map[readOrdinalKey]int{}
	var prev int
	for _, row := range g.tables["composition_facts"] {
		f, err := inspectionDecode[compositionFact](row, "descriptor")
		if err != nil {
			return err
		}
		var node int
		if first, same := group[f.ID]; same {
			node = facts[first]
			if node == 0 || node != prev {
				return corrupt("composition records split an atomic transaction")
			}
		} else {
			node = g.transaction()
		}
		facts[f.ID] = node
		factOrder[f.Order] = node
		switch f.Kind {
		case compositionExecutionFact:
			g.reductions[ReductionRef{f.Run, f.Sequence}] = node
		case compositionReadFact:
			// Each spent ordinal has its own resolver activation witness.
			w, ok := g.reads[readOrdinalKey{f.OperationID, f.Sequence}]
			if !ok {
				return corrupt("receipt-read fact without historical resolver")
			}
			g.readWitnesses[node] = w
			readOrder[readOrdinalKey{f.OperationID, f.Sequence}] = node
		default:
			g.records[protocolRecord{f.Kind, f.OperationID}] = node
		}
		g.appendRecord(&prev, node)
	}
	prev = 0
	for _, s := range g.history {
		n := g.reductions[s.ReductionRef]
		if n == 0 {
			n = g.transaction()
			g.reductions[s.ReductionRef] = n
		}
		g.appendRecord(&prev, n)

	}
	prev = 0
	for _, d := range g.decisions {
		n := g.record(compositionLifecycleFact, d.Request.Intent.ID)
		if d.Accepted && d.Successor != 0 {
			g.successor[d.Before.Run] = n
		}
		if d.Accepted && !d.Before.Retired && d.After.Retired {
			g.retirement[d.Before.Run.Subject] = n
		}
		g.appendRecord(&prev, n)
	}
	prev = 0
	for _, row := range g.tables["lifecycle_results"] {
		g.appendRecord(&prev, g.record(compositionLifecycleFact, inspectText(row, "id")))
	}
	for _, row := range g.tables["runs"] {
		r := g.runs[inspectionRun(row)]
		switch r.origin {
		case "invocation":
			g.creation[r.ID] = g.record(compositionAdmissionFact, r.operation)
			g.member[r.ID.Subject] = g.creation[r.ID]
		case "retry":
			g.creation[r.ID] = g.record(compositionLifecycleFact, r.operation)
		}
		g.before(g.creation[r.ID], g.reductions[ReductionRef{r.ID, 1}])
	}
	prev = 0
	for _, row := range g.tables["terminal_obligations"] {
		run := inspectionRun(row)
		n := g.reductions[ReductionRef{run, inspectInt(row, "sequence")}]
		if id := inspectText(row, "operation_id"); id != "" {
			n = g.record(compositionLifecycleFact, id)
			g.closedByDecision[run] = true
		}
		g.closure[run] = n
		g.appendRecord(&prev, n)
	}
	return g.orderMirroredRows(factOrder, readOrder)
}

func (g *memberOrder) orderMirroredRows(factOrder map[int64]int, readOrder map[readOrdinalKey]int) error {
	var prev int
	// These journals mirror transactions indexed above. Their rowids impose
	// an independent order even if every retained descriptor is authentic.
	for _, field := range []struct {
		table, key string
		kind       compositionFactKind
	}{
		{"composition_admission_decisions", "operation_id", compositionAdmissionFact},
		{"composition_invocations", "operation_id", compositionAdmissionFact},
		{"composition_bindings", "invocation_id", compositionAdmissionFact},
		{"composition_members", "invocation_id", compositionAdmissionFact},
		{"composition_requests", "invocation_id", compositionAdmissionFact},
		{"composition_reservations", "operation_id", compositionReservationFact},
		{"composition_replies", "operation_id", compositionReplyFact},
		{"composition_freezes", "scope_id", compositionFreezeFact},
		{"composition_publications", "operation_id", compositionOutcomeFact},
		{"composition_outcome_deliveries", "operation_id", compositionOutcomeFact},
	} {
		prev = 0
		for _, row := range g.tables[field.table] {
			op := inspectText(row, field.key)
			if field.key == "invocation_id" {
				op = g.invocationOperations[op]
			}
			g.appendRecord(&prev, g.records[protocolRecord{field.kind, op}])
		}
	}
	for _, table := range []string{"composition_charges", "composition_closures", "composition_results", "composition_root_publications"} {
		prev = 0
		for _, row := range g.tables[table] {
			g.appendRecord(&prev, factOrder[inspectInt(row, "decision_order")])
		}
	}
	prev = 0
	for _, row := range g.tables["receipt_read_ordinals"] {
		g.appendRecord(&prev, readOrder[readOrdinalKey{inspectText(row, "intent_id"), inspectInt(row, "ordinal")}])
	}
	prev = 0
	for _, row := range g.tables["intents"] {
		g.appendRecord(&prev, g.reductions[ReductionRef{inspectionRun(row), inspectInt(row, "sequence")}])
	}
	// Provision precedes all later subject, generation and origin insertions.
	for _, table := range []string{"subjects", "runs", "run_origins"} {
		prev = 0
		for _, row := range g.tables[table] {
			n := g.creation[inspectionRun(row)]
			if table == "subjects" {
				n = g.member[inspectText(row, "id")]
			}
			if n == 0 && prev != 0 {
				return corrupt("provisioned identity inserted after a derived generation")
			}
			g.appendRecord(&prev, n)
		}
	}
	return nil
}

// An authority snapshot at sequence S precedes S+1 (when it exists), even
// if an owner reopened the Store between those two reductions.
func (g *memberOrder) witness(node int, run RunID, seq int64) {
	g.before(g.creation[run], node)
	g.before(g.reductions[ReductionRef{run, seq}], node)
	g.before(node, g.reductions[ReductionRef{run, seq + 1}])
}

func (g *memberOrder) authorityOrder() {
	var prev int
	for _, row := range g.tables["attempts"] {
		op := inspectText(row, "intent_id")
		a := g.authorizations[op]
		n := g.record(compositionAuthorizationFact, op)
		r := a.Source.Run
		g.witness(n, r, a.Source.Sequence)
		if a.ClosureOperation != "" {
			g.before(g.record(compositionLifecycleFact, a.ClosureOperation), n)
		} else if g.closedByDecision[r] {
			// AuthorizeIntent always captures a prior lifecycle closure.
			g.before(n, g.closure[r])
		}
		if a.Reason == "source_generation_changed" {
			g.before(g.successor[r], n)
		} else if a.Reason != "source_revision_changed" {
			g.before(n, g.successor[r])
		}
		if a.Reason == "source_retired" {
			g.before(g.retirement[r.Subject], n)
		} else if a.Allowed || a.Reason == "source_closed" {
			// An entry member can authorize its public reply after retirement,
			// including a still-undelivered reply without a transport record.
			if !g.publicReplies[op] {
				g.before(n, g.retirement[r.Subject])
			}
		}
		g.appendRecord(&prev, n)
	}
	for _, d := range g.decisions {
		op := d.Request.Intent.ID
		n := g.record(compositionLifecycleFact, op)
		g.before(g.record(compositionAuthorizationFact, op), n)
		if w := d.Before; w != nil {
			g.witness(n, w.Run, w.Sequence)
			g.before(n, g.successor[w.Run])
			if w.Retired {
				g.before(g.retirement[w.Run.Subject], n)
			} else {
				g.before(n, g.retirement[w.Run.Subject])
			}
			if w.Projection.Status == "open" {
				g.before(n, g.closure[w.Run])
			} else {
				g.before(g.closure[w.Run], n)
			}
		} else {
			g.before(n, g.member[d.Request.Intent.Target])
		}
	}
	for _, row := range g.tables["composition_admission_decisions"] {
		op := inspectText(row, "operation_id")
		g.before(g.record(compositionAuthorizationFact, op), g.record(compositionAdmissionFact, op))
	}
	for _, field := range []struct {
		table string
		kind  compositionFactKind
	}{{"external_attempts", compositionReservationFact}, {"adapter_attempts", compositionReservationFact}, {"external_receipts", compositionExternalResultFact}, {"adapter_results", compositionExternalResultFact}} {
		prev = 0
		for _, row := range g.tables[field.table] {
			op := inspectText(row, "intent_id")
			n := g.record(field.kind, op)
			pred := compositionReservationFact
			if field.kind == compositionReservationFact {
				pred = compositionAuthorizationFact
			}
			g.before(g.record(pred, op), n)
			g.appendRecord(&prev, n)
		}
	}
}

func (g *memberOrder) transportOrder() error {
	firstReply := map[string]bool{}
	observation := map[string]int{}
	originalSettlement := map[string]int{}
	var prev int
	for _, row := range g.tables["admissions"] {
		a, err := inspectionDecode[AdmissionResult](row, "result")
		if err != nil {
			return err
		}
		d, err := inspectionDecode[AdmissionDescriptor](row, "descriptor")
		if err != nil {
			return err
		}
		op := d.IntentID
		n := 0
		if d.Kind == "message" {
			if first := g.records[protocolRecord{compositionReplyFact, op}]; first != 0 && !firstReply[op] {
				n = first // first delivery recorded the reply fact in this transaction
				firstReply[op] = true
			}
		}
		if n == 0 {
			n = g.transaction()
		}
		g.appendRecord(&prev, n)
		if a.Bound == nil {
			if g.provisioned[d.Target] {
				return corrupt("transport missing a provisioned target")
			}
			if a.Accepted || (a.Reason != "target_missing" && !strings.HasPrefix(a.Reason, "source_denied:")) {
				return corrupt("transport refusal lacks its target witness")
			}
			g.before(n, g.member[d.Target])
		} else {
			if _, ok := g.runs[*a.Bound]; !ok || a.Bound.Subject != d.Target || a.Reason == "target_missing" {
				return corrupt("transport cites a foreign target witness")
			}
			g.before(g.creation[*a.Bound], n)
		}
		switch d.Kind {
		case "terminal":
			g.before(g.closure[d.Source], n)
		case "input":
		default:
			g.before(g.record(compositionAuthorizationFact, op), n)
		}
		switch d.Kind {
		case "lifecycle", "adapter", "receipt":
			if originalSettlement[op] == 0 {
				originalSettlement[op] = n
			}
			if d.Kind == "lifecycle" {
				g.before(g.records[protocolRecord{compositionLifecycleFact, op}], n) // source-denied settlements have no target decision
			} else {
				g.before(g.records[protocolRecord{compositionReservationFact, op}], n)
				g.before(g.records[protocolRecord{compositionExternalResultFact, op}], n)
			}
		default:
			if a.Bound != nil && g.records[protocolRecord{compositionReplyFact, op}] == 0 {
				id := *a.Bound
				g.before(n, g.successor[id])
				if a.Reason == "target_retired" {
					g.before(g.retirement[id.Subject], n)
				} else if !strings.HasPrefix(a.Reason, "source_denied:") {
					g.before(n, g.retirement[id.Subject])
				}
				if a.Reason == "target_closed" {
					g.before(g.closure[id], n)
				} else if a.Accepted || a.Reason == "route_denied" || a.Reason == "schema_invalid" {
					g.before(n, g.closure[id])
				}
			}
		}
		if a.ObservationID != "" && observation[a.ObservationID] == 0 {
			observation[a.ObservationID] = n
		}
	}
	prev = 0
	for _, row := range g.tables["settlements"] {
		g.appendRecord(&prev, originalSettlement[inspectText(row, "intent_id")])
	}
	for _, row := range g.tables["composition_requests"] {
		op := g.invocationOperations[inspectText(row, "invocation_id")]
		observation[inspectText(row, "observation_id")] = g.record(compositionAdmissionFact, op)
	}
	for _, row := range g.tables["composition_outcome_deliveries"] {
		observation[inspectText(row, "observation_id")] = g.record(compositionOutcomeFact, inspectText(row, "operation_id"))
	}
	prev = 0
	for _, row := range g.tables["observations"] {
		id := inspectText(row, "id")
		run := inspectionRun(row)
		n := observation[id]
		if n == 0 {
			n = g.transaction() // trusted envelope with no represented transport
		}
		g.before(g.creation[run], n)
		if seq, ok := row["consumed_sequence"].(int64); ok {
			g.before(n, g.reductions[ReductionRef{run, seq}])
		}
		g.appendRecord(&prev, n)
	}
	return nil
}
