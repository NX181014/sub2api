package admin

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestParsePoolPaidAtUsesSubmissionTimeWhenOnlyDateIsKnown(t *testing.T) {
	submittedAt := time.Date(2026, 8, 1, 8, 11, 56, 0, time.FixedZone("CST", 8*60*60))
	dateOnly := "2026-07-31"

	paidAt, err := parsePoolPaidAt(&dateOnly, submittedAt)

	require.NoError(t, err)
	require.Equal(t, "2026-07-31T08:11:56+08:00", paidAt.Format(time.RFC3339))
}
