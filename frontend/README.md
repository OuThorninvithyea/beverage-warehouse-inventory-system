# BWIMS frontend

The frontend foundation uses Vue 3, TypeScript, Vite, PrimeVue, Pinia, and Vue
Router. It includes a responsive application shell, dashboard and login
placeholders, shared API response types, and a fetch client aligned with
`docs/api-contract.md`.

## Local setup

```bash
cd frontend
cp .env.example .env
npm install
npm run dev
```

Open <http://localhost:5173>. Vite proxies `/api`, `/health`, and `/ready` to
the Go API at <http://localhost:8080>, avoiding a separate development CORS
configuration.

## Environment

`VITE_API_BASE_URL` controls the versioned API root. Its default is `/api/v1`.
Never put credentials or secrets in a `VITE_` variable because Vite exposes
these values to browser code.

## Validation

```bash
npm run typecheck
npm run build
```

From the repository root, `docker compose up --build` serves the frontend at
<http://localhost:5173> and proxies API requests to the `api` container.
