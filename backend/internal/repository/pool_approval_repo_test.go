package repository

import (
	"context"
	"testing"

	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/sql"
	"github.com/DATA-DOG/go-sqlmock"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

func newPoolApprovalRepositoryTest(t *testing.T) (service.PoolApprovalRepository, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatal(err)
	}
	client := dbent.NewClient(dbent.Driver(sql.OpenDB(dialect.Postgres, db)))
	t.Cleanup(func() { _ = client.Close() })
	return NewPoolApprovalRepository(client), mock
}

func TestPoolApprovalDeleteImpactUsesAccountTraceRows(t *testing.T) {
	repo, mock := newPoolApprovalRepositoryTest(t)
	mock.ExpectQuery(`(?s)WITH RECURSIVE family AS.*doomed_costs\(id\) AS \(\s*SELECT id FROM account_cost_entries WHERE account_id IN \(SELECT id FROM family\)\s*UNION.*pool_settlement_account_costs.*pool_settlement_account_lines.*orphaned_purchase_sources`).
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{
			"accounts", "credentials", "schedules", "costs", "settlements", "settlement_costs", "settlement_lines",
			"mixed_settlements", "empty_settlements", "purchase_sources", "groups", "events", "usage",
		}).AddRow(
			int64(2), int64(4), int64(3), int64(5), int64(2), int64(6), int64(7),
			int64(1), int64(1), int64(2), int64(8), int64(9), int64(10),
		))

	reader := repo.(interface {
		GetAccountDeleteImpact(context.Context, int64) (*service.PoolAccountDeleteImpact, error)
	})
	impact, err := reader.GetAccountDeleteImpact(context.Background(), 7)
	if err != nil {
		t.Fatal(err)
	}
	if impact.Settlements != 2 || impact.SettlementAccountCosts != 6 || impact.SettlementAccountLines != 7 ||
		impact.MixedSettlements != 1 || impact.EmptySettlements != 1 || impact.PurchaseSources != 2 || impact.UsageRecords != 10 {
		t.Fatalf("unexpected delete impact: %#v", impact)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPoolApprovalCostUpdateChecksNormalizedSettlementCosts(t *testing.T) {
	repo, mock := newPoolApprovalRepositoryTest(t)
	mock.ExpectQuery(`SELECT account_id,COALESCE\(expected_token_count,0\)`).WithArgs(int64(8)).
		WillReturnRows(sqlmock.NewRows([]string{"account_id", "expected_token_count"}).AddRow(int64(3), int64(1000)))
	mock.ExpectQuery(`(?s)SELECT EXISTS\(.*pool_settlement_account_costs.*cost_entry_id=\$1`).WithArgs(int64(8)).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	err := repo.UpdateCostApproved(context.Background(), service.PoolCostUpdate{
		CostID: 8,
		Cost:   service.CreateAccountCostInput{AccountID: 3},
	})
	if infraerrors.Reason(err) != "POOL_COST_ALREADY_SETTLED" {
		t.Fatalf("unexpected settled cost result: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
