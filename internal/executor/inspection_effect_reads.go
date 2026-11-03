package executor

import (
	"encoding/json"
	"reflect"
	"slices"

	"github.com/JakeFAU/alan/adapters/receiptquery"
	"github.com/JakeFAU/alan/effects"
)

type readGrantKey struct {
	resolver RunID
	target   string
}

func (c *effectCut) readQueries() error {
	if protocolSchemaNumber(c.snap.StorageVersion) != reconciliationSchemaVersion {
		return nil
	}
	rows := c.tables["reconciliation_profile"]
	if len(rows) != 1 || rows[0]["singleton"] != int64(1) {
		return corrupt("missing original receipt grants")
	}
	p, err := inspectionDecode[ReconciliationProfile](rows[0], "descriptor")
	if err != nil || p.check() != nil {
		return corrupt("invalid original receipt grants")
	}
	c.snap.Reconciliation = &p
	for _, g := range p.Grants {
		_, id := g.target(c.snap.Namespace)
		c.readGrants[readGrantKey{g.Resolver, id}] = g
	}
	for _, row := range c.tables["receipt_read_ordinals"] {
		o, err := inspectionDecode[receiptReadOrdinal](row, "descriptor")
		if err != nil {
			return err
		}
		q := o.Query
		x := c.effects[q.QueryID]
		if x == nil || x.Capability != receiptReadCapability || x.AdapterAttempt == nil || !x.AdapterAttempt.Allowed ||
			q.QueryID != row["intent_id"] || int64(q.Ordinal) != row["ordinal"] || q.Check() != nil {
			return corrupt("misbound receipt-read opportunity")
		}
		c.ordinals[q.QueryID] = append(c.ordinals[q.QueryID], o)
	}
	adoptions := map[string]bool{}
	positive := map[string]receiptquery.Receipt{}
	for _, id := range sortedKeys(c.effects) {
		x := c.effects[id]
		if x.Capability != receiptReadCapability || x.AdapterAttempt == nil {
			if x.AdapterResult != nil && x.AdapterResult.Read != nil {
				return corrupt("receipt-read result on another capability")
			}
			if x.Capability == receiptCounterCapability && x.AdapterAttempt != nil && receiptquery.CounterPrepared(x.AdapterAttempt.Prepared) != nil {
				return corrupt("invalid retained counter preparation")
			}
			continue
		}
		a := x.AdapterAttempt
		if !validReadPreparation(a.Prepared) {
			return corrupt("invalid receipt query preparation")
		}
		ord := c.ordinals[id]
		if len(ord) > 2 {
			return corrupt("too many receipt-read opportunities")
		}
		slices.SortFunc(ord, func(a, b receiptReadOrdinal) int { return a.Query.Ordinal - b.Query.Ordinal })
		x.Dispatch = EffectDispatch{Maximum: new(len(ord))}
		for j, o := range ord {
			if o.Query.Ordinal != j+1 || j > 0 && (ord[j-1].Resolver.Epoch >= o.Resolver.Epoch || ord[j-1].Prefix >= o.Prefix) {
				return corrupt("receipt read repeated without new authority")
			}
			g, ok := c.readGrants[readGrantKey{x.Source, o.Query.TargetID}]
			if !ok {
				return corrupt("receipt read without original grant")
			}
			if err := c.checkOrdinal(x, o, g); err != nil {
				return err
			}
			original := c.effects[o.Query.TargetID]
			if j == 0 {
				original.ReadsOfOriginal = append(original.ReadsOfOriginal, id)
			}
			x.ReadOpportunities = append(x.ReadOpportunities, EffectRead{Query: o.Query, OriginalRun: o.OriginalRun,
				OriginalAuthorization: o.OriginalAuthorization, Resolver: o.Resolver, MayAdopt: o.Adopt})
		}
		r := x.AdapterResult
		if r == nil {
			continue
		}
		if r.Reconciled != nil {
			return corrupt("receipt query cannot itself be reconstructed")
		}
		if r.Read == nil {
			if !reflect.DeepEqual(r.Candidate, effects.Candidate{ID: id, Key: a.Prepared.Request.Key, Disposition: r.Disposition}) {
				return corrupt("local receipt-read refusal with transport output")
			}
			if a.Allowed {
				if c.membership[x.Source] == "" && r.Disposition != "unscoped" {
					return corrupt("unscoped receipt read with unexplained refusal")
				}
				if (len(ord) == 2) != (r.Disposition == "read_exhausted") || len(ord) == 1 && r.Disposition != "original_unsettled" {
					return corrupt("receipt refusal disagrees with consumed allowance")
				}
				switch r.Disposition {
				case "unscoped", "read_denied", "target_missing", "target_mismatch", "attempt_missing", "original_denied", "endpoint_mismatch", "unsupported_original", "original_open", "read_exhausted", "original_busy", "original_unsettled":
				default:
					return corrupt("unsupported local receipt-read refusal")
				}
				if r.Disposition == "read_exhausted" && c.runs[x.Source].Epoch <= ord[1].Resolver.Epoch {
					return corrupt("exhaustion without later activation")
				}
				if r.RefusalWitness != "" {
					w := c.effects[r.RefusalWitness]
					if w == nil || w.Source != c.intents[a.Prepared.Request.Payload["target"].Text].Run ||
						(w.ExternalAttempt == nil && w.AdapterAttempt == nil) {
						return corrupt("unbound receipt-read refusal witness")
					}
				}
			}
			continue
		}
		if len(ord) == 0 || r.Read.Ordinal != len(ord) {
			return corrupt("read result without its original physical opportunity")
		}
		o := ord[len(ord)-1]
		s, err := readSnapshot(o, r.Candidate)
		if err != nil {
			return err
		}
		sequence := c.resultOrder[id]
		if sequence <= o.Prefix+1 {
			return corrupt("receipt read result before its physical allowance")
		}
		original := c.effects[o.Query.TargetID]
		var prior *AdapterResult
		if order := c.resultOrder[original.OperationID]; order > 0 && order < sequence {
			prior = original.AdapterResult
		}
		decision := decideRead(o, s, prior)
		want := readResult(*a, r.Candidate, o.Query.Ordinal, decision)
		if !reflect.DeepEqual(r, &want) {
			return corrupt("receipt read decision does not match its original")
		}
		last := &x.ReadOpportunities[len(ord)-1]
		last.Adoption = decision
		x.Dispatch.Minimum = 1
		if r.Disposition == "validated" {
			last.Snapshot = &s
			if s.Status == "conflict" || decision == "conflict" {
				original.ReceiptEvidenceState = "conflict"
			}
			if s.Receipt != nil {
				if previous, ok := positive[original.OperationID]; ok && previous != *s.Receipt {
					original.ReceiptEvidenceState = "conflict"
				}
				positive[original.OperationID] = *s.Receipt
				if original.ReceiptEvidenceState == "none" {
					original.ReceiptEvidenceState = "known"
				}
			}
		}
		if decision == "adopted" {
			want := reconciledResult(o, *s.Receipt)
			if !reflect.DeepEqual(original.AdapterResult, &want) || c.resultOrder[original.OperationID] != sequence+1 ||
				x.Settlement != nil && original.Settlement == nil {
				return corrupt("invalid atomic original receipt adoption")
			}
			adoptions[original.OperationID] = true
		}
	}
	for _, id := range sortedKeys(c.effects) {
		x := c.effects[id]
		if x.AdapterResult != nil && x.AdapterResult.Reconciled != nil && !adoptions[id] {
			return corrupt("reconstructed original lacks original adopting query")
		}
	}
	return nil
}

// Match the fixed reader protocol without calling an adapter method, including Prepare.
func validReadPreparation(p effects.Prepared) bool {
	in, _ := receiptquery.ReadInput().Retain()
	out, _ := receiptquery.ReadOutput().Retain()
	_, err := receiptquery.EndpointID(p.Endpoint, "lookup")
	return err == nil && p.Protocol == receiptquery.ReadProtocol && p.Model == "" && p.Wire == "" && p.MaxOutputTokens == 0 && p.MaxResponseBytes == receiptquery.MaxSnapshotBytes &&
		p.Request.Input.ID == in.ID && p.Request.Output.ID == out.ID
}

func (c *effectCut) checkOrdinal(x *ExecutionEffect, o receiptReadOrdinal, g ReceiptGrant) error {
	q := o.Query
	original := c.effects[q.TargetID]
	run, targetID := g.target(c.snap.Namespace)
	a := x.AdapterAttempt
	resolver := c.runs[x.Source]
	if original == nil || original.Authorization == nil || !original.Authorization.Allowed || c.membership[x.Source] == "" || c.membership[run] == "" ||
		run != original.Source || g.Sequence != original.Sequence || o.OriginalRun != run || o.OriginalAuthorization != original.Authorization.ID ||
		o.Adopt != g.Adopt || q.TargetID != targetID || q.TargetID != a.Prepared.Request.Payload["target"].Text || q.QueryKey != a.Prepared.Request.Key ||
		q.TargetKey != ExternalKey(c.snap.Namespace, targetID) || o.Resolver.Run != x.Source || o.Resolver.Revision != x.Authorization.Source.Revision ||
		o.Resolver.Epoch < x.Authorization.Source.Epoch || o.Resolver.Epoch > resolver.Epoch || o.Resolver.Sequence < x.Authorization.Source.Sequence || o.Resolver.Sequence > resolver.Sequence ||
		o.Prefix <= 0 || o.OriginalTerminalSequence != c.runs[run].Sequence || c.runs[run].Projection.Status == "open" {
		return corrupt("receipt read source or target authority mismatch")
	}
	if g.Invocation != "" {
		v := c.invocations[g.Invocation]
		if v == nil || v.Admission == nil || !v.Admission.Accepted || !slices.Contains(v.Admission.Members, run) {
			return corrupt("receipt grant identifies a different member")
		}
	}
	expectedEndpoint, err := receiptquery.LookupEndpoint(g.Endpoint)
	ledger, ledgerErr := receiptquery.EndpointID(g.Endpoint, "apply")
	if err != nil || ledgerErr != nil || a.Prepared.Endpoint != expectedEndpoint || q.LedgerID != ledger {
		return corrupt("receipt query endpoint mismatch")
	}
	if original.AdapterAttempt != nil {
		old := original.AdapterAttempt
		m, e := receiptquery.DecodeMutation(old.Prepared.Wire)
		if !old.Allowed || old.Prepared.Protocol != receiptquery.CounterProtocol || old.Prepared.Request.Capability != receiptCounterCapability ||
			old.Prepared.Endpoint != g.Endpoint || e != nil || m.Key != q.TargetKey || m.Delta != old.Prepared.Request.Payload["delta"].Number ||
			q.Wire != old.Prepared.Wire || o.OriginalMode != old.Prepared.Model || o.OriginalPreparedDigest != old.Prepared.Digest ||
			o.OriginalAttemptDigest != compositionIdentity("alan/original-attempt/v1", *old) {
			return corrupt("receipt query original adapter binding")
		}
	} else {
		old := original.ExternalAttempt
		if old == nil || !old.Allowed || old.Intent.Target != g.Endpoint || o.OriginalMode != "raw" || o.OriginalPreparedDigest != "" ||
			o.OriginalAttemptDigest != compositionIdentity("alan/original-attempt/v1", *old) {
			return corrupt("receipt query original external binding")
		}
		raw, _ := json.Marshal(old.Request)
		if string(raw) != q.Wire {
			return corrupt("receipt query changed original wire")
		}
	}
	f := c.facts[compositionIdentity("alan/fact/receipt-read/v1", []any{x.OperationID, q.Ordinal})]
	if f.Kind != compositionReadFact || f.Order != o.Prefix+1 || f.Run != x.Source || f.Sequence != int64(q.Ordinal) || f.OperationID != x.OperationID ||
		!slices.Equal(f.RootIDs, []string{c.membership[x.Source]}) || f.RecordDigest != compositionIdentity("alan/receipt-read-ordinal/v1", o) {
		return corrupt("missing receipt-read fact at original prefix")
	}
	for _, id := range []string{x.OperationID, original.OperationID} {
		res := c.facts[compositionIdentity("alan/fact/reservation/v1", id)]
		if res.Order == 0 || res.Order > o.Prefix {
			return corrupt("receipt read predates original reservation")
		}
	}
	return nil
}
