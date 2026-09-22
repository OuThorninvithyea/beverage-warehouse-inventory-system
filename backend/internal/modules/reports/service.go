package reports

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/OuThorninvithyea/beverage-warehouse-inventory-system/backend/internal/modules/auth"
)

var (
	ErrForbidden  = errors.New("actor is not permitted to read this report")
	ErrValidation = errors.New("report request data is invalid")
)

// Window and row limits. Reports are aggregates, not paginated lists, so they
// are bounded here rather than by a cursor.
const (
	defaultWindowDays = 30
	maxWindowDays     = 365
	defaultRowLimit   = 50
	maxRowLimit       = 500
)

type Service interface {
	Dashboard(ctx context.Context, actor Actor, filter Filter) (Dashboard, error)
	Valuation(ctx context.Context, actor Actor, filter Filter) (Valuation, error)
	MovementSummary(ctx context.Context, actor Actor, filter Filter) (MovementSummary, error)
	Velocity(ctx context.Context, actor Actor, filter Filter) (Velocity, error)
}

type service struct {
	repository Repository
}

func NewService(repository Repository) Service {
	return &service{repository: repository}
}

// canRead follows plan.md: reports are a management view, so pickers and
// viewers are excluded.
func canRead(role string) bool {
	switch role {
	case auth.RoleAdmin, auth.RoleWarehouseManager:
		return true
	}
	return false
}

// resolveFilter enforces warehouse scoping and clamps the window and limit.
// A non-admin is always pinned to their own warehouse, and asking for another
// one is a refusal rather than a silent rewrite.
func resolveFilter(actor Actor, filter Filter) (Filter, error) {
	if !canRead(actor.Role) {
		return Filter{}, ErrForbidden
	}

	if actor.Role != auth.RoleAdmin {
		if actor.WarehouseID == nil || strings.TrimSpace(*actor.WarehouseID) == "" {
			return Filter{}, ErrForbidden
		}
		scoped := *actor.WarehouseID
		if filter.WarehouseID != nil && *filter.WarehouseID != scoped {
			return Filter{}, ErrForbidden
		}
		filter.WarehouseID = &scoped
	}

	if filter.WarehouseID != nil {
		if _, err := uuid.Parse(strings.TrimSpace(*filter.WarehouseID)); err != nil {
			return Filter{}, ErrValidation
		}
	}

	if filter.Days <= 0 {
		filter.Days = defaultWindowDays
	}
	if filter.Days > maxWindowDays {
		filter.Days = maxWindowDays
	}
	if filter.Limit <= 0 {
		filter.Limit = defaultRowLimit
	}
	if filter.Limit > maxRowLimit {
		filter.Limit = maxRowLimit
	}
	return filter, nil
}

func (s *service) Dashboard(ctx context.Context, actor Actor, filter Filter) (Dashboard, error) {
	resolved, err := resolveFilter(actor, filter)
	if err != nil {
		return Dashboard{}, err
	}
	dashboard, err := s.repository.Dashboard(ctx, resolved)
	if err != nil {
		return Dashboard{}, fmt.Errorf("dashboard report: %w", err)
	}
	return dashboard, nil
}

func (s *service) Valuation(ctx context.Context, actor Actor, filter Filter) (Valuation, error) {
	resolved, err := resolveFilter(actor, filter)
	if err != nil {
		return Valuation{}, err
	}
	valuation, err := s.repository.Valuation(ctx, resolved)
	if err != nil {
		return Valuation{}, fmt.Errorf("valuation report: %w", err)
	}
	return valuation, nil
}

func (s *service) MovementSummary(ctx context.Context, actor Actor, filter Filter) (MovementSummary, error) {
	resolved, err := resolveFilter(actor, filter)
	if err != nil {
		return MovementSummary{}, err
	}
	summary, err := s.repository.MovementSummary(ctx, resolved)
	if err != nil {
		return MovementSummary{}, fmt.Errorf("movement summary report: %w", err)
	}
	return summary, nil
}

func (s *service) Velocity(ctx context.Context, actor Actor, filter Filter) (Velocity, error) {
	resolved, err := resolveFilter(actor, filter)
	if err != nil {
		return Velocity{}, err
	}
	velocity, err := s.repository.Velocity(ctx, resolved)
	if err != nil {
		return Velocity{}, fmt.Errorf("velocity report: %w", err)
	}
	return velocity, nil
}
