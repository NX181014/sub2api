package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type approvalGateSettingRepo struct {
	service.SettingRepository
	primaryID string
	stepUp    string
	err       error
}

func (r approvalGateSettingRepo) GetValue(_ context.Context, key string) (string, error) {
	if key == service.SettingKeyStepUpEnabled {
		return r.stepUp, r.err
	}
	return r.primaryID, r.err
}

type approvalGateRepo struct {
	service.PoolApprovalRepository
	created *service.PoolApproval
}

func (r *approvalGateRepo) ExpireStale(context.Context, time.Time) error { return nil }

func (r *approvalGateRepo) GetApprovalAccountState(context.Context, int64) (*service.PoolApprovalAccountState, error) {
	return &service.PoolApprovalAccountState{}, nil
}

func (r *approvalGateRepo) CreateApproval(_ context.Context, item *service.PoolApproval) (*service.PoolApproval, error) {
	item.ID = 99
	r.created = item
	return item, nil
}

func newApprovalGateHandler(actorID int64) (*gin.Engine, *stubAdminService, *approvalGateRepo) {
	gin.SetMode(gin.TestMode)
	adminSvc := newStubAdminService()
	adminSvc.getAccountResult = &service.Account{
		ID: 7, Name: "oauth-account", Platform: service.PlatformOpenAI, Type: service.AccountTypeOAuth,
		Status: service.StatusError, Credentials: map[string]any{"access_token": "stored-secret"}, Extra: map[string]any{},
	}
	repo := &approvalGateRepo{}
	settings := service.NewSettingService(approvalGateSettingRepo{primaryID: "1"}, nil)
	poolService := service.NewPoolService(nil, repo, adminSvc, nil, settings)
	h := NewAccountHandler(adminSvc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	h.SetPoolService(poolService)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: actorID})
		c.Next()
	})
	router.POST("/accounts/:id/apply-oauth-credentials", h.ApplyOAuthCredentials)
	router.POST("/accounts/bulk-update", h.BulkUpdate)
	router.POST("/accounts/batch-update-credentials", h.BatchUpdateCredentials)
	router.GET("/accounts/data", h.ExportData)
	return router, adminSvc, repo
}

func TestNonPrimaryAdminCannotBypassApprovalWithBulkOrExport(t *testing.T) {
	router, adminSvc, _ := newApprovalGateHandler(2)
	tests := []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodPost, "/accounts/bulk-update", `{"account_ids":[7],"name":"changed"}`},
		{http.MethodPost, "/accounts/batch-update-credentials", `{"account_ids":[7],"field":"access_token","value":"secret"}`},
		{http.MethodGet, "/accounts/data?ids=7", ""},
	}
	for _, tt := range tests {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
		if w.Code != http.StatusForbidden {
			t.Fatalf("%s %s status = %d, body=%s", tt.method, tt.path, w.Code, w.Body.String())
		}
	}
	if adminSvc.lastBulkUpdateAccountInput != nil || adminSvc.updateAccountCalls != 0 || adminSvc.lastListAccounts.calls != 0 {
		t.Fatal("a blocked bulk/export request reached the account service")
	}
}

func TestPrimaryAdminMayUseBulkUpdate(t *testing.T) {
	router, adminSvc, _ := newApprovalGateHandler(1)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/accounts/bulk-update", strings.NewReader(`{"account_ids":[7],"name":"changed"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK || adminSvc.lastBulkUpdateAccountInput == nil {
		t.Fatalf("primary bulk update status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestCredentialExportMiddlewareRequiresPrimaryAdmin(t *testing.T) {
	for _, tt := range []struct {
		actorID    int64
		authMethod string
		want       int
	}{
		{actorID: 1, want: http.StatusNoContent},
		{actorID: 2, want: http.StatusForbidden},
		{actorID: 1, authMethod: service.AuditAuthMethodAdminAPIKey, want: http.StatusForbidden},
	} {
		settings := service.NewSettingService(approvalGateSettingRepo{primaryID: "1"}, nil)
		poolHandler := NewPoolHandler(service.NewPoolService(nil, nil, nil, nil, settings))
		router := gin.New()
		router.Use(func(c *gin.Context) {
			c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: tt.actorID})
			c.Set("auth_method", tt.authMethod)
			c.Next()
		})
		router.GET("/raw-export", poolHandler.RequirePrimaryAdmin, func(c *gin.Context) { c.Status(http.StatusNoContent) })

		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/raw-export", nil))
		if w.Code != tt.want {
			t.Fatalf("actor %d status=%d body=%s", tt.actorID, w.Code, w.Body.String())
		}
	}
}

func TestNonPrimaryReAuthReturnsAccountCompatiblePendingApproval(t *testing.T) {
	router, adminSvc, repo := newApprovalGateHandler(2)
	w := httptest.NewRecorder()
	body := `{"type":"oauth","credentials":{"access_token":"new-secret"},"extra":{"account_uuid":"uuid-1"}}`
	req := httptest.NewRequest(http.MethodPost, "/accounts/7/apply-oauth-credentials", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if adminSvc.updateAccountCalls != 0 || repo.created == nil {
		t.Fatal("reauthorization bypassed or failed to create approval")
	}
	if strings.Contains(w.Body.String(), "new-secret") || strings.Contains(w.Body.String(), "stored-secret") {
		t.Fatal("pending approval response leaked credentials")
	}
	var envelope struct {
		Data struct {
			ID               int64 `json:"id"`
			ApprovalRequired bool  `json:"approval_required"`
			Approval         struct {
				ID int64 `json:"id"`
			} `json:"approval"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Data.ID != 7 || !envelope.Data.ApprovalRequired || envelope.Data.Approval.ID != 99 {
		t.Fatalf("unexpected compatible approval response: %+v", envelope.Data)
	}
	if repo.created.Changes.CredentialKeys[0] != "access_token" || len(repo.created.Changes.ExtraKeys) != 1 || repo.created.Changes.ExtraKeys[0] != "account_uuid" {
		t.Fatalf("unexpected approval summary: %+v", repo.created.Changes)
	}
}
