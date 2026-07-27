DROP TRIGGER IF EXISTS audit_records_immutable ON audit_records;
DROP TRIGGER IF EXISTS stock_movements_immutable ON stock_movements;
DROP FUNCTION IF EXISTS prevent_immutable_change();

DROP TABLE IF EXISTS audit_records;
DROP TABLE IF EXISTS cost_layers;
DROP TABLE IF EXISTS stock_movements;
DROP TABLE IF EXISTS inventory_balances;
DROP TABLE IF EXISTS lots;
DROP TABLE IF EXISTS products;
DROP TABLE IF EXISTS categories;
DROP TABLE IF EXISTS locations;
DROP TABLE IF EXISTS refresh_tokens;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS warehouses;
DROP TABLE IF EXISTS roles;
