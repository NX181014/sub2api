// Package routes provides HTTP route registration and handlers.
package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// RegisterAdminRoutes 注册管理员路由
func RegisterAdminRoutes(
	v1 *gin.RouterGroup,
	h *handler.Handlers,
	adminAuth middleware.AdminAuthMiddleware,
	auditLog middleware.AuditLogMiddleware,
	stepUpAuth middleware.StepUpAuthMiddleware,
	settingService *service.SettingService,
	panelRateLimiter *middleware.PanelRateLimiter,
) {
	admin := v1.Group("/admin")
	admin.Use(gin.HandlerFunc(adminAuth))
	// 面板全局按用户限流（默认管理员豁免，可在系统设置中关闭豁免）
	admin.Use(panelRateLimiter.Global())
	// 审计中间件挂在认证之后：所有管理面变更类操作 + 敏感读取入审计日志
	admin.Use(gin.HandlerFunc(auditLog))
	admin.Use(middleware.AdminComplianceGuard(settingService))
	{
		// 部署与运营合规确认
		registerAdminComplianceRoutes(admin, h)

		// 仪表盘
		registerDashboardRoutes(admin, h)

		// 用户管理
		registerUserManagementRoutes(admin, h)

		// 分组管理
		registerGroupRoutes(admin, h)

		// 账号管理
		registerAccountRoutes(admin, h, stepUpAuth)

		// 共享号池资产、结算与回本
		registerPoolRoutes(admin, h, stepUpAuth)

		// 公告管理
		registerAnnouncementRoutes(admin, h)

		// OpenAI OAuth
		registerOpenAIOAuthRoutes(admin, h)

		// Gemini OAuth
		registerGeminiOAuthRoutes(admin, h)

		// Antigravity OAuth
		registerAntigravityOAuthRoutes(admin, h)

		// Grok OAuth
		registerGrokOAuthRoutes(admin, h)

		// 代理管理
		registerProxyRoutes(admin, h, stepUpAuth)
		registerMihomoRoutes(admin, h)

		// 卡密管理
		registerRedeemCodeRoutes(admin, h)

		// 优惠码管理
		registerPromoCodeRoutes(admin, h)

		// 系统设置
		registerSettingsRoutes(admin, h, stepUpAuth)

		// 数据管理
		registerDataManagementRoutes(admin, h, stepUpAuth)

		// 数据库备份恢复
		registerBackupRoutes(admin, h, stepUpAuth)

		// 运维监控（Ops）
		registerOpsRoutes(admin, h)

		// 系统管理
		registerSystemRoutes(admin, h, stepUpAuth)

		// 订阅管理
		registerSubscriptionRoutes(admin, h)

		// 使用记录管理
		registerUsageRoutes(admin, h)

		// 用户属性管理
		registerUserAttributeRoutes(admin, h)

		// 错误透传规则管理
		registerErrorPassthroughRoutes(admin, h)

		// TLS 指纹模板管理
		registerTLSFingerprintProfileRoutes(admin, h)

		// API Key 管理
		registerAdminAPIKeyRoutes(admin, h)

		// 定时测试计划
		registerScheduledTestRoutes(admin, h)

		// 渠道管理
		registerChannelRoutes(admin, h)

		// 渠道监控
		registerChannelMonitorRoutes(admin, h)

		// 风控中心
		registerContentModerationRoutes(admin, h)

		// 独立提示词输入审计
		registerPromptAuditRoutes(admin, h)

		// 邀请返利（专属用户管理）
		registerAffiliateRoutes(admin, h)

		// 操作审计日志
		registerAuditLogRoutes(admin, h, stepUpAuth)
	}
}

func registerMihomoRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	mihomo := admin.Group("/mihomo")
	{
		mihomo.GET("/status", h.Admin.Mihomo.Status)
		mihomo.GET("/workbench", h.Admin.Mihomo.Workbench)
		mihomo.GET("/import-preview", h.Admin.Mihomo.LegacyImportPreview)
		mihomo.POST("/import", h.Admin.Mihomo.ImportLegacy)
		mihomo.POST("/subscriptions", h.Admin.Mihomo.CreateManagedSubscription)
		mihomo.PUT("/subscriptions/:id", h.Admin.Mihomo.UpdateManagedSubscription)
		mihomo.DELETE("/subscriptions/:id", h.Admin.Mihomo.DeleteManagedSubscription)
		mihomo.POST("/subscriptions/:id/refresh", h.Admin.Mihomo.RefreshManagedSubscription)
		mihomo.POST("/routes", h.Admin.Mihomo.CreateManagedRoute)
		mihomo.PUT("/routes/:id", h.Admin.Mihomo.UpdateManagedRoute)
		mihomo.DELETE("/routes/:id", h.Admin.Mihomo.DeleteManagedRoute)
		mihomo.POST("/routes/:id/test", h.Admin.Mihomo.TestManagedRoute)
		mihomo.POST("/nodes/actions", h.Admin.Mihomo.ManagedNodeAction)
		mihomo.POST("/subscription", h.Admin.Mihomo.UpdateSubscription)
		mihomo.POST("/refresh", h.Admin.Mihomo.Refresh)
		mihomo.POST("/modes", h.Admin.Mihomo.UpdateMode)
	}
}

func registerPoolRoutes(admin *gin.RouterGroup, h *handler.Handlers, stepUpAuth middleware.StepUpAuthMiddleware) {
	pool := admin.Group("/pool")
	{
		pool.GET("/overview", h.Admin.Pool.GetOverview)
		pool.GET("/accounts", h.Admin.Pool.ListAccounts)
		pool.POST("/accounts/:id/intake", h.Admin.Pool.CreateAccountIntake)
		pool.PUT("/accounts/:id", h.Admin.Pool.UpdateAccount)
		pool.GET("/approvals", h.Admin.Pool.ListApprovals)
		pool.POST("/approvals", h.Admin.Pool.CreateApproval)
		pool.GET("/approvals/:id", h.Admin.Pool.GetApproval)
		pool.POST("/approvals/:id/approve", h.Admin.Pool.ApproveApproval)
		pool.POST("/approvals/:id/reject", h.Admin.Pool.RejectApproval)
		pool.POST("/approvals/:id/reveal", gin.HandlerFunc(stepUpAuth), h.Admin.Pool.RevealCredential)
		pool.POST("/approvals/:id/reveal-proxy", gin.HandlerFunc(stepUpAuth), h.Admin.Pool.RevealProxyCredential)
		pool.POST("/approvals/:id/reveal-proxy-export", gin.HandlerFunc(stepUpAuth), h.Admin.Pool.RevealProxyExport)
		pool.GET("/sources", h.Admin.Pool.ListSources)
		pool.POST("/sources", h.Admin.Pool.CreateSource)
		pool.GET("/costs", h.Admin.Pool.ListCosts)
		pool.POST("/costs", h.Admin.Pool.CreateCost)
		pool.GET("/cost-entries", h.Admin.Pool.ListCostEntries)
		pool.GET("/cost-summaries", h.Admin.Pool.ListCostSummaries)
		pool.GET("/cost-uploader-summaries", h.Admin.Pool.ListCostUploaderSummaries)
		pool.POST("/costs/batch", h.Admin.Pool.CreateCostsBatch)
		pool.GET("/lifecycle", h.Admin.Pool.ListLifecycle)
		pool.POST("/lifecycle", h.Admin.Pool.CreateLifecycle)
		pool.GET("/fx-rates", h.Admin.Pool.ListFXRates)
		pool.POST("/fx-rates", h.Admin.Pool.CreateFXRate)
		pool.GET("/settlements", h.Admin.Pool.ListSettlements)
		pool.POST("/settlements/draft", h.Admin.Pool.CreateSettlementDraft)
		pool.GET("/settlements/:id", h.Admin.Pool.GetSettlement)
		pool.POST("/settlements/:id/recalculate", h.Admin.Pool.RecalculateSettlement)
		pool.POST("/settlements/:id/lock", h.Admin.Pool.LockSettlement)
		pool.POST("/settlements/:id/confirm", h.Admin.Pool.ConfirmSettlementLine)
		pool.POST("/settlements/:id/paid", h.Admin.Pool.MarkSettlementPaid)
		pool.POST("/settlements/:id/transfers/:transfer_id/paid", h.Admin.Pool.MarkSettlementTransferPaid)
		pool.POST("/settlements/:id/members/:user_id/paid", h.Admin.Pool.MarkSettlementMemberPaid)
	}
}

func registerPromptAuditRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	promptAudit := admin.Group("/prompt-audit")
	{
		promptAudit.GET("/config", h.Admin.PromptAudit.GetConfig)
		promptAudit.PUT("/config", h.Admin.PromptAudit.UpdateConfig)
		promptAudit.POST("/endpoints/probe", h.Admin.PromptAudit.ProbeEndpoint)
		promptAudit.GET("/runtime", h.Admin.PromptAudit.GetRuntime)
		promptAudit.GET("/events", h.Admin.PromptAudit.ListEvents)
		promptAudit.GET("/events/:id", h.Admin.PromptAudit.GetEvent)
		promptAudit.DELETE("/events/:id", h.Admin.PromptAudit.DeleteEvent)
		promptAudit.POST("/events/batch-delete", h.Admin.PromptAudit.BatchDelete)
		promptAudit.POST("/events/delete-preview", h.Admin.PromptAudit.DeletePreview)
		promptAudit.POST("/events/delete-by-filter", h.Admin.PromptAudit.DeleteByFilter)
	}
}

func registerAuditLogRoutes(admin *gin.RouterGroup, h *handler.Handlers, _ middleware.StepUpAuthMiddleware) {
	auditLogs := admin.Group("/audit-logs")
	{
		auditLogs.GET("", h.Admin.AuditLog.List)
		auditLogs.GET("/:id", h.Admin.AuditLog.Get)
		// 清空需现场 TOTP 校验（在 handler 内强制），不复用 step-up sudo 窗口
		auditLogs.POST("/clear", h.Admin.AuditLog.Clear)
	}
}

func registerAdminComplianceRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	compliance := admin.Group("/compliance")
	{
		compliance.GET("", h.Admin.Compliance.GetStatus)
		compliance.POST("/accept", h.Admin.Compliance.Accept)
	}
}

func registerContentModerationRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	risk := admin.Group("/risk-control")
	{
		risk.GET("/config", h.Admin.ContentModeration.GetConfig)
		risk.PUT("/config", h.Admin.ContentModeration.UpdateConfig)
		risk.POST("/api-keys/test", h.Admin.ContentModeration.TestAPIKeys)
		risk.GET("/status", h.Admin.ContentModeration.GetStatus)
		risk.GET("/logs", h.Admin.ContentModeration.ListLogs)
		risk.POST("/users/:user_id/unban", h.Admin.ContentModeration.UnbanUser)
		risk.DELETE("/hashes", h.Admin.ContentModeration.DeleteFlaggedHash)
		risk.DELETE("/hashes/all", h.Admin.ContentModeration.ClearFlaggedHashes)
	}
}

func registerAdminAPIKeyRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	apiKeys := admin.Group("/api-keys")
	{
		apiKeys.PUT("/:id", h.Admin.APIKey.UpdateGroup)
	}
}

func registerOpsRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	ops := admin.Group("/ops")
	{
		// Realtime ops signals
		ops.GET("/concurrency", h.Admin.Ops.GetConcurrencyStats)
		ops.GET("/user-concurrency", h.Admin.Ops.GetUserConcurrencyStats)
		ops.GET("/account-availability", h.Admin.Ops.GetAccountAvailability)
		ops.GET("/realtime-traffic", h.Admin.Ops.GetRealtimeTrafficSummary)

		// Alerts (rules + events)
		ops.GET("/alert-rules", h.Admin.Ops.ListAlertRules)
		ops.POST("/alert-rules", h.Admin.Ops.CreateAlertRule)
		ops.PUT("/alert-rules/:id", h.Admin.Ops.UpdateAlertRule)
		ops.DELETE("/alert-rules/:id", h.Admin.Ops.DeleteAlertRule)
		ops.GET("/alert-events", h.Admin.Ops.ListAlertEvents)
		ops.GET("/alert-events/:id", h.Admin.Ops.GetAlertEvent)
		ops.PUT("/alert-events/:id/status", h.Admin.Ops.UpdateAlertEventStatus)
		ops.POST("/alert-silences", h.Admin.Ops.CreateAlertSilence)

		// Email notification config (DB-backed)
		ops.GET("/email-notification/config", h.Admin.Ops.GetEmailNotificationConfig)
		ops.PUT("/email-notification/config", h.Admin.Ops.UpdateEmailNotificationConfig)

		// Runtime settings (DB-backed)
		runtime := ops.Group("/runtime")
		{
			runtime.GET("/alert", h.Admin.Ops.GetAlertRuntimeSettings)
			runtime.PUT("/alert", h.Admin.Ops.UpdateAlertRuntimeSettings)
			runtime.GET("/logging", h.Admin.Ops.GetRuntimeLogConfig)
			runtime.PUT("/logging", h.Admin.Ops.UpdateRuntimeLogConfig)
			runtime.POST("/logging/reset", h.Admin.Ops.ResetRuntimeLogConfig)
		}

		// Advanced settings (DB-backed)
		ops.GET("/advanced-settings", h.Admin.Ops.GetAdvancedSettings)
		ops.PUT("/advanced-settings", h.Admin.Ops.UpdateAdvancedSettings)

		// Settings group (DB-backed)
		settings := ops.Group("/settings")
		{
			settings.GET("/metric-thresholds", h.Admin.Ops.GetMetricThresholds)
			settings.PUT("/metric-thresholds", h.Admin.Ops.UpdateMetricThresholds)
		}

		// WebSocket realtime (QPS/TPS)
		ws := ops.Group("/ws")
		{
			ws.GET("/qps", h.Admin.Ops.QPSWSHandler)
		}

		// Error logs (legacy)
		ops.GET("/errors", h.Admin.Ops.GetErrorLogs)
		ops.GET("/errors/:id", h.Admin.Ops.GetErrorLogByID)
		ops.PUT("/errors/:id/resolve", h.Admin.Ops.UpdateErrorResolution)

		// Request errors (client-visible failures)
		ops.GET("/request-errors", h.Admin.Ops.ListRequestErrors)
		ops.GET("/request-errors/:id", h.Admin.Ops.GetRequestError)
		ops.GET("/request-errors/:id/upstream-errors", h.Admin.Ops.ListRequestErrorUpstreamErrors)
		ops.PUT("/request-errors/:id/resolve", h.Admin.Ops.ResolveRequestError)

		// Bounded ingress-admission rejection aggregates.
		ops.GET("/ingress-rejections", h.Admin.Ops.ListIngressRejects)
		ops.GET("/ingress-rejections/health", h.Admin.Ops.GetIngressRejectHealth)
		ops.GET("/auth-cache-invalidation/health", h.Admin.Ops.GetAuthCacheInvalidationHealth)

		// Upstream errors (independent upstream failures)
		ops.GET("/upstream-errors", h.Admin.Ops.ListUpstreamErrors)
		ops.GET("/upstream-errors/:id", h.Admin.Ops.GetUpstreamError)
		ops.PUT("/upstream-errors/:id/resolve", h.Admin.Ops.ResolveUpstreamError)

		// Request drilldown (success + error)
		ops.GET("/requests", h.Admin.Ops.ListRequestDetails)

		// Indexed system logs
		ops.GET("/system-logs", h.Admin.Ops.ListSystemLogs)
		ops.POST("/system-logs/cleanup", h.Admin.Ops.CleanupSystemLogs)
		ops.GET("/system-logs/health", h.Admin.Ops.GetSystemLogIngestionHealth)

		// Dashboard (vNext - raw path for MVP)
		ops.GET("/dashboard/snapshot-v2", h.Admin.Ops.GetDashboardSnapshotV2)
		ops.GET("/dashboard/overview", h.Admin.Ops.GetDashboardOverview)
		ops.GET("/dashboard/throughput-trend", h.Admin.Ops.GetDashboardThroughputTrend)
		ops.GET("/dashboard/latency-histogram", h.Admin.Ops.GetDashboardLatencyHistogram)
		ops.GET("/dashboard/error-trend", h.Admin.Ops.GetDashboardErrorTrend)
		ops.GET("/dashboard/error-distribution", h.Admin.Ops.GetDashboardErrorDistribution)
		ops.GET("/dashboard/openai-token-stats", h.Admin.Ops.GetDashboardOpenAITokenStats)
	}
}

func registerDashboardRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	dashboard := admin.Group("/dashboard")
	{
		dashboard.GET("/snapshot-v2", h.Admin.Dashboard.GetSnapshotV2)
		dashboard.GET("/stats", h.Admin.Dashboard.GetStats)
		dashboard.GET("/realtime", h.Admin.Dashboard.GetRealtimeMetrics)
		dashboard.GET("/trend", h.Admin.Dashboard.GetUsageTrend)
		dashboard.GET("/models", h.Admin.Dashboard.GetModelStats)
		dashboard.GET("/groups", h.Admin.Dashboard.GetGroupStats)
		dashboard.GET("/api-keys-trend", h.Admin.Dashboard.GetAPIKeyUsageTrend)
		dashboard.GET("/users-trend", h.Admin.Dashboard.GetUserUsageTrend)
		dashboard.GET("/users-ranking", h.Admin.Dashboard.GetUserSpendingRanking)
		dashboard.POST("/users-usage", h.Admin.Dashboard.GetBatchUsersUsage)
		dashboard.POST("/api-keys-usage", h.Admin.Dashboard.GetBatchAPIKeysUsage)
		dashboard.GET("/user-breakdown", h.Admin.Dashboard.GetUserBreakdown)
		dashboard.POST("/aggregation/backfill", h.Admin.Dashboard.BackfillAggregation)
	}
}

func registerUserManagementRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	users := admin.Group("/users")
	{
		users.GET("", h.Admin.User.List)
		users.GET("/:id", h.Admin.User.GetByID)
		users.POST("/:id/auth-identities", h.Admin.User.BindAuthIdentity)
		users.POST("", h.Admin.User.Create)
		users.PUT("/:id", h.Admin.User.Update)
		users.DELETE("/:id", h.Admin.User.Delete)
		users.POST("/:id/balance", h.Admin.User.UpdateBalance)
		users.GET("/:id/api-keys", h.Admin.User.GetUserAPIKeys)
		users.GET("/:id/usage", h.Admin.User.GetUserUsage)
		users.GET("/:id/balance-history", h.Admin.User.GetBalanceHistory)
		users.POST("/:id/replace-group", h.Admin.User.ReplaceGroup)
		users.GET("/:id/rpm-status", h.Admin.User.GetUserRPMStatus)
		users.POST("/batch-concurrency", h.Admin.User.BatchUpdateConcurrency)
		users.POST("/batch-limits", h.Admin.User.BatchUpdateLimits)
		users.GET("/:id/platform-quotas", h.Admin.User.GetUserPlatformQuotas)
		users.PUT("/:id/platform-quotas", h.Admin.User.UpdateUserPlatformQuotas)
		users.POST("/:id/platform-quotas/reset", h.Admin.User.ResetUserPlatformQuotaWindow)

		// User attribute values
		users.GET("/:id/attributes", h.Admin.UserAttribute.GetUserAttributes)
		users.PUT("/:id/attributes", h.Admin.UserAttribute.UpdateUserAttributes)
	}
}

func registerGroupRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	groups := admin.Group("/groups")
	{
		groups.GET("", h.Admin.Group.List)
		groups.GET("/all", h.Admin.Group.GetAll)
		groups.GET("/usage-summary", h.Admin.Group.GetUsageSummary)
		groups.GET("/capacity-summary", h.Admin.Group.GetCapacitySummary)
		groups.GET("/live-capability", h.Admin.Group.GetLiveCapability)
		groups.PUT("/sort-order", h.Admin.Group.UpdateSortOrder)
		groups.GET("/:id/models-list-candidates", h.Admin.Group.GetModelsListCandidates)
		groups.GET("/:id/composite-routes", h.Admin.Group.ListCompositeRoutes)
		groups.POST("/:id/composite-routes", h.Admin.Group.CreateCompositeRoute)
		groups.POST("/:id/composite-routes/preview", h.Admin.Group.PreviewCompositeRoute)
		groups.PUT("/:id/composite-routes/:route_id", h.Admin.Group.UpdateCompositeRoute)
		groups.DELETE("/:id/composite-routes/:route_id", h.Admin.Group.DeleteCompositeRoute)
		groups.GET("/:id", h.Admin.Group.GetByID)
		groups.POST("", h.Admin.Group.Create)
		groups.POST("/:id/duplicate", h.Admin.Group.Duplicate)
		groups.PUT("/:id", h.Admin.Group.Update)
		groups.DELETE("/:id", h.Admin.Group.Delete)
		groups.GET("/:id/stats", h.Admin.Group.GetStats)
		groups.GET("/:id/rate-multipliers", h.Admin.Group.GetGroupRateMultipliers)
		groups.PUT("/:id/rate-multipliers", h.Admin.Group.BatchSetGroupRateMultipliers)
		groups.DELETE("/:id/rate-multipliers", h.Admin.Group.ClearGroupRateMultipliers)
		groups.PUT("/:id/rpm-overrides", h.Admin.Group.BatchSetGroupRPMOverrides)
		groups.DELETE("/:id/rpm-overrides", h.Admin.Group.ClearGroupRPMOverrides)
		groups.GET("/:id/api-keys", h.Admin.Group.GetGroupAPIKeys)
	}
}

func registerAccountRoutes(admin *gin.RouterGroup, h *handler.Handlers, stepUpAuth middleware.StepUpAuthMiddleware) {
	accounts := admin.Group("/accounts")
	{
		accounts.GET("", h.Admin.Account.List)
		accounts.GET("/rows", h.Admin.Account.ListRows)
		accounts.GET("/selection-summary", h.Admin.Account.SelectionSummary)
		accounts.GET("/upstream-billing-probe/settings", h.Admin.Account.GetUpstreamBillingProbeSettings)
		accounts.PUT("/upstream-billing-probe/settings", h.Admin.Account.UpdateUpstreamBillingProbeSettings)
		accounts.POST("/upstream-billing-probe/batch", h.Admin.Account.ProbeUpstreamBillingBatch)
		accounts.GET("/ollama-cloud-usage/settings", h.Admin.Account.GetOllamaCloudUsageSettings)
		accounts.PUT("/ollama-cloud-usage/settings", h.Admin.Account.UpdateOllamaCloudUsageSettings)
		accounts.GET("/:id", h.Admin.Account.GetByID)
		accounts.POST("", h.Admin.Account.Create)
		accounts.POST("/:id/duplicate", h.Admin.Account.Duplicate)
		accounts.POST("/check-mixed-channel", h.Admin.Account.CheckMixedChannel)
		accounts.POST("/import/codex-session", h.Admin.Account.ImportCodexSession)
		accounts.POST("/sync/crs", h.Admin.Account.SyncFromCRS)
		accounts.POST("/sync/crs/preview", h.Admin.Account.PreviewFromCRS)
		accounts.PUT("/:id", h.Admin.Account.Update)
		accounts.PUT("/:id/upstream-billing-probe", h.Admin.Account.SetUpstreamBillingProbeEnabled)
		accounts.POST("/:id/upstream-billing-probe", h.Admin.Account.ProbeUpstreamBilling)
		accounts.GET("/:id/ollama-cloud-usage", h.Admin.Account.GetOllamaCloudUsage)
		accounts.PUT("/:id/ollama-cloud-usage/session", h.Admin.Account.SaveOllamaCloudUsageSession)
		accounts.DELETE("/:id/ollama-cloud-usage/session", h.Admin.Account.DeleteOllamaCloudUsageSession)
		accounts.PUT("/:id/ollama-cloud-usage/auto-refresh", h.Admin.Account.SetOllamaCloudUsageAutoRefresh)
		accounts.POST("/:id/ollama-cloud-usage/refresh", h.Admin.Account.RefreshOllamaCloudUsage)
		accounts.POST("/bulk-delete", h.Admin.Account.BulkDelete)
		accounts.DELETE("/:id", h.Admin.Account.Delete)
		accounts.POST("/:id/test", h.Admin.Account.Test)
		accounts.POST("/:id/recover-state", h.Admin.Account.RecoverState)
		accounts.POST("/:id/refresh", h.Admin.Account.Refresh)
		accounts.POST("/:id/apply-oauth-credentials", h.Admin.Account.ApplyOAuthCredentials)
		accounts.POST("/:id/set-privacy", h.Admin.Account.SetPrivacy)
		accounts.POST("/:id/refresh-tier", h.Admin.Account.RefreshTier)
		accounts.GET("/:id/stats", h.Admin.Account.GetStats)
		accounts.POST("/:id/clear-error", h.Admin.Account.ClearError)
		accounts.POST("/:id/revert-proxy-fallback", h.Admin.Account.RevertProxyFallback)
		accounts.GET("/:id/usage", h.Admin.Account.GetUsage)
		accounts.GET("/:id/today-stats", h.Admin.Account.GetTodayStats)
		accounts.POST("/today-stats/batch", h.Admin.Account.GetBatchTodayStats)
		accounts.POST("/:id/clear-rate-limit", h.Admin.Account.ClearRateLimit)
		accounts.POST("/:id/reset-quota", h.Admin.Account.ResetQuota)
		accounts.GET("/:id/temp-unschedulable", h.Admin.Account.GetTempUnschedulable)
		accounts.DELETE("/:id/temp-unschedulable", h.Admin.Account.ClearTempUnschedulable)
		accounts.POST("/:id/schedulable", h.Admin.Account.SetSchedulable)
		accounts.POST("/models/sync-upstream-preview", h.Admin.Account.SyncUpstreamModelsPreview)
		accounts.GET("/:id/models", h.Admin.Account.GetAvailableModels)
		accounts.POST("/:id/models/sync-upstream", h.Admin.Account.SyncUpstreamModels)
		accounts.POST("/batch", h.Admin.Account.BatchCreate)
		// 账号导出泄露上游凭证原文——要求 step-up 2FA
		accounts.GET("/data", gin.HandlerFunc(stepUpAuth), h.Admin.Account.ExportData)
		accounts.POST("/data", h.Admin.Account.ImportData)
		accounts.POST("/batch-update-credentials", h.Admin.Account.BatchUpdateCredentials)
		accounts.POST("/batch-refresh-tier", h.Admin.Account.BatchRefreshTier)
		accounts.POST("/bulk-update", h.Admin.Account.BulkUpdate)
		accounts.POST("/batch-delete", h.Admin.Account.BatchDelete)
		accounts.POST("/batch-clear-error", h.Admin.Account.BatchClearError)
		accounts.POST("/batch-refresh", h.Admin.Account.BatchRefresh)

		// Antigravity 默认模型映射
		accounts.GET("/antigravity/default-model-mapping", h.Admin.Account.GetAntigravityDefaultModelMapping)

		// Spark 影子账号
		accounts.POST("/:id/shadow", h.Admin.OpenAIOAuth.CreateShadow)

		// Claude OAuth routes
		accounts.POST("/generate-auth-url", h.Admin.OAuth.GenerateAuthURL)
		accounts.POST("/generate-setup-token-url", h.Admin.OAuth.GenerateSetupTokenURL)
		accounts.POST("/exchange-code", h.Admin.OAuth.ExchangeCode)
		accounts.POST("/exchange-setup-token-code", h.Admin.OAuth.ExchangeSetupTokenCode)
		accounts.POST("/cookie-auth", h.Admin.OAuth.CookieAuth)
		accounts.POST("/setup-token-cookie-auth", h.Admin.OAuth.SetupTokenCookieAuth)
	}
}

func registerAnnouncementRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	announcements := admin.Group("/announcements")
	{
		announcements.GET("", h.Admin.Announcement.List)
		announcements.POST("", h.Admin.Announcement.Create)
		announcements.GET("/:id", h.Admin.Announcement.GetByID)
		announcements.PUT("/:id", h.Admin.Announcement.Update)
		announcements.DELETE("/:id", h.Admin.Announcement.Delete)
		announcements.GET("/:id/read-status", h.Admin.Announcement.ListReadStatus)
	}
}

func registerOpenAIOAuthRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	openai := admin.Group("/openai")
	{
		openai.POST("/generate-auth-url", h.Admin.OpenAIOAuth.GenerateAuthURL)
		openai.POST("/exchange-code", h.Admin.OpenAIOAuth.ExchangeCode)
		openai.POST("/refresh-token", h.Admin.OpenAIOAuth.RefreshToken)
		openai.POST("/accounts/:id/refresh", h.Admin.OpenAIOAuth.RefreshAccountToken)
		openai.POST("/create-from-oauth", h.Admin.OpenAIOAuth.CreateAccountFromOAuth)
		openai.POST("/create-from-codex-pat", h.Admin.OpenAIOAuth.CreateAccountFromCodexPAT)
		openai.GET("/accounts/:id/quota", h.Admin.OpenAIOAuth.QueryQuota)
		openai.POST("/accounts/:id/quota/refresh", h.Admin.OpenAIOAuth.RefreshQuota)
		openai.POST("/accounts/:id/reset-quota", h.Admin.OpenAIOAuth.ResetQuota)
	}
}

func registerGeminiOAuthRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	gemini := admin.Group("/gemini")
	{
		gemini.POST("/oauth/auth-url", h.Admin.GeminiOAuth.GenerateAuthURL)
		gemini.POST("/oauth/exchange-code", h.Admin.GeminiOAuth.ExchangeCode)
		gemini.GET("/oauth/capabilities", h.Admin.GeminiOAuth.GetCapabilities)
	}
}

func registerAntigravityOAuthRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	antigravity := admin.Group("/antigravity")
	{
		antigravity.POST("/oauth/auth-url", h.Admin.AntigravityOAuth.GenerateAuthURL)
		antigravity.POST("/oauth/exchange-code", h.Admin.AntigravityOAuth.ExchangeCode)
		antigravity.POST("/oauth/refresh-token", h.Admin.AntigravityOAuth.RefreshToken)
	}
}

func registerGrokOAuthRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	grok := admin.Group("/grok")
	{
		grok.POST("/oauth/auth-url", h.Admin.GrokOAuth.GenerateAuthURL)
		grok.POST("/oauth/exchange-code", h.Admin.GrokOAuth.ExchangeCode)
		grok.POST("/oauth/refresh-token", h.Admin.GrokOAuth.RefreshToken)
		grok.POST("/oauth/create-from-oauth", h.Admin.GrokOAuth.CreateAccountFromOAuth)
		grok.POST("/sso-to-oauth", h.Admin.GrokOAuth.CreateAccountsFromSSO)
		grok.POST("/oauth/reconcile", h.Admin.GrokOAuth.ReconcileOAuthAccounts)
		grok.POST("/accounts/:id/refresh", h.Admin.GrokOAuth.RefreshAccountToken)
		grok.GET("/accounts/:id/quota", h.Admin.GrokOAuth.QueryQuota)
		grok.POST("/accounts/:id/reset-quota", h.Admin.GrokOAuth.ResetQuota)
		grok.GET("/runtime-sanity", h.Admin.GrokOAuth.RuntimeSanity)
	}
}

func registerProxyRoutes(admin *gin.RouterGroup, h *handler.Handlers, stepUpAuth middleware.StepUpAuthMiddleware) {
	proxies := admin.Group("/proxies")
	{
		proxies.GET("", h.Admin.Proxy.List)
		proxies.GET("/all", h.Admin.Proxy.GetAll)
		// 代理导出泄露账号密码原文——要求 step-up 2FA
		proxies.GET("/data", gin.HandlerFunc(stepUpAuth), h.Admin.Proxy.ExportData)
		proxies.POST("/data", h.Admin.Proxy.ImportData)
		proxies.GET("/:id", h.Admin.Proxy.GetByID)
		proxies.POST("", h.Admin.Proxy.Create)
		proxies.PUT("/:id", h.Admin.Proxy.Update)
		proxies.DELETE("/:id", h.Admin.Proxy.Delete)
		proxies.POST("/:id/test", h.Admin.Proxy.Test)
		proxies.POST("/:id/quality-check", h.Admin.Proxy.CheckQuality)
		proxies.GET("/:id/stats", h.Admin.Proxy.GetStats)
		proxies.GET("/:id/accounts", h.Admin.Proxy.GetProxyAccounts)
		proxies.POST("/batch-delete", h.Admin.Proxy.BatchDelete)
		proxies.POST("/batch", h.Admin.Proxy.BatchCreate)
	}
}

func registerRedeemCodeRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	codes := admin.Group("/redeem-codes")
	{
		codes.GET("", h.Admin.Redeem.List)
		codes.GET("/stats", h.Admin.Redeem.GetStats)
		codes.GET("/export", h.Admin.Redeem.Export)
		codes.GET("/:id", h.Admin.Redeem.GetByID)
		codes.POST("/create-and-redeem", h.Admin.Redeem.CreateAndRedeem)
		codes.POST("/generate", h.Admin.Redeem.Generate)
		codes.DELETE("/:id", h.Admin.Redeem.Delete)
		codes.POST("/batch-delete", h.Admin.Redeem.BatchDelete)
		codes.POST("/batch-update", h.Admin.Redeem.BatchUpdate)
		codes.POST("/:id/expire", h.Admin.Redeem.Expire)
	}
}

func registerPromoCodeRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	promoCodes := admin.Group("/promo-codes")
	{
		promoCodes.GET("", h.Admin.Promo.List)
		promoCodes.GET("/:id", h.Admin.Promo.GetByID)
		promoCodes.POST("", h.Admin.Promo.Create)
		promoCodes.PUT("/:id", h.Admin.Promo.Update)
		promoCodes.DELETE("/:id", h.Admin.Promo.Delete)
		promoCodes.GET("/:id/usages", h.Admin.Promo.GetUsages)
	}
}

func registerSettingsRoutes(admin *gin.RouterGroup, h *handler.Handlers, stepUpAuth middleware.StepUpAuthMiddleware) {
	adminSettings := admin.Group("/settings")
	primaryAdmin := gin.HandlerFunc(h.Admin.Pool.RequirePrimaryAdmin)
	{
		adminSettings.GET("", h.Admin.Setting.GetSettings)
		adminSettings.PUT("", h.Admin.Setting.UpdateSettings)
		adminSettings.POST("/test-smtp", h.Admin.Setting.TestSMTPConnection)
		adminSettings.POST("/send-test-email", h.Admin.Setting.SendTestEmail)
		adminSettings.GET("/email-templates", h.Admin.Setting.ListEmailTemplates)
		adminSettings.POST("/email-template-preview", h.Admin.Setting.PreviewEmailTemplate)
		adminSettings.GET("/email-templates/:event/:locale", h.Admin.Setting.GetEmailTemplate)
		adminSettings.PUT("/email-templates/:event/:locale", h.Admin.Setting.UpdateEmailTemplate)
		adminSettings.POST("/email-templates/:event/:locale/restore-official", h.Admin.Setting.RestoreOfficialEmailTemplate)
		// Admin API Key 管理
		adminSettings.GET("/admin-api-key", h.Admin.Setting.GetAdminAPIKey)
		adminSettings.POST("/admin-api-key/regenerate", primaryAdmin, gin.HandlerFunc(stepUpAuth), h.Admin.Setting.RegenerateAdminAPIKey)
		adminSettings.DELETE("/admin-api-key", primaryAdmin, gin.HandlerFunc(stepUpAuth), h.Admin.Setting.DeleteAdminAPIKey)
		// 529过载冷却配置
		adminSettings.GET("/overload-cooldown", h.Admin.Setting.GetOverloadCooldownSettings)
		adminSettings.PUT("/overload-cooldown", h.Admin.Setting.UpdateOverloadCooldownSettings)
		// 429默认回避配置
		adminSettings.GET("/rate-limit-429-cooldown", h.Admin.Setting.GetRateLimit429CooldownSettings)
		adminSettings.PUT("/rate-limit-429-cooldown", h.Admin.Setting.UpdateRateLimit429CooldownSettings)
		// 面板 API 限流配置
		adminSettings.GET("/panel-rate-limit", h.Admin.Setting.GetPanelRateLimitSettings)
		adminSettings.PUT("/panel-rate-limit", h.Admin.Setting.UpdatePanelRateLimitSettings)
		// 流超时处理配置
		adminSettings.GET("/stream-timeout", h.Admin.Setting.GetStreamTimeoutSettings)
		adminSettings.PUT("/stream-timeout", h.Admin.Setting.UpdateStreamTimeoutSettings)
		// 请求整流器配置
		adminSettings.GET("/rectifier", h.Admin.Setting.GetRectifierSettings)
		adminSettings.PUT("/rectifier", h.Admin.Setting.UpdateRectifierSettings)
		// Beta 策略配置
		adminSettings.GET("/beta-policy", h.Admin.Setting.GetBetaPolicySettings)
		adminSettings.PUT("/beta-policy", h.Admin.Setting.UpdateBetaPolicySettings)
		// Web Search 模拟配置
		adminSettings.GET("/web-search-emulation", h.Admin.Setting.GetWebSearchEmulationConfig)
		adminSettings.PUT("/web-search-emulation", h.Admin.Setting.UpdateWebSearchEmulationConfig)
		adminSettings.POST("/web-search-emulation/test", h.Admin.Setting.TestWebSearchEmulation)
		adminSettings.POST("/web-search-emulation/reset-usage", h.Admin.Setting.ResetWebSearchUsage)
	}
}

func registerDataManagementRoutes(admin *gin.RouterGroup, h *handler.Handlers, stepUpAuth middleware.StepUpAuthMiddleware) {
	dataManagement := admin.Group("/data-management")
	primaryAdmin := gin.HandlerFunc(h.Admin.Pool.RequirePrimaryAdmin)
	{
		dataManagement.GET("/agent/health", h.Admin.DataManagement.GetAgentHealth)
		dataManagement.GET("/config", h.Admin.DataManagement.GetConfig)
		dataManagement.PUT("/config", h.Admin.DataManagement.UpdateConfig)
		dataManagement.GET("/sources/:source_type/profiles", h.Admin.DataManagement.ListSourceProfiles)
		dataManagement.POST("/sources/:source_type/profiles", h.Admin.DataManagement.CreateSourceProfile)
		dataManagement.PUT("/sources/:source_type/profiles/:profile_id", h.Admin.DataManagement.UpdateSourceProfile)
		dataManagement.DELETE("/sources/:source_type/profiles/:profile_id", h.Admin.DataManagement.DeleteSourceProfile)
		dataManagement.POST("/sources/:source_type/profiles/:profile_id/activate", h.Admin.DataManagement.SetActiveSourceProfile)
		dataManagement.POST("/s3/test", h.Admin.DataManagement.TestS3)
		dataManagement.GET("/s3/profiles", h.Admin.DataManagement.ListS3Profiles)
		// 修改 S3 目标可将数据备份外泄——要求 step-up 2FA
		dataManagement.POST("/s3/profiles", primaryAdmin, gin.HandlerFunc(stepUpAuth), h.Admin.DataManagement.CreateS3Profile)
		dataManagement.PUT("/s3/profiles/:profile_id", primaryAdmin, gin.HandlerFunc(stepUpAuth), h.Admin.DataManagement.UpdateS3Profile)
		dataManagement.DELETE("/s3/profiles/:profile_id", h.Admin.DataManagement.DeleteS3Profile)
		dataManagement.POST("/s3/profiles/:profile_id/activate", primaryAdmin, gin.HandlerFunc(stepUpAuth), h.Admin.DataManagement.SetActiveS3Profile)
		dataManagement.POST("/backups", primaryAdmin, gin.HandlerFunc(stepUpAuth), h.Admin.DataManagement.CreateBackupJob)
		dataManagement.GET("/backups", h.Admin.DataManagement.ListBackupJobs)
		dataManagement.GET("/backups/:job_id", h.Admin.DataManagement.GetBackupJob)
	}
}

func registerBackupRoutes(admin *gin.RouterGroup, h *handler.Handlers, stepUpAuth middleware.StepUpAuthMiddleware) {
	backup := admin.Group("/backups")
	primaryAdmin := gin.HandlerFunc(h.Admin.Pool.RequirePrimaryAdmin)
	{
		// S3 存储配置
		backup.GET("/s3-config", h.Admin.Backup.GetS3Config)
		// 修改 S3 目标可将数据库备份外泄——要求 step-up 2FA
		backup.PUT("/s3-config", primaryAdmin, gin.HandlerFunc(stepUpAuth), h.Admin.Backup.UpdateS3Config)
		backup.POST("/s3-config/test", h.Admin.Backup.TestS3Connection)

		// 异步生图对象存储配置（与备份共用 S3 客户端，可直接复用备份凭证）
		backup.GET("/image-storage", h.Admin.Backup.GetImageStorageConfig)
		// 同 S3 配置：改写对象存储目标可将生成内容导向外部账号——要求 step-up 2FA
		backup.PUT("/image-storage", gin.HandlerFunc(stepUpAuth), h.Admin.Backup.UpdateImageStorageConfig)
		backup.POST("/image-storage/test", h.Admin.Backup.TestImageStorageConnection)

		// 定时备份配置
		backup.GET("/schedule", h.Admin.Backup.GetSchedule)
		backup.PUT("/schedule", primaryAdmin, h.Admin.Backup.UpdateSchedule)

		// 备份操作
		backup.POST("", primaryAdmin, gin.HandlerFunc(stepUpAuth), h.Admin.Backup.CreateBackup)
		backup.GET("", h.Admin.Backup.ListBackups)
		backup.GET("/:id", h.Admin.Backup.GetBackup)
		backup.DELETE("/:id", h.Admin.Backup.DeleteBackup)
		// 备份下载链接可直接取走整库数据——要求 step-up 2FA
		backup.GET("/:id/download-url", primaryAdmin, gin.HandlerFunc(stepUpAuth), h.Admin.Backup.GetDownloadURL)

		// 恢复操作：整库覆盖可回滚安全设置（含 step-up 开关本身）——要求 step-up 2FA
		backup.POST("/:id/restore", primaryAdmin, gin.HandlerFunc(stepUpAuth), h.Admin.Backup.RestoreBackup)
	}
}

func registerSystemRoutes(admin *gin.RouterGroup, h *handler.Handlers, stepUpAuth middleware.StepUpAuthMiddleware) {
	system := admin.Group("/system")
	primaryAdmin := gin.HandlerFunc(h.Admin.Pool.RequirePrimaryAdmin)
	{
		system.GET("/version", h.Admin.System.GetVersion)
		system.GET("/check-updates", h.Admin.System.CheckUpdates)
		system.GET("/rollback-versions", h.Admin.System.GetRollbackVersions)
		system.POST("/update", primaryAdmin, gin.HandlerFunc(stepUpAuth), h.Admin.System.PerformUpdate)
		system.POST("/rollback", primaryAdmin, gin.HandlerFunc(stepUpAuth), h.Admin.System.Rollback)
		system.POST("/restart", primaryAdmin, gin.HandlerFunc(stepUpAuth), h.Admin.System.RestartService)
	}
}

func registerSubscriptionRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	subscriptions := admin.Group("/subscriptions")
	{
		subscriptions.GET("", h.Admin.Subscription.List)
		subscriptions.GET("/:id", h.Admin.Subscription.GetByID)
		subscriptions.GET("/:id/progress", h.Admin.Subscription.GetProgress)
		subscriptions.POST("/assign", h.Admin.Subscription.Assign)
		subscriptions.POST("/bulk-assign", h.Admin.Subscription.BulkAssign)
		subscriptions.POST("/:id/extend", h.Admin.Subscription.Extend)
		subscriptions.POST("/:id/reset-quota", h.Admin.Subscription.ResetQuota)
		subscriptions.POST("/:id/revoke", h.Admin.Subscription.Revoke)
		subscriptions.POST("/:id/restore", h.Admin.Subscription.Restore)
		subscriptions.DELETE("/:id", h.Admin.Subscription.Revoke)
	}

	// 分组下的订阅列表
	admin.GET("/groups/:id/subscriptions", h.Admin.Subscription.ListByGroup)

	// 用户下的订阅列表
	admin.GET("/users/:id/subscriptions", h.Admin.Subscription.ListByUser)
}

func registerUsageRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	usage := admin.Group("/usage")
	{
		usage.GET("", h.Admin.Usage.List)
		usage.GET("/stats", h.Admin.Usage.Stats)
		usage.GET("/search-users", h.Admin.Usage.SearchUsers)
		usage.GET("/search-api-keys", h.Admin.Usage.SearchAPIKeys)
		usage.GET("/cleanup-tasks", h.Admin.Usage.ListCleanupTasks)
		usage.POST("/cleanup-tasks", h.Admin.Usage.CreateCleanupTask)
		usage.POST("/cleanup-tasks/:id/cancel", h.Admin.Usage.CancelCleanupTask)
	}
}

func registerUserAttributeRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	attrs := admin.Group("/user-attributes")
	{
		attrs.GET("", h.Admin.UserAttribute.ListDefinitions)
		attrs.POST("", h.Admin.UserAttribute.CreateDefinition)
		attrs.POST("/batch", h.Admin.UserAttribute.GetBatchUserAttributes)
		attrs.PUT("/reorder", h.Admin.UserAttribute.ReorderDefinitions)
		attrs.PUT("/:id", h.Admin.UserAttribute.UpdateDefinition)
		attrs.DELETE("/:id", h.Admin.UserAttribute.DeleteDefinition)
	}
}

func registerScheduledTestRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	plans := admin.Group("/scheduled-test-plans")
	{
		plans.POST("", h.Admin.ScheduledTest.Create)
		plans.PUT("/:id", h.Admin.ScheduledTest.Update)
		plans.DELETE("/:id", h.Admin.ScheduledTest.Delete)
		plans.GET("/:id/results", h.Admin.ScheduledTest.ListResults)
	}
	// Nested under accounts
	admin.GET("/accounts/:id/scheduled-test-plans", h.Admin.ScheduledTest.ListByAccount)
}

func registerErrorPassthroughRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	rules := admin.Group("/error-passthrough-rules")
	{
		rules.GET("", h.Admin.ErrorPassthrough.List)
		rules.GET("/:id", h.Admin.ErrorPassthrough.GetByID)
		rules.POST("", h.Admin.ErrorPassthrough.Create)
		rules.PUT("/:id", h.Admin.ErrorPassthrough.Update)
		rules.DELETE("/:id", h.Admin.ErrorPassthrough.Delete)
	}
}

func registerTLSFingerprintProfileRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	profiles := admin.Group("/tls-fingerprint-profiles")
	{
		profiles.GET("", h.Admin.TLSFingerprintProfile.List)
		profiles.GET("/:id", h.Admin.TLSFingerprintProfile.GetByID)
		profiles.POST("", h.Admin.TLSFingerprintProfile.Create)
		profiles.PUT("/:id", h.Admin.TLSFingerprintProfile.Update)
		profiles.DELETE("/:id", h.Admin.TLSFingerprintProfile.Delete)
	}
}

func registerChannelRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	channels := admin.Group("/channels")
	{
		channels.GET("", h.Admin.Channel.List)
		channels.GET("/model-pricing", h.Admin.Channel.GetModelDefaultPricing)
		channels.GET("/pricing/sync-models", h.Admin.Channel.SyncPricingModels)
		channels.GET("/:id", h.Admin.Channel.GetByID)
		channels.POST("", h.Admin.Channel.Create)
		channels.PUT("/:id", h.Admin.Channel.Update)
		channels.DELETE("/:id", h.Admin.Channel.Delete)
	}
}

func registerChannelMonitorRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	monitors := admin.Group("/channel-monitors")
	{
		monitors.GET("", h.Admin.ChannelMonitor.List)
		monitors.POST("", h.Admin.ChannelMonitor.Create)
		monitors.GET("/:id", h.Admin.ChannelMonitor.Get)
		monitors.POST("/:id/duplicate", h.Admin.ChannelMonitor.Duplicate)
		monitors.PUT("/:id", h.Admin.ChannelMonitor.Update)
		monitors.DELETE("/:id", h.Admin.ChannelMonitor.Delete)
		monitors.POST("/:id/run", h.Admin.ChannelMonitor.Run)
		monitors.GET("/:id/history", h.Admin.ChannelMonitor.History)
	}

	templates := admin.Group("/channel-monitor-templates")
	{
		templates.GET("", h.Admin.ChannelMonitorTemplate.List)
		templates.POST("", h.Admin.ChannelMonitorTemplate.Create)
		templates.GET("/:id", h.Admin.ChannelMonitorTemplate.Get)
		templates.PUT("/:id", h.Admin.ChannelMonitorTemplate.Update)
		templates.DELETE("/:id", h.Admin.ChannelMonitorTemplate.Delete)
		templates.GET("/:id/monitors", h.Admin.ChannelMonitorTemplate.AssociatedMonitors)
		templates.POST("/:id/apply", h.Admin.ChannelMonitorTemplate.Apply)
	}
}

// registerAffiliateRoutes 注册邀请返利的管理端路由（专属用户配置）
func registerAffiliateRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	affiliates := admin.Group("/affiliates")
	{
		affiliates.GET("/invites", h.Admin.Affiliate.ListInviteRecords)
		affiliates.GET("/rebates", h.Admin.Affiliate.ListRebateRecords)
		affiliates.GET("/transfers", h.Admin.Affiliate.ListTransferRecords)

		users := affiliates.Group("/users")
		{
			users.GET("", h.Admin.Affiliate.ListUsers)
			users.GET("/lookup", h.Admin.Affiliate.LookupUsers)
			users.POST("/batch-rate", h.Admin.Affiliate.BatchSetRate)
			users.GET("/:user_id/overview", h.Admin.Affiliate.GetUserOverview)
			users.PUT("/:user_id", h.Admin.Affiliate.UpdateUserSettings)
			users.DELETE("/:user_id", h.Admin.Affiliate.ClearUserSettings)
		}
	}
}
