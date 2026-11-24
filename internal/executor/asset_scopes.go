package executor

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/JakeFAU/alan/effects"
	publicschema "github.com/JakeFAU/alan/schema"
)

// Each scoped format preserves its underlying protocol, including conflict promotion.
const assetScopeFormatOffset = 14

func hasAssetScopeFormat(v int) bool {
	return v >= SchemaVersion+assetScopeFormatOffset && v <= requestAuthorityBindingConflictSchema+assetScopeFormatOffset
}

func unscopedFormat(v int) int {
	if hasAssetScopeFormat(v) {
		return v - assetScopeFormatOffset
	}
	return v
}

type assetBinding struct{ ID, Version string }
type assetManifest map[string][]assetBinding

func bindAssets(assets []effects.Asset, scopes map[string][]string) (assetManifest, error) {
	versions := map[string]string{}
	for _, a := range assets {
		versions[a.ID] = a.Version
	}
	m := assetManifest{}
	for capability, ids := range scopes {
		if err := publicschema.CheckIdentifier(capability); err != nil {
			return nil, fmt.Errorf("asset scope capability: %w", err)
		}
		names := slices.Sorted(slices.Values(ids))
		refs := make([]assetBinding, 0, len(names))
		for i, id := range names {
			version, ok := versions[id]
			if !ok {
				return nil, fmt.Errorf("asset scope %q: missing asset %q", capability, id)
			}
			if i > 0 && names[i-1] == id {
				return nil, fmt.Errorf("asset scope %q: duplicate asset %q", capability, id)
			}
			refs = append(refs, assetBinding{id, version})
		}
		m[capability] = refs
	}
	return m, nil
}

func writeAssetScopes(ctx context.Context, tx *sql.Tx, p Provision) error {
	m, err := bindAssets(p.Assets, p.AssetScopes)
	if err != nil {
		return err
	}
	raw, err := json.Marshal(m)
	if err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, "CREATE TABLE capability_asset_scopes(singleton INTEGER PRIMARY KEY CHECK(singleton=1), descriptor BLOB NOT NULL)"); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO capability_asset_scopes VALUES(1,?)", raw); err != nil {
		return err
	}
	// This independent format witness survives loss of the scope table and
	// binds the deployment declaration separately from every paid request.
	if _, err = tx.ExecContext(ctx, "ALTER TABLE metadata ADD COLUMN asset_scope_digest TEXT"); err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, "UPDATE metadata SET asset_scope_digest=? WHERE singleton=1", compositionIdentity("alan/asset-scope/v1", m))
	return err
}

func decodeAssetScopes(raw []byte, digest string) (assetManifest, error) {
	var m assetManifest
	if decode(raw, &m) != nil || m == nil || !sameJSON(raw, m) || compositionIdentity("alan/asset-scope/v1", m) != digest {
		return nil, corrupt("asset scope manifest")
	}
	for capability, refs := range m {
		if publicschema.CheckIdentifier(capability) != nil || refs == nil || len(refs) > effects.MaxAssets {
			return nil, corrupt("invalid asset scope")
		}
		for i, b := range refs {
			if publicschema.CheckIdentifier(b.ID) != nil || len(b.Version) != 64 || i > 0 && refs[i-1].ID >= b.ID {
				return nil, corrupt("invalid asset binding")
			}
		}
	}
	return m, nil
}

func readAssetScopes(ctx context.Context, q queryer) (assetManifest, error) {
	var raw []byte
	var digest string
	err := q.QueryRowContext(ctx, "SELECT s.descriptor,m.asset_scope_digest FROM capability_asset_scopes s JOIN metadata m USING(singleton) WHERE s.singleton=1").Scan(&raw, &digest)
	if err == sql.ErrNoRows {
		return nil, corrupt("missing asset scope manifest")
	}
	if err != nil {
		return nil, err
	}
	return decodeAssetScopes(raw, digest)
}

func (m assetManifest) verify(assets []effects.Asset) error {
	versions := map[string]string{}
	for _, a := range assets {
		versions[a.ID] = a.Version
	}
	for _, refs := range m {
		for _, b := range refs {
			if versions[b.ID] != b.Version {
				return corrupt("asset scope version or identity")
			}
		}
	}
	return nil
}

func scopedAssets(ctx context.Context, q queryer, capability string) ([]effects.Asset, error) {
	v, err := storedSchemaNumber(ctx, q)
	if err != nil {
		return nil, err
	}
	if !hasAssetScopeFormat(v) {
		return readAssets(ctx, q)
	}
	m, err := readAssetScopes(ctx, q)
	if err != nil {
		return nil, err
	}
	result := make([]effects.Asset, 0, len(m[capability]))
	for _, b := range m[capability] {
		a, err := readAsset(ctx, q, b.ID)
		if err != nil {
			return nil, err
		}
		if a.Version != b.Version {
			return nil, corrupt("selected asset version")
		}
		result = append(result, a)
	}
	return result, nil
}

// Called under the owner lock. Permission to prepare selected bytes is checked
// before the bytes leave the runtime. Spending and dispatch have independent,
// later checks. An already released copy cannot be recalled by revocation.
func (s *Store) withheldAssetPreparation(ctx context.Context, i retainedIntent, req effects.Request) (*effects.Prepared, error) {
	if len(req.Assets) == 0 {
		return nil, nil
	}
	v, err := storedSchemaNumber(ctx, s.conn)
	if err != nil || !hasAssetScopeFormat(v) {
		return nil, err
	}
	reason, err := s.dispatchSourceReason(ctx, i.Run, i.revision)
	if err != nil {
		return nil, err
	}
	allowed := reason == ""
	if allowed {
		if hasRequestAuthoritySchema(v) {
			w, err := evaluateEffectAuthority(ctx, s.conn, i, authorityCurrentPrefix)
			if err != nil {
				return nil, err
			}
			allowed = w.Allowed
		} else {
			allowed, err = adapterGrant(ctx, s.conn, i)
			if err != nil {
				return nil, err
			}
		}
	}
	if allowed {
		allowed, _, err = compositionReservationDecision(ctx, s.conn, i.Intent, true, "authorized")
		if err != nil {
			return nil, err
		}
	}
	if allowed {
		return nil, nil
	}

	// A refusal needs a retained reservation/result for the existing accounting
	// protocol. Its bytes are constructed locally, without adapter access. A
	// protected continuation also has to match the original frozen action.
	p := effects.Prepared{Request: req, Protocol: "alan/withheld-assets/v1", MaxResponseBytes: 1}
	if slot, ok := s.continuationSlot(i.Run); ok {
		a, err := selectedContinuation(ctx, s.conn, slot, i.Sequence, i.revision)
		if err != nil {
			return nil, err
		}
		prior, err := loadAdapterAttempt(ctx, s.conn, a.PredecessorID)
		if err != nil {
			return nil, err
		}
		p = prior.Prepared
		p.Request = req
	}
	return &p, nil
}
