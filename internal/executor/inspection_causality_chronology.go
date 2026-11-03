package executor

import "strings"

// Most cuts do not contain a negative existence claim about a subject that
// exists now. In the exceptional case, ask whether every journal could have
// been written in a single order respecting the claimed absence. The order
// is a consistency test; it is never exposed as reconstructed history.
func (b *inspectionBuilder) checkCreationChronology() error {
	if !b.snap.Features.Composition {
		return nil
	}
	futureMember := func(subject string) bool {
		run := b.runs[RunID{subject, 1}]
		return run != nil && run.Origin == "invocation"
	}
	for _, d := range b.snap.LifecycleDecisions {
		if d.Before == nil && futureMember(d.Request.Intent.Target) {
			return b.checkMemberOrder()
		}
	}
	for _, a := range b.snap.Admissions {
		if a.Bound == nil && futureMember(a.Target) {
			return b.checkMemberOrder()
		}
	}
	return nil
}

type protocolRecord struct {
	kind compositionFactKind
	id   string
}

// Each positive index denotes a transaction with durable evidence. Zero
// denotes an absent predecessor, for example a provisioned run's first step.
// Records known to be written atomically use the same index across journals.
type memberOrder struct {
	*inspectionBuilder
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

func (b *inspectionBuilder) checkMemberOrder() error {
	g := memberOrder{
		inspectionBuilder: b,
		edges:             make([][]int, 1), incoming: make([]int, 1),
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
	g.transportOrder()
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
		return corrupt("missing target contradicts retained creation chronology")
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
	for _, x := range g.effects {
		if r := x.AdapterResult; r != nil && r.Reconciled != nil {
			group[compositionIdentity("alan/fact/external_result/v1", x.OperationID)] = compositionIdentity("alan/fact/external_result/v1", r.Reconciled.QueryID)
		}
	}
	facts := map[string]int{}
	var prev int
	for _, row := range g.tables["composition_facts"] {
		f := g.facts[inspectText(row, "id")]
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
		switch f.Kind {
		case compositionExecutionFact:
			g.reductions[ReductionRef{f.Run, f.Sequence}] = node
		case compositionReadFact:
			// Each spent ordinal has its own resolver activation witness.
			x := g.effects[f.OperationID]
			if x == nil || f.Sequence < 1 || f.Sequence > int64(len(x.ReadOpportunities)) {
				return corrupt("receipt-read fact without historical resolver")
			}
			g.readWitnesses[node] = x.ReadOpportunities[f.Sequence-1].Resolver
		default:
			g.records[protocolRecord{f.Kind, f.OperationID}] = node
		}
		g.appendRecord(&prev, node)
	}
	prev = 0
	for _, s := range g.snap.history {
		n := g.reductions[s.ReductionRef]
		if n == 0 {
			n = g.transaction()
			g.reductions[s.ReductionRef] = n
		}
		g.appendRecord(&prev, n)
	}
	prev = 0
	for _, d := range g.snap.LifecycleDecisions {
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
	prev = 0
	for _, row := range g.tables["runs"] {
		r := g.runs[inspectionRun(row)]
		switch r.Origin {
		case "invocation":
			g.creation[r.ID] = g.record(compositionAdmissionFact, r.OriginOperationID)
			g.member[r.ID.Subject] = g.creation[r.ID]
		case "retry":
			g.creation[r.ID] = g.record(compositionLifecycleFact, r.OriginOperationID)
		}
		if n := g.creation[r.ID]; n != 0 {
			g.appendRecord(&prev, n)
			g.before(n, g.reductions[ReductionRef{r.ID, 1}])
		}
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
	c := causalCut{inspectionBuilder: g.inspectionBuilder, templates: map[string]compositionTemplate{}}
	for _, t := range g.snap.Composition.Templates {
		c.templates[t.ID] = t
	}
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
			v := g.invocations[g.runs[r].OriginOperationID]
			if v == nil || !c.isInvocationReply(v, g.intents[op].Intent) {
				g.before(n, g.retirement[r.Subject])
			}
		}
		g.appendRecord(&prev, n)
	}
	for _, d := range g.snap.LifecycleDecisions {
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
	for _, v := range g.invocations {
		if v.Admission != nil {
			g.before(g.record(compositionAuthorizationFact, v.OperationID), g.record(compositionAdmissionFact, v.OperationID))
		}
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

func (g *memberOrder) transportOrder() {
	admissions := map[string]*AdmissionView{}
	for _, a := range g.snap.Admissions {
		admissions[a.ID] = a
	}
	firstReply := map[string]bool{}
	observation := map[string]int{}
	originalSettlement := map[string]int{}
	var prev int
	for _, row := range g.tables["admissions"] {
		a := admissions[inspectText(row, "id")]
		op := a.SourceOperationID
		n := 0
		if a.Domain == "message" {
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
			g.before(n, g.member[a.Target])
		} else {
			g.before(g.creation[*a.Bound], n)
		}
		switch a.Domain {
		case "terminal":
			g.before(g.closure[a.Source], n)
		case "input":
		default:
			g.before(g.record(compositionAuthorizationFact, op), n)
		}
		switch a.Domain {
		case "lifecycle", "adapter", "receipt":
			if originalSettlement[op] == 0 {
				originalSettlement[op] = n
			}
			if d := g.records[protocolRecord{compositionLifecycleFact, op}]; a.Domain == "lifecycle" {
				g.before(d, n) // source-denied settlements have no target decision
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
	for _, v := range g.invocations {
		if v.RequestObservationID != "" {
			observation[v.RequestObservationID] = g.record(compositionAdmissionFact, v.OperationID)
		}
		if v.OutcomeObservationID != "" {
			observation[v.OutcomeObservationID] = g.record(compositionOutcomeFact, v.OperationID)
		}
	}
	prev = 0
	for _, row := range g.tables["observations"] {
		id := inspectText(row, "id")
		o := g.observations[id]
		n := observation[id]
		if n == 0 {
			n = g.transaction() // trusted envelope with no represented transport
		}
		g.before(g.creation[o.Run], n)
		if o.SelectedAt != nil {
			g.before(n, g.reductions[*o.SelectedAt])
		}
		g.appendRecord(&prev, n)
	}
}
