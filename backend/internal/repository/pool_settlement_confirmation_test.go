package repository

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestConfirmSettlementLineConfirmsOnlyActorLine(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	repo := &poolRepository{db: db}

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT status FROM pool_settlements`).WithArgs(int64(9)).
		WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow("locked"))
	mock.ExpectQuery(`SELECT id,net_amount_minor,confirmation_status`).WithArgs(int64(9), int64(23)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "net_amount_minor", "confirmation_status"}).AddRow(int64(51), int64(1200), "pending"))
	mock.ExpectExec(`UPDATE pool_settlement_lines`).WithArgs(int64(51), int64(23)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	require.NoError(t, repo.ConfirmSettlementLine(context.Background(), 9, 23, 23))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestMarkSettlementPaidRequiresEveryNonZeroConfirmation(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	repo := &poolRepository{db: db}

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT status FROM pool_settlements`).WithArgs(int64(9)).
		WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow("locked"))
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM pool_settlement_lines`).WithArgs(int64(9)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(1)))
	mock.ExpectRollback()

	err = repo.MarkSettlementPaid(context.Background(), 9, 4)
	require.Equal(t, "SETTLEMENT_CONFIRMATIONS_PENDING", infraerrors.Reason(err))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestMarkSettlementPaidAfterEveryNonZeroConfirmation(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	repo := &poolRepository{db: db}

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT status FROM pool_settlements`).WithArgs(int64(9)).
		WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow("locked"))
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM pool_settlement_lines`).WithArgs(int64(9)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(0)))
	mock.ExpectExec(`UPDATE pool_settlement_lines`).WithArgs(int64(9)).WillReturnResult(sqlmock.NewResult(0, 3))
	mock.ExpectExec(`UPDATE pool_settlements`).WithArgs(int64(9), int64(4)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	require.NoError(t, repo.MarkSettlementPaid(context.Background(), 9, 4))
	require.NoError(t, mock.ExpectationsWereMet())
}
