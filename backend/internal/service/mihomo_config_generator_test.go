package service

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerateManagedMihomoConfigBuildsMultipleProvidersAndRoutes(t *testing.T) {
	base := map[string]any{
		"secret": "controller-secret", "external-controller": "0.0.0.0:26790", "socks-port": 1080,
		"rules": []string{"MATCH,PROXY"},
	}
	generated, err := generateManagedMihomoConfig(base,
		[]mihomoConfigSubscription{
			{ID: 1, ProviderKey: "alpha", URL: "https://alpha.example/sub", RefreshInterval: 600},
			{ID: 2, ProviderKey: "beta", URL: "https://beta.example/sub", RefreshInterval: 900},
		},
		[]mihomoConfigNode{
			{ID: 11, SubscriptionID: 1, Name: "Hong Kong (01)+"},
			{ID: 12, SubscriptionID: 2, Name: "Tokyo"},
			{ID: 13, SubscriptionID: 2, Name: "Excluded", Excluded: true},
			{ID: 14, SubscriptionID: 1, Name: "📊 剩余流量：995.36 GB"},
			{ID: 15, SubscriptionID: 2, Name: "Traffic Remaining: 100 GB"},
		},
		[]mihomoConfigRoute{
			{ID: 21, Kind: "select", ListenerPort: 26781, NodeIDs: []int64{11}},
			{ID: 22, Kind: "url-test", ListenerPort: 26784, NodeIDs: []int64{11, 12}},
			{ID: 23, Kind: "load-balance", ListenerPort: 26785, NodeIDs: []int64{11, 12, 14, 15}, Selector: json.RawMessage(`{"strategy":"consistent-hashing"}`)},
		})
	require.NoError(t, err)
	providers, ok := generated["proxy-providers"].(map[string]any)
	require.True(t, ok)
	if len(providers) != 2 {
		t.Fatalf("providers = %#v", generated["proxy-providers"])
	}
	groups, ok := generated["proxy-groups"].([]any)
	require.True(t, ok)
	require.Len(t, groups, 3)
	selectGroup, ok := groups[0].(map[string]any)
	require.True(t, ok)
	urlTestGroup, ok := groups[1].(map[string]any)
	require.True(t, ok)
	loadBalanceGroup, ok := groups[2].(map[string]any)
	require.True(t, ok)
	require.Equal(t, []string{"alpha"}, selectGroup["use"])
	require.Equal(t, `^(?:\[alpha\] Hong Kong \(01\)\+)$`, selectGroup["filter"])
	require.Equal(t, []string{"alpha", "beta"}, urlTestGroup["use"])
	require.Equal(t, `^(?:\[alpha\] Hong Kong \(01\)\+|\[beta\] Tokyo)$`, urlTestGroup["filter"])
	for _, group := range []map[string]any{selectGroup, urlTestGroup, loadBalanceGroup} {
		require.NotContains(t, group, "proxies")
		require.NotContains(t, group["filter"], "Excluded")
	}
	if urlTestGroup["type"] != "url-test" || loadBalanceGroup["strategy"] != "consistent-hashing" {
		t.Fatalf("groups = %#v", groups)
	}
	listeners, ok := generated["listeners"].([]any)
	require.True(t, ok)
	require.Len(t, listeners, 3)
	urlTestListener, ok := listeners[1].(map[string]any)
	require.True(t, ok)
	if urlTestListener["port"] != 26784 {
		t.Fatalf("listeners = %#v", listeners)
	}
	if _, exists := generated["socks-port"]; exists {
		t.Fatal("legacy socks-port was not removed")
	}
	rules, ok := generated["rules"].([]string)
	require.True(t, ok)
	if len(rules) != 1 || rules[0] != "MATCH,DIRECT" {
		t.Fatalf("rules = %#v", rules)
	}
	if generated["secret"] != "controller-secret" || generated["external-controller"] != "0.0.0.0:26790" {
		t.Fatalf("base controller config was not preserved: %#v", generated)
	}
}

func TestMihomoSubscriptionMetadataNode(t *testing.T) {
	for _, name := range []string{"剩余流量：995.36 GB", "📅 套餐到期: 长期有效", "Traffic Remaining: 100 GB", "Expiration: 2026-12-31"} {
		require.True(t, isMihomoSubscriptionMetadataNode(name), name)
	}
	for _, name := range []string{"剩余流量专线", "Traffic Relay", "Hong Kong"} {
		require.False(t, isMihomoSubscriptionMetadataNode(name), name)
	}
}

func TestGenerateManagedMihomoConfigRejectsPortCollision(t *testing.T) {
	_, err := generateManagedMihomoConfig(map[string]any{},
		[]mihomoConfigSubscription{{ID: 1, ProviderKey: "one", URL: "https://one.example/sub"}},
		[]mihomoConfigNode{{ID: 1, SubscriptionID: 1, Name: "Node"}},
		[]mihomoConfigRoute{
			{ID: 1, Kind: "select", ListenerPort: 26781, NodeIDs: []int64{1}},
			{ID: 2, Kind: "fallback", ListenerPort: 26781, NodeIDs: []int64{1}},
		})
	if err == nil {
		t.Fatal("expected duplicate listener port to be rejected")
	}
}
