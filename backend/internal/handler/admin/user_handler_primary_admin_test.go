package admin

import (
	"errors"
	"net/http"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func primaryAdminUserRouter() *gin.Engine {
	adminSvc := newStubAdminService()
	adminSvc.users[0].Role = service.RoleAdmin
	settings := service.NewSettingService(approvalGateSettingRepo{primaryID: "1"}, nil)
	h := NewUserHandler(adminSvc, nil, nil, nil, nil, nil, settings)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: 2})
		c.Next()
	})
	router.PUT("/users/:id", h.Update)
	router.POST("/users/:id/auth-identities", h.BindAuthIdentity)
	return router
}

func TestNonPrimaryAdminCannotTakeOverPrimaryAdmin(t *testing.T) {
	router := primaryAdminUserRouter()

	update := doJSON(t, router, http.MethodPut, "/users/1", map[string]any{"email": "attacker@example.com", "password": "pass123"})
	require.Equal(t, http.StatusForbidden, update.Code)

	bind := doJSON(t, router, http.MethodPost, "/users/1/auth-identities", map[string]any{
		"provider_type": "oidc", "provider_key": "issuer", "provider_subject": "attacker",
	})
	require.Equal(t, http.StatusForbidden, bind.Code)
}

func TestPrimaryAdminLookupFailureBlocksSensitiveUserRoutes(t *testing.T) {
	adminSvc := newStubAdminService()
	settings := service.NewSettingService(approvalGateSettingRepo{err: errors.New("database unavailable")}, nil)
	h := NewUserHandler(adminSvc, nil, nil, nil, nil, nil, settings)
	router := gin.New()
	router.PUT("/users/:id", h.Update)
	router.POST("/users/:id/auth-identities", h.BindAuthIdentity)

	update := doJSON(t, router, http.MethodPut, "/users/1", map[string]any{"email": "changed@example.com"})
	require.Equal(t, http.StatusServiceUnavailable, update.Code)

	bind := doJSON(t, router, http.MethodPost, "/users/1/auth-identities", map[string]any{
		"provider_type": "oidc", "provider_key": "issuer", "provider_subject": "subject",
	})
	require.Equal(t, http.StatusServiceUnavailable, bind.Code)
}
