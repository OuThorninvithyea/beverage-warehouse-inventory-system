package warehouse

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	ListWarehouses(context.Context, *string, ListFilter) ([]Warehouse, error)
	GetWarehouse(context.Context, string) (Warehouse, error)
	CreateWarehouse(context.Context, WarehouseInput) (Warehouse, error)
	UpdateWarehouse(context.Context, string, WarehouseInput) (Warehouse, error)
	DeactivateWarehouse(context.Context, string) error
	ListLocations(context.Context, string, ListFilter) ([]Location, error)
	GetLocation(context.Context, string, string) (Location, error)
	CreateLocation(context.Context, string, LocationInput) (Location, error)
	UpdateLocation(context.Context, string, string, LocationInput) (Location, error)
	DeactivateLocation(context.Context, string, string) error
}

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) Repository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) ListWarehouses(
	ctx context.Context,
	scope *string,
	filter ListFilter,
) ([]Warehouse, error) {
	afterTime, afterID := cursorArguments(filter.After)
	rows, err := r.pool.Query(ctx, `
		SELECT id::text, code, name, address, is_active, created_at, updated_at
		FROM warehouses
		WHERE ($1::uuid IS NULL OR id = $1::uuid)
		  AND ($2 = '' OR code ILIKE '%' || $2 || '%' OR name ILIKE '%' || $2 || '%')
		  AND ($3::boolean IS NULL OR is_active = $3)
		  AND (
			$4::timestamptz IS NULL
			OR (created_at, id) < ($4::timestamptz, $5::uuid)
		  )
		ORDER BY created_at DESC, id DESC
		LIMIT $6`, nullableString(scope), filter.Search, nullableBool(filter.IsActive), afterTime, afterID, filter.Limit)
	if err != nil {
		return nil, fmt.Errorf("list warehouses: %w", err)
	}
	defer rows.Close()

	warehouses := make([]Warehouse, 0)
	for rows.Next() {
		warehouse, err := scanWarehouse(rows)
		if err != nil {
			return nil, fmt.Errorf("scan warehouse list: %w", err)
		}
		warehouses = append(warehouses, warehouse)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate warehouses: %w", err)
	}
	return warehouses, nil
}

func (r *PostgresRepository) GetWarehouse(ctx context.Context, id string) (Warehouse, error) {
	warehouse, err := scanWarehouse(r.pool.QueryRow(ctx, `
		SELECT id::text, code, name, address, is_active, created_at, updated_at
		FROM warehouses
		WHERE id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Warehouse{}, ErrWarehouseNotFound
	}
	if err != nil {
		return Warehouse{}, fmt.Errorf("get warehouse: %w", err)
	}
	return warehouse, nil
}

func (r *PostgresRepository) CreateWarehouse(
	ctx context.Context,
	input WarehouseInput,
) (Warehouse, error) {
	warehouse, err := scanWarehouse(r.pool.QueryRow(ctx, `
		INSERT INTO warehouses (code, name, address, is_active)
		VALUES ($1, $2, $3, $4)
		RETURNING id::text, code, name, address, is_active, created_at, updated_at`,
		input.Code, input.Name, nullableString(input.Address), boolOrDefault(input.IsActive, true)))
	if err != nil {
		return Warehouse{}, mapWriteError("create warehouse", err)
	}
	return warehouse, nil
}

func (r *PostgresRepository) UpdateWarehouse(
	ctx context.Context,
	id string,
	input WarehouseInput,
) (Warehouse, error) {
	warehouse, err := scanWarehouse(r.pool.QueryRow(ctx, `
		UPDATE warehouses
		SET code = $2,
		    name = $3,
		    address = $4,
		    is_active = COALESCE($5, is_active),
		    updated_at = NOW()
		WHERE id = $1
		RETURNING id::text, code, name, address, is_active, created_at, updated_at`,
		id, input.Code, input.Name, nullableString(input.Address), nullableBool(input.IsActive)))
	if errors.Is(err, pgx.ErrNoRows) {
		return Warehouse{}, ErrWarehouseNotFound
	}
	if err != nil {
		return Warehouse{}, mapWriteError("update warehouse", err)
	}
	return warehouse, nil
}

func (r *PostgresRepository) DeactivateWarehouse(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE warehouses
		SET is_active = FALSE, updated_at = NOW()
		WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("deactivate warehouse: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrWarehouseNotFound
	}
	return nil
}

func (r *PostgresRepository) ListLocations(
	ctx context.Context,
	warehouseID string,
	filter ListFilter,
) ([]Location, error) {
	afterTime, afterID := cursorArguments(filter.After)
	rows, err := r.pool.Query(ctx, `
		SELECT id::text, warehouse_id::text, code, zone, aisle, rack, shelf, barcode,
		       is_pickable, is_active, created_at, updated_at
		FROM locations
		WHERE warehouse_id = $1
		  AND (
			$2 = ''
			OR code ILIKE '%' || $2 || '%'
			OR COALESCE(zone, '') ILIKE '%' || $2 || '%'
			OR COALESCE(aisle, '') ILIKE '%' || $2 || '%'
			OR COALESCE(rack, '') ILIKE '%' || $2 || '%'
			OR COALESCE(shelf, '') ILIKE '%' || $2 || '%'
			OR COALESCE(barcode, '') ILIKE '%' || $2 || '%'
		  )
		  AND ($3::boolean IS NULL OR is_active = $3)
		  AND ($4::boolean IS NULL OR is_pickable = $4)
		  AND (
			$5::timestamptz IS NULL
			OR (created_at, id) < ($5::timestamptz, $6::uuid)
		  )
		ORDER BY created_at DESC, id DESC
		LIMIT $7`, warehouseID, filter.Search, nullableBool(filter.IsActive),
		nullableBool(filter.IsPickable), afterTime, afterID, filter.Limit)
	if err != nil {
		return nil, fmt.Errorf("list locations: %w", err)
	}
	defer rows.Close()

	locations := make([]Location, 0)
	for rows.Next() {
		location, err := scanLocation(rows)
		if err != nil {
			return nil, fmt.Errorf("scan location list: %w", err)
		}
		locations = append(locations, location)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate locations: %w", err)
	}
	return locations, nil
}

func (r *PostgresRepository) GetLocation(
	ctx context.Context,
	warehouseID string,
	locationID string,
) (Location, error) {
	location, err := scanLocation(r.pool.QueryRow(ctx, `
		SELECT id::text, warehouse_id::text, code, zone, aisle, rack, shelf, barcode,
		       is_pickable, is_active, created_at, updated_at
		FROM locations
		WHERE warehouse_id = $1 AND id = $2`, warehouseID, locationID))
	if errors.Is(err, pgx.ErrNoRows) {
		return Location{}, ErrLocationNotFound
	}
	if err != nil {
		return Location{}, fmt.Errorf("get location: %w", err)
	}
	return location, nil
}

func (r *PostgresRepository) CreateLocation(
	ctx context.Context,
	warehouseID string,
	input LocationInput,
) (Location, error) {
	location, err := scanLocation(r.pool.QueryRow(ctx, `
		INSERT INTO locations (
			warehouse_id, code, zone, aisle, rack, shelf, barcode, is_pickable, is_active
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id::text, warehouse_id::text, code, zone, aisle, rack, shelf, barcode,
		          is_pickable, is_active, created_at, updated_at`,
		warehouseID, input.Code, nullableString(input.Zone), nullableString(input.Aisle),
		nullableString(input.Rack), nullableString(input.Shelf), nullableString(input.Barcode),
		boolOrDefault(input.IsPickable, true), boolOrDefault(input.IsActive, true)))
	if err != nil {
		return Location{}, mapWriteError("create location", err)
	}
	return location, nil
}

func (r *PostgresRepository) UpdateLocation(
	ctx context.Context,
	warehouseID string,
	locationID string,
	input LocationInput,
) (Location, error) {
	location, err := scanLocation(r.pool.QueryRow(ctx, `
		UPDATE locations
		SET code = $3,
		    zone = $4,
		    aisle = $5,
		    rack = $6,
		    shelf = $7,
		    barcode = $8,
		    is_pickable = COALESCE($9, is_pickable),
		    is_active = COALESCE($10, is_active),
		    updated_at = NOW()
		WHERE warehouse_id = $1 AND id = $2
		RETURNING id::text, warehouse_id::text, code, zone, aisle, rack, shelf, barcode,
		          is_pickable, is_active, created_at, updated_at`,
		warehouseID, locationID, input.Code, nullableString(input.Zone), nullableString(input.Aisle),
		nullableString(input.Rack), nullableString(input.Shelf), nullableString(input.Barcode),
		nullableBool(input.IsPickable), nullableBool(input.IsActive)))
	if errors.Is(err, pgx.ErrNoRows) {
		return Location{}, ErrLocationNotFound
	}
	if err != nil {
		return Location{}, mapWriteError("update location", err)
	}
	return location, nil
}

func (r *PostgresRepository) DeactivateLocation(
	ctx context.Context,
	warehouseID string,
	locationID string,
) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE locations
		SET is_active = FALSE, updated_at = NOW()
		WHERE warehouse_id = $1 AND id = $2`, warehouseID, locationID)
	if err != nil {
		return fmt.Errorf("deactivate location: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrLocationNotFound
	}
	return nil
}

type rowScanner interface {
	Scan(...any) error
}

func scanWarehouse(row rowScanner) (Warehouse, error) {
	var warehouse Warehouse
	err := row.Scan(
		&warehouse.ID,
		&warehouse.Code,
		&warehouse.Name,
		&warehouse.Address,
		&warehouse.IsActive,
		&warehouse.CreatedAt,
		&warehouse.UpdatedAt,
	)
	return warehouse, err
}

func scanLocation(row rowScanner) (Location, error) {
	var location Location
	err := row.Scan(
		&location.ID,
		&location.WarehouseID,
		&location.Code,
		&location.Zone,
		&location.Aisle,
		&location.Rack,
		&location.Shelf,
		&location.Barcode,
		&location.IsPickable,
		&location.IsActive,
		&location.CreatedAt,
		&location.UpdatedAt,
	)
	return location, err
}

func mapWriteError(operation string, err error) error {
	var pgError *pgconn.PgError
	if errors.As(err, &pgError) {
		switch pgError.ConstraintName {
		case "warehouses_code_key", "warehouses_code_lower_unique_idx":
			return ErrWarehouseCodeConflict
		case "locations_warehouse_code_unique", "locations_warehouse_code_lower_unique_idx":
			return ErrLocationCodeConflict
		case "locations_barcode_unique_idx":
			return ErrLocationBarcodeConflict
		}
	}
	return fmt.Errorf("%s: %w", operation, err)
}

func cursorArguments(cursor *Cursor) (any, any) {
	if cursor == nil {
		return nil, nil
	}
	return cursor.CreatedAt, cursor.ID
}

func nullableString(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullableBool(value *bool) any {
	if value == nil {
		return nil
	}
	return *value
}

func boolOrDefault(value *bool, fallback bool) bool {
	if value == nil {
		return fallback
	}
	return *value
}
