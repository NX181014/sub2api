package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

func TestPoolApprovalRevisionRejectsConcurrentCredentialReplacement(t *testing.T) {
	account := &Account{
		Name:        "before",
		Credentials: map[string]any{"access_token": "secret-a", "unrelated": "one"},
		Extra:       map[string]any{"managed": "same", "codex_usage_5h_percent": 10},
	}
	pool := &PoolApprovalAccountState{}
	payload := PoolApprovalPayload{AccountUpdate: &UpdateAccountInput{
		Name:        "after",
		Credentials: map[string]any{"access_token": "secret-new"},
		Extra:       map[string]any{"managed": "new"},
	}}

	base, err := poolApprovalRevision(account, pool, payload)
	if err != nil {
		t.Fatal(err)
	}
	account.Credentials["unrelated"] = "two"
	account.Extra["codex_usage_5h_percent"] = 90
	afterRuntimeUpdate, err := poolApprovalRevision(account, pool, payload)
	if err != nil {
		t.Fatal(err)
	}
	if base == afterRuntimeUpdate {
		t.Fatal("a concurrent credential change must stale a replacement approval")
	}

	account.Credentials["access_token"] = "secret-b"
	afterCredentialUpdate, err := poolApprovalRevision(account, pool, payload)
	if err != nil {
		t.Fatal(err)
	}
	if base == afterCredentialUpdate {
		t.Fatal("a requested credential key change must stale an approval")
	}
}

func TestPoolApprovalSummaryListsOnlyChangedCredentialKeys(t *testing.T) {
	summary := buildPoolApprovalSummary(
		&Account{Credentials: map[string]any{"access_token": "preserved", "base_url": "before", "removed": true}},
		&PoolApprovalAccountState{},
		nil,
		PoolApprovalPayload{AccountUpdate: &UpdateAccountInput{Credentials: map[string]any{
			"base_url": "after",
		}}},
	)
	want := []string{"base_url", "removed"}
	if len(summary.CredentialKeys) != len(want) {
		t.Fatalf("unexpected changed credential keys: %#v", summary.CredentialKeys)
	}
	for index := range want {
		if summary.CredentialKeys[index] != want[index] {
			t.Fatalf("unexpected changed credential keys: %#v", summary.CredentialKeys)
		}
	}
}

func TestPoolApprovalRevisionTracksOnlyMergedExtraKeys(t *testing.T) {
	account := &Account{Extra: map[string]any{"account_uuid": "before", "runtime": "one"}}
	pool := &PoolApprovalAccountState{}
	payload := PoolApprovalPayload{ExtraMerge: map[string]any{"account_uuid": "after"}}
	base, err := poolApprovalRevision(account, pool, payload)
	if err != nil {
		t.Fatal(err)
	}
	account.Extra["runtime"] = "two"
	unrelated, err := poolApprovalRevision(account, pool, payload)
	if err != nil {
		t.Fatal(err)
	}
	if base != unrelated {
		t.Fatal("an unrelated extra update must not stale a merge approval")
	}
	account.Extra["account_uuid"] = "changed"
	relevant, err := poolApprovalRevision(account, pool, payload)
	if err != nil {
		t.Fatal(err)
	}
	if base == relevant {
		t.Fatal("a requested merge key change must stale an approval")
	}
}

func TestPoolApprovalSummaryRedactsCredentialAndProviderValues(t *testing.T) {
	beforeProvider := "provider-secret-before"
	afterProvider := "provider-secret-after"
	summary := buildPoolApprovalSummary(
		&Account{Credentials: map[string]any{"access_token": "old-secret"}},
		&PoolApprovalAccountState{ProviderIdentity: &beforeProvider},
		nil,
		PoolApprovalPayload{
			AccountUpdate: &UpdateAccountInput{Credentials: map[string]any{"access_token": "new-secret"}},
			PoolUpdate:    &UpdatePoolAccountInput{ProviderIdentity: &afterProvider},
		},
	)
	raw, err := json.Marshal(summary)
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	for _, secret := range []string{"old-secret", "new-secret", beforeProvider, afterProvider} {
		if strings.Contains(text, secret) {
			t.Fatalf("approval summary leaked %q", secret)
		}
	}
	if len(summary.CredentialKeys) != 1 || summary.CredentialKeys[0] != "access_token" {
		t.Fatalf("unexpected credential key summary: %#v", summary.CredentialKeys)
	}
}

func TestPoolApprovalDecisionRequiresDifferentAdminExceptPrimaryBypass(t *testing.T) {
	item := &PoolApproval{RequestedByUserID: 10}
	if err := validatePoolApprovalDecisionActor(item, 11, false); err != nil {
		t.Fatalf("different administrator should be allowed: %v", err)
	}
	if err := validatePoolApprovalDecisionActor(item, 10, false); infraerrors.Reason(err) != "APPROVAL_SELF_DECISION_FORBIDDEN" {
		t.Fatalf("unexpected self-decision result: %v", err)
	}
	item.PrimaryBypass = true
	if err := validatePoolApprovalDecisionActor(item, 10, true); err != nil {
		t.Fatalf("fixed primary administrator bypass should be allowed: %v", err)
	}
	if err := validatePoolApprovalDecisionActor(item, 10, false); infraerrors.Reason(err) != "APPROVAL_SELF_DECISION_FORBIDDEN" {
		t.Fatalf("stored bypass must still require the current fixed primary identity: %v", err)
	}
}

func TestValidatePoolApprovalPayload(t *testing.T) {
	if err := validateApprovalPayload(PoolApprovalViewCredential, PoolApprovalPayload{}); err != nil {
		t.Fatal(err)
	}
	if err := validateApprovalPayload(PoolApprovalUpdateAccount, PoolApprovalPayload{}); infraerrors.Reason(err) != "EMPTY_APPROVAL_UPDATE" {
		t.Fatalf("unexpected empty update result: %v", err)
	}
	if err := validateApprovalPayload(PoolApprovalViewCredential, PoolApprovalPayload{AccountUpdate: &UpdateAccountInput{Name: "x"}}); infraerrors.Reason(err) != "INVALID_CREDENTIAL_APPROVAL_PAYLOAD" {
		t.Fatalf("unexpected credential payload result: %v", err)
	}
	if err := validateApprovalPayload(PoolApprovalViewCredential, PoolApprovalPayload{Reauthorize: true}); infraerrors.Reason(err) != "INVALID_CREDENTIAL_APPROVAL_PAYLOAD" {
		t.Fatalf("unexpected reauthorization credential payload result: %v", err)
	}
	deleteOptions := &AccountDeleteOptions{}
	if err := validateApprovalPayload(PoolApprovalDeleteAccount, PoolApprovalPayload{DeleteOptions: deleteOptions}); err != nil {
		t.Fatalf("valid delete payload rejected: %v", err)
	}
	if err := validateApprovalPayload(PoolApprovalDeleteAccount, PoolApprovalPayload{}); infraerrors.Reason(err) != "INVALID_DELETE_APPROVAL_PAYLOAD" {
		t.Fatalf("unexpected empty delete payload result: %v", err)
	}
	if err := validateApprovalPayload(PoolApprovalUpdateAccount, PoolApprovalPayload{AccountUpdate: &UpdateAccountInput{Name: "x"}, DeleteOptions: deleteOptions}); infraerrors.Reason(err) != "INVALID_UPDATE_APPROVAL_PAYLOAD" {
		t.Fatalf("unexpected mixed update/delete payload result: %v", err)
	}
}

type primaryAdminSettingRepo struct {
	SettingRepository
	value string
	err   error
}

func (r primaryAdminSettingRepo) GetValue(context.Context, string) (string, error) {
	return r.value, r.err
}

func TestSettingServiceIsPrimaryAdminFailsClosed(t *testing.T) {
	tests := []struct {
		name   string
		repo   SettingRepository
		userID int64
		want   bool
	}{
		{name: "fixed primary", repo: primaryAdminSettingRepo{value: "42"}, userID: 42, want: true},
		{name: "different administrator", repo: primaryAdminSettingRepo{value: "42"}, userID: 41},
		{name: "malformed setting", repo: primaryAdminSettingRepo{value: "not-an-id"}, userID: 42},
		{name: "missing setting", repo: primaryAdminSettingRepo{err: errors.New("missing")}, userID: 42},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewSettingService(tt.repo, nil).IsPrimaryAdmin(context.Background(), tt.userID); got != tt.want {
				t.Fatalf("IsPrimaryAdmin() = %v, want %v", got, tt.want)
			}
		})
	}
}
