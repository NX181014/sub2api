package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
	"github.com/shopspring/decimal"
)

type poolRepository struct{ db *sql.DB }

func NewPoolRepository(db *sql.DB) service.PoolRepository { return &poolRepository{db: db} }

const poolAccountSelect = `
SELECT a.id, a.name, a.platform, a.provider_identity,
       a.contributor_user_id, contributor.email,
       a.created_by_user_id, creator.email,
       a.cost_sharing_enabled,
       COALESCE(lifecycle.event_type, 'active'), lifecycle.occurred_at,
       COALESCE(costs.net_cost_minor, 0)
FROM accounts a
LEFT JOIN users contributor ON contributor.id = a.contributor_user_id
LEFT JOIN users creator ON creator.id = a.created_by_user_id
	LEFT JOIN LATERAL (
	    SELECT event_type, occurred_at
	    FROM account_lifecycle_events e
	    WHERE e.account_id = a.id AND e.event_type <> 'refund'
    ORDER BY e.occurred_at DESC, e.id DESC
    LIMIT 1
) lifecycle ON TRUE
LEFT JOIN LATERAL (
    SELECT SUM(cny_amount_minor)::bigint AS net_cost_minor
    FROM account_cost_entries c
    WHERE c.account_id = a.id
) costs ON TRUE
WHERE a.deleted_at IS NULL`

func scanPoolAccount(scanner interface{ Scan(...any) error }) (*service.PoolAccount, error) {
	var item service.PoolAccount
	err := scanner.Scan(&item.ID, &item.Name, &item.Platform, &item.ProviderIdentity,
		&item.ContributorUserID, &item.ContributorEmail, &item.CreatedByUserID, &item.CreatedByEmail,
		&item.CostSharingEnabled, &item.LatestLifecycleStatus, &item.LatestLifecycleAt, &item.NetCostMinor)
	return &item, err
}

func (r *poolRepository) ListAccounts(ctx context.Context) ([]service.PoolAccount, error) {
	rows, err := r.db.QueryContext(ctx, poolAccountSelect+` ORDER BY a.cost_sharing_enabled DESC, a.id DESC`)
	if err != nil {
		return nil, fmt.Errorf("list pool accounts: %w", err)
	}
	defer rows.Close()
	items := make([]service.PoolAccount, 0)
	for rows.Next() {
		item, scanErr := scanPoolAccount(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan pool account: %w", scanErr)
		}
		items = append(items, *item)
	}
	return items, rows.Err()
}

func (r *poolRepository) UpdateAccount(ctx context.Context, id int64, input service.UpdatePoolAccountInput) (*service.PoolAccount, error) {
	result, err := r.db.ExecContext(ctx, `
UPDATE accounts SET
    provider_identity = CASE WHEN $2 THEN NULLIF($3, '') ELSE provider_identity END,
    contributor_user_id = CASE WHEN $4 THEN NULLIF($5, 0) ELSE contributor_user_id END,
    created_by_user_id = CASE WHEN $6 THEN NULLIF($7, 0) ELSE created_by_user_id END,
    cost_sharing_enabled = CASE WHEN $8 THEN $9 ELSE cost_sharing_enabled END,
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL`, id,
		input.ProviderIdentity != nil, stringValue(input.ProviderIdentity),
		input.ContributorUserID != nil, int64Value(input.ContributorUserID),
		input.CreatedByUserID != nil, int64Value(input.CreatedByUserID),
		input.CostSharingEnabled != nil, boolValue(input.CostSharingEnabled))
	if err != nil {
		return nil, fmt.Errorf("update pool account: %w", err)
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return nil, service.ErrPoolAccountNotFound
	}
	return scanPoolAccount(r.db.QueryRowContext(ctx, poolAccountSelect+` AND a.id = $1`, id))
}

func stringValue(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
func int64Value(v *int64) int64 {
	if v == nil {
		return 0
	}
	return *v
}
func boolValue(v *bool) bool { return v != nil && *v }

func (r *poolRepository) ListSources(ctx context.Context) ([]service.PurchaseSource, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, name, website_url, notes, active, created_at, updated_at FROM purchase_sources ORDER BY active DESC, name ASC`)
	if err != nil {
		return nil, fmt.Errorf("list purchase sources: %w", err)
	}
	defer rows.Close()
	items := make([]service.PurchaseSource, 0)
	for rows.Next() {
		var item service.PurchaseSource
		if err := rows.Scan(&item.ID, &item.Name, &item.WebsiteURL, &item.Notes, &item.Active, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *poolRepository) CreateSource(ctx context.Context, input service.CreatePurchaseSourceInput) (*service.PurchaseSource, error) {
	var item service.PurchaseSource
	err := r.db.QueryRowContext(ctx, `
INSERT INTO purchase_sources(name, website_url, notes)
VALUES ($1, $2, $3)
RETURNING id, name, website_url, notes, active, created_at, updated_at`, input.Name, input.WebsiteURL, input.Notes).
		Scan(&item.ID, &item.Name, &item.WebsiteURL, &item.Notes, &item.Active, &item.CreatedAt, &item.UpdatedAt)
	if isPoolUniqueViolation(err) {
		return nil, infraerrors.Conflict("PURCHASE_SOURCE_EXISTS", "purchase source already exists")
	}
	if err != nil {
		return nil, fmt.Errorf("create purchase source: %w", err)
	}
	return &item, nil
}

func isPoolUniqueViolation(err error) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && pqErr.Code == "23505"
}

const costSelect = `
SELECT c.id, c.account_id, a.name, c.payer_user_id, u.email,
       c.purchase_source_id, ps.name, c.entry_type, c.currency,
       c.original_amount, c.cny_amount_minor, c.fx_rate,
       c.service_start, c.service_end, c.warranty_end, c.paid_at,
       c.order_no, c.purchase_url, c.note, c.supersedes_id,
       c.related_account_id, c.expected_token_count, c.created_by_user_id, c.created_at
FROM account_cost_entries c
JOIN accounts a ON a.id = c.account_id
JOIN users u ON u.id = c.payer_user_id
LEFT JOIN purchase_sources ps ON ps.id = c.purchase_source_id`

func scanCost(scanner interface{ Scan(...any) error }) (*service.AccountCostEntry, error) {
	var item service.AccountCostEntry
	err := scanner.Scan(&item.ID, &item.AccountID, &item.AccountName, &item.PayerUserID, &item.PayerEmail,
		&item.PurchaseSourceID, &item.PurchaseSource, &item.EntryType, &item.Currency,
		&item.OriginalAmount, &item.CNYAmountMinor, &item.FXRate, &item.ServiceStart, &item.ServiceEnd,
		&item.WarrantyEnd, &item.PaidAt, &item.OrderNo, &item.PurchaseURL, &item.Note, &item.SupersedesID,
		&item.RelatedAccountID, &item.ExpectedTokenCount, &item.CreatedByUserID, &item.CreatedAt)
	return &item, err
}

func (r *poolRepository) ListCosts(ctx context.Context, accountID *int64) ([]service.AccountCostEntry, error) {
	query := costSelect
	args := []any{}
	if accountID != nil {
		query += ` WHERE c.account_id = $1`
		args = append(args, *accountID)
	}
	query += ` ORDER BY c.paid_at DESC, c.id DESC`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list account costs: %w", err)
	}
	defer rows.Close()
	items := make([]service.AccountCostEntry, 0)
	for rows.Next() {
		item, scanErr := scanCost(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, *item)
	}
	return items, rows.Err()
}

func (r *poolRepository) ListCostEntries(ctx context.Context, filter service.AccountCostEntryFilter, limit, offset int) ([]service.AccountCostEntry, int64, error) {
	conditions := make([]string, 0, 8)
	args := make([]any, 0, 8)
	arg := func(value any) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}
	if filter.Search != "" {
		p := arg("%" + filter.Search + "%")
		conditions = append(conditions, `(a.name ILIKE `+p+` OR COALESCE(a.provider_identity,'') ILIKE `+p+` OR u.email ILIKE `+p+` OR COALESCE(ps.name,'') ILIKE `+p+` OR COALESCE(c.order_no,'') ILIKE `+p+`)`)
	}
	if filter.AccountID != nil {
		conditions = append(conditions, "c.account_id="+arg(*filter.AccountID))
	}
	if filter.UploaderUserID != nil {
		conditions = append(conditions, "a.created_by_user_id="+arg(*filter.UploaderUserID))
	}
	if filter.PayerUserID != nil {
		conditions = append(conditions, "c.payer_user_id="+arg(*filter.PayerUserID))
	}
	if filter.PurchaseSourceID != nil {
		conditions = append(conditions, "c.purchase_source_id="+arg(*filter.PurchaseSourceID))
	}
	if filter.EntryType != "" {
		conditions = append(conditions, "c.entry_type="+arg(filter.EntryType))
	}
	if filter.PaidFrom != nil {
		conditions = append(conditions, "c.paid_at>="+arg(*filter.PaidFrom))
	}
	if filter.PaidTo != nil {
		conditions = append(conditions, "c.paid_at<"+arg(*filter.PaidTo))
	}
	where := ""
	if len(conditions) > 0 {
		where = " WHERE " + strings.Join(conditions, " AND ")
	}
	var total int64
	countQuery := `SELECT COUNT(*) FROM account_cost_entries c JOIN accounts a ON a.id=c.account_id JOIN users u ON u.id=c.payer_user_id LEFT JOIN purchase_sources ps ON ps.id=c.purchase_source_id` + where
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count account cost entries: %w", err)
	}
	queryArgs := append(append([]any{}, args...), limit, offset)
	query := costSelect + where + fmt.Sprintf(" ORDER BY c.paid_at DESC,c.id DESC LIMIT $%d OFFSET $%d", len(args)+1, len(args)+2)
	rows, err := r.db.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list account cost entries: %w", err)
	}
	defer rows.Close()
	items := make([]service.AccountCostEntry, 0)
	for rows.Next() {
		item, scanErr := scanCost(rows)
		if scanErr != nil {
			return nil, 0, fmt.Errorf("scan account cost entry: %w", scanErr)
		}
		items = append(items, *item)
	}
	return items, total, rows.Err()
}

func (r *poolRepository) CreateCost(ctx context.Context, input service.CreateAccountCostInput) (*service.AccountCostEntry, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err = tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, fmt.Sprintf("pool-cost:%d", input.AccountID)); err != nil {
		return nil, err
	}
	var accountExists bool
	if err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM accounts WHERE id=$1 AND deleted_at IS NULL)`, input.AccountID).Scan(&accountExists); err != nil {
		return nil, fmt.Errorf("validate cost account: %w", err)
	}
	if !accountExists {
		return nil, service.ErrPoolAccountNotFound
	}
	if input.OrderAccountKey != "" {
		var conflictID int64
		err = tx.QueryRowContext(ctx, `
SELECT id FROM account_cost_entries WHERE
  order_account_key=$1 OR (account_id=$2 AND COALESCE(purchase_source_id,0)=COALESCE($3::bigint,0)
  AND LOWER(BTRIM(order_no))=LOWER(BTRIM($4::text)))
LIMIT 1`, input.OrderAccountKey, input.AccountID, input.PurchaseSourceID, input.OrderNo).Scan(&conflictID)
		if err == nil {
			return nil, infraerrors.Conflict("DUPLICATE_ORDER_ACCOUNT", fmt.Sprintf("order already exists for account %d in cost entry %d", input.AccountID, conflictID))
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("check duplicate order account: %w", err)
		}
	}
	if input.EntryType == "purchase" || input.EntryType == "renewal" || input.EntryType == "price_version" {
		var conflictID int64
		err = tx.QueryRowContext(ctx, `
SELECT id FROM account_cost_entries
WHERE account_id=$1 AND entry_type IN ('purchase','renewal','price_version')
  AND service_start < $3::date AND service_end > $2::date
LIMIT 1`, input.AccountID, input.ServiceStart, input.ServiceEnd).Scan(&conflictID)
		if err == nil {
			return nil, infraerrors.Conflict("COST_PERIOD_OVERLAP", fmt.Sprintf("service period overlaps cost entry %d", conflictID))
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
	}
	row := tx.QueryRowContext(ctx, `
INSERT INTO account_cost_entries(
    account_id, payer_user_id, purchase_source_id, entry_type, currency,
    original_amount, cny_amount_minor, fx_rate, service_start, service_end, warranty_end,
	    paid_at, order_no, purchase_url, note, supersedes_id, related_account_id, expected_token_count,
	    created_by_user_id, operation_key, order_account_key)
	VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,NULLIF(BTRIM($20),''),NULLIF(BTRIM($21),''))
	ON CONFLICT (created_by_user_id, operation_key) WHERE operation_key IS NOT NULL
	DO NOTHING
	RETURNING id`, input.AccountID, input.PayerUserID, input.PurchaseSourceID, input.EntryType,
		input.Currency, input.OriginalAmount, input.CNYAmountMinor, input.FXRate,
		input.ServiceStart, input.ServiceEnd, input.WarrantyEnd, input.PaidAt, input.OrderNo, input.PurchaseURL,
		input.Note, input.SupersedesID, input.RelatedAccountID, input.ExpectedTokenCount, input.CreatedByUserID, input.OperationKey, input.OrderAccountKey)
	var id int64
	if err := row.Scan(&id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, infraerrors.Conflict("DUPLICATE_COST_OPERATION", "cost operation was already submitted")
		}
		if isPoolUniqueViolation(err) {
			return nil, infraerrors.Conflict("DUPLICATE_ORDER_ACCOUNT", "order already exists for this account")
		}
		return nil, fmt.Errorf("create account cost: %w", err)
	}
	if input.ExpectedTokenCount != nil {
		if _, err = tx.ExecContext(ctx, `UPDATE accounts SET expected_token_count=COALESCE(expected_token_count,0)+$2,updated_at=NOW() WHERE id=$1 AND deleted_at IS NULL`, input.AccountID, *input.ExpectedTokenCount); err != nil {
			return nil, fmt.Errorf("update expected token count: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return scanCost(r.db.QueryRowContext(ctx, costSelect+` WHERE c.id = $1`, id))
}

func (r *poolRepository) ListCostSummaries(ctx context.Context, filter service.AccountCostSummaryFilter, limit, offset int) ([]service.AccountCostSummary, int64, error) {
	where, args := buildCostSummaryWhere(filter)
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM accounts a WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count account cost summaries: %w", err)
	}
	queryArgs := append(append([]any{}, args...), limit, offset)
	limitArg, offsetArg := len(args)+1, len(args)+2
	rows, err := r.db.QueryContext(ctx, fmt.Sprintf(`
WITH filtered AS (
    SELECT a.id,a.name,a.provider_identity,a.status,a.created_by_user_id,a.contributor_user_id,a.expected_token_count
    FROM accounts a
    WHERE %s
    ORDER BY a.id DESC
    LIMIT $%d OFFSET $%d
)
SELECT a.id,a.name,a.provider_identity,a.status,
       a.created_by_user_id,uploader.email,a.contributor_user_id,contributor.email,a.expected_token_count,
       COALESCE(usage.total_tokens,0)::bigint,
       COALESCE(costs.purchase_cost_minor,0)::bigint,COALESCE(costs.refund_minor,0)::bigint,
       COALESCE(costs.transferred_out_minor,0)::bigint,
       COALESCE(costs.written_off_minor,0)::bigint,
       COALESCE(costs.cost_basis_minor,0)::bigint,
       COALESCE(costs.net_cost_minor,0)::bigint,COALESCE(costs.tranches,'[]'::jsonb),COALESCE(costs.entry_count,0)::bigint,
       COALESCE(lifecycle.event_type,'active'),lifecycle.occurred_at,
       latest.payer_user_id,payer.email,latest.purchase_source_id,source.name,latest.order_no,
       latest.service_start,latest.service_end,costs.purchased_at
FROM filtered a
LEFT JOIN users uploader ON uploader.id=a.created_by_user_id
LEFT JOIN users contributor ON contributor.id=a.contributor_user_id
LEFT JOIN LATERAL (
    SELECT COALESCE(SUM(c.cny_amount_minor) FILTER (WHERE c.cny_amount_minor>0),0)::bigint purchase_cost_minor,
           COALESCE(-SUM(c.cny_amount_minor) FILTER (WHERE c.entry_type='refund' AND c.cny_amount_minor<0),0)::bigint refund_minor,
           COALESCE(-SUM(c.cny_amount_minor) FILTER (WHERE c.entry_type='replacement_out' AND c.cny_amount_minor<0),0)::bigint transferred_out_minor,
           COALESCE(-SUM(c.cny_amount_minor) FILTER (WHERE c.entry_type='write_off' AND c.cny_amount_minor<0),0)::bigint written_off_minor,
           COALESCE(SUM(c.cny_amount_minor) FILTER (WHERE c.entry_type NOT IN ('refund','replacement_out','write_off')),0)::bigint cost_basis_minor,
           COALESCE(SUM(c.cny_amount_minor) FILTER (WHERE c.entry_type<>'write_off'),0)::bigint net_cost_minor,
	       COALESCE((SELECT jsonb_agg(jsonb_build_object(
	         'id',priced.id,'cost_minor',priced.cny_amount_minor,'expected_tokens',priced.expected_token_count,
	         'paid_at',priced.paid_at,'payer_user_id',priced.payer_user_id,'purchase_source_id',priced.purchase_source_id,
	         'service_start',priced.service_start,'service_end',priced.service_end,
	         'usage_tokens',COALESCE((SELECT SUM(ul.input_tokens::bigint+ul.output_tokens::bigint+ul.cache_creation_tokens::bigint+
	           ul.cache_read_tokens::bigint+ul.image_output_tokens::bigint+ul.image_input_tokens::bigint)
	           FROM usage_logs ul WHERE ul.account_id=a.id AND ul.created_at>=priced.paid_at
	             AND (priced.next_paid_at IS NULL OR ul.created_at<priced.next_paid_at)),0)) ORDER BY priced.paid_at,priced.id)
	         FROM (SELECT c2.*,LEAD(c2.paid_at) OVER (ORDER BY c2.paid_at,c2.id) next_paid_at
	               FROM account_cost_entries c2 WHERE c2.account_id=a.id AND c2.cny_amount_minor>0 AND c2.expected_token_count>0) priced),'[]'::jsonb) tranches,
           COUNT(*)::bigint entry_count,
           MIN(c.paid_at) FILTER (WHERE c.entry_type IN ('purchase','renewal','topup','price_version')) purchased_at
    FROM account_cost_entries c WHERE c.account_id=a.id
) costs ON TRUE
LEFT JOIN LATERAL (
    SELECT c.payer_user_id,c.purchase_source_id,c.order_no,c.service_start,c.service_end
    FROM account_cost_entries c
    WHERE c.account_id=a.id AND c.entry_type IN ('purchase','renewal','topup','price_version')
    ORDER BY c.paid_at DESC,c.id DESC LIMIT 1
) latest ON TRUE
LEFT JOIN users payer ON payer.id=latest.payer_user_id
LEFT JOIN purchase_sources source ON source.id=latest.purchase_source_id
LEFT JOIN LATERAL (
    SELECT e.event_type,e.occurred_at FROM account_lifecycle_events e
    WHERE e.account_id=a.id AND e.event_type<>'refund'
    ORDER BY e.occurred_at DESC,e.id DESC LIMIT 1
) lifecycle ON TRUE
LEFT JOIN LATERAL (
    SELECT COALESCE(SUM(ul.input_tokens::bigint+ul.output_tokens::bigint+ul.cache_creation_tokens::bigint+
           ul.cache_read_tokens::bigint+ul.image_output_tokens::bigint+ul.image_input_tokens::bigint),0)::bigint total_tokens
    FROM usage_logs ul WHERE ul.account_id=a.id
) usage ON TRUE
ORDER BY a.id DESC`, where, limitArg, offsetArg), queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list account cost summaries: %w", err)
	}
	defer rows.Close()
	items := make([]service.AccountCostSummary, 0)
	for rows.Next() {
		var item service.AccountCostSummary
		var trancheJSON []byte
		if err := rows.Scan(
			&item.AccountID, &item.AccountName, &item.ProviderIdentity, &item.AccountStatus,
			&item.UploaderUserID, &item.UploaderEmail, &item.ContributorUserID, &item.ContributorEmail, &item.ExpectedTokenCount,
			&item.TotalUsageTokens, &item.PurchaseCostMinor, &item.RefundMinor, &item.TransferredOutMinor, &item.WrittenOffMinor, &item.CostBasisMinor, &item.NetCostMinor, &trancheJSON, &item.EntryCount,
			&item.LatestLifecycleStatus, &item.LatestLifecycleAt, &item.LatestPayerUserID, &item.LatestPayerEmail,
			&item.LatestPurchaseSourceID, &item.LatestPurchaseSource, &item.LatestOrderNo,
			&item.LatestServiceStart, &item.LatestServiceEnd, &item.PurchasedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan account cost summary: %w", err)
		}
		item.CostTranches, err = decodePoolCostTranches(trancheJSON)
		if err != nil {
			return nil, 0, fmt.Errorf("decode account cost tranches: %w", err)
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func buildCostSummaryWhere(filter service.AccountCostSummaryFilter) (string, []any) {
	conditions := []string{"(a.cost_sharing_enabled=TRUE OR EXISTS (SELECT 1 FROM account_cost_entries pc WHERE pc.account_id=a.id))"}
	args := make([]any, 0)
	arg := func(value any) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}
	if filter.Search != "" {
		p := arg("%" + filter.Search + "%")
		conditions = append(conditions, `(a.name ILIKE `+p+` OR COALESCE(a.provider_identity,'') ILIKE `+p+` OR EXISTS (SELECT 1 FROM users su WHERE su.id=a.created_by_user_id AND su.email ILIKE `+p+`))`)
	}
	if filter.UploaderUserID != nil {
		conditions = append(conditions, "a.created_by_user_id="+arg(*filter.UploaderUserID))
	}
	costConditions := make([]string, 0, 3)
	if filter.PayerUserID != nil {
		costConditions = append(costConditions, "fc.payer_user_id="+arg(*filter.PayerUserID))
	}
	if filter.PurchaseSourceID != nil {
		costConditions = append(costConditions, "fc.purchase_source_id="+arg(*filter.PurchaseSourceID))
	}
	if filter.EntryType != "" {
		costConditions = append(costConditions, "fc.entry_type="+arg(filter.EntryType))
	}
	if len(costConditions) > 0 {
		conditions = append(conditions, "EXISTS (SELECT 1 FROM account_cost_entries fc WHERE fc.account_id=a.id AND "+strings.Join(costConditions, " AND ")+")")
	}
	if filter.LifecycleStatus != "" {
		conditions = append(conditions, `COALESCE((SELECT e.event_type FROM account_lifecycle_events e WHERE e.account_id=a.id AND e.event_type<>'refund' ORDER BY e.occurred_at DESC,e.id DESC LIMIT 1),'active')=`+arg(filter.LifecycleStatus))
	}
	if filter.HasCost != nil {
		prefix := ""
		if !*filter.HasCost {
			prefix = "NOT "
		}
		conditions = append(conditions, prefix+"EXISTS (SELECT 1 FROM account_cost_entries hc WHERE hc.account_id=a.id)")
	}
	return strings.Join(conditions, " AND "), args
}

func (r *poolRepository) CreateCostsBatch(ctx context.Context, inputs []service.CreateAccountCostInput) ([]service.AccountCostEntry, error) {
	if len(inputs) == 0 {
		return []service.AccountCostEntry{}, nil
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin account cost batch: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err = tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, "pool-cost-batch:"+inputs[0].OperationKey); err != nil {
		return nil, fmt.Errorf("lock account cost batch: %w", err)
	}
	accountIDs := make([]int64, len(inputs))
	for i := range inputs {
		accountIDs[i] = inputs[i].AccountID
	}
	sort.Slice(accountIDs, func(i, j int) bool { return accountIDs[i] < accountIDs[j] })
	for _, accountID := range accountIDs {
		if _, err = tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, fmt.Sprintf("pool-cost:%d", accountID)); err != nil {
			return nil, fmt.Errorf("lock account cost: %w", err)
		}
	}
	var existingAccounts int
	if err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM accounts WHERE id=ANY($1) AND deleted_at IS NULL`, pq.Array(accountIDs)).Scan(&existingAccounts); err != nil {
		return nil, fmt.Errorf("validate batch accounts: %w", err)
	}
	if existingAccounts != len(accountIDs) {
		return nil, service.ErrPoolAccountNotFound
	}

	ids := make([]int64, 0, len(inputs))
	for _, input := range inputs {
		if input.OrderAccountKey != "" {
			var conflictID int64
			err = tx.QueryRowContext(ctx, `SELECT id FROM account_cost_entries WHERE (
order_account_key=$1 OR (account_id=$2 AND COALESCE(purchase_source_id,0)=COALESCE($3::bigint,0)
AND LOWER(BTRIM(order_no))=LOWER(BTRIM($4::text)))) AND operation_key IS DISTINCT FROM $5 LIMIT 1`,
				input.OrderAccountKey, input.AccountID, input.PurchaseSourceID, input.OrderNo, input.OperationKey).Scan(&conflictID)
			if err == nil {
				return nil, infraerrors.Conflict("DUPLICATE_ORDER_ACCOUNT", fmt.Sprintf("order already exists for account %d in cost entry %d", input.AccountID, conflictID))
			}
			if !errors.Is(err, sql.ErrNoRows) {
				return nil, fmt.Errorf("check duplicate order account: %w", err)
			}
		}
		if input.EntryType == "purchase" || input.EntryType == "renewal" || input.EntryType == "price_version" {
			var conflictID int64
			err = tx.QueryRowContext(ctx, `SELECT id FROM account_cost_entries
WHERE account_id=$1 AND entry_type IN ('purchase','renewal','price_version')
  AND service_start<$3::date AND service_end>$2::date
  AND operation_key IS DISTINCT FROM $4 LIMIT 1`, input.AccountID, input.ServiceStart, input.ServiceEnd, input.OperationKey).Scan(&conflictID)
			if err == nil {
				return nil, infraerrors.Conflict("COST_PERIOD_OVERLAP", fmt.Sprintf("service period overlaps cost entry %d", conflictID))
			}
			if !errors.Is(err, sql.ErrNoRows) {
				return nil, fmt.Errorf("check batch cost period: %w", err)
			}
		}
		var id int64
		err = tx.QueryRowContext(ctx, `
INSERT INTO account_cost_entries(
	    account_id,payer_user_id,purchase_source_id,entry_type,currency,original_amount,cny_amount_minor,fx_rate,
	    service_start,service_end,warranty_end,paid_at,order_no,purchase_url,note,supersedes_id,related_account_id,
	    expected_token_count,created_by_user_id,operation_key,order_account_key)
	VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,NULLIF(BTRIM($20),''),NULLIF(BTRIM($21),''))
	ON CONFLICT (created_by_user_id,operation_key) WHERE operation_key IS NOT NULL
	DO NOTHING
	RETURNING id`, input.AccountID, input.PayerUserID, input.PurchaseSourceID, input.EntryType, input.Currency,
			input.OriginalAmount, input.CNYAmountMinor, input.FXRate, input.ServiceStart, input.ServiceEnd,
			input.WarrantyEnd, input.PaidAt, input.OrderNo, input.PurchaseURL, input.Note, input.SupersedesID,
			input.RelatedAccountID, input.ExpectedTokenCount, input.CreatedByUserID, input.OperationKey, input.OrderAccountKey).Scan(&id)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, infraerrors.Conflict("DUPLICATE_COST_OPERATION", "cost batch was already submitted")
		}
		if isPoolUniqueViolation(err) {
			return nil, infraerrors.Conflict("DUPLICATE_ORDER_ACCOUNT", "order already exists for this account")
		}
		if err != nil {
			return nil, fmt.Errorf("create batch account cost: %w", err)
		}
		ids = append(ids, id)
	}
	for _, input := range inputs {
		if input.ExpectedTokenCount == nil {
			continue
		}
		if _, err = tx.ExecContext(ctx, `UPDATE accounts SET expected_token_count=COALESCE(expected_token_count,0)+$2,updated_at=NOW() WHERE id=$1 AND deleted_at IS NULL`, input.AccountID, *input.ExpectedTokenCount); err != nil {
			return nil, fmt.Errorf("update expected token count: %w", err)
		}
	}
	items := make([]service.AccountCostEntry, 0, len(ids))
	for _, id := range ids {
		item, scanErr := scanCost(tx.QueryRowContext(ctx, costSelect+` WHERE c.id=$1`, id))
		if scanErr != nil {
			return nil, fmt.Errorf("load batch account cost: %w", scanErr)
		}
		items = append(items, *item)
	}
	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit account cost batch: %w", err)
	}
	return items, nil
}

const lifecycleSelect = `
SELECT e.id, e.account_id, a.name, e.event_type, e.occurred_at, e.reason,
	   e.replacement_account_id, e.transferred_cost_minor,
	   e.source, e.created_by_user_id, e.created_at
FROM account_lifecycle_events e
JOIN accounts a ON a.id = e.account_id`

func scanLifecycle(scanner interface{ Scan(...any) error }) (*service.AccountLifecycleEvent, error) {
	var item service.AccountLifecycleEvent
	err := scanner.Scan(&item.ID, &item.AccountID, &item.AccountName, &item.EventType, &item.OccurredAt,
		&item.Reason, &item.ReplacementAccountID, &item.TransferredCostMinor, &item.Source, &item.CreatedByUserID, &item.CreatedAt)
	return &item, err
}

func (r *poolRepository) ListLifecycle(ctx context.Context, accountID *int64) ([]service.AccountLifecycleEvent, error) {
	query := lifecycleSelect
	args := []any{}
	if accountID != nil {
		query += ` WHERE e.account_id = $1`
		args = append(args, *accountID)
	}
	query += ` ORDER BY e.occurred_at DESC, e.id DESC`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list lifecycle events: %w", err)
	}
	defer rows.Close()
	items := make([]service.AccountLifecycleEvent, 0)
	for rows.Next() {
		item, scanErr := scanLifecycle(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, *item)
	}
	return items, rows.Err()
}

func (r *poolRepository) CreateLifecycle(ctx context.Context, input service.CreateLifecycleEventInput) (*service.AccountLifecycleEvent, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	var id int64
	err = tx.QueryRowContext(ctx, `
INSERT INTO account_lifecycle_events(account_id, event_type, occurred_at, reason, replacement_account_id, transferred_cost_minor, created_by_user_id)
VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING id`, input.AccountID, input.EventType, input.OccurredAt,
		input.Reason, input.ReplacementAccountID, input.TransferredCostMinor, input.CreatedByUserID).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("create lifecycle event: %w", err)
	}
	if input.EventType == "replaced" {
		_, err = tx.ExecContext(ctx, `
UPDATE accounts target SET
  cost_sharing_enabled=TRUE,
  contributor_user_id=COALESCE(target.contributor_user_id, source.contributor_user_id),
  created_by_user_id=COALESCE(target.created_by_user_id, source.created_by_user_id, $3),
  updated_at=NOW()
FROM accounts source
WHERE target.id=$2 AND source.id=$1`, input.AccountID, *input.ReplacementAccountID, input.CreatedByUserID)
		if err != nil {
			return nil, fmt.Errorf("activate replacement pool account: %w", err)
		}
	}
	if input.EventType == "replaced" && input.TransferredCostMinor > 0 {
		loc, _ := time.LoadLocation(service.PoolTimezone)
		occurred := input.OccurredAt.In(loc)
		day := time.Date(occurred.Year(), occurred.Month(), occurred.Day(), 0, 0, 0, 0, loc)
		common := []any{*input.PayerUserID, day, day.AddDate(0, 0, 1), input.OccurredAt, input.CreatedByUserID}
		_, err = tx.ExecContext(ctx, `
INSERT INTO account_cost_entries(account_id,payer_user_id,entry_type,currency,original_amount,cny_amount_minor,fx_rate,service_start,service_end,paid_at,note,related_account_id,created_by_user_id)
VALUES ($1,$2,'replacement_out','CNY',$3,$4,'1',$5,$6,$7,'replacement cost transfer',$8,$9),
       ($8,$2,'replacement_in','CNY',$10,$11,'1',$5,$6,$7,'replacement cost transfer',$1,$9)`,
			input.AccountID, common[0], decimal.NewFromInt(-input.TransferredCostMinor).Div(decimal.NewFromInt(100)).StringFixed(2), -input.TransferredCostMinor,
			common[1], common[2], common[3], *input.ReplacementAccountID, common[4],
			decimal.NewFromInt(input.TransferredCostMinor).Div(decimal.NewFromInt(100)).StringFixed(2), input.TransferredCostMinor)
		if err != nil {
			return nil, fmt.Errorf("create replacement cost transfer: %w", err)
		}
	}
	if input.EventType == "refund" && input.RefundAmountMinor > 0 {
		loc, _ := time.LoadLocation(service.PoolTimezone)
		occurred := input.OccurredAt.In(loc)
		day := time.Date(occurred.Year(), occurred.Month(), occurred.Day(), 0, 0, 0, 0, loc)
		_, err = tx.ExecContext(ctx, `
INSERT INTO account_cost_entries(account_id,payer_user_id,entry_type,currency,original_amount,cny_amount_minor,fx_rate,service_start,service_end,paid_at,note,created_by_user_id)
VALUES ($1,$2,'refund','CNY',$3,$4,'1',$5,$6,$7,'lifecycle refund',$8)`,
			input.AccountID, *input.PayerUserID, decimal.NewFromInt(-input.RefundAmountMinor).Div(decimal.NewFromInt(100)).StringFixed(2),
			-input.RefundAmountMinor, day, day.AddDate(0, 0, 1), input.OccurredAt, input.CreatedByUserID)
		if err != nil {
			return nil, fmt.Errorf("create lifecycle refund: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return scanLifecycle(r.db.QueryRowContext(ctx, lifecycleSelect+` WHERE e.id = $1`, id))
}

func (r *poolRepository) ListFXRates(ctx context.Context) ([]service.ValuationFXRate, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, base_currency, quote_currency, rate, effective_from, source, created_by_user_id, created_at FROM valuation_fx_rates ORDER BY effective_from DESC, id DESC`)
	if err != nil {
		return nil, fmt.Errorf("list valuation fx rates: %w", err)
	}
	defer rows.Close()
	items := make([]service.ValuationFXRate, 0)
	for rows.Next() {
		var item service.ValuationFXRate
		if err := rows.Scan(&item.ID, &item.BaseCurrency, &item.QuoteCurrency, &item.Rate, &item.EffectiveFrom, &item.Source, &item.CreatedByUserID, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *poolRepository) CreateFXRate(ctx context.Context, input service.CreateFXRateInput) (*service.ValuationFXRate, error) {
	var item service.ValuationFXRate
	err := r.db.QueryRowContext(ctx, `
INSERT INTO valuation_fx_rates(base_currency,quote_currency,rate,effective_from,source,created_by_user_id)
VALUES ($1,$2,$3,$4,$5,$6)
RETURNING id,base_currency,quote_currency,rate,effective_from,source,created_by_user_id,created_at`,
		input.BaseCurrency, input.QuoteCurrency, input.Rate, input.EffectiveFrom, input.Source, input.CreatedByUserID).
		Scan(&item.ID, &item.BaseCurrency, &item.QuoteCurrency, &item.Rate, &item.EffectiveFrom, &item.Source, &item.CreatedByUserID, &item.CreatedAt)
	if isPoolUniqueViolation(err) {
		return nil, infraerrors.Conflict("FX_RATE_EXISTS", "an exchange rate already exists at this effective time")
	}
	if err != nil {
		return nil, fmt.Errorf("create valuation fx rate: %w", err)
	}
	return &item, nil
}

func (r *poolRepository) LatestFXRate(ctx context.Context, at time.Time) (decimal.Decimal, error) {
	var raw string
	err := r.db.QueryRowContext(ctx, `SELECT rate FROM valuation_fx_rates WHERE base_currency='USD' AND quote_currency='CNY' AND effective_from <= $1 ORDER BY effective_from DESC LIMIT 1`, at).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return decimal.Zero, infraerrors.BadRequest("POOL_FX_RATE_REQUIRED", "USD/CNY valuation rate is required for this period")
	}
	if err != nil {
		return decimal.Zero, fmt.Errorf("get latest fx rate: %w", err)
	}
	value, err := decimal.NewFromString(raw)
	if err != nil {
		return decimal.Zero, fmt.Errorf("parse latest fx rate: %w", err)
	}
	return value, nil
}

func (r *poolRepository) SettlementInputs(ctx context.Context, start, end time.Time, filter service.SettlementFilterSnapshot) ([]service.PoolCostSlice, []service.PoolUsageWeight, service.PoolUsageCoverage, []service.SettlementCostSnapshot, error) {
	filterRaw, err := json.Marshal(filter)
	if err != nil {
		return nil, nil, service.PoolUsageCoverage{}, nil, fmt.Errorf("encode settlement filter: %w", err)
	}
	args := []any{start, end, nullableSettlementID(filter.AccountID), nullableSettlementID(filter.UploaderUserID), nullableSettlementID(filter.PayerUserID), nullableSettlementID(filter.PurchaseSourceID)}
	costRows, err := r.db.QueryContext(ctx, `
SELECT c.id,c.account_id,c.payer_user_id,c.entry_type,c.cny_amount_minor,c.service_start,c.service_end
FROM account_cost_entries c JOIN accounts a ON a.id=c.account_id
WHERE a.cost_sharing_enabled=TRUE
	AND c.entry_type<>'write_off'
  AND c.service_start < $2::date AND c.service_end > $1::date
  AND ($3::bigint IS NULL OR c.account_id=$3)
  AND ($4::bigint IS NULL OR a.created_by_user_id=$4)
  AND ($5::bigint IS NULL OR c.payer_user_id=$5)
  AND ($6::bigint IS NULL OR c.purchase_source_id=$6)
ORDER BY c.id`, args...)
	if err != nil {
		return nil, nil, service.PoolUsageCoverage{}, nil, fmt.Errorf("load settlement costs: %w", err)
	}
	costs := make([]service.PoolCostSlice, 0)
	for costRows.Next() {
		var item service.PoolCostSlice
		if err := costRows.Scan(&item.EntryID, &item.AccountID, &item.PayerUserID, &item.EntryType, &item.AmountMinor, &item.ServiceStart, &item.ServiceEnd); err != nil {
			costRows.Close()
			return nil, nil, service.PoolUsageCoverage{}, nil, err
		}
		costs = append(costs, item)
	}
	if err := costRows.Close(); err != nil {
		return nil, nil, service.PoolUsageCoverage{}, nil, err
	}

	usageRows, err := r.db.QueryContext(ctx, `
SELECT ul.user_id, u.email, u.username, SUM(ul.total_cost)::text
FROM usage_logs ul
JOIN accounts a ON a.id=ul.account_id AND a.cost_sharing_enabled=TRUE
JOIN users u ON u.id=ul.user_id AND u.deleted_at IS NULL
WHERE ul.created_at >= $1 AND ul.created_at < $2 AND ul.total_cost > 0
  AND ($3::bigint IS NULL OR a.id=$3)
  AND ($4::bigint IS NULL OR a.created_by_user_id=$4)
  AND EXISTS (
    SELECT 1 FROM account_cost_entries c
    WHERE c.account_id=a.id AND c.entry_type<>'write_off'
      AND c.service_start < $2::date AND c.service_end > $1::date
      AND ($5::bigint IS NULL OR c.payer_user_id=$5)
      AND ($6::bigint IS NULL OR c.purchase_source_id=$6)
  )
GROUP BY ul.user_id,u.email,u.username ORDER BY ul.user_id`, args...)
	if err != nil {
		return nil, nil, service.PoolUsageCoverage{}, nil, fmt.Errorf("load settlement usage: %w", err)
	}
	weights := make([]service.PoolUsageWeight, 0)
	for usageRows.Next() {
		var item service.PoolUsageWeight
		var raw string
		if err := usageRows.Scan(&item.UserID, &item.Email, &item.Username, &raw); err != nil {
			usageRows.Close()
			return nil, nil, service.PoolUsageCoverage{}, nil, err
		}
		item.Weight, err = decimal.NewFromString(raw)
		if err != nil {
			usageRows.Close()
			return nil, nil, service.PoolUsageCoverage{}, nil, err
		}
		weights = append(weights, item)
	}
	if err := usageRows.Close(); err != nil {
		return nil, nil, service.PoolUsageCoverage{}, nil, err
	}

	var coverage service.PoolUsageCoverage
	err = r.db.QueryRowContext(ctx, `
SELECT COUNT(*)::bigint,
	       COUNT(*) FILTER (WHERE ul.total_cost <= 0)::bigint,
	       COALESCE(SUM(GREATEST(ul.input_tokens::bigint+ul.output_tokens::bigint+ul.cache_creation_tokens::bigint+
	         ul.cache_read_tokens::bigint+ul.image_output_tokens::bigint+ul.image_input_tokens::bigint,1)),0)::bigint,
	       COALESCE(SUM(GREATEST(ul.input_tokens::bigint+ul.output_tokens::bigint+ul.cache_creation_tokens::bigint+
	         ul.cache_read_tokens::bigint+ul.image_output_tokens::bigint+ul.image_input_tokens::bigint,1))
	         FILTER (WHERE ul.total_cost <= 0),0)::bigint
FROM usage_logs ul
JOIN accounts a ON a.id=ul.account_id AND a.cost_sharing_enabled=TRUE
JOIN users u ON u.id=ul.user_id AND u.deleted_at IS NULL
WHERE ul.created_at >= $1 AND ul.created_at < $2
  AND ($3::bigint IS NULL OR a.id=$3)
  AND ($4::bigint IS NULL OR a.created_by_user_id=$4)
  AND EXISTS (
    SELECT 1 FROM account_cost_entries c
    WHERE c.account_id=a.id AND c.entry_type<>'write_off'
      AND c.service_start < $2::date AND c.service_end > $1::date
      AND ($5::bigint IS NULL OR c.payer_user_id=$5)
      AND ($6::bigint IS NULL OR c.purchase_source_id=$6)
  )
  AND (ul.actual_cost > 0 OR ul.total_cost > 0
       OR ul.input_tokens + ul.output_tokens + ul.cache_creation_tokens + ul.cache_read_tokens + ul.image_output_tokens + ul.image_input_tokens > 0
       OR ul.image_count > 0 OR ul.video_count > 0)`, args...).
		Scan(&coverage.CandidateCount, &coverage.UnpricedCount, &coverage.CandidateMaterial, &coverage.UnpricedMaterial)
	if err != nil {
		return nil, nil, service.PoolUsageCoverage{}, nil, fmt.Errorf("load settlement pricing coverage: %w", err)
	}

	var raw []byte
	var carryOut int64
	err = r.db.QueryRowContext(ctx, `SELECT cost_snapshot,carry_out_minor FROM pool_settlements WHERE status IN ('locked','paid') AND period_end <= $1 AND filter_snapshot=$2::jsonb ORDER BY period_end DESC,id DESC LIMIT 1`, start, string(filterRaw)).Scan(&raw, &carryOut)
	carry := make([]service.SettlementCostSnapshot, 0)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, nil, service.PoolUsageCoverage{}, nil, fmt.Errorf("load settlement carry: %w", err)
	}
	if err == nil && carryOut != 0 {
		if err := json.Unmarshal(raw, &carry); err != nil {
			return nil, nil, service.PoolUsageCoverage{}, nil, fmt.Errorf("decode settlement carry: %w", err)
		}
	}
	return costs, weights, coverage, carry, nil
}

func nullableSettlementID(id *int64) any {
	if id == nil {
		return nil
	}
	return *id
}

func (r *poolRepository) LockedAllocatedByCostEntry(ctx context.Context, ids []int64) (map[int64]int64, error) {
	result := make(map[int64]int64, len(ids))
	if len(ids) == 0 {
		return result, nil
	}
	rows, err := r.db.QueryContext(ctx, `
SELECT (item->>'entry_id')::bigint, COALESCE(SUM((item->>'amount_minor')::bigint),0)::bigint
FROM pool_settlements s CROSS JOIN LATERAL jsonb_array_elements(s.cost_snapshot) item
WHERE s.status IN ('locked','paid') AND item->>'kind'='period' AND (item->>'entry_id')::bigint=ANY($1)
GROUP BY (item->>'entry_id')::bigint`, pq.Array(ids))
	if err != nil {
		return nil, fmt.Errorf("sum locked cost allocations: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id, amount int64
		if err := rows.Scan(&id, &amount); err != nil {
			return nil, err
		}
		result[id] = amount
	}
	return result, rows.Err()
}

func (r *poolRepository) SaveDraftSettlement(ctx context.Context, settlement *service.PoolSettlement) (*service.PoolSettlement, error) {
	filterRaw, err := json.Marshal(settlement.FilterSnapshot)
	if err != nil {
		return nil, err
	}
	var existingID int64
	err = r.db.QueryRowContext(ctx, `
SELECT id FROM pool_settlements
WHERE status IN ('locked','paid') AND period_start=$1 AND period_end=$2 AND filter_snapshot=$3::jsonb
ORDER BY id DESC LIMIT 1`, settlement.PeriodStart, settlement.PeriodEnd, string(filterRaw)).Scan(&existingID)
	if err == nil {
		return r.GetSettlement(ctx, existingID)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err = tx.ExecContext(ctx, `DELETE FROM pool_settlements WHERE status='draft' AND period_start=$1 AND period_end=$2 AND filter_snapshot=$3::jsonb`, settlement.PeriodStart, settlement.PeriodEnd, string(filterRaw)); err != nil {
		return nil, err
	}
	raw, err := json.Marshal(settlement.CostSnapshot)
	if err != nil {
		return nil, err
	}
	err = tx.QueryRowContext(ctx, `
	INSERT INTO pool_settlements(period_type,period_start,period_end,timezone,status,period_cost_minor,carry_in_minor,carry_out_minor,total_cost_minor,total_usage_weight,pricing_coverage,unpriced_usage_count,fx_rate,formula_version,cost_snapshot,filter_snapshot,generated_by_user_id)
	VALUES($1,$2,$3,$4,'draft',$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16) RETURNING id,created_at,updated_at`, settlement.PeriodType, settlement.PeriodStart, settlement.PeriodEnd, settlement.Timezone, settlement.PeriodCostMinor, settlement.CarryInMinor, settlement.CarryOutMinor, settlement.TotalCostMinor, settlement.TotalUsageWeight, settlement.PricingCoverage, settlement.UnpricedCount, settlement.FXRate, settlement.FormulaVersion, raw, string(filterRaw), settlement.GeneratedBy).Scan(&settlement.ID, &settlement.CreatedAt, &settlement.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("save draft settlement: %w", err)
	}
	for i := range settlement.Lines {
		line := &settlement.Lines[i]
		line.SettlementID = settlement.ID
		line.ConfirmationStatus = "pending"
		err = tx.QueryRowContext(ctx, `
INSERT INTO pool_settlement_lines(settlement_id,user_id,usage_weight,usage_share,allocated_cost_minor,contribution_credit_minor,adjustment_minor,net_amount_minor,payment_status)
VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING id`, settlement.ID, line.UserID, line.UsageWeight, line.UsageShare, line.AllocatedCostMinor, line.ContributionCreditMinor, line.AdjustmentMinor, line.NetAmountMinor, line.PaymentStatus).Scan(&line.ID)
		if err != nil {
			return nil, fmt.Errorf("save settlement line: %w", err)
		}
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return settlement, nil
}

func (r *poolRepository) LockSettlement(ctx context.Context, id, actorID int64) (*service.PoolSettlement, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err = tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(73029191)`); err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(ctx, `LOCK TABLE account_cost_entries, usage_logs, valuation_fx_rates, accounts, users, pool_settlements IN SHARE MODE`); err != nil {
		return nil, fmt.Errorf("lock settlement inputs: %w", err)
	}
	var status string
	var start, end time.Time
	var pricingReady bool
	var filter service.SettlementFilterSnapshot
	var filterRaw []byte
	err = tx.QueryRowContext(ctx, `SELECT status,period_start,period_end,pricing_coverage::numeric >= 0.99,filter_snapshot FROM pool_settlements WHERE id=$1 FOR UPDATE`, id).Scan(&status, &start, &end, &pricingReady, &filterRaw)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrPoolSettlementNotFound
	}
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(filterRaw, &filter); err != nil {
		return nil, fmt.Errorf("decode settlement filter: %w", err)
	}
	if status == "locked" || status == "paid" {
		if err = tx.Commit(); err != nil {
			return nil, err
		}
		return r.GetSettlement(ctx, id)
	}
	if !pricingReady {
		return nil, infraerrors.Conflict("SETTLEMENT_PRICING_INCOMPLETE", "pricing coverage must reach 99% before locking")
	}
	var inputsChanged bool
	err = tx.QueryRowContext(ctx, `
WITH draft_cost_ids AS (
  SELECT (item->>'entry_id')::bigint id
  FROM pool_settlements s CROSS JOIN LATERAL jsonb_array_elements(s.cost_snapshot) item
  WHERE s.id=$1 AND item->>'kind'='period'
), current_cost_ids AS (
  SELECT c.id FROM account_cost_entries c JOIN accounts a ON a.id=c.account_id
	WHERE a.cost_sharing_enabled=TRUE AND c.entry_type<>'write_off'
    AND c.service_start < $3::date AND c.service_end > $2::date
    AND ($4::bigint IS NULL OR c.account_id=$4)
    AND ($5::bigint IS NULL OR a.created_by_user_id=$5)
    AND ($6::bigint IS NULL OR c.payer_user_id=$6)
    AND ($7::bigint IS NULL OR c.purchase_source_id=$7)
), current_weights AS (
  SELECT ul.user_id,SUM(ul.total_cost)::numeric weight
  FROM usage_logs ul
	JOIN accounts a ON a.id=ul.account_id AND a.cost_sharing_enabled=TRUE
  JOIN users u ON u.id=ul.user_id AND u.deleted_at IS NULL
  WHERE ul.created_at >= $2 AND ul.created_at < $3 AND ul.total_cost > 0
    AND ($4::bigint IS NULL OR a.id=$4)
    AND ($5::bigint IS NULL OR a.created_by_user_id=$5)
    AND EXISTS (
      SELECT 1 FROM account_cost_entries c
      WHERE c.account_id=a.id AND c.entry_type<>'write_off'
        AND c.service_start < $3::date AND c.service_end > $2::date
        AND ($6::bigint IS NULL OR c.payer_user_id=$6)
        AND ($7::bigint IS NULL OR c.purchase_source_id=$7)
    )
  GROUP BY ul.user_id
), draft_weights AS (
  SELECT user_id,usage_weight::numeric weight FROM pool_settlement_lines WHERE settlement_id=$1
), current_coverage AS (
  SELECT COUNT(*)::bigint candidate_count,
         COUNT(*) FILTER (WHERE ul.total_cost <= 0)::bigint unpriced_count
  FROM usage_logs ul
	JOIN accounts a ON a.id=ul.account_id AND a.cost_sharing_enabled=TRUE
  JOIN users u ON u.id=ul.user_id AND u.deleted_at IS NULL
  WHERE ul.created_at >= $2 AND ul.created_at < $3
    AND ($4::bigint IS NULL OR a.id=$4)
    AND ($5::bigint IS NULL OR a.created_by_user_id=$5)
    AND EXISTS (
      SELECT 1 FROM account_cost_entries c
      WHERE c.account_id=a.id AND c.entry_type<>'write_off'
        AND c.service_start < $3::date AND c.service_end > $2::date
        AND ($6::bigint IS NULL OR c.payer_user_id=$6)
        AND ($7::bigint IS NULL OR c.purchase_source_id=$7)
    )
    AND (ul.actual_cost > 0 OR ul.total_cost > 0
      OR ul.input_tokens + ul.output_tokens + ul.cache_creation_tokens + ul.cache_read_tokens + ul.image_output_tokens + ul.image_input_tokens > 0
      OR ul.image_count > 0 OR ul.video_count > 0)
), expected AS (
  SELECT s.unpriced_usage_count,
         s.pricing_coverage::numeric pricing_coverage,
         s.fx_rate::numeric fx_rate
  FROM pool_settlements s WHERE s.id=$1
), current_fx AS (
  SELECT rate::numeric fx_rate FROM valuation_fx_rates
  WHERE base_currency='USD' AND quote_currency='CNY' AND effective_from <= $3
  ORDER BY effective_from DESC LIMIT 1
)
SELECT
  EXISTS ((SELECT id FROM draft_cost_ids EXCEPT SELECT id FROM current_cost_ids)
          UNION ALL
          (SELECT id FROM current_cost_ids EXCEPT SELECT id FROM draft_cost_ids))
  OR EXISTS (
    SELECT 1 FROM current_weights c FULL JOIN draft_weights d USING(user_id)
    WHERE COALESCE(c.weight,0) <> COALESCE(d.weight,0)
  )
  OR EXISTS (
    SELECT 1 FROM current_coverage c CROSS JOIN expected e CROSS JOIN current_fx f
    WHERE c.unpriced_count <> e.unpriced_usage_count
       OR (CASE WHEN c.candidate_count=0 THEN 1::numeric
                ELSE (c.candidate_count-c.unpriced_count)::numeric/c.candidate_count END) <> e.pricing_coverage
       OR f.fx_rate <> e.fx_rate
  )`, id, start, end, nullableSettlementID(filter.AccountID), nullableSettlementID(filter.UploaderUserID), nullableSettlementID(filter.PayerUserID), nullableSettlementID(filter.PurchaseSourceID)).Scan(&inputsChanged)
	if err != nil {
		return nil, fmt.Errorf("verify settlement inputs: %w", err)
	}
	if inputsChanged {
		return nil, infraerrors.Conflict("SETTLEMENT_STALE", "settlement changed after preview; recalculate before locking")
	}
	var conflictID int64
	err = tx.QueryRowContext(ctx, `
SELECT other.id
FROM pool_settlements other
WHERE other.status IN ('locked','paid') AND other.id<>$1
  AND other.period_start<$3 AND other.period_end>$2
  AND EXISTS (
    SELECT 1
    FROM jsonb_array_elements((SELECT cost_snapshot FROM pool_settlements WHERE id=$1)) draft_item
    CROSS JOIN jsonb_array_elements(other.cost_snapshot) other_item
    WHERE (draft_item->>'entry_id')::bigint=(other_item->>'entry_id')::bigint
  )
LIMIT 1`, id, start, end).Scan(&conflictID)
	if err == nil {
		return nil, infraerrors.Conflict("SETTLEMENT_PERIOD_OVERLAP", fmt.Sprintf("settlement period overlaps locked settlement %d", conflictID))
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	var stale bool
	err = tx.QueryRowContext(ctx, `
WITH draft_items AS (
  SELECT (item->>'entry_id')::bigint entry_id, SUM((item->>'amount_minor')::bigint)::bigint amount_minor
  FROM pool_settlements s CROSS JOIN LATERAL jsonb_array_elements(s.cost_snapshot) item
  WHERE s.id=$1 AND item->>'kind'='period' GROUP BY (item->>'entry_id')::bigint
), locked_items AS (
  SELECT (item->>'entry_id')::bigint entry_id, SUM((item->>'amount_minor')::bigint)::bigint amount_minor
  FROM pool_settlements s CROSS JOIN LATERAL jsonb_array_elements(s.cost_snapshot) item
  WHERE s.status IN ('locked','paid') AND item->>'kind'='period' GROUP BY (item->>'entry_id')::bigint
)
SELECT EXISTS (
  SELECT 1 FROM draft_items d JOIN account_cost_entries c ON c.id=d.entry_id
  LEFT JOIN locked_items l ON l.entry_id=d.entry_id
  WHERE (c.cny_amount_minor >= 0 AND COALESCE(l.amount_minor,0)+d.amount_minor > c.cny_amount_minor)
     OR (c.cny_amount_minor < 0 AND COALESCE(l.amount_minor,0)+d.amount_minor < c.cny_amount_minor)
)`, id).Scan(&stale)
	if err != nil {
		return nil, err
	}
	var carryIn, expectedCarry int64
	if err = tx.QueryRowContext(ctx, `SELECT carry_in_minor FROM pool_settlements WHERE id=$1`, id).Scan(&carryIn); err != nil {
		return nil, err
	}
	if err = tx.QueryRowContext(ctx, `SELECT COALESCE((SELECT carry_out_minor FROM pool_settlements WHERE status IN ('locked','paid') AND period_end <= $1 AND filter_snapshot=$2::jsonb ORDER BY period_end DESC,id DESC LIMIT 1),0)`, start, string(filterRaw)).Scan(&expectedCarry); err != nil {
		return nil, err
	}
	if stale || carryIn != expectedCarry {
		return nil, infraerrors.Conflict("SETTLEMENT_STALE", "settlement changed after preview; recalculate before locking")
	}
	result, err := tx.ExecContext(ctx, `UPDATE pool_settlements SET status='locked',locked_by_user_id=$2,locked_at=NOW(),updated_at=NOW() WHERE id=$1 AND status='draft'`, id, actorID)
	if err != nil {
		return nil, err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return nil, infraerrors.Conflict("SETTLEMENT_ALREADY_LOCKED", "settlement is already locked")
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return r.GetSettlement(ctx, id)
}

func (r *poolRepository) ConfirmSettlementLine(ctx context.Context, id, userID, actorID int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var status string
	if err = tx.QueryRowContext(ctx, `SELECT status FROM pool_settlements WHERE id=$1 FOR UPDATE`, id).Scan(&status); errors.Is(err, sql.ErrNoRows) {
		return service.ErrPoolSettlementNotFound
	} else if err != nil {
		return err
	}
	if status != "locked" {
		return infraerrors.Conflict("SETTLEMENT_NOT_LOCKED", "only a locked settlement can be confirmed")
	}

	var lineID, netAmount int64
	var confirmationStatus string
	err = tx.QueryRowContext(ctx, `
SELECT id,net_amount_minor,confirmation_status
FROM pool_settlement_lines
WHERE settlement_id=$1 AND user_id=$2
FOR UPDATE`, id, userID).Scan(&lineID, &netAmount, &confirmationStatus)
	if errors.Is(err, sql.ErrNoRows) {
		return infraerrors.NotFound("SETTLEMENT_MEMBER_LINE_NOT_FOUND", "the signed-in member has no line in this settlement")
	}
	if err != nil {
		return err
	}
	if netAmount == 0 {
		return infraerrors.Conflict("SETTLEMENT_CONFIRMATION_NOT_REQUIRED", "a zero amount line does not require confirmation")
	}
	if confirmationStatus == "confirmed" {
		return tx.Commit()
	}
	if _, err = tx.ExecContext(ctx, `
UPDATE pool_settlement_lines
SET confirmation_status='confirmed',confirmed_by_user_id=$2,confirmed_at=NOW(),updated_at=NOW()
WHERE id=$1`, lineID, actorID); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *poolRepository) MarkSettlementPaid(ctx context.Context, id, actorID int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var status string
	if err = tx.QueryRowContext(ctx, `SELECT status FROM pool_settlements WHERE id=$1 FOR UPDATE`, id).Scan(&status); errors.Is(err, sql.ErrNoRows) {
		return service.ErrPoolSettlementNotFound
	} else if err != nil {
		return err
	}
	if status == "paid" {
		return tx.Commit()
	}
	if status != "locked" {
		return infraerrors.Conflict("SETTLEMENT_NOT_LOCKED", "only a locked settlement can be marked paid")
	}
	var pending int64
	if err = tx.QueryRowContext(ctx, `
SELECT COUNT(*) FROM pool_settlement_lines
WHERE settlement_id=$1 AND net_amount_minor<>0 AND confirmation_status<>'confirmed'`, id).Scan(&pending); err != nil {
		return err
	}
	if pending > 0 {
		return infraerrors.Conflict("SETTLEMENT_CONFIRMATIONS_PENDING", "all non-zero settlement lines must be confirmed before marking paid")
	}
	if _, err = tx.ExecContext(ctx, `UPDATE pool_settlement_lines SET payment_status='paid',updated_at=NOW() WHERE settlement_id=$1`, id); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE pool_settlements SET status='paid',paid_by_user_id=$2,paid_at=NOW(),updated_at=NOW() WHERE id=$1`, id, actorID); err != nil {
		return err
	}
	return tx.Commit()
}

const settlementSelect = `
SELECT id,period_type,period_start,period_end,timezone,status,period_cost_minor,carry_in_minor,carry_out_minor,total_cost_minor,total_usage_weight,pricing_coverage,unpriced_usage_count,fx_rate,formula_version,cost_snapshot,filter_snapshot,generated_by_user_id,locked_by_user_id,locked_at,paid_by_user_id,paid_at,created_at,updated_at
FROM pool_settlements`

func scanSettlement(scanner interface{ Scan(...any) error }) (*service.PoolSettlement, error) {
	var item service.PoolSettlement
	var costRaw, filterRaw []byte
	err := scanner.Scan(&item.ID, &item.PeriodType, &item.PeriodStart, &item.PeriodEnd, &item.Timezone, &item.Status, &item.PeriodCostMinor, &item.CarryInMinor, &item.CarryOutMinor, &item.TotalCostMinor, &item.TotalUsageWeight, &item.PricingCoverage, &item.UnpricedCount, &item.FXRate, &item.FormulaVersion, &costRaw, &filterRaw, &item.GeneratedBy, &item.LockedBy, &item.LockedAt, &item.PaidBy, &item.PaidAt, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(costRaw, &item.CostSnapshot); err != nil {
		return nil, err
	}
	if err = json.Unmarshal(filterRaw, &item.FilterSnapshot); err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *poolRepository) loadSettlementLines(ctx context.Context, id int64) ([]service.PoolSettlementLine, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT l.id,l.settlement_id,l.user_id,u.email,u.username,l.usage_weight,l.usage_share,l.allocated_cost_minor,l.contribution_credit_minor,l.adjustment_minor,l.net_amount_minor,l.payment_status,l.confirmation_status,l.confirmed_by_user_id,l.confirmed_at
FROM pool_settlement_lines l JOIN users u ON u.id=l.user_id WHERE l.settlement_id=$1 ORDER BY l.user_id`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]service.PoolSettlementLine, 0)
	for rows.Next() {
		var item service.PoolSettlementLine
		if err := rows.Scan(&item.ID, &item.SettlementID, &item.UserID, &item.UserEmail, &item.Username, &item.UsageWeight, &item.UsageShare, &item.AllocatedCostMinor, &item.ContributionCreditMinor, &item.AdjustmentMinor, &item.NetAmountMinor, &item.PaymentStatus, &item.ConfirmationStatus, &item.ConfirmedByUserID, &item.ConfirmedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *poolRepository) GetSettlement(ctx context.Context, id int64) (*service.PoolSettlement, error) {
	item, err := scanSettlement(r.db.QueryRowContext(ctx, settlementSelect+` WHERE id=$1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrPoolSettlementNotFound
	}
	if err != nil {
		return nil, err
	}
	item.Lines, err = r.loadSettlementLines(ctx, id)
	return item, err
}

func (r *poolRepository) ListSettlements(ctx context.Context, limit, offset int) ([]service.PoolSettlement, int64, error) {
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM pool_settlements`).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.QueryContext(ctx, settlementSelect+` ORDER BY period_start DESC,id DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]service.PoolSettlement, 0)
	for rows.Next() {
		item, err := scanSettlement(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, *item)
	}
	return items, total, rows.Err()
}

func (r *poolRepository) GetRecovery(ctx context.Context, start, end time.Time) ([]service.AccountRecovery, error) {
	_ = start // Recovery is cumulative as of end; start is used by period AA only.
	rows, err := r.db.QueryContext(ctx, `
WITH RECURSIVE pool_accounts AS (
  SELECT a.id,a.name,a.provider_identity,a.created_at FROM accounts a
	WHERE a.cost_sharing_enabled=TRUE
), costs AS (
  SELECT a.id account_id,
	       COALESCE(SUM(c.cny_amount_minor) FILTER (WHERE c.entry_type NOT IN ('refund','replacement_out','write_off')),0)::bigint cost_basis_minor,
	       COALESCE(-SUM(c.cny_amount_minor) FILTER (WHERE c.entry_type='refund' AND c.cny_amount_minor<0),0)::bigint refund_minor,
	       COALESCE(-SUM(c.cny_amount_minor) FILTER (WHERE c.entry_type='replacement_out' AND c.cny_amount_minor<0),0)::bigint transferred_minor,
	       COALESCE(-SUM(c.cny_amount_minor) FILTER (WHERE c.entry_type='write_off' AND c.cny_amount_minor<0),0)::bigint written_off_minor,
	       COALESCE(SUM(c.expected_token_count) FILTER (WHERE c.cny_amount_minor>0 AND c.expected_token_count>0),0)::bigint expected_tokens
	FROM pool_accounts a LEFT JOIN account_cost_entries c ON c.account_id=a.id AND c.paid_at<$1
  GROUP BY a.id
), usage_values AS (
  SELECT ul.id,ul.account_id,ul.created_at,
	     (ul.input_tokens::bigint+ul.output_tokens::bigint+ul.cache_creation_tokens::bigint+
	      ul.cache_read_tokens::bigint+ul.image_output_tokens::bigint+ul.image_input_tokens::bigint) value_minor,
         DATE(ul.created_at AT TIME ZONE 'Asia/Shanghai') usage_day
  FROM usage_logs ul JOIN pool_accounts a ON a.id=ul.account_id
  WHERE ul.created_at<$1 AND
	(ul.input_tokens::bigint+ul.output_tokens::bigint+ul.cache_creation_tokens::bigint+
	 ul.cache_read_tokens::bigint+ul.image_output_tokens::bigint+ul.image_input_tokens::bigint)>0
), priced_tranches AS (
  SELECT c.account_id,c.cny_amount_minor,c.expected_token_count,
	     COALESCE(SUM(c.expected_token_count) OVER (
	       PARTITION BY c.account_id ORDER BY c.paid_at,c.id ROWS BETWEEN UNBOUNDED PRECEDING AND 1 PRECEDING
	     ),0)::bigint prior_tokens
  FROM account_cost_entries c JOIN pool_accounts a ON a.id=c.account_id
  WHERE c.paid_at<$1 AND c.cny_amount_minor>0 AND c.expected_token_count>0
), source AS (
  SELECT DISTINCT ON (a.id) a.id account_id,ps.name,c.paid_at
  FROM pool_accounts a JOIN account_cost_entries c ON c.account_id=a.id AND c.entry_type='purchase' AND c.paid_at<$1
  LEFT JOIN purchase_sources ps ON ps.id=c.purchase_source_id
  ORDER BY a.id,c.paid_at,c.id
), investment_start AS (
  SELECT c.account_id,MIN(c.paid_at) invested_at
  FROM account_cost_entries c JOIN pool_accounts a ON a.id=c.account_id
  WHERE c.paid_at<$1 AND c.cny_amount_minor>0 GROUP BY c.account_id
), observation AS (
  SELECT a.id account_id,
         GREATEST(
           COALESCE(DATE(i.invested_at AT TIME ZONE 'Asia/Shanghai'),DATE(a.created_at AT TIME ZONE 'Asia/Shanghai')),
           DATE($1 AT TIME ZONE 'Asia/Shanghai' - INTERVAL '1 microsecond')-13
         ) window_start,
         DATE($1 AT TIME ZONE 'Asia/Shanghai' - INTERVAL '1 microsecond') window_end
  FROM pool_accounts a LEFT JOIN investment_start i ON i.account_id=a.id
), recent_daily AS (
  SELECT o.account_id,
         GREATEST(o.window_end-o.window_start+1,0)::bigint observation_days,
         COUNT(DISTINCT uv.usage_day)::bigint effective_days,
         CASE WHEN o.window_end>=o.window_start
              THEN COALESCE(ROUND(SUM(uv.value_minor)/(o.window_end-o.window_start+1)),0)::bigint
              ELSE 0::bigint END avg_daily
  FROM observation o LEFT JOIN usage_values uv
    ON uv.account_id=o.account_id AND uv.usage_day BETWEEN o.window_start AND o.window_end
  GROUP BY o.account_id,o.window_start,o.window_end
), lifecycle AS (
  SELECT DISTINCT ON (a.id) a.id account_id,e.event_type,e.occurred_at
  FROM pool_accounts a JOIN account_lifecycle_events e ON e.account_id=a.id AND e.occurred_at<$1 AND e.event_type<>'refund'
  ORDER BY a.id,e.occurred_at DESC,e.id DESC
), banned AS (
  SELECT a.id account_id,MAX(e.occurred_at) banned_at FROM pool_accounts a
  JOIN account_lifecycle_events e ON e.account_id=a.id
  WHERE e.event_type='banned_confirmed' AND e.occurred_at<$1 GROUP BY a.id
), refunds AS (
  SELECT a.id account_id,TRUE refunded FROM pool_accounts a
  WHERE EXISTS (SELECT 1 FROM account_cost_entries c WHERE c.account_id=a.id AND c.entry_type='refund' AND c.paid_at<$1)
     OR EXISTS (SELECT 1 FROM account_lifecycle_events e WHERE e.account_id=a.id AND e.event_type='refund' AND e.occurred_at<$1)
), financial_events AS (
  SELECT c.account_id,c.paid_at event_at,0 event_kind,c.id event_id,c.expected_token_count::bigint delta
	FROM account_cost_entries c JOIN pool_accounts a ON a.id=c.account_id
	WHERE c.paid_at<$1 AND c.cny_amount_minor>0 AND c.expected_token_count>0
  UNION ALL
  SELECT uv.account_id,uv.created_at,1,uv.id,-uv.value_minor FROM usage_values uv WHERE uv.value_minor>0
), ordered_events AS (
  SELECT e.*,ROW_NUMBER() OVER(PARTITION BY e.account_id ORDER BY e.event_at,e.event_kind,e.event_id) event_no
  FROM financial_events e
), token_flow AS (
  SELECT e.*,GREATEST(e.delta,0)::bigint balance
  FROM ordered_events e WHERE e.event_no=1
  UNION ALL
  SELECT e.*,GREATEST(flow.balance+e.delta,0)::bigint
  FROM token_flow flow JOIN ordered_events e ON e.account_id=flow.account_id AND e.event_no=flow.event_no+1
), transitions AS (
  SELECT flow.*,LAG(flow.balance) OVER(PARTITION BY flow.account_id ORDER BY flow.event_no) previous_balance
  FROM token_flow flow
), recoveries AS (
  SELECT account_id,
         MIN(event_at) FILTER(WHERE balance<=0 AND previous_balance>0) first_recovery_at,
	     MAX(event_at) FILTER(WHERE balance<=0 AND previous_balance>0) latest_recovery_at
  FROM transitions GROUP BY account_id
), current_balances AS (
  SELECT DISTINCT ON (account_id) account_id,balance
  FROM token_flow ORDER BY account_id,event_no DESC
), recognized AS (
  SELECT p.account_id,COALESCE(SUM(ROUND(
	  p.cny_amount_minor::numeric * GREATEST(LEAST(
	    GREATEST(COALESCE(c.expected_tokens,0)-COALESCE(current_balances.balance,0),0)-p.prior_tokens,
	    p.expected_token_count),0) / p.expected_token_count
	)),0)::bigint recognized_minor
  FROM priced_tranches p
  JOIN costs c ON c.account_id=p.account_id
  LEFT JOIN current_balances ON current_balances.account_id=p.account_id
  GROUP BY p.account_id
), investments AS (
  SELECT c.account_id,TRUE has_investment FROM account_cost_entries c JOIN pool_accounts a ON a.id=c.account_id
	WHERE c.paid_at<$1 AND c.cny_amount_minor>0 AND c.expected_token_count>0 GROUP BY c.account_id
)
SELECT a.id,a.name,a.provider_identity,source.name,COALESCE(lifecycle.event_type,'active'),
	   COALESCE(c.cost_basis_minor,0),
	   LEAST(COALESCE(c.cost_basis_minor,0),
	     LEAST(COALESCE(recognized.recognized_minor,0),GREATEST(COALESCE(c.cost_basis_minor,0)-COALESCE(c.refund_minor,0)-COALESCE(c.transferred_minor,0)-COALESCE(c.written_off_minor,0),0))+
	     COALESCE(c.refund_minor,0)+COALESCE(c.transferred_minor,0)),
	   COALESCE(rd.avg_daily,0),
       investment_start.invested_at,banned.banned_at,COALESCE(refunds.refunded,FALSE),COALESCE(rd.effective_days,0),
	   COALESCE(rd.observation_days,0),COALESCE(c.written_off_minor,0)+CASE WHEN COALESCE(lifecycle.event_type,'active')='banned_confirmed' THEN
	     GREATEST(COALESCE(c.cost_basis_minor,0)-COALESCE(c.refund_minor,0)-COALESCE(c.transferred_minor,0)-COALESCE(c.written_off_minor,0)-COALESCE(recognized.recognized_minor,0),0)
	   ELSE 0 END,
       recoveries.first_recovery_at,recoveries.latest_recovery_at,
	   COALESCE(current_balances.balance<=0 AND investments.has_investment,FALSE),
	   COALESCE(c.expected_tokens,0),GREATEST(COALESCE(c.expected_tokens,0)-COALESCE(current_balances.balance,0),0),COALESCE(current_balances.balance,0)
FROM pool_accounts a
LEFT JOIN costs c ON c.account_id=a.id
LEFT JOIN recognized ON recognized.account_id=a.id
LEFT JOIN recent_daily rd ON rd.account_id=a.id LEFT JOIN source ON source.account_id=a.id
LEFT JOIN investment_start ON investment_start.account_id=a.id
LEFT JOIN lifecycle ON lifecycle.account_id=a.id LEFT JOIN banned ON banned.account_id=a.id
LEFT JOIN refunds ON refunds.account_id=a.id
LEFT JOIN recoveries ON recoveries.account_id=a.id LEFT JOIN current_balances ON current_balances.account_id=a.id
LEFT JOIN investments ON investments.account_id=a.id
ORDER BY a.id DESC`, end)
	if err != nil {
		return nil, fmt.Errorf("get pool recovery: %w", err)
	}
	defer rows.Close()
	items := make([]service.AccountRecovery, 0)
	for rows.Next() {
		var item service.AccountRecovery
		if err := rows.Scan(&item.AccountID, &item.AccountName, &item.ProviderIdentity, &item.PurchaseSource, &item.LifecycleStatus, &item.NetCostMinor, &item.ValueMinor, &item.AverageDailyTokens, &item.PurchasedAt, &item.BannedAt, &item.Refunded, &item.EffectiveUsageDays, &item.ObservationDays, &item.BannedLossMinor, &item.FirstRecoveryAt, &item.LatestRecoveryAt, &item.CurrentlyRecovered, &item.ExpectedTokens, &item.UsedTokens, &item.RemainingTokens); err != nil {
			return nil, err
		}
		finalizeAccountRecovery(&item, end)
		items = append(items, item)
	}
	return items, rows.Err()
}

func finalizeAccountRecovery(item *service.AccountRecovery, end time.Time) {
	item.NetProfitMinor = item.ValueMinor - item.NetCostMinor
	item.UnrecoveredMinor = item.NetCostMinor - item.ValueMinor
	if item.UnrecoveredMinor < 0 {
		item.UnrecoveredMinor = 0
	}
	item.CurrentNetLossMinor = item.UnrecoveredMinor
	if item.PurchasedAt != nil {
		survivalEnd := end
		if item.BannedAt != nil && item.BannedAt.Before(survivalEnd) {
			survivalEnd = *item.BannedAt
		}
		if survivalEnd.After(*item.PurchasedAt) {
			item.SurvivalDays = int64(survivalEnd.Sub(*item.PurchasedAt).Hours() / 24)
		}
	}
	if item.NetCostMinor > 0 {
		item.RecoveryRate = decimal.NewFromInt(item.ValueMinor).Div(decimal.NewFromInt(item.NetCostMinor)).String()
	} else {
		item.RecoveryRate = "0"
	}
	item.CurrentlyRecovered = item.NetCostMinor > 0 && item.ValueMinor >= item.NetCostMinor
	terminal := item.LifecycleStatus == "banned_confirmed" || item.LifecycleStatus == "retired" || item.LifecycleStatus == "replaced"
	if !item.CurrentlyRecovered && !terminal && item.RemainingTokens > 0 && item.AverageDailyTokens > 0 && item.ObservationDays >= 7 && item.EffectiveUsageDays >= 3 {
		days := decimal.NewFromInt(item.RemainingTokens).Div(decimal.NewFromInt(item.AverageDailyTokens)).Ceil().IntPart()
		item.EstimatedRecoveryDays = &days
	}
}

var _ service.PoolRepository = (*poolRepository)(nil)

// Keep source names stable for grouping while preserving their display form.
func normalizePurchaseSourceName(name string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(name)), " ")
}
