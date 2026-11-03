package executor

import (
	"reflect"
	"slices"
	"strings"
)

// InvocationView follows one committed invocation operation through its original
// admission, members, publication and the parent reduction that selected it.
// State is committed, authorized, source_refused, admitted, refused, frozen,
// closed, published or consumed. Authorization retains the source decision
// even before the separate creation admission.
// An admission can refuse creation yet still publish an outcome for its caller.
type InvocationView struct {
	OperationID, TemplateID  string
	Caller                   RunID
	Sequence, SourceRevision int64
	Inputs                   map[string]Value
	State                    InvocationStage
	// ParentSelection is unpublished, unconsumed, consumer_closed or selected.
	ParentSelection string
	Authorization   *Authorization
	Admission       *InvocationAdmission
	Freeze          *InvocationFreeze
	Closure         *InvocationClosure
	Outcome         *InvocationOutcome
	// Replies are authenticated reply attempts, in operation-ID order.
	// ResponseOperationID identifies the one reply whose payload was
	// published. Conflicting or absent replies have no selected response.
	Replies                                    []InvocationReply
	ResponseOperationID                        string
	RequestObservationID, OutcomeObservationID string
	SelectedAt                                 *ReductionRef
	// Dependencies name the member, reply, descendant, result, or local
	// publication still required at this cut. A published invocation has none.
	Dependencies []PendingObligation
}

// InvocationReply is an entry member's retained reply and its authorization.
type InvocationReply = compositionReply

// InvocationClosure retains the lifecycle decision that closed this scope. It
// is separate from a member's original terminal freeze.
type InvocationClosure struct {
	OperationID, Reason string
	Freeze              InvocationFreeze
}

// ObservationView describes an admitted observation and its actual selection.
// Source is the retained protocol principal (which can be an adapter, template
// or operator). ProducerRun is set only for a provable logical producer.
// SourceOperationID links a genuine producing request, independently of the
// envelope's Operation field which is a correlation selector. Multiple delivery
// attempts can refer to the same logical observation.
type ObservationView struct {
	Observation
	ProducerRun *RunID
	// ProducerReduction is the exact emitting or closing reduction for an
	// ordinary message, invocation request or ordinary terminal notice.
	// For an authority-caused closure, SourceOperationID identifies the
	// lifecycle decision instead. A retained envelope with no transport
	// evidence is explicitly marked unrepresented in Origin.
	ProducerReduction *ReductionRef
	Origin            string
	SourceOperationID string
	Admissions        []string
	SelectedAt        *ReductionRef
}

// AdmissionView describes one transport attempt. A refused transport has no
// observation. An accepted invocation reply is recorded for publication instead
// of creating a separate caller observation.
type AdmissionView struct {
	AdmissionResult
	ID, Domain, LogicalID                      string
	Target                                     string
	RequestedGeneration                        int64
	Source                                     RunID
	Obligation, Correlation, SourceOperationID string
}

type causalIndex map[string]map[string]any

func indexCausalRows(rows []map[string]any, key string) (causalIndex, error) {
	index := make(causalIndex, len(rows))
	for _, row := range rows {
		id, ok := row[key].(string)
		if !ok || id == "" || index[id] != nil {
			return nil, corrupt("missing or duplicate %s", key)
		}
		index[id] = row
	}
	return index, nil
}

func (rows causalIndex) take(key string) map[string]any {
	row := rows[key]
	delete(rows, key)
	return row
}

type causalCut struct {
	*inspectionBuilder
	auth           map[string]Authorization
	replies        map[string]compositionReply
	byID           map[string]*InvocationView
	templates      map[string]compositionTemplate
	lifecycle      map[string]LifecycleResult
	deniedExternal map[string]int64
}

func (b *inspectionBuilder) buildCausality() error {
	c := causalCut{inspectionBuilder: b, auth: map[string]Authorization{}, replies: map[string]compositionReply{}, byID: map[string]*InvocationView{}, templates: map[string]compositionTemplate{}, lifecycle: map[string]LifecycleResult{}}
	for _, row := range b.tables["attempts"] {
		auth, err := inspectionDecode[Authorization](row, "result")
		if err != nil {
			return err
		}
		intent, ok := b.intents[auth.IntentID]
		req, err := inspectionDecode[authorizationRequest](row, "descriptor")
		if err != nil {
			return err
		}
		source := b.runs[intent.Run]
		if !ok || source == nil || row["id"] != auth.ID || auth.ID != identity("authorization", auth.IntentID) || row["intent_id"] != auth.IntentID ||
			!reflect.DeepEqual(req, authorizationRequest{intent.Intent, intent.revision}) || auth.Source.Run != intent.Run ||
			auth.Source.Sequence < intent.Sequence || auth.Source.Sequence > source.Sequence || auth.Source.Epoch < 1 || auth.Source.Epoch > source.Epoch || auth.Source.Revision < 1 ||
			auth.Allowed != (auth.Reason == "authorized") {
			return corrupt("authorization identity or original source")
		}
		c.auth[auth.IntentID] = auth
	}
	if err := c.readLifecycleSources(); err != nil {
		return err
	}
	for _, row := range b.tables["observations"] {
		if len(inspectText(row, "descriptor")) > c.limits.MaxObservationBytes {
			return ErrLimit
		}
		o, err := inspectionDecode[Observation](row, "descriptor")
		if err != nil {
			return err
		}
		if o.ID == "" || row["id"] != o.ID || o.Run != inspectionRun(row) || b.runs[o.Run] == nil || o.Kind == "" || o.Provenance == "" || o.Source == "" || o.SourceGeneration <= 0 || o.Payload == nil {
			return corrupt("observation identity or consumer")
		}
		for key, v := range o.Payload {
			if key == "" || !validValue(v) {
				return corrupt("observation %q payload", o.ID)
			}
		}
		view := &ObservationView{Observation: o, Origin: "unrepresented"}
		if value := row["consumed_sequence"]; value != nil {
			seq, ok := value.(int64)
			ref := ReductionRef{Run: o.Run, Sequence: seq}
			step := b.steps[ref]
			if !ok || step == nil || step.SelectedObservationID != o.ID {
				return corrupt("observation %q selection", o.ID)
			}
			view.SelectedAt = &ref
		}
		b.observations[o.ID] = view
	}
	for ref, step := range b.steps {
		if step.SelectedObservationID != "" {
			o := b.observations[step.SelectedObservationID]
			if o == nil || o.SelectedAt == nil || *o.SelectedAt != ref {
				return corrupt("dangling or multiply selected observation %q", step.SelectedObservationID)
			}
		}
	}
	if err := c.readInvocations(); err != nil {
		return err
	}
	if err := c.invocationDependencies(); err != nil {
		return err
	}
	if err := c.checkAuthorizations(); err != nil {
		return err
	}
	if err := c.readAdmissions(); err != nil {
		return err
	}
	if err := c.checkHistoryOrder(); err != nil {
		return err
	}
	b.authorizations = c.auth
	b.lifecycle = c.lifecycle
	return nil
}

type terminalSource struct {
	seq        int64
	projection Projection
	routes     map[TerminalDestination]bool
	closure    string
}

func (c *causalCut) readAdmissions() error {
	terminal := map[RunID]terminalSource{}
	routes := map[Permission]bool{}
	type terminalAuthority struct {
		source   string
		revision int64
	}
	terminalRoutes := map[terminalAuthority][]TerminalDestination{}
	for _, p := range c.snap.Permissions {
		routes[p] = true
		if p.Kind == "observation" && p.Operation == "terminal" {
			key := terminalAuthority{p.Source, p.Revision}
			terminalRoutes[key] = append(terminalRoutes[key], TerminalDestination{p.Target, p.Obligation})
		}
	}
	for _, grants := range terminalRoutes {
		slices.SortFunc(grants, func(a, b TerminalDestination) int {
			if n := strings.Compare(a.Target, b.Target); n != 0 {
				return n
			}
			return strings.Compare(a.Obligation, b.Obligation)
		})
	}
	type terminalAck struct {
		run   RunID
		route string
	}
	acked := map[terminalAck]bool{}
	for _, row := range c.tables["terminal_obligations"] {
		p, err := inspectionDecode[Projection](row, "outcome")
		if err != nil {
			return err
		}
		dest, err := inspectionDecode[[]TerminalDestination](row, "destinations")
		if err != nil {
			return err
		}
		id := inspectionRun(row)
		run := c.runs[id]
		seq, ok := row["sequence"].(int64)
		if !ok || run == nil || seq != run.Sequence || p.Status == "open" || !reflect.DeepEqual(p, run.Projection) {
			return corrupt("terminal producer %v", id)
		}
		revision := inspectInt(row, "source_revision")
		if op := inspectText(row, "operation_id"); op == "" {
			step := c.steps[ReductionRef{id, seq}]
			if step == nil || step.SourceRevision != revision {
				return corrupt("terminal source revision differs from closing reduction")
			}
		}
		if dest == nil || !slices.Equal(dest, terminalRoutes[terminalAuthority{id.Subject, revision}]) {
			return corrupt("frozen terminal routes differ from original grants")
		}
		route := map[TerminalDestination]bool{}
		for _, d := range dest {
			if d.Target == "" || route[d] {
				return corrupt("invalid frozen terminal destination")
			}
			route[d] = true
		}
		status, err := inspectionDecode[map[string]bool](row, "delivery_status")
		if err != nil || status == nil {
			return corrupt("invalid frozen terminal delivery status")
		}
		for key, delivered := range status {
			if !delivered {
				return corrupt("invalid terminal acknowledgment")
			}
			acked[terminalAck{id, key}] = true
		}
		terminal[id] = terminalSource{seq, p, route, inspectText(row, "operation_id")}
	}
	settlements, err := indexCausalRows(c.tables["settlements"], "id")
	if err != nil {
		return err
	}
	messageAdmissions := map[string]string{}
	replyAdmissions := map[string]bool{}
	lifecycleAdmissions := map[string]bool{}
	bounds := transportBounds{generation: map[string]int64{}, retired: map[string]bool{}, closed: map[RunID]bool{}}
	for _, run := range c.snap.Runs {
		if run.Origin == "root" {
			bounds.generation[run.ID.Subject] = run.ID.Generation
		}
	}
	for _, row := range c.tables["admissions"] {
		d, err := inspectionDecode[AdmissionDescriptor](row, "descriptor")
		if err != nil {
			return err
		}
		if len(inspectText(row, "descriptor")) > c.limits.MaxObservationBytes {
			return ErrLimit
		}
		r, err := inspectionDecode[AdmissionResult](row, "result")
		if err != nil {
			return err
		}
		id := admissionKey(d.Kind, d.TransportID)
		if row["id"] != id || row["domain"] != d.Kind || row["transport_id"] != d.TransportID || r.TransportID != d.TransportID || d.TransportID == "" || d.LogicalID == "" || d.Target == "" || d.Generation < 0 || d.Payload == nil || (r.Bound != nil && (r.Bound.Subject != d.Target || c.runs[*r.Bound] == nil)) {
			return corrupt("transport admission identity or binding")
		}
		for k, v := range d.Payload {
			if k == "" || !validValue(v) {
				return corrupt("admission payload")
			}
		}
		view := &AdmissionView{AdmissionResult: r, ID: id, Domain: d.Kind, LogicalID: d.LogicalID, Target: d.Target, RequestedGeneration: d.Generation, Source: d.Source, Obligation: d.Obligation, Correlation: d.Operation, SourceOperationID: d.IntentID}
		c.snap.Admissions = append(c.snap.Admissions, view)
		switch d.Kind {
		case "input":
			if d.Source != (RunID{"local", 1}) || d.IntentID != "" || d.TerminalSequence != 0 {
				return corrupt("local input source")
			}
		case "message":
			i, ok := c.intents[d.IntentID]
			auth, authorized := c.auth[d.IntentID]
			if !ok || i.Kind != "message" || !authorized || !reflect.DeepEqual(d, messageDescriptor(i.Intent, d.TransportID)) || !auth.Allowed && (r.Accepted || r.Reason != "source_denied:"+auth.Reason) {
				return corrupt("message admission source or authorization")
			}
			messageAdmissions[id] = i.ID
			if reply, ok := c.replies[i.ID]; ok {
				inv := c.byID[reply.InvocationID]
				if inv == nil || r.Bound == nil || *r.Bound != inv.Caller || r.Accepted != reply.Accepted || r.Reason != reply.Reason || r.ObservationID != "" || row["observation_id"] != nil {
					return corrupt("invocation reply admission")
				}
				replyAdmissions[i.ID] = true
				continue
			}
		case "terminal":
			t, ok := terminal[d.Source]
			want := AdmissionDescriptor{TransportID: d.TransportID, Kind: "terminal", LogicalID: identity("terminal", d.Source.Subject, d.Source.Generation, t.seq, d.Target, d.Obligation), Source: d.Source, Target: d.Target, Obligation: d.Obligation, Payload: map[string]Value{"outcome": Text(t.projection.Status)}, TerminalSequence: t.seq}
			if !ok || !t.routes[TerminalDestination{d.Target, d.Obligation}] || !reflect.DeepEqual(d, want) {
				return corrupt("terminal admission does not match its producer and frozen route")
			}
			delete(acked, terminalAck{d.Source, (TerminalDestination{d.Target, d.Obligation}).key()})
		case "lifecycle":
			key := identity("lifecycle-outcome", d.IntentID)
			row := settlements[key]
			if row == nil || row["intent_id"] != d.IntentID || row["observation_id"] != r.ObservationID {
				return corrupt("lifecycle admission lacks original settlement")
			}
			o, err := inspectionDecode[LifecycleOutcome](row, "descriptor")
			if err != nil {
				return err
			}
			i, ok := c.intents[d.IntentID]
			auth, authorized := c.auth[d.IntentID]
			req := LifecycleRequest{Intent: i.Intent, SourceRevision: i.revision, Operation: i.Payload["operation"].Text, ExpectedRevision: i.Payload["revision"].Number}
			if !ok || i.Kind != "lifecycle" || !authorized || !reflect.DeepEqual(req, o.Request) || !reflect.DeepEqual(d, lifecycleDescriptor(o, d.TransportID)) {
				return corrupt("lifecycle outcome or admission identity")
			}
			want := LifecycleOutcome{Request: req, Disposition: "source_denied", Reason: auth.Reason}
			if auth.Allowed {
				result, ok := c.lifecycle[i.ID]
				if !ok || !reflect.DeepEqual(result.Request, req) {
					return corrupt("lifecycle decision original request")
				}
				want = LifecycleOutcome{Request: req, Disposition: "target_decision", Accepted: result.Accepted, Reason: result.Reason, Successor: result.Successor}
			}
			if !reflect.DeepEqual(o, want) {
				return corrupt("lifecycle settlement differs from retained decision")
			}
			lifecycleAdmissions[key] = true
		case "adapter", "receipt":
			// The effect builder additionally checks the complete source result
			// and settlement descriptor in both directions.
			if _, ok := c.intents[d.IntentID]; !ok {
				return corrupt("effect admission without original intent")
			}
		default:
			return corrupt("unknown admission domain %q", d.Kind)
		}
		settled := d.Kind == "adapter" || d.Kind == "receipt" || d.Kind == "lifecycle"
		if settled {
			if !r.Accepted || r.Bound == nil {
				return corrupt("invalid refused settlement transport")
			}
			// Settlements target the original generation even after Retry.
			bounds.generation[d.Target] = max(bounds.generation[d.Target], r.Bound.Generation)
		} else if !c.validTransport(d, r, &bounds, routes, terminal) {
			return corrupt("impossible %s transport decision %q", d.Kind, r.Reason)
		}
		if !r.Accepted {
			if r.ObservationID != "" || row["observation_id"] != nil {
				return corrupt("invalid refused admission")
			}
			continue
		}
		if r.Bound == nil || r.ObservationID == "" || row["observation_id"] != r.ObservationID || d.Generation != 0 && d.Generation != r.Bound.Generation || (!settled && r.Reason != "accepted") || (settled && r.Reason != "settled") {
			return corrupt("invalid accepted admission")
		}
		o := c.observations[r.ObservationID]
		if o == nil || !reflect.DeepEqual(o.Observation, observationFor(d, *r.Bound)) {
			return corrupt("missing or mismatched admitted observation")
		}
		o.Admissions = append(o.Admissions, id)
		o.Origin = d.Kind
		o.SourceOperationID = d.IntentID
		if d.Kind == "message" || d.Kind == "terminal" {
			source := d.Source
			o.ProducerRun = &source
		}
		if d.Kind == "message" {
			i := c.intents[d.IntentID]
			o.ProducerReduction = &ReductionRef{i.Run, i.Sequence}
		}
		if d.Kind == "terminal" {
			t := terminal[d.Source]
			o.SourceOperationID = t.closure
			if t.closure == "" {
				o.ProducerReduction = &ReductionRef{d.Source, t.seq}
			}
		}
		if err := c.checkTransportOrder(o); err != nil {
			return err
		}
	}
	if len(acked) > 0 {
		return corrupt("terminal acknowledgment lacks its original transport")
	}
	for _, row := range c.tables["message_acknowledgments"] {
		id, _ := row["admission_id"].(string)
		if messageAdmissions[id] == "" || row["intent_id"] != messageAdmissions[id] {
			return corrupt("message acknowledgment bound to another operation")
		}
	}
	for id := range c.replies {
		if !replyAdmissions[id] {
			return corrupt("reply has no originating transport")
		}
	}
	for key, row := range settlements {
		id, _ := row["intent_id"].(string)
		if c.intents[id].Kind == "lifecycle" && !lifecycleAdmissions[key] {
			return corrupt("unexplained lifecycle settlement")
		}
	}
	for _, o := range c.observations {
		slices.Sort(o.Admissions)
		switch o.Provenance {
		case "local-input", "ordinary-message", "runtime-terminal", "runtime-lifecycle", "resource-receipt", "runtime-adapter":
			if len(o.Admissions) == 0 {
				return corrupt("observation %s has no admission", o.ID)
			}
		case "runtime-invocation", "runtime-invocation-request":
			if o.SourceOperationID == "" {
				return corrupt("observation %s has no original invocation", o.ID)
			}
		}
	}
	slices.SortFunc(c.snap.Admissions, func(a, b *AdmissionView) int { return strings.Compare(a.ID, b.ID) })
	return nil
}
