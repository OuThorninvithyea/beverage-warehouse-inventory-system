ALTER TABLE inventory_balances
    ADD COLUMN created_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

CREATE INDEX inventory_balances_created_idx
    ON inventory_balances (created_at DESC, id DESC);
