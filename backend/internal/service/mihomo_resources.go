package service

import (
	"context"
	"encoding/json"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

var (
	ErrMihomoSubscriptionNotFound = infraerrors.NotFound("MIHOMO_SUBSCRIPTION_NOT_FOUND", "mihomo subscription not found")
	ErrMihomoSubscriptionExists   = infraerrors.Conflict("MIHOMO_SUBSCRIPTION_EXISTS", "mihomo subscription already exists")
	ErrMihomoNodeNotFound         = infraerrors.NotFound("MIHOMO_NODE_NOT_FOUND", "mihomo node not found")
	ErrMihomoRouteNotFound        = infraerrors.NotFound("MIHOMO_ROUTE_NOT_FOUND", "mihomo route not found")
	ErrMihomoRouteExists          = infraerrors.Conflict("MIHOMO_ROUTE_EXISTS", "mihomo route name, port, or proxy already exists")
)

type MihomoSubscription struct {
	ID                     int64      `json:"id"`
	Name                   string     `json:"name"`
	ProviderKey            string     `json:"provider_key"`
	URLCiphertext          []byte     `json:"-"`
	MaskedHost             string     `json:"masked_host"`
	RefreshIntervalSeconds int        `json:"refresh_interval_seconds"`
	QuotaTotalBytes        *int64     `json:"quota_total_bytes,omitempty"`
	QuotaUsedBytes         *int64     `json:"quota_used_bytes,omitempty"`
	ExpiresAt              *time.Time `json:"expires_at,omitempty"`
	Status                 string     `json:"status"`
	LastRefreshedAt        *time.Time `json:"last_refreshed_at,omitempty"`
	LastError              string     `json:"last_error,omitempty"`
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at"`
	DeletedAt              *time.Time `json:"deleted_at,omitempty"`
}

// MihomoManagedNode is the persisted node inventory. MihomoNode remains the
// lightweight controller response used by the existing status endpoint.
type MihomoManagedNode struct {
	ID                int64      `json:"id"`
	SubscriptionID    int64      `json:"subscription_id"`
	NodeKey           string     `json:"node_key"`
	OriginalName      string     `json:"original_name"`
	DisplayName       string     `json:"display_name"`
	Alive             bool       `json:"alive"`
	DelayMS           *int       `json:"delay_ms,omitempty"`
	Region            string     `json:"region"`
	Tags              []string   `json:"tags"`
	Excluded          bool       `json:"excluded"`
	LastSeenAt        time.Time  `json:"last_seen_at"`
	UpstreamRemovedAt *time.Time `json:"upstream_removed_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	DeletedAt         *time.Time `json:"deleted_at,omitempty"`
}

type MihomoRoute struct {
	ID            int64           `json:"id"`
	Name          string          `json:"name"`
	Kind          string          `json:"kind"`
	ListenerPort  int             `json:"listener_port"`
	ProxyID       *int64          `json:"proxy_id,omitempty"`
	Status        string          `json:"status"`
	Selector      json.RawMessage `json:"selector"`
	CurrentNodeID *int64          `json:"current_node_id,omitempty"`
	ExitIP        string          `json:"exit_ip,omitempty"`
	ExitHealthy   *bool           `json:"exit_healthy,omitempty"`
	ExitDelayMS   *int            `json:"exit_delay_ms,omitempty"`
	LastCheckedAt *time.Time      `json:"last_checked_at,omitempty"`
	LastError     string          `json:"last_error,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
	DeletedAt     *time.Time      `json:"deleted_at,omitempty"`
}

type MihomoRouteNode struct {
	ID        int64      `json:"id"`
	RouteID   int64      `json:"route_id"`
	NodeID    int64      `json:"node_id"`
	Priority  int        `json:"priority"`
	Weight    int        `json:"weight"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

type MihomoSubscriptionFilter struct {
	Status         string
	Search         string
	IncludeDeleted bool
}

type MihomoNodeFilter struct {
	SubscriptionID *int64
	Alive          *bool
	Excluded       *bool
	Search         string
	IncludeRemoved bool
	IncludeDeleted bool
}

type MihomoRouteFilter struct {
	SubscriptionID *int64
	Kind           string
	Status         string
	Search         string
	IncludeDeleted bool
}

type MihomoRepository interface {
	CreateSubscription(ctx context.Context, item *MihomoSubscription) error
	GetSubscriptionByID(ctx context.Context, id int64) (*MihomoSubscription, error)
	UpdateSubscription(ctx context.Context, item *MihomoSubscription) error
	DeleteSubscription(ctx context.Context, id int64) error
	ListSubscriptions(ctx context.Context, params pagination.PaginationParams, filter MihomoSubscriptionFilter) ([]MihomoSubscription, *pagination.PaginationResult, error)

	SyncNodes(ctx context.Context, subscriptionID int64, nodes []MihomoManagedNode, observedAt time.Time) error
	GetNodeByID(ctx context.Context, id int64) (*MihomoManagedNode, error)
	UpdateNode(ctx context.Context, item *MihomoManagedNode) error
	ListNodes(ctx context.Context, params pagination.PaginationParams, filter MihomoNodeFilter) ([]MihomoManagedNode, *pagination.PaginationResult, error)

	CreateRoute(ctx context.Context, item *MihomoRoute) error
	GetRouteByID(ctx context.Context, id int64) (*MihomoRoute, error)
	UpdateRoute(ctx context.Context, item *MihomoRoute) error
	DeleteRoute(ctx context.Context, id int64) error
	ListRoutes(ctx context.Context, params pagination.PaginationParams, filter MihomoRouteFilter) ([]MihomoRoute, *pagination.PaginationResult, error)
	ReplaceRouteNodes(ctx context.Context, routeID int64, nodes []MihomoRouteNode) error
	ListRouteNodes(ctx context.Context, routeID int64) ([]MihomoRouteNode, error)
}
