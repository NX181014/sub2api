package repository

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

const pricedTranchesJSON = `[{"cost_minor":10000,"expected_tokens":1000},{"cost_minor":30000,"expected_tokens":1000}]`
const earlyExcessTranchesJSON = `[{"id":1,"cost_minor":10000,"expected_tokens":1000,"paid_at":"2026-06-01T00:00:00Z","usage_tokens":2000},{"id":2,"cost_minor":30000,"expected_tokens":1000,"paid_at":"2026-07-01T00:00:00Z","usage_tokens":0}]`

func TestDecodePoolCostTranchesAcceptsPostgresDates(t *testing.T) {
	items, err := decodePoolCostTranches([]byte(`[{"id":9,"cost_minor":10000,"expected_tokens":1000,"paid_at":"2026-07-29T10:00:00+08:00","service_start":"2026-08-01","service_end":"2026-09-01"}]`))
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ServiceStart.Format(time.DateOnly) != "2026-08-01" || items[0].ServiceEnd.Format(time.DateOnly) != "2026-09-01" {
		t.Fatalf("unexpected decoded tranche: %#v", items)
	}
}

func TestPoolCostRecognitionQueriesUsePricedTranches(t *testing.T) {
	t.Run("uploader summary", func(t *testing.T) {
		db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = db.Close() }()
		mock.ExpectQuery(`SELECT COUNT`).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery(`WITH filtered AS`).WithArgs(20, 0).WillReturnRows(sqlmock.NewRows([]string{
			"uploader_id", "email", "username", "net", "basis", "disposed", "tranches",
		}).AddRow(8, "owner@example.com", "owner", 40000, 40000, 0, earlyExcessTranchesJSON))

		items, total, err := NewPoolRepository(db).ListCostUploaderSummaries(context.Background(), service.AccountCostSummaryFilter{}, 20, 0)
		if err != nil {
			t.Fatal(err)
		}
		if total != 1 || len(items) != 1 || items[0].AccountCount != 1 || items[0].RecognizedCostMinor != 10000 || items[0].RemainingCostMinor != 30000 {
			t.Fatalf("unexpected uploader summary: total=%d items=%#v", total, items)
		}
	})

	t.Run("cost summary", func(t *testing.T) {
		db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = db.Close() }()
		mock.ExpectQuery(`SELECT COUNT`).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery(`WITH filtered AS`).WithArgs(20, 0).WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "identity", "status", "uploader_id", "uploader", "uploader_username", "contributor_id", "contributor", "expected",
			"usage", "purchase", "refund", "transferred", "written_off", "basis", "net", "tranches", "entries",
			"lifecycle", "lifecycle_at", "payer_id", "payer", "source_id", "source", "order_no", "service_start", "service_end", "purchased_at",
		}).AddRow(9, "account-9", nil, "active", nil, nil, nil, nil, nil, 2000,
			1000, 40000, 0, 0, 0, 40000, 40000, pricedTranchesJSON, 2,
			"active", nil, nil, nil, nil, nil, nil, nil, nil, nil))

		items, _, err := NewPoolRepository(db).ListCostSummaries(context.Background(), service.AccountCostSummaryFilter{}, 20, 0)
		if err != nil {
			t.Fatal(err)
		}
		if len(items) != 1 || len(items[0].CostTranches) != 2 {
			t.Fatalf("unexpected cost summary tranches: %#v", items)
		}
	})

	t.Run("account list metrics", func(t *testing.T) {
		db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = db.Close() }()
		mock.ExpectQuery(`SELECT a.id,uploader.email`).WithArgs(int64(9)).WillReturnRows(sqlmock.NewRows([]string{
			"id", "uploader", "uploader_username", "expected", "basis", "refund", "transferred", "written_off", "net", "tranches", "usage", "lifecycle",
		}).AddRow(9, nil, nil, 2000, 40000, 0, 0, 0, 40000, pricedTranchesJSON, 1000, "active"))

		accounts := []service.Account{{ID: 9}}
		if err := newAccountRepositoryWithSQL(nil, db, nil).enrichAccountListPoolMetrics(context.Background(), accounts); err != nil {
			t.Fatal(err)
		}
		if accounts[0].PoolRemainingCostMinor != 30000 || accounts[0].PoolCostProgress == nil || *accounts[0].PoolCostProgress != "0.5" {
			t.Fatalf("unexpected account metrics: %#v", accounts[0])
		}
	})

	t.Run("recovery overview", func(t *testing.T) {
		db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = db.Close() }()
		end := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
		mock.ExpectQuery(`WITH RECURSIVE pool_accounts AS`).WithArgs(end).WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "identity", "uploader_id", "uploader_username", "uploaded_at", "source", "lifecycle", "basis", "value", "avg_tokens", "purchased_at", "banned_at",
			"refunded", "effective_days", "observation_days", "banned_loss", "first_recovery", "latest_recovery", "recovered",
			"expected_tokens", "used_tokens", "remaining_tokens",
		}).AddRow(9, "account-9", nil, nil, nil, end.AddDate(0, -2, 0), nil, "active", 40000, 10000, 100, nil, nil,
			false, 3, 7, 0, nil, nil, false, 2000, 1000, 1000))

		items, err := NewPoolRepository(db).GetRecovery(context.Background(), end.AddDate(0, -1, 0), end)
		if err != nil {
			t.Fatal(err)
		}
		if len(items) != 1 || items[0].UnrecoveredMinor != 30000 || items[0].EstimatedRecoveryDays == nil || *items[0].EstimatedRecoveryDays != 10 {
			t.Fatalf("unexpected recovery overview: %#v", items)
		}
	})
}
