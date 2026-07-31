CREATE TABLE IF NOT EXISTS pool_settlement_account_costs (
    id BIGSERIAL PRIMARY KEY,
    settlement_id BIGINT NOT NULL REFERENCES pool_settlements(id) ON DELETE CASCADE,
    account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    cost_entry_id BIGINT NOT NULL REFERENCES account_cost_entries(id) ON DELETE CASCADE,
    kind VARCHAR(20) NOT NULL CHECK (kind IN ('period', 'carry')),
    payer_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    amount_minor BIGINT NOT NULL,
    UNIQUE(settlement_id, kind, cost_entry_id)
);

CREATE INDEX IF NOT EXISTS idx_pool_settlement_account_costs_account
    ON pool_settlement_account_costs(account_id, settlement_id);
CREATE INDEX IF NOT EXISTS idx_pool_settlement_account_costs_entry
    ON pool_settlement_account_costs(cost_entry_id, settlement_id);

CREATE TABLE IF NOT EXISTS pool_settlement_account_lines (
    id BIGSERIAL PRIMARY KEY,
    settlement_id BIGINT NOT NULL REFERENCES pool_settlements(id) ON DELETE CASCADE,
    account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    account_usage_weight VARCHAR(50) NOT NULL DEFAULT '0',
    usage_share VARCHAR(40) NOT NULL DEFAULT '0',
    allocated_cost_minor BIGINT NOT NULL DEFAULT 0,
    contribution_credit_minor BIGINT NOT NULL DEFAULT 0,
    adjustment_minor BIGINT NOT NULL DEFAULT 0,
    net_amount_minor BIGINT NOT NULL DEFAULT 0,
    trace_quality VARCHAR(20) NOT NULL DEFAULT 'exact' CHECK (trace_quality IN ('exact', 'derived', 'unavailable')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(settlement_id, account_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_pool_settlement_account_lines_account
    ON pool_settlement_account_lines(account_id, settlement_id);
CREATE INDEX IF NOT EXISTS idx_pool_settlement_account_lines_user
    ON pool_settlement_account_lines(user_id, settlement_id);

-- Existing finalized settlements retain their exact cost provenance. Drafts are
-- previews and are rebuilt by the normal recalculation path with account lines.
INSERT INTO pool_settlement_account_costs(
    settlement_id, account_id, cost_entry_id, kind, payer_user_id, amount_minor
)
SELECT s.id,
       (item->>'account_id')::bigint,
       (item->>'entry_id')::bigint,
       item->>'kind',
       (item->>'payer_user_id')::bigint,
       (item->>'amount_minor')::bigint
FROM pool_settlements s
CROSS JOIN LATERAL jsonb_array_elements(
    CASE WHEN jsonb_typeof(s.cost_snapshot)='array' THEN s.cost_snapshot ELSE '[]'::jsonb END
) item
JOIN accounts a ON a.id=(item->>'account_id')::bigint
JOIN account_cost_entries c ON c.id=(item->>'entry_id')::bigint
WHERE s.status IN ('locked', 'paid')
  AND item->>'kind' IN ('period', 'carry')
ON CONFLICT (settlement_id, kind, cost_entry_id) DO NOTHING;

UPDATE pool_settlements SET cost_snapshot='[]'::jsonb WHERE status IN ('locked', 'paid');
DELETE FROM pool_settlements WHERE status='draft';
DELETE FROM purchase_sources ps
WHERE NOT EXISTS (SELECT 1 FROM account_cost_entries c WHERE c.purchase_source_id=ps.id);
