package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type poolApprovalRepository struct{ client *dbent.Client }

func NewPoolApprovalRepository(client *dbent.Client) service.PoolApprovalRepository {
	return &poolApprovalRepository{client: client}
}

func (r *poolApprovalRepository) executor(ctx context.Context) *dbent.Client {
	return clientFromContext(ctx, r.client)
}

const poolApprovalSelect = `
SELECT r.id, r.action_type, r.account_id, a.name, r.status, r.reason,
       r.base_revision, r.requested_by_user_id, requester.email,
       r.decided_by_user_id, decider.email, r.decision_reason,
       r.requested_at, r.expires_at, r.decided_at, r.reveal_expires_at,
       r.revealed_at, r.primary_bypass, r.change_summary, r.payload
FROM pool_approval_requests r
JOIN accounts a ON a.id = r.account_id
JOIN users requester ON requester.id = r.requested_by_user_id
LEFT JOIN users decider ON decider.id = r.decided_by_user_id`

func scanPoolApproval(scanner interface{ Scan(...any) error }) (*service.PoolApproval, error) {
	item := &service.PoolApproval{}
	var summaryRaw, payloadRaw []byte
	if err := scanner.Scan(
		&item.ID, &item.ActionType, &item.AccountID, &item.AccountName, &item.Status, &item.Reason,
		&item.BaseRevision, &item.RequestedByUserID, &item.RequestedByEmail,
		&item.DecidedByUserID, &item.DecidedByEmail, &item.DecisionReason,
		&item.RequestedAt, &item.ExpiresAt, &item.DecidedAt, &item.RevealExpiresAt,
		&item.RevealedAt, &item.PrimaryBypass, &summaryRaw, &payloadRaw,
	); err != nil {
		return nil, err
	}
	if len(summaryRaw) > 0 {
		if err := json.Unmarshal(summaryRaw, &item.Changes); err != nil {
			return nil, fmt.Errorf("decode approval summary: %w", err)
		}
	}
	item.Payload = append(json.RawMessage(nil), payloadRaw...)
	return item, nil
}

func (r *poolApprovalRepository) ExpireStale(ctx context.Context, now time.Time) error {
	_, err := r.executor(ctx).ExecContext(ctx, `
UPDATE pool_approval_requests
SET status='expired',
    decision_reason=COALESCE(decision_reason, 'approval window expired'),
    decided_at=COALESCE(decided_at, $1), payload='{}'::jsonb, updated_at=$1
WHERE (status='pending' AND expires_at <= $1)
   OR (action_type='VIEW_CREDENTIAL' AND status='approved'
       AND revealed_at IS NULL AND reveal_expires_at <= $1)`, now.UTC())
	return err
}

func (r *poolApprovalRepository) CreateApproval(ctx context.Context, item *service.PoolApproval) (*service.PoolApproval, error) {
	if item == nil {
		return nil, fmt.Errorf("nil approval request")
	}
	summary, err := json.Marshal(item.Changes)
	if err != nil {
		return nil, err
	}
	rows, err := r.executor(ctx).QueryContext(ctx, `
INSERT INTO pool_approval_requests(
    action_type,account_id,status,payload,change_summary,reason,base_revision,
    requested_by_user_id,requested_at,expires_at,primary_bypass)
VALUES($1,$2,$3,$4::jsonb,$5::jsonb,$6,$7,$8,$9,$10,$11)
RETURNING id`, item.ActionType, item.AccountID, item.Status, []byte(item.Payload), summary,
		item.Reason, item.BaseRevision, item.RequestedByUserID, item.RequestedAt, item.ExpiresAt, item.PrimaryBypass)
	if err != nil {
		if isPoolApprovalUniqueViolation(err) {
			return nil, service.ErrPoolApprovalConflict
		}
		return nil, fmt.Errorf("create pool approval: %w", err)
	}
	if !rows.Next() {
		_ = rows.Close()
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("create pool approval returned no id")
	}
	var id int64
	if err := rows.Scan(&id); err != nil {
		_ = rows.Close()
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	return r.GetApproval(ctx, id, false)
}

func isPoolApprovalUniqueViolation(err error) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && pqErr.Code == "23505"
}

func (r *poolApprovalRepository) ListApprovals(ctx context.Context, filter service.PoolApprovalFilter, limit, offset int) ([]service.PoolApproval, int64, error) {
	clauses := []string{"1=1"}
	args := make([]any, 0, 6)
	add := func(clause string, value any) {
		args = append(args, value)
		clauses = append(clauses, fmt.Sprintf(clause, len(args)))
	}
	if filter.Status != "" {
		add("r.status=$%d", filter.Status)
	}
	if filter.ActionType != "" {
		add("r.action_type=$%d", filter.ActionType)
	}
	if filter.AccountID != nil {
		add("r.account_id=$%d", *filter.AccountID)
	}
	if filter.RequestedByUserID != nil {
		add("r.requested_by_user_id=$%d", *filter.RequestedByUserID)
	}
	switch filter.Scope {
	case "reviewable":
		add("r.requested_by_user_id<>$%d", filter.ActorID)
		clauses = append(clauses, "r.status='pending'")
	case "mine":
		add("r.requested_by_user_id=$%d", filter.ActorID)
	case "processed":
		add("r.decided_by_user_id=$%d", filter.ActorID)
		clauses = append(clauses, "r.status<>'pending'")
	}
	if filter.HighRisk != nil {
		add("COALESCE((r.change_summary#>>'{business,high_risk}')::boolean,false)=$%d", *filter.HighRisk)
	}
	where := " WHERE " + strings.Join(clauses, " AND ")

	countRows, err := r.executor(ctx).QueryContext(ctx, `SELECT COUNT(*) FROM pool_approval_requests r`+where, args...)
	if err != nil {
		return nil, 0, err
	}
	var total int64
	if countRows.Next() {
		err = countRows.Scan(&total)
	} else if countRows.Err() != nil {
		err = countRows.Err()
	} else {
		err = sql.ErrNoRows
	}
	_ = countRows.Close()
	if err != nil {
		return nil, 0, err
	}

	args = append(args, limit, offset)
	query := poolApprovalSelect + where + fmt.Sprintf(" ORDER BY r.requested_at DESC, r.id DESC LIMIT $%d OFFSET $%d", len(args)-1, len(args))
	rows, err := r.executor(ctx).QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()
	items := make([]service.PoolApproval, 0, limit)
	for rows.Next() {
		item, scanErr := scanPoolApproval(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		items = append(items, *item)
	}
	return items, total, rows.Err()
}

func (r *poolApprovalRepository) GetApproval(ctx context.Context, id int64, forUpdate bool) (*service.PoolApproval, error) {
	query := poolApprovalSelect + ` WHERE r.id=$1`
	if forUpdate {
		query += ` FOR UPDATE OF r`
	}
	rows, err := r.executor(ctx).QueryContext(ctx, query, id)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return nil, service.ErrPoolApprovalNotFound
	}
	item, err := scanPoolApproval(rows)
	if err != nil {
		return nil, err
	}
	return item, rows.Err()
}

func (r *poolApprovalRepository) LockAccount(ctx context.Context, accountID int64) error {
	rows, err := r.executor(ctx).QueryContext(ctx, `SELECT id FROM accounts WHERE id=$1 AND deleted_at IS NULL FOR UPDATE`, accountID)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return err
		}
		return service.ErrPoolAccountNotFound
	}
	return nil
}

func (r *poolApprovalRepository) GetApprovalAccountState(ctx context.Context, accountID int64) (*service.PoolApprovalAccountState, error) {
	rows, err := r.executor(ctx).QueryContext(ctx, `
SELECT provider_identity,contributor_user_id,created_by_user_id,cost_sharing_enabled
FROM accounts WHERE id=$1 AND deleted_at IS NULL`, accountID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return nil, service.ErrPoolAccountNotFound
	}
	item := &service.PoolApprovalAccountState{}
	if err := rows.Scan(&item.ProviderIdentity, &item.ContributorUserID, &item.CreatedByUserID, &item.CostSharingEnabled); err != nil {
		return nil, err
	}
	return item, rows.Err()
}

func (r *poolApprovalRepository) GetAccountDeleteImpact(ctx context.Context, accountID int64) (*service.PoolAccountDeleteImpact, error) {
	rows, err := r.executor(ctx).QueryContext(ctx, `
WITH RECURSIVE family AS (
  SELECT id,credentials FROM accounts WHERE (id=$1 OR parent_account_id=$1) AND deleted_at IS NULL
), doomed_costs(id) AS (
  SELECT id FROM account_cost_entries WHERE account_id IN (SELECT id FROM family)
  UNION
  SELECT child.id FROM account_cost_entries child JOIN doomed_costs parent ON child.supersedes_id=parent.id
), affected_settlements(id) AS (
  SELECT settlement_id FROM pool_settlement_account_costs
  WHERE account_id IN (SELECT id FROM family) OR cost_entry_id IN (SELECT id FROM doomed_costs)
  UNION
  SELECT settlement_id FROM pool_settlement_account_lines WHERE account_id IN (SELECT id FROM family)
), settlement_accounts(settlement_id,account_id) AS (
  SELECT settlement_id,account_id FROM pool_settlement_account_costs
  WHERE account_id NOT IN (SELECT id FROM family) AND cost_entry_id NOT IN (SELECT id FROM doomed_costs)
  UNION
  SELECT settlement_id,account_id FROM pool_settlement_account_lines
  WHERE account_id NOT IN (SELECT id FROM family)
), orphaned_purchase_sources(id) AS (
  SELECT source.id FROM purchase_sources source
  WHERE NOT EXISTS (
    SELECT 1 FROM account_cost_entries kept
    WHERE kept.purchase_source_id=source.id
      AND kept.id NOT IN (SELECT id FROM doomed_costs)
  )
)
SELECT
  (SELECT COUNT(*) FROM family),
  (SELECT COALESCE(SUM(jsonb_object_length(CASE WHEN jsonb_typeof(credentials)='object' THEN credentials ELSE '{}'::jsonb END)),0) FROM family),
  (SELECT COUNT(*) FROM scheduled_test_plans WHERE account_id IN (SELECT id FROM family))
    + (SELECT COUNT(*) FROM scheduler_outbox outbox WHERE outbox.account_id IN (SELECT id FROM family) OR EXISTS (
      SELECT 1 FROM jsonb_array_elements(CASE WHEN jsonb_typeof(outbox.payload->'account_ids')='array' THEN outbox.payload->'account_ids' ELSE '[]'::jsonb END) item
      WHERE item.value::text IN (SELECT id::text FROM family)
    )),
  (SELECT COUNT(*) FROM doomed_costs),
  (SELECT COUNT(*) FROM affected_settlements),
  (SELECT COUNT(*) FROM pool_settlement_account_costs
    WHERE account_id IN (SELECT id FROM family) OR cost_entry_id IN (SELECT id FROM doomed_costs)),
  (SELECT COUNT(*) FROM pool_settlement_account_lines WHERE account_id IN (SELECT id FROM family)),
  (SELECT COUNT(*) FROM affected_settlements affected WHERE EXISTS (
    SELECT 1 FROM settlement_accounts account
    WHERE account.settlement_id=affected.id
  )),
  (SELECT COUNT(*) FROM affected_settlements affected WHERE NOT EXISTS (
    SELECT 1 FROM settlement_accounts account
    WHERE account.settlement_id=affected.id
  )),
  (SELECT COUNT(*) FROM orphaned_purchase_sources),
  (SELECT COUNT(*) FROM account_groups WHERE account_id IN (SELECT id FROM family)),
  (SELECT COUNT(*) FROM account_lifecycle_events WHERE account_id IN (SELECT id FROM family) OR replacement_account_id IN (SELECT id FROM family)),
  (SELECT COUNT(*) FROM usage_logs WHERE account_id IN (SELECT id FROM family))`, accountID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return nil, service.ErrPoolAccountNotFound
	}
	impact := &service.PoolAccountDeleteImpact{}
	if err := rows.Scan(
		&impact.Accounts,
		&impact.CredentialKeys,
		&impact.SchedulingRecords,
		&impact.CostEntries,
		&impact.Settlements,
		&impact.SettlementAccountCosts,
		&impact.SettlementAccountLines,
		&impact.MixedSettlements,
		&impact.EmptySettlements,
		&impact.PurchaseSources,
		&impact.GroupLinks,
		&impact.LifecycleEvents,
		&impact.UsageRecords,
	); err != nil {
		return nil, err
	}
	if impact.Accounts == 0 {
		return nil, service.ErrPoolAccountNotFound
	}
	return impact, rows.Err()
}

func (r *poolApprovalRepository) UpdatePoolAccountApproved(ctx context.Context, id int64, input service.UpdatePoolAccountInput) error {
	result, err := r.executor(ctx).ExecContext(ctx, `
UPDATE accounts SET
    provider_identity=CASE WHEN $2 THEN NULLIF($3,'') ELSE provider_identity END,
    contributor_user_id=CASE WHEN $4 THEN NULLIF($5,0) ELSE contributor_user_id END,
    created_by_user_id=CASE WHEN $6 THEN NULLIF($7,0) ELSE created_by_user_id END,
    cost_sharing_enabled=CASE WHEN $8 THEN $9 ELSE cost_sharing_enabled END,
    updated_at=NOW()
WHERE id=$1 AND deleted_at IS NULL`, id,
		input.ProviderIdentity != nil, stringValue(input.ProviderIdentity),
		input.ContributorUserID != nil, int64Value(input.ContributorUserID),
		input.CreatedByUserID != nil, int64Value(input.CreatedByUserID),
		input.CostSharingEnabled != nil, boolValue(input.CostSharingEnabled))
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return service.ErrPoolAccountNotFound
	}
	return nil
}

func (r *poolApprovalRepository) UpdateCostApproved(ctx context.Context, update service.PoolCostUpdate) error {
	exec := r.executor(ctx)
	rows, err := exec.QueryContext(ctx, `
SELECT account_id,COALESCE(expected_token_count,0)
FROM account_cost_entries WHERE id=$1 FOR UPDATE`, update.CostID)
	if err != nil {
		return err
	}
	var accountID, oldExpected int64
	if !rows.Next() {
		_ = rows.Close()
		return service.ErrPoolAccountNotFound
	}
	if err = rows.Scan(&accountID, &oldExpected); err != nil {
		_ = rows.Close()
		return err
	}
	if err = rows.Close(); err != nil {
		return err
	}
	if accountID != update.Cost.AccountID {
		return service.ErrPoolAccountNotFound
	}

	rows, err = exec.QueryContext(ctx, `
SELECT EXISTS(
  SELECT 1 FROM pool_settlement_account_costs account_cost
  JOIN pool_settlements settlement ON settlement.id=account_cost.settlement_id
  WHERE settlement.status IN ('locked','paid') AND account_cost.cost_entry_id=$1
)`, update.CostID)
	if err != nil {
		return err
	}
	var settled bool
	if !rows.Next() || rows.Scan(&settled) != nil {
		_ = rows.Close()
		return fmt.Errorf("check cost settlement state")
	}
	if err = rows.Close(); err != nil {
		return err
	}
	if settled {
		return infraerrors.Conflict("POOL_COST_ALREADY_SETTLED", "locked or paid cost entries are immutable")
	}

	if update.Cost.OrderAccountKey != "" {
		rows, err = exec.QueryContext(ctx, `SELECT EXISTS(
  SELECT 1 FROM account_cost_entries
  WHERE id<>$1 AND (order_account_key=$2 OR (
    account_id=$3 AND COALESCE(purchase_source_id,0)=COALESCE($4::bigint,0)
    AND LOWER(BTRIM(order_no))=LOWER(BTRIM($5::text))
  )))`, update.CostID, update.Cost.OrderAccountKey, update.Cost.AccountID, update.Cost.PurchaseSourceID, update.Cost.OrderNo)
		if err != nil {
			return err
		}
		var duplicate bool
		if !rows.Next() || rows.Scan(&duplicate) != nil {
			_ = rows.Close()
			return fmt.Errorf("check duplicate cost order")
		}
		_ = rows.Close()
		if duplicate {
			return service.ErrPoolApprovalConflict
		}
	}
	if update.Cost.EntryType == "purchase" || update.Cost.EntryType == "renewal" || update.Cost.EntryType == "price_version" {
		rows, err = exec.QueryContext(ctx, `SELECT EXISTS(
  SELECT 1 FROM account_cost_entries
  WHERE id<>$1 AND account_id=$2 AND entry_type IN ('purchase','renewal','price_version')
    AND service_start<$4::date AND service_end>$3::date
)`, update.CostID, update.Cost.AccountID, update.Cost.ServiceStart, update.Cost.ServiceEnd)
		if err != nil {
			return err
		}
		var overlap bool
		if !rows.Next() || rows.Scan(&overlap) != nil {
			_ = rows.Close()
			return fmt.Errorf("check cost period overlap")
		}
		_ = rows.Close()
		if overlap {
			return service.ErrPoolApprovalConflict
		}
	}

	result, err := exec.ExecContext(ctx, `
UPDATE account_cost_entries SET
  payer_user_id=$2,purchase_source_id=$3,entry_type=$4,currency=$5,original_amount=$6,
  cny_amount_minor=$7,fx_rate=$8,service_start=$9,service_end=$10,warranty_end=$11,
  paid_at=$12,order_no=$13,purchase_url=$14,note=$15,related_account_id=$16,
  expected_token_count=$17,order_account_key=NULLIF(BTRIM($18),'')
WHERE id=$1`, update.CostID, update.Cost.PayerUserID, update.Cost.PurchaseSourceID, update.Cost.EntryType,
		update.Cost.Currency, update.Cost.OriginalAmount, update.Cost.CNYAmountMinor, update.Cost.FXRate,
		update.Cost.ServiceStart, update.Cost.ServiceEnd, update.Cost.WarrantyEnd, update.Cost.PaidAt,
		update.Cost.OrderNo, update.Cost.PurchaseURL, update.Cost.Note, update.Cost.RelatedAccountID,
		update.Cost.ExpectedTokenCount, update.Cost.OrderAccountKey)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return service.ErrPoolAccountNotFound
	}
	newExpected := int64(0)
	if update.Cost.ExpectedTokenCount != nil {
		newExpected = *update.Cost.ExpectedTokenCount
	}
	result, err = exec.ExecContext(ctx, `
UPDATE accounts SET
  expected_token_count=CASE WHEN COALESCE(expected_token_count,0)+$2>0 THEN COALESCE(expected_token_count,0)+$2 ELSE NULL END,
  updated_at=NOW()
WHERE id=$1 AND deleted_at IS NULL`, accountID, newExpected-oldExpected)
	if err != nil {
		return err
	}
	affected, err = result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return service.ErrPoolAccountNotFound
	}
	return nil
}

func (r *poolApprovalRepository) InvalidateCredentialApprovals(ctx context.Context, accountID int64, reason string) error {
	_, err := r.executor(ctx).ExecContext(ctx, `
UPDATE pool_approval_requests
SET status='expired', decision_reason=$2, decided_at=COALESCE(decided_at,NOW()),
    reveal_expires_at=NULL, payload='{}'::jsonb, updated_at=NOW()
WHERE account_id=$1 AND action_type='VIEW_CREDENTIAL' AND status IN ('pending','approved')`, accountID, reason)
	return err
}

func (r *poolApprovalRepository) SetDecision(ctx context.Context, id int64, status string, actorID int64, reason *string, revealExpiresAt *time.Time) error {
	result, err := r.executor(ctx).ExecContext(ctx, `
UPDATE pool_approval_requests
SET status=$2,decided_by_user_id=$3,decision_reason=$4,decided_at=NOW(),
    reveal_expires_at=$5,payload='{}'::jsonb,updated_at=NOW()
WHERE id=$1 AND status='pending'`, id, status, actorID, reason, revealExpiresAt)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return service.ErrPoolApprovalConflict
	}
	return nil
}

func (r *poolApprovalRepository) MarkExpired(ctx context.Context, id int64, reason string) error {
	result, err := r.executor(ctx).ExecContext(ctx, `
UPDATE pool_approval_requests
SET status='expired',decision_reason=$2,decided_at=COALESCE(decided_at,NOW()),payload='{}'::jsonb,updated_at=NOW()
WHERE id=$1 AND status='pending'`, id, reason)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return service.ErrPoolApprovalConflict
	}
	return nil
}

func (r *poolApprovalRepository) LoadCredentials(ctx context.Context, accountID int64) (map[string]any, error) {
	rows, err := r.executor(ctx).QueryContext(ctx, `SELECT credentials FROM accounts WHERE id=$1 AND deleted_at IS NULL`, accountID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return nil, service.ErrPoolAccountNotFound
	}
	var raw []byte
	if err := rows.Scan(&raw); err != nil {
		return nil, err
	}
	out := make(map[string]any)
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			return nil, err
		}
	}
	return out, rows.Err()
}

func (r *poolApprovalRepository) ConsumeReveal(ctx context.Context, id int64, revealedAt time.Time) error {
	result, err := r.executor(ctx).ExecContext(ctx, `
UPDATE pool_approval_requests
SET status='consumed',revealed_at=$2,updated_at=$2
WHERE id=$1 AND action_type='VIEW_CREDENTIAL' AND status='approved'
  AND revealed_at IS NULL AND reveal_expires_at > $2`, id, revealedAt.UTC())
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return service.ErrPoolApprovalConflict
	}
	return nil
}
