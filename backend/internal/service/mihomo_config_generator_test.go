package service

import (
	"encoding/json"
	"testing"
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
			{ID: 11, SubscriptionID: 1, Name: "Hong Kong"},
			{ID: 12, SubscriptionID: 2, Name: "Tokyo"},
			{ID: 13, SubscriptionID: 2, Name: "Excluded", Excluded: true},
		},
		[]mihomoConfigRoute{
			{ID: 21, Kind: "select", ListenerPort: 26781, NodeIDs: []int64{11}},
			{ID: 22, Kind: "url-test", ListenerPort: 26784, NodeIDs: []int64{11, 12}},
			{ID: 23, Kind: "load-balance", ListenerPort: 26785, NodeIDs: []int64{11, 12}, Selector: json.RawMessage(`{"strategy":"consistent-hashing"}`)},
		})
	if err != nil {
		t.Fatal(err)
	}
	if len(generated["proxy-providers"].(map[string]any)) != 2 {
		t.Fatalf("providers = %#v", generated["proxy-providers"])
	}
	groups := generated["proxy-groups"].([]any)
	if len(groups) != 3 || groups[1].(map[string]any)["type"] != "url-test" || groups[2].(map[string]any)["strategy"] != "consistent-hashing" {
		t.Fatalf("groups = %#v", groups)
	}
	listeners := generated["listeners"].([]any)
	if len(listeners) != 3 || listeners[1].(map[string]any)["port"] != 26784 {
		t.Fatalf("listeners = %#v", listeners)
	}
	if _, exists := generated["socks-port"]; exists {
		t.Fatal("legacy socks-port was not removed")
	}
	if rules := generated["rules"].([]string); len(rules) != 1 || rules[0] != "MATCH,DIRECT" {
		t.Fatalf("rules = %#v", rules)
	}
	if generated["secret"] != "controller-secret" || generated["external-controller"] != "0.0.0.0:26790" {
		t.Fatalf("base controller config was not preserved: %#v", generated)
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
