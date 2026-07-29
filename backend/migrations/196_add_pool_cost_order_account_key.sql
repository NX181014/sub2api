ALTER TABLE account_cost_entries
    ADD COLUMN IF NOT EXISTS order_account_key VARCHAR(512),
    ADD COLUMN IF NOT EXISTS expected_token_count BIGINT;

ALTER TABLE account_cost_entries
    DROP CONSTRAINT IF EXISTS account_cost_entries_expected_token_count_positive;

ALTER TABLE account_cost_entries
    ADD CONSTRAINT account_cost_entries_expected_token_count_positive
        CHECK (expected_token_count IS NULL OR expected_token_count > 0);

ALTER TABLE account_cost_entries
    DROP CONSTRAINT IF EXISTS account_cost_entries_entry_type_check;

ALTER TABLE account_cost_entries
    ADD CONSTRAINT account_cost_entries_entry_type_check CHECK (entry_type IN (
        'purchase', 'renewal', 'topup', 'price_version', 'refund', 'adjustment',
        'replacement_in', 'replacement_out', 'write_off'
    ));

CREATE UNIQUE INDEX IF NOT EXISTS idx_account_cost_entries_order_account_key
    ON account_cost_entries(order_account_key)
    WHERE order_account_key IS NOT NULL;

ALTER TABLE pool_approval_requests
    DROP CONSTRAINT IF EXISTS pool_approval_requests_action_type_check;

ALTER TABLE pool_approval_requests
    ADD CONSTRAINT pool_approval_requests_action_type_check
        CHECK (action_type IN ('UPDATE_ACCOUNT', 'VIEW_CREDENTIAL', 'DELETE_ACCOUNT'));

CREATE UNIQUE INDEX IF NOT EXISTS idx_pool_approval_pending_delete_account
    ON pool_approval_requests(account_id)
    WHERE action_type = 'DELETE_ACCOUNT' AND status = 'pending';
