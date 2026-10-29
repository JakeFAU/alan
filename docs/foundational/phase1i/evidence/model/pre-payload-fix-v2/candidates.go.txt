package substitution

// CandidateGraph is independently hand-transcribed from the frozen retained
// instruction arrays. It evaluates branch arms and response operands below;
// expected responses remain solely in the separately frozen oracle. Common
// protocol/environment transitions implement the declared LTS assumptions.
func CandidateGraph(v Variant) Graph {
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
			if e.Event.Kind == "outcome_consume" && i == helper {
				actor = entry
			}
			if e.Environment || (actor != entry && actor != helper) {
				edges = append(edges, e)
				continue
			}
			if actor == helper && v != Helper {
				continue
			}
			pc := s.PrivatePC[actor]
			o := s.Ops[actor]
			allow := false
			switch e.Event.Kind {
			case "invoke_commit":
				if i == entry {
					allow = true
				} else if v == Helper && pc == 1 {
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
				allow = v == Local && actor == entry
			case "choose_helper":
				allow = v == Helper && actor == entry
			case "outcome_consume":
				if i == helper && v == Helper && pc == 2 {
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
				if actor == entry && v == Local && (pc == 3 || pc == 5) {
					allow = true
					id = s.PrivateRegisters[entry][0]
					if pc == 3 {
						value = 1
					}
					e.Next.PrivatePC[entry] = pc + 1
				}
				if actor == entry && v == Helper && pc == 6 {
					allow = true
					id = s.PrivateRegisters[entry][5]
					value = s.PrivateRegisters[entry][6]
					e.Next.PrivatePC[entry] = 7
				}
				if actor == helper && pc == 5 {
					allow = true
					id = s.PrivateRegisters[helper][0]
					value = s.PrivateRegisters[helper][2]
					e.Next.PrivatePC[helper] = 6
				}
				if allow {
					e.Next.Ops[actor].ReplyID = id
					e.Next.Ops[actor].ReplyValue = value
					e.Event.ID = id
					e.Event.Value = value
				}
			case "terminal_freeze":
				desired := none
				if actor == entry && v == Local {
					switch pc {
					case 6, 10:
						desired = success
					case 8:
						desired = failure
					}
				}
				if actor == entry && v == Helper {
					switch pc {
					case 7, 11:
						desired = success
					case 9, 14, 15:
						desired = failure
					}
				}
				if actor == helper && pc == 6 {
					desired = success
				}
				if desired != none {
					allow = true
					e.Next.Ops[actor].Freeze = desired
				}
			case "pending_receive":
				allow = (v == Local && pc == 11) || (v == Helper && pc == 12)
			}
			if allow {
				edges = append(edges, e)
			}
		}
		for _, i := range []int{entry, helper} {
			o := s.Ops[i]
			if o.Admission != 1 || o.Freeze != none || o.PC == receive || o.PC == pending || (i == entry && o.PC == choose) || (i == helper && v != Helper) {
				continue
			}
			pc := s.PrivatePC[i]
			next := -1
			n := s
			if i == entry && v == Local {
				switch pc {
				case 1:
					next = 7
					if o.Input.Mode == "answer" {
						next = 2
					}
				case 2:
					next = 5
					if s.PrivateRegisters[i][1] == 0 {
						next = 3
					}
				case 4:
					next = 6
				case 7:
					next = 9
					if o.Input.Mode == "fail" {
						next = 8
					}
				case 9:
					next = 11
					if o.Input.Mode == "absent" {
						next = 10
					}
				}
			}
			if i == entry && v == Helper {
				switch pc {
				case 3:
					next = 15
					if s.PrivateRegisters[i][3] == 1 {
						next = 4
					}
				case 4:
					next = 14
					if s.PrivateRegisters[i][4] == 1 {
						next = 5
					}
				case 5:
					next = 8
					if o.Input.Mode == "answer" {
						next = 6
					}
				case 8:
					next = 10
					if o.Input.Mode == "fail" {
						next = 9
					}
				case 10:
					next = 12
					if o.Input.Mode == "absent" {
						next = 11
					}
				}
			}
			if i == helper {
				switch pc {
				case 1:
					next = 4
					if s.PrivateRegisters[i][1] == 1 {
						next = 2
					}
				case 2:
					next = 3
					n.PrivateRegisters[i][2] = 0
				case 3:
					next = 5
				case 4:
					next = 5
					n.PrivateRegisters[i][2] = 1
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
