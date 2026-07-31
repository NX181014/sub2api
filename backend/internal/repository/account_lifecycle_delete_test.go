package repository

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func newAccountLifecycleDeleteTestRepo(t *testing.T) (*accountRepository, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return newAccountRepositoryWithSQL(nil, db, nil), mock
}

func TestDeleteAccountWithLifecycleHardDeletesFamily(t *testing.T) {
	repo, mock := newAccountLifecycleDeleteTestRepo(t)
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT id\s+FROM accounts`).WithArgs(int64(7)).WillReturnRows(
		sqlmock.NewRows([]string{"id"}).AddRow(int64(8)).AddRow(int64(7)),
	)
	mock.ExpectQuery(`SELECT s.id\s+FROM pool_settlements`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mock.ExpectQuery(`SELECT MIN\(created_at\),MAX\(created_at\) FROM usage_logs`).
		WillReturnRows(sqlmock.NewRows([]string{"min", "max"}).AddRow(nil, nil))
	for _, query := range []string{
		`DELETE FROM batch_image_jobs`,
		`DELETE FROM ops_system_logs`,
		`DELETE FROM ops_error_logs`,
		`WITH RECURSIVE doomed`,
		`UPDATE account_cost_entries SET related_account_id=NULL`,
		`DELETE FROM purchase_sources`,
		`DELETE FROM account_lifecycle_events`,
		`DELETE FROM pool_approval_requests`,
		`UPDATE channel_account_stats_pricing_rules`,
		`UPDATE groups g SET model_routing`,
		`UPDATE accounts SET extra`,
		`DELETE FROM scheduler_outbox`,
		`DELETE FROM accounts WHERE id=ANY`,
		`DELETE FROM accounts WHERE id=\$1`,
		`INSERT INTO scheduler_outbox`,
	} {
		mock.ExpectExec(query).WillReturnResult(sqlmock.NewResult(0, 1))
	}
	mock.ExpectCommit()

	result, err := repo.DeleteAccountWithLifecycle(context.Background(), 7, service.AccountDeleteOptions{})
	require.NoError(t, err)
	require.Equal(t, []int64{8, 7}, result.AffectedAccountIDs)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteAccountWithLifecycleRollsBackWhenCleanupFails(t *testing.T) {
	repo, mock := newAccountLifecycleDeleteTestRepo(t)
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT id\s+FROM accounts`).WithArgs(int64(7)).WillReturnRows(
		sqlmock.NewRows([]string{"id"}).AddRow(int64(7)),
	)
	mock.ExpectQuery(`SELECT s.id\s+FROM pool_settlements`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(51)))
	mock.ExpectExec(`DELETE FROM pool_settlement_account_costs`).WillReturnError(context.Canceled)
	mock.ExpectRollback()

	_, err := repo.DeleteAccountWithLifecycle(context.Background(), 7, service.AccountDeleteOptions{})
	require.ErrorIs(t, err, context.Canceled)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteAccountWithLifecycleReturnsNotFound(t *testing.T) {
	repo, mock := newAccountLifecycleDeleteTestRepo(t)
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT id\s+FROM accounts`).WithArgs(int64(99)).WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mock.ExpectRollback()

	_, err := repo.DeleteAccountWithLifecycle(context.Background(), 99, service.AccountDeleteOptions{})
	require.ErrorIs(t, err, service.ErrAccountNotFound)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestHardDeleteAccountSettlementsKeepsMixedSettlement(t *testing.T) {
	repo, mock := newAccountLifecycleDeleteTestRepo(t)
	mock.ExpectQuery(`SELECT s.id\s+FROM pool_settlements`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(51)))
	mock.ExpectExec(`DELETE FROM pool_settlement_account_costs`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`DELETE FROM pool_settlement_account_lines`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`SELECT account_id,cost_entry_id,kind,payer_user_id,amount_minor`).
		WithArgs(int64(51)).
		WillReturnRows(sqlmock.NewRows([]string{"account_id", "cost_entry_id", "kind", "payer_user_id", "amount_minor"}).
			AddRow(int64(2), int64(102), "period", int64(20), int64(100)))
	mock.ExpectQuery(`SELECT account_id,user_id,account_usage_weight::text`).
		WithArgs(int64(51)).
		WillReturnRows(sqlmock.NewRows([]string{"account_id", "user_id", "account_usage_weight"}).
			AddRow(int64(2), int64(20), "3"))
	mock.ExpectQuery(`SELECT status,generated_by_user_id`).WithArgs(int64(51)).
		WillReturnRows(sqlmock.NewRows([]string{"status", "generated_by_user_id", "locked_by_user_id", "locked_at", "paid_by_user_id", "paid_at"}).
			AddRow("draft", int64(20), nil, nil, nil, nil))
	mock.ExpectQuery(`WITH coverage AS`).WithArgs(int64(51), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"unpriced_count", "pricing_coverage"}).AddRow(int64(0), "1"))
	mock.ExpectExec(`DELETE FROM pool_settlement_lines`).WithArgs(int64(51)).WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectExec(`DELETE FROM pool_settlement_account_lines`).WithArgs(int64(51)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO pool_settlement_lines`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO pool_settlement_account_lines`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE pool_settlements`).WillReturnResult(sqlmock.NewResult(0, 1))

	err := hardDeleteAccountSettlements(context.Background(), repo.sql, []int64{1}, []string{"1"})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestHardDeleteAccountSettlementsDeletesOnlyEmptySettlement(t *testing.T) {
	repo, mock := newAccountLifecycleDeleteTestRepo(t)
	mock.ExpectQuery(`SELECT s.id\s+FROM pool_settlements`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(51)))
	mock.ExpectExec(`DELETE FROM pool_settlement_account_costs`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`DELETE FROM pool_settlement_account_lines`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`SELECT account_id,cost_entry_id,kind,payer_user_id,amount_minor`).
		WillReturnRows(sqlmock.NewRows([]string{"account_id", "cost_entry_id", "kind", "payer_user_id", "amount_minor"}))
	mock.ExpectQuery(`SELECT account_id,user_id,account_usage_weight::text`).
		WillReturnRows(sqlmock.NewRows([]string{"account_id", "user_id", "account_usage_weight"}))
	mock.ExpectExec(`DELETE FROM pool_settlements WHERE id=\$1`).WithArgs(int64(51)).WillReturnResult(sqlmock.NewResult(0, 1))

	err := hardDeleteAccountSettlements(context.Background(), repo.sql, []int64{1}, []string{"1"})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestHardDeleteAccountUsageRecomputesDashboardAggregates(t *testing.T) {
	repo, mock := newAccountLifecycleDeleteTestRepo(t)
	start := time.Date(2026, 7, 30, 8, 15, 0, 0, time.UTC)
	end := start.Add(2 * time.Hour)
	mock.ExpectQuery(`SELECT MIN\(created_at\),MAX\(created_at\) FROM usage_logs`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"min", "max"}).AddRow(start, end))
	mock.ExpectExec(`DELETE FROM usage_logs WHERE account_id=ANY`).
		WithArgs(sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectBegin()
	for _, query := range []string{
		`DELETE FROM usage_dashboard_hourly WHERE`,
		`DELETE FROM usage_dashboard_hourly_users WHERE`,
		`DELETE FROM usage_dashboard_daily WHERE`,
		`DELETE FROM usage_dashboard_daily_users WHERE`,
		`INSERT INTO usage_dashboard_hourly_users`,
		`INSERT INTO usage_dashboard_daily_users`,
		`INSERT INTO usage_dashboard_hourly`,
		`INSERT INTO usage_dashboard_daily`,
	} {
		mock.ExpectExec(query).WillReturnResult(sqlmock.NewResult(0, 1))
	}
	mock.ExpectCommit()

	err := hardDeleteAccountUsage(context.Background(), repo.sql, []int64{7, 8})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestHardDeleteAccountUsageDoesNotDeleteWithoutLogs(t *testing.T) {
	repo, mock := newAccountLifecycleDeleteTestRepo(t)
	mock.ExpectQuery(`SELECT MIN\(created_at\),MAX\(created_at\) FROM usage_logs`).
		WillReturnRows(sqlmock.NewRows([]string{"min", "max"}).AddRow(nil, nil))

	err := hardDeleteAccountUsage(context.Background(), repo.sql, []int64{7})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}
