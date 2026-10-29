package substitution

import "fmt"

// actualProbeGraph hand-transcribes the source/hash-bound retained probe programs
// in probe-artifacts. It shares only the declared protocol/environment rules
// with K1, not CandidateGraph's B instructions or K1's complement oracle.
// Entry PCs are P's actual PCs for p/weak-b; q uses those same control locations
// plus register7 to distinguish its inserted scratch=0 Set. Q's actual PC is
// normalizedPC+1 after that Set (except the initial Receive). This avoids
// identifying instruction addresses that differ only by the inserted Set.
func actualProbeGraph(profile string) Graph {
	helperNeeded := profile != "weak-a"
	zero := profile == "weak-a" || profile == "weak-b"
	extra := profile == "q"
	return func(c Config, s State) []Edge {
		var edges []Edge
		for _, e := range Successors(c, Reference, s) {
			i := -1
			for j := range s.Ops {
				if e.Event.Operation == name(j) {
					i = j
				}
			}
			actor := i
			if e.Event.Kind == "outcome_consume" && i >= 0 {
				actor = s.Ops[i].Source
			}
			if e.Event.Kind == "invoke_commit" && i == helper {
				actor = entry
			}
			if e.Environment || (actor != entry && actor != helper) {
				edges = append(edges, e)
				continue
			}
			if actor == helper && !helperNeeded {
				continue
			}
			pc := s.PrivatePC[actor]
			o := s.Ops[actor]
			allow := false
			switch e.Event.Kind {
			case "invoke_commit":
				if i == entry {
					allow = true
				} else if i == helper && pc == 1 && helperNeeded && (!extra || s.PrivateRegisters[entry][7] == 1) {
					allow = true
					e.Next.PrivatePC[entry] = 2
				}
			case "request_consume":
				if pc == 0 {
					allow = true
					e.Next.PrivatePC[actor] = 1
					e.Next.PrivateRegisters[actor][0] = o.Input.ID
					e.Next.PrivateRegisters[actor][1] = o.Input.Value
				}
			case "choose_local":
				allow = actor == entry && !helperNeeded
			case "choose_helper":
				allow = actor == entry && helperNeeded
			case "outcome_consume":
				if i == helper && helperNeeded && pc == 2 {
					allow = true
					e.Next.PrivatePC[entry] = 3
					h := s.Ops[helper]
					if h.Status == "succeeded" {
						e.Next.PrivateRegisters[entry][3] = 1
					}
					if h.Present {
						e.Next.PrivateRegisters[entry][4] = 1
					}
					e.Next.PrivateRegisters[entry][5] = h.ID
					e.Next.PrivateRegisters[entry][6] = h.Value
				}
			case "reply_commit":
				id, value := 0, 0
				if actor == entry && ((!helperNeeded && pc == 3) || (helperNeeded && pc == 6)) {
					allow = true
					id = s.PrivateRegisters[entry][5]
					value = s.PrivateRegisters[entry][6]
					e.Next.PrivatePC[entry] = pc + 1
				}
				if actor == helper && ((zero && pc == 2) || (!zero && pc == 5)) {
					allow = true
					id = s.PrivateRegisters[helper][0]
					value = s.PrivateRegisters[helper][2]
					e.Next.PrivatePC[helper] = pc + 1
				}
				if allow {
					e.Next.Ops[actor].ReplyID = id
					e.Next.Ops[actor].ReplyValue = value
					e.Event.ID = id
					e.Event.Value = value
				}
			case "terminal_freeze":
				desired := none
				if actor == entry {
					if (!helperNeeded && pc == 4) || (helperNeeded && pc == 7) {
						desired = success
					}
					if helperNeeded && pc == 5 {
						desired = failure
					}
				}
				if actor == helper && ((zero && pc == 3) || (!zero && pc == 6)) {
					desired = success
				}
				if desired != none {
					allow = true
					e.Next.Ops[actor].Freeze = desired
				}
			}
			if allow {
				edges = append(edges, e)
			}
		}
		for _, i := range []int{entry, helper} {
			o := s.Ops[i]
			if o.Admission != 1 || o.Freeze != none || o.PC == receive || (i == entry && o.PC == choose) || (i == helper && !helperNeeded) {
				continue
			}
			pc := s.PrivatePC[i]
			next := -1
			n := s
			if i == entry {
				if !helperNeeded {
					switch pc {
					case 1:
						n.PrivateRegisters[i][5] = s.PrivateRegisters[i][0]
						next = 2
					case 2:
						n.PrivateRegisters[i][6] = 0
						next = 3
					}
				} else {
					switch pc {
					case 1:
						if extra && s.PrivateRegisters[i][7] == 0 {
							n.PrivateRegisters[i][7] = 1
							next = 1
						}
					case 3:
						next = 5
						if s.PrivateRegisters[i][3] == 1 {
							next = 4
						}
					case 4:
						next = 6
					}
				}
			} else if zero {
				if pc == 1 {
					n.PrivateRegisters[i][2] = 0
					next = 2
				}
			} else {
				switch pc {
				case 1:
					next = 4
					if s.PrivateRegisters[i][1] == 0 {
						next = 2
					}
				case 2:
					n.PrivateRegisters[i][2] = 1
					next = 3
				case 3:
					next = 5
				case 4:
					n.PrivateRegisters[i][2] = 0
					next = 5
				}
			}
			if next >= 0 {
				n.PrivatePC[i] = next
				edges = append(edges, Edge{Observation{Kind: "private_step", Operation: name(i), Role: o.Role}, n, false})
			}
		}
		return edges
	}
}

// CheckProbeContext checks maximal outcomes of the exact transcribed artifact
// within the frozen probe domain. boundary selects the separately inspected
// root-terminal/peer-lifecycle extension for concrete runtime samples. It does
// not change the weak/equal-charge contracts or the K1 CheckTrace oracle.
func CheckProbeContext(c Config, profile string, observed ContextObservation, boundary bool) (ProbeResult, error) {
	if profile != "weak-a" && profile != "weak-b" && profile != "p" && profile != "q" {
		return ProbeResult{}, fmt.Errorf("unknown probe %q", profile)
	}
	weak := profile == "weak-a" || profile == "weak-b"
	if c.Request != (Request{1, 0, "answer"}) || c.Depth != 2 || c.I < 1 || c.I > 3 || c.R < 1 || c.R > 3 || c.Local || c.Closure || (c.I != c.R && !(weak && c.I == 3 && c.R == 2)) {
		return ProbeResult{}, fmt.Errorf("outside frozen probe profile: %+v", c)
	}
	shape := "parallel"
	if weak {
		shape = "sequential"
	}
	if c.Shape != shape {
		return ProbeResult{}, fmt.Errorf("probe %s requires %s", profile, shape)
	}
	graph := actualProbeGraph(profile)
	if boundary {
		graph = RuntimeBoundaryGraph(graph)
	}
	result := Probe(c, graph, weak)
	if len(result.Counterexamples) > 0 || result.Stuck != 0 || result.RankFailures != 0 {
		return result, fmt.Errorf("probe graph violation: %+v", result)
	}
	for _, o := range result.Outcomes {
		if o == observed {
			return result, nil
		}
	}
	return result, fmt.Errorf("observation outside actual %s probe product: %+v", profile, observed)
}
