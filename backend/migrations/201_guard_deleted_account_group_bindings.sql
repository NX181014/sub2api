DELETE FROM account_groups ag
USING accounts a
WHERE a.id=ag.account_id AND a.deleted_at IS NOT NULL;

CREATE OR REPLACE FUNCTION guard_deleted_account_group_binding()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    PERFORM 1
    FROM accounts
    WHERE id=NEW.account_id AND deleted_at IS NULL
    FOR UPDATE;

    IF NOT FOUND THEN
        RAISE EXCEPTION 'account % is deleted', NEW.account_id
            USING ERRCODE='foreign_key_violation';
    END IF;
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS account_groups_guard_deleted_account ON account_groups;
CREATE TRIGGER account_groups_guard_deleted_account
BEFORE INSERT OR UPDATE OF account_id ON account_groups
FOR EACH ROW EXECUTE FUNCTION guard_deleted_account_group_binding();
