package service

import (
	"bytes"
	"context"
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
	"gopkg.in/yaml.v3"
)

const mihomoResponseLimit = 8 << 20

type MihomoNode struct {
	Name  string `json:"name"`
	Alive bool   `json:"alive"`
	Delay int    `json:"delay,omitempty"`
}

type MihomoModeStatus struct {
	Mode      string `json:"mode"`
	Selection string `json:"selection"`
}

type MihomoStatus struct {
	Enabled      bool                        `json:"enabled"`
	Version      string                      `json:"version,omitempty"`
	Configured   bool                        `json:"configured"`
	ProviderName string                      `json:"provider_name,omitempty"`
	UpdatedAt    string                      `json:"updated_at,omitempty"`
	NodeCount    int                         `json:"node_count"`
	AliveCount   int                         `json:"alive_count"`
	Nodes        []MihomoNode                `json:"nodes"`
	Modes        map[string]MihomoModeStatus `json:"modes"`
	ProxyIDs     map[string]int64            `json:"proxy_ids"`
}

type MihomoService struct {
	cfg       config.MihomoConfig
	proxyRepo ProxyRepository
	encryptor SecretEncryptor
	client    *http.Client
	mu        sync.Mutex
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
		if resourceKey != "mihomo:"+update.Mode || update.Subscription != "" || update.SubscriptionHost != "" {
			return infraerrors.BadRequest("INVALID_MIHOMO_UPDATE", "invalid mihomo mode update")
		}
		return s.validateModeSelection(ctx, update.Mode, update.Selection)
	case "refresh":
		if resourceKey != "mihomo:refresh" || update.Mode != "" || update.Selection != "" || update.Subscription != "" || update.SubscriptionHost != "" {
			return infraerrors.BadRequest("INVALID_MIHOMO_UPDATE", "invalid mihomo refresh update")
		}
		return nil
	default:
		return infraerrors.BadRequest("INVALID_MIHOMO_UPDATE", "invalid mihomo update")
	}
}

func (s *MihomoService) Refresh(ctx context.Context) error {
	return s.controllerJSON(ctx, http.MethodPut, "/providers/proxies/"+url.PathEscape(s.cfg.ProviderName), map[string]any{}, nil)
}

func (s *MihomoService) ApprovalRevision(ctx context.Context, resourceKey string) (string, error) {
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
	s.mu.Lock()
	lockTransferred := false
	defer func() {
		if !lockTransferred {
			s.mu.Unlock()
		}
	}()
	var rollback func() error
	var err error
	if update.Kind == "subscription" {
		rollback, err = s.applySubscription(ctx, update.Subscription)
	} else if update.Kind == "mode" {
		rollback, err = s.applyMode(ctx, update.Mode, update.Selection)
	} else if update.Kind == "refresh" {
		err = s.Refresh(ctx)
	} else {
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
	Configured bool
	UpdatedAt  string
	Nodes      []MihomoNode
}

func (s *MihomoService) providerState(ctx context.Context) (*mihomoProviderState, error) {
	var raw struct {
		Providers map[string]struct {
			UpdatedAt string `json:"updatedAt"`
			Proxies   []struct {
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
	p, ok := raw.Providers[s.cfg.ProviderName]
	if !ok {
		return &mihomoProviderState{}, nil
	}
	out := &mihomoProviderState{Configured: true, UpdatedAt: p.UpdatedAt, Nodes: make([]MihomoNode, 0, len(p.Proxies))}
	for _, node := range p.Proxies {
		delay := 0
		if len(node.History) > 0 {
			delay = node.History[len(node.History)-1].Delay
		}
		out.Nodes = append(out.Nodes, MihomoNode{Name: node.Name, Alive: node.Alive, Delay: delay})
	}
	sort.Slice(out.Nodes, func(i, j int) bool { return out.Nodes[i].Name < out.Nodes[j].Name })
	return out, nil
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
		return infraerrors.ServiceUnavailable("MIHOMO_ERROR", "mihomo rejected the operation")
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
