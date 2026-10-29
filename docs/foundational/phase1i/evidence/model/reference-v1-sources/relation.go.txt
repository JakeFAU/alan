package substitution

import "fmt"

// Graph supplies finite component instructions over the frozen protocol state.
// Reference protocol/environment steps are shared model assumptions, never
// production code. Candidate instruction semantics are transcribed separately.
type Graph func(Config, State) []Edge

type RelationResult struct {
	Initials, Pairs, ComponentChecks, EnvironmentChecks, Stutters, Failures int
	Counterexamples                                                         []Counterexample
}

func publicState(s State) State {
	s.PrivatePC = [slots]int{}
	s.PrivateRegisters = [slots][8]int{}
	return s
}
func equalFact(a, b Observation) bool { return a == b }
func LocalConfigurations() []Config {
	var configs []Config
	for id := 1; id <= 2; id++ {
		for value := 0; value <= 1; value++ {
			for _, m := range []string{"answer", "fail", "absent", "pending"} {
				for depth := 1; depth <= 2; depth++ {
					for cap := 1; cap <= 3; cap++ {
						for i := 0; i <= 3; i++ {
							for r := 0; r <= 3; r++ {
								for _, peer := range []bool{false, true} {
									configs = append(configs, Config{Shape: "local", Request: Request{id, value, m}, I: i, R: r, Depth: cap, Local: true, Peer: peer, Closure: true, EntryDepth: depth})
								}
							}
						}
					}
				}
			}
		}
	}
	return configs
}

// CheckLocalRelation checks asymmetric event simulation and bilateral declared
// environment readiness at every reachable candidate instruction boundary. Its
// domain is LocalConfigurations, never the five context executions. A pure
// private_step may stutter only if all protocol/accounting fields are unchanged.
// Progress/cycle checks are reported separately by ExploreGraph.
func CheckLocalRelation(c Config, graph Graph) RelationResult {
	result := RelationResult{Initials: 1}
	initial := Initial(c)
	nodes := []node{{s: initial, parent: -1}}
	seen := map[State]bool{initial: true}
	failed := map[string]bool{}
	report := func(code, detail string, index int, e Observation) {
		result.Failures++
		if !failed[code] {
			failed[code] = true
			result.Counterexamples = append(result.Counterexamples, Counterexample{code, detail, path(nodes, index, &e)})
		}
	}
	for index := 0; index < len(nodes); index++ {
		m := nodes[index].s
		k := publicState(m)
		result.Pairs++
		me := graph(c, m)
		ke := Successors(c, Reference, k)
		for _, edge := range me {
			if edge.Environment {
				result.EnvironmentChecks++
			} else {
				result.ComponentChecks++
			}
			matched := false
			if edge.Event.Kind == "private_step" {
				matched = publicState(edge.Next) == k
				if matched {
					result.Stutters++
				}
			}
			for _, candidate := range ke {
				if equalFact(edge.Event, candidate.Event) && publicState(edge.Next) == candidate.Next && edge.Environment == candidate.Environment {
					matched = true
					break
				}
			}
			if !matched {
				report("local_simulation", fmt.Sprintf("candidate event %s lacks a K1 match", edge.Event.Kind), index, edge.Event)
			}
			if !seen[edge.Next] {
				seen[edge.Next] = true
				nodes = append(nodes, node{edge.Next, index, edge.Event})
			}
		}
		for _, edge := range ke {
			if !edge.Environment {
				continue
			}
			result.EnvironmentChecks++
			found := false
			for _, candidate := range me {
				if candidate.Environment && equalFact(edge.Event, candidate.Event) && publicState(candidate.Next) == edge.Next {
					found = true
					break
				}
			}
			if !found {
				report("environment_readiness", "declared environment event missing at candidate instruction boundary", index, edge.Event)
			}
		}
	}
	return result
}
