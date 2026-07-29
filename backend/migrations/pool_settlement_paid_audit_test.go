package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPoolSettlementPaidAuditMigration(t *testing.T) {
	content, err := FS.ReadFile("198_add_pool_settlement_paid_audit.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS paid_by_user_id BIGINT")
	require.Contains(t, sql, "paid_by_user_id = COALESCE(paid_by_user_id, locked_by_user_id, generated_by_user_id)")
	require.Contains(t, sql, "status = 'paid' AND paid_by_user_id IS NOT NULL AND paid_at IS NOT NULL")
	require.Contains(t, sql, "FOREIGN KEY (paid_by_user_id) REFERENCES users(id) ON DELETE RESTRICT")
}
