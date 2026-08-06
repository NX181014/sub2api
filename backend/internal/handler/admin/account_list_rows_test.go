package admin

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func TestBuildAccountLogicalRowsGroupsBatchesAndSummarizesStatus(t *testing.T) {
	now := time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC)
	batchID := "7a7897f8-0b96-4d0f-994a-916613a66459"
	uploader := "alice"
	avatar := "https://cdn.example/alice.png"
	uploaderStatus := service.StatusDisabled
	accounts := []service.Account{
		{ID: 9, Name: "standalone", Status: service.StatusActive, Schedulable: true, CreatedAt: now},
		{ID: 8, Name: "batch-b", Status: service.StatusActive, Schedulable: true, CreatedAt: now, UploaderUsername: &uploader, UploaderAvatarURL: &avatar, UploaderStatus: &uploaderStatus, Extra: map[string]any{"import_batch_id": batchID}},
		{ID: 7, Name: "batch-a", Status: service.StatusError, Schedulable: true, CreatedAt: now.Add(-time.Minute), UploaderUsername: &uploader, Extra: map[string]any{"import_batch_id": batchID}},
	}

	rows := buildAccountLogicalRows(accounts, now, map[int64]int{8: 1})
	require.Len(t, rows, 2)
	require.Equal(t, "account", rows[0].Kind)
	require.Equal(t, "import_batch", rows[1].Kind)
	require.Equal(t, 2, rows[1].Batch.MatchedCount)
	require.Equal(t, 1, rows[1].Batch.SchedulableCount)
	require.Equal(t, 1, rows[1].Batch.Status.Normal)
	require.Equal(t, 1, rows[1].Batch.Status.Error)
	require.Equal(t, 1, rows[1].Batch.Status.InUse)
	require.Equal(t, 1, rows[1].Batch.Status.Available)
	require.Equal(t, 1, rows[1].Batch.Status.Faults)
	require.Equal(t, avatar, *rows[1].Batch.UploaderAvatarURL)
	require.Equal(t, uploaderStatus, *rows[1].Batch.UploaderStatus)
	require.Equal(t, now.Add(-time.Minute), rows[1].Batch.CreatedAt)
}

func TestSortAccountLogicalRowsUsesParentValues(t *testing.T) {
	now := time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC)
	rows := buildAccountLogicalRows([]service.Account{
		{ID: 3, Name: "newer", Status: service.StatusActive, Schedulable: true, CreatedAt: now},
		{ID: 2, Name: "batch-ok", Status: service.StatusActive, Schedulable: true, CreatedAt: now.Add(-time.Minute), Extra: map[string]any{"import_batch_id": "batch"}},
		{ID: 1, Name: "batch-error", Status: service.StatusError, CreatedAt: now.Add(-2 * time.Minute), Extra: map[string]any{"import_batch_id": "batch"}},
	}, now, nil)

	sortAccountLogicalRows(rows, "created_at", "desc", now)
	require.Equal(t, int64(3), rows[0].account.ID)
	require.Equal(t, "batch", rows[1].Batch.ID)

	sortAccountLogicalRows(rows, "status", "desc", now)
	require.Equal(t, "batch", rows[0].Batch.ID)
	require.Equal(t, int64(3), rows[1].account.ID)
}
