package semantics

import "slices"

// These facts are search-only. Runtime/reducer code never reads this structure.
type supervisionFacts struct {
	Seen map[string]bool
}

func inspectSupervision(before, after machine, e event, mon *monitor) string {
	if before.Lifecycle != nil {
		if after.Lifecycle == nil {
			return "supervision-ledger-retention"
		}
		for id := range before.Lifecycle.Authorized {
			if _, ok := after.Lifecycle.Authorized[id]; !ok {
				return "supervision-authorization-retention"
			}
		}
		for id := range before.Lifecycle.Results {
			if _, ok := after.Lifecycle.Results[id]; !ok {
				return "supervision-result-retention"
			}
		}
		for id := range before.Lifecycle.Notices {
			if _, ok := after.Lifecycle.Notices[id]; !ok {
				return "supervision-notice-retention"
			}
		}
	}
	if after.Lifecycle != nil {
		for id, au := range after.Lifecycle.Authorized {
			in := after.Intents[id]
			if in == nil || au.Request != in.Descriptor.Lifecycle {
				return "supervision-source-intent"
			}
			if before.Lifecycle != nil {
				if old, ok := before.Lifecycle.Authorized[id]; ok {
					if old != au {
						return "supervision-authorization-immutable"
					}
					continue
				}
			}
			q := requestValue(au.Request)
			r, s := before.Runs[runID(q.Requester, q.RequesterG)], before.Subjects[q.Requester]
			if e.Kind != authorizeLifecycle || r == nil || r.Projection.Status != "active" || s.Current != q.RequesterG || s.Retired || au.Owner != s.Owner || au.W != s.W {
				return "supervision-source-authorization"
			}
		}
		for id, result := range after.Lifecycle.Results {
			au, ok := after.Lifecycle.Authorized[id]
			if !ok || au.Request != result.Request {
				return "supervision-decision-authorization"
			}
			if before.Lifecycle != nil {
				if old, ok := before.Lifecycle.Results[id]; ok {
					if old != result {
						return "supervision-operation-immutable"
					}
					continue
				}
			}
			if e.Kind != decideLifecycle {
				return "supervision-decision-boundary"
			}
			q := requestValue(result.Request)
			if result.Accepted {
				// Independently enumerate fixture grants rather than invoke the runtime guard.
				granted := false
				if after.Config.Supervision != nil {
					for _, g := range after.Config.Supervision.Grants {
						if g.Requester == q.Requester && g.Target == q.Target && g.Operation == q.Operation && g.W == q.W {
							granted = true
						}
					}
				}
				if !granted {
					return "supervision-lifecycle-authority"
				}
				s := before.Subjects[q.Target]
				r := before.Runs[runID(q.Target, q.ExpectedG)]
				if s == nil || r == nil || s.W != q.W || s.Current != q.ExpectedG {
					return "supervision-decision-precondition"
				}
				if q.Operation == retry {
					eligible := r.Projection.Status == "failed" || r.Projection.Status == "abandoned" || r.Projection.Status == "cancelled" && after.Config.RetryCancelled
					if s.Retired || !eligible || result.Successor != q.ExpectedG+1 || after.Subjects[q.Target].Current != result.Successor {
						return "supervision-retry-transition"
					}
				}
			} else if canonical(before.Subjects) != canonical(after.Subjects) || canonical(before.Runs) != canonical(after.Runs) {
				return "supervision-rejection-mutated-target"
			}
		}
		for id, ad := range after.Lifecycle.Notices {
			if before.Lifecycle != nil {
				if old, ok := before.Lifecycle.Notices[id]; ok && old != ad {
					return "supervision-notice-binding"
				}
			}
			n := ad.Notice
			switch n.Kind {
			case "terminal":
				r := after.Runs[runID(n.Subject, n.G)]
				if r == nil || r.Outcome == nil || n.Classification != r.Outcome.Status || n.Logical != runID(n.Subject, n.G)+"/terminal" || !after.Config.Edges[n.Subject+">"+r.Outcome.Route] {
					return "supervision-terminal-provenance"
				}
			case "lifecycle":
				if after.Lifecycle.Results[n.Logical] != n.Result {
					return "supervision-result-provenance"
				}
			case "suspicion":
				if n.E < 1 || n.G < 1 || after.Runs[runID(n.Subject, n.G)] == nil {
					return "supervision-suspicion-provenance"
				}
			default:
				return "supervision-notice-kind"
			}
			existed := false
			if before.Lifecycle != nil {
				_, existed = before.Lifecycle.Notices[id]
			}
			if !existed {
				target := after.Config.Routes[n.Subject]
				if n.Kind == "lifecycle" {
					target = requestValue(n.Result.Request).Requester
				}
				s := before.Subjects[target]
				if s == nil || ad.Bound != runID(target, s.Current) {
					return "supervision-initial-service-binding"
				}
				r := before.Runs[ad.Bound]
				if ad.Accepted != (!s.Retired && r.Projection.Status == "active") {
					return "supervision-admission-eligibility"
				}
				if n.Kind == "suspicion" && (n.E != before.Runs[runID(n.Subject, n.G)].E || before.Runs[runID(n.Subject, n.G)].Projection.Status != "active") {
					return "supervision-suspicion-activation"
				}
			}
			count := 0
			for rid, r := range after.Runs {
				if o, ok := r.Inbox[id]; ok {
					count++
					if rid != ad.Bound || o.Notice != canonical(n) || o.Origin != "runtime" || o.Kind != n.Kind {
						return "supervision-inbox-provenance"
					}
				}
			}
			if ad.Accepted && count != 1 || !ad.Accepted && count != 0 {
				return "supervision-admission-cardinality"
			}
		}
	}
	if e.Kind != commit {
		return ""
	}
	p := before.Proposals[e.Slot]
	if p == nil {
		return ""
	}
	old := before.Runs[runID(p.A, p.G)]
	now := after.Runs[runID(p.A, p.G)]
	if old.N == now.N {
		return ""
	}
	if p.Observed != nil && p.Observed.Notice != "" {
		o := p.Observed
		n := noticeValue(o.Notice)
		if before.Lifecycle == nil {
			return "supervision-consume-provenance"
		}
		ad, ok := before.Lifecycle.Notices[o.ID]
		if !ok || !ad.Accepted || ad.Bound != runID(p.A, p.G) || canonical(ad.Notice) != o.Notice {
			return "supervision-consume-provenance"
		}
		d := old.Projection.Local
		if d == nil {
			return "supervision-uninitialized-consumption"
		}
		if (n.Kind == "terminal" || n.Kind == "suspicion") && (n.Subject != d.Names[0] || n.G != d.Values[childGeneration] || slices.Contains(d.Seen, n.Logical)) {
			return "supervision-failure-correlation"
		}
		if n.Kind == "lifecycle" {
			q := requestValue(n.Result.Request)
			if q.Requester != p.A || q.RequesterG != p.G || q.Obligation != d.Names[1] || q.ID != d.Pending || q.Target != d.Names[0] || q.ExpectedG != d.Values[childGeneration] || q.W != d.Values[revision] {
				return "supervision-old-lifecycle-result"
			}
		}
	}
	if after.Config.Supervision == nil {
		return ""
	}
	policy, ok := after.Config.Supervision.Policies[p.A]
	if !ok || now.Projection.Local == nil {
		return ""
	}
	d := now.Projection.Local
	count := 0
	decisions := map[string]bool{}
	for _, h := range now.History {
		if h.Intent == nil || h.Intent.Lifecycle == "" {
			continue
		}
		q := requestValue(h.Intent.Lifecycle)
		if q.Obligation != d.Names[1] {
			continue
		}
		count++
		key := runID(q.Target, q.ExpectedG)
		if decisions[key] {
			return "supervision-single-failure-decision"
		}
		decisions[key] = true
	}
	if count > policy.Limit {
		return "supervision-retry-budget"
	}
	if d.Values[spent] != count {
		return "supervision-budget-persistence"
	}
	if now.Projection.Status == "succeeded" {
		if d.Last == "" {
			return "supervision-success-without-child"
		}
		n := noticeValue(d.Last)
		if n.Kind != "terminal" || n.Subject != policy.Child || n.G != d.Values[childGeneration] || n.Classification != "succeeded" {
			return "supervision-success-without-child"
		}
	}
	if now.Projection.Status == "failed" && old.Projection.Status == "active" {
		valid := false
		if d.Last != "" {
			n := noticeValue(d.Last)
			valid = n.Kind == "lifecycle" && !n.Result.Accepted
			if n.Kind == "terminal" && n.Subject == policy.Child && n.G == d.Values[childGeneration] {
				valid = n.Classification == "cancelled" || n.Classification == "abandoned" && !policy.AllowAbandoned || (n.Classification == "failed" || n.Classification == "abandoned" && policy.AllowAbandoned) && count >= policy.Limit
			}
		}
		if !valid {
			return "supervision-unjustified-escalation"
		}
	}
	if p.Intent != nil && p.Intent.Lifecycle != "" {
		prev := old.Projection.Local
		if prev == nil || prev.Last == "" {
			return "supervision-policy-evidence"
		}
		n := noticeValue(prev.Last)
		if n.Kind != "terminal" || n.Subject != policy.Child || n.G != prev.Values[childGeneration] || !(n.Classification == "failed" || n.Classification == "abandoned" && policy.AllowAbandoned) {
			return "supervision-policy-evidence"
		}
	}
	return ""
}
