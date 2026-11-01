package effectfulcomposition

import "fmt"

type Step struct {
	Action string
	To     State
}
type offer func(string, State)

func role(i int) string                 { return []string{"L", "R", "X"}[i] }
func label(i int, action string) string { return role(i) + "." + action }
func Next(p Profile, s State) []Step {
	out := []Step{}
	add := func(a string, n State) { out = append(out, Step{Action: a, To: n}) }
	if p.Slice == "launch" {
		launch(p, s, add)
		return out
	}
	if p.Case == "slow-read" && !s.TreeStep && s.S[0].Query[0].Phase == "spent" {
		n := s
		n.TreeStep = true
		add("pure.private-commit", n)
	}
	if delayed(p) && s.World[1].Role == 0 {
		q := s.S[1].Query
		release := p.Case == "absent-refresh" && q[0].Phase == "selected" || p.Case == "late-application" && q[1].Phase == "selected" || p.Case == "loss-refresh" && s.Losses > 0
		if release {
			n := s
			apply(&n, p, 1)
			add("R.apply-held-packet", n)
		}
	}
	if s.Owner == 0 {
		n := s
		n.Owner = 1
		n.Epoch++
		add("owner.open", n)
		return out
	}
	if p.Slice == "read" {
		for i := range s.S {
			if i == 1 && s.S[0].Report != "booked" {
				continue
			}
			reader(p, s, i, add)
			report(p, s, i, add)
			if s.S[i].Original != "" && !s.S[i].Published {
				n := s
				n.S[i].Published = true
				add(label(i, "publish-original"), n)
			}
		}
		if p.Case == "competing" {
			competitor(p, s, add)
		}
	}
	// This cut services both report dispatches and all eligible publications
	// before the parent join. A report does not guarantee an original exists.
	for _, v := range s.S {
		if v.Report != "booked" || v.Original != "" && !v.Published {
			return out
		}
	}
	if p.Case == "competing" && s.Extra.Phase != "selected" {
		return out
	}
	join(p, s, add)
	return out
}
func launch(p Profile, s State, add offer) {
	active := 0
	for _, v := range s.S {
		if v.Mut == "admitted" || v.Mut == "called" || v.Mut == "packet" || v.Mut == "applied" {
			active++
		}
	}
	for i, v := range s.S {
		n := s
		w := &n.S[i]
		emit := func(a string) { add(label(i, a), n) }
		if !v.Closed && (p.Capacity == 1 || v.Mut == "admitted" || v.Mut == "called" || v.Mut == "packet" || v.Mut == "applied" || v.Mut == "handled") {
			w.Closed = true
			emit("end-after-handoff")
			n = s
		}
		if !v.Q && v.Closed && (v.Mut == "handled" || v.Mut == "denied") {
			w.Q = true
			emit("Q")
			n = s
		}
		switch v.Mut {
		case "requested":
			if active < p.Capacity {
				if v.Closed {
					w.Mut = "first-denied"
					emit("deny-first-permission")
				} else {
					n.Cost[i].Mut++
					w.Mut = "reserved"
					emit("allow-and-reserve")
				}
			}
		case "first-denied":
			w.Mut = "recorded-denial"
			w.Original = "source-denied"
			emit("record-source-denied")
		case "recorded-denial":
			w.Mut = "denied"
			w.Settled = true
			emit("settle-source-denied")
		case "granted":
			if active < p.Capacity {
				w.Mut = "reserved"
				n.Cost[i].Mut++
				emit("reserve-historical-grant")
			}
		case "reserved":
			if active < p.Capacity {
				if v.Closed {
					w.Mut = "denied"
					emit("refuse-current-gate")
				} else {
					w.Mut = "admitted"
					emit("admit")
				}
			}
		case "admitted":
			w.Mut = "called"
			emit("enter")
		case "called":
			n.Cost[i].Post++
			w.Mut = "packet"
			emit("ingress")
		case "packet":
			apply(&n, p, i)
			w.Mut = "applied"
			emit("apply")
		case "applied":
			w.Mut = "handled"
			emit("handle-no-result")
		}
	}
}

func lose(s State) State { s.Losses++; s.Owner = 0; return s }
func reader(p Profile, s State, i int, add offer) {
	v := s.S[i]
	for j := 0; j < p.Queries[i]; j++ {
		if j > 0 && v.Query[j-1].Phase != "selected" {
			continue
		}
		q := v.Query[j]
		n := s
		w := &n.S[i]
		z := &w.Query[j]
		c := &n.Cost[i]
		emit := func(a string) { add(fmt.Sprintf("%s.q%d.%s", role(i), j+1, a), n) }
		switch q.Phase {
		case "":
			z.Phase = "issued"
			emit("request")
		case "issued":
			z.Phase = "reserved"
			c.Read++
			emit("allow-and-reserve")
		case "reserved":
			if q.Ord == 2 {
				z.Phase = "retained"
				z.Status = "read_exhausted"
				emit("exhaust")
				break
			}
			if !v.Closed || !v.Q || v.Original != "" && !v.Settled {
				z.Phase = "retained"
				z.Status = "original_unsettled"
				emit("refuse")
				break
			}
			z.Phase = "spent"
			z.Ord++
			c.Ord++
			emit("spend")
		case "spent":
			if p.Case == "exhaust" && i == 1 {
				z.Phase = "reserved"
				n = lose(n)
				emit("kill-before-contact")
				break
			}
			if p.Case == "slow-read" && !s.TreeStep {
				break
			}
			if p.Case == "competing" && i == 0 && s.Extra.Phase != "spent" {
				break
			}
			if p.Case == "loss-refresh" && i == 1 && q.Ord == 2 && s.World[1].Role == 0 {
				break
			}
			if p.Case == "absent-refresh" && i == 1 && j == 1 && s.World[1].Role == 0 {
				break
			}
			z.Phase = "contact"
			c.Get++
			emit("lookup-ingress")
		case "contact":
			snapshot(&n, z, i, j)
			emit("snapshot")
		case "snapshot":
			if (p.Case == "loss-refresh" || p.Case == "lost-found") && i == 1 && q.Ord == 1 {
				spent := q.Ord
				*z = Query{Phase: "reserved", Ord: spent}
				n = lose(n)
				emit("kill-before-retain")
				break
			}
			z.Phase = "retained"
			if p.Case == "later-error" && i == 0 && j == 1 {
				z.Status = "transport_error"
				z.Adoption = ""
				z.Receipt = Receipt{}
				z.Serial = 0
				z.At = 0
				emit("retain-transport-error")
				break
			}
			decideRead(z, w, p.Case != "read-only" || i != 1)
			emit("retain-and-assess")
		case "retained":
			if p.Case == "import-kill" && i == 0 && s.Losses == 0 {
				n = lose(n)
				emit("kill-after-import")
				break
			}
			if q.Adoption == "adopted" && !v.Settled {
				w.Settled = true
				emit("settle-original")
				break
			}
			z.Phase = "settled"
			emit("settle-query")
		case "settled":
			z.Phase = "selected"
			if p.Case == "incompatible" && i == 0 && j == 1 {
				z.Receipt.Total++
			}
			remember(w, *z, j)
			emit("select")
		}
	}
}
func snapshot(s *State, q *Query, i, j int) {
	q.Phase = "snapshot"
	s.Serial++
	q.Serial = s.Serial
	q.At = s.Cost[0].Apply + s.Cost[1].Apply
	q.Receipt = s.World[i]
	q.Status = "not_found"
	if q.Receipt.Role != 0 {
		q.Status = "found"
	}
	s.Log[s.Serial-1] = Snapshot{Target: i + 1, Query: j + 1, Ordinal: q.Ord, At: q.At, Receipt: q.Receipt}
}
func decideRead(q *Query, v *Side, allowed bool) {
	switch {
	case q.Status != "found":
		q.Adoption = "none"
	case !allowed:
		q.Adoption = "denied"
	case v.Original == "":
		v.Original = "imported"
		q.Adoption = "adopted"
	case v.Original == "imported":
		q.Adoption = "duplicate"
	default:
		q.Adoption = "existing"
	}
}
func remember(w *Side, q Query, j int) {
	if q.Status == "conflict" || q.Adoption == "conflict" {
		w.Conflict = true
		return
	}
	if q.Status != "found" {
		return
	}
	if w.Witness == 0 {
		w.Proof = q.Receipt
		w.Witness = j + 1
	} else if w.Proof != q.Receipt {
		w.Conflict = true
	}
}

func competitor(p Profile, s State, add offer) {
	n := s
	z := &n.Extra
	v := s.S[0]
	switch s.Extra.Phase {
	case "":
		if v.Query[0].Phase == "spent" {
			z.Phase = "reserved"
			n.Cost[3].Read++
			add("X.request-and-reserve", n)
		}
	case "reserved":
		z.Phase = "spent"
		z.Ord = 1
		n.Cost[3].Ord++
		add("X.spend", n)
	case "spent":
		if v.Original == "imported" {
			z.Phase = "contact"
			n.Cost[3].Get++
			add("X.lookup-ingress", n)
		}
	case "contact":
		snapshot(&n, z, 0, 2)
		add("X.snapshot", n)
	case "snapshot":
		z.Phase = "retained"
		decideRead(z, &n.S[0], true)
		add("X.retain", n)
	case "retained":
		z.Phase = "settled"
		add("X.settle-query", n)
	case "settled":
		z.Phase = "selected"
		add("X.select", n)
	}
}
func report(p Profile, s State, i int, add offer) {
	n := s
	v := &n.S[i]
	switch v.Report {
	case "":
		if v.Query[p.Queries[i]-1].Phase == "selected" {
			v.Report = "sent"
			v.ReportTarget = i + 1
			v.ReportProof = v.Proof
			if p.Case == "invalid-role" && i == 1 {
				v.ReportProof = s.S[0].Proof
				v.ReportTarget = 1
			}
			add(label(i, "send-report"), n)
		}
	case "sent":
		v.Report = "allowed"
		add(label(i, "authorize-report"), n)
	case "allowed":
		v.Report = "delivered"
		add(label(i, "admit-report"), n)
	case "delivered":
		v.Report = "booked"
		add(label(i, "book-report"), n)
	}
}
