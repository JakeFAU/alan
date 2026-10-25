package openai

import (
	"encoding/json"
	"errors"
	"strings"
)

// exactProtocolKeys guards only fields decoded into the provider structs. Go's
// struct decoder folds case even with JSON tags; noncanonical aliases must never
// override the exact protocol spelling. Application output JSON is map-backed
// and intentionally does not pass through this guard.
func exactProtocolKeys(raw []byte) error {
	root, e := protocolObject(raw, "id", "model", "status", "error", "output", "usage")
	if e != nil {
		return e
	}
	if output, ok := root["output"]; ok {
		var items []json.RawMessage
		if e = json.Unmarshal(output, &items); e != nil {
			return e
		}
		for _, item := range items {
			object, e := protocolObject(item, "type", "role", "content")
			if e != nil {
				return e
			}
			if content, ok := object["content"]; ok {
				var parts []json.RawMessage
				if e = json.Unmarshal(content, &parts); e != nil {
					return e
				}
				for _, part := range parts {
					if _, e = protocolObject(part, "type", "text", "refusal"); e != nil {
						return e
					}
				}
			}
		}
	}
	if usage, ok := root["usage"]; ok && string(usage) != "null" {
		if _, e = protocolObject(usage, "input_tokens", "output_tokens", "total_tokens"); e != nil {
			return e
		}
	}
	return nil
}
func protocolObject(raw []byte, known ...string) (map[string]json.RawMessage, error) {
	var object map[string]json.RawMessage
	if e := json.Unmarshal(raw, &object); e != nil {
		return nil, e
	}
	if object == nil {
		return nil, errors.New("provider object required")
	}
	for key := range object {
		for _, canonical := range known {
			if key != canonical && strings.EqualFold(key, canonical) {
				return nil, errors.New("noncanonical provider key")
			}
		}
	}
	return object, nil
}
