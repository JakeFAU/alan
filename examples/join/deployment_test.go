package join_test

import (
	"bytes"
	"cmp"
	"encoding/json"
	"reflect"
	"slices"
	"testing"

	"github.com/JakeFAU/alan/author"
	join "github.com/JakeFAU/alan/examples/join"
	"github.com/JakeFAU/alan/examples/substitution"
	"github.com/JakeFAU/alan/examples/substitution/a"
	"github.com/JakeFAU/alan/runtime"
)

func config() substitution.Config {
	return substitution.Config{Shape: "parallel", Request: substitution.Request{ID: 1, Value: 0, Mode: "answer"}, Limits: runtime.ResourceLimits{MaxInvocations: 3, MaxCreatedRuns: 3, MaxDepth: 3, MaxInFlight: 1}}
}
func candidate(t *testing.T) author.Template {
	t.Helper()
	c, e := a.New()
	if e != nil {
		t.Fatal(e)
	}
	return c
}
func program(t *testing.T, d substitution.Deployment, subject string) map[string]any {
	t.Helper()
	for _, r := range d.Provision.Runs {
		if r.ID.Subject == subject {
			for _, p := range d.Provision.Programs {
				if p.ID == r.ProgramID {
					var v map[string]any
					if e := json.Unmarshal(p.Bytes, &v); e != nil {
						t.Fatal(e)
					}
					return v
				}
			}
		}
	}
	t.Fatalf("missing %s program", subject)
	return nil
}
func TestJoinPreservesPrimaryArtifactsAndPrefixes(t *testing.T) {
	c := candidate(t)
	old, e := substitution.DelayedParallel(c, config(), 32)
	if e != nil {
		t.Fatal(e)
	}
	d, e := join.New(c, config(), join.Options{})
	if e != nil {
		t.Fatal(e)
	}
	if !bytes.Equal(c.Bytes(), d.Candidate.Bytes()) || !bytes.Equal(c.Bytes(), d.Entry.Bytes()) || !bytes.Equal(old.Peer.Bytes(), d.Peer.Bytes()) {
		t.Fatal("frozen candidate/S changed")
	}
	if d.FixedRuns != 2 || len(d.Provision.Runs) != 2 || !reflect.DeepEqual(old.Provision.Composition.Roots, d.Provision.Composition.Roots) {
		t.Fatal("static topology or resource root changed")
	}
	for _, tc := range []struct {
		subject string
		prefix  int
	}{{"root", 2}, {"peer", 34}} {
		p, q := program(t, old, tc.subject), program(t, d, tc.subject)
		if !reflect.DeepEqual(p["Code"].([]any)[:tc.prefix], q["Code"].([]any)[:tc.prefix]) {
			t.Fatalf("%s invocation prefix changed", tc.subject)
		}
		if reflect.DeepEqual(p, q) {
			t.Fatalf("%s artifact identity unchanged", tc.subject)
		}
	}
	if len(d.Provision.Permissions) != 1 || d.Provision.Permissions[0].Source != "peer" || d.Provision.Permissions[0].Target != "root" || d.Provision.Permissions[0].Obligation != "phase1j_s_join" {
		t.Fatal("ordinary join grant", d.Provision.Permissions)
	}
}

func TestJoinConfigurationControls(t *testing.T) {
	c := candidate(t)
	for _, tc := range []struct {
		name    string
		change  func(*substitution.Config)
		options join.Options
	}{
		{name: "sequential", change: func(c *substitution.Config) { c.Shape = "sequential" }},
		{name: "invalid_id", change: func(c *substitution.Config) { c.Request.ID = 0 }},
		{name: "invalid_value", change: func(c *substitution.Config) { c.Request.Value = 2 }},
		{name: "invalid_s", options: join.Options{SMode: "unknown"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := config()
			if tc.change != nil {
				tc.change(&cfg)
			}
			if _, e := join.New(c, cfg, tc.options); e == nil {
				t.Fatal("accepted outside profile")
			}
		})
	}
	cfg := config()
	cfg.Shape = ""
	d, e := join.New(c, cfg, join.Options{SMode: "answer"})
	if e != nil {
		t.Fatal(e)
	}
	primary, e := join.New(c, config(), join.Options{})
	if e != nil {
		t.Fatal(e)
	}
	if !sameDeployment(d, primary) {
		t.Fatal("empty shape / explicit answer changed artifacts")
	}
}

// sameDeployment ignores only catalogue order. Public provision construction
// gathers programs from application maps; every artifact field and every other
// deployment field remains part of the equality check. Sorting copied slices
// preserves the original public provision and frozen artifact evidence.
func sameDeployment(a, b substitution.Deployment) bool {
	a.Provision.Programs = slices.Clone(a.Provision.Programs)
	b.Provision.Programs = slices.Clone(b.Provision.Programs)
	order := func(a, b runtime.Artifact) int { return cmp.Compare(a.ID, b.ID) }
	slices.SortFunc(a.Provision.Programs, order)
	slices.SortFunc(b.Provision.Programs, order)
	return reflect.DeepEqual(a, b)
}

func TestJoinDeploymentComparison(t *testing.T) {
	d, err := join.New(candidate(t), config(), join.Options{})
	if err != nil {
		t.Fatal(err)
	}
	original := slices.Clone(d.Provision.Programs)
	reordered := d
	reordered.Provision.Programs = slices.Clone(d.Provision.Programs)
	slices.Reverse(reordered.Provision.Programs)
	beforeComparison := slices.Clone(reordered.Provision.Programs)
	if !sameDeployment(d, reordered) {
		t.Fatal("catalogue order changed deployment equality")
	}
	if !reflect.DeepEqual(d.Provision.Programs, original) || !reflect.DeepEqual(reordered.Provision.Programs, beforeComparison) {
		t.Fatal("comparison mutated a provision catalogue")
	}
	for _, tc := range []struct {
		name   string
		mutate func(*substitution.Deployment)
	}{
		{"program_id", func(d *substitution.Deployment) { d.Provision.Programs[0].ID += "-altered" }},
		{"program_bytes", func(d *substitution.Deployment) { d.Provision.Programs[0].Bytes[0] ^= 1 }},
		{"program_semantics", func(d *substitution.Deployment) { d.Provision.Programs[0].Semantics += "-altered" }},
		{"missing_program", func(d *substitution.Deployment) { d.Provision.Programs = d.Provision.Programs[1:] }},
		{"fixed_runs", func(d *substitution.Deployment) { d.FixedRuns++ }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			changed := d
			changed.Provision.Programs = slices.Clone(d.Provision.Programs)
			for i := range changed.Provision.Programs {
				changed.Provision.Programs[i].Bytes = bytes.Clone(changed.Provision.Programs[i].Bytes)
			}
			tc.mutate(&changed)
			if sameDeployment(d, changed) {
				t.Fatal("comparison accepted altered deployment")
			}
		})
	}
}
