package main

import (
	"cmp"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"github.com/JakeFAU/alan/adapters/openmeteo"
	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/effects"
	"github.com/JakeFAU/alan/runtime"
	"github.com/JakeFAU/alan/schema"
)

func TestDelegatedWeatherPublicPrograms(t *testing.T) {
	for _, tc := range []struct {
		name           string
		eligible       int
		allowance      int64
		cedar, hemlock string
		outcome, park  string
	}{
		{name: "empty", allowance: 1, outcome: noMatch},
		{name: "one", eligible: 1, allowance: 1, cedar: "1", outcome: recommended, park: "cedar"},
		{name: "two", eligible: 2, allowance: 1, cedar: "97", hemlock: "1", outcome: recommended, park: "hemlock"},
		{name: "known unavailable", eligible: 2, allowance: 1, cedar: "999", hemlock: "97", outcome: unavailable},
		{name: "refused", eligible: 2, allowance: 0},
		{name: "empty refused", allowance: 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server, calls := fixtureServer(t, tc.cedar, tc.hemlock)
			c := fixtureConfiguration()
			c.Endpoint = server.URL + fixturePath
			switch tc.eligible {
			case 0:
				c.MaxTravelMinutes = 2
			case 1:
				c.MaxTravelMinutes = 10
			}
			options := deploymentOptions{Weather: delegatedWeather, HelperAdmission: tc.allowance}
			p, err := c.provisionDeployment(options)
			if err != nil {
				t.Fatal(err)
			}
			initial, cut := executeDelegatedProvision(t, p, c)
			before, err := retainedDeployment(initial)
			if err != nil || before.Options != options || before.HelperRun != nil ||
				before.HelperTemplateID == "" || !reflect.DeepEqual(before.Configuration, c) {
				t.Fatalf("retained initial deployment: %+v, %v", before, err)
			}
			info, err := retainedDeployment(cut)
			if err != nil || info.Options != options || info.HelperTemplateID != before.HelperTemplateID ||
				!reflect.DeepEqual(info.Configuration, c) {
				t.Fatalf("retained deployment: %+v, %v", info, err)
			}
			if len(cut.Invocations) != 1 {
				t.Fatalf("invocations: %d, want 1", len(cut.Invocations))
			}
			call := cut.Invocations[0]
			if call.Caller != weatherRun || call.TemplateID != info.HelperTemplateID ||
				call.Admission == nil || call.Outcome == nil || call.SelectedAt == nil || call.SelectedAt.Run != weatherRun {
				t.Fatalf("unexpected original invocation: %+v", call)
			}
			weatherRoot := rootForTest(t, cut, weatherRun)
			outingRoot := rootForTest(t, cut, outingRun)
			if tc.allowance == 0 {
				weather, _ := cut.Run(weatherRun)
				outing, _ := cut.Run(outingRun)
				if call.Admission.Accepted || call.Admission.Reason != "invocation_limit" ||
					call.Outcome.Status != "rejected" || call.Outcome.ResponsePresent ||
					info.HelperRun != nil || len(cut.Runs) != 2 || calls.Load() != 0 || len(cut.Effects) != 0 ||
					weatherRoot.Charges.Invocations != 0 || weatherRoot.Charges.CreatedRuns != 0 ||
					weatherRoot.Charges.Effects != 0 || len(weatherRoot.PendingInvocations) != 0 ||
					weather.Projection.Status != "failed" || outing.Wait == nil || outing.Wait.Obligation != assessmentPort ||
					weatherRoot.Outcome == nil || weatherRoot.Outcome.Status != "failed" || outingRoot.Outcome != nil {
					t.Fatalf("unexpected refusal: %+v, weather root %+v, outing root %+v", call, weatherRoot, outingRoot)
				}
				requireNoDelegatedPublicResult(t, cut)
				return
			}
			if !call.Admission.Accepted || len(call.Admission.Members) != 1 || info.HelperRun == nil ||
				*info.HelperRun != call.Admission.Members[0] || len(cut.Runs) != 3 ||
				call.Outcome.Status != "succeeded" || !call.Outcome.ResponsePresent ||
				calls.Load() != int32(tc.eligible) || len(cut.Effects) != tc.eligible ||
				weatherRoot.Charges.Invocations != 1 || weatherRoot.Charges.CreatedRuns != 1 ||
				weatherRoot.Charges.Effects != int64(tc.eligible) {
				t.Fatalf("unexpected accepted call: %+v, %d contacts, weather root %+v", call, calls.Load(), weatherRoot)
			}
			assertDelegatedOrderedResponse(t, cut, c, call, *info.HelperRun)
			assertDelegatedDecision(t, cut, tc.outcome, tc.park)
			status := "succeeded"
			if tc.outcome == unavailable {
				status = "failed"
			}
			worker, _ := cut.Run(*info.HelperRun)
			if worker.Projection.Status != "succeeded" || weatherRoot.Outcome == nil ||
				weatherRoot.Outcome.Status != status || outingRoot.Outcome == nil || outingRoot.Outcome.Status != status {
				t.Fatalf("terminal state: helper %s, weather root %+v, outing root %+v",
					worker.Projection.Status, weatherRoot, outingRoot)
			}
		})
	}
}

func TestDelegatedWeatherRefusalRetainsCallerObligation(t *testing.T) {
	c := fixtureConfiguration()
	c.MaxTravelMinutes = 2
	options := deploymentOptions{Weather: delegatedWeather, HelperAdmission: 0}
	p, err := c.provisionDeployment(options)
	if err != nil {
		t.Fatal(err)
	}
	ctx, limits := t.Context(), runtime.DefaultLimits()
	path := filepath.Join(t.TempDir(), "refusal.db")
	store, err := runtime.Create(ctx, path, p, limits)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := store.Close(); err != nil {
			t.Error(err)
		}
	}()

	// Use individual public commits so the dispatch loop cannot settle the
	// refusal before the two waiting states have been inspected.
	inspect := func() *runtime.ExecutionSnapshot {
		t.Helper()
		s, err := runtime.InspectExecution(ctx, path, limits)
		if err != nil {
			t.Fatal(err)
		}
		info, err := retainedDeployment(s)
		if err != nil || info.Options != options || info.HelperRun != nil {
			t.Fatalf("unexpected deployment at partial refusal cut: %+v, %v", info, err)
		}
		return s
	}
	advanceTo := func(id runtime.RunID, kind string, selectID string) string {
		t.Helper()
		for range 40 {
			s := inspect()
			for _, op := range s.Operations {
				if op.Run == id && op.Kind == kind {
					return op.ID
				}
			}
			r, _ := s.Run(id)
			observationID := ""
			if r.Wait != nil {
				observationID = selectID
			}
			if _, err := store.Commit(ctx, r.Authority(), observationID); err != nil {
				t.Fatal(err)
			}
		}
		t.Fatalf("run %+v did not reach %s", id, kind)
		return ""
	}
	proposal := advanceTo(outingRun, "message", "")
	delivered, err := store.DeliverMessage(ctx, proposal, "delegated-proposal-test")
	if err != nil || !delivered.Accepted {
		t.Fatalf("deliver original proposal: %+v, %v", delivered, err)
	}
	op := advanceTo(weatherRun, "invocation", delivered.ObservationID)
	before := inspect()
	if got := rootForTest(t, before, weatherRun).PendingInvocations; !slices.Equal(got, []string{op}) {
		t.Fatalf("original invocation awaiting admission: %v", got)
	}
	admission, err := store.AdmitInvocation(ctx, op)
	if err != nil || admission.Accepted || admission.Reason != "invocation_limit" {
		t.Fatalf("admit invocation: %+v, %v", admission, err)
	}
	refused := inspect()
	weather, _ := refused.Run(weatherRun)
	if len(rootForTest(t, refused, weatherRun).PendingInvocations) != 0 ||
		refused.Invocations[0].Outcome != nil || weather.Wait == nil || weather.Wait.OperationID != op ||
		!hasDelegatedPending(refused, "waiting_invocation", op, "") ||
		!hasDelegatedPending(refused, "invocation_awaiting_outcome", op, "") {
		t.Fatalf("refusal should retain caller wait and delivery debt: %+v, %+v", weather.Wait, refused.Pending)
	}
	settled, err := store.SettleInvocation(ctx, op)
	if err != nil || !settled.Accepted || settled.ObservationID == "" {
		t.Fatalf("publish refusal: %+v, %v", settled, err)
	}
	published := inspect()
	if len(rootForTest(t, published, weatherRun).PendingInvocations) != 0 ||
		published.Invocations[0].ParentSelection != "unconsumed" ||
		!hasDelegatedPending(published, "matching_observation", op, settled.ObservationID) ||
		!hasDelegatedPending(published, "published_unconsumed", op, settled.ObservationID) {
		t.Fatalf("unconsumed refusal should retain caller delivery evidence: %+v", published.Pending)
	}
}

func hasDelegatedPending(s *runtime.ExecutionSnapshot, kind, operationID, observationID string) bool {
	return slices.ContainsFunc(s.Pending, func(p runtime.PendingObligation) bool {
		return p.Kind == kind && p.Run == weatherRun && p.OperationID == operationID && p.ObservationID == observationID
	})
}

func assertDelegatedOrderedResponse(
	t *testing.T,
	cut *runtime.ExecutionSnapshot,
	c configuration,
	call *runtime.InvocationView,
	helper runtime.RunID,
) {
	t.Helper()
	plan, err := c.normalize()
	if err != nil {
		t.Fatal(err)
	}
	rows, err := call.Outcome.Response[assessments].Records()
	if err != nil || len(rows) != len(cut.Effects) {
		t.Fatalf("response coverage: %v, %v", rows, err)
	}
	forecast := slices.Clone(cut.Effects)
	slices.SortFunc(forecast, func(a, b *runtime.ExecutionEffect) int {
		return cmp.Compare(a.Sequence, b.Sequence)
	})
	request, ok := cut.Observation(call.RequestObservationID)
	if !ok || request.SelectedAt == nil || request.SelectedAt.Run != helper {
		t.Fatalf("helper did not select original request: %+v", request)
	}
	previous := request.SelectedAt.Sequence
	next := 0
	for _, candidate := range plan.Candidates {
		if !candidate.Eligible {
			continue
		}
		row, effect := rows[next], forecast[next]
		next++
		for key, value := range candidateRecord(candidate) {
			if !row[key].Equal(value) {
				t.Fatalf("response row for %s differs at %s", candidate.ID, key)
			}
		}
		if row["forecast_operation"].Text != effect.OperationID || effect.Source != helper ||
			effect.Sequence <= previous || effect.Settlement == nil {
			t.Fatalf("response does not follow the original sequential forecast: %+v", effect)
		}
		result, ok := cut.Observation(effect.Settlement.ObservationID)
		if !ok || result.SelectedAt == nil || result.SelectedAt.Run != helper {
			t.Fatalf("helper did not select forecast result for %s", effect.OperationID)
		}
		previous = result.SelectedAt.Sequence
	}
	if next != len(rows) {
		t.Fatalf("assessed %d eligible candidates, response contained %d", next, len(rows))
	}
	reply, ok := cut.Operation(call.ResponseOperationID)
	if !ok || reply.Run != helper || reply.Sequence <= previous || !equalPayload(reply.Payload, call.Outcome.Response) {
		t.Fatalf("published response did not come from helper reply: %+v", reply)
	}
	for _, op := range cut.Operations {
		if op.Kind == "message" && op.Obligation == assessmentPort && op.Run == weatherRun {
			if op.Sequence <= call.SelectedAt.Sequence || !equalPayload(op.Payload, call.Outcome.Response) {
				t.Fatalf("public assessment precedes selection or differs from the original response: %+v", op)
			}
			return
		}
	}
	t.Fatal("missing public assessment")
}

func assertDelegatedDecision(t *testing.T, cut *runtime.ExecutionSnapshot, wantOutcome, park string) {
	t.Helper()
	for _, op := range cut.Operations {
		if op.Kind != "message" || op.Obligation != decisionPort {
			continue
		}
		chosen, err := op.Payload[selection].Records()
		if err != nil || op.Payload[outcome].Text != wantOutcome ||
			(park == "" && len(chosen) != 0) ||
			(park != "" && (len(chosen) != 1 || chosen[0]["park_id"].Text != park)) ||
			selectedMessage(cut, &op, weatherRun) == nil {
			t.Fatalf("unexpected public decision: %+v (%v)", op, err)
		}
		return
	}
	t.Fatal("missing public decision")
}

func TestDelegatedWeatherRejectsBadHelperResponses(t *testing.T) {
	for _, defect := range []string{"absent", "failed", "identity", "missing row", "order", "candidate field", "duplicate handle", "empty handle"} {
		t.Run(defect, func(t *testing.T) {
			c := fixtureConfiguration()
			p := provisionMalformedHelper(t, c, defect)
			_, cut := executeDelegatedProvision(t, p, c)
			if len(cut.Invocations) != 1 || cut.Invocations[0].SelectedAt == nil {
				t.Fatalf("bad helper response was not selected: %+v", cut.Invocations)
			}
			call := cut.Invocations[0]
			wantStatus := "succeeded"
			if defect == "absent" || defect == "failed" {
				wantStatus = "failed"
			}
			if call.Outcome == nil || call.Outcome.ResponsePresent != (defect != "absent") ||
				call.Outcome.Status != wantStatus {
				t.Fatalf("wrong adversarial outcome for %s: %+v", defect, call.Outcome)
			}
			weather, _ := cut.Run(weatherRun)
			if weather.Projection.Status != "failed" {
				t.Fatalf("weather accepted %s: %+v", defect, weather.Projection)
			}
			requireNoDelegatedPublicResult(t, cut)
		})
	}
}

func provisionMalformedHelper(t *testing.T, c configuration, defect string) runtime.Provision {
	t.Helper()
	plan, err := c.normalize()
	if err != nil {
		t.Fatal(err)
	}
	rows := make([]map[string]schema.Value, 0, len(plan.Candidates))
	for i, candidate := range plan.Candidates {
		row := candidateRecord(candidate)
		row["forecast_operation"] = schema.Text(string(rune('a' + i)))
		row["weather_status"] = schema.Text(suitable)
		rows = append(rows, row)
	}
	responseID := author.Var(proposalID)
	switch defect {
	case "identity":
		responseID = author.String("wrong")
	case "missing row":
		rows = rows[:1]
	case "order":
		rows[0], rows[1] = rows[1], rows[0]
	case "candidate field":
		rows[0]["travel_minutes"] = schema.Int(29)
	case "duplicate handle":
		rows[1]["forecast_operation"] = rows[0]["forecast_operation"]
	case "empty handle":
		rows[0]["forecast_operation"] = schema.Text("")
	}
	request := extendRecord(proposalSchema, schema.Record{
		openmeteo.Units: hourlyInput[openmeteo.Units], "allowed_weather_codes": weatherCodes,
	})
	reply := []author.Step{author.Receive("work", recordBindings(request, ""))}
	if defect != "absent" {
		reply = append(reply, author.Send("work", map[string]author.Expr{
			proposalID: responseID, assessments: author.Value(schema.List(rows...)),
		}))
	}
	if defect == "failed" {
		reply = append(reply, author.Failure())
	} else {
		reply = append(reply, author.Success())
	}
	worker, err := author.Compile(author.Definition{
		Ports: map[string]author.Port{"work": {
			Peer: "caller", Obligation: "assess", Receive: request, Send: assessmentSchema,
		}},
		Steps: reply,
	})
	if err != nil {
		t.Fatal(err)
	}
	helper, err := author.NewTemplate(author.Application{"worker": {Artifact: worker}}, "worker", "work")
	if err != nil {
		t.Fatal(err)
	}
	parent, err := author.Compile(delegatedWeatherDefinition(helper))
	if err != nil {
		t.Fatal(err)
	}
	outing, err := author.Compile(outingDefinition())
	if err != nil {
		t.Fatal(err)
	}
	base, err := c.provision(delegatedWeather)
	if err != nil {
		t.Fatal(err)
	}
	p, err := (author.Application{
		outingSubject:  {Artifact: outing, Inputs: base.Runs[0].Projection.Data},
		weatherSubject: {Artifact: parent, Inputs: base.Runs[1].Projection.Data},
	}).Provision()
	if err != nil {
		t.Fatal(err)
	}
	p.Composition.Roots = base.Composition.Roots
	p.Composition.Roots[1].Templates = []string{helper.ID()}
	p.Composition.RecordDispatchAdmissions = true
	return p
}

func requireNoDelegatedPublicResult(t *testing.T, cut *runtime.ExecutionSnapshot) {
	t.Helper()
	for _, op := range cut.Operations {
		if op.Kind == "message" && (op.Obligation == assessmentPort || op.Obligation == decisionPort) {
			t.Fatalf("bad or refused helper call produced a public result: %+v", op)
		}
	}
}

func rootForTest(t *testing.T, cut *runtime.ExecutionSnapshot, run runtime.RunID) runtime.RootCompletion {
	t.Helper()
	for _, root := range cut.Roots {
		if root.Run == run {
			return root
		}
	}
	t.Fatalf("missing root outcome for %+v", run)
	return runtime.RootCompletion{}
}

func executeDelegatedProvision(
	t *testing.T,
	p runtime.Provision,
	c configuration,
) (before, after *runtime.ExecutionSnapshot) {
	t.Helper()
	ctx, limits := t.Context(), runtime.DefaultLimits()
	path := filepath.Join(t.TempDir(), "delegated.db")
	store, err := runtime.Create(ctx, path, p, limits)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := store.Close(); err != nil {
			t.Error(err)
		}
	}()
	before, err = runtime.InspectExecution(ctx, path, limits)
	if err != nil {
		t.Fatal(err)
	}
	adapter, err := openmeteo.NewHourly(c.Endpoint)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Run(ctx, runtime.RunOptions{
		Adapters: map[string]effects.Adapter{capability: adapter},
		MaxSteps: 8000, MaxDispatch: 128, MaxInFlight: 1,
	}); err != nil {
		t.Fatal(err)
	}
	after, err = runtime.InspectExecution(ctx, path, limits)
	if err != nil {
		t.Fatal(err)
	}
	replay, err := runtime.Replay(ctx, path, limits)
	if err != nil {
		t.Fatal(err)
	}
	for _, state := range replay.Runs {
		r, ok := after.Run(state.ID)
		if !ok || !reflect.DeepEqual(r.RunState, state) {
			t.Fatalf("replay differs for %+v", state.ID)
		}
	}
	return before, after
}
