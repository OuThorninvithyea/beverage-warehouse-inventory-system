.PHONY: backend-format backend-test backend-vet migrate-up migrate-down up down

backend-format:
	cd backend && gofmt -w $$(find . -name '*.go' -type f)

backend-test:
	cd backend && go test ./...

backend-vet:
	cd backend && go vet ./...

migrate-up:
	docker compose run --rm migrate

migrate-down:
	docker compose run --rm migrate -path=/migrations -database='postgres://bwims:bwims@postgres:5432/bwims?sslmode=disable' down 1

up:
	docker compose up --build

down:
	docker compose down
