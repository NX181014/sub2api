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
	"github.com/shopspring/decimal"
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

	ids, groupIDs, err := lockAccountDeleteFamily(ctx, exec, accountID)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return nil, service.ErrAccountNotFound
	}
	if err := deleteAccountFamilyPreservingUsage(ctx, exec, ids, groupIDs); err != nil {
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

func lockAccountDeleteFamily(ctx context.Context, exec accountDeleteExecutor, accountID int64) ([]int64, []int64, error) {
	rows, err := exec.QueryContext(ctx, `
SELECT id
FROM accounts
WHERE (id=$1 OR parent_account_id=$1) AND deleted_at IS NULL
ORDER BY CASE WHEN id=$1 THEN 1 ELSE 0 END,id
FOR UPDATE`, accountID)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = rows.Close() }()
	ids := make([]int64, 0, 2)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, nil, err
	}
	if len(ids) == 0 {
		return ids, nil, nil
	}
	groupRows, err := exec.QueryContext(ctx, `
SELECT DISTINCT group_id
FROM account_groups
WHERE account_id=ANY($1)
ORDER BY group_id`, pq.Array(ids))
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = groupRows.Close() }()
	groupIDs := make([]int64, 0)
	for groupRows.Next() {
		var groupID int64
		if err := groupRows.Scan(&groupID); err != nil {
			return nil, nil, err
		}
		groupIDs = append(groupIDs, groupID)
	}
	if err := groupRows.Err(); err != nil {
		return nil, nil, err
	}
	return ids, groupIDs, nil
}

func deleteAccountFamilyPreservingUsage(ctx context.Context, exec accountDeleteExecutor, ids, groupIDs []int64) error {
	idArray := pq.Array(ids)
	textIDs := make([]string, len(ids))
	for i, id := range ids {
		textIDs[i] = strconv.FormatInt(id, 10)
	}
	if err := hardDeleteAccountSettlements(ctx, exec, ids, textIDs); err != nil {
		return err
	}

	queries := []struct {
		sql  string
		args []any
	}{
		{`DELETE FROM account_groups WHERE account_id=ANY($1)`, []any{idArray}},
		{`DELETE FROM scheduled_test_plans WHERE account_id=ANY($1)`, []any{idArray}},
		{`DELETE FROM sora_accounts WHERE account_id=ANY($1)`, []any{idArray}},
		{`DELETE FROM batch_image_jobs WHERE account_id=ANY($1)`, []any{idArray}},
		{`DELETE FROM ops_system_logs WHERE account_id=ANY($1)`, []any{idArray}},
		{`DELETE FROM ops_error_logs WHERE account_id=ANY($1)`, []any{idArray}},
		{`WITH RECURSIVE doomed(id) AS (
			SELECT id FROM account_cost_entries WHERE account_id=ANY($1)
			UNION
			SELECT c.id FROM account_cost_entries c JOIN doomed d ON c.supersedes_id=d.id
		) DELETE FROM account_cost_entries WHERE id IN (SELECT id FROM doomed)`, []any{idArray}},
		{`UPDATE account_cost_entries SET related_account_id=NULL,updated_at=NOW()
			WHERE related_account_id=ANY($1)`, []any{idArray}},
		{`DELETE FROM purchase_sources ps WHERE NOT EXISTS (
			SELECT 1 FROM account_cost_entries c WHERE c.purchase_source_id=ps.id)`, nil},
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
		{`UPDATE accounts SET
			name='deleted-account-'||id::text,notes=NULL,provider_identity=NULL,
			contributor_user_id=NULL,created_by_user_id=NULL,cost_sharing_enabled=FALSE,
			credentials='{}'::jsonb,
			extra=CASE WHEN platform='openai' THEN '{"openai_long_context_billing_enabled":false}'::jsonb ELSE '{}'::jsonb END,
			proxy_id=NULL,proxy_fallback_origin_id=NULL,concurrency=0,load_factor=NULL,priority=50,rate_multiplier=1,
			status='disabled',error_message=NULL,last_used_at=NULL,expires_at=NULL,auto_pause_on_expired=FALSE,
			schedulable=FALSE,rate_limited_at=NULL,rate_limit_reset_at=NULL,overload_until=NULL,
			temp_unschedulable_until=NULL,temp_unschedulable_reason=NULL,session_window_start=NULL,
			session_window_end=NULL,session_window_status=NULL,parent_account_id=NULL,quota_dimension='global',
			expected_token_count=NULL,deleted_at=COALESCE(deleted_at,NOW()),updated_at=NOW()
			WHERE id=ANY($1)`, []any{idArray}},
		{`DELETE FROM scheduler_outbox o WHERE account_id=ANY($1) OR EXISTS (
			SELECT 1 FROM jsonb_array_elements(CASE WHEN jsonb_typeof(o.payload->'account_ids')='array' THEN o.payload->'account_ids' ELSE '[]'::jsonb END) AS payload_id(value)
			WHERE payload_id.value::text=ANY($2::text[]))`, []any{idArray, pq.Array(textIDs)}},
	}
	for _, query := range queries {
		if _, err := exec.ExecContext(ctx, query.sql, query.args...); err != nil {
			return err
		}
	}
	payload := buildSchedulerGroupPayload(groupIDs)
	for _, id := range ids {
		if err := enqueueSchedulerOutbox(ctx, exec, service.SchedulerOutboxEventAccountChanged, &id, nil, payload); err != nil {
			return err
		}
	}
	return nil
}

func hardDeleteAccountSettlements(ctx context.Context, exec accountDeleteExecutor, ids []int64, textIDs []string) error {
	rows, err := exec.QueryContext(ctx, `
WITH RECURSIVE doomed_costs(id) AS (
  SELECT id FROM account_cost_entries WHERE account_id=ANY($2)
  UNION
  SELECT child.id FROM account_cost_entries child JOIN doomed_costs parent ON child.supersedes_id=parent.id
)
SELECT s.id
FROM pool_settlements s
WHERE s.filter_snapshot->>'account_id'=ANY($1::text[])
   OR EXISTS (SELECT 1 FROM pool_settlement_account_costs c
      WHERE c.settlement_id=s.id AND (c.account_id=ANY($2) OR c.cost_entry_id IN (SELECT id FROM doomed_costs)))
   OR EXISTS (SELECT 1 FROM pool_settlement_account_lines l WHERE l.settlement_id=s.id AND l.account_id=ANY($2))
ORDER BY s.id
FOR UPDATE`, pq.Array(textIDs), pq.Array(ids))
	if err != nil {
		return err
	}
	settlementIDs := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			_ = rows.Close()
			return err
		}
		settlementIDs = append(settlementIDs, id)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}
	if len(settlementIDs) == 0 {
		return nil
	}
	if _, err := exec.ExecContext(ctx, `
WITH RECURSIVE doomed_costs(id) AS (
  SELECT id FROM account_cost_entries WHERE account_id=ANY($1)
  UNION
  SELECT child.id FROM account_cost_entries child JOIN doomed_costs parent ON child.supersedes_id=parent.id
)
DELETE FROM pool_settlement_account_costs
WHERE account_id=ANY($1) OR cost_entry_id IN (SELECT id FROM doomed_costs)`, pq.Array(ids)); err != nil {
		return err
	}
	if _, err := exec.ExecContext(ctx, `DELETE FROM pool_settlement_account_lines WHERE account_id=ANY($1)`, pq.Array(ids)); err != nil {
		return err
	}
	for _, settlementID := range settlementIDs {
		if err := rebuildSettlementAfterAccountDelete(ctx, exec, settlementID, ids, textIDs); err != nil {
			return err
		}
	}
	return nil
}

func rebuildSettlementAfterAccountDelete(ctx context.Context, exec accountDeleteExecutor, settlementID int64, ids []int64, textIDs []string) error {
	costRows, err := exec.QueryContext(ctx, `
SELECT account_id,cost_entry_id,kind,payer_user_id,amount_minor
FROM pool_settlement_account_costs WHERE settlement_id=$1 ORDER BY account_id,kind,cost_entry_id`, settlementID)
	if err != nil {
		return err
	}
	costs := make([]service.SettlementCostSnapshot, 0)
	periodCost, carryIn := int64(0), int64(0)
	for costRows.Next() {
		var item service.SettlementCostSnapshot
		if err := costRows.Scan(&item.AccountID, &item.EntryID, &item.Kind, &item.PayerUserID, &item.AmountMinor); err != nil {
			_ = costRows.Close()
			return err
		}
		if item.Kind == "carry" {
			carryIn += item.AmountMinor
		} else {
			periodCost += item.AmountMinor
		}
		costs = append(costs, item)
	}
	if err := costRows.Err(); err != nil {
		_ = costRows.Close()
		return err
	}
	if err := costRows.Close(); err != nil {
		return err
	}
	weightRows, err := exec.QueryContext(ctx, `
SELECT account_id,user_id,account_usage_weight::text
FROM pool_settlement_account_lines WHERE settlement_id=$1 ORDER BY account_id,user_id`, settlementID)
	if err != nil {
		return err
	}
	weights := make([]service.PoolUsageWeight, 0)
	for weightRows.Next() {
		var item service.PoolUsageWeight
		var raw string
		if err := weightRows.Scan(&item.AccountID, &item.UserID, &raw); err != nil {
			_ = weightRows.Close()
			return err
		}
		item.Weight, err = decimal.NewFromString(raw)
		if err != nil {
			_ = weightRows.Close()
			return err
		}
		weights = append(weights, item)
	}
	if err := weightRows.Err(); err != nil {
		_ = weightRows.Close()
		return err
	}
	if err := weightRows.Close(); err != nil {
		return err
	}
	if len(costs) == 0 && len(weights) == 0 {
		_, err := exec.ExecContext(ctx, `DELETE FROM pool_settlements WHERE id=$1`, settlementID)
		return err
	}

	lines, accountLines, totalWeight, carryOut := service.BuildSettlementAllocation(costs, weights)
	var status string
	var generatedBy int64
	var lockedBy, paidBy sql.NullInt64
	var lockedAt, paidAt sql.NullTime
	headerRows, err := exec.QueryContext(ctx, `
SELECT status,generated_by_user_id,locked_by_user_id,locked_at,paid_by_user_id,paid_at
FROM pool_settlements WHERE id=$1`, settlementID)
	if err != nil {
		return err
	}
	if !headerRows.Next() {
		_ = headerRows.Close()
		return sql.ErrNoRows
	}
	if err := headerRows.Scan(&status, &generatedBy, &lockedBy, &lockedAt, &paidBy, &paidAt); err != nil {
		_ = headerRows.Close()
		return err
	}
	if err := headerRows.Close(); err != nil {
		return err
	}

	var unpricedCount int64
	var pricingCoverage string
	coverageRows, err := exec.QueryContext(ctx, `
WITH coverage AS (
  SELECT COUNT(*) FILTER (WHERE ul.total_cost<=0)::bigint unpriced_count,
         COALESCE(SUM(GREATEST(ul.input_tokens::bigint+ul.output_tokens::bigint+ul.cache_creation_tokens::bigint+
           ul.cache_read_tokens::bigint+ul.image_output_tokens::bigint+ul.image_input_tokens::bigint,1)),0)::numeric material,
         COALESCE(SUM(GREATEST(ul.input_tokens::bigint+ul.output_tokens::bigint+ul.cache_creation_tokens::bigint+
           ul.cache_read_tokens::bigint+ul.image_output_tokens::bigint+ul.image_input_tokens::bigint,1))
           FILTER (WHERE ul.total_cost<=0),0)::numeric unpriced_material
  FROM pool_settlements s
  JOIN usage_logs ul ON ul.created_at>=s.period_start AND ul.created_at<s.period_end
  JOIN accounts a ON a.id=ul.account_id AND a.cost_sharing_enabled=TRUE AND NOT a.id=ANY($2)
  JOIN users u ON u.id=ul.user_id AND u.deleted_at IS NULL
  WHERE s.id=$1
    AND (s.filter_snapshot->>'account_id' IS NULL OR a.id=(s.filter_snapshot->>'account_id')::bigint)
    AND (s.filter_snapshot->>'uploader_user_id' IS NULL OR a.created_by_user_id=(s.filter_snapshot->>'uploader_user_id')::bigint)
    AND EXISTS (
      SELECT 1 FROM account_cost_entries c
      WHERE c.account_id=a.id AND c.entry_type<>'write_off'
        AND c.service_start<s.period_end::date AND c.service_end>s.period_start::date
        AND (s.filter_snapshot->>'payer_user_id' IS NULL OR c.payer_user_id=(s.filter_snapshot->>'payer_user_id')::bigint)
        AND (s.filter_snapshot->>'purchase_source_id' IS NULL OR c.purchase_source_id=(s.filter_snapshot->>'purchase_source_id')::bigint)
    )
    AND (ul.actual_cost>0 OR ul.total_cost>0
      OR ul.input_tokens+ul.output_tokens+ul.cache_creation_tokens+ul.cache_read_tokens+ul.image_output_tokens+ul.image_input_tokens>0
      OR ul.image_count>0 OR ul.video_count>0)
)
SELECT unpriced_count,
       CASE WHEN material=0 THEN '1' ELSE ((material-unpriced_material)/material)::text END
FROM coverage`, settlementID, pq.Array(ids))
	if err != nil {
		return err
	}
	if !coverageRows.Next() {
		_ = coverageRows.Close()
		return sql.ErrNoRows
	}
	if err := coverageRows.Scan(&unpricedCount, &pricingCoverage); err != nil {
		_ = coverageRows.Close()
		return err
	}
	if err := coverageRows.Close(); err != nil {
		return err
	}

	if _, err := exec.ExecContext(ctx, `DELETE FROM pool_settlement_lines WHERE settlement_id=$1`, settlementID); err != nil {
		return err
	}
	if _, err := exec.ExecContext(ctx, `DELETE FROM pool_settlement_account_lines WHERE settlement_id=$1`, settlementID); err != nil {
		return err
	}
	for _, line := range lines {
		paymentStatus, confirmationStatus := "unpaid", "pending"
		var confirmedBy any
		var confirmedAt any
		if status == "paid" {
			paymentStatus, confirmationStatus = "paid", "confirmed"
			confirmedBy = generatedBy
			confirmedAt = time.Now()
			if lockedBy.Valid {
				confirmedBy = lockedBy.Int64
			}
			if paidBy.Valid {
				confirmedBy = paidBy.Int64
			}
			if lockedAt.Valid {
				confirmedAt = lockedAt.Time
			}
			if paidAt.Valid {
				confirmedAt = paidAt.Time
			}
		}
		if _, err := exec.ExecContext(ctx, `
INSERT INTO pool_settlement_lines(settlement_id,user_id,usage_weight,usage_share,allocated_cost_minor,
  contribution_credit_minor,adjustment_minor,net_amount_minor,payment_status,confirmation_status,confirmed_by_user_id,confirmed_at)
VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`, settlementID, line.UserID, line.UsageWeight, line.UsageShare,
			line.AllocatedCostMinor, line.ContributionCreditMinor, line.AdjustmentMinor, line.NetAmountMinor,
			paymentStatus, confirmationStatus, confirmedBy, confirmedAt); err != nil {
			return err
		}
	}
	for _, line := range accountLines {
		if _, err := exec.ExecContext(ctx, `
INSERT INTO pool_settlement_account_lines(settlement_id,account_id,user_id,account_usage_weight,usage_share,
  allocated_cost_minor,contribution_credit_minor,adjustment_minor,net_amount_minor,trace_quality)
VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, settlementID, line.AccountID, line.UserID, line.AccountUsageWeight,
			line.UsageShare, line.AllocatedCostMinor, line.ContributionCreditMinor, line.AdjustmentMinor,
			line.NetAmountMinor, line.TraceQuality); err != nil {
			return err
		}
	}
	_, err = exec.ExecContext(ctx, `
UPDATE pool_settlements
SET period_cost_minor=$2,carry_in_minor=$3,carry_out_minor=$4,total_cost_minor=$5,total_usage_weight=$6,
    pricing_coverage=$7,unpriced_usage_count=$8,cost_snapshot='[]'::jsonb,
    filter_snapshot=CASE WHEN filter_snapshot->>'account_id'=ANY($9::text[]) THEN filter_snapshot-'account_id' ELSE filter_snapshot END,
    updated_at=NOW()
WHERE id=$1`, settlementID, periodCost, carryIn, carryOut, periodCost+carryIn, totalWeight.String(), pricingCoverage, unpricedCount, pq.Array(textIDs))
	return err
}
