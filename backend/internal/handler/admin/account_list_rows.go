package admin

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"golang.org/x/sync/errgroup"
)

const accountLogicalRowFetchSize = 200

type accountBatchStatusSummary struct {
	Normal              int `json:"normal"`
	Error               int `json:"error"`
	Inactive            int `json:"inactive"`
	RateLimited         int `json:"rate_limited"`
	Overloaded          int `json:"overloaded"`
	TempUnschedulable   int `json:"temp_unschedulable"`
	ManualUnschedulable int `json:"manual_unschedulable"`
}

type accountImportBatchSummary struct {
	ID               string                    `json:"id"`
	UploaderUserID   *int64                    `json:"uploader_user_id,omitempty"`
	UploaderEmail    *string                   `json:"uploader_email,omitempty"`
	UploaderUsername *string                   `json:"uploader_username,omitempty"`
	CreatedAt        time.Time                 `json:"created_at"`
	MatchedCount     int                       `json:"matched_count"`
	TotalCount       int                       `json:"total_count"`
	SchedulableCount int                       `json:"schedulable_count"`
	Names            []string                  `json:"names"`
	Status           accountBatchStatusSummary `json:"status"`
}

type accountLogicalRow struct {
	Kind    string                     `json:"kind"`
	Account *AccountWithConcurrency    `json:"account,omitempty"`
	Batch   *accountImportBatchSummary `json:"batch,omitempty"`
	account *service.Account
}

func importBatchID(account *service.Account) string {
	if account == nil || account.Extra == nil {
		return ""
	}
	value, _ := account.Extra["import_batch_id"].(string)
	return value
}

func accountEffectivelySchedulable(account *service.Account, now time.Time) bool {
	effectivelySchedulable := account.Status == service.StatusActive && account.Schedulable
	if account.AutoPauseOnExpired && account.ExpiresAt != nil && !now.Before(*account.ExpiresAt) {
		effectivelySchedulable = false
	}
	if account.OverloadUntil != nil && account.OverloadUntil.After(now) {
		effectivelySchedulable = false
	}
	if account.RateLimitResetAt != nil && account.RateLimitResetAt.After(now) {
		effectivelySchedulable = false
	}
	if account.TempUnschedulableUntil != nil && account.TempUnschedulableUntil.After(now) {
		effectivelySchedulable = false
	}
	if effectivelySchedulable && account.IsAPIKeyOrBedrock() && account.IsQuotaExceeded() {
		effectivelySchedulable = false
	}
	return effectivelySchedulable
}

func addBatchStatus(summary *accountImportBatchSummary, account *service.Account, now time.Time) {
	effectivelySchedulable := accountEffectivelySchedulable(account, now)
	if effectivelySchedulable {
		summary.SchedulableCount++
	}
	switch {
	case account.Status == service.StatusError:
		summary.Status.Error++
	case account.Status != service.StatusActive:
		summary.Status.Inactive++
	case !account.Schedulable:
		summary.Status.ManualUnschedulable++
	case account.AutoPauseOnExpired && account.ExpiresAt != nil && !now.Before(*account.ExpiresAt):
		summary.Status.ManualUnschedulable++
	case account.OverloadUntil != nil && account.OverloadUntil.After(now):
		summary.Status.Overloaded++
	case account.RateLimitResetAt != nil && account.RateLimitResetAt.After(now):
		summary.Status.RateLimited++
	case account.TempUnschedulableUntil != nil && account.TempUnschedulableUntil.After(now):
		summary.Status.TempUnschedulable++
	case !effectivelySchedulable:
		summary.Status.ManualUnschedulable++
	default:
		summary.Status.Normal++
	}
}

func buildAccountLogicalRows(accounts []service.Account, now time.Time) []accountLogicalRow {
	rows := make([]accountLogicalRow, 0, len(accounts))
	batches := make(map[string]*accountImportBatchSummary)
	for i := range accounts {
		account := &accounts[i]
		batchID := importBatchID(account)
		if batchID == "" {
			rows = append(rows, accountLogicalRow{Kind: "account", account: account})
			continue
		}
		batch := batches[batchID]
		if batch == nil {
			batch = &accountImportBatchSummary{
				ID:               batchID,
				UploaderUserID:   account.CreatedByUserID,
				UploaderEmail:    account.UploaderEmail,
				UploaderUsername: account.UploaderUsername,
				CreatedAt:        account.CreatedAt,
			}
			batches[batchID] = batch
			rows = append(rows, accountLogicalRow{Kind: "import_batch", Batch: batch})
		}
		batch.MatchedCount++
		batch.TotalCount++
		if account.CreatedAt.Before(batch.CreatedAt) {
			batch.CreatedAt = account.CreatedAt
		}
		if len(batch.Names) < 4 {
			batch.Names = append(batch.Names, account.Name)
		}
		addBatchStatus(batch, account, now)
	}
	return rows
}

func accountRowStatusRank(account *service.Account, now time.Time) int {
	summary := accountImportBatchSummary{}
	addBatchStatus(&summary, account, now)
	return batchStatusRank(summary.Status)
}

func batchStatusRank(status accountBatchStatusSummary) int {
	switch {
	case status.Error > 0:
		return 6
	case status.RateLimited > 0:
		return 5
	case status.Overloaded > 0:
		return 4
	case status.TempUnschedulable > 0:
		return 3
	case status.Inactive > 0:
		return 2
	case status.ManualUnschedulable > 0:
		return 1
	default:
		return 0
	}
}

func sortAccountLogicalRows(rows []accountLogicalRow, sortBy, sortOrder string, now time.Time) {
	desc := strings.EqualFold(sortOrder, "desc")
	compare := func(left, right int) int {
		if left < right {
			return -1
		}
		if left > right {
			return 1
		}
		return 0
	}
	rowTime := func(row accountLogicalRow) time.Time {
		if row.account != nil {
			return row.account.CreatedAt
		}
		return row.Batch.CreatedAt
	}
	rowName := func(row accountLogicalRow) string {
		if row.account != nil {
			return strings.ToLower(row.account.Name)
		}
		if len(row.Batch.Names) > 0 {
			return strings.ToLower(row.Batch.Names[0])
		}
		return ""
	}
	rowStatus := func(row accountLogicalRow) int {
		if row.account != nil {
			return accountRowStatusRank(row.account, now)
		}
		return batchStatusRank(row.Batch.Status)
	}
	rowSchedulable := func(row accountLogicalRow) (int, int) {
		if row.account != nil {
			if accountEffectivelySchedulable(row.account, now) {
				return 1, 1
			}
			return 0, 1
		}
		return row.Batch.SchedulableCount, max(row.Batch.MatchedCount, 1)
	}

	sort.SliceStable(rows, func(i, j int) bool {
		var order int
		switch sortBy {
		case "name":
			order = strings.Compare(rowName(rows[i]), rowName(rows[j]))
		case "status":
			order = compare(rowStatus(rows[i]), rowStatus(rows[j]))
		case "schedulable":
			leftCount, leftTotal := rowSchedulable(rows[i])
			rightCount, rightTotal := rowSchedulable(rows[j])
			order = compare(leftCount*rightTotal, rightCount*leftTotal)
		default:
			order = rowTime(rows[i]).Compare(rowTime(rows[j]))
		}
		if order == 0 {
			order = rowTime(rows[i]).Compare(rowTime(rows[j]))
		}
		if desc {
			return order > 0
		}
		return order < 0
	})
}

func (h *AccountHandler) loadAllAccountRows(ctx context.Context, filters accountFilterQuery, includePoolMetrics bool) ([]service.Account, error) {
	// ponytail: Fold admin rows in memory; move this to SQL only when account volume makes it measurable.
	var all []service.Account
	for page := 1; ; page++ {
		items, total, err := h.listAccountPage(ctx, page, accountLogicalRowFetchSize, filters, includePoolMetrics)
		if err != nil {
			return nil, err
		}
		all = append(all, items...)
		if len(all) >= int(total) || len(items) == 0 {
			return all, nil
		}
	}
}

func (h *AccountHandler) enrichLogicalRowRuntime(ctx context.Context, accounts []*service.Account, responses []AccountWithConcurrency, filters accountFilterQuery, includeSchedulerScore bool) {
	// Standalone logical rows keep the same runtime fields as the regular account list.
	accountIDs := make([]int64, len(accounts))
	pageAccounts := make([]service.Account, len(accounts))
	pageHasOpenAI := false
	for i, account := range accounts {
		accountIDs[i] = account.ID
		pageAccounts[i] = *account
		pageHasOpenAI = pageHasOpenAI || account.Platform == service.PlatformOpenAI
	}

	if h.concurrencyService != nil && len(accountIDs) > 0 {
		if counts, err := h.concurrencyService.GetAccountConcurrencyBatch(ctx, accountIDs); err == nil {
			for i, account := range accounts {
				responses[i].CurrentConcurrency = counts[account.ID]
			}
		}
	}

	if includeSchedulerScore && pageHasOpenAI {
		pool := h.listAccountSchedulerScoreFilterPool(ctx, filters.Platform, filters.Type, filters.Status, filters.Search, filters.GroupID, filters.PrivacyMode)
		if filters.UploaderUserID > 0 || filters.UploaderUnassigned || filters.ImportBatchID != "" || filters.ImportBatchScope != "" {
			filtered := pool[:0]
			for i := range pool {
				account := &pool[i]
				batchID, _ := account.Extra["import_batch_id"].(string)
				if filters.UploaderUserID > 0 && (account.CreatedByUserID == nil || *account.CreatedByUserID != filters.UploaderUserID) {
					continue
				}
				if filters.UploaderUnassigned && account.CreatedByUserID != nil {
					continue
				}
				if filters.ImportBatchID != "" && batchID != filters.ImportBatchID {
					continue
				}
				if filters.ImportBatchScope == "standalone" && batchID != "" {
					continue
				}
				if filters.ImportBatchScope == "batched" && batchID == "" {
					continue
				}
				filtered = append(filtered, *account)
			}
			pool = filtered
		}
		scores, groupScores := h.buildOpenAIAccountSchedulerScores(ctx, pageAccounts, pool)
		for i, account := range accounts {
			responses[i].SchedulerScore = scores[account.ID]
			responses[i].SchedulerScores = groupScores[account.ID]
		}
	}

	windowIDs := make([]int64, 0)
	sessionIDs := make([]int64, 0)
	rpmIDs := make([]int64, 0)
	idleTimeouts := make(map[int64]time.Duration)
	for _, account := range accounts {
		if !account.IsAnthropicOAuthOrSetupToken() {
			continue
		}
		if account.GetWindowCostLimit() > 0 {
			windowIDs = append(windowIDs, account.ID)
		}
		if account.GetMaxSessions() > 0 {
			sessionIDs = append(sessionIDs, account.ID)
			idleTimeouts[account.ID] = time.Duration(account.GetSessionIdleTimeoutMinutes()) * time.Minute
		}
		if account.GetBaseRPM() > 0 {
			rpmIDs = append(rpmIDs, account.ID)
		}
	}

	if len(rpmIDs) > 0 && h.rpmCache != nil {
		if counts, _ := h.rpmCache.GetRPMBatch(ctx, rpmIDs); counts != nil {
			for i, account := range accounts {
				if count, ok := counts[account.ID]; ok {
					responses[i].CurrentRPM = &count
				}
			}
		}
	}
	if len(sessionIDs) > 0 && h.sessionLimitCache != nil {
		if counts, _ := h.sessionLimitCache.GetActiveSessionCountBatch(ctx, sessionIDs, idleTimeouts); counts != nil {
			for i, account := range accounts {
				if count, ok := counts[account.ID]; ok {
					responses[i].ActiveSessions = &count
				}
			}
		}
	}
	if len(windowIDs) > 0 && h.accountUsageService != nil {
		costs := make(map[int64]float64)
		var mu sync.Mutex
		group, groupCtx := errgroup.WithContext(ctx)
		group.SetLimit(10)
		for _, account := range accounts {
			if !account.IsAnthropicOAuthOrSetupToken() || account.GetWindowCostLimit() <= 0 {
				continue
			}
			account := account
			group.Go(func() error {
				stats, err := h.accountUsageService.GetAccountWindowStats(groupCtx, account.ID, account.GetCurrentWindowStartTime())
				if err == nil && stats != nil {
					mu.Lock()
					costs[account.ID] = stats.StandardCost
					mu.Unlock()
				}
				return nil
			})
		}
		_ = group.Wait()
		for i, account := range accounts {
			if cost, ok := costs[account.ID]; ok {
				responses[i].CurrentWindowCost = &cost
			}
		}
	}
}

// ListRows returns globally unique first-level account and import-batch rows.
func (h *AccountHandler) ListRows(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	filters, err := parseAccountFilterQuery(c)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if _, exists := c.GetQuery("sort_by"); !exists {
		filters.SortBy = "created_at"
		filters.SortOrder = "desc"
	}
	accounts, err := h.loadAllAccountRows(c.Request.Context(), filters, true)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	now := time.Now()
	rows := buildAccountLogicalRows(accounts, now)
	sortAccountLogicalRows(rows, filters.SortBy, filters.SortOrder, now)
	total := int64(len(rows))
	start := (page - 1) * pageSize
	if start >= len(rows) {
		response.Paginated(c, []accountLogicalRow{}, total, page, pageSize)
		return
	}
	end := min(start+pageSize, len(rows))
	pageRows := rows[start:end]

	standalone := make([]*service.Account, 0, len(pageRows))
	for i := range pageRows {
		row := &pageRows[i]
		if row.account != nil {
			standalone = append(standalone, row.account)
			continue
		}
		batchFilters := accountFilterQuery{
			AccountSelectionFilters: service.AccountSelectionFilters{ImportBatchID: row.Batch.ID},
			SortBy:                  "id",
			SortOrder:               "asc",
		}
		_, batchTotal, countErr := h.listAccountPage(c.Request.Context(), 1, 1, batchFilters, false)
		if countErr != nil {
			response.ErrorFrom(c, countErr)
			return
		}
		row.Batch.TotalCount = int(batchTotal)
	}

	if h.ollamaCloudUsage != nil && len(standalone) > 0 {
		if err := h.ollamaCloudUsage.ResolveAccounts(c.Request.Context(), standalone); err != nil {
			response.ErrorFrom(c, err)
			return
		}
	}
	responses := make([]AccountWithConcurrency, len(standalone))
	for i, account := range standalone {
		responses[i] = AccountWithConcurrency{Account: h.accountResponseFromService(account)}
	}
	h.enrichLogicalRowRuntime(c.Request.Context(), standalone, responses, filters, parseBoolQueryWithDefault(c.Query("include_scheduler_score"), false))
	h.enrichShadowParents(c.Request.Context(), responses)
	responseIndex := 0
	for i := range pageRows {
		pageRows[i].account = nil
		if pageRows[i].Kind == "account" {
			pageRows[i].Account = &responses[responseIndex]
			responseIndex++
		}
	}

	c.Header("Cache-Control", "no-store")
	response.Paginated(c, pageRows, total, page, pageSize)
}
