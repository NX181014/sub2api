ALTER TABLE accounts
    ADD COLUMN IF NOT EXISTS expected_token_count BIGINT;

DO $$ BEGIN
    ALTER TABLE accounts ADD CONSTRAINT accounts_expected_token_count_positive
        CHECK (expected_token_count IS NULL OR expected_token_count > 0);
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

ALTER TABLE pool_settlements
    ADD COLUMN IF NOT EXISTS filter_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb;

ALTER TABLE pool_settlement_lines
    ADD COLUMN IF NOT EXISTS confirmation_status VARCHAR(20) NOT NULL DEFAULT 'pending',
    ADD COLUMN IF NOT EXISTS confirmed_by_user_id BIGINT,
    ADD COLUMN IF NOT EXISTS confirmed_at TIMESTAMPTZ;

DO $$ BEGIN
    ALTER TABLE pool_settlement_lines ADD CONSTRAINT pool_settlement_lines_confirmation_status
        CHECK (confirmation_status IN ('pending', 'confirmed'));
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    ALTER TABLE pool_settlement_lines ADD CONSTRAINT pool_settlement_lines_confirmed_by_fkey
        FOREIGN KEY (confirmed_by_user_id) REFERENCES users(id) ON DELETE RESTRICT;
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    ALTER TABLE pool_settlement_lines ADD CONSTRAINT pool_settlement_lines_confirmation_fields
        CHECK (
            (confirmation_status = 'pending' AND confirmed_by_user_id IS NULL AND confirmed_at IS NULL)
            OR (confirmation_status = 'confirmed' AND confirmed_by_user_id IS NOT NULL AND confirmed_at IS NOT NULL)
        );
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

UPDATE pool_settlement_lines l
SET confirmation_status = 'confirmed',
    confirmed_by_user_id = COALESCE(s.locked_by_user_id, s.generated_by_user_id),
    confirmed_at = COALESCE(s.locked_at, s.updated_at)
FROM pool_settlements s
WHERE s.id = l.settlement_id
  AND s.status = 'locked'
  AND l.confirmation_status = 'pending';

CREATE INDEX IF NOT EXISTS idx_pool_settlement_lines_confirmation
    ON pool_settlement_lines(settlement_id, confirmation_status);
