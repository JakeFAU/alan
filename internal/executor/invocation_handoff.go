package executor

import (
	"context"
	"fmt"
	"reflect"
	"slices"

	recordschema "github.com/JakeFAU/alan/schema"
)

func validHandoffPort(p PortDeclaration, peer string) bool {
	want := recordschema.Record{"target": {Kind: "string", MaxBytes: 1024}}
	return !p.Local && p.Peer == peer && p.SendUsed && p.ReceiveUsed && reflect.DeepEqual(p.Send.Record, want) && reflect.DeepEqual(p.Receive.Record, want)
}

func handoffOperation(h *InvocationHandoff) string {
	return identity(h.Caller.Subject, h.Caller.Generation, h.Sequence, 0)
}

func validateHandoff(ctx context.Context, q queryer, c compositionProvision, l Limits) error {
	var edges []compositionTemplate
	for _, t := range c.Templates {
		if t.Handoff != nil {
			edges = append(edges, t)
		}
	}
	if len(edges) == 0 {
		return nil
	}
	paired := edges[0].Version == PairedHandoffTemplateVersion
	if c.Version != compositionProtocolVersion || (paired && len(edges) != 2) || (!paired && len(edges) != 1) {
		return fmt.Errorf("invalid number of fixed handoff edges")
	}
	caller := edges[0].Handoff.Caller
	r, err := loadRun(ctx, q, caller)
	if err != nil {
		return err
	}
	p, err := loadProgram(ctx, q, r.ProgramID, l)
	if err != nil {
		return err
	}
	if p.Author == nil {
		return fmt.Errorf("handoff caller lacks declarations")
	}
	ports := map[string]bool{}
	obligations := map[string]bool{}
	sequences := map[int64]bool{}
	for _, t := range edges {
		h := t.Handoff
		if h.Caller != caller || (t.Version == PairedHandoffTemplateVersion) != paired || ports[h.CallerPort] || sequences[h.Sequence] {
			return fmt.Errorf("ambiguous handoff locator or caller port")
		}
		ports[h.CallerPort] = true
		sequences[h.Sequence] = true
		port := p.Author.Ports[h.CallerPort]
		if !validHandoffPort(port, HandoffPeer) {
			return fmt.Errorf("invalid caller handoff port")
		}
		if obligations[port.Obligation] {
			return fmt.Errorf("ambiguous handoff obligation")
		}
		obligations[port.Obligation] = true
		declared := false
		for _, d := range p.Author.Invocations {
			if d.TemplateID == t.ID {
				declared = true
			}
		}
		if !declared {
			return fmt.Errorf("handoff caller has no matching invocation")
		}
		child, err := loadProgram(ctx, q, t.Members[t.EntryMember].ProgramID, l)
		if err != nil {
			return err
		}
		if port.Obligation != child.Author.Ports[h.EntryPort].Obligation {
			return fmt.Errorf("handoff obligations differ")
		}
	}
	for name, other := range p.Author.Ports {
		if other.Peer == HandoffPeer && !ports[name] {
			return fmt.Errorf("ambiguous caller handoff peer")
		}
	}
	var collision int
	if err = q.QueryRowContext(ctx, "SELECT count(*) FROM run_origins WHERE subject=?", HandoffPeer).Scan(&collision); err != nil {
		return err
	}
	if collision != 0 {
		return fmt.Errorf("handoff peer aliases static subject")
	}
	if paired {
		return validatePairedHandoff(ctx, q, c, caller, edges)
	}
	return nil
}

// Both materialized endpoints belong to the one fixed root. Only actual
// admission may create its directed message permissions.
func validatePairedHandoff(ctx context.Context, q queryer, c compositionProvision, caller RunID, edges []compositionTemplate) error {
	root, _, err := loadRoot(ctx, q, caller.Subject)
	if err != nil {
		return err
	}
	if root.Subject.Generation != 1 || root.Subject.Revision != 1 {
		return fmt.Errorf("paired handoff requires static generation and revision one")
	}
	allowed := false
	for _, r := range c.Roots {
		if r.Run == caller && slices.Contains(r.Templates, edges[0].ID) && slices.Contains(r.Templates, edges[1].ID) {
			allowed = true
		}
	}
	if !allowed {
		return fmt.Errorf("paired handoff templates require their owning caller root")
	}
	ns, err := readNamespace(ctx, q)
	if err != nil {
		return err
	}
	var raw []byte
	var grants []Permission
	if err = q.QueryRowContext(ctx, "SELECT descriptor FROM composition_permissions WHERE singleton=1").Scan(&raw); err != nil {
		return err
	}
	if err = decode(raw, &grants); err != nil {
		return err
	}
	for _, t := range edges {
		child := compositionMemberID(compositionInvocationID(ns, handoffOperation(t.Handoff)), t.EntryMember)
		var collision int
		if err = q.QueryRowContext(ctx, "SELECT count(*) FROM run_origins WHERE subject=?", child.Subject).Scan(&collision); err != nil {
			return err
		}
		if collision != 0 {
			return fmt.Errorf("handoff member aliases static subject")
		}
		for _, g := range grants {
			if g.Kind == "message" && (g.Target == child.Subject || g.Source == child.Subject || g.Target == HandoffPeer || g.Source == HandoffPeer) {
				return fmt.Errorf("static grant aliases a handoff endpoint")
			}
		}
		for _, outer := range c.Templates {
			if slices.Contains(outer.Templates, t.ID) {
				return fmt.Errorf("paired handoff cannot be nested")
			}
		}
	}
	return nil
}

func bindHandoffCaller(ctx context.Context, q queryer, p Program, run RunID) (Program, error) {
	if p.Author != nil {
		for _, port := range p.Author.Ports {
			if port.Peer == HandoffPeer {
				return bindHandoffPorts(ctx, q, p, run)
			}
		}
	}
	return p, nil
}
func bindHandoffPorts(ctx context.Context, q queryer, p Program, run RunID) (Program, error) {
	namespace, err := readNamespace(ctx, q)
	if err != nil {
		return p, err
	}
	rows, err := q.QueryContext(ctx, "SELECT descriptor FROM composition_templates WHERE json_extract(descriptor,'$.Handoff.Caller.Subject')=? AND json_extract(descriptor,'$.Handoff.Caller.Generation')=? ORDER BY id", run.Subject, run.Generation)
	if err != nil {
		return p, err
	}
	defer rows.Close()
	for rows.Next() {
		var raw []byte
		if err = rows.Scan(&raw); err != nil {
			return p, err
		}
		var t compositionTemplate
		if err = decode(raw, &t); err != nil {
			return p, err
		}
		h := t.Handoff
		peer := compositionMemberID(compositionInvocationID(namespace, handoffOperation(h)), t.EntryMember)
		hp := p.Author.Ports[h.CallerPort]
		for i, in := range p.Code {
			if e := in.Effect; e != nil && e.Kind == "message" && reflect.DeepEqual(e.Target, Lit(Text(HandoffPeer))) && reflect.DeepEqual(e.Obligation, Lit(Text(hp.Obligation))) {
				p.Code[i].Effect.Target = Lit(Text(peer.Subject))
			}
			if m := in.Match; m != nil && m.Kind == "message" && reflect.DeepEqual(m.Source, Lit(Text(HandoffPeer))) && reflect.DeepEqual(m.Obligation, Lit(Text(hp.Obligation))) {
				p.Code[i].Match.Source = Lit(Text(peer.Subject))
			}
		}
		hp.Peer = peer.Subject
		p.Author.Ports[h.CallerPort] = hp
	}
	return p, rows.Err()
}

// The grant exists only after admission of the exact authorized invocation.
func handoffPermissions(t compositionTemplate, i compositionInvocation, p Program) []Permission {
	h := t.Handoff
	if h == nil || h.Caller != i.Caller || handoffOperation(h) != i.OperationID {
		return nil
	}
	child := compositionMemberID(i.ID, t.EntryMember)
	ob := p.Author.Ports[h.EntryPort].Obligation
	return []Permission{
		{Kind: "message", Source: child.Subject, Target: h.Caller.Subject, Operation: "message", Revision: 1, Obligation: ob},
		{Kind: "message", Source: h.Caller.Subject, Target: child.Subject, Operation: "message", Revision: 1, Obligation: ob},
	}
}
