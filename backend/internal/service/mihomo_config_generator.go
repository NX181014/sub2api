package service

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

var mihomoProviderKeyPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

func isMihomoSubscriptionMetadataNode(name string) bool {
	name = strings.NewReplacer("：", ":", "﹕", ":").Replace(strings.TrimSpace(name))
	name = strings.ToLower(strings.TrimLeftFunc(name, func(r rune) bool {
		return unicode.IsSpace(r) || unicode.IsPunct(r) || unicode.IsSymbol(r)
	}))
	for _, prefix := range []string{
		"剩余流量:", "流量剩余:", "套餐到期:", "到期时间:", "过期时间:", "有效期:",
		"重置倒计时:", "下次重置:", "官网:", "remaining traffic:", "traffic remaining:",
		"expires:", "expiration:", "reset:",
	} {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

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
	nodeProviderByID := make(map[int64]string, len(nodes))
	for _, node := range nodes {
		subscription, ok := subscriptionByID[node.SubscriptionID]
		if node.ID <= 0 || !ok || node.Excluded || strings.TrimSpace(node.Name) == "" || isMihomoSubscriptionMetadataNode(node.Name) {
			continue
		}
		nodeNameByID[node.ID] = "[" + subscription.ProviderKey + "] " + strings.TrimSpace(node.Name)
		nodeProviderByID[node.ID] = subscription.ProviderKey
	}

	groups := make([]any, 0, len(routes))
	listeners := make([]any, 0, len(routes))
	ports := make(map[int]int64, len(routes))
	for _, route := range routes {
		if route.ID <= 0 || route.ListenerPort < 1 || route.ListenerPort > 65535 {
			return nil, fmt.Errorf("invalid Mihomo route %d", route.ID)
		}
		if otherID, exists := ports[route.ListenerPort]; exists {
			return nil, fmt.Errorf("mihomo routes %d and %d share listener port %d", otherID, route.ID, route.ListenerPort)
		}
		ports[route.ListenerPort] = route.ID

		proxyNames := make([]string, 0, len(route.NodeIDs))
		providerKeys := make([]string, 0, len(route.NodeIDs))
		seenNames := make(map[string]struct{}, len(route.NodeIDs))
		seenProviders := make(map[string]struct{}, len(route.NodeIDs))
		for _, nodeID := range route.NodeIDs {
			name, ok := nodeNameByID[nodeID]
			if !ok {
				continue
			}
			if providerKey := nodeProviderByID[nodeID]; providerKey != "" {
				if _, exists := seenProviders[providerKey]; !exists {
					seenProviders[providerKey] = struct{}{}
					providerKeys = append(providerKeys, providerKey)
				}
			}
			if _, exists := seenNames[name]; !exists {
				seenNames[name] = struct{}{}
				proxyNames = append(proxyNames, name)
			}
		}
		if len(proxyNames) == 0 {
			return nil, fmt.Errorf("mihomo route %d has no usable nodes", route.ID)
		}
		groupName := fmt.Sprintf("SUB2API-ROUTE-%d", route.ID)
		filters := make([]string, 0, len(proxyNames))
		for _, name := range proxyNames {
			filters = append(filters, regexp.QuoteMeta(name))
		}
		group, err := buildMihomoRouteGroup(groupName, route.Kind, providerKeys, "^(?:"+strings.Join(filters, "|")+")$", route.Selector)
		if err != nil {
			return nil, fmt.Errorf("mihomo route %d: %w", route.ID, err)
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

func buildMihomoRouteGroup(name, kind string, providers []string, filter string, selector json.RawMessage) (map[string]any, error) {
	group := map[string]any{"name": name, "use": providers, "filter": filter}
	kind = strings.ToLower(strings.TrimSpace(kind))
	switch kind {
	case "dedicated", "directional", "select":
		group["type"] = "select"
		return group, nil
	case "automatic", "latency", "url-test":
		group["type"], group["url"], group["interval"], group["tolerance"], group["lazy"] = "url-test", "https://www.gstatic.com/generate_204", 300, 50, true
		return group, nil
	case "fallback":
		group["type"], group["url"], group["interval"], group["lazy"] = "fallback", "https://www.gstatic.com/generate_204", 300, true
		return group, nil
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
		group["type"], group["url"], group["interval"], group["strategy"], group["lazy"] = "load-balance", "https://www.gstatic.com/generate_204", 300, strategy, true
		return group, nil
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
