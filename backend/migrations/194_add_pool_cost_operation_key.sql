ALTER TABLE account_cost_entries
    ADD COLUMN IF NOT EXISTS operation_key VARCHAR(255);

CREATE UNIQUE INDEX IF NOT EXISTS idx_account_cost_entries_operation_key
    ON account_cost_entries(created_by_user_id, operation_key)
    WHERE operation_key IS NOT NULL;
