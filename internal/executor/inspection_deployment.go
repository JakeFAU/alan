package executor

import (
	"encoding/json"
	"reflect"
	"slices"
)

// Index deployment identity once. A dynamic member's subject is derived from
// its admitted invocation and member name, never from an application payload.
func (b *inspectionBuilder) buildDeployment() error {
	origins := make(map[string]rootOrigin)
	retries := make(map[string]LifecycleResult)
	for _, r := range b.tables["lifecycle_results"] {
		v, err := inspectionDecode[LifecycleResult](r, "result")
		if err != nil {
			return err
		}
		if v.Request.Intent.ID != r["id"] || v.DecisionSequence != r["decision_sequence"] {
			return corrupt("lifecycle identity")
		}
		retries[v.Request.Intent.ID] = v
	}
	for _, row := range b.tables["run_origins"] {
		id := inspectionRun(row)
		run := b.runs[id]
		o, err := inspectionDecode[rootOrigin](row, "descriptor")
		if err != nil {
			return err
		}
		if run == nil || run.Origin != "" || o.Subject.ID != id.Subject || o.ProgramID != run.ProgramID || !reflect.DeepEqual(o.Initial, run.Initial) {
			return corrupt("original run binding %+v", id)
		}
		run.Origin = inspectText(row, "kind")
		switch run.Origin {
		case "root":
			if o.Subject.Generation != id.Generation || row["operation_id"] != nil || row["predecessor_generation"] != nil || o.Subject.Revision < 1 || o.Subject.Owner != run.Subject.Owner {
				return corrupt("invalid static origin %+v", id)
			}
			origins[id.Subject] = o
		case "retry":
			prev := RunID{id.Subject, inspectInt(row, "predecessor_generation")}
			op := inspectText(row, "operation_id")
			res, ok := retries[op]
			if !ok || !res.Accepted || res.Successor != id.Generation || res.Before == nil || res.Before.Run != prev || b.runs[prev] == nil || prev.Generation+1 != id.Generation {
				return corrupt("retry predecessor %+v", id)
			}
			run.Predecessor = &prev
			run.OriginOperationID = op
		default:
			return corrupt("unknown run origin")
		}
	}
	if !isCompositionSchema(protocolSchemaNumber(b.snap.StorageVersion)) {
		for _, run := range b.snap.Runs {
			if run.Origin == "" {
				return corrupt("missing origin for %+v", run.ID)
			}
		}
		return nil
	}
	profiles := b.tables["composition_profile"]
	deployments := b.tables["composition_deployment"]
	if len(profiles) != 1 || len(deployments) != 1 {
		return corrupt("missing or duplicated composition deployment")
	}
	if len(inspectText(profiles[0], "descriptor")) > b.limits.MaxProgramBytes {
		return ErrLimit
	}
	profile, err := inspectionDecode[compositionProvision](profiles[0], "descriptor")
	if err != nil {
		return err
	}
	if (profile.RequestAuthority != "") != b.snap.Features.RequestAuthority || profile.RequestAuthority != "" && (profile.RequestAuthority != RequestAuthorityVersion || len(profile.Templates) != 0 || len(profile.Continuations) != 0 || profile.RecordDispatchAdmissions) {
		return ErrVersion
	}
	if profile.Version != compositionProtocolVersion || profiles[0]["version"] != compositionProtocolVersion || profile.RecordDispatchAdmissions != b.snap.Features.DispatchAdmissions ||
		(len(profile.Continuations) != 0) != hasContinuationSchema(b.snap.StorageVersion) {
		return ErrVersion
	}
	if len(profile.Roots) == 0 || validateCompositionEncoding(profile) != nil {
		return corrupt("invalid resource profile")
	}
	b.snap.Composition = &profile
	deploy, err := inspectionDecode[compositionDeployment](deployments[0], "descriptor")
	if err != nil {
		return err
	}
	if len(deploy.Runs) != len(origins) || len(deploy.Subjects) != len(origins) {
		return corrupt("original deployment cardinality")
	}
	seenRuns, seenSubjects := map[RunID]bool{}, map[string]bool{}
	for _, r := range deploy.Runs {
		o, ok := origins[r.ID.Subject]
		if !ok || seenRuns[r.ID] || o.Subject.Generation != r.ID.Generation || o.ProgramID != r.ProgramID || !reflect.DeepEqual(o.Initial, r.Projection) {
			return corrupt("original deployment run")
		}
		seenRuns[r.ID] = true
	}
	for _, s := range deploy.Subjects {
		if seenSubjects[s.ID] || origins[s.ID].Subject != s {
			return corrupt("original deployment subject")
		}
		seenSubjects[s.ID] = true
	}
	grants := b.tables["composition_permissions"]
	if len(grants) != 1 {
		return corrupt("missing original permissions")
	}
	perms, err := inspectionDecode[[]Permission](grants[0], "descriptor")
	if err != nil {
		return err
	}
	b.snap.Deployment = &ExecutionDeployment{Subjects: deploy.Subjects, Runs: deploy.Runs, Permissions: perms}
	templates := map[string]compositionTemplate{}
	lookup := func(id string) (Program, error) {
		p, ok := b.programs[id]
		if !ok {
			return p, ErrCorrupt
		}
		return p, nil
	}
	if len(profile.Continuations) != 0 {
		if err := validateContinuationSlots(profile, deploy, perms, lookup); err != nil {
			return corrupt("invalid continuation deployment: %v", err)
		}
		x := indexContinuations(profile.Continuations)
		b.continuations = &x
	}
	for _, t := range profile.Templates {
		checked, err := retainTemplate(t, b.limits, lookup)
		if err != nil || checked.ID != t.ID || templates[t.ID].ID != "" {
			return corrupt("invalid retained template %s", t.ID)
		}
		templates[t.ID] = t
	}
	for _, t := range profile.Templates {
		for _, id := range t.Templates {
			if _, ok := templates[id]; !ok {
				return corrupt("undeployed template dependency")
			}
		}
		allowed := t.Templates
		if allowed == nil {
			allowed = []string{}
		}
		for _, member := range t.Members {
			if err := validateInvocationDeclarations(b.programs[member.ProgramID], allowed, templates, lookup); err != nil {
				return corrupt("invalid template declaration: %v", err)
			}
		}
	}
	if len(templates) != len(b.tables["composition_templates"]) {
		return corrupt("template catalogue size")
	}
	for _, r := range b.tables["composition_templates"] {
		t, err := inspectionDecode[compositionTemplate](r, "descriptor")
		if err != nil || t.ID != r["id"] || !reflect.DeepEqual(templates[t.ID], t) {
			return corrupt("template catalogue binding")
		}
	}
	roots := map[string]compositionRoot{}
	b.snap.Roots = make([]RootCompletion, 0, len(profile.Roots))
	for _, r := range profile.Roots {
		run := b.runs[r.Run]
		if run == nil || r.ID != compositionRootID(b.snap.Namespace, r.Run) || roots[r.ID].ID != "" || !slices.Contains(r.Members, r.Run) || r.Version != compositionResourceVersion || r.Limits.MaxInvocations < 0 || r.Limits.MaxCreatedRuns < 0 || r.Limits.MaxEffects < 0 || r.Limits.MaxDepth < 0 || r.Limits.MaxInFlight <= 0 {
			return corrupt("invalid resource root")
		}
		allowed, err := canonicalStrings(r.Templates)
		capabilities, capErr := canonicalStrings(r.Capabilities)
		if err != nil || capErr != nil || !slices.Equal(allowed, r.Templates) || !slices.Equal(capabilities, r.Capabilities) || !slices.IsSortedFunc(r.Members, runLess) {
			return corrupt("noncanonical resource authority")
		}
		for _, tid := range r.Templates {
			if _, ok := templates[tid]; !ok {
				return corrupt("undeployed root template")
			}
		}
		for _, id := range r.Members {
			if b.runs[id] == nil || b.runs[id].Origin != "root" || b.membership[id] != "" {
				return corrupt("invalid static root membership")
			}
			if id.Generation != 1 || b.runs[id].Subject.Generation != 1 {
				return corrupt("invalid static member generation")
			}
			p := b.programs[b.runs[id].ProgramID]
			if err := validateInvocationDeclarations(p, nil, templates, lookup); err != nil {
				return corrupt("invalid static invocation declaration: %v", err)
			}
			if p.Author != nil {
				for _, e := range p.Author.Effects {
					if !slices.Contains(r.Capabilities, e.Capability) {
						return corrupt("static effect lacks root capability")
					}
				}
			}
			b.membership[id] = r.ID
			b.runs[id].RootID = r.ID
		}
		roots[r.ID] = r
		b.snap.Roots = append(b.snap.Roots, RootCompletion{RootID: r.ID, Run: r.Run, LocalStatus: run.Projection.Status, Limits: r.Limits, Charges: compositionCharge{RootID: r.ID}, PendingInvocations: []string{}, UnresolvedExternal: []string{}, UnknownExternal: []string{}})
	}
	for _, run := range b.snap.Runs {
		if run.Origin == "root" && run.RootID == "" && programHasInvocations(b.programs[run.ProgramID]) {
			return corrupt("unscoped invocation source")
		}
	}
	if len(roots) != len(b.tables["composition_roots"]) || len(b.membership) != len(b.tables["composition_static_members"]) {
		return corrupt("resource root cardinality")
	}
	for _, row := range b.tables["composition_roots"] {
		r, e := inspectionDecode[compositionRoot](row, "descriptor")
		if e != nil || r.ID != row["id"] || !reflect.DeepEqual(roots[r.ID], r) {
			return corrupt("resource root descriptor")
		}
	}
	for _, row := range b.tables["composition_static_members"] {
		v, e := inspectionDecode[compositionStaticMember](row, "descriptor")
		if e != nil || v.Run != inspectionRun(row) || v.RootID != row["root_id"] || b.membership[v.Run] != v.RootID {
			return corrupt("static membership binding")
		}
	}
	expectedGrants := map[Permission]bool{}
	for _, grant := range perms {
		expectedGrants[grant] = true
	}
	bindings := map[string]compositionBinding{}
	for _, row := range b.tables["composition_bindings"] {
		v, e := inspectionDecode[compositionBinding](row, "descriptor")
		if e != nil || v.ID != row["id"] || v.InvocationID != row["invocation_id"] || v.Namespace != b.snap.Namespace {
			return corrupt("composition binding identity")
		}
		if _, e := retainCompositionBinding(b.snap.Namespace, v.RootID, templates[v.TemplateID], v); e != nil {
			return corrupt("invalid retained invocation binding: %v", e)
		}
		bindings[v.InvocationID] = v
	}
	for _, row := range b.tables["composition_members"] {
		v, e := inspectionDecode[compositionMemberOrigin](row, "descriptor")
		if e != nil {
			return e
		}
		run := b.runs[v.Run]
		bind := bindings[v.InvocationID]
		tmpl := templates[bind.TemplateID]
		member, ok := tmpl.Members[v.Member]
		if run == nil || run.Origin != "" || !ok || v.Run != inspectionRun(row) || v.Run != compositionMemberID(v.InvocationID, v.Member) || v.RootID != row["root_id"] || v.InvocationID != row["invocation_id"] || v.Member != row["member"] || bind.RootID != v.RootID || roots[v.RootID].ID == "" || bind.Members[v.Member] != v.Run || member.ProgramID != v.ProgramID || run.ProgramID != v.ProgramID || !reflect.DeepEqual(member.Initial, v.Initial) || !reflect.DeepEqual(run.Initial, v.Initial) || v.Owner != run.Subject.Owner || v.Owner != "invocation:"+v.InvocationID || run.Subject.Revision != 1 || run.Subject.Generation != 1 || run.Subject.RetryCancelled {
			return corrupt("invoked member origin %+v", v.Run)
		}
		p, e := resolveMemberProgram(b.programs[v.ProgramID], tmpl, bind, v.Member)
		if e != nil {
			return corrupt("invalid member program: %v", e)
		}
		for _, g := range memberPermissions(v.Run, bind.Caller, p) {
			expectedGrants[g] = true
		}
		if v.Member == tmpl.EntryMember {
			i := compositionInvocation{ID: v.InvocationID, Caller: bind.Caller, OperationID: bind.OperationID}
			for _, g := range handoffPermissions(tmpl, i, b.programs[v.ProgramID]) {
				expectedGrants[g] = true
			}
		}
		b.resolved[v.Run] = p
		b.membership[v.Run] = v.RootID
		run.Origin = "invocation"
		run.OwningInvocationID = v.InvocationID
		run.OriginOperationID = bind.OperationID
		run.RootID = v.RootID
	}
	if len(expectedGrants) != len(b.tables["permissions"]) {
		return corrupt("deployment grant cardinality")
	}
	for _, g := range b.snap.Permissions {
		if !expectedGrants[g] {
			return corrupt("permission has no original deployment grant")
		}
	}
	// A static handoff caller also has a dynamically resolved peer. Clone only
	// that caller's program before rewriting its ephemeral receive addresses.
	cloned := map[RunID]bool{}
	for _, t := range profile.Templates {
		if h := t.Handoff; h != nil {
			r := b.runs[h.Caller]
			if r == nil || r.Origin != "root" {
				return corrupt("missing handoff caller")
			}
			p := b.resolved[h.Caller]
			if !cloned[h.Caller] {
				raw, _ := json.Marshal(p)
				var own Program
				if err := decode(raw, &own); err != nil {
					return err
				}
				p = own
				cloned[h.Caller] = true
			}
			if p.Author == nil {
				return corrupt("missing authored handoff port")
			}
			if _, ok := p.Author.Ports[h.CallerPort]; !ok {
				return corrupt("missing declared handoff port")
			}
			bindHandoffPort(&p, b.snap.Namespace, t)
			b.resolved[h.Caller] = p
		}
	}
	for _, run := range b.snap.Runs {
		if run.Origin == "" {
			return corrupt("missing run origin %+v", run.ID)
		}
	}
	for i, row := range b.tables["composition_facts"] {
		f, e := inspectionDecode[compositionFact](row, "descriptor")
		if e != nil {
			return e
		}
		if f.Version != compositionFactVersion || f.Order != int64(i+1) || f.Order != row["decision_order"] || f.ID != row["id"] || string(f.Kind) != row["kind"] || f.ID == "" || f.RecordDigest == "" || len(f.RootIDs) == 0 {
			return corrupt("invalid ordered composition fact")
		}
		for _, id := range f.RootIDs {
			if roots[id].ID == "" {
				return corrupt("fact references missing root")
			}
		}
		if f.Kind != compositionAuthorityAdmissionFact && f.AuthorityAdmission != nil {
			return ErrVersion
		}
		if f.Kind == compositionAuthorityAdmissionFact && !b.authorityValidated {
			return ErrVersion
		}
		b.facts[f.ID] = f
	}
	return nil
}
