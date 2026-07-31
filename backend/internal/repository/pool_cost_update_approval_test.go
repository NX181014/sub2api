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

func TestGetAccountDeleteImpactReturnsLifecycleCounts(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatal(err)
	}
	client := dbent.NewClient(dbent.Driver(sql.OpenDB(dialect.Postgres, db)))
	t.Cleanup(func() { _ = client.Close() })

	mock.ExpectQuery(`WITH RECURSIVE family AS`).WithArgs(int64(7)).WillReturnRows(
		sqlmock.NewRows([]string{"accounts", "credentials", "schedules", "costs", "settlements", "settlement_costs", "settlement_lines", "mixed", "empty", "sources", "groups", "events", "usage"}).
			AddRow(int64(2), int64(4), int64(1), int64(3), int64(2), int64(8), int64(9), int64(1), int64(1), int64(2), int64(5), int64(6), int64(7)),
	)
	repo, ok := NewPoolApprovalRepository(client).(interface {
		GetAccountDeleteImpact(context.Context, int64) (*service.PoolAccountDeleteImpact, error)
	})
	if !ok {
		t.Fatal("pool approval repository must expose delete impact preview")
	}
	impact, err := repo.GetAccountDeleteImpact(context.Background(), 7)
	if err != nil {
		t.Fatal(err)
	}
	if impact.Accounts != 2 || impact.CredentialKeys != 4 || impact.UsageRecords != 7 {
		t.Fatalf("unexpected delete impact: %#v", impact)
	}
	if impact.SettlementAccountCosts != 8 || impact.SettlementAccountLines != 9 || impact.MixedSettlements != 1 || impact.EmptySettlements != 1 || impact.PurchaseSources != 2 {
		t.Fatalf("unexpected settlement delete impact: %#v", impact)
	}
	if err = mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestListApprovalsScopesReviewableRequestsToOtherAdmins(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatal(err)
	}
	client := dbent.NewClient(dbent.Driver(sql.OpenDB(dialect.Postgres, db)))
	t.Cleanup(func() { _ = client.Close() })

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM pool_approval_requests r.*requested_by_user_id<>\$1.*status='pending'.*high_risk.*=\$2`).
		WithArgs(int64(5), true).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(0)))
	mock.ExpectQuery(`SELECT r.id, r.action_type.*requested_by_user_id<>\$1.*status='pending'.*high_risk.*=\$2.*LIMIT \$3 OFFSET \$4`).
		WithArgs(int64(5), true, 20, 0).WillReturnRows(sqlmock.NewRows([]string{"id"}))

	highRisk := true
	items, total, err := NewPoolApprovalRepository(client).ListApprovals(context.Background(), service.PoolApprovalFilter{
		Scope: "reviewable", ActorID: 5, HighRisk: &highRisk,
	}, 20, 0)
	if err != nil {
		t.Fatal(err)
	}
	if total != 0 || len(items) != 0 {
		t.Fatalf("unexpected scoped approvals: total=%d items=%#v", total, items)
	}
	if err = mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
