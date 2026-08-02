CREATE TABLE IF NOT EXISTS mihomo_subscriptions (
    id                       BIGSERIAL PRIMARY KEY,
    name                     VARCHAR(100) NOT NULL,
    provider_key             VARCHAR(100) NOT NULL,
    url_ciphertext           BYTEA NOT NULL CHECK (octet_length(url_ciphertext) > 0),
    masked_host              VARCHAR(255) NOT NULL DEFAULT '',
    refresh_interval_seconds INT NOT NULL DEFAULT 3600 CHECK (refresh_interval_seconds >= 60),
    quota_total_bytes        BIGINT CHECK (quota_total_bytes IS NULL OR quota_total_bytes >= 0),
    quota_used_bytes         BIGINT CHECK (quota_used_bytes IS NULL OR quota_used_bytes >= 0),
    expires_at               TIMESTAMPTZ,
    status                   VARCHAR(20) NOT NULL DEFAULT 'active'
                             CHECK (status IN ('active', 'disabled', 'error')),
    last_refreshed_at        TIMESTAMPTZ,
    last_error               TEXT,
    created_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at               TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_mihomo_subscriptions_provider_key_live
    ON mihomo_subscriptions(provider_key) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_mihomo_subscriptions_name_live
    ON mihomo_subscriptions(LOWER(name)) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_mihomo_subscriptions_refresh_due
    ON mihomo_subscriptions(last_refreshed_at, id)
    WHERE deleted_at IS NULL AND status = 'active';

CREATE TABLE IF NOT EXISTS mihomo_nodes (
    id                  BIGSERIAL PRIMARY KEY,
    subscription_id     BIGINT NOT NULL REFERENCES mihomo_subscriptions(id) ON DELETE RESTRICT,
    node_key             VARCHAR(160) NOT NULL,
    original_name        VARCHAR(255) NOT NULL,
    display_name         VARCHAR(255) NOT NULL,
    alive                BOOLEAN NOT NULL DEFAULT FALSE,
    delay_ms             INT CHECK (delay_ms IS NULL OR delay_ms >= 0),
    region               VARCHAR(100) NOT NULL DEFAULT '',
    tags                 TEXT[] NOT NULL DEFAULT '{}',
    excluded             BOOLEAN NOT NULL DEFAULT FALSE,
    last_seen_at         TIMESTAMPTZ NOT NULL,
    upstream_removed_at  TIMESTAMPTZ,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at           TIMESTAMPTZ,
    UNIQUE (subscription_id, node_key)
);

CREATE INDEX IF NOT EXISTS idx_mihomo_nodes_subscription_live
    ON mihomo_nodes(subscription_id, upstream_removed_at, id)
    WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_mihomo_nodes_health_live
    ON mihomo_nodes(alive, excluded, delay_ms)
    WHERE deleted_at IS NULL AND upstream_removed_at IS NULL;

CREATE TABLE IF NOT EXISTS mihomo_routes (
    id                  BIGSERIAL PRIMARY KEY,
    name                VARCHAR(100) NOT NULL,
    kind                VARCHAR(20) NOT NULL
                        CHECK (kind IN ('dedicated', 'automatic', 'latency', 'fallback', 'dynamic', 'directional')),
    listener_port       INT NOT NULL CHECK (listener_port BETWEEN 1 AND 65535),
    proxy_id            BIGINT REFERENCES proxies(id) ON DELETE RESTRICT,
    status              VARCHAR(20) NOT NULL DEFAULT 'active'
                        CHECK (status IN ('active', 'disabled', 'error')),
    selector            JSONB NOT NULL DEFAULT '{}',
    current_node_id     BIGINT REFERENCES mihomo_nodes(id) ON DELETE RESTRICT,
    exit_ip             INET,
    exit_healthy        BOOLEAN,
    exit_delay_ms       INT CHECK (exit_delay_ms IS NULL OR exit_delay_ms >= 0),
    last_checked_at     TIMESTAMPTZ,
    last_error          TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at          TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_mihomo_routes_name_live
    ON mihomo_routes(LOWER(name)) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_mihomo_routes_listener_port_live
    ON mihomo_routes(listener_port) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_mihomo_routes_proxy_live
    ON mihomo_routes(proxy_id) WHERE proxy_id IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_mihomo_routes_status_kind_live
    ON mihomo_routes(status, kind, id) WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS mihomo_route_nodes (
    id          BIGSERIAL PRIMARY KEY,
    route_id    BIGINT NOT NULL REFERENCES mihomo_routes(id) ON DELETE RESTRICT,
    node_id     BIGINT NOT NULL REFERENCES mihomo_nodes(id) ON DELETE RESTRICT,
    priority    INT NOT NULL DEFAULT 0,
    weight      INT NOT NULL DEFAULT 1 CHECK (weight > 0),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_mihomo_route_nodes_live
    ON mihomo_route_nodes(route_id, node_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_mihomo_route_nodes_node_live
    ON mihomo_route_nodes(node_id, route_id) WHERE deleted_at IS NULL;
