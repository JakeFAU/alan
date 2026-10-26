package evidence

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/JakeFAU/alan/adapters/lookup"
	"github.com/JakeFAU/alan/adapters/openai"
	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/effects"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
	"time"
)

//go:embed corpus.txt
var bundledCorpus []byte

// Config holds only non-secret selected settings, retained as ordinary caller
// input. API keys are supplied exclusively to Run and never serialized here.
type Config struct {
	Model, Endpoint                     string
	MaxModelRequests, MaxLookupRequests int
	MaxOutputTokens, MaxResponseBytes   int
	RequestTimeout                      time.Duration
}

// DefaultConfig selects finite ceilings but deliberately leaves Model unset.
func DefaultConfig() Config {
	return Config{Endpoint: "https://api.openai.com/v1/responses", MaxModelRequests: 3, MaxLookupRequests: 2, MaxOutputTokens: 1024, MaxResponseBytes: 65536, RequestTimeout: 60 * time.Second}
}
func (c Config) check() error {
	if c.MaxModelRequests < 0 || c.MaxModelRequests > 10 || c.MaxLookupRequests < 0 || c.MaxLookupRequests > 10 {
		return fmt.Errorf("logical budgets must be between 0 and 10")
	}
	if c.RequestTimeout <= 0 || c.RequestTimeout > 10*time.Minute {
		return fmt.Errorf("request timeout must be positive and at most 10m")
	}
	if c.MaxResponseBytes > 65536 || c.MaxResponseBytes > runtime.DefaultLimits().MaxObservationBytes/8 {
		return fmt.Errorf("response limit exceeds observation capacity")
	}
	_, err := openai.New(openai.Config{APIKey: "configuration-validation-only", Model: c.Model, Endpoint: c.Endpoint, MaxOutputTokens: c.MaxOutputTokens, MaxResponseBytes: c.MaxResponseBytes})
	return err
}

// Provision compiles once and retains supplied assets; nil selects the bundled
// nonsensitive corpus. An empty non-nil slice intentionally selects no documents.
func Provision(question string, c Config, assets []effects.Asset) (runtime.Provision, error) {
	if err := c.check(); err != nil {
		return runtime.Provision{}, err
	}
	if question == "" {
		return runtime.Provision{}, fmt.Errorf("question must not be empty")
	}
	if assets == nil {
		a, err := effects.NewAsset("durability", "bundled:corpus.txt", bundledCorpus)
		if err != nil {
			return runtime.Provision{}, err
		}
		assets = []effects.Asset{a}
	}
	if err := effects.CheckAssets(assets); err != nil {
		return runtime.Provision{}, err
	}
	component, err := New("assistant", "caller", "evidence", Budgets{c.MaxModelRequests, c.MaxLookupRequests})
	if err != nil {
		return runtime.Provision{}, err
	}
	config, _ := json.Marshal(c)
	inputs := RequestSchema()
	inputs["config"] = str(8192)
	caller, err := author.Compile(author.Definition{Name: "evidence caller", Inputs: inputs, Outputs: ResultSchema(), Ports: map[string]author.Port{"evidence": component.Port}, Steps: []author.Step{author.Send("evidence", map[string]author.Expr{"question": author.Var("question")}), author.Receive("evidence", bindings(ResultSchema(), "")), author.IfEqual(author.Var("disposition"), author.String("invalid_result"), []author.Step{author.Failure()}, []author.Step{author.Success()})}})
	if err != nil {
		return runtime.Provision{}, err
	}
	app := component.Instances
	app["caller"] = author.Instance{Artifact: caller, Inputs: map[string]schema.Value{"question": schema.Text(question), "config": schema.Text(string(config))}}
	p, err := app.Provision()
	p.Assets = effects.Copy(assets)
	return p, err
}

// View summarizes operator evidence and includes the complete runtime snapshot.
// Unknown usage remains nil in Results; Attempts is not a token or billing count.
type View struct {
	Config                                  Config
	Result                                  map[string]schema.Value
	ModelRequests, LookupRequests, Attempts int
	DeniedAttempts                          int
	Unknown                                 []string
	Results                                 []runtime.AdapterResult
	Runs                                    []runtime.RunState
	Snapshot                                runtime.Inspection
}

// Inspect reads retained artifacts, names, counters, request IDs and public result.
// All summaries are derived from the same runtime snapshot transaction. It does
// not replay history, compile application code, read corpus sources or dispatch adapters.
func Inspect(ctx context.Context, path string) (View, error) {
	var v View
	snapshot, err := runtime.Inspect(ctx, path, runtime.DefaultLimits())
	if err != nil {
		return v, err
	}
	v.Snapshot = snapshot
	v.Runs, err = inspectionRuns(snapshot)
	if err != nil {
		return v, err
	}
	found := false
	for _, r := range v.Runs {
		if r.ID.Subject == "caller" {
			found = true
			if err := json.Unmarshal([]byte(r.Projection.Data["config"].Text), &v.Config); err != nil {
				return v, fmt.Errorf("retained application config: %w", err)
			}
			if _, ok := r.Projection.Data["disposition"]; ok {
				v.Result = map[string]schema.Value{}
				for k := range ResultSchema() {
					v.Result[k] = r.Projection.Data[k]
				}
			}
		}
	}
	if !found {
		return v, fmt.Errorf("not an evidence application store")
	}
	if err := v.Config.check(); err != nil {
		return v, fmt.Errorf("retained application config: %w", err)
	}
	// Counts derive from committed effect intents, independently of actual attempts.
	for _, row := range snapshot.Tables["intents"] {
		var intent struct{ Kind, Target string }
		raw, _ := row["descriptor"].(string)
		if json.Unmarshal([]byte(raw), &intent) == nil && intent.Kind == "adapter" {
			switch intent.Target {
			case ModelCapability:
				v.ModelRequests++
			case LookupCapability:
				v.LookupRequests++
			}
		}
	}
	resultIDs := map[string]bool{}
	for _, row := range snapshot.Tables["adapter_results"] {
		var result runtime.AdapterResult
		raw, _ := row["descriptor"].(string)
		if err := json.Unmarshal([]byte(raw), &result); err != nil {
			return v, err
		}
		v.Results = append(v.Results, result)
		resultIDs[result.Candidate.ID] = true
	}
	for _, row := range snapshot.Tables["adapter_attempts"] {
		var attempt runtime.AdapterAttempt
		raw, _ := row["descriptor"].(string)
		if err := json.Unmarshal([]byte(raw), &attempt); err != nil {
			return v, err
		}
		if attempt.Allowed {
			v.Attempts++
		} else {
			v.DeniedAttempts++
		}
		if attempt.Allowed && !resultIDs[attempt.Prepared.Request.ID] {
			v.Unknown = append(v.Unknown, attempt.Prepared.Request.ID)
		}
	}
	return v, nil
}

// inspectionRuns decodes the public inspection rows without another database
// read. Subject revisions come from that same snapshot, as do run projections.
func inspectionRuns(snapshot runtime.Inspection) ([]runtime.RunState, error) {
	revisions := make(map[string]int64, len(snapshot.Tables["subjects"]))
	for _, row := range snapshot.Tables["subjects"] {
		var subject struct {
			ID       string
			Revision int64
		}
		if err := decodeInspectionRow(row, &subject, "id", "revision"); err != nil {
			return nil, err
		}
		if subject.ID == "" || subject.Revision <= 0 {
			return nil, fmt.Errorf("invalid inspection subject identity or revision")
		}
		revisions[subject.ID] = subject.Revision
	}
	runs := make([]runtime.RunState, 0, len(snapshot.Tables["runs"]))
	for _, row := range snapshot.Tables["runs"] {
		var retained struct {
			Subject                     string
			Generation, Epoch, Sequence int64
			ProgramID                   string `json:"program_id"`
			Projection                  string
		}
		if err := decodeInspectionRow(row, &retained, "subject", "generation", "program_id", "epoch", "sequence", "projection"); err != nil {
			return nil, err
		}
		if retained.Subject == "" || retained.ProgramID == "" || retained.Generation <= 0 || retained.Epoch <= 0 || retained.Sequence < 0 {
			return nil, fmt.Errorf("invalid inspection run identity or sequence")
		}
		revision, ok := revisions[retained.Subject]
		if !ok {
			return nil, fmt.Errorf("inspection run %q has no subject revision", retained.Subject)
		}
		run := runtime.RunState{
			ID:        runtime.RunID{Subject: retained.Subject, Generation: retained.Generation},
			ProgramID: retained.ProgramID, Epoch: retained.Epoch,
			Sequence: retained.Sequence, Revision: revision,
		}
		var projection *runtime.Projection
		if err := json.Unmarshal([]byte(retained.Projection), &projection); err != nil {
			return nil, fmt.Errorf("inspection projection: %w", err)
		}
		if projection == nil {
			return nil, fmt.Errorf("null inspection projection")
		}
		run.Projection = *projection
		runs = append(runs, run)
	}
	return runs, nil
}

func decodeInspectionRow(row map[string]any, target any, required ...string) error {
	for _, field := range required {
		if value, ok := row[field]; !ok || value == nil {
			return fmt.Errorf("inspection row missing non-null field %q", field)
		}
	}
	raw, err := json.Marshal(row)
	if err != nil {
		return fmt.Errorf("inspection row: %w", err)
	}
	if err := json.Unmarshal(raw, target); err != nil {
		return fmt.Errorf("inspection row: %w", err)
	}
	return nil
}

// Run loads retained configuration and resumes the existing store. It never
// compiles, snapshots sources or retries an unknown effect. A successful return
// describes execution, not necessarily a successful answer; inspect Result.
func Run(ctx context.Context, path, apiKey string) (runtime.RunReport, error) {
	v, err := Inspect(ctx, path)
	if err != nil {
		return runtime.RunReport{}, err
	}
	c := v.Config
	model, err := openai.New(openai.Config{APIKey: apiKey, Model: c.Model, Endpoint: c.Endpoint, MaxOutputTokens: c.MaxOutputTokens, MaxResponseBytes: c.MaxResponseBytes})
	if err != nil {
		return runtime.RunReport{}, err
	}
	s, err := runtime.Open(ctx, path, runtime.DefaultLimits())
	if err != nil {
		return runtime.RunReport{}, err
	}
	report, runErr := s.Run(ctx, runtime.RunOptions{MaxSteps: 10000, MaxDispatch: 10000, MaxInFlight: 2, ExternalTimeout: c.RequestTimeout, Adapters: map[string]effects.Adapter{ModelCapability: model, LookupCapability: lookup.New()}})
	return report, errors.Join(runErr, s.Close())
}
