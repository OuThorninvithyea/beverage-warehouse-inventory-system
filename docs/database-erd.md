# BWIMS database design and ERD

This is the initial PostgreSQL 16 schema for GitHub Issue #2. It supports
multi-warehouse stock, product and location barcodes, lot and expiry tracking,
FEFO physical picking, FIFO valuation, immutable stock movements, and an
append-only audit trail.

## Entity relationship diagram

```mermaid
erDiagram
    ROLES ||--o{ USERS : grants
    WAREHOUSES ||--o{ USERS : scopes
    USERS ||--o{ REFRESH_TOKENS : owns
    WAREHOUSES ||--o{ LOCATIONS : contains
    CATEGORIES ||--o{ CATEGORIES : groups
    CATEGORIES ||--o{ PRODUCTS : classifies
    PRODUCTS ||--o{ LOTS : identifies
    LOCATIONS ||--o{ INVENTORY_BALANCES : holds
    PRODUCTS ||--o{ INVENTORY_BALANCES : balances
    LOTS ||--o{ INVENTORY_BALANCES : separates
    PRODUCTS ||--o{ STOCK_MOVEMENTS : moves
    LOTS ||--o{ STOCK_MOVEMENTS : traces
    LOCATIONS ||--o{ STOCK_MOVEMENTS : source
    LOCATIONS ||--o{ STOCK_MOVEMENTS : destination
    USERS ||--o{ STOCK_MOVEMENTS : performs
    WAREHOUSES ||--o{ COST_LAYERS : values
    PRODUCTS ||--o{ COST_LAYERS : costs
    LOTS ||--o{ COST_LAYERS : traces
    STOCK_MOVEMENTS ||--o| COST_LAYERS : creates
    USERS ||--o{ AUDIT_RECORDS : acts

    ROLES {
        uuid id PK
        text code UK
        text name
    }
    WAREHOUSES {
        uuid id PK
        text code UK
        text name
        boolean is_active
    }
    USERS {
        uuid id PK
        uuid role_id FK
        uuid warehouse_id FK
        text email UK
        text password_hash
        text full_name
        boolean is_active
    }
    REFRESH_TOKENS {
        uuid id PK
        uuid user_id FK
        text token_hash UK
        timestamptz expires_at
        timestamptz revoked_at
    }
    LOCATIONS {
        uuid id PK
        uuid warehouse_id FK
        text code
        text barcode UK
        boolean is_pickable
    }
    CATEGORIES {
        uuid id PK
        uuid parent_id FK
        text name UK
    }
    PRODUCTS {
        uuid id PK
        uuid category_id FK
        text sku UK
        text barcode UK
        text name
        text unit
        boolean is_lot_tracked
    }
    LOTS {
        uuid id PK
        uuid product_id FK
        text lot_number
        date expiration_date
        timestamptz received_at
    }
    INVENTORY_BALANCES {
        uuid id PK
        uuid location_id FK
        uuid product_id FK
        uuid lot_id FK
        numeric quantity
        numeric reserved_quantity
    }
    STOCK_MOVEMENTS {
        uuid id PK
        text movement_type
        uuid product_id FK
        uuid lot_id FK
        uuid from_location_id FK
        uuid to_location_id FK
        numeric quantity
        numeric unit_cost
        uuid performed_by FK
        timestamptz created_at
    }
    COST_LAYERS {
        uuid id PK
        uuid warehouse_id FK
        uuid product_id FK
        uuid lot_id FK
        uuid source_movement_id FK
        numeric original_quantity
        numeric remaining_quantity
        numeric unit_cost
        timestamptz received_at
    }
    AUDIT_RECORDS {
        uuid id PK
        uuid actor_user_id FK
        text action
        text entity_type
        uuid entity_id
        jsonb before_state
        jsonb after_state
        timestamptz occurred_at
    }
```

## Business rules represented in the schema

- Product and location barcodes are unique when present.
- A lot number is unique within a product; expiry is indexed with receipt time
  so a service can select stock using FEFO.
- Each location/product/lot combination has one inventory balance, including
  non-lot-tracked products where `lot_id` is null.
- Inventory and reservations cannot be negative, and reservations cannot exceed
  the physical balance.
- Receive, pick, transfer, and adjustment movements require valid source and
  destination shapes. Stock movement rows cannot be changed or deleted.
- FIFO valuation remains separate from FEFO selection. Open cost layers are
  indexed by warehouse, product, receipt time, and ID.
- Audit records capture before/after JSON and cannot be changed or deleted.
- Email uniqueness is case-insensitive.

## Transaction boundary

The application service must execute each receive, pick, transfer, or adjustment
inside one PostgreSQL transaction:

1. Lock the affected inventory balance and open cost-layer rows.
2. Validate stock, reservation, lot, and location rules.
3. Insert the immutable stock movement.
4. Update the source and/or destination inventory balances.
5. Create or consume FIFO cost layers when valuation changes.
6. Insert the immutable audit record.
7. Commit all changes together, or roll back all changes on any error.

The migration supplies constraints and indexes for these transactions; the Go
inventory module will implement the service operations in Weeks 9–10.

## Migration commands

```bash
docker compose up -d postgres
docker compose run --rm migrate
```

To roll back only this schema migration:

```bash
make migrate-down
```

The `pgcrypto` extension is intentionally retained during rollback because it
may be shared by later migrations.
