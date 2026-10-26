package author

import "github.com/JakeFAU/alan/schema"

// Validate every identifier emitted by a step, including unreachable code.
// JSON must preserve distinct Go names exactly rather than replacing bad UTF-8.
func checkStepIdentifiers(s Step) error {
	var required []string
	switch s.kind {
	case "set", "add", "append":
		required = append(required, s.destination)
	case "label", "goto":
		required = append(required, s.name)
	case "send", "receive":
		required = append(required, s.declaration)
	case "request":
		required = append(required, s.destination, s.declaration)
	case "await":
		required = append(required, s.declaration, s.request)
	}
	for _, name := range required {
		if err := schema.CheckIdentifier(name); err != nil {
			return err
		}
	}
	expressions := []Expr{s.a, s.b}
	for _, values := range []map[string]Expr{s.values, s.updates} {
		for name, value := range values {
			if err := schema.CheckIdentifier(name); err != nil {
				return err
			}
			expressions = append(expressions, value)
		}
	}
	for field, destination := range s.bindings {
		for _, name := range []string{field, destination} {
			if err := schema.CheckIdentifier(name); err != nil {
				return err
			}
		}
	}
	for _, expression := range expressions {
		if expression.operand.Ref != "" {
			if err := schema.CheckIdentifier(expression.operand.Ref); err != nil {
				return err
			}
		}
	}
	return nil
}
