package service

import (
	"context"
	"fmt"
	"sort"
	"strconv"
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

const (
	PoolAvailabilityNormal              = "normal"
	PoolAvailabilityError               = "error"
	PoolAvailabilityRateLimited         = "rate_limited"
	PoolAvailabilityOverloaded          = "overloaded"
	PoolAvailabilityTempUnschedulable   = "temp_unschedulable"
	PoolAvailabilityInactive            = "inactive"
	PoolAvailabilityManualUnschedulable = "manual_unschedulable"
)

type PoolAccountRuntime struct {
	AccountStatus           string     `json:"account_status"`
	AvailabilityStatus      string     `json:"availability_status"`
	Schedulable             bool       `json:"schedulable"`
	ErrorMessage            string     `json:"error_message"`
	RateLimitedAt           *time.Time `json:"rate_limited_at"`
	RateLimitResetAt        *time.Time `json:"rate_limit_reset_at"`
	OverloadUntil           *time.Time `json:"overload_until"`
	TempUnschedulableUntil  *time.Time `json:"temp_unschedulable_until"`
	TempUnschedulableReason string     `json:"temp_unschedulable_reason"`
	ExpiresAt               *time.Time `json:"expires_at"`
	AutoPauseOnExpired      bool       `json:"auto_pause_on_expired"`
}

func ResolvePoolAvailability(state PoolAccountRuntime, now time.Time) string {
	switch {
	case state.AccountStatus == StatusError:
		return PoolAvailabilityError
	case state.AccountStatus != StatusActive:
		return PoolAvailabilityInactive
	case !state.Schedulable:
		return PoolAvailabilityManualUnschedulable
	case state.AutoPauseOnExpired && state.ExpiresAt != nil && !now.Before(*state.ExpiresAt):
		return PoolAvailabilityManualUnschedulable
	case state.OverloadUntil != nil && state.OverloadUntil.After(now):
		return PoolAvailabilityOverloaded
	case state.RateLimitResetAt != nil && state.RateLimitResetAt.After(now):
		return PoolAvailabilityRateLimited
	case state.TempUnschedulableUntil != nil && state.TempUnschedulableUntil.After(now):
		return PoolAvailabilityTempUnschedulable
	default:
		return PoolAvailabilityNormal
	}
}

type PoolAccount struct {
	PoolAccountRuntime
	ID                    int64      `json:"id"`
	Name                  string     `json:"name"`
	Platform              string     `json:"platform"`
	Type                  string     `json:"type"`
	CreatedAt             time.Time  `json:"created_at"`
	ImportBatchID         string     `json:"import_batch_id"`
	ProviderIdentity      *string    `json:"provider_identity"`
	ContributorUserID     *int64     `json:"contributor_user_id"`
	ContributorEmail      *string    `json:"contributor_email"`
	CreatedByUserID       *int64     `json:"created_by_user_id"`
	CreatedByEmail        *string    `json:"created_by_email"`
	CreatedByUsername     *string    `json:"created_by_username"`
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
	PoolAccountRuntime
	ID                 int64      `json:"id"`
	AccountID          int64      `json:"account_id"`
	AccountName        string     `json:"account_name"`
	PayerUserID        int64      `json:"payer_user_id"`
	PayerEmail         string     `json:"payer_email"`
	PurchaseSourceID   *int64     `json:"purchase_source_id"`
	PurchaseSource     *string    `json:"purchase_source"`
	EntryType          string     `json:"entry_type"`
	Currency           string     `json:"currency"`
	OriginalAmount     string     `json:"original_amount"`
	CNYAmountMinor     int64      `json:"cny_amount_minor"`
	FXRate             string     `json:"fx_rate"`
	ServiceStart       time.Time  `json:"service_start"`
	ServiceEnd         time.Time  `json:"service_end"`
	WarrantyEnd        *time.Time `json:"warranty_end"`
	PaidAt             time.Time  `json:"paid_at"`
	OrderNo            *string    `json:"order_no,omitempty"`
	PurchaseURL        *string    `json:"purchase_url,omitempty"`
	Note               *string    `json:"note"`
	SupersedesID       *int64     `json:"supersedes_id"`
	RelatedAccountID   *int64     `json:"related_account_id"`
	ExpectedTokenCount *int64     `json:"expected_token_count"`
	CreatedByUserID    int64      `json:"created_by_user_id"`
	CreatedAt          time.Time  `json:"created_at"`
}

type CreateAccountCostInput struct {
	AccountID          int64
	PayerUserID        int64
	PurchaseSourceID   *int64
	EntryType          string
	Currency           string
	OriginalAmount     string
	CNYAmountMinor     int64
	FXRate             string
	ServiceStart       time.Time
	ServiceEnd         time.Time
	WarrantyEnd        *time.Time
	PaidAt             time.Time
	OrderNo            *string
	PurchaseURL        *string
	Note               *string
	SupersedesID       *int64
	RelatedAccountID   *int64
	CreatedByUserID    int64
	OperationKey       string
	OrderAccountKey    string
	ExpectedTokenCount *int64
}

type AccountCostSummary struct {
	PoolAccountRuntime
	AccountID               int64                `json:"account_id"`
	AccountName             string               `json:"account_name"`
	ProviderIdentity        *string              `json:"provider_identity"`
	UploaderUserID          *int64               `json:"uploader_user_id"`
	UploaderEmail           *string              `json:"uploader_email"`
	UploaderUsername        *string              `json:"uploader_username"`
	ContributorUserID       *int64               `json:"contributor_user_id"`
	ContributorEmail        *string              `json:"contributor_email"`
	ExpectedTokenCount      *int64               `json:"expected_token_count"`
	PricedExpectedTokens    int64                `json:"priced_expected_token_count"`
	RemainingExpectedTokens int64                `json:"remaining_expected_token_count"`
	TotalUsageTokens        int64                `json:"total_usage_tokens"`
	PurchaseCostMinor       int64                `json:"purchase_cost_minor"`
	RefundMinor             int64                `json:"refund_minor"`
	TransferredOutMinor     int64                `json:"transferred_out_minor"`
	WrittenOffMinor         int64                `json:"written_off_minor"`
	CostBasisMinor          int64                `json:"cost_basis_minor"`
	NetCostMinor            int64                `json:"net_cost_minor"`
	RecognizedCostMinor     int64                `json:"recognized_cost_minor"`
	RemainingCostMinor      int64                `json:"remaining_cost_minor"`
	CostProgress            *string              `json:"cost_progress"`
	EntryCount              int64                `json:"entry_count"`
	LatestLifecycleStatus   string               `json:"latest_lifecycle_status"`
	LatestLifecycleAt       *time.Time           `json:"latest_lifecycle_at"`
	LatestPayerUserID       *int64               `json:"latest_payer_user_id"`
	LatestPayerEmail        *string              `json:"latest_payer_email"`
	LatestPurchaseSourceID  *int64               `json:"latest_purchase_source_id"`
	LatestPurchaseSource    *string              `json:"latest_purchase_source"`
	LatestOrderNo           *string              `json:"latest_order_no"`
	LatestServiceStart      *time.Time           `json:"latest_service_start"`
	LatestServiceEnd        *time.Time           `json:"latest_service_end"`
	PurchasedAt             *time.Time           `json:"purchased_at"`
	RecoveryDataQuality     string               `json:"recovery_data_quality"`
	UnpricedPositiveCount   int64                `json:"-"`
	FuturePurchaseCount     int64                `json:"-"`
	CostTranches            []AccountCostTranche `json:"-"`
}

type AccountCostTranche struct {
	ID               int64     `json:"id"`
	CostMinor        int64     `json:"cost_minor"`
	ExpectedTokens   int64     `json:"expected_tokens"`
	PaidAt           time.Time `json:"paid_at"`
	UsageTokens      int64     `json:"usage_tokens"`
	PayerUserID      int64     `json:"payer_user_id"`
	PurchaseSourceID *int64    `json:"purchase_source_id"`
	ServiceStart     time.Time `json:"service_start"`
	ServiceEnd       time.Time `json:"service_end"`
}

type AccountCostTrancheRecognition struct {
	Tranche             AccountCostTranche
	ConsumedTokens      int64
	RemainingTokens     int64
	RecognizedCostMinor int64
	RemainingCostMinor  int64
}

type AccountCostSummaryFilter struct {
	Search             string
	UploaderUserID     *int64
	UploaderUnassigned bool
	PayerUserID        *int64
	PurchaseSourceID   *int64
	AccountStatus      string
	AvailabilityStatus string
	LifecycleStatus    string
	EntryType          string
	HasCost            *bool
}

type AccountCostEntryFilter struct {
	Search           string
	AccountID        *int64
	UploaderUserID   *int64
	PayerUserID      *int64
	PurchaseSourceID *int64
	EntryType        string
	PaidFrom         *time.Time
	PaidTo           *time.Time
}

type BatchAccountCostItemInput struct {
	AccountID          int64
	OriginalAmount     *string
	ExpectedTokenCount *int64
}

type BatchCreateAccountCostsInput struct {
	AmountMode         string
	Common             CreateAccountCostInput
	Accounts           []BatchAccountCostItemInput
	ExpectedTokenCount *int64
	CreatedByUserID    int64
	OperationKey       string
}

type BatchCreateAccountCostsResult struct {
	AmountMode          string             `json:"amount_mode"`
	AccountCount        int                `json:"account_count"`
	TotalOriginalAmount string             `json:"total_original_amount"`
	TotalCNYAmountMinor int64              `json:"total_cny_amount_minor"`
	Entries             []AccountCostEntry `json:"entries"`
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
	Filter   SettlementFilterSnapshot
}

type SettlementFilterSnapshot struct {
	AccountID        *int64 `json:"account_id,omitempty"`
	UploaderUserID   *int64 `json:"uploader_user_id,omitempty"`
	PayerUserID      *int64 `json:"payer_user_id,omitempty"`
	PurchaseSourceID *int64 `json:"purchase_source_id,omitempty"`
}

type PoolUsageWeight struct {
	AccountID int64
	UserID    int64
	Email     string
	Username  string
	Weight    decimal.Decimal
}

type PoolUsageCoverage struct {
	CandidateCount    int64
	UnpricedCount     int64
	CandidateMaterial int64
	UnpricedMaterial  int64
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

type PoolSettlementAccountCost struct {
	ID           int64  `json:"id"`
	SettlementID int64  `json:"settlement_id"`
	AccountID    int64  `json:"account_id"`
	CostEntryID  int64  `json:"cost_entry_id"`
	Kind         string `json:"kind"`
	PayerUserID  int64  `json:"payer_user_id"`
	AmountMinor  int64  `json:"amount_minor"`
}

type PoolSettlementAccountLine struct {
	ID                      int64  `json:"id"`
	SettlementID            int64  `json:"settlement_id"`
	AccountID               int64  `json:"account_id"`
	UserID                  int64  `json:"user_id"`
	UserEmail               string `json:"user_email"`
	Username                string `json:"username"`
	AccountUsageWeight      string `json:"account_usage_weight"`
	UsageShare              string `json:"usage_share"`
	AllocatedCostMinor      int64  `json:"allocated_cost_minor"`
	ContributionCreditMinor int64  `json:"contribution_credit_minor"`
	AdjustmentMinor         int64  `json:"adjustment_minor"`
	NetAmountMinor          int64  `json:"net_amount_minor"`
	TraceQuality            string `json:"trace_quality"`
}

type PoolSettlementLine struct {
	ID                      int64      `json:"id"`
	SettlementID            int64      `json:"settlement_id"`
	UserID                  int64      `json:"user_id"`
	UserEmail               string     `json:"user_email"`
	Username                string     `json:"username"`
	UsageWeight             string     `json:"usage_weight"`
	UsageShare              string     `json:"usage_share"`
	AllocatedCostMinor      int64      `json:"allocated_cost_minor"`
	ContributionCreditMinor int64      `json:"contribution_credit_minor"`
	AdjustmentMinor         int64      `json:"adjustment_minor"`
	NetAmountMinor          int64      `json:"net_amount_minor"`
	PaymentStatus           string     `json:"payment_status"`
	ConfirmationStatus      string     `json:"confirmation_status"`
	ConfirmedByUserID       *int64     `json:"confirmed_by_user_id"`
	ConfirmedAt             *time.Time `json:"confirmed_at"`
}

type PoolSettlement struct {
	ID               int64                       `json:"id"`
	PeriodType       string                      `json:"period_type"`
	PeriodStart      time.Time                   `json:"period_start"`
	PeriodEnd        time.Time                   `json:"period_end"`
	Timezone         string                      `json:"timezone"`
	Status           string                      `json:"status"`
	PeriodCostMinor  int64                       `json:"period_cost_minor"`
	CarryInMinor     int64                       `json:"carry_in_minor"`
	CarryOutMinor    int64                       `json:"carry_out_minor"`
	TotalCostMinor   int64                       `json:"total_cost_minor"`
	TotalUsageWeight string                      `json:"total_usage_weight"`
	PricingCoverage  string                      `json:"pricing_coverage"`
	UnpricedCount    int64                       `json:"unpriced_usage_count"`
	FXRate           string                      `json:"fx_rate"`
	FormulaVersion   string                      `json:"formula_version"`
	CostSnapshot     []SettlementCostSnapshot    `json:"cost_snapshot"`
	FilterSnapshot   SettlementFilterSnapshot    `json:"filter_snapshot"`
	GeneratedBy      int64                       `json:"generated_by_user_id"`
	LockedBy         *int64                      `json:"locked_by_user_id"`
	LockedAt         *time.Time                  `json:"locked_at"`
	PaidBy           *int64                      `json:"paid_by_user_id"`
	PaidAt           *time.Time                  `json:"paid_at"`
	CreatedAt        time.Time                   `json:"created_at"`
	UpdatedAt        time.Time                   `json:"updated_at"`
	Lines            []PoolSettlementLine        `json:"lines"`
	AccountCosts     []PoolSettlementAccountCost `json:"account_costs"`
	AccountLines     []PoolSettlementAccountLine `json:"account_lines"`
	AccountContexts  []PoolAccount               `json:"account_contexts"`
}

type AccountRecovery struct {
	PoolAccountRuntime
	AccountID              int64      `json:"account_id"`
	AccountName            string     `json:"account_name"`
	ProviderIdentity       *string    `json:"provider_identity"`
	UploaderUserID         *int64     `json:"uploader_user_id"`
	UploaderUsername       *string    `json:"uploader_username"`
	UploadedAt             time.Time  `json:"uploaded_at"`
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
	AverageDailyTokens     int64      `json:"average_daily_tokens"`
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
	ExpectedTokens         int64      `json:"expected_tokens"`
	UsedTokens             int64      `json:"used_tokens"`
	RemainingTokens        int64      `json:"remaining_tokens"`
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

type AccountCostUploaderSummary struct {
	UploaderUserID      *int64  `json:"uploader_user_id"`
	UploaderEmail       *string `json:"uploader_email"`
	UploaderUsername    *string `json:"uploader_username"`
	AccountCount        int64   `json:"account_count"`
	NetCostMinor        int64   `json:"net_cost_minor"`
	RecognizedCostMinor int64   `json:"recognized_cost_minor"`
	RemainingCostMinor  int64   `json:"remaining_cost_minor"`
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
	ListSources(ctx context.Context, referencedOnly bool) ([]PurchaseSource, error)
	CreateSource(ctx context.Context, input CreatePurchaseSourceInput) (*PurchaseSource, error)
	ListCosts(ctx context.Context, accountID *int64) ([]AccountCostEntry, error)
	CreateCost(ctx context.Context, input CreateAccountCostInput) (*AccountCostEntry, error)
	ListCostEntries(ctx context.Context, filter AccountCostEntryFilter, limit, offset int) ([]AccountCostEntry, int64, error)
	ListCostSummaries(ctx context.Context, filter AccountCostSummaryFilter, limit, offset int) ([]AccountCostSummary, int64, error)
	ListCostUploaderSummaries(ctx context.Context, filter AccountCostSummaryFilter, limit, offset int) ([]AccountCostUploaderSummary, int64, error)
	CreateCostsBatch(ctx context.Context, inputs []CreateAccountCostInput) ([]AccountCostEntry, error)
	CreateAccountIntake(ctx context.Context, input CreateAccountIntakeInput) (*AccountIntakeResult, error)
	ListLifecycle(ctx context.Context, accountID *int64) ([]AccountLifecycleEvent, error)
	CreateLifecycle(ctx context.Context, input CreateLifecycleEventInput) (*AccountLifecycleEvent, error)
	ListFXRates(ctx context.Context) ([]ValuationFXRate, error)
	CreateFXRate(ctx context.Context, input CreateFXRateInput) (*ValuationFXRate, error)
	LatestFXRate(ctx context.Context, at time.Time) (decimal.Decimal, error)
	SettlementInputs(ctx context.Context, start, end time.Time, filter SettlementFilterSnapshot) ([]PoolCostSlice, []PoolUsageWeight, PoolUsageCoverage, []SettlementCostSnapshot, error)
	LockedAllocatedByCostEntry(ctx context.Context, ids []int64) (map[int64]int64, error)
	SaveDraftSettlement(ctx context.Context, settlement *PoolSettlement) (*PoolSettlement, error)
	LockSettlement(ctx context.Context, id, actorID int64) (*PoolSettlement, error)
	ConfirmSettlementLine(ctx context.Context, id, userID, actorID int64) error
	MarkSettlementPaid(ctx context.Context, id, actorID int64) error
	ListSettlements(ctx context.Context, accountID *int64, limit, offset int) ([]PoolSettlement, int64, error)
	GetSettlement(ctx context.Context, id int64) (*PoolSettlement, error)
	GetRecovery(ctx context.Context, start, end time.Time, accountID ...*int64) ([]AccountRecovery, error)
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

func (s *PoolService) ListSources(ctx context.Context, referencedOnly bool) ([]PurchaseSource, error) {
	return s.repo.ListSources(ctx, referencedOnly)
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
	input.OrderAccountKey = poolCostOrderAccountKey(input)
	return s.repo.CreateCost(ctx, input)
}

func (s *PoolService) ListCostEntries(ctx context.Context, filter AccountCostEntryFilter, page, pageSize int) ([]AccountCostEntry, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 1000 {
		pageSize = 20
	}
	filter.Search = strings.TrimSpace(filter.Search)
	filter.EntryType = strings.TrimSpace(filter.EntryType)
	return s.repo.ListCostEntries(ctx, filter, pageSize, (page-1)*pageSize)
}

func (s *PoolService) ListCostSummaries(ctx context.Context, filter AccountCostSummaryFilter, page, pageSize int) ([]AccountCostSummary, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 1000 {
		pageSize = 20
	}
	filter.Search = strings.TrimSpace(filter.Search)
	filter.AccountStatus = strings.TrimSpace(filter.AccountStatus)
	filter.AvailabilityStatus = strings.TrimSpace(filter.AvailabilityStatus)
	filter.LifecycleStatus = strings.TrimSpace(filter.LifecycleStatus)
	filter.EntryType = strings.TrimSpace(filter.EntryType)
	items, total, err := s.repo.ListCostSummaries(ctx, filter, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, err
	}
	for i := range items {
		applyCostRecognition(&items[i])
		items[i].RecoveryDataQuality = ResolvePoolRecoveryDataQuality(
			items[i].PurchaseCostMinor,
			items[i].CostProgress,
			items[i].UnpricedPositiveCount,
			items[i].FuturePurchaseCount > 0,
		)
	}
	return items, total, nil
}

func (s *PoolService) ListCostUploaderSummaries(ctx context.Context, filter AccountCostSummaryFilter, page, pageSize int) ([]AccountCostUploaderSummary, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	filter.Search = strings.TrimSpace(filter.Search)
	filter.AccountStatus = strings.TrimSpace(filter.AccountStatus)
	filter.AvailabilityStatus = strings.TrimSpace(filter.AvailabilityStatus)
	filter.LifecycleStatus = strings.TrimSpace(filter.LifecycleStatus)
	filter.EntryType = strings.TrimSpace(filter.EntryType)
	return s.repo.ListCostUploaderSummaries(ctx, filter, pageSize, (page-1)*pageSize)
}

func (s *PoolService) CreateCostsBatch(ctx context.Context, input BatchCreateAccountCostsInput) (*BatchCreateAccountCostsResult, error) {
	input.AmountMode = strings.TrimSpace(input.AmountMode)
	items, totalOriginal, totalCNY, err := prepareBatchCostInputs(input)
	if err != nil {
		return nil, err
	}
	entries, err := s.repo.CreateCostsBatch(ctx, items)
	if err != nil {
		return nil, err
	}
	return &BatchCreateAccountCostsResult{
		AmountMode: input.AmountMode, AccountCount: len(entries), TotalOriginalAmount: totalOriginal,
		TotalCNYAmountMinor: totalCNY, Entries: entries,
	}, nil
}

func prepareBatchCostInputs(input BatchCreateAccountCostsInput) ([]CreateAccountCostInput, string, int64, error) {
	input.AmountMode = strings.TrimSpace(input.AmountMode)
	if input.AmountMode != "per_account" && input.AmountMode != "order_total" {
		return nil, "", 0, infraerrors.BadRequest("INVALID_AMOUNT_MODE", "amount_mode must be per_account or order_total")
	}
	if len(input.Accounts) == 0 || len(input.Accounts) > 500 {
		return nil, "", 0, infraerrors.BadRequest("INVALID_BATCH_ACCOUNTS", "accounts must contain between 1 and 500 items")
	}
	operationKey, err := NormalizeIdempotencyKey(input.OperationKey)
	if err != nil {
		return nil, "", 0, err
	}
	if operationKey == "" {
		return nil, "", 0, ErrIdempotencyKeyRequired
	}
	if input.CreatedByUserID <= 0 {
		return nil, "", 0, infraerrors.BadRequest("INVALID_COST_PARTY", "cost creator is required")
	}
	if input.ExpectedTokenCount != nil && *input.ExpectedTokenCount <= 0 {
		return nil, "", 0, infraerrors.BadRequest("INVALID_EXPECTED_TOKEN_COUNT", "expected_token_count must be positive")
	}

	seen := make(map[int64]struct{}, len(input.Accounts))
	items := make([]CreateAccountCostInput, len(input.Accounts))
	for i, account := range input.Accounts {
		if account.AccountID <= 0 {
			return nil, "", 0, infraerrors.BadRequest("INVALID_ACCOUNT_ID", "invalid account id")
		}
		if _, exists := seen[account.AccountID]; exists {
			return nil, "", 0, infraerrors.BadRequest("DUPLICATE_BATCH_ACCOUNT", "an account may appear only once in a cost batch")
		}
		seen[account.AccountID] = struct{}{}
		if account.ExpectedTokenCount != nil && *account.ExpectedTokenCount <= 0 {
			return nil, "", 0, infraerrors.BadRequest("INVALID_EXPECTED_TOKEN_COUNT", "expected_token_count must be positive")
		}
		items[i] = input.Common
		items[i].AccountID = account.AccountID
		items[i].CreatedByUserID = input.CreatedByUserID
		items[i].OperationKey = operationKey
		if i > 0 {
			items[i].OperationKey += ":" + strconv.FormatInt(account.AccountID, 10)
		}
		items[i].ExpectedTokenCount = input.ExpectedTokenCount
		if account.ExpectedTokenCount != nil {
			items[i].ExpectedTokenCount = account.ExpectedTokenCount
		}
		if account.OriginalAmount != nil {
			items[i].OriginalAmount = strings.TrimSpace(*account.OriginalAmount)
		}
	}

	if input.AmountMode == "per_account" {
		totalOriginal := decimal.Zero
		var totalCNY int64
		for i := range items {
			items[i], err = normalizePoolCostInput(items[i])
			if err != nil {
				return nil, "", 0, err
			}
			amount, _ := decimal.NewFromString(items[i].OriginalAmount)
			totalOriginal = totalOriginal.Add(amount)
			totalCNY += items[i].CNYAmountMinor
			items[i].OrderAccountKey = poolCostOrderAccountKey(items[i])
		}
		return items, totalOriginal.String(), totalCNY, nil
	}

	total := input.Common
	total.AccountID = input.Accounts[0].AccountID
	total.CreatedByUserID = input.CreatedByUserID
	total.ExpectedTokenCount = items[0].ExpectedTokenCount
	total, err = normalizePoolCostInput(total)
	if err != nil {
		return nil, "", 0, err
	}
	fixedCNY := int64(0)
	unassigned := make(map[int64]decimal.Decimal)
	for i, account := range input.Accounts {
		if account.OriginalAmount == nil {
			unassigned[account.AccountID] = decimal.NewFromInt(1)
			continue
		}
		items[i], err = normalizePoolCostInput(items[i])
		if err != nil {
			return nil, "", 0, err
		}
		fixedCNY += items[i].CNYAmountMinor
	}
	remaining := total.CNYAmountMinor - fixedCNY
	if len(unassigned) == 0 && remaining != 0 {
		return nil, "", 0, infraerrors.BadRequest("ORDER_TOTAL_MISMATCH", "per-account amounts must equal order total")
	}
	if (total.CNYAmountMinor > 0 && remaining < 0) || (total.CNYAmountMinor < 0 && remaining > 0) {
		return nil, "", 0, infraerrors.BadRequest("ORDER_TOTAL_EXCEEDED", "per-account overrides exceed order total")
	}
	allocated, _ := AllocateLargestRemainder(remaining, unassigned)
	rate, _ := decimal.NewFromString(total.FXRate)
	for i := range items {
		if _, ok := unassigned[items[i].AccountID]; ok {
			amount := decimal.NewFromInt(allocated[items[i].AccountID]).Div(decimal.NewFromInt(100)).Div(rate)
			items[i].OriginalAmount = amount.String()
			items[i], err = normalizePoolCostInput(items[i])
			if err != nil {
				return nil, "", 0, err
			}
			if items[i].CNYAmountMinor != allocated[items[i].AccountID] {
				return nil, "", 0, infraerrors.BadRequest("ORDER_TOTAL_ALLOCATION_FAILED", "order total cannot be allocated exactly with the selected fx rate")
			}
		}
		items[i].OrderAccountKey = poolCostOrderAccountKey(items[i])
	}
	return items, total.OriginalAmount, total.CNYAmountMinor, nil
}

func poolCostOrderAccountKey(input CreateAccountCostInput) string {
	orderNo := ""
	if input.OrderNo != nil {
		orderNo = strings.ToLower(strings.TrimSpace(*input.OrderNo))
	}
	if orderNo == "" || (input.EntryType != "purchase" && input.EntryType != "renewal" && input.EntryType != "topup" && input.EntryType != "price_version") {
		return ""
	}
	sourceID := int64(0)
	if input.PurchaseSourceID != nil {
		sourceID = *input.PurchaseSourceID
	}
	return fmt.Sprintf("%d:%d:%s", input.AccountID, sourceID, orderNo)
}

func applyCostRecognition(item *AccountCostSummary) {
	if item == nil {
		return
	}
	_, _, item.PricedExpectedTokens, item.RemainingExpectedTokens = RecognizeAccountCostTranches(item.TotalUsageTokens, item.CostTranches)
	item.RecognizedCostMinor, item.RemainingCostMinor, item.CostProgress = CalculateAccountCostRecognitionByTranches(
		item.CostBasisMinor, item.RefundMinor+item.TransferredOutMinor+item.WrittenOffMinor, item.TotalUsageTokens, item.CostTranches,
	)
}

func ResolvePoolRecoveryDataQuality(purchaseCostMinor int64, progress *string, unpricedPositiveCount int64, futurePurchase bool) string {
	switch {
	case futurePurchase:
		return "future_purchase_time"
	case purchaseCostMinor <= 0:
		return "no_cost"
	case progress == nil:
		return "missing_expected_tokens"
	case unpricedPositiveCount > 0:
		return "partial_expected_tokens"
	default:
		return "ready"
	}
}

// CalculateAccountCostRecognitionByTranches recognizes historical usage against
// each priced token tranche in purchase order, so later top-ups do not absorb old usage.
func CalculateAccountCostRecognitionByTranches(costBasisMinor, disposedMinor, usedTokens int64, tranches []AccountCostTranche) (int64, int64, *string) {
	available := costBasisMinor - disposedMinor
	if available < 0 {
		available = 0
	}
	states, consumed, totalExpected, _ := RecognizeAccountCostTranches(usedTokens, tranches)
	var recognized int64
	for _, state := range states {
		recognized += state.RecognizedCostMinor
	}
	if totalExpected == 0 {
		return 0, available, nil
	}
	if recognized > available {
		recognized = available
	}
	progress := decimal.NewFromInt(consumed).Div(decimal.NewFromInt(totalExpected)).String()
	return recognized, available - recognized, &progress
}

// RecognizeAccountCostTranches consumes only usage that happened after a priced
// tranche existed. Excess usage before a renewal is discarded instead of charging
// a future purchase.
func RecognizeAccountCostTranches(usedTokens int64, tranches []AccountCostTranche) ([]AccountCostTrancheRecognition, int64, int64, int64) {
	priced := make([]AccountCostTranche, 0, len(tranches))
	timeline := false
	for _, tranche := range tranches {
		if tranche.CostMinor <= 0 || tranche.ExpectedTokens <= 0 {
			continue
		}
		priced = append(priced, tranche)
		timeline = timeline || !tranche.PaidAt.IsZero()
	}
	sort.SliceStable(priced, func(i, j int) bool {
		if priced[i].PaidAt.Equal(priced[j].PaidAt) {
			return priced[i].ID < priced[j].ID
		}
		return priced[i].PaidAt.Before(priced[j].PaidAt)
	})
	states := make([]AccountCostTrancheRecognition, 0, len(priced))
	consume := func(tokens int64) {
		if tokens <= 0 {
			return
		}
		for i := range states {
			if tokens == 0 {
				break
			}
			take := states[i].RemainingTokens
			if take > tokens {
				take = tokens
			}
			states[i].ConsumedTokens += take
			states[i].RemainingTokens -= take
			tokens -= take
		}
	}
	for _, tranche := range priced {
		states = append(states, AccountCostTrancheRecognition{Tranche: tranche, RemainingTokens: tranche.ExpectedTokens})
		if timeline {
			consume(tranche.UsageTokens)
		}
	}
	if !timeline {
		consume(usedTokens)
	}
	var consumed, totalExpected, remainingTokens int64
	for i := range states {
		state := &states[i]
		state.RecognizedCostMinor = decimal.NewFromInt(state.Tranche.CostMinor).Mul(decimal.NewFromInt(state.ConsumedTokens)).Div(decimal.NewFromInt(state.Tranche.ExpectedTokens)).Round(0).IntPart()
		state.RemainingCostMinor = state.Tranche.CostMinor - state.RecognizedCostMinor
		consumed += state.ConsumedTokens
		totalExpected += state.Tranche.ExpectedTokens
		remainingTokens += state.RemainingTokens
	}
	return states, consumed, totalExpected, remainingTokens
}

// CalculateAccountCostRecognition applies the same per-account cost model to
// ledger summaries and lifecycle disposal.
func CalculateAccountCostRecognition(costBasisMinor, disposedMinor, usedTokens int64, expectedTokens *int64) (int64, int64, *string) {
	basis := costBasisMinor
	if basis < 0 {
		basis = 0
	}
	if disposedMinor < 0 {
		disposedMinor = 0
	}
	available := basis - disposedMinor
	if available < 0 {
		available = 0
	}
	if expectedTokens == nil || *expectedTokens <= 0 || basis == 0 {
		return 0, available, nil
	}
	used := usedTokens
	if used < 0 {
		used = 0
	}
	if used > *expectedTokens {
		used = *expectedTokens
	}
	progress := decimal.NewFromInt(used).Div(decimal.NewFromInt(*expectedTokens))
	progressString := progress.String()
	recognized := decimal.NewFromInt(basis).Mul(progress).Round(0).IntPart()
	if recognized > available {
		recognized = available
	}
	return recognized, available - recognized, &progressString
}

func normalizePoolCostInput(input CreateAccountCostInput) (CreateAccountCostInput, error) {
	now := time.Now().UTC()
	if input.PaidAt.IsZero() {
		input.PaidAt = now
	} else if input.PaidAt.After(now.Add(5 * time.Minute)) {
		return input, infraerrors.BadRequest("FUTURE_PURCHASE_TIME", "paid_at must not be more than 5 minutes in the future")
	}
	if input.AccountID <= 0 || input.PayerUserID <= 0 || input.CreatedByUserID <= 0 {
		return input, infraerrors.BadRequest("INVALID_COST_PARTY", "account, payer and creator are required")
	}
	if input.SupersedesID != nil {
		return input, infraerrors.BadRequest("COST_UPDATE_APPROVAL_REQUIRED", "cost corrections must use the approval update flow")
	}
	if input.ExpectedTokenCount != nil && *input.ExpectedTokenCount <= 0 {
		return input, infraerrors.BadRequest("INVALID_EXPECTED_TOKEN_COUNT", "expected_token_count must be positive")
	}
	validType := map[string]bool{"purchase": true, "renewal": true, "topup": true, "price_version": true, "refund": true, "adjustment": true}
	if !validType[input.EntryType] {
		return input, infraerrors.BadRequest("INVALID_COST_TYPE", "invalid cost entry type")
	}
	if input.EntryType != "refund" && input.EntryType != "adjustment" && input.ExpectedTokenCount == nil {
		return input, infraerrors.BadRequest("EXPECTED_TOKEN_COUNT_REQUIRED", "expected_token_count is required for cost entries")
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
	if input.CNYAmountMinor > 0 && input.ExpectedTokenCount == nil {
		return input, infraerrors.BadRequest("EXPECTED_TOKEN_COUNT_REQUIRED", "expected_token_count is required for positive cost entries")
	}
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
	valid := map[string]bool{"banned_confirmed": true, "recovered": true, "retired": true}
	if input.AccountID <= 0 || input.CreatedByUserID <= 0 || !valid[input.EventType] {
		return nil, infraerrors.BadRequest("INVALID_LIFECYCLE_EVENT", "invalid lifecycle event")
	}
	input.ReplacementAccountID = nil
	input.TransferredCostMinor = 0
	input.RefundAmountMinor = 0
	input.PayerUserID = nil
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
	if err := validateSettlementFilter(period.Filter); err != nil {
		return nil, err
	}
	costs, weights, coverage, carrySnapshot, err := s.repo.SettlementInputs(ctx, period.Start, period.End, period.Filter)
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
	periodCost := int64(0)
	for _, item := range costs {
		amount := proratedCostMinor(item, period.Start, period.End, allocated[item.EntryID])
		if amount == 0 {
			continue
		}
		periodCost += amount
		snapshot = append(snapshot, SettlementCostSnapshot{Kind: "period", EntryID: item.EntryID, AccountID: item.AccountID, PayerUserID: item.PayerUserID, AmountMinor: amount})
	}
	carryIn := int64(0)
	for _, item := range carrySnapshot {
		item.Kind = "carry"
		carryIn += item.AmountMinor
		snapshot = append(snapshot, item)
	}
	totalCost := periodCost + carryIn
	lines, accountLines, totalWeight, carryOut := BuildSettlementAllocation(snapshot, weights)
	accountCosts := make([]PoolSettlementAccountCost, 0, len(snapshot))
	for _, item := range snapshot {
		accountCosts = append(accountCosts, PoolSettlementAccountCost{
			AccountID: item.AccountID, CostEntryID: item.EntryID, Kind: item.Kind,
			PayerUserID: item.PayerUserID, AmountMinor: item.AmountMinor,
		})
	}
	fxRate, err := s.repo.LatestFXRate(ctx, period.End)
	if err != nil {
		return nil, err
	}
	pricingCoverage := decimal.NewFromInt(1)
	if coverage.CandidateMaterial > 0 {
		pricingCoverage = decimal.NewFromInt(coverage.CandidateMaterial - coverage.UnpricedMaterial).
			Div(decimal.NewFromInt(coverage.CandidateMaterial))
	}
	settlement := &PoolSettlement{
		PeriodType: period.Type, PeriodStart: period.Start, PeriodEnd: period.End, Timezone: period.Timezone,
		Status: "draft", PeriodCostMinor: periodCost, CarryInMinor: carryIn, CarryOutMinor: carryOut,
		TotalCostMinor: totalCost, TotalUsageWeight: totalWeight.String(), PricingCoverage: pricingCoverage.String(),
		UnpricedCount: coverage.UnpricedCount, FXRate: fxRate.String(),
		FormulaVersion: "v1", CostSnapshot: snapshot, FilterSnapshot: period.Filter, GeneratedBy: actorID,
		Lines: lines, AccountCosts: accountCosts, AccountLines: accountLines,
	}
	return s.repo.SaveDraftSettlement(ctx, settlement)
}

func validateSettlementFilter(filter SettlementFilterSnapshot) error {
	for _, id := range []*int64{filter.AccountID, filter.UploaderUserID, filter.PayerUserID, filter.PurchaseSourceID} {
		if id != nil && *id <= 0 {
			return infraerrors.BadRequest("INVALID_SETTLEMENT_FILTER", "settlement filter ids must be positive")
		}
	}
	return nil
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

// BuildSettlementAllocation keeps the v1 user totals and adds an exact account trace beneath them.
func BuildSettlementAllocation(costs []SettlementCostSnapshot, weights []PoolUsageWeight) ([]PoolSettlementLine, []PoolSettlementAccountLine, decimal.Decimal, int64) {
	userWeights := make(map[int64]decimal.Decimal)
	accountWeights := make(map[int64]map[int64]decimal.Decimal)
	users := make(map[int64]PoolUsageWeight)
	credits := make(map[int64]int64)
	accountCredits := make(map[int64]map[int64]int64)
	totalCost := int64(0)
	for _, item := range weights {
		userWeights[item.UserID] = userWeights[item.UserID].Add(item.Weight)
		if accountWeights[item.UserID] == nil {
			accountWeights[item.UserID] = make(map[int64]decimal.Decimal)
		}
		accountWeights[item.UserID][item.AccountID] = accountWeights[item.UserID][item.AccountID].Add(item.Weight)
		users[item.UserID] = item
	}
	for _, item := range costs {
		totalCost += item.AmountMinor
		credits[item.PayerUserID] += item.AmountMinor
		if accountCredits[item.PayerUserID] == nil {
			accountCredits[item.PayerUserID] = make(map[int64]int64)
		}
		accountCredits[item.PayerUserID][item.AccountID] += item.AmountMinor
		if _, ok := users[item.PayerUserID]; !ok {
			users[item.PayerUserID] = PoolUsageWeight{UserID: item.PayerUserID}
		}
	}
	allocations, totalWeight := AllocateLargestRemainder(totalCost, userWeights)
	carryOut := int64(0)
	if totalWeight.IsZero() {
		carryOut = totalCost
	}
	userIDs := make([]int64, 0, len(users))
	for userID := range users {
		userIDs = append(userIDs, userID)
	}
	sort.Slice(userIDs, func(i, j int) bool { return userIDs[i] < userIDs[j] })
	lines := make([]PoolSettlementLine, 0, len(userIDs))
	accountLines := make([]PoolSettlementAccountLine, 0, len(weights)+len(costs))
	for _, userID := range userIDs {
		user := users[userID]
		weight := userWeights[userID]
		share := decimal.Zero
		if !totalWeight.IsZero() {
			share = weight.Div(totalWeight)
		}
		allocatedCost, credit := allocations[userID], credits[userID]
		if carryOut != 0 {
			allocatedCost, credit = 0, 0
		}
		lines = append(lines, PoolSettlementLine{
			UserID: userID, UserEmail: user.Email, Username: user.Username,
			UsageWeight: weight.String(), UsageShare: share.String(),
			AllocatedCostMinor: allocatedCost, ContributionCreditMinor: credit,
			NetAmountMinor: allocatedCost - credit, PaymentStatus: "unpaid",
		})
		accountAllocations, _ := AllocateLargestRemainder(allocatedCost, accountWeights[userID])
		accountShareUnits, _ := AllocateLargestRemainder(
			share.Shift(int32(decimal.DivisionPrecision)).IntPart(), accountWeights[userID],
		)
		accountIDs := make(map[int64]struct{}, len(accountWeights[userID])+len(accountCredits[userID]))
		for accountID := range accountWeights[userID] {
			accountIDs[accountID] = struct{}{}
		}
		for accountID := range accountCredits[userID] {
			accountIDs[accountID] = struct{}{}
		}
		orderedAccountIDs := make([]int64, 0, len(accountIDs))
		for accountID := range accountIDs {
			orderedAccountIDs = append(orderedAccountIDs, accountID)
		}
		sort.Slice(orderedAccountIDs, func(i, j int) bool { return orderedAccountIDs[i] < orderedAccountIDs[j] })
		for _, accountID := range orderedAccountIDs {
			accountWeight := accountWeights[userID][accountID]
			accountShare := decimal.NewFromInt(accountShareUnits[accountID]).Shift(-int32(decimal.DivisionPrecision))
			accountAllocated, accountCredit := accountAllocations[accountID], accountCredits[userID][accountID]
			if carryOut != 0 {
				accountAllocated, accountCredit = 0, 0
			}
			accountLines = append(accountLines, PoolSettlementAccountLine{
				AccountID: accountID, UserID: userID, UserEmail: user.Email, Username: user.Username,
				AccountUsageWeight: accountWeight.String(), UsageShare: accountShare.String(),
				AllocatedCostMinor: accountAllocated, ContributionCreditMinor: accountCredit,
				NetAmountMinor: accountAllocated - accountCredit, TraceQuality: "exact",
			})
		}
	}
	sort.Slice(accountLines, func(i, j int) bool {
		if accountLines[i].AccountID == accountLines[j].AccountID {
			return accountLines[i].UserID < accountLines[j].UserID
		}
		return accountLines[i].AccountID < accountLines[j].AccountID
	})
	return lines, accountLines, totalWeight, carryOut
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
	if item.Status != "draft" {
		return item, nil
	}
	// Lock only a freshly rebuilt draft; repository locking verifies the live cost,
	// usage and FX inputs again under PostgreSQL table locks.
	fresh, err := s.RecalculateSettlement(ctx, SettlementPeriod{
		Type: item.PeriodType, Start: item.PeriodStart, End: item.PeriodEnd, Timezone: item.Timezone, Filter: item.FilterSnapshot,
	}, actorID)
	if err != nil {
		return nil, err
	}
	return s.repo.LockSettlement(ctx, fresh.ID, actorID)
}

func (s *PoolService) ListSettlements(ctx context.Context, accountID *int64, page, pageSize int) ([]PoolSettlement, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.repo.ListSettlements(ctx, accountID, pageSize, (page-1)*pageSize)
}

func (s *PoolService) GetSettlement(ctx context.Context, id int64) (*PoolSettlement, error) {
	return s.repo.GetSettlement(ctx, id)
}

func (s *PoolService) ListOwnPendingSettlements(ctx context.Context, userID int64) ([]PoolSettlement, error) {
	if userID <= 0 {
		return nil, infraerrors.BadRequest("SETTLEMENT_CONFIRMATION_ACTOR_REQUIRED", "a signed-in member is required")
	}
	items, _, err := s.repo.ListSettlements(ctx, nil, 1000, 0)
	if err != nil {
		return nil, err
	}
	pending := make([]PoolSettlement, 0)
	for _, item := range items {
		if item.Status != "locked" {
			continue
		}
		for _, line := range item.Lines {
			if line.UserID == userID && line.NetAmountMinor != 0 && line.ConfirmationStatus != "confirmed" {
				item.Lines = []PoolSettlementLine{line}
				item.CostSnapshot = nil
				item.AccountCosts = nil
				item.AccountLines = nil
				item.AccountContexts = nil
				item.FilterSnapshot = SettlementFilterSnapshot{}
				pending = append(pending, item)
				break
			}
		}
	}
	return pending, nil
}

func (s *PoolService) ConfirmSettlementLine(ctx context.Context, id, userID, actorID int64) (*PoolSettlement, error) {
	if id <= 0 || userID <= 0 || actorID <= 0 {
		return nil, infraerrors.BadRequest("SETTLEMENT_CONFIRMATION_ACTOR_REQUIRED", "a settlement member and signed-in actor are required")
	}
	if userID != actorID && !s.IsPrimaryAdmin(ctx, actorID) {
		return nil, infraerrors.Forbidden("SETTLEMENT_CONFIRMATION_FORBIDDEN", "only the primary administrator can resolve another member's line")
	}
	if err := s.repo.ConfirmSettlementLine(ctx, id, userID, actorID); err != nil {
		return nil, err
	}
	return s.repo.GetSettlement(ctx, id)
}

func (s *PoolService) ConfirmOwnSettlementLine(ctx context.Context, id, actorID int64) error {
	if id <= 0 {
		return ErrPoolSettlementNotFound
	}
	if actorID <= 0 {
		return infraerrors.BadRequest("SETTLEMENT_CONFIRMATION_ACTOR_REQUIRED", "a signed-in member is required")
	}
	return s.repo.ConfirmSettlementLine(ctx, id, actorID, actorID)
}

func (s *PoolService) MarkSettlementPaid(ctx context.Context, id, actorID int64) (*PoolSettlement, error) {
	if id <= 0 {
		return nil, ErrPoolSettlementNotFound
	}
	if actorID <= 0 {
		return nil, infraerrors.BadRequest("SETTLEMENT_PAID_ACTOR_REQUIRED", "a signed-in administrator is required")
	}
	if err := s.repo.MarkSettlementPaid(ctx, id, actorID); err != nil {
		return nil, err
	}
	return s.repo.GetSettlement(ctx, id)
}

func (s *PoolService) GetRecovery(ctx context.Context, start, end time.Time, accountID ...*int64) (*PoolRecoveryOverview, error) {
	if !end.After(start) {
		return nil, infraerrors.BadRequest("INVALID_RECOVERY_PERIOD", "end_at must be after start_at")
	}
	if len(accountID) > 0 && accountID[0] != nil && *accountID[0] <= 0 {
		return nil, infraerrors.BadRequest("INVALID_ACCOUNT_ID", "invalid account id")
	}
	accounts, err := s.repo.GetRecovery(ctx, start, end, accountID...)
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
