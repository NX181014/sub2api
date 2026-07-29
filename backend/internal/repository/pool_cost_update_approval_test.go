package repository

import (
	"context"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/sql"
	"github.com/DATA-DOG/go-sqlmock"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

func TestUpdateCostApprovedAdjustsExpectedTokensByDifference(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatal(err)
	}
	client := dbent.NewClient(dbent.Driver(sql.OpenDB(dialect.Postgres, db)))
	t.Cleanup(func() { _ = client.Close() })

	mock.ExpectQuery(`SELECT account_id,COALESCE\(expected_token_count,0\)`).WithArgs(int64(8)).
		WillReturnRows(sqlmock.NewRows([]string{"account_id", "expected_token_count"}).AddRow(int64(3), int64(1000)))
	mock.ExpectQuery(`SELECT EXISTS`).WithArgs(int64(8)).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	mock.ExpectExec(`UPDATE account_cost_entries SET`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE accounts SET`).WithArgs(int64(3), int64(500)).WillReturnResult(sqlmock.NewResult(0, 1))

	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	expected := int64(1500)
	repo := NewPoolApprovalRepository(client)
	err = repo.UpdateCostApproved(context.Background(), service.PoolCostUpdate{
		CostID: 8,
		Cost: service.CreateAccountCostInput{
			AccountID: 3, PayerUserID: 2, EntryType: "topup", Currency: "CNY",
			OriginalAmount: "10", CNYAmountMinor: 1000, FXRate: "1",
			ServiceStart: start, ServiceEnd: start.AddDate(0, 1, 0), PaidAt: start,
			ExpectedTokenCount: &expected,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err = mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
