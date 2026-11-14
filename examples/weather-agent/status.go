package main

import (
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/JakeFAU/alan/runtime"
)

func (s summary) writeReport(out io.Writer, cut *runtime.ExecutionSnapshot, path string, showStatus bool) error {
	var buf strings.Builder
	// The ordinary CLI checks every legacy approval before running or approving
	// anything. An unrelated invalid input can therefore block those commands.
	rejected := slices.ContainsFunc(s.Reads, func(r readSummary) bool {
		return r.State == "refresh_input_rejected"
	})
	var canResume, mayContact bool
	var eligible []int
	for _, r := range s.Reads {
		next := s.nextAction(cut, r)
		if rejected {
			next = noAction
		}
		if next == approveRead {
			eligible = append(eligible, r.Read)
		}
		canResume = canResume || next == resumeLocal || next == resumeProvider
		mayContact = mayContact || next == resumeProvider
		fmt.Fprintf(&buf, "Read %d. ", r.Read)
		r.writeHumanRead(&buf, next)
		if showStatus {
			if next == resumeLocal && r.LocalStatus != "open" {
				buf.WriteString(" Local completion remains pending.")
			}
			r.writeEvidence(&buf)
		}
		buf.WriteByte('\n')
	}
	fmt.Fprintf(&buf, "Location %s, %s. Units %s. Opportunities spent %d of %d. Provider cost unknown.\n", s.Latitude, s.Longitude, s.Units, s.OpportunitiesSpent, s.Allowance)
	if showStatus {
		switch {
		case rejected:
			buf.WriteString("Invalid retained approval prevents further weather commands that advance the Store.\n")
		case canResume:
			if mayContact {
				buf.WriteString("Resume may contact the provider for provisioned or already approved work.\n")
			}
			fmt.Fprintf(&buf, "Continue with\n  weather-agent resume --db %s\n", shellQuote(path))
		}
		for _, n := range eligible {
			fmt.Fprintf(&buf, "Optional separate approval for read %d\n  weather-agent refresh --db %s --read %d\n", n, shellQuote(path), n)
		}
		if s.OpportunitiesSpent >= int64(s.Allowance) {
			buf.WriteString("No remaining read slot.\n")
		}
	}
	if s.ReplayedCommits != nil {
		fmt.Fprintf(&buf, "Replay verified %d committed reductions.\n", *s.ReplayedCommits)
	}
	fmt.Fprintln(&buf, s.Attribution)
	_, err := io.WriteString(out, buf.String())
	return err
}

type nextAction uint8

const (
	noAction nextAction = iota
	resumeProvider
	resumeLocal
	approveRead
	approvalOccupied
)

func (s summary) nextAction(cut *runtime.ExecutionSnapshot, r readSummary) nextAction {
	if r.State == "refresh_input_rejected" {
		return noAction
	}
	if r.LocalStatus != "open" {
		run, _ := cut.Run(r.Agent)
		if root, ok := cut.Root(run.RootID); ok && root.Outcome == nil && root.Pending == "" {
			return resumeLocal
		}
		return noAction
	}
	switch r.EffectStage {
	case "":
		switch {
		case r.Read == 1 && s.Allowance > 0, r.ApprovalObservationID != "":
			return resumeProvider
		case r.Read > 1:
			return nextApprovalAction(cut, r.Read)
		}
	case runtime.EffectCommitted, runtime.EffectAuthorized:
		if s.Allowance > 0 {
			return resumeProvider
		}
		return resumeLocal
	case runtime.EffectResultUnsettled, runtime.EffectSettled,
		runtime.EffectAuthorizationRefused, runtime.EffectReservationRefused:
		return resumeLocal
	}
	return noAction
}

func nextApprovalAction(cut *runtime.ExecutionSnapshot, n int) nextAction {
	if len(cut.Composition.Continuations) == 0 {
		// Valid retained approvals are handled before reaching this check.
		// A decision still occupies its input transport ID after a refusal
		// or when it targeted a different route.
		id := approvalID(n)
		for _, a := range cut.Admissions {
			if a.Domain == "input" && a.TransportID == id {
				return approvalOccupied
			}
		}
		if _, err := legacyApprovalPayload(cut, n); err == nil {
			return approveRead
		}
	} else if _, _, err := weatherApprovalBasis(cut, n); err == nil {
		return approveRead
	}
	return noAction
}

func (r readSummary) writeHumanRead(buf *strings.Builder, next nextAction) {
	if r.EffectStage == runtime.EffectReservedUnknown {
		if r.DispatchAdmission == nil {
			buf.WriteString("The opportunity was spent before local dispatch. The provider was not contacted.")
		} else {
			buf.WriteString("The local dispatch gate admitted this lookup. Its provider response is unknown. A separate read does not settle it.")
		}
		buf.WriteString(" Resume will not retry this original.")
		return
	}
	switch r.State {
	case "reported":
		if r.ResultOrigin == "previously_retained" {
			buf.WriteString("Previously retained. ")
		} else {
			buf.WriteString("Read during this run. ")
		}
		buf.WriteString(r.Report)
	case "awaiting_refresh_approval":
		switch next {
		case approveRead:
			buf.WriteString("Eligible for separate approval of a new observation.")
		case approvalOccupied:
			fmt.Fprintf(buf, "Approval blocked because a retained input already uses transport ID %q.", approvalID(r.Read))
		default:
			fmt.Fprintf(buf, "Waiting for read %d to qualify as a predecessor.", r.Read-1)
		}
	case "refresh_approved":
		buf.WriteString("Approved.")
		if next == resumeProvider {
			buf.WriteString(" Run or resume to request the new observation.")
		}
	case "refresh_input_rejected":
		buf.WriteString(r.RefreshInputIssue)
	case "unavailable":
		if r.Disposition == "" {
			buf.WriteString("Weather unavailable.")
		} else {
			fmt.Fprintf(buf, "Weather unavailable (%s).", r.Disposition)
		}
	default:
		switch {
		case next == resumeLocal && r.HTTPStatus != 0:
			buf.WriteString("Retained result needs local work.")
		case next == resumeLocal:
			buf.WriteString("Local work remains.")
		case next == resumeProvider && r.Read == 1:
			buf.WriteString("Original read can run with its provisioned opportunity.")
		case next == resumeProvider:
			buf.WriteString("Approved read can run with its own opportunity.")
		default:
			buf.WriteString(strings.ReplaceAll(r.State, "_", " "))
		}
	}
}

func (r readSummary) writeEvidence(buf *strings.Builder) {
	if r.OperationID != "" {
		fmt.Fprintf(buf, " Original operation %s.", r.OperationID)
	}
	if r.HTTPStatus != 0 {
		fmt.Fprintf(buf, " Response retained (HTTP %d).", r.HTTPStatus)
	} else {
		buf.WriteString(" Response absent.")
	}
	if r.SelectedAt != nil {
		buf.WriteString(" Observation selected.")
	} else {
		buf.WriteString(" Observation unselected.")
	}
	if r.OpportunitySpent {
		buf.WriteString(" Opportunity spent.")
	} else {
		buf.WriteString(" Opportunity unspent.")
	}
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}
