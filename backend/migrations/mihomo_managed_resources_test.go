package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMihomoManagedResourcesMigration(t *testing.T) {
	content, err := FS.ReadFile("203_mihomo_managed_resources.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")
	for _, table := range []string{
		"mihomo_subscriptions", "mihomo_nodes", "mihomo_routes", "mihomo_route_nodes",
	} {
		require.Contains(t, sql, "CREATE TABLE IF NOT EXISTS "+table)
	}
	require.Contains(t, sql, "url_ciphertext BYTEA NOT NULL")
	require.NotContains(t, sql, "subscription_url")
	require.Contains(t, sql, "UNIQUE (subscription_id, node_key)")
	require.Contains(t, sql, "idx_mihomo_routes_listener_port_live")
	require.Contains(t, sql, "idx_mihomo_routes_proxy_live")
	require.Contains(t, sql, "upstream_removed_at TIMESTAMPTZ")
	require.Contains(t, sql, "WHERE deleted_at IS NULL")
	require.Contains(t, sql, "CHECK (kind IN ('dedicated', 'automatic', 'latency', 'fallback', 'dynamic', 'directional'))")
}

func TestCleanupDeletedMihomoRouteProxiesMigration(t *testing.T) {
	content, err := FS.ReadFile("204_cleanup_deleted_mihomo_route_proxies.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")
	require.Contains(t, sql, "r.deleted_at IS NOT NULL")
	require.Contains(t, sql, "p.managed_source LIKE 'mihomo:%'")
	require.Contains(t, sql, "NOT EXISTS ( SELECT 1 FROM accounts AS a WHERE a.proxy_id = p.id AND a.deleted_at IS NULL )")
}
