package semantics

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

type sourceFile struct {
	Path, SHA256 string
	Bytes        int
}
type sourceManifest struct {
	Algorithm, SHA256 string
	Files             []sourceFile
}

func compositionSources(t *testing.T) sourceManifest {
	t.Helper()
	paths, err := filepath.Glob("*_test.go")
	if err != nil {
		t.Fatal(err)
	}
	paths = append(paths, "../../go.mod")
	files := map[string]string{}
	for _, p := range paths {
		name := "research/semantics/" + p
		if p == "../../go.mod" {
			name = "go.mod"
		}
		files[name] = p
	}
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	slices.Sort(names)
	h := sha256.New()
	manifest := sourceManifest{Algorithm: "SHA-256 of sorted repository paths: uint64-BE path byte length, UTF-8 path bytes, uint64-BE file byte length, raw file bytes; go.mod and research/semantics/*_test.go only; evidence excluded"}
	for _, name := range names {
		data, err := os.ReadFile(files[name])
		if err != nil {
			t.Fatal(err)
		}
		for _, part := range [][]byte{[]byte(name), data} {
			if err := binary.Write(h, binary.BigEndian, uint64(len(part))); err != nil {
				t.Fatal(err)
			}
			h.Write(part)
		}
		manifest.Files = append(manifest.Files, sourceFile{Path: name, SHA256: fmt.Sprintf("%x", sha256.Sum256(data)), Bytes: len(data)})
	}
	manifest.SHA256 = fmt.Sprintf("%x", h.Sum(nil))
	return manifest
}
func TestCompositionQualification(t *testing.T) {
	sources := compositionSources(t)
	t.Log("COMPOSITION " + canonical(struct {
		Name     string
		Manifest sourceManifest
	}{"source_manifest", sources}))
	for _, spec := range compositionScenarios() {
		t.Run(spec.Name, func(t *testing.T) {
			stats, err := exploreComposition(spec)
			stats.SourceID = sources.SHA256
			t.Log("COMPOSITION " + canonical(stats))
			if err != nil {
				t.Fatal(err)
			}
			if stats.Violation != spec.WantViolation {
				t.Fatalf("violation=%q want=%q", stats.Violation, spec.WantViolation)
			}
			if stats.Transitions == 0 || stats.Rejected == 0 {
				t.Fatal("vacuous exploration")
			}
			for _, w := range spec.WantWitnesses {
				if _, ok := stats.Witnesses[w]; !ok {
					t.Errorf("missing combined witness %s", w)
				}
			}
			for _, why := range spec.WantRejects {
				if stats.Rejections[why] == 0 {
					t.Errorf("missing rejection %s", why)
				}
			}
		})
	}
	for _, nested := range []bool{false, true} {
		t.Run(fmt.Sprintf("complete_nested_%t", nested), func(t *testing.T) {
			c := compositionFixture(nested)
			b := traceBuilder{Config: c}
			b.requests(1)
			b.providers(1, true)
			for _, a := range []string{"provider", "peer"} {
				b.local("outer", 1, sendObservationID(intentID(a, 1, 4)))
			}
			b.local("outer", 1, "")
			b.local("outer", 1, "")
			names := []string{"outer", "peer", "provider"}
			if nested {
				names = append(names, "worker")
			}
			for _, a := range names {
				if a != "outer" {
					b.local(a, 1, "")
					b.local(a, 1, "")
				}
				b.add(event{Kind: deliver, A: a, G: 1})
			}
			spec := compositionSpec{Name: fmt.Sprintf("complete_nested_%t", nested), Config: c, Seed: b.Events}
			n, err := replayCompositionSeed(spec)
			if err != nil {
				t.Fatal(err)
			}
			before := canonical(n.Machine)
			for _, a := range names {
				r := n.Machine.Runs[runID(a, 1)]
				if r.Projection.Status != "succeeded" {
					t.Fatal("incomplete", a)
				}
				if err := replay(n.Machine, runID(a, 1), c.Version); err != nil {
					t.Fatal(err)
				}
			}
			if before != canonical(n.Machine) {
				t.Fatal("replay mutated world")
			}
			if n.Machine.World.Mutations != 2 {
				t.Fatal("wrong public resource count")
			}
			t.Log("COMPOSITION " + canonical(struct {
				Name, Category, ConfigID, SourceID string
				Config                             config
				Trace                              []event
				Admissions                         map[string]*admission
				Contributions                      map[string][]string
				Intents                            map[string]*intent
				World                              resource
				Delivered                          map[string]outcome
			}{spec.Name, "reachability-only", digest(c), sources.SHA256, c, b.Events, n.Machine.Admissions, n.Monitor.Composition.Contributions, n.Machine.Intents, n.Machine.World, n.Machine.Delivered}))
		})
	}
}
func TestCompositionOriginRegression(t *testing.T) {
	// Retains the original behavioral RED input: environmental result, same tag
	// and value as the legitimate peer. The richer envelope also matches exactly.
	for _, spec := range compositionScenarios() {
		if spec.Name == "environment_origin" {
			n, err := replayCompositionSeed(spec)
			if err != nil {
				t.Fatal(err)
			}
			next, why := step(n.Machine, spec.Alphabet[0])
			if why != "" {
				t.Fatal(why)
			}
			_, why = step(next, spec.Alphabet[1])
			if why != "origin" {
				t.Fatalf("Prepare rejection=%q want origin", why)
			}
		}
	}
}
func TestCompositionReplayIncludesOrdinaryMetadata(t *testing.T) {
	spec := compositionScenarios()[6]
	n, err := replayCompositionSeed(spec)
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"saved", "pending", "accepted", "sender", "envelope"} {
		t.Run(field, func(t *testing.T) {
			m := copyOf(n.Machine)
			r := m.Runs[runID("provider", 1)]
			switch field {
			case "saved":
				r.Projection.Saved[0].Generation++
			case "pending":
				r = m.Runs[runID("outer", 1)]
				r.Projection.Pending[0].Peer = "peer"
			case "accepted":
				r.Projection.Accepted = append(r.Projection.Accepted, "forged")
			case "sender", "envelope":
				for id, o := range r.Inbox {
					if o.Consumed && o.Kind == "message" {
						if field == "sender" {
							o.Sender = "peer"
						} else {
							o.Envelope = "{}"
						}
						r.Inbox[id] = o
						break
					}
				}
			}
			if replay(m, runID(r.A, r.G), m.Config.Version) == nil {
				t.Fatal("tampered", field, "accepted")
			}
		})
	}
}
func TestCompositionHardCapFails(t *testing.T) {
	spec := compositionScenarios()[0]
	spec.StateCap = 1
	if _, err := exploreComposition(spec); err == nil || !strings.Contains(err.Error(), "hard state cap") {
		t.Fatal("hard cap was not an error", err)
	}
}
func TestCompositionSeedMonitorsAreCarried(t *testing.T) {
	spec := compositionScenarios()[6]
	spec.Seed = append(spec.Seed, event{Kind: prepare, A: "outer", G: 1, C: 1, Observation: sendObservationID(intentID("provider", 1, 4))}, event{Kind: commit})
	spec.Config.Mutant = "logical-duplicate"
	spec.Alphabet = []event{{Kind: prepare, A: "outer", G: 1, C: 1, Observation: sendObservationID(intentID("provider", 1, 5))}, {Kind: commit}}
	stats, err := exploreComposition(spec)
	if err != nil || stats.Violation != "composition-logical-duplicate" {
		t.Fatalf("seed facts lost: %s %v", stats.Violation, err)
	}
}

func TestCompositionResourceCardinalityMonitor(t *testing.T) {
	c := compositionFixture(false)
	c.Programs["provider"][4] = instruction{Op: emit, Tag: "unit", Value: 1, Mode: "effect"}
	b := traceBuilder{Config: c}
	b.requests(1)
	b.leaf("provider", 1, intentID("outer", 1, 1), true)
	b.local("provider", 1, "")
	b.add(event{Kind: authorize, ID: intentID("provider", 1, 5), Attempt: 1, Owner: "owner", W: 7})
	n, err := replayCompositionSeed(compositionSpec{Config: c, Seed: b.Events})
	if err != nil {
		t.Fatal(err)
	}
	e := event{Kind: applyExternal, ID: intentID("provider", 1, 5), Attempt: 1}
	after, why := step(n.Machine, e)
	if why != "" {
		t.Fatal(why)
	}
	if _, bad := inspect(n.Machine, after, e, n.Monitor); bad != "composition-resource-cardinality" {
		t.Fatalf("monitor=%q", bad)
	}
}
func TestCompositionSeedReplayAndUnrecordedWorld(t *testing.T) {
	for _, spec := range compositionScenarios() {
		if spec.WantViolation != "" {
			continue
		}
		t.Run(spec.Name, func(t *testing.T) {
			n, err := replayCompositionSeed(spec)
			if err != nil {
				t.Fatal(err)
			}
			for id := range n.Machine.Runs {
				if err := replay(n.Machine, id, spec.Config.Version); err != nil {
					t.Fatal(id, err)
				}
			}
			changed := copyOf(n.Machine)
			changed.World.Mutations += 100
			changed.World.Keys["unrecorded"] = 7
			for _, e := range spec.Alphabet {
				if e.Kind != prepare {
					continue
				}
				left, lwhy := step(n.Machine, e)
				right, rwhy := step(changed, e)
				right.World = left.World
				if lwhy != rwhy || canonical(left) != canonical(right) {
					t.Fatal("ordinary preparation inspected world")
				}
			}
		})
	}
}

func TestCompositionNestedRootResourceCardinalityRegression(t *testing.T) {
	c, trace := compositionFanoutCounterexample()
	if len(trace) != 37 {
		t.Fatalf("reviewer trace changed: %d", len(trace))
	}
	n, err := replayCompositionSeed(compositionSpec{Config: c, Seed: trace[:len(trace)-1]})
	if err != nil {
		t.Fatal(err)
	}
	last := trace[len(trace)-1]
	after, why := step(n.Machine, last)
	if why != "" || after.World.Mutations != 2 {
		t.Fatalf("offending input no longer applies: %s mutations=%d", why, after.World.Mutations)
	}
	if _, bad := inspect(n.Machine, after, last, n.Monitor); bad != "composition-resource-cardinality" {
		t.Fatalf("monitor=%q, want composition-resource-cardinality for two effects under one public root", bad)
	}
}

func TestCompositionPublicResourceRootsKeepGenerationsIndependent(t *testing.T) {
	var spec compositionSpec
	for _, candidate := range compositionScenarios() {
		if candidate.Name == "nested_old_result_eligible_generation2" {
			spec = candidate
			break
		}
	}
	n, err := replayCompositionSeed(spec)
	if err != nil {
		t.Fatal(err)
	}
	roots := map[string]bool{}
	for _, g := range []int{1, 2} {
		id := intentID("worker", g, 2)
		if _, ok := n.Machine.World.Keys[id]; !ok {
			t.Fatal("missing applied generation", g)
		}
		d := n.Machine.Intents[id].Descriptor
		root, ok := requestRoot(n.Machine, decodeEnvelope(d.Envelope), map[string]bool{})
		if !ok || root != (envelope{Caller: "outer", Generation: g, Request: 1, Key: "left", ReplyTo: "outer"}) {
			t.Fatalf("generation %d root=%+v valid=%t", g, root, ok)
		}
		roots[canonical(root)] = true
	}
	if len(roots) != 2 {
		t.Fatal("distinct root generations collapsed")
	}
}
