package executor

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"

	recordschema "github.com/JakeFAU/alan/schema"
)

func retainCompositionTemplate(ctx context.Context, q queryer, t compositionTemplate, l Limits) (compositionTemplate, error) {
	return retainTemplate(t, l, func(id string) (Program, error) { return loadProgram(ctx, q, id, l) })
}
func retainTemplate(t compositionTemplate, l Limits, lookup func(string) (Program, error)) (compositionTemplate, error) {
	if err := validateCompositionEncoding(compositionProvision{Templates: []compositionTemplate{t}}); err != nil {
		return t, err
	}
	raw, err := json.Marshal(t)
	if err != nil {
		return t, err
	}
	if len(raw) > l.MaxProgramBytes {
		return t, ErrLimit
	}
	var retained compositionTemplate
	if err = decode(raw, &retained); err != nil {
		return t, err
	}
	t = retained
	if t.Version != "" && t.Version != compositionTemplateVersion {
		return t, ErrVersion
	}
	t.Version = compositionTemplateVersion
	if len(t.Members) == 0 || recordschema.CheckIdentifier(t.EntryMember) != nil || recordschema.CheckIdentifier(t.EntryPort) != nil || recordschema.CheckIdentifier(t.CallerPeer) != nil {
		return t, fmt.Errorf("invalid template boundary")
	}
	if _, ok := t.Members[t.CallerPeer]; ok {
		return t, fmt.Errorf("caller aliases template member")
	}
	t.Templates, err = canonicalStrings(t.Templates)
	if err != nil {
		return t, err
	}
	t.Capabilities, err = canonicalStrings(t.Capabilities)
	if err != nil {
		return t, err
	}
	programs := map[string]Program{}
	for name, m := range t.Members {
		if recordschema.CheckIdentifier(name) != nil {
			return t, fmt.Errorf("invalid template member")
		}
		p, err := lookup(m.ProgramID)
		if err != nil {
			return t, err
		}
		if p.Author == nil {
			return t, fmt.Errorf("template members require declared ports")
		}
		if err = validateAuthorInitial(p, m.Initial); err != nil {
			return t, err
		}
		if err = validateProjection(m.Initial, p, l); err != nil {
			return t, err
		}
		if m.Initial.Status != "open" {
			return t, fmt.Errorf("closed template member")
		}
		for _, e := range p.Author.Effects {
			if !slices.Contains(t.Capabilities, e.Capability) {
				return t, fmt.Errorf("undeclared template capability")
			}
		}
		if err = validateCompositionEndpoints(p); err != nil {
			return t, err
		}
		programs[name] = p
	}
	public := 0
	for name, p := range programs {
		for portName, port := range p.Author.Ports {
			if port.Local {
				return t, fmt.Errorf("template cannot expose local input")
			}
			if port.Peer == t.CallerPeer {
				if name != t.EntryMember || portName != t.EntryPort {
					return t, fmt.Errorf("extra caller boundary")
				}
				public++
				continue
			}
			peer, ok := programs[port.Peer]
			if !ok {
				return t, fmt.Errorf("template peer outside closure")
			}
			if port.SendUsed {
				matched := false
				for _, other := range peer.Author.Ports {
					matched = matched || !other.Local && other.ReceiveUsed && other.Peer == name && other.Obligation == port.Obligation && other.Receive.ID == port.Send.ID
				}
				if !matched {
					return t, fmt.Errorf("template send has no compatible internal receiver")
				}
			}
			if port.ReceiveUsed {
				matched := false
				for _, other := range peer.Author.Ports {
					matched = matched || !other.Local && other.SendUsed && other.Peer == name && other.Obligation == port.Obligation && other.Send.ID == port.Receive.ID
				}
				if !matched {
					return t, fmt.Errorf("template receive has no compatible internal sender")
				}
			}
		}
	}
	if public != 1 {
		return t, fmt.Errorf("template requires one caller boundary")
	}
	id := t.ID
	t.ID = ""
	expected := compositionIdentity(compositionTemplateVersion, t)
	if id != "" && id != expected {
		return t, fmt.Errorf("template identity mismatch")
	}
	t.ID = expected
	return t, nil
}

// Verify that every authority-bearing endpoint is justified by immutable author
// metadata before a template enters the catalogue. Dynamic targets cannot be
// resolved from payloads or unrelated local variables.
func validateCompositionEndpoints(p Program) error {
	for pc, in := range p.Code {
		if in.Op == "emit" {
			if in.Effect == nil {
				return fmt.Errorf("missing template effect")
			}
			if in.Effect.Kind == "invocation" {
				matched := false
				for name, e := range p.Author.Invocations {
					matched = matched || reflect.DeepEqual(in.Effect.Target, Lit(Text(e.TemplateID))) && reflect.DeepEqual(in.Effect.Obligation, Lit(Text(name))) && reflect.DeepEqual(in.Effect.Generation, Lit(Int(1)))
				}
				if !matched {
					return fmt.Errorf("undeclared invocation endpoint")
				}
				continue
			}
			if in.Effect.Kind == "adapter" {
				matched := false
				for name, e := range p.Author.Effects {
					matched = matched || reflect.DeepEqual(in.Effect.Target, Lit(Text(e.Capability))) && reflect.DeepEqual(in.Effect.Obligation, Lit(Text(name))) && reflect.DeepEqual(in.Effect.Generation, Lit(Int(1)))
				}
				if !matched {
					return fmt.Errorf("template adapter lacks capability declaration")
				}
				continue
			}
			if in.Effect.Kind != "message" {
				return fmt.Errorf("unsupported template effect %q", in.Effect.Kind)
			}
		}
		send := in.Op == "emit" && in.Effect != nil && in.Effect.Kind == "message"
		receive := in.Op == "receive" && in.Match != nil
		if !send && !receive {
			continue
		}
		if receive && (in.Match.Kind == "adapter" || in.Match.Kind == "invocation") {
			continue
		}
		if receive && (in.Match.Kind != "message" || in.Match.Provenance != "ordinary-message" || !reflect.DeepEqual(in.Match.Operation, Lit(Text("")))) {
			return fmt.Errorf("unsupported template receive")
		}
		matched := false
		for _, port := range p.Author.Ports {
			if port.Local || send && !port.SendUsed || receive && !port.ReceiveUsed {
				continue
			}
			if send {
				matched = matched || reflect.DeepEqual(in.Effect.Target, Lit(Text(port.Peer))) && reflect.DeepEqual(in.Effect.Obligation, Lit(Text(port.Obligation))) && reflect.DeepEqual(in.Effect.Generation, Lit(Int(1)))
			} else {
				matched = matched || reflect.DeepEqual(in.Match.Source, Lit(Text(port.Peer))) && reflect.DeepEqual(in.Match.Obligation, Lit(Text(port.Obligation))) && reflect.DeepEqual(in.Match.Generation, Lit(Int(1))) && p.Author.Receives[pc].ID == port.Receive.ID
			}
		}
		if !matched {
			return fmt.Errorf("template endpoint lacks declared metadata at %d", pc)
		}
	}
	return nil
}

type compositionEnvironment struct{ binding, template string }

func digestIdentity(v string) bool {
	raw, err := hex.DecodeString(v)
	return err == nil && len(raw) == 32 && hex.EncodeToString(raw) == v
}
func retainCompositionBinding(namespace, rootID string, t compositionTemplate, b compositionBinding) (compositionEnvironment, error) {
	var empty compositionEnvironment
	if err := validateCompositionEncoding(compositionProvision{Templates: []compositionTemplate{t}}); err != nil {
		return empty, err
	}
	if b.Version != compositionBindingVersion || t.Version != compositionTemplateVersion || b.Protocol != "" && b.Protocol != compositionProtocolVersion {
		return empty, ErrVersion
	}
	templateID := t.ID
	t.ID = ""
	expected := compositionIdentity(compositionTemplateVersion, t)
	t.ID = templateID
	if templateID != expected || !digestIdentity(namespace) || !digestIdentity(rootID) || b.Namespace != namespace || b.RootID != rootID || b.TemplateID != t.ID || recordschema.CheckIdentifier(b.OperationID) != nil || b.InvocationID != compositionInvocationID(namespace, b.OperationID) || recordschema.CheckIdentifier(b.Caller.Subject) != nil || b.Caller.Generation != 1 || b.Depth < 1 {
		return empty, fmt.Errorf("invalid binding identity")
	}
	if len(b.Members) != len(t.Members) || len(b.Programs) != len(t.Members) {
		return empty, fmt.Errorf("incomplete binding")
	}
	for name, m := range t.Members {
		id := compositionMemberID(b.InvocationID, name)
		if b.Members[name] != id || b.Programs[name] != m.ProgramID || b.Caller == id {
			return empty, fmt.Errorf("cross-scope or inconsistent member binding")
		}
	}
	id := b.ID
	b.ID = ""
	expected = compositionIdentity(compositionBindingVersion, b)
	if id != "" && id != expected {
		return empty, fmt.Errorf("binding identity mismatch")
	}
	b.ID = expected
	raw, err := json.Marshal(b)
	if err != nil {
		return empty, err
	}
	template, err := json.Marshal(t)
	if err != nil {
		return empty, err
	}
	return compositionEnvironment{string(raw), string(template)}, nil
}

// resolveCompositionProgram produces an ephemeral interpretation, never a stored
// artifact or an ID alias. Only exact ordinary message endpoint operands backed
// by used port metadata are resolved. Payloads and unrelated strings stay literal.
// Later runtime/replay integration must use this same interpretation on each side.
func resolveCompositionProgram(p Program, env compositionEnvironment, member string) (Program, error) {
	var b compositionBinding
	var t compositionTemplate
	if err := decode([]byte(env.binding), &b); err != nil {
		return Program{}, err
	}
	if err := decode([]byte(env.template), &t); err != nil {
		return Program{}, err
	}
	validated, err := retainCompositionBinding(b.Namespace, b.RootID, t, b)
	if err != nil {
		return Program{}, fmt.Errorf("invalid retained binding: %w", err)
	}
	if validated != env {
		return Program{}, fmt.Errorf("noncanonical retained binding")
	}
	m, ok := t.Members[member]
	if !ok {
		return Program{}, fmt.Errorf("unknown bound member")
	}
	raw, id, err := EncodeProgram(p)
	if err != nil {
		return Program{}, err
	}
	if id != m.ProgramID || p.Author == nil {
		return Program{}, fmt.Errorf("binding artifact mismatch")
	}
	var out Program
	if err = decode(raw, &out); err != nil {
		return Program{}, err
	}
	resolve := func(peer string) (RunID, error) {
		if peer == t.CallerPeer {
			return b.Caller, nil
		}
		id, ok := b.Members[peer]
		if !ok {
			return RunID{}, fmt.Errorf("undeclared binding peer")
		}
		return id, nil
	}
	for pc, in := range p.Code {
		isSend := in.Op == "emit" && in.Effect != nil && in.Effect.Kind == "message"
		isReceive := in.Op == "receive" && in.Match != nil && in.Match.Kind == "message" && in.Match.Provenance == "ordinary-message"
		if !isSend && !isReceive {
			continue
		}
		found := false
		for _, port := range p.Author.Ports {
			if port.Local || isSend && !port.SendUsed || isReceive && !port.ReceiveUsed {
				continue
			}
			endpoint, obligation, generation := Operand{}, Operand{}, Operand{}
			if isSend {
				endpoint, obligation, generation = in.Effect.Target, in.Effect.Obligation, in.Effect.Generation
			} else {
				endpoint, obligation, generation = in.Match.Source, in.Match.Obligation, in.Match.Generation
			}
			if !reflect.DeepEqual(endpoint, Lit(Text(port.Peer))) || !reflect.DeepEqual(obligation, Lit(Text(port.Obligation))) || !reflect.DeepEqual(generation, Lit(Int(1))) {
				continue
			}
			peer, err := resolve(port.Peer)
			if err != nil {
				return Program{}, err
			}
			if isSend {
				out.Code[pc].Effect.Target = Lit(Text(peer.Subject))
				out.Code[pc].Effect.Generation = Lit(Int(peer.Generation))
			} else {
				out.Code[pc].Match.Source = Lit(Text(peer.Subject))
				out.Code[pc].Match.Generation = Lit(Int(peer.Generation))
				if b.Protocol == compositionProtocolVersion && port.Peer == t.CallerPeer {
					out.Code[pc].Match.Provenance = "runtime-invocation-request"
					out.Code[pc].Match.Operation = Lit(Text(b.OperationID))
				}
			}
			found = true
		}
		if !found {
			return Program{}, fmt.Errorf("message endpoint lacks declared binding at %d", pc)
		}
	}
	for name, port := range out.Author.Ports {
		if port.Local {
			return Program{}, fmt.Errorf("local port in binding")
		}
		peer, err := resolve(port.Peer)
		if err != nil {
			return Program{}, err
		}
		port.Peer = peer.Subject
		out.Author.Ports[name] = port
	}
	return out, nil
}
