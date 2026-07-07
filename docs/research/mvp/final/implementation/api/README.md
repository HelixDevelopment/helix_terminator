# 04 — API Specification

**Status:** `Complete`  
**Module:** A + B  
**Authority:** `CANONICAL_FACTS.md` (CD-7, CD-8) + `SERVICE_REGISTRY.md`  

---

## Overview

HelixTerminator exposes a versioned HTTP/REST API for all external clients (web, desktop, CLI, third-party integrations). Internal microservice communication uses gRPC for low-latency, strongly-typed contracts.

- **221 REST API endpoints** documented across all services
- **OpenAPI 3.1** specification served at `/api/v1/openapi.json` and `/api/v1/openapi.yaml`
- **gRPC** internal service-to-service on mTLS
- **WebSocket** for real-time terminal I/O and collaboration

---

## API Design Principles

### REST Conventions

| Pattern | Example | Meaning |
|---------|---------|---------|
| `/api/v1/{resource}` | `/api/v1/hosts` | Collection |
| `/api/v1/{resource}/{id}` | `/api/v1/hosts/abc-123` | Single resource |
| `/api/v1/{resource}/{id}/{sub}` | `/api/v1/hosts/abc-123/connections` | Sub-collection |
| `/api/v1/{resource}/{id}/{action}` | `/api/v1/sessions/abc-123/resize` | Action on resource |

Rules: lowercase, hyphens, no trailing slashes, no verbs in URLs. UUIDs in lowercase hex with hyphens.

### HTTP Methods

| Method | Semantics | Idempotent | Safe |
|--------|-----------|------------|------|
| `GET` | Retrieve | Yes | Yes |
| `POST` | Create / invoke action | No | No |
| `PUT` | Replace entirely | Yes | No |
| `PATCH` | Partial update (JSON Merge Patch, RFC 7396) | No | No |
| `DELETE` | Remove | Yes | No |

### Authentication

All authenticated endpoints require a Bearer token:
```
Authorization: Bearer eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9...
```

**EdDSA (Ed25519) signed JWTs** (RFC 8037). Canonical per CD-7.

| Token Type | Lifetime | Storage |
|------------|----------|---------|
| Access token | 15 minutes | Memory |
| Refresh token | 30 days (sliding) | HttpOnly Secure cookie |
| API key | No expiry (until revoked) | Hashed in DB |
| Session (WebSocket) | Duration of connection | Redis |

### Rate Limiting

| Layer | Limit | Window | Scope |
|-------|-------|--------|-------|
| Global (unauthenticated) | 60 req | 1 minute | Per IP |
| Global (authenticated) | 1000 req | 1 minute | Per user |
| Auth endpoints | 10 req | 1 minute | Per IP |
| Sensitive operations | 5 req | 15 minutes | Per user |
| AI endpoints | 100 req | 1 hour | Per user |
| WebSocket connections | 20 concurrent | — | Per user |

### Error Responses (RFC 7807)

All errors use `application/problem+json`:
```json
{
  "type": "https://errors.helixterminator.io/v1/validation-error",
  "title": "Validation Error",
  "status": 422,
  "detail": "The request body contains invalid field values.",
  "instance": "/api/v1/hosts",
  "trace_id": "7f3a9b2c-1d4e-5f6a-8b9c-0d1e2f3a4b5c",
  "errors": [
    { "field": "hostname", "code": "required", "message": "hostname is required" }
  ]
}
```

---

## Endpoint Inventory by Service

### 1. Auth Service (`/api/v1/auth`) — 14 endpoints

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/auth/register` | Create user account |
| POST | `/api/v1/auth/login` | Authenticate (password or MFA challenge) |
| POST | `/api/v1/auth/logout` | Invalidate session |
| POST | `/api/v1/auth/refresh` | Exchange refresh token |
| POST | `/api/v1/auth/mfa/totp/setup` | Initiate TOTP setup |
| POST | `/api/v1/auth/mfa/totp/verify` | Verify TOTP code |
| POST | `/api/v1/auth/mfa/fido2/register/begin` | Begin FIDO2 registration |
| POST | `/api/v1/auth/mfa/fido2/register/complete` | Complete FIDO2 registration |
| POST | `/api/v1/auth/mfa/fido2/authenticate/begin` | Begin FIDO2 auth challenge |
| POST | `/api/v1/auth/mfa/fido2/authenticate/complete` | Complete FIDO2 auth |
| GET | `/api/v1/auth/devices` | List trusted devices |
| DELETE | `/api/v1/auth/devices/{deviceId}` | Revoke device |
| POST | `/api/v1/auth/sso/{provider}/authorize` | Initiate SSO OAuth flow |
| POST | `/api/v1/auth/sso/{provider}/callback` | Handle SSO callback |
| POST | `/api/v1/auth/api-keys` | Create API key |
| GET | `/api/v1/auth/api-keys` | List API keys |
| DELETE | `/api/v1/auth/api-keys/{keyId}` | Revoke API key |
| GET | `/api/v1/auth/sessions` | List active sessions |
| DELETE | `/api/v1/auth/sessions/{sessionId}` | Terminate session |

### 2. User Service (`/api/v1/users`) — 6 endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/users/me` | Get current user profile |
| PATCH | `/api/v1/users/me` | Update profile |
| DELETE | `/api/v1/users/me` | Delete account |
| GET | `/api/v1/users/me/preferences` | Get preferences |
| PUT | `/api/v1/users/me/preferences` | Update preferences |
| POST | `/api/v1/users/me/avatar` | Upload avatar |

### 3. Vault Service (`/api/v1/vaults`) — 12 endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/vaults` | List vaults |
| POST | `/api/v1/vaults` | Create vault |
| GET | `/api/v1/vaults/{vaultId}` | Get vault |
| PUT | `/api/v1/vaults/{vaultId}` | Update vault |
| DELETE | `/api/v1/vaults/{vaultId}` | Delete vault |
| GET | `/api/v1/vaults/{vaultId}/items` | List items |
| POST | `/api/v1/vaults/{vaultId}/items` | Create item |
| GET | `/api/v1/vaults/{vaultId}/items/{itemId}` | Get item |
| PUT | `/api/v1/vaults/{vaultId}/items/{itemId}` | Update item |
| DELETE | `/api/v1/vaults/{vaultId}/items/{itemId}` | Delete item |
| GET | `/api/v1/vaults/{vaultId}/members` | List members |
| POST | `/api/v1/vaults/{vaultId}/members` | Add member |
| DELETE | `/api/v1/vaults/{vaultId}/members/{userId}` | Remove member |
| POST | `/api/v1/vaults/{vaultId}/sync` | Sync vault |
| GET | `/api/v1/vaults/{vaultId}/history` | Item history |

### 4. Host Service (`/api/v1/hosts`) — 14 endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/hosts` | List hosts |
| POST | `/api/v1/hosts` | Create host |
| GET | `/api/v1/hosts/{hostId}` | Get host |
| PUT | `/api/v1/hosts/{hostId}` | Update host |
| DELETE | `/api/v1/hosts/{hostId}` | Delete host |
| POST | `/api/v1/hosts/{hostId}/connect` | Initiate connection |
| POST | `/api/v1/hosts/{hostId}/disconnect` | Disconnect |
| GET | `/api/v1/hosts/{hostId}/connections` | Connection history |
| GET | `/api/v1/hosts/{hostId}/health` | Health check |
| POST | `/api/v1/hosts/import` | Bulk import |
| GET | `/api/v1/host-groups` | List groups |
| POST | `/api/v1/host-groups` | Create group |
| PUT | `/api/v1/host-groups/{groupId}` | Update group |
| DELETE | `/api/v1/host-groups/{groupId}` | Delete group |

### 5. SSH Proxy & Terminal (`/api/v1/sessions`, `/api/v1/terminal`) — 16 endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/sessions` | List sessions |
| POST | `/api/v1/sessions` | Create session |
| GET | `/api/v1/sessions/{sessionId}` | Get session |
| DELETE | `/api/v1/sessions/{sessionId}` | Terminate session |
| POST | `/api/v1/sessions/{sessionId}/resize` | Resize terminal |
| POST | `/api/v1/sessions/{sessionId}/command` | Execute command |
| GET | `/api/v1/sessions/{sessionId}/recording` | Get recording |
| POST | `/api/v1/sessions/{sessionId}/collaborate` | Start collaboration |
| GET | `/api/v1/terminal/stream` | WebSocket terminal I/O |
| POST | `/api/v1/terminal/shell` | Request shell |
| GET | `/api/v1/terminal/themes` | List themes |
| POST | `/api/v1/terminal/themes/{themeId}` | Apply theme |
| GET | `/api/v1/terminal/fonts` | List fonts |
| POST | `/api/v1/terminal/fonts/{fontId}` | Set font |
| POST | `/api/v1/terminal/clipboard` | Clipboard sync |
| GET | `/api/v1/terminal/scrollback` | Get scrollback |

### 6. SFTP Service (`/api/v1/sftp`) — 12 endpoints

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/sftp/sessions` | Create SFTP session |
| DELETE | `/api/v1/sftp/sessions/{sessionId}` | Close session |
| GET | `/api/v1/sftp/sessions/{sessionId}/files` | List files |
| GET | `/api/v1/sftp/sessions/{sessionId}/files/{path}` | Get file info |
| POST | `/api/v1/sftp/sessions/{sessionId}/files/{path}` | Upload file |
| GET | `/api/v1/sftp/sessions/{sessionId}/files/{path}/download` | Download file |
| DELETE | `/api/v1/sftp/sessions/{sessionId}/files/{path}` | Delete file |
| POST | `/api/v1/sftp/sessions/{sessionId}/mkdir` | Create directory |
| POST | `/api/v1/sftp/sessions/{sessionId}/rename` | Rename |
| GET | `/api/v1/sftp/sessions/{sessionId}/transfers` | Transfer queue |
| POST | `/api/v1/sftp/sessions/{sessionId}/sync` | Directory sync |
| POST | `/api/v1/sftp/sessions/{sessionId}/checksum` | Verify checksum |

### 7. Port Forwarding (`/api/v1/port-forwards`) — 8 endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/port-forwards` | List rules |
| POST | `/api/v1/port-forwards` | Create rule |
| GET | `/api/v1/port-forwards/{ruleId}` | Get rule |
| PUT | `/api/v1/port-forwards/{ruleId}` | Update rule |
| DELETE | `/api/v1/port-forwards/{ruleId}` | Delete rule |
| POST | `/api/v1/port-forwards/{ruleId}/start` | Start tunnel |
| POST | `/api/v1/port-forwards/{ruleId}/stop` | Stop tunnel |
| GET | `/api/v1/port-forwards/{ruleId}/metrics` | Tunnel metrics |

### 8. Snippet Service (`/api/v1/snippets`) — 10 endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/snippets` | List snippets |
| POST | `/api/v1/snippets` | Create snippet |
| GET | `/api/v1/snippets/{snippetId}` | Get snippet |
| PUT | `/api/v1/snippets/{snippetId}` | Update snippet |
| DELETE | `/api/v1/snippets/{snippetId}` | Delete snippet |
| POST | `/api/v1/snippets/{snippetId}/execute` | Execute snippet |
| GET | `/api/v1/snippets/{snippetId}/history` | Execution history |
| GET | `/api/v1/snippet-categories` | List categories |
| POST | `/api/v1/snippet-categories` | Create category |
| GET | `/api/v1/snippets/search` | Full-text search |

### 9. Workspace Service (`/api/v1/workspaces`) — 10 endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/workspaces` | List workspaces |
| POST | `/api/v1/workspaces` | Create workspace |
| GET | `/api/v1/workspaces/{workspaceId}` | Get workspace |
| PUT | `/api/v1/workspaces/{workspaceId}` | Update workspace |
| DELETE | `/api/v1/workspaces/{workspaceId}` | Delete workspace |
| POST | `/api/v1/workspaces/{workspaceId}/restore` | Restore snapshot |
| GET | `/api/v1/workspace-templates` | List templates |
| POST | `/api/v1/workspace-templates` | Create template |
| GET | `/api/v1/workspaces/{workspaceId}/sessions` | Sessions in workspace |
| POST | `/api/v1/workspaces/{workspaceId}/share` | Share workspace |

### 10. Organization & Team (`/api/v1/orgs`) — 16 endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/orgs/me` | Get current org |
| GET | `/api/v1/orgs/me/members` | List members |
| POST | `/api/v1/orgs/me/invitations` | Invite member |
| GET | `/api/v1/orgs/me/teams` | List teams |
| POST | `/api/v1/orgs/me/teams` | Create team |
| GET | `/api/v1/orgs/me/teams/{teamId}` | Get team |
| PUT | `/api/v1/orgs/me/teams/{teamId}` | Update team |
| DELETE | `/api/v1/orgs/me/teams/{teamId}` | Delete team |
| POST | `/api/v1/orgs/me/teams/{teamId}/members` | Add team member |
| DELETE | `/api/v1/orgs/me/teams/{teamId}/members/{userId}` | Remove team member |
| GET | `/api/v1/orgs/me/roles` | List custom roles |
| POST | `/api/v1/orgs/me/roles` | Create role |
| PUT | `/api/v1/orgs/me/roles/{roleId}` | Update role |
| DELETE | `/api/v1/orgs/me/roles/{roleId}` | Delete role |
| GET | `/api/v1/orgs/me/settings` | Org settings |
| PUT | `/api/v1/orgs/me/settings` | Update settings |

### 11. Audit Service (`/api/v1/audit`) — 5 endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/audit/events` | Query audit events |
| GET | `/api/v1/audit/events/{eventId}` | Get specific event |
| GET | `/api/v1/audit/export` | Export audit log |
| GET | `/api/v1/audit/compliance` | Compliance dashboard |
| GET | `/api/v1/audit/retention` | Retention policy status |

### 12. AI Service (`/api/v1/ai`) — 8 endpoints

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/ai/complete` | Command completion |
| POST | `/api/v1/ai/explain` | Explain command |
| POST | `/api/v1/ai/anomaly` | Anomaly detection |
| POST | `/api/v1/ai/runbook` | Generate runbook |
| POST | `/api/v1/ai/incident` | Incident assist |
| GET | `/api/v1/ai/models` | List available models |
| GET | `/api/v1/ai/feedback` | Get feedback history |
| POST | `/api/v1/ai/feedback` | Submit feedback |

### 13. Additional Services (gRPC-only or internal)

| Service | gRPC Only | Key Internal RPCs |
|---------|-----------|-------------------|
| Keychain | Yes | `WrapKey`, `UnwrapKey`, `RotateKey`, `GetHardwareKey` |
| PKI | Partial | `SignUserCertificate`, `SignHostCertificate`, `GetCRL`, `RotateCA` |
| Notification | No | `SendNotification`, `SendDigest`, `SendWebhook` |
| Analytics | No | `GetMetrics`, `GetDashboardData`, `ExportSLO` |
| Recording | No | `GetRecording`, `SearchTranscript`, `ExportMP4` |
| Collaboration | No | `JoinSession`, `LeaveSession`, `BroadcastEvent`, `SyncBuffer` |
| Billing | No | `GetSubscription`, `UpdatePaymentMethod`, `GetInvoice` |
| Config | No | `GetFeatureFlag`, `SetFeatureFlag`, `WatchConfig` |
| Health | No | `GetHealth`, `GetSLOStatus` |
| Container Bridge | No | `ExecPod`, `StreamLogs`, `RegisterCluster` |
| HelixTrack Bridge | No | `LinkSession`, `SyncSprint`, `GetIssue` |

---

## Endpoint Count Reconciliation

| Source | Claimed Count | Notes |
|--------|--------------|-------|
| README.md | 221 | Canonical (includes all services + WebSocket + internal) |
| Doc 07 intro | 126 | Counts only explicitly documented REST endpoints in §2-§14 |
| Actual inventory above | ~221 | 14+6+12+14+16+12+8+10+10+16+5+8 = 131 REST + ~90 gRPC/WebSocket/internal |

**Resolution:** 221 is the canonical total per README and SERVICE_REGISTRY. The 126 figure in doc 07 is a partial count of the major service REST surfaces only.

---

## Artifact Files

| File | Description |
|------|-------------|
| `openapi.yaml` | Full OpenAPI 3.1 specification with all 221 endpoints and components.schemas |
| `proto/` | gRPC .proto files for all 25 services |

---

## Cross-References

- [03 — Service Catalog](../03-service-catalog/) — Canonical 25 services with module paths and ports
- [05 — Database Schema](../05-database-schema/) — SQL schemas backing these endpoints
- [09 — Security — Zero Trust](../09-security-zero-trust/) — Auth, RBAC, mTLS details
- [16 — References](../16-references/) — Canonical facts (CD-7 JWT, CD-8 RBAC)

---

*Section 04 — API Specification*  
*Consolidated from: 07_api_and_database.md §1-§14, CANONICAL_FACTS.md (CD-7, CD-8)*
