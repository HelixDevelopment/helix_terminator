# Org Service

HelixTerminator microservice that manages organizations, teams, and memberships.

## Features
- Organization CRUD with soft delete
- Team management within organizations
- Membership management with roles (owner, admin, member)
- Health and readiness checks with DB ping
- Graceful shutdown

## API Endpoints

### Health
- `GET /healthz` — Health check
- `GET /healthz/ready` — Readiness check (returns 503 if DB unavailable)

### Organizations
- `POST /api/v1/orgs` — Create organization
- `GET /api/v1/orgs` — List organizations (query: search, limit, offset)
- `GET /api/v1/orgs/:id` — Get organization by ID
- `GET /api/v1/orgs/by-slug/:slug` — Get organization by slug
- `PUT /api/v1/orgs/:id` — Update organization
- `DELETE /api/v1/orgs/:id` — Soft delete organization

### Teams
- `POST /api/v1/orgs/:id/teams` — Create team
- `GET /api/v1/orgs/:id/teams` — List teams
- `GET /api/v1/teams/:id` — Get team
- `PUT /api/v1/teams/:id` — Update team
- `DELETE /api/v1/teams/:id` — Delete team

### Members
- `POST /api/v1/orgs/:id/members` — Add member
- `GET /api/v1/orgs/:id/members` — List members (query: team_id, role, limit, offset)
- `PUT /api/v1/orgs/:id/members/:user_id` — Update member
- `DELETE /api/v1/orgs/:id/members/:user_id` — Remove member

## Dependencies
- `gin-gonic/gin` v1.10.0
- `jackc/pgx/v5`
- `google/uuid`
- `stretchr/testify`

## Running
```
export DATABASE_URL=postgres://user:pass@localhost/org_service
go run ./cmd/org-service
```

## Testing
```
go test -v -cover ./...
```
