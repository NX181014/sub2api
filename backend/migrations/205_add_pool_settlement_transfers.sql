CREATE TABLE IF NOT EXISTS pool_settlement_transfers (
    id BIGSERIAL PRIMARY KEY,
    settlement_id BIGINT NOT NULL REFERENCES pool_settlements(id) ON DELETE CASCADE,
    transfer_key TEXT,
    input_hash TEXT NOT NULL DEFAULT '',
    from_user_id BIGINT NOT NULL REFERENCES users(id),
    to_user_id BIGINT NOT NULL REFERENCES users(id),
    amount_minor BIGINT NOT NULL CHECK (amount_minor > 0),
    currency VARCHAR(3) NOT NULL DEFAULT 'CNY',
    payment_status TEXT NOT NULL DEFAULT 'pending',
    account_line_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
    account_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
    paid_by_user_id BIGINT REFERENCES users(id),
    paid_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT pool_settlement_transfers_parties CHECK (from_user_id <> to_user_id),
    CONSTRAINT pool_settlement_transfers_status CHECK (payment_status IN ('pending','paid','void'))
);

CREATE INDEX IF NOT EXISTS idx_pool_settlement_transfers_settlement
    ON pool_settlement_transfers(settlement_id, payment_status, id);

CREATE UNIQUE INDEX IF NOT EXISTS uq_pool_settlement_transfers_key
    ON pool_settlement_transfers(settlement_id, transfer_key)
    WHERE transfer_key IS NOT NULL AND payment_status <> 'void';

CREATE INDEX IF NOT EXISTS idx_pool_settlement_transfers_input
    ON pool_settlement_transfers(settlement_id, input_hash, payment_status);
