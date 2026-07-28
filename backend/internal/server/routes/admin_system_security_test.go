package routes

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	adminhandler "github.com/Wei-Shaw/sub2api/internal/handler/admin"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type systemSecuritySettingRepo struct{ service.SettingRepository }

func (systemSecuritySettingRepo) GetValue(context.Context, string) (string, error) { return "1", nil }

func TestSystemMutationsRequirePrimaryAdminThenStepUp(t *testing.T) {
	gin.SetMode(gin.TestMode)
	settings := service.NewSettingService(systemSecuritySettingRepo{}, nil)
	pool := adminhandler.NewPoolHandler(service.NewPoolService(nil, nil, nil, nil, settings))
	h := &handler.Handlers{Admin: &handler.AdminHandlers{
		Pool: pool, System: adminhandler.NewSystemHandler(nil, nil),
	}}
	stepUpCalls := 0
	stepUp := middleware.StepUpAuthMiddleware(func(c *gin.Context) {
		stepUpCalls++
		c.AbortWithStatus(http.StatusTeapot)
	})

	for _, actorID := range []int64{2, 1} {
		for _, path := range []string{
			"/system/update", "/system/rollback", "/system/restart",
			"/settings/admin-api-key/regenerate", "/settings/admin-api-key",
		} {
			router := gin.New()
			router.Use(func(c *gin.Context) {
				c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: actorID})
			})
			registerSystemRoutes(&router.RouterGroup, h, stepUp)
			registerSettingsRoutes(&router.RouterGroup, h, stepUp)
			rec := httptest.NewRecorder()
			method := http.MethodPost
			if path == "/settings/admin-api-key" {
				method = http.MethodDelete
			}
			router.ServeHTTP(rec, httptest.NewRequest(method, path, nil))
			want := http.StatusForbidden
			if actorID == 1 {
				want = http.StatusTeapot
			}
			require.Equal(t, want, rec.Code, "actor=%s path=%s", strconv.FormatInt(actorID, 10), path)
		}
	}
	require.Equal(t, 5, stepUpCalls)
}
