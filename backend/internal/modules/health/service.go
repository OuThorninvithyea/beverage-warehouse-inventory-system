package health

import "context"

type Status struct {
	Status      string `json:"status"`
	Environment string `json:"environment"`
	Database    string `json:"database,omitempty"`
}

type Service interface {
	Liveness() Status
	Readiness(context.Context) (Status, error)
}

type service struct {
	repository  Repository
	environment string
}

func NewService(repository Repository, environment string) Service {
	return &service{repository: repository, environment: environment}
}

func (s *service) Liveness() Status {
	return Status{Status: "ok", Environment: s.environment}
}

func (s *service) Readiness(ctx context.Context) (Status, error) {
	if err := s.repository.Ping(ctx); err != nil {
		return Status{Status: "not_ready", Environment: s.environment, Database: "down"}, err
	}
	return Status{Status: "ready", Environment: s.environment, Database: "up"}, nil
}
