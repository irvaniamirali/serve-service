# Serve Service frontend

React 19 + TypeScript application using Vite, Tailwind CSS v4, and Space UI source components.

## Local development

```fish
cp .env.example .env
npm install
npm run dev
```

Start the Go API separately with `make run` from the repository root. Vite proxies `/api` and `/health` to `BACKEND_URL` (default `http://localhost:8000`). The API prefix defaults to `/api/v1`; override it with `VITE_API_BASE_URL`.

Run the production TypeScript/Vite build and API client tests with `npm run build` and `npm test`.

## Existing API boundary

The frontend consumes:

- `POST /api/v1/applications` and `GET /api/v1/applications/{id}`
- `POST /api/v1/applications/{id}/deployments`
- `GET /api/v1/deployments/{id}`
- `GET /api/v1/applications/{id}/deployments?limit=20&offset=0`

There is no application collection endpoint or global deployment list. Listing therefore requires a known application ID. A successful deployment request only records a queued request; GitHub webhooks, builds, and runtime deployment are not implemented.

Space UI components are installed as editable source under `src/components/ui` from the official `@spaceui` registry.
