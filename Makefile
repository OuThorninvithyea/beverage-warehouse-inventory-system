.PHONY: backend-format backend-test backend-vet frontend-install frontend-dev frontend-test frontend-typecheck frontend-build migrate-up migrate-down seed-admin seed-demo up down

backend-format:
	cd backend && gofmt -w $$(find . -name '*.go' -type f)

backend-test:
	cd backend && go test ./...

backend-vet:
	cd backend && go vet ./...

frontend-install:
	cd frontend && npm install

frontend-dev:
	cd frontend && npm run dev

frontend-test:
	cd frontend && npm test

frontend-typecheck:
	cd frontend && npm run typecheck

frontend-build:
	cd frontend && npm run build

migrate-up:
	docker compose run --rm migrate

migrate-down:
	docker compose run --rm migrate -path=/migrations -database='postgres://bwims:bwims@postgres:5432/bwims?sslmode=disable' down 1

seed-admin: migrate-up
	docker compose run --build --rm -e SEED_DEMO_DATA=false seed

seed-demo: migrate-up
	docker compose run --build --rm seed

up:
	docker compose up --build

down:
	docker compose down
