ALTER TABLE accounts
    ADD COLUMN IF NOT EXISTS provider_identity VARCHAR(255),
    ADD COLUMN IF NOT EXISTS contributor_user_id BIGINT,
    ADD COLUMN IF NOT EXISTS created_by_user_id BIGINT,
    ADD COLUMN IF NOT EXISTS cost_sharing_enabled BOOLEAN NOT NULL DEFAULT FALSE;

DO $$ BEGIN
    ALTER TABLE accounts ADD CONSTRAINT accounts_contributor_user_id_fkey
        FOREIGN KEY (contributor_user_id) REFERENCES users(id) ON DELETE SET NULL;
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    ALTER TABLE accounts ADD CONSTRAINT accounts_created_by_user_id_fkey
        FOREIGN KEY (created_by_user_id) REFERENCES users(id) ON DELETE SET NULL;
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

CREATE INDEX IF NOT EXISTS idx_accounts_cost_sharing_enabled
    ON accounts(cost_sharing_enabled) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_accounts_contributor_user_id
    ON accounts(contributor_user_id) WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS purchase_sources (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    website_url VARCHAR(2048),
    notes TEXT,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_purchase_sources_name_ci
    ON purchase_sources(LOWER(BTRIM(name)));

CREATE TABLE IF NOT EXISTS account_cost_entries (
    id BIGSERIAL PRIMARY KEY,
    account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE RESTRICT,
    payer_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    purchase_source_id BIGINT REFERENCES purchase_sources(id) ON DELETE SET NULL,
    entry_type VARCHAR(30) NOT NULL CHECK (entry_type IN
        ('purchase', 'renewal', 'topup', 'price_version', 'refund', 'adjustment', 'replacement_in', 'replacement_out')),
    currency VARCHAR(3) NOT NULL DEFAULT 'CNY',
    original_amount VARCHAR(40) NOT NULL,
    cny_amount_minor BIGINT NOT NULL,
    fx_rate VARCHAR(40) NOT NULL DEFAULT '1',
    service_start DATE NOT NULL,
    service_end DATE NOT NULL,
    warranty_end DATE,
    paid_at TIMESTAMPTZ NOT NULL,
    order_no VARCHAR(255),
    purchase_url VARCHAR(2048),
    note TEXT,
    supersedes_id BIGINT REFERENCES account_cost_entries(id) ON DELETE RESTRICT,
    related_account_id BIGINT REFERENCES accounts(id) ON DELETE RESTRICT,
    created_by_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT account_cost_entries_service_period CHECK (service_end > service_start)
);
CREATE INDEX IF NOT EXISTS idx_account_cost_entries_period
    ON account_cost_entries(account_id, service_start, service_end);
CREATE INDEX IF NOT EXISTS idx_account_cost_entries_payer
    ON account_cost_entries(payer_user_id);
CREATE INDEX IF NOT EXISTS idx_account_cost_entries_source
    ON account_cost_entries(purchase_source_id);

CREATE TABLE IF NOT EXISTS account_lifecycle_events (
    id BIGSERIAL PRIMARY KEY,
    account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE RESTRICT,
    event_type VARCHAR(30) NOT NULL CHECK (event_type IN
        ('banned_confirmed', 'recovered', 'refund', 'replaced', 'retired')),
    occurred_at TIMESTAMPTZ NOT NULL,
    reason TEXT,
    replacement_account_id BIGINT REFERENCES accounts(id) ON DELETE RESTRICT,
    transferred_cost_minor BIGINT NOT NULL DEFAULT 0 CHECK (transferred_cost_minor >= 0),
    created_by_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT lifecycle_replacement_fields CHECK (
        (event_type = 'replaced' AND replacement_account_id IS NOT NULL)
        OR (event_type <> 'replaced' AND replacement_account_id IS NULL AND transferred_cost_minor = 0)
    )
);
CREATE INDEX IF NOT EXISTS idx_account_lifecycle_events_time
    ON account_lifecycle_events(account_id, occurred_at DESC, id DESC);

CREATE TABLE IF NOT EXISTS valuation_fx_rates (
    id BIGSERIAL PRIMARY KEY,
    base_currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    quote_currency VARCHAR(3) NOT NULL DEFAULT 'CNY',
    rate VARCHAR(40) NOT NULL,
    effective_from TIMESTAMPTZ NOT NULL,
    source VARCHAR(100),
    created_by_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(base_currency, quote_currency, effective_from)
);
CREATE INDEX IF NOT EXISTS idx_valuation_fx_rates_lookup
    ON valuation_fx_rates(base_currency, quote_currency, effective_from DESC);

CREATE TABLE IF NOT EXISTS pool_settlements (
    id BIGSERIAL PRIMARY KEY,
    period_type VARCHAR(10) NOT NULL CHECK (period_type IN ('day', 'week', 'month', 'custom')),
    period_start TIMESTAMPTZ NOT NULL,
    period_end TIMESTAMPTZ NOT NULL,
    timezone VARCHAR(50) NOT NULL DEFAULT 'Asia/Shanghai',
    status VARCHAR(20) NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'locked')),
    period_cost_minor BIGINT NOT NULL DEFAULT 0,
    carry_in_minor BIGINT NOT NULL DEFAULT 0,
    carry_out_minor BIGINT NOT NULL DEFAULT 0,
    total_cost_minor BIGINT NOT NULL DEFAULT 0,
    total_usage_weight VARCHAR(50) NOT NULL DEFAULT '0',
    pricing_coverage VARCHAR(40) NOT NULL DEFAULT '1',
    unpriced_usage_count BIGINT NOT NULL DEFAULT 0 CHECK (unpriced_usage_count >= 0),
    fx_rate VARCHAR(40) NOT NULL DEFAULT '1',
    formula_version VARCHAR(20) NOT NULL DEFAULT 'v1',
    cost_snapshot JSONB NOT NULL DEFAULT '[]'::jsonb,
    generated_by_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    locked_by_user_id BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    locked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT pool_settlement_period CHECK (period_end > period_start),
    CONSTRAINT pool_settlement_lock_fields CHECK (
        (status = 'draft' AND locked_by_user_id IS NULL AND locked_at IS NULL)
        OR (status = 'locked' AND locked_by_user_id IS NOT NULL AND locked_at IS NOT NULL)
    )
);
CREATE INDEX IF NOT EXISTS idx_pool_settlements_period
    ON pool_settlements(period_start DESC, period_end DESC);
CREATE INDEX IF NOT EXISTS idx_pool_settlements_locked_period
    ON pool_settlements(period_start, period_end) WHERE status = 'locked';

CREATE TABLE IF NOT EXISTS pool_settlement_lines (
    id BIGSERIAL PRIMARY KEY,
    settlement_id BIGINT NOT NULL REFERENCES pool_settlements(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    usage_weight VARCHAR(50) NOT NULL DEFAULT '0',
    usage_share VARCHAR(40) NOT NULL DEFAULT '0',
    allocated_cost_minor BIGINT NOT NULL DEFAULT 0,
    contribution_credit_minor BIGINT NOT NULL DEFAULT 0,
    adjustment_minor BIGINT NOT NULL DEFAULT 0,
    net_amount_minor BIGINT NOT NULL DEFAULT 0,
    payment_status VARCHAR(20) NOT NULL DEFAULT 'unpaid' CHECK (payment_status IN ('unpaid', 'paid')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(settlement_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_pool_settlement_lines_user
    ON pool_settlement_lines(user_id, settlement_id);
