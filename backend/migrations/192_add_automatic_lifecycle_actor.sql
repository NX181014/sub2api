ALTER TABLE account_lifecycle_events
    ADD COLUMN IF NOT EXISTS source VARCHAR(20) NOT NULL DEFAULT 'manual';

ALTER TABLE account_lifecycle_events
    ALTER COLUMN created_by_user_id DROP NOT NULL;

DO $$ BEGIN
    ALTER TABLE account_lifecycle_events
        ADD CONSTRAINT account_lifecycle_events_actor_check CHECK (
            (source = 'automatic' AND created_by_user_id IS NULL)
            OR (source = 'manual' AND created_by_user_id IS NOT NULL)
        );
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
