package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPoolSettlementAccountTraceMigration(t *testing.T) {
	content, err := FS.ReadFile("200_add_pool_settlement_account_trace.sql")
	require.NoError(t, err)
	sql := strings.Join(strings.Fields(string(content)), " ")
	require.Contains(t, sql, "CREATE TABLE IF NOT EXISTS pool_settlement_account_costs")
	require.Contains(t, sql, "UNIQUE(settlement_id, kind, cost_entry_id)")
	require.Contains(t, sql, "CREATE TABLE IF NOT EXISTS pool_settlement_account_lines")
	require.Contains(t, sql, "UNIQUE(settlement_id, account_id, user_id)")
	require.Contains(t, sql, "CHECK (trace_quality IN ('exact', 'derived', 'unavailable'))")
	require.Contains(t, sql, "DELETE FROM pool_settlements WHERE status='draft'")
}
