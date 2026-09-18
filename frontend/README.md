# BWIMS frontend

The frontend foundation uses Vue 3, TypeScript, Vite, PrimeVue, Pinia, and Vue
Router. It includes a responsive application shell, dashboard and login
placeholders, shared API response types, a fetch client aligned with
`docs/api-contract.md`, and a `/barcode-test` route for camera/manual barcode
validation.

## Local setup

```bash
cd frontend
cp .env.example .env
npm install
npm run dev
```

Open <http://localhost:6000>. Vite proxies `/api`, `/health`, and `/ready` to
the Go API at <http://localhost:8080>, avoiding a separate development CORS
configuration.

After `make seed-admin`, sign in with the documented development administrator.
The Pinia session store rotates the refresh token when restoring a browser-tab
session. Tokens are kept only in `sessionStorage` for this MVP foundation.

## Environment

`VITE_API_BASE_URL` controls the versioned API root. Its default is `/api/v1`.
Never put credentials or secrets in a `VITE_` variable because Vite exposes
these values to browser code.

## Validation

```bash
npm run typecheck
npm test
npm run build
```

From the repository root, `docker compose up --build` serves the frontend at
<http://localhost:6000> and proxies API requests to the `api` container.
