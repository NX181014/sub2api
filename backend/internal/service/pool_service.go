package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/shopspring/decimal"
)

const PoolTimezone = "Asia/Shanghai"

var (
	ErrPoolAccountNotFound    = infraerrors.NotFound("POOL_ACCOUNT_NOT_FOUND", "pool account not found")
	ErrPoolSettlementNotFound = infraerrors.NotFound("POOL_SETTLEMENT_NOT_FOUND", "pool settlement not found")
)

type PoolAccount struct {
	ID                    int64      `json:"id"`
	Name                  string     `json:"name"`
	Platform              string     `json:"platform"`
	ProviderIdentity      *string    `json:"provider_identity"`
	ContributorUserID     *int64     `json:"contributor_user_id"`
	ContributorEmail      *string    `json:"contributor_email"`
	CreatedByUserID       *int64     `json:"created_by_user_id"`
	CreatedByEmail        *string    `json:"created_by_email"`
	CostSharingEnabled    bool       `json:"cost_sharing_enabled"`
	LatestLifecycleStatus string     `json:"latest_lifecycle_status"`
	LatestLifecycleAt     *time.Time `json:"latest_lifecycle_at"`
	NetCostMinor          int64      `json:"net_cost_minor"`
}

type UpdatePoolAccountInput struct {
	ProviderIdentity   *string `json:"provider_identity,omitempty"`
	ContributorUserID  *int64  `json:"contributor_user_id,omitempty"`
	CreatedByUserID    *int64  `json:"created_by_user_id,omitempty"`
	CostSharingEnabled *bool   `json:"cost_sharing_enabled,omitempty"`
}

type PurchaseSource struct {
	ID         int64     `json:"id"`
	Name       string    `json:"name"`
	WebsiteURL *string   `json:"website_url"`
	Notes      *string   `json:"notes"`
	Active     bool      `json:"active"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type CreatePurchaseSourceInput struct {
	Name       string
	WebsiteURL *string
	Notes      *string
}

type AccountCostEntry struct {
	ID               int64      `json:"id"`
	AccountID        int64      `json:"account_id"`
	AccountName      string     `json:"account_name"`
	PayerUserID      int64      `json:"payer_user_id"`
	PayerEmail       string     `json:"payer_email"`
	PurchaseSourceID *int64     `json:"purchase_source_id"`
	PurchaseSource   *string    `json:"purchase_source"`
	EntryType        string     `json:"entry_type"`
	Currency         string     `json:"currency"`
	OriginalAmount   string     `json:"original_amount"`
	CNYAmountMinor   int64      `json:"cny_amount_minor"`
	FXRate           string     `json:"fx_rate"`
	ServiceStart     time.Time  `json:"service_start"`
	ServiceEnd       time.Time  `json:"service_end"`
	WarrantyEnd      *time.Time `json:"warranty_end"`
	PaidAt           time.Time  `json:"paid_at"`
	OrderNo          *string    `json:"order_no,omitempty"`
	PurchaseURL      *string    `json:"purchase_url,omitempty"`
	Note             *string    `json:"note"`
	SupersedesID     *int64     `json:"supersedes_id"`
	RelatedAccountID *int64     `json:"related_account_id"`
	CreatedByUserID  int64      `json:"created_by_user_id"`
	CreatedAt        time.Time  `json:"created_at"`
}

type CreateAccountCostInput struct {
	AccountID        int64
	PayerUserID      int64
	PurchaseSourceID *int64
	EntryType        string
	Currency         string
	OriginalAmount   string
	CNYAmountMinor   int64
	FXRate           string
	ServiceStart     time.Time
	ServiceEnd       time.Time
	WarrantyEnd      *time.Time
	PaidAt           time.Time
	OrderNo          *string
	PurchaseURL      *string
	Note             *string
	SupersedesID     *int64
	RelatedAccountID *int64
	CreatedByUserID  int64
	OperationKey     string
}

type AccountLifecycleEvent struct {
	ID                   int64     `json:"id"`
	AccountID            int64     `json:"account_id"`
	AccountName          string    `json:"account_name"`
	EventType            string    `json:"event_type"`
	OccurredAt           time.Time `json:"occurred_at"`
	Reason               *string   `json:"reason"`
	ReplacementAccountID *int64    `json:"replacement_account_id"`
	TransferredCostMinor int64     `json:"transferred_cost_minor"`
	Source               string    `json:"source"`
	CreatedByUserID      *int64    `json:"created_by_user_id"`
	CreatedAt            time.Time `json:"created_at"`
}

type CreateLifecycleEventInput struct {
	AccountID            int64
	EventType            string
	OccurredAt           time.Time
	Reason               *string
	ReplacementAccountID *int64
	TransferredCostMinor int64
	RefundAmountMinor    int64
	PayerUserID          *int64
	CreatedByUserID      int64
}

type ValuationFXRate struct {
	ID              int64     `json:"id"`
	BaseCurrency    string    `json:"base_currency"`
	QuoteCurrency   string    `json:"quote_currency"`
	Rate            string    `json:"rate"`
	EffectiveFrom   time.Time `json:"effective_from"`
	Source          *string   `json:"source"`
	CreatedByUserID int64     `json:"created_by_user_id"`
	CreatedAt       time.Time `json:"created_at"`
}

type CreateFXRateInput struct {
	BaseCurrency    string
	QuoteCurrency   string
	Rate            string
	EffectiveFrom   time.Time
	Source          *string
	CreatedByUserID int64
}

type SettlementPeriod struct {
	Type     string
	Start    time.Time
	End      time.Time
	Timezone string
}

type PoolUsageWeight struct {
	UserID   int64
	Email    string
	Username string
	Weight   decimal.Decimal
}

type PoolUsageCoverage struct {
	CandidateCount int64
	UnpricedCount  int64
}

type PoolCostSlice struct {
	EntryID      int64
	AccountID    int64
	PayerUserID  int64
	EntryType    string
	AmountMinor  int64
	ServiceStart time.Time
	ServiceEnd   time.Time
}

type SettlementCostSnapshot struct {
	Kind        string `json:"kind"`
	EntryID     int64  `json:"entry_id"`
	AccountID   int64  `json:"account_id"`
	PayerUserID int64  `json:"payer_user_id"`
	AmountMinor int64  `json:"amount_minor"`
}

type PoolSettlementLine struct {
	ID                      int64  `json:"id"`
	SettlementID            int64  `json:"settlement_id"`
	UserID                  int64  `json:"user_id"`
	UserEmail               string `json:"user_email"`
	Username                string `json:"username"`
	UsageWeight             string `json:"usage_weight"`
	UsageShare              string `json:"usage_share"`
	AllocatedCostMinor      int64  `json:"allocated_cost_minor"`
	ContributionCreditMinor int64  `json:"contribution_credit_minor"`
	AdjustmentMinor         int64  `json:"adjustment_minor"`
	NetAmountMinor          int64  `json:"net_amount_minor"`
	PaymentStatus           string `json:"payment_status"`
}

type PoolSettlement struct {
	ID               int64                    `json:"id"`
	PeriodType       string                   `json:"period_type"`
	PeriodStart      time.Time                `json:"period_start"`
	PeriodEnd        time.Time                `json:"period_end"`
	Timezone         string                   `json:"timezone"`
	Status           string                   `json:"status"`
	PeriodCostMinor  int64                    `json:"period_cost_minor"`
	CarryInMinor     int64                    `json:"carry_in_minor"`
	CarryOutMinor    int64                    `json:"carry_out_minor"`
	TotalCostMinor   int64                    `json:"total_cost_minor"`
	TotalUsageWeight string                   `json:"total_usage_weight"`
	PricingCoverage  string                   `json:"pricing_coverage"`
	UnpricedCount    int64                    `json:"unpriced_usage_count"`
	FXRate           string                   `json:"fx_rate"`
	FormulaVersion   string                   `json:"formula_version"`
	CostSnapshot     []SettlementCostSnapshot `json:"cost_snapshot"`
	GeneratedBy      int64                    `json:"generated_by_user_id"`
	LockedBy         *int64                   `json:"locked_by_user_id"`
	LockedAt         *time.Time               `json:"locked_at"`
	CreatedAt        time.Time                `json:"created_at"`
	UpdatedAt        time.Time                `json:"updated_at"`
	Lines            []PoolSettlementLine     `json:"lines"`
}

type AccountRecovery struct {
	AccountID              int64      `json:"account_id"`
	AccountName            string     `json:"account_name"`
	ProviderIdentity       *string    `json:"provider_identity"`
	PurchaseSource         *string    `json:"purchase_source"`
	LifecycleStatus        string     `json:"lifecycle_status"`
	NetCostMinor           int64      `json:"net_cost_minor"`
	ValueMinor             int64      `json:"value_minor"`
	UnrecoveredMinor       int64      `json:"unrecovered_minor"`
	NetProfitMinor         int64      `json:"net_profit_minor"`
	BannedLossMinor        int64      `json:"banned_loss_minor"`
	CurrentNetLossMinor    int64      `json:"current_net_loss_minor"`
	RecoveryRate           string     `json:"recovery_rate"`
	AverageDailyValueMinor int64      `json:"average_daily_value_minor"`
	EstimatedRecoveryDays  *int64     `json:"estimated_recovery_days"`
	FirstRecoveryAt        *time.Time `json:"first_recovery_at"`
	LatestRecoveryAt       *time.Time `json:"latest_recovery_at"`
	CurrentlyRecovered     bool       `json:"currently_recovered"`
	PurchasedAt            *time.Time `json:"purchased_at"`
	BannedAt               *time.Time `json:"banned_at"`
	Refunded               bool       `json:"refunded"`
	SurvivalDays           int64      `json:"survival_days"`
	EffectiveUsageDays     int64      `json:"effective_usage_days"`
	ObservationDays        int64      `json:"observation_days"`
}

type PurchaseSourceRecovery struct {
	Name                string `json:"name"`
	AccountCount        int    `json:"account_count"`
	SampleSize          int    `json:"sample_size"`
	PurchaseCostMinor   int64  `json:"purchase_cost_minor"`
	ValueMinor          int64  `json:"value_minor"`
	RecoveryRate        string `json:"recovery_rate"`
	BanRate7Days        string `json:"ban_rate_7d"`
	BanRate30Days       string `json:"ban_rate_30d"`
	BanRate90Days       string `json:"ban_rate_90d"`
	RefundRate          string `json:"refund_rate"`
	AverageSurvivalDays string `json:"average_survival_days"`
	RankEligible        bool   `json:"rank_eligible"`
}

type PoolRecoveryOverview struct {
	Start             time.Time                `json:"start_at"`
	End               time.Time                `json:"end_at"`
	TotalCostMinor    int64                    `json:"total_cost_minor"`
	TotalValueMinor   int64                    `json:"total_value_minor"`
	UnrecoveredMinor  int64                    `json:"unrecovered_minor"`
	BannedLossMinor   int64                    `json:"banned_loss_minor"`
	RecoveryRate      string                   `json:"recovery_rate"`
	RecoveredAccounts int                      `json:"recovered_accounts"`
	TotalAccounts     int                      `json:"total_accounts"`
	Accounts          []AccountRecovery        `json:"accounts"`
	SourceStats       []PurchaseSourceRecovery `json:"source_stats"`
}

type PoolRepository interface {
	ListAccounts(ctx context.Context) ([]PoolAccount, error)
	UpdateAccount(ctx context.Context, id int64, input UpdatePoolAccountInput) (*PoolAccount, error)
	ListSources(ctx context.Context) ([]PurchaseSource, error)
	CreateSource(ctx context.Context, input CreatePurchaseSourceInput) (*PurchaseSource, error)
	ListCosts(ctx context.Context, accountID *int64) ([]AccountCostEntry, error)
	CreateCost(ctx context.Context, input CreateAccountCostInput) (*AccountCostEntry, error)
	CreateAccountIntake(ctx context.Context, input CreateAccountIntakeInput) (*AccountIntakeResult, error)
	ListLifecycle(ctx context.Context, accountID *int64) ([]AccountLifecycleEvent, error)
	CreateLifecycle(ctx context.Context, input CreateLifecycleEventInput) (*AccountLifecycleEvent, error)
	ListFXRates(ctx context.Context) ([]ValuationFXRate, error)
	CreateFXRate(ctx context.Context, input CreateFXRateInput) (*ValuationFXRate, error)
	LatestFXRate(ctx context.Context, at time.Time) (decimal.Decimal, error)
	SettlementInputs(ctx context.Context, start, end time.Time) ([]PoolCostSlice, []PoolUsageWeight, PoolUsageCoverage, []SettlementCostSnapshot, error)
	LockedAllocatedByCostEntry(ctx context.Context, ids []int64) (map[int64]int64, error)
	SaveDraftSettlement(ctx context.Context, settlement *PoolSettlement) (*PoolSettlement, error)
	LockSettlement(ctx context.Context, id, actorID int64) (*PoolSettlement, error)
	ListSettlements(ctx context.Context, limit, offset int) ([]PoolSettlement, int64, error)
	GetSettlement(ctx context.Context, id int64) (*PoolSettlement, error)
	GetRecovery(ctx context.Context, start, end time.Time) ([]AccountRecovery, error)
}

type PoolService struct {
	repo         PoolRepository
	approvalRepo PoolApprovalRepository
	adminService AdminService
	entClient    *dbent.Client
	settings     *SettingService
	tokenCache   TokenCacheInvalidator
}

func NewPoolService(repo PoolRepository, approvalRepo PoolApprovalRepository, adminService AdminService, entClient *dbent.Client, settings *SettingService) *PoolService {
	return &PoolService{repo: repo, approvalRepo: approvalRepo, adminService: adminService, entClient: entClient, settings: settings}
}

func (s *PoolService) SetTokenCacheInvalidator(invalidator TokenCacheInvalidator) {
	if s != nil {
		s.tokenCache = invalidator
	}
}

func (s *PoolService) ListAccounts(ctx context.Context) ([]PoolAccount, error) {
	return s.repo.ListAccounts(ctx)
}

func (s *PoolService) UpdateAccount(ctx context.Context, id int64, input UpdatePoolAccountInput) (*PoolAccount, error) {
	if id <= 0 {
		return nil, infraerrors.BadRequest("INVALID_ACCOUNT_ID", "invalid account id")
	}
	if input.ProviderIdentity != nil {
		v := strings.TrimSpace(*input.ProviderIdentity)
		if len(v) > 255 {
			return nil, infraerrors.BadRequest("INVALID_PROVIDER_IDENTITY", "provider identity is too long")
		}
		input.ProviderIdentity = &v
	}
	return s.repo.UpdateAccount(ctx, id, input)
}

func (s *PoolService) ListSources(ctx context.Context) ([]PurchaseSource, error) {
	return s.repo.ListSources(ctx)
}

func (s *PoolService) CreateSource(ctx context.Context, input CreatePurchaseSourceInput) (*PurchaseSource, error) {
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" || len(input.Name) > 100 {
		return nil, infraerrors.BadRequest("INVALID_SOURCE_NAME", "source name is required and must not exceed 100 characters")
	}
	return s.repo.CreateSource(ctx, input)
}

func (s *PoolService) ListCosts(ctx context.Context, accountID *int64) ([]AccountCostEntry, error) {
	return s.repo.ListCosts(ctx, accountID)
}

func (s *PoolService) CreateCost(ctx context.Context, input CreateAccountCostInput) (*AccountCostEntry, error) {
	var err error
	input, err = normalizePoolCostInput(input)
	if err != nil {
		return nil, err
	}
	return s.repo.CreateCost(ctx, input)
}

func normalizePoolCostInput(input CreateAccountCostInput) (CreateAccountCostInput, error) {
	if input.AccountID <= 0 || input.PayerUserID <= 0 || input.CreatedByUserID <= 0 {
		return input, infraerrors.BadRequest("INVALID_COST_PARTY", "account, payer and creator are required")
	}
	validType := map[string]bool{"purchase": true, "renewal": true, "topup": true, "price_version": true, "refund": true, "adjustment": true}
	if !validType[input.EntryType] {
		return input, infraerrors.BadRequest("INVALID_COST_TYPE", "invalid cost entry type")
	}
	if !input.ServiceEnd.After(input.ServiceStart) {
		return input, infraerrors.BadRequest("INVALID_SERVICE_PERIOD", "service_end must be after service_start")
	}
	input.Currency = strings.ToUpper(strings.TrimSpace(input.Currency))
	if len(input.Currency) != 3 {
		return input, infraerrors.BadRequest("INVALID_CURRENCY", "currency must be a three-letter code")
	}
	amount, err := decimal.NewFromString(strings.TrimSpace(input.OriginalAmount))
	if err != nil {
		return input, infraerrors.BadRequest("INVALID_ORIGINAL_AMOUNT", "original_amount must be decimal")
	}
	rate, err := decimal.NewFromString(strings.TrimSpace(input.FXRate))
	if err != nil || !rate.IsPositive() {
		return input, infraerrors.BadRequest("INVALID_FX_RATE", "fx_rate must be positive")
	}
	if input.Currency == "CNY" && !rate.Equal(decimal.NewFromInt(1)) {
		return input, infraerrors.BadRequest("INVALID_FX_RATE", "CNY costs must use fx_rate 1")
	}
	// The server owns the accounting amount; callers cannot provide a conflicting CNY value.
	input.CNYAmountMinor = amount.Mul(rate).Mul(decimal.NewFromInt(100)).Round(0).IntPart()
	if input.EntryType == "refund" {
		if !amount.IsNegative() || input.CNYAmountMinor >= 0 {
			return input, infraerrors.BadRequest("INVALID_REFUND_AMOUNT", "refund amounts must be negative")
		}
	} else if input.EntryType != "adjustment" && (!amount.IsPositive() || input.CNYAmountMinor <= 0) {
		return input, infraerrors.BadRequest("INVALID_COST_AMOUNT", "cost amounts must be positive")
	}
	input.OriginalAmount = amount.String()
	input.FXRate = rate.String()
	return input, nil
}

func (s *PoolService) ListLifecycle(ctx context.Context, accountID *int64) ([]AccountLifecycleEvent, error) {
	return s.repo.ListLifecycle(ctx, accountID)
}

func (s *PoolService) CreateLifecycle(ctx context.Context, input CreateLifecycleEventInput) (*AccountLifecycleEvent, error) {
	valid := map[string]bool{"banned_confirmed": true, "recovered": true, "refund": true, "replaced": true, "retired": true}
	if input.AccountID <= 0 || input.CreatedByUserID <= 0 || !valid[input.EventType] {
		return nil, infraerrors.BadRequest("INVALID_LIFECYCLE_EVENT", "invalid lifecycle event")
	}
	if input.EventType == "replaced" {
		if input.ReplacementAccountID == nil || *input.ReplacementAccountID <= 0 || input.TransferredCostMinor < 0 || (input.TransferredCostMinor > 0 && input.PayerUserID == nil) {
			return nil, infraerrors.BadRequest("INVALID_REPLACEMENT", "replacement account and transfer payer are required")
		}
		if *input.ReplacementAccountID == input.AccountID {
			return nil, infraerrors.BadRequest("INVALID_REPLACEMENT", "replacement account must be different")
		}
	} else {
		input.ReplacementAccountID = nil
		input.TransferredCostMinor = 0
	}
	if input.EventType == "refund" {
		if input.RefundAmountMinor < 0 || (input.RefundAmountMinor > 0 && input.PayerUserID == nil) {
			return nil, infraerrors.BadRequest("INVALID_REFUND", "refund amount and payer are invalid")
		}
	} else {
		input.RefundAmountMinor = 0
	}
	if input.TransferredCostMinor == 0 && input.RefundAmountMinor == 0 {
		input.PayerUserID = nil
	}
	return s.repo.CreateLifecycle(ctx, input)
}

func (s *PoolService) ListFXRates(ctx context.Context) ([]ValuationFXRate, error) {
	return s.repo.ListFXRates(ctx)
}

func (s *PoolService) CreateFXRate(ctx context.Context, input CreateFXRateInput) (*ValuationFXRate, error) {
	input.BaseCurrency = strings.ToUpper(strings.TrimSpace(input.BaseCurrency))
	input.QuoteCurrency = strings.ToUpper(strings.TrimSpace(input.QuoteCurrency))
	if len(input.BaseCurrency) != 3 || len(input.QuoteCurrency) != 3 || input.BaseCurrency == input.QuoteCurrency {
		return nil, infraerrors.BadRequest("INVALID_FX_PAIR", "invalid currency pair")
	}
	rate, err := decimal.NewFromString(strings.TrimSpace(input.Rate))
	if err != nil || !rate.IsPositive() {
		return nil, infraerrors.BadRequest("INVALID_FX_RATE", "rate must be positive")
	}
	input.Rate = rate.String()
	return s.repo.CreateFXRate(ctx, input)
}

func ResolveSettlementPeriod(periodType, startDate, endDate string) (SettlementPeriod, error) {
	loc, _ := time.LoadLocation(PoolTimezone)
	start, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(startDate), loc)
	if err != nil {
		return SettlementPeriod{}, infraerrors.BadRequest("INVALID_PERIOD_START", "start_date must be YYYY-MM-DD")
	}
	var end time.Time
	switch periodType {
	case "day":
		end = start.AddDate(0, 0, 1)
	case "week":
		start = start.AddDate(0, 0, -((int(start.Weekday()) + 6) % 7))
		end = start.AddDate(0, 0, 7)
	case "month":
		start = time.Date(start.Year(), start.Month(), 1, 0, 0, 0, 0, loc)
		end = start.AddDate(0, 1, 0)
	case "custom":
		end, err = time.ParseInLocation("2006-01-02", strings.TrimSpace(endDate), loc)
		if err != nil || end.Before(start) {
			return SettlementPeriod{}, infraerrors.BadRequest("INVALID_PERIOD_END", "custom end_date must not precede start_date")
		}
		end = end.AddDate(0, 0, 1)
	default:
		return SettlementPeriod{}, infraerrors.BadRequest("INVALID_PERIOD_TYPE", "period_type must be day, week, month or custom")
	}
	return SettlementPeriod{Type: periodType, Start: start, End: end, Timezone: PoolTimezone}, nil
}

func (s *PoolService) RecalculateSettlement(ctx context.Context, period SettlementPeriod, actorID int64) (*PoolSettlement, error) {
	costs, weights, coverage, carrySnapshot, err := s.repo.SettlementInputs(ctx, period.Start, period.End)
	if err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(costs))
	for _, item := range costs {
		ids = append(ids, item.EntryID)
	}
	allocated, err := s.repo.LockedAllocatedByCostEntry(ctx, ids)
	if err != nil {
		return nil, err
	}

	snapshot := make([]SettlementCostSnapshot, 0, len(costs)+len(carrySnapshot))
	credits := make(map[int64]int64)
	periodCost := int64(0)
	for _, item := range costs {
		amount := proratedCostMinor(item, period.Start, period.End, allocated[item.EntryID])
		if amount == 0 {
			continue
		}
		periodCost += amount
		credits[item.PayerUserID] += amount
		snapshot = append(snapshot, SettlementCostSnapshot{Kind: "period", EntryID: item.EntryID, AccountID: item.AccountID, PayerUserID: item.PayerUserID, AmountMinor: amount})
	}
	carryIn := int64(0)
	for _, item := range carrySnapshot {
		item.Kind = "carry"
		carryIn += item.AmountMinor
		credits[item.PayerUserID] += item.AmountMinor
		snapshot = append(snapshot, item)
	}
	totalCost := periodCost + carryIn
	weightMap := make(map[int64]decimal.Decimal, len(weights))
	users := make(map[int64]PoolUsageWeight, len(weights)+len(credits))
	for _, item := range weights {
		weightMap[item.UserID] = item.Weight
		users[item.UserID] = item
	}
	for userID := range credits {
		if _, ok := users[userID]; !ok {
			users[userID] = PoolUsageWeight{UserID: userID}
		}
	}
	allocations, totalWeight := AllocateLargestRemainder(totalCost, weightMap)
	carryOut := int64(0)
	if totalWeight.IsZero() {
		carryOut = totalCost
	}
	lines := make([]PoolSettlementLine, 0, len(users))
	userIDs := make([]int64, 0, len(users))
	for id := range users {
		userIDs = append(userIDs, id)
	}
	sort.Slice(userIDs, func(i, j int) bool { return userIDs[i] < userIDs[j] })
	for _, userID := range userIDs {
		user := users[userID]
		share := decimal.Zero
		if !totalWeight.IsZero() {
			share = user.Weight.Div(totalWeight)
		}
		allocatedCost := allocations[userID]
		credit := credits[userID]
		if carryOut != 0 {
			allocatedCost, credit = 0, 0
		}
		lines = append(lines, PoolSettlementLine{
			UserID: userID, UserEmail: user.Email, Username: user.Username,
			UsageWeight: user.Weight.String(), UsageShare: share.String(),
			AllocatedCostMinor: allocatedCost, ContributionCreditMinor: credit,
			NetAmountMinor: allocatedCost - credit, PaymentStatus: "unpaid",
		})
	}
	fxRate, err := s.repo.LatestFXRate(ctx, period.End)
	if err != nil {
		return nil, err
	}
	pricingCoverage := decimal.NewFromInt(1)
	if coverage.CandidateCount > 0 {
		pricingCoverage = decimal.NewFromInt(coverage.CandidateCount - coverage.UnpricedCount).
			Div(decimal.NewFromInt(coverage.CandidateCount))
	}
	settlement := &PoolSettlement{
		PeriodType: period.Type, PeriodStart: period.Start, PeriodEnd: period.End, Timezone: period.Timezone,
		Status: "draft", PeriodCostMinor: periodCost, CarryInMinor: carryIn, CarryOutMinor: carryOut,
		TotalCostMinor: totalCost, TotalUsageWeight: totalWeight.String(), PricingCoverage: pricingCoverage.String(),
		UnpricedCount: coverage.UnpricedCount, FXRate: fxRate.String(),
		FormulaVersion: "v1", CostSnapshot: snapshot, GeneratedBy: actorID, Lines: lines,
	}
	return s.repo.SaveDraftSettlement(ctx, settlement)
}

func proratedCostMinor(item PoolCostSlice, start, end time.Time, previouslyAllocated int64) int64 {
	item.ServiceStart = poolCalendarDate(item.ServiceStart)
	item.ServiceEnd = poolCalendarDate(item.ServiceEnd)
	start = poolCalendarDate(start)
	end = poolCalendarDate(end)
	overlapStart := item.ServiceStart
	if start.After(overlapStart) {
		overlapStart = start
	}
	overlapEnd := item.ServiceEnd
	if end.Before(overlapEnd) {
		overlapEnd = end
	}
	if !overlapEnd.After(overlapStart) {
		return 0
	}
	serviceDays := int64(item.ServiceEnd.Sub(item.ServiceStart).Hours() / 24)
	overlapDays := int64(overlapEnd.Sub(overlapStart).Hours() / 24)
	if serviceDays <= 0 || overlapDays <= 0 {
		return 0
	}
	if !overlapEnd.Before(item.ServiceEnd) {
		daysBefore := int64(overlapStart.Sub(item.ServiceStart).Hours() / 24)
		expected := decimal.NewFromInt(item.AmountMinor).Mul(decimal.NewFromInt(daysBefore)).Div(decimal.NewFromInt(serviceDays))
		low, high := expected.Floor().IntPart(), expected.Ceil().IntPart()
		if previouslyAllocated >= low && previouslyAllocated <= high {
			return item.AmountMinor - previouslyAllocated
		}
	}
	amount := decimal.NewFromInt(item.AmountMinor).Mul(decimal.NewFromInt(overlapDays)).Div(decimal.NewFromInt(serviceDays)).Round(0).IntPart()
	remaining := item.AmountMinor - previouslyAllocated
	if item.AmountMinor >= 0 && amount > remaining {
		return remaining
	}
	if item.AmountMinor < 0 && amount < remaining {
		return remaining
	}
	return amount
}

func poolCalendarDate(value time.Time) time.Time {
	loc, _ := time.LoadLocation(PoolTimezone)
	year, month, day := value.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, loc)
}

type allocationRemainder struct {
	userID    int64
	remainder decimal.Decimal
}

func AllocateLargestRemainder(totalMinor int64, weights map[int64]decimal.Decimal) (map[int64]int64, decimal.Decimal) {
	result := make(map[int64]int64, len(weights))
	totalWeight := decimal.Zero
	for id, weight := range weights {
		if weight.IsPositive() {
			totalWeight = totalWeight.Add(weight)
		} else {
			result[id] = 0
		}
	}
	if totalMinor == 0 || totalWeight.IsZero() {
		return result, totalWeight
	}
	sign := int64(1)
	absTotal := totalMinor
	if totalMinor < 0 {
		sign, absTotal = -1, -totalMinor
	}
	remainders := make([]allocationRemainder, 0, len(weights))
	allocated := int64(0)
	for userID, weight := range weights {
		if !weight.IsPositive() {
			continue
		}
		exact := decimal.NewFromInt(absTotal).Mul(weight).Div(totalWeight)
		base := exact.Floor().IntPart()
		result[userID] = base * sign
		allocated += base
		remainders = append(remainders, allocationRemainder{userID: userID, remainder: exact.Sub(decimal.NewFromInt(base))})
	}
	sort.Slice(remainders, func(i, j int) bool {
		cmp := remainders[i].remainder.Cmp(remainders[j].remainder)
		if cmp == 0 {
			return remainders[i].userID < remainders[j].userID
		}
		return cmp > 0
	})
	for i := int64(0); i < absTotal-allocated; i++ {
		result[remainders[i%int64(len(remainders))].userID] += sign
	}
	return result, totalWeight
}

func (s *PoolService) LockSettlement(ctx context.Context, id, actorID int64) (*PoolSettlement, error) {
	item, err := s.repo.GetSettlement(ctx, id)
	if err != nil {
		return nil, err
	}
	coverage, err := decimal.NewFromString(item.PricingCoverage)
	if err != nil || coverage.LessThan(decimal.RequireFromString("0.99")) {
		return nil, infraerrors.Conflict("SETTLEMENT_PRICING_INCOMPLETE", "pricing coverage must reach 99% before locking")
	}
	if item.Status == "locked" {
		return item, nil
	}
	// Lock only a freshly rebuilt draft; repository locking verifies the live cost,
	// usage and FX inputs again under PostgreSQL table locks.
	fresh, err := s.RecalculateSettlement(ctx, SettlementPeriod{
		Type: item.PeriodType, Start: item.PeriodStart, End: item.PeriodEnd, Timezone: item.Timezone,
	}, actorID)
	if err != nil {
		return nil, err
	}
	return s.repo.LockSettlement(ctx, fresh.ID, actorID)
}

func (s *PoolService) ListSettlements(ctx context.Context, page, pageSize int) ([]PoolSettlement, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.repo.ListSettlements(ctx, pageSize, (page-1)*pageSize)
}

func (s *PoolService) GetSettlement(ctx context.Context, id int64) (*PoolSettlement, error) {
	return s.repo.GetSettlement(ctx, id)
}

func (s *PoolService) GetRecovery(ctx context.Context, start, end time.Time) (*PoolRecoveryOverview, error) {
	if !end.After(start) {
		return nil, infraerrors.BadRequest("INVALID_RECOVERY_PERIOD", "end_at must be after start_at")
	}
	if _, err := s.repo.LatestFXRate(ctx, end); err != nil {
		return nil, err
	}
	accounts, err := s.repo.GetRecovery(ctx, start, end)
	if err != nil {
		return nil, err
	}
	overview := &PoolRecoveryOverview{Start: start, End: end, Accounts: accounts, TotalAccounts: len(accounts)}
	type sourceAccumulator struct {
		stat                 PurchaseSourceRecovery
		eligible7, banned7   int64
		eligible30, banned30 int64
		eligible90, banned90 int64
		refunded             int64
		totalSurvivalDays    int64
	}
	sources := make(map[string]*sourceAccumulator)
	for _, account := range accounts {
		overview.TotalCostMinor += account.NetCostMinor
		overview.TotalValueMinor += account.ValueMinor
		overview.UnrecoveredMinor += account.UnrecoveredMinor
		overview.BannedLossMinor += account.BannedLossMinor
		if account.CurrentlyRecovered {
			overview.RecoveredAccounts++
		}
		name := "Unspecified"
		if account.PurchaseSource != nil && strings.TrimSpace(*account.PurchaseSource) != "" {
			name = strings.TrimSpace(*account.PurchaseSource)
		}
		acc := sources[name]
		if acc == nil {
			acc = &sourceAccumulator{stat: PurchaseSourceRecovery{Name: name}}
			sources[name] = acc
		}
		acc.stat.AccountCount++
		acc.stat.SampleSize++
		acc.stat.PurchaseCostMinor += account.NetCostMinor
		acc.stat.ValueMinor += account.ValueMinor
		acc.totalSurvivalDays += account.SurvivalDays
		if account.Refunded {
			acc.refunded++
		}
		if account.PurchasedAt != nil {
			ageDays := int64(end.Sub(*account.PurchasedAt).Hours() / 24)
			if ageDays >= 7 {
				acc.eligible7++
				if account.BannedAt != nil && !account.BannedAt.After(account.PurchasedAt.AddDate(0, 0, 7)) {
					acc.banned7++
				}
			}
			if ageDays >= 30 {
				acc.eligible30++
				if account.BannedAt != nil && !account.BannedAt.After(account.PurchasedAt.AddDate(0, 0, 30)) {
					acc.banned30++
				}
			}
			if ageDays >= 90 {
				acc.eligible90++
				if account.BannedAt != nil && !account.BannedAt.After(account.PurchasedAt.AddDate(0, 0, 90)) {
					acc.banned90++
				}
			}
		}
	}
	if overview.TotalCostMinor > 0 {
		overview.RecoveryRate = decimal.NewFromInt(overview.TotalValueMinor).Div(decimal.NewFromInt(overview.TotalCostMinor)).String()
	} else {
		overview.RecoveryRate = "0"
	}
	for _, acc := range sources {
		if acc.stat.PurchaseCostMinor > 0 {
			acc.stat.RecoveryRate = decimal.NewFromInt(acc.stat.ValueMinor).Div(decimal.NewFromInt(acc.stat.PurchaseCostMinor)).String()
		} else {
			acc.stat.RecoveryRate = "0"
		}
		acc.stat.BanRate7Days = decimalRatio(acc.banned7, acc.eligible7)
		acc.stat.BanRate30Days = decimalRatio(acc.banned30, acc.eligible30)
		acc.stat.BanRate90Days = decimalRatio(acc.banned90, acc.eligible90)
		acc.stat.RefundRate = decimalRatio(acc.refunded, int64(acc.stat.AccountCount))
		acc.stat.AverageSurvivalDays = decimalRatio(acc.totalSurvivalDays, int64(acc.stat.AccountCount))
		acc.stat.RankEligible = acc.stat.SampleSize >= 5
		overview.SourceStats = append(overview.SourceStats, acc.stat)
	}
	sort.Slice(overview.SourceStats, func(i, j int) bool {
		if overview.SourceStats[i].RankEligible != overview.SourceStats[j].RankEligible {
			return overview.SourceStats[i].RankEligible
		}
		return overview.SourceStats[i].Name < overview.SourceStats[j].Name
	})
	return overview, nil
}

func decimalRatio(numerator, denominator int64) string {
	if denominator <= 0 {
		return "0"
	}
	return decimal.NewFromInt(numerator).Div(decimal.NewFromInt(denominator)).String()
}

func marshalCostSnapshot(items []SettlementCostSnapshot) ([]byte, error) { return json.Marshal(items) }
func unmarshalCostSnapshot(raw []byte) ([]SettlementCostSnapshot, error) {
	var items []SettlementCostSnapshot
	if len(raw) == 0 {
		return items, nil
	}
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, fmt.Errorf("decode settlement cost snapshot: %w", err)
	}
	return items, nil
}
