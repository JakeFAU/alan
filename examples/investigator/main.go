// Command investigator runs a composed document investigation against retained
// SQLite state. Only the authored parent and its workers decide what runs next.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"reflect"
	"syscall"
	"time"

	"github.com/JakeFAU/alan/adapters/lookup"
	"github.com/JakeFAU/alan/effects"
	"github.com/JakeFAU/alan/runtime"
)

const usage = "usage: investigator init STORE SPEC.json | {run|resume} STORE [--live] | {inspect|replay|answer} STORE"

func writeJSON(out io.Writer, value any) error {
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(value)
}

func initialize(ctx context.Context, path, specification string, out io.Writer) error {
	spec, assets, err := loadSpec(specification)
	if err != nil {
		return err
	}
	settings, err := json.Marshal(spec.deployment)
	if err != nil {
		return err
	}
	p, err := investigatorProvision(spec.Question, [2]string(spec.Queries), string(settings), spec.resources())
	if err != nil {
		return err
	}
	p.Assets = assets
	store, err := runtime.Create(ctx, path, p, runtime.DefaultLimits())
	if err != nil {
		return err
	}
	if err := store.Close(); err != nil {
		return err
	}
	return writeJSON(out, struct {
		Store     string `json:"store"`
		Documents int    `json:"documents"`
	}{path, len(assets)})
}

// The number of chargeable calls follows from the original program and its
// resource root. A different valid ALAN database can contain other programs.
func approvedDeployment(snapshot runtime.Inspection) (deployment, error) {
	var d deployment
	root := runtime.RunID{Subject: "investigator", Generation: 1}
	origins := snapshot.Tables["run_origins"]
	if len(snapshot.Roots) != 1 || snapshot.Roots[0].Run != root ||
		len(origins) != 1 || origins[0]["kind"] != "root" || origins[0]["subject"] != root.Subject ||
		origins[0]["generation"] != root.Generation {
		return d, errors.New("expected a single original investigator deployment")
	}
	row := findRow(snapshot.Tables["runs"], "subject", root.Subject)
	initial, err := cell[runtime.Projection](row, "initial")
	if err != nil || initial == nil {
		return d, errors.New("original investigator deployment is missing")
	}
	data := initial.Data
	settings := data["settings"].Text
	if err := decodeSettings([]byte(settings), &d); err != nil {
		return d, fmt.Errorf("investigator deployment settings: %w", err)
	}
	if err := d.validate(); err != nil {
		return d, err
	}
	queries := [2]string{data["primary_query"].Text, data["counter_query"].Text}
	p, err := investigatorProvision(data["question"].Text, queries, settings, d.resources())
	if err != nil {
		return d, err
	}
	expected := p.Runs[0]
	if row["generation"] != root.Generation || row["program_id"] != expected.ProgramID ||
		!reflect.DeepEqual(*initial, expected.Projection) || snapshot.Roots[0].Limits != d.resources() {
		return d, errors.New("investigator deployment program, initial inputs or resource limits differ")
	}
	// Inspect has already validated these descriptors against every dependent row.
	// Compare the trusted application's complete worker catalogue and original
	// static deployment, including the root's permitted capabilities and members.
	profile, err := cell[runtime.CompositionProvision](findRow(snapshot.Tables["composition_profile"], "singleton", int64(1)), "descriptor")
	if err != nil || profile == nil || len(profile.Roots) != 1 {
		return d, errors.New("missing investigator deployment composition profile")
	}
	actual := profile.Roots[0]
	want := p.Composition.Roots[0]
	want.ID, want.Version = snapshot.Roots[0].RootID, actual.Version
	static, err := cell[runtime.Provision](findRow(snapshot.Tables["composition_deployment"], "singleton", int64(1)), "descriptor")
	if err != nil || static == nil || !reflect.DeepEqual(static.Subjects, p.Subjects) || !reflect.DeepEqual(static.Runs, p.Runs) ||
		!reflect.DeepEqual(actual, want) || !reflect.DeepEqual(profile.Templates, p.Composition.Templates) {
		return d, errors.New("investigator deployment root, static runs or worker catalogue differs")
	}
	if err := d.checkReservations(snapshot); err != nil {
		return d, err
	}
	return d, nil
}

func (d deployment) checkReservations(snapshot runtime.Inspection) error {
	// JSON marshaling escapes tabs, so this verification-only credential cannot
	// collide with any serialized prompt. Pure Prepare requires no real secret.
	model, err := d.Provider.adapter("investigator\tverification")
	if err != nil {
		return err
	}
	rows, ok := snapshot.Tables["adapter_attempts"]
	if !ok {
		return errors.New("investigator deployment has no adapter reservation history")
	}
	remaining := d.Limits.MaxTotalOutputTokens
	for _, row := range rows {
		a, err := cell[runtime.AdapterAttempt](row, "descriptor")
		if err != nil || a == nil || a.Intent.ID == "" || a.Intent.ID != row["intent_id"] ||
			a.Intent.Target != a.Prepared.Request.Capability || a.Intent.Obligation != a.Prepared.Request.Effect {
			return errors.New("invalid investigator deployment adapter reservation")
		}
		if a.Intent.Target == "corpus" && a.Intent.Obligation == "search" {
			continue
		}
		if a.Intent.Target != "model" || a.Intent.Obligation != "reason" {
			return errors.New("unexpected investigator deployment adapter operation")
		}
		// The digest includes model, endpoint, output/response limits, original
		// typed input and the exact HTTP body. A prior owner could supply different
		// wire bytes even when its displayed settings matched the deployment.
		expected, err := model.Prepare(a.Prepared.Request)
		if err != nil || expected.Digest != a.Prepared.Digest {
			return fmt.Errorf("investigator deployment differs from retained model reservation %s", a.Intent.ID)
		}
		result, err := cell[runtime.AdapterResult](findRow(snapshot.Tables["adapter_results"], "intent_id", a.Intent.ID), "descriptor")
		if err != nil {
			return err
		}
		if result != nil {
			switch disposition := result.Candidate.Disposition; disposition {
			case "validated", invalidInterpretation, invalidQuestion:
				if disposition != proposalDisposition(result.Candidate.Payload) {
					return fmt.Errorf("retained model proposal %s differs from the investigator's validation policy", a.Intent.ID)
				}
			}
		}
		if a.Allowed {
			// A missing result cannot refund this reservation.
			remaining -= int64(expected.MaxOutputTokens)
			if remaining < 0 {
				return errors.New("investigator deployment exceeds its lifetime output-token allowance")
			}
		}
	}
	return nil
}

func operate(ctx context.Context, path string, live bool, out io.Writer) error {
	limits := runtime.DefaultLimits()
	snapshot, err := runtime.Inspect(ctx, path, limits)
	if err != nil {
		return err
	}
	d, err := approvedDeployment(snapshot)
	if err != nil {
		return err
	}
	key := controlledCredential
	if d.Provider.Endpoint == officialEndpoint {
		if !live {
			return errors.New("live provider is disabled; add --live to authorize this run against the retained official endpoint")
		}
		key = os.Getenv("OPENAI_API_KEY")
	} else if live {
		return errors.New("--live requires a store provisioned for the official Responses endpoint")
	}
	model, err := d.Provider.adapter(key)
	if err != nil {
		return err
	}
	return d.runOwned(ctx, path, snapshot.Namespace, proposalAdapter{model}, out)
}

func (d deployment) runOwned(ctx context.Context, path, namespace string, model proposalAdapter, out io.Writer) (err error) {
	limits := runtime.DefaultLimits()
	store, err := runtime.Open(ctx, path, limits)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, store.Close()) }()
	// A previous owner can reserve another attempt between our read-only
	// preflight and acquisition. Revalidate the current history while holding
	// the real runtime owner before supplying any adapter to the runner.
	owned, err := runtime.Inspect(ctx, path, limits)
	if err != nil {
		return err
	}
	if store.Namespace() != namespace || owned.Namespace != store.Namespace() {
		return errors.New("store changed after approving the investigator deployment")
	}
	current, err := approvedDeployment(owned)
	if err != nil {
		return err
	}
	if current != d {
		return errors.New("investigator deployment settings changed while acquiring ownership")
	}
	report, err := store.Run(ctx, runtime.RunOptions{
		Adapters: map[string]effects.Adapter{"corpus": lookup.New(), "model": model},
		MaxSteps: 256, MaxDispatch: 128, MaxInFlight: int(d.Limits.MaxInFlight),
		ExternalTimeout: 60 * time.Second,
	})
	if err != nil {
		return err
	}
	return writeJSON(out, report)
}

func run(ctx context.Context, args []string, out io.Writer) error {
	if len(args) < 2 || args[1] == "" {
		return errors.New(usage)
	}
	action, path := args[0], args[1]
	switch action {
	case "init":
		if len(args) == 3 {
			return initialize(ctx, path, args[2], out)
		}
	case "run", "resume":
		if len(args) == 2 || (len(args) == 3 && args[2] == "--live") {
			return operate(ctx, path, len(args) == 3, out)
		}
	case "inspect", "replay", "answer":
		if len(args) != 2 {
			break
		}
		limits := runtime.DefaultLimits()
		var value any
		var err error
		switch action {
		case "inspect":
			value, err = runtime.Inspect(ctx, path, limits)
		case "replay":
			value, err = runtime.Replay(ctx, path, limits)
		case "answer":
			value, err = readReport(ctx, path, limits)
		}
		if err != nil {
			return err
		}
		return writeJSON(out, value)
	}
	return errors.New(usage)
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
