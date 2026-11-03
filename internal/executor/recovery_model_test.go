package executor

import (
	"crypto/sha256"
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"
)

// Durable fields only. The model starts after both intents have been emitted
// and the root invocation authorized. Its root stays open for the entire cut.
// An ordinary source closure freezes its epoch. MemberEpoch is zero before
// admission. Each R represents owner loss followed by successful recovery.
type recoveryModelCut struct {
	Scoped                                                   bool
	Facts                                                    string
	SourceEpoch, RootEpoch, MemberEpoch, CloseEpoch          int
	AuthEpoch, AuthSequence, AuthGeneration, AuthRevision    int
	TargetEpoch, TargetGeneration, TargetRevision, Creations int
	AuthReason, Reason                                       string
}

func recoveryInitial(scoped bool) recoveryModelCut {
	return recoveryModelCut{Scoped: scoped, SourceEpoch: 1, RootEpoch: 1}
}

func (c recoveryModelCut) step(action rune) (recoveryModelCut, bool) {
	visible := c.Scoped
	switch action {
	case 'C':
		if c.Creations != 0 {
			return c, false
		}
		c.Creations, c.MemberEpoch = 1, 1
		visible = true
	case 'A':
		if c.AuthEpoch != 0 {
			return c, false
		}
		c.AuthEpoch, c.AuthGeneration, c.AuthRevision = c.SourceEpoch, 1, 1
		c.AuthReason, c.AuthSequence = "authorized", 2
		if c.CloseEpoch != 0 {
			c.AuthReason, c.AuthSequence = "source_closed", 3
		}
	case 'D':
		if c.AuthReason != "authorized" || c.Reason != "" {
			return c, false
		}
		c.Reason = "target_missing"
		if c.Creations != 0 {
			c.Reason, c.TargetEpoch, c.TargetGeneration, c.TargetRevision = "grant_denied", c.MemberEpoch, 1, 1
			visible = true
		}
	case 'X':
		if c.CloseEpoch != 0 {
			return c, false
		}
		c.CloseEpoch = c.SourceEpoch
	case 'R':
		if c.RootEpoch == 3 {
			return c, false
		}
		c.RootEpoch++
		if c.CloseEpoch == 0 {
			c.SourceEpoch++
		}
		if c.Creations != 0 {
			c.MemberEpoch++
		}
		visible = false
	default:
		panic("invalid model action")
	}
	if visible {
		c.Facts += string(action)
	}
	return c, true
}

// Explore every enabled action at every prefix, retaining the first shortest
// witness for each distinct projection without pruning alternative schedules.
func recoveryReachable() (map[recoveryModelCut]string, int) {
	cuts := map[recoveryModelCut]string{}
	schedules := 0
	var visit func(recoveryModelCut, string)
	visit = func(c recoveryModelCut, prefix string) {
		schedules++
		if old, found := cuts[c]; !found || len(prefix) < len(old) {
			cuts[c] = prefix
		}
		for _, action := range "CADXR" {
			if next, ok := c.step(action); ok {
				visit(next, prefix+string(action))
			}
		}
	}
	for _, scoped := range []bool{false, true} {
		visit(recoveryInitial(scoped), "")
	}
	return cuts, schedules
}

func TestRecoveryModel(t *testing.T) {
	reachable, schedules := recoveryReachable()
	compared := map[recoveryModelCut]bool{}
	var lines []string
	pass, fail := 0, 0
	check := func(c recoveryModelCut) {
		if compared[c] {
			return
		}
		compared[c] = true
		witness, want := reachable[c]
		if got := recoveryCutFeasible(c); got != want {
			t.Fatalf("independent oracle = %t; forward membership = %t (witness %q); cut %+v", got, want, witness, c)
		}
		lines = append(lines, fmt.Sprintf("%+v %t", c, want))
		if want {
			pass++
		} else {
			fail++
		}
	}
	for c := range reachable {
		check(c)
		// Each mutation is judged by membership, never presumed corrupt. Some
		// changes yield another reachable cut because independent order is lost.
		for _, field := range []func(*recoveryModelCut) *int{
			func(c *recoveryModelCut) *int { return &c.RootEpoch },
			func(c *recoveryModelCut) *int { return &c.SourceEpoch },
			func(c *recoveryModelCut) *int { return &c.CloseEpoch },
			func(c *recoveryModelCut) *int { return &c.MemberEpoch },
			func(c *recoveryModelCut) *int { return &c.AuthEpoch },
			func(c *recoveryModelCut) *int { return &c.AuthSequence },
			func(c *recoveryModelCut) *int { return &c.AuthGeneration },
			func(c *recoveryModelCut) *int { return &c.AuthRevision },
			func(c *recoveryModelCut) *int { return &c.TargetEpoch },
			func(c *recoveryModelCut) *int { return &c.TargetGeneration },
			func(c *recoveryModelCut) *int { return &c.TargetRevision },
			func(c *recoveryModelCut) *int { return &c.Creations },
		} {
			for value := range 5 {
				m := c
				*field(&m) = value
				check(m)
			}
		}
		for _, reason := range []string{"", "grant_denied", "target_missing", "closed"} {
			m := c
			m.Reason = reason
			check(m)
		}
		for _, reason := range []string{"", "authorized", "source_closed", "source_retired"} {
			m := c
			m.AuthReason = reason
			check(m)
		}
		for i := range len(c.Facts) {
			m := c
			m.Facts = c.Facts[:i] + c.Facts[i+1:]
			check(m)
			m.Facts = c.Facts[:i] + c.Facts[i:i+1] + c.Facts[i:]
			check(m)
			if i+1 < len(c.Facts) {
				m.Facts = c.Facts[:i] + c.Facts[i+1:i+2] + c.Facts[i:i+1] + c.Facts[i+2:]
				check(m)
			}
		}
		m := c
		m.Scoped = !m.Scoped
		check(m)
		if c.Reason == "grant_denied" {
			m = c
			m.Reason, m.TargetEpoch, m.TargetGeneration, m.TargetRevision = "target_missing", 0, 0, 0
			if !m.Scoped {
				m.Facts = strings.ReplaceAll(m.Facts, "D", "")
			}
			check(m)
		}
	}
	slices.Sort(lines)

	var result strings.Builder
	fmt.Fprintf(&result, "prefixes\treachable\tinfeasible\tsorted-cut-verdict-sha256\n%d\t%d\t%d\t%x\n", schedules, pass, fail, sha256.Sum256([]byte(strings.Join(lines, "\n"))))
	result.WriteString("scope\twriter_schedule\tmutation\tfacts\troot_epoch\tsource_epoch\tmember_epoch\tauth_epoch\ttarget_epoch\treason\tfeasible\n")
	for _, scoped := range []bool{false, true} {
		for _, example := range []struct {
			schedule string
			missing  bool
		}{{"ADCRR", false}, {"CADRR", true}, {"CRAD", true}, {"XRA", false}} {
			c := recoveryInitial(scoped)
			for _, action := range example.schedule {
				var ok bool
				c, ok = c.step(action)
				if !ok {
					t.Fatalf("invalid evidence schedule %q", example.schedule)
				}
			}
			mutation := "none"
			if example.missing {
				mutation = "missing"
				c.Reason, c.TargetEpoch, c.TargetGeneration, c.TargetRevision = "target_missing", 0, 0, 0
				if !scoped {
					c.Facts = strings.ReplaceAll(c.Facts, "D", "")
				}
			}
			_, feasible := reachable[c]
			scope := "unscoped"
			if scoped {
				scope = "scoped"
			}
			fmt.Fprintf(&result, "%s\t%s\t%s\t%s\t%d\t%d\t%d\t%d\t%d\t%s\t%t\n", scope, example.schedule, mutation, c.Facts, c.RootEpoch, c.SourceEpoch, c.MemberEpoch, c.AuthEpoch, c.TargetEpoch, c.Reason, feasible)
		}
	}
	golden, err := os.ReadFile("testdata/recovery-model.tsv")
	if err != nil || string(golden) != result.String() {
		t.Fatalf("recovery-model.tsv differs: %v\n%s", err, result.String())
	}
	t.Log(result.String())
}
