package repository

import (
	"strings"
	"testing"
)

func TestAccountIntakeIsFirstWriteOnly(t *testing.T) {
	for _, guard := range []string{
		"COALESCE(BTRIM(provider_identity), '') = ''",
		"contributor_user_id IS NULL",
		"cost_sharing_enabled = FALSE",
		"NOT EXISTS (SELECT 1 FROM account_cost_entries",
	} {
		if !strings.Contains(accountIntakeUpdateSQL, guard) {
			t.Fatalf("account intake first-write guard missing %q", guard)
		}
	}
}
