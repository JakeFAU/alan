package executor

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"

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
	seen := false
	for _, t := range c.Templates {
		h := t.Handoff
		if h == nil {
			continue
		}
		if seen || c.Version != compositionProtocolVersion {
			return fmt.Errorf("handoff requires one bounded invocation profile")
		}
		seen = true
		r, err := loadRun(ctx, q, h.Caller)
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
		port := p.Author.Ports[h.CallerPort]
		if !validHandoffPort(port, HandoffPeer) {
			return fmt.Errorf("invalid caller handoff port")
		}
		for name, other := range p.Author.Ports {
			if name != h.CallerPort && other.Peer == HandoffPeer {
				return fmt.Errorf("ambiguous caller handoff peer")
			}
		}
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
		var collision int
		if err = q.QueryRowContext(ctx, "SELECT count(*) FROM run_origins WHERE subject=?", HandoffPeer).Scan(&collision); err != nil {
			return err
		}
		if collision != 0 {
			return fmt.Errorf("handoff peer aliases static subject")
		}
	}
	return nil
}

func bindHandoffCaller(ctx context.Context, q queryer, p Program, run RunID) (Program, error) {
	if p.Author == nil {
		return p, nil
	}
	for _, port := range p.Author.Ports {
		if port.Peer != HandoffPeer {
			continue
		}
		var raw []byte
		err := q.QueryRowContext(ctx, "SELECT descriptor FROM composition_templates WHERE json_extract(descriptor,'$.Handoff.Caller.Subject')=? AND json_extract(descriptor,'$.Handoff.Caller.Generation')=?", run.Subject, run.Generation).Scan(&raw)
		if errors.Is(err, sql.ErrNoRows) {
			return p, nil
		}
		if err != nil {
			return p, err
		}
		var t compositionTemplate
		if err = decode(raw, &t); err != nil {
			return p, err
		}
		h := t.Handoff
		namespace, err := readNamespace(ctx, q)
		if err != nil {
			return p, err
		}
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
		return p, nil
	}
	return p, nil
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
