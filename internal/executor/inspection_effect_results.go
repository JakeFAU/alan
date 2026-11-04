package executor

import (
	"encoding/json"
	"reflect"
	"slices"
	"strings"
)

func (c *effectCut) readResults() error {
	for _, row := range c.tables["adapter_results"] {
		id, _ := row["intent_id"].(string)
		x := c.effects[id]
		r, err := inspectionDecode[AdapterResult](row, "descriptor")
		if err != nil {
			return err
		}
		if x == nil || x.AdapterAttempt == nil {
			return corrupt("adapter result without original reservation")
		}
		if err := validateAdapterResult(*x.AdapterAttempt, r, c.limits); err != nil {
			return err
		}
		if (r.Read != nil || r.Reconciled != nil) && protocolSchemaNumber(c.snap.StorageVersion) != reconciliationSchemaVersion {
			return corrupt("reconciliation evidence in incompatible format")
		}
		x.AdapterResult = &r
		x.State = EffectResultUnsettled
		x.ResultDisposition = r.Disposition
		x.Usage = r.Candidate.Usage
		if x.AdapterAttempt.Allowed && r.Reconciled == nil && !(x.Capability == receiptReadCapability && protocolSchemaNumber(c.snap.StorageVersion) == reconciliationSchemaVersion) {
			x.Dispatch.Minimum = 1
		}
		if r.Disposition == "validated" {
			x.ReceiptEvidenceState = "known"
		}
		order, err := c.checkFact(compositionExternalResultFact, x, inspectText(row, "descriptor"))
		if err != nil {
			return err
		}
		c.resultOrder[id] = order
	}
	for _, row := range c.tables["external_receipts"] {
		id, _ := row["intent_id"].(string)
		x := c.effects[id]
		r, err := inspectionDecode[ExternalReceipt](row, "descriptor")
		if err != nil {
			return err
		}
		if x == nil || x.ExternalAttempt == nil || !x.ExternalAttempt.Allowed || !validReceipt(*x.ExternalAttempt, r) {
			return corrupt("external receipt without its original allowed reservation")
		}
		x.ExternalReceipt = &r
		x.ResultDisposition = "applied"
		x.ReceiptEvidenceState = "known"
		x.State = EffectResultUnsettled
		x.Dispatch.Minimum = 1
		order, err := c.checkFact(compositionExternalResultFact, x, inspectText(row, "descriptor"))
		if err != nil {
			return err
		}
		c.resultOrder[id] = order
	}
	for _, x := range c.effects {
		if x.ExternalAttempt != nil && !x.ExternalAttempt.Allowed {
			x.ResultDisposition = "source_denied"
			if x.ExternalAttempt.Reason == "grant_denied" || x.ExternalAttempt.Authority != nil && !x.ExternalAttempt.Authority.Allowed {
				x.ResultDisposition = "grant_denied"
			}
		}
	}
	return nil
}

func (c *effectCut) readSettlements() error {
	admissions := map[string]map[string]any{}
	for _, row := range c.tables["admissions"] {
		if row["domain"] == "adapter" || row["domain"] == "receipt" {
			id, ok := row["observation_id"].(string)
			if !ok || id == "" || admissions[id] != nil {
				return corrupt("effect delivery without unique observation")
			}
			admissions[id] = row
		}
	}
	for _, row := range c.tables["settlements"] {
		id, _ := row["intent_id"].(string)
		x := c.effects[id]
		if x == nil {
			if _, ok := c.intents[id]; !ok {
				return corrupt("orphan settlement")
			}
			continue
		}
		if x.Settlement != nil {
			return corrupt("multiple settlements of original effect")
		}
		var d AdmissionDescriptor
		var key string
		switch x.Kind {
		case "adapter":
			if x.AdapterResult == nil || row["descriptor"] == nil || !sameJSON([]byte(inspectText(row, "descriptor")), x.AdapterResult) {
				return corrupt("settled adapter result differs from original")
			}
			d = adapterDescriptor(*x.AdapterAttempt, *x.AdapterResult)
			key = identity("adapter-outcome", id)
		case "external":
			if x.ExternalAttempt == nil || x.ResultDisposition == "" {
				return corrupt("external settlement without outcome")
			}
			o := ExternalOutcome{Attempt: *x.ExternalAttempt, Disposition: x.ResultDisposition, Receipt: x.ExternalReceipt}
			if row["descriptor"] == nil || !sameJSON([]byte(inspectText(row, "descriptor")), o) {
				return corrupt("settled external outcome differs from original")
			}
			d = externalDescriptor(o)
			key = identity("external-outcome", id)
		}
		o := observationFor(d, x.Source)
		ad := admissions[o.ID]
		want := AdmissionResult{TransportID: d.TransportID, Accepted: true, Reason: "settled", Bound: &x.Source, ObservationID: o.ID}
		expectedObs := c.observations[o.ID]
		if key != row["id"] || o.ID != row["observation_id"] || ad == nil || ad["domain"] != d.Kind || ad["id"] != admissionKey(d.Kind, d.TransportID) || ad["transport_id"] != d.TransportID ||
			!sameJSON([]byte(inspectText(ad, "descriptor")), d) || !sameJSON([]byte(inspectText(ad, "result")), want) || expectedObs == nil ||
			!reflect.DeepEqual(expectedObs.Observation, o) {
			return corrupt("effect settlement, delivery or observation tied to wrong original")
		}
		if at := expectedObs.SelectedAt; at != nil && c.membership[x.Source] != "" {
			result := c.resultOrder[id]
			if x.ExternalAttempt != nil && !x.ExternalAttempt.Allowed {
				result = c.facts[compositionIdentity("alan/fact/reservation/v1", id)].Order
			}
			selected := c.facts[compositionIdentity("alan/fact/execution/v1", []any{at.Run, at.Sequence})].Order
			if result == 0 || result >= selected {
				return corrupt("effect selected before the original outcome")
			}
		}
		x.Settlement = &EffectSettlement{ID: key, ObservationID: o.ID}
		x.State = EffectSettled
		delete(admissions, o.ID)
	}
	if len(admissions) != 0 {
		return corrupt("effect observation admitted without its original settlement")
	}
	return nil
}

func (c *effectCut) readAdditionalReceipts() error {
	rows := c.tables["receipt_conflicts"]
	if hasReceiptConflictSchema(c.snap.StorageVersion) != (len(rows) > 0) {
		return corrupt("receipt evidence inconsistent with storage format")
	}
	seen := map[[2]string]bool{}
	firstDigests := map[string]string{}
	for _, row := range rows {
		id, _ := row["intent_id"].(string)
		x := c.effects[id]
		winner, _ := row["winner_digest"].(string)
		offer, _ := row["offer_digest"].(string)
		raw, ok := row["descriptor"].(string)
		if x == nil || !ok || len(raw) > c.limits.MaxObservationBytes || winner == offer || programHash([]byte(raw)) != offer || seen[[2]string{id, offer}] {
			return corrupt("unbound additional receipt")
		}
		seen[[2]string{id, offer}] = true
		item := EffectReceipt{Digest: offer}
		var first any
		switch x.Kind {
		case "adapter":
			a, r := x.AdapterAttempt, x.AdapterResult
			if a == nil || !a.Allowed || r == nil || r.Reconciled != nil || r.Read != nil || r.RefusalWitness != "" || x.Capability == receiptReadCapability {
				return corrupt("additional receipt cannot anchor to this adapter result")
			}
			first = r
			candidate, err := inspectionDecode[AdapterResult](row, "descriptor")
			if err != nil {
				return err
			}
			if candidate.Disposition != "validated" || candidate.Reconciled != nil || candidate.Read != nil || candidate.RefusalWitness != "" || validateAdapterResult(*a, candidate, c.limits) != nil {
				return corrupt("invalid additional adapter receipt")
			}
			item.Adapter = &candidate
		case "external":
			if x.ExternalReceipt == nil || x.ExternalAttempt == nil || !x.ExternalAttempt.Allowed {
				return corrupt("additional receipt without original external outcome")
			}
			first = x.ExternalReceipt
			candidate, err := inspectionDecode[ExternalReceipt](row, "descriptor")
			if err != nil {
				return err
			}
			if !validReceipt(*x.ExternalAttempt, candidate) {
				return corrupt("invalid additional external receipt")
			}
			item.External = &candidate
		}
		// The winning digest names the canonical exact first result, including
		// an unsuccessful adapter outcome that supplied no successful receipt.
		if _, ok := firstDigests[id]; !ok {
			raw, err := json.Marshal(first)
			if err != nil {
				return corrupt("invalid first receipt")
			}
			firstDigests[id] = programHash(raw)
		}
		if firstDigests[id] != winner {
			return corrupt("additional receipt does not match first result")
		}
		x.AdditionalReceipts = append(x.AdditionalReceipts, item)
		if len(x.AdditionalReceipts) > 1 || x.ExternalReceipt != nil || x.AdapterResult != nil && x.AdapterResult.Disposition == "validated" {
			x.ReceiptEvidenceState = "conflict"
		} else if x.ReceiptEvidenceState == "none" {
			x.ReceiptEvidenceState = "known"
		}
	}
	for _, x := range c.effects {
		slices.SortFunc(x.AdditionalReceipts, func(a, b EffectReceipt) int { return strings.Compare(a.Digest, b.Digest) })
	}
	return nil
}
