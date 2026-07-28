INSERT INTO settings (key, value, updated_at)
SELECT 'primary_admin_user_id', id::text, NOW()
FROM users
WHERE role = 'admin'
ORDER BY created_at ASC, id ASC
LIMIT 1
ON CONFLICT (key) DO NOTHING;

CREATE TABLE IF NOT EXISTS pool_approval_requests (
    id BIGSERIAL PRIMARY KEY,
    action_type VARCHAR(30) NOT NULL CHECK (action_type IN ('UPDATE_ACCOUNT', 'VIEW_CREDENTIAL')),
    account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE RESTRICT,
    status VARCHAR(20) NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'approved', 'rejected', 'expired', 'consumed')),
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    change_summary JSONB NOT NULL DEFAULT '{}'::jsonb,
    reason TEXT NOT NULL,
    base_revision VARCHAR(64) NOT NULL,
    requested_by_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    decided_by_user_id BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    decision_reason TEXT,
    requested_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    decided_at TIMESTAMPTZ,
    reveal_expires_at TIMESTAMPTZ,
    revealed_at TIMESTAMPTZ,
    primary_bypass BOOLEAN NOT NULL DEFAULT FALSE,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT pool_approval_decision_fields CHECK (
        (status = 'pending' AND decided_by_user_id IS NULL AND decided_at IS NULL)
        OR (status <> 'pending')
    ),
    CONSTRAINT pool_approval_reveal_fields CHECK (
        action_type = 'VIEW_CREDENTIAL'
        OR (reveal_expires_at IS NULL AND revealed_at IS NULL AND status <> 'consumed')
    )
);

CREATE INDEX IF NOT EXISTS idx_pool_approval_requests_queue
    ON pool_approval_requests(status, requested_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_pool_approval_requests_account
    ON pool_approval_requests(account_id, requested_at DESC, id DESC);
CREATE UNIQUE INDEX IF NOT EXISTS idx_pool_approval_pending_update
    ON pool_approval_requests(account_id)
    WHERE action_type = 'UPDATE_ACCOUNT' AND status = 'pending';
CREATE UNIQUE INDEX IF NOT EXISTS idx_pool_approval_live_view
    ON pool_approval_requests(account_id, requested_by_user_id)
    WHERE action_type = 'VIEW_CREDENTIAL' AND status IN ('pending', 'approved');
