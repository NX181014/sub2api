ALTER TABLE pool_settlements
    ADD COLUMN IF NOT EXISTS paid_by_user_id BIGINT,
    ADD COLUMN IF NOT EXISTS paid_at TIMESTAMPTZ;

UPDATE pool_settlements
SET paid_by_user_id = COALESCE(paid_by_user_id, locked_by_user_id, generated_by_user_id),
    paid_at = COALESCE(paid_at, updated_at, locked_at)
WHERE status = 'paid'
  AND (paid_by_user_id IS NULL OR paid_at IS NULL);

DO $$ BEGIN
    ALTER TABLE pool_settlements ADD CONSTRAINT pool_settlements_paid_by_fkey
        FOREIGN KEY (paid_by_user_id) REFERENCES users(id) ON DELETE RESTRICT;
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    ALTER TABLE pool_settlements ADD CONSTRAINT pool_settlement_paid_fields CHECK (
        (status = 'paid' AND paid_by_user_id IS NOT NULL AND paid_at IS NOT NULL)
        OR (status <> 'paid' AND paid_by_user_id IS NULL AND paid_at IS NULL)
    );
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
