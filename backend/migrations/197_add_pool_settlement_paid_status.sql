ALTER TABLE pool_settlements
    DROP CONSTRAINT IF EXISTS pool_settlements_status_check;

ALTER TABLE pool_settlements
    ADD CONSTRAINT pool_settlements_status_check
        CHECK (status IN ('draft', 'locked', 'paid'));

ALTER TABLE pool_settlements
    DROP CONSTRAINT IF EXISTS pool_settlement_lock_fields;

ALTER TABLE pool_settlements
    ADD CONSTRAINT pool_settlement_lock_fields CHECK (
        (status = 'draft' AND locked_by_user_id IS NULL AND locked_at IS NULL)
        OR (status IN ('locked', 'paid') AND locked_by_user_id IS NOT NULL AND locked_at IS NOT NULL)
    );

-- Migration 195 grandfathered locked lines as confirmed by the locker before
-- member self-confirmation existed. Re-open only those non-zero legacy lines.
UPDATE pool_settlement_lines l
SET confirmation_status = 'pending',
    confirmed_by_user_id = NULL,
    confirmed_at = NULL,
    updated_at = NOW()
FROM pool_settlements s
WHERE s.id = l.settlement_id
  AND s.status = 'locked'
  AND l.net_amount_minor <> 0;
