package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type mihomoRepository struct{ db *sql.DB }

func NewMihomoRepository(db *sql.DB) service.MihomoRepository { return &mihomoRepository{db: db} }

type mihomoScanner interface{ Scan(...any) error }

type mihomoExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

type mihomoRow struct {
	rows *sql.Rows
	err  error
}

func (row *mihomoRow) Scan(dest ...any) error {
	if row.err != nil {
		return row.err
	}
	defer func() { _ = row.rows.Close() }()
	if !row.rows.Next() {
		if err := row.rows.Err(); err != nil {
			return err
		}
		return sql.ErrNoRows
	}
	if err := row.rows.Scan(dest...); err != nil {
		return err
	}
	return row.rows.Close()
}

func queryMihomoRow(ctx context.Context, exec mihomoExecutor, query string, args ...any) mihomoScanner {
	rows, err := exec.QueryContext(ctx, query, args...)
	return &mihomoRow{rows: rows, err: err}
}

func (r *mihomoRepository) executor(ctx context.Context) mihomoExecutor {
	if tx := dbent.TxFromContext(ctx); tx != nil {
		return tx.Client()
	}
	return r.db
}

const mihomoSubscriptionColumns = `id, name, provider_key, url_ciphertext, masked_host,
refresh_interval_seconds, quota_total_bytes, quota_used_bytes, expires_at, status,
last_refreshed_at, COALESCE(last_error, ''), created_at, updated_at, deleted_at`

func scanMihomoSubscription(row mihomoScanner) (*service.MihomoSubscription, error) {
	var item service.MihomoSubscription
	err := row.Scan(
		&item.ID, &item.Name, &item.ProviderKey, &item.URLCiphertext, &item.MaskedHost,
		&item.RefreshIntervalSeconds, &item.QuotaTotalBytes, &item.QuotaUsedBytes, &item.ExpiresAt, &item.Status,
		&item.LastRefreshedAt, &item.LastError, &item.CreatedAt, &item.UpdatedAt, &item.DeletedAt,
	)
	return &item, err
}

func (r *mihomoRepository) CreateSubscription(ctx context.Context, item *service.MihomoSubscription) error {
	err := queryMihomoRow(ctx, r.executor(ctx), `
INSERT INTO mihomo_subscriptions (
    name, provider_key, url_ciphertext, masked_host, refresh_interval_seconds,
    quota_total_bytes, quota_used_bytes, expires_at, status, last_refreshed_at, last_error
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,NULLIF($11,''))
RETURNING `+mihomoSubscriptionColumns,
		item.Name, item.ProviderKey, item.URLCiphertext, item.MaskedHost, item.RefreshIntervalSeconds,
		item.QuotaTotalBytes, item.QuotaUsedBytes, item.ExpiresAt, item.Status, item.LastRefreshedAt, item.LastError,
	).Scan(
		&item.ID, &item.Name, &item.ProviderKey, &item.URLCiphertext, &item.MaskedHost,
		&item.RefreshIntervalSeconds, &item.QuotaTotalBytes, &item.QuotaUsedBytes, &item.ExpiresAt, &item.Status,
		&item.LastRefreshedAt, &item.LastError, &item.CreatedAt, &item.UpdatedAt, &item.DeletedAt,
	)
	if isUniqueViolation(err) {
		return service.ErrMihomoSubscriptionExists
	}
	if err != nil {
		return fmt.Errorf("create mihomo subscription: %w", err)
	}
	return nil
}

func (r *mihomoRepository) GetSubscriptionByID(ctx context.Context, id int64) (*service.MihomoSubscription, error) {
	item, err := scanMihomoSubscription(queryMihomoRow(ctx, r.executor(ctx),
		`SELECT `+mihomoSubscriptionColumns+` FROM mihomo_subscriptions WHERE id=$1 AND deleted_at IS NULL`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrMihomoSubscriptionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get mihomo subscription: %w", err)
	}
	return item, nil
}

func (r *mihomoRepository) UpdateSubscription(ctx context.Context, item *service.MihomoSubscription) error {
	updated, err := scanMihomoSubscription(queryMihomoRow(ctx, r.executor(ctx), `
UPDATE mihomo_subscriptions SET
    name=$2, provider_key=$3, url_ciphertext=$4, masked_host=$5, refresh_interval_seconds=$6,
    quota_total_bytes=$7, quota_used_bytes=$8, expires_at=$9, status=$10,
    last_refreshed_at=$11, last_error=NULLIF($12,''), updated_at=NOW()
WHERE id=$1 AND deleted_at IS NULL
RETURNING `+mihomoSubscriptionColumns,
		item.ID, item.Name, item.ProviderKey, item.URLCiphertext, item.MaskedHost, item.RefreshIntervalSeconds,
		item.QuotaTotalBytes, item.QuotaUsedBytes, item.ExpiresAt, item.Status, item.LastRefreshedAt, item.LastError))
	if errors.Is(err, sql.ErrNoRows) {
		return service.ErrMihomoSubscriptionNotFound
	}
	if isUniqueViolation(err) {
		return service.ErrMihomoSubscriptionExists
	}
	if err != nil {
		return fmt.Errorf("update mihomo subscription: %w", err)
	}
	*item = *updated
	return nil
}

func (r *mihomoRepository) DeleteSubscription(ctx context.Context, id int64) error {
	return r.inMihomoTx(ctx, func(tx mihomoExecutor) error {
		result, err := tx.ExecContext(ctx, `
UPDATE mihomo_subscriptions SET status='disabled', deleted_at=NOW(), updated_at=NOW()
WHERE id=$1 AND deleted_at IS NULL`, id)
		if err != nil {
			return fmt.Errorf("delete mihomo subscription: %w", err)
		}
		affected, _ := result.RowsAffected()
		if affected == 0 {
			return service.ErrMihomoSubscriptionNotFound
		}
		if _, err = tx.ExecContext(ctx, `
UPDATE mihomo_route_nodes SET deleted_at=NOW(), updated_at=NOW()
WHERE deleted_at IS NULL AND node_id IN (
    SELECT id FROM mihomo_nodes WHERE subscription_id=$1 AND deleted_at IS NULL
)`, id); err != nil {
			return fmt.Errorf("delete subscription route nodes: %w", err)
		}
		if _, err = tx.ExecContext(ctx, `
UPDATE mihomo_nodes SET alive=FALSE, deleted_at=NOW(), updated_at=NOW()
WHERE subscription_id=$1 AND deleted_at IS NULL`, id); err != nil {
			return fmt.Errorf("delete subscription nodes: %w", err)
		}
		return nil
	})
}

func (r *mihomoRepository) ListSubscriptions(ctx context.Context, params pagination.PaginationParams, filter service.MihomoSubscriptionFilter) ([]service.MihomoSubscription, *pagination.PaginationResult, error) {
	where := []string{"1=1"}
	args := make([]any, 0, 5)
	if !filter.IncludeDeleted {
		where = append(where, "deleted_at IS NULL")
	}
	if filter.Status != "" {
		args = append(args, filter.Status)
		where = append(where, fmt.Sprintf("status=$%d", len(args)))
	}
	if filter.Search != "" {
		args = append(args, "%"+escapeLike(filter.Search)+"%")
		where = append(where, fmt.Sprintf("(name ILIKE $%d ESCAPE '\\' OR masked_host ILIKE $%d ESCAPE '\\')", len(args), len(args)))
	}
	clause := strings.Join(where, " AND ")
	exec := r.executor(ctx)
	var total int64
	if err := queryMihomoRow(ctx, exec, `SELECT COUNT(*) FROM mihomo_subscriptions WHERE `+clause, args...).Scan(&total); err != nil {
		return nil, nil, fmt.Errorf("count mihomo subscriptions: %w", err)
	}
	args = append(args, params.Limit(), params.Offset())
	rows, err := exec.QueryContext(ctx, `SELECT `+mihomoSubscriptionColumns+`
FROM mihomo_subscriptions WHERE `+clause+` ORDER BY `+mihomoSubscriptionOrder(params)+fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(args)-1, len(args)), args...)
	if err != nil {
		return nil, nil, fmt.Errorf("list mihomo subscriptions: %w", err)
	}
	defer func() { _ = rows.Close() }()
	items := make([]service.MihomoSubscription, 0)
	for rows.Next() {
		item, scanErr := scanMihomoSubscription(rows)
		if scanErr != nil {
			return nil, nil, fmt.Errorf("scan mihomo subscription: %w", scanErr)
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterate mihomo subscriptions: %w", err)
	}
	return items, mihomoPagination(total, params), nil
}

const mihomoNodeColumns = `id, subscription_id, node_key, original_name, display_name, alive,
delay_ms, region, tags, excluded, last_seen_at, upstream_removed_at, created_at, updated_at, deleted_at`

func scanMihomoNode(row mihomoScanner) (*service.MihomoManagedNode, error) {
	var item service.MihomoManagedNode
	err := row.Scan(
		&item.ID, &item.SubscriptionID, &item.NodeKey, &item.OriginalName, &item.DisplayName, &item.Alive,
		&item.DelayMS, &item.Region, pq.Array(&item.Tags), &item.Excluded, &item.LastSeenAt,
		&item.UpstreamRemovedAt, &item.CreatedAt, &item.UpdatedAt, &item.DeletedAt,
	)
	return &item, err
}

func (r *mihomoRepository) SyncNodes(ctx context.Context, subscriptionID int64, nodes []service.MihomoManagedNode, observedAt time.Time) error {
	return r.inMihomoTx(ctx, func(tx mihomoExecutor) error {
		var lockedID int64
		if err := queryMihomoRow(ctx, tx, `SELECT id FROM mihomo_subscriptions WHERE id=$1 AND deleted_at IS NULL FOR UPDATE`, subscriptionID).Scan(&lockedID); errors.Is(err, sql.ErrNoRows) {
			return service.ErrMihomoSubscriptionNotFound
		} else if err != nil {
			return fmt.Errorf("lock mihomo subscription: %w", err)
		}
		for i := range nodes {
			node := &nodes[i]
			if node.DisplayName == "" {
				node.DisplayName = node.OriginalName
			}
			_, err := tx.ExecContext(ctx, `
INSERT INTO mihomo_nodes (
    subscription_id, node_key, original_name, display_name, alive, delay_ms,
    region, tags, excluded, last_seen_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
ON CONFLICT (subscription_id, node_key) DO UPDATE SET
    original_name=EXCLUDED.original_name,
    display_name=CASE WHEN mihomo_nodes.display_name=mihomo_nodes.original_name
                      THEN EXCLUDED.display_name ELSE mihomo_nodes.display_name END,
    alive=EXCLUDED.alive, delay_ms=EXCLUDED.delay_ms, region=EXCLUDED.region,
    tags=EXCLUDED.tags, last_seen_at=EXCLUDED.last_seen_at,
	upstream_removed_at=NULL, deleted_at=NULL, updated_at=NOW()
WHERE GREATEST(
    mihomo_nodes.last_seen_at,
    COALESCE(mihomo_nodes.upstream_removed_at, '-infinity'::timestamptz)
) <= EXCLUDED.last_seen_at`,
				subscriptionID, node.NodeKey, node.OriginalName, node.DisplayName, node.Alive,
				node.DelayMS, node.Region, pq.Array(node.Tags), node.Excluded, observedAt)
			if err != nil {
				return fmt.Errorf("upsert mihomo node %q: %w", node.NodeKey, err)
			}
		}
		if _, err := tx.ExecContext(ctx, `
UPDATE mihomo_nodes SET alive=FALSE, upstream_removed_at=$2, updated_at=NOW()
WHERE subscription_id=$1 AND deleted_at IS NULL AND upstream_removed_at IS NULL AND last_seen_at < $2`, subscriptionID, observedAt); err != nil {
			return fmt.Errorf("tombstone missing mihomo nodes: %w", err)
		}
		return nil
	})
}

func (r *mihomoRepository) GetNodeByID(ctx context.Context, id int64) (*service.MihomoManagedNode, error) {
	item, err := scanMihomoNode(queryMihomoRow(ctx, r.executor(ctx),
		`SELECT `+mihomoNodeColumns+` FROM mihomo_nodes WHERE id=$1 AND deleted_at IS NULL`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrMihomoNodeNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get mihomo node: %w", err)
	}
	return item, nil
}

func (r *mihomoRepository) UpdateNode(ctx context.Context, item *service.MihomoManagedNode) error {
	updated, err := scanMihomoNode(queryMihomoRow(ctx, r.executor(ctx), `
UPDATE mihomo_nodes SET display_name=$2, alive=$3, delay_ms=$4, region=$5, tags=$6,
    excluded=$7, last_seen_at=$8, upstream_removed_at=$9, updated_at=NOW()
WHERE id=$1 AND deleted_at IS NULL
RETURNING `+mihomoNodeColumns,
		item.ID, item.DisplayName, item.Alive, item.DelayMS, item.Region, pq.Array(item.Tags),
		item.Excluded, item.LastSeenAt, item.UpstreamRemovedAt))
	if errors.Is(err, sql.ErrNoRows) {
		return service.ErrMihomoNodeNotFound
	}
	if err != nil {
		return fmt.Errorf("update mihomo node: %w", err)
	}
	*item = *updated
	return nil
}

func (r *mihomoRepository) ListNodes(ctx context.Context, params pagination.PaginationParams, filter service.MihomoNodeFilter) ([]service.MihomoManagedNode, *pagination.PaginationResult, error) {
	where := []string{"1=1"}
	args := make([]any, 0, 8)
	if !filter.IncludeDeleted {
		where = append(where, "deleted_at IS NULL")
	}
	if !filter.IncludeRemoved {
		where = append(where, "upstream_removed_at IS NULL")
	}
	if filter.SubscriptionID != nil {
		args = append(args, *filter.SubscriptionID)
		where = append(where, fmt.Sprintf("subscription_id=$%d", len(args)))
	}
	if filter.Alive != nil {
		args = append(args, *filter.Alive)
		where = append(where, fmt.Sprintf("alive=$%d", len(args)))
	}
	if filter.Excluded != nil {
		args = append(args, *filter.Excluded)
		where = append(where, fmt.Sprintf("excluded=$%d", len(args)))
	}
	if filter.Search != "" {
		args = append(args, "%"+escapeLike(filter.Search)+"%")
		where = append(where, fmt.Sprintf("(original_name ILIKE $%d ESCAPE '\\' OR display_name ILIKE $%d ESCAPE '\\' OR region ILIKE $%d ESCAPE '\\')", len(args), len(args), len(args)))
	}
	clause := strings.Join(where, " AND ")
	exec := r.executor(ctx)
	var total int64
	if err := queryMihomoRow(ctx, exec, `SELECT COUNT(*) FROM mihomo_nodes WHERE `+clause, args...).Scan(&total); err != nil {
		return nil, nil, fmt.Errorf("count mihomo nodes: %w", err)
	}
	args = append(args, params.Limit(), params.Offset())
	rows, err := exec.QueryContext(ctx, `SELECT `+mihomoNodeColumns+`
FROM mihomo_nodes WHERE `+clause+` ORDER BY `+mihomoNodeOrder(params)+fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(args)-1, len(args)), args...)
	if err != nil {
		return nil, nil, fmt.Errorf("list mihomo nodes: %w", err)
	}
	defer func() { _ = rows.Close() }()
	items := make([]service.MihomoManagedNode, 0)
	for rows.Next() {
		item, scanErr := scanMihomoNode(rows)
		if scanErr != nil {
			return nil, nil, fmt.Errorf("scan mihomo node: %w", scanErr)
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterate mihomo nodes: %w", err)
	}
	return items, mihomoPagination(total, params), nil
}

const mihomoRouteColumns = `id, name, kind, listener_port, proxy_id, status, selector,
current_node_id, COALESCE(host(exit_ip), ''), exit_healthy, exit_delay_ms,
last_checked_at, COALESCE(last_error, ''), created_at, updated_at, deleted_at`

func scanMihomoRoute(row mihomoScanner) (*service.MihomoRoute, error) {
	var item service.MihomoRoute
	var selector []byte
	err := row.Scan(
		&item.ID, &item.Name, &item.Kind, &item.ListenerPort, &item.ProxyID, &item.Status, &selector,
		&item.CurrentNodeID, &item.ExitIP, &item.ExitHealthy, &item.ExitDelayMS, &item.LastCheckedAt,
		&item.LastError, &item.CreatedAt, &item.UpdatedAt, &item.DeletedAt,
	)
	item.Selector = json.RawMessage(selector)
	return &item, err
}

func normalizedMihomoSelector(selector json.RawMessage) (json.RawMessage, error) {
	if len(selector) == 0 {
		return json.RawMessage(`{}`), nil
	}
	if !json.Valid(selector) {
		return nil, fmt.Errorf("invalid mihomo route selector")
	}
	return selector, nil
}

func (r *mihomoRepository) CreateRoute(ctx context.Context, item *service.MihomoRoute) error {
	selector, err := normalizedMihomoSelector(item.Selector)
	if err != nil {
		return err
	}
	created, err := scanMihomoRoute(queryMihomoRow(ctx, r.executor(ctx), `
INSERT INTO mihomo_routes (
    name, kind, listener_port, proxy_id, status, selector, current_node_id,
    exit_ip, exit_healthy, exit_delay_ms, last_checked_at, last_error
) VALUES ($1,$2,$3,$4,$5,$6,$7,NULLIF($8,'')::inet,$9,$10,$11,NULLIF($12,''))
RETURNING `+mihomoRouteColumns,
		item.Name, item.Kind, item.ListenerPort, item.ProxyID, item.Status, selector, item.CurrentNodeID,
		item.ExitIP, item.ExitHealthy, item.ExitDelayMS, item.LastCheckedAt, item.LastError))
	if isUniqueViolation(err) {
		return service.ErrMihomoRouteExists
	}
	if err != nil {
		return fmt.Errorf("create mihomo route: %w", err)
	}
	*item = *created
	return nil
}

func (r *mihomoRepository) GetRouteByID(ctx context.Context, id int64) (*service.MihomoRoute, error) {
	item, err := scanMihomoRoute(queryMihomoRow(ctx, r.executor(ctx),
		`SELECT `+mihomoRouteColumns+` FROM mihomo_routes WHERE id=$1 AND deleted_at IS NULL`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrMihomoRouteNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get mihomo route: %w", err)
	}
	return item, nil
}

func (r *mihomoRepository) UpdateRoute(ctx context.Context, item *service.MihomoRoute) error {
	selector, err := normalizedMihomoSelector(item.Selector)
	if err != nil {
		return err
	}
	updated, err := scanMihomoRoute(queryMihomoRow(ctx, r.executor(ctx), `
UPDATE mihomo_routes SET
    name=$2, kind=$3, listener_port=$4, proxy_id=$5, status=$6, selector=$7,
    current_node_id=$8, exit_ip=NULLIF($9,'')::inet, exit_healthy=$10,
    exit_delay_ms=$11, last_checked_at=$12, last_error=NULLIF($13,''), updated_at=NOW()
WHERE id=$1 AND deleted_at IS NULL
RETURNING `+mihomoRouteColumns,
		item.ID, item.Name, item.Kind, item.ListenerPort, item.ProxyID, item.Status, selector,
		item.CurrentNodeID, item.ExitIP, item.ExitHealthy, item.ExitDelayMS, item.LastCheckedAt, item.LastError))
	if errors.Is(err, sql.ErrNoRows) {
		return service.ErrMihomoRouteNotFound
	}
	if isUniqueViolation(err) {
		return service.ErrMihomoRouteExists
	}
	if err != nil {
		return fmt.Errorf("update mihomo route: %w", err)
	}
	*item = *updated
	return nil
}

func (r *mihomoRepository) DeleteRoute(ctx context.Context, id int64) error {
	return r.inMihomoTx(ctx, func(tx mihomoExecutor) error {
		result, err := tx.ExecContext(ctx, `
UPDATE mihomo_routes SET status='disabled', deleted_at=NOW(), updated_at=NOW()
WHERE id=$1 AND deleted_at IS NULL`, id)
		if err != nil {
			return fmt.Errorf("delete mihomo route: %w", err)
		}
		affected, _ := result.RowsAffected()
		if affected == 0 {
			return service.ErrMihomoRouteNotFound
		}
		if _, err := tx.ExecContext(ctx, `
UPDATE mihomo_route_nodes SET deleted_at=NOW(), updated_at=NOW()
WHERE route_id=$1 AND deleted_at IS NULL`, id); err != nil {
			return fmt.Errorf("delete mihomo route nodes: %w", err)
		}
		return nil
	})
}

func (r *mihomoRepository) ListRoutes(ctx context.Context, params pagination.PaginationParams, filter service.MihomoRouteFilter) ([]service.MihomoRoute, *pagination.PaginationResult, error) {
	where := []string{"1=1"}
	args := make([]any, 0, 7)
	if !filter.IncludeDeleted {
		where = append(where, "r.deleted_at IS NULL")
	}
	if filter.SubscriptionID != nil {
		args = append(args, *filter.SubscriptionID)
		where = append(where, fmt.Sprintf(`EXISTS (
SELECT 1 FROM mihomo_route_nodes rn JOIN mihomo_nodes n ON n.id=rn.node_id
WHERE rn.route_id=r.id AND rn.deleted_at IS NULL AND n.deleted_at IS NULL AND n.subscription_id=$%d
)`, len(args)))
	}
	if filter.Kind != "" {
		args = append(args, filter.Kind)
		where = append(where, fmt.Sprintf("r.kind=$%d", len(args)))
	}
	if filter.Status != "" {
		args = append(args, filter.Status)
		where = append(where, fmt.Sprintf("r.status=$%d", len(args)))
	}
	if filter.Search != "" {
		args = append(args, "%"+escapeLike(filter.Search)+"%")
		where = append(where, fmt.Sprintf("r.name ILIKE $%d ESCAPE '\\'", len(args)))
	}
	clause := strings.Join(where, " AND ")
	exec := r.executor(ctx)
	var total int64
	if err := queryMihomoRow(ctx, exec, `SELECT COUNT(*) FROM mihomo_routes r WHERE `+clause, args...).Scan(&total); err != nil {
		return nil, nil, fmt.Errorf("count mihomo routes: %w", err)
	}
	args = append(args, params.Limit(), params.Offset())
	rows, err := exec.QueryContext(ctx, `SELECT `+mihomoRouteColumns+`
FROM mihomo_routes r WHERE `+clause+` ORDER BY `+mihomoRouteOrder(params)+fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(args)-1, len(args)), args...)
	if err != nil {
		return nil, nil, fmt.Errorf("list mihomo routes: %w", err)
	}
	defer func() { _ = rows.Close() }()
	items := make([]service.MihomoRoute, 0)
	for rows.Next() {
		item, scanErr := scanMihomoRoute(rows)
		if scanErr != nil {
			return nil, nil, fmt.Errorf("scan mihomo route: %w", scanErr)
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterate mihomo routes: %w", err)
	}
	return items, mihomoPagination(total, params), nil
}

const mihomoRouteNodeColumns = `id, route_id, node_id, priority, weight, created_at, updated_at, deleted_at`

func scanMihomoRouteNode(row mihomoScanner) (*service.MihomoRouteNode, error) {
	var item service.MihomoRouteNode
	err := row.Scan(&item.ID, &item.RouteID, &item.NodeID, &item.Priority, &item.Weight, &item.CreatedAt, &item.UpdatedAt, &item.DeletedAt)
	return &item, err
}

func (r *mihomoRepository) ReplaceRouteNodes(ctx context.Context, routeID int64, nodes []service.MihomoRouteNode) error {
	return r.inMihomoTx(ctx, func(tx mihomoExecutor) error {
		var lockedID int64
		if err := queryMihomoRow(ctx, tx, `SELECT id FROM mihomo_routes WHERE id=$1 AND deleted_at IS NULL FOR UPDATE`, routeID).Scan(&lockedID); errors.Is(err, sql.ErrNoRows) {
			return service.ErrMihomoRouteNotFound
		} else if err != nil {
			return fmt.Errorf("lock mihomo route: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `UPDATE mihomo_route_nodes SET deleted_at=NOW(), updated_at=NOW() WHERE route_id=$1 AND deleted_at IS NULL`, routeID); err != nil {
			return fmt.Errorf("retire mihomo route nodes: %w", err)
		}
		seen := make(map[int64]struct{}, len(nodes))
		for _, node := range nodes {
			if _, duplicate := seen[node.NodeID]; duplicate {
				continue
			}
			seen[node.NodeID] = struct{}{}
			if node.Weight <= 0 {
				node.Weight = 1
			}
			var id int64
			err := queryMihomoRow(ctx, tx, `
INSERT INTO mihomo_route_nodes (route_id, node_id, priority, weight)
SELECT $1,$2,$3,$4 FROM mihomo_nodes
WHERE id=$2 AND deleted_at IS NULL AND upstream_removed_at IS NULL
RETURNING id`, routeID, node.NodeID, node.Priority, node.Weight).Scan(&id)
			if errors.Is(err, sql.ErrNoRows) {
				return service.ErrMihomoNodeNotFound
			}
			if err != nil {
				return fmt.Errorf("add mihomo route node %d: %w", node.NodeID, err)
			}
		}
		if _, err := tx.ExecContext(ctx, `
UPDATE mihomo_routes r SET current_node_id=NULL, updated_at=NOW()
WHERE r.id=$1 AND r.current_node_id IS NOT NULL AND NOT EXISTS (
    SELECT 1 FROM mihomo_route_nodes rn
    WHERE rn.route_id=r.id AND rn.node_id=r.current_node_id AND rn.deleted_at IS NULL
)`, routeID); err != nil {
			return fmt.Errorf("clear stale mihomo route selection: %w", err)
		}
		return nil
	})
}

func (r *mihomoRepository) ListRouteNodes(ctx context.Context, routeID int64) ([]service.MihomoRouteNode, error) {
	rows, err := r.executor(ctx).QueryContext(ctx, `SELECT `+mihomoRouteNodeColumns+`
FROM mihomo_route_nodes WHERE route_id=$1 AND deleted_at IS NULL ORDER BY priority, id`, routeID)
	if err != nil {
		return nil, fmt.Errorf("list mihomo route nodes: %w", err)
	}
	defer func() { _ = rows.Close() }()
	items := make([]service.MihomoRouteNode, 0)
	for rows.Next() {
		item, scanErr := scanMihomoRouteNode(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan mihomo route node: %w", scanErr)
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate mihomo route nodes: %w", err)
	}
	return items, nil
}

func (r *mihomoRepository) inMihomoTx(ctx context.Context, fn func(mihomoExecutor) error) error {
	if tx := dbent.TxFromContext(ctx); tx != nil {
		return fn(tx.Client())
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin mihomo transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if err := fn(tx); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit mihomo transaction: %w", err)
	}
	return nil
}

func mihomoPagination(total int64, params pagination.PaginationParams) *pagination.PaginationResult {
	page := params.Page
	if page < 1 {
		page = 1
	}
	limit := params.Limit()
	pages := 0
	if total > 0 {
		pages = int((total + int64(limit) - 1) / int64(limit))
	}
	return &pagination.PaginationResult{Total: total, Page: page, PageSize: limit, Pages: pages}
}

func mihomoSubscriptionOrder(params pagination.PaginationParams) string {
	column := "updated_at"
	switch strings.ToLower(strings.TrimSpace(params.SortBy)) {
	case "id":
		column = "id"
	case "name":
		column = "name"
	case "status":
		column = "status"
	case "created_at":
		column = "created_at"
	case "updated_at", "":
		column = "updated_at"
	}
	return column + " " + strings.ToUpper(params.NormalizedSortOrder(pagination.SortOrderDesc)) + ", id DESC"
}

func mihomoNodeOrder(params pagination.PaginationParams) string {
	column := "updated_at"
	switch strings.ToLower(strings.TrimSpace(params.SortBy)) {
	case "id":
		column = "id"
	case "name":
		column = "display_name"
	case "delay_ms":
		column = "delay_ms"
	case "last_seen_at":
		column = "last_seen_at"
	case "updated_at", "":
		column = "updated_at"
	}
	return column + " " + strings.ToUpper(params.NormalizedSortOrder(pagination.SortOrderDesc)) + " NULLS LAST, id DESC"
}

func mihomoRouteOrder(params pagination.PaginationParams) string {
	column := "r.updated_at"
	switch strings.ToLower(strings.TrimSpace(params.SortBy)) {
	case "id":
		column = "r.id"
	case "name":
		column = "r.name"
	case "kind":
		column = "r.kind"
	case "status":
		column = "r.status"
	case "listener_port":
		column = "r.listener_port"
	case "updated_at", "":
		column = "r.updated_at"
	}
	return column + " " + strings.ToUpper(params.NormalizedSortOrder(pagination.SortOrderDesc)) + ", r.id DESC"
}
