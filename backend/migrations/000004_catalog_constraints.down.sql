DROP INDEX IF EXISTS products_active_created_idx;
DROP INDEX IF EXISTS products_category_active_idx;
DROP INDEX IF EXISTS categories_parent_active_idx;
DROP INDEX IF EXISTS products_sku_lower_unique_idx;

ALTER TABLE products
    ADD CONSTRAINT products_sku_key UNIQUE (sku);

ALTER TABLE categories DROP COLUMN is_active;
