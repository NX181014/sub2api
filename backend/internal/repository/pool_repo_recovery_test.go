package repository

import (
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestFinalizeAccountRecoveryUsesNaturalDayProjectionAndBanCutoff(t *testing.T) {
	purchasedAt := time.Date(2026, time.July, 1, 8, 0, 0, 0, time.UTC)
	end := purchasedAt.Add(10 * 24 * time.Hour)
	item := service.AccountRecovery{
		NetCostMinor:           10000,
		ValueMinor:             4000,
		AverageDailyValueMinor: 1000,
		PurchasedAt:            &purchasedAt,
		EffectiveUsageDays:     3,
		ObservationDays:        7,
	}

	finalizeAccountRecovery(&item, end)
	require.Equal(t, int64(-6000), item.NetProfitMinor)
	require.Equal(t, int64(6000), item.UnrecoveredMinor)
	require.Equal(t, int64(6000), item.CurrentNetLossMinor)
	require.Equal(t, "0.4", item.RecoveryRate)
	require.Equal(t, int64(10), item.SurvivalDays)
	require.Equal(t, int64(6), *item.EstimatedRecoveryDays)

	bannedAt := purchasedAt.Add(4 * 24 * time.Hour)
	item.BannedAt = &bannedAt
	item.LifecycleStatus = "banned_confirmed"
	item.EstimatedRecoveryDays = nil
	finalizeAccountRecovery(&item, end)
	require.Equal(t, int64(6000), item.CurrentNetLossMinor)
	require.Equal(t, int64(4), item.SurvivalDays)
	require.Nil(t, item.EstimatedRecoveryDays)
}
