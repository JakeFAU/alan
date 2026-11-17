package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"
)

func TestDelegatedReplacementRetainsEvidence(t *testing.T) {
	for _, tc := range []struct {
		name                  string
		n, allowance          int
		firstCode, secondCode string
		want, selectedPark    string
	}{
		{name: "empty", allowance: 1, firstCode: "1", secondCode: "3", want: noMatch},
		{name: "one suitable", n: 1, allowance: 1, firstCode: "1", secondCode: "3", want: recommended, selectedPark: "cedar"},
		{name: "one unsuitable", n: 1, allowance: 1, firstCode: "97", secondCode: "1", want: noMatch},
		{name: "one unavailable", n: 1, allowance: 1, firstCode: "999", secondCode: "1", want: unavailable},
		{name: "two suitable", n: 2, allowance: 1, firstCode: "1", secondCode: "1", want: recommended, selectedPark: "cedar"},
		{name: "unsuitable then suitable", n: 2, allowance: 1, firstCode: "97", secondCode: "1", want: recommended, selectedPark: "hemlock"},
		{name: "unavailable then suitable", n: 2, allowance: 1, firstCode: "999", secondCode: "1", want: recommended, selectedPark: "hemlock"},
		{name: "unavailable and unsuitable", n: 2, allowance: 1, firstCode: "999", secondCode: "97", want: unavailable},
		{name: "refused empty", allowance: 0, firstCode: "1", secondCode: "3", want: helperRefused},
		{name: "refused one", n: 1, allowance: 0, firstCode: "1", secondCode: "3", want: helperRefused},
		{name: "refused two", n: 2, allowance: 0, firstCode: "1", secondCode: "3", want: helperRefused},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c := fixtureConfiguration()
			switch tc.n {
			case 0:
				c.MaxTravelMinutes = 2
			case 1:
				c.MaxTravelMinutes = 10
			}
			fixture := writePairedFixture(t, tc.firstCode, tc.secondCode)
			dir := t.TempDir()
			output := plannerOutput(t, "check-replacement", "--config", writeDelegatedConfig(t, c),
				"--fixture", fixture, "--weather", delegatedWeather, "--helper-admission", strconv.Itoa(tc.allowance), "--output-dir", dir)
			plannerOutcome(t, output, tc.want)

			var pair replacementReport
			if err := json.Unmarshal([]byte(output), &pair); err != nil {
				t.Fatal(err)
			}
			comparison := pair.Delegated
			delegated := pair.Candidate.Report.Delegated
			if comparison == nil || delegated == nil || pair.Baseline.Report.Delegated != nil ||
				comparison.PositiveReplacement != completedPlan(tc.want) ||
				delegated.HelperAdmission != int64(tc.allowance) || pair.Candidate.Cycle != nil || pair.Candidate.Reopened == nil ||
				pair.Baseline.FixtureRequests != tc.n || pair.Candidate.FixtureRequests != tc.n*tc.allowance ||
				len(comparison.ForecastOperations) != tc.n {
				t.Fatalf("unexpected delegated comparison: %+v", comparison)
			}
			if tc.allowance == 0 {
				if delegated.HelperRun != nil || len(pair.Candidate.Report.Effects) != 0 ||
					pair.Candidate.Report.Decision != nil || pair.Candidate.Report.Assessment != nil ||
					delegated.Invocations[0].Admission.Reason != "invocation_limit" {
					t.Fatal("refusal manufactured downstream helper or plan evidence")
				}
			}
			if comparison.BaselineCost != (replacementCost{Effects: int64(tc.n)}) ||
				comparison.CandidateCost != (replacementCost{Invocations: int64(tc.allowance), CreatedRuns: int64(tc.allowance), Effects: int64(tc.n * tc.allowance)}) {
				t.Fatalf("incorrect costs: %+v", comparison)
			}
			if tc.selectedPark != "" && (pair.Candidate.Report.Recommendation == nil ||
				pair.Candidate.Report.Recommendation.ParkID != tc.selectedPark) {
				t.Fatalf("incorrect recommended park: %+v", pair.Candidate.Report.Recommendation)
			}
			direct, helper := orderedForecasts(pair.Baseline.Report.Effects), orderedForecasts(pair.Candidate.Report.Effects)
			for i, matched := range comparison.ForecastOperations {
				want := forecastOperationPair{CandidateID: pair.Candidate.Report.Candidates[i].ID, Baseline: direct[i].OperationID}
				if tc.allowance != 0 {
					want.Delegated = helper[i].OperationID
				}
				if matched != want || matched.Baseline == matched.Delegated {
					t.Fatalf("operation map %d: actual %+v, expected %+v", i, matched, want)
				}
			}
			verifyFrozenManifest(t, dir, pair)
		})
	}
}

func writePairedFixture(t *testing.T, cedar, hemlock string) string {
	t.Helper()
	raw, err := os.ReadFile("testdata/fixture.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Version       int            `json:"version"`
		ReferenceTime string         `json:"reference_time"`
		Entries       []fixtureEntry `json:"entries"`
	}
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	for i, code := range []string{cedar, hemlock} {
		entry := &fixture.Entries[i]
		before := []string{"\"weather_code\":[1,2]", "\"weather_code\":[3,2]"}[i]
		if !strings.Contains(*entry.Body, before) {
			t.Fatal("fixture lacks its expected original weather codes")
		}
		entry.Body = new(strings.Replace(*entry.Body, before, "\"weather_code\":["+code+","+code+"]", 1))
	}
	raw, err = json.Marshal(fixture)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "fixture.json")
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func verifyFrozenManifest(t *testing.T, dir string, pair replacementReport) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(dir, "deployment-manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var m deploymentManifest
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if m.Version != 1 || m.ConfigurationSHA != pair.DerivedConfigSHA || !slices.Equal(m.Candidates, pair.Baseline.Report.Candidates) ||
		len(m.Artifacts) != 4 || len(m.Deployments) != 2 || m.Source.FeatureBase != delegatedFeatureBase ||
		m.Source.BuildRevision == "" || m.Source.BuildModified == "" {
		t.Fatalf("unexpected frozen deployment manifest envelope")
	}
	roles := []string{"helper:worker", "outing", "weather:assess-first", "weather:delegated"}
	for i, artifact := range m.Artifacts {
		if artifact.Role != roles[i] || digestText(string(artifact.Canonical)) != artifact.ProgramID || artifact.Semantics != "alan/1g/v1" {
			t.Fatalf("manifest artifact %s failed exact byte validation", artifact.Role)
		}
	}
	for i, r := range []replacementRun{pair.Baseline, pair.Candidate} {
		spec := m.Deployments[i]
		var allowance int64
		if r.Report.Delegated != nil {
			allowance = r.Report.Delegated.HelperAdmission
			if len(spec.Composition.Templates) != 1 || spec.Composition.Templates[0].ID != r.Report.Delegated.HelperTemplate ||
				spec.Composition.Templates[0].Members[helperMember].ProgramID != m.Artifacts[0].ProgramID {
				t.Fatal("manifest lost the allowed helper template and program")
			}
		}
		if spec.Weather != r.Report.Variant || len(spec.InitialRuns) != 2 || spec.HelperAdmission != allowance {
			t.Fatalf("frozen deployment options disagree with retained report")
		}
		for _, initial := range spec.InitialRuns {
			got := findReportRun(r.Report, initial.ID)
			if got == nil || got.ProgramID != initial.ProgramID || !reflect.DeepEqual(got.Initial, initial.Projection) {
				t.Fatalf("frozen inputs differ for %s", initial.ID.Subject)
			}
		}
	}
}

func TestDelegatedReplacementRejectsUnsupportedWithoutWork(t *testing.T) {
	c := fixtureConfiguration()
	c.Slots = []string{"10:00", "11:00"}
	file := writeDelegatedConfig(t, c)
	for _, tc := range []struct {
		name, want string
		flags      []string
	}{
		{"domain", "at most two eligible candidates", []string{"--weather", delegatedWeather}},
		{"allowance", "must be 0 or 1", []string{"--weather", delegatedWeather, "--helper-admission", "2"}},
		{"legacy flag", "requires delegated Weather", []string{"--weather", confirmFirst, "--helper-admission", "0"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out := filepath.Join(t.TempDir(), "unsupported")
			args := []string{"check-replacement", "--config", file, "--fixture", "testdata/fixture.json", "--output-dir", out}
			err := runCLI(t.Context(), append(args, tc.flags...), new(strings.Builder))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("want %q, got %v", tc.want, err)
			}
			if _, err := os.Stat(out); !os.IsNotExist(err) {
				t.Fatalf("unsupported replacement created output directory: %v", err)
			}
		})
	}
}

func TestDelegatedReplacementBoundRetainsUnderlyingOutcome(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bounded.db")
	bounds := standardBounds
	bounds.MaxSteps = 1
	caseReport, err := runReplacementScenario(t.Context(), path, fixtureConfiguration(), delegatedWeather, bounds)
	if err != nil {
		t.Fatal(err)
	}
	if caseReport.Outcome != boundExhausted || caseReport.Report.Outcome != pendingOutcome ||
		caseReport.BoundError == "" || caseReport.Cycle != nil || caseReport.Report.Progress.Quiescent {
		t.Fatalf("unexpected delegated bound outcome %q with retained %q: %q", caseReport.Outcome, caseReport.Report.Outcome, caseReport.BoundError)
	}
}

func TestDelegatedReplacementPreservesExactLegacyCycleWitness(t *testing.T) {
	dir := t.TempDir()
	output := plannerOutput(t, "check-replacement", "--config", "testdata/configuration.json",
		"--fixture", "testdata/fixture.json", "--weather", confirmFirst, "--output-dir", dir)
	var pair replacementReport
	if err := json.Unmarshal([]byte(output), &pair); err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(pair.Candidate.Cycle)
	if err != nil {
		t.Fatal(err)
	}
	// Captured before the prerequisite ports from the same two-park fixture.
	const expected = "692358c9b8224a8e92c8bf5c3dae33b6d190b3687df75a053c98a0ee7f72e728"
	if pair.Outcome != circularDependency || pair.Candidate.FixtureRequests != 0 ||
		digestText(string(raw)) != expected || !reflect.DeepEqual(pair.Candidate.Cycle, pair.Candidate.Reopened.Cycle) {
		t.Fatalf("legacy cycle witness changed: %s", raw)
	}
	if strings.Contains(output, `"delegated":`) || strings.Contains(output, `"delegated_comparison":`) {
		t.Fatal("legacy replacement JSON acquired delegated fields")
	}
	if _, err := os.Stat(filepath.Join(dir, "deployment-manifest.json")); !os.IsNotExist(err) {
		t.Fatalf("legacy replacement added a delegated manifest: %v", err)
	}
}
