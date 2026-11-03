package main

import (
	"context"
	"strings"
	"unicode"

	"github.com/JakeFAU/alan/adapters/openai"
	"github.com/JakeFAU/alan/effects"
)

const invalidInterpretation = "proposal_invalid_interpretation"
const invalidQuestion = "proposal_invalid_question"

// proposalAdapter delegates the original request once and checks whether each
// prose field contains a Unicode letter or number. This is a local text-shape
// check. The parent still selects the response and verifies its proposed quote.
type proposalAdapter struct{ *openai.Adapter }

func (a proposalAdapter) Execute(ctx context.Context, prepared effects.Prepared) effects.Candidate {
	result := a.Adapter.Execute(ctx, prepared)
	if result.Disposition != "validated" {
		return result
	}
	hasText := func(s string) bool {
		return strings.ContainsFunc(s, func(r rune) bool { return unicode.IsLetter(r) || unicode.IsNumber(r) })
	}
	if !hasText(result.Payload["interpretation"].Text) {
		result.Disposition = invalidInterpretation
	} else if question := result.Payload["unresolved"].Text; question != "" && !hasText(question) {
		result.Disposition = invalidQuestion
	}
	// Keep the exact sanitized response, parsed proposal, usage and original
	// identity on failure. ALAN supplies an unsuccessful typed envelope.
	return result
}
