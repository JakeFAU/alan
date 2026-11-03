package executor

import (
	"fmt"
	recordschema "github.com/JakeFAU/alan/schema"
	"strings"
)

// InvocationOutcomeRecord gives names/types; absent responses are validated
// separately, never filled with defaults that could violate an enum schema.
func InvocationOutcomeRecord(output recordschema.Record) recordschema.Record {
	r := recordschema.Record{"status": {Kind: "string",
		MaxBytes: 20,
		Enum: []string{"succeeded",
			"failed",
			"cancelled",
			"abandoned",
			"rejected"}},
		"reason": {Kind: "string",
			MaxBytes: 4096},
		"response_present": {Kind: "bool"}}
	for k, f := range output {
		r["response."+k] = f
	}
	return r
}
func validateInvocationPayload(output recordschema.Record, payload map[string]Value) error {
	envelope := InvocationOutcomeRecord(nil)
	values := map[string]Value{}
	for k := range envelope {
		values[k] = payload[k]
	}
	if err := envelope.Validate(values); err != nil {
		return err
	}
	response := map[string]Value{}
	for k, v := range payload {
		if _, ok := envelope[k]; ok {
			continue
		}
		if !strings.HasPrefix(k, "response.") {
			return fmt.Errorf("unknown invocation outcome field")
		}
		response[strings.TrimPrefix(k, "response.")] = v
	}
	if !payload["response_present"].Flag {
		if len(response) != 0 || payload["status"].Text == "succeeded" {
			return fmt.Errorf("invalid absent response")
		}
		return nil
	}
	return output.Validate(response)
}
