//go:build integration

package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestPoolRepository_GetRecoveryKeepsPhysicalAccountsSeparate(t *testing.T) {
	ctx := context.Background()
	stamp := time.Now().UnixNano()
	user := mustCreateUser(t, integrationEntClient, &service.User{
		Email: fmt.Sprintf("pool-%d@example.com", stamp),
	})
	original := mustCreateAccount(t, integrationEntClient, &service.Account{Name: fmt.Sprintf("pool-original-%d", stamp)})
	replacement := mustCreateAccount(t, integrationEntClient, &service.Account{Name: fmt.Sprintf("pool-replacement-%d", stamp)})
	key := mustCreateApiKey(t, integrationEntClient, &service.APIKey{UserID: user.ID, Key: fmt.Sprintf("sk-pool-%d", stamp)})

	t.Cleanup(func() {
		_, _ = integrationDB.ExecContext(ctx, `DELETE FROM account_lifecycle_events WHERE account_id IN ($1,$2)`, original.ID, replacement.ID)
		_, _ = integrationDB.ExecContext(ctx, `DELETE FROM account_cost_entries WHERE account_id IN ($1,$2)`, original.ID, replacement.ID)
		_, _ = integrationDB.ExecContext(ctx, `DELETE FROM valuation_fx_rates WHERE created_by_user_id=$1`, user.ID)
		_, _ = integrationDB.ExecContext(ctx, `DELETE FROM purchase_sources WHERE name=$1`, fmt.Sprintf("pool-source-%d", stamp))
		_, _ = integrationDB.ExecContext(ctx, `DELETE FROM usage_logs WHERE account_id IN ($1,$2)`, original.ID, replacement.ID)
		_, _ = integrationDB.ExecContext(ctx, `DELETE FROM api_keys WHERE id=$1`, key.ID)
		_, _ = integrationDB.ExecContext(ctx, `DELETE FROM accounts WHERE id IN ($1,$2)`, original.ID, replacement.ID)
		_, _ = integrationDB.ExecContext(ctx, `DELETE FROM users WHERE id=$1`, user.ID)
	})

	_, err := integrationEntClient.Account.UpdateOneID(original.ID).
		SetProviderIdentity("upstream@example.com").
		SetContributorUserID(user.ID).
		SetCreatedByUserID(user.ID).
		SetCostSharingEnabled(true).
		Save(ctx)
	require.NoError(t, err)
	_, err = integrationEntClient.Account.UpdateOneID(replacement.ID).
		SetCostSharingEnabled(true).
		Save(ctx)
	require.NoError(t, err)

	repo := NewPoolRepository(integrationDB)
	source, err := repo.CreateSource(ctx, service.CreatePurchaseSourceInput{Name: fmt.Sprintf("pool-source-%d", stamp)})
	require.NoError(t, err)

	loc, err := time.LoadLocation(service.PoolTimezone)
	require.NoError(t, err)
	date := func(day int) time.Time { return time.Date(2026, time.January, day, 0, 0, 0, 0, loc) }
	originalExpected := int64(1000)
	_, err = repo.CreateCost(ctx, service.CreateAccountCostInput{
		AccountID: original.ID, PayerUserID: user.ID, PurchaseSourceID: &source.ID,
		EntryType: "purchase", Currency: "CNY", OriginalAmount: "100.00", CNYAmountMinor: 10000, FXRate: "1",
		ServiceStart: date(1), ServiceEnd: date(31), PaidAt: date(1), CreatedByUserID: user.ID, ExpectedTokenCount: &originalExpected,
	})
	require.NoError(t, err)
	replacementExpected := int64(600)
	_, err = repo.CreateCost(ctx, service.CreateAccountCostInput{
		AccountID: replacement.ID, PayerUserID: user.ID,
		EntryType: "purchase", Currency: "CNY", OriginalAmount: "60.00", CNYAmountMinor: 6000, FXRate: "1",
		ServiceStart: date(1), ServiceEnd: date(31), PaidAt: date(1), CreatedByUserID: user.ID, ExpectedTokenCount: &replacementExpected,
	})
	require.NoError(t, err)

	addUsage := func(accountID int64, day int, tokens int64) {
		t.Helper()
		_, createErr := integrationEntClient.UsageLog.Create().
			SetUserID(user.ID).
			SetAPIKeyID(key.ID).
			SetAccountID(accountID).
			SetRequestID(fmt.Sprintf("pool-%d-%d", stamp, day)).
			SetModel("pool-test-model").
			SetInputTokens(int(tokens)).
			SetCreatedAt(date(day).Add(12 * time.Hour)).
			Save(ctx)
		require.NoError(t, createErr)
	}
	addUsage(original.ID, 2, 400)
	_, err = repo.CreateLifecycle(ctx, service.CreateLifecycleEventInput{
		AccountID: original.ID, EventType: "banned_confirmed", OccurredAt: date(2).Add(18 * time.Hour),
		CreatedByUserID: user.ID,
	})
	require.NoError(t, err)
	_, err = repo.CreateLifecycle(ctx, service.CreateLifecycleEventInput{
		AccountID: original.ID, EventType: "refund", OccurredAt: date(3).Add(time.Hour),
		RefundAmountMinor: 100, PayerUserID: &user.ID, CreatedByUserID: user.ID,
	})
	require.NoError(t, err)

	addUsage(replacement.ID, 4, 240)
	addUsage(replacement.ID, 5, 240)
	addUsage(replacement.ID, 7, 240) // Three effective days across seven observed natural days.

	items, err := repo.GetRecovery(ctx, date(1), date(10))
	require.NoError(t, err)
	var gotOriginal, gotReplacement *service.AccountRecovery
	for i := range items {
		if items[i].AccountID == original.ID {
			gotOriginal = &items[i]
		}
		if items[i].AccountID == replacement.ID {
			gotReplacement = &items[i]
		}
	}
	require.NotNil(t, gotOriginal)
	require.NotNil(t, gotReplacement)

	require.Equal(t, int64(10000), gotOriginal.NetCostMinor)
	require.Equal(t, int64(4100), gotOriginal.ValueMinor)
	require.Equal(t, int64(-5900), gotOriginal.NetProfitMinor)
	require.Equal(t, int64(5900), gotOriginal.BannedLossMinor)
	require.Equal(t, int64(5900), gotOriginal.CurrentNetLossMinor)
	require.True(t, gotOriginal.Refunded)
	require.False(t, gotOriginal.CurrentlyRecovered)
	require.Nil(t, gotOriginal.FirstRecoveryAt)
	require.Equal(t, "banned_confirmed", gotOriginal.LifecycleStatus)
	require.Equal(t, source.Name, *gotOriginal.PurchaseSource)

	require.Equal(t, int64(6000), gotReplacement.NetCostMinor)
	require.Equal(t, int64(6000), gotReplacement.ValueMinor)
	require.Zero(t, gotReplacement.NetProfitMinor)
	require.Equal(t, "1", gotReplacement.RecoveryRate)
	require.Equal(t, int64(103), gotReplacement.AverageDailyTokens)
	require.Equal(t, int64(3), gotReplacement.EffectiveUsageDays)
	require.Equal(t, int64(7), gotReplacement.ObservationDays)
	require.True(t, gotReplacement.CurrentlyRecovered)
	require.WithinDuration(t, date(7).Add(12*time.Hour), *gotReplacement.FirstRecoveryAt, time.Second)
	require.WithinDuration(t, *gotReplacement.FirstRecoveryAt, *gotReplacement.LatestRecoveryAt, time.Second)
	require.Equal(t, "active", gotReplacement.LifecycleStatus)
	require.Nil(t, gotReplacement.PurchaseSource)
}
