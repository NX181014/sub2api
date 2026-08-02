ALTER TABLE proxies
    ADD COLUMN IF NOT EXISTS managed_source VARCHAR(64);

CREATE UNIQUE INDEX IF NOT EXISTS idx_proxies_managed_source
    ON proxies(managed_source)
    WHERE managed_source IS NOT NULL AND deleted_at IS NULL;

ALTER TABLE pool_approval_requests
    ADD COLUMN IF NOT EXISTS object_type VARCHAR(20) NOT NULL DEFAULT 'account',
    ADD COLUMN IF NOT EXISTS proxy_id BIGINT REFERENCES proxies(id) ON DELETE RESTRICT,
    ADD COLUMN IF NOT EXISTS resource_key VARCHAR(100);

ALTER TABLE pool_approval_requests
    ALTER COLUMN account_id DROP NOT NULL;

ALTER TABLE pool_approval_requests
    DROP CONSTRAINT IF EXISTS pool_approval_requests_action_type_check,
    DROP CONSTRAINT IF EXISTS pool_approval_reveal_fields,
    DROP CONSTRAINT IF EXISTS pool_approval_object_check;

ALTER TABLE pool_approval_requests
    ADD CONSTRAINT pool_approval_requests_action_type_check CHECK (action_type IN (
        'UPDATE_ACCOUNT', 'VIEW_CREDENTIAL', 'DELETE_ACCOUNT',
        'UPDATE_PROXY', 'VIEW_PROXY_CREDENTIAL', 'EXPORT_PROXY_CREDENTIALS', 'UPDATE_MIHOMO'
    )),
    ADD CONSTRAINT pool_approval_object_check CHECK (
        (object_type = 'account' AND account_id IS NOT NULL AND proxy_id IS NULL AND resource_key IS NULL)
        OR (object_type = 'proxy' AND account_id IS NULL AND proxy_id IS NOT NULL AND resource_key IS NULL)
        OR (object_type = 'mihomo' AND account_id IS NULL AND proxy_id IS NULL AND resource_key IS NOT NULL)
        OR (object_type = 'proxy_export' AND account_id IS NULL AND proxy_id IS NULL AND resource_key = 'proxy-export')
    ),
    ADD CONSTRAINT pool_approval_reveal_fields CHECK (
        action_type IN ('VIEW_CREDENTIAL', 'VIEW_PROXY_CREDENTIAL', 'EXPORT_PROXY_CREDENTIALS')
        OR (reveal_expires_at IS NULL AND revealed_at IS NULL AND status <> 'consumed')
    );

CREATE INDEX IF NOT EXISTS idx_pool_approval_requests_proxy
    ON pool_approval_requests(proxy_id, requested_at DESC, id DESC)
    WHERE proxy_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_pool_approval_pending_update_proxy
    ON pool_approval_requests(proxy_id)
    WHERE action_type = 'UPDATE_PROXY' AND status = 'pending';

CREATE UNIQUE INDEX IF NOT EXISTS idx_pool_approval_live_view_proxy
    ON pool_approval_requests(proxy_id, requested_by_user_id)
    WHERE action_type = 'VIEW_PROXY_CREDENTIAL' AND status IN ('pending', 'approved');

CREATE UNIQUE INDEX IF NOT EXISTS idx_pool_approval_live_proxy_export
    ON pool_approval_requests(requested_by_user_id)
    WHERE action_type = 'EXPORT_PROXY_CREDENTIALS' AND status IN ('pending', 'approved');

CREATE UNIQUE INDEX IF NOT EXISTS idx_pool_approval_pending_mihomo
    ON pool_approval_requests(resource_key)
    WHERE action_type = 'UPDATE_MIHOMO' AND status = 'pending';
