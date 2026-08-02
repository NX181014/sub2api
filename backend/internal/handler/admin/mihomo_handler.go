package admin

import (
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type MihomoHandler struct {
	mihomo *service.MihomoService
	pool   *service.PoolService
}

func NewMihomoHandler(mihomo *service.MihomoService, pool *service.PoolService) *MihomoHandler {
	return &MihomoHandler{mihomo: mihomo, pool: pool}
}

func (h *MihomoHandler) Status(c *gin.Context) {
	item, err := h.mihomo.Status(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

func (h *MihomoHandler) Workbench(c *gin.Context) {
	item, err := h.mihomo.Workbench(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

func (h *MihomoHandler) LegacyImportPreview(c *gin.Context) {
	item, err := h.mihomo.LegacyImportPreview(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

func (h *MihomoHandler) ImportLegacy(c *gin.Context) {
	var req mihomoReasonRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	actorID, ok := poolActorID(c)
	if !ok {
		return
	}
	update, resourceKey, err := h.mihomo.PrepareLegacyImportApproval(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	h.createApproval(c, actorID, resourceKey, strings.TrimSpace(req.Reason), update)
}

type mihomoManagedSubscriptionRequest struct {
	Name                   string `json:"name" binding:"required,max=100"`
	SubscriptionURL        string `json:"subscription_url" binding:"max=4096"`
	Enabled                bool   `json:"enabled"`
	RefreshIntervalMinutes int    `json:"refresh_interval_minutes"`
	Reason                 string `json:"reason" binding:"max=1000"`
}

func (h *MihomoHandler) CreateManagedSubscription(c *gin.Context) {
	h.prepareManagedSubscriptionApproval(c, service.MihomoApprovalSubscriptionCreate, 0)
}

func (h *MihomoHandler) UpdateManagedSubscription(c *gin.Context) {
	id, ok := poolPathID(c)
	if !ok {
		return
	}
	h.prepareManagedSubscriptionApproval(c, service.MihomoApprovalSubscriptionUpdate, id)
}

func (h *MihomoHandler) prepareManagedSubscriptionApproval(c *gin.Context, operation string, id int64) {
	var req mihomoManagedSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	actorID, ok := poolActorID(c)
	if !ok {
		return
	}
	update, err := h.mihomo.PrepareManagedSubscriptionApproval(
		c.Request.Context(), operation, id, req.Name, req.SubscriptionURL, req.Enabled, req.RefreshIntervalMinutes,
	)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	h.createApproval(c, actorID, mihomoManagedResourceKey(update), strings.TrimSpace(req.Reason), update)
}

func (h *MihomoHandler) DeleteManagedSubscription(c *gin.Context) {
	id, ok := poolPathID(c)
	if !ok {
		return
	}
	var req mihomoReasonRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	actorID, ok := poolActorID(c)
	if !ok {
		return
	}
	update, err := h.mihomo.PrepareManagedSubscriptionApproval(
		c.Request.Context(), service.MihomoApprovalSubscriptionDelete, id, "", "", false, 0,
	)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	h.createApproval(c, actorID, mihomoManagedResourceKey(update), strings.TrimSpace(req.Reason), update)
}

func (h *MihomoHandler) RefreshManagedSubscription(c *gin.Context) {
	id, ok := poolPathID(c)
	if !ok {
		return
	}
	var req mihomoReasonRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	actorID, ok := poolActorID(c)
	if !ok {
		return
	}
	update, err := h.mihomo.PrepareManagedSubscriptionApproval(
		c.Request.Context(), service.MihomoApprovalSubscriptionRefresh, id, "", "", false, 0,
	)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	h.createApproval(c, actorID, mihomoManagedResourceKey(update), strings.TrimSpace(req.Reason), update)
}

type mihomoManagedRouteRequest struct {
	Name            string  `json:"name" binding:"required,max=100"`
	Kind            string  `json:"kind" binding:"required,max=20"`
	SubscriptionIDs []int64 `json:"subscription_ids"`
	NodeIDs         []int64 `json:"node_ids"`
	Enabled         bool    `json:"enabled"`
	Reason          string  `json:"reason" binding:"max=1000"`
}

func (h *MihomoHandler) CreateManagedRoute(c *gin.Context) {
	h.prepareManagedRouteApproval(c, service.MihomoApprovalRouteCreate, 0)
}

func (h *MihomoHandler) UpdateManagedRoute(c *gin.Context) {
	id, ok := poolPathID(c)
	if !ok {
		return
	}
	h.prepareManagedRouteApproval(c, service.MihomoApprovalRouteUpdate, id)
}

func (h *MihomoHandler) prepareManagedRouteApproval(c *gin.Context, operation string, id int64) {
	var req mihomoManagedRouteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	actorID, ok := poolActorID(c)
	if !ok {
		return
	}
	update, err := h.mihomo.PrepareManagedRouteApproval(
		c.Request.Context(), operation, id, req.Name, req.Kind, req.SubscriptionIDs, req.NodeIDs, req.Enabled,
	)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	h.createApproval(c, actorID, mihomoManagedResourceKey(update), strings.TrimSpace(req.Reason), update)
}

func (h *MihomoHandler) DeleteManagedRoute(c *gin.Context) {
	id, ok := poolPathID(c)
	if !ok {
		return
	}
	var req mihomoReasonRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	actorID, ok := poolActorID(c)
	if !ok {
		return
	}
	update, err := h.mihomo.PrepareManagedRouteApproval(
		c.Request.Context(), service.MihomoApprovalRouteDelete, id, "", "", nil, nil, false,
	)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	h.createApproval(c, actorID, mihomoManagedResourceKey(update), strings.TrimSpace(req.Reason), update)
}

func (h *MihomoHandler) TestManagedRoute(c *gin.Context) {
	id, ok := poolPathID(c)
	if !ok {
		return
	}
	if err := h.mihomo.TestManagedRoute(c.Request.Context(), id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": "Route tested"})
}

type mihomoManagedNodeActionRequest struct {
	Action  string  `json:"action" binding:"required,max=40"`
	NodeIDs []int64 `json:"node_ids" binding:"required,min=1,dive,gt=0"`
	Reason  string  `json:"reason" binding:"max=1000"`
}

func (h *MihomoHandler) ManagedNodeAction(c *gin.Context) {
	var req mihomoManagedNodeActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if req.Action == "test" {
		if err := h.mihomo.TestManagedNodes(c.Request.Context(), req.NodeIDs); err != nil {
			response.ErrorFrom(c, err)
			return
		}
		response.Success(c, gin.H{"message": "Nodes tested"})
		return
	}
	actorID, ok := poolActorID(c)
	if !ok {
		return
	}
	update, resourceKey, err := h.mihomo.PrepareManagedNodeApproval(c.Request.Context(), req.Action, req.NodeIDs)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	h.createApproval(c, actorID, resourceKey, strings.TrimSpace(req.Reason), update)
}

func mihomoManagedResourceKey(update *service.MihomoApprovalUpdate) string {
	if update == nil {
		return "mihomo"
	}
	target, id := "", int64(0)
	switch update.Kind {
	case service.MihomoApprovalSubscriptionCreate, service.MihomoApprovalSubscriptionUpdate, service.MihomoApprovalSubscriptionDelete, service.MihomoApprovalSubscriptionRefresh:
		target, id = "subscription", update.SubscriptionID
	case service.MihomoApprovalRouteCreate, service.MihomoApprovalRouteUpdate, service.MihomoApprovalRouteDelete:
		target, id = "route", update.RouteID
	case service.MihomoApprovalNodeAction:
		return "mihomo:nodes"
	default:
		return "mihomo:" + update.Kind
	}
	if id <= 0 {
		return "mihomo:" + target + ":new"
	}
	return "mihomo:" + target + ":" + strconv.FormatInt(id, 10)
}

type mihomoSubscriptionRequest struct {
	SubscriptionURL string `json:"subscription_url" binding:"required,max=4096"`
	Reason          string `json:"reason" binding:"max=1000"`
}

func (h *MihomoHandler) UpdateSubscription(c *gin.Context) {
	var req mihomoSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	actorID, ok := poolActorID(c)
	if !ok {
		return
	}
	update, _, err := h.mihomo.PrepareSubscriptionApproval(c.Request.Context(), req.SubscriptionURL)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	h.createApproval(c, actorID, "mihomo:subscription", strings.TrimSpace(req.Reason), update)
}

type mihomoReasonRequest struct {
	Reason string `json:"reason" binding:"max=1000"`
}

func (h *MihomoHandler) Refresh(c *gin.Context) {
	var req mihomoReasonRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	actorID, ok := poolActorID(c)
	if !ok {
		return
	}
	update, err := h.mihomo.PrepareRefreshApproval()
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	h.createApproval(c, actorID, "mihomo:refresh", strings.TrimSpace(req.Reason), update)
}

type mihomoModeRequest struct {
	Mode      string `json:"mode" binding:"required"`
	Selection string `json:"selection" binding:"required"`
	Reason    string `json:"reason" binding:"max=1000"`
}

func (h *MihomoHandler) UpdateMode(c *gin.Context) {
	var req mihomoModeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	actorID, ok := poolActorID(c)
	if !ok {
		return
	}
	update, err := h.mihomo.PrepareModeApproval(c.Request.Context(), req.Mode, req.Selection)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	h.createApproval(c, actorID, "mihomo:"+update.Mode, strings.TrimSpace(req.Reason), update)
}

func (h *MihomoHandler) createApproval(c *gin.Context, actorID int64, resourceKey, reason string, update *service.MihomoApprovalUpdate) {
	item, err := h.pool.CreateApproval(c.Request.Context(), service.CreatePoolApprovalInput{
		ActionType: service.PoolApprovalUpdateMihomo, ResourceKey: resourceKey,
		RequesterID: actorID, Reason: reason,
		Payload: service.PoolApprovalPayload{MihomoUpdate: update},
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	result := gin.H{"approval_required": !item.PrimaryBypass, "approval": item}
	if item.PrimaryBypass {
		response.Success(c, result)
		return
	}
	response.Accepted(c, result)
}
