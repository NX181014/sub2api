package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

// CreateAccountIntake records account metadata, the normalized source and the
// initial cost in one transaction. A failed cost write leaves the account in
// its previous pool state so the attached dialog can be retried safely.
func (r *poolRepository) CreateAccountIntake(ctx context.Context, input service.CreateAccountIntakeInput) (*service.AccountIntakeResult, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin account intake: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err = tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, fmt.Sprintf("pool-cost:%d", input.AccountID)); err != nil {
		return nil, fmt.Errorf("lock account intake: %w", err)
	}
	if _, err = tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, "pool-source:"+strings.ToLower(input.PurchaseSourceName)); err != nil {
		return nil, fmt.Errorf("lock purchase source: %w", err)
	}

	var source service.PurchaseSource
	err = tx.QueryRowContext(ctx, `
SELECT id,name,website_url,notes,active,created_at,updated_at
FROM purchase_sources
WHERE LOWER(BTRIM(name))=LOWER(BTRIM($1))
ORDER BY id LIMIT 1`, input.PurchaseSourceName).Scan(
		&source.ID, &source.Name, &source.WebsiteURL, &source.Notes, &source.Active, &source.CreatedAt, &source.UpdatedAt,
	)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("find purchase source: %w", err)
	}
	if err == sql.ErrNoRows {
		err = tx.QueryRowContext(ctx, `
INSERT INTO purchase_sources(name,website_url)
VALUES($1,$2)
RETURNING id,name,website_url,notes,active,created_at,updated_at`, input.PurchaseSourceName, input.SourceWebsiteURL).Scan(
			&source.ID, &source.Name, &source.WebsiteURL, &source.Notes, &source.Active, &source.CreatedAt, &source.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("create purchase source: %w", err)
		}
	}

	result, err := tx.ExecContext(ctx, accountIntakeUpdateSQL, input.AccountID, input.ProviderIdentity, input.ContributorUserID, input.UploaderUserID, input.Cost.ExpectedTokenCount)
	if err != nil {
		return nil, fmt.Errorf("update intake account: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("read intake account result: %w", err)
	}
	if affected == 0 {
		var exists bool
		if err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM accounts WHERE id=$1 AND deleted_at IS NULL)`, input.AccountID).Scan(&exists); err != nil {
			return nil, fmt.Errorf("check intake account: %w", err)
		}
		if exists {
			return nil, infraerrors.Conflict("POOL_ACCOUNT_ALREADY_INTAKE", "account pool profile already exists; submit an approval request to change it")
		}
		return nil, service.ErrPoolAccountNotFound
	}

	if input.Cost.EntryType == "purchase" || input.Cost.EntryType == "renewal" || input.Cost.EntryType == "price_version" {
		var conflictID int64
		err = tx.QueryRowContext(ctx, `
SELECT id FROM account_cost_entries
WHERE account_id=$1 AND entry_type IN ('purchase','renewal','price_version')
  AND service_start<$3::date AND service_end>$2::date
LIMIT 1`, input.AccountID, input.Cost.ServiceStart, input.Cost.ServiceEnd).Scan(&conflictID)
		if err == nil {
			return nil, infraerrors.Conflict("COST_PERIOD_OVERLAP", fmt.Sprintf("service period overlaps cost entry %d", conflictID))
		}
		if err != sql.ErrNoRows {
			return nil, fmt.Errorf("check intake cost period: %w", err)
		}
	}

	var costID int64
	err = tx.QueryRowContext(ctx, `
INSERT INTO account_cost_entries(
    account_id,payer_user_id,purchase_source_id,entry_type,currency,original_amount,
    cny_amount_minor,fx_rate,service_start,service_end,warranty_end,paid_at,
	    order_no,purchase_url,note,supersedes_id,related_account_id,expected_token_count,created_by_user_id)
	VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19)
RETURNING id`, input.AccountID, input.ContributorUserID, source.ID, input.Cost.EntryType,
		input.Cost.Currency, input.Cost.OriginalAmount, input.Cost.CNYAmountMinor, input.Cost.FXRate,
		input.Cost.ServiceStart, input.Cost.ServiceEnd, input.Cost.WarrantyEnd, input.Cost.PaidAt,
		input.Cost.OrderNo, input.Cost.PurchaseURL, input.Cost.Note, input.Cost.SupersedesID,
		input.Cost.RelatedAccountID, input.Cost.ExpectedTokenCount, input.ActorUserID).Scan(&costID)
	if err != nil {
		return nil, fmt.Errorf("create intake cost: %w", err)
	}
	account, err := scanPoolAccount(tx.QueryRowContext(ctx, poolAccountSelect+` AND a.id=$1`, input.AccountID))
	if err != nil {
		return nil, fmt.Errorf("load intake account: %w", err)
	}
	cost, err := scanCost(tx.QueryRowContext(ctx, costSelect+` WHERE c.id=$1`, costID))
	if err != nil {
		return nil, fmt.Errorf("load intake cost: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit account intake: %w", err)
	}
	return &service.AccountIntakeResult{Account: *account, Source: source, Cost: *cost}, nil
}

const accountIntakeUpdateSQL = `
UPDATE accounts SET
    provider_identity=$2,
	    contributor_user_id=$3,
	    created_by_user_id=$4,
	    expected_token_count=$5,
	    cost_sharing_enabled=TRUE,updated_at=NOW()
WHERE id=$1 AND deleted_at IS NULL
  AND COALESCE(BTRIM(provider_identity), '') = ''
  AND contributor_user_id IS NULL
  AND cost_sharing_enabled = FALSE
  AND NOT EXISTS (SELECT 1 FROM account_cost_entries WHERE account_id=$1)`
