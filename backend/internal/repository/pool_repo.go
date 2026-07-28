package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
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
       c.related_account_id, c.created_by_user_id, c.created_at
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
		&item.RelatedAccountID, &item.CreatedByUserID, &item.CreatedAt)
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

func (r *poolRepository) CreateCost(ctx context.Context, input service.CreateAccountCostInput) (*service.AccountCostEntry, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err = tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, fmt.Sprintf("pool-cost:%d", input.AccountID)); err != nil {
		return nil, err
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
    paid_at, order_no, purchase_url, note, supersedes_id, related_account_id, created_by_user_id)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)
RETURNING id`, input.AccountID, input.PayerUserID, input.PurchaseSourceID, input.EntryType,
		input.Currency, input.OriginalAmount, input.CNYAmountMinor, input.FXRate,
		input.ServiceStart, input.ServiceEnd, input.WarrantyEnd, input.PaidAt, input.OrderNo, input.PurchaseURL,
		input.Note, input.SupersedesID, input.RelatedAccountID, input.CreatedByUserID)
	var id int64
	if err := row.Scan(&id); err != nil {
		return nil, fmt.Errorf("create account cost: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return scanCost(r.db.QueryRowContext(ctx, costSelect+` WHERE c.id = $1`, id))
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

func (r *poolRepository) SettlementInputs(ctx context.Context, start, end time.Time) ([]service.PoolCostSlice, []service.PoolUsageWeight, service.PoolUsageCoverage, []service.SettlementCostSnapshot, error) {
	costRows, err := r.db.QueryContext(ctx, `
SELECT c.id,c.account_id,c.payer_user_id,c.entry_type,c.cny_amount_minor,c.service_start,c.service_end
FROM account_cost_entries c JOIN accounts a ON a.id=c.account_id
WHERE a.deleted_at IS NULL AND a.cost_sharing_enabled=TRUE
  AND c.service_start < $2::date AND c.service_end > $1::date
ORDER BY c.id`, start, end)
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
JOIN accounts a ON a.id=ul.account_id AND a.deleted_at IS NULL AND a.cost_sharing_enabled=TRUE
JOIN users u ON u.id=ul.user_id AND u.deleted_at IS NULL
WHERE ul.created_at >= $1 AND ul.created_at < $2 AND ul.total_cost > 0
GROUP BY ul.user_id,u.email,u.username ORDER BY ul.user_id`, start, end)
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
       COUNT(*) FILTER (WHERE ul.total_cost <= 0)::bigint
FROM usage_logs ul
JOIN accounts a ON a.id=ul.account_id AND a.deleted_at IS NULL AND a.cost_sharing_enabled=TRUE
JOIN users u ON u.id=ul.user_id AND u.deleted_at IS NULL
WHERE ul.created_at >= $1 AND ul.created_at < $2
  AND (ul.actual_cost > 0 OR ul.total_cost > 0
       OR ul.input_tokens + ul.output_tokens + ul.cache_creation_tokens + ul.cache_read_tokens + ul.image_output_tokens + ul.image_input_tokens > 0
       OR ul.image_count > 0 OR ul.video_count > 0)`, start, end).
		Scan(&coverage.CandidateCount, &coverage.UnpricedCount)
	if err != nil {
		return nil, nil, service.PoolUsageCoverage{}, nil, fmt.Errorf("load settlement pricing coverage: %w", err)
	}

	var raw []byte
	var carryOut int64
	err = r.db.QueryRowContext(ctx, `SELECT cost_snapshot,carry_out_minor FROM pool_settlements WHERE status='locked' AND period_end <= $1 ORDER BY period_end DESC,id DESC LIMIT 1`, start).Scan(&raw, &carryOut)
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

func (r *poolRepository) LockedAllocatedByCostEntry(ctx context.Context, ids []int64) (map[int64]int64, error) {
	result := make(map[int64]int64, len(ids))
	if len(ids) == 0 {
		return result, nil
	}
	rows, err := r.db.QueryContext(ctx, `
SELECT (item->>'entry_id')::bigint, COALESCE(SUM((item->>'amount_minor')::bigint),0)::bigint
FROM pool_settlements s CROSS JOIN LATERAL jsonb_array_elements(s.cost_snapshot) item
WHERE s.status='locked' AND item->>'kind'='period' AND (item->>'entry_id')::bigint=ANY($1)
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
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err = tx.ExecContext(ctx, `DELETE FROM pool_settlements WHERE status='draft' AND period_start=$1 AND period_end=$2`, settlement.PeriodStart, settlement.PeriodEnd); err != nil {
		return nil, err
	}
	raw, err := json.Marshal(settlement.CostSnapshot)
	if err != nil {
		return nil, err
	}
	err = tx.QueryRowContext(ctx, `
INSERT INTO pool_settlements(period_type,period_start,period_end,timezone,status,period_cost_minor,carry_in_minor,carry_out_minor,total_cost_minor,total_usage_weight,pricing_coverage,unpriced_usage_count,fx_rate,formula_version,cost_snapshot,generated_by_user_id)
VALUES($1,$2,$3,$4,'draft',$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15) RETURNING id,created_at,updated_at`, settlement.PeriodType, settlement.PeriodStart, settlement.PeriodEnd, settlement.Timezone, settlement.PeriodCostMinor, settlement.CarryInMinor, settlement.CarryOutMinor, settlement.TotalCostMinor, settlement.TotalUsageWeight, settlement.PricingCoverage, settlement.UnpricedCount, settlement.FXRate, settlement.FormulaVersion, raw, settlement.GeneratedBy).Scan(&settlement.ID, &settlement.CreatedAt, &settlement.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("save draft settlement: %w", err)
	}
	for i := range settlement.Lines {
		line := &settlement.Lines[i]
		line.SettlementID = settlement.ID
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
	err = tx.QueryRowContext(ctx, `SELECT status,period_start,period_end,pricing_coverage::numeric >= 0.99 FROM pool_settlements WHERE id=$1 FOR UPDATE`, id).Scan(&status, &start, &end, &pricingReady)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrPoolSettlementNotFound
	}
	if err != nil {
		return nil, err
	}
	if status == "locked" {
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
  WHERE a.deleted_at IS NULL AND a.cost_sharing_enabled=TRUE
    AND c.service_start < $3::date AND c.service_end > $2::date
), current_weights AS (
  SELECT ul.user_id,SUM(ul.total_cost)::numeric weight
  FROM usage_logs ul
  JOIN accounts a ON a.id=ul.account_id AND a.deleted_at IS NULL AND a.cost_sharing_enabled=TRUE
  JOIN users u ON u.id=ul.user_id AND u.deleted_at IS NULL
  WHERE ul.created_at >= $2 AND ul.created_at < $3 AND ul.total_cost > 0
  GROUP BY ul.user_id
), draft_weights AS (
  SELECT user_id,usage_weight::numeric weight FROM pool_settlement_lines WHERE settlement_id=$1
), current_coverage AS (
  SELECT COUNT(*)::bigint candidate_count,
         COUNT(*) FILTER (WHERE ul.total_cost <= 0)::bigint unpriced_count
  FROM usage_logs ul
  JOIN accounts a ON a.id=ul.account_id AND a.deleted_at IS NULL AND a.cost_sharing_enabled=TRUE
  JOIN users u ON u.id=ul.user_id AND u.deleted_at IS NULL
  WHERE ul.created_at >= $2 AND ul.created_at < $3
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
  )`, id, start, end).Scan(&inputsChanged)
	if err != nil {
		return nil, fmt.Errorf("verify settlement inputs: %w", err)
	}
	if inputsChanged {
		return nil, infraerrors.Conflict("SETTLEMENT_STALE", "settlement changed after preview; recalculate before locking")
	}
	var conflictID int64
	err = tx.QueryRowContext(ctx, `SELECT id FROM pool_settlements WHERE status='locked' AND id<>$1 AND period_start<$3 AND period_end>$2 LIMIT 1`, id, start, end).Scan(&conflictID)
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
  WHERE s.status='locked' AND item->>'kind'='period' GROUP BY (item->>'entry_id')::bigint
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
	if err = tx.QueryRowContext(ctx, `SELECT COALESCE((SELECT carry_out_minor FROM pool_settlements WHERE status='locked' AND period_end <= $1 ORDER BY period_end DESC,id DESC LIMIT 1),0)`, start).Scan(&expectedCarry); err != nil {
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

const settlementSelect = `
SELECT id,period_type,period_start,period_end,timezone,status,period_cost_minor,carry_in_minor,carry_out_minor,total_cost_minor,total_usage_weight,pricing_coverage,unpriced_usage_count,fx_rate,formula_version,cost_snapshot,generated_by_user_id,locked_by_user_id,locked_at,created_at,updated_at
FROM pool_settlements`

func scanSettlement(scanner interface{ Scan(...any) error }) (*service.PoolSettlement, error) {
	var item service.PoolSettlement
	var raw []byte
	err := scanner.Scan(&item.ID, &item.PeriodType, &item.PeriodStart, &item.PeriodEnd, &item.Timezone, &item.Status, &item.PeriodCostMinor, &item.CarryInMinor, &item.CarryOutMinor, &item.TotalCostMinor, &item.TotalUsageWeight, &item.PricingCoverage, &item.UnpricedCount, &item.FXRate, &item.FormulaVersion, &raw, &item.GeneratedBy, &item.LockedBy, &item.LockedAt, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(raw, &item.CostSnapshot); err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *poolRepository) loadSettlementLines(ctx context.Context, id int64) ([]service.PoolSettlementLine, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT l.id,l.settlement_id,l.user_id,u.email,u.username,l.usage_weight,l.usage_share,l.allocated_cost_minor,l.contribution_credit_minor,l.adjustment_minor,l.net_amount_minor,l.payment_status
FROM pool_settlement_lines l JOIN users u ON u.id=l.user_id WHERE l.settlement_id=$1 ORDER BY l.user_id`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]service.PoolSettlementLine, 0)
	for rows.Next() {
		var item service.PoolSettlementLine
		if err := rows.Scan(&item.ID, &item.SettlementID, &item.UserID, &item.UserEmail, &item.Username, &item.UsageWeight, &item.UsageShare, &item.AllocatedCostMinor, &item.ContributionCreditMinor, &item.AdjustmentMinor, &item.NetAmountMinor, &item.PaymentStatus); err != nil {
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
	var missingFX int64
	if err := r.db.QueryRowContext(ctx, `
SELECT COUNT(*) FROM usage_logs ul
JOIN accounts a ON a.id=ul.account_id AND a.deleted_at IS NULL AND a.cost_sharing_enabled=TRUE
WHERE ul.created_at<$1 AND ul.total_cost>0
  AND NOT EXISTS (
    SELECT 1 FROM valuation_fx_rates fx
    WHERE fx.base_currency='USD' AND fx.quote_currency='CNY' AND fx.effective_from<=ul.created_at
  )`, end).Scan(&missingFX); err != nil {
		return nil, fmt.Errorf("check pool recovery fx coverage: %w", err)
	}
	if missingFX > 0 {
		return nil, infraerrors.BadRequest("POOL_FX_RATE_REQUIRED", fmt.Sprintf("USD/CNY valuation rate is missing for %d usage records", missingFX))
	}
	var unpriced int64
	if err := r.db.QueryRowContext(ctx, `
SELECT COUNT(*) FROM usage_logs ul
JOIN accounts a ON a.id=ul.account_id AND a.deleted_at IS NULL AND a.cost_sharing_enabled=TRUE
WHERE ul.created_at<$1 AND ul.total_cost<=0
  AND (ul.actual_cost>0
    OR ul.input_tokens + ul.output_tokens + ul.cache_creation_tokens + ul.cache_read_tokens + ul.image_output_tokens + ul.image_input_tokens > 0
    OR ul.image_count>0 OR ul.video_count>0)`, end).Scan(&unpriced); err != nil {
		return nil, fmt.Errorf("check pool recovery pricing coverage: %w", err)
	}
	if unpriced > 0 {
		return nil, infraerrors.Conflict("RECOVERY_PRICING_INCOMPLETE", fmt.Sprintf("%d successful usage records are not priced", unpriced))
	}
	rows, err := r.db.QueryContext(ctx, `
WITH pool_accounts AS (
  SELECT a.id,a.name,a.provider_identity,a.created_at FROM accounts a
  WHERE a.deleted_at IS NULL AND a.cost_sharing_enabled=TRUE
), costs AS (
  SELECT a.id account_id,COALESCE(SUM(c.cny_amount_minor),0)::bigint net_cost_minor
  FROM pool_accounts a LEFT JOIN account_cost_entries c ON c.account_id=a.id AND c.paid_at<$1
  GROUP BY a.id
), usage_values AS (
  SELECT ul.id,ul.account_id,ul.created_at,
         ul.total_cost::numeric*rate.value*100 value_minor,
         DATE(ul.created_at AT TIME ZONE 'Asia/Shanghai') usage_day
  FROM usage_logs ul JOIN pool_accounts a ON a.id=ul.account_id
  JOIN LATERAL (
    SELECT fx.rate::numeric value FROM valuation_fx_rates fx
    WHERE fx.base_currency='USD' AND fx.quote_currency='CNY' AND fx.effective_from<=ul.created_at
    ORDER BY fx.effective_from DESC LIMIT 1
  ) rate ON TRUE
  WHERE ul.created_at<$1 AND ul.total_cost>0
), valued AS (
  SELECT account_id,ROUND(SUM(value_minor))::bigint value_minor
  FROM usage_values GROUP BY account_id
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
), banned_costs AS (
  SELECT b.account_id,COALESCE(SUM(c.cny_amount_minor),0)::numeric cost_minor
  FROM banned b LEFT JOIN account_cost_entries c ON c.account_id=b.account_id AND c.paid_at<=b.banned_at
  GROUP BY b.account_id
), banned_values AS (
  SELECT b.account_id,COALESCE(SUM(uv.value_minor),0)::numeric value_minor
  FROM banned b LEFT JOIN usage_values uv ON uv.account_id=b.account_id AND uv.created_at<=b.banned_at
  GROUP BY b.account_id
), banned_losses AS (
  SELECT b.account_id,GREATEST(ROUND(bc.cost_minor-bv.value_minor),0)::bigint loss_minor
  FROM banned b JOIN banned_costs bc ON bc.account_id=b.account_id JOIN banned_values bv ON bv.account_id=b.account_id
), refunds AS (
  SELECT a.id account_id,TRUE refunded FROM pool_accounts a
  WHERE EXISTS (SELECT 1 FROM account_cost_entries c WHERE c.account_id=a.id AND c.entry_type='refund' AND c.paid_at<$1)
     OR EXISTS (SELECT 1 FROM account_lifecycle_events e WHERE e.account_id=a.id AND e.event_type='refund' AND e.occurred_at<$1)
), financial_events AS (
  SELECT c.account_id,c.paid_at event_at,0 event_kind,c.id event_id,-c.cny_amount_minor::numeric delta
  FROM account_cost_entries c JOIN pool_accounts a ON a.id=c.account_id WHERE c.paid_at<$1
  UNION ALL
  SELECT uv.account_id,uv.created_at,1,uv.id,uv.value_minor FROM usage_values uv
), balances AS (
  SELECT e.*,SUM(e.delta) OVER(PARTITION BY e.account_id ORDER BY e.event_at,e.event_kind,e.event_id ROWS UNBOUNDED PRECEDING) balance
  FROM financial_events e
), transitions AS (
  SELECT b.*,LAG(b.balance) OVER(PARTITION BY b.account_id ORDER BY b.event_at,b.event_kind,b.event_id) previous_balance
  FROM balances b
), recoveries AS (
  SELECT account_id,
         MIN(event_at) FILTER(WHERE balance>=0 AND previous_balance<0) first_recovery_at,
         MAX(event_at) FILTER(WHERE balance>=0 AND previous_balance<0) latest_recovery_at
  FROM transitions GROUP BY account_id
), current_balances AS (
  SELECT DISTINCT ON (account_id) account_id,balance
  FROM balances ORDER BY account_id,event_at DESC,event_kind DESC,event_id DESC
), investments AS (
  SELECT c.account_id,TRUE has_investment FROM account_cost_entries c JOIN pool_accounts a ON a.id=c.account_id
  WHERE c.paid_at<$1 AND c.cny_amount_minor>0 GROUP BY c.account_id
)
SELECT a.id,a.name,a.provider_identity,source.name,COALESCE(lifecycle.event_type,'active'),
       COALESCE(c.net_cost_minor,0),COALESCE(v.value_minor,0),COALESCE(rd.avg_daily,0),
       investment_start.invested_at,banned.banned_at,COALESCE(refunds.refunded,FALSE),COALESCE(rd.effective_days,0),
       COALESCE(rd.observation_days,0),COALESCE(bl.loss_minor,0),
       recoveries.first_recovery_at,recoveries.latest_recovery_at,
       COALESCE(current_balances.balance>=0 AND investments.has_investment,FALSE)
FROM pool_accounts a
LEFT JOIN costs c ON c.account_id=a.id LEFT JOIN valued v ON v.account_id=a.id
LEFT JOIN recent_daily rd ON rd.account_id=a.id LEFT JOIN source ON source.account_id=a.id
LEFT JOIN investment_start ON investment_start.account_id=a.id
LEFT JOIN lifecycle ON lifecycle.account_id=a.id LEFT JOIN banned ON banned.account_id=a.id
LEFT JOIN banned_losses bl ON bl.account_id=a.id LEFT JOIN refunds ON refunds.account_id=a.id
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
		if err := rows.Scan(&item.AccountID, &item.AccountName, &item.ProviderIdentity, &item.PurchaseSource, &item.LifecycleStatus, &item.NetCostMinor, &item.ValueMinor, &item.AverageDailyValueMinor, &item.PurchasedAt, &item.BannedAt, &item.Refunded, &item.EffectiveUsageDays, &item.ObservationDays, &item.BannedLossMinor, &item.FirstRecoveryAt, &item.LatestRecoveryAt, &item.CurrentlyRecovered); err != nil {
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
	terminal := item.LifecycleStatus == "banned_confirmed" || item.LifecycleStatus == "retired" || item.LifecycleStatus == "replaced"
	if !item.CurrentlyRecovered && !terminal && item.UnrecoveredMinor > 0 && item.AverageDailyValueMinor > 0 && item.ObservationDays >= 7 && item.EffectiveUsageDays >= 3 {
		days := decimal.NewFromInt(item.UnrecoveredMinor).Div(decimal.NewFromInt(item.AverageDailyValueMinor)).Ceil().IntPart()
		item.EstimatedRecoveryDays = &days
	}
}

var _ service.PoolRepository = (*poolRepository)(nil)

// Keep source names stable for grouping while preserving their display form.
func normalizePurchaseSourceName(name string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(name)), " ")
}
