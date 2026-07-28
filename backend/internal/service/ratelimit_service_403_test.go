//go:build unit

package service

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type runtimeBlockRecorder struct {
	accounts   []*Account
	until      []time.Time
	reasons    []string
	clearedIDs []int64
}

type automaticBanRecord struct {
	accountID  int64
	occurredAt time.Time
	reason     string
}

type automaticBanAccountRepoStub struct {
	*rateLimitAccountRepoStub
	records []automaticBanRecord
}

func (r *automaticBanAccountRepoStub) RecordAutomaticBan(_ context.Context, accountID int64, occurredAt time.Time, reason string) (bool, error) {
	r.records = append(r.records, automaticBanRecord{accountID: accountID, occurredAt: occurredAt, reason: reason})
	return true, nil
}

func (r *runtimeBlockRecorder) BlockAccountScheduling(account *Account, until time.Time, reason string) {
	r.accounts = append(r.accounts, account)
	r.until = append(r.until, until)
	r.reasons = append(r.reasons, reason)
}

func (r *runtimeBlockRecorder) ClearAccountSchedulingBlock(accountID int64) {
	r.clearedIDs = append(r.clearedIDs, accountID)
}

func TestRateLimitService_HandleUpstreamError_OpenAI403FirstHitTempUnschedulable(t *testing.T) {
	repo := &rateLimitAccountRepoStub{}
	counter := &openAI403CounterCacheStub{counts: []int64{1}}
	blocker := &runtimeBlockRecorder{}
	service := NewRateLimitService(repo, nil, &config.Config{}, nil, nil)
	service.SetOpenAI403CounterCache(counter)
	service.SetAccountRuntimeBlocker(blocker)
	account := &Account{
		ID:       301,
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
	}

	shouldDisable := service.HandleUpstreamError(
		context.Background(),
		account,
		http.StatusForbidden,
		http.Header{},
		[]byte(`{"error":{"message":"temporary edge rejection"}}`),
	)

	require.True(t, shouldDisable)
	require.Equal(t, 0, repo.setErrorCalls)
	require.Equal(t, 1, repo.tempCalls)
	require.Contains(t, repo.lastTempReason, "temporary edge rejection")
	require.Contains(t, repo.lastTempReason, "(1/3)")
	require.Len(t, blocker.accounts, 1)
	require.Equal(t, account.ID, blocker.accounts[0].ID)
	require.Equal(t, "openai_403_temp", blocker.reasons[0])
	require.True(t, blocker.until[0].After(time.Now()))
}

func TestRateLimitService_HandleUpstreamError_OpenAI403ThresholdDisables(t *testing.T) {
	repo := &automaticBanAccountRepoStub{rateLimitAccountRepoStub: &rateLimitAccountRepoStub{}}
	counter := &openAI403CounterCacheStub{counts: []int64{3}}
	service := NewRateLimitService(repo, nil, &config.Config{}, nil, nil)
	service.SetOpenAI403CounterCache(counter)
	account := &Account{
		ID:       302,
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
	}

	shouldDisable := service.HandleUpstreamError(
		context.Background(),
		account,
		http.StatusForbidden,
		http.Header{},
		[]byte(`{"error":{"message":"workspace forbidden by policy"}}`),
	)

	require.True(t, shouldDisable)
	require.Equal(t, 1, repo.setErrorCalls)
	require.Equal(t, 0, repo.tempCalls)
	require.Contains(t, repo.lastErrorMsg, "workspace forbidden by policy")
	require.Contains(t, repo.lastErrorMsg, "consecutive_403=3/3")
	require.Len(t, repo.records, 1)
	require.Equal(t, int64(302), repo.records[0].accountID)
	require.Equal(t, "openai_consecutive_403", repo.records[0].reason)
	require.Equal(t, time.UTC, repo.records[0].occurredAt.Location())
}

func TestRateLimitService_AutomaticBanUsesConfirmedSignalsOnly(t *testing.T) {
	tests := []struct {
		name       string
		account    *Account
		statusCode int
		body       string
		wantReason string
	}{
		{
			name:       "antigravity violation",
			account:    &Account{ID: 401, Platform: PlatformAntigravity, Type: AccountTypeOAuth},
			statusCode: http.StatusForbidden,
			body:       `{"error":{"message":"terms of service violation"}}`,
			wantReason: "terms_of_service_violation",
		},
		{
			name:       "explicit suspension",
			account:    &Account{ID: 402, Platform: PlatformGemini, Type: AccountTypeOAuth},
			statusCode: http.StatusForbidden,
			body:       `{"error":{"message":"account suspended"}}`,
			wantReason: "account_suspended",
		},
		{
			name:       "validation is not a ban",
			account:    &Account{ID: 403, Platform: PlatformAntigravity, Type: AccountTypeOAuth},
			statusCode: http.StatusForbidden,
			body:       `{"error":{"message":"validation_required","validation_url":"https://example.invalid"}}`,
		},
		{
			name:       "401 is not a ban",
			account:    &Account{ID: 404, Platform: PlatformGemini, Type: AccountTypeAPIKey},
			statusCode: http.StatusUnauthorized,
			body:       `{"error":{"message":"account suspended"}}`,
		},
		{
			name:       "402 is not a ban",
			account:    &Account{ID: 405, Platform: PlatformOpenAI, Type: AccountTypeOAuth},
			statusCode: http.StatusPaymentRequired,
			body:       `{"detail":{"code":"deactivated_workspace"}}`,
		},
		{
			name:       "server error is not a ban",
			account:    &Account{ID: 406, Platform: PlatformGemini, Type: AccountTypeOAuth},
			statusCode: http.StatusBadGateway,
			body:       `{"error":{"message":"account suspended"}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := &rateLimitAccountRepoStub{}
			repo := &automaticBanAccountRepoStub{rateLimitAccountRepoStub: base}
			svc := NewRateLimitService(repo, nil, &config.Config{}, nil, nil)
			before := time.Now().UTC()

			svc.HandleUpstreamError(context.Background(), tt.account, tt.statusCode, http.Header{}, []byte(tt.body))

			if tt.wantReason == "" {
				require.Empty(t, repo.records)
				return
			}
			require.Len(t, repo.records, 1)
			require.Equal(t, tt.account.ID, repo.records[0].accountID)
			require.Equal(t, tt.wantReason, repo.records[0].reason)
			require.False(t, repo.records[0].occurredAt.Before(before))
			require.False(t, repo.records[0].occurredAt.After(time.Now().UTC()))
		})
	}
}
