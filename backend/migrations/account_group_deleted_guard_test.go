package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDeletedAccountGroupGuardMigration(t *testing.T) {
	content, err := FS.ReadFile("201_guard_deleted_account_group_bindings.sql")
	require.NoError(t, err)
	sql := strings.Join(strings.Fields(string(content)), " ")
	require.Contains(t, sql, "WHERE id=NEW.account_id AND deleted_at IS NULL FOR UPDATE")
	require.Contains(t, sql, "BEFORE INSERT OR UPDATE OF account_id ON account_groups")
}
