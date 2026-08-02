package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
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
		svc.mu.Unlock()
		close(acquired)
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
