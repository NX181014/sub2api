package repository

import (
	"context"
	"regexp"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	sqlmock "github.com/DATA-DOG/go-sqlmock"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

func TestMihomoRepositorySyncNodesUpsertsAndTombstonesMissing(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	const subscriptionID int64 = 7
	observedAt := time.Date(2026, 8, 2, 4, 30, 0, 0, time.UTC)
	delay := 25
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id FROM mihomo_subscriptions WHERE id=$1 AND deleted_at IS NULL FOR UPDATE")).
		WithArgs(subscriptionID).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(subscriptionID))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO mihomo_nodes")).
		WithArgs(subscriptionID, "provider-a:node-a", "Node A", "Node A", true, delay, "JP", pq.Array([]string{"fast"}), false, observedAt).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE mihomo_nodes SET alive=FALSE, upstream_removed_at=$2, updated_at=NOW()")).
		WithArgs(subscriptionID, observedAt).
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectCommit()

	repo := NewMihomoRepository(db)
	err = repo.SyncNodes(context.Background(), subscriptionID, []service.MihomoManagedNode{{
		NodeKey: "provider-a:node-a", OriginalName: "Node A", Alive: true,
		DelayMS: &delay, Region: "JP", Tags: []string{"fast"},
	}}, observedAt)

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestMihomoRepositoryReusesOuterEntTransaction(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	client := dbent.NewClient(dbent.Driver(entsql.OpenDB(dialect.Postgres, db)))
	defer func() { _ = client.Close() }()
	mock.ExpectBegin()
	tx, err := client.Tx(context.Background())
	require.NoError(t, err)

	const subscriptionID int64 = 9
	observedAt := time.Date(2026, 8, 2, 5, 0, 0, 0, time.UTC)
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO mihomo_subscriptions")).
		WithArgs("Provider A", "provider-a", []byte{1, 2, 3}, "example.com", 3600, nil, nil, nil, "active", nil, "").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "provider_key", "url_ciphertext", "masked_host", "refresh_interval_seconds",
			"quota_total_bytes", "quota_used_bytes", "expires_at", "status", "last_refreshed_at",
			"last_error", "created_at", "updated_at", "deleted_at",
		}).AddRow(
			subscriptionID, "Provider A", "provider-a", []byte{1, 2, 3}, "example.com", 3600,
			nil, nil, nil, "active", nil, "", observedAt, observedAt, nil,
		))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id FROM mihomo_subscriptions WHERE id=$1 AND deleted_at IS NULL FOR UPDATE")).
		WithArgs(subscriptionID).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(subscriptionID))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE mihomo_nodes SET alive=FALSE, upstream_removed_at=$2, updated_at=NOW()")).
		WithArgs(subscriptionID, observedAt).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectRollback()

	repo := NewMihomoRepository(db)
	txCtx := dbent.NewTxContext(context.Background(), tx)
	item := &service.MihomoSubscription{
		Name: "Provider A", ProviderKey: "provider-a", URLCiphertext: []byte{1, 2, 3},
		MaskedHost: "example.com", RefreshIntervalSeconds: 3600, Status: "active",
	}
	err = repo.CreateSubscription(txCtx, item)
	require.NoError(t, err)
	require.Equal(t, subscriptionID, item.ID)
	err = repo.SyncNodes(txCtx, subscriptionID, nil, observedAt)
	require.NoError(t, err)
	require.NoError(t, tx.Rollback())
	require.NoError(t, mock.ExpectationsWereMet())
}
