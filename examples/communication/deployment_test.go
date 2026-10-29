package communication

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/JakeFAU/alan/author"
	join "github.com/JakeFAU/alan/examples/join"
	"github.com/JakeFAU/alan/examples/substitution"
	"github.com/JakeFAU/alan/examples/substitution/a"
	"github.com/JakeFAU/alan/examples/substitution/b"
	"github.com/JakeFAU/alan/runtime"
)

const handleObstruction = "branch type mismatch: string and request:invocation:peer"

func config() substitution.Config {
	return substitution.Config{
		Shape: "parallel", Request: substitution.Request{ID: 1, Value: 0, Mode: "answer"},
		Limits: runtime.ResourceLimits{MaxInvocations: 3, MaxCreatedRuns: 3, MaxDepth: 2, MaxInFlight: 1},
	}
}
func candidate(t *testing.T, name string) author.Template {
	t.Helper()
	var c author.Template
	var err error
	if name == "b" {
		c, err = b.New()
	} else {
		c, err = a.New()
	}
	if err != nil {
		t.Fatal(err)
	}
	return c
}

type decodedProgram struct {
	Semantics string
	Code      []json.RawMessage
	Author    author.Metadata
}

func artifact(t *testing.T, d substitution.Deployment, subject string) runtime.Artifact {
	t.Helper()
	for _, r := range d.Provision.Runs {
		if r.ID.Subject != subject {
			continue
		}
		for _, p := range d.Provision.Programs {
			if p.ID == r.ProgramID {
				return p
			}
		}
	}
	t.Fatalf("missing %s program", subject)
	return runtime.Artifact{}
}
func decodeProgram(t *testing.T, p runtime.Artifact) decodedProgram {
	t.Helper()
	var v decodedProgram
	if err := json.Unmarshal(p.Bytes, &v); err != nil {
		t.Fatal(err)
	}
	return v
}
func catalogue(t *testing.T, p runtime.Provision) map[string]runtime.Artifact {
	t.Helper()
	out := map[string]runtime.Artifact{}
	for _, a := range p.Programs {
		if _, exists := out[a.ID]; exists {
			t.Fatalf("duplicate artifact %s", a.ID)
		}
		out[a.ID] = a
	}
	return out
}

// Only the root is constructible. This pins actual available bytes and the
// original inputs; it does not claim a complete J2 peer/provision correspondence.
func TestCommunicationArtifacts(t *testing.T) {
	for _, name := range []string{"a", "b"} {
		t.Run(name, func(t *testing.T) {
			c := candidate(t, name)
			old, err := join.New(c, config(), join.Options{})
			if err != nil {
				t.Fatal(err)
			}
			got, err := New(c, config(), Options{})
			if err == nil || !strings.Contains(err.Error(), handleObstruction) {
				t.Fatalf("expected exact peer authoring obstruction, got %v", err)
			}
			candidateSame := bytes.Equal(c.Bytes(), got.Candidate.Bytes())
			entrySame := bytes.Equal(old.Entry.Bytes(), got.Entry.Bytes())
			peerSame := bytes.Equal(old.Peer.Bytes(), got.Peer.Bytes())
			if !candidateSame || !entrySame || !peerSame || got.FixedRuns != 2 {
				t.Fatal("original A/B/S or fixed membership changed")
			}
			if !reflect.DeepEqual(got.Provision, runtime.Provision{}) {
				t.Fatal("failed construction leaked a runnable provision")
			}
			root, err := rootCaller(c, Options{})
			if err != nil {
				t.Fatal(err)
			}
			newRoot := runtime.Artifact{ID: root.ID(), Bytes: root.Bytes()}
			p, q := decodeProgram(t, artifact(t, old, "root")), decodeProgram(t, newRoot)
			if q.Author.Name != "phase1k-ack-root" || root.ID() == artifact(t, old, "root").ID {
				t.Fatal("root lacks new context identity")
			}
			if !reflect.DeepEqual(p.Code[:2], q.Code[:2]) {
				t.Fatal("original ordered candidate Invoke/Await changed")
			}
			if q.Author.Ports["report"].Obligation != "phase1k_s_report" || q.Author.Ports["ack"].Obligation != "phase1k_s_ack" {
				t.Fatal("wrong new root obligations")
			}
			if !reflect.DeepEqual(q.Author.Ports["report"].Receive, p.Author.Ports["join"].Receive) {
				t.Fatal("truthful J1 report schema changed")
			}
			if len(q.Author.Invocations) != 1 || len(q.Author.Effects) != 0 {
				t.Fatal("root adds dynamic/effect declarations")
			}
			// Original peer prefix is available from J1; a compiled J2 peer is not.
			originalPeer := decodeProgram(t, artifact(t, old, "peer"))
			if len(originalPeer.Code) < 34 {
				t.Fatal("original peer prefix absent")
			}
			for i := 0; i < 32; i++ {
				var in struct{ Op, Dst string }
				if err := json.Unmarshal(originalPeer.Code[i], &in); err != nil {
					t.Fatal(err)
				}
				if in.Op != "set" || in.Dst != "schedule_marker" {
					t.Fatal("original delayed peer prefix changed")
				}
			}
			for _, id := range []string{c.ID(), old.Peer.ID()} {
				if id == "" {
					t.Fatal("missing frozen template identity")
				}
			}
		})
	}
}

// This is an expected-failure regression for the frozen full-policy attempt.
// No desired runtime branch is labeled passing merely because construction fails.
func TestCommunicationPolicy(t *testing.T) {
	for _, tc := range []struct {
		name    string
		options Options
	}{
		{name: "answer"}, {name: "s_fail", options: Options{SMode: "fail"}},
		{name: "s_absent", options: Options{SMode: "absent"}}, {name: "s_pending", options: Options{SMode: "pending"}},
		{name: "wrong_report_request", options: Options{WrongReportRequest: true}},
		{name: "wrong_ack_request", options: Options{WrongACKRequest: true}},
		{name: "wrong_ack_handle", options: Options{WrongACKHandle: true}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := candidate(t, "a")
			original, err := join.New(c, config(), join.Options{SMode: tc.options.SMode})
			if err != nil {
				t.Fatal(err)
			}
			if _, err = rootCaller(c, tc.options); err != nil {
				t.Fatal("independent root construction:", err)
			}
			if _, err = peerCaller(original.Peer, tc.options); err == nil || !strings.Contains(err.Error(), handleObstruction) {
				t.Fatalf("expected exact typed-handle obstruction, got %v", err)
			}
			attempted, err := New(c, config(), tc.options)
			if err == nil || !strings.Contains(err.Error(), handleObstruction) || len(attempted.Provision.Programs) != 0 {
				t.Fatalf("incomplete attempt became runnable: %v", err)
			}
			if !bytes.Equal(original.Peer.Bytes(), attempted.Peer.Bytes()) {
				t.Fatal("S/control was reconstructed")
			}
		})
	}
}
func TestCommunicationConfiguration(t *testing.T) {
	c := candidate(t, "a")
	for _, tc := range []struct {
		name    string
		change  func(*substitution.Config)
		options Options
	}{
		{name: "shape", change: func(c *substitution.Config) { c.Shape = "sequential" }},
		{name: "id", change: func(c *substitution.Config) { c.Request.ID = 0 }},
		{name: "value", change: func(c *substitution.Config) { c.Request.Value = 2 }},
		{name: "s_mode", options: Options{SMode: "unknown"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := config()
			if tc.change != nil {
				tc.change(&cfg)
			}
			if _, err := New(c, cfg, tc.options); err == nil || strings.Contains(err.Error(), handleObstruction) {
				t.Fatalf("invalid input did not fail before attempted peer: %v", err)
			}
		})
	}
	cfg := config()
	cfg.Shape = ""
	first, err := New(c, cfg, Options{SMode: "answer"})
	if err == nil || !strings.Contains(err.Error(), handleObstruction) {
		t.Fatal(err)
	}
	second, err := New(c, config(), Options{})
	if err == nil || !strings.Contains(err.Error(), handleObstruction) {
		t.Fatal(err)
	}
	if !bytes.Equal(first.Peer.Bytes(), second.Peer.Bytes()) {
		t.Fatal("default shape/answer changed original S")
	}
}

// Export only actually available artifacts; absence of a J2 peer/provision is
// explicit in the manifest. The independent model is not consulted here.
func TestCommunicationArtifactExport(t *testing.T) {
	dir := os.Getenv("ALAN_COMMUNICATION_EXPORT")
	if dir == "" {
		t.Skip("explicit artifact export not requested")
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	hashes := map[string]string{}
	write := func(name string, raw []byte) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), raw, 0644); err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(raw)
		hashes[name] = hex.EncodeToString(sum[:])
	}
	manifest := map[string]any{
		"status":      "J2 peer construction obstructed; complete J2 artifact/provision/runtime trace unavailable",
		"obstruction": handleObstruction, "compiled_j2_peer": false, "compiled_j2_provision": false,
	}
	for _, name := range []string{"a", "b"} {
		c := candidate(t, name)
		old, err := join.New(c, config(), join.Options{})
		if err != nil {
			t.Fatal(err)
		}
		root, err := rootCaller(c, Options{})
		if err != nil {
			t.Fatal(err)
		}
		write(name+"-candidate-template.json", c.Bytes())
		write(name+"-s-template.json", old.Peer.Bytes())
		write(name+"-j2-root-program.json", root.Bytes())
		write(name+"-original-j1-peer-program.json", artifact(t, old, "peer").Bytes)
		ids := []string{}
		byID := catalogue(t, old.Provision)
		for id := range byID {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		catalogueRows := []map[string]any{}
		for _, id := range ids {
			p := byID[id]
			h := sha256.Sum256(p.Bytes)
			catalogueRows = append(catalogueRows, map[string]any{
				"id": id, "semantics": p.Semantics, "bytes_sha256": hex.EncodeToString(h[:]),
			})
		}
		raw, err := json.MarshalIndent(map[string]any{
			"original_j1_catalogue": catalogueRows, "original_resource_roots": old.Provision.Composition.Roots,
			"fixed_runs": old.FixedRuns, "j2_root_id": root.ID(),
		}, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		write(name+"-available-manifest.json", append(raw, '\n'))
		if _, err = peerCaller(old.Peer, Options{}); err == nil {
			t.Fatal("missing expected obstruction")
		}
		write(name+"-peer-error.txt", []byte(err.Error()+"\n"))
	}
	for _, mode := range []string{"fail", "absent", "pending"} {
		old, err := join.New(candidate(t, "a"), config(), join.Options{SMode: mode})
		if err != nil {
			t.Fatal(err)
		}
		write("s-"+mode+"-original-template.json", old.Peer.Bytes())
	}
	raw, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	write("availability.json", append(raw, '\n'))
	raw, err = json.MarshalIndent(hashes, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(dir, "sha256.json"), append(raw, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
}
