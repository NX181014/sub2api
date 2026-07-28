package service

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestPoolSettlementMath(t *testing.T) {
	period, err := ResolveSettlementPeriod("week", "2026-07-29", "")
	require.NoError(t, err)
	require.Equal(t, "2026-07-27", period.Start.Format("2006-01-02"))
	require.Equal(t, "2026-08-03", period.End.Format("2006-01-02"))
	month, err := ResolveSettlementPeriod("month", "2026-07-29", "")
	require.NoError(t, err)
	require.Equal(t, "2026-07-01", month.Start.Format("2006-01-02"))
	require.Equal(t, "2026-08-01", month.End.Format("2006-01-02"))
	custom, err := ResolveSettlementPeriod("custom", "2026-07-29", "2026-07-29")
	require.NoError(t, err)
	require.Equal(t, "2026-07-30", custom.End.Format("2006-01-02"))

	utcStart := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	shanghai, err := time.LoadLocation(PoolTimezone)
	require.NoError(t, err)
	periodStart := time.Date(2026, 7, 1, 0, 0, 0, 0, shanghai)
	cost := PoolCostSlice{AmountMinor: 100, ServiceStart: utcStart, ServiceEnd: utcStart.AddDate(0, 0, 3)}
	require.Equal(t, int64(33), proratedCostMinor(cost, periodStart, periodStart.AddDate(0, 0, 1), 0))
	require.Equal(t, int64(33), proratedCostMinor(cost, periodStart.AddDate(0, 0, 2), periodStart.AddDate(0, 0, 3), 0))
	require.Equal(t, int64(34), proratedCostMinor(cost, periodStart.AddDate(0, 0, 2), periodStart.AddDate(0, 0, 3), 66))

	allocated, totalWeight := AllocateLargestRemainder(100, map[int64]decimal.Decimal{
		2: decimal.NewFromInt(1),
		1: decimal.NewFromInt(1),
		3: decimal.NewFromInt(1),
	})
	require.True(t, decimal.NewFromInt(3).Equal(totalWeight))
	require.Equal(t, map[int64]int64{1: 34, 2: 33, 3: 33}, allocated)
}

func TestNormalizePoolCostInput(t *testing.T) {
	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	input := CreateAccountCostInput{
		AccountID: 1, PayerUserID: 2, CreatedByUserID: 3,
		EntryType: "purchase", Currency: " cny ", OriginalAmount: "20.00", CNYAmountMinor: 2000,
		FXRate: "1.0", ServiceStart: start, ServiceEnd: start.AddDate(0, 1, 0),
	}

	normalized, err := normalizePoolCostInput(input)
	require.NoError(t, err)
	require.Equal(t, "CNY", normalized.Currency)
	require.Equal(t, "20", normalized.OriginalAmount)
	require.Equal(t, "1", normalized.FXRate)
	require.Equal(t, int64(2000), normalized.CNYAmountMinor)

	input.Currency, input.OriginalAmount, input.FXRate, input.CNYAmountMinor = "USD", "100", "7.2", 1
	normalized, err = normalizePoolCostInput(input)
	require.NoError(t, err)
	require.Equal(t, int64(72000), normalized.CNYAmountMinor)

	input.ServiceEnd = input.ServiceStart
	_, err = normalizePoolCostInput(input)
	require.Error(t, err)
}
