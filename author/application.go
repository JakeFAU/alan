package author

import (
	"encoding/json"
	"fmt"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

// Instance is one statically provisioned component execution. A reusable component
// is a Go authoring function returning an Artifact for its explicitly wired ports.
type Instance struct {
	Artifact Artifact
	Inputs   map[string]schema.Value
}
type Application map[string]Instance

// Provision validates static wiring and fills generation/revision defaults. It
// derives grants only for used ports and declared effects. It does not create a DB.
func (a Application) Provision() (runtime.Provision, error) {
	p := runtime.Provision{}
	if len(a) == 0 {
		return p, fmt.Errorf("empty application")
	}
	metadata := map[string]Metadata{}
	for _, name := range keys(a) {
		instance := a[name]
		if err := schema.CheckIdentifier(name); err != nil {
			return p, fmt.Errorf("component: %w", err)
		}
		if name == "" || instance.Artifact.id == "" {
			return p, fmt.Errorf("invalid component %q", name)
		}
		m := instance.Artifact.Metadata()
		if err := m.Inputs.Record.Validate(instance.Inputs); err != nil {
			return p, fmt.Errorf("component %q inputs: %w", name, err)
		}
		metadata[name] = m
	}
	permissions := map[string]runtime.Permission{}
	programs := map[string]bool{}
	add := func(permission runtime.Permission) {
		raw, _ := json.Marshal(permission)
		permissions[string(raw)] = permission
	}
	for _, name := range keys(a) {
		instance := a[name]
		m := metadata[name]
		if !programs[instance.Artifact.id] {
			p.Programs = append(p.Programs, runtime.Artifact{ID: instance.Artifact.id, Semantics: SemanticVersion, Bytes: instance.Artifact.Bytes()})
			programs[instance.Artifact.id] = true
		}
		data := map[string]schema.Value{}
		for k, v := range instance.Inputs {
			data[k] = v
		}
		p.Subjects = append(p.Subjects, runtime.Subject{ID: name, Generation: 1, Owner: "owner:" + name, Revision: 1})
		p.Runs = append(p.Runs, runtime.InitialRun{ID: runtime.RunID{Subject: name, Generation: 1}, ProgramID: instance.Artifact.id, Projection: runtime.Projection{Status: "open", Data: data}})
		for _, portName := range keys(m.Ports) {
			port := m.Ports[portName]
			if port.Local {
				if port.ReceiveUsed {
					add(runtime.Permission{Kind: "input", Source: "local", Target: name, Operation: "input", Obligation: port.Obligation})
				}
				continue
			}
			peer, ok := metadata[port.Peer]
			if !ok {
				return p, fmt.Errorf("component %q port %q: missing peer %q", name, portName, port.Peer)
			}
			if !port.SendUsed {
				continue
			}
			matched := false
			for _, other := range peer.Ports {
				if !other.Local && other.ReceiveUsed && other.Peer == name && other.Obligation == port.Obligation && other.Receive.ID == port.Send.ID {
					matched = true
				}
			}
			if !matched {
				return p, fmt.Errorf("component %q port %q: missing compatible receive boundary", name, portName)
			}
			add(runtime.Permission{Kind: "message", Source: name, Target: port.Peer, Operation: "message", Revision: 1, Obligation: port.Obligation})
		}
		for _, effectName := range keys(m.Effects) {
			effect := m.Effects[effectName]
			add(runtime.Permission{Kind: "adapter", Source: name, Target: effect.Capability, Operation: effectName, Revision: 1, Obligation: effectName})
		}
	}
	for _, key := range keys(permissions) {
		p.Permissions = append(p.Permissions, permissions[key])
	}
	return p, nil
}
