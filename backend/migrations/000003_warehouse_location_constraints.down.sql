DROP INDEX IF EXISTS locations_warehouse_code_lower_unique_idx;

ALTER TABLE locations
ADD CONSTRAINT locations_warehouse_code_unique UNIQUE (warehouse_id, code);

DROP INDEX IF EXISTS warehouses_code_lower_unique_idx;

ALTER TABLE warehouses
ADD CONSTRAINT warehouses_code_key UNIQUE (code);
