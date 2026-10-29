// Package protocol declares the experimental Phase 1I public record shapes.
// Behavioral requirements and finite input domains live in the frozen contract;
// the scalar schema validates kinds, not integer ranges.
package protocol

import "github.com/JakeFAU/alan/schema"

const (
	Port       = "public"
	Peer       = "caller"
	Obligation = "substitution"
)

// Request returns a fresh request schema without application policy.
func Request() schema.Record {
	return schema.Record{
		"id":    {Kind: "int"},
		"value": {Kind: "int"},
		"mode": {
			Kind: "string", MaxBytes: 8,
			Enum: []string{"answer", "fail", "absent", "pending"},
		},
	}
}

// Response returns a fresh response schema without a response oracle.
func Response() schema.Record {
	return schema.Record{
		"id":    {Kind: "int"},
		"value": {Kind: "int"},
	}
}
