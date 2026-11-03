package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/JakeFAU/alan/adapters/lookup"
	"github.com/JakeFAU/alan/adapters/openai"
	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/effects"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

func priorDeployment(t *testing.T, endpoint string, queries [2]string) (runtime.Provision, deployment) {
	t.Helper()
	d := deployment{
		Provider: providerConfig{Model: "approved", Endpoint: endpoint + "/v1/responses", MaxOutputTokens: 96, MaxResponseBytes: 4096},
		Limits:   spendingLimits{MaxEffects: 4, MaxInFlight: 2, MaxTotalOutputTokens: 192},
	}
	settings, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	p, err := investigatorProvision("Were votes counted?", queries, string(settings), d.resources())
	if err != nil {
		t.Fatal(err)
	}
	doc, err := effects.NewAsset("minutes", "minutes.txt", []byte("Alpha votes were counted."))
	if err != nil {
		t.Fatal(err)
	}
	p.Assets = []effects.Asset{doc}
	return p, d
}

// A prior public-API owner chooses its own adapter. Stop before a second
// reservation so that an unguarded resume could still send another request.
type firstModelOnly struct {
	*openai.Adapter
	firstPrepared bool
	wireMismatch  bool
}

func (m *firstModelOnly) Prepare(r effects.Request) (effects.Prepared, error) {
	if m.firstPrepared {
		return effects.Prepared{}, errors.New("prior owner stops before second model reservation")
	}
	m.firstPrepared = true
	p, err := m.Adapter.Prepare(r)
	if err == nil && m.wireMismatch {
		p.Wire = strings.Replace(p.Wire, `"max_output_tokens":96`, `"max_output_tokens":256`, 1)
		p = effects.Freeze(p)
	}
	return p, err
}

func TestPriorModelDeploymentMustMatchBeforeResumeAndAnswer(t *testing.T) {
	for _, scenario := range []string{"larger allowance", "different model", "different endpoint", "different response bound", "wire only", "unknown wire"} {
		t.Run(scenario, func(t *testing.T) {
			var contacts atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				contacts.Add(1)
				var body struct {
					MaxOutputTokens int `json:"max_output_tokens"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				if contacts.Load() == 1 && (scenario == "wire only" || scenario == "unknown wire" || scenario == "larger allowance") && body.MaxOutputTokens != 256 {
					t.Errorf("wrong prior physical allowance %d", body.MaxOutputTokens)
				}
				if scenario == "unknown wire" {
					conn, _, err := w.(http.Hijacker).Hijack()
					if err != nil {
						t.Error(err)
						return
					}
					_ = conn.Close()
					return
				}
				_, response := fixtureResponse("primary", "supports", "Alpha votes were counted.")
				_, _ = io.WriteString(w, response)
			}))
			defer server.Close()
			p, approved := priorDeployment(t, server.URL, [2]string{"alpha", "counted"})
			previous := approved.Provider
			switch scenario {
			case "larger allowance":
				previous.MaxOutputTokens = 256
			case "different model":
				previous.Model = "prior-model"
			case "different endpoint":
				previous.Endpoint = server.URL + "/alternate"
			case "different response bound":
				previous.MaxResponseBytes *= 2
			}
			model, err := previous.adapter(controlledCredential)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(t.TempDir(), "mixed.sqlite")
			limits := runtime.DefaultLimits()
			store, err := runtime.Create(t.Context(), path, p, limits)
			if err != nil {
				t.Fatal(err)
			}
			var approvedNamespace string
			if scenario == "unknown wire" {
				initial, err := runtime.Inspect(t.Context(), path, limits)
				if err != nil {
					t.Fatal(err)
				}
				got, err := approvedDeployment(initial)
				if err != nil || got != approved {
					t.Fatalf("original preflight: %v", err)
				}
				approvedNamespace = initial.Namespace
			}
			_, runErr := store.Run(t.Context(), runtime.RunOptions{
				MaxSteps: 256, MaxDispatch: 128, MaxInFlight: 1,
				Adapters: map[string]effects.Adapter{"corpus": lookup.New(), "model": &firstModelOnly{
					Adapter: model, wireMismatch: scenario == "wire only" || scenario == "unknown wire",
				}},
			})
			if err := store.Close(); err != nil {
				t.Fatal(err)
			}
			if runErr == nil || contacts.Load() != 1 {
				t.Fatalf("prior owner: %v, contacts %d", runErr, contacts.Load())
			}
			replayed, err := runtime.Replay(t.Context(), path, limits)
			if err != nil {
				t.Fatal(err)
			}
			if got := len(replayed.Roots[0].UnknownExternal); (scenario == "unknown wire") != (got == 1) {
				t.Fatalf("prior unknown effects: %d", got)
			}
			before := retainedBytes(t, path)
			for _, action := range []string{"answer", "resume"} {
				var out bytes.Buffer
				if err := run(t.Context(), []string{action, path}, &out); err == nil || out.Len() != 0 {
					t.Errorf("%s accepted foreign historical reservation: %v", action, err)
				}
			}
			if contacts.Load() != 1 || !reflect.DeepEqual(before, retainedBytes(t, path)) {
				t.Errorf("rejecting prior deployment changed store bytes or issued a new POST (contacts %d)", contacts.Load())
			}
			if approvedNamespace != "" {
				// Reproduce the interval between a valid preflight and fresh ownership.
				// Returning from Open may renew epochs. No execution step or dispatch
				// may cross the owner-side check against the newer reservation.
				valid, err := approved.Provider.adapter(controlledCredential)
				if err != nil {
					t.Fatal(err)
				}
				err = approved.runOwned(t.Context(), path, approvedNamespace, proposalAdapter{valid}, io.Discard)
				if err == nil || contacts.Load() != 1 {
					t.Fatalf("stale preflight authorized a dispatch: %v", err)
				}
				after, err := runtime.Replay(t.Context(), path, limits)
				if err != nil || after.HistoryCount != replayed.HistoryCount || after.IntentCount != replayed.IntentCount ||
					!reflect.DeepEqual(after.Roots, replayed.Roots) {
					t.Fatalf("stale approval changed durable execution or accounting: %v", err)
				}
			}
		})
	}
}

func TestAlternateParentCannotPublishAnInvestigatorAnswer(t *testing.T) {
	p, _ := priorDeployment(t, "http://127.0.0.1:1", [2]string{"absent", "unmatched"})
	worker, err := researchTemplate()
	if err != nil {
		t.Fatal(err)
	}
	emptyList := schema.Field{Kind: "list", Element: schema.Record{"unused": textField(1)}, MaxBytes: 512, MaxItems: 1}
	d := author.Definition{
		Name: "investigator", Invocations: map[string]author.Template{"research": worker},
		Inputs:      schema.Record{"question": textField(4096), "primary_query": textField(512), "counter_query": textField(512), "settings": textField(4096)},
		Collections: map[string]schema.Field{"findings": emptyList, "gaps": emptyList},
	}
	for _, role := range []string{"primary", "counter"} {
		d.Steps = append(d.Steps, author.Invoke(role+"_call", "research", map[string]author.Expr{
			"role": author.String(role), "query": author.Var(role + "_query"), "question": author.Var("question"),
		}, nil))
	}
	for _, role := range []string{"primary", "counter"} {
		d.Steps = append(d.Steps, author.AwaitInvocation("research", role+"_call", author.InvocationOutcomeBindings{
			Status: role + "_status", Reason: role + "_reason", ResponsePresent: role + "_present", Response: fieldBindings(researchReply(), role+"_"),
		}))
	}
	// Both lookups failed, but this alternate parent always chooses completed.
	d.Steps = append(d.Steps, author.Set("completion", author.String("completed")), author.Success())
	fake, err := author.Compile(d)
	if err != nil {
		t.Fatal(err)
	}
	other, err := (author.Application{"investigator": {Artifact: fake, Inputs: p.Runs[0].Projection.Data}}).Provision()
	if err != nil {
		t.Fatal(err)
	}
	other.Assets, other.Composition.Roots = p.Assets, p.Composition.Roots
	path := filepath.Join(t.TempDir(), "false-answer.sqlite")
	store, err := runtime.Create(t.Context(), path, other, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	_, runErr := store.Run(t.Context(), runtime.RunOptions{MaxSteps: 256, MaxDispatch: 128, Adapters: map[string]effects.Adapter{"corpus": lookup.New()}})
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	if runErr != nil {
		t.Fatal(runErr)
	}
	history, err := runtime.Replay(t.Context(), path, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	var falseClaim bool
	for _, run := range history.Runs {
		if run.ID.Subject == "investigator" {
			v := run.Projection.Data
			falseClaim = v["completion"].Text == "completed" && v["primary_status"].Text == "failed" && v["counter_status"].Text == "failed"
		}
	}
	if !falseClaim {
		t.Fatal("foreign parent failed to produce its replay-consistent false completion")
	}
	var out bytes.Buffer
	if err := run(t.Context(), []string{"answer", path}, &out); err == nil || out.Len() != 0 {
		t.Fatalf("answer accepted foreign parent: %v, %s", err, out.String())
	}
}

// The genuine provider adapter validates the schema. A previous public-API
// owner can use it directly, without the investigator's additional text check.
func TestPriorProviderMustApplyLocalProposalRules(t *testing.T) {
	for _, tc := range []struct {
		name, verdict, interpretation, question string
		pending                                 bool
	}{
		{"completed words missing", "supports", "!!!", "", false},
		{"pending malformed follow-up", "insufficient", "The record is inconclusive.", "???", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var posts atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				posts.Add(1)
				proposal, err := json.Marshal(map[string]string{
					"verdict": tc.verdict, "quote": "Alpha votes were counted.",
					"interpretation": tc.interpretation, "unresolved": tc.question,
				})
				if err != nil {
					t.Error(err)
					return
				}
				message := map[string]any{"type": "message", "role": "assistant",
					"content": []any{map[string]string{"type": "output_text", "text": string(proposal)}}}
				if err := json.NewEncoder(w).Encode(map[string]any{"status": "completed", "output": []any{message}}); err != nil {
					t.Error(err)
				}
			}))
			defer server.Close()
			p, d := priorDeployment(t, server.URL, [2]string{"alpha", "counted"})
			model, err := d.Provider.adapter(controlledCredential)
			if err != nil {
				t.Fatal(err)
			}
			var prior effects.Adapter = model
			if tc.pending {
				prior = &firstModelOnly{Adapter: model}
			}
			path := filepath.Join(t.TempDir(), "prior.sqlite")
			limits := runtime.DefaultLimits()
			store, err := runtime.Create(t.Context(), path, p, limits)
			if err != nil {
				t.Fatal(err)
			}
			_, runErr := store.Run(t.Context(), runtime.RunOptions{MaxSteps: 256, MaxDispatch: 128, MaxInFlight: 1,
				Adapters: map[string]effects.Adapter{"corpus": lookup.New(), "model": prior}})
			if err := store.Close(); err != nil {
				t.Fatal(err)
			}
			if (runErr != nil) != tc.pending {
				t.Fatalf("wrong setup: %v", runErr)
			}
			contacts := posts.Load()
			if (tc.pending && contacts != 1) || (!tc.pending && contacts != 2) {
				t.Fatalf("wrong original contacts %d", contacts)
			}
			before, err := runtime.Inspect(t.Context(), path, limits)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := runtime.Replay(t.Context(), path, limits); err != nil {
				t.Fatal("valid ALAN history", err)
			}
			for _, action := range []string{"answer", "resume"} {
				var out bytes.Buffer
				err := run(t.Context(), []string{action, path}, &out)
				if err == nil || !strings.Contains(err.Error(), "proposal") || out.Len() != 0 {
					t.Fatalf("%s accepted or misidentified the prior result: %v (output %d bytes)", action, err, out.Len())
				}
			}
			after, err := runtime.Inspect(t.Context(), path, limits)
			if err != nil || !reflect.DeepEqual(before, after) || posts.Load() != contacts {
				t.Fatalf("changed a valid historical result or issued a new request: %v", err)
			}
		})
	}
}
