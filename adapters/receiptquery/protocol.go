// Package receiptquery implements the opt-in controlled counter and its separate
// read-only application receipt protocol. It has no executor or Store access.
package receiptquery

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"unicode/utf8"
)

const (
	CounterCapability = "phase2c.counter"
	ReadCapability    = "phase2c.receipt-read"
	CounterProtocol   = "phase2c/counter/v1"
	ReadProtocol      = "phase2c/receipt-read/v1"
	MaxMutationBytes  = 8192
	MaxQueryBytes     = 32768
	MaxSnapshotBytes  = 65536
)

type Query struct {
	QueryID, QueryKey, TargetID, TargetKey, LedgerID, Wire, WireSHA256 string
	Ordinal                                                            int
}
type Receipt struct {
	Key, Profile           string
	Delta, Mutation, Total int64
	Process                string
	PID                    int
}
type Snapshot struct {
	Query       Query
	Number      int64
	Status      string
	Count       int64
	AppliedWire string
	Receipt     *Receipt
}

// Mutation is the extended raw external request. The controlled adapter sends
// just Key, Profile, Delta. Both forms bind the received bytes themselves.
type Mutation struct {
	Key, Profile                string
	Subject, Authorization      string
	Generation, Epoch, Sequence int64
	Delta                       int64
}
type shortMutation struct {
	Key, Profile string
	Delta        int64
}

func validID(s string) bool {
	if len(s) != 64 || strings.ToLower(s) != s {
		return false
	}
	_, err := hex.DecodeString(s)
	return err == nil
}

// EndpointID checks the complete route before extracting its pinned ledger ID.
func EndpointID(endpoint, verb string) (string, error) {
	errRoute := errors.New("invalid ledger endpoint")
	if verb != "apply" && verb != "lookup" {
		return "", errRoute
	}
	u, err := url.Parse(endpoint)
	if err != nil {
		return "", errRoute
	}
	addr, err := netip.ParseAddr(u.Hostname())
	if err != nil || !addr.IsLoopback() || addr.Zone() != "" {
		return "", errRoute
	}
	port, err := strconv.ParseUint(u.Port(), 10, 16)
	if err != nil || port == 0 {
		return "", errRoute
	}
	parts := strings.Split(u.Path, "/")
	if len(parts) != 4 || parts[1] != "phase2c" || !validID(parts[2]) || parts[3] != verb {
		return "", errRoute
	}
	if endpoint != "http://"+netip.AddrPortFrom(addr, uint16(port)).String()+"/phase2c/"+parts[2]+"/"+verb {
		return "", errRoute
	}
	return parts[2], nil
}
func LookupEndpoint(applyURL string) (string, error) {
	if _, err := EndpointID(applyURL, "apply"); err != nil {
		return "", err
	}
	return strings.TrimSuffix(applyURL, "/apply") + "/lookup", nil
}

func canonical(raw []byte, v any) bool {
	if !utf8.Valid(raw) || json.Unmarshal(raw, v) != nil {
		return false
	}
	b, err := json.Marshal(v)
	return err == nil && bytes.Equal(b, raw)
}

func DecodeMutation(raw string) (Mutation, error) {
	var m Mutation
	bad := errors.New("invalid bounded mutation wire")
	if len(raw) > MaxMutationBytes || !utf8.ValidString(raw) {
		return m, bad
	}
	var compact bytes.Buffer
	if json.Compact(&compact, []byte(raw)) != nil {
		return m, bad
	}
	if !canonical(compact.Bytes(), &m) {
		var s shortMutation
		if !canonical(compact.Bytes(), &s) {
			return Mutation{}, bad
		}
		m = Mutation{Key: s.Key, Profile: s.Profile, Delta: s.Delta}
	}
	if m.Key == "" || (m.Profile != "idempotent" && m.Profile != "non-idempotent") {
		return Mutation{}, bad
	}
	return m, nil
}

// namespaceOf checks the versioned key encoding without relying on the Store.
func namespaceOf(key, id string) string {
	var tuple []json.RawMessage
	if len(id) == 0 || len(id) > 1024 || len(key) > 8192 || json.Unmarshal([]byte(key), &tuple) != nil || len(tuple) != 3 {
		return ""
	}
	var ns string
	if json.Unmarshal(tuple[1], &ns) != nil || !validID(ns) {
		return ""
	}
	b, _ := json.Marshal([]any{2, ns, id})
	if string(b) != key {
		return ""
	}
	return ns
}

func (q Query) Check() error {
	ns := namespaceOf(q.QueryKey, q.QueryID)
	if ns == "" || namespaceOf(q.TargetKey, q.TargetID) != ns || q.QueryID == q.TargetID || !validID(q.LedgerID) || (q.Ordinal != 1 && q.Ordinal != 2) {
		return errors.New("invalid query or target identity")
	}
	original, err := DecodeMutation(q.Wire)
	if err != nil || original.Key != q.TargetKey || fmt.Sprintf("%x", sha256.Sum256([]byte(q.Wire))) != q.WireSHA256 {
		return errors.New("invalid original wire binding")
	}
	wire, _ := json.Marshal(q)
	if len(wire) > MaxQueryBytes {
		return errors.New("query wire too large")
	}
	return nil
}
func DecodeQuery(raw []byte) (Query, error) {
	var q Query
	if len(raw) > MaxQueryBytes || !canonical(raw, &q) {
		return q, errors.New("invalid query wire")
	}
	return q, q.Check()
}
func DecodeSnapshot(q Query, response string) (Snapshot, error) {
	var s Snapshot
	if err := q.Check(); err != nil {
		return s, err
	}
	if len(response) > MaxSnapshotBytes || !canonical([]byte(response), &s) || s.Query != q || s.Number < 1 {
		return Snapshot{}, errors.New("invalid snapshot envelope")
	}
	switch s.Status {
	case "found":
		m, _ := DecodeMutation(q.Wire)
		if s.Count == 1 && s.AppliedWire == q.Wire && s.Receipt != nil && s.Receipt.matches(m) {
			return s, nil
		}
	case "not_found":
		if s.Count == 0 && s.Receipt == nil && s.AppliedWire == "" {
			return s, nil
		}
	case "conflict":
		if s.Count > 0 && s.Receipt == nil && s.AppliedWire == "" {
			return s, nil
		}
	}
	return Snapshot{}, errors.New("invalid snapshot evidence")
}
func (r Receipt) matches(m Mutation) bool {
	return r.Key == m.Key && r.Profile == m.Profile && r.Delta == m.Delta && r.Mutation >= 1 && r.PID > 0 && r.Process != ""
}
