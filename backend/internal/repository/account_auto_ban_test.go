package repository

import (
	"context"
	"regexp"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestAccountRepositoryRecordAutomaticBanIsSerializedAndReportsInsert(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	repo := newAccountRepositoryWithSQL(nil, db, nil)
	detectedAt := time.Date(2026, time.July, 28, 1, 2, 3, 0, time.FixedZone("CST", 8*60*60))

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`)).
		WithArgs("account-auto-ban:42").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`(?s)INSERT INTO account_lifecycle_events.*recovered\.event_type = 'recovered'`).
		WithArgs(int64(42), detectedAt.UTC(), "account_suspended").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	inserted, err := repo.RecordAutomaticBan(context.Background(), 42, detectedAt, " account_suspended ")

	require.NoError(t, err)
	require.True(t, inserted)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAccountRepositoryRecordAutomaticBanReportsExistingPeriod(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	repo := newAccountRepositoryWithSQL(nil, db, nil)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`)).
		WithArgs("account-auto-ban:7").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`(?s)INSERT INTO account_lifecycle_events.*recovered\.event_type = 'recovered'`).
		WithArgs(int64(7), sqlmock.AnyArg(), "terms_of_service_violation").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()

	inserted, err := repo.RecordAutomaticBan(context.Background(), 7, time.Now(), "terms_of_service_violation")

	require.NoError(t, err)
	require.False(t, inserted)
	require.NoError(t, mock.ExpectationsWereMet())
}
