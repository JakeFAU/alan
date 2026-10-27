package executor

import (
	"context"
	"fmt"
	"slices"
)

func programHasInvocations(p Program) bool {
	if p.Author != nil && len(p.Author.Invocations) > 0 {
		return true
	}
	for _, in := range p.Code {
		if in.Effect != nil && in.Effect.Kind == "invocation" || in.Match != nil && in.Match.Kind == "invocation" {
			return true
		}
	}
	return false
}
func validateInvocationCatalogue(ctx context.Context, q queryer, c compositionProvision, l Limits) error {
	templates := map[string]compositionTemplate{}
	for _, t := range c.Templates {
		templates[t.ID] = t
	}
	validate := func(p Program, allowed []string) error {
		if p.Author == nil {
			if programHasInvocations(p) {
				return fmt.Errorf("invocation requires author declaration")
			}
			return nil
		}
		for _, d := range p.Author.Invocations {
			t, ok := templates[d.TemplateID]
			if !ok {
				return fmt.Errorf("undeployed invocation declaration")
			}
			if allowed != nil && !slices.Contains(allowed, d.TemplateID) {
				return fmt.Errorf("undeclared child template")
			}
			entry, err := loadProgram(ctx, q, t.Members[t.EntryMember].ProgramID, l)
			if err != nil {
				return err
			}
			port := entry.Author.Ports[t.EntryPort]
			if d.Input.ID != port.Receive.ID || d.Output.ID != port.Send.ID {
				return fmt.Errorf("invocation public schema mismatch")
			}
		}
		for _, in := range p.Code {
			if in.Effect != nil && in.Effect.Kind == "invocation" {
				e := in.Effect
				if e.Obligation.Literal == nil || e.Target.Literal == nil || e.Generation.Literal == nil {
					return fmt.Errorf("dynamic invocation authority")
				}
				d, ok := p.Author.Invocations[e.Obligation.Literal.Text]
				if !ok || d.TemplateID != e.Target.Literal.Text || e.Generation.Literal.Number != 1 {
					return fmt.Errorf("undeclared invocation instruction")
				}
			}
			if in.Match != nil && in.Match.Kind == "invocation" {
				m := in.Match
				if m.Obligation.Literal == nil || m.Source.Literal == nil || m.Generation.Literal == nil || m.Provenance != "runtime-invocation" {
					return fmt.Errorf("invalid invocation outcome boundary")
				}
				d, ok := p.Author.Invocations[m.Obligation.Literal.Text]
				if !ok || m.Source.Literal.Text != d.TemplateID || m.Generation.Literal.Number != 1 {
					return fmt.Errorf("undeclared invocation outcome")
				}
			}
		}
		return nil
	}
	staticMembers := map[RunID]bool{}
	for _, root := range c.Roots {
		for _, member := range root.Members {
			staticMembers[member] = true
		}
	}
	rows, err := q.QueryContext(ctx, "SELECT subject,generation FROM run_origins WHERE kind='root'")
	if err != nil {
		return err
	}
	staticRuns := []RunID{}
	for rows.Next() {
		var run RunID
		if err = rows.Scan(&run.Subject, &run.Generation); err != nil {
			rows.Close()
			return err
		}
		staticRuns = append(staticRuns, run)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for _, id := range staticRuns {
		run, err := loadRun(ctx, q, id)
		if err != nil {
			return err
		}
		p, err := loadProgram(ctx, q, run.ProgramID, l)
		if err != nil {
			return err
		}
		if programHasInvocations(p) && !staticMembers[id] {
			return fmt.Errorf("invocation source requires a declared resource root")
		}
	}
	for _, root := range c.Roots {
		for _, member := range root.Members {
			run, err := loadRun(ctx, q, member)
			if err != nil {
				return err
			}
			p, err := loadProgram(ctx, q, run.ProgramID, l)
			if err != nil {
				return err
			}
			if err = validate(p, nil); err != nil {
				return err
			}
		}
	}
	for _, t := range c.Templates {
		allowed := t.Templates
		if allowed == nil {
			allowed = []string{}
		}
		for _, member := range t.Members {
			p, err := loadProgram(ctx, q, member.ProgramID, l)
			if err != nil {
				return err
			}
			if err = validate(p, allowed); err != nil {
				return err
			}
		}
	}
	return nil
}
