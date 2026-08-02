package service

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

var mihomoProviderKeyPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

type mihomoConfigSubscription struct {
	ID              int64
	Name            string
	ProviderKey     string
	URL             string
	RefreshInterval int
}

type mihomoConfigNode struct {
	ID             int64
	SubscriptionID int64
	Name           string
	Excluded       bool
}

type mihomoConfigRoute struct {
	ID           int64
	Name         string
	Kind         string
	ListenerPort int
	Selector     json.RawMessage
	NodeIDs      []int64
}

func generateManagedMihomoConfig(base map[string]any, subscriptions []mihomoConfigSubscription, nodes []mihomoConfigNode, routes []mihomoConfigRoute) (map[string]any, error) {
	providers := make(map[string]any, len(subscriptions))
	subscriptionByID := make(map[int64]mihomoConfigSubscription, len(subscriptions))
	for _, subscription := range subscriptions {
		key := strings.TrimSpace(subscription.ProviderKey)
		if subscription.ID <= 0 || key == "" || !mihomoProviderKeyPattern.MatchString(key) || strings.TrimSpace(subscription.URL) == "" {
			return nil, fmt.Errorf("invalid Mihomo subscription %d", subscription.ID)
		}
		if _, exists := providers[key]; exists {
			return nil, fmt.Errorf("duplicate Mihomo provider key %q", key)
		}
		interval := subscription.RefreshInterval
		if interval < 300 {
			interval = 300
		}
		providers[key] = map[string]any{
			"type":     "http",
			"url":      subscription.URL,
			"path":     "./providers/" + key + ".yaml",
			"interval": interval,
			"health-check": map[string]any{
				"enable":   true,
				"url":      "https://www.gstatic.com/generate_204",
				"interval": 300,
			},
			"override": map[string]any{"additional-prefix": "[" + key + "] "},
		}
		subscription.ProviderKey = key
		subscriptionByID[subscription.ID] = subscription
	}

	nodeNameByID := make(map[int64]string, len(nodes))
	for _, node := range nodes {
		subscription, ok := subscriptionByID[node.SubscriptionID]
		if node.ID <= 0 || !ok || node.Excluded || strings.TrimSpace(node.Name) == "" {
			continue
		}
		nodeNameByID[node.ID] = "[" + subscription.ProviderKey + "] " + strings.TrimSpace(node.Name)
	}

	groups := make([]any, 0, len(routes))
	listeners := make([]any, 0, len(routes))
	ports := make(map[int]int64, len(routes))
	for _, route := range routes {
		if route.ID <= 0 || route.ListenerPort < 1 || route.ListenerPort > 65535 {
			return nil, fmt.Errorf("invalid Mihomo route %d", route.ID)
		}
		if otherID, exists := ports[route.ListenerPort]; exists {
			return nil, fmt.Errorf("Mihomo routes %d and %d share listener port %d", otherID, route.ID, route.ListenerPort)
		}
		ports[route.ListenerPort] = route.ID

		proxyNames := make([]string, 0, len(route.NodeIDs))
		seen := make(map[string]struct{}, len(route.NodeIDs))
		for _, nodeID := range route.NodeIDs {
			name, ok := nodeNameByID[nodeID]
			if !ok {
				continue
			}
			if _, exists := seen[name]; exists {
				continue
			}
			seen[name] = struct{}{}
			proxyNames = append(proxyNames, name)
		}
		if len(proxyNames) == 0 {
			return nil, fmt.Errorf("Mihomo route %d has no usable nodes", route.ID)
		}
		groupName := fmt.Sprintf("SUB2API-ROUTE-%d", route.ID)
		group, err := buildMihomoRouteGroup(groupName, route.Kind, proxyNames, route.Selector)
		if err != nil {
			return nil, fmt.Errorf("Mihomo route %d: %w", route.ID, err)
		}
		groups = append(groups, group)
		listeners = append(listeners, map[string]any{
			"name":   fmt.Sprintf("sub2api-route-%d", route.ID),
			"type":   "socks",
			"listen": "0.0.0.0",
			"port":   route.ListenerPort,
			"proxy":  groupName,
		})
	}

	managed := cloneYAMLMap(base)
	managed["proxy-providers"] = providers
	managed["proxy-groups"] = groups
	managed["listeners"] = listeners
	managed["mode"] = "rule"
	managed["rules"] = []string{"MATCH,DIRECT"}
	for _, legacyPortKey := range []string{"port", "socks-port", "mixed-port", "redir-port", "tproxy-port"} {
		delete(managed, legacyPortKey)
	}
	return managed, nil
}

func buildMihomoRouteGroup(name, kind string, proxies []string, selector json.RawMessage) (map[string]any, error) {
	kind = strings.ToLower(strings.TrimSpace(kind))
	switch kind {
	case "dedicated", "directional", "select":
		return map[string]any{"name": name, "type": "select", "proxies": proxies}, nil
	case "automatic", "latency", "url-test":
		return map[string]any{"name": name, "type": "url-test", "proxies": proxies, "url": "https://www.gstatic.com/generate_204", "interval": 300, "tolerance": 50, "lazy": true}, nil
	case "fallback":
		return map[string]any{"name": name, "type": "fallback", "proxies": proxies, "url": "https://www.gstatic.com/generate_204", "interval": 300, "lazy": true}, nil
	case "dynamic", "load-balance":
		strategy := "round-robin"
		if len(selector) > 0 {
			var options struct {
				Strategy string `json:"strategy"`
			}
			if err := json.Unmarshal(selector, &options); err != nil {
				return nil, fmt.Errorf("invalid selector: %w", err)
			}
			if options.Strategy != "" {
				strategy = options.Strategy
			}
		}
		if strategy != "round-robin" && strategy != "consistent-hashing" && strategy != "sticky-sessions" {
			return nil, fmt.Errorf("unsupported load-balance strategy %q", strategy)
		}
		return map[string]any{"name": name, "type": "load-balance", "proxies": proxies, "url": "https://www.gstatic.com/generate_204", "interval": 300, "strategy": strategy, "lazy": true}, nil
	default:
		return nil, fmt.Errorf("unsupported route kind %q", kind)
	}
}

func cloneYAMLMap(source map[string]any) map[string]any {
	cloned := make(map[string]any, len(source)+3)
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}
