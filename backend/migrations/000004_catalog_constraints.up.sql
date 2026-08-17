ALTER TABLE categories
    ADD COLUMN is_active BOOLEAN NOT NULL DEFAULT TRUE;

ALTER TABLE products DROP CONSTRAINT products_sku_key;

CREATE UNIQUE INDEX products_sku_lower_unique_idx
    ON products (LOWER(sku));

CREATE INDEX categories_parent_active_idx
    ON categories (parent_id, is_active);

CREATE INDEX products_category_active_idx
    ON products (category_id, is_active);

CREATE INDEX products_active_created_idx
    ON products (is_active, created_at DESC, id DESC);
