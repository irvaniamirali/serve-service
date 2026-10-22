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

## API workflow

The frontend consumes:

- `POST /api/v1/applications` and `GET /api/v1/applications/{id}`
- `GET /api/v1/applications?limit=20&offset=0`
- `POST /api/v1/applications/{id}/deployments`
- `GET /api/v1/deployments/{id}`
- `GET /api/v1/deployments/{id}/logs`
- `GET /api/v1/applications/{id}/deployments?limit=20&offset=0`

Applications are listed in pages and deployment history is scoped to a selected application; there is no global deployment list. Register a repository and select it to request a deployment. Leave the commit SHA blank to build the latest commit on the repository's GitHub default branch, or enter a full 40- or 64-character SHA to build that exact revision. The API resolves and persists the default-branch commit before queueing the deployment, so later branch changes do not affect the build. Deployment details expose the resolved commit, image identity, and persisted build output. Runtime deployment is not implemented.

GitHub webhook matching can be enabled by providing the numeric GitHub repository ID when registering an application. The webhook secret remains server configuration and is never requested by the UI. The API has no authentication; treat the UI and build-log endpoint as operator-only. See `../docs/build-engine.md` for Docker prerequisites and build security limitations.

Space UI components are installed as editable source under `src/components/ui` from the official `@spaceui` registry.
