package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type accountDeleteTxStarter interface {
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
}

type accountDeleteExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func (r *accountRepository) DeleteAccountWithLifecycle(ctx context.Context, accountID int64, _ service.AccountDeleteOptions) (*service.AccountDeleteResult, error) {
	if accountID <= 0 {
		return nil, service.ErrAccountNotFound
	}

	var exec accountDeleteExecutor
	var tx *sql.Tx
	if dbent.TxFromContext(ctx) != nil {
		exec = clientFromContext(ctx, r.client)
	} else {
		starter, ok := r.sql.(accountDeleteTxStarter)
		if !ok {
			return nil, fmt.Errorf("account repository SQL transaction support is not configured")
		}
		var err error
		tx, err = starter.BeginTx(ctx, nil)
		if err != nil {
			return nil, fmt.Errorf("begin account delete: %w", err)
		}
		defer func() { _ = tx.Rollback() }()
		exec = tx
	}

	ids, err := lockAccountDeleteFamily(ctx, exec, accountID)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return nil, service.ErrAccountNotFound
	}
	if err := hardDeleteAccountFamily(ctx, exec, ids); err != nil {
		return nil, err
	}
	if tx != nil {
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		for _, id := range ids {
			r.deleteSchedulerAccountSnapshot(ctx, id)
		}
	}
	return &service.AccountDeleteResult{AccountID: accountID, AffectedAccountIDs: ids}, nil
}

func lockAccountDeleteFamily(ctx context.Context, exec accountDeleteExecutor, accountID int64) ([]int64, error) {
	rows, err := exec.QueryContext(ctx, `
SELECT id
FROM accounts
WHERE id=$1 OR parent_account_id=$1
ORDER BY CASE WHEN id=$1 THEN 1 ELSE 0 END, id
FOR UPDATE`, accountID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	ids := make([]int64, 0, 2)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func hardDeleteAccountFamily(ctx context.Context, exec accountDeleteExecutor, ids []int64) error {
	idArray := pq.Array(ids)
	textIDs := make([]string, len(ids))
	for i, id := range ids {
		textIDs[i] = strconv.FormatInt(id, 10)
	}
	if err := hardDeleteAccountUsage(ctx, exec, ids); err != nil {
		return err
	}

	queries := []struct {
		sql  string
		args []any
	}{
		{`DELETE FROM pool_settlements s WHERE s.filter_snapshot->>'account_id'=ANY($1::text[]) OR EXISTS (
			SELECT 1 FROM jsonb_array_elements(CASE WHEN jsonb_typeof(s.cost_snapshot)='array' THEN s.cost_snapshot ELSE '[]'::jsonb END) item
			WHERE item->>'account_id'=ANY($1::text[]))`, []any{pq.Array(textIDs)}},
		{`DELETE FROM batch_image_jobs WHERE account_id=ANY($1)`, []any{idArray}},
		{`DELETE FROM ops_system_logs WHERE account_id=ANY($1)`, []any{idArray}},
		{`DELETE FROM ops_error_logs WHERE account_id=ANY($1)`, []any{idArray}},
		{`WITH RECURSIVE doomed(id) AS (
			SELECT id FROM account_cost_entries WHERE account_id=ANY($1) OR related_account_id=ANY($1)
			UNION
			SELECT c.id FROM account_cost_entries c JOIN doomed d ON c.supersedes_id=d.id
		) DELETE FROM account_cost_entries WHERE id IN (SELECT id FROM doomed)`, []any{idArray}},
		{`DELETE FROM account_lifecycle_events WHERE account_id=ANY($1) OR replacement_account_id=ANY($1)`, []any{idArray}},
		{`DELETE FROM pool_approval_requests WHERE account_id=ANY($1)
			OR payload#>>'{delete_options,ReplacementAccountID}'=ANY($2::text[])
			OR payload#>>'{delete_options,replacement_account_id}'=ANY($2::text[])
			OR change_summary#>>'{fields,replacement_account_id,after}'=ANY($2::text[])`, []any{idArray, pq.Array(textIDs)}},
		{`UPDATE channel_account_stats_pricing_rules SET account_ids=ARRAY(
			SELECT account_id FROM unnest(account_ids) AS account_id WHERE NOT account_id=ANY($1)) WHERE account_ids && $1`, []any{idArray}},
		{`UPDATE groups g SET model_routing=(
			SELECT COALESCE(jsonb_object_agg(entry.key, (
				SELECT COALESCE(jsonb_agg(route.value), '[]'::jsonb)
				FROM jsonb_array_elements(entry.value) AS route(value)
				WHERE route.value::text<>ALL($1::text[])
			)), '{}'::jsonb)
			FROM jsonb_each(COALESCE(g.model_routing, '{}'::jsonb)) entry
		) WHERE EXISTS (
			SELECT 1 FROM jsonb_each(COALESCE(g.model_routing, '{}'::jsonb)) entry
			CROSS JOIN LATERAL jsonb_array_elements(entry.value) AS route(value)
			WHERE route.value::text=ANY($1::text[]))`, []any{pq.Array(textIDs)}},
		{`UPDATE accounts SET extra=COALESCE(extra, '{}'::jsonb)-'linked_openai_account_id',updated_at=NOW()
			WHERE extra->>'linked_openai_account_id'=ANY($1::text[]) AND NOT id=ANY($2)`, []any{pq.Array(textIDs), idArray}},
		{`DELETE FROM scheduler_outbox o WHERE account_id=ANY($1) OR EXISTS (
			SELECT 1 FROM jsonb_array_elements(CASE WHEN jsonb_typeof(o.payload->'account_ids')='array' THEN o.payload->'account_ids' ELSE '[]'::jsonb END) AS payload_id(value)
			WHERE payload_id.value::text=ANY($2::text[]))`, []any{idArray, pq.Array(textIDs)}},
		{`DELETE FROM accounts WHERE id=ANY($1) AND id<>$2`, []any{idArray, ids[len(ids)-1]}},
		{`DELETE FROM accounts WHERE id=$1`, []any{ids[len(ids)-1]}},
		{`INSERT INTO scheduler_outbox(event_type,account_id)
			SELECT $2,account_id FROM unnest($1::bigint[]) AS account_id`, []any{idArray, service.SchedulerOutboxEventAccountChanged}},
	}
	for _, query := range queries {
		if _, err := exec.ExecContext(ctx, query.sql, query.args...); err != nil {
			return err
		}
	}
	return nil
}

func hardDeleteAccountUsage(ctx context.Context, exec accountDeleteExecutor, ids []int64) error {
	rows, err := exec.QueryContext(ctx, `SELECT MIN(created_at),MAX(created_at) FROM usage_logs WHERE account_id=ANY($1)`, pq.Array(ids))
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	var start, end sql.NullTime
	if !rows.Next() {
		return rows.Err()
	}
	if err := rows.Scan(&start, &end); err != nil {
		return err
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}
	if !start.Valid || !end.Valid {
		return nil
	}
	if _, err := exec.ExecContext(ctx, `DELETE FROM usage_logs WHERE account_id=ANY($1)`, pq.Array(ids)); err != nil {
		return err
	}
	// ponytail: account deletion is rare; reuse the existing exact range rebuild instead of duplicating aggregate SQL.
	return newDashboardAggregationRepositoryWithSQL(exec).RecomputeRange(ctx, start.Time, end.Time.Add(time.Nanosecond))
}
