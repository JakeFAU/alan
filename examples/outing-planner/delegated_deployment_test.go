package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/JakeFAU/alan/runtime"
)

func TestDelegatedDeploymentPreservesLegacyArtifacts(t *testing.T) {
	c := fixtureConfiguration()
	assess, err := c.provision(assessFirst)
	if err != nil {
		t.Fatal(err)
	}
	confirm, err := c.provision(confirmFirst)
	if err != nil {
		t.Fatal(err)
	}
	delegated, err := c.provision(delegatedWeather)
	if err != nil {
		t.Fatal(err)
	}
	const outing = "5cffe9d8d1e2235f03b6c8c23945196f6579dbd632479bac4077106483d8d516"
	const canonical = "7da883dbefa445ee346008c47a0bead6566933e81755aa1b466460ca62fc7f9b"
	for _, tc := range []struct {
		variant string
		weather string
		before  runtime.Provision
	}{
		{assessFirst, "d9c6d7c7ea715fcf0fe8e86b3ae0b40ece4e0a730f06236ffa5daef3f794bfbc", assess},
		{confirmFirst, "82dda63983a442146a666b06d6c061bb6616a2382b6b5626d405f1c5802df009", confirm},
	} {
		t.Run(tc.variant, func(t *testing.T) {
			again, err := c.provision(tc.variant)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(tc.before, again) {
				t.Fatal("legacy provisioning changed after delegated construction")
			}
			for i, want := range []string{outing, tc.weather} {
				a := again.Programs[i]
				if a.ID != want || fmt.Sprintf("%x", sha256.Sum256(a.Bytes)) != want {
					t.Fatalf("pinned artifact changed: %s, want %s", a.ID, want)
				}
			}
			raw := []byte(again.Runs[0].Projection.Data[configInput].Text)
			if fmt.Sprintf("%x", sha256.Sum256(raw)) != canonical ||
				!reflect.DeepEqual(again.Runs[0], delegated.Runs[0]) ||
				!bytes.Equal(again.Programs[0].Bytes, delegated.Programs[0].Bytes) {
				t.Fatal("canonical configuration or exact Outing initial state and bytes changed")
			}
		})
	}
}

func TestDelegatedDeploymentBoundsAndExactRoots(t *testing.T) {
	c := fixtureConfiguration()
	for _, tc := range []struct {
		variant string
		want    int64
	}{{delegatedWeather, 1}, {assessFirst, 0}, {confirmFirst, 0}} {
		if got := defaultDeploymentOptions(tc.variant); got.Weather != tc.variant || got.HelperAdmission != tc.want {
			t.Fatalf("default deployment options for %s: %+v", tc.variant, got)
		}
	}
	for _, options := range []deploymentOptions{
		{Weather: delegatedWeather, HelperAdmission: -1}, {Weather: delegatedWeather, HelperAdmission: 2},
		{Weather: assessFirst, HelperAdmission: 1}, {Weather: confirmFirst, HelperAdmission: 1},
	} {
		if _, err := c.provisionDeployment(options); err == nil {
			t.Fatalf("accepted unsupported options %+v", options)
		}
	}
	for _, allowance := range []int64{0, 1} {
		p, err := c.provisionDeployment(deploymentOptions{Weather: delegatedWeather, HelperAdmission: allowance})
		if err != nil {
			t.Fatal(err)
		}
		if len(p.Programs) != 3 || len(p.Runs) != 2 || len(p.Composition.Templates) != 1 ||
			len(p.Composition.Roots) != 2 {
			t.Fatalf("unexpected deployment shape for allowance %d", allowance)
		}
		template := p.Composition.Templates[0]
		if template.EntryMember != "worker" || template.EntryPort != "work" || template.CallerPeer != "caller" ||
			len(template.Members) != 1 || len(template.Templates) != 0 || template.Handoff != nil ||
			!reflect.DeepEqual(template.Capabilities, []string{capability}) {
			t.Fatalf("unexpected retained helper catalogue: %+v", template)
		}
		outing, weather := p.Composition.Roots[0], p.Composition.Roots[1]
		if !reflect.DeepEqual(outing, runtime.ResourceRoot{
			Run: outingRun, Members: []runtime.RunID{outingRun}, Limits: runtime.ResourceLimits{MaxInFlight: 1},
		}) || !reflect.DeepEqual(weather, runtime.ResourceRoot{
			Run: weatherRun, Members: []runtime.RunID{weatherRun},
			Capabilities: []string{capability}, Templates: []string{template.ID},
			Limits: runtime.ResourceLimits{
				MaxInvocations: allowance, MaxCreatedRuns: 1, MaxDepth: 1, MaxEffects: 2, MaxInFlight: 1,
			},
		}) {
			t.Fatalf("wrong exact root bounds: %+v, %+v", outing, weather)
		}
	}
	c.Slots = append(c.Slots, "12:00")
	p, err := c.provision(delegatedWeather)
	if err == nil || !strings.Contains(err.Error(), "at most two eligible candidates") || !reflect.DeepEqual(p, runtime.Provision{}) {
		t.Fatalf("delegated provisioning allowed four eligible candidates: %v", err)
	}
	if _, err := c.provision(assessFirst); err != nil {
		t.Fatalf("legacy eligible bound changed: %v", err)
	}
}

func TestDelegatedDeploymentRejectsUnsupportedSnapshotShape(t *testing.T) {
	c := fixtureConfiguration()
	c.MaxTravelMinutes = 2
	p, err := c.provision(delegatedWeather)
	if err != nil {
		t.Fatal(err)
	}
	_, cut := executeDelegatedProvision(t, p, c)
	info, err := retainedDeployment(cut)
	if err != nil || info.HelperRun == nil {
		t.Fatalf("valid deployed helper: %+v %v", info, err)
	}
	worker, _ := cut.Run(*info.HelperRun)
	call := cut.Invocations[0]
	for _, tc := range []struct {
		name   string
		mutate func(*runtime.ExecutionSnapshot)
	}{
		{"extra run", func(s *runtime.ExecutionSnapshot) { s.Runs = append(s.Runs, worker) }},
		{"extra grant", func(s *runtime.ExecutionSnapshot) {
			s.Permissions = append(s.Permissions, runtime.Permission{Kind: "input", Source: "local", Target: worker.ID.Subject})
		}},
		{"missing worker grant", func(s *runtime.ExecutionSnapshot) {
			grants := make([]runtime.Permission, 0, len(s.Permissions))
			for _, grant := range s.Permissions {
				if grant.Source != worker.ID.Subject {
					grants = append(grants, grant)
				}
			}
			s.Permissions = grants
		}},
		{"extra invocation", func(s *runtime.ExecutionSnapshot) { s.Invocations = append(s.Invocations, call) }},
		{"stray committed effect", func(s *runtime.ExecutionSnapshot) {
			s.Operations = append(s.Operations, runtime.Intent{Kind: "adapter", Run: weatherRun})
		}},
		{"stray effect view", func(s *runtime.ExecutionSnapshot) {
			s.Effects = append(s.Effects, &runtime.ExecutionEffect{Source: worker.ID})
		}},
		{"missing binding", func(s *runtime.ExecutionSnapshot) {
			v, a := *call, *call.Admission
			a.BindingID = ""
			v.Admission = &a
			s.Invocations = []*runtime.InvocationView{&v}
		}},
		{"wrong original call owner", func(s *runtime.ExecutionSnapshot) {
			v, a := *call, *call.Admission
			a.Invocation.Caller = outingRun
			v.Admission = &a
			s.Invocations = []*runtime.InvocationView{&v}
		}},
		{"wrong admitted run", func(s *runtime.ExecutionSnapshot) {
			v, a := *call, *call.Admission
			a.Members = []runtime.RunID{{Subject: worker.ID.Subject, Generation: 2}}
			v.Admission = &a
			s.Invocations = []*runtime.InvocationView{&v}
		}},
		{"wrong catalogue", func(s *runtime.ExecutionSnapshot) {
			composition := *s.Composition
			catalogue := slices.Clone(composition.Templates)
			catalogue[0].EntryPort = "foreign"
			composition.Templates = catalogue
			s.Composition = &composition
		}},
		{"unsupported root bound", func(s *runtime.ExecutionSnapshot) {
			composition := *s.Composition
			roots := slices.Clone(composition.Roots)
			roots[0].Limits.MaxEffects++
			composition.Roots = roots
			s.Composition = &composition
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tampered := *cut
			tc.mutate(&tampered)
			if _, err := retainedDeployment(&tampered); err == nil {
				t.Fatalf("accepted %s", tc.name)
			}
		})
	}
}
