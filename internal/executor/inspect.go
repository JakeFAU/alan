package executor

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
)

type ConnectionSettings struct {
	JournalMode, SQLiteVersion, SQLiteSourceID         string
	Synchronous, ForeignKeys, BusyTimeoutMS, QueryOnly int
}

// Inspection exposes exact retained BLOB bytes as UTF-8 JSON strings, including
// program artifacts. It is a coherent read-only snapshot, with no policy reducer.
type Inspection struct {
	Namespace  string
	KeyVersion int
	Settings   ConnectionSettings
	Tables     map[string][]map[string]any
}

var inspectionTables = []string{"metadata", "programs", "subjects", "permissions", "runs", "history", "observations", "intents", "attempts", "admissions", "lifecycle_results", "settlements", "terminal_obligations", "message_acknowledgments", "run_origins", "external_attempts", "external_receipts", "corpus_assets", "adapter_attempts", "adapter_results"}

func Inspect(ctx context.Context, path string, limits Limits) (snapshot Inspection, err error) {
	real, err := canonicalPath(path, false)
	if err != nil {
		return snapshot, err
	}
	store, err := connect(ctx, real, true)
	if err != nil {
		return snapshot, err
	}
	defer func() { err = joinClose(err, store.Close()) }()
	tx, err := store.conn.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return snapshot, err
	}
	defer tx.Rollback()
	version, source, err := verifyConnection(ctx, tx, true)
	if err != nil {
		return snapshot, err
	}
	snapshot.Settings.SQLiteVersion = version
	snapshot.Settings.SQLiteSourceID = source
	for _, entry := range []struct {
		query  string
		target any
	}{
		{"PRAGMA journal_mode", &snapshot.Settings.JournalMode}, {"PRAGMA synchronous", &snapshot.Settings.Synchronous}, {"PRAGMA foreign_keys", &snapshot.Settings.ForeignKeys}, {"PRAGMA busy_timeout", &snapshot.Settings.BusyTimeoutMS}, {"PRAGMA query_only", &snapshot.Settings.QueryOnly},
	} {
		if err = tx.QueryRowContext(ctx, entry.query).Scan(entry.target); err != nil {
			return snapshot, err
		}
	}
	snapshot.Namespace, err = readNamespace(ctx, tx)
	if err != nil {
		return snapshot, err
	}
	snapshot.KeyVersion = ExternalKeyVersion
	ids, err := stringList(ctx, tx, "SELECT id FROM programs ORDER BY id")
	if err != nil {
		return snapshot, err
	}
	for _, id := range ids {
		if _, err = loadProgram(ctx, tx, id, limits); err != nil {
			return snapshot, err
		}
	}
	if err = validateAdapters(ctx, tx, limits); err != nil {
		return snapshot, err
	}
	snapshot.Tables = map[string][]map[string]any{}
	for _, table := range inspectionTables {
		records, e := inspectTable(ctx, tx, table)
		if e != nil {
			return snapshot, e
		}
		snapshot.Tables[table] = records
	}
	return snapshot, tx.Commit()
}
func inspectTable(ctx context.Context, q queryer, table string) ([]map[string]any, error) {
	// table comes only from the closed inspectionTables list, never command input.
	rows, err := q.QueryContext(ctx, "SELECT * FROM "+table+" ORDER BY rowid")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	records := []map[string]any{}
	for rows.Next() {
		values := make([]any, len(columns))
		targets := make([]any, len(columns))
		for i := range targets {
			targets[i] = &values[i]
		}
		if err = rows.Scan(targets...); err != nil {
			return nil, err
		}
		record := map[string]any{}
		for i, column := range columns {
			value := values[i]
			if blob, ok := value.([]byte); ok {
				if !json.Valid(blob) {
					return nil, fmt.Errorf("invalid JSON in %s.%s: %w", table, column, ErrCorrupt)
				}
				value = string(blob)
			}
			record[column] = value
		}
		records = append(records, record)
	}
	return records, rows.Err()
}
