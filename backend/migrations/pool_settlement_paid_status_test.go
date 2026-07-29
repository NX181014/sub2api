package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPoolSettlementPaidStatusMigration(t *testing.T) {
	content, err := FS.ReadFile("197_add_pool_settlement_paid_status.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")
	require.Contains(t, sql, "CHECK (status IN ('draft', 'locked', 'paid'))")
	require.Contains(t, sql, "status IN ('locked', 'paid') AND locked_by_user_id IS NOT NULL")
	require.Contains(t, sql, "SET confirmation_status = 'pending', confirmed_by_user_id = NULL, confirmed_at = NULL")
	require.Contains(t, sql, "s.status = 'locked' AND l.net_amount_minor <> 0")
}
