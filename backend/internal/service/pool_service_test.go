package service

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestCreateLifecycleRejectsFinancialBypass(t *testing.T) {
	service := &PoolService{}
	for _, eventType := range []string{"refund", "replaced"} {
		_, err := service.CreateLifecycle(context.Background(), CreateLifecycleEventInput{
			AccountID: 1, CreatedByUserID: 2, EventType: eventType,
		})
		require.Error(t, err)
	}
}

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

func TestBuildSettlementAllocationRollsUpExactly(t *testing.T) {
	costs := []SettlementCostSnapshot{
		{Kind: "period", EntryID: 101, AccountID: 1, PayerUserID: 1, AmountMinor: 60},
		{Kind: "period", EntryID: 102, AccountID: 2, PayerUserID: 2, AmountMinor: 40},
	}
	weights := []PoolUsageWeight{
		{AccountID: 1, UserID: 1, Weight: decimal.NewFromInt(1)},
		{AccountID: 2, UserID: 1, Weight: decimal.NewFromInt(1)},
		{AccountID: 2, UserID: 2, Weight: decimal.NewFromInt(1)},
	}
	lines, accountLines, totalWeight, carryOut := BuildSettlementAllocation(costs, weights)
	require.Equal(t, "3", totalWeight.String())
	require.Zero(t, carryOut)
	require.Len(t, lines, 2)
	require.Equal(t, int64(67), lines[0].AllocatedCostMinor)
	require.Equal(t, int64(60), lines[0].ContributionCreditMinor)
	require.Equal(t, int64(33), lines[1].AllocatedCostMinor)
	require.Equal(t, int64(40), lines[1].ContributionCreditMinor)

	byUserAllocated, byUserCredit := map[int64]int64{}, map[int64]int64{}
	for _, line := range accountLines {
		byUserAllocated[line.UserID] += line.AllocatedCostMinor
		byUserCredit[line.UserID] += line.ContributionCreditMinor
		require.Equal(t, "exact", line.TraceQuality)
	}
	require.Equal(t, map[int64]int64{1: 67, 2: 33}, byUserAllocated)
	require.Equal(t, map[int64]int64{1: 60, 2: 40}, byUserCredit)
	require.Equal(t, int64(34), accountLines[0].AllocatedCostMinor)
	require.Equal(t, int64(33), accountLines[1].AllocatedCostMinor)
}

func TestBuildSettlementAllocationCarriesWhenThereIsNoUsage(t *testing.T) {
	lines, accountLines, totalWeight, carryOut := BuildSettlementAllocation([]SettlementCostSnapshot{
		{Kind: "period", EntryID: 101, AccountID: 1, PayerUserID: 9, AmountMinor: 100},
	}, nil)
	require.True(t, totalWeight.IsZero())
	require.Equal(t, int64(100), carryOut)
	require.Len(t, lines, 1)
	require.Zero(t, lines[0].NetAmountMinor)
	require.Len(t, accountLines, 1)
	require.Zero(t, accountLines[0].NetAmountMinor)
}

func TestValidateSettlementFilter(t *testing.T) {
	accountID, uploaderID, payerID, sourceID := int64(1), int64(2), int64(3), int64(4)
	require.NoError(t, validateSettlementFilter(SettlementFilterSnapshot{
		AccountID: &accountID, UploaderUserID: &uploaderID,
		PayerUserID: &payerID, PurchaseSourceID: &sourceID,
	}))

	invalid := int64(0)
	require.Error(t, validateSettlementFilter(SettlementFilterSnapshot{AccountID: &invalid}))
}

func TestNormalizePoolCostInput(t *testing.T) {
	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	expectedTokens := int64(1_000_000)
	input := CreateAccountCostInput{
		AccountID: 1, PayerUserID: 2, CreatedByUserID: 3,
		EntryType: "purchase", Currency: " cny ", OriginalAmount: "20.00", CNYAmountMinor: 2000,
		FXRate: "1.0", ServiceStart: start, ServiceEnd: start.AddDate(0, 1, 0), ExpectedTokenCount: &expectedTokens,
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

	input.ServiceEnd = start.AddDate(0, 1, 0)
	input.ExpectedTokenCount = nil
	_, err = normalizePoolCostInput(input)
	require.Error(t, err)
}

func TestPrepareBatchCostInputsPerAccount(t *testing.T) {
	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	overrideAmount := "12"
	commonTokens, overrideTokens := int64(1_000_000), int64(2_000_000)
	items, totalOriginal, totalCNY, err := prepareBatchCostInputs(BatchCreateAccountCostsInput{
		AmountMode: "per_account",
		Common: CreateAccountCostInput{
			PayerUserID: 2, EntryType: "purchase", Currency: "CNY", OriginalAmount: "10", FXRate: "1",
			ServiceStart: start, ServiceEnd: start.AddDate(0, 1, 0), PaidAt: start,
		},
		Accounts: []BatchAccountCostItemInput{
			{AccountID: 3},
			{AccountID: 1, OriginalAmount: &overrideAmount, ExpectedTokenCount: &overrideTokens},
			{AccountID: 2},
		},
		ExpectedTokenCount: &commonTokens, CreatedByUserID: 9, OperationKey: "batch-1",
	})
	require.NoError(t, err)
	require.Equal(t, "32", totalOriginal)
	require.Equal(t, int64(3200), totalCNY)
	require.Equal(t, []int64{1000, 1200, 1000}, []int64{items[0].CNYAmountMinor, items[1].CNYAmountMinor, items[2].CNYAmountMinor})
	require.Equal(t, commonTokens, *items[0].ExpectedTokenCount)
	require.Equal(t, overrideTokens, *items[1].ExpectedTokenCount)
	require.Equal(t, "batch-1", items[0].OperationKey)
}

func TestPrepareBatchCostInputsOrderTotalUsesLargestRemainder(t *testing.T) {
	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	orderNo := "ORDER-1"
	expectedTokens := int64(1_000_000)
	items, totalOriginal, totalCNY, err := prepareBatchCostInputs(BatchCreateAccountCostsInput{
		AmountMode: "order_total",
		Common: CreateAccountCostInput{
			PayerUserID: 2, EntryType: "purchase", Currency: "CNY", OriginalAmount: "100", FXRate: "1",
			ServiceStart: start, ServiceEnd: start.AddDate(0, 1, 0), PaidAt: start, OrderNo: &orderNo,
		},
		Accounts:           []BatchAccountCostItemInput{{AccountID: 3}, {AccountID: 1}, {AccountID: 2}},
		ExpectedTokenCount: &expectedTokens, CreatedByUserID: 9, OperationKey: "batch-2",
	})
	require.NoError(t, err)
	require.Equal(t, "100", totalOriginal)
	require.Equal(t, int64(10000), totalCNY)
	byAccount := map[int64]int64{}
	for _, item := range items {
		byAccount[item.AccountID] = item.CNYAmountMinor
		require.NotEmpty(t, item.OrderAccountKey)
	}
	require.Equal(t, map[int64]int64{1: 3334, 2: 3333, 3: 3333}, byAccount)
}

func TestPrepareBatchCostInputsOrderTotalHonorsOverrides(t *testing.T) {
	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	fixed := "40"
	expectedTokens := int64(1_000_000)
	items, _, totalCNY, err := prepareBatchCostInputs(BatchCreateAccountCostsInput{
		AmountMode: "order_total",
		Common: CreateAccountCostInput{
			PayerUserID: 2, EntryType: "purchase", Currency: "CNY", OriginalAmount: "100", FXRate: "1",
			ServiceStart: start, ServiceEnd: start.AddDate(0, 1, 0), PaidAt: start,
		},
		Accounts:           []BatchAccountCostItemInput{{AccountID: 1}, {AccountID: 2, OriginalAmount: &fixed}, {AccountID: 3}},
		ExpectedTokenCount: &expectedTokens, CreatedByUserID: 9, OperationKey: "batch-3",
	})
	require.NoError(t, err)
	require.Equal(t, int64(10000), totalCNY)
	byAccount := map[int64]int64{}
	for _, item := range items {
		byAccount[item.AccountID] = item.CNYAmountMinor
	}
	require.Equal(t, map[int64]int64{1: 3000, 2: 4000, 3: 3000}, byAccount)

	tooMuch := "101"
	_, _, _, err = prepareBatchCostInputs(BatchCreateAccountCostsInput{
		AmountMode: "order_total", Common: CreateAccountCostInput{
			PayerUserID: 2, EntryType: "purchase", Currency: "CNY", OriginalAmount: "100", FXRate: "1",
			ServiceStart: start, ServiceEnd: start.AddDate(0, 1, 0), PaidAt: start,
		},
		Accounts:           []BatchAccountCostItemInput{{AccountID: 1, OriginalAmount: &tooMuch}, {AccountID: 2}},
		ExpectedTokenCount: &expectedTokens, CreatedByUserID: 9, OperationKey: "batch-4",
	})
	require.Error(t, err)
}

func TestPrepareBatchCostInputsRejectsDuplicateAccountsAndMissingKey(t *testing.T) {
	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	base := BatchCreateAccountCostsInput{
		AmountMode: "per_account",
		Common: CreateAccountCostInput{
			PayerUserID: 2, EntryType: "purchase", Currency: "CNY", OriginalAmount: "10", FXRate: "1",
			ServiceStart: start, ServiceEnd: start.AddDate(0, 1, 0), PaidAt: start,
		},
		Accounts: []BatchAccountCostItemInput{{AccountID: 1}, {AccountID: 1}}, CreatedByUserID: 9, OperationKey: "batch-5",
	}
	_, _, _, err := prepareBatchCostInputs(base)
	require.Error(t, err)
	base.Accounts = []BatchAccountCostItemInput{{AccountID: 1}}
	base.OperationKey = ""
	_, _, _, err = prepareBatchCostInputs(base)
	require.ErrorIs(t, err, ErrIdempotencyKeyRequired)
}

func TestApplyCostRecognition(t *testing.T) {
	expected := int64(1_000)
	item := AccountCostSummary{NetCostMinor: 10_000, CostBasisMinor: 10_000, TotalUsageTokens: 250, ExpectedTokenCount: &expected,
		CostTranches: []AccountCostTranche{{CostMinor: 10_000, ExpectedTokens: expected}}}
	applyCostRecognition(&item)
	require.Equal(t, int64(2500), item.RecognizedCostMinor)
	require.Equal(t, int64(7500), item.RemainingCostMinor)
	require.NotNil(t, item.CostProgress)
	require.Equal(t, "0.25", *item.CostProgress)

	item.TotalUsageTokens = 2_000
	applyCostRecognition(&item)
	require.Equal(t, int64(10_000), item.RecognizedCostMinor)
	require.Zero(t, item.RemainingCostMinor)

	item.TotalUsageTokens = 250
	item.WrittenOffMinor = 3_000
	applyCostRecognition(&item)
	require.Equal(t, int64(2500), item.RecognizedCostMinor)
	require.Equal(t, int64(4500), item.RemainingCostMinor)

	item.TotalUsageTokens = 1_000
	applyCostRecognition(&item)
	require.Equal(t, int64(7000), item.RecognizedCostMinor)
	require.Zero(t, item.RemainingCostMinor)

	item.TotalUsageTokens = 250
	item.WrittenOffMinor = 0
	item.RefundMinor = 7_500
	applyCostRecognition(&item)
	require.Equal(t, int64(2500), item.RecognizedCostMinor)
	require.Zero(t, item.RemainingCostMinor)

	item.RefundMinor = 0
	item.TransferredOutMinor = 7_500
	applyCostRecognition(&item)
	require.Equal(t, int64(2500), item.RecognizedCostMinor)
	require.Zero(t, item.RemainingCostMinor)
}

func TestApplyCostRecognitionUsesTranchePricesInPurchaseOrder(t *testing.T) {
	item := AccountCostSummary{
		CostBasisMinor:   40_000,
		TotalUsageTokens: 1_000,
		CostTranches: []AccountCostTranche{
			{CostMinor: 10_000, ExpectedTokens: 1_000},
			{CostMinor: 30_000, ExpectedTokens: 1_000},
		},
	}
	applyCostRecognition(&item)
	require.Equal(t, int64(10_000), item.RecognizedCostMinor)
	require.Equal(t, int64(30_000), item.RemainingCostMinor)
	require.Equal(t, "0.5", *item.CostProgress)

	item.TotalUsageTokens = 1_500
	applyCostRecognition(&item)
	require.Equal(t, int64(25_000), item.RecognizedCostMinor)
	require.Equal(t, int64(15_000), item.RemainingCostMinor)
}

func TestCostRecognitionDoesNotCarryPreRenewalOveruseIntoFutureTranche(t *testing.T) {
	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	tranches := []AccountCostTranche{
		{ID: 1, CostMinor: 10_000, ExpectedTokens: 1_000, PaidAt: start, UsageTokens: 1_500},
		{ID: 2, CostMinor: 30_000, ExpectedTokens: 1_000, PaidAt: start.AddDate(0, 1, 0), UsageTokens: 0},
	}
	recognized, remaining, progress := CalculateAccountCostRecognitionByTranches(40_000, 0, 1_500, tranches)
	require.Equal(t, int64(10_000), recognized)
	require.Equal(t, int64(30_000), remaining)
	require.Equal(t, "0.5", *progress)

	tranches[1].UsageTokens = 500
	recognized, remaining, progress = CalculateAccountCostRecognitionByTranches(40_000, 0, 2_000, tranches)
	require.Equal(t, int64(25_000), recognized)
	require.Equal(t, int64(15_000), remaining)
	require.Equal(t, "0.75", *progress)
}
