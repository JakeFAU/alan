// Package author compiles provisional declarative definitions into durable ALAN programs.
// Builders run only at authoring time. No callback or Go stack is retained at runtime.
package author

import (
	"encoding/json"
	"github.com/JakeFAU/alan/internal/executor"
	"github.com/JakeFAU/alan/schema"
)

const CompilerVersion = executor.AuthorCompilerVersion
const SemanticVersion = executor.AuthorSemanticVersion
const CollectionSemanticVersion = executor.CollectionSemanticVersion
const CollectionCompilerVersion = executor.CollectionCompilerVersion
const MaxInstructions = 10000
const MaxNesting = 64

type Definition struct {
	Name            string
	Invocations     map[string]Template
	Inputs, Outputs schema.Record
	Collections     map[string]schema.Field
	Values          map[string]schema.Value
	Templates       map[string]string
	Effects         map[string]Effect
	Ports           map[string]Port
	Steps           []Step
}

// Effect names one deployment capability and its complete request/result records.
// Transport execution is supplied separately by the runtime adapter registry.
type Effect struct {
	Capability    string
	Input, Output schema.Record
}

// Port is one statically wired request/reply boundary. Peer execution is generation 1.
// Local ports accept operator input and cannot send. Obligation is an explicit contract identity.
type Port struct {
	Peer, Obligation string
	Send, Receive    schema.Record
	Local            bool
}

type Expr struct{ operand executor.Operand }

func Var(name string) Expr                 { return Expr{executor.Ref(name)} }
func String(v string) Expr                 { return Expr{executor.Lit(executor.Text(v))} }
func Integer(v int64) Expr                 { return Expr{executor.Lit(executor.Int(v))} }
func Boolean(v bool) Expr                  { return Expr{executor.Lit(schema.Bool(v))} }
func Offset(name string, delta int64) Expr { return Expr{executor.Operand{Ref: name, Offset: &delta}} }

type Step struct {
	kind, name, destination, declaration, request string
	a, b                                          Expr
	values, updates                               map[string]Expr
	bindings                                      map[string]string
	yes, no                                       []Step
	count                                         int
}

// Value freezes a literal, including a bounded list used with a declared collection.
func Value(v schema.Value) Expr { return Expr{executor.Lit(v.Clone())} }
func Append(name string, values map[string]Expr) Step {
	return Step{kind: "append", destination: name, values: values}
}
func ForEach(collection Expr, bindings map[string]string, body ...Step) Step {
	return Step{kind: "foreach", a: collection, bindings: bindings, yes: body}
}

func Set(name string, value Expr) Step { return Step{kind: "set", destination: name, a: value} }
func Add(name string, a, b Expr) Step  { return Step{kind: "add", destination: name, a: a, b: b} }
func Label(name string) Step           { return Step{kind: "label", name: name} }

// Goto may only target a later label. Repeat and ForEach provide structured repetition.
func Goto(label string) Step                 { return Step{kind: "goto", name: label} }
func IfEqual(a, b Expr, yes, no []Step) Step { return Step{kind: "if", a: a, b: b, yes: yes, no: no} }
func Repeat(count int, body ...Step) Step    { return Step{kind: "repeat", count: count, yes: body} }
func Success() Step                          { return Step{kind: "end", name: "succeeded"} }
func Failure() Step                          { return Step{kind: "end", name: "failed"} }
func Send(port string, values map[string]Expr) Step {
	return Step{kind: "send", declaration: port, values: values}
}
func Receive(port string, bindings map[string]string) Step {
	return Step{kind: "receive", declaration: port, bindings: bindings}
}

// Request stores the committed intent identity in request and applies updates atomically.
func Request(request, effect string, values, updates map[string]Expr) Step {
	return Step{kind: "request", destination: request, declaration: effect, values: values, updates: updates}
}
func Await(effect, request string, bindings map[string]string) Step {
	return Step{kind: "await", declaration: effect, request: request, bindings: bindings}
}

// Artifact owns immutable encoded bytes. Accessors return independent copies.
type Artifact struct {
	id        string
	raw       string
	catalogue string
}

func (a Artifact) ID() string    { return a.id }
func (a Artifact) Bytes() []byte { return []byte(a.raw) }

type Metadata = executor.ProgramMetadata
type EffectMetadata = executor.EffectDeclaration
type PortMetadata = executor.PortDeclaration

func (a Artifact) Metadata() Metadata {
	var p executor.Program
	_ = json.Unmarshal([]byte(a.raw), &p)
	if p.Author == nil {
		return executor.ProgramMetadata{}
	}
	return *p.Author
}
