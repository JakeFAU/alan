package executor

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	resourcefixture "github.com/JakeFAU/alan/internal/fixture"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestRequestBindingProfile(t *testing.T) {
	p, c := compositionFixture(t)
	c.RequestAuthority = "alan/request-authority/v2"
	p.Composition = &c
	s, err := Create(t.Context(), filepath.Join(t.TempDir(), "binding.sqlite"), p, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	v, err := storedSchemaNumber(t.Context(), s.conn)
	if err != nil || v != 15 {
		t.Fatalf("format=%d: %v", v, err)
	}
}

func bindingFixture(t *testing.T, version string) Provision {
	p := requestAuthorityFixture(t, func(c *DelegationClaim, u *AuthorityUse, p *Provision) {
		c.Version = version
		u.Version = version
		c.ServiceOperationID = identity("A", int64(1), int64(3), 0)
	})
	p.Composition.RequestAuthority = version
	p.Permissions = append(p.Permissions, Permission{Kind: "message", Source: "B", Target: "A", Operation: "message", Revision: 1, Obligation: "ready"})
	authorityReplaceProgram(t, &p, "A", func(p *Program) {
		ready := p.Code[0]
		m := *ready.Match
		m.Obligation = Lit(Text("ready"))
		ready.Match = &m
		p.Code = append([]Instruction{p.Code[0], ready}, p.Code[1:]...)
	})
	authorityReplaceProgram(t, &p, "B", func(p *Program) {
		p.Code = append(p.Code[:2], Instruction{Op: "emit", Effect: &Effect{Kind: "message", Target: Lit(Text("A")), Generation: Lit(Int(1)), Obligation: Lit(Text("ready")), Payload: map[string]Operand{}}}, p.Code[2])
	})
	return p
}

func TestRequestBindingNonadjacent(t *testing.T) {
	for _, version := range []string{RequestAuthorityVersion, "alan/request-authority/v2"} {
		t.Run(version, func(t *testing.T) {
			s, path := messageStore(t, bindingFixture(t, version))
			if _, err := s.Commit(t.Context(), current(t, s, "B").Authority(), ""); err != nil {
				t.Fatal(err)
			}
			if d := authorizedDecision(t, s, committedRequest(t, s, "B")); !d.Accepted {
				t.Fatal(d)
			}
			q := identity("B", int64(1), int64(1), 0)
			admission, err := s.DeliverMessage(t.Context(), q, "arbitrary 🧭 request")
			if err != nil {
				t.Fatal(err)
			}
			if _, err = s.Commit(t.Context(), current(t, s, "A").Authority(), admission.ObservationID); err != nil {
				t.Fatal(err)
			}
			if _, err = s.Commit(t.Context(), current(t, s, "B").Authority(), ""); err != nil {
				t.Fatal(err)
			}
			ready, err := s.DeliverMessage(t.Context(), identity("B", int64(1), int64(3), 0), "ready")
			if err != nil {
				t.Fatal(err)
			}
			if _, err = s.Commit(t.Context(), current(t, s, "A").Authority(), ready.ObservationID); err != nil {
				t.Fatal(err)
			}
			if _, err = s.Commit(t.Context(), current(t, s, "A").Authority(), ""); err != nil {
				t.Fatal(err)
			}
			a, _, err := s.prepareExternal(t.Context(), identity("A", int64(1), int64(3), 0))
			if err != nil {
				t.Fatal(err)
			}
			if version == RequestAuthorityVersion {
				if a.Allowed || a.Authority.Reason != "authority_request_not_admitted" {
					t.Fatalf("v1 changed: %+v", a.Authority)
				}
			} else if !a.Allowed || a.Authority.RequestObservationID != admission.ObservationID || a.Authority.RequestConsumedSequence != 1 {
				t.Fatalf("nonadjacent binding: %+v", a.Authority)
			}
			if _, err = Replay(t.Context(), path, DefaultLimits()); err != nil {
				t.Fatal(err)
			}
		})
	}
}

// All version changes are inputs before provisioning; no retained repair occurs.
func bindingV2Fixture(t *testing.T, mutate func(*DelegationClaim, *AuthorityUse, *Provision)) Provision {
	p := requestAuthorityFixture(t, func(c *DelegationClaim, u *AuthorityUse, p *Provision) {
		c.Version = RequestAuthorityBindingVersion
		u.Version = RequestAuthorityBindingVersion
		if mutate != nil {
			mutate(c, u, p)
		}
	})
	p.Composition.RequestAuthority = RequestAuthorityBindingVersion
	return p
}

func TestRequestBindingFutureSelectionSticky(t *testing.T) {
	for _, decisionFirst := range []bool{false, true} {
		t.Run(fmt.Sprint(decisionFirst), func(t *testing.T) {
			p := bindingV2Fixture(t, func(c *DelegationClaim, u *AuthorityUse, p *Provision) {
				c.ServiceOperationID = identity("A", int64(1), int64(1), 0)
			})
			authorityReplaceProgram(t, &p, "A", func(p *Program) { p.Code[0], p.Code[1] = p.Code[1], p.Code[0] })
			s, path := messageStore(t, p)
			if _, err := s.Commit(t.Context(), current(t, s, "B").Authority(), ""); err != nil {
				t.Fatal(err)
			}
			if d := authorizedDecision(t, s, committedRequest(t, s, "B")); !d.Accepted {
				t.Fatal(d)
			}
			admission, err := s.DeliverMessage(t.Context(), identity("B", int64(1), int64(1), 0), "genuine")
			if err != nil {
				t.Fatal(err)
			}
			if _, err = s.Commit(t.Context(), current(t, s, "A").Authority(), ""); err != nil {
				t.Fatal(err)
			}
			id := identity("A", int64(1), int64(1), 0)
			var before ExternalAttempt
			if decisionFirst {
				before, _, err = s.prepareExternal(t.Context(), id)
				if err != nil {
					t.Fatal(err)
				}
			}
			if _, err = s.Commit(t.Context(), current(t, s, "A").Authority(), admission.ObservationID); err != nil {
				t.Fatal(err)
			}
			after, _, err := s.prepareExternal(t.Context(), id)
			if err != nil {
				t.Fatal(err)
			}
			if after.Allowed || after.Authority.Reason != "authority_request_not_selected" || after.Authority.RequestObservationID != "" || after.Authority.RequestConsumedSequence != 0 {
				t.Fatal(after.Authority)
			}
			if decisionFirst && !reflect.DeepEqual(before, after) {
				t.Fatal("later selection repaired refusal")
			}
			if _, err = Replay(t.Context(), path, DefaultLimits()); err != nil {
				t.Fatal(err)
			}
			s.Close()
			reopened, err := Open(t.Context(), path, DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			defer reopened.Close()
			retained, _, err := reopened.prepareExternal(t.Context(), id)
			if err != nil || !reflect.DeepEqual(after, retained) {
				t.Fatal("reopen changed refusal", err)
			}
		})
	}
}

func TestRequestBindingDuplicateTransportAndIndex(t *testing.T) {
	s, path, _, a := authorityReady(t, bindingV2Fixture(t, nil), true)
	original := a.Authority.RequestObservationID
	if _, err := s.DeliverMessage(t.Context(), identity("B", int64(1), int64(1), 0), "another transport 🧭"); err != nil {
		t.Fatal(err)
	}
	if _, err := Replay(t.Context(), path, DefaultLimits()); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := s.conn.QueryRowContext(t.Context(), "SELECT count(*) FROM admissions WHERE observation_id=?", original).Scan(&count); err != nil || count != 2 {
		t.Fatal(count, err)
	}
	var planID, parent, unused int
	var detail string
	if err := s.conn.QueryRowContext(t.Context(), "EXPLAIN QUERY PLAN SELECT id FROM admissions WHERE observation_id=? ORDER BY id", original).Scan(&planID, &parent, &unused, &detail); err != nil || !strings.Contains(detail, "admissions_observation_id") {
		t.Fatal(detail, err)
	}
	t.Log(detail)
}

func TestRequestBindingVersionInputs(t *testing.T) {
	for _, name := range []string{"use", "claim", "profile"} {
		t.Run(name, func(t *testing.T) {
			p := bindingV2Fixture(t, func(c *DelegationClaim, u *AuthorityUse, p *Provision) {
				if name == "use" {
					u.Version = RequestAuthorityVersion
				}
				if name == "claim" {
					c.Version = RequestAuthorityVersion
				}
			})
			if name == "profile" {
				p.Composition.RequestAuthority = RequestAuthorityVersion
			}
			s, err := Create(t.Context(), filepath.Join(t.TempDir(), "mixed.sqlite"), p, DefaultLimits())
			if s != nil {
				s.Close()
			}
			if err == nil {
				t.Fatal("mixed version accepted")
			}
		})
	}
}

func TestRequestBindingExactOriginalAfterEqualPayload(t *testing.T) {
	for _, selectOriginal := range []bool{false, true} {
		t.Run(fmt.Sprint(selectOriginal), func(t *testing.T) {
			p := bindingFixture(t, RequestAuthorityBindingVersion)
			authorityReplaceProgram(t, &p, "B", func(p *Program) { p.Code[2].Effect.Obligation = Lit(Text("request")) })
			authorityReplaceProgram(t, &p, "A", func(p *Program) {
				m := *p.Code[0].Match
				p.Code[1].Match = &m
				if !selectOriginal {
					p.Code[0] = Instruction{Op: "set", Dst: "unrelated", A: Lit(Int(1))}
				}
			})
			s, path := messageStore(t, p)
			if _, err := s.Commit(t.Context(), current(t, s, "B").Authority(), ""); err != nil {
				t.Fatal(err)
			}
			if d := authorizedDecision(t, s, committedRequest(t, s, "B")); !d.Accepted {
				t.Fatal(d)
			}
			original, err := s.DeliverMessage(t.Context(), identity("B", int64(1), int64(1), 0), "original")
			if err != nil {
				t.Fatal(err)
			}
			selected := ""
			if selectOriginal {
				selected = original.ObservationID
			}
			if _, err = s.Commit(t.Context(), current(t, s, "A").Authority(), selected); err != nil {
				t.Fatal(err)
			}
			if _, err = s.Commit(t.Context(), current(t, s, "B").Authority(), ""); err != nil {
				t.Fatal(err)
			}
			other, err := s.DeliverMessage(t.Context(), identity("B", int64(1), int64(3), 0), "equal-payload-other")
			if err != nil {
				t.Fatal(err)
			}
			if _, err = s.Commit(t.Context(), current(t, s, "A").Authority(), other.ObservationID); err != nil {
				t.Fatal(err)
			}
			if _, err = s.Commit(t.Context(), current(t, s, "A").Authority(), ""); err != nil {
				t.Fatal(err)
			}
			a, _, err := s.prepareExternal(t.Context(), identity("A", int64(1), int64(3), 0))
			if err != nil {
				t.Fatal(err)
			}
			if a.Allowed != selectOriginal {
				t.Fatal(a.Authority)
			}
			if selectOriginal {
				if a.Authority.RequestObservationID != original.ObservationID {
					t.Fatal("latest receive substituted")
				}
			} else if a.Authority.Reason != "authority_request_not_selected" {
				t.Fatal(a.Authority)
			}
			if _, err = Replay(t.Context(), path, DefaultLimits()); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestRequestBindingMalformedOpenDoesNotRenew(t *testing.T) {
	for _, tc := range []struct{ name, sql string }{
		{"missing_index", "DROP INDEX admissions_observation_id"},
		{"wrong_index", "DROP INDEX admissions_observation_id; CREATE INDEX admissions_observation_id ON admissions(transport_id)"},
		{"unique_index", "DROP INDEX admissions_observation_id; CREATE UNIQUE INDEX admissions_observation_id ON admissions(observation_id)"},
		{"metadata", "UPDATE metadata SET schema_version=13"},
		{"pragma", "PRAGMA user_version=13"},
		{"profile", "UPDATE composition_profile SET descriptor=replace(cast(descriptor AS text),'alan/request-authority/v2','alan/request-authority/v1')"},
		{"observation", "UPDATE observations SET consumed_sequence=2 WHERE consumed_sequence IS NOT NULL"},
		{"admission", "DELETE FROM admissions"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, path, _, _ := authorityReady(t, bindingV2Fixture(t, nil), true)
			if _, err := s.conn.ExecContext(t.Context(), tc.sql); err != nil {
				t.Fatal(err)
			}
			if err := s.Close(); err != nil {
				t.Fatal(err)
			}
			before, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			opened, err := Open(t.Context(), path, DefaultLimits())
			if opened != nil {
				opened.Close()
			}
			if err == nil {
				t.Fatal("malformed ledger accepted")
			}
			after, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(before, after) {
				t.Fatal("failed Open changed immutable store")
			}
		})
	}
}

func TestRequestBindingConflictAndOldReader(t *testing.T) {
	for _, conflict := range []bool{false, true} {
		t.Run(fmt.Sprint(conflict), func(t *testing.T) {
			s, path, _, a := authorityReady(t, bindingV2Fixture(t, nil), true)
			op, reason, err := s.registerDispatch(t.Context(), a.Intent.ID)
			if err != nil || reason != "" {
				t.Fatal(reason, err)
			}
			s.finishDispatch(op, nil)
			if conflict {
				receipt := resourcefixture.Receipt{Key: a.Request.Key, Profile: a.Request.Profile, Delta: a.Request.Delta, Mutation: 1, Total: 1, Process: "controlled-test", PID: 1}
				if err = s.recordExternalReceipt(t.Context(), a, receipt); err != nil {
					t.Fatal(err)
				}
				receipt.Total++
				if err = s.recordExternalReceipt(t.Context(), a, receipt); !errors.Is(err, ErrConflict) {
					t.Fatal(err)
				}
			}
			want := 15
			if conflict {
				want = 16
			}
			version, err := storedSchemaNumber(t.Context(), s.conn)
			if err != nil || version != want {
				t.Fatal(version, err)
			}
			if _, err = Replay(t.Context(), path, DefaultLimits()); err != nil {
				t.Fatal(err)
			}
			if _, err = InspectExecution(t.Context(), path, DefaultLimits()); err != nil {
				t.Fatal(err)
			}
			s.Close()
			opened, err := Open(t.Context(), path, DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			opened.Close()
			t.Run("frozen_reader", func(t *testing.T) {
				binary := os.Getenv("ALAN_PHASE3B4_OLD_READER")
				if binary == "" {
					t.Skip("set ALAN_PHASE3B4_OLD_READER for frozen source reader")
				}
				copyPath := filepath.Join(t.TempDir(), "copy.sqlite")
				for _, suffix := range []string{"", "-wal", "-shm"} {
					b, err := os.ReadFile(path + suffix)
					if errors.Is(err, os.ErrNotExist) && suffix != "" {
						continue
					}
					if err != nil {
						t.Fatal(err)
					}
					if err = os.WriteFile(copyPath+suffix, b, 0600); err != nil {
						t.Fatal(err)
					}
				}
				before, err := os.ReadFile(copyPath)
				if err != nil {
					t.Fatal(err)
				}
				output := t.TempDir()
				cmd := exec.CommandContext(t.Context(), binary, "-test.run=^TestPhase3B3Owner$", "-test.v", "-test.timeout=2m")
				for _, v := range os.Environ() {
					if !strings.HasPrefix(v, "ALAN_") {
						cmd.Env = append(cmd.Env, v)
					}
				}
				cmd.Env = append(cmd.Env, "ALAN_PHASE3B3_CHILD=1", "ALAN_PHASE2_DB="+copyPath, "ALAN_PHASE3B3_OUTPUT="+output)
				out, err := cmd.CombinedOutput()
				if err == nil {
					t.Fatal("old reader accepted new format", string(out))
				}
				raw, err := os.ReadFile(filepath.Join(output, "open.json"))
				if err != nil {
					t.Fatal(string(out), err)
				}
				var result struct{ Error string }
				if err = json.Unmarshal(raw, &result); err != nil || result.Error != ErrVersion.Error() {
					t.Fatal(string(raw), err)
				}
				after, err := os.ReadFile(copyPath)
				if err != nil || !bytes.Equal(before, after) {
					t.Fatal("old reader changed store", err)
				}
			})
		})
	}
}

func TestRequestBindingHistoricalByteBounds(t *testing.T) {
	p := bindingV2Fixture(t, nil)
	for n := range p.Runs {
		if p.Runs[n].ID.Subject == "A" {
			p.Runs[n].Projection.Data["large"] = Text(strings.Repeat("x", DefaultLimits().MaxDataBytes+1))
		}
	}
	for n := range p.Programs {
		a := &p.Programs[n]
		old := a.ID
		// The supported raw artifact may omit zero-valued wire fields. Hash the
		// actual bytes, then let the loader reconstruct the larger typed encoding.
		a.Bytes = bytes.ReplaceAll(a.Bytes, []byte(`"Dst":"",`), nil)
		a.ID = programHash(a.Bytes)
		for j := range p.Runs {
			if p.Runs[j].ProgramID == old {
				p.Runs[j].ProgramID = a.ID
			}
		}
	}
	limits := Limits{2 << 20, 2 << 20, 2 << 20, 2 << 20}
	path := filepath.Join(t.TempDir(), "large.sqlite")
	s, err := Create(t.Context(), path, p, limits)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if _, err = s.Commit(t.Context(), current(t, s, "B").Authority(), ""); err != nil {
		t.Fatal(err)
	}
	if d := authorizedDecision(t, s, committedRequest(t, s, "B")); !d.Accepted {
		t.Fatal(d)
	}
	admission, err := s.DeliverMessage(t.Context(), identity("B", int64(1), int64(1), 0), "large-history")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Commit(t.Context(), current(t, s, "A").Authority(), admission.ObservationID); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Commit(t.Context(), current(t, s, "A").Authority(), ""); err != nil {
		t.Fatal(err)
	}
	a, _, err := s.prepareExternal(t.Context(), identity("A", int64(1), int64(2), 0))
	if err != nil || !a.Allowed {
		t.Fatal(a.Authority, err)
	}
	if _, err = Replay(t.Context(), path, limits); err != nil {
		t.Fatal(err)
	}
}

func TestRequestBindingOriginalCommitJoin(t *testing.T) {
	for _, subject := range []string{"A", "B"} {
		t.Run(subject, func(t *testing.T) {
			s, _, _, a := authorityReady(t, bindingV2Fixture(t, nil), true)
			sequence := int64(1)
			if subject == "A" {
				sequence = 2
			}
			if _, err := s.conn.ExecContext(t.Context(), "UPDATE history SET intents='[]' WHERE subject=? AND generation=1 AND sequence=?", subject, sequence); err != nil {
				t.Fatal(err)
			}
			i, err := loadIntent(t.Context(), s.conn, a.Intent.ID)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = evaluateEffectAuthority(t.Context(), s.conn, i, authorityCurrentPrefix); err == nil {
				t.Fatal("binding accepted intent absent from its committing reduction")
			}
		})
	}
}
