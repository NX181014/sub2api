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
	mock.ExpectExec(`(?s)DELETE FROM ops_retry_attempts.*pinned_account_id.*used_account_id.*source_error_id.*result_error_id.*result_usage_request_id`).
		WillReturnResult(sqlmock.NewResult(0, 4))
	mock.ExpectQuery(`SELECT MIN\(created_at\),MAX\(created_at\) FROM usage_logs`).
		WillReturnRows(sqlmock.NewRows([]string{"min", "max"}).AddRow(nil, nil))
	for _, query := range []string{
		`DELETE FROM pool_settlements`,
		`DELETE FROM batch_image_jobs`,
		`DELETE FROM ops_system_logs`,
		`DELETE FROM ops_error_logs`,
		`WITH RECURSIVE doomed`,
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
	mock.ExpectExec(`DELETE FROM ops_retry_attempts`).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(`SELECT MIN\(created_at\),MAX\(created_at\) FROM usage_logs`).
		WillReturnRows(sqlmock.NewRows([]string{"min", "max"}).AddRow(nil, nil))
	mock.ExpectExec(`DELETE FROM pool_settlements`).WillReturnError(context.Canceled)
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
