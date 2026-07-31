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

	mock.ExpectQuery(`(?s)SELECT c.id.*c.account_id=\$3.*a.created_by_user_id=\$4.*c.payer_user_id=\$5.*c.purchase_source_id=\$6`).
		WithArgs(start, end, accountID, uploaderID, payerID, sourceID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "account_id", "payer_user_id", "entry_type", "cny_amount_minor", "service_start", "service_end"}))
	mock.ExpectQuery(`(?s)SELECT a.id,ul.user_id.*EXISTS.*c.purchase_source_id=\$6.*GROUP BY`).
		WithArgs(start, end, accountID, uploaderID, payerID, sourceID).
		WillReturnRows(sqlmock.NewRows([]string{"account_id", "user_id", "email", "username", "weight"}))
	mock.ExpectQuery(`(?s)SELECT COUNT.*EXISTS.*c.purchase_source_id=\$6`).
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
