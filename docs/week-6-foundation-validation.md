# Week 6 foundation validation

Validation was run on 28 July 2026 for GitHub Issues #2 and #3.

## Database and ERD

```text
$ docker compose run --rm migrate
2/u initial_schema
```

PostgreSQL inspection confirmed:

- 12 application tables created.
- `admin`, `warehouse_manager`, `picker`, and `viewer` roles seeded.
- update/delete protection installed on `stock_movements`.
- update/delete protection installed on `audit_records`.

The schema diagram and domain rules are documented in
[`database-erd.md`](database-erd.md).

## Frontend foundation

```text
$ npm run typecheck
passed

$ npm run build
vite v8.1.5
180 modules transformed
production build completed

$ docker compose build frontend
completed
```

## Integrated stack

`docker compose up -d --build api frontend` completed and the smoke tests
returned:

```json
{"success":true,"data":{"status":"ready","environment":"development","database":"up"}}
```

- <http://localhost:5173> returned the compiled BWIMS application shell.
- <http://localhost:5173/ready> successfully proxied to the API.
- <http://localhost:8080/ready> returned database readiness.

## Regression checks

```text
$ go test ./...
passed

$ go vet ./...
passed

$ docker compose config --quiet
passed
```

This evidence covers implementation validation. Issue closure still requires
the assigned QA review and the repository's pull-request workflow.
