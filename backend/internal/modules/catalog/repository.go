package catalog

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) Repository {
	return &PostgresRepository{pool: pool}
}

func (repository *PostgresRepository) ListCategories(
	ctx context.Context,
	filter ListFilter,
) ([]Category, error) {
	afterTime, afterID := cursorArguments(filter.After)
	rows, err := repository.pool.Query(ctx, `
		SELECT id::text, parent_id::text, name, is_active, created_at, updated_at
		FROM categories
		WHERE ($1 = '' OR name ILIKE '%' || $1 || '%')
		  AND ($2::uuid IS NULL OR parent_id = $2::uuid)
		  AND ($3::boolean IS NULL OR is_active = $3)
		  AND (
			$4::timestamptz IS NULL
			OR (created_at, id) < ($4::timestamptz, $5::uuid)
		  )
		ORDER BY created_at DESC, id DESC
		LIMIT $6`, filter.Search, nullableString(filter.ParentID), nullableBool(filter.IsActive),
		afterTime, afterID, filter.Limit)
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	defer rows.Close()

	items := make([]Category, 0)
	for rows.Next() {
		item, err := scanCategory(rows)
		if err != nil {
			return nil, fmt.Errorf("scan category list: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate categories: %w", err)
	}
	return items, nil
}

func (repository *PostgresRepository) GetCategory(ctx context.Context, id string) (Category, error) {
	category, err := scanCategory(repository.pool.QueryRow(ctx, `
		SELECT id::text, parent_id::text, name, is_active, created_at, updated_at
		FROM categories
		WHERE id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Category{}, ErrCategoryNotFound
	}
	if err != nil {
		return Category{}, fmt.Errorf("get category: %w", err)
	}
	return category, nil
}

func (repository *PostgresRepository) CreateCategory(
	ctx context.Context,
	input CategoryInput,
) (Category, error) {
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return Category{}, fmt.Errorf("begin create category: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if input.ParentID.Set && input.ParentID.Value != nil {
		if err := lockActiveCategory(ctx, tx, *input.ParentID.Value); err != nil {
			return Category{}, err
		}
	}
	category, err := scanCategory(tx.QueryRow(ctx, `
		INSERT INTO categories (parent_id, name, is_active)
		VALUES ($1, $2, $3)
		RETURNING id::text, parent_id::text, name, is_active, created_at, updated_at`,
		optionalStringValue(input.ParentID), input.Name, boolOrDefault(input.IsActive, true)))
	if err != nil {
		return Category{}, mapCatalogWriteError("create category", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Category{}, fmt.Errorf("commit create category: %w", err)
	}
	return category, nil
}

func (repository *PostgresRepository) UpdateCategory(
	ctx context.Context,
	id string,
	input CategoryInput,
) (Category, error) {
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return Category{}, fmt.Errorf("begin update category: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := lockCategory(ctx, tx, id); err != nil {
		return Category{}, err
	}
	if input.ParentID.Set && input.ParentID.Value != nil {
		parentID := *input.ParentID.Value
		if parentID == id {
			return Category{}, ErrCategoryCycle
		}
		if err := lockActiveCategory(ctx, tx, parentID); err != nil {
			return Category{}, err
		}
		var cycle bool
		if err := tx.QueryRow(ctx, `
			WITH RECURSIVE descendants AS (
				SELECT id FROM categories WHERE parent_id = $1
				UNION ALL
				SELECT category.id
				FROM categories category
				JOIN descendants descendant ON category.parent_id = descendant.id
			)
			SELECT EXISTS (SELECT 1 FROM descendants WHERE id = $2)`, id, parentID).Scan(&cycle); err != nil {
			return Category{}, fmt.Errorf("check category cycle: %w", err)
		}
		if cycle {
			return Category{}, ErrCategoryCycle
		}
	}

	category, err := scanCategory(tx.QueryRow(ctx, `
		UPDATE categories
		SET name = $2,
		    parent_id = CASE WHEN $3 THEN $4::uuid ELSE parent_id END,
		    is_active = COALESCE($5, is_active),
		    updated_at = NOW()
		WHERE id = $1
		RETURNING id::text, parent_id::text, name, is_active, created_at, updated_at`,
		id, input.Name, input.ParentID.Set, optionalStringValue(input.ParentID), nullableBool(input.IsActive)))
	if err != nil {
		return Category{}, mapCatalogWriteError("update category", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Category{}, fmt.Errorf("commit update category: %w", err)
	}
	return category, nil
}

func (repository *PostgresRepository) DeactivateCategory(ctx context.Context, id string) error {
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin deactivate category: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var active bool
	if err := tx.QueryRow(ctx, "SELECT is_active FROM categories WHERE id=$1 FOR UPDATE", id).Scan(&active); errors.Is(err, pgx.ErrNoRows) {
		return ErrCategoryNotFound
	} else if err != nil {
		return fmt.Errorf("lock category for deactivation: %w", err)
	}
	if !active {
		return tx.Commit(ctx)
	}

	var inUse bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM categories WHERE parent_id=$1 AND is_active
			UNION ALL
			SELECT 1 FROM products WHERE category_id=$1 AND is_active
		)`, id).Scan(&inUse); err != nil {
		return fmt.Errorf("check category usage: %w", err)
	}
	if inUse {
		return ErrCategoryInUse
	}
	if _, err := tx.Exec(ctx, "UPDATE categories SET is_active=FALSE, updated_at=NOW() WHERE id=$1", id); err != nil {
		return fmt.Errorf("deactivate category: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit deactivate category: %w", err)
	}
	return nil
}

func (repository *PostgresRepository) ListProducts(
	ctx context.Context,
	filter ListFilter,
) ([]Product, error) {
	afterTime, afterID := cursorArguments(filter.After)
	rows, err := repository.pool.Query(ctx, `
		SELECT id::text, category_id::text, sku, barcode, name, unit,
		       is_lot_tracked, is_active, created_at, updated_at
		FROM products
		WHERE (
			$1 = ''
			OR sku ILIKE '%' || $1 || '%'
			OR name ILIKE '%' || $1 || '%'
			OR COALESCE(barcode, '') ILIKE '%' || $1 || '%'
		)
		  AND ($2::uuid IS NULL OR category_id = $2::uuid)
		  AND ($3::boolean IS NULL OR is_active = $3)
		  AND (
			$4::timestamptz IS NULL
			OR (created_at, id) < ($4::timestamptz, $5::uuid)
		  )
		ORDER BY created_at DESC, id DESC
		LIMIT $6`, filter.Search, nullableString(filter.CategoryID), nullableBool(filter.IsActive),
		afterTime, afterID, filter.Limit)
	if err != nil {
		return nil, fmt.Errorf("list products: %w", err)
	}
	defer rows.Close()

	items := make([]Product, 0)
	for rows.Next() {
		item, err := scanProduct(rows)
		if err != nil {
			return nil, fmt.Errorf("scan product list: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate products: %w", err)
	}
	return items, nil
}

func (repository *PostgresRepository) GetProduct(ctx context.Context, id string) (Product, error) {
	product, err := scanProduct(repository.pool.QueryRow(ctx, `
		SELECT id::text, category_id::text, sku, barcode, name, unit,
		       is_lot_tracked, is_active, created_at, updated_at
		FROM products
		WHERE id=$1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Product{}, ErrProductNotFound
	}
	if err != nil {
		return Product{}, fmt.Errorf("get product: %w", err)
	}
	return product, nil
}

func (repository *PostgresRepository) GetProductByBarcode(
	ctx context.Context,
	barcode string,
) (Product, error) {
	product, err := scanProduct(repository.pool.QueryRow(ctx, `
		SELECT id::text, category_id::text, sku, barcode, name, unit,
		       is_lot_tracked, is_active, created_at, updated_at
		FROM products
		WHERE barcode=$1 AND is_active=TRUE`, barcode))
	if errors.Is(err, pgx.ErrNoRows) {
		return Product{}, ErrProductNotFound
	}
	if err != nil {
		return Product{}, fmt.Errorf("get product by barcode: %w", err)
	}
	return product, nil
}

func (repository *PostgresRepository) CreateProduct(
	ctx context.Context,
	input ProductInput,
) (Product, error) {
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return Product{}, fmt.Errorf("begin create product: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if input.CategoryID.Set && input.CategoryID.Value != nil {
		if err := lockActiveCategory(ctx, tx, *input.CategoryID.Value); err != nil {
			return Product{}, err
		}
	}
	product, err := scanProduct(tx.QueryRow(ctx, `
		INSERT INTO products (
			category_id, sku, barcode, name, unit, is_lot_tracked, is_active
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id::text, category_id::text, sku, barcode, name, unit,
		          is_lot_tracked, is_active, created_at, updated_at`,
		optionalStringValue(input.CategoryID), input.SKU, optionalStringValue(input.Barcode),
		input.Name, input.Unit, boolOrDefault(input.IsLotTracked, true), boolOrDefault(input.IsActive, true)))
	if err != nil {
		return Product{}, mapCatalogWriteError("create product", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Product{}, fmt.Errorf("commit create product: %w", err)
	}
	return product, nil
}

func (repository *PostgresRepository) UpdateProduct(
	ctx context.Context,
	id string,
	input ProductInput,
) (Product, error) {
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return Product{}, fmt.Errorf("begin update product: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := lockProduct(ctx, tx, id); err != nil {
		return Product{}, err
	}
	if input.CategoryID.Set && input.CategoryID.Value != nil {
		if err := lockActiveCategory(ctx, tx, *input.CategoryID.Value); err != nil {
			return Product{}, err
		}
	}
	product, err := scanProduct(tx.QueryRow(ctx, `
		UPDATE products
		SET category_id = CASE WHEN $2 THEN $3::uuid ELSE category_id END,
		    sku = $4,
		    barcode = CASE WHEN $5 THEN $6 ELSE barcode END,
		    name = $7,
		    unit = $8,
		    is_lot_tracked = COALESCE($9, is_lot_tracked),
		    is_active = COALESCE($10, is_active),
		    updated_at = NOW()
		WHERE id = $1
		RETURNING id::text, category_id::text, sku, barcode, name, unit,
		          is_lot_tracked, is_active, created_at, updated_at`,
		id, input.CategoryID.Set, optionalStringValue(input.CategoryID), input.SKU,
		input.Barcode.Set, optionalStringValue(input.Barcode), input.Name, input.Unit,
		nullableBool(input.IsLotTracked), nullableBool(input.IsActive)))
	if err != nil {
		return Product{}, mapCatalogWriteError("update product", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Product{}, fmt.Errorf("commit update product: %w", err)
	}
	return product, nil
}

func (repository *PostgresRepository) DeactivateProduct(ctx context.Context, id string) error {
	tag, err := repository.pool.Exec(ctx, `
		UPDATE products SET is_active=FALSE, updated_at=NOW() WHERE id=$1`, id)
	if err != nil {
		return fmt.Errorf("deactivate product: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrProductNotFound
	}
	return nil
}

type rowScanner interface {
	Scan(...any) error
}

func scanCategory(row rowScanner) (Category, error) {
	var category Category
	err := row.Scan(&category.ID, &category.ParentID, &category.Name, &category.IsActive,
		&category.CreatedAt, &category.UpdatedAt)
	return category, err
}

func scanProduct(row rowScanner) (Product, error) {
	var product Product
	err := row.Scan(&product.ID, &product.CategoryID, &product.SKU, &product.Barcode,
		&product.Name, &product.Unit, &product.IsLotTracked, &product.IsActive,
		&product.CreatedAt, &product.UpdatedAt)
	return product, err
}

func lockCategory(ctx context.Context, tx pgx.Tx, id string) error {
	var lockedID string
	if err := tx.QueryRow(ctx, "SELECT id::text FROM categories WHERE id=$1 FOR UPDATE", id).Scan(&lockedID); errors.Is(err, pgx.ErrNoRows) {
		return ErrCategoryNotFound
	} else if err != nil {
		return fmt.Errorf("lock category: %w", err)
	}
	return nil
}

func lockActiveCategory(ctx context.Context, tx pgx.Tx, id string) error {
	var lockedID string
	if err := tx.QueryRow(ctx, "SELECT id::text FROM categories WHERE id=$1 AND is_active=TRUE FOR UPDATE", id).Scan(&lockedID); errors.Is(err, pgx.ErrNoRows) {
		return ErrCategoryNotFound
	} else if err != nil {
		return fmt.Errorf("lock active category: %w", err)
	}
	return nil
}

func lockProduct(ctx context.Context, tx pgx.Tx, id string) error {
	var lockedID string
	if err := tx.QueryRow(ctx, "SELECT id::text FROM products WHERE id=$1 FOR UPDATE", id).Scan(&lockedID); errors.Is(err, pgx.ErrNoRows) {
		return ErrProductNotFound
	} else if err != nil {
		return fmt.Errorf("lock product: %w", err)
	}
	return nil
}

func mapCatalogWriteError(operation string, err error) error {
	var pgError *pgconn.PgError
	if errors.As(err, &pgError) {
		switch pgError.ConstraintName {
		case "categories_name_unique_idx":
			return ErrCategoryNameConflict
		case "products_sku_lower_unique_idx", "products_sku_key":
			return ErrProductSKUConflict
		case "products_barcode_unique_idx":
			return ErrProductBarcodeConflict
		case "categories_parent_id_fkey", "products_category_id_fkey":
			return ErrCategoryNotFound
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

func optionalStringValue(value OptionalString) any {
	if !value.Set || value.Value == nil {
		return nil
	}
	return *value.Value
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
