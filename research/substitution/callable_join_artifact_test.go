package substitution

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/examples/callablejoin"
	example "github.com/JakeFAU/alan/examples/substitution"
	"github.com/JakeFAU/alan/examples/substitution/a"
	"github.com/JakeFAU/alan/examples/substitution/b"
	"github.com/JakeFAU/alan/internal/executor"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

type callableSchemaCase struct {
	Payload  map[string]schema.Value
	Accepted bool
	Error    string
}

type callableArtifactRecord struct {
	CandidateID, SuccessorID, JoinID, JoinProgramID, OuterProgramID string
	Candidate, Successor, Join, JoinProgram, Outer                  json.RawMessage
	Provision                                                       runtime.Provision
	InputCases                                                      []example.Request
	FrozenTemplates                                                 map[string]bool
	InnerEnvelopes, OuterEnvelopes                                  []callableSchemaCase
}

// Only the original receive instruction is reduced here to establish actual
// schema admission and absent-field behavior. The policy oracle/evaluator is
// separate Python code and does not call this reducer or the construction policy.
func callableReceiveSchemaCase(t *testing.T, raw []byte, pc int, payload map[string]schema.Value) callableSchemaCase {
	t.Helper()
	var p executor.Program
	if err := json.Unmarshal(raw, &p); err != nil {
		t.Fatal(err)
	}
	in := p.Code[pc]
	id := executor.RunID{Subject: "schema-fixture", Generation: 1}
	s := executor.Projection{PC: pc, Status: "open", Data: map[string]schema.Value{
		"request_id": schema.Int(1), "request_value": schema.Int(0), "request_mode": schema.Text("answer"),
		in.Match.Operation.Ref: schema.Text("original-operation"),
	}}
	o := executor.Observation{ID: "schema-observation", Run: id, Kind: "invocation",
		Provenance: "runtime-invocation", Source: in.Match.Source.Literal.Text, SourceGeneration: 1,
		Obligation: in.Match.Obligation.Literal.Text, Operation: "original-operation", Payload: payload}
	r, err := executor.Reduce(p, s, &o, id, 1, executor.DefaultLimits())
	result := callableSchemaCase{Payload: payload, Accepted: err == nil}
	if err != nil {
		result.Error = err.Error()
		return result
	}
	if !payload["response_present"].Flag {
		for field, dst := range in.Bind {
			if len(field) > 9 && field[:9] == "response." {
				if _, exists := r.Projection.Data[dst]; exists {
					t.Fatalf("absent field was invented: %s", dst)
				}
			}
		}
	}
	return result
}

func callableEnvelopePayload(status, reason string, response map[string]schema.Value) map[string]schema.Value {
	p := map[string]schema.Value{"status": schema.Text(status), "reason": schema.Text(reason),
		"response_present": schema.Bool(response != nil)}
	for field, value := range response {
		p["response."+field] = value
	}
	return p
}

func TestCallableJoinArtifactCorrespondenceInputs(t *testing.T) {
	records := map[string]callableArtifactRecord{}
	for _, candidate := range []struct {
		name  string
		build func() (author.Template, error)
	}{{name: "a", build: a.New}, {name: "b", build: b.New}} {
		t.Run(candidate.name, func(t *testing.T) {
			template, err := candidate.build()
			if err != nil {
				t.Fatal(err)
			}
			config := example.Config{Request: example.Request{ID: 1, Value: 0, Mode: "answer"},
				Limits: runtime.ResourceLimits{MaxInvocations: 4, MaxCreatedRuns: 4, MaxDepth: 3, MaxInFlight: 1}}
			d, err := callablejoin.New(template, config)
			if err != nil {
				t.Fatal(err)
			}
			r := callableArtifactRecord{CandidateID: template.ID(), SuccessorID: d.Successor.ID(), JoinID: d.Join.ID(),
				JoinProgramID: d.JoinArtifact.ID(), OuterProgramID: d.Outer.ID(), Candidate: template.Bytes(),
				Successor: d.Successor.Bytes(), Join: d.Join.Bytes(), JoinProgram: d.JoinArtifact.Bytes(),
				Outer: d.Outer.Bytes(), Provision: d.Provision, InputCases: []example.Request{},
				FrozenTemplates: map[string]bool{}, InnerEnvelopes: []callableSchemaCase{}, OuterEnvelopes: []callableSchemaCase{}}
			for _, c := range CallableJoinConfigurations() {
				otherConfig := config
				otherConfig.Request = example.Request{ID: int64(c.Request.ID), Value: int64(c.Request.Value), Mode: c.Request.Mode}
				other, err := callablejoin.New(template, otherConfig)
				if err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(d.Join.Bytes(), other.Join.Bytes()) || !bytes.Equal(d.Outer.Bytes(), other.Outer.Bytes()) {
					t.Fatal("input changed compiled policy identity")
				}
				data := other.Provision.Runs[0].Projection.Data
				if len(other.Provision.Runs) != 1 || len(data) != 3 || data["request_id"].Number != int64(c.Request.ID) ||
					data["request_value"].Number != int64(c.Request.Value) || data["request_mode"].Text != c.Request.Mode {
					t.Fatal("wrong original input or fixed topology")
				}
				other.Provision.Runs[0].Projection.Data = d.Provision.Runs[0].Projection.Data
				if !reflect.DeepEqual(d.Provision, other.Provision) {
					t.Fatal("input changed non-input deployment")
				}
				r.InputCases = append(r.InputCases, otherConfig.Request)
			}
			for role, raw := range map[string][]byte{"candidate": template.Bytes(), "s": d.Successor.Bytes()} {
				path := filepath.Join("..", "..", "docs/foundational/phase1k/evidence/context/frozen", candidate.name+"-"+role+"-template.json")
				old, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				r.FrozenTemplates[role] = bytes.Equal(raw, old)
				if !r.FrozenTemplates[role] {
					t.Fatal("original template changed: " + role)
				}
			}
			for _, p := range d.Provision.Programs {
				if fmt.Sprintf("%x", sha256.Sum256(p.Bytes)) != p.ID {
					t.Fatal("program byte identity mismatch")
				}
			}
			child := []map[string]schema.Value{}
			reports := []map[string]schema.Value{}
			for id := int64(0); id <= 2; id++ {
				for value := int64(0); value <= 1; value++ {
					child = append(child, map[string]schema.Value{"id": schema.Int(id), "value": schema.Int(value)})
				}
			}
			for id := int64(0); id <= 2; id++ {
				for _, tag := range []string{ReasonAnswer, ReasonFailed, ReasonMissingReply, ReasonInvalid} {
					lists := []schema.Value{schema.List()}
					for _, row := range child {
						lists = append(lists, schema.List(row))
					}
					for _, list := range lists {
						report := map[string]schema.Value{"id": schema.Int(id), "classification": schema.Text(tag), "answers": list}
						if err := callablejoin.Report().Validate(report); err != nil {
							t.Fatal(err)
						}
						reports = append(reports, report)
					}
				}
			}
			for _, status := range []string{"succeeded", "failed", "rejected", "cancelled", "abandoned"} {
				for _, reason := range []string{"member_succeeded", "member_failed", "missing_reply", "invocation_limit"} {
					absent := callableEnvelopePayload(status, reason, nil)
					r.InnerEnvelopes = append(r.InnerEnvelopes, callableReceiveSchemaCase(t, d.JoinArtifact.Bytes(), 5, absent))
					r.OuterEnvelopes = append(r.OuterEnvelopes, callableReceiveSchemaCase(t, d.Outer.Bytes(), 1, absent))
					for _, response := range child {
						r.InnerEnvelopes = append(r.InnerEnvelopes, callableReceiveSchemaCase(t, d.JoinArtifact.Bytes(), 5,
							callableEnvelopePayload(status, reason, response)))
					}
					for _, response := range reports {
						r.OuterEnvelopes = append(r.OuterEnvelopes, callableReceiveSchemaCase(t, d.Outer.Bytes(), 1,
							callableEnvelopePayload(status, reason, response)))
					}
				}
			}
			for _, response := range []map[string]schema.Value{
				{"id": schema.Int(1), "classification": schema.Text("unknown"), "answers": schema.List()},
				{"id": schema.Int(1), "classification": schema.Text(ReasonAnswer), "answers": schema.List(child[0], child[1])},
				{"id": schema.Int(1), "classification": schema.Text(ReasonAnswer)},
			} {
				bad := callableReceiveSchemaCase(t, d.Outer.Bytes(), 1, callableEnvelopePayload("failed", "member_failed", response))
				if bad.Accepted {
					t.Fatal("schema-invalid report accepted")
				}
				r.OuterEnvelopes = append(r.OuterEnvelopes, bad)
			}
			records[candidate.name] = r
			t.Logf("inputs=%d inner_schema_cases=%d outer_schema_cases=%d", len(r.InputCases), len(r.InnerEnvelopes), len(r.OuterEnvelopes))
		})
	}
	emit(t, "callable-join-artifacts", records)
}
