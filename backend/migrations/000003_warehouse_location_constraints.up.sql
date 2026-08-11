ALTER TABLE warehouses
DROP CONSTRAINT warehouses_code_key;

CREATE UNIQUE INDEX warehouses_code_lower_unique_idx
ON warehouses (LOWER(code));

ALTER TABLE locations
DROP CONSTRAINT locations_warehouse_code_unique;

CREATE UNIQUE INDEX locations_warehouse_code_lower_unique_idx
ON locations (warehouse_id, LOWER(code));
