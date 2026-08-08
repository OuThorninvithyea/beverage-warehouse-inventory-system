package warehouse

import "context"

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
