package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/util/logredact"
	"gopkg.in/yaml.v3"
)

const mihomoResponseLimit = 8 << 20

type MihomoNode struct {
	Name  string `json:"name"`
	Alive bool   `json:"alive"`
	Delay *int   `json:"delay,omitempty"`
}

type MihomoModeStatus struct {
	Mode      string `json:"mode"`
	Selection string `json:"selection"`
}

type MihomoStatus struct {
	Enabled                bool                        `json:"enabled"`
	Version                string                      `json:"version,omitempty"`
	Configured             bool                        `json:"configured"`
	ProviderName           string                      `json:"provider_name,omitempty"`
	SubscriptionConfigured bool                        `json:"subscription_configured"`
	SubscriptionHost       string                      `json:"subscription_host,omitempty"`
	UpdatedAt              string                      `json:"updated_at,omitempty"`
	NodeCount              int                         `json:"node_count"`
	AliveCount             int                         `json:"alive_count"`
	Nodes                  []MihomoNode                `json:"nodes"`
	Modes                  map[string]MihomoModeStatus `json:"modes"`
	ProxyIDs               map[string]int64            `json:"proxy_ids"`
}

type MihomoRuntimeStatus struct {
	Enabled             bool       `json:"enabled"`
	Version             string     `json:"version,omitempty"`
	Configured          bool       `json:"configured"`
	ControllerConnected bool       `json:"controller_connected"`
	ConfigValid         bool       `json:"config_valid"`
	GeneratedAt         *time.Time `json:"generated_at,omitempty"`
	LastReloadAt        *time.Time `json:"last_reload_at,omitempty"`
	LastReloadError     string     `json:"last_reload_error,omitempty"`
}

type MihomoWorkbenchSubscription struct {
	ID                     int64      `json:"id"`
	Name                   string     `json:"name"`
	Enabled                bool       `json:"enabled"`
	Status                 string     `json:"status"`
	MaskedURL              string     `json:"masked_url,omitempty"`
	SourceHost             string     `json:"source_host,omitempty"`
	RefreshIntervalMinutes int        `json:"refresh_interval_minutes"`
	NodeCount              int        `json:"node_count"`
	AliveCount             int        `json:"alive_count"`
	UsedBytes              *int64     `json:"used_bytes,omitempty"`
	TotalBytes             *int64     `json:"total_bytes,omitempty"`
	ExpiresAt              *time.Time `json:"expires_at,omitempty"`
	LastRefreshedAt        *time.Time `json:"last_refreshed_at,omitempty"`
	LastError              string     `json:"last_error,omitempty"`
}

type MihomoWorkbenchNode struct {
	ID                int64      `json:"id"`
	Key               string     `json:"key"`
	Name              string     `json:"name"`
	DisplayName       string     `json:"display_name,omitempty"`
	SubscriptionID    int64      `json:"subscription_id"`
	SubscriptionName  string     `json:"subscription_name,omitempty"`
	Alive             bool       `json:"alive"`
	Delay             *int       `json:"delay,omitempty"`
	Region            string     `json:"region,omitempty"`
	Tags              []string   `json:"tags"`
	Enabled           bool       `json:"enabled"`
	Excluded          bool       `json:"excluded"`
	LastSeenAt        *time.Time `json:"last_seen_at,omitempty"`
	UpstreamRemovedAt *time.Time `json:"upstream_removed_at,omitempty"`
}

type MihomoWorkbenchRoute struct {
	ID                int64      `json:"id"`
	Name              string     `json:"name"`
	Kind              string     `json:"kind"`
	SubscriptionIDs   []int64    `json:"subscription_ids"`
	SubscriptionNames []string   `json:"subscription_names,omitempty"`
	NodeIDs           []int64    `json:"node_ids"`
	ListenerPort      int        `json:"listener_port"`
	ProxyID           int64      `json:"proxy_id"`
	Enabled           bool       `json:"enabled"`
	CurrentNode       string     `json:"current_node,omitempty"`
	ExitIP            string     `json:"exit_ip,omitempty"`
	Health            string     `json:"health"`
	LatencyMS         *int       `json:"latency_ms,omitempty"`
	AccountCount      int64      `json:"account_count"`
	LastCheckedAt     *time.Time `json:"last_checked_at,omitempty"`
}

type MihomoWorkbench struct {
	Status        MihomoRuntimeStatus           `json:"status"`
	Subscriptions []MihomoWorkbenchSubscription `json:"subscriptions"`
	Nodes         []MihomoWorkbenchNode         `json:"nodes"`
	Routes        []MihomoWorkbenchRoute        `json:"routes"`
}

type MihomoService struct {
	cfg         config.MihomoConfig
	proxyRepo   ProxyRepository
	proxyProber ProxyExitInfoProber
	resources   MihomoRepository
	encryptor   SecretEncryptor
	client      *http.Client
	mu          sync.Mutex
	generatedAt *time.Time
	reloadedAt  *time.Time
	reloadError string
}

func NewMihomoService(cfg *config.Config, proxyRepo ProxyRepository, encryptor SecretEncryptor) *MihomoService {
	return &MihomoService{
		cfg: cfg.Mihomo, proxyRepo: proxyRepo, encryptor: encryptor,
		client: &http.Client{
			Timeout:       12 * time.Second,
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
	}
}

// SetResourceRepository enables the database-backed multi-subscription runtime.
// The existing constructor stays source-compatible during the one-time migration.
func (s *MihomoService) SetResourceRepository(resources MihomoRepository) *MihomoService {
	s.resources = resources
	return s
}

func (s *MihomoService) SetProxyProber(prober ProxyExitInfoProber) *MihomoService {
	s.proxyProber = prober
	return s
}

func (s *MihomoService) Status(ctx context.Context) (*MihomoStatus, error) {
	status := &MihomoStatus{Enabled: s.cfg.Enabled, Nodes: []MihomoNode{}, Modes: map[string]MihomoModeStatus{}, ProxyIDs: map[string]int64{}}
	if !s.cfg.Enabled {
		return status, nil
	}
	var version struct {
		Version string `json:"version"`
	}
	if err := s.controllerJSON(ctx, http.MethodGet, "/version", nil, &version); err != nil {
		return nil, err
	}
	provider, err := s.providerState(ctx)
	if err != nil {
		return nil, err
	}
	groups, err := s.proxyGroups(ctx)
	if err != nil {
		return nil, err
	}
	status.Version, status.ProviderName, status.UpdatedAt = version.Version, s.cfg.ProviderName, provider.UpdatedAt
	status.Configured = provider.Configured
	status.SubscriptionConfigured, status.SubscriptionHost, err = s.subscriptionState()
	if err != nil {
		return nil, err
	}
	status.Nodes = provider.Nodes
	status.NodeCount = len(provider.Nodes)
	for _, node := range provider.Nodes {
		if node.Alive {
			status.AliveCount++
		}
	}
	for mode, group := range mihomoGroups {
		status.Modes[mode] = MihomoModeStatus{Mode: mode, Selection: groups[group]}
	}
	status.ProxyIDs, err = s.managedProxyIDs(ctx)
	return status, err
}

func (s *MihomoService) Workbench(ctx context.Context) (*MihomoWorkbench, error) {
	if s.resources == nil {
		return nil, infraerrors.ServiceUnavailable("MIHOMO_RESOURCES_UNAVAILABLE", "mihomo managed resources are not configured")
	}
	subscriptions, err := s.allMihomoSubscriptions(ctx, MihomoSubscriptionFilter{})
	if err != nil {
		return nil, err
	}
	nodes, err := s.allMihomoNodes(ctx, MihomoNodeFilter{IncludeRemoved: true})
	if err != nil {
		return nil, err
	}
	routes, err := s.allMihomoRoutes(ctx, MihomoRouteFilter{})
	if err != nil {
		return nil, err
	}

	workbench := &MihomoWorkbench{
		Status:        MihomoRuntimeStatus{Enabled: s.cfg.Enabled, Configured: len(subscriptions) > 0 && len(routes) > 0},
		Subscriptions: make([]MihomoWorkbenchSubscription, 0, len(subscriptions)),
		Nodes:         make([]MihomoWorkbenchNode, 0, len(nodes)),
		Routes:        make([]MihomoWorkbenchRoute, 0, len(routes)),
	}
	if _, _, readErr := s.readConfig(); readErr == nil {
		workbench.Status.ConfigValid = true
	}
	var version struct {
		Version string `json:"version"`
	}
	runtimeGroups := map[string]string{}
	if s.cfg.Enabled {
		if controllerErr := s.controllerJSON(ctx, http.MethodGet, "/version", nil, &version); controllerErr == nil {
			workbench.Status.ControllerConnected = true
			workbench.Status.Version = version.Version
			if groups, groupsErr := s.proxyGroups(ctx); groupsErr == nil {
				runtimeGroups = groups
			}
		}
	}
	s.mu.Lock()
	workbench.Status.GeneratedAt = s.generatedAt
	workbench.Status.LastReloadAt = s.reloadedAt
	workbench.Status.LastReloadError = s.reloadError
	s.mu.Unlock()

	subscriptionNameByID := make(map[int64]string, len(subscriptions))
	subscriptionProviderByID := make(map[int64]string, len(subscriptions))
	counts := make(map[int64][2]int, len(subscriptions))
	for _, node := range nodes {
		count := counts[node.SubscriptionID]
		count[0]++
		if node.Alive && node.UpstreamRemovedAt == nil && !node.Excluded {
			count[1]++
		}
		counts[node.SubscriptionID] = count
	}
	for _, subscription := range subscriptions {
		subscriptionNameByID[subscription.ID] = subscription.Name
		subscriptionProviderByID[subscription.ID] = subscription.ProviderKey
		count := counts[subscription.ID]
		maskedURL := ""
		if subscription.MaskedHost != "" {
			maskedURL = "https://" + subscription.MaskedHost + "/..."
		}
		workbench.Subscriptions = append(workbench.Subscriptions, MihomoWorkbenchSubscription{
			ID: subscription.ID, Name: subscription.Name, Enabled: subscription.Status == StatusActive,
			Status: subscription.Status, MaskedURL: maskedURL, SourceHost: subscription.MaskedHost,
			RefreshIntervalMinutes: subscription.RefreshIntervalSeconds / 60, NodeCount: count[0], AliveCount: count[1],
			UsedBytes: subscription.QuotaUsedBytes, TotalBytes: subscription.QuotaTotalBytes, ExpiresAt: subscription.ExpiresAt,
			LastRefreshedAt: subscription.LastRefreshedAt, LastError: subscription.LastError,
		})
	}

	nodeByID := make(map[int64]MihomoManagedNode, len(nodes))
	for _, node := range nodes {
		nodeByID[node.ID] = node
		lastSeen := node.LastSeenAt
		workbench.Nodes = append(workbench.Nodes, MihomoWorkbenchNode{
			ID: node.ID, Key: node.NodeKey, Name: node.OriginalName, DisplayName: node.DisplayName,
			SubscriptionID: node.SubscriptionID, SubscriptionName: subscriptionNameByID[node.SubscriptionID],
			Alive: node.Alive, Delay: node.DelayMS, Region: node.Region, Tags: node.Tags,
			Enabled: node.UpstreamRemovedAt == nil && !node.Excluded, Excluded: node.Excluded,
			LastSeenAt: &lastSeen, UpstreamRemovedAt: node.UpstreamRemovedAt,
		})
	}

	for _, route := range routes {
		routeNodes, listErr := s.resources.ListRouteNodes(ctx, route.ID)
		if listErr != nil {
			return nil, listErr
		}
		nodeIDs := make([]int64, 0, len(routeNodes))
		subscriptionIDs := make([]int64, 0, len(routeNodes))
		subscriptionNames := make([]string, 0, len(routeNodes))
		seenSubscriptions := make(map[int64]struct{}, len(routeNodes))
		for _, relation := range routeNodes {
			nodeIDs = append(nodeIDs, relation.NodeID)
			node, ok := nodeByID[relation.NodeID]
			if !ok {
				continue
			}
			if _, exists := seenSubscriptions[node.SubscriptionID]; exists {
				continue
			}
			seenSubscriptions[node.SubscriptionID] = struct{}{}
			subscriptionIDs = append(subscriptionIDs, node.SubscriptionID)
			subscriptionNames = append(subscriptionNames, subscriptionNameByID[node.SubscriptionID])
		}
		proxyID, accountCount := int64(0), int64(0)
		if route.ProxyID != nil {
			proxyID = *route.ProxyID
			if s.proxyRepo != nil {
				accountCount, err = s.proxyRepo.CountAccountsByProxyID(ctx, proxyID)
				if err != nil {
					return nil, err
				}
			}
		}
		currentNode := ""
		runtimeNode := runtimeGroups[fmt.Sprintf("SUB2API-ROUTE-%d", route.ID)]
		for _, relation := range routeNodes {
			node, ok := nodeByID[relation.NodeID]
			if !ok || node.UpstreamRemovedAt != nil || node.Excluded || isMihomoSubscriptionMetadataNode(node.OriginalName) {
				continue
			}
			configuredName := "[" + subscriptionProviderByID[node.SubscriptionID] + "] " + strings.TrimSpace(node.OriginalName)
			if runtimeNode != configuredName && runtimeNode != node.OriginalName {
				continue
			}
			currentNode = node.DisplayName
			if currentNode == "" {
				currentNode = node.OriginalName
			}
			break
		}
		if currentNode == "" && route.CurrentNodeID != nil {
			if node, ok := nodeByID[*route.CurrentNodeID]; ok && node.UpstreamRemovedAt == nil && !node.Excluded && !isMihomoSubscriptionMetadataNode(node.OriginalName) {
				currentNode = node.DisplayName
				if currentNode == "" {
					currentNode = node.OriginalName
				}
			}
		}
		health := "unknown"
		if route.ExitHealthy != nil {
			if *route.ExitHealthy {
				health = "healthy"
			} else {
				health = "failed"
			}
		}
		workbench.Routes = append(workbench.Routes, MihomoWorkbenchRoute{
			ID: route.ID, Name: route.Name, Kind: route.Kind, SubscriptionIDs: subscriptionIDs,
			SubscriptionNames: subscriptionNames, NodeIDs: nodeIDs, ListenerPort: route.ListenerPort,
			ProxyID: proxyID, Enabled: route.Status == StatusActive, CurrentNode: currentNode,
			ExitIP: route.ExitIP, Health: health, LatencyMS: route.ExitDelayMS,
			AccountCount: accountCount, LastCheckedAt: route.LastCheckedAt,
		})
	}
	return workbench, nil
}

func (s *MihomoService) subscriptionState() (bool, string, error) {
	_, cfg, err := s.readConfig()
	if err != nil {
		return false, "", err
	}
	provider := yamlMap(yamlMap(cfg["proxy-providers"])[s.cfg.ProviderName])
	rawURL := strings.TrimSpace(fmt.Sprint(provider["url"]))
	if rawURL == "" || rawURL == "<nil>" {
		return false, "", nil
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return true, "", nil
	}
	return true, parsed.Hostname(), nil
}

func (s *MihomoService) PrepareSubscriptionApproval(ctx context.Context, rawURL string) (*MihomoApprovalUpdate, string, error) {
	if !s.cfg.Enabled {
		return nil, "", infraerrors.ServiceUnavailable("MIHOMO_DISABLED", "mihomo integration is disabled")
	}
	rawURL = strings.TrimSpace(rawURL)
	if err := validateMihomoSubscriptionURL(ctx, rawURL); err != nil {
		return nil, "", err
	}
	if s.encryptor == nil {
		return nil, "", fmt.Errorf("secret encryptor is not configured")
	}
	ciphertext, err := s.encryptor.Encrypt(rawURL)
	if err != nil {
		return nil, "", err
	}
	u, _ := url.Parse(rawURL)
	host := u.Hostname()
	return &MihomoApprovalUpdate{Kind: "subscription", Subscription: ciphertext, SubscriptionHost: host}, host, nil
}

func (s *MihomoService) PrepareModeApproval(ctx context.Context, mode, selection string) (*MihomoApprovalUpdate, error) {
	if !s.cfg.Enabled {
		return nil, infraerrors.ServiceUnavailable("MIHOMO_DISABLED", "mihomo integration is disabled")
	}
	mode = strings.ToLower(strings.TrimSpace(mode))
	selection = strings.TrimSpace(selection)
	if err := s.validateModeSelection(ctx, mode, selection); err != nil {
		return nil, err
	}
	return &MihomoApprovalUpdate{Kind: "mode", Mode: mode, Selection: selection}, nil
}

func (s *MihomoService) PrepareRefreshApproval() (*MihomoApprovalUpdate, error) {
	if !s.cfg.Enabled {
		return nil, infraerrors.ServiceUnavailable("MIHOMO_DISABLED", "mihomo integration is disabled")
	}
	return &MihomoApprovalUpdate{Kind: "refresh"}, nil
}

func (s *MihomoService) ValidateApproval(ctx context.Context, resourceKey string, update MihomoApprovalUpdate) error {
	if !s.cfg.Enabled {
		return infraerrors.ServiceUnavailable("MIHOMO_DISABLED", "mihomo integration is disabled")
	}
	switch update.Kind {
	case "subscription":
		if err := s.ensureLegacyMutationAvailable(ctx); err != nil {
			return err
		}
		if resourceKey != "mihomo:subscription" || update.Subscription == "" || update.Mode != "" || update.Selection != "" || s.encryptor == nil {
			return infraerrors.BadRequest("INVALID_MIHOMO_UPDATE", "invalid mihomo subscription update")
		}
		rawURL, err := s.encryptor.Decrypt(update.Subscription)
		if err != nil {
			return infraerrors.BadRequest("INVALID_MIHOMO_UPDATE", "invalid mihomo subscription update")
		}
		if err = validateMihomoSubscriptionURL(ctx, rawURL); err != nil {
			return err
		}
		u, _ := url.Parse(rawURL)
		if update.SubscriptionHost != u.Hostname() {
			return infraerrors.BadRequest("INVALID_MIHOMO_UPDATE", "subscription host does not match the encrypted URL")
		}
		return nil
	case "mode":
		if err := s.ensureLegacyMutationAvailable(ctx); err != nil {
			return err
		}
		if resourceKey != "mihomo:"+update.Mode || update.Subscription != "" || update.SubscriptionHost != "" {
			return infraerrors.BadRequest("INVALID_MIHOMO_UPDATE", "invalid mihomo mode update")
		}
		return s.validateModeSelection(ctx, update.Mode, update.Selection)
	case "refresh":
		if err := s.ensureLegacyMutationAvailable(ctx); err != nil {
			return err
		}
		if resourceKey != "mihomo:refresh" || update.Mode != "" || update.Selection != "" || update.Subscription != "" || update.SubscriptionHost != "" {
			return infraerrors.BadRequest("INVALID_MIHOMO_UPDATE", "invalid mihomo refresh update")
		}
		return nil
	default:
		if isManagedMihomoApprovalKind(update.Kind) {
			return s.validateManagedApproval(ctx, resourceKey, update)
		}
		return infraerrors.BadRequest("INVALID_MIHOMO_UPDATE", "invalid mihomo update")
	}
}

func (s *MihomoService) Refresh(ctx context.Context) error {
	return s.controllerJSON(ctx, http.MethodPut, "/providers/proxies/"+url.PathEscape(s.cfg.ProviderName), map[string]any{}, nil)
}

func (s *MihomoService) ApprovalRevision(ctx context.Context, resourceKey string) (string, error) {
	if isManagedMihomoResourceKey(resourceKey) {
		return s.managedApprovalRevision(ctx, resourceKey)
	}
	if resourceKey == "mihomo:subscription" || resourceKey == "mihomo:refresh" {
		raw, cfg, err := s.readConfig()
		if err != nil {
			return "", err
		}
		provider := yamlMap(cfg["proxy-providers"])[s.cfg.ProviderName]
		return approvalDigest(resourceKey, raw, yamlMap(provider)["url"]), nil
	}
	mode := strings.TrimPrefix(resourceKey, "mihomo:")
	groups, err := s.proxyGroups(ctx)
	if err != nil {
		return "", err
	}
	group, ok := mihomoGroups[mode]
	if !ok {
		return "", infraerrors.BadRequest("INVALID_MIHOMO_TARGET", "invalid mihomo target")
	}
	return approvalDigest(resourceKey, groups[group]), nil
}

func (s *MihomoService) ApplyApproved(ctx context.Context, update MihomoApprovalUpdate) (func(bool) error, error) {
	if isManagedMihomoApprovalKind(update.Kind) {
		return s.applyManagedApproved(ctx, update)
	}
	if err := s.ensureLegacyMutationAvailable(ctx); err != nil {
		return nil, err
	}
	s.mu.Lock()
	lockTransferred := false
	defer func() {
		if !lockTransferred {
			s.mu.Unlock()
		}
	}()
	var rollback func() error
	var err error
	switch update.Kind {
	case "subscription":
		rollback, err = s.applySubscription(ctx, update.Subscription)
	case "mode":
		rollback, err = s.applyMode(ctx, update.Mode, update.Selection)
	case "refresh":
		err = s.Refresh(ctx)
	default:
		return nil, infraerrors.BadRequest("INVALID_MIHOMO_UPDATE", "invalid mihomo update")
	}
	if err != nil {
		return nil, err
	}
	if _, err = s.reconcileManagedProxies(ctx); err != nil {
		if rollback != nil {
			if rollbackErr := rollback(); rollbackErr != nil {
				slog.Error("mihomo rollback failed after proxy reconciliation error", "error", rollbackErr)
			}
		}
		return nil, err
	}
	lockTransferred = true
	return func(committed bool) error {
		defer s.mu.Unlock()
		if !committed && rollback != nil {
			return rollback()
		}
		return nil
	}, nil
}

// ApplyManagedRuntime loads a database-backed candidate config and keeps the
// config lock until the surrounding approval transaction is finalized.
func (s *MihomoService) ApplyManagedRuntime(ctx context.Context) (func(bool) error, error) {
	if !s.cfg.Enabled {
		return nil, infraerrors.ServiceUnavailable("MIHOMO_DISABLED", "mihomo integration is disabled")
	}
	if s.resources == nil {
		return nil, infraerrors.ServiceUnavailable("MIHOMO_RESOURCES_UNAVAILABLE", "mihomo managed resources are not configured")
	}
	s.mu.Lock()
	lockTransferred := false
	defer func() {
		if !lockTransferred {
			s.mu.Unlock()
		}
	}()

	oldRaw, base, err := s.readConfig()
	if err != nil {
		return nil, err
	}
	subscriptions, nodes, routes, err := s.managedConfigInputs(ctx)
	if err != nil {
		return nil, err
	}
	generated, err := generateManagedMihomoConfig(base, subscriptions, nodes, routes)
	if err != nil {
		return nil, infraerrors.BadRequest("INVALID_MIHOMO_CONFIG", err.Error())
	}
	newRaw, err := yaml.Marshal(generated)
	if err != nil {
		return nil, err
	}
	if err = s.loadManagedPayload(ctx, newRaw); err != nil {
		if rollbackErr := s.loadManagedPayload(context.Background(), oldRaw); rollbackErr != nil {
			slog.Error("mihomo controller rollback failed after managed config load error", "error", rollbackErr)
		}
		s.reloadError = err.Error()
		return nil, err
	}
	if err = s.reconcileManagedRouteSelections(ctx, routes); err != nil {
		if rollbackErr := s.loadManagedPayload(context.Background(), oldRaw); rollbackErr != nil {
			slog.Error("mihomo controller rollback failed after route selection reconciliation", "error", rollbackErr)
		}
		s.reloadError = err.Error()
		return nil, err
	}
	if err = atomicWriteFile(s.cfg.ConfigPath, newRaw, 0o600); err != nil {
		if rollbackErr := s.loadManagedPayload(context.Background(), oldRaw); rollbackErr != nil {
			slog.Error("mihomo controller rollback failed after managed config write error", "error", rollbackErr)
		}
		s.reloadError = err.Error()
		return nil, err
	}
	rollback := func() error {
		if writeErr := atomicWriteFile(s.cfg.ConfigPath, oldRaw, 0o600); writeErr != nil {
			return writeErr
		}
		return s.loadManagedPayload(context.Background(), oldRaw)
	}
	if _, err = s.reconcileRouteProxies(ctx); err != nil {
		if rollbackErr := rollback(); rollbackErr != nil {
			slog.Error("mihomo managed config rollback failed after proxy reconciliation", "error", rollbackErr)
		}
		s.reloadError = err.Error()
		return nil, err
	}
	now := time.Now().UTC()
	s.generatedAt, s.reloadedAt, s.reloadError = &now, &now, ""
	lockTransferred = true
	return func(committed bool) error {
		defer s.mu.Unlock()
		if committed {
			return nil
		}
		if rollbackErr := rollback(); rollbackErr != nil {
			s.reloadError = rollbackErr.Error()
			return rollbackErr
		}
		s.generatedAt, s.reloadedAt = nil, nil
		return nil
	}, nil
}

func (s *MihomoService) managedConfigInputs(ctx context.Context) ([]mihomoConfigSubscription, []mihomoConfigNode, []mihomoConfigRoute, error) {
	subscriptions, err := s.allMihomoSubscriptions(ctx, MihomoSubscriptionFilter{Status: StatusActive})
	if err != nil {
		return nil, nil, nil, err
	}
	nodes, err := s.allMihomoNodes(ctx, MihomoNodeFilter{})
	if err != nil {
		return nil, nil, nil, err
	}
	routes, err := s.allMihomoRoutes(ctx, MihomoRouteFilter{Status: StatusActive})
	if err != nil {
		return nil, nil, nil, err
	}

	configSubscriptions := make([]mihomoConfigSubscription, 0, len(subscriptions))
	for _, subscription := range subscriptions {
		if s.encryptor == nil {
			return nil, nil, nil, fmt.Errorf("secret encryptor is not configured")
		}
		rawURL, decryptErr := s.encryptor.Decrypt(string(subscription.URLCiphertext))
		if decryptErr != nil {
			return nil, nil, nil, fmt.Errorf("decrypt Mihomo subscription %d: %w", subscription.ID, decryptErr)
		}
		configSubscriptions = append(configSubscriptions, mihomoConfigSubscription{
			ID: subscription.ID, Name: subscription.Name, ProviderKey: subscription.ProviderKey,
			URL: rawURL, RefreshInterval: subscription.RefreshIntervalSeconds,
		})
	}
	configNodes := make([]mihomoConfigNode, 0, len(nodes))
	for _, node := range nodes {
		if node.UpstreamRemovedAt != nil {
			continue
		}
		configNodes = append(configNodes, mihomoConfigNode{
			ID: node.ID, SubscriptionID: node.SubscriptionID, Name: node.OriginalName, Excluded: node.Excluded,
		})
	}
	configRoutes := make([]mihomoConfigRoute, 0, len(routes))
	for _, route := range routes {
		relations, listErr := s.resources.ListRouteNodes(ctx, route.ID)
		if listErr != nil {
			return nil, nil, nil, listErr
		}
		nodeIDs := make([]int64, 0, len(relations))
		for _, relation := range relations {
			nodeIDs = append(nodeIDs, relation.NodeID)
		}
		configRoutes = append(configRoutes, mihomoConfigRoute{
			ID: route.ID, Name: route.Name, Kind: route.Kind, ListenerPort: route.ListenerPort,
			Selector: route.Selector, NodeIDs: nodeIDs,
		})
	}
	return configSubscriptions, configNodes, configRoutes, nil
}

var mihomoGroups = map[string]string{
	"automatic": "PROXY", "directional": "DIRECTIONAL", "dynamic": "DYNAMIC-ENTRY",
}

func (s *MihomoService) applyMode(ctx context.Context, mode, selection string) (func() error, error) {
	if err := s.validateModeSelection(ctx, mode, selection); err != nil {
		return nil, err
	}
	groups, err := s.proxyGroups(ctx)
	if err != nil {
		return nil, err
	}
	group := mihomoGroups[mode]
	old := groups[group]
	if err = s.selectProxy(ctx, group, selection); err != nil {
		return nil, err
	}
	return func() error { return s.selectProxy(context.Background(), group, old) }, nil
}

func (s *MihomoService) validateModeSelection(ctx context.Context, mode, selection string) error {
	switch mode {
	case "automatic":
		if selection != "AUTO" && selection != "FALLBACK" {
			return infraerrors.BadRequest("INVALID_MIHOMO_SELECTION", "automatic mode must be AUTO or FALLBACK")
		}
	case "dynamic":
		if selection != "DYNAMIC" && selection != "REJECT" {
			return infraerrors.BadRequest("INVALID_MIHOMO_SELECTION", "dynamic mode must be DYNAMIC or REJECT")
		}
	case "directional":
		if selection == "REJECT" {
			return nil
		}
		provider, err := s.providerState(ctx)
		if err != nil {
			return err
		}
		for _, node := range provider.Nodes {
			if node.Name == selection {
				return nil
			}
		}
		return infraerrors.BadRequest("MIHOMO_NODE_NOT_FOUND", "selected node is not in the provider")
	default:
		return infraerrors.BadRequest("INVALID_MIHOMO_MODE", "invalid mihomo mode")
	}
	return nil
}

func (s *MihomoService) applySubscription(ctx context.Context, ciphertext string) (func() error, error) {
	if s.encryptor == nil {
		return nil, fmt.Errorf("secret encryptor is not configured")
	}
	subscription, err := s.encryptor.Decrypt(ciphertext)
	if err != nil {
		return nil, err
	}
	if err = validateMihomoSubscriptionURL(ctx, subscription); err != nil {
		return nil, err
	}
	oldRaw, cfg, err := s.readConfig()
	if err != nil {
		return nil, err
	}
	providers := yamlMap(cfg["proxy-providers"])
	provider := yamlMap(providers[s.cfg.ProviderName])
	if provider == nil {
		return nil, fmt.Errorf("mihomo provider %q is missing", s.cfg.ProviderName)
	}
	provider["url"] = subscription
	providers[s.cfg.ProviderName] = provider
	cfg["proxy-providers"] = providers
	newRaw, err := yaml.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	if err = s.loadPayload(ctx, newRaw); err != nil {
		return nil, err
	}
	if err = atomicWriteFile(s.cfg.ConfigPath, newRaw, 0o600); err != nil {
		if rollbackErr := s.loadPayload(context.Background(), oldRaw); rollbackErr != nil {
			slog.Error("mihomo controller rollback failed after config write error", "error", rollbackErr)
		}
		return nil, err
	}
	rollback := func() error {
		if err := atomicWriteFile(s.cfg.ConfigPath, oldRaw, 0o600); err != nil {
			return err
		}
		return s.loadPayload(context.Background(), oldRaw)
	}
	if err = s.Refresh(ctx); err != nil {
		if rollbackErr := rollback(); rollbackErr != nil {
			slog.Error("mihomo rollback failed after provider refresh error", "error", rollbackErr)
		}
		return nil, err
	}
	providerState, err := s.providerState(ctx)
	if err != nil || len(providerState.Nodes) == 0 {
		if rollbackErr := rollback(); rollbackErr != nil {
			slog.Error("mihomo rollback failed after empty provider refresh", "error", rollbackErr)
		}
		return nil, fmt.Errorf("mihomo provider refresh returned no nodes")
	}
	return rollback, nil
}

func (s *MihomoService) selectProxy(ctx context.Context, group, selection string) error {
	return s.controllerJSON(ctx, http.MethodPut, "/proxies/"+url.PathEscape(group), map[string]string{"name": selection}, nil)
}

type mihomoProviderState struct {
	Configured       bool
	UpdatedAt        string
	SubscriptionInfo *mihomoSubscriptionInfo
	Nodes            []MihomoNode
}

type mihomoSubscriptionInfo struct {
	Download int64 `json:"Download"`
	Upload   int64 `json:"Upload"`
	Total    int64 `json:"Total"`
	Expire   int64 `json:"Expire"`
}

func (s *MihomoService) providerState(ctx context.Context) (*mihomoProviderState, error) {
	providers, err := s.providerStates(ctx)
	if err != nil {
		return nil, err
	}
	provider, ok := providers[s.cfg.ProviderName]
	if !ok {
		return &mihomoProviderState{}, nil
	}
	return &provider, nil
}

func (s *MihomoService) providerStates(ctx context.Context) (map[string]mihomoProviderState, error) {
	var raw struct {
		Providers map[string]struct {
			UpdatedAt        string                  `json:"updatedAt"`
			SubscriptionInfo *mihomoSubscriptionInfo `json:"subscriptionInfo"`
			Proxies          []struct {
				Name    string `json:"name"`
				Alive   bool   `json:"alive"`
				History []struct {
					Delay int `json:"delay"`
				} `json:"history"`
			} `json:"proxies"`
		} `json:"providers"`
	}
	if err := s.controllerJSON(ctx, http.MethodGet, "/providers/proxies", nil, &raw); err != nil {
		return nil, err
	}
	out := make(map[string]mihomoProviderState, len(raw.Providers))
	for providerName, provider := range raw.Providers {
		state := mihomoProviderState{Configured: true, UpdatedAt: provider.UpdatedAt, SubscriptionInfo: provider.SubscriptionInfo, Nodes: make([]MihomoNode, 0, len(provider.Proxies))}
		for _, node := range provider.Proxies {
			var delay *int
			if len(node.History) > 0 {
				latest := node.History[len(node.History)-1].Delay
				delay = &latest
			}
			state.Nodes = append(state.Nodes, MihomoNode{Name: node.Name, Alive: node.Alive, Delay: delay})
		}
		sort.Slice(state.Nodes, func(i, j int) bool { return state.Nodes[i].Name < state.Nodes[j].Name })
		out[providerName] = state
	}
	return out, nil
}

func (s *MihomoService) RefreshManagedSubscription(ctx context.Context, subscriptionID int64) error {
	if s.resources == nil {
		return infraerrors.ServiceUnavailable("MIHOMO_RESOURCES_UNAVAILABLE", "mihomo managed resources are not configured")
	}
	subscription, err := s.resources.GetSubscriptionByID(ctx, subscriptionID)
	if err != nil {
		return err
	}
	if subscription.Status != StatusActive {
		return infraerrors.BadRequest("MIHOMO_SUBSCRIPTION_DISABLED", "mihomo subscription is disabled")
	}
	refreshErr := s.controllerJSON(ctx, http.MethodPut, "/providers/proxies/"+url.PathEscape(subscription.ProviderKey), map[string]any{}, nil)
	if refreshErr != nil {
		subscription.LastError = refreshErr.Error()
		_ = s.resources.UpdateSubscription(ctx, subscription)
		return refreshErr
	}
	providers, refreshErr := s.providerStates(ctx)
	if refreshErr != nil {
		subscription.LastError = refreshErr.Error()
		_ = s.resources.UpdateSubscription(ctx, subscription)
		return refreshErr
	}
	provider, ok := providers[subscription.ProviderKey]
	if !ok {
		refreshErr = fmt.Errorf("mihomo provider %q is missing after refresh", subscription.ProviderKey)
		subscription.LastError = refreshErr.Error()
		_ = s.resources.UpdateSubscription(ctx, subscription)
		return refreshErr
	}
	observedAt := time.Now().UTC()
	prefix := "[" + subscription.ProviderKey + "] "
	managedNodes := make([]MihomoManagedNode, 0, len(provider.Nodes))
	for _, node := range provider.Nodes {
		originalName := strings.TrimSpace(strings.TrimPrefix(node.Name, prefix))
		if isMihomoSubscriptionMetadataNode(originalName) {
			continue
		}
		digest := sha256.Sum256([]byte(subscription.ProviderKey + "\x00" + originalName))
		managedNodes = append(managedNodes, MihomoManagedNode{
			SubscriptionID: subscription.ID, NodeKey: fmt.Sprintf("%x", digest[:]), OriginalName: originalName,
			DisplayName: originalName, Alive: node.Alive, DelayMS: node.Delay, Tags: []string{}, LastSeenAt: observedAt,
		})
	}
	if err = s.resources.SyncNodes(ctx, subscription.ID, managedNodes, observedAt); err != nil {
		return err
	}
	subscription.LastRefreshedAt = &observedAt
	subscription.LastError = ""
	if provider.SubscriptionInfo != nil {
		used := provider.SubscriptionInfo.Upload + provider.SubscriptionInfo.Download
		total := provider.SubscriptionInfo.Total
		subscription.QuotaUsedBytes, subscription.QuotaTotalBytes = &used, &total
		if provider.SubscriptionInfo.Expire > 0 {
			expiresAt := time.Unix(provider.SubscriptionInfo.Expire, 0).UTC()
			subscription.ExpiresAt = &expiresAt
		} else {
			subscription.ExpiresAt = nil
		}
	}
	return s.resources.UpdateSubscription(ctx, subscription)
}

func (s *MihomoService) proxyGroups(ctx context.Context) (map[string]string, error) {
	var raw struct {
		Proxies map[string]struct {
			Now string `json:"now"`
		} `json:"proxies"`
	}
	if err := s.controllerJSON(ctx, http.MethodGet, "/proxies", nil, &raw); err != nil {
		return nil, err
	}
	out := make(map[string]string, len(raw.Proxies))
	for name, group := range raw.Proxies {
		out[name] = group.Now
	}
	return out, nil
}

func (s *MihomoService) reconcileManagedRouteSelections(ctx context.Context, routes []mihomoConfigRoute) error {
	var raw struct {
		Proxies map[string]struct {
			Type string   `json:"type"`
			Now  string   `json:"now"`
			All  []string `json:"all"`
		} `json:"proxies"`
	}
	if err := s.controllerJSON(ctx, http.MethodGet, "/proxies", nil, &raw); err != nil {
		return err
	}
	for _, route := range routes {
		groupName := fmt.Sprintf("SUB2API-ROUTE-%d", route.ID)
		group, ok := raw.Proxies[groupName]
		if !ok || len(group.All) == 0 {
			return fmt.Errorf("mihomo route %d has no runtime nodes", route.ID)
		}
		if group.Type != "Selector" {
			continue
		}
		valid := false
		for _, candidate := range group.All {
			if candidate == group.Now {
				valid = true
				break
			}
		}
		if !valid {
			if err := s.selectProxy(ctx, groupName, group.All[0]); err != nil {
				return fmt.Errorf("select Mihomo route %d fallback: %w", route.ID, err)
			}
		}
	}
	return nil
}

func (s *MihomoService) controllerJSON(ctx context.Context, method, path string, input, output any) error {
	secret, err := s.controllerSecret()
	if err != nil {
		return err
	}
	var body io.Reader
	if input != nil {
		raw, marshalErr := json.Marshal(input)
		if marshalErr != nil {
			return marshalErr
		}
		body = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(s.cfg.ControllerURL, "/")+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+secret)
	if input != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return infraerrors.ServiceUnavailable("MIHOMO_UNAVAILABLE", "mihomo controller is unavailable")
	}
	defer func() { _ = resp.Body.Close() }()
	limited := io.LimitReader(resp.Body, mihomoResponseLimit+1)
	raw, err := io.ReadAll(limited)
	if err != nil {
		return err
	}
	if len(raw) > mihomoResponseLimit {
		return fmt.Errorf("mihomo response is too large")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		message := "mihomo rejected the operation"
		var detail struct {
			Message string `json:"message"`
		}
		if json.Unmarshal(raw, &detail) == nil {
			controllerMessage := logredact.RedactText(detail.Message, "secret", "payload", "token", "url")
			controllerMessage = strings.ReplaceAll(controllerMessage, secret, "***")
			controllerMessage = strings.TrimSpace(truncateString(controllerMessage, 512))
			if controllerMessage != "" {
				message += ": " + controllerMessage
			}
		}
		logPath, _, _ := strings.Cut(path, "?")
		slog.Warn("mihomo_controller_rejected_operation", "status_code", resp.StatusCode, "path", logPath, "message", message)
		return infraerrors.ServiceUnavailable("MIHOMO_ERROR", message)
	}
	if output != nil && len(raw) > 0 {
		return json.Unmarshal(raw, output)
	}
	return nil
}

func (s *MihomoService) readConfig() ([]byte, map[string]any, error) {
	raw, err := os.ReadFile(s.cfg.ConfigPath)
	if err != nil {
		return nil, nil, err
	}
	if len(raw) > 4<<20 {
		return nil, nil, fmt.Errorf("mihomo config is too large")
	}
	var cfg map[string]any
	if err = yaml.Unmarshal(raw, &cfg); err != nil {
		return nil, nil, err
	}
	return raw, cfg, nil
}

func (s *MihomoService) controllerSecret() (string, error) {
	_, cfg, err := s.readConfig()
	if err != nil {
		return "", err
	}
	secret := strings.TrimSpace(fmt.Sprint(cfg["secret"]))
	if secret == "" || secret == "<nil>" {
		return "", fmt.Errorf("mihomo controller secret is missing")
	}
	return secret, nil
}

func (s *MihomoService) loadPayload(ctx context.Context, raw []byte) error {
	return s.controllerJSON(ctx, http.MethodPut, "/configs?force=true", map[string]string{"payload": string(raw)}, nil)
}

func (s *MihomoService) loadManagedPayload(ctx context.Context, raw []byte) error {
	var reset map[string]any
	if err := yaml.Unmarshal(raw, &reset); err != nil {
		return err
	}
	reset["listeners"] = []any{}
	resetRaw, err := yaml.Marshal(reset)
	if err != nil {
		return err
	}
	if err = s.loadPayload(ctx, resetRaw); err != nil {
		return err
	}
	return s.loadPayload(ctx, raw)
}

func (s *MihomoService) reconcileRouteProxies(ctx context.Context) (map[string]int64, error) {
	if s.resources == nil {
		return nil, infraerrors.ServiceUnavailable("MIHOMO_RESOURCES_UNAVAILABLE", "mihomo managed resources are not configured")
	}
	routes, err := s.allMihomoRoutes(ctx, MihomoRouteFilter{IncludeDeleted: true})
	if err != nil {
		return nil, err
	}
	existing, err := s.proxyRepo.ListActive(ctx)
	if err != nil {
		return nil, err
	}
	result := make(map[string]int64, len(routes))
	for i := range routes {
		route := &routes[i]
		if route.DeletedAt != nil {
			if route.ProxyID == nil {
				continue
			}
			count, countErr := s.proxyRepo.CountAccountsByProxyID(ctx, *route.ProxyID)
			if countErr != nil {
				return nil, countErr
			}
			if count > 0 {
				return nil, infraerrors.Conflict("MIHOMO_ROUTE_IN_USE", "move bound accounts before deleting this route")
			}
			if deleteErr := s.proxyRepo.Delete(ctx, *route.ProxyID); deleteErr != nil {
				return nil, deleteErr
			}
			continue
		}
		active := route.Status == StatusActive && route.DeletedAt == nil
		if !active {
			if route.ProxyID == nil {
				continue
			}
			proxy, getErr := s.proxyRepo.GetByID(ctx, *route.ProxyID)
			if getErr != nil {
				return nil, getErr
			}
			if proxy.Status != StatusDisabled {
				proxy.Status = StatusDisabled
				if updateErr := s.proxyRepo.Update(ctx, proxy); updateErr != nil {
					return nil, updateErr
				}
			}
			continue
		}

		canonicalSource := fmt.Sprintf("mihomo:route:%d", route.ID)
		var managed *Proxy
		if route.ProxyID != nil {
			managed, err = s.proxyRepo.GetByID(ctx, *route.ProxyID)
			if err != nil {
				return nil, err
			}
		} else {
			legacySource := s.legacyManagedSource(route.ListenerPort)
			for j := range existing {
				candidate := &existing[j]
				if candidate.ManagedSource != nil && (*candidate.ManagedSource == canonicalSource || legacySource != "" && *candidate.ManagedSource == legacySource) {
					managed = candidate
					break
				}
			}
		}
		for j := range existing {
			candidate := &existing[j]
			if managed != nil && candidate.ID == managed.ID {
				continue
			}
			if candidate.Host == s.cfg.ProxyHost && candidate.Port == route.ListenerPort {
				return nil, infraerrors.Conflict("MIHOMO_PROXY_CONFLICT", "another proxy already uses the Mihomo route endpoint")
			}
		}
		if managed == nil {
			source := canonicalSource
			managed = &Proxy{
				Name: "Mihomo · " + route.Name, Protocol: "socks5h", Host: s.cfg.ProxyHost,
				Port: route.ListenerPort, Status: StatusActive, FallbackMode: FallbackModeNone,
				ExpiryWarnDays: 7, ManagedSource: &source,
			}
			if err = s.proxyRepo.Create(ctx, managed); err != nil {
				return nil, err
			}
			existing = append(existing, *managed)
		} else {
			changed := managed.Name != "Mihomo · "+route.Name || managed.Protocol != "socks5h" || managed.Host != s.cfg.ProxyHost || managed.Port != route.ListenerPort || managed.Status != StatusActive
			managed.Name, managed.Protocol, managed.Host, managed.Port, managed.Status = "Mihomo · "+route.Name, "socks5h", s.cfg.ProxyHost, route.ListenerPort, StatusActive
			if managed.ManagedSource == nil || !isLegacyMihomoSource(*managed.ManagedSource) {
				if managed.ManagedSource == nil || *managed.ManagedSource != canonicalSource {
					changed = true
				}
				managed.ManagedSource = &canonicalSource
			}
			if changed {
				if err = s.proxyRepo.Update(ctx, managed); err != nil {
					return nil, err
				}
			}
		}
		result[fmt.Sprint(route.ID)] = managed.ID
		if route.ProxyID == nil || *route.ProxyID != managed.ID {
			route.ProxyID = &managed.ID
			if err = s.resources.UpdateRoute(ctx, route); err != nil {
				return nil, err
			}
		}
	}
	return result, nil
}

func (s *MihomoService) legacyManagedSource(port int) string {
	switch port {
	case s.cfg.AutomaticPort:
		return "mihomo:automatic"
	case s.cfg.DirectionalPort:
		return "mihomo:directional"
	case s.cfg.DynamicPort:
		return "mihomo:dynamic"
	default:
		return ""
	}
}

func isLegacyMihomoSource(source string) bool {
	return source == "mihomo:automatic" || source == "mihomo:directional" || source == "mihomo:dynamic"
}

func (s *MihomoService) reconcileManagedProxies(ctx context.Context) (map[string]int64, error) {
	definitions := []struct {
		mode, name string
		port       int
	}{
		{"automatic", "Mihomo 自动", s.cfg.AutomaticPort}, {"directional", "Mihomo 定向", s.cfg.DirectionalPort}, {"dynamic", "Mihomo 动态", s.cfg.DynamicPort},
	}
	existing, err := s.proxyRepo.ListActive(ctx)
	if err != nil {
		return nil, err
	}
	result := make(map[string]int64, 3)
	for _, definition := range definitions {
		source := "mihomo:" + definition.mode
		var found *Proxy
		for i := range existing {
			if existing[i].ManagedSource != nil && *existing[i].ManagedSource == source {
				found = &existing[i]
				break
			}
			if existing[i].Host == s.cfg.ProxyHost && existing[i].Port == definition.port && existing[i].ManagedSource == nil {
				return nil, infraerrors.Conflict("MIHOMO_PROXY_CONFLICT", "an unmanaged proxy already uses a Mihomo endpoint")
			}
		}
		if found == nil {
			proxy := &Proxy{Name: definition.name, Protocol: "socks5h", Host: s.cfg.ProxyHost, Port: definition.port, Status: StatusActive, FallbackMode: FallbackModeNone, ExpiryWarnDays: 7, ManagedSource: &source}
			if err = s.proxyRepo.Create(ctx, proxy); err != nil {
				return nil, err
			}
			found = proxy
			existing = append(existing, *proxy)
		}
		result[definition.mode] = found.ID
	}
	return result, nil
}

func (s *MihomoService) managedProxyIDs(ctx context.Context) (map[string]int64, error) {
	existing, err := s.proxyRepo.ListActive(ctx)
	if err != nil {
		return nil, err
	}
	result := make(map[string]int64, 3)
	for i := range existing {
		if existing[i].ManagedSource == nil || !strings.HasPrefix(*existing[i].ManagedSource, "mihomo:") {
			continue
		}
		result[strings.TrimPrefix(*existing[i].ManagedSource, "mihomo:")] = existing[i].ID
	}
	return result, nil
}

func yamlMap(value any) map[string]any {
	if value == nil {
		return nil
	}
	if typed, ok := value.(map[string]any); ok {
		return typed
	}
	return nil
}

func atomicWriteFile(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".mihomo-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()
	if err = tmp.Chmod(mode); err == nil {
		_, err = tmp.Write(data)
	}
	if err == nil {
		err = tmp.Sync()
	}
	closeErr := tmp.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(tmpName, path)
}

func validateMihomoSubscriptionURL(ctx context.Context, raw string) error {
	if len(raw) == 0 || len(raw) > 4096 {
		return infraerrors.BadRequest("INVALID_SUBSCRIPTION_URL", "subscription URL is required and must not exceed 4096 characters")
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.Fragment != "" {
		return infraerrors.BadRequest("INVALID_SUBSCRIPTION_URL", "subscription URL must be an HTTPS URL without userinfo or fragment")
	}
	addresses, err := net.DefaultResolver.LookupIPAddr(ctx, u.Hostname())
	if err != nil {
		return infraerrors.BadRequest("INVALID_SUBSCRIPTION_URL", "subscription host could not be resolved")
	}
	for _, address := range addresses {
		ip := address.IP
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsUnspecified() {
			return infraerrors.BadRequest("INVALID_SUBSCRIPTION_URL", "subscription host resolves to a private address")
		}
	}
	return nil
}
