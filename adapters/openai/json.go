package openai

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
)

// uniqueJSON rejects duplicate fields at every object depth before typed decoding.
func uniqueJSON(raw []byte) error {
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	if e := uniqueValue(d, 0); e != nil {
		return e
	}
	if _, e := d.Token(); e != io.EOF {
		return errors.New("trailing JSON")
	}
	return nil
}
func uniqueValue(d *json.Decoder, depth int) error {
	if depth > 64 {
		return errors.New("JSON nesting limit")
	}
	t, e := d.Token()
	if e != nil {
		return e
	}
	delim, ok := t.(json.Delim)
	if !ok {
		return nil
	}
	switch delim {
	case '{':
		seen := map[string]bool{}
		for d.More() {
			key, e := d.Token()
			if e != nil {
				return e
			}
			k, ok := key.(string)
			if !ok || seen[k] {
				return errors.New("duplicate JSON field")
			}
			seen[k] = true
			if e = uniqueValue(d, depth+1); e != nil {
				return e
			}
		}
	case '[':
		for d.More() {
			if e = uniqueValue(d, depth+1); e != nil {
				return e
			}
		}
	default:
		return errors.New("invalid JSON delimiter")
	}
	_, e = d.Token()
	return e
}
