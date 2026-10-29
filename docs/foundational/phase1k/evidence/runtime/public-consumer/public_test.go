package publicprobe

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	join "github.com/JakeFAU/alan/examples/join"
	app "github.com/JakeFAU/alan/examples/substitution"
	"github.com/JakeFAU/alan/examples/substitution/b"
	"github.com/JakeFAU/alan/runtime"
)

// This function is compiled from a different module with no internal imports.
// Its purpose is return-type/field accessibility, not a new scheduling sample.
func publicMethods(ctx context.Context, s *runtime.Store, id runtime.RunID, intent, transport, root string) error {
	state, err := s.Snapshot(ctx, id)
	if err != nil {
		return err
	}
	_, err = s.Commit(ctx, state.Authority(), "")
	if err != nil {
		return err
	}
	auth, err := s.AuthorizeIntent(ctx, intent)
	if err != nil {
		return err
	}
	_ = auth.Source
	admission, err := s.AdmitInvocation(ctx, intent)
	if err != nil {
		return err
	}
	_ = admission.Members
	_, err = s.DeliverMessage(ctx, intent, transport)
	if err != nil {
		return err
	}
	err = s.AcknowledgeMessage(ctx, intent, transport)
	if err != nil {
		return err
	}
	_, err = s.PrepareInvocation(ctx, intent)
	if err != nil {
		return err
	}
	_, err = s.SettleInvocation(ctx, intent)
	if err != nil {
		return err
	}
	_, err = s.PrepareRoot(ctx, root)
	if err != nil {
		return err
	}
	_, err = s.SettleRoot(ctx, root)
	return err
}
func TestExternalDriverSurface(t *testing.T) {
	candidate, err := b.New()
	if err != nil {
		t.Fatal(err)
	}
	d, err := join.New(candidate, app.Config{Shape: "parallel", Request: app.Request{ID: 1, Value: 0, Mode: "answer"}, Limits: runtime.ResourceLimits{MaxInvocations: 3, MaxCreatedRuns: 3, MaxDepth: 2, MaxInFlight: 1}}, join.Options{})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "public.db")
	s, err := runtime.Create(t.Context(), path, d.Provision, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	state, err := s.Snapshot(t.Context(), runtime.RunID{Subject: "root", Generation: 1})
	if err != nil {
		t.Fatal(err)
	}
	next, err := s.Commit(t.Context(), state.Authority(), "")
	if err != nil {
		t.Fatal(err)
	}
	if next.Sequence != 1 {
		t.Fatal("original Invoke not executed")
	}
	snapshot, err := runtime.Inspect(t.Context(), path, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	var operation string
	for _, row := range snapshot.Tables["intents"] {
		if row["subject"] == "root" {
			operation = row["id"].(string)
		}
	}
	auth, err := s.AuthorizeIntent(t.Context(), operation)
	if err != nil || !auth.Allowed {
		t.Fatal("public authorization", auth, err)
	}
	a, err := s.AdmitInvocation(t.Context(), operation)
	if err != nil || !a.Accepted || len(a.Members) != 1 {
		t.Fatal("public invocation admission", a, err)
	}
	p, err := s.PrepareInvocation(t.Context(), operation)
	if err != nil || p.Ready {
		t.Fatal("actual pending child", p, err)
	}
	root, err := s.PrepareRoot(t.Context(), snapshot.Roots[0].RootID)
	if err != nil || root.Ready {
		t.Fatal("actual pending root", root, err)
	}
	after, err := runtime.Inspect(t.Context(), path, runtime.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	var code struct{ Code []json.RawMessage }
	for _, program := range after.Tables["programs"] {
		if program["id"] == state.ProgramID {
			if err := json.Unmarshal([]byte(program["bytes"].(string)), &code); err != nil {
				t.Fatal(err)
			}
		}
	}
	evidence := map[string]any{"profile": "external public API visibility probe", "pre": state, "authority": state.Authority(), "instruction": code.Code[state.Projection.PC], "post": next, "history": snapshot.Tables["history"], "inspection": after, "authorization": auth, "admission": a}
	if dir := os.Getenv("ALAN_I_ARTIFACTS"); dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		f, err := os.Create(filepath.Join(dir, "public-probe.json"))
		if err != nil {
			t.Fatal(err)
		}
		if err = json.NewEncoder(f).Encode(evidence); err != nil {
			t.Fatal(err)
		}
		if err = f.Close(); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("original root sequence %d; original operation %s admitted %d member; source sequence %d", next.Sequence, operation, len(a.Members), auth.Source.Sequence)
}
