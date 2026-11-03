// Command inspect-execution explains one retained ALAN store without becoming
// its owner. It uses only the public snapshot and never registers an adapter.
package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/JakeFAU/alan/runtime"
)

func run(ctx context.Context, path string, out io.Writer) error {
	s, err := runtime.InspectExecution(ctx, path, runtime.DefaultLimits())
	if err != nil {
		return err
	}
	var b strings.Builder
	page, err := s.History(0, 1)
	if err != nil {
		return err
	}
	fmt.Fprintf(&b, "store version=%d namespace=%s history_entries=%d\n", s.StorageVersion, s.Namespace, page.Total)
	if s.Composition == nil {
		fmt.Fprintln(&b, "composition_profile=not-retained")
	}
	for _, root := range s.Roots {
		fmt.Fprintf(&b, "root %q generation=%d local=%s", root.Run.Subject, root.Run.Generation, root.LocalStatus)
		if root.Outcome != nil {
			fmt.Fprintf(&b, " published_outcome=%s", root.Outcome.Status)
		}
		fmt.Fprintf(&b, " invocations=%d/%d created_runs=%d/%d effect_reservations=%d/%d local_capacity=%d\n",
			root.Charges.Invocations, root.Limits.MaxInvocations,
			root.Charges.CreatedRuns, root.Limits.MaxCreatedRuns,
			root.Charges.Effects, root.Limits.MaxEffects, root.Limits.MaxInFlight)
	}

	for _, r := range s.Runs {
		fmt.Fprintf(&b, "run %q generation=%d epoch=%d sequence=%d program=%s semantics=%s local=%s",
			r.ID.Subject, r.ID.Generation, r.Epoch, r.Sequence, r.ProgramID, r.Semantics, r.Projection.Status)
		if r.OwningInvocationID != "" {
			fmt.Fprintf(&b, " owning_invocation=%s", r.OwningInvocationID)
			if parent, ok := s.InvocationByID(r.OwningInvocationID); ok {
				if role := parent.Inputs["role"]; role.Kind == "string" {
					fmt.Fprintf(&b, " worker_role=%q", role.Text)
				}
			}
		}
		fmt.Fprintln(&b)
		// These are application fields. All edges below come from ALAN.
		if question := r.Initial.Data["question"]; question.Kind == "string" {
			fmt.Fprintf(&b, "  question=%q\n", question.Text)
		}
		if answer := r.Projection.Data["completion"]; answer.Kind == "string" {
			fmt.Fprintf(&b, "  authored_completion=%q\n", answer.Text)
		}
		for _, op := range r.InvocationIDs {
			call, ok := s.Invocation(op)
			if !ok {
				return fmt.Errorf("snapshot references absent invocation %q", op)
			}
			fmt.Fprintf(&b, "  invocation operation=%q state=%s", op, call.State)
			if a := call.Admission; a != nil {
				fmt.Fprintf(&b, " id=%s admitted=%t reason=%q", a.Invocation.ID, a.Accepted, a.Reason)
				for _, child := range a.Members {
					fmt.Fprintf(&b, " child=%q/%d", child.Subject, child.Generation)
				}
			}
			if outcome := call.Outcome; outcome != nil {
				fmt.Fprintf(&b, " published=%s outcome=%s reason=%q response=%t",
					outcome.ID, outcome.Status, outcome.Reason, outcome.ResponsePresent)
				if verdict := outcome.Response["verdict"]; outcome.ResponsePresent && verdict.Kind == "string" && verdict.Text != "" {
					fmt.Fprintf(&b, " worker_verdict=%q", verdict.Text)
				}
				if call.SelectedAt == nil {
					fmt.Fprintf(&b, " parent_selection=%s", call.ParentSelection)
				} else {
					fmt.Fprintf(&b, " parent_selected_sequence=%d", call.SelectedAt.Sequence)
				}
			}
			fmt.Fprintln(&b)
		}
		for _, op := range r.EffectIDs {
			effect, ok := s.Effect(op)
			if !ok {
				return fmt.Errorf("snapshot references absent effect %q", op)
			}
			writeEffect(&b, s, effect)
		}
	}
	for _, obligation := range s.Pending {
		fmt.Fprintf(&b, "pending run=%q/%d kind=%s operation=%q observation=%q depends_on=%q",
			obligation.Run.Subject, obligation.Run.Generation, obligation.Kind,
			obligation.OperationID, obligation.ObservationID, obligation.DependsOn)
		for _, admission := range obligation.AdmissionIDs {
			fmt.Fprintf(&b, " delivery_attempt=%q", admission)
		}
		if member := obligation.DependsOnRun; member != nil {
			fmt.Fprintf(&b, " depends_on_run=%q/%d", member.Subject, member.Generation)
		}
		fmt.Fprintln(&b)
	}
	_, err = io.WriteString(out, b.String())
	return err
}

func writeEffect(b *strings.Builder, s *runtime.ExecutionSnapshot, e *runtime.ExecutionEffect) {
	fmt.Fprintf(b, "  effect operation=%q capability=%q name=%q state=%s physical_executions=%d..",
		e.OperationID, e.Capability, e.Name, e.State, e.Dispatch.Minimum)
	if e.Dispatch.Maximum == nil {
		fmt.Fprint(b, "?")
	} else {
		fmt.Fprint(b, *e.Dispatch.Maximum)
	}
	fmt.Fprintf(b, " consequence=%s", e.Consequence)
	if auth := e.Authorization; auth != nil {
		fmt.Fprintf(b, " authorization=%q source_revision=%d", auth.ID, auth.Source.Revision)
	}
	if settlement := e.Settlement; settlement != nil {
		fmt.Fprintf(b, " settlement=%q", settlement.ID)
	}
	a, r := e.AdapterAttempt, e.AdapterResult
	if a != nil {
		fmt.Fprintf(b, " allowed=%t reason=%q", a.Allowed, a.Reason)
	}
	if r != nil {
		fmt.Fprintf(b, " retained_disposition=%q", r.Disposition)
	}
	if a != nil && a.Prepared.Protocol == "openai/responses/v1" {
		fmt.Fprintf(b, " model=%q", a.Prepared.Model)
		if r != nil {
			fmt.Fprintf(b, " http_status=%d response_id=%q", r.Candidate.HTTPStatus, r.Candidate.ResponseID)
		}
		if e.Usage == nil {
			fmt.Fprint(b, " reported_usage=unknown")
		} else {
			fmt.Fprintf(b, " reported_tokens[input=%s output=%s total=%s]",
				counter(e.Usage.InputTokens), counter(e.Usage.OutputTokens), counter(e.Usage.TotalTokens))
		}
		if a.Allowed && r == nil {
			fmt.Fprint(b, " remote_outcome=unknown retry=not-automatically-authorized")
		}
	}
	fmt.Fprintln(b)
	if a != nil && a.Prepared.Protocol == "lookup/tokens/v1" && r != nil && r.Disposition == "validated" {
		p := r.Candidate.Payload
		id, version, quote := p["document"].Text, p["version"].Text, p["snippet"].Text
		start, end := p["start"].Number, p["end"].Number
		doc, ok := s.Asset(id)
		verified := ok && doc.Version == version && doc.Check() == nil && start >= 0 && start < end &&
			end <= int64(len(doc.Content)) && utf8.ValidString(quote) && doc.Content[start:end] == quote
		fmt.Fprintf(b, "    evidence document=%q sha256=%s bytes=[%d,%d) matches_source=%t quote=%q\n",
			id, version, start, end, verified, quote)
	}
}

func counter(n *int64) string {
	if n == nil {
		return "?"
	}
	return fmt.Sprint(*n)
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: inspect-execution STORE")
		os.Exit(2)
	}
	if err := run(context.Background(), os.Args[1], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
