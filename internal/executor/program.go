package executor

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	recordschema "github.com/JakeFAU/alan/schema"
	"io"
	"math"
	"strings"
)

const SemanticVersion = "alan/1d/v1"

var ErrLimit = errors.New("resource limit")
var ErrIneligible = errors.New("ineligible observation")
var ErrClosed = errors.New("closed run")

type Value = recordschema.Value

func Int(n int64) Value   { return Value{Kind: "int", Number: n} }
func Text(s string) Value { return Value{Kind: "string", Text: s} }

type Operand struct {
	Ref     string
	Literal *Value
	Offset  *int64
}

func Lit(v Value) Operand  { return Operand{Literal: &v} }
func Ref(s string) Operand { return Operand{Ref: s} }

type RunID struct {
	Subject    string
	Generation int64
}
type Program struct {
	Semantics string
	Code      []Instruction
	Author    *ProgramMetadata `json:",omitempty"`
}
type Instruction struct {
	Op                string
	Dst               string
	A, B              Operand
	Target, Otherwise int
	Match             *Match
	Bind              map[string]string
	Effect            *Effect
	Updates           map[string]Operand
	Collection        *recordschema.Field `json:",omitempty"`
	Outcome           string
}
type Match struct {
	Kind, Provenance                          string
	Source, Generation, Obligation, Operation Operand
}
type Effect struct {
	Kind                           string
	Target, Generation, Obligation Operand
	Payload                        map[string]Operand
}
type Projection struct {
	PC      int
	Data    map[string]Value
	Status  string
	Cursors map[int]Cursor `json:",omitempty"`
}
type Observation struct {
	ID                       string
	Run                      RunID
	Kind, Provenance, Source string
	SourceGeneration         int64
	Obligation, Operation    string
	Payload                  map[string]Value
}
type Intent struct {
	ID           string
	Run          RunID
	Sequence     int64
	Ordinal      int
	Kind, Target string
	Generation   int64
	Obligation   string
	Payload      map[string]Value
}
type Reduction struct {
	Projection Projection
	Intents    []Intent
}

// Limits are operational byte bounds. MaxOutputBytes covers the complete JSON
// intent array for one reduction, including descriptor metadata and delimiters.
type Limits struct{ MaxProgramBytes, MaxDataBytes, MaxObservationBytes, MaxOutputBytes int }

func DefaultLimits() Limits {
	return Limits{MaxProgramBytes: 1 << 20, MaxDataBytes: 1 << 20, MaxObservationBytes: 1 << 20, MaxOutputBytes: 1 << 20}
}

var ErrVersion = errors.New("unsupported version")
var ErrCorrupt = errors.New("corrupt durable state")

// EncodeProgram retains exact canonical bytes separately from the semantic version.
func EncodeProgram(p Program) ([]byte, string, error) {
	if err := validateProgram(p); err != nil {
		return nil, "", err
	}
	b, err := json.Marshal(p)
	if err != nil {
		return nil, "", err
	}
	return b, programHash(b), nil
}
func programHash(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
func decode(b []byte, v any) error {
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if err := d.Decode(v); err != nil {
		return err
	}
	var extra any
	if err := d.Decode(&extra); err != io.EOF {
		return fmt.Errorf("trailing JSON: %w", ErrCorrupt)
	}
	return nil
}
func validateProgram(p Program) error {
	if !supportedSemantics(p.Semantics) {
		return ErrVersion
	}
	if err := validateMetadata(p); err != nil {
		return err
	}
	if len(p.Code) == 0 {
		return fmt.Errorf("empty program")
	}
	for i, in := range p.Code {
		invocationEffect := in.Effect != nil && in.Effect.Kind == "invocation"
		invocationReceive := in.Match != nil && in.Match.Kind == "invocation"
		if invocationEffect || invocationReceive {
			if p.Semantics != CollectionSemanticVersion || p.Author == nil || len(p.Author.Invocations) == 0 {
				return ErrVersion
			}
		}
		switch in.Op {
		case "foreach", "next", "loop", "append":
			if p.Semantics != CollectionSemanticVersion {
				return ErrVersion
			}
		case "set", "add":
			if in.Dst == "" {
				return fmt.Errorf("instruction %d: empty destination", i)
			}
		case "branch", "jump":
			if in.Target < 0 || in.Target >= len(p.Code) || in.Op == "branch" && (in.Otherwise < 0 || in.Otherwise >= len(p.Code)) {
				return fmt.Errorf("instruction %d: invalid target", i)
			}
		case "receive":
			if in.Match == nil || in.Match.Kind == "" || in.Match.Provenance == "" {
				return fmt.Errorf("instruction %d: missing receive type/provenance", i)
			}

			seen := map[string]bool{}
			for field, dst := range in.Bind {
				if field == "" || dst == "" || seen[dst] {
					return fmt.Errorf("instruction %d: ambiguous or empty receive binding", i)
				}
				seen[dst] = true
			}
		case "emit":
			if in.Effect == nil || in.Effect.Kind == "" {
				return fmt.Errorf("instruction %d: missing effect", i)
			}
		case "end":
			if in.Outcome != "succeeded" && in.Outcome != "failed" {
				return fmt.Errorf("instruction %d: invalid outcome", i)
			}
		default:
			return fmt.Errorf("instruction %d: unknown opcode %q", i, in.Op)
		}
	}
	return validateCollections(p)
}

// Keep the original untyped scalar reducer's acceptance and comparison rules.
// Native list values and author schemas apply their stricter UTF-8 validation.
func validValue(v Value) bool {
	if v.List != nil || v.Kind == "list" {
		return v.Valid()
	}
	switch v.Kind {
	case "int":
		return v.Text == "" && !v.Flag
	case "string":
		return v.Number == 0 && !v.Flag
	case "bool":
		return v.Number == 0 && v.Text == ""
	}
	return false
}
func eval(o Operand, data map[string]Value) (Value, error) {
	if (o.Ref == "") == (o.Literal == nil) {
		return Value{}, fmt.Errorf("operand must have exactly one reference or literal")
	}
	var v Value
	if o.Ref != "" {
		var ok bool
		v, ok = data[o.Ref]
		if !ok || !validValue(v) {
			return Value{}, fmt.Errorf("missing or invalid data %q", o.Ref)
		}
	} else {
		v = *o.Literal
		if !validValue(v) {
			return Value{}, fmt.Errorf("invalid literal")
		}
	}
	if o.Offset != nil {
		if v.Kind != "int" {
			return Value{}, fmt.Errorf("offset requires integer")
		}
		b := *o.Offset
		a := v.Number
		if b > 0 && a > math.MaxInt64-b || b < 0 && a < math.MinInt64-b {
			return Value{}, fmt.Errorf("integer overflow: %w", ErrLimit)
		}
		v.Number += b
	}
	return v.Clone(), nil
}
func stringOperand(o Operand, d map[string]Value) (string, error) {
	v, e := eval(o, d)
	if e != nil {
		return "", e
	}
	if v.Kind != "string" {
		return "", fmt.Errorf("expected string")
	}
	return v.Text, nil
}
func intOperand(o Operand, d map[string]Value) (int64, error) {
	v, e := eval(o, d)
	if e != nil {
		return 0, e
	}
	if v.Kind != "int" {
		return 0, fmt.Errorf("expected int")
	}
	return v.Number, nil
}
func checkSize(v any, max int) error {
	if max <= 0 {
		return ErrLimit
	}
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	if len(b) > max {
		return ErrLimit
	}
	return nil
}
func validateProjection(s Projection, p Program, l Limits) error {
	if s.PC < 0 || s.PC >= len(p.Code) {
		return fmt.Errorf("invalid program counter")
	}
	switch s.Status {
	case "open", "succeeded", "failed", "abandoned", "cancelled":
	default:
		return fmt.Errorf("invalid status")
	}
	if s.Data == nil {
		return fmt.Errorf("missing data")
	}
	for k, v := range s.Data {
		if k == "" || !validValue(v) || p.Semantics != CollectionSemanticVersion && v.Kind == "list" || p.Semantics == CollectionSemanticVersion && !v.Valid() {
			return fmt.Errorf("invalid data")
		}
	}
	if err := validateCursors(p, s); err != nil {
		return err
	}
	return checkSize(s, l.MaxDataBytes)
}

// Reduce is pure and performs exactly one finite instruction. Effects are descriptors,
// never callbacks; all loops yield to a subsequent durable execution commit.
func Reduce(p Program, s Projection, o *Observation, id RunID, seq int64, l Limits) (Reduction, error) {
	if err := validateProgram(p); err != nil {
		return Reduction{}, err
	}
	if err := checkSize(p, l.MaxProgramBytes); err != nil {
		return Reduction{}, err
	}
	if err := validateProjection(s, p, l); err != nil {
		return Reduction{}, err
	}
	if s.Status != "open" {
		return Reduction{}, ErrClosed
	}
	if id.Subject == "" || id.Generation <= 0 || seq <= 0 {
		return Reduction{}, fmt.Errorf("invalid execution identity")
	}
	in := p.Code[s.PC]
	if in.Op != "receive" && o != nil {
		return Reduction{}, ErrIneligible
	}
	next := Projection{PC: s.PC + 1, Status: s.Status, Data: make(map[string]Value, len(s.Data))}
	for k, v := range s.Data {
		next.Data[k] = v.Clone()
	}
	next.Cursors = cloneCursors(s.Cursors)
	result := Reduction{Projection: next, Intents: []Intent{}}
	switch in.Op {
	case "foreach", "next", "loop", "append":
		if err := reduceCollection(p, in, s, &next); err != nil {
			return Reduction{}, err
		}
	case "set":
		v, err := eval(in.A, s.Data)
		if err != nil {
			return Reduction{}, err
		}
		if in.Collection != nil {
			if err := in.Collection.Validate(v); err != nil {
				return Reduction{}, err
			}
		}
		next.Data[in.Dst] = v
	case "add":
		a, err := intOperand(in.A, s.Data)
		if err != nil {
			return Reduction{}, err
		}
		b, err := intOperand(in.B, s.Data)
		if err != nil {
			return Reduction{}, err
		}
		if b > 0 && a > math.MaxInt64-b || b < 0 && a < math.MinInt64-b {
			return Reduction{}, fmt.Errorf("integer overflow: %w", ErrLimit)
		}
		next.Data[in.Dst] = Int(a + b)
	case "branch":
		a, err := eval(in.A, s.Data)
		if err != nil {
			return Reduction{}, err
		}
		b, err := eval(in.B, s.Data)
		if err != nil {
			return Reduction{}, err
		}
		next.PC = in.Otherwise
		if a.Kind != "list" && b.Kind != "list" && a == b || a.Kind == "list" && a.Equal(b) {
			next.PC = in.Target
		}
	case "jump":
		next.PC = in.Target
	case "receive":
		if o == nil || o.Run != id {
			return Reduction{}, ErrIneligible
		}
		if err := checkSize(o, l.MaxObservationBytes); err != nil {
			return Reduction{}, err
		}
		m := in.Match
		source, err := stringOperand(m.Source, s.Data)
		if err != nil {
			return Reduction{}, err
		}
		gen, err := intOperand(m.Generation, s.Data)
		if err != nil {
			return Reduction{}, err
		}
		ob, err := stringOperand(m.Obligation, s.Data)
		if err != nil {
			return Reduction{}, err
		}
		op, err := stringOperand(m.Operation, s.Data)
		if err != nil {
			return Reduction{}, err
		}
		if o.Kind != m.Kind || o.Provenance != m.Provenance || o.Source != source || o.SourceGeneration != gen || o.Obligation != ob || o.Operation != op {
			return Reduction{}, ErrIneligible
		}
		if p.Author != nil {
			if record, ok := p.Author.Receives[s.PC]; ok {
				var err error
				if in.Match.Kind == "invocation" && in.Match.Obligation.Literal != nil {
					declaration, ok := p.Author.Invocations[in.Match.Obligation.Literal.Text]
					if !ok {
						return Reduction{}, ErrCorrupt
					}
					err = validateInvocationPayload(declaration.Output.Record, o.Payload)
				} else if in.Match.Kind == "adapter" && in.Match.Obligation.Literal != nil {
					effect, ok := p.Author.Effects[in.Match.Obligation.Literal.Text]
					if !ok {
						return Reduction{}, fmt.Errorf("unknown adapter declaration")
					}
					err = recordschema.ValidateResult(effect.Output.Record, o.Payload)
				} else {
					err = record.Record.Validate(o.Payload)
				}
				if err != nil {
					return Reduction{}, fmt.Errorf("receive schema: %w", err)
				}
			}
		}
		for field, dst := range in.Bind {
			if in.Match.Kind == "invocation" && strings.HasPrefix(field, "response.") && !o.Payload["response_present"].Flag {
				delete(next.Data, dst)
				continue
			}
			v, ok := o.Payload[field]
			if !ok || dst == "" || !validValue(v) {
				return Reduction{}, fmt.Errorf("invalid receive binding %q", field)
			}
			next.Data[dst] = v.Clone()
		}
	case "emit":
		e := in.Effect
		target, err := stringOperand(e.Target, s.Data)
		if err != nil {
			return Reduction{}, err
		}
		gen, err := intOperand(e.Generation, s.Data)
		if err != nil {
			return Reduction{}, err
		}
		ob, err := stringOperand(e.Obligation, s.Data)
		if err != nil {
			return Reduction{}, err
		}
		key, _ := json.Marshal([]any{id.Subject, id.Generation, seq, 0})
		intent := Intent{ID: string(key), Run: id, Sequence: seq, Ordinal: 0, Kind: e.Kind, Target: target, Generation: gen, Obligation: ob, Payload: map[string]Value{}}
		for k, v := range e.Payload {
			x, err := eval(v, s.Data)
			if err != nil {
				return Reduction{}, err
			}
			intent.Payload[k] = x
		}
		if err := validateAuthorIntent(p, intent); err != nil {
			return Reduction{}, err
		}
		for k, v := range in.Updates {
			if k == "" || k == in.Dst {
				return Reduction{}, fmt.Errorf("invalid update destination")
			}
			x, err := eval(v, s.Data)
			if err != nil {
				return Reduction{}, err
			}
			next.Data[k] = x
		}
		if in.Dst != "" {
			next.Data[in.Dst] = Text(intent.ID)
		}
		result.Intents = append(result.Intents, intent)
	case "end":
		if in.Outcome == "succeeded" {
			if err := validateAuthorOutput(p, s.Data); err != nil {
				return Reduction{}, fmt.Errorf("output schema: %w", err)
			}
		}
		next.PC = s.PC
		next.Status = in.Outcome
	}
	cleanCursors(p, &next)
	if err := checkOutputSize(result.Intents, l.MaxOutputBytes); err != nil {
		return Reduction{}, fmt.Errorf("generated intents: %w", err)
	}
	if err := validateProjection(next, p, l); err != nil {
		return Reduction{}, err
	}
	result.Projection = next
	return result, nil
}

// checkOutputSize accounts for exact encoding/json bytes without serializing the
// amplified intent array or payload map. Each temporary encoding is bounded by
// the remaining budget's raw string bytes (at most sixfold JSON escaping plus
// fixed scalar/struct overhead). Repeated references never form one large buffer.
func checkOutputSize(intents []Intent, limit int) error {
	remaining := limit
	take := func(n int) error {
		if n < 0 || n > remaining {
			return ErrLimit
		}
		remaining -= n
		return nil
	}
	if err := take(2); err != nil {
		return err
	} // array brackets
	for i, intent := range intents {
		if i > 0 {
			if err := take(1); err != nil {
				return err
			}
		}
		if !rawStringsFit(remaining, intent.ID, intent.Run.Subject, intent.Kind, intent.Target, intent.Obligation) {
			return ErrLimit
		}
		header := intent
		header.Payload = map[string]Value{}
		encoded, err := json.Marshal(header)
		if err != nil {
			return err
		}
		if err = take(len(encoded)); err != nil {
			return err
		}
		first := true
		for key, value := range intent.Payload {
			if !first {
				if err = take(1); err != nil {
					return err
				}
			}
			first = false
			if !rawStringsFit(remaining, key) {
				return ErrLimit
			}
			encoded, err = json.Marshal(key)
			if err != nil {
				return err
			}
			if err = take(len(encoded)); err != nil {
				return err
			}
			if err = take(1); err != nil {
				return err
			} // key/value colon
			if !rawStringsFit(remaining, value.Kind, value.Text) {
				return ErrLimit
			}
			encoded, err = json.Marshal(value)
			if err != nil {
				return err
			}
			if err = take(len(encoded)); err != nil {
				return err
			}
		}
	}
	return nil
}

func rawStringsFit(remaining int, values ...string) bool {
	for _, value := range values {
		if len(value) > remaining {
			return false
		}
		remaining -= len(value)
	}
	return true
}
