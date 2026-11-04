package delegation

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/url"

	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

// Scenario fixes an ordinary retained reciprocal-service program.
type Scenario string

const (
	S1Own       Scenario = "s1-own"
	S2Handshake Scenario = "s2-handshake"
	S3Missing   Scenario = "s3-missing"
	S3Claim     Scenario = "s3-claim"
)

// Options contains trusted deployment endpoints and the finite program policy.
type Options struct {
	EndpointA, EndpointB string
	Scenario             Scenario
	TerminalHandshake    bool
}

// Manifest records immutable original locators; these strings confer no authority.
// Changes to instruction order require changing and requalifying these locators.
type Manifest struct {
	Version                                                                    string
	Options                                                                    Options
	A, B                                                                       runtime.RunID
	ABRequest, BARequest, Grant, Ready, BAService, BAReply, ABService, ABReply string
	BAServiceRoot, ABServiceRoot                                               runtime.AuthorityRoot
	Claim                                                                      runtime.DelegationClaim
}

// The public artifact route accepts canonical retained-program JSON. These local
// wire structs intentionally avoid importing the private executor AST.
type operand struct {
	Ref     string
	Literal *schema.Value
	Offset  *int64
}
type match struct {
	Kind, Provenance                          string
	Source, Generation, Obligation, Operation operand
}
type effect struct {
	Authority                      *runtime.AuthorityUse `json:",omitempty"`
	Kind                           string
	Target, Generation, Obligation operand
	Payload                        map[string]operand
}
type instruction struct {
	Op, Dst           string
	A, B              operand
	Target, Otherwise int
	Match             *match
	Bind              map[string]string
	Effect            *effect
	Updates           map[string]operand
	Outcome           string
}
type program struct {
	Semantics string
	Code      []instruction
}

func lit(v schema.Value) operand { return operand{Literal: &v} }
func str(v string) operand       { return lit(schema.Text(v)) }
func ref(v string) operand       { return operand{Ref: v} }
func locator(subject string, sequence int) string {
	raw, _ := json.Marshal([]any{subject, 1, sequence, 0})
	return string(raw)
}
func send(kind, target, obligation, dst string, payload map[string]operand) instruction {
	return instruction{Op: "emit", Dst: dst, Effect: &effect{Kind: kind, Target: str(target), Generation: lit(schema.Int(1)), Obligation: str(obligation), Payload: payload}}
}
func receive(kind, provenance, source, obligation string, operation operand, bind map[string]string) instruction {
	return instruction{Op: "receive", Match: &match{Kind: kind, Provenance: provenance, Source: str(source), Generation: lit(schema.Int(1)), Obligation: str(obligation), Operation: operation}, Bind: bind}
}
func branch(value string, yes, no int) instruction {
	return instruction{Op: "branch", A: ref(value), B: lit(schema.Bool(true)), Target: yes, Otherwise: no}
}
func endpointValid(v string) bool {
	u, e := url.Parse(v)
	return e == nil && u.Scheme == "http" && u.User == nil && u.RawQuery == "" && u.Fragment == "" && net.ParseIP(u.Hostname()) != nil && net.ParseIP(u.Hostname()).IsLoopback()
}

// Reciprocal constructs exactly two generation-one principals. B issues its grant
// through an ordinary lifecycle instruction and waits for acceptance before ready.
// A services while its own request is unpaid; B services only after the BA reply.
// S1 uses own authority with an ordinary ready handshake and no delegation.
// S3 variants retain the S2 grant but omit or falsify A's selected authority.
func Reciprocal(o Options) (runtime.Provision, Manifest, error) {
	p := runtime.Provision{}
	m := Manifest{Version: "alan/reciprocal-fixture/v1", Options: o, A: runtime.RunID{Subject: "a", Generation: 1}, B: runtime.RunID{Subject: "b", Generation: 1}, ABRequest: locator("a", 1), BARequest: locator("b", 1), Grant: locator("b", 2), Ready: locator("b", 5), BAService: locator("a", 4), BAReply: locator("a", 7), ABService: locator("b", 9), ABReply: locator("b", 12)}
	if !endpointValid(o.EndpointA) || !endpointValid(o.EndpointB) || o.EndpointA == o.EndpointB {
		return p, m, fmt.Errorf("two distinct literal loopback HTTP endpoints required")
	}
	switch o.Scenario {
	case S1Own, S2Handshake, S3Missing, S3Claim:
	default:
		return p, m, fmt.Errorf("unknown reciprocal scenario %q", o.Scenario)
	}
	root := func(run runtime.RunID, target, obligation string) runtime.AuthorityRoot {
		return runtime.AuthorityRoot{Run: run, RootRun: run, Permission: runtime.Permission{Kind: "external", Source: run.Subject, Target: target, Operation: "apply", Revision: 1, Obligation: obligation}}
	}
	delegated := root(m.B, o.EndpointB, "ba-service")
	m.BAServiceRoot = delegated
	m.ABServiceRoot = root(m.B, o.EndpointA, "ab-service")
	m.Claim = runtime.DelegationClaim{Version: runtime.RequestAuthorityVersion, Root: delegated, Recipient: m.A, RecipientRoot: m.A, RecipientRevision: 1, OriginalRequestID: m.BARequest, ServiceOperationID: m.BAService}
	claim, err := json.Marshal(m.Claim)
	if err != nil {
		return p, m, err
	}
	use := &runtime.AuthorityUse{Version: runtime.RequestAuthorityVersion, Path: "delegated", Root: delegated, GrantID: m.Grant}
	p.Permissions = []runtime.Permission{delegated.Permission, m.ABServiceRoot.Permission}
	if o.Scenario == S1Own {
		m.BAServiceRoot = root(m.A, o.EndpointB, "ba-service")
		use = &runtime.AuthorityUse{Version: runtime.RequestAuthorityVersion, Path: "own", Root: m.BAServiceRoot}
		p.Permissions = append(p.Permissions, m.BAServiceRoot.Permission)
	}
	if o.Scenario == S3Missing {
		use = nil
	}
	if o.Scenario == S3Claim {
		use.GrantID = m.BARequest
	}
	service := func(target, obligation string, use *runtime.AuthorityUse) instruction {
		in := send("external", target, obligation, "service_operation", map[string]operand{"profile": str("idempotent"), "delta": lit(schema.Int(1))})
		in.Effect.Authority = use
		return in
	}
	result := func(endpoint, obligation string) instruction {
		return receive("receipt", "resource-receipt", endpoint, obligation, ref("service_operation"), map[string]string{"accepted": "service_accepted", "disposition": "disposition", "mutation": "mutation", "total": "total"})
	}
	reply := func(target, obligation, request string) instruction {
		return send("message", target, obligation, "reply", map[string]operand{"accepted": ref("service_accepted"), "request_id": str(request), "service_operation": ref("service_operation"), "mutation": ref("mutation"), "total": ref("total")})
	}
	replyReceive := func(source, obligation string) instruction {
		return receive("message", "ordinary-message", source, obligation, str(""), map[string]string{"accepted": "paid", "request_id": "reply_request", "service_operation": "reply_service", "mutation": "reply_mutation", "total": "reply_total"})
	}
	a := []instruction{
		send("message", "b", "ab-request", "ab_request", map[string]operand{"request_id": str(m.ABRequest), "claim": str(string(claim))}),
		receive("message", "ordinary-message", "b", "grant-ready", str(""), map[string]string{"grant": "grant_data"}),
		receive("message", "ordinary-message", "b", "ba-request", str(""), map[string]string{"request_id": "peer_request", "claim": "peer_claim"}),
		service(o.EndpointB, "ba-service", use), result(o.EndpointB, "ba-service"), branch("service_accepted", 6, 10), reply("b", "ba-reply", m.BARequest), replyReceive("b", "ab-reply"), branch("paid", 9, 10), {Op: "end", Outcome: "succeeded"}, {Op: "end", Outcome: "failed"},
	}
	b := []instruction{
		send("message", "a", "ba-request", "ba_request", map[string]operand{"request_id": str(m.BARequest), "claim": str(string(claim))}),
		send("lifecycle", "a", "delegation-result", "grant", map[string]operand{"operation": str("DelegateAuthority"), "revision": lit(schema.Int(1)), "claim": str(string(claim))}),
		receive("lifecycle", "runtime-lifecycle", "a", "delegation-result", ref("grant"), map[string]string{"accepted": "grant_accepted"}), branch("grant_accepted", 4, 14), send("message", "a", "grant-ready", "ready", map[string]operand{"grant": ref("grant")}), replyReceive("a", "ba-reply"), branch("paid", 7, 14), receive("message", "ordinary-message", "a", "ab-request", str(""), map[string]string{"request_id": "peer_request", "claim": "peer_claim"}), service(o.EndpointA, "ab-service", &runtime.AuthorityUse{Version: runtime.RequestAuthorityVersion, Path: "own", Root: m.ABServiceRoot}), result(o.EndpointA, "ab-service"), branch("service_accepted", 11, 14), reply("a", "ab-reply", m.ABRequest), receive("terminal", "runtime-terminal", "a", "a-complete", str(""), map[string]string{"outcome": "peer_outcome"}), {Op: "end", Outcome: "succeeded"}, {Op: "end", Outcome: "failed"},
	}
	if o.Scenario == S1Own {
		m.Grant = ""
		b[1] = instruction{Op: "set", Dst: "grant", A: str("")}
		b[2] = instruction{Op: "set", Dst: "grant_accepted", A: lit(schema.Bool(true))}
	}
	if !o.TerminalHandshake {
		b[12] = instruction{Op: "end", Outcome: "succeeded"}
	}
	for _, r := range []struct {
		subject string
		code    []instruction
	}{{"a", a}, {"b", b}} {
		bytes, err := json.Marshal(program{Semantics: "alan/1d/v1", Code: r.code})
		if err != nil {
			return p, m, err
		}
		hash := sha256.Sum256(bytes)
		id := hex.EncodeToString(hash[:])
		p.Programs = append(p.Programs, runtime.Artifact{ID: id, Semantics: "alan/1d/v1", Bytes: bytes})
		p.Subjects = append(p.Subjects, runtime.Subject{ID: r.subject, Generation: 1, Owner: "reciprocal:" + r.subject, Revision: 1})
		data := map[string]schema.Value{"paid": schema.Bool(false), "service_accepted": schema.Bool(false), "grant_accepted": schema.Bool(false)}
		p.Runs = append(p.Runs, runtime.InitialRun{ID: runtime.RunID{Subject: r.subject, Generation: 1}, ProgramID: id, Projection: runtime.Projection{Status: "open", Data: data}})
	}
	for _, v := range [][3]string{{"a", "b", "ab-request"}, {"b", "a", "ba-request"}, {"b", "a", "grant-ready"}, {"a", "b", "ba-reply"}, {"b", "a", "ab-reply"}} {
		p.Permissions = append(p.Permissions, runtime.Permission{Kind: "message", Source: v[0], Target: v[1], Operation: "message", Revision: 1, Obligation: v[2]})
	}
	if o.TerminalHandshake {
		p.Permissions = append(p.Permissions, runtime.Permission{Kind: "observation", Source: "a", Target: "b", Operation: "terminal", Revision: 1, Obligation: "a-complete"})
	}
	p.Composition = &runtime.CompositionProvision{RequestAuthority: runtime.RequestAuthorityVersion, Roots: []runtime.ResourceRoot{{Run: m.A, Members: []runtime.RunID{m.A}, Limits: runtime.ResourceLimits{MaxEffects: 1, MaxInFlight: 1}}, {Run: m.B, Members: []runtime.RunID{m.B}, Limits: runtime.ResourceLimits{MaxEffects: 1, MaxInFlight: 1}}}}
	return p, m, nil
}
