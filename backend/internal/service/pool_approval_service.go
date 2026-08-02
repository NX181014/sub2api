package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"reflect"
	"sort"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	PoolApprovalUpdateAccount          = "UPDATE_ACCOUNT"
	PoolApprovalViewCredential         = "VIEW_CREDENTIAL"
	PoolApprovalDeleteAccount          = "DELETE_ACCOUNT"
	PoolApprovalUpdateProxy            = "UPDATE_PROXY"
	PoolApprovalViewProxyCredential    = "VIEW_PROXY_CREDENTIAL"
	PoolApprovalExportProxyCredentials = "EXPORT_PROXY_CREDENTIALS"
	PoolApprovalUpdateMihomo           = "UPDATE_MIHOMO"
	MihomoApprovalSubscriptionCreate   = "subscription_create"
	MihomoApprovalSubscriptionUpdate   = "subscription_update"
	MihomoApprovalSubscriptionDelete   = "subscription_delete"
	MihomoApprovalSubscriptionRefresh  = "subscription_refresh"
	MihomoApprovalRouteCreate          = "route_create"
	MihomoApprovalRouteUpdate          = "route_update"
	MihomoApprovalRouteDelete          = "route_delete"
	MihomoApprovalNodeAction           = "node_action"
	MihomoApprovalLegacyImport         = "legacy_import"

	PoolApprovalPending  = "pending"
	PoolApprovalApproved = "approved"
	PoolApprovalRejected = "rejected"
	PoolApprovalExpired  = "expired"
	PoolApprovalConsumed = "consumed"

	poolApprovalPendingTTL = 7 * 24 * time.Hour
	poolRevealGrantTTL     = 5 * time.Minute
)

var (
	ErrPoolApprovalNotFound = infraerrors.NotFound("POOL_APPROVAL_NOT_FOUND", "approval request not found")
	ErrPoolApprovalConflict = infraerrors.Conflict("POOL_APPROVAL_CONFLICT", "an active approval request already exists")
	ErrPoolApprovalStale    = infraerrors.Conflict("POOL_APPROVAL_STALE", "account state changed after the request was created")
)

type PoolApprovalPayload struct {
	AccountUpdate *UpdateAccountInput     `json:"account_update,omitempty"`
	PoolUpdate    *UpdatePoolAccountInput `json:"pool_update,omitempty"`
	CostUpdate    *PoolCostUpdate         `json:"cost_update,omitempty"`
	ExtraMerge    map[string]any          `json:"extra_merge,omitempty"`
	Reauthorize   bool                    `json:"reauthorize,omitempty"`
	DeleteOptions *AccountDeleteOptions   `json:"delete_options,omitempty"`
	ProxyUpdate   *UpdateProxyInput       `json:"proxy_update,omitempty"`
	ProxyIDs      []int64                 `json:"proxy_ids,omitempty"`
	MihomoUpdate  *MihomoApprovalUpdate   `json:"mihomo_update,omitempty"`
}

type MihomoApprovalUpdate struct {
	Kind                   string                `json:"kind"`
	Mode                   string                `json:"mode,omitempty"`
	Selection              string                `json:"selection,omitempty"`
	Subscription           string                `json:"subscription,omitempty"`
	SubscriptionID         int64                 `json:"subscription_id,omitempty"`
	SubscriptionName       string                `json:"subscription_name,omitempty"`
	SubscriptionHost       string                `json:"subscription_host,omitempty"`
	RefreshIntervalMinutes int                   `json:"refresh_interval_minutes,omitempty"`
	RouteID                int64                 `json:"route_id,omitempty"`
	RouteName              string                `json:"route_name,omitempty"`
	RouteKind              string                `json:"route_kind,omitempty"`
	ListenerPort           int                   `json:"listener_port,omitempty"`
	ProxyID                int64                 `json:"proxy_id,omitempty"`
	SubscriptionIDs        []int64               `json:"subscription_ids,omitempty"`
	NodeIDs                []int64               `json:"node_ids,omitempty"`
	NodeAction             string                `json:"node_action,omitempty"`
	Enabled                *bool                 `json:"enabled,omitempty"`
	NodeCount              int64                 `json:"node_count,omitempty"`
	RouteCount             int64                 `json:"route_count,omitempty"`
	AccountCount           int64                 `json:"account_count,omitempty"`
	ImportProviderName     string                `json:"import_provider_name,omitempty"`
	ImportRoutes           []MihomoApprovalRoute `json:"import_routes,omitempty"`
}

type MihomoApprovalRoute struct {
	Name            string   `json:"name"`
	Kind            string   `json:"kind"`
	ListenerPort    int      `json:"listener_port"`
	ProxyID         int64    `json:"proxy_id"`
	SubscriptionIDs []int64  `json:"subscription_ids,omitempty"`
	NodeIDs         []int64  `json:"node_ids,omitempty"`
	NodeNames       []string `json:"node_names,omitempty"`
	NodeCount       int64    `json:"node_count,omitempty"`
	AccountCount    int64    `json:"account_count,omitempty"`
}

type PoolCostUpdate struct {
	CostID int64                  `json:"cost_id"`
	Cost   CreateAccountCostInput `json:"cost"`
}

type PoolApprovalValueChange struct {
	Before any `json:"before,omitempty"`
	After  any `json:"after,omitempty"`
}

type PoolApprovalChangeSummary struct {
	Fields         map[string]PoolApprovalValueChange `json:"fields,omitempty"`
	CredentialKeys []string                           `json:"credential_keys,omitempty"`
	ExtraKeys      []string                           `json:"extra_keys,omitempty"`
	Business       PoolApprovalBusinessSummary        `json:"business"`
}

type PoolApprovalBusinessSummary struct {
	Action   string                       `json:"action"`
	Object   PoolApprovalBusinessObject   `json:"object"`
	Scope    []string                     `json:"scope,omitempty"`
	Groups   []PoolApprovalBusinessGroup  `json:"groups,omitempty"`
	Impacts  []PoolApprovalBusinessImpact `json:"impacts,omitempty"`
	HighRisk bool                         `json:"high_risk"`
}

type PoolApprovalBusinessObject struct {
	Type string `json:"type"`
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type PoolApprovalBusinessGroup struct {
	Key   string                       `json:"key"`
	Items []PoolApprovalBusinessChange `json:"items"`
}

type PoolApprovalBusinessChange struct {
	Key       string `json:"key"`
	Before    any    `json:"before,omitempty"`
	After     any    `json:"after,omitempty"`
	Sensitive bool   `json:"sensitive,omitempty"`
	Impact    string `json:"impact"`
}

type PoolApprovalBusinessImpact struct {
	Key   string `json:"key"`
	Count int64  `json:"count"`
}

type PoolAccountDeleteImpact struct {
	Accounts               int64
	CredentialKeys         int64
	SchedulingRecords      int64
	CostEntries            int64
	Settlements            int64
	SettlementAccountCosts int64
	SettlementAccountLines int64
	MixedSettlements       int64
	EmptySettlements       int64
	PurchaseSources        int64
	GroupLinks             int64
	LifecycleEvents        int64
	UsageRecords           int64
}

// PoolApproval never serializes Payload: it can contain pending credential
// replacements. Reviewers receive the intentionally redacted Changes instead.
type PoolApproval struct {
	ID                int64                     `json:"id"`
	ActionType        string                    `json:"action_type"`
	AccountID         int64                     `json:"account_id"`
	AccountName       string                    `json:"account_name"`
	ObjectType        string                    `json:"object_type"`
	ProxyID           *int64                    `json:"proxy_id,omitempty"`
	ProxyName         string                    `json:"proxy_name,omitempty"`
	ResourceKey       string                    `json:"resource_key,omitempty"`
	Status            string                    `json:"status"`
	Reason            string                    `json:"reason"`
	BaseRevision      string                    `json:"base_revision"`
	RequestedByUserID int64                     `json:"requested_by_user_id"`
	RequestedByEmail  string                    `json:"requested_by_email"`
	DecidedByUserID   *int64                    `json:"decided_by_user_id,omitempty"`
	DecidedByEmail    *string                   `json:"decided_by_email,omitempty"`
	DecisionReason    *string                   `json:"decision_reason,omitempty"`
	RequestedAt       time.Time                 `json:"requested_at"`
	ExpiresAt         time.Time                 `json:"expires_at"`
	DecidedAt         *time.Time                `json:"decided_at,omitempty"`
	RevealExpiresAt   *time.Time                `json:"reveal_expires_at,omitempty"`
	RevealedAt        *time.Time                `json:"revealed_at,omitempty"`
	PrimaryBypass     bool                      `json:"is_primary_bypass"`
	Changes           PoolApprovalChangeSummary `json:"changes"`
	Payload           json.RawMessage           `json:"-"`
}

type PoolApprovalFilter struct {
	Status            string
	ActionType        string
	AccountID         *int64
	RequestedByUserID *int64
	Scope             string
	ActorID           int64
	HighRisk          *bool
	ObjectType        string
	ProxyID           *int64
}

type CreatePoolApprovalInput struct {
	ActionType        string
	AccountID         int64
	Reason            string
	RequesterID       int64
	Payload           PoolApprovalPayload
	ProxyID           *int64
	ResourceKey       string
	RequirePeerReview bool
}

type PoolApprovalAccountState struct {
	ProviderIdentity   *string
	ContributorUserID  *int64
	CreatedByUserID    *int64
	CostSharingEnabled bool
}

type CredentialReveal struct {
	AccountID   int64          `json:"account_id"`
	Credentials map[string]any `json:"credentials"`
	RevealedAt  time.Time      `json:"revealed_at"`
}

type ProxyCredentialReveal struct {
	Proxy      Proxy     `json:"proxy"`
	RevealedAt time.Time `json:"revealed_at"`
}

type ProxyExportReveal struct {
	Proxies    []Proxy   `json:"proxies"`
	RevealedAt time.Time `json:"revealed_at"`
}

type PoolApprovalRepository interface {
	ExpireStale(ctx context.Context, now time.Time) error
	CreateApproval(ctx context.Context, approval *PoolApproval) (*PoolApproval, error)
	ListApprovals(ctx context.Context, filter PoolApprovalFilter, limit, offset int) ([]PoolApproval, int64, error)
	GetApproval(ctx context.Context, id int64, forUpdate bool) (*PoolApproval, error)
	LockAccount(ctx context.Context, accountID int64) error
	LockProxy(ctx context.Context, proxyID int64) error
	GetApprovalAccountState(ctx context.Context, accountID int64) (*PoolApprovalAccountState, error)
	UpdatePoolAccountApproved(ctx context.Context, accountID int64, input UpdatePoolAccountInput) error
	UpdateCostApproved(ctx context.Context, update PoolCostUpdate) error
	InvalidateCredentialApprovals(ctx context.Context, accountID int64, reason string) error
	SetDecision(ctx context.Context, id int64, status string, actorID int64, reason *string, revealExpiresAt *time.Time) error
	MarkExpired(ctx context.Context, id int64, reason string) error
	LoadCredentials(ctx context.Context, accountID int64) (map[string]any, error)
	ConsumeReveal(ctx context.Context, id int64, revealedAt time.Time) error
	InvalidateProxyCredentialApprovals(ctx context.Context, proxyID int64, reason string) error
}

type poolApprovalDeleteImpactReader interface {
	GetAccountDeleteImpact(ctx context.Context, accountID int64) (*PoolAccountDeleteImpact, error)
}

func (s *PoolService) IsPrimaryAdmin(ctx context.Context, userID int64) bool {
	return s != nil && s.settings != nil && s.settings.IsPrimaryAdmin(ctx, userID)
}

func (s *PoolService) CreateApproval(ctx context.Context, input CreatePoolApprovalInput) (*PoolApproval, error) {
	if s == nil || s.approvalRepo == nil || s.adminService == nil {
		return nil, fmt.Errorf("pool approval service is not configured")
	}
	input.ActionType = strings.ToUpper(strings.TrimSpace(input.ActionType))
	input.Reason = strings.TrimSpace(input.Reason)
	if input.RequesterID <= 0 {
		return nil, infraerrors.BadRequest("INVALID_APPROVAL_PARTY", "requester is required")
	}
	primaryBypass := s.approvalPrimaryBypass(ctx, input)
	if input.Reason == "" && primaryBypass && input.ActionType == PoolApprovalViewCredential {
		input.Reason = "primary administrator direct credential access"
	}
	if input.Reason == "" || len(input.Reason) > 1000 {
		return nil, infraerrors.BadRequest("INVALID_APPROVAL_REASON", "reason is required and must not exceed 1000 characters")
	}
	if err := validateApprovalPayload(input.ActionType, input.Payload); err != nil {
		return nil, err
	}
	if err := s.approvalRepo.ExpireStale(ctx, time.Now().UTC()); err != nil {
		return nil, err
	}

	var revision string
	var summary PoolApprovalChangeSummary
	objectType := "account"
	var accountID int64
	var proxyID *int64
	resourceKey := ""
	var err error

	switch {
	case isAccountApprovalAction(input.ActionType):
		if input.AccountID <= 0 {
			return nil, infraerrors.BadRequest("INVALID_APPROVAL_PARTY", "account is required")
		}
		accountID = input.AccountID
		account, getErr := s.adminService.GetAccount(ctx, accountID)
		if getErr != nil {
			return nil, getErr
		}
		poolState, stateErr := s.approvalRepo.GetApprovalAccountState(ctx, accountID)
		if stateErr != nil {
			return nil, stateErr
		}
		var costBefore *AccountCostEntry
		if input.Payload.CostUpdate != nil {
			costBefore, err = s.approvalCostEntry(ctx, accountID, input.Payload.CostUpdate.CostID)
			if err != nil {
				return nil, err
			}
		}
		revision, err = poolApprovalRevision(account, poolState, input.Payload)
		if err != nil {
			return nil, err
		}
		summary = buildPoolApprovalSummary(account, poolState, costBefore, input.Payload)
		var deleteImpact *PoolAccountDeleteImpact
		if input.Payload.DeleteOptions != nil {
			deleteImpact = &PoolAccountDeleteImpact{Accounts: 1, CredentialKeys: int64(len(account.Credentials))}
			if reader, ok := s.approvalRepo.(poolApprovalDeleteImpactReader); ok {
				deleteImpact, err = reader.GetAccountDeleteImpact(ctx, accountID)
				if err != nil {
					return nil, err
				}
			}
		}
		summary.Business = buildPoolApprovalBusinessSummary(input.ActionType, account, summary, deleteImpact)

	case input.ActionType == PoolApprovalUpdateProxy || input.ActionType == PoolApprovalViewProxyCredential:
		if input.ProxyID == nil || *input.ProxyID <= 0 {
			return nil, infraerrors.BadRequest("INVALID_APPROVAL_PARTY", "proxy is required")
		}
		objectType, proxyID = "proxy", input.ProxyID
		proxy, getErr := s.adminService.GetProxy(ctx, *proxyID)
		if getErr != nil {
			return nil, getErr
		}
		if input.ActionType == PoolApprovalUpdateProxy && proxy.ManagedSource != nil {
			return nil, infraerrors.Conflict("MANAGED_PROXY_UPDATE_FORBIDDEN", "managed proxies are changed from the Mihomo panel")
		}
		accounts, listErr := s.adminService.GetProxyAccounts(ctx, *proxyID)
		if listErr != nil {
			return nil, listErr
		}
		revision, err = proxyApprovalRevision(proxy, accounts)
		if err != nil {
			return nil, err
		}
		summary = buildProxyApprovalSummary(input.ActionType, proxy, input.Payload.ProxyUpdate, int64(len(accounts)))

	case input.ActionType == PoolApprovalExportProxyCredentials:
		objectType, resourceKey = "proxy_export", "proxy-export"
		proxies, getErr := s.adminService.GetProxiesByIDs(ctx, input.Payload.ProxyIDs)
		if getErr != nil {
			return nil, getErr
		}
		if len(proxies) == 0 || len(proxies) != len(input.Payload.ProxyIDs) {
			return nil, infraerrors.BadRequest("INVALID_PROXY_EXPORT", "at least one valid proxy is required")
		}
		revision, err = proxyExportApprovalRevision(proxies)
		if err != nil {
			return nil, err
		}
		summary = buildProxyExportApprovalSummary(proxies)

	case input.ActionType == PoolApprovalUpdateMihomo:
		objectType, resourceKey = "mihomo", strings.TrimSpace(input.ResourceKey)
		if resourceKey == "" || input.Payload.MihomoUpdate == nil || s.mihomoApprovalExecutor == nil {
			return nil, infraerrors.BadRequest("INVALID_MIHOMO_APPROVAL", "mihomo target and update are required")
		}
		if err = s.mihomoApprovalExecutor.ValidateApproval(ctx, resourceKey, *input.Payload.MihomoUpdate); err != nil {
			return nil, err
		}
		revision, err = s.mihomoApprovalExecutor.ApprovalRevision(ctx, resourceKey)
		if err != nil {
			return nil, err
		}
		summary = buildMihomoApprovalSummary(resourceKey, input.Payload.MihomoUpdate)

	default:
		return nil, infraerrors.BadRequest("INVALID_APPROVAL_ACTION", "unsupported approval action")
	}

	payload, err := json.Marshal(input.Payload)
	if err != nil {
		return nil, infraerrors.BadRequest("INVALID_APPROVAL_PAYLOAD", "approval payload is not serializable")
	}
	now := time.Now().UTC()
	created, err := s.approvalRepo.CreateApproval(ctx, &PoolApproval{
		ActionType: input.ActionType, AccountID: accountID, ObjectType: objectType,
		ProxyID: proxyID, ResourceKey: resourceKey, Status: PoolApprovalPending,
		Reason: input.Reason, BaseRevision: revision, RequestedByUserID: input.RequesterID,
		RequestedAt: now, ExpiresAt: now.Add(poolApprovalPendingTTL),
		PrimaryBypass: primaryBypass,
		Changes:       summary,
		Payload:       payload,
	})
	if err != nil {
		return nil, err
	}
	if created.PrimaryBypass {
		return s.ApproveApproval(ctx, created.ID, input.RequesterID, "primary administrator bypass")
	}
	return created, nil
}

func (s *PoolService) approvalPrimaryBypass(ctx context.Context, input CreatePoolApprovalInput) bool {
	if !s.IsPrimaryAdmin(ctx, input.RequesterID) {
		return false
	}
	return input.ActionType == PoolApprovalUpdateMihomo ||
		(!input.RequirePeerReview && isAccountApprovalAction(input.ActionType))
}

func (s *PoolService) RequestCostUpdate(ctx context.Context, costID, accountID, actorID int64, reason string, cost CreateAccountCostInput, poolUpdate *UpdatePoolAccountInput) (*PoolApproval, error) {
	cost.AccountID = accountID
	cost.CreatedByUserID = actorID
	cost.SupersedesID = nil
	cost.OperationKey = ""
	normalized, err := normalizePoolCostInput(cost)
	if err != nil {
		return nil, err
	}
	normalized.OrderAccountKey = poolCostOrderAccountKey(normalized)
	return s.CreateApproval(ctx, CreatePoolApprovalInput{
		ActionType: PoolApprovalUpdateAccount, AccountID: accountID, RequesterID: actorID,
		Reason: strings.TrimSpace(reason), Payload: PoolApprovalPayload{
			PoolUpdate: poolUpdate, CostUpdate: &PoolCostUpdate{CostID: costID, Cost: normalized},
		},
	})
}

func (s *PoolService) approvalCostEntry(ctx context.Context, accountID, costID int64) (*AccountCostEntry, error) {
	if costID <= 0 {
		return nil, infraerrors.BadRequest("INVALID_COST_ENTRY", "invalid cost entry")
	}
	items, err := s.repo.ListCosts(ctx, &accountID)
	if err != nil {
		return nil, err
	}
	for i := range items {
		if items[i].ID == costID {
			return &items[i], nil
		}
	}
	return nil, infraerrors.NotFound("POOL_COST_NOT_FOUND", "cost entry not found")
}

func validateApprovalPayload(action string, payload PoolApprovalPayload) error {
	switch action {
	case PoolApprovalUpdateAccount:
		if payload.DeleteOptions != nil {
			return infraerrors.BadRequest("INVALID_UPDATE_APPROVAL_PAYLOAD", "account update requests do not accept delete_options")
		}
		if !hasAccountApprovalUpdate(payload.AccountUpdate) && !hasPoolApprovalUpdate(payload.PoolUpdate) && payload.CostUpdate == nil && len(payload.ExtraMerge) == 0 {
			return infraerrors.BadRequest("EMPTY_APPROVAL_UPDATE", "account_update, pool_update, or cost_update is required")
		}
	case PoolApprovalViewCredential:
		if payload.AccountUpdate != nil || payload.PoolUpdate != nil || payload.CostUpdate != nil || len(payload.ExtraMerge) > 0 || payload.Reauthorize || payload.DeleteOptions != nil {
			return infraerrors.BadRequest("INVALID_CREDENTIAL_APPROVAL_PAYLOAD", "credential view requests do not accept update payloads")
		}
	case PoolApprovalDeleteAccount:
		if payload.DeleteOptions == nil || payload.AccountUpdate != nil || payload.PoolUpdate != nil || payload.CostUpdate != nil || len(payload.ExtraMerge) > 0 || payload.Reauthorize {
			return infraerrors.BadRequest("INVALID_DELETE_APPROVAL_PAYLOAD", "delete approval requires delete_options only")
		}
	case PoolApprovalUpdateProxy:
		if hasNonProxyApprovalPayload(payload) || len(payload.ProxyIDs) > 0 || payload.MihomoUpdate != nil {
			return infraerrors.BadRequest("INVALID_PROXY_APPROVAL_PAYLOAD", "proxy update accepts proxy_update only")
		}
		if err := validateProxyApprovalUpdate(payload.ProxyUpdate); err != nil {
			return err
		}
	case PoolApprovalViewProxyCredential:
		if hasNonProxyApprovalPayload(payload) || payload.ProxyUpdate != nil || len(payload.ProxyIDs) > 0 || payload.MihomoUpdate != nil {
			return infraerrors.BadRequest("INVALID_PROXY_VIEW_PAYLOAD", "proxy credential view does not accept updates")
		}
	case PoolApprovalExportProxyCredentials:
		if hasNonProxyApprovalPayload(payload) || len(payload.ProxyIDs) == 0 || payload.ProxyUpdate != nil || payload.MihomoUpdate != nil {
			return infraerrors.BadRequest("INVALID_PROXY_EXPORT_PAYLOAD", "proxy_ids are required")
		}
	case PoolApprovalUpdateMihomo:
		if hasNonProxyApprovalPayload(payload) || payload.MihomoUpdate == nil || payload.ProxyUpdate != nil || len(payload.ProxyIDs) > 0 {
			return infraerrors.BadRequest("INVALID_MIHOMO_APPROVAL_PAYLOAD", "mihomo_update is required")
		}
	default:
		return infraerrors.BadRequest("INVALID_APPROVAL_ACTION", "unsupported approval action")
	}
	return nil
}

func hasNonProxyApprovalPayload(payload PoolApprovalPayload) bool {
	return payload.AccountUpdate != nil || payload.PoolUpdate != nil || payload.CostUpdate != nil || len(payload.ExtraMerge) > 0 || payload.Reauthorize || payload.DeleteOptions != nil
}

func validateProxyApprovalUpdate(update *UpdateProxyInput) error {
	if update == nil {
		return infraerrors.BadRequest("EMPTY_PROXY_APPROVAL_UPDATE", "proxy_update is required")
	}
	if strings.TrimSpace(update.Name) == "" || strings.TrimSpace(update.Host) == "" || update.Port < 1 || update.Port > 65535 {
		return infraerrors.BadRequest("INVALID_PROXY_APPROVAL_UPDATE", "proxy name, host, and port are required")
	}
	switch update.Protocol {
	case "http", "https", "socks5", "socks5h":
	default:
		return infraerrors.BadRequest("INVALID_PROXY_APPROVAL_UPDATE", "unsupported proxy protocol")
	}
	if update.Status != StatusActive && update.Status != StatusDisabled {
		return infraerrors.BadRequest("INVALID_PROXY_APPROVAL_UPDATE", "proxy status must be active or inactive")
	}
	if update.FallbackMode != FallbackModeNone && update.FallbackMode != FallbackModeProxy && update.FallbackMode != FallbackModeDirect {
		return infraerrors.BadRequest("INVALID_PROXY_APPROVAL_UPDATE", "invalid proxy fallback mode")
	}
	if (update.FallbackMode == FallbackModeProxy) != (update.BackupProxyID != nil) {
		return infraerrors.BadRequest("INVALID_PROXY_APPROVAL_UPDATE", "backup proxy is required only for proxy fallback mode")
	}
	if update.ExpiryWarnDays < 0 {
		return infraerrors.BadRequest("INVALID_PROXY_APPROVAL_UPDATE", "expiry warning days must not be negative")
	}
	return nil
}

func isAccountApprovalAction(action string) bool {
	return action == PoolApprovalUpdateAccount || action == PoolApprovalViewCredential || action == PoolApprovalDeleteAccount
}

func hasAccountApprovalUpdate(v *UpdateAccountInput) bool {
	return v != nil && (v.Name != "" || v.Notes != nil || v.Type != "" || len(v.Credentials) > 0 || v.Extra != nil ||
		v.ProxyID != nil || v.Concurrency != nil || v.Priority != nil || v.RateMultiplier != nil || v.LoadFactor != nil ||
		v.Status != "" || v.GroupIDs != nil || v.ExpiresAt != nil || v.AutoPauseOnExpired != nil)
}

func hasPoolApprovalUpdate(v *UpdatePoolAccountInput) bool {
	return v != nil && (v.ProviderIdentity != nil || v.ContributorUserID != nil || v.CreatedByUserID != nil || v.CostSharingEnabled != nil)
}

func (s *PoolService) ListApprovals(ctx context.Context, filter PoolApprovalFilter, page, pageSize int) ([]PoolApproval, int64, error) {
	if err := s.approvalRepo.ExpireStale(ctx, time.Now().UTC()); err != nil {
		return nil, 0, err
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 200 {
		pageSize = 20
	}
	filter.Status = strings.ToLower(strings.TrimSpace(filter.Status))
	filter.ActionType = strings.ToUpper(strings.TrimSpace(filter.ActionType))
	filter.Scope = strings.ToLower(strings.TrimSpace(filter.Scope))
	if filter.Scope != "" {
		if filter.ActorID <= 0 {
			return nil, 0, infraerrors.BadRequest("INVALID_APPROVAL_SCOPE", "approval scope requires an actor")
		}
		switch filter.Scope {
		case "reviewable", "mine", "processed":
		default:
			return nil, 0, infraerrors.BadRequest("INVALID_APPROVAL_SCOPE", "approval scope must be reviewable, mine, or processed")
		}
	}
	return s.approvalRepo.ListApprovals(ctx, filter, pageSize, (page-1)*pageSize)
}

func (s *PoolService) GetApproval(ctx context.Context, id int64) (*PoolApproval, error) {
	if id <= 0 {
		return nil, infraerrors.BadRequest("INVALID_APPROVAL_ID", "invalid approval id")
	}
	if err := s.approvalRepo.ExpireStale(ctx, time.Now().UTC()); err != nil {
		return nil, err
	}
	return s.approvalRepo.GetApproval(ctx, id, false)
}

func (s *PoolService) ApproveApproval(ctx context.Context, id, actorID int64, reason string) (*PoolApproval, error) {
	return s.decideApproval(ctx, id, actorID, strings.TrimSpace(reason), true)
}

func (s *PoolService) RejectApproval(ctx context.Context, id, actorID int64, reason string) (*PoolApproval, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil, infraerrors.BadRequest("APPROVAL_REJECTION_REASON_REQUIRED", "rejection reason is required")
	}
	return s.decideApproval(ctx, id, actorID, reason, false)
}

func (s *PoolService) decideApproval(ctx context.Context, id, actorID int64, reason string, approve bool) (*PoolApproval, error) {
	if id <= 0 || actorID <= 0 {
		return nil, infraerrors.BadRequest("INVALID_APPROVAL_DECISION", "approval and actor are required")
	}
	if len(reason) > 1000 {
		return nil, infraerrors.BadRequest("INVALID_APPROVAL_DECISION_REASON", "decision reason must not exceed 1000 characters")
	}
	if err := s.approvalRepo.ExpireStale(ctx, time.Now().UTC()); err != nil {
		return nil, err
	}
	if s.entClient == nil {
		return nil, fmt.Errorf("approval transaction client is not configured")
	}
	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	txCtx := dbent.NewTxContext(ctx, tx)

	item, err := s.approvalRepo.GetApproval(txCtx, id, true)
	if err != nil {
		return nil, err
	}
	if item.Status != PoolApprovalPending {
		return nil, infraerrors.Conflict("APPROVAL_ALREADY_DECIDED", "approval request is no longer pending")
	}
	canSelfDecide := item.PrimaryBypass && s.IsPrimaryAdmin(txCtx, actorID)
	if item.ActionType == PoolApprovalUpdateMihomo {
		canSelfDecide = s.approvalPrimaryBypass(txCtx, CreatePoolApprovalInput{
			ActionType: PoolApprovalUpdateMihomo, RequesterID: actorID, RequirePeerReview: true,
		})
	}
	if err := validatePoolApprovalDecisionActor(item, actorID, canSelfDecide); err != nil {
		return nil, err
	}

	decisionReason := optionalTrimmedString(reason)
	if !approve {
		if err := s.approvalRepo.SetDecision(txCtx, id, PoolApprovalRejected, actorID, decisionReason, nil); err != nil {
			return nil, err
		}
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		return s.approvalRepo.GetApproval(ctx, id, false)
	}

	var payload PoolApprovalPayload
	if err := json.Unmarshal(item.Payload, &payload); err != nil {
		return nil, fmt.Errorf("decode approval payload: %w", err)
	}
	var account *Account
	var poolState *PoolApprovalAccountState
	var revision string
	switch {
	case isAccountApprovalAction(item.ActionType):
		if err := s.approvalRepo.LockAccount(txCtx, item.AccountID); err != nil {
			return nil, err
		}
		account, err = s.adminService.GetAccount(txCtx, item.AccountID)
		if err != nil {
			return nil, err
		}
		poolState, err = s.approvalRepo.GetApprovalAccountState(txCtx, item.AccountID)
		if err != nil {
			return nil, err
		}
		revision, err = poolApprovalRevision(account, poolState, payload)
	case item.ActionType == PoolApprovalUpdateProxy || item.ActionType == PoolApprovalViewProxyCredential:
		if item.ProxyID == nil {
			return nil, infraerrors.BadRequest("INVALID_PROXY_APPROVAL", "proxy target is missing")
		}
		if err = s.approvalRepo.LockProxy(txCtx, *item.ProxyID); err != nil {
			return nil, err
		}
		var proxy *Proxy
		proxy, err = s.adminService.GetProxy(txCtx, *item.ProxyID)
		if err == nil {
			if item.ActionType == PoolApprovalUpdateProxy && proxy.ManagedSource != nil {
				return nil, infraerrors.Conflict("MANAGED_PROXY_UPDATE_FORBIDDEN", "managed proxies are changed from the Mihomo panel")
			}
			var accounts []ProxyAccountSummary
			accounts, err = s.adminService.GetProxyAccounts(txCtx, *item.ProxyID)
			if err == nil {
				revision, err = proxyApprovalRevision(proxy, accounts)
			}
		}
	case item.ActionType == PoolApprovalExportProxyCredentials:
		for _, proxyID := range payload.ProxyIDs {
			if err = s.approvalRepo.LockProxy(txCtx, proxyID); err != nil {
				return nil, err
			}
		}
		var proxies []Proxy
		proxies, err = s.adminService.GetProxiesByIDs(txCtx, payload.ProxyIDs)
		if err == nil {
			revision, err = proxyExportApprovalRevision(proxies)
		}
	case item.ActionType == PoolApprovalUpdateMihomo:
		if s.mihomoApprovalExecutor == nil {
			return nil, fmt.Errorf("mihomo approval executor is not configured")
		}
		if update := payload.MihomoUpdate; update != nil && update.Kind == MihomoApprovalRouteDelete && update.ProxyID > 0 {
			if err = s.approvalRepo.LockProxy(txCtx, update.ProxyID); err != nil {
				return nil, err
			}
		}
		revision, err = s.mihomoApprovalExecutor.ApprovalRevision(txCtx, item.ResourceKey)
	default:
		return nil, infraerrors.BadRequest("INVALID_APPROVAL_ACTION", "invalid stored approval action")
	}
	if err != nil {
		return nil, err
	}
	if revision != item.BaseRevision {
		if err := s.approvalRepo.MarkExpired(txCtx, item.ID, "target changed after request creation"); err != nil {
			return nil, err
		}
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		return s.approvalRepo.GetApproval(ctx, id, false)
	}

	var revealExpiresAt *time.Time
	var approvedAccount *Account
	var finalizeExternal func(bool) error
	externalCommitted := false
	defer func() {
		if finalizeExternal != nil {
			if finalizeErr := finalizeExternal(externalCommitted); finalizeErr != nil {
				slog.Error("mihomo approval compensation failed", "approval_id", id, "error", finalizeErr)
			}
		}
	}()
	switch item.ActionType {
	case PoolApprovalUpdateAccount:
		if payload.AccountUpdate != nil {
			approvedAccount, err = s.adminService.UpdateAccount(txCtx, item.AccountID, payload.AccountUpdate)
			if err != nil {
				return nil, err
			}
			if len(payload.AccountUpdate.Credentials) > 0 {
				if err := s.approvalRepo.InvalidateCredentialApprovals(txCtx, item.AccountID, "credentials changed"); err != nil {
					return nil, err
				}
			}
		}
		if len(payload.ExtraMerge) > 0 {
			if err := s.adminService.UpdateAccountExtra(txCtx, item.AccountID, payload.ExtraMerge); err != nil {
				return nil, err
			}
		}
		if payload.PoolUpdate != nil {
			if err := s.approvalRepo.UpdatePoolAccountApproved(txCtx, item.AccountID, *payload.PoolUpdate); err != nil {
				return nil, err
			}
		}
		if payload.CostUpdate != nil {
			if err := s.approvalRepo.UpdateCostApproved(txCtx, *payload.CostUpdate); err != nil {
				return nil, err
			}
		}
	case PoolApprovalViewCredential:
		expires := time.Now().UTC().Add(poolRevealGrantTTL)
		revealExpiresAt = &expires
	case PoolApprovalUpdateProxy:
		if item.ProxyID == nil || payload.ProxyUpdate == nil {
			return nil, infraerrors.BadRequest("INVALID_PROXY_APPROVAL", "proxy update is missing")
		}
		if _, err = s.adminService.UpdateProxy(txCtx, *item.ProxyID, payload.ProxyUpdate); err != nil {
			return nil, err
		}
		if err = s.approvalRepo.InvalidateProxyCredentialApprovals(txCtx, *item.ProxyID, "proxy connection changed"); err != nil {
			return nil, err
		}
	case PoolApprovalViewProxyCredential, PoolApprovalExportProxyCredentials:
		expires := time.Now().UTC().Add(poolRevealGrantTTL)
		revealExpiresAt = &expires
	case PoolApprovalUpdateMihomo:
		if payload.MihomoUpdate == nil {
			return nil, infraerrors.BadRequest("INVALID_MIHOMO_APPROVAL", "mihomo update is missing")
		}
		finalizeExternal, err = s.mihomoApprovalExecutor.ApplyApproved(txCtx, *payload.MihomoUpdate)
		if err != nil {
			return nil, err
		}
	case PoolApprovalDeleteAccount:
		deleter, ok := s.adminService.(interface {
			DeleteAccountWithOptions(context.Context, int64, AccountDeleteOptions) (*AccountDeleteResult, error)
		})
		if !ok || payload.DeleteOptions == nil {
			return nil, fmt.Errorf("account lifecycle deletion service is not configured")
		}
		if err := s.approvalRepo.SetDecision(txCtx, id, PoolApprovalApproved, actorID, decisionReason, nil); err != nil {
			return nil, err
		}
		if _, err = deleter.DeleteAccountWithOptions(txCtx, item.AccountID, *payload.DeleteOptions); err != nil {
			return nil, err
		}
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		now := time.Now().UTC()
		item.Status, item.DecidedByUserID, item.DecisionReason, item.DecidedAt = PoolApprovalApproved, &actorID, decisionReason, &now
		item.Payload = nil
		return item, nil
	default:
		return nil, infraerrors.BadRequest("INVALID_APPROVAL_ACTION", "invalid stored approval action")
	}
	if err := s.approvalRepo.SetDecision(txCtx, id, PoolApprovalApproved, actorID, decisionReason, revealExpiresAt); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	externalCommitted = true
	if payload.Reauthorize {
		if cleared, clearErr := s.adminService.ClearAccountError(ctx, item.AccountID); clearErr != nil {
			slog.Warn("pool approval reauthorization clear error failed", "account_id", item.AccountID, "error", clearErr)
		} else if cleared != nil {
			approvedAccount = cleared
		}
		if s.tokenCache != nil && approvedAccount != nil && approvedAccount.IsOAuth() {
			if invalidateErr := s.tokenCache.InvalidateToken(ctx, approvedAccount); invalidateErr != nil {
				slog.Warn("pool approval reauthorization token invalidation failed", "account_id", item.AccountID, "error", invalidateErr)
			}
		}
	}
	return s.approvalRepo.GetApproval(ctx, id, false)
}

func (s *PoolService) RevealCredential(ctx context.Context, id, actorID int64) (*CredentialReveal, error) {
	if id <= 0 || actorID <= 0 {
		return nil, infraerrors.BadRequest("INVALID_CREDENTIAL_REVEAL", "approval and actor are required")
	}
	if err := s.approvalRepo.ExpireStale(ctx, time.Now().UTC()); err != nil {
		return nil, err
	}
	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	txCtx := dbent.NewTxContext(ctx, tx)
	item, err := s.approvalRepo.GetApproval(txCtx, id, true)
	if err != nil {
		return nil, err
	}
	if item.ActionType != PoolApprovalViewCredential {
		return nil, infraerrors.BadRequest("APPROVAL_NOT_CREDENTIAL_VIEW", "approval does not grant credential access")
	}
	if item.RequestedByUserID != actorID {
		return nil, infraerrors.Forbidden("CREDENTIAL_GRANT_OWNER_MISMATCH", "credential grant belongs to another administrator")
	}
	if item.Status != PoolApprovalApproved || item.RevealExpiresAt == nil || !time.Now().UTC().Before(*item.RevealExpiresAt) {
		return nil, infraerrors.Conflict("CREDENTIAL_GRANT_UNAVAILABLE", "credential grant is expired, consumed, or not approved")
	}
	credentials, err := s.approvalRepo.LoadCredentials(txCtx, item.AccountID)
	if err != nil {
		return nil, err
	}
	revealedAt := time.Now().UTC()
	if err := s.approvalRepo.ConsumeReveal(txCtx, item.ID, revealedAt); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &CredentialReveal{AccountID: item.AccountID, Credentials: credentials, RevealedAt: revealedAt}, nil
}

func (s *PoolService) RevealProxyCredential(ctx context.Context, id, actorID int64) (*ProxyCredentialReveal, error) {
	item, tx, txCtx, err := s.lockRevealApproval(ctx, id, actorID, PoolApprovalViewProxyCredential)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	if item.ProxyID == nil {
		return nil, infraerrors.BadRequest("INVALID_PROXY_APPROVAL", "proxy target is missing")
	}
	if err = s.approvalRepo.LockProxy(txCtx, *item.ProxyID); err != nil {
		return nil, err
	}
	proxy, err := s.adminService.GetProxy(txCtx, *item.ProxyID)
	if err != nil {
		return nil, err
	}
	revealedAt := time.Now().UTC()
	if err = s.approvalRepo.ConsumeReveal(txCtx, item.ID, revealedAt); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return &ProxyCredentialReveal{Proxy: *proxy, RevealedAt: revealedAt}, nil
}

func (s *PoolService) RevealProxyExport(ctx context.Context, id, actorID int64) (*ProxyExportReveal, error) {
	item, tx, txCtx, err := s.lockRevealApproval(ctx, id, actorID, PoolApprovalExportProxyCredentials)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	var payload PoolApprovalPayload
	if err = json.Unmarshal(item.Payload, &payload); err != nil {
		return nil, fmt.Errorf("decode approval payload: %w", err)
	}
	for _, proxyID := range payload.ProxyIDs {
		if err = s.approvalRepo.LockProxy(txCtx, proxyID); err != nil {
			return nil, err
		}
	}
	proxies, err := s.adminService.GetProxiesByIDs(txCtx, payload.ProxyIDs)
	if err != nil {
		return nil, err
	}
	revision, err := proxyExportApprovalRevision(proxies)
	if err != nil {
		return nil, err
	}
	if len(proxies) != len(payload.ProxyIDs) || revision != item.BaseRevision {
		return nil, ErrPoolApprovalStale
	}
	revealedAt := time.Now().UTC()
	if err = s.approvalRepo.ConsumeReveal(txCtx, item.ID, revealedAt); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return &ProxyExportReveal{Proxies: proxies, RevealedAt: revealedAt}, nil
}

func (s *PoolService) lockRevealApproval(ctx context.Context, id, actorID int64, action string) (*PoolApproval, *dbent.Tx, context.Context, error) {
	if id <= 0 || actorID <= 0 {
		return nil, nil, nil, infraerrors.BadRequest("INVALID_CREDENTIAL_REVEAL", "approval and actor are required")
	}
	if err := s.approvalRepo.ExpireStale(ctx, time.Now().UTC()); err != nil {
		return nil, nil, nil, err
	}
	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return nil, nil, nil, err
	}
	txCtx := dbent.NewTxContext(ctx, tx)
	item, err := s.approvalRepo.GetApproval(txCtx, id, true)
	if err != nil {
		_ = tx.Rollback()
		return nil, nil, nil, err
	}
	if item.ActionType != action {
		_ = tx.Rollback()
		return nil, nil, nil, infraerrors.BadRequest("APPROVAL_NOT_CREDENTIAL_VIEW", "approval does not grant this access")
	}
	if item.RequestedByUserID != actorID {
		_ = tx.Rollback()
		return nil, nil, nil, infraerrors.Forbidden("CREDENTIAL_GRANT_OWNER_MISMATCH", "credential grant belongs to another administrator")
	}
	if item.Status != PoolApprovalApproved || item.RevealExpiresAt == nil || !time.Now().UTC().Before(*item.RevealExpiresAt) {
		_ = tx.Rollback()
		return nil, nil, nil, infraerrors.Conflict("CREDENTIAL_GRANT_UNAVAILABLE", "credential grant is expired, consumed, or not approved")
	}
	return item, tx, txCtx, nil
}

func poolApprovalRevision(account *Account, pool *PoolApprovalAccountState, payload PoolApprovalPayload) (string, error) {
	if account == nil || pool == nil {
		return "", ErrPoolAccountNotFound
	}
	snapshot := make(map[string]any)
	if u := payload.AccountUpdate; u != nil {
		if u.Name != "" {
			snapshot["name"] = account.Name
		}
		if u.Notes != nil {
			snapshot["notes"] = account.Notes
		}
		if u.Type != "" {
			snapshot["type"] = account.Type
		}
		if len(u.Credentials) > 0 {
			// Credential updates replace every non-sensitive key. Hash the complete
			// current object so a pending request cannot erase a concurrent change.
			snapshot["credentials"] = account.Credentials
		}
		if u.Extra != nil {
			snapshot["extra"] = approvalManagedExtra(account.Extra)
		}
		if u.ProxyID != nil {
			snapshot["proxy_id"] = account.ProxyID
		}
		if u.Concurrency != nil {
			snapshot["concurrency"] = account.Concurrency
		}
		if u.Priority != nil {
			snapshot["priority"] = account.Priority
		}
		if u.RateMultiplier != nil {
			snapshot["rate_multiplier"] = account.RateMultiplier
		}
		if u.LoadFactor != nil {
			snapshot["load_factor"] = account.LoadFactor
		}
		if u.Status != "" {
			snapshot["status"] = account.Status
		}
		if u.GroupIDs != nil {
			snapshot["group_ids"] = account.GroupIDs
		}
		if u.ExpiresAt != nil {
			snapshot["expires_at"] = account.ExpiresAt
		}
		if u.AutoPauseOnExpired != nil {
			snapshot["auto_pause_on_expired"] = account.AutoPauseOnExpired
		}
	}
	if u := payload.PoolUpdate; u != nil {
		if u.ProviderIdentity != nil {
			snapshot["provider_identity"] = pool.ProviderIdentity
		}
		if u.ContributorUserID != nil {
			snapshot["contributor_user_id"] = pool.ContributorUserID
		}
		if u.CreatedByUserID != nil {
			snapshot["created_by_user_id"] = pool.CreatedByUserID
		}
		if u.CostSharingEnabled != nil {
			snapshot["cost_sharing_enabled"] = pool.CostSharingEnabled
		}
	}
	if len(payload.ExtraMerge) > 0 {
		current := make(map[string]any, len(payload.ExtraMerge))
		for key := range payload.ExtraMerge {
			current[key] = account.Extra[key]
		}
		snapshot["extra_merge"] = current
	}
	if payload.CostUpdate != nil {
		snapshot["cost_entry_updated_at"] = account.UpdatedAt
	}
	if payload.DeleteOptions != nil {
		snapshot["delete_account"] = map[string]any{
			"account_updated_at": account.UpdatedAt,
			"provider_identity":  pool.ProviderIdentity,
			"cost_sharing":       pool.CostSharingEnabled,
		}
	}
	raw, err := json.Marshal(snapshot)
	if err != nil {
		return "", fmt.Errorf("encode approval revision: %w", err)
	}
	digest := sha256.Sum256(raw)
	return hex.EncodeToString(digest[:]), nil
}

func approvalManagedExtra(extra map[string]any) map[string]any {
	out := make(map[string]any)
	for key, value := range extra {
		if isApprovalRuntimeExtraKey(key) {
			continue
		}
		out[key] = value
	}
	return out
}

func isApprovalRuntimeExtraKey(key string) bool {
	for _, prefix := range []string{"codex_", "passive_usage_", "model_rate_limits", "upstream_billing_probe", "ollama_cloud_usage_", "grok_billing_", "grok_usage_", "quota_used", "quota_daily_used", "quota_weekly_used"} {
		if strings.HasPrefix(key, prefix) {
			return true
		}
	}
	return false
}

func buildPoolApprovalSummary(account *Account, pool *PoolApprovalAccountState, costBefore *AccountCostEntry, payload PoolApprovalPayload) PoolApprovalChangeSummary {
	result := PoolApprovalChangeSummary{Fields: make(map[string]PoolApprovalValueChange)}
	add := func(key string, before, after any) {
		before = approvalSummaryValue(before)
		after = approvalSummaryValue(after)
		if reflect.DeepEqual(before, after) {
			return
		}
		result.Fields[key] = PoolApprovalValueChange{Before: before, After: after}
	}
	addMasked := func(key string, before, after *string) {
		if reflect.DeepEqual(approvalSummaryValue(before), approvalSummaryValue(after)) {
			return
		}
		result.Fields[key] = PoolApprovalValueChange{Before: maskApprovalValue(before), After: maskApprovalValue(after)}
	}
	if u := payload.AccountUpdate; u != nil {
		if u.Name != "" {
			add("name", account.Name, u.Name)
		}
		if u.Notes != nil {
			add("notes", account.Notes, u.Notes)
		}
		if u.Type != "" {
			add("type", account.Type, u.Type)
		}
		if len(u.Credentials) > 0 {
			result.CredentialKeys = changedApprovalCredentialKeys(account.Credentials, u.Credentials)
		}
		if u.Extra != nil {
			result.ExtraKeys = replacedApprovalMapKeys(account.Extra, u.Extra)
		}
		if u.ProxyID != nil {
			add("proxy_id", account.ProxyID, u.ProxyID)
		}
		if u.Concurrency != nil {
			add("concurrency", account.Concurrency, *u.Concurrency)
		}
		if u.Priority != nil {
			add("priority", account.Priority, *u.Priority)
		}
		if u.RateMultiplier != nil {
			add("rate_multiplier", account.RateMultiplier, *u.RateMultiplier)
		}
		if u.LoadFactor != nil {
			add("load_factor", account.LoadFactor, u.LoadFactor)
		}
		if u.Status != "" {
			add("status", account.Status, u.Status)
		}
		if u.GroupIDs != nil {
			add("group_ids", account.GroupIDs, *u.GroupIDs)
		}
		if u.ExpiresAt != nil {
			var currentExpiresAt any
			if account.ExpiresAt != nil {
				currentExpiresAt = account.ExpiresAt.Unix()
			}
			add("expires_at", currentExpiresAt, *u.ExpiresAt)
		}
		if u.AutoPauseOnExpired != nil {
			add("auto_pause_on_expired", account.AutoPauseOnExpired, *u.AutoPauseOnExpired)
		}
	}
	if u := payload.PoolUpdate; u != nil {
		if u.ProviderIdentity != nil {
			addMasked("provider_identity", pool.ProviderIdentity, u.ProviderIdentity)
		}
		if u.ContributorUserID != nil {
			add("contributor_user_id", pool.ContributorUserID, *u.ContributorUserID)
		}
		if u.CreatedByUserID != nil {
			add("created_by_user_id", pool.CreatedByUserID, *u.CreatedByUserID)
		}
		if u.CostSharingEnabled != nil {
			add("cost_sharing_enabled", pool.CostSharingEnabled, *u.CostSharingEnabled)
		}
	}
	if u := payload.CostUpdate; u != nil && costBefore != nil {
		add("cost_entry_id", costBefore.ID, u.CostID)
		add("cost_payer_user_id", costBefore.PayerUserID, u.Cost.PayerUserID)
		add("cost_purchase_source_id", costBefore.PurchaseSourceID, u.Cost.PurchaseSourceID)
		add("cost_entry_type", costBefore.EntryType, u.Cost.EntryType)
		add("cost_original_amount", costBefore.OriginalAmount, u.Cost.OriginalAmount)
		add("cost_currency", costBefore.Currency, u.Cost.Currency)
		add("cost_fx_rate", costBefore.FXRate, u.Cost.FXRate)
		add("cost_service_start", costBefore.ServiceStart, u.Cost.ServiceStart)
		add("cost_service_end", costBefore.ServiceEnd, u.Cost.ServiceEnd)
		add("cost_warranty_end", costBefore.WarrantyEnd, u.Cost.WarrantyEnd)
		add("cost_paid_at", costBefore.PaidAt, u.Cost.PaidAt)
		add("cost_order_no", costBefore.OrderNo, u.Cost.OrderNo)
		add("cost_purchase_url", costBefore.PurchaseURL, u.Cost.PurchaseURL)
		add("cost_note", costBefore.Note, u.Cost.Note)
		add("cost_expected_token_count", costBefore.ExpectedTokenCount, u.Cost.ExpectedTokenCount)
	}
	if u := payload.DeleteOptions; u != nil {
		add("delete_account", false, true)
	}
	if len(payload.ExtraMerge) > 0 {
		result.ExtraKeys = mergeSortedKeys(result.ExtraKeys, changedApprovalMapKeys(account.Extra, payload.ExtraMerge))
	}
	if len(result.Fields) == 0 {
		result.Fields = nil
	}
	return result
}

func buildPoolApprovalBusinessSummary(action string, account *Account, changes PoolApprovalChangeSummary, deleteImpact *PoolAccountDeleteImpact) PoolApprovalBusinessSummary {
	result := PoolApprovalBusinessSummary{
		Action:   action,
		Object:   PoolApprovalBusinessObject{Type: "account", ID: account.ID, Name: account.Name},
		HighRisk: action == PoolApprovalDeleteAccount || action == PoolApprovalViewCredential || len(changes.CredentialKeys) > 0,
	}
	if action == PoolApprovalDeleteAccount {
		result.Scope = []string{"credentials", "scheduling", "cost_settlement", "linked_data"}
		result.Groups = []PoolApprovalBusinessGroup{{Key: "linked_data", Items: []PoolApprovalBusinessChange{{
			Key: "delete_account", Before: false, After: true, Impact: "permanent_cleanup",
		}}}}
		if deleteImpact != nil {
			result.Impacts = []PoolApprovalBusinessImpact{
				{Key: "accounts", Count: deleteImpact.Accounts},
				{Key: "credential_keys", Count: deleteImpact.CredentialKeys},
				{Key: "scheduling_records", Count: deleteImpact.SchedulingRecords},
				{Key: "cost_entries", Count: deleteImpact.CostEntries},
				{Key: "settlements", Count: deleteImpact.Settlements},
				{Key: "settlement_account_costs", Count: deleteImpact.SettlementAccountCosts},
				{Key: "settlement_account_lines", Count: deleteImpact.SettlementAccountLines},
				{Key: "mixed_settlements", Count: deleteImpact.MixedSettlements},
				{Key: "empty_settlements", Count: deleteImpact.EmptySettlements},
				{Key: "purchase_sources", Count: deleteImpact.PurchaseSources},
				{Key: "group_links", Count: deleteImpact.GroupLinks},
				{Key: "lifecycle_events", Count: deleteImpact.LifecycleEvents},
				{Key: "retained_usage_records", Count: deleteImpact.UsageRecords},
			}
		}
		return result
	}
	if action == PoolApprovalViewCredential {
		result.Scope = []string{"credentials"}
		result.Groups = []PoolApprovalBusinessGroup{{Key: "credentials", Items: []PoolApprovalBusinessChange{{
			Key: "credentials", Sensitive: true, Impact: "credential_access",
		}}}}
		return result
	}

	grouped := make(map[string][]PoolApprovalBusinessChange)
	keys := make([]string, 0, len(changes.Fields))
	for key := range changes.Fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		change := changes.Fields[key]
		group, impact := poolApprovalBusinessGroup(key)
		grouped[group] = append(grouped[group], PoolApprovalBusinessChange{
			Key: key, Before: change.Before, After: change.After,
			Sensitive: key == "provider_identity", Impact: impact,
		})
	}
	for _, key := range changes.CredentialKeys {
		grouped["credentials"] = append(grouped["credentials"], PoolApprovalBusinessChange{
			Key: key, After: "updated", Sensitive: true, Impact: "credential_replacement",
		})
	}
	for _, key := range changes.ExtraKeys {
		grouped["linked_data"] = append(grouped["linked_data"], PoolApprovalBusinessChange{
			Key: key, After: "updated", Impact: "linked_data_changed",
		})
	}
	for _, group := range []string{"identity", "visibility", "scheduling", "capacity", "cost_settlement", "linked_data", "credentials"} {
		items := grouped[group]
		if len(items) == 0 {
			continue
		}
		result.Scope = append(result.Scope, group)
		result.Groups = append(result.Groups, PoolApprovalBusinessGroup{Key: group, Items: items})
	}
	return result
}

func proxyApprovalRevision(proxy *Proxy, accounts []ProxyAccountSummary) (string, error) {
	if proxy == nil {
		return "", ErrProxyNotFound
	}
	ids := make([]int64, 0, len(accounts))
	for i := range accounts {
		ids = append(ids, accounts[i].ID)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return approvalDigest(proxy, ids), nil
}

func proxyExportApprovalRevision(proxies []Proxy) (string, error) {
	sort.Slice(proxies, func(i, j int) bool { return proxies[i].ID < proxies[j].ID })
	return approvalDigest(proxies), nil
}

func approvalDigest(values ...any) string {
	raw, _ := json.Marshal(values)
	digest := sha256.Sum256(raw)
	return hex.EncodeToString(digest[:])
}

func buildProxyApprovalSummary(action string, proxy *Proxy, update *UpdateProxyInput, accountCount int64) PoolApprovalChangeSummary {
	business := PoolApprovalBusinessSummary{
		Action:   action,
		Object:   PoolApprovalBusinessObject{Type: "proxy", ID: proxy.ID, Name: proxy.Name},
		HighRisk: true,
		Impacts:  []PoolApprovalBusinessImpact{{Key: "bound_accounts", Count: accountCount}},
	}
	if action == PoolApprovalViewProxyCredential {
		business.Scope = []string{"connection", "credentials"}
		business.Groups = []PoolApprovalBusinessGroup{
			{Key: "connection", Items: []PoolApprovalBusinessChange{{Key: "proxy_endpoint", Sensitive: true, Impact: "connection_access"}}},
			{Key: "credentials", Items: []PoolApprovalBusinessChange{{Key: "proxy_credentials", Sensitive: true, Impact: "credential_access"}}},
		}
		return PoolApprovalChangeSummary{Business: business}
	}

	groups := []PoolApprovalBusinessGroup{}
	connection := []PoolApprovalBusinessChange{}
	runtime := []PoolApprovalBusinessChange{}
	credentials := []PoolApprovalBusinessChange{}
	add := func(items *[]PoolApprovalBusinessChange, key string, before, after any, impact string) {
		if !reflect.DeepEqual(before, after) {
			*items = append(*items, PoolApprovalBusinessChange{Key: key, Before: before, After: after, Impact: impact})
		}
	}
	add(&connection, "name", proxy.Name, update.Name, "identity_changed")
	add(&connection, "protocol", proxy.Protocol, update.Protocol, "connection_changed")
	beforeEndpoint := net.JoinHostPort(proxy.Host, fmt.Sprint(proxy.Port))
	afterEndpoint := net.JoinHostPort(update.Host, fmt.Sprint(update.Port))
	if beforeEndpoint != afterEndpoint {
		connection = append(connection, PoolApprovalBusinessChange{Key: "proxy_endpoint", Before: "configured", After: afterEndpoint, Impact: "connection_changed"})
	}
	add(&credentials, "username", proxy.Username != "", update.Username != "", "credential_replacement")
	if proxy.Password != update.Password {
		credentials = append(credentials, PoolApprovalBusinessChange{Key: "password", Before: proxy.Password != "", After: update.Password != "", Sensitive: true, Impact: "credential_replacement"})
	}
	add(&runtime, "status", proxy.Status, update.Status, "scheduling_changed")
	add(&runtime, "expires_at", proxy.ExpiresAt, update.ExpiresAt, "scheduling_changed")
	add(&runtime, "fallback_mode", proxy.FallbackMode, update.FallbackMode, "scheduling_changed")
	add(&runtime, "backup_proxy_id", proxy.BackupProxyID, update.BackupProxyID, "scheduling_changed")
	add(&runtime, "expiry_warn_days", proxy.ExpiryWarnDays, update.ExpiryWarnDays, "scheduling_changed")
	for _, group := range []struct {
		key   string
		items []PoolApprovalBusinessChange
	}{{"connection", connection}, {"credentials", credentials}, {"runtime", runtime}} {
		if len(group.items) > 0 {
			business.Scope = append(business.Scope, group.key)
			groups = append(groups, PoolApprovalBusinessGroup{Key: group.key, Items: group.items})
		}
	}
	business.Groups = groups
	return PoolApprovalChangeSummary{Business: business}
}

func buildProxyExportApprovalSummary(proxies []Proxy) PoolApprovalChangeSummary {
	return PoolApprovalChangeSummary{Business: PoolApprovalBusinessSummary{
		Action: PoolApprovalExportProxyCredentials,
		Object: PoolApprovalBusinessObject{Type: "proxy_export", Name: "代理连接信息导出"},
		Scope:  []string{"connection", "credentials"}, HighRisk: true,
		Groups:  []PoolApprovalBusinessGroup{{Key: "credentials", Items: []PoolApprovalBusinessChange{{Key: "proxy_export", After: len(proxies), Sensitive: true, Impact: "credential_export"}}}},
		Impacts: []PoolApprovalBusinessImpact{{Key: "proxies", Count: int64(len(proxies))}},
	}}
}

func buildMihomoApprovalSummary(resourceKey string, update *MihomoApprovalUpdate) PoolApprovalChangeSummary {
	objectName := "Mihomo 工作台"
	groups := make([]PoolApprovalBusinessGroup, 0, 2)
	impacts := make([]PoolApprovalBusinessImpact, 0, 3)
	change := func(key string, after any) PoolApprovalBusinessChange {
		return PoolApprovalBusinessChange{Key: key, After: after, Impact: "mihomo_configuration_changed"}
	}
	addImpact := func(key string, count int64) {
		if count > 0 {
			impacts = append(impacts, PoolApprovalBusinessImpact{Key: key, Count: count})
		}
	}

	switch update.Kind {
	case "subscription", MihomoApprovalSubscriptionCreate, MihomoApprovalSubscriptionUpdate, MihomoApprovalSubscriptionDelete, MihomoApprovalSubscriptionRefresh:
		if update.SubscriptionName != "" {
			objectName = update.SubscriptionName
		}
		items := []PoolApprovalBusinessChange{change("subscription_operation", update.Kind)}
		if update.SubscriptionName != "" {
			items = append(items, change("subscription_name", update.SubscriptionName))
		}
		if update.SubscriptionHost != "" {
			items = append(items, change("subscription_host", update.SubscriptionHost))
		}
		if update.Enabled != nil {
			items = append(items, change("enabled", *update.Enabled))
		}
		if update.RefreshIntervalMinutes > 0 {
			items = append(items, change("refresh_interval_minutes", update.RefreshIntervalMinutes))
		}
		items = append(items,
			change("node_count", mihomoApprovalCount(update.NodeCount, update.NodeIDs)),
			change("route_count", update.RouteCount),
		)
		groups = append(groups, PoolApprovalBusinessGroup{Key: "subscription", Items: items})
		addImpact("nodes", mihomoApprovalCount(update.NodeCount, update.NodeIDs))
		addImpact("routes", update.RouteCount)

	case MihomoApprovalRouteCreate, MihomoApprovalRouteUpdate, MihomoApprovalRouteDelete:
		if update.RouteName != "" {
			objectName = update.RouteName
		}
		items := []PoolApprovalBusinessChange{
			change("route_operation", update.Kind),
			change("route_name", update.RouteName),
			change("route_kind", update.RouteKind),
		}
		if update.ListenerPort > 0 {
			items = append(items, change("listener_port", update.ListenerPort))
		}
		if update.ProxyID > 0 {
			items = append(items, change("proxy_id", update.ProxyID))
		}
		if update.Enabled != nil {
			items = append(items, change("enabled", *update.Enabled))
		}
		items = append(items,
			change("subscription_count", len(update.SubscriptionIDs)),
			change("node_count", mihomoApprovalCount(update.NodeCount, update.NodeIDs)),
			change("bound_account_count", update.AccountCount),
		)
		groups = append(groups, PoolApprovalBusinessGroup{Key: "route", Items: items})
		addImpact("nodes", mihomoApprovalCount(update.NodeCount, update.NodeIDs))
		addImpact("bound_accounts", update.AccountCount)

	case MihomoApprovalNodeAction:
		objectName = "Mihomo 节点批量操作"
		items := []PoolApprovalBusinessChange{
			change("node_action", update.NodeAction),
			change("node_count", mihomoApprovalCount(update.NodeCount, update.NodeIDs)),
		}
		if len(update.NodeIDs) > 0 {
			items = append(items, change("node_ids", update.NodeIDs))
		}
		if update.RouteCount > 0 {
			items = append(items, change("route_count", update.RouteCount))
		}
		groups = append(groups, PoolApprovalBusinessGroup{Key: "nodes", Items: items})
		addImpact("nodes", mihomoApprovalCount(update.NodeCount, update.NodeIDs))
		addImpact("routes", update.RouteCount)

	case MihomoApprovalLegacyImport:
		objectName = "Mihomo 旧配置导入"
		items := []PoolApprovalBusinessChange{
			change("import_provider", update.ImportProviderName),
			change("subscription_name", update.SubscriptionName),
			change("subscription_host", update.SubscriptionHost),
			change("route_count", len(update.ImportRoutes)),
		}
		groups = append(groups, PoolApprovalBusinessGroup{Key: "subscription", Items: items})
		var nodes, accounts int64
		for index, route := range update.ImportRoutes {
			nodeCount := mihomoApprovalCount(route.NodeCount, route.NodeIDs)
			if nodeCount == 0 {
				nodeCount = int64(len(route.NodeNames))
			}
			groups = append(groups, PoolApprovalBusinessGroup{
				Key: fmt.Sprintf("route_%d", index+1),
				Items: []PoolApprovalBusinessChange{
					change("route_name", route.Name),
					change("route_kind", route.Kind),
					change("listener_port", route.ListenerPort),
					change("proxy_id", route.ProxyID),
					change("node_count", nodeCount),
					change("bound_account_count", route.AccountCount),
				},
			})
			nodes += nodeCount
			accounts += route.AccountCount
		}
		addImpact("routes", int64(len(update.ImportRoutes)))
		addImpact("nodes", nodes)
		addImpact("bound_accounts", accounts)

	case "mode":
		groups = append(groups, PoolApprovalBusinessGroup{Key: "mihomo", Items: []PoolApprovalBusinessChange{change("mode", update.Selection)}})
	case "refresh":
		groups = append(groups, PoolApprovalBusinessGroup{Key: "mihomo", Items: []PoolApprovalBusinessChange{change("refresh", "provider_nodes_refreshed")}})
	default:
		groups = append(groups, PoolApprovalBusinessGroup{Key: "mihomo", Items: []PoolApprovalBusinessChange{change("mihomo_operation", update.Kind)}})
	}
	if objectName == "Mihomo 工作台" && resourceKey != "" {
		objectName = resourceKey
	}
	return PoolApprovalChangeSummary{Business: PoolApprovalBusinessSummary{
		Action: PoolApprovalUpdateMihomo,
		Object: PoolApprovalBusinessObject{Type: "mihomo", Name: objectName},
		Scope:  []string{"mihomo"}, HighRisk: true,
		Groups: groups, Impacts: impacts,
	}}
}

func mihomoApprovalCount(explicit int64, ids []int64) int64 {
	if explicit > 0 {
		return explicit
	}
	return int64(len(ids))
}

func poolApprovalBusinessGroup(field string) (string, string) {
	switch field {
	case "group_ids":
		return "visibility", "visibility_changed"
	case "status", "proxy_id", "concurrency", "priority", "rate_multiplier", "load_factor", "expires_at", "auto_pause_on_expired":
		return "scheduling", "scheduling_changed"
	case "cost_expected_token_count":
		return "capacity", "capacity_changed"
	case "cost_entry_id", "cost_payer_user_id", "cost_purchase_source_id", "cost_entry_type", "cost_original_amount", "cost_currency", "cost_fx_rate", "cost_service_start", "cost_service_end", "cost_warranty_end", "cost_paid_at", "cost_order_no", "cost_purchase_url", "cost_note":
		return "cost_settlement", "settlement_changed"
	case "delete_account":
		return "linked_data", "permanent_cleanup"
	default:
		return "identity", "identity_changed"
	}
}

func approvalSummaryValue(value any) any {
	v := reflect.ValueOf(value)
	for v.IsValid() && v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}
	if !v.IsValid() {
		return nil
	}
	return v.Interface()
}

func changedApprovalCredentialKeys(current, requested map[string]any) []string {
	merged := MergePreservingSensitiveCreds(current, requested)
	keys := make(map[string]any, len(current)+len(merged))
	for key := range current {
		keys[key] = nil
	}
	for key := range merged {
		keys[key] = nil
	}
	changed := make(map[string]any)
	for key := range keys {
		before, beforeOK := current[key]
		after, afterOK := merged[key]
		if beforeOK != afterOK || !reflect.DeepEqual(before, after) {
			changed[key] = nil
		}
	}
	return sortedMapKeys(changed)
}

func changedApprovalMapKeys(current, requested map[string]any) []string {
	changed := make(map[string]any)
	for key, after := range requested {
		before, exists := current[key]
		if !exists || !reflect.DeepEqual(before, after) {
			changed[key] = nil
		}
	}
	return sortedMapKeys(changed)
}

func replacedApprovalMapKeys(current, requested map[string]any) []string {
	current = approvalManagedExtra(current)
	requested = approvalManagedExtra(requested)
	keys := make(map[string]any, len(current)+len(requested))
	for key := range current {
		keys[key] = nil
	}
	for key := range requested {
		keys[key] = nil
	}
	changed := make(map[string]any)
	for key := range keys {
		before, beforeOK := current[key]
		after, afterOK := requested[key]
		if beforeOK != afterOK || !reflect.DeepEqual(before, after) {
			changed[key] = nil
		}
	}
	return sortedMapKeys(changed)
}

func sortedMapKeys(values map[string]any) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func mergeSortedKeys(existing, values []string) []string {
	keys := make(map[string]any, len(existing)+len(values))
	for _, key := range existing {
		keys[key] = nil
	}
	for _, key := range values {
		keys[key] = nil
	}
	return sortedMapKeys(keys)
}

func maskApprovalValue(value *string) string {
	if value == nil || strings.TrimSpace(*value) == "" {
		return ""
	}
	v := []rune(strings.TrimSpace(*value))
	if len(v) <= 4 {
		return "****"
	}
	return string(v[:2]) + "****" + string(v[len(v)-2:])
}

func optionalTrimmedString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func validatePoolApprovalDecisionActor(item *PoolApproval, actorID int64, canSelfDecide bool) error {
	if item == nil || actorID <= 0 {
		return infraerrors.BadRequest("INVALID_APPROVAL_DECISION", "approval and actor are required")
	}
	if actorID == item.RequestedByUserID && !canSelfDecide {
		return infraerrors.Forbidden("APPROVAL_SELF_DECISION_FORBIDDEN", "requester cannot approve or reject their own request")
	}
	return nil
}
