//go:build integration

package durabletest

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/JakeFAU/alan/runtime"
)

func TestPhase3B4CampaignSecondService(t *testing.T) {
	for _, name := range []string{"N12", "N10b"} {
		t.Run(name, func(t *testing.T) {
			dir := bindingDirectory(t, name)
			ca, ua, pa := reciprocalEndpoint(t, dir, "a", nil)
			var endpointCut *barrier
			if name == "N10b" {
				endpointCut = &barrier{Boundary: "resource.applied", Phase: "post-commit", Subject: "a", Sequence: 4}
			}
			cb, ub, pb := reciprocalEndpoint(t, dir, "b", endpointCut)
			p, m := bindingFixture(t, ua, ub, 0, nil, false)
			p.Composition.Roots[0].Limits.MaxEffects = 2
			reciprocalEdit(t, &p, "a", func(program map[string]any) {
				code := program["Code"].([]any)
				reciprocalShift(code, 4, 1)
				raw, _ := json.Marshal(code[3])
				var second map[string]any
				json.Unmarshal(raw, &second)
				if name == "N10b" {
					root := m.Reciprocal.BAServiceRoot
					root.Run = m.Reciprocal.A
					root.RootRun = m.Reciprocal.A
					root.Permission.Source = "a"
					p.Permissions = append(p.Permissions, root.Permission)
					second["Effect"].(map[string]any)["Authority"] = runtime.AuthorityUse{Version: runtime.RequestAuthorityBindingVersion, Path: "own", Root: root}
				}
				program["Code"] = append(append(append([]any{}, code[:4]...), second), code[4:]...)
				if name == "N10b" {
					program["Code"] = append(append([]any{}, code[:4]...), second, reciprocalInput("capacity-hold"))
				}
			})
			baseline3b3Write(t, dir, "provision.json", p)
			baseline3b3Write(t, dir, "manifest.json", m)
			db := filepath.Join(t.TempDir(), "runtime.sqlite")
			s, err := runtime.Create(t.Context(), db, p, runtime.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			var snap *runtime.ExecutionSnapshot
			c := bindingCase{Name: name, Manifest: m, Schedule: "automatic-run-two-distinct-service-intents"}
			if name == "N12" {
				_, err = bindingRun(t, s, dir, 256)
				s.Close()
				if err != nil {
					t.Fatal(err)
				}
				snap = bindingCapture(t, db, filepath.Join(dir, "final"))
				c.Captures = []bindingCaptureRef{{Ordinal: 1, Label: "final", Directory: "final"}}
				c.Classification = "second-service-mismatch"
			} else {
				s.Close()
				initial := bindingCapture(t, db, filepath.Join(dir, "initial"))
				owner := bindingOwner(t, db, filepath.Join(dir, "owner"), &barrier{Boundary: "runner.waiting", Phase: "in-flight"}, filepath.Join(dir, "manifest.json"), false)
				endpointAck := reciprocalAwait(t, cb)
				ack := reciprocalAwait(t, owner)
				baseline3b3Write(t, dir, "acknowledged-capacity-cut.json", map[string]any{"Endpoint": endpointAck, "Owner": ack, "Killed": false})
				opened := bindingReadCapture(t, filepath.Join(dir, "owner", "post-open"))
				bindingRenewal(t, initial, opened)
				snap = bindingCaptureLive(t, db, filepath.Join(dir, "occupied"))
				c.Captures = []bindingCaptureRef{{Ordinal: 1, Label: "initial", Directory: "initial"}, {Ordinal: 2, Predecessor: "initial", Label: "post-open", Directory: "owner/post-open", OwnerPID: ack.PID}, {Ordinal: 3, Predecessor: "owner/post-open", Label: "acknowledged-capacity-hold", Directory: "occupied", OwnerPID: ack.PID}}
				c.Classification = "occupied-inflight-blocks-second-reservation"
				if _, err = cb.release.Write([]byte("c")); err != nil {
					t.Fatal(err)
				}
				if _, err = owner.release.Write([]byte("c")); err != nil {
					t.Fatal(err)
				}
				phaseFinish(t, owner)
				final := bindingCapture(t, db, filepath.Join(dir, "final"))
				secondAfter, _ := final.Effect(`["a",1,5,0]`)
				if secondAfter == nil || secondAfter.ExternalAttempt == nil || !secondAfter.ExternalAttempt.Allowed || secondAfter.ExternalReceipt == nil || len(secondAfter.AuthorityAdmissions) != 1 || !secondAfter.AuthorityAdmissions[0].Allowed {
					t.Fatal("second operation not otherwise valid", secondAfter)
				}
				c.Captures = append(c.Captures, bindingCaptureRef{Ordinal: 4, Predecessor: "occupied", Label: "after-capacity-release", Directory: "final", OwnerPID: ack.PID})
				baseline3b3Write(t, dir, "capacity-assertions.json", map[string]any{"MaxEffects": 2, "MaxInFlight": 1, "OriginalService": m.Reciprocal.BAService, "AuxiliaryService": `["a",1,5,0]`, "AuxiliaryAuthority": "explicit own exact tuple, added before Create", "AfterRelease": secondAfter, "PrincipalCompletionClaim": false})

			}
			first, _ := snap.Effect(`["a",1,4,0]`)
			second, _ := snap.Effect(`["a",1,5,0]`)
			if first == nil || second == nil || first.ExternalAttempt == nil || !first.ExternalAttempt.Allowed {
				t.Fatal(first, second)
			}
			a, _ := snap.Operation(first.OperationID)
			b, _ := snap.Operation(second.OperationID)
			ap, _ := json.Marshal(a.Payload)
			bp, _ := json.Marshal(b.Payload)
			if string(ap) != string(bp) {
				t.Fatal("service payloads differ")
			}
			if name == "N12" {
				if second.ExternalAttempt == nil || second.ExternalAttempt.Allowed || second.ExternalAttempt.Authority.Reason != "authority_service_mismatch" || second.ExternalReceipt != nil || len(second.AuthorityAdmissions) != 0 {
					t.Fatal(second)
				}
			} else {
				if second.ExternalAttempt != nil || first.ExternalReceipt != nil || len(first.AuthorityAdmissions) != 1 {
					t.Fatal("capacity cut", first, second)
				}
			}
			endpoints := bindingEndpoints(t, dir, pa, pb, ca, cb)
			wantContacts := int64(1)
			if name == "N10b" {
				wantContacts = 2
			}
			if endpoints["a"].Contacts != 0 || endpoints["b"].Contacts != wantContacts || int64(len(endpoints["b"].Applications)) != wantContacts {
				t.Fatal(endpoints)
			}
			bindingIndex(t, dir, c, p)
		})
	}
}
