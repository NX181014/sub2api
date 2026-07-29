package repository

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

func newAccountLifecycleDeleteTestRepo(t *testing.T) (*accountRepository, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return newAccountRepositoryWithSQL(nil, db, nil), mock
}

func expectDeleteFamilyLock(mock sqlmock.Sqlmock, accountID int64, rows *sqlmock.Rows) {
	mock.ExpectQuery(`SELECT id, deleted_at IS NOT NULL`).WithArgs(accountID).WillReturnRows(rows)
}

func expectDeleteHistory(mock sqlmock.Sqlmock, accountID int64, usage, costs, lifecycle, decided, locked bool, net int64) {
	mock.ExpectQuery(`SELECT\s+EXISTS`).WithArgs(accountID).WillReturnRows(
		sqlmock.NewRows([]string{"usage", "costs", "lifecycle", "decided", "locked", "unpriced", "net", "written_off", "tokens", "tranches"}).
			AddRow(usage, costs, lifecycle, decided, locked, false, net, int64(0), int64(0), `[]`),
	)
}

func expectDeleteCostTranches(mock sqlmock.Sqlmock, accountID int64, rows *sqlmock.Rows) {
	mock.ExpectQuery(`WITH priced AS`).WithArgs(accountID).WillReturnRows(rows)
	mock.ExpectQuery(`SELECT payer_user_id,purchase_source_id,-SUM`).WithArgs(accountID).
		WillReturnRows(sqlmock.NewRows([]string{"payer_user_id", "purchase_source_id", "disposed"}))
}

func expectDeleteBindings(mock sqlmock.Sqlmock, accountID int64) {
	mock.ExpectExec(`DELETE FROM account_groups`).WithArgs(accountID).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`DELETE FROM scheduled_test_plans`).WithArgs(accountID).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE channel_account_stats_pricing_rules`).WithArgs(accountID).WillReturnResult(sqlmock.NewResult(0, 1))
}

func expectArchiveAccount(mock sqlmock.Sqlmock, accountID int64) {
	mock.ExpectExec(`UPDATE pool_approval_requests`).WithArgs(accountID).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE accounts SET deleted_at`).WithArgs(accountID).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO scheduler_outbox`).WithArgs(service.SchedulerOutboxEventAccountChanged, accountID).
		WillReturnResult(sqlmock.NewResult(1, 1))
}

func TestDeleteAccountWithLifecyclePhysicallyDeletesEmptyAccount(t *testing.T) {
	repo, mock := newAccountLifecycleDeleteTestRepo(t)
	mock.ExpectBegin()
	expectDeleteFamilyLock(mock, 7, sqlmock.NewRows([]string{"id", "deleted"}).AddRow(int64(7), false))
	expectDeleteHistory(mock, 7, false, false, false, false, false, 0)
	expectDeleteBindings(mock, 7)
	mock.ExpectExec(`DELETE FROM pool_approval_requests`).WithArgs(int64(7)).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`DELETE FROM accounts`).WithArgs(int64(7)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO scheduler_outbox`).WithArgs(service.SchedulerOutboxEventAccountChanged, int64(7)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	result, err := repo.DeleteAccountWithLifecycle(context.Background(), 7, service.AccountDeleteOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if result.Archived || result.AlreadyDeleted || len(result.AffectedAccountIDs) != 1 {
		t.Fatalf("unexpected physical deletion result: %#v", result)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestDeleteAccountWithLifecycleRejectsDispositionFieldsThatWillBeIgnored(t *testing.T) {
	replacementID := int64(2)
	repo := &accountRepository{}
	_, err := repo.DeleteAccountWithLifecycle(context.Background(), 1, service.AccountDeleteOptions{
		CostDisposition: "write_off", ReplacementAccountID: &replacementID,
	})
	if err == nil {
		t.Fatal("write-off disposition accepted a replacement account")
	}
	_, err = repo.DeleteAccountWithLifecycle(context.Background(), 1, service.AccountDeleteOptions{CostDisposition: "transfer"})
	if err == nil {
		t.Fatal("transfer disposition accepted a missing replacement account")
	}
}

func TestDeleteAccountWithLifecycleRequiresCostDisposition(t *testing.T) {
	repo, mock := newAccountLifecycleDeleteTestRepo(t)
	mock.ExpectBegin()
	expectDeleteFamilyLock(mock, 8, sqlmock.NewRows([]string{"id", "deleted"}).AddRow(int64(8), false))
	expectDeleteHistory(mock, 8, false, true, false, false, false, 1200)
	mock.ExpectRollback()

	_, err := repo.DeleteAccountWithLifecycle(context.Background(), 8, service.AccountDeleteOptions{ActorUserID: 2})
	if infraerrors.Reason(err) != "ACCOUNT_DELETE_COST_DISPOSITION_REQUIRED" {
		t.Fatalf("unexpected remaining-cost result: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestDeleteAccountWithLifecycleArchivesLockedSettlementSnapshot(t *testing.T) {
	repo, mock := newAccountLifecycleDeleteTestRepo(t)
	mock.ExpectBegin()
	expectDeleteFamilyLock(mock, 9, sqlmock.NewRows([]string{"id", "deleted"}).AddRow(int64(9), false))
	expectDeleteHistory(mock, 9, false, false, false, false, true, 0)
	expectDeleteBindings(mock, 9)
	expectArchiveAccount(mock, 9)
	mock.ExpectExec(`INSERT INTO account_lifecycle_events`).
		WithArgs(int64(9), "account archived", int64(3)).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	result, err := repo.DeleteAccountWithLifecycle(context.Background(), 9, service.AccountDeleteOptions{ActorUserID: 3})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Archived {
		t.Fatalf("locked settlement account must be archived: %#v", result)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestDeleteAccountWithLifecycleWritesOffRemainingCost(t *testing.T) {
	repo, mock := newAccountLifecycleDeleteTestRepo(t)
	now := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	mock.ExpectBegin()
	expectDeleteFamilyLock(mock, 10, sqlmock.NewRows([]string{"id", "deleted"}).AddRow(int64(10), false))
	expectDeleteHistory(mock, 10, false, true, false, false, false, 1500)
	expectDeleteCostTranches(mock, 10,
		sqlmock.NewRows([]string{"id", "payer_user_id", "purchase_source_id", "cost", "expected", "paid_at", "service_from", "service_to", "usage"}).
			AddRow(int64(1), int64(4), nil, int64(1500), int64(1000), now, now, now.AddDate(0, 1, 0), int64(0)),
	)
	mock.ExpectExec(`INSERT INTO account_cost_entries`).
		WithArgs(int64(10), int64(4), nil, "write_off", "-15.00", int64(-1500), now, now.AddDate(0, 1, 0),
			"account deletion write_off", nil, nil, int64(5), "account-delete:10:write_off:4:0").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`INSERT INTO account_lifecycle_events`).
		WithArgs(int64(10), "account deletion write_off", int64(5)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	expectDeleteBindings(mock, 10)
	expectArchiveAccount(mock, 10)
	mock.ExpectCommit()

	result, err := repo.DeleteAccountWithLifecycle(context.Background(), 10, service.AccountDeleteOptions{
		ActorUserID: 5, CostDisposition: "write_off",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Archived || result.RemainingCostMinor != 1500 || result.CostDisposition != "write_off" {
		t.Fatalf("unexpected write-off result: %#v", result)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestDeleteAccountWithLifecyclePartiallyRefundsByPayerAndRetires(t *testing.T) {
	repo, mock := newAccountLifecycleDeleteTestRepo(t)
	now := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	refund := int64(300)
	mock.ExpectBegin()
	expectDeleteFamilyLock(mock, 12, sqlmock.NewRows([]string{"id", "deleted"}).AddRow(int64(12), false))
	expectDeleteHistory(mock, 12, false, true, false, false, false, 1000)
	expectDeleteCostTranches(mock, 12,
		sqlmock.NewRows([]string{"id", "payer_user_id", "purchase_source_id", "cost", "expected", "paid_at", "service_from", "service_to", "usage"}).
			AddRow(int64(1), int64(4), int64(8), int64(600), int64(600), now, now, now.AddDate(0, 1, 0), int64(0)).
			AddRow(int64(2), int64(5), nil, int64(400), int64(400), now.Add(time.Second), now, now.AddDate(0, 1, 0), int64(0)),
	)
	mock.ExpectExec(`INSERT INTO account_cost_entries`).WithArgs(
		int64(12), int64(4), int64(8), "refund", "-1.80", int64(-180), now, now.AddDate(0, 1, 0),
		"partial vendor refund", nil, nil, int64(9), "account-delete:12:refund:4:8",
	).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`INSERT INTO account_cost_entries`).WithArgs(
		int64(12), int64(4), int64(8), "write_off", "-4.20", int64(-420), now, now.AddDate(0, 1, 0),
		"partial vendor refund", nil, nil, int64(9), "account-delete:12:write_off:4:8",
	).WillReturnResult(sqlmock.NewResult(2, 1))
	mock.ExpectExec(`INSERT INTO account_cost_entries`).WithArgs(
		int64(12), int64(5), nil, "refund", "-1.20", int64(-120), now, now.AddDate(0, 1, 0),
		"partial vendor refund", nil, nil, int64(9), "account-delete:12:refund:5:0",
	).WillReturnResult(sqlmock.NewResult(3, 1))
	mock.ExpectExec(`INSERT INTO account_cost_entries`).WithArgs(
		int64(12), int64(5), nil, "write_off", "-2.80", int64(-280), now, now.AddDate(0, 1, 0),
		"partial vendor refund", nil, nil, int64(9), "account-delete:12:write_off:5:0",
	).WillReturnResult(sqlmock.NewResult(4, 1))
	mock.ExpectExec(`INSERT INTO account_lifecycle_events`).WithArgs(int64(12), "partial vendor refund", int64(9)).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`INSERT INTO account_lifecycle_events`).WithArgs(int64(12), "partial vendor refund", int64(9)).WillReturnResult(sqlmock.NewResult(2, 1))
	expectDeleteBindings(mock, 12)
	expectArchiveAccount(mock, 12)
	mock.ExpectCommit()

	result, err := repo.DeleteAccountWithLifecycle(context.Background(), 12, service.AccountDeleteOptions{
		ActorUserID: 9, CostDisposition: "refund", RefundAmountMinor: &refund, Reason: "partial vendor refund",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.RefundAmountMinor != 300 || result.WrittenOffMinor != 700 {
		t.Fatalf("unexpected partial refund result: %#v", result)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestDeleteAccountWithLifecycleRejectsUnavailableReplacement(t *testing.T) {
	repo, mock := newAccountLifecycleDeleteTestRepo(t)
	now := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	replacementID := int64(22)
	mock.ExpectBegin()
	expectDeleteFamilyLock(mock, 13, sqlmock.NewRows([]string{"id", "deleted"}).AddRow(int64(13), false))
	expectDeleteHistory(mock, 13, false, true, false, false, false, 500)
	expectDeleteCostTranches(mock, 13,
		sqlmock.NewRows([]string{"id", "payer_user_id", "purchase_source_id", "cost", "expected", "paid_at", "service_from", "service_to", "usage"}).
			AddRow(int64(1), int64(4), nil, int64(500), int64(700), now, now, now.AddDate(0, 1, 0), int64(0)),
	)
	mock.ExpectQuery(`status='active' AND schedulable=TRUE AND cost_sharing_enabled=TRUE`).WithArgs(replacementID).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mock.ExpectRollback()

	_, err := repo.DeleteAccountWithLifecycle(context.Background(), 13, service.AccountDeleteOptions{
		ActorUserID: 9, CostDisposition: "transfer", ReplacementAccountID: &replacementID,
	})
	if infraerrors.Reason(err) != "ACCOUNT_DELETE_REPLACEMENT_NOT_AVAILABLE" {
		t.Fatalf("unexpected replacement validation result: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAccountDeleteTransferMovesRemainingExpectedTokens(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	now := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	replacementID := int64(22)
	expectDeleteCostTranches(mock, 13,
		sqlmock.NewRows([]string{"id", "payer_user_id", "purchase_source_id", "cost", "expected", "paid_at", "service_from", "service_to", "usage"}).
			AddRow(int64(1), int64(4), nil, int64(500), int64(700), now, now, now.AddDate(0, 1, 0), int64(0)),
	)
	mock.ExpectQuery(`status='active' AND schedulable=TRUE AND cost_sharing_enabled=TRUE`).WithArgs(replacementID).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(replacementID))
	mock.ExpectExec(`INSERT INTO account_cost_entries`).WithArgs(
		int64(13), int64(4), nil, "replacement_out", "-5.00", int64(-500), now, now.AddDate(0, 1, 0),
		"replacement", replacementID, nil, int64(9), "account-delete:13:replacement_out:4:0:22",
	).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`INSERT INTO account_cost_entries`).WithArgs(
		replacementID, int64(4), nil, "replacement_in", "5.00", int64(500), now, now.AddDate(0, 1, 0),
		"replacement", int64(13), int64(700), int64(9), "account-delete:22:replacement_in:4:0:13",
	).WillReturnResult(sqlmock.NewResult(2, 1))
	mock.ExpectExec(`UPDATE accounts SET expected_token_count`).WithArgs(replacementID, int64(700)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO account_lifecycle_events`).WithArgs(int64(13), "replacement", replacementID, int64(500), int64(9)).WillReturnResult(sqlmock.NewResult(3, 1))

	err = applyAccountDeleteCostDisposition(context.Background(), db, 13, 500, 700, service.AccountDeleteOptions{
		ActorUserID: 9, CostDisposition: "transfer", ReplacementAccountID: &replacementID, Reason: "replacement",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAllocateAccountDeleteAmountsUsesLargestRemainder(t *testing.T) {
	got := allocateAccountDeleteAmounts(2, []int64{1, 1, 1})
	want := []int64{1, 1, 0}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("allocation = %v, want %v", got, want)
		}
	}
}

func TestDeleteAccountWithLifecycleIsIdempotentAfterArchive(t *testing.T) {
	repo, mock := newAccountLifecycleDeleteTestRepo(t)
	mock.ExpectBegin()
	expectDeleteFamilyLock(mock, 11, sqlmock.NewRows([]string{"id", "deleted"}).AddRow(int64(11), true))
	mock.ExpectRollback()

	result, err := repo.DeleteAccountWithLifecycle(context.Background(), 11, service.AccountDeleteOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !result.AlreadyDeleted {
		t.Fatalf("expected idempotent result: %#v", result)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
