package executor

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"slices"
)

// Schema3 is not altered. These tables exist only in internally created schema4
// stores. Protocol tables remain empty while public activation is gated.
const compositionSchema = `
CREATE TABLE composition_profile (singleton INTEGER PRIMARY KEY CHECK(singleton=1), version TEXT NOT NULL, descriptor BLOB NOT NULL);
CREATE TABLE composition_roots (id TEXT PRIMARY KEY, descriptor BLOB NOT NULL);
CREATE TABLE composition_static_members (subject TEXT NOT NULL, generation INTEGER NOT NULL CHECK(generation=1), root_id TEXT NOT NULL REFERENCES composition_roots(id), descriptor BLOB NOT NULL, PRIMARY KEY(subject,generation), FOREIGN KEY(subject,generation) REFERENCES runs(subject,generation));
CREATE TABLE composition_templates (id TEXT PRIMARY KEY, descriptor BLOB NOT NULL);
CREATE TABLE composition_facts (decision_order INTEGER PRIMARY KEY CHECK(decision_order>0), id TEXT NOT NULL UNIQUE, kind TEXT NOT NULL CHECK(kind IN ('execution','lifecycle','admission','reservation','reply','freeze','outcome')), descriptor BLOB NOT NULL);
CREATE TABLE composition_invocations (id TEXT PRIMARY KEY, operation_id TEXT NOT NULL UNIQUE REFERENCES intents(id), template_id TEXT NOT NULL REFERENCES composition_templates(id), root_id TEXT NOT NULL REFERENCES composition_roots(id), descriptor BLOB NOT NULL, admission BLOB NOT NULL);
CREATE TABLE composition_bindings (id TEXT PRIMARY KEY, invocation_id TEXT NOT NULL UNIQUE REFERENCES composition_invocations(id), descriptor BLOB NOT NULL);
CREATE TABLE composition_members (subject TEXT NOT NULL, generation INTEGER NOT NULL CHECK(generation=1), invocation_id TEXT NOT NULL REFERENCES composition_invocations(id), root_id TEXT NOT NULL REFERENCES composition_roots(id), member TEXT NOT NULL, descriptor BLOB NOT NULL, PRIMARY KEY(subject,generation), UNIQUE(invocation_id,member), FOREIGN KEY(subject,generation) REFERENCES runs(subject,generation));
CREATE TABLE composition_results (invocation_id TEXT PRIMARY KEY REFERENCES composition_invocations(id), decision_order INTEGER NOT NULL UNIQUE REFERENCES composition_facts(decision_order), descriptor BLOB NOT NULL);
CREATE TABLE composition_charges (operation_id TEXT NOT NULL REFERENCES intents(id), kind TEXT NOT NULL CHECK(kind IN ('invocation','effect')), root_id TEXT NOT NULL REFERENCES composition_roots(id), decision_order INTEGER NOT NULL REFERENCES composition_facts(decision_order), descriptor BLOB NOT NULL, PRIMARY KEY(operation_id,kind));
`

func createCompositionStore(ctx context.Context, path string, p Provision, c compositionProvision, l Limits) (*Store, error) {
	return createStore(ctx, path, p, l, &c)
}
func openCompositionStore(ctx context.Context, path string, l Limits) (*Store, error) {
	return openStore(ctx, path, l, true)
}
func schemaNumber(ctx context.Context, q queryer) (int, error) {
	var n int
	err := q.QueryRowContext(ctx, "SELECT schema_version FROM metadata WHERE singleton=1").Scan(&n)
	return n, err
}
func requireSchema(ctx context.Context, q queryer, want int) error {
	n, err := schemaNumber(ctx, q)
	if err != nil {
		return err
	}
	if n != want {
		return ErrVersion
	}
	return nil
}
func retainCompositionProfile(ctx context.Context, q queryer, namespace string, c compositionProvision, l Limits) (*compositionProfile, error) {
	if err := validateCompositionEncoding(c); err != nil {
		return nil, err
	}
	if c.Version != "" && c.Version != compositionProtocolVersion {
		return nil, ErrVersion
	}
	// JSON is both the retained boundary and the defensive copy; no caller-owned
	// map, list or pointer is kept on Store.
	raw, err := json.Marshal(c)
	if err != nil {
		return nil, err
	}
	if len(raw) > l.MaxProgramBytes {
		return nil, ErrLimit
	}
	var out compositionProvision
	if err = decode(raw, &out); err != nil {
		return nil, err
	}
	if len(out.Roots) == 0 {
		return nil, fmt.Errorf("composition requires a resource root")
	}
	templates := map[string]compositionTemplate{}
	for i, t := range out.Templates {
		t, err = retainCompositionTemplate(ctx, q, t, l)
		if err != nil {
			return nil, err
		}
		if _, exists := templates[t.ID]; exists {
			return nil, fmt.Errorf("duplicate template")
		}
		templates[t.ID] = t
		out.Templates[i] = t
	}
	for _, t := range out.Templates {
		for _, id := range t.Templates {
			if _, ok := templates[id]; !ok {
				return nil, fmt.Errorf("undeployed template dependency")
			}
		}
	}
	members := map[RunID]bool{}
	roots := map[RunID]bool{}
	for i, r := range out.Roots {
		if r.Version != "" && r.Version != compositionResourceVersion {
			return nil, ErrVersion
		}
		r.Version = compositionResourceVersion
		id := compositionRootID(namespace, r.Run)
		if r.ID != "" && r.ID != id {
			return nil, fmt.Errorf("root identity mismatch")
		}
		r.ID = id
		if r.Run.Subject == "" || r.Run.Generation != 1 || roots[r.Run] || len(r.Members) == 0 {
			return nil, fmt.Errorf("invalid resource root")
		}
		roots[r.Run] = true
		if r.Limits.MaxInvocations < 0 || r.Limits.MaxCreatedRuns < 0 || r.Limits.MaxDepth < 0 || r.Limits.MaxEffects < 0 || r.Limits.MaxInFlight <= 0 {
			return nil, fmt.Errorf("invalid root limits")
		}
		r.Templates, err = canonicalStrings(r.Templates)
		if err != nil {
			return nil, err
		}
		r.Capabilities, err = canonicalStrings(r.Capabilities)
		if err != nil {
			return nil, err
		}
		for _, id := range r.Templates {
			if _, ok := templates[id]; !ok {
				return nil, fmt.Errorf("root permits undeployed template")
			}
		}
		slices.SortFunc(r.Members, func(a, b RunID) int {
			if a.Subject < b.Subject {
				return -1
			}
			if a.Subject > b.Subject {
				return 1
			}
			return 0
		})
		hasRoot := false
		for _, run := range r.Members {
			if run.Subject == "" || run.Generation != 1 || members[run] {
				return nil, fmt.Errorf("duplicate or invalid static member")
			}
			members[run] = true
			hasRoot = hasRoot || run == r.Run
			state, err := loadRun(ctx, q, run)
			if err != nil {
				return nil, err
			}
			subject, err := loadSubject(ctx, q, run.Subject)
			if err != nil {
				return nil, err
			}
			if subject.Generation != 1 {
				return nil, corrupt("scoped member changed generation")
			}
			var kind string
			if err = q.QueryRowContext(ctx, "SELECT kind FROM run_origins WHERE subject=? AND generation=?", run.Subject, run.Generation).Scan(&kind); err != nil {
				return nil, err
			}
			if kind != "root" {
				return nil, fmt.Errorf("static member is not a provisioned origin")
			}
			p, err := loadProgram(ctx, q, state.ProgramID, l)
			if err != nil {
				return nil, err
			}
			if p.Author != nil {
				for _, e := range p.Author.Effects {
					if !slices.Contains(r.Capabilities, e.Capability) {
						return nil, fmt.Errorf("static member capability not permitted")
					}
				}
			}
		}
		if !hasRoot {
			return nil, fmt.Errorf("root omitted from static membership")
		}
		out.Roots[i] = r
	}
	if out.Version == compositionProtocolVersion {
		if err = validateInvocationCatalogue(ctx, q, out, l); err != nil {
			return nil, err
		}
	}
	slices.SortFunc(out.Roots, func(a, b compositionRoot) int {
		if a.ID < b.ID {
			return -1
		}
		if a.ID > b.ID {
			return 1
		}
		return 0
	})
	slices.SortFunc(out.Templates, func(a, b compositionTemplate) int {
		if a.ID < b.ID {
			return -1
		}
		if a.ID > b.ID {
			return 1
		}
		return 0
	})
	raw, err = json.Marshal(out)
	if err != nil {
		return nil, err
	}
	if len(raw) > l.MaxProgramBytes {
		return nil, ErrLimit
	}
	return &compositionProfile{raw: string(raw)}, nil
}
func provisionComposition(ctx context.Context, tx *sql.Tx, namespace string, c compositionProvision, l Limits) (*compositionProfile, error) {
	p, err := retainCompositionProfile(ctx, tx, namespace, c, l)
	if err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO composition_profile VALUES(1,?,?)", compositionVersion(c), []byte(p.raw)); err != nil {
		return nil, err
	}
	d, err := p.descriptor()
	if err != nil {
		return nil, err
	}
	for _, r := range d.Roots {
		raw, _ := json.Marshal(r)
		if _, err = tx.ExecContext(ctx, "INSERT INTO composition_roots VALUES(?,?)", r.ID, raw); err != nil {
			return nil, err
		}
		for _, m := range r.Members {
			raw, _ := json.Marshal(compositionStaticMember{r.ID, m})
			if _, err = tx.ExecContext(ctx, "INSERT INTO composition_static_members VALUES(?,?,?,?)", m.Subject, m.Generation, r.ID, raw); err != nil {
				return nil, err
			}
		}
	}
	for _, t := range d.Templates {
		raw, _ := json.Marshal(t)
		if _, err = tx.ExecContext(ctx, "INSERT INTO composition_templates VALUES(?,?)", t.ID, raw); err != nil {
			return nil, err
		}
	}
	return p, nil
}
func readCompositionProfile(ctx context.Context, q queryer, l Limits) (*compositionProfile, error) {
	if err := requireSchema(ctx, q, compositionSchemaVersion); err != nil {
		return nil, err
	}
	namespace, err := readNamespace(ctx, q)
	if err != nil {
		return nil, err
	}
	var version string
	var raw []byte
	if err = q.QueryRowContext(ctx, "SELECT version,descriptor FROM composition_profile WHERE singleton=1").Scan(&version, &raw); err != nil {
		return nil, err
	}
	if version != compositionProfileVersion && version != compositionProtocolVersion {
		return nil, ErrVersion
	}
	var c compositionProvision
	if err = decode(raw, &c); err != nil {
		return nil, err
	}
	if version != compositionVersion(c) {
		return nil, ErrVersion
	}
	retained, err := retainCompositionProfile(ctx, q, namespace, c, l)
	if err != nil {
		return nil, err
	}
	if string(raw) != retained.raw {
		return nil, corrupt("noncanonical composition profile")
	}
	return retained, nil
}
func validateCompositionStore(ctx context.Context, q queryer, l Limits) (ReplayReport, error) {
	p, err := readCompositionProfile(ctx, q, l)
	if err != nil {
		return ReplayReport{}, err
	}
	c0, err := p.descriptor()
	if err != nil {
		return ReplayReport{}, err
	}
	if c0.Version == compositionProtocolVersion {
		var n int
		if err = q.QueryRowContext(ctx, "SELECT (SELECT count(*) FROM composition_admission_decisions) + (SELECT count(*) FROM composition_facts WHERE kind NOT IN ('execution','lifecycle')) + (SELECT count(*) FROM intents WHERE json_extract(descriptor,'$.Kind')='invocation')").Scan(&n); err != nil {
			return ReplayReport{}, err
		}
		if n != 0 {
			return ReplayReport{}, fmt.Errorf("composition protocol replay pending Task2c: %w", ErrVersion)
		}
	}
	report, err := validateOrdinaryLedger(ctx, q, l)
	if err != nil {
		return report, err
	}
	c, err := p.descriptor()
	if err != nil {
		return report, err
	}
	expectedRoots := map[string]string{}
	expectedMembers := map[RunID]string{}
	expectedTemplates := map[string]string{}
	for _, r := range c.Roots {
		raw, _ := json.Marshal(r)
		expectedRoots[r.ID] = string(raw)
		for _, m := range r.Members {
			raw, _ := json.Marshal(compositionStaticMember{r.ID, m})
			expectedMembers[m] = string(raw)
		}
	}
	for _, t := range c.Templates {
		raw, _ := json.Marshal(t)
		expectedTemplates[t.ID] = string(raw)
	}
	for _, table := range []string{"composition_roots", "composition_templates"} {
		expected := expectedRoots
		if table == "composition_templates" {
			expected = expectedTemplates
		}
		rows, e := q.QueryContext(ctx, "SELECT id,descriptor FROM "+table)
		if e != nil {
			return report, e
		}
		seen := map[string]bool{}
		for rows.Next() {
			var id string
			var raw []byte
			if e = rows.Scan(&id, &raw); e != nil {
				rows.Close()
				return report, e
			}
			if expected[id] != string(raw) || seen[id] {
				rows.Close()
				return report, corrupt("unexpected %s descriptor", table)
			}
			seen[id] = true
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return report, e
		}
		if len(seen) != len(expected) {
			return report, corrupt("missing %s descriptor", table)
		}
	}
	rows, err := q.QueryContext(ctx, "SELECT subject,generation,root_id,descriptor FROM composition_static_members")
	if err != nil {
		return report, err
	}
	seen := map[RunID]bool{}
	for rows.Next() {
		var m compositionStaticMember
		var raw []byte
		if err = rows.Scan(&m.Run.Subject, &m.Run.Generation, &m.RootID, &raw); err != nil {
			rows.Close()
			return report, err
		}
		if expectedMembers[m.Run] != string(raw) || !sameJSON(raw, m) || seen[m.Run] {
			rows.Close()
			return report, corrupt("static membership mismatch")
		}
		seen[m.Run] = true
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return report, err
	}
	if len(seen) != len(expectedMembers) {
		return report, corrupt("missing static membership")
	}
	for _, table := range []string{"composition_invocations", "composition_bindings", "composition_members", "composition_results", "composition_charges"} {
		var n int
		if err = q.QueryRowContext(ctx, "SELECT count(*) FROM "+table).Scan(&n); err != nil {
			return report, err
		}
		if n != 0 {
			return report, fmt.Errorf("unfinished composition protocol in %s: %w", table, ErrVersion)
		}
	}
	return report, validateCompositionFacts(ctx, q)
}

func compositionVersion(c compositionProvision) string {
	if c.Version != "" {
		return c.Version
	}
	return compositionProfileVersion
}
func createInvocationStore(ctx context.Context, path string, p Provision, c compositionProvision, l Limits) (*Store, error) {
	c.Version = compositionProtocolVersion
	return createStore(ctx, path, p, l, &c)
}
func (s *Store) invocationEnabled() bool {
	if s.composition == nil {
		return false
	}
	c, err := s.composition.descriptor()
	return err == nil && c.Version == compositionProtocolVersion
}
