package admin

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestMihomoManagedResourceKey(t *testing.T) {
	tests := []struct {
		update *service.MihomoApprovalUpdate
		want   string
	}{
		{&service.MihomoApprovalUpdate{Kind: service.MihomoApprovalSubscriptionCreate}, "mihomo:subscription:new"},
		{&service.MihomoApprovalUpdate{Kind: service.MihomoApprovalSubscriptionDelete, SubscriptionID: 17}, "mihomo:subscription:17"},
		{&service.MihomoApprovalUpdate{Kind: service.MihomoApprovalSubscriptionRefresh, SubscriptionID: 17}, "mihomo:subscription:17"},
		{&service.MihomoApprovalUpdate{Kind: service.MihomoApprovalRouteUpdate, RouteID: 23}, "mihomo:route:23"},
		{&service.MihomoApprovalUpdate{Kind: service.MihomoApprovalNodeAction, NodeIDs: []int64{3, 4}}, "mihomo:nodes"},
	}
	for _, test := range tests {
		require.Equal(t, test.want, mihomoManagedResourceKey(test.update))
	}
}
