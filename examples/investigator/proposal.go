package main

import (
	"context"
	"strings"
	"unicode"

	"github.com/JakeFAU/alan/adapters/openai"
	"github.com/JakeFAU/alan/effects"
	"github.com/JakeFAU/alan/schema"
)

const invalidInterpretation = "proposal_invalid_interpretation"
const invalidQuestion = "proposal_invalid_question"

// proposalAdapter delegates the original request once and checks whether each
// prose field contains a Unicode letter or number. This is a local text-shape
// check. The parent still selects the response and verifies its proposed quote.
type proposalAdapter struct{ *openai.Adapter }

func (a proposalAdapter) Execute(ctx context.Context, prepared effects.Prepared) effects.Candidate {
	result := a.Adapter.Execute(ctx, prepared)
	if result.Disposition == "validated" {
		// Preserve the sanitized response, typed proposal, usage and identity.
		result.Disposition = proposalDisposition(result.Payload)
	}
	return result
}

// The same rule applies to a live response and a previously recorded response
// from an owner that may have used the unwrapped Responses adapter.
func proposalDisposition(payload map[string]schema.Value) string {
	hasText := func(s string) bool {
		return strings.ContainsFunc(s, func(r rune) bool { return unicode.IsLetter(r) || unicode.IsNumber(r) })
	}
	if !hasText(payload["interpretation"].Text) {
		return invalidInterpretation
	}
	if question := payload["unresolved"].Text; question != "" && !hasText(question) {
		return invalidQuestion
	}
	return "validated"
}
