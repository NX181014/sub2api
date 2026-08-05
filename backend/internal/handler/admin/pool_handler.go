package admin

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// PoolHandler manages shared-pool assets, costs, settlements, and recovery statistics.
type PoolHandler struct{ poolService *service.PoolService }

func NewPoolHandler(poolService *service.PoolService) *PoolHandler {
	return &PoolHandler{poolService: poolService}
}

// RequirePrimaryAdmin protects operations that can export credentials outside
// the one-time, single-account reveal flow.
func (h *PoolHandler) RequirePrimaryAdmin(c *gin.Context) {
	actorID, ok := poolActorID(c)
	if !ok {
		c.Abort()
		return
	}
	if h == nil || h.poolService == nil || !h.poolService.IsPrimaryAdmin(c.Request.Context(), actorID) {
		response.ErrorFrom(c, infraerrors.Forbidden(
			"PRIMARY_ADMIN_REQUIRED",
			"this credential export operation is limited to the primary administrator",
		))
		c.Abort()
		return
	}
	c.Next()
}

func poolActorID(c *gin.Context) (int64, bool) {
	if c.GetString("auth_method") == service.AuditAuthMethodAdminAPIKey {
		response.Forbidden(c, "A signed-in administrator session is required")
		return 0, false
	}
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		response.Unauthorized(c, "User not found in context")
		return 0, false
	}
	return subject.UserID, true
}

func optionalPoolActorID(c *gin.Context) (int64, bool) {
	if c.GetString("auth_method") == service.AuditAuthMethodAdminAPIKey {
		return 0, false
	}
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	return subject.UserID, ok && subject.UserID > 0
}

func optionalPositiveInt64(value int64) *int64 {
	if value <= 0 {
		return nil
	}
	return &value
}

func poolPathID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid ID")
		return 0, false
	}
	return id, true
}

func optionalPoolQueryID(c *gin.Context, key string) (*int64, bool) {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return nil, true
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid "+key)
		return nil, false
	}
	return &id, true
}

func parsePoolDate(value string) (time.Time, error) {
	loc, _ := time.LoadLocation(service.PoolTimezone)
	return time.ParseInLocation("2006-01-02", strings.TrimSpace(value), loc)
}

func parsePoolPaidAt(value *string, submittedAt time.Time) (time.Time, error) {
	if value == nil || strings.TrimSpace(*value) == "" {
		return submittedAt, nil
	}
	raw := strings.TrimSpace(*value)
	if date, err := parsePoolDate(raw); err == nil {
		localSubmittedAt := submittedAt.In(date.Location())
		return time.Date(date.Year(), date.Month(), date.Day(), localSubmittedAt.Hour(), localSubmittedAt.Minute(), localSubmittedAt.Second(), localSubmittedAt.Nanosecond(), date.Location()), nil
	}
	return time.Parse(time.RFC3339, raw)
}

func (h *PoolHandler) ListAccounts(c *gin.Context) {
	items, err := h.poolService.ListAccounts(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, items)
}

type updatePoolAccountRequest struct {
	ProviderIdentity   *string `json:"provider_identity"`
	ContributorUserID  *int64  `json:"contributor_user_id"`
	CreatedByUserID    *int64  `json:"created_by_user_id"`
	CostSharingEnabled *bool   `json:"cost_sharing_enabled"`
	Reason             string  `json:"reason"`
}

func (h *PoolHandler) UpdateAccount(c *gin.Context) {
	id, ok := poolPathID(c)
	if !ok {
		return
	}
	var req updatePoolAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	actorID, ok := poolActorID(c)
	if !ok {
		return
	}
	update := service.UpdatePoolAccountInput{
		ProviderIdentity: req.ProviderIdentity, ContributorUserID: req.ContributorUserID,
		CreatedByUserID: req.CreatedByUserID, CostSharingEnabled: req.CostSharingEnabled,
	}
	if h.poolService.IsPrimaryAdmin(c.Request.Context(), actorID) {
		item, err := h.poolService.UpdateAccount(c.Request.Context(), id, update)
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
		response.Success(c, item)
		return
	}
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		reason = "update shared pool account metadata"
	}
	item, err := h.poolService.CreateApproval(c.Request.Context(), service.CreatePoolApprovalInput{
		ActionType: service.PoolApprovalUpdateAccount, AccountID: id, Reason: reason,
		RequesterID: actorID, Payload: service.PoolApprovalPayload{PoolUpdate: &update},
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Accepted(c, gin.H{"approval_required": true, "approval": item})
}

type createPoolApprovalRequest struct {
	ActionType  string                      `json:"action_type"`
	RequestType string                      `json:"request_type"`
	AccountID   int64                       `json:"account_id"`
	ProxyID     *int64                      `json:"proxy_id"`
	ResourceKey string                      `json:"resource_key"`
	Reason      string                      `json:"reason"`
	Payload     service.PoolApprovalPayload `json:"payload"`
}

func (h *PoolHandler) CreateApproval(c *gin.Context) {
	actorID, ok := poolActorID(c)
	if !ok {
		return
	}
	var req createPoolApprovalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	action := req.ActionType
	if strings.TrimSpace(action) == "" {
		action = req.RequestType
	}
	item, err := h.poolService.CreateApproval(c.Request.Context(), service.CreatePoolApprovalInput{
		ActionType: action, AccountID: req.AccountID, ProxyID: req.ProxyID, ResourceKey: req.ResourceKey,
		Reason: req.Reason, RequesterID: actorID, Payload: req.Payload,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if item.PrimaryBypass {
		response.Success(c, item)
		return
	}
	response.Created(c, item)
}

func (h *PoolHandler) ListApprovals(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	filter := service.PoolApprovalFilter{Status: c.Query("status"), ActionType: c.Query("action_type"), Scope: c.Query("scope")}
	if filter.ActionType == "" {
		filter.ActionType = c.Query("request_type")
	}
	var ok bool
	if filter.AccountID, ok = optionalPoolQueryID(c, "account_id"); !ok {
		return
	}
	if filter.ProxyID, ok = optionalPoolQueryID(c, "proxy_id"); !ok {
		return
	}
	filter.ObjectType = strings.TrimSpace(c.Query("object_type"))
	if filter.RequestedByUserID, ok = optionalPoolQueryID(c, "requested_by_user_id"); !ok {
		return
	}
	if raw, exists := c.GetQuery("high_risk"); exists {
		value, err := strconv.ParseBool(raw)
		if err != nil {
			response.BadRequest(c, "Invalid high_risk")
			return
		}
		filter.HighRisk = &value
	}
	if strings.TrimSpace(filter.Scope) != "" {
		if filter.ActorID, ok = poolActorID(c); !ok {
			return
		}
	}
	items, total, err := h.poolService.ListApprovals(c.Request.Context(), filter, page, pageSize)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, items, total, page, pageSize)
}

func (h *PoolHandler) GetApproval(c *gin.Context) {
	id, ok := poolPathID(c)
	if !ok {
		return
	}
	item, err := h.poolService.GetApproval(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

type decidePoolApprovalRequest struct {
	Reason string `json:"reason"`
}

func (h *PoolHandler) ApproveApproval(c *gin.Context) {
	id, ok := poolPathID(c)
	if !ok {
		return
	}
	actorID, ok := poolActorID(c)
	if !ok {
		return
	}
	var req decidePoolApprovalRequest
	if c.Request.ContentLength != 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "Invalid request: "+err.Error())
			return
		}
	}
	item, err := h.poolService.ApproveApproval(c.Request.Context(), id, actorID, req.Reason)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

func (h *PoolHandler) RejectApproval(c *gin.Context) {
	id, ok := poolPathID(c)
	if !ok {
		return
	}
	actorID, ok := poolActorID(c)
	if !ok {
		return
	}
	var req decidePoolApprovalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	item, err := h.poolService.RejectApproval(c.Request.Context(), id, actorID, req.Reason)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

func (h *PoolHandler) RevealCredential(c *gin.Context) {
	if c.GetString("auth_method") == service.AuditAuthMethodAdminAPIKey {
		response.Forbidden(c, "A signed-in administrator session is required")
		return
	}
	id, ok := poolPathID(c)
	if !ok {
		return
	}
	actorID, ok := poolActorID(c)
	if !ok {
		return
	}
	item, err := h.poolService.RevealCredential(c.Request.Context(), id, actorID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

func (h *PoolHandler) RevealProxyCredential(c *gin.Context) {
	if c.GetString("auth_method") == service.AuditAuthMethodAdminAPIKey {
		response.Forbidden(c, "A signed-in administrator session is required")
		return
	}
	id, ok := poolPathID(c)
	if !ok {
		return
	}
	actorID, ok := poolActorID(c)
	if !ok {
		return
	}
	item, err := h.poolService.RevealProxyCredential(c.Request.Context(), id, actorID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"proxy": dto.ProxyFromServiceAdmin(&item.Proxy), "revealed_at": item.RevealedAt})
}

func (h *PoolHandler) RevealProxyExport(c *gin.Context) {
	if c.GetString("auth_method") == service.AuditAuthMethodAdminAPIKey {
		response.Forbidden(c, "A signed-in administrator session is required")
		return
	}
	id, ok := poolPathID(c)
	if !ok {
		return
	}
	actorID, ok := poolActorID(c)
	if !ok {
		return
	}
	item, err := h.poolService.RevealProxyExport(c.Request.Context(), id, actorID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	proxies := make([]dto.AdminProxy, 0, len(item.Proxies))
	for i := range item.Proxies {
		proxies = append(proxies, *dto.ProxyFromServiceAdmin(&item.Proxies[i]))
	}
	response.Success(c, gin.H{"proxies": proxies, "revealed_at": item.RevealedAt})
}

type createPoolAccountIntakeRequest struct {
	ProviderIdentity   string  `json:"provider_identity" binding:"required"`
	ContributorUserID  int64   `json:"contributor_user_id" binding:"required"`
	UploaderUserID     int64   `json:"uploader_user_id" binding:"required"`
	PurchaseSourceName string  `json:"purchase_source_name" binding:"required"`
	EntryType          string  `json:"entry_type" binding:"required"`
	OriginalAmount     string  `json:"original_amount" binding:"required"`
	Currency           string  `json:"currency" binding:"required"`
	FXRate             string  `json:"fx_rate" binding:"required"`
	CNYAmountMinor     int64   `json:"cny_amount_minor" binding:"required"`
	ServiceStart       string  `json:"service_start" binding:"required"`
	ServiceEnd         string  `json:"service_end" binding:"required"`
	WarrantyEnd        *string `json:"warranty_end"`
	PaidAt             *string `json:"paid_at"`
	OrderNo            *string `json:"order_no"`
	PurchaseURL        *string `json:"purchase_url"`
	Notes              *string `json:"notes"`
	ExpectedTokenCount int64   `json:"expected_token_count" binding:"required,gt=0"`
}

func (h *PoolHandler) CreateAccountIntake(c *gin.Context) {
	accountID, ok := poolPathID(c)
	if !ok {
		return
	}
	actorID, ok := poolActorID(c)
	if !ok {
		return
	}
	var req createPoolAccountIntakeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	serviceStart, err := parsePoolDate(req.ServiceStart)
	if err != nil {
		response.BadRequest(c, "service_start must be YYYY-MM-DD")
		return
	}
	serviceEnd, err := parsePoolDate(req.ServiceEnd)
	if err != nil {
		response.BadRequest(c, "service_end must be YYYY-MM-DD")
		return
	}
	paidAt, err := parsePoolPaidAt(req.PaidAt, time.Now().UTC())
	if err != nil {
		response.BadRequest(c, "paid_at must be RFC3339 or YYYY-MM-DD")
		return
	}
	var warrantyEnd *time.Time
	if req.WarrantyEnd != nil && strings.TrimSpace(*req.WarrantyEnd) != "" {
		parsed, parseErr := parsePoolDate(*req.WarrantyEnd)
		if parseErr != nil {
			response.BadRequest(c, "warranty_end must be YYYY-MM-DD")
			return
		}
		warrantyEnd = &parsed
	}
	payload := struct {
		AccountID int64                          `json:"account_id"`
		Request   createPoolAccountIntakeRequest `json:"request"`
	}{AccountID: accountID, Request: req}
	executeAdminIdempotentJSON(c, "admin.pool.account_intake", payload, service.DefaultWriteIdempotencyTTL(), func(ctx context.Context) (any, error) {
		return h.poolService.CreateAccountIntake(ctx, service.CreateAccountIntakeInput{
			AccountID: accountID, ProviderIdentity: req.ProviderIdentity,
			ContributorUserID: req.ContributorUserID, UploaderUserID: req.UploaderUserID,
			PurchaseSourceName: req.PurchaseSourceName,
			ActorUserID:        actorID,
			Cost: service.CreateAccountCostInput{
				EntryType: req.EntryType, OriginalAmount: req.OriginalAmount, Currency: req.Currency,
				FXRate: req.FXRate, CNYAmountMinor: req.CNYAmountMinor,
				ServiceStart: serviceStart, ServiceEnd: serviceEnd, WarrantyEnd: warrantyEnd,
				PaidAt: paidAt, OrderNo: req.OrderNo, PurchaseURL: req.PurchaseURL, Note: req.Notes,
				ExpectedTokenCount: &req.ExpectedTokenCount,
			},
		})
	})
}

func (h *PoolHandler) ListSources(c *gin.Context) {
	referencedOnly := false
	if raw := strings.TrimSpace(c.Query("referenced")); raw != "" {
		value, err := strconv.ParseBool(raw)
		if err != nil {
			response.BadRequest(c, "referenced must be true or false")
			return
		}
		referencedOnly = value
	}
	items, err := h.poolService.ListSources(c.Request.Context(), referencedOnly)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, items)
}

type createPoolSourceRequest struct {
	Name       string  `json:"name" binding:"required"`
	WebsiteURL *string `json:"website_url"`
	URL        *string `json:"url"`
	Notes      *string `json:"notes"`
}

func (h *PoolHandler) CreateSource(c *gin.Context) {
	var req createPoolSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	websiteURL := req.WebsiteURL
	if websiteURL == nil {
		websiteURL = req.URL
	}
	item, err := h.poolService.CreateSource(c.Request.Context(), service.CreatePurchaseSourceInput{
		Name: req.Name, WebsiteURL: websiteURL, Notes: req.Notes,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

func (h *PoolHandler) ListCosts(c *gin.Context) {
	accountID, ok := optionalPoolQueryID(c, "account_id")
	if !ok {
		return
	}
	items, err := h.poolService.ListCosts(c.Request.Context(), accountID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, items)
}

func (h *PoolHandler) ListCostEntries(c *gin.Context) {
	accountID, ok := optionalPoolQueryID(c, "account_id")
	if !ok {
		return
	}
	uploaderID, ok := optionalPoolQueryID(c, "uploader_user_id")
	if !ok {
		return
	}
	payerID, ok := optionalPoolQueryID(c, "payer_user_id")
	if !ok {
		return
	}
	sourceID, ok := optionalPoolQueryID(c, "purchase_source_id")
	if !ok {
		return
	}
	var paidFrom, paidTo *time.Time
	startRaw := strings.TrimSpace(c.Query("start"))
	if startRaw == "" {
		startRaw = strings.TrimSpace(c.Query("start_date"))
	}
	if raw := startRaw; raw != "" {
		parsed, err := parsePoolDate(raw)
		if err != nil {
			response.BadRequest(c, "start must be YYYY-MM-DD")
			return
		}
		paidFrom = &parsed
	}
	endRaw := strings.TrimSpace(c.Query("end"))
	if endRaw == "" {
		endRaw = strings.TrimSpace(c.Query("end_date"))
	}
	if raw := endRaw; raw != "" {
		parsed, err := parsePoolDate(raw)
		if err != nil {
			response.BadRequest(c, "end must be YYYY-MM-DD")
			return
		}
		parsed = parsed.AddDate(0, 0, 1)
		paidTo = &parsed
	}
	page, pageSize := response.ParsePagination(c)
	items, total, err := h.poolService.ListCostEntries(c.Request.Context(), service.AccountCostEntryFilter{
		Search: strings.TrimSpace(c.Query("search")), AccountID: accountID, UploaderUserID: uploaderID,
		PayerUserID: payerID, PurchaseSourceID: sourceID, EntryType: strings.TrimSpace(c.Query("entry_type")),
		PaidFrom: paidFrom, PaidTo: paidTo,
	}, page, pageSize)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, items, total, page, pageSize)
}

func (h *PoolHandler) ListCostSummaries(c *gin.Context) {
	filter, ok := poolCostSummaryFilter(c)
	if !ok {
		return
	}
	page, pageSize := response.ParsePagination(c)
	items, total, err := h.poolService.ListCostSummaries(c.Request.Context(), filter, page, pageSize)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, items, total, page, pageSize)
}

func (h *PoolHandler) ListCostUploaderSummaries(c *gin.Context) {
	filter, ok := poolCostSummaryFilter(c)
	if !ok {
		return
	}
	page, pageSize := response.ParsePagination(c)
	items, total, err := h.poolService.ListCostUploaderSummaries(c.Request.Context(), filter, page, pageSize)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, items, total, page, pageSize)
}

func poolCostSummaryFilter(c *gin.Context) (service.AccountCostSummaryFilter, bool) {
	uploaderID, ok := optionalPoolQueryID(c, "uploader_user_id")
	if !ok {
		return service.AccountCostSummaryFilter{}, false
	}
	payerID, ok := optionalPoolQueryID(c, "payer_user_id")
	if !ok {
		return service.AccountCostSummaryFilter{}, false
	}
	sourceID, ok := optionalPoolQueryID(c, "purchase_source_id")
	if !ok {
		return service.AccountCostSummaryFilter{}, false
	}
	var hasCost *bool
	if raw := strings.TrimSpace(c.Query("has_cost")); raw != "" {
		value, err := strconv.ParseBool(raw)
		if err != nil {
			response.BadRequest(c, "has_cost must be true or false")
			return service.AccountCostSummaryFilter{}, false
		}
		hasCost = &value
	}
	uploaderUnassigned := false
	if raw := strings.TrimSpace(c.Query("uploader_unassigned")); raw != "" {
		value, err := strconv.ParseBool(raw)
		if err != nil {
			response.BadRequest(c, "uploader_unassigned must be true or false")
			return service.AccountCostSummaryFilter{}, false
		}
		uploaderUnassigned = value
	}
	return service.AccountCostSummaryFilter{
		Search: strings.TrimSpace(c.Query("search")), UploaderUserID: uploaderID, UploaderUnassigned: uploaderUnassigned, PayerUserID: payerID,
		PurchaseSourceID: sourceID, AccountStatus: strings.TrimSpace(c.Query("account_status")),
		AvailabilityStatus: strings.TrimSpace(c.Query("availability_status")), LifecycleStatus: strings.TrimSpace(c.Query("lifecycle_status")),
		EntryType: strings.TrimSpace(c.Query("entry_type")), HasCost: hasCost,
	}, true
}

type createPoolCostRequest struct {
	AccountID          int64   `json:"account_id" binding:"required"`
	PayerUserID        int64   `json:"payer_user_id" binding:"required"`
	PurchaseSourceID   *int64  `json:"purchase_source_id"`
	EntryType          string  `json:"entry_type" binding:"required"`
	OriginalAmount     string  `json:"original_amount" binding:"required"`
	Currency           string  `json:"currency" binding:"required"`
	FXRate             string  `json:"fx_rate"`
	CNYAmountMinor     int64   `json:"cny_amount_minor"`
	ServiceStart       string  `json:"service_start" binding:"required"`
	ServiceEnd         string  `json:"service_end" binding:"required"`
	WarrantyEnd        *string `json:"warranty_end"`
	PaidAt             *string `json:"paid_at"`
	OrderNo            *string `json:"order_no"`
	PurchaseURL        *string `json:"purchase_url"`
	Note               *string `json:"notes"`
	SupersedesID       *int64  `json:"supersedes_id"`
	RelatedAccountID   *int64  `json:"related_account_id"`
	ExpectedTokenCount *int64  `json:"expected_token_count"`
	ProviderIdentity   *string `json:"provider_identity"`
	ContributorUserID  *int64  `json:"contributor_user_id"`
	UploaderUserID     *int64  `json:"uploader_user_id"`
	CostSharingEnabled *bool   `json:"cost_sharing_enabled"`
	ApprovalReason     string  `json:"approval_reason"`
}

type batchPoolCostCommonRequest struct {
	PayerUserID        int64   `json:"payer_user_id" binding:"required"`
	PurchaseSourceID   *int64  `json:"purchase_source_id"`
	EntryType          string  `json:"entry_type" binding:"required"`
	OriginalAmount     string  `json:"original_amount" binding:"required"`
	Currency           string  `json:"currency" binding:"required"`
	FXRate             string  `json:"fx_rate"`
	ServiceStart       string  `json:"service_start" binding:"required"`
	ServiceEnd         string  `json:"service_end" binding:"required"`
	WarrantyEnd        *string `json:"warranty_end"`
	PaidAt             *string `json:"paid_at"`
	OrderNo            *string `json:"order_no"`
	PurchaseURL        *string `json:"purchase_url"`
	Note               *string `json:"notes"`
	SupersedesID       *int64  `json:"supersedes_id"`
	RelatedAccountID   *int64  `json:"related_account_id"`
	ExpectedTokenCount *int64  `json:"expected_token_count"`
}

type batchPoolCostAccountRequest struct {
	AccountID          int64   `json:"account_id" binding:"required"`
	OriginalAmount     *string `json:"original_amount"`
	ExpectedTokenCount *int64  `json:"expected_token_count"`
}

type batchPoolCostRequest struct {
	AmountMode string                        `json:"amount_mode" binding:"required"`
	Common     batchPoolCostCommonRequest    `json:"common" binding:"required"`
	Accounts   []batchPoolCostAccountRequest `json:"accounts" binding:"required,min=1,max=500,dive"`
}

func (h *PoolHandler) CreateCost(c *gin.Context) {
	actorID, ok := poolActorID(c)
	if !ok {
		return
	}
	var req createPoolCostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	start, err := parsePoolDate(req.ServiceStart)
	if err != nil {
		response.BadRequest(c, "service_start must be YYYY-MM-DD")
		return
	}
	end, err := parsePoolDate(req.ServiceEnd)
	if err != nil {
		response.BadRequest(c, "service_end must be YYYY-MM-DD")
		return
	}
	paidAt, err := parsePoolPaidAt(req.PaidAt, time.Now().UTC())
	if err != nil {
		response.BadRequest(c, "paid_at must be RFC3339 or YYYY-MM-DD")
		return
	}
	if req.FXRate == "" {
		req.FXRate = "1"
	}
	var warrantyEnd *time.Time
	if req.WarrantyEnd != nil && strings.TrimSpace(*req.WarrantyEnd) != "" {
		parsed, parseErr := parsePoolDate(*req.WarrantyEnd)
		if parseErr != nil {
			response.BadRequest(c, "warranty_end must be YYYY-MM-DD")
			return
		}
		warrantyEnd = &parsed
	}
	costInput := service.CreateAccountCostInput{
		AccountID: req.AccountID, PayerUserID: req.PayerUserID, PurchaseSourceID: req.PurchaseSourceID,
		EntryType: req.EntryType, OriginalAmount: req.OriginalAmount, Currency: req.Currency,
		FXRate: req.FXRate, CNYAmountMinor: req.CNYAmountMinor, ServiceStart: start, ServiceEnd: end, WarrantyEnd: warrantyEnd,
		PaidAt: paidAt, OrderNo: req.OrderNo, PurchaseURL: req.PurchaseURL, Note: req.Note,
		SupersedesID: req.SupersedesID, RelatedAccountID: req.RelatedAccountID, CreatedByUserID: actorID,
		ExpectedTokenCount: req.ExpectedTokenCount,
		OperationKey:       strings.TrimSpace(c.GetHeader("Idempotency-Key")),
	}
	if req.SupersedesID != nil {
		poolUpdate := &service.UpdatePoolAccountInput{
			ProviderIdentity: req.ProviderIdentity, ContributorUserID: req.ContributorUserID,
			CreatedByUserID: req.UploaderUserID, CostSharingEnabled: req.CostSharingEnabled,
		}
		if req.ProviderIdentity == nil && req.ContributorUserID == nil && req.UploaderUserID == nil && req.CostSharingEnabled == nil {
			poolUpdate = nil
		}
		reason := strings.TrimSpace(req.ApprovalReason)
		if reason == "" {
			reason = "update shared pool account cost record"
		}
		executeAdminIdempotentJSON(c, "admin.pool.cost.update", req, service.DefaultWriteIdempotencyTTL(), func(ctx context.Context) (any, error) {
			approval, updateErr := h.poolService.RequestCostUpdate(ctx, *req.SupersedesID, req.AccountID, actorID, reason, costInput, poolUpdate)
			if updateErr != nil {
				return nil, updateErr
			}
			return gin.H{"approval_required": !approval.PrimaryBypass, "approval": approval}, nil
		})
		return
	}
	executeAdminIdempotentJSON(c, "admin.pool.cost.create", req, service.DefaultWriteIdempotencyTTL(), func(ctx context.Context) (any, error) {
		return h.poolService.CreateCost(ctx, costInput)
	})
}

func (h *PoolHandler) CreateCostsBatch(c *gin.Context) {
	actorID, ok := poolActorID(c)
	if !ok {
		return
	}
	var req batchPoolCostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	common, err := parseBatchPoolCostCommon(req.Common)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	accounts := make([]service.BatchAccountCostItemInput, len(req.Accounts))
	for i := range req.Accounts {
		accounts[i] = service.BatchAccountCostItemInput{
			AccountID: req.Accounts[i].AccountID, OriginalAmount: req.Accounts[i].OriginalAmount,
			ExpectedTokenCount: req.Accounts[i].ExpectedTokenCount,
		}
	}
	payload := req
	executeAdminIdempotentJSON(c, "admin.pool.cost.batch_create", payload, service.DefaultWriteIdempotencyTTL(), func(ctx context.Context) (any, error) {
		return h.poolService.CreateCostsBatch(ctx, service.BatchCreateAccountCostsInput{
			AmountMode: req.AmountMode, Common: common, Accounts: accounts,
			ExpectedTokenCount: req.Common.ExpectedTokenCount, CreatedByUserID: actorID,
			OperationKey: strings.TrimSpace(c.GetHeader("Idempotency-Key")),
		})
	})
}

func parseBatchPoolCostCommon(req batchPoolCostCommonRequest) (service.CreateAccountCostInput, error) {
	start, err := parsePoolDate(req.ServiceStart)
	if err != nil {
		return service.CreateAccountCostInput{}, fmt.Errorf("service_start must be YYYY-MM-DD")
	}
	end, err := parsePoolDate(req.ServiceEnd)
	if err != nil {
		return service.CreateAccountCostInput{}, fmt.Errorf("service_end must be YYYY-MM-DD")
	}
	paidAt, err := parsePoolPaidAt(req.PaidAt, time.Now().UTC())
	if err != nil {
		return service.CreateAccountCostInput{}, fmt.Errorf("paid_at must be RFC3339 or YYYY-MM-DD")
	}
	var warrantyEnd *time.Time
	if req.WarrantyEnd != nil && strings.TrimSpace(*req.WarrantyEnd) != "" {
		parsed, parseErr := parsePoolDate(*req.WarrantyEnd)
		if parseErr != nil {
			return service.CreateAccountCostInput{}, fmt.Errorf("warranty_end must be YYYY-MM-DD")
		}
		warrantyEnd = &parsed
	}
	if strings.TrimSpace(req.FXRate) == "" {
		req.FXRate = "1"
	}
	return service.CreateAccountCostInput{
		PayerUserID: req.PayerUserID, PurchaseSourceID: req.PurchaseSourceID, EntryType: req.EntryType,
		OriginalAmount: req.OriginalAmount, Currency: req.Currency, FXRate: req.FXRate,
		ServiceStart: start, ServiceEnd: end, WarrantyEnd: warrantyEnd, PaidAt: paidAt,
		OrderNo: req.OrderNo, PurchaseURL: req.PurchaseURL, Note: req.Note,
		SupersedesID: req.SupersedesID, RelatedAccountID: req.RelatedAccountID,
	}, nil
}

func (h *PoolHandler) ListLifecycle(c *gin.Context) {
	accountID, ok := optionalPoolQueryID(c, "account_id")
	if !ok {
		return
	}
	items, err := h.poolService.ListLifecycle(c.Request.Context(), accountID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, items)
}

type createPoolLifecycleRequest struct {
	AccountID            int64   `json:"account_id" binding:"required"`
	EventType            string  `json:"event_type" binding:"required"`
	EventAt              string  `json:"event_at" binding:"required"`
	Reason               *string `json:"reason"`
	ReplacementAccountID *int64  `json:"replacement_account_id"`
	TransferredCostMinor int64   `json:"transferred_cost_minor"`
	RefundAmountMinor    int64   `json:"refund_amount_minor"`
	PayerUserID          *int64  `json:"payer_user_id"`
}

func (h *PoolHandler) CreateLifecycle(c *gin.Context) {
	actorID, ok := poolActorID(c)
	if !ok {
		return
	}
	var req createPoolLifecycleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	eventAt, err := time.Parse(time.RFC3339, strings.TrimSpace(req.EventAt))
	if err != nil {
		response.BadRequest(c, "event_at must be RFC3339")
		return
	}
	aliases := map[string]string{"banned": "banned_confirmed", "restored": "recovered", "refunded": "refund"}
	if normalized, exists := aliases[req.EventType]; exists {
		req.EventType = normalized
	}
	item, err := h.poolService.CreateLifecycle(c.Request.Context(), service.CreateLifecycleEventInput{
		AccountID: req.AccountID, EventType: req.EventType, OccurredAt: eventAt, Reason: req.Reason,
		ReplacementAccountID: req.ReplacementAccountID, TransferredCostMinor: req.TransferredCostMinor,
		RefundAmountMinor: req.RefundAmountMinor, PayerUserID: req.PayerUserID, CreatedByUserID: actorID,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

func (h *PoolHandler) ListFXRates(c *gin.Context) {
	items, err := h.poolService.ListFXRates(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, items)
}

type createPoolFXRateRequest struct {
	BaseCurrency  string  `json:"base_currency"`
	QuoteCurrency string  `json:"quote_currency"`
	Currency      string  `json:"currency"`
	Rate          string  `json:"rate" binding:"required"`
	EffectiveFrom string  `json:"effective_from" binding:"required"`
	Source        *string `json:"source"`
}

func (h *PoolHandler) CreateFXRate(c *gin.Context) {
	actorID, ok := poolActorID(c)
	if !ok {
		return
	}
	var req createPoolFXRateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if req.BaseCurrency == "" {
		req.BaseCurrency = req.Currency
	}
	if req.BaseCurrency == "" {
		req.BaseCurrency = "USD"
	}
	if req.QuoteCurrency == "" {
		req.QuoteCurrency = "CNY"
	}
	effectiveFrom, err := time.Parse(time.RFC3339, strings.TrimSpace(req.EffectiveFrom))
	if err != nil {
		response.BadRequest(c, "effective_from must be RFC3339")
		return
	}
	item, err := h.poolService.CreateFXRate(c.Request.Context(), service.CreateFXRateInput{
		BaseCurrency: req.BaseCurrency, QuoteCurrency: req.QuoteCurrency, Rate: req.Rate,
		EffectiveFrom: effectiveFrom, Source: req.Source, CreatedByUserID: actorID,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

type poolSettlementPeriodRequest struct {
	PeriodType       string `json:"period_type" binding:"required"`
	StartDate        string `json:"start_date" binding:"required"`
	EndDate          string `json:"end_date"`
	AccountID        *int64 `json:"account_id"`
	UploaderUserID   *int64 `json:"uploader_user_id"`
	PayerUserID      *int64 `json:"payer_user_id"`
	PurchaseSourceID *int64 `json:"purchase_source_id"`
}

func (h *PoolHandler) CreateSettlementDraft(c *gin.Context) {
	actorID, ok := poolActorID(c)
	if !ok {
		return
	}
	var req poolSettlementPeriodRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	period, err := service.ResolveSettlementPeriod(req.PeriodType, req.StartDate, req.EndDate)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	period.Filter = service.SettlementFilterSnapshot{
		AccountID: req.AccountID, UploaderUserID: req.UploaderUserID,
		PayerUserID: req.PayerUserID, PurchaseSourceID: req.PurchaseSourceID,
	}
	item, err := h.poolService.RecalculateSettlement(c.Request.Context(), period, actorID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

func (h *PoolHandler) RecalculateSettlement(c *gin.Context) {
	id, ok := poolPathID(c)
	if !ok {
		return
	}
	actorID, ok := poolActorID(c)
	if !ok {
		return
	}
	existing, err := h.poolService.GetSettlement(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	period := service.SettlementPeriod{Type: existing.PeriodType, Start: existing.PeriodStart, End: existing.PeriodEnd, Timezone: existing.Timezone, Filter: existing.FilterSnapshot}
	item, err := h.poolService.RecalculateSettlement(c.Request.Context(), period, actorID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

func (h *PoolHandler) ListSettlements(c *gin.Context) {
	accountID, ok := optionalPoolQueryID(c, "account_id")
	if !ok {
		return
	}
	page, pageSize := response.ParsePagination(c)
	items, total, err := h.poolService.ListSettlements(c.Request.Context(), accountID, page, pageSize)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, items, total, page, pageSize)
}

func (h *PoolHandler) GetSettlement(c *gin.Context) {
	id, ok := poolPathID(c)
	if !ok {
		return
	}
	item, err := h.poolService.GetSettlement(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

func (h *PoolHandler) LockSettlement(c *gin.Context) {
	id, ok := poolPathID(c)
	if !ok {
		return
	}
	actorID, ok := poolActorID(c)
	if !ok {
		return
	}
	item, err := h.poolService.LockSettlement(c.Request.Context(), id, actorID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

func (h *PoolHandler) ConfirmSettlementLine(c *gin.Context) {
	id, ok := poolPathID(c)
	if !ok {
		return
	}
	actorID, ok := poolActorID(c)
	if !ok {
		return
	}
	userID := actorID
	if raw := strings.TrimSpace(c.Query("user_id")); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || parsed <= 0 {
			response.BadRequest(c, "invalid settlement member")
			return
		}
		userID = parsed
	}
	item, err := h.poolService.ConfirmSettlementLine(c.Request.Context(), id, userID, actorID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

func (h *PoolHandler) ListOwnPendingSettlements(c *gin.Context) {
	actorID, ok := poolActorID(c)
	if !ok {
		return
	}
	items, err := h.poolService.ListOwnPendingSettlements(c.Request.Context(), actorID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, items)
}

func (h *PoolHandler) ConfirmOwnSettlementLine(c *gin.Context) {
	id, ok := poolPathID(c)
	if !ok {
		return
	}
	actorID, ok := poolActorID(c)
	if !ok {
		return
	}
	if err := h.poolService.ConfirmOwnSettlementLine(c.Request.Context(), id, actorID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"confirmed": true})
}

func (h *PoolHandler) MarkSettlementPaid(c *gin.Context) {
	id, ok := poolPathID(c)
	if !ok {
		return
	}
	actorID, ok := poolActorID(c)
	if !ok {
		return
	}
	item, err := h.poolService.MarkSettlementPaid(c.Request.Context(), id, actorID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

type poolSettlementMemberPaidRequest struct {
	SettlementUserID int64 `json:"settlement_user_id" binding:"required"`
}

func (h *PoolHandler) MarkSettlementMemberPaid(c *gin.Context) {
	id, ok := poolPathID(c)
	if !ok {
		return
	}
	memberUserID, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	if err != nil || memberUserID <= 0 {
		response.BadRequest(c, "invalid settlement member")
		return
	}
	actorID, ok := poolActorID(c)
	if !ok {
		return
	}
	var req poolSettlementMemberPaidRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	item, err := h.poolService.MarkSettlementMemberPaid(c.Request.Context(), id, memberUserID, req.SettlementUserID, actorID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

func (h *PoolHandler) MarkSettlementTransferPaid(c *gin.Context) {
	id, ok := poolPathID(c)
	if !ok {
		return
	}
	transferID, err := strconv.ParseInt(c.Param("transfer_id"), 10, 64)
	if err != nil || transferID <= 0 {
		response.BadRequest(c, "invalid settlement transfer")
		return
	}
	actorID, ok := poolActorID(c)
	if !ok {
		return
	}
	item, err := h.poolService.MarkSettlementTransferPaid(c.Request.Context(), id, transferID, actorID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

func (h *PoolHandler) GetOverview(c *gin.Context) {
	accountID, ok := optionalPoolQueryID(c, "account_id")
	if !ok {
		return
	}
	loc, _ := time.LoadLocation(service.PoolTimezone)
	now := time.Now().In(loc)
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)
	end := start.AddDate(0, 1, 0)
	var err error
	if raw := strings.TrimSpace(c.Query("start_date")); raw != "" {
		start, err = parsePoolDate(raw)
		if err != nil {
			response.BadRequest(c, "start_date must be YYYY-MM-DD")
			return
		}
	}
	if raw := strings.TrimSpace(c.Query("end_date")); raw != "" {
		end, err = parsePoolDate(raw)
		if err != nil {
			response.BadRequest(c, "end_date must be YYYY-MM-DD")
			return
		}
	}
	item, err := h.poolService.GetRecovery(c.Request.Context(), start, end, accountID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}
