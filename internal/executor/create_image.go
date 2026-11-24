package executor

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io/fs"

	"github.com/JakeFAU/alan/internal/boundary"
	"github.com/mattn/go-sqlite3"
)

// The inputs have not reached disk. A failed integrity check here describes
// the proposed deployment. Storage and cancellation failures keep their cause.
func creationInputError(err error) error {
	if errors.Is(err, ErrCorrupt) {
		return fmt.Errorf("%w: inconsistent initial provision (%v)", fs.ErrInvalid, err)
	}
	var engine sqlite3.Error
	if errors.As(err, &engine) && engine.Code != sqlite3.ErrConstraint ||
		errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) ||
		errors.Is(err, sql.ErrTxDone) || errors.Is(err, sql.ErrConnDone) || errors.Is(err, driver.ErrBadConn) {
		return err
	}
	return fmt.Errorf("%w: initial provision: %w", fs.ErrInvalid, err)
}

// SQLite's backup API copies pages, preserving row IDs and indexes. The
// destination holds its write transaction across steps and commits the entire
// image on the last step under the verified WAL/FULL connection settings.
func persistStoreImage(ctx context.Context, dst, src *sql.Conn) error {
	return dst.Raw(func(d any) error {
		return src.Raw(func(s any) (err error) {
			backup, err := d.(*sqlite3.SQLiteConn).Backup("main", s.(*sqlite3.SQLiteConn), "main")
			if err != nil {
				return err
			}
			defer func() { err = errors.Join(err, backup.Finish()) }()
			done, err := backup.Step(1)
			if err != nil || done {
				return err
			}
			if err := boundary.Reached(ctx, boundary.Event{Boundary: "store.create", Phase: "before-commit"}); err != nil {
				return err
			}
			if err := ctx.Err(); err != nil {
				return err
			}
			done, err = backup.Step(-1)
			if err == nil && !done {
				return sqlite3.ErrBusy
			}
			return err
		})
	})
}
