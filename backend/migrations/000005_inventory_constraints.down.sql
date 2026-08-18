DROP INDEX IF EXISTS inventory_balances_created_idx;

ALTER TABLE inventory_balances
    DROP COLUMN created_at;
