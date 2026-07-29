package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPoolCostOrderAccountKeyMigration(t *testing.T) {
	content, err := FS.ReadFile("196_add_pool_cost_order_account_key.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS order_account_key VARCHAR(512)")
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS expected_token_count BIGINT")
	require.Contains(t, sql, "account_cost_entries_expected_token_count_positive")
	require.Contains(t, sql, "DROP CONSTRAINT IF EXISTS account_cost_entries_entry_type_check")
	require.Contains(t, sql, "'replacement_in', 'replacement_out', 'write_off'")
	require.Contains(t, sql, "ON account_cost_entries(order_account_key)")
	require.Contains(t, sql, "WHERE order_account_key IS NOT NULL")
	require.Contains(t, sql, "'UPDATE_ACCOUNT', 'VIEW_CREDENTIAL', 'DELETE_ACCOUNT'")
	require.Contains(t, sql, "idx_pool_approval_pending_delete_account")
}
