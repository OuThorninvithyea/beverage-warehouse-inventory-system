CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT roles_code_not_blank CHECK (BTRIM(code) <> ''),
    CONSTRAINT roles_name_not_blank CHECK (BTRIM(name) <> '')
);

CREATE TABLE warehouses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    address TEXT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT warehouses_code_not_blank CHECK (BTRIM(code) <> ''),
    CONSTRAINT warehouses_name_not_blank CHECK (BTRIM(name) <> '')
);

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    role_id UUID NOT NULL REFERENCES roles(id),
    warehouse_id UUID REFERENCES warehouses(id),
    email TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    full_name TEXT NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT users_email_not_blank CHECK (BTRIM(email) <> ''),
    CONSTRAINT users_password_hash_not_blank CHECK (BTRIM(password_hash) <> ''),
    CONSTRAINT users_full_name_not_blank CHECK (BTRIM(full_name) <> '')
);

CREATE UNIQUE INDEX users_email_unique_idx ON users (LOWER(email));
CREATE INDEX users_role_id_idx ON users (role_id);
CREATE INDEX users_warehouse_id_idx ON users (warehouse_id);

CREATE TABLE refresh_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT refresh_tokens_hash_not_blank CHECK (BTRIM(token_hash) <> ''),
    CONSTRAINT refresh_tokens_expiry_after_creation CHECK (expires_at > created_at)
);

CREATE INDEX refresh_tokens_user_id_idx ON refresh_tokens (user_id);
CREATE INDEX refresh_tokens_active_expiry_idx
    ON refresh_tokens (expires_at)
    WHERE revoked_at IS NULL;

CREATE TABLE locations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    warehouse_id UUID NOT NULL REFERENCES warehouses(id),
    code TEXT NOT NULL,
    zone TEXT,
    aisle TEXT,
    rack TEXT,
    shelf TEXT,
    barcode TEXT,
    is_pickable BOOLEAN NOT NULL DEFAULT TRUE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT locations_code_not_blank CHECK (BTRIM(code) <> ''),
    CONSTRAINT locations_warehouse_code_unique UNIQUE (warehouse_id, code)
);

CREATE UNIQUE INDEX locations_barcode_unique_idx
    ON locations (barcode)
    WHERE barcode IS NOT NULL;
CREATE INDEX locations_warehouse_id_idx ON locations (warehouse_id);

CREATE TABLE categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    parent_id UUID REFERENCES categories(id),
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT categories_name_not_blank CHECK (BTRIM(name) <> ''),
    CONSTRAINT categories_not_own_parent CHECK (parent_id IS NULL OR parent_id <> id)
);

CREATE UNIQUE INDEX categories_name_unique_idx ON categories (LOWER(name));
CREATE INDEX categories_parent_id_idx ON categories (parent_id);

CREATE TABLE products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    category_id UUID REFERENCES categories(id),
    sku TEXT NOT NULL UNIQUE,
    barcode TEXT,
    name TEXT NOT NULL,
    unit TEXT NOT NULL DEFAULT 'case',
    is_lot_tracked BOOLEAN NOT NULL DEFAULT TRUE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT products_sku_not_blank CHECK (BTRIM(sku) <> ''),
    CONSTRAINT products_name_not_blank CHECK (BTRIM(name) <> ''),
    CONSTRAINT products_unit_not_blank CHECK (BTRIM(unit) <> '')
);

CREATE UNIQUE INDEX products_barcode_unique_idx
    ON products (barcode)
    WHERE barcode IS NOT NULL;
CREATE INDEX products_category_id_idx ON products (category_id);

CREATE TABLE lots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id UUID NOT NULL REFERENCES products(id),
    lot_number TEXT NOT NULL,
    expiration_date DATE,
    received_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT lots_number_not_blank CHECK (BTRIM(lot_number) <> ''),
    CONSTRAINT lots_product_number_unique UNIQUE (product_id, lot_number)
);

CREATE INDEX lots_product_expiration_idx
    ON lots (product_id, expiration_date, received_at);

CREATE TABLE inventory_balances (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    location_id UUID NOT NULL REFERENCES locations(id),
    product_id UUID NOT NULL REFERENCES products(id),
    lot_id UUID REFERENCES lots(id),
    quantity NUMERIC(18, 3) NOT NULL DEFAULT 0,
    reserved_quantity NUMERIC(18, 3) NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT inventory_quantity_nonnegative CHECK (quantity >= 0),
    CONSTRAINT inventory_reserved_nonnegative CHECK (reserved_quantity >= 0),
    CONSTRAINT inventory_reserved_within_quantity CHECK (reserved_quantity <= quantity),
    CONSTRAINT inventory_balance_unique
        UNIQUE NULLS NOT DISTINCT (location_id, product_id, lot_id)
);

CREATE INDEX inventory_balances_product_lot_idx
    ON inventory_balances (product_id, lot_id);
CREATE INDEX inventory_balances_location_idx
    ON inventory_balances (location_id);

CREATE TABLE stock_movements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    movement_type TEXT NOT NULL,
    product_id UUID NOT NULL REFERENCES products(id),
    lot_id UUID REFERENCES lots(id),
    from_location_id UUID REFERENCES locations(id),
    to_location_id UUID REFERENCES locations(id),
    quantity NUMERIC(18, 3) NOT NULL,
    unit_cost NUMERIC(18, 4),
    reference TEXT,
    notes TEXT,
    performed_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT stock_movements_type_valid
        CHECK (movement_type IN ('receive', 'pick', 'transfer', 'adjust')),
    CONSTRAINT stock_movements_quantity_positive CHECK (quantity > 0),
    CONSTRAINT stock_movements_cost_nonnegative CHECK (unit_cost IS NULL OR unit_cost >= 0),
    CONSTRAINT stock_movements_location_shape CHECK (
        (movement_type = 'receive' AND from_location_id IS NULL AND to_location_id IS NOT NULL)
        OR (movement_type = 'pick' AND from_location_id IS NOT NULL AND to_location_id IS NULL)
        OR (
            movement_type = 'transfer'
            AND from_location_id IS NOT NULL
            AND to_location_id IS NOT NULL
            AND from_location_id <> to_location_id
        )
        OR (
            movement_type = 'adjust'
            AND ((from_location_id IS NULL) <> (to_location_id IS NULL))
        )
    )
);

CREATE INDEX stock_movements_product_created_idx
    ON stock_movements (product_id, created_at DESC);
CREATE INDEX stock_movements_lot_id_idx ON stock_movements (lot_id);
CREATE INDEX stock_movements_from_location_idx ON stock_movements (from_location_id);
CREATE INDEX stock_movements_to_location_idx ON stock_movements (to_location_id);
CREATE INDEX stock_movements_reference_idx
    ON stock_movements (reference)
    WHERE reference IS NOT NULL;

CREATE TABLE cost_layers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    warehouse_id UUID NOT NULL REFERENCES warehouses(id),
    product_id UUID NOT NULL REFERENCES products(id),
    lot_id UUID REFERENCES lots(id),
    source_movement_id UUID NOT NULL UNIQUE REFERENCES stock_movements(id),
    original_quantity NUMERIC(18, 3) NOT NULL,
    remaining_quantity NUMERIC(18, 3) NOT NULL,
    unit_cost NUMERIC(18, 4) NOT NULL,
    received_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT cost_layers_original_positive CHECK (original_quantity > 0),
    CONSTRAINT cost_layers_remaining_valid
        CHECK (remaining_quantity >= 0 AND remaining_quantity <= original_quantity),
    CONSTRAINT cost_layers_unit_cost_nonnegative CHECK (unit_cost >= 0)
);

CREATE INDEX cost_layers_fifo_idx
    ON cost_layers (warehouse_id, product_id, received_at, id)
    WHERE remaining_quantity > 0;

CREATE TABLE audit_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_user_id UUID REFERENCES users(id),
    action TEXT NOT NULL,
    entity_type TEXT NOT NULL,
    entity_id UUID,
    before_state JSONB,
    after_state JSONB,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT audit_records_action_not_blank CHECK (BTRIM(action) <> ''),
    CONSTRAINT audit_records_entity_type_not_blank CHECK (BTRIM(entity_type) <> '')
);

CREATE INDEX audit_records_entity_idx
    ON audit_records (entity_type, entity_id, occurred_at DESC);
CREATE INDEX audit_records_actor_idx
    ON audit_records (actor_user_id, occurred_at DESC);
CREATE INDEX audit_records_occurred_at_idx ON audit_records (occurred_at DESC);

CREATE FUNCTION prevent_immutable_change()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    RAISE EXCEPTION '% records are append-only', TG_TABLE_NAME;
END;
$$;

CREATE TRIGGER stock_movements_immutable
BEFORE UPDATE OR DELETE ON stock_movements
FOR EACH ROW EXECUTE FUNCTION prevent_immutable_change();

CREATE TRIGGER audit_records_immutable
BEFORE UPDATE OR DELETE ON audit_records
FOR EACH ROW EXECUTE FUNCTION prevent_immutable_change();

INSERT INTO roles (code, name)
VALUES
    ('admin', 'Administrator'),
    ('warehouse_manager', 'Warehouse Manager'),
    ('picker', 'Picker'),
    ('viewer', 'Viewer')
ON CONFLICT (code) DO NOTHING;
