package admin

import (
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

type mihomoSubscriptionRequest struct {
	SubscriptionURL string `json:"subscription_url" binding:"required,max=4096"`
	Reason          string `json:"reason" binding:"required,max=1000"`
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
	Reason string `json:"reason" binding:"required,max=1000"`
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
	Reason    string `json:"reason" binding:"required,max=1000"`
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
		RequesterID: actorID, Reason: reason, RequirePeerReview: true,
		Payload: service.PoolApprovalPayload{MihomoUpdate: update},
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Accepted(c, gin.H{"approval_required": true, "approval": item})
}
