package health

import (
	"context"
	"errors"
	"testing"
)

type fakeRepository struct {
	err error
}

func (f fakeRepository) Ping(context.Context) error {
	return f.err
}

func TestLiveness(t *testing.T) {
	service := NewService(fakeRepository{}, "test")
	status := service.Liveness()

	if status.Status != "ok" || status.Environment != "test" {
		t.Fatalf("Liveness() = %+v", status)
	}
}

func TestReadiness(t *testing.T) {
	tests := []struct {
		name       string
		repoError  error
		wantStatus string
		wantDB     string
		wantError  bool
	}{
		{name: "database is available", wantStatus: "ready", wantDB: "up"},
		{name: "database is unavailable", repoError: errors.New("unavailable"), wantStatus: "not_ready", wantDB: "down", wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewService(fakeRepository{err: tt.repoError}, "test")
			status, err := service.Readiness(context.Background())
			if (err != nil) != tt.wantError {
				t.Fatalf("Readiness() error = %v, wantError %v", err, tt.wantError)
			}
			if status.Status != tt.wantStatus || status.Database != tt.wantDB {
				t.Fatalf("Readiness() = %+v", status)
			}
		})
	}
}
