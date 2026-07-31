package repository

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestSettlementInputsAppliesAndPartitionsByFilter(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	repo := &poolRepository{db: db}

	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 0, 7)
	accountID, uploaderID, payerID, sourceID := int64(11), int64(12), int64(13), int64(14)

	mock.ExpectQuery(`(?s)SELECT c.id.*a.deleted_at IS NULL.*c.account_id=\$3.*a.created_by_user_id=\$4.*c.payer_user_id=\$5.*c.purchase_source_id=\$6`).
		WithArgs(start, end, accountID, uploaderID, payerID, sourceID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "account_id", "payer_user_id", "entry_type", "cny_amount_minor", "service_start", "service_end"}))
	mock.ExpectQuery(`(?s)SELECT a.id,ul.user_id.*a.deleted_at IS NULL.*EXISTS.*c.purchase_source_id=\$6.*GROUP BY`).
		WithArgs(start, end, accountID, uploaderID, payerID, sourceID).
		WillReturnRows(sqlmock.NewRows([]string{"account_id", "user_id", "email", "username", "weight"}))
	mock.ExpectQuery(`(?s)SELECT COUNT.*a.deleted_at IS NULL.*EXISTS.*c.purchase_source_id=\$6`).
		WithArgs(start, end, accountID, uploaderID, payerID, sourceID).
		WillReturnRows(sqlmock.NewRows([]string{"candidate_count", "unpriced_count", "candidate_material", "unpriced_material"}).AddRow(int64(0), int64(0), int64(0), int64(0)))
	mock.ExpectQuery(`filter_snapshot=\$2::jsonb`).
		WithArgs(start, `{"account_id":11,"uploader_user_id":12,"payer_user_id":13,"purchase_source_id":14}`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "carry_out_minor"}))

	_, _, _, _, err = repo.SettlementInputs(context.Background(), start, end, service.SettlementFilterSnapshot{
		AccountID: &accountID, UploaderUserID: &uploaderID,
		PayerUserID: &payerID, PurchaseSourceID: &sourceID,
	})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListSettlementsFiltersBeforePagination(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	repo := &poolRepository{db: db}
	accountID := int64(11)

	mock.ExpectQuery(`(?s)SELECT COUNT\(\*\).*pool_settlement_account_costs.*pool_settlement_account_lines`).
		WithArgs(&accountID).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(1)))
	mock.ExpectQuery(`(?s)SELECT id,period_type.*WHERE \(\$1::bigint IS NULL OR EXISTS.*ORDER BY period_start DESC,id DESC LIMIT \$2 OFFSET \$3`).
		WithArgs(&accountID, 20, 20).WillReturnRows(sqlmock.NewRows([]string{}))

	items, total, err := repo.ListSettlements(context.Background(), &accountID, 20, 20)
	require.NoError(t, err)
	require.Empty(t, items)
	require.Equal(t, int64(1), total)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListSettlementsLoadsAccountLines(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	repo := &poolRepository{db: db}
	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 0, 7)

	mock.ExpectQuery(`(?s)SELECT COUNT\(\*\).*pool_settlement_account_costs.*pool_settlement_account_lines`).
		WithArgs((*int64)(nil)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(1)))
	mock.ExpectQuery(`(?s)SELECT id,period_type.*ORDER BY period_start DESC,id DESC LIMIT \$2 OFFSET \$3`).
		WithArgs((*int64)(nil), 20, 0).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "period_type", "period_start", "period_end", "timezone", "status",
			"period_cost_minor", "carry_in_minor", "carry_out_minor", "total_cost_minor",
			"total_usage_weight", "pricing_coverage", "unpriced_usage_count", "fx_rate",
			"formula_version", "cost_snapshot", "filter_snapshot", "generated_by_user_id",
			"locked_by_user_id", "locked_at", "paid_by_user_id", "paid_at", "created_at", "updated_at",
		}).AddRow(
			int64(51), "weekly", start, end, "UTC", "draft",
			int64(100), int64(0), int64(0), int64(100),
			"1", "1", int64(0), "1", "v1", []byte(`[]`), []byte(`{}`), int64(9),
			nil, nil, nil, nil, start, start,
		))
	mock.ExpectQuery(`(?s)FROM pool_settlement_account_costs c.*WHERE c.settlement_id=\$1`).
		WithArgs(int64(51)).
		WillReturnRows(sqlmock.NewRows([]string{}))
	mock.ExpectQuery(`(?s)FROM accounts a.*pool_settlement_account_costs.*pool_settlement_account_lines.*ORDER BY a.id`).
		WithArgs(int64(51)).
		WillReturnRows(sqlmock.NewRows([]string{}))
	mock.ExpectQuery(`(?s)FROM pool_settlement_lines l.*WHERE l.settlement_id=\$1`).
		WithArgs(int64(51)).
		WillReturnRows(sqlmock.NewRows([]string{}))
	mock.ExpectQuery(`(?s)FROM pool_settlement_account_lines l.*WHERE l.settlement_id=\$1`).
		WithArgs(int64(51)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "settlement_id", "account_id", "user_id", "email", "username",
			"account_usage_weight", "usage_share", "allocated_cost_minor",
			"contribution_credit_minor", "adjustment_minor", "net_amount_minor", "trace_quality",
		}).AddRow(
			int64(71), int64(51), int64(11), int64(12), "member@example.com", "member",
			"1", "1", int64(100), int64(0), int64(0), int64(100), "priced",
		))

	items, total, err := repo.ListSettlements(context.Background(), nil, 20, 0)
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, items, 1)
	require.Len(t, items[0].AccountLines, 1)
	require.Equal(t, int64(11), items[0].AccountLines[0].AccountID)
	require.Equal(t, int64(12), items[0].AccountLines[0].UserID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSettlementAccountContextsProjectCurrentIdentityAndRuntime(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	repo := &poolRepository{db: db}
	createdAt := time.Date(2026, 7, 30, 8, 0, 0, 0, time.UTC)

	mock.ExpectQuery(`(?s)SELECT a.id, a.name, a.platform, a.type, a.created_at.*pool_settlement_account_costs.*pool_settlement_account_lines`).
		WithArgs(int64(51)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "platform", "type", "created_at", "import_batch_id", "provider_identity",
			"contributor_user_id", "contributor_email", "created_by_user_id", "created_by_email", "created_by_username",
			"cost_sharing_enabled", "account_status", "error_message", "schedulable", "rate_limited_at", "rate_limit_reset_at",
			"overload_until", "temp_unschedulable_until", "temp_unschedulable_reason", "expires_at", "auto_pause_on_expired",
			"lifecycle_status", "lifecycle_at", "net_cost_minor",
		}).AddRow(
			7, "account-7", "openai", "oauth", createdAt, "batch-7", "provider-7",
			nil, nil, 9, "owner@example.com", "owner", true,
			"error", "credential invalid", false, nil, nil, nil, nil, "", nil, true,
			"retired", nil, 1200,
		))

	items, err := repo.loadSettlementAccountContexts(context.Background(), 51)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, "batch-7", items[0].ImportBatchID)
	require.Equal(t, "oauth", items[0].Type)
	require.Equal(t, service.PoolAvailabilityError, items[0].AvailabilityStatus)
	require.Equal(t, "retired", items[0].LatestLifecycleStatus)
	require.NoError(t, mock.ExpectationsWereMet())
}
