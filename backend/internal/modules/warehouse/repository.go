package warehouse

import "context"

type Repository interface {
	ListWarehouses(context.Context, *string, ListFilter) ([]Warehouse, error)
	GetWarehouse(context.Context, string) (Warehouse, error)
	CreateWarehouse(context.Context, WarehouseInput) (Warehouse, error)
	UpdateWarehouse(context.Context, string, WarehouseInput) (Warehouse, error)
	DeactivateWarehouse(context.Context, string) error
}
