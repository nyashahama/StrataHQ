# Render Migration Automation Design

**Problem**

The backend deploy path on Render starts the application container directly from [`backend/Dockerfile`](./backend/Dockerfile), but database migrations only exist as manual `goose` commands in the Makefile and shell scripts. That means a deploy can succeed while schema changes remain unapplied, and the operator has to remember to run migrations separately from a terminal.

**Goal**

Automate database migrations for Render deployments so schema updates happen without manual terminal work, while keeping migrations separate from normal app startup.

## Options Considered

### Option 1: Run migrations inside the app container on every startup

The container entrypoint would run `goose up` and then launch the server.

**Pros**
- Minimal Render configuration
- Works even without a dedicated pre-deploy hook

**Cons**
- Multiple instances can race migrations during restarts or scale-out
- Normal restarts become schema-changing events
- Harder to reason about failure boundaries
- Startup failures mix migration failure and app failure into one path

### Option 2: Recommended: Render pre-deploy migration command

The Docker image is extended so it can run either `serve` or `migrate`. Render runs a pre-deploy command that executes migrations before the new release becomes active, and normal runtime containers only start the app server.

**Pros**
- Clean operational boundary between schema changes and app process startup
- Safer with multiple instances
- Migration failure blocks the deploy early
- Preserves a single Docker artifact for both migration and app runtime

**Cons**
- Requires a small amount of Render configuration
- The image must include the migration binary or equivalent runner

### Option 3: Keep manual migrations

No code changes.

**Pros**
- Lowest implementation cost

**Cons**
- Current developer pain remains
- Easy to forget during deploy
- Deploy correctness depends on operator memory

## Chosen Approach

Use **Option 2**.

The backend image should support two explicit modes:
- `serve`: run the backend server only
- `migrate`: run pending goose migrations only

Render should call `migrate` as a pre-deploy step and continue using `serve` for the runtime process.

## Design

### Container Behavior

The backend Docker image will:
- include the compiled server binary
- include the `db/migrations` directory
- include the `goose` binary in the runtime image
- include a small entrypoint script that dispatches to `serve` or `migrate`

The dispatch contract will be:
- `./entrypoint serve` → start the backend server
- `./entrypoint migrate` → run `goose up` against `DATABASE_URL`

The default container command should remain the normal server path so local Docker usage and existing runtime expectations stay stable.

### Render Behavior

Render should be configured to:
- build the same backend Docker image as today
- run a pre-deploy command equivalent to `./entrypoint migrate`
- run the normal service command equivalent to `./entrypoint serve`

This keeps one deploy artifact and one migration implementation, with Render choosing the mode per phase.

### Failure Handling

Migration mode should:
- fail fast if `DATABASE_URL` is missing
- return a non-zero exit code if goose fails
- never start the server in migration mode

Serve mode should:
- only start the server
- not attempt migrations

This makes migration failure visible as a deployment failure rather than an application boot problem.

### Local Development Impact

Local development should keep working with:
- `make migrate-up` for explicit local migration runs
- `make run` for direct server startup
- Docker Compose using the backend container in `serve` mode unless the team later chooses to add an automatic local migration service

No existing local flow should break.

### Documentation

Backend docs should be updated to explain:
- the new container modes
- the recommended Render pre-deploy command
- the fact that production deploys should no longer require a manual migration terminal step

## Testing Strategy

We should add:
- a small backend shell-level verification that `entrypoint migrate` refuses to run without `DATABASE_URL`
- a verification that `entrypoint serve` launches the server path
- Dockerfile sanity verification that goose and migrations are present in the runtime image

At minimum, code changes must still pass:
- `npm test`
- `cd backend && go test ./...`

## Constraints

- Do not make runtime app containers auto-run migrations on every startup
- Do not require a second separate repository or dedicated migration image
- Keep Render setup straightforward enough that one documented pre-deploy command is sufficient

## Success Criteria

- Deploying the backend on Render no longer requires manually opening a terminal to run migrations
- Migrations execute before the new app version starts serving traffic
- Normal container restarts do not trigger migrations
- The same Docker artifact supports both migration and serving modes
