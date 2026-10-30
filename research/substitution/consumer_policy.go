package substitution

// joinConsumption is a closed internal choice between the two stated parents,
// not an API for arbitrary context generation. Commit order is always e then s.
type joinConsumption uint8

const (
	consumeCandidateFirst joinConsumption = iota // Historical Phase1L C_es.
	consumeSFirst                                // Phase1M C_se.
)

// ConsumerPolicyConfigurations is the four-input C_se positive domain. It has
// the same entitlement as C_es; no old product result is inherited by this API.
func ConsumerPolicyConfigurations() []JoinConfig {
	return JoinConfigurations()
}

// ConsumerPolicyInitial retains e/h/s demand and one open parent. Both contexts
// have identical initial representations; their first receive gates differ.
func ConsumerPolicyInitial(c JoinConfig) JoinState {
	return JoinInitial(c)
}

// ConsumerPolicySuccessors implements fixed C_se: commit e then s, consume s
// then e, then make the guarded strict decision, End and root publication.
// JoinState is reused as finite storage, with stage2 awaiting s and stage3 e.
// K1 choice, protocol events, exact envelopes and lifetime guards are unchanged.
func ConsumerPolicySuccessors(c JoinConfig, v Variant, s JoinState) []JoinEdge {
	return joinSuccessors(c, v, s, consumeSFirst)
}

// CheckConsumerPolicyState checks C_se representation, original required joins
// and resource invariants. K1 payload conformance and G_answer remain separate.
func CheckConsumerPolicyState(c JoinConfig, s JoinState) error {
	return checkJoinState(c, s, consumeSFirst)
}

// CheckConsumerPolicyPrefix accepts truthful C_se prefixes, including failed
// outcomes and S consumed while e is still required. It does not certify raw
// authentication, component conformance, progress, or G_answer.
func CheckConsumerPolicyPrefix(c JoinConfig, events []JoinEvent) []Violation {
	_, violations := checkJoinPolicyPrefix(c, events, consumeSFirst)
	return violations
}

// CheckConsumerPolicyGuarantee requires the original G_answer after s-then-e
// consumption. CheckJoinComponent remains the separate unchanged K1 checker.
func CheckConsumerPolicyGuarantee(c JoinConfig, events []JoinEvent) []Violation {
	return checkJoinPolicyGuarantee(c, events, consumeSFirst)
}

// ConsumerPolicyCensus labels the new product without altering the historical
// JoinCensus fields or their JSON. Counts and maxima are obtained only by runs.
type ConsumerPolicyCensus struct {
	Context string
	JoinCensus
}

// ExploreConsumerPolicy explores all C_se model x prefix-monitor states without
// a cutoff or failure pruning. Run only after independent reference review.
// It retains every maximum and shortest counterexamples, checking every edge
// before deduplication. JoinRank/JoinRankCap apply to the unchanged finite fields.
func ExploreConsumerPolicy(c JoinConfig, variant Variant) (ConsumerPolicyCensus, error) {
	census, err := exploreJoinPolicy(c, variant, consumeSFirst)
	return ConsumerPolicyCensus{Context: "C_se", JoinCensus: census}, err
}
