package executor

import "context"

// Replay and owner reopen call this after authenticating the underlying
// protocols and before leaving the validating transaction. A lifecycle-free
// history needs no additional order check unless a transport claimed absence
// for a member created in this cut.
func validateLifecycleChronology(ctx context.Context, q queryer, profile string, publicReplies map[string]bool) error {
	query := `SELECT EXISTS (SELECT 1 FROM lifecycle_results)
		OR EXISTS (SELECT 1 FROM attempts
			WHERE json_extract(CAST(descriptor AS TEXT),'$.Intent.Kind')='lifecycle')`
	if profile == compositionProtocolVersion {
		query += ` OR EXISTS (SELECT 1 FROM admissions a JOIN composition_members m
			ON m.subject=json_extract(CAST(a.descriptor AS TEXT),'$.Target')
			WHERE json_extract(CAST(a.result AS TEXT),'$.Bound') IS NULL)`
	}
	var needed bool
	if err := q.QueryRowContext(ctx, query).Scan(&needed); err != nil || !needed {
		return err
	}
	names := []string{
		"subjects", "runs", "run_origins", "history", "intents", "attempts", "lifecycle_results",
		"terminal_obligations", "observations", "admissions", "settlements",
		"external_attempts", "adapter_attempts", "external_receipts", "adapter_results",
	}
	if profile != "" {
		names = append(names, "composition_facts")
	}
	if profile == compositionProtocolVersion {
		names = append(names, "composition_members", "composition_invocations",
			"composition_freezes", "composition_admission_decisions",
			"composition_requests", "composition_outcome_deliveries",
			"composition_bindings", "composition_reservations", "composition_replies",
			"composition_publications", "composition_results", "composition_root_publications",
			"composition_closures", "composition_charges")
		n, err := schemaNumber(ctx, q)
		if err != nil {
			return err
		}
		if n == reconciliationSchemaVersion {
			names = append(names, "receipt_read_ordinals")
		}
	}
	tables := make(map[string][]map[string]any, len(names))
	for _, name := range names {
		rows, err := scanInspectionTable(ctx, q, name, true)
		if err != nil {
			return err
		}
		tables[name] = rows
	}
	return checkMemberOrder(tables, publicReplies)
}

type creationRun struct {
	ID                RunID
	origin, operation string
	epoch             int64
	open              bool
}

type creationStep struct {
	ReductionRef
	Epoch int64
}

type readOrdinalKey struct {
	id      string
	ordinal int64
}

// These are indexes over the original protocol rows shared by both readers.
// Rows stay in append order. decision_sequence independently orders the target
// journal, so the graph checks it against insertion order as well.
type creationEvidence struct {
	tables               map[string][]map[string]any
	publicReplies        map[string]bool
	provisioned          map[string]bool
	runs                 map[RunID]creationRun
	history              []creationStep
	openAfter            map[ReductionRef]bool
	decisions            []LifecycleResult
	authorizations       map[string]Authorization
	invocationOperations map[string]string
	reads                map[readOrdinalKey]Authority
}

func indexCreationEvidence(tables map[string][]map[string]any, publicReplies map[string]bool) (*creationEvidence, error) {
	e := &creationEvidence{
		tables: tables, publicReplies: publicReplies, runs: map[RunID]creationRun{}, provisioned: map[string]bool{},
		openAfter: map[ReductionRef]bool{}, authorizations: map[string]Authorization{},
		invocationOperations: map[string]string{}, reads: map[readOrdinalKey]Authority{},
		decisions: make([]LifecycleResult, len(tables["lifecycle_results"])),
	}
	for _, row := range tables["composition_invocations"] {
		e.invocationOperations[inspectText(row, "id")] = inspectText(row, "operation_id")
	}
	for _, row := range tables["run_origins"] {
		id := inspectionRun(row)
		origin := inspectText(row, "kind")
		e.runs[id] = creationRun{ID: id, origin: origin, operation: inspectText(row, "operation_id")}
		if origin == "root" {
			e.provisioned[id.Subject] = true
		}
	}
	for _, row := range tables["composition_members"] {
		id := inspectionRun(row)
		e.runs[id] = creationRun{ID: id, origin: "invocation", operation: e.invocationOperations[inspectText(row, "invocation_id")]}
	}
	for _, row := range tables["runs"] {
		id := inspectionRun(row)
		run, ok := e.runs[id]
		if !ok {
			return nil, corrupt("missing run creation")
		}
		p, err := inspectionDecode[Projection](row, "projection")
		if err != nil {
			return nil, err
		}
		run.epoch, run.open = inspectInt(row, "epoch"), p.Status == "open"
		e.runs[id] = run
	}
	lastSequence := map[RunID]int64{}
	for _, row := range tables["history"] {
		ref := ReductionRef{inspectionRun(row), inspectInt(row, "sequence")}
		if ref.Sequence != lastSequence[ref.Run]+1 {
			return nil, corrupt("run reductions out of append order")
		}
		lastSequence[ref.Run] = ref.Sequence
		p, err := inspectionDecode[Projection](row, "successor")
		if err != nil {
			return nil, err
		}
		e.openAfter[ref] = p.Status == "open"
		e.history = append(e.history, creationStep{ref, inspectInt(row, "epoch")})
	}
	for _, row := range tables["attempts"] {
		a, err := inspectionDecode[Authorization](row, "result")
		if err != nil {
			return nil, err
		}
		e.authorizations[a.IntentID] = a
	}
	for _, row := range tables["lifecycle_results"] {
		d, err := inspectionDecode[LifecycleResult](row, "result")
		if err != nil {
			return nil, err
		}
		if d.DecisionSequence <= 0 || d.DecisionSequence > int64(len(e.decisions)) {
			return nil, corrupt("invalid lifecycle decision order")
		}
		e.decisions[d.DecisionSequence-1] = d
	}
	for _, row := range tables["receipt_read_ordinals"] {
		o, err := inspectionDecode[receiptReadOrdinal](row, "descriptor")
		if err != nil {
			return nil, err
		}
		e.reads[readOrdinalKey{inspectText(row, "intent_id"), inspectInt(row, "ordinal")}] = o.Resolver
	}
	return e, nil
}
