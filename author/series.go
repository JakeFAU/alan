package author

import (
	"encoding/json"
	"fmt"
	"maps"
	"math"
	"strconv"

	"github.com/JakeFAU/alan/internal/executor"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

// SeriesStart makes the authority for the first observation explicit.
type SeriesStart uint8

// ApproveAtProvision authorizes only the first observation. Later positions
// require Store.ApproveContinuation with their original paid predecessor.
const ApproveAtProvision SeriesStart = 1

// SeriesOptions declares finite capacity, not approval of all positions.
// Empty ApprovalReasons permits any reason accepted by ContinuationPort's schema.
type SeriesOptions struct {
	ID              string
	Count           int
	Inputs          map[string]schema.Value
	First           SeriesStart
	ApprovalReasons []string
}

// Series owns an immutable finite deployment. It is not an invocation template
// or a single root outcome. Its zero value cannot provision or inspect work.
type Series struct{ declaration *seriesDeclaration }
type seriesDeclaration struct {
	id                 string
	count              int
	effect, capability string
	provision          runtime.Provision
}

const seriesApproval = "series-approval"
const seriesProtocol = "alan/composition/v1"
const seriesResourceVersion = "alan/resource-profile/v1"

// BoundedSeries compiles one self-contained adapter observation into count
// original generation-one runs. Subjects are ID + "-" + the canonical decimal
// one-based index. Later runs use port and obligation "series-approval" and
// approval identity subject + "-approval". Each run has an independent one-effect,
// one-in-flight root. No extra inputs or observation instructions are added to
// the first program. Definition ports and invocation templates are unsupported;
// requests and awaits inside Repeat or ForEach are rejected, including Repeat(1).
// Every successful control-flow path must consume the sole adapter result.
// Default runtime encoded-program/profile byte limits apply to construction.
// For multiple positions, predecessor operation identities must fit the existing
// continuation input field after JSON escaping. A collection loop before the
// request conservatively reserves the full int64 sequence width; other prefixes
// use their instruction-count bound. Single-position series have no such limit.
func BoundedSeries(d Definition, options SeriesOptions) (Series, error) {
	fail := func(err error) (Series, error) { return Series{}, fmt.Errorf("bounded series: %w", err) }
	if options.Count < 1 || options.First != ApproveAtProvision {
		return fail(fmt.Errorf("positive count and explicit first approval required"))
	}
	if err := schema.CheckIdentifier(options.ID); err != nil {
		return fail(err)
	}
	// Compile the unmodified observation first, retaining its original identity.
	original, err := Compile(d)
	if err != nil {
		return fail(err)
	}
	if len(d.Ports) != 0 || len(d.Invocations) != 0 || len(d.Effects) != 1 {
		return fail(fmt.Errorf("one self-contained adapter effect required"))
	}
	var program executor.Program
	if err = json.Unmarshal(original.Bytes(), &program); err != nil {
		return fail(err)
	}
	if err = seriesObservation(d.Steps, program); err != nil {
		return fail(err)
	}
	effect := ""
	capability := ""
	for name, e := range d.Effects {
		effect, capability = name, e.Capability
	}
	// Each encoded normalized root alone consumes at least this many bytes.
	// Division bounds the count before allocation and cannot overflow at MaxInt.
	sample := seriesRoot(runtime.RunID{Subject: options.ID + "-1", Generation: 1}, capability)
	sample.Version = seriesResourceVersion
	// Hex identities contain no JSON escapes; use 64 printable bytes for sizing.
	sample.ID = "0000000000000000000000000000000000000000000000000000000000000000"
	rootBytes, _ := json.Marshal(sample)
	limit := runtime.DefaultLimits().MaxProgramBytes
	if options.Count > limit/len(rootBytes) {
		return fail(fmt.Errorf("resource profile: %w", runtime.ErrLimit))
	}
	if err := seriesContinuationIdentities(program, options.ID, options.Count); err != nil {
		return fail(err)
	}
	if err := original.Metadata().Inputs.Record.Validate(options.Inputs); err != nil {
		return fail(fmt.Errorf("inputs: %w", err))
	}
	inputBytes, err := json.Marshal(runtime.Projection{Status: "open", Data: options.Inputs})
	if err != nil {
		return fail(err)
	}
	if len(inputBytes) > runtime.DefaultLimits().MaxDataBytes {
		return fail(fmt.Errorf("inputs: %w", runtime.ErrLimit))
	}
	// Clone before adding the private receive. Only immutable artifact bytes
	// and the owned provision survive construction.
	next := original
	if options.Count > 1 {
		copyDef, err := cloneSeriesDefinition(d)
		if err != nil {
			return fail(err)
		}
		copyDef.Ports = map[string]Port{seriesApproval: ContinuationPort(seriesApproval, append([]string(nil), options.ApprovalReasons...)...)}
		copyDef.Steps = append([]Step{Receive(seriesApproval, nil)}, copyDef.Steps...)
		next, err = Compile(copyDef)
		if err != nil {
			return fail(err)
		}
	}
	app := make(Application, options.Count)
	profile := &runtime.CompositionProvision{Version: seriesProtocol, RecordDispatchAdmissions: true}
	for index := 1; index <= options.Count; index++ {
		run := runtime.RunID{Subject: options.ID + "-" + strconv.Itoa(index), Generation: 1}
		artifact := next
		if index == 1 {
			artifact = original
		}
		app[run.Subject] = Instance{Artifact: artifact, Inputs: options.Inputs}
		profile.Roots = append(profile.Roots, seriesRoot(run, capability))
		if index > 1 {
			profile.Continuations = append(profile.Continuations, runtime.ContinuationSlot{Target: run, Predecessor: runtime.RunID{Subject: options.ID + "-" + strconv.Itoa(index-1), Generation: 1}, Effect: effect, Obligation: seriesApproval, ApprovalID: run.Subject + "-approval"})
		}
	}
	// Provisioning adds namespace-derived root IDs and versions. Measure that
	// retained form, not the smaller pre-normalized declaration.
	normalized := *profile
	normalized.Roots = append([]runtime.ResourceRoot(nil), profile.Roots...)
	for i := range normalized.Roots {
		normalized.Roots[i].ID = sample.ID
		normalized.Roots[i].Version = seriesResourceVersion
	}
	raw, err := json.Marshal(normalized)
	if err != nil {
		return fail(err)
	}
	if len(raw) > limit {
		return fail(fmt.Errorf("resource profile: %w", runtime.ErrLimit))
	}
	provision, err := app.Provision()
	if err != nil {
		return fail(err)
	}
	provision.Composition = profile
	return Series{&seriesDeclaration{id: options.ID, count: options.Count, effect: effect, capability: capability, provision: provision}}, nil
}

// The ordinary reducer encodes an operation as JSON [subject, generation,
// sequence, ordinal] (executor.Reduce's emit case). Original generation is 1
// and the single emitted intent's ordinal is 0. Before the sole request, each
// instruction executes at most once because Compile permits only forward jumps,
// except for explicit collection loops. Thus request PC+1 bounds its sequence;
// the protected successor adds exactly one approval receive. A preceding loop
// reserves the full positive int64 width without simulating potentially huge
// nested collections at authoring time.
func seriesContinuationIdentities(program executor.Program, id string, count int) error {
	if count == 1 {
		return nil
	}
	sequence := int64(0)
	loop := false
	for pc, instruction := range program.Code {
		if instruction.Op == "foreach" {
			loop = true
		}
		if instruction.Op == "emit" {
			sequence = int64(pc) + 1
			break
		}
	}
	if loop {
		sequence = math.MaxInt64
	}
	field := ContinuationPort(seriesApproval).Receive["previous_operation_id"]
	check := func(index int, sequence int64) error {
		operation, err := json.Marshal([]any{id + "-" + strconv.Itoa(index), int64(1), sequence, 0})
		if err != nil {
			return err
		}
		if len(operation) > field.MaxBytes {
			return fmt.Errorf("continuation predecessor identity requires %d bytes (limit %d): %w", len(operation), field.MaxBytes, runtime.ErrLimit)
		}
		return nil
	}
	if err := check(1, sequence); err != nil {
		return err
	}
	if count > 2 {
		if sequence < math.MaxInt64 {
			sequence++
		}
		// All successor predecessors share one program and the same name prefix.
		// Canonical decimal width is monotone, so the last predecessor covers all
		// earlier positions, including index digit transitions.
		return check(count-1, sequence)
	}
	return nil
}

func seriesRoot(run runtime.RunID, capability string) runtime.ResourceRoot {
	return runtime.ResourceRoot{Run: run, Members: []runtime.RunID{run}, Capabilities: []string{capability}, Limits: runtime.ResourceLimits{MaxEffects: 1, MaxInFlight: 1}}
}

// ID returns the declaration name, or the empty string for a zero Series.
func (s Series) ID() string {
	if s.declaration == nil {
		return ""
	}
	return s.declaration.id
}

// Count returns finite capacity, or zero for a zero Series.
func (s Series) Count() int {
	if s.declaration == nil {
		return 0
	}
	return s.declaration.count
}

// Run returns the original generation-one identity at a one-based position.
func (s Series) Run(index int) (runtime.RunID, error) {
	if s.declaration == nil || index < 1 || index > s.Count() {
		return runtime.RunID{}, fmt.Errorf("bounded series: invalid position %d", index)
	}
	return runtime.RunID{Subject: s.ID() + "-" + strconv.Itoa(index), Generation: 1}, nil
}

// Provision returns an independent deployment copy, without creating a store or
// contacting an adapter. Mutating it never changes this declaration.
func (s Series) Provision() (runtime.Provision, error) {
	if s.declaration == nil {
		return runtime.Provision{}, fmt.Errorf("bounded series: zero declaration")
	}
	return seriesClone(s.declaration.provision)
}

func seriesClone[T any](value T) (T, error) {
	var out T
	raw, err := json.Marshal(value)
	if err == nil {
		err = json.Unmarshal(raw, &out)
	}
	return out, err
}
func cloneSeriesDefinition(d Definition) (Definition, error) {
	out, err := seriesClone(d)
	if err != nil {
		return Definition{}, err
	}
	var cloneSteps func([]Step) []Step
	cloneExpr := func(e Expr) Expr { v, _ := seriesClone(e.operand); return Expr{v} }
	cloneSteps = func(steps []Step) []Step {
		if steps == nil {
			return nil
		}
		out := make([]Step, len(steps))
		for i, s := range steps {
			s.a = cloneExpr(s.a)
			s.b = cloneExpr(s.b)
			s.bindings = maps.Clone(s.bindings)
			s.values = maps.Clone(s.values)
			s.updates = maps.Clone(s.updates)
			for k, v := range s.values {
				s.values[k] = cloneExpr(v)
			}
			for k, v := range s.updates {
				s.updates[k] = cloneExpr(v)
			}
			s.yes = cloneSteps(s.yes)
			s.no = cloneSteps(s.no)
			out[i] = s
		}
		return out
	}
	out.Steps = cloneSteps(d.Steps)
	return out, nil
}

func seriesObservation(steps []Step, p executor.Program) error {
	var walk func([]Step, bool) error
	walk = func(steps []Step, repeated bool) error {
		for _, s := range steps {
			if repeated && (s.kind == "request" || s.kind == "await") {
				return fmt.Errorf("request/await inside repetition")
			}
			for _, body := range [][]Step{s.yes, s.no} {
				if err := walk(body, repeated || s.kind == "repeat" || s.kind == "foreach"); err != nil {
					return err
				}
			}
		}
		return nil
	}
	if err := walk(steps, false); err != nil {
		return err
	}
	requests, awaits := 0, 0
	for _, in := range p.Code {
		switch in.Op {
		case "emit":
			if in.Effect == nil || in.Effect.Kind != "adapter" {
				return fmt.Errorf("non-adapter side effect")
			}
			requests++
		case "receive":
			if in.Match == nil || in.Match.Kind != "adapter" {
				return fmt.Errorf("extra input")
			}
			awaits++
		}
	}
	if requests != 1 || awaits != 1 {
		return fmt.Errorf("exactly one expanded request and await required")
	}
	// Explore finite (PC, consumed) states. Structured collection loops are
	// allowed outside the operation; revisiting a state never adds new evidence.
	type state struct {
		pc       int
		consumed bool
	}
	pending := []state{{0, false}}
	seen := map[state]bool{}
	for len(pending) > 0 {
		current := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		if seen[current] {
			continue
		}
		seen[current] = true
		if current.pc < 0 || current.pc >= len(p.Code) {
			return fmt.Errorf("observation falls through")
		}
		in := p.Code[current.pc]
		consumed := current.consumed || in.Op == "receive"
		switch in.Op {
		case "end":
			if in.Outcome == "succeeded" && !consumed {
				return fmt.Errorf("successful path skips result consumption")
			}
		case "jump":
			pending = append(pending, state{in.Target, consumed})
		case "branch":
			pending = append(pending, state{in.Target, consumed}, state{in.Otherwise, consumed})
		case "foreach":
			pending = append(pending, state{current.pc + 1, consumed}, state{in.Otherwise, consumed})
		case "loop":
			pending = append(pending, state{in.Target, consumed})
		default:
			pending = append(pending, state{current.pc + 1, consumed})
		}
	}
	return nil
}
