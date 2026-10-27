package executor

import (
	"encoding/json"
	"fmt"
	"slices"
	"unicode/utf8"
)

// The complete invocation profile is public. The earlier foundation profile
// remains private and is never promoted. Descriptor versions are independent of
// program IDs; legacy stores retain schema3 without conversion.
const compositionSchemaVersion = 4
const compositionProtocolVersion = "alan/composition/v1"
const compositionProfileVersion = "alan/composition-foundation/v1"
const compositionTemplateVersion = "alan/template/v1"
const compositionBindingVersion = "alan/binding/v1"
const compositionResourceVersion = "alan/resource-profile/v1"
const compositionFactVersion = "alan/composition-fact/v1"

type compositionLimits struct {
	MaxInvocations, MaxCreatedRuns, MaxDepth, MaxEffects, MaxInFlight int64
}
type compositionRoot struct {
	ID                      string
	Version                 string
	Run                     RunID
	Members                 []RunID
	Templates, Capabilities []string
	Limits                  compositionLimits
}
type compositionTemplateMember struct {
	ProgramID string
	Initial   Projection
}
type compositionTemplate struct {
	ID, Version                        string
	Members                            map[string]compositionTemplateMember
	EntryMember, EntryPort, CallerPeer string
	Templates, Capabilities            []string
}
type compositionProvision struct {
	Version   string `json:",omitempty"`
	Roots     []compositionRoot
	Templates []compositionTemplate
}

// Store keeps only encoded profile bytes; callers cannot mutate retained slices
// or maps after construction. Decoding returns a fresh independent value.
type compositionProfile struct{ raw string }

func (p *compositionProfile) descriptor() (compositionProvision, error) {
	var c compositionProvision
	if p == nil {
		return c, fmt.Errorf("missing composition profile")
	}
	err := decode([]byte(p.raw), &c)
	return c, err
}

type compositionStaticMember struct {
	RootID string
	Run    RunID
}
type compositionBinding struct {
	Protocol                                                              string `json:",omitempty"`
	ID, Version, Namespace, InvocationID, OperationID, TemplateID, RootID string
	Caller                                                                RunID
	Depth                                                                 int64
	Members                                                               map[string]RunID
	Programs                                                              map[string]string
}

// Retained protocol envelopes are checked against their ordered source evidence.
// They carry no caller-supplied authority to create work.
type compositionInvocation struct {
	ID, OperationID, TemplateID, RootID, ParentInvocationID string
	Caller                                                  RunID
	SourceRevision, SourceSequence, Depth                   int64
	Inputs                                                  map[string]Value
}
type compositionAdmission struct {
	Invocation compositionInvocation
	Accepted   bool
	Reason     string
	BindingID  string
	Members    []RunID
}
type compositionOutcome struct {
	ID, InvocationID, OperationID, TemplateID string
	Caller                                    RunID
	Status, Reason                            string
	ResponsePresent                           bool
	Response                                  map[string]Value
	DecisionOrder                             int64
}
type compositionCharge struct {
	RootID, OperationID, Kind         string
	Invocations, CreatedRuns, Effects int64
	DecisionOrder                     int64
}
type compositionFreeze struct {
	InvocationID, Trigger                            string
	Run                                              RunID
	Sequence, DecisionOrder                          int64
	Status                                           string
	ReplyIntents, InvocationIntents, ExternalIntents []string
}

func compositionIdentity(domain string, v any) string {
	raw, _ := json.Marshal([]any{domain, v})
	return programHash(raw)
}
func compositionRootID(namespace string, run RunID) string {
	return compositionIdentity(compositionResourceVersion, []any{namespace, run})
}
func compositionInvocationID(namespace, operation string) string {
	return compositionIdentity("alan/invocation/v1", []string{namespace, operation})
}
func compositionMemberID(invocation, member string) RunID {
	return RunID{compositionIdentity("alan/invocation-member/v1", []string{invocation, member}), 1}
}
func canonicalStrings(in []string) ([]string, error) {
	out := slices.Clone(in)
	slices.Sort(out)
	for i, v := range out {
		if v == "" || i > 0 && out[i-1] == v {
			return nil, fmt.Errorf("empty or duplicate identifier")
		}
	}
	return out, nil
}

// Reject malformed identities before encoding/json can replace bytes with U+FFFD.
// Otherwise the copy boundary could silently select another valid root/member.
func validateCompositionEncoding(c compositionProvision) error {
	check := func(values ...string) error {
		for _, v := range values {
			if !utf8.ValidString(v) {
				return fmt.Errorf("invalid UTF-8 composition descriptor")
			}
		}
		return nil
	}
	for _, r := range c.Roots {
		if err := check(r.ID, r.Version, r.Run.Subject); err != nil {
			return err
		}
		if err := check(r.Templates...); err != nil {
			return err
		}
		if err := check(r.Capabilities...); err != nil {
			return err
		}
		for _, m := range r.Members {
			if err := check(m.Subject); err != nil {
				return err
			}
		}
	}
	for _, t := range c.Templates {
		if err := check(t.ID, t.Version, t.EntryMember, t.EntryPort, t.CallerPeer); err != nil {
			return err
		}
		if err := check(t.Templates...); err != nil {
			return err
		}
		if err := check(t.Capabilities...); err != nil {
			return err
		}
		for name, m := range t.Members {
			if err := check(name, m.ProgramID, m.Initial.Status); err != nil {
				return err
			}
			for key, v := range m.Initial.Data {
				if err := check(key); err != nil {
					return err
				}
				if !v.Valid() {
					return fmt.Errorf("invalid template initial value")
				}
			}
		}
	}
	return nil
}

// Public immutable provisioning descriptors for the complete composition profile.
type CompositionProvision = compositionProvision
type ResourceRoot = compositionRoot
type ResourceLimits = compositionLimits
type TemplateDescriptor = compositionTemplate
type TemplateMember = compositionTemplateMember

// RetainTemplate validates the same closed template used at store provisioning.
func RetainTemplate(t TemplateDescriptor, artifacts []Artifact) (TemplateDescriptor, error) {
	programs := map[string]Program{}
	for _, a := range artifacts {
		var p Program
		if err := decode(a.Bytes, &p); err != nil {
			return t, err
		}
		raw, id, err := EncodeProgram(p)
		if err != nil {
			return t, err
		}
		if id != a.ID || p.Semantics != a.Semantics || string(raw) != string(a.Bytes) {
			return t, ErrCorrupt
		}
		programs[id] = p
	}
	return retainTemplate(t, DefaultLimits(), func(id string) (Program, error) {
		p, ok := programs[id]
		if !ok {
			return p, fmt.Errorf("missing template artifact")
		}
		return p, nil
	})
}
