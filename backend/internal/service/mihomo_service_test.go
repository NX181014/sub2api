package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"gopkg.in/yaml.v3"
)

type mihomoProxyRepoStub struct {
	ProxyRepository
	items []Proxy
}

func (s *mihomoProxyRepoStub) ListActive(context.Context) ([]Proxy, error) {
	return append([]Proxy(nil), s.items...), nil
}

func (s *mihomoProxyRepoStub) Create(_ context.Context, proxy *Proxy) error {
	proxy.ID = int64(len(s.items) + 1)
	s.items = append(s.items, *proxy)
	return nil
}

func (s *mihomoProxyRepoStub) GetByID(_ context.Context, id int64) (*Proxy, error) {
	for i := range s.items {
		if s.items[i].ID == id {
			item := s.items[i]
			return &item, nil
		}
	}
	return nil, ErrProxyNotFound
}

func (s *mihomoProxyRepoStub) Update(_ context.Context, proxy *Proxy) error {
	for i := range s.items {
		if s.items[i].ID == proxy.ID {
			s.items[i] = *proxy
			return nil
		}
	}
	return ErrProxyNotFound
}

type mihomoResourceRepoStub struct {
	MihomoRepository
	subscriptions []MihomoSubscription
	nodes         []MihomoManagedNode
	routes        []MihomoRoute
	routeNodes    map[int64][]MihomoRouteNode
}

func (s *mihomoResourceRepoStub) ListSubscriptions(_ context.Context, params pagination.PaginationParams, _ MihomoSubscriptionFilter) ([]MihomoSubscription, *pagination.PaginationResult, error) {
	return paginateMihomoStub(s.subscriptions, params), nil, nil
}

func (s *mihomoResourceRepoStub) GetSubscriptionByID(_ context.Context, id int64) (*MihomoSubscription, error) {
	for i := range s.subscriptions {
		if s.subscriptions[i].ID == id {
			item := s.subscriptions[i]
			return &item, nil
		}
	}
	return nil, ErrMihomoSubscriptionNotFound
}

func (s *mihomoResourceRepoStub) UpdateSubscription(_ context.Context, item *MihomoSubscription) error {
	for i := range s.subscriptions {
		if s.subscriptions[i].ID == item.ID {
			s.subscriptions[i] = *item
			return nil
		}
	}
	return ErrMihomoSubscriptionNotFound
}

func (s *mihomoResourceRepoStub) SyncNodes(_ context.Context, _ int64, nodes []MihomoManagedNode, _ time.Time) error {
	s.nodes = append([]MihomoManagedNode(nil), nodes...)
	return nil
}

func (s *mihomoResourceRepoStub) ListNodes(_ context.Context, params pagination.PaginationParams, _ MihomoNodeFilter) ([]MihomoManagedNode, *pagination.PaginationResult, error) {
	return paginateMihomoStub(s.nodes, params), nil, nil
}

func (s *mihomoResourceRepoStub) GetNodeByID(_ context.Context, id int64) (*MihomoManagedNode, error) {
	for i := range s.nodes {
		if s.nodes[i].ID == id {
			item := s.nodes[i]
			return &item, nil
		}
	}
	return nil, ErrMihomoNodeNotFound
}

func (s *mihomoResourceRepoStub) UpdateNode(_ context.Context, item *MihomoManagedNode) error {
	for i := range s.nodes {
		if s.nodes[i].ID == item.ID {
			s.nodes[i] = *item
			return nil
		}
	}
	return ErrMihomoNodeNotFound
}

func (s *mihomoResourceRepoStub) ListRoutes(_ context.Context, params pagination.PaginationParams, filter MihomoRouteFilter) ([]MihomoRoute, *pagination.PaginationResult, error) {
	items := make([]MihomoRoute, 0, len(s.routes))
	for _, route := range s.routes {
		if filter.Status == "" || route.Status == filter.Status {
			items = append(items, route)
		}
	}
	return paginateMihomoStub(items, params), nil, nil
}

func paginateMihomoStub[T any](items []T, params pagination.PaginationParams) []T {
	start := params.Offset()
	if start >= len(items) {
		return []T{}
	}
	end := min(start+params.Limit(), len(items))
	return append([]T(nil), items[start:end]...)
}

func (s *mihomoResourceRepoStub) ListRouteNodes(_ context.Context, routeID int64) ([]MihomoRouteNode, error) {
	return append([]MihomoRouteNode(nil), s.routeNodes[routeID]...), nil
}

func (s *mihomoResourceRepoStub) UpdateRoute(_ context.Context, route *MihomoRoute) error {
	for i := range s.routes {
		if s.routes[i].ID == route.ID {
			s.routes[i] = *route
			return nil
		}
	}
	return ErrMihomoRouteNotFound
}

type mihomoPlainEncryptor struct{}

func (mihomoPlainEncryptor) Encrypt(value string) (string, error) { return value, nil }
func (mihomoPlainEncryptor) Decrypt(value string) (string, error) { return value, nil }

func TestPrepareManagedSubscriptionDeleteUsesApprovalKind(t *testing.T) {
	resources := &mihomoResourceRepoStub{subscriptions: []MihomoSubscription{{ID: 17, Name: "existing"}}}
	svc := NewMihomoService(&config.Config{Mihomo: config.MihomoConfig{Enabled: true}}, &mihomoProxyRepoStub{}, nil).SetResourceRepository(resources)

	update, err := svc.PrepareManagedSubscriptionApproval(context.Background(), MihomoApprovalSubscriptionDelete, 17, "", "", false, 0)
	if err != nil {
		t.Fatal(err)
	}
	if update.Kind != MihomoApprovalSubscriptionDelete || update.SubscriptionID != 17 {
		t.Fatalf("unexpected update: %+v", update)
	}
}

func TestPrepareManagedSubscriptionRefreshUsesApprovalKind(t *testing.T) {
	resources := &mihomoResourceRepoStub{subscriptions: []MihomoSubscription{{ID: 17, Name: "existing"}}}
	svc := NewMihomoService(&config.Config{Mihomo: config.MihomoConfig{Enabled: true}}, &mihomoProxyRepoStub{}, nil).SetResourceRepository(resources)

	update, err := svc.PrepareManagedSubscriptionApproval(context.Background(), MihomoApprovalSubscriptionRefresh, 17, "", "", false, 0)
	if err != nil {
		t.Fatal(err)
	}
	if update.Kind != MihomoApprovalSubscriptionRefresh || update.SubscriptionName != "existing" {
		t.Fatalf("unexpected update: %+v", update)
	}
}

func TestMihomoWorkbenchIncludesEveryNodeAndUpstreamTombstones(t *testing.T) {
	removedAt := time.Now().UTC()
	nodes := make([]MihomoManagedNode, 1001)
	for i := range nodes {
		nodes[i] = MihomoManagedNode{ID: int64(i + 1), SubscriptionID: 1, NodeKey: fmt.Sprint(i + 1), OriginalName: fmt.Sprintf("node-%d", i+1), LastSeenAt: removedAt}
	}
	nodes[len(nodes)-1].UpstreamRemovedAt = &removedAt
	resources := &mihomoResourceRepoStub{subscriptions: []MihomoSubscription{{ID: 1, Name: "all"}}, nodes: nodes}
	svc := NewMihomoService(&config.Config{}, &mihomoProxyRepoStub{}, nil).SetResourceRepository(resources)

	workbench, err := svc.Workbench(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(workbench.Nodes) != len(nodes) || workbench.Nodes[len(nodes)-1].UpstreamRemovedAt == nil {
		t.Fatalf("workbench nodes = %d, tombstone = %v", len(workbench.Nodes), workbench.Nodes[len(workbench.Nodes)-1].UpstreamRemovedAt)
	}
}

func TestMihomoControllerUsesConfiguredSecret(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-secret" {
			t.Fatalf("Authorization = %q", got)
		}
		_, _ = w.Write([]byte(`{"version":"v-test"}`))
	}))
	defer server.Close()

	configPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(configPath, []byte("secret: test-secret\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	svc := NewMihomoService(&config.Config{Mihomo: config.MihomoConfig{
		Enabled: true, ControllerURL: server.URL, ConfigPath: configPath,
	}}, nil, nil)
	var response struct {
		Version string `json:"version"`
	}
	if err := svc.controllerJSON(context.Background(), http.MethodGet, "/version", nil, &response); err != nil {
		t.Fatal(err)
	}
	if response.Version != "v-test" {
		t.Fatalf("version = %q", response.Version)
	}
}

func TestMihomoControllerReturnsSanitizedErrorMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"message":"config invalid; secret=test-secret; payload=private-config; url=https://example.com/sub?token=url-secret"}`))
	}))
	defer server.Close()

	configPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(configPath, []byte("secret: test-secret\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var logs strings.Builder
	previousLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	t.Cleanup(func() { slog.SetDefault(previousLogger) })

	svc := NewMihomoService(&config.Config{Mihomo: config.MihomoConfig{
		Enabled: true, ControllerURL: server.URL, ConfigPath: configPath,
	}}, nil, nil)
	err := svc.controllerJSON(context.Background(), http.MethodPut, "/configs?force=true", map[string]string{"payload": "private-config"}, nil)
	if infraerrors.Reason(err) != "MIHOMO_ERROR" || infraerrors.Message(err) != "mihomo rejected the operation: config invalid; secret=*** payload=*** url=***" {
		t.Fatalf("unexpected error: %v", err)
	}
	logOutput := logs.String()
	for _, want := range []string{"mihomo_controller_rejected_operation", "status_code=400", "path=/configs", "message="} {
		if !strings.Contains(logOutput, want) {
			t.Fatalf("log missing %q: %s", want, logOutput)
		}
	}
	for _, secret := range []string{"test-secret", "private-config", "url-secret", "force=true"} {
		if strings.Contains(logOutput, secret) || strings.Contains(infraerrors.Message(err), secret) {
			t.Fatalf("controller error leaked %q: error=%q log=%q", secret, infraerrors.Message(err), logOutput)
		}
	}
}

func TestMihomoSubscriptionRejectsNonHTTPS(t *testing.T) {
	if err := validateMihomoSubscriptionURL(context.Background(), "http://example.com/sub"); err == nil {
		t.Fatal("expected non-HTTPS subscription to be rejected")
	}
}

func TestMihomoStatusExposesSubscriptionHostWithoutSecretURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/version":
			_, _ = w.Write([]byte(`{"version":"v-test"}`))
		case "/providers/proxies":
			_, _ = w.Write([]byte(`{"providers":{"subscription":{"updatedAt":"now","proxies":[]}}}`))
		case "/proxies":
			_, _ = w.Write([]byte(`{"proxies":{}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	configPath := filepath.Join(t.TempDir(), "config.yaml")
	const subscriptionURL = "https://subscription.example/sub?token=secret-token"
	configYAML := "secret: test-secret\nproxy-providers:\n  subscription:\n    url: " + subscriptionURL + "\n"
	if err := os.WriteFile(configPath, []byte(configYAML), 0o600); err != nil {
		t.Fatal(err)
	}
	svc := NewMihomoService(&config.Config{Mihomo: config.MihomoConfig{
		Enabled: true, ControllerURL: server.URL, ConfigPath: configPath, ProviderName: "subscription",
	}}, &mihomoProxyRepoStub{}, nil)

	status, err := svc.Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !status.SubscriptionConfigured || status.SubscriptionHost != "subscription.example" {
		t.Fatalf("unexpected subscription status: %+v", status)
	}
	raw, err := json.Marshal(status)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), subscriptionURL) || strings.Contains(string(raw), "secret-token") {
		t.Fatalf("status leaked subscription credential: %s", raw)
	}
}

func TestMihomoApprovalLockHeldUntilFinalized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/providers/proxies/subscription" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	configPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(configPath, []byte("secret: test-secret\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	repo := &mihomoProxyRepoStub{}
	svc := NewMihomoService(&config.Config{Mihomo: config.MihomoConfig{
		Enabled: true, ControllerURL: server.URL, ConfigPath: configPath, ProviderName: "subscription",
		ProxyHost: "mihomo", AutomaticPort: 26781, DirectionalPort: 26782, DynamicPort: 26783,
	}}, repo, nil)

	finalize, err := svc.ApplyApproved(context.Background(), MihomoApprovalUpdate{Kind: "refresh"})
	if err != nil {
		t.Fatal(err)
	}
	started, acquired := make(chan struct{}), make(chan struct{})
	go func() {
		close(started)
		svc.mu.Lock()
		close(acquired)
		svc.mu.Unlock()
	}()
	<-started
	select {
	case <-acquired:
		t.Fatal("mihomo lock released before approval transaction finalized")
	case <-time.After(50 * time.Millisecond):
	}
	if err = finalize(true); err != nil {
		t.Fatal(err)
	}
	select {
	case <-acquired:
	case <-time.After(time.Second):
		t.Fatal("mihomo lock remained held after approval transaction finalized")
	}
}

func TestMihomoManagedRuntimeRollbackRestoresConfigAndLegacyProxy(t *testing.T) {
	var loadedPayloads []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/configs" {
			http.NotFound(w, r)
			return
		}
		var request struct {
			Payload string `json:"payload"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		loadedPayloads = append(loadedPayloads, request.Payload)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	configPath := filepath.Join(t.TempDir(), "config.yaml")
	oldConfig := "secret: test-secret\nexternal-controller: 0.0.0.0:26790\nproxy-groups:\n  - name: PROXY\n    type: select\n    proxies: [DIRECT]\n"
	if err := os.WriteFile(configPath, []byte(oldConfig), 0o600); err != nil {
		t.Fatal(err)
	}
	legacySource := "mihomo:automatic"
	proxyRepo := &mihomoProxyRepoStub{items: []Proxy{{
		ID: 44, Name: "Mihomo automatic", Protocol: "socks5h", Host: "mihomo", Port: 26781,
		Status: StatusActive, ManagedSource: &legacySource,
	}}}
	resources := &mihomoResourceRepoStub{
		subscriptions: []MihomoSubscription{{
			ID: 1, Name: "Primary", ProviderKey: "primary", URLCiphertext: []byte("https://subscription.example/sub"),
			RefreshIntervalSeconds: 600, Status: StatusActive,
		}},
		nodes: []MihomoManagedNode{{
			ID: 10, SubscriptionID: 1, NodeKey: "node-10", OriginalName: "Hong Kong", Alive: true,
		}},
		routes:     []MihomoRoute{{ID: 20, Name: "Automatic", Kind: "latency", ListenerPort: 26781, Status: StatusActive}},
		routeNodes: map[int64][]MihomoRouteNode{20: {{RouteID: 20, NodeID: 10, Priority: 1}}},
	}
	svc := NewMihomoService(&config.Config{Mihomo: config.MihomoConfig{
		Enabled: true, ControllerURL: server.URL, ConfigPath: configPath, ProxyHost: "mihomo",
		AutomaticPort: 26781, DirectionalPort: 26782, DynamicPort: 26783,
	}}, proxyRepo, mihomoPlainEncryptor{}).SetResourceRepository(resources)

	finalize, err := svc.ApplyManagedRuntime(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if resources.routes[0].ProxyID == nil || *resources.routes[0].ProxyID != 44 {
		t.Fatalf("legacy proxy was not reused: %+v", resources.routes[0])
	}
	generated, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(generated), "SUB2API-ROUTE-20") || strings.Contains(string(generated), "name: PROXY") {
		t.Fatalf("managed config was not installed:\n%s", generated)
	}
	if err = finalize(false); err != nil {
		t.Fatal(err)
	}
	restored, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(restored) != oldConfig {
		t.Fatalf("config was not restored:\n%s", restored)
	}
	if len(loadedPayloads) != 4 || loadedPayloads[1] == oldConfig || loadedPayloads[3] != oldConfig {
		t.Fatalf("controller payload sequence = %s", fmt.Sprint(loadedPayloads))
	}
	for _, index := range []int{0, 2} {
		var reset, full map[string]any
		if err = yaml.Unmarshal([]byte(loadedPayloads[index]), &reset); err != nil {
			t.Fatal(err)
		}
		if listeners, ok := reset["listeners"].([]any); !ok || len(listeners) != 0 {
			t.Fatalf("payload %d did not reset listeners: %#v", index, reset["listeners"])
		}
		if err = yaml.Unmarshal([]byte(loadedPayloads[index+1]), &full); err != nil {
			t.Fatal(err)
		}
		delete(reset, "listeners")
		delete(full, "listeners")
		if !reflect.DeepEqual(reset, full) {
			t.Fatalf("payload %d reset a different config", index)
		}
	}
	var applied map[string]any
	if err = yaml.Unmarshal([]byte(loadedPayloads[1]), &applied); err != nil {
		t.Fatal(err)
	}
	if listeners, ok := applied["listeners"].([]any); !ok || len(listeners) != 1 {
		t.Fatalf("managed payload did not apply its listener: %#v", applied["listeners"])
	}
}

func TestRefreshManagedSubscriptionKeepsUnknownDelayAndSkipsProviderMetadata(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPut && r.URL.Path == "/providers/proxies/primary":
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && r.URL.Path == "/providers/proxies":
			_, _ = w.Write([]byte(`{"providers":{"primary":{"proxies":[` +
				`{"name":"[primary] 剩余流量：995.36 GB","alive":true,"history":[]},` +
				`{"name":"[primary] 套餐到期: 长期有效","alive":true,"history":[]},` +
				`{"name":"[primary] 未检测节点","alive":false,"history":[]},` +
				`{"name":"[primary] 已检测节点","alive":true,"history":[{"delay":42}]},` +
				`{"name":"[primary] 剩余流量专线","alive":true,"history":[{"delay":55}]}` +
				`]}}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	configPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(configPath, []byte("secret: test-secret\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	resources := &mihomoResourceRepoStub{subscriptions: []MihomoSubscription{{
		ID: 1, Name: "Primary", ProviderKey: "primary", Status: StatusActive,
	}}}
	svc := NewMihomoService(&config.Config{Mihomo: config.MihomoConfig{
		Enabled: true, ControllerURL: server.URL, ConfigPath: configPath,
	}}, &mihomoProxyRepoStub{}, nil).SetResourceRepository(resources)

	if err := svc.RefreshManagedSubscription(context.Background(), 1); err != nil {
		t.Fatal(err)
	}
	if len(resources.nodes) != 3 {
		t.Fatalf("synced nodes = %+v", resources.nodes)
	}
	byName := make(map[string]MihomoManagedNode, len(resources.nodes))
	for _, node := range resources.nodes {
		byName[node.OriginalName] = node
	}
	if node, ok := byName["未检测节点"]; !ok || node.DelayMS != nil {
		t.Fatalf("unknown node = %+v", node)
	}
	if node := byName["已检测节点"]; node.DelayMS == nil || *node.DelayMS != 42 {
		t.Fatalf("measured node = %+v", node)
	}
	if node := byName["剩余流量专线"]; node.DelayMS == nil || *node.DelayMS != 55 {
		t.Fatalf("normal similarly named node = %+v", node)
	}
}

func TestManagedSubscriptionRefreshReloadsRuntime(t *testing.T) {
	configLoads := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPut && r.URL.Path == "/providers/proxies/primary":
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && r.URL.Path == "/providers/proxies":
			_, _ = w.Write([]byte(`{"providers":{"primary":{"proxies":[{"name":"[primary] Node","alive":true,"history":[{"delay":42}]}]}}}`))
		case r.Method == http.MethodPut && r.URL.Path == "/configs":
			configLoads++
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	configPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(configPath, []byte("secret: test-secret\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	resources := &mihomoResourceRepoStub{subscriptions: []MihomoSubscription{{
		ID: 1, Name: "Primary", ProviderKey: "primary", URLCiphertext: []byte("https://subscription.example/sub"),
		RefreshIntervalSeconds: 600, Status: StatusActive,
	}}}
	svc := NewMihomoService(&config.Config{Mihomo: config.MihomoConfig{
		Enabled: true, ControllerURL: server.URL, ConfigPath: configPath, ProxyHost: "mihomo",
	}}, &mihomoProxyRepoStub{}, mihomoPlainEncryptor{}).SetResourceRepository(resources)

	finalize, err := svc.applyManagedApproved(context.Background(), MihomoApprovalUpdate{
		Kind: MihomoApprovalSubscriptionRefresh, SubscriptionID: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if finalize == nil || configLoads != 2 {
		t.Fatalf("finalize = %v, config loads = %d", finalize != nil, configLoads)
	}
	if err = finalize(true); err != nil {
		t.Fatal(err)
	}
}

func TestManagedNodesHealthchecksProviderOnceAndSyncsHealth(t *testing.T) {
	healthchecks, providerRefreshes, configLoads := 0, 0, 0
	removedAt := time.Now().Add(-time.Hour)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/providers/proxies/primary/healthcheck":
			healthchecks++
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && r.URL.Path == "/providers/proxies":
			_, _ = w.Write([]byte(`{"providers":{"primary":{"proxies":[` +
				`{"name":"[primary] Node A","alive":true,"history":[{"delay":31}]},` +
				`{"name":"[primary] Node B","alive":false,"history":[{"delay":87}]},` +
				`{"name":"[primary] Removed","alive":true,"history":[{"delay":25}]}` +
				`]}}}`))
		case r.Method == http.MethodPut && r.URL.Path == "/providers/proxies/primary":
			providerRefreshes++
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodPut && r.URL.Path == "/configs":
			configLoads++
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	configPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(configPath, []byte("secret: test-secret\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	resources := &mihomoResourceRepoStub{
		subscriptions: []MihomoSubscription{{ID: 1, Name: "Primary", ProviderKey: "primary", Status: StatusActive}},
		nodes: []MihomoManagedNode{
			{ID: 10, SubscriptionID: 1, OriginalName: "Node A"},
			{ID: 11, SubscriptionID: 1, OriginalName: "Node B", Alive: true},
			{ID: 12, SubscriptionID: 1, OriginalName: "Removed", UpstreamRemovedAt: &removedAt},
		},
	}
	svc := NewMihomoService(&config.Config{Mihomo: config.MihomoConfig{
		Enabled: true, ControllerURL: server.URL, ConfigPath: configPath,
	}}, &mihomoProxyRepoStub{}, nil).SetResourceRepository(resources)

	if err := svc.TestManagedNodes(context.Background(), []int64{10, 11, 10}); err != nil {
		t.Fatal(err)
	}
	if healthchecks != 1 || providerRefreshes != 0 || configLoads != 0 {
		t.Fatalf("healthchecks=%d provider refreshes=%d config loads=%d", healthchecks, providerRefreshes, configLoads)
	}
	if got := resources.nodes[0]; !got.Alive || got.DelayMS == nil || *got.DelayMS != 31 {
		t.Fatalf("Node A health = %+v", got)
	}
	if got := resources.nodes[1]; got.Alive || got.DelayMS == nil || *got.DelayMS != 87 {
		t.Fatalf("Node B health = %+v", got)
	}
	if got := resources.nodes[2]; got.UpstreamRemovedAt == nil || got.DelayMS != nil {
		t.Fatalf("removed node was resurrected = %+v", got)
	}
}
