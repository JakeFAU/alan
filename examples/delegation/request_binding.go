package delegation

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

// RequestBindingOptions selects a finite, statically located service path.
// ForwardBranch executes three local reductions and excludes LocalSteps.
type RequestBindingOptions struct {
	EndpointA, EndpointB string
	LocalSteps           int
	ForwardBranch        *bool
	ReadyFirst           bool
	TerminalHandshake    bool
}

// RequestBindingManifest records actual program hashes and planned locators.
// A planned locator is not evidence that the corresponding service occurred.
type RequestBindingManifest struct {
	Version                        string
	Options                        RequestBindingOptions
	Reciprocal                     Manifest
	ServiceSequence, ReplySequence int64
	InsertedReductions             int
	ProgramIDs                     map[string]string
	PrincipalRequests              [2]string
}

// RequestBinding constructs ordinary v2 programs with exact earlier selection.
// The optional local work changes only scratch state; B issues the grant through
// its retained lifecycle program. Runtime authority is never supplied by this host.
func RequestBinding(o RequestBindingOptions) (runtime.Provision, RequestBindingManifest, error) {
	m := RequestBindingManifest{Version: "alan/request-binding-fixture/v1", Options: o}
	if o.LocalSteps < 0 || o.LocalSteps > 32 || o.ForwardBranch != nil && o.LocalSteps != 0 {
		return runtime.Provision{}, m, fmt.Errorf("LocalSteps must be 0..32 and zero with ForwardBranch")
	}
	if o.ForwardBranch != nil {
		v := *o.ForwardBranch
		m.Options.ForwardBranch = &v
	}
	p, inner, err := Reciprocal(Options{EndpointA: o.EndpointA, EndpointB: o.EndpointB, Scenario: S2Handshake, TerminalHandshake: o.TerminalHandshake})
	if err != nil {
		return p, m, err
	}
	n := o.LocalSteps
	if o.ForwardBranch != nil {
		n = 3
	}
	m.InsertedReductions = n
	m.ServiceSequence = int64(4 + n)
	m.ReplySequence = int64(7 + n)
	inner.BAService = locator("a", 4+n)
	inner.BAReply = locator("a", 7+n)
	inner.Claim.Version = runtime.RequestAuthorityBindingVersion
	inner.Claim.ServiceOperationID = inner.BAService
	claim, err := json.Marshal(inner.Claim)
	if err != nil {
		return p, m, err
	}
	m.Reciprocal = inner
	m.PrincipalRequests = [2]string{inner.ABRequest, inner.BARequest}
	m.ProgramIDs = map[string]string{}
	for ri := range p.Runs {
		run := &p.Runs[ri]
		for ai := range p.Programs {
			artifact := &p.Programs[ai]
			if artifact.ID != run.ProgramID {
				continue
			}
			var program program
			if err = json.Unmarshal(artifact.Bytes, &program); err != nil {
				return p, m, err
			}
			for i := range program.Code {
				e := program.Code[i].Effect
				if e == nil {
					continue
				}
				if e.Authority != nil {
					e.Authority.Version = runtime.RequestAuthorityBindingVersion
				}
				if _, ok := e.Payload["claim"]; ok {
					e.Payload["claim"] = str(string(claim))
				}
			}
			if run.ID.Subject == "a" {
				if !o.ReadyFirst {
					program.Code[1], program.Code[2] = program.Code[2], program.Code[1]
				}
				var inserted []instruction
				if o.ForwardBranch != nil {
					inserted = []instruction{
						{Op: "branch", A: lit(schema.Bool(*o.ForwardBranch)), B: lit(schema.Bool(true)), Target: 4, Otherwise: 6},
						{Op: "set", Dst: "scratch", A: lit(schema.Int(1))},
						{Op: "branch", A: lit(schema.Bool(true)), B: lit(schema.Bool(true)), Target: 8, Otherwise: 8},
						{Op: "set", Dst: "scratch", A: lit(schema.Int(2))},
						{Op: "set", Dst: "scratch", A: ref("scratch")},
					}
				} else {
					for i := 0; i < n; i++ {
						a := lit(schema.Int(int64(i + 1)))
						if i%2 == 1 {
							a = ref("scratch")
						}
						inserted = append(inserted, instruction{Op: "set", Dst: "scratch", A: a})
					}
				}
				for i := 3; i < len(program.Code); i++ {
					in := &program.Code[i]
					if in.Op == "branch" {
						in.Target += len(inserted)
						in.Otherwise += len(inserted)
					}
				}
				code := append([]instruction{}, program.Code[:3]...)
				code = append(code, inserted...)
				program.Code = append(code, program.Code[3:]...)
			}
			raw, err := json.Marshal(program)
			if err != nil {
				return p, m, err
			}
			sum := sha256.Sum256(raw)
			id := hex.EncodeToString(sum[:])
			artifact.ID = id
			artifact.Bytes = raw
			run.ProgramID = id
			m.ProgramIDs[run.ID.Subject] = id
			break
		}
	}
	p.Composition.RequestAuthority = runtime.RequestAuthorityBindingVersion
	return p, m, nil
}
