package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/JakeFAU/alan/internal/boundary"
	"github.com/JakeFAU/alan/runtime"
)

type caseLedger struct {
	runtime.Inspection
	parent   runtime.Projection
	attempts map[string]runtime.AdapterAttempt
	results  map[string]runtime.AdapterResult
	outcomes map[string]runtime.InvocationOutcome
}

func decodeRow[T any](t *testing.T, row map[string]any, key string) T {
	t.Helper()
	var value T
	raw, ok := row[key].(string)
	if !ok {
		t.Fatalf("expected retained bytes in %s", key)
	}
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		t.Fatal(err)
	}
	return value
}

func inspectCase(t *testing.T, db string) caseLedger {
	t.Helper()
	snap, err := runtime.Inspect(t.Context(), db, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	l := caseLedger{Inspection: snap, attempts: map[string]runtime.AdapterAttempt{}, results: map[string]runtime.AdapterResult{}, outcomes: map[string]runtime.InvocationOutcome{}}
	for _, r := range l.Tables["runs"] {
		if r["subject"] == "investigator" {
			l.parent = decodeRow[runtime.Projection](t, r, "projection")
		}
	}
	for _, r := range l.Tables["adapter_attempts"] {
		a := decodeRow[runtime.AdapterAttempt](t, r, "descriptor")
		l.attempts[a.Intent.ID] = a
	}
	for _, r := range l.Tables["adapter_results"] {
		e := decodeRow[runtime.AdapterResult](t, r, "descriptor")
		l.results[e.Candidate.ID] = e
	}
	for _, r := range l.Tables["composition_publications"] {
		o := decodeRow[runtime.InvocationOutcome](t, r, "descriptor")
		l.outcomes[o.OperationID] = o
	}
	if len(l.Roots) != 1 || l.Roots[0].Run.Subject != "investigator" {
		t.Fatal("missing original resource root")
	}
	return l
}

func (l caseLedger) model(role string) runtime.AdapterAttempt {
	for _, a := range l.attempts {
		if a.Intent.Target == "model" && a.Prepared.Request.Payload["role"].Text == role {
			return a
		}
	}
	return runtime.AdapterAttempt{}
}

func (l caseLedger) invocation(role string) string { return l.parent.Data[role+"_call"].Text }

func (l caseLedger) requirePublication(t *testing.T, e boundary.Event, role string) {
	t.Helper()
	op := l.invocation(role)
	o := l.outcomes[op]
	if e.Boundary != "invocation.result-recorded" || e.Phase != "pre-consumption" || e.Subject != "investigator" || e.Intent != op || e.Attempt == "" || e.Attempt != o.ID || o.Caller != l.Roots[0].Run || e.Root != l.Roots[0].RootID {
		t.Fatalf("wrong published %s outcome %+v, original %+v", role, e, o)
	}
	for _, r := range l.Tables["intents"] {
		if r["id"] == op && r["sequence"] == e.Sequence {
			return
		}
	}
	t.Fatal("publication acknowledgment lacks the original invocation sequence")
}

func (l caseLedger) requireResult(t *testing.T, e boundary.Event) runtime.AdapterAttempt {
	t.Helper()
	a := l.attempts[e.Intent]
	r := l.results[e.Intent]
	if e.Subject != a.Intent.Run.Subject || e.Sequence != a.Intent.Sequence || e.Attempt != a.AuthorizationID || e.Intent == "" || r.Candidate.ID != e.Intent || r.Candidate.Key != a.Prepared.Request.Key {
		t.Fatalf("uncorrelated result acknowledgment %+v", e)
	}
	return a
}

func (l caseLedger) requireSelection(t *testing.T, role string, count int) {
	t.Helper()
	op := l.invocation(role)
	id := ""
	for _, r := range l.Tables["composition_outcome_deliveries"] {
		if r["operation_id"] == op {
			id = r["observation_id"].(string)
		}
	}
	if id == "" || l.outcomes[op].ID == "" {
		t.Fatalf("original %s outcome not published", role)
	}
	selected := 0
	for _, r := range l.Tables["history"] {
		if r["selected_observation"] == id {
			if r["subject"] != "investigator" || r["generation"] != int64(1) {
				t.Fatal("outcome selected outside original caller")
			}
			selected++
		}
	}
	if selected != count {
		t.Fatalf("%s original outcome %s selected %d times, want %d", role, op, selected, count)
	}
}

func requirePreserved(t *testing.T, before, after caseLedger) {
	t.Helper()
	if before.Namespace != after.Namespace {
		t.Fatal("store namespace changed")
	}
	for _, table := range []string{"intents", "history", "adapter_attempts", "adapter_results", "composition_invocations", "composition_charges", "composition_publications", "composition_facts", "composition_outcome_deliveries"} {
		old, newRows := before.Tables[table], after.Tables[table]
		if len(old) > len(newRows) {
			t.Fatalf("lost %s", table)
		}
		for i := range old {
			if !reflect.DeepEqual(old[i], newRows[i]) {
				t.Fatalf("original %s row %d changed after recovery", table, i)
			}
		}
	}
}

// Compare the independently observed HTTP identity, request bytes and response
// with the original paid-effect reservation. A reservation alone proves no POST.
func (l caseLedger) compareContacts(t *testing.T, p *responsesFixture) {
	t.Helper()
	byID := map[string]runtime.AdapterAttempt{}
	charges := 0
	for _, a := range l.attempts {
		if !a.Allowed {
			continue
		}
		charges++
		if a.Intent.Target == "model" {
			if a.Prepared.Protocol != "openai/responses/v1" || a.Prepared.Endpoint != p.server.URL+"/v1/responses" || a.Prepared.Model != "test-model" || a.Prepared.MaxOutputTokens != 96 || a.Prepared.MaxResponseBytes != 16384 {
				t.Fatal("different original provider deployment")
			}
			id := fmt.Sprintf("alan-%x", sha256.Sum256([]byte(a.Prepared.Request.Key)))
			if _, ok := byID[id]; ok {
				t.Fatal("two reservations share a client identity")
			}
			byID[id] = a
		}
	}
	if got := l.Roots[0].Charges.Effects; got != int64(charges) || got > l.Roots[0].Limits.MaxEffects {
		t.Fatalf("durable debit %d for %d allowed reservations", got, charges)
	}
	for _, c := range p.contacts() {
		a, ok := byID[c.clientID]
		if !ok || a.Prepared.Wire != c.wire || a.Prepared.Request.ID != a.Intent.ID || a.Prepared.Request.Payload["role"].Text != c.role {
			t.Fatalf("repeated or unmatched physical %s contact %s", c.role, c.clientID)
		}
		delete(byID, c.clientID)
		r, known := l.results[a.Intent.ID]
		if !known {
			if !slices.Contains(l.Roots[0].UnknownExternal, a.Intent.ID) {
				t.Fatal("unanswered POST was lost")
			}
			continue
		}
		candidate := r.Candidate
		if candidate.ID != a.Intent.ID || candidate.Key != a.Prepared.Request.Key || candidate.RequestID != c.requestID || candidate.HTTPStatus != c.status {
			t.Fatal("HTTP result does not match original reservation or contact")
		}
		checkDiagnostic(t, c, candidate.Response)
		want := c.mode
		switch c.mode {
		case "supports", "opposes", "fabricated_quote", "no_usage":
			want = "validated"
		case "malformed":
			want = "protocol_invalid"
		}
		if r.Disposition != want || candidate.Disposition != want {
			t.Fatalf("%s: retained disposition %s, want %s", c.role, r.Disposition, want)
		}
		if c.mode == "malformed" || c.mode == "http_rejected" {
			if candidate.Usage != nil || candidate.ResponseID != "" {
				t.Fatal("manufactured metadata for unreadable/HTTP-rejected envelope")
			}
			continue
		}
		if candidate.ResponseID != "resp-"+c.role || candidate.ReturnedModel != "served-test-model" {
			t.Fatal("lost supplied provider identity")
		}
		if c.mode == "no_usage" {
			if candidate.Usage != nil {
				t.Fatal("absent usage became measured")
			}
			continue
		}
		u := candidate.Usage
		if u == nil || u.InputTokens == nil || *u.InputTokens != 17 || u.OutputTokens == nil || *u.OutputTokens != 3 || u.TotalTokens == nil || *u.TotalTokens != 20 || !strings.Contains(string(u.Details), `"reasoning_tokens":1`) {
			t.Fatal("lost actual provider usage and breakdown")
		}
	}
	for _, a := range byID {
		if r, known := l.results[a.Intent.ID]; known && r.Candidate.HTTPStatus > 0 {
			t.Fatal("retained HTTP response without independent contact")
		}
	}
}

// A killed owner leaves live WAL frames. Prove they remain byte-for-byte
// readable and unmodified through replay before any fresh owner opens the DB.
func replayStoppedCase(t *testing.T, db string, p *responsesFixture) caseLedger {
	t.Helper()
	before := retainedBytes(t, db)
	if len(before["-wal"]) <= 32 {
		t.Fatal("no surviving WAL frames at crash boundary")
	}
	replayCase(t, db, p)
	if !reflect.DeepEqual(before, retainedBytes(t, db)) {
		t.Fatal("reading stopped store changed database or WAL frames")
	}
	return inspectCase(t, db)
}

func replayCase(t *testing.T, db string, p *responsesFixture) {
	t.Helper()
	before := inspectCase(t, db)
	contents := retainedBytes(t, db)
	contacts := p.contacts()
	raw := cliOutput(t, "replay", db)
	var replay runtime.ReplayReport
	if err := json.Unmarshal(raw, &replay); err != nil {
		t.Fatal(err)
	}
	checkAnswerCommand(t, db, before, p)
	if replay.Namespace != before.Namespace || replay.HistoryCount != len(before.Tables["history"]) || replay.IntentCount != len(before.Tables["intents"]) {
		t.Fatal("replay did not reconstruct the original history")
	}
	after := inspectCase(t, db)
	if !reflect.DeepEqual(before.Inspection, after.Inspection) || !reflect.DeepEqual(contents, retainedBytes(t, db)) || !reflect.DeepEqual(contacts, p.contacts()) {
		t.Fatal("replay changed retained data, database/WAL bytes or provider contacts")
	}
}

func checkCompletedCase(t *testing.T, l caseLedger, p *responsesFixture, completion string) {
	t.Helper()
	if got := l.parent.Data["completion"].Text; l.parent.Status == "open" || got != completion || l.Roots[0].Outcome == nil {
		t.Fatalf("expected %s terminal answer, got %q root=%+v", completion, got, l.Roots[0])
	}
	if l.Roots[0].Charges.Invocations != 2 || l.Roots[0].Charges.CreatedRuns != 2 || len(l.Tables["runs"]) != 3 || len(l.outcomes) != 2 {
		t.Fatal("missing distinct worker runs and original outcomes")
	}
	for _, role := range []string{"primary", "counter"} {
		l.requireSelection(t, role, 1)
	}
	l.compareContacts(t, p)
}

// Read the shipped command's JSON as an external consumer, then check the
// visible findings and contact bounds against the source files and HTTP oracle.
func checkAnswerCommand(t *testing.T, db string, l caseLedger, p *responsesFixture) {
	t.Helper()
	var answer struct {
		Question, Status string
		Findings         []struct {
			Role   string
			Source struct {
				Document, SHA256, Quote string
				Start, End              int
			} `json:"verified_source"`
		}
		Workers []struct {
			Role  string
			Model struct {
				Operation          string `json:"operation_id"`
				State, Disposition string
				Usage              json.RawMessage `json:"provider_reported_usage"`
			}
		}
		Accounting struct {
			Contacts struct{ Minimum, Maximum int } `json:"provider_http_contacts"`
			Effects  int64                          `json:"effect_reservations_charged"`
			Tokens   int                            `json:"output_token_allowance_reserved"`
			Cost     json.RawMessage                `json:"cost_estimate_usd"`
		}
	}
	if err := json.Unmarshal(cliOutput(t, "answer", db), &answer); err != nil {
		t.Fatal(err)
	}
	status := "pending"
	if l.Roots[0].Outcome != nil {
		status = l.parent.Data["completion"].Text
	}
	if answer.Question != caseQuestion || answer.Status != status || len(answer.Workers) != 2 {
		t.Fatal("answer does not describe original parent status", answer.Status)
	}
	roles := map[string]bool{}
	for _, f := range answer.Findings {
		doc, ok := caseFacts[f.Role]
		c := f.Source
		start := strings.Index(doc.content, doc.quote)
		if !ok || roles[f.Role] || !l.parent.Data[f.Role+"_accepted"].Flag || c.Document != doc.document || c.SHA256 != fmt.Sprintf("%x", sha256.Sum256([]byte(doc.content))) || c.Start != start || c.End != start+len(doc.quote) || c.Quote != doc.quote {
			t.Fatal("wrong role or verified source bytes in answer", f)
		}
		roles[f.Role] = true
	}
	minContacts, maxContacts, tokens := 0, 0, 0
	for _, w := range answer.Workers {
		if status != "pending" && roles[w.Role] != l.parent.Data[w.Role+"_accepted"].Flag {
			t.Fatal("missing accepted finding", w.Role)
		}
		a := l.model(w.Role)
		if a.Intent.ID == "" {
			continue
		}
		if w.Model.Operation != a.Intent.ID {
			t.Fatal("answer no longer identifies model operation")
		}
		r, known := l.results[a.Intent.ID]
		if a.Allowed {
			maxContacts++
			tokens += a.Prepared.MaxOutputTokens
		}
		if r.Candidate.HTTPStatus > 0 {
			minContacts++
		}
		if !a.Allowed && w.Model.State != "dispatch_refused" || a.Allowed && !known && w.Model.State != "reserved_outcome_unknown" || known && w.Model.Disposition != r.Disposition {
			t.Fatal("incorrect model state in answer", w.Role, w.Model.State)
		}
		u := r.Candidate.Usage
		if u == nil {
			if string(w.Model.Usage) != "null" {
				t.Fatal("answer manufactured provider usage")
			}
		} else {
			var counts map[string]*int64
			if err := json.Unmarshal(w.Model.Usage, &counts); err != nil || !reflect.DeepEqual(counts["input_tokens"], u.InputTokens) || !reflect.DeepEqual(counts["output_tokens"], u.OutputTokens) || !reflect.DeepEqual(counts["total_tokens"], u.TotalTokens) {
				t.Fatal("answer substituted a provider token count")
			}
		}
	}
	b := answer.Accounting.Contacts
	count := len(p.contacts())
	if b.Minimum != minContacts || b.Maximum != maxContacts || count < b.Minimum || count > b.Maximum || answer.Accounting.Effects != l.Roots[0].Charges.Effects || answer.Accounting.Tokens != tokens || string(answer.Accounting.Cost) != "null" {
		t.Fatalf("answer accounting contradicts independent log: %+v, physical=%d", answer.Accounting, count)
	}
}
