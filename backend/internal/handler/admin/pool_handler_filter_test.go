package admin

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestPoolCostSummaryFilterKeepsRuntimeAndLifecycleSeparate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest("GET", "/?account_status=error&availability_status=rate_limited&lifecycle_status=retired", nil)

	filter, ok := poolCostSummaryFilter(ctx)

	require.True(t, ok)
	require.Equal(t, "error", filter.AccountStatus)
	require.Equal(t, "rate_limited", filter.AvailabilityStatus)
	require.Equal(t, "retired", filter.LifecycleStatus)
}

func TestPoolOverviewRejectsInvalidAccountID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/?account_id=invalid", nil)

	NewPoolHandler(nil).GetOverview(ctx)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
}
