package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

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

func TestPoolApprovalSummaryOmitsUnchangedFieldsAndExtraKeys(t *testing.T) {
	priority := 10
	contributorID := int64(7)
	expiresAt := time.Unix(1_800_000_000, 0)
	expiresAtUnix := expiresAt.Unix()
	summary := buildPoolApprovalSummary(
		&Account{Name: "same", Priority: priority, ExpiresAt: &expiresAt, Extra: map[string]any{"same": true, "changed": "before", "removed": true}},
		&PoolApprovalAccountState{ContributorUserID: &contributorID},
		nil,
		PoolApprovalPayload{AccountUpdate: &UpdateAccountInput{
			Name: "same", Priority: &priority, ExpiresAt: &expiresAtUnix, Extra: map[string]any{"same": true, "changed": "after"},
		}, PoolUpdate: &UpdatePoolAccountInput{ContributorUserID: &contributorID}},
	)
	if len(summary.Fields) != 0 {
		t.Fatalf("unchanged fields should be omitted: %#v", summary.Fields)
	}
	if len(summary.ExtraKeys) != 2 || summary.ExtraKeys[0] != "changed" || summary.ExtraKeys[1] != "removed" {
		t.Fatalf("unexpected extra key summary: %#v", summary.ExtraKeys)
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
	beforeProvider := "ab-provider-one-yz"
	afterProvider := "ab-provider-two-yz"
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
	if _, ok := summary.Fields["provider_identity"]; !ok {
		t.Fatal("a provider identity change with the same mask must remain visible")
	}
}

func TestPoolApprovalBusinessSummaryGroupsChangesAndRedactsCredentials(t *testing.T) {
	changes := PoolApprovalChangeSummary{
		Fields: map[string]PoolApprovalValueChange{
			"status":               {Before: "active", After: "disabled"},
			"cost_original_amount": {Before: "10.00", After: "12.00"},
		},
		CredentialKeys: []string{"access_token"},
	}
	summary := buildPoolApprovalBusinessSummary(PoolApprovalUpdateAccount, &Account{ID: 9, Name: "pool-account"}, changes, nil)
	if !summary.HighRisk || summary.Object.ID != 9 || summary.Object.Name != "pool-account" {
		t.Fatalf("unexpected business object: %#v", summary)
	}
	if strings.Join(summary.Scope, ",") != "scheduling,cost_settlement,credentials" {
		t.Fatalf("unexpected scope order: %#v", summary.Scope)
	}
	credential := summary.Groups[len(summary.Groups)-1].Items[0]
	if !credential.Sensitive || credential.Before != nil || credential.After != "updated" {
		t.Fatalf("credential summary must expose only the key and update marker: %#v", credential)
	}
}

func TestPoolApprovalBusinessSummaryIncludesDeleteImpact(t *testing.T) {
	summary := buildPoolApprovalBusinessSummary(PoolApprovalDeleteAccount, &Account{ID: 7, Name: "doomed"}, PoolApprovalChangeSummary{}, &PoolAccountDeleteImpact{
		Accounts: 2, CredentialKeys: 4, SchedulingRecords: 1, CostEntries: 3,
		Settlements: 2, SettlementAccountCosts: 8, SettlementAccountLines: 9,
		MixedSettlements: 1, EmptySettlements: 1, PurchaseSources: 2,
		GroupLinks: 5, LifecycleEvents: 6, UsageRecords: 7,
	})
	if !summary.HighRisk || summary.Action != PoolApprovalDeleteAccount {
		t.Fatalf("delete must be high risk: %#v", summary)
	}
	if len(summary.Impacts) != 13 || summary.Impacts[0].Key != "accounts" || summary.Impacts[0].Count != 2 ||
		summary.Impacts[5].Key != "settlement_account_costs" || summary.Impacts[5].Count != 8 ||
		summary.Impacts[8].Key != "empty_settlements" || summary.Impacts[8].Count != 1 ||
		summary.Impacts[12].Key != "retained_usage_records" || summary.Impacts[12].Count != 7 {
		t.Fatalf("unexpected delete impact: %#v", summary.Impacts)
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
		t.Fatalf("self decision must still require a current bypass: %v", err)
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
	proxyUpdate := &UpdateProxyInput{Name: "proxy", Protocol: "socks5", Host: "proxy.example", Port: 1080, Status: StatusActive, FallbackMode: FallbackModeNone}
	if err := validateApprovalPayload(PoolApprovalUpdateProxy, PoolApprovalPayload{ProxyUpdate: proxyUpdate}); err != nil {
		t.Fatalf("valid proxy update rejected: %v", err)
	}
	if err := validateApprovalPayload(PoolApprovalUpdateProxy, PoolApprovalPayload{ProxyUpdate: proxyUpdate, Reauthorize: true}); infraerrors.Reason(err) != "INVALID_PROXY_APPROVAL_PAYLOAD" {
		t.Fatalf("unexpected mixed proxy payload result: %v", err)
	}
}

func TestProxyExportApprovalRevisionTracksRevealedConnection(t *testing.T) {
	proxies := []Proxy{{ID: 2, Host: "second.example", Port: 1080}, {ID: 1, Host: "first.example", Port: 1080, Password: "before"}}
	base, err := proxyExportApprovalRevision(append([]Proxy(nil), proxies...))
	if err != nil {
		t.Fatal(err)
	}
	proxies[1].Password = "after"
	changed, err := proxyExportApprovalRevision(append([]Proxy(nil), proxies...))
	if err != nil {
		t.Fatal(err)
	}
	if base == changed {
		t.Fatal("a changed proxy connection must stale an approved export")
	}
}

func TestProxyUpdateSummaryDoesNotRevealCurrentEndpoint(t *testing.T) {
	summary := buildProxyApprovalSummary(PoolApprovalUpdateProxy, &Proxy{
		ID: 1, Name: "proxy", Protocol: "socks5", Host: "current.example", Port: 1080,
	}, &UpdateProxyInput{
		Name: "proxy", Protocol: "socks5", Host: "next.example", Port: 1080,
		Status: StatusActive, FallbackMode: FallbackModeNone,
	}, 0)
	encoded, err := json.Marshal(summary)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "current.example") || !strings.Contains(string(encoded), "next.example") {
		t.Fatalf("summary must hide the current endpoint and show the requested endpoint: %s", encoded)
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

func TestApprovalPrimaryBypassUsesFixedPrimaryAdmin(t *testing.T) {
	tests := []struct {
		name  string
		input CreatePoolApprovalInput
		want  bool
	}{
		{name: "primary mihomo", input: CreatePoolApprovalInput{ActionType: PoolApprovalUpdateMihomo, RequesterID: 42, RequirePeerReview: true}, want: true},
		{name: "non-primary mihomo", input: CreatePoolApprovalInput{ActionType: PoolApprovalUpdateMihomo, RequesterID: 41, RequirePeerReview: true}},
		{name: "primary account", input: CreatePoolApprovalInput{ActionType: PoolApprovalUpdateAccount, RequesterID: 42}, want: true},
		{name: "forced account peer review", input: CreatePoolApprovalInput{ActionType: PoolApprovalUpdateAccount, RequesterID: 42, RequirePeerReview: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool := NewPoolService(nil, nil, nil, nil, NewSettingService(primaryAdminSettingRepo{value: "42"}, nil))
			got := pool.approvalPrimaryBypass(context.Background(), tt.input)
			if got != tt.want {
				t.Fatalf("approvalPrimaryBypass() = %v, want %v", got, tt.want)
			}
		})
	}
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
