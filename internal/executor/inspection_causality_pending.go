package executor

// An unchanged failed child publication can coexist with work below it. A
// successful ancestor must still join that work. Mark incomplete subtrees once
// upwards, even when different parents are waiting on the same retained cut.
func (c *causalCut) invocationDependencies() error {
	incomplete := map[string]bool{}
	for _, src := range c.intents {
		scope := c.runs[src.Run].OwningInvocationID
		if scope == "" {
			continue
		}
		switch src.Kind {
		case "external", "adapter":
			if c.knownExternalOrder(src.ID) == 0 {
				incomplete[scope] = true
			}
		case "invocation":
			child := c.invocations[src.ID]
			if child.Admission == nil || (child.Admission.Accepted && child.Outcome == nil) {
				incomplete[scope] = true
			}
		}
	}
	for id := range incomplete {
		for parent := c.byID[id].Admission.Invocation.ParentInvocationID; parent != "" && !incomplete[parent]; parent = c.byID[parent].Admission.Invocation.ParentInvocationID {
			incomplete[parent] = true
		}
	}
	for _, v := range c.invocations {
		if v.Admission == nil || v.Outcome != nil {
			continue
		}
		depend := func(kind, original string, member *RunID) {
			v.Dependencies = append(v.Dependencies, PendingObligation{Kind: kind, OperationID: v.OperationID, Run: v.Caller, DependsOn: original, DependsOnRun: member})
		}
		f := v.Freeze
		if v.Closure != nil {
			f = &v.Closure.Freeze
		}
		if v.Admission.Accepted && f == nil {
			for _, id := range v.Admission.Members {
				switch c.runs[id].Projection.Status {
				case "open":
					depend("invocation_member", "", &id)
				case "succeeded":
				default:
					return corrupt("terminal member missing scope freeze")
				}
			}
			if len(v.Dependencies) == 0 {
				return corrupt("all members succeeded without original scope freeze")
			}
		}
		if f != nil {
			for _, id := range f.ReplyIntents {
				if _, recorded := c.replies[id]; !recorded {
					depend("invocation_reply_admission", id, nil)
				}
			}
			if f.Status == "succeeded" {
				for _, id := range f.InvocationIntents {
					child := c.invocations[id]
					switch {
					case child.Admission == nil:
						depend("descendant_admission", id, nil)
					case child.Admission.Accepted && child.Outcome == nil:
						depend("descendant_publication", id, nil)
					case child.Admission.Accepted && incomplete[child.Admission.Invocation.ID]:
						depend("descendant_work", id, nil)
					}
				}
				for _, id := range f.ExternalIntents {
					if c.knownExternalOrder(id) == 0 {
						depend("invocation_external_outcome", id, nil)
					}
				}
			}
		}
		if len(v.Dependencies) == 0 {
			depend("invocation_publication", v.OperationID, nil)
		}
		c.snap.Pending = append(c.snap.Pending, v.Dependencies...)
	}
	return nil
}
