package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPoolCostOperationKeyMigration(t *testing.T) {
	content, err := FS.ReadFile("194_add_pool_cost_operation_key.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS operation_key VARCHAR(255)")
	require.Contains(t, sql, "ON account_cost_entries(created_by_user_id, operation_key)")
	require.Contains(t, sql, "WHERE operation_key IS NOT NULL")
}
