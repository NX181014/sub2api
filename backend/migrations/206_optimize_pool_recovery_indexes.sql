-- Recovery overview filters and orders cost entries by account and paid time.
CREATE INDEX IF NOT EXISTS idx_account_cost_entries_account_paid_at
    ON account_cost_entries(account_id, paid_at, id);

-- Source attribution narrows the same access path to purchase rows.
CREATE INDEX IF NOT EXISTS idx_account_cost_entries_account_type_paid_at
    ON account_cost_entries(account_id, entry_type, paid_at, id);
