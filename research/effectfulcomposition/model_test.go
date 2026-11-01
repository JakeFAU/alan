package effectfulcomposition

import "testing"

func TestEveryProfile(t *testing.T) {
	for _, p := range Profiles() {
		t.Run(p.Name, func(t *testing.T) {
			g := Explore(p)
			if err := Check(g); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestOutcomesAndAccounting(t *testing.T) {
	got := map[string]int{}
	var intact, lost State
	claims := map[string]bool{}
	orders := map[int]bool{}
	for _, p := range Profiles() {
		g := Explore(p)
		for _, x := range Attacks(g) {
			claims[x.Claim] = true
		}
		for _, r := range g.Rows {
			if len(r.Edges) > 0 {
				continue
			}
			s := r.State
			if p.Slice == "launch" {
				if p.Name == "equal-order" {
					orders[s.World[0].Mutation] = true
					if s.Total != 2 || s.World[0].Role == s.World[1].Role {
						t.Fatal(p.Name, s)
					}
				}
				if s.S[1].Mut == "denied" {
					if p.Case == "cap-one-granted" && (s.S[1].Original != "" || s.Cost[1].Mut != 1) {
						t.Fatal(s)
					}
					if p.Case == "cap-one" && s.Cost[1].Mut == 0 && (s.S[1].Original != "source-denied" || !s.S[1].Settled) {
						t.Fatal(s)
					}
				}
				continue
			}
			if p.Slice == "tree" && p.Shape == "fork" {
				got[r.Class]++
				continue
			}
			switch p.Name {
			case "intact-found":
				intact = s
			case "lost-found":
				lost = s
			case "later-error":
				if s.S[0].Witness != 1 || s.S[0].Query[1].Status != "transport_error" || s.Decision != "applied" {
					t.Fatal(s)
				}
			case "retained-failure":
				if s.S[0].Original != "failure" || s.S[0].Query[0].Adoption != "existing" || s.Decision != "applied" {
					t.Fatal(s)
				}
			case "exhaust":
				if s.Cost[1].Read != 1 || s.Cost[1].Ord != 2 || s.Cost[1].Get != 0 || s.S[1].Query[0].Status != "read_exhausted" || s.Joined != 1 {
					t.Fatal(s)
				}
			case "invalid-role", "incompatible":
				if s.Decision != "invalid_evidence" {
					t.Fatal(p.Name, s)
				}
			case "import-kill", "selection-kill", "fresh-ack":
				if s.Epoch != 2 || s.Decision != "applied" {
					t.Fatal(p.Name, s)
				}
			}
		}
	}
	if len(claims) != 9 || len(orders) != 2 || got["all-roots/applied"] != 8 || got["all-roots/applied_tree_diagnosed"] != 64 || got["pending/tree/await-left"] != 32 || got["pending/tree/left-selected/await-right"] != 24 {
		t.Fatal(claims, orders, got)
	}
	if intact.World != lost.World || intact.Decision != lost.Decision || intact.S[1].Proof != lost.S[1].Proof || intact.S[1].Original != lost.S[1].Original || intact.Cost[1].Read != lost.Cost[1].Read || intact.Cost[1].Ord+1 != lost.Cost[1].Ord || intact.Cost[1].Get+1 != lost.Cost[1].Get {
		t.Fatalf("incorrect coupling: %+v %+v", intact, lost)
	}
}

func TestAutomaticDeliveryAndRootService(t *testing.T) {
	for _, p := range Profiles() {
		if p.Name != "intact-found" {
			continue
		}
		g := Explore(p)
		for _, r := range g.Rows {
			want := ""
			s := r.State
			for i, v := range s.S {
				switch v.Ack {
				case "sent":
					want = label(i, "authorize-ack")
				case "allowed":
					want = label(i, "admit-ack")
				case "delivered":
					want = label(i, "book-ack")
				}
				if v.ResolverEnd && !v.Root {
					want = label(i, "publish-root")
				}
			}
			if s.ParentEnd && !s.ParentRoot {
				want = "P.publish-root"
			}
			if want != "" && (len(r.Edges) != 1 || r.Edges[0].Action != want) {
				t.Fatalf("row %d: stock turn owes %s, got %+v", r.ID, want, r.Edges)
			}
		}
	}
}
