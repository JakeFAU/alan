package openai

import (
	"bytes"
	"encoding/json"
	"strings"
	"unicode/utf8"
)

// evidenceSanitizer applies one conservative policy to every provider-returned
// string, including JSON object keys and JSON encoded inside strings. Empty
// evidence replaces secret-bearing or uninspectable data. No limit/failure path
// returns content it could not inspect. These are resource bounds, not a promise
// to recognize arbitrary encryption or encodings other than JSON escapes.
type evidenceSanitizer struct {
	secret               string
	bytesLeft, nodesLeft int
}

const maxEvidenceStringLayers = 4
const maxEvidenceStructureDepth = 64

func (s *evidenceSanitizer) text(raw string, layers int) (string, bool) {
	if layers > maxEvidenceStringLayers || len(raw) > s.bytesLeft || s.nodesLeft <= 0 || !utf8.ValidString(raw) {
		return "", true
	}
	s.bytesLeft -= len(raw)
	s.nodesLeft--
	if json.Valid([]byte(raw)) {
		if uniqueJSON([]byte(raw)) != nil {
			return "", true
		}
		var value any
		d := json.NewDecoder(bytes.NewBufferString(raw))
		d.UseNumber()
		if d.Decode(&value) != nil {
			return "", true
		}
		value, changed := s.value(value, layers, 0)
		if !changed {
			if strings.Contains(raw, s.secret) {
				return "", true
			}
			return raw, false
		}
		encoded, e := json.Marshal(value)
		if e != nil || strings.Contains(string(encoded), s.secret) {
			return "", true
		}
		return string(encoded), true
	}
	if strings.Contains(raw, `\`) {
		// Headers and metadata can contain an unquoted JSON escape fragment. Decode
		// only valid fragments; unsupported/malformed escaping is omitted wholesale.
		var decoded string
		if json.Unmarshal([]byte(`"`+raw+`"`), &decoded) != nil || decoded == raw {
			return "", true
		}
		if _, changed := s.text(decoded, layers+1); changed {
			return "", true
		}
	}
	if strings.Contains(raw, s.secret) {
		return "", true
	}
	return raw, false
}
func (s *evidenceSanitizer) value(v any, layers, depth int) (any, bool) {
	if depth > maxEvidenceStructureDepth || s.nodesLeft <= 0 {
		return nil, true
	}
	s.nodesLeft--
	switch x := v.(type) {
	case string:
		return s.text(x, layers+1)
	case []any:
		changed := false
		for i, element := range x {
			clean, different := s.value(element, layers, depth+1)
			x[i] = clean
			changed = changed || different
		}
		return x, changed
	case map[string]any:
		clean := make(map[string]any, len(x))
		changed := false
		for key, element := range x {
			safeKey, keyChanged := s.text(key, layers+1)
			safeValue, valueChanged := s.value(element, layers, depth+1)
			changed = changed || keyChanged || valueChanged
			if keyChanged {
				continue
			}
			clean[safeKey] = safeValue
		}
		return clean, changed
	default:
		return v, false
	}
}
