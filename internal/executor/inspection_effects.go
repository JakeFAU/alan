package executor

import (
	"encoding/json"
	"reflect"
	"slices"

	"github.com/JakeFAU/alan/adapters/receiptquery"
	"github.com/JakeFAU/alan/effects"
	resource "github.com/JakeFAU/alan/internal/fixture"
)

// These are the retained wire records of the built-in external counter protocol.
type ExternalRequest = resource.Request
type ExternalReceipt = resource.Receipt

// EffectDispatch bounds local transport calls before this cut. A spent opportunity
// can precede the call. A nil maximum denotes uncounted retries of an idempotent
// external request. These bounds do not measure remote outstanding work.
type EffectDispatch struct {
	Minimum int
	Maximum *int
}

type EffectSettlement struct {
	ID, ObservationID string
}

// EffectReceipt is an additional validated offer for the same original
// operation. Its protocol determines which member is present. Exact duplicate
// offers coalesce and no later offer becomes the original selectable result.
type EffectReceipt struct {
	Digest   string
	Adapter  *AdapterResult
	External *ExternalReceipt
}

// EffectRead is a spent receipt-query transport opportunity. Its Snapshot and
// Adoption are present only for an ordinal with a recorded response/decision.
// The original can remain unresolved even when a read found a receipt.
type EffectRead struct {
	Query                 receiptquery.Query
	OriginalRun           RunID
	OriginalAuthorization string
	Resolver              Authority
	MayAdopt              bool
	Snapshot              *receiptquery.Snapshot
	Adoption              string
}

// ExecutionEffect ties a committed operation to its historical authority,
// reservation, original result and any later evidence. The adapter and external
// records are mutually exclusive. ResultDisposition is present only when a
// result (including a refusal) is determined. Usage is the original adapter's
// reported usage. A nil value represents absent counters, not zero expenditure.
type ExecutionEffect struct {
	OperationID              string
	Source                   RunID
	Sequence, SourceRevision int64
	Kind, Capability, Name   string
	State                    EffectStage
	ResultDisposition        string
	// Consequence separates the first local outcome from evidence about application.
	// Values are no_dispatch, unresolved, receipt_retained, conflicting_receipts
	// and read_only (the controlled receipt lookup protocol). A retained receipt
	// is an assertion by the adapter or ledger, not independent remote finality.
	Consequence     ExternalConsequence
	Authorization   *Authorization
	AdapterAttempt  *AdapterAttempt
	AdapterResult   *AdapterResult
	ExternalAttempt *ExternalAttempt
	ExternalReceipt *ExternalReceipt
	Settlement      *EffectSettlement
	Dispatch        EffectDispatch
	// DispatchAdmission is the witnessed local gate in an opted-in scope.
	// A paid reservation alone does not establish admission or transmission.
	DispatchAdmission    *DispatchAdmission   `json:",omitempty"`
	AuthorityAdmissions  []AuthorityAdmission `json:",omitempty"`
	Usage                *effects.Usage
	ReceiptEvidenceState string
	AdditionalReceipts   []EffectReceipt
	ReadOpportunities    []EffectRead
	ReadsOfOriginal      []string
}

type effectCut struct {
	*inspectionBuilder
	grants       map[Permission]bool
	terminals    map[RunID]map[string]any
	reservations map[string]compositionReservation
	charges      map[string]compositionCharge
	ordinals     map[string][]receiptReadOrdinal
	readGrants   map[readGrantKey]ReceiptGrant
	resultOrder  map[string]int64
}

func (b *inspectionBuilder) buildEffects() error {
	c := &effectCut{inspectionBuilder: b, grants: map[Permission]bool{}, terminals: map[RunID]map[string]any{},
		reservations: map[string]compositionReservation{}, charges: map[string]compositionCharge{},
		ordinals: map[string][]receiptReadOrdinal{}, readGrants: map[readGrantKey]ReceiptGrant{}, resultOrder: map[string]int64{}}
	for _, i := range b.intents {
		if i.Kind == "adapter" || i.Kind == "external" {
			b.effects[i.ID] = &ExecutionEffect{OperationID: i.ID, Source: i.Run, Sequence: i.Sequence, SourceRevision: i.revision,
				Kind: i.Kind, Capability: i.Target, Name: i.Obligation, State: EffectCommitted, ReceiptEvidenceState: "none", Dispatch: EffectDispatch{Maximum: new(0)}}
		}
	}
	for _, p := range b.snap.Permissions {
		if p.Kind == "external" {
			p.Obligation = ""
		}
		c.grants[p] = true
	}
	for _, r := range b.tables["terminal_obligations"] {
		c.terminals[inspectionRun(r)] = r
	}
	for id, x := range b.effects {
		if a, ok := b.authorizations[id]; ok {
			x.Authorization = &a
			x.State = EffectAuthorizationRefused
			if a.Allowed {
				x.State = EffectAuthorized
			}
		}
	}
	if err := c.readReservations(); err != nil {
		return err
	}
	if err := c.readResults(); err != nil {
		return err
	}
	if err := c.readSettlements(); err != nil {
		return err
	}
	if err := c.readQueries(); err != nil {
		return err
	}
	if err := c.checkEffectOrder(); err != nil {
		return err
	}
	if err := c.readAdditionalReceipts(); err != nil {
		return err
	}
	for _, x := range c.effects {
		switch {
		case x.Capability == receiptReadCapability && protocolSchemaNumber(b.snap.StorageVersion) == reconciliationSchemaVersion:
			x.Consequence = ConsequenceReadOnly
		case x.Dispatch.Maximum != nil && *x.Dispatch.Maximum == 0:
			x.Consequence = ConsequenceNoDispatch
		case x.ReceiptEvidenceState == "known":
			x.Consequence = ConsequenceReceiptRetained
		case x.ReceiptEvidenceState == "conflict":
			x.Consequence = ConsequenceConflictingReceipts
		default:
			x.Consequence = ConsequenceUnresolved
		}
	}
	return nil
}

func (c *effectCut) checkFact(kind compositionFactKind, x *ExecutionEffect, raw string) (int64, error) {
	root := c.membership[x.Source]
	id := compositionIdentity("alan/fact/"+string(kind)+"/v1", x.OperationID)
	f, exists := c.facts[id]
	if root == "" {
		if exists {
			return 0, corrupt("unscoped effect fact")
		}
		return 0, nil
	}
	if !exists || f.Kind != kind || f.OperationID != x.OperationID || f.Run != x.Source || f.Sequence != x.Sequence || !slices.Equal(f.RootIDs, []string{root}) ||
		f.RecordDigest != compositionIdentity("alan/"+string(kind)+"-row/v1", json.RawMessage(raw)) {
		return 0, corrupt("missing or misbound %s effect fact", kind)
	}
	return f.Order, nil
}

func (c *effectCut) readReservations() error {
	for _, r := range c.tables["composition_reservations"] {
		v, err := inspectionDecode[compositionReservation](r, "descriptor")
		if err != nil {
			return err
		}
		x := c.effects[v.OperationID]
		if x == nil || r["operation_id"] != v.OperationID || r["root_id"] != v.RootID || v.RootID == "" || v.RootID != c.membership[x.Source] ||
			v.Run != x.Source || v.Kind != x.Kind || v.Sequence != x.Sequence {
			return corrupt("effect resource reservation binding")
		}
		c.reservations[v.OperationID] = v
		if _, err = c.checkFact(compositionReservationFact, x, inspectText(r, "descriptor")); err != nil {
			return err
		}
	}
	for _, r := range c.tables["composition_charges"] {
		if r["kind"] != "effect" {
			continue
		}
		v, err := inspectionDecode[compositionCharge](r, "descriptor")
		if err != nil {
			return err
		}
		res, ok := c.reservations[v.OperationID]
		f := c.facts[compositionIdentity("alan/fact/reservation/v1", v.OperationID)]
		want := compositionCharge{RootID: res.RootID, OperationID: v.OperationID, Kind: "effect", Effects: 1, DecisionOrder: f.Order}
		if !ok || !res.Allowed || v != want || r["root_id"] != v.RootID || r["operation_id"] != v.OperationID || r["decision_order"] != f.Order {
			return corrupt("effect expenditure without the original reservation")
		}
		c.charges[v.OperationID] = v
	}
	for _, r := range c.tables["adapter_attempts"] {
		a, err := inspectionDecode[AdapterAttempt](r, "descriptor")
		if err != nil {
			return err
		}
		x := c.effects[a.Intent.ID]
		if x == nil || x.Kind != "adapter" || r["intent_id"] != a.Intent.ID || a.Intent.Generation != 1 || a.Prepared.Digest == "" || effects.CheckPrepared(a.Prepared) != nil ||
			a.Prepared.MaxResponseBytes > c.limits.MaxObservationBytes/8 {
			return corrupt("invalid adapter preparation")
		}
		program := c.programs[c.runs[x.Source].ProgramID]
		if program.Author == nil {
			return corrupt("adapter lacks program declaration")
		}
		decl, ok := program.Author.Effects[x.Name]
		w := effects.Request{ID: x.OperationID, Key: ExternalKey(c.snap.Namespace, x.OperationID), Capability: x.Capability, Effect: x.Name,
			Input: decl.Input, Output: decl.Output, Payload: a.Intent.Payload, Assets: a.Prepared.Request.Assets}
		if !ok || decl.Capability != x.Capability || decl.Input.Record.Validate(a.Intent.Payload) != nil ||
			!reflect.DeepEqual(w, a.Prepared.Request) || !slices.Equal(w.Assets, c.snap.Assets) {
			return corrupt("adapter request differs from original program, inputs or assets")
		}
		x.AdapterAttempt = &a
		if err := c.reservation(x, a.Intent, a.AuthorizationID, a.Allowed, a.Reason, a); err != nil {
			return err
		}
	}
	for _, r := range c.tables["external_attempts"] {
		a, err := inspectionDecode[ExternalAttempt](r, "descriptor")
		if err != nil {
			return err
		}
		x := c.effects[a.Intent.ID]
		if x == nil || x.Kind != "external" || r["intent_id"] != a.Intent.ID || x.Authorization == nil {
			return corrupt("external attempt without original effect")
		}
		want, err := externalRequest(a.Intent)
		if err != nil {
			return corrupt("invalid external request")
		}
		want.Epoch = x.Authorization.Source.Epoch
		want.Authorization = x.Authorization.ID
		want.Key = ExternalKey(c.snap.Namespace, x.OperationID)
		if want != a.Request {
			return corrupt("external wire request differs from original effect")
		}
		x.ExternalAttempt = &a
		if err := c.reservation(x, a.Intent, a.AuthorizationID, a.Allowed, a.Reason, a); err != nil {
			return err
		}
	}
	for id := range c.reservations {
		x := c.effects[id]
		if x.AdapterAttempt == nil && x.ExternalAttempt == nil {
			return corrupt("resource reservation without attempt")
		}
	}
	return nil
}

func (c *effectCut) reservation(x *ExecutionEffect, i Intent, authID string, allowed bool, reason string, record any) error {
	a := x.Authorization
	if a == nil || a.ID != authID || !reflect.DeepEqual(c.intents[x.OperationID].Intent, i) {
		return corrupt("effect reservation for a different original authorization")
	}
	grant := Permission{Kind: x.Kind, Source: x.Source.Subject, Target: x.Capability, Operation: x.Name, Revision: x.SourceRevision, Obligation: x.Name}
	if x.Kind == "external" {
		grant.Operation = "apply"
		grant.Obligation = ""
	}
	base := a.Allowed && c.grants[grant]
	baseReason := a.Reason
	if a.Allowed && !base {
		baseReason = "grant_denied"
	}
	var w *EffectAuthority
	if x.AdapterAttempt != nil {
		w = x.AdapterAttempt.Authority
	}
	if x.ExternalAttempt != nil {
		w = x.ExternalAttempt.Authority
	}
	if c.snap.Features.RequestAuthority {
		if !c.authorityValidated {
			return ErrCorrupt
		}
		if w == nil {
			return ErrCorrupt
		}
		base = a.Allowed && w.Allowed
		baseReason = a.Reason
		if a.Allowed && !w.Allowed {
			baseReason = w.Reason
		}
	} else if i.Authority != nil || w != nil {
		return ErrVersion
	}
	res, scoped := c.reservations[x.OperationID]
	if scoped != (c.membership[x.Source] != "") {
		return corrupt("missing or unexpected scoped reservation")
	}
	if scoped {
		if res.Allowed != allowed || res.Reason != reason || res.AttemptDigest != compositionIdentity("alan/attempt-row/v1", record) {
			return corrupt("resource reservation differs from original attempt")
		}
		if _, charged := c.charges[x.OperationID]; charged != allowed {
			return corrupt("effect debit disagrees with reservation")
		}
	}
	if !(base && scoped && !allowed && (reason == "scope_closed" || reason == "effect_limit")) && (base != allowed || reason != baseReason) {
		return corrupt("effect reservation differs from authority or grant")
	}
	if allowed {
		x.State = EffectReservedUnknown
		x.Dispatch.Maximum = new(1)
		if c.snap.Features.DispatchAdmissions && scoped && x.Kind == "adapter" {
			x.Dispatch.Maximum = new(0)
		}
		if x.Kind == "external" && x.ExternalAttempt.Request.Profile == "idempotent" {
			x.Dispatch.Maximum = nil
		}
	} else {
		x.State = EffectReservationRefused
	}
	return nil
}
