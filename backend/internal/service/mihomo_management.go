package service

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

type MihomoManagedSubscriptionInput struct {
	Name                   string
	SubscriptionURL        string
	Enabled                bool
	RefreshIntervalMinutes int
}

type MihomoManagedRouteInput struct {
	Name            string
	Kind            string
	SubscriptionIDs []int64
	NodeIDs         []int64
	Enabled         bool
}

type MihomoLegacyImportPreview struct {
	Available            bool                  `json:"available"`
	AlreadyImported      bool                  `json:"already_imported"`
	ProviderName         string                `json:"provider_name,omitempty"`
	SubscriptionHost     string                `json:"subscription_host,omitempty"`
	NodeCount            int64                 `json:"node_count"`
	RouteCount           int64                 `json:"route_count"`
	AffectedAccountCount int64                 `json:"affected_account_count"`
	Routes               []MihomoApprovalRoute `json:"routes"`
}

func isManagedMihomoApprovalKind(kind string) bool {
	switch kind {
	case MihomoApprovalSubscriptionCreate, MihomoApprovalSubscriptionUpdate,
		MihomoApprovalSubscriptionDelete, MihomoApprovalSubscriptionRefresh,
		MihomoApprovalRouteCreate, MihomoApprovalRouteUpdate, MihomoApprovalRouteDelete,
		MihomoApprovalNodeAction, MihomoApprovalLegacyImport:
		return true
	default:
		return false
	}
}

func isManagedMihomoResourceKey(key string) bool {
	return key == "mihomo:import" || key == "mihomo:nodes" || strings.HasPrefix(key, "mihomo:subscription:") ||
		strings.HasPrefix(key, "mihomo:route:") || strings.HasPrefix(key, "mihomo:nodes:")
}

func (s *MihomoService) PrepareManagedSubscriptionApproval(ctx context.Context, operation string, id int64, name, subscriptionURL string, enabled bool, refreshIntervalMinutes int) (*MihomoApprovalUpdate, error) {
	input := MihomoManagedSubscriptionInput{Name: name, SubscriptionURL: subscriptionURL, Enabled: enabled, RefreshIntervalMinutes: refreshIntervalMinutes}
	if err := s.managedResourcesReady(); err != nil {
		return nil, err
	}
	input.Name = strings.TrimSpace(input.Name)
	if (operation == MihomoApprovalSubscriptionCreate || operation == MihomoApprovalSubscriptionUpdate) &&
		(input.Name == "" || len(input.Name) > 100 || input.RefreshIntervalMinutes < 5 || input.RefreshIntervalMinutes > 10080) {
		return nil, infraerrors.BadRequest("INVALID_MIHOMO_SUBSCRIPTION", "name and refresh interval are invalid")
	}
	if operation == MihomoApprovalSubscriptionCreate && id != 0 || operation != MihomoApprovalSubscriptionCreate && id <= 0 {
		return nil, infraerrors.BadRequest("INVALID_MIHOMO_SUBSCRIPTION", "subscription target is invalid")
	}

	var current *MihomoSubscription
	var err error
	if id > 0 {
		current, err = s.resources.GetSubscriptionByID(ctx, id)
		if err != nil {
			return nil, err
		}
	}
	update := &MihomoApprovalUpdate{SubscriptionID: id, SubscriptionName: input.Name}
	switch operation {
	case MihomoApprovalSubscriptionCreate:
		update.Kind = MihomoApprovalSubscriptionCreate
	case MihomoApprovalSubscriptionUpdate:
		update.Kind = MihomoApprovalSubscriptionUpdate
	case MihomoApprovalSubscriptionDelete:
		update.Kind = MihomoApprovalSubscriptionDelete
		update.SubscriptionName = current.Name
		nodes, listErr := s.allMihomoNodes(ctx, MihomoNodeFilter{SubscriptionID: &id})
		if listErr != nil {
			return nil, listErr
		}
		routes, listErr := s.allMihomoRoutes(ctx, MihomoRouteFilter{SubscriptionID: &id})
		if listErr != nil {
			return nil, listErr
		}
		if len(routes) > 0 {
			return nil, infraerrors.Conflict("MIHOMO_SUBSCRIPTION_IN_USE", "remove or reassign associated routes before deleting this subscription")
		}
		update.NodeCount, update.RouteCount = int64(len(nodes)), int64(len(routes))
		return update, nil
	case MihomoApprovalSubscriptionRefresh:
		update.Kind, update.SubscriptionName = MihomoApprovalSubscriptionRefresh, current.Name
		if err = s.validateManagedApproval(ctx, managedResourceKey(update), *update); err != nil {
			return nil, err
		}
		return update, nil
	default:
		return nil, infraerrors.BadRequest("INVALID_MIHOMO_SUBSCRIPTION", "subscription operation is invalid")
	}

	rawURL := strings.TrimSpace(input.SubscriptionURL)
	if rawURL == "" && current != nil {
		update.Subscription = string(current.URLCiphertext)
		update.SubscriptionHost = current.MaskedHost
	} else {
		if rawURL == "" || s.encryptor == nil {
			return nil, infraerrors.BadRequest("INVALID_MIHOMO_SUBSCRIPTION", "subscription URL is required")
		}
		if err = validateMihomoSubscriptionURL(ctx, rawURL); err != nil {
			return nil, err
		}
		ciphertext, encryptErr := s.encryptor.Encrypt(rawURL)
		if encryptErr != nil {
			return nil, encryptErr
		}
		parsed, _ := url.Parse(rawURL)
		update.Subscription, update.SubscriptionHost = ciphertext, parsed.Hostname()
	}
	update.RefreshIntervalMinutes = input.RefreshIntervalMinutes
	update.Enabled = boolPointer(input.Enabled)
	key := managedResourceKey(update)
	if err = s.validateManagedApproval(ctx, key, *update); err != nil {
		return nil, err
	}
	return update, nil
}

func (s *MihomoService) PrepareManagedRouteApproval(ctx context.Context, operation string, id int64, name, kind string, subscriptionIDs, nodeIDs []int64, enabled bool) (*MihomoApprovalUpdate, error) {
	input := MihomoManagedRouteInput{Name: name, Kind: kind, SubscriptionIDs: subscriptionIDs, NodeIDs: nodeIDs, Enabled: enabled}
	if err := s.managedResourcesReady(); err != nil {
		return nil, err
	}
	if operation == MihomoApprovalRouteCreate && id != 0 || operation != MihomoApprovalRouteCreate && id <= 0 {
		return nil, infraerrors.BadRequest("INVALID_MIHOMO_ROUTE", "route target is invalid")
	}
	update := &MihomoApprovalUpdate{RouteID: id, RouteName: strings.TrimSpace(input.Name)}
	var current *MihomoRoute
	var err error
	if id > 0 {
		current, err = s.resources.GetRouteByID(ctx, id)
		if err != nil {
			return nil, err
		}
	}
	switch operation {
	case MihomoApprovalRouteCreate, MihomoApprovalRouteUpdate:
		if update.RouteName == "" || len(update.RouteName) > 100 || !validMihomoRouteKind(input.Kind) {
			return nil, infraerrors.BadRequest("INVALID_MIHOMO_ROUTE", "route name or strategy is invalid")
		}
		nodeIDs, nodeErr := s.validateManagedNodeIDs(ctx, input.NodeIDs, input.SubscriptionIDs)
		if nodeErr != nil {
			return nil, nodeErr
		}
		update.Kind = MihomoApprovalRouteCreate
		if operation == MihomoApprovalRouteUpdate {
			update.Kind, update.ListenerPort = MihomoApprovalRouteUpdate, current.ListenerPort
			if current.ProxyID != nil {
				update.ProxyID = *current.ProxyID
				update.AccountCount, err = s.proxyRepo.CountAccountsByProxyID(ctx, *current.ProxyID)
				if err != nil {
					return nil, err
				}
			}
		} else {
			update.ListenerPort, err = s.nextMihomoListenerPort(ctx, nil)
			if err != nil {
				return nil, err
			}
		}
		update.RouteKind = strings.ToLower(strings.TrimSpace(input.Kind))
		update.SubscriptionIDs = uniquePositiveInt64s(input.SubscriptionIDs)
		update.NodeIDs = nodeIDs
		update.Enabled = boolPointer(input.Enabled)
	case MihomoApprovalRouteDelete:
		update.Kind, update.RouteName, update.RouteKind = MihomoApprovalRouteDelete, current.Name, current.Kind
		update.ListenerPort = current.ListenerPort
		if current.ProxyID != nil {
			update.ProxyID = *current.ProxyID
			update.AccountCount, err = s.proxyRepo.CountAccountsByProxyID(ctx, *current.ProxyID)
			if err != nil {
				return nil, err
			}
			if update.AccountCount > 0 {
				return nil, infraerrors.Conflict("MIHOMO_ROUTE_IN_USE", "move bound accounts before deleting this route")
			}
		}
	default:
		return nil, infraerrors.BadRequest("INVALID_MIHOMO_ROUTE", "route operation is invalid")
	}
	key := managedResourceKey(update)
	if err = s.validateManagedApproval(ctx, key, *update); err != nil {
		return nil, err
	}
	return update, nil
}

func (s *MihomoService) PrepareManagedNodeApproval(ctx context.Context, action string, nodeIDs []int64) (*MihomoApprovalUpdate, string, error) {
	if err := s.managedResourcesReady(); err != nil {
		return nil, "", err
	}
	action = strings.ToLower(strings.TrimSpace(action))
	if action != "exclude" && action != "restore" && action != "enable" && action != "disable" && action != "create_dedicated_routes" {
		return nil, "", infraerrors.BadRequest("INVALID_MIHOMO_NODE_ACTION", "node action is invalid")
	}
	ids, err := s.validateManagedNodeIDs(ctx, nodeIDs, nil)
	if err != nil {
		return nil, "", err
	}
	update := &MihomoApprovalUpdate{Kind: MihomoApprovalNodeAction, NodeAction: action, NodeIDs: ids, NodeCount: int64(len(ids))}
	if action == "create_dedicated_routes" {
		reserved := make(map[int]bool, len(ids))
		for _, id := range ids {
			node, getErr := s.resources.GetNodeByID(ctx, id)
			if getErr != nil {
				return nil, "", getErr
			}
			port, portErr := s.nextMihomoListenerPort(ctx, reserved)
			if portErr != nil {
				return nil, "", portErr
			}
			reserved[port] = true
			name := strings.TrimSpace(node.DisplayName)
			if name == "" {
				name = node.OriginalName
			}
			update.ImportRoutes = append(update.ImportRoutes, MihomoApprovalRoute{
				Name: fmt.Sprintf("%s #%d", name, id), Kind: "dedicated", ListenerPort: port,
				SubscriptionIDs: []int64{node.SubscriptionID}, NodeIDs: []int64{id}, NodeCount: 1,
			})
		}
		update.RouteCount = int64(len(update.ImportRoutes))
	}
	key := managedResourceKey(update)
	if err = s.validateManagedApproval(ctx, key, *update); err != nil {
		return nil, "", err
	}
	return update, key, nil
}

func (s *MihomoService) LegacyImportPreview(ctx context.Context) (*MihomoLegacyImportPreview, error) {
	if err := s.managedResourcesReady(); err != nil {
		return nil, err
	}
	subscriptions, err := s.allMihomoSubscriptions(ctx, MihomoSubscriptionFilter{})
	if err != nil {
		return nil, err
	}
	routes, err := s.allMihomoRoutes(ctx, MihomoRouteFilter{})
	if err != nil {
		return nil, err
	}
	if len(subscriptions) > 0 || len(routes) > 0 {
		return &MihomoLegacyImportPreview{AlreadyImported: true, Routes: []MihomoApprovalRoute{}}, nil
	}
	_, cfg, err := s.readConfig()
	if err != nil {
		return nil, err
	}
	provider := yamlMap(yamlMap(cfg["proxy-providers"])[s.cfg.ProviderName])
	rawURL := strings.TrimSpace(fmt.Sprint(provider["url"]))
	if rawURL == "" || rawURL == "<nil>" {
		return &MihomoLegacyImportPreview{Routes: []MihomoApprovalRoute{}}, nil
	}
	parsed, _ := url.Parse(rawURL)
	state, err := s.providerState(ctx)
	if err != nil {
		return nil, err
	}
	nodeNames := make([]string, 0, len(state.Nodes))
	for _, node := range state.Nodes {
		nodeNames = append(nodeNames, node.Name)
	}
	groups, err := s.proxyGroups(ctx)
	if err != nil {
		return nil, err
	}
	proxyIDs, err := s.managedProxyIDs(ctx)
	if err != nil {
		return nil, err
	}
	definitions := []struct {
		mode, name, kind string
		port             int
	}{
		{"automatic", "自动线路", "automatic", s.cfg.AutomaticPort},
		{"directional", "定向线路", "directional", s.cfg.DirectionalPort},
		{"dynamic", "动态线路", "dynamic", s.cfg.DynamicPort},
	}
	preview := &MihomoLegacyImportPreview{
		Available: true, ProviderName: s.cfg.ProviderName, SubscriptionHost: parsed.Hostname(),
		NodeCount: int64(len(nodeNames)), Routes: make([]MihomoApprovalRoute, 0, len(definitions)),
	}
	for _, definition := range definitions {
		routeNodes := append([]string(nil), nodeNames...)
		if selected := groups[mihomoGroups[definition.mode]]; selected != "" {
			routeNodes = preferredStringFirst(routeNodes, selected)
		}
		route := MihomoApprovalRoute{
			Name: definition.name, Kind: definition.kind, ListenerPort: definition.port,
			ProxyID: proxyIDs[definition.mode], NodeNames: routeNodes, NodeCount: int64(len(routeNodes)),
		}
		if route.ProxyID > 0 {
			route.AccountCount, err = s.proxyRepo.CountAccountsByProxyID(ctx, route.ProxyID)
			if err != nil {
				return nil, err
			}
			preview.AffectedAccountCount += route.AccountCount
		}
		preview.Routes = append(preview.Routes, route)
	}
	preview.RouteCount = int64(len(preview.Routes))
	return preview, nil
}

func (s *MihomoService) PrepareLegacyImportApproval(ctx context.Context) (*MihomoApprovalUpdate, string, error) {
	preview, err := s.LegacyImportPreview(ctx)
	if err != nil {
		return nil, "", err
	}
	if !preview.Available || preview.AlreadyImported || preview.NodeCount == 0 {
		return nil, "", infraerrors.Conflict("MIHOMO_IMPORT_UNAVAILABLE", "legacy Mihomo resources are not available for import")
	}
	_, cfg, err := s.readConfig()
	if err != nil {
		return nil, "", err
	}
	provider := yamlMap(yamlMap(cfg["proxy-providers"])[s.cfg.ProviderName])
	rawURL := strings.TrimSpace(fmt.Sprint(provider["url"]))
	if err = validateMihomoSubscriptionURL(ctx, rawURL); err != nil {
		return nil, "", err
	}
	if s.encryptor == nil {
		return nil, "", fmt.Errorf("secret encryptor is not configured")
	}
	ciphertext, err := s.encryptor.Encrypt(rawURL)
	if err != nil {
		return nil, "", err
	}
	update := &MihomoApprovalUpdate{
		Kind: MihomoApprovalLegacyImport, SubscriptionName: "现有订阅", Subscription: ciphertext,
		SubscriptionHost: preview.SubscriptionHost, RefreshIntervalMinutes: 60, Enabled: boolPointer(true),
		NodeCount: preview.NodeCount, RouteCount: preview.RouteCount, AccountCount: preview.AffectedAccountCount,
		ImportProviderName: preview.ProviderName, ImportRoutes: preview.Routes,
	}
	key := managedResourceKey(update)
	if err = s.validateManagedApproval(ctx, key, *update); err != nil {
		return nil, "", err
	}
	return update, key, nil
}

func (s *MihomoService) TestManagedNodes(ctx context.Context, nodeIDs []int64) error {
	ids, err := s.validateManagedNodeIDs(ctx, nodeIDs, nil)
	if err != nil {
		return err
	}
	subscriptions := make(map[int64]struct{})
	for _, id := range ids {
		node, getErr := s.resources.GetNodeByID(ctx, id)
		if getErr != nil {
			return getErr
		}
		subscriptions[node.SubscriptionID] = struct{}{}
	}
	for id := range subscriptions {
		if err = s.RefreshManagedSubscription(ctx, id); err != nil {
			return err
		}
	}
	return nil
}

func (s *MihomoService) TestManagedRoute(ctx context.Context, routeID int64) error {
	if err := s.managedResourcesReady(); err != nil {
		return err
	}
	route, err := s.resources.GetRouteByID(ctx, routeID)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	route.LastCheckedAt = &now
	if route.ProxyID == nil || s.proxyProber == nil {
		route.ExitHealthy = boolPointer(false)
		route.LastError = "route proxy probe is not configured"
		if updateErr := s.resources.UpdateRoute(ctx, route); updateErr != nil {
			return updateErr
		}
		return nil
	}
	proxy, err := s.proxyRepo.GetByID(ctx, *route.ProxyID)
	if err != nil {
		return err
	}
	exit, latency, probeErr := s.proxyProber.ProbeProxy(ctx, proxy.URL())
	if probeErr != nil {
		route.ExitHealthy = boolPointer(false)
		route.LastError = probeErr.Error()
	} else {
		delay := int(latency)
		route.ExitIP, route.ExitDelayMS, route.ExitHealthy, route.LastError = exit.IP, &delay, boolPointer(true), ""
	}
	if err = s.resources.UpdateRoute(ctx, route); err != nil {
		return err
	}
	return nil
}

func (s *MihomoService) validateManagedApproval(ctx context.Context, resourceKey string, update MihomoApprovalUpdate) error {
	if err := s.managedResourcesReady(); err != nil {
		return err
	}
	if resourceKey != managedResourceKey(&update) {
		return infraerrors.BadRequest("INVALID_MIHOMO_TARGET", "mihomo resource target does not match the update")
	}
	switch update.Kind {
	case MihomoApprovalSubscriptionCreate, MihomoApprovalSubscriptionUpdate:
		if update.Kind == MihomoApprovalSubscriptionCreate && update.SubscriptionID != 0 || update.Kind == MihomoApprovalSubscriptionUpdate && update.SubscriptionID <= 0 ||
			strings.TrimSpace(update.SubscriptionName) == "" || len(update.SubscriptionName) > 100 || update.RefreshIntervalMinutes < 5 || update.RefreshIntervalMinutes > 10080 || update.Enabled == nil || update.Subscription == "" || s.encryptor == nil {
			return infraerrors.BadRequest("INVALID_MIHOMO_SUBSCRIPTION", "subscription update is invalid")
		}
		rawURL, err := s.encryptor.Decrypt(update.Subscription)
		if err != nil {
			return infraerrors.BadRequest("INVALID_MIHOMO_SUBSCRIPTION", "subscription URL is invalid")
		}
		if err = validateMihomoSubscriptionURL(ctx, rawURL); err != nil {
			return err
		}
		parsed, _ := url.Parse(rawURL)
		if parsed.Hostname() != update.SubscriptionHost {
			return infraerrors.BadRequest("INVALID_MIHOMO_SUBSCRIPTION", "subscription host does not match the encrypted URL")
		}
	case MihomoApprovalSubscriptionDelete, MihomoApprovalSubscriptionRefresh:
		if update.SubscriptionID <= 0 {
			return infraerrors.BadRequest("INVALID_MIHOMO_SUBSCRIPTION", "subscription target is invalid")
		}
	case MihomoApprovalRouteCreate, MihomoApprovalRouteUpdate:
		if update.Kind == MihomoApprovalRouteCreate && update.RouteID != 0 || update.Kind == MihomoApprovalRouteUpdate && update.RouteID <= 0 ||
			strings.TrimSpace(update.RouteName) == "" || len(update.RouteName) > 100 || !validMihomoRouteKind(update.RouteKind) ||
			update.ListenerPort < 1 || update.ListenerPort > 65535 || update.Enabled == nil || len(uniquePositiveInt64s(update.NodeIDs)) == 0 {
			return infraerrors.BadRequest("INVALID_MIHOMO_ROUTE", "route update is invalid")
		}
		if _, err := s.validateManagedNodeIDs(ctx, update.NodeIDs, update.SubscriptionIDs); err != nil {
			return err
		}
	case MihomoApprovalRouteDelete:
		if update.RouteID <= 0 {
			return infraerrors.BadRequest("INVALID_MIHOMO_ROUTE", "route target is invalid")
		}
	case MihomoApprovalNodeAction:
		if update.NodeAction != "exclude" && update.NodeAction != "restore" && update.NodeAction != "enable" && update.NodeAction != "disable" && update.NodeAction != "create_dedicated_routes" {
			return infraerrors.BadRequest("INVALID_MIHOMO_NODE_ACTION", "node action is invalid")
		}
		if _, err := s.validateManagedNodeIDs(ctx, update.NodeIDs, nil); err != nil {
			return err
		}
	case MihomoApprovalLegacyImport:
		if update.ImportProviderName != s.cfg.ProviderName || update.Subscription == "" || update.SubscriptionHost == "" || len(update.ImportRoutes) == 0 || update.NodeCount <= 0 {
			return infraerrors.BadRequest("INVALID_MIHOMO_IMPORT", "legacy import payload is invalid")
		}
		if s.encryptor == nil {
			return infraerrors.BadRequest("INVALID_MIHOMO_IMPORT", "legacy subscription is invalid")
		}
		rawURL, err := s.encryptor.Decrypt(update.Subscription)
		if err != nil {
			return infraerrors.BadRequest("INVALID_MIHOMO_IMPORT", "legacy subscription is invalid")
		}
		parsed, err := url.Parse(rawURL)
		if err != nil || parsed.Hostname() != update.SubscriptionHost {
			return infraerrors.BadRequest("INVALID_MIHOMO_IMPORT", "legacy subscription host is invalid")
		}
		ports := make(map[int]struct{}, len(update.ImportRoutes))
		for _, route := range update.ImportRoutes {
			if strings.TrimSpace(route.Name) == "" || !validMihomoRouteKind(route.Kind) || route.ListenerPort < 1 || route.ListenerPort > 65535 || len(route.NodeNames) == 0 {
				return infraerrors.BadRequest("INVALID_MIHOMO_IMPORT", "legacy route is invalid")
			}
			if _, exists := ports[route.ListenerPort]; exists {
				return infraerrors.BadRequest("INVALID_MIHOMO_IMPORT", "legacy routes share a listener port")
			}
			ports[route.ListenerPort] = struct{}{}
		}
	default:
		return infraerrors.BadRequest("INVALID_MIHOMO_UPDATE", "invalid mihomo update")
	}
	return nil
}

func (s *MihomoService) managedApprovalRevision(ctx context.Context, resourceKey string) (string, error) {
	if err := s.managedResourcesReady(); err != nil {
		return "", err
	}
	subscriptions, err := s.allMihomoSubscriptions(ctx, MihomoSubscriptionFilter{})
	if err != nil {
		return "", err
	}
	nodes, err := s.allMihomoNodes(ctx, MihomoNodeFilter{IncludeRemoved: true})
	if err != nil {
		return "", err
	}
	routes, err := s.allMihomoRoutes(ctx, MihomoRouteFilter{})
	if err != nil {
		return "", err
	}
	type subscriptionRevision struct {
		ID, URLHash             string
		Name, Key, Host, Status string
		Interval                int
	}
	type nodeRevision struct {
		ID, SubscriptionID int64
		Name               string
		Excluded, Removed  bool
	}
	type routeRevision struct {
		ID, ProxyID        int64
		Name, Kind, Status string
		Port               int
		NodeIDs            []int64
	}
	subscriptionState := make([]subscriptionRevision, 0, len(subscriptions))
	for _, item := range subscriptions {
		digest := sha256.Sum256(item.URLCiphertext)
		subscriptionState = append(subscriptionState, subscriptionRevision{
			ID: fmt.Sprint(item.ID), URLHash: fmt.Sprintf("%x", digest[:]), Name: item.Name,
			Key: item.ProviderKey, Host: item.MaskedHost, Status: item.Status, Interval: item.RefreshIntervalSeconds,
		})
	}
	nodeState := make([]nodeRevision, 0, len(nodes))
	for _, item := range nodes {
		nodeState = append(nodeState, nodeRevision{item.ID, item.SubscriptionID, item.DisplayName, item.Excluded, item.UpstreamRemovedAt != nil})
	}
	routeState := make([]routeRevision, 0, len(routes))
	for _, item := range routes {
		proxyID := int64(0)
		if item.ProxyID != nil {
			proxyID = *item.ProxyID
		}
		relations, listErr := s.resources.ListRouteNodes(ctx, item.ID)
		if listErr != nil {
			return "", listErr
		}
		nodeIDs := make([]int64, 0, len(relations))
		for _, relation := range relations {
			nodeIDs = append(nodeIDs, relation.NodeID)
		}
		routeState = append(routeState, routeRevision{item.ID, proxyID, item.Name, item.Kind, item.Status, item.ListenerPort, nodeIDs})
	}
	return approvalDigest(resourceKey, subscriptionState, nodeState, routeState), nil
}

func (s *MihomoService) applyManagedApproved(ctx context.Context, update MihomoApprovalUpdate) (func(bool) error, error) {
	if err := s.validateManagedApproval(ctx, managedResourceKey(&update), update); err != nil {
		return nil, err
	}
	var err error
	switch update.Kind {
	case MihomoApprovalSubscriptionCreate:
		digest := sha256.Sum256([]byte(update.Subscription))
		item := &MihomoSubscription{
			Name: update.SubscriptionName, ProviderKey: fmt.Sprintf("sub_%x", digest[:6]),
			URLCiphertext: []byte(update.Subscription), MaskedHost: update.SubscriptionHost,
			RefreshIntervalSeconds: update.RefreshIntervalMinutes * 60, Status: enabledStatus(update.Enabled),
		}
		err = s.resources.CreateSubscription(ctx, item)
	case MihomoApprovalSubscriptionUpdate:
		var item *MihomoSubscription
		item, err = s.resources.GetSubscriptionByID(ctx, update.SubscriptionID)
		if err == nil {
			item.Name, item.URLCiphertext, item.MaskedHost = update.SubscriptionName, []byte(update.Subscription), update.SubscriptionHost
			item.RefreshIntervalSeconds, item.Status = update.RefreshIntervalMinutes*60, enabledStatus(update.Enabled)
			err = s.resources.UpdateSubscription(ctx, item)
		}
	case MihomoApprovalSubscriptionDelete:
		err = s.resources.DeleteSubscription(ctx, update.SubscriptionID)
	case MihomoApprovalSubscriptionRefresh:
		err = s.RefreshManagedSubscription(ctx, update.SubscriptionID)
		if err == nil {
			return nil, nil
		}
	case MihomoApprovalRouteCreate, MihomoApprovalRouteUpdate:
		err = s.applyManagedRoute(ctx, update)
	case MihomoApprovalRouteDelete:
		var route *MihomoRoute
		route, err = s.resources.GetRouteByID(ctx, update.RouteID)
		if err == nil && route.ProxyID != nil {
			var count int64
			count, err = s.proxyRepo.CountAccountsByProxyID(ctx, *route.ProxyID)
			if err == nil && count > 0 {
				err = infraerrors.Conflict("MIHOMO_ROUTE_IN_USE", "move bound accounts before deleting this route")
			}
		}
		if err == nil {
			err = s.resources.DeleteRoute(ctx, update.RouteID)
		}
	case MihomoApprovalNodeAction:
		err = s.applyManagedNodeAction(ctx, update)
	case MihomoApprovalLegacyImport:
		err = s.applyLegacyImport(ctx, update)
	}
	if err != nil {
		return nil, err
	}
	return s.ApplyManagedRuntime(ctx)
}

func (s *MihomoService) applyManagedRoute(ctx context.Context, update MihomoApprovalUpdate) error {
	var route *MihomoRoute
	var err error
	if update.Kind == MihomoApprovalRouteUpdate {
		route, err = s.resources.GetRouteByID(ctx, update.RouteID)
		if err != nil {
			return err
		}
	} else {
		route = &MihomoRoute{ListenerPort: update.ListenerPort, Selector: json.RawMessage(`{}`)}
	}
	route.Name, route.Kind, route.Status = update.RouteName, update.RouteKind, enabledStatus(update.Enabled)
	if update.Kind == MihomoApprovalRouteCreate {
		if err = s.resources.CreateRoute(ctx, route); err != nil {
			return err
		}
	} else if err = s.resources.UpdateRoute(ctx, route); err != nil {
		return err
	}
	return s.resources.ReplaceRouteNodes(ctx, route.ID, mihomoRouteRelations(route.ID, update.NodeIDs))
}

func (s *MihomoService) applyManagedNodeAction(ctx context.Context, update MihomoApprovalUpdate) error {
	if update.NodeAction == "create_dedicated_routes" {
		for _, prepared := range update.ImportRoutes {
			route := &MihomoRoute{Name: prepared.Name, Kind: "dedicated", ListenerPort: prepared.ListenerPort, Status: StatusActive, Selector: json.RawMessage(`{}`)}
			if err := s.resources.CreateRoute(ctx, route); err != nil {
				return err
			}
			if err := s.resources.ReplaceRouteNodes(ctx, route.ID, mihomoRouteRelations(route.ID, prepared.NodeIDs)); err != nil {
				return err
			}
		}
		return nil
	}
	excluded := update.NodeAction == "exclude" || update.NodeAction == "disable"
	for _, id := range update.NodeIDs {
		node, err := s.resources.GetNodeByID(ctx, id)
		if err != nil {
			return err
		}
		node.Excluded = excluded
		if err = s.resources.UpdateNode(ctx, node); err != nil {
			return err
		}
	}
	return nil
}

func (s *MihomoService) applyLegacyImport(ctx context.Context, update MihomoApprovalUpdate) error {
	subscriptions, err := s.allMihomoSubscriptions(ctx, MihomoSubscriptionFilter{})
	if err != nil {
		return err
	}
	routes, err := s.allMihomoRoutes(ctx, MihomoRouteFilter{})
	if err != nil {
		return err
	}
	if len(subscriptions) > 0 || len(routes) > 0 {
		return infraerrors.Conflict("MIHOMO_IMPORT_ALREADY_DONE", "managed Mihomo resources already exist")
	}
	subscription := &MihomoSubscription{
		Name: update.SubscriptionName, ProviderKey: update.ImportProviderName,
		URLCiphertext: []byte(update.Subscription), MaskedHost: update.SubscriptionHost,
		RefreshIntervalSeconds: update.RefreshIntervalMinutes * 60, Status: StatusActive,
	}
	if err = s.resources.CreateSubscription(ctx, subscription); err != nil {
		return err
	}
	now := time.Now().UTC()
	seenNames := make(map[string]struct{})
	managedNodes := make([]MihomoManagedNode, 0, update.NodeCount)
	for _, route := range update.ImportRoutes {
		for _, name := range route.NodeNames {
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			if _, exists := seenNames[name]; exists {
				continue
			}
			seenNames[name] = struct{}{}
			digest := sha256.Sum256([]byte(subscription.ProviderKey + "\x00" + name))
			managedNodes = append(managedNodes, MihomoManagedNode{
				SubscriptionID: subscription.ID, NodeKey: fmt.Sprintf("%x", digest[:]), OriginalName: name,
				DisplayName: name, Alive: true, Tags: []string{}, LastSeenAt: now,
			})
		}
	}
	if err = s.resources.SyncNodes(ctx, subscription.ID, managedNodes, now); err != nil {
		return err
	}
	nodes, err := s.allMihomoNodes(ctx, MihomoNodeFilter{SubscriptionID: &subscription.ID})
	if err != nil {
		return err
	}
	nodeIDByName := make(map[string]int64, len(nodes))
	for _, node := range nodes {
		nodeIDByName[node.OriginalName] = node.ID
	}
	for _, prepared := range update.ImportRoutes {
		var proxyID *int64
		if prepared.ProxyID > 0 {
			proxy, getErr := s.proxyRepo.GetByID(ctx, prepared.ProxyID)
			if getErr != nil {
				return getErr
			}
			if proxy.Host != s.cfg.ProxyHost || proxy.Port != prepared.ListenerPort {
				return infraerrors.Conflict("MIHOMO_IMPORT_PROXY_MISMATCH", "legacy proxy endpoint no longer matches the import preview")
			}
			proxyID = &prepared.ProxyID
		}
		route := &MihomoRoute{
			Name: prepared.Name, Kind: prepared.Kind, ListenerPort: prepared.ListenerPort,
			ProxyID: proxyID, Status: StatusActive, Selector: json.RawMessage(`{}`),
		}
		if err = s.resources.CreateRoute(ctx, route); err != nil {
			return err
		}
		nodeIDs := make([]int64, 0, len(prepared.NodeNames))
		for _, name := range prepared.NodeNames {
			if id := nodeIDByName[name]; id > 0 {
				nodeIDs = append(nodeIDs, id)
			}
		}
		if len(nodeIDs) == 0 {
			return infraerrors.BadRequest("MIHOMO_IMPORT_EMPTY_ROUTE", "legacy route has no importable nodes")
		}
		current := nodeIDs[0]
		route.CurrentNodeID = &current
		if err = s.resources.UpdateRoute(ctx, route); err != nil {
			return err
		}
		if err = s.resources.ReplaceRouteNodes(ctx, route.ID, mihomoRouteRelations(route.ID, nodeIDs)); err != nil {
			return err
		}
	}
	return nil
}

func (s *MihomoService) managedResourcesReady() error {
	if !s.cfg.Enabled {
		return infraerrors.ServiceUnavailable("MIHOMO_DISABLED", "mihomo integration is disabled")
	}
	if s.resources == nil {
		return infraerrors.ServiceUnavailable("MIHOMO_RESOURCES_UNAVAILABLE", "mihomo managed resources are not configured")
	}
	return nil
}

func (s *MihomoService) ensureLegacyMutationAvailable(ctx context.Context) error {
	if s.resources == nil {
		return nil
	}
	subscriptions, err := s.allMihomoSubscriptions(ctx, MihomoSubscriptionFilter{})
	if err != nil {
		return err
	}
	routes, err := s.allMihomoRoutes(ctx, MihomoRouteFilter{})
	if err != nil {
		return err
	}
	if len(subscriptions) > 0 || len(routes) > 0 {
		return infraerrors.Conflict("MIHOMO_MANAGED_MODE_ACTIVE", "use the managed Mihomo workbench for this change")
	}
	return nil
}

func (s *MihomoService) validateManagedNodeIDs(ctx context.Context, rawIDs, subscriptionIDs []int64) ([]int64, error) {
	ids := uniquePositiveInt64s(rawIDs)
	if len(ids) == 0 {
		return nil, infraerrors.BadRequest("MIHOMO_ROUTE_NODES_REQUIRED", "at least one managed node is required")
	}
	allowedSubscriptions := make(map[int64]struct{}, len(subscriptionIDs))
	for _, id := range uniquePositiveInt64s(subscriptionIDs) {
		allowedSubscriptions[id] = struct{}{}
	}
	for _, id := range ids {
		node, err := s.resources.GetNodeByID(ctx, id)
		if err != nil {
			return nil, err
		}
		if node.UpstreamRemovedAt != nil {
			return nil, infraerrors.BadRequest("MIHOMO_NODE_REMOVED", "an upstream-removed node cannot be assigned")
		}
		if len(allowedSubscriptions) > 0 {
			if _, ok := allowedSubscriptions[node.SubscriptionID]; !ok {
				return nil, infraerrors.BadRequest("MIHOMO_NODE_SUBSCRIPTION_MISMATCH", "a selected node is outside the selected subscriptions")
			}
		}
	}
	return ids, nil
}

func (s *MihomoService) nextMihomoListenerPort(ctx context.Context, reserved map[int]bool) (int, error) {
	used := make(map[int]bool)
	routes, err := s.allMihomoRoutes(ctx, MihomoRouteFilter{})
	if err != nil {
		return 0, err
	}
	for _, route := range routes {
		used[route.ListenerPort] = true
	}
	proxies, err := s.proxyRepo.ListActive(ctx)
	if err != nil {
		return 0, err
	}
	for _, proxy := range proxies {
		if proxy.Host == s.cfg.ProxyHost {
			used[proxy.Port] = true
		}
	}
	start := maxInt(maxInt(s.cfg.AutomaticPort, s.cfg.DirectionalPort), maxInt(s.cfg.DynamicPort, 20000))
	for port := start + 1; port <= 65535; port++ {
		if !used[port] && (reserved == nil || !reserved[port]) {
			return port, nil
		}
	}
	return 0, infraerrors.Conflict("MIHOMO_PORTS_EXHAUSTED", "no listener port is available")
}

func managedResourceKey(update *MihomoApprovalUpdate) string {
	if update == nil {
		return ""
	}
	switch update.Kind {
	case MihomoApprovalSubscriptionCreate:
		return "mihomo:subscription:new"
	case MihomoApprovalSubscriptionUpdate, MihomoApprovalSubscriptionDelete, MihomoApprovalSubscriptionRefresh:
		return fmt.Sprintf("mihomo:subscription:%d", update.SubscriptionID)
	case MihomoApprovalRouteCreate:
		return "mihomo:route:new"
	case MihomoApprovalRouteUpdate, MihomoApprovalRouteDelete:
		return fmt.Sprintf("mihomo:route:%d", update.RouteID)
	case MihomoApprovalNodeAction:
		return "mihomo:nodes"
	case MihomoApprovalLegacyImport:
		return "mihomo:import"
	default:
		return ""
	}
}

const mihomoAllPageSize = 1000

func (s *MihomoService) allMihomoSubscriptions(ctx context.Context, filter MihomoSubscriptionFilter) ([]MihomoSubscription, error) {
	all := make([]MihomoSubscription, 0)
	for page := 1; ; page++ {
		items, _, err := s.resources.ListSubscriptions(ctx, mihomoAllPage(page), filter)
		if err != nil {
			return nil, err
		}
		all = append(all, items...)
		if len(items) < mihomoAllPageSize {
			return all, nil
		}
	}
}

func (s *MihomoService) allMihomoNodes(ctx context.Context, filter MihomoNodeFilter) ([]MihomoManagedNode, error) {
	all := make([]MihomoManagedNode, 0)
	for page := 1; ; page++ {
		items, _, err := s.resources.ListNodes(ctx, mihomoAllPage(page), filter)
		if err != nil {
			return nil, err
		}
		all = append(all, items...)
		if len(items) < mihomoAllPageSize {
			return all, nil
		}
	}
}

func (s *MihomoService) allMihomoRoutes(ctx context.Context, filter MihomoRouteFilter) ([]MihomoRoute, error) {
	all := make([]MihomoRoute, 0)
	for page := 1; ; page++ {
		items, _, err := s.resources.ListRoutes(ctx, mihomoAllPage(page), filter)
		if err != nil {
			return nil, err
		}
		all = append(all, items...)
		if len(items) < mihomoAllPageSize {
			return all, nil
		}
	}
}

func mihomoAllPage(page int) pagination.PaginationParams {
	return pagination.PaginationParams{Page: page, PageSize: mihomoAllPageSize, SortBy: "id", SortOrder: pagination.SortOrderAsc}
}

func validMihomoRouteKind(kind string) bool {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "dedicated", "automatic", "latency", "fallback", "dynamic", "directional":
		return true
	default:
		return false
	}
}

func uniquePositiveInt64s(values []int64) []int64 {
	result := make([]int64, 0, len(values))
	seen := make(map[int64]struct{}, len(values))
	for _, value := range values {
		if value <= 0 {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func mihomoRouteRelations(routeID int64, nodeIDs []int64) []MihomoRouteNode {
	relations := make([]MihomoRouteNode, 0, len(nodeIDs))
	for priority, nodeID := range uniquePositiveInt64s(nodeIDs) {
		relations = append(relations, MihomoRouteNode{RouteID: routeID, NodeID: nodeID, Priority: priority, Weight: 1})
	}
	return relations
}

func boolPointer(value bool) *bool { return &value }

func enabledStatus(enabled *bool) string {
	if enabled != nil && *enabled {
		return StatusActive
	}
	return StatusDisabled
}

func preferredStringFirst(values []string, preferred string) []string {
	for i, value := range values {
		if value == preferred {
			return append([]string{value}, append(values[:i:i], values[i+1:]...)...)
		}
	}
	return values
}
