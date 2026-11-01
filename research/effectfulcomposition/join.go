package effectfulcomposition

// decision requires an actual selection of all five operands. The shared ledger
// test checks role-specific receipts in either lawful application order.
func decision(p Profile, s State) string {
	if s.Joined != 5 {
		return ""
	}
	sum := p.Deltas[0] + p.Deltas[1]
	first := 0
	if s.S[1].Proof.Mutation == 1 {
		first = 1
	}
	for i, v := range s.S {
		r := v.ReportProof
		if v.ReportTarget != i+1 {
			return "invalid_evidence"
		}
		if !v.Published || v.Original == "" || v.Conflict || v.Witness < 1 || r.Role != i+1 {
			return "invalid_evidence"
		}
		q := v.Query[v.Witness-1]
		if q.Phase != "selected" || q.Status != "found" || q.Receipt != r {
			return "invalid_evidence"
		}
		if q.Adoption != "adopted" && q.Adoption != "existing" && q.Adoption != "duplicate" {
			return "invalid_evidence"
		}
		if i == first {
			if r.Mutation != 1 || r.Total != p.Deltas[i] {
				return "invalid_evidence"
			}
		} else if r.Mutation != 2 || r.Total != sum {
			return "invalid_evidence"
		}
	}
	if Tree(p).Outcome == "answer" {
		return "applied"
	}
	return "applied_tree_diagnosed"
}
func join(p Profile, s State, add offer) {
	n := s
	if p.Case == "selection-kill" && s.Joined == 1 && s.Losses == 0 {
		add("owner.kill-after-selection", lose(n))
		return
	}
	if s.Joined < 5 {
		enabled := false
		action := ""
		switch s.Joined {
		case 0, 1:
			i := s.Joined
			enabled = s.S[i].Published
			action = label(i, "parent-select-original")
		case 2:
			enabled = Tree(p).Outcome != "pending" && s.TreeStep
			action = "P.select-tree"
		case 3, 4:
			i := s.Joined - 3
			enabled = s.S[i].Report == "booked"
			action = label(i, "parent-select-report")
		}
		if enabled {
			n.Joined++
			add(action, n)
		}
		return
	}
	if s.Decision == "" {
		n.Decision = decision(p, s)
		add("P.decide", n)
		return
	}
	if p.Case == "fresh-ack" && s.S[0].Ack == "sent" && s.Losses == 0 {
		add("owner.kill-before-ack-service", lose(n))
		return
	}
	manual := p.Case == "manual-ack"
	if !manual {
		// The turn that ended a root services ready publications before the
		// next turn. A new turn then services its sole pending internal ACK
		// before it can commit any actor's next ordinary instruction.
		if s.ParentEnd && !s.ParentRoot {
			n.ParentRoot = true
			add("P.publish-root", n)
			return
		}
		for i, v := range s.S {
			if v.ResolverEnd && !v.Root {
				n.S[i].Root = true
				add(label(i, "publish-root"), n)
				return
			}
		}
		for i, v := range s.S {
			w := &n.S[i]
			switch v.Ack {
			case "sent":
				w.Ack = "allowed"
				add(label(i, "authorize-ack"), n)
				return
			case "allowed":
				w.Ack = "delivered"
				add(label(i, "admit-ack"), n)
				return
			case "delivered":
				w.Ack = "booked"
				add(label(i, "book-ack"), n)
				return
			}
		}
	}
	for i, v := range s.S {
		n = s
		w := &n.S[i]
		emit := func(a string) { add(label(i, a), n) }
		switch v.Ack {
		case "":
			if !s.ParentEnd && (i == 0 || s.S[0].Ack != "" && (manual || s.S[0].Ack == "booked")) {
				w.Ack = "sent"
				emit("send-ack")
			}
		case "sent":
			if s.ParentEnd {
				w.Ack = "denied"
				emit("deny-ack")
			} else {
				w.Ack = "allowed"
				emit("authorize-ack")
			}
		case "allowed":
			w.Ack = "delivered"
			emit("admit-ack")
		case "denied":
			w.Ack = "refused"
			emit("refuse-ack")
		case "refused":
			w.Ack = "closed"
			emit("book-refusal")
		case "delivered":
			w.Ack = "booked"
			emit("book-ack")
		case "booked":
			if !v.AckSelected {
				w.AckSelected = true
				emit("select-ack")
			} else if !v.ResolverEnd {
				w.ResolverEnd = true
				emit("resolver-end")
			} else if !v.Root {
				w.Root = true
				emit("publish-root")
			}
		}
	}
	if !s.ParentEnd && s.S[1].Ack != "" && (manual || s.S[0].Ack == "booked" && s.S[1].Ack == "booked") {
		n = s
		n.ParentEnd = true
		add("P.end", n)
	}
	if s.ParentEnd && !s.ParentRoot {
		n = s
		n.ParentRoot = true
		add("P.publish-root", n)
	}
}

func Class(p Profile, s State) string {
	if s.ParentRoot && s.S[0].Root && s.S[1].Root {
		if p.Case == "competing" {
			return "designated-roots/" + s.Decision + "/extra-reader-selected"
		}
		return "all-roots/" + s.Decision
	}
	if s.ParentRoot && (s.S[0].Ack == "closed" || s.S[1].Ack == "closed") {
		return "parent-published/ack-refused"
	}
	if p.Slice == "launch" && s.S[0].Q && s.S[1].Q {
		return "both-originals-quiescent"
	}
	if s.S[0].Report == "booked" && s.S[1].Report == "booked" {
		if s.Joined == 1 && s.S[1].Original == "" {
			return "pending/original-R"
		}
		if s.Joined == 2 && Tree(p).Outcome == "pending" {
			return "pending/tree/" + Tree(p).Frontier
		}
	}
	return "prefix"
}
