// This program is compiled and run outside the ALAN module.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/JakeFAU/alan/adapters/lookup"
	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/effects"
	"github.com/JakeFAU/alan/examples/evidence"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

func must(err error) {
	if err != nil {
		panic(err)
	}
}

type model struct{}

func (model) Prepare(r effects.Request) (effects.Prepared, error) {
	return effects.Prepared{Request: r, Protocol: "controlled/v1", MaxResponseBytes: 65536}, nil
}
func (model) Execute(_ context.Context, p effects.Prepared) effects.Candidate {
	r := p.Request
	result := map[string]schema.Value{"decision": schema.Text("lookup"), "tool": schema.Text("lookup"), "query": schema.Text("match"), "message": schema.Text("")}
	if r.Effect == "synthesize" {
		result = map[string]schema.Value{"answer": r.Payload["snippet"]}
		for _, k := range []string{"document", "version", "snippet", "start", "end"} {
			result[k] = r.Payload[k]
		}
	}
	return effects.Candidate{ID: r.ID, Key: r.Key, Disposition: "validated", Payload: result}
}
func main() {
	ctx := context.Background()
	a, err := effects.NewAsset("doc", "fixture", []byte("match first\nmatch second\nmatch third\n"))
	must(err)
	c, err := evidence.NewExcluding("helper", "customer", "job", evidence.Budgets{MaxModelRequests: 2, MaxLookupRequests: 1})
	must(err)
	request := map[string]author.Expr{"question": author.String("match")}
	for i, offset := range [][2]int64{{0, 11}, {12, 24}} {
		prefix := fmt.Sprintf("exclude%d_", i+1)
		request[prefix+"document"] = author.String(a.ID)
		request[prefix+"version"] = author.String(a.Version)
		request[prefix+"start"] = author.Integer(offset[0])
		request[prefix+"end"] = author.Integer(offset[1])
	}
	bindings := map[string]string{}
	for k := range evidence.ResultSchema() {
		bindings[k] = k
	}
	caller, err := author.Compile(author.Definition{Name: "public excluding customer", Outputs: evidence.ResultSchema(), Ports: map[string]author.Port{"helper": c.Port}, Steps: []author.Step{author.Send("helper", request), author.Receive("helper", bindings), author.Success()}})
	must(err)
	c.Instances["customer"] = author.Instance{Artifact: caller}
	p, err := c.Instances.Provision()
	must(err)
	p.Assets = []effects.Asset{a}
	dir, err := os.MkdirTemp("", "excluding-public-")
	must(err)
	defer os.RemoveAll(dir)
	path := filepath.Join(dir, "run.sqlite")
	store, err := runtime.Create(ctx, path, p, runtime.DefaultLimits())
	must(err)
	_, err = store.Run(ctx, runtime.RunOptions{MaxSteps: 1000, MaxDispatch: 1000, Adapters: map[string]effects.Adapter{evidence.ModelCapability: model{}, evidence.LookupCapability: lookup.New()}})
	must(err)
	must(store.Close())
	snapshot, err := runtime.Inspect(ctx, path, runtime.DefaultLimits())
	must(err)
	models, lookups := 0, 0
	for _, row := range snapshot.Tables["adapter_attempts"] {
		var attempt runtime.AdapterAttempt
		must(json.Unmarshal([]byte(row["descriptor"].(string)), &attempt))
		r := attempt.Prepared.Request
		if r.Capability == evidence.ModelCapability {
			models++
		}
		if r.Capability == evidence.LookupCapability {
			lookups++
			if attempt.Prepared.Protocol != "lookup/tokens-excluding/v1" || r.Payload["exclude1_start"].Number != 0 || r.Payload["exclude1_end"].Number != 11 || r.Payload["exclude2_start"].Number != 12 || r.Payload["exclude2_end"].Number != 24 || r.Payload["exclude1_document"].Text != a.ID || r.Payload["exclude2_version"].Text != a.Version {
				panic("public exclusion did not reach ordinary effect")
			}
		}
	}
	if models != 2 || lookups != 1 {
		panic("helper budget changed")
	}
	replay, err := runtime.Replay(ctx, path, runtime.DefaultLimits())
	must(err)
	for _, run := range replay.Runs {
		if run.ID.Subject == "customer" {
			data := run.Projection.Data
			if data["disposition"].Text != "answer" || data["snippet"].Text != "match third" || data["start"].Number != 25 || data["end"].Number != 36 {
				panic(fmt.Sprintf("wrong public result: %v", data))
			}
		}
	}
	fmt.Println("public exclusion reached next span")
}
