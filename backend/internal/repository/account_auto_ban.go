package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type sqlTxBeginner interface {
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
}

// RecordAutomaticBan appends one system lifecycle event per uninterrupted ban
// period. The account-scoped advisory lock makes repeated concurrent failures
// idempotent; a later recovered event opens a new period.
func (r *accountRepository) RecordAutomaticBan(ctx context.Context, accountID int64, occurredAt time.Time, reason string) (bool, error) {
	beginner, ok := r.sql.(sqlTxBeginner)
	if !ok {
		return false, fmt.Errorf("automatic ban repository requires transaction support")
	}
	tx, err := beginner.BeginTx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("begin automatic ban transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err = tx.ExecContext(ctx,
		`SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`,
		fmt.Sprintf("account-auto-ban:%d", accountID),
	); err != nil {
		return false, fmt.Errorf("lock automatic ban account: %w", err)
	}

	result, err := tx.ExecContext(ctx, `
INSERT INTO account_lifecycle_events(
    account_id, event_type, occurred_at, reason, replacement_account_id,
    transferred_cost_minor, created_by_user_id, source)
SELECT $1, 'banned_confirmed', $2, $3, NULL, 0, NULL, 'automatic'
WHERE EXISTS (
    SELECT 1 FROM accounts a WHERE a.id = $1 AND a.deleted_at IS NULL
)
AND NOT EXISTS (
    SELECT 1
    FROM account_lifecycle_events banned
    WHERE banned.account_id = $1
      AND banned.event_type = 'banned_confirmed'
      AND NOT EXISTS (
          SELECT 1
          FROM account_lifecycle_events recovered
          WHERE recovered.account_id = banned.account_id
            AND recovered.event_type = 'recovered'
            AND (recovered.occurred_at, recovered.id) > (banned.occurred_at, banned.id)
      )
)`, accountID, occurredAt.UTC(), strings.TrimSpace(reason))
	if err != nil {
		return false, fmt.Errorf("insert automatic ban lifecycle: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("read automatic ban result: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return false, fmt.Errorf("commit automatic ban transaction: %w", err)
	}
	return affected > 0, nil
}
