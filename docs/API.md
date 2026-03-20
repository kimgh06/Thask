# API Reference

Base URL: `http://localhost:7244`

All endpoints return JSON: `{ "data": T }` or `{ "error": "string" }`.

Authentication: session cookie (`thask_session`) or API key (`Authorization: Bearer <key>`). All endpoints except login/register require authentication.

Backend: Go (Echo v4). Request validation via struct tags.

---

## Authentication

### POST /api/auth/register

Create a new account.

```json
// Request
{ "email": "user@example.com", "password": "min8chars", "displayName": "User" }

// Response 201
{ "data": { "id": "uuid", "email": "user@example.com", "displayName": "User" } }
```

### POST /api/auth/login

Logs in and sets session cookie. Performs **session rotation** — deletes all previous sessions for the user.

```json
// Request
{ "email": "user@example.com", "password": "..." }

// Response 200
{ "data": { "id": "uuid", "email": "user@example.com", "displayName": "User" } }
```

### GET /api/auth/me

Returns the current authenticated user.

```json
// Response 200
{ "data": { "id": "uuid", "email": "user@example.com", "displayName": "User" } }
```

### POST /api/auth/logout

Invalidates the session and clears the cookie.

```json
// Response 200
{ "data": { "success": true } }
```

---

## Authorization & Roles

Teams use role-based access control with four levels:

| Role | Level | Capabilities |
|---|---|---|
| `owner` | 4 | Full control: delete team, transfer ownership |
| `admin` | 3 | Manage members, invite, update team settings |
| `member` | 2 | Create projects, edit nodes/edges, batch operations |
| `viewer` | 1 | Read-only access to projects and graphs |

Middleware chain: `Auth` → `TeamAccess` (resolves slug, verifies membership, sets role) → `RequireRole(minRole)`.

---

## API Keys

### POST /api/auth/api-keys

Create an API key. Maximum 10 keys per user.

```json
// Request
{ "name": "CLI Token", "expiresIn": 90 }

// Response 201
{
  "data": {
    "id": "uuid",
    "name": "CLI Token",
    "keyPrefix": "thsk_ab12345",
    "key": "thsk_ab1234567890...",
    "expiresAt": "2025-06-01T00:00:00Z",
    "createdAt": "2025-03-01T00:00:00Z"
  }
}
```

> **Note:** The `key` field is only returned on creation. Store it securely.

| Field | Validation |
|---|---|
| `name` | Required, 1-100 chars |
| `expiresIn` | Optional, 1-365 days. Omit for no expiration. |

### GET /api/auth/api-keys

List your API keys (key hash is never returned).

```json
// Response 200
{ "data": [{ "id": "uuid", "name": "CLI Token", "keyPrefix": "thsk_ab12345", "lastUsedAt": null, "expiresAt": "...", "createdAt": "..." }] }
```

### DELETE /api/auth/api-keys/:keyId

Delete an API key.

```json
// Response 200
{ "data": { "success": true } }
```

---

## Teams

### GET /api/teams

List all teams the user is a member of, with their projects.

```json
// Response 200
{ "data": [{ "id": "uuid", "name": "Team", "slug": "team", "projects": [...] }] }
```

### POST /api/teams

Create a new team. The creator becomes `owner`.

```json
// Request
{ "name": "My Team", "slug": "my-team" }

// Response 201
{ "data": { "id": "uuid", "name": "My Team", "slug": "my-team", ... } }
```

### GET /api/teams/:slug

Get a team by slug.

```json
// Response 200
{ "data": { "id": "uuid", "name": "My Team", "slug": "my-team", ... } }
```

### PATCH /api/teams/:slug

Update team name. Requires `admin` role or above.

```json
// Request
{ "name": "New Team Name" }

// Response 200
{ "data": { "id": "uuid", "name": "New Team Name", "slug": "my-team", ... } }
```

### DELETE /api/teams/:slug

Delete a team. Requires `owner` role.

```json
// Response 200
{ "data": { "success": true } }
```

### GET /api/teams/:slug/members

List team members. Requires team membership (authorization enforced).

```json
// Response 200
{ "data": [{ "id": "uuid", "userId": "uuid", "role": "owner", "email": "...", "displayName": "..." }] }
```

### POST /api/teams/:slug/members

Invite a user by email. Requires `admin` role or above. Admins cannot invite as admin or owner.

```json
// Request
{ "email": "invite@example.com", "role": "member" }

// Response 201
{ "data": { "success": true } }
```

| Field | Validation |
|---|---|
| `email` | Required, valid email |
| `role` | Optional: `admin`, `member`, `viewer`. Default: `member` |

### PATCH /api/teams/:slug/members/:userId

Change a member's role. Requires `admin` role or above. Admins cannot manage other admins.

```json
// Request
{ "role": "admin" }

// Response 200
{ "data": { "success": true } }
```

### DELETE /api/teams/:slug/members/:userId

Remove a member. Requires `admin` role or above. Cannot remove owner.

```json
// Response 200
{ "data": { "success": true } }
```

### POST /api/teams/:slug/leave

Leave a team. Any member can leave. Sole owner must transfer ownership first.

```json
// Response 200
{ "data": { "success": true } }
```

### POST /api/teams/:slug/transfer

Transfer team ownership. Requires `owner` role. Target must be a team member.

```json
// Request
{ "userId": "uuid" }

// Response 200
{ "data": { "success": true } }
```

### GET /api/teams/:slug/projects

List projects in a team.

```json
// Response 200
{ "data": [{ "id": "uuid", "name": "Project", ... }] }
```

### POST /api/teams/:slug/projects

Create a project in a team.

```json
// Request
{ "name": "New Project", "description": "optional" }
```

---

## Projects

### GET /api/projects/:projectId

Get project details. Requires team membership (any role).

```json
// Response 200
{ "data": { "id": "uuid", "teamId": "uuid", "name": "Project", "description": "...", ... } }
```

### PATCH /api/projects/:projectId

Update project. Requires `member` role.

```json
// Request (all fields optional)
{ "name": "Updated Name", "description": "Updated desc" }
```

### DELETE /api/projects/:projectId

Delete project. Requires `member` role.

```json
// Response 200
{ "data": { "success": true } }
```

---

## Nodes

All node endpoints require project access (verified via `ProjectAccess` middleware).

### GET /api/projects/:projectId/nodes

Query params: `?type=TASK&status=PASS` (optional filters)

```json
// Response 200
{ "data": [{ "id": "uuid", "type": "TASK", "title": "...", "status": "IN_PROGRESS", ... }] }
```

### POST /api/projects/:projectId/nodes

```json
// Request
{
  "type": "TASK",
  "title": "New Node",
  "description": "optional",
  "status": "IN_PROGRESS",
  "positionX": 100,
  "positionY": 200
}

// Response 201
{ "data": { "id": "uuid", ... } }
```

### GET /api/projects/:projectId/nodes/:nodeId

Returns node with connected edges, connected node IDs, and history.

```json
// Response 200
{
  "data": {
    "id": "uuid", "type": "TASK", "title": "...",
    "connectedEdges": [...],
    "connectedNodeIds": ["uuid", ...],
    "history": [{ "id": "uuid", "action": "updated", "fieldName": "title", ... }]
  }
}
```

### PATCH /api/projects/:projectId/nodes/:nodeId

Updates a node. Records history for each changed field. Triggers **waterfall status propagation** when status changes.

```json
// Request (all fields optional)
{
  "title": "Updated",
  "status": "PASS",
  "type": "BUG",
  "description": "...",
  "assigneeId": "uuid",
  "tags": ["tag1", "tag2"],
  "parentId": "group-uuid | null"
}
```

### DELETE /api/projects/:projectId/nodes/:nodeId

Deletes the node. If it's a GROUP, children are unparented (preserved).

```json
// Response 200
{ "data": { "success": true } }
```

### PATCH /api/projects/:projectId/nodes/positions

Batch update node positions (after drag or layout).

```json
// Request
{
  "positions": [
    { "id": "uuid", "x": 100, "y": 200, "width": 300, "height": 200 },
    { "id": "uuid", "x": 400, "y": 100 }
  ]
}
```

### POST /api/projects/:projectId/nodes/batch-delete

Delete multiple nodes. Requires `member` role.

```json
// Request
{ "ids": ["uuid1", "uuid2"] }

// Response 200
{ "data": { "success": true } }
```

### PATCH /api/projects/:projectId/nodes/batch-status

Batch update status for multiple nodes. Requires `member` role.

```json
// Request
{ "ids": ["uuid1", "uuid2"], "status": "PASS" }

// Response 200
{ "data": [...] }
```

---

## Edges

### GET /api/projects/:projectId/edges

```json
// Response 200
{ "data": [{ "id": "uuid", "sourceId": "uuid", "targetId": "uuid", "edgeType": "depends_on", "label": "" }] }
```

### POST /api/projects/:projectId/edges

```json
// Request
{ "sourceId": "uuid", "targetId": "uuid", "edgeType": "depends_on", "label": "optional" }
```

Constraints: no self-loops (validated server-side).

### PATCH /api/projects/:projectId/edges/:edgeId

```json
// Request
{ "edgeType": "blocks", "label": "updated label" }
```

### DELETE /api/projects/:projectId/edges/:edgeId

```json
// Response 200
{ "data": { "success": true } }
```

---

## Impact Analysis

### GET /api/projects/:projectId/impact

Query params: `?since=2025-01-01T00:00:00Z&depth=2`

Finds changed nodes and their downstream dependencies via bidirectional BFS.

```json
// Response 200
{
  "data": {
    "changedNodes": [...],
    "impactedNodes": [...],
    "failNodes": [...],
    "impactEdges": [...]
  }
}
```

| Param | Default | Description |
|---|---|---|
| `since` | 7 days ago | ISO date — nodes updated after this time |
| `depth` | 2 | BFS depth for downstream search |

---

## Summary

### GET /api/projects/summary

Returns project and team counts for the authenticated user.

```json
// Response 200
{ "data": { "teamCount": 3, "projectCount": 7 } }
```

---

## Middleware

| Middleware | Scope | Description |
|---|---|---|
| CORS | Global | Allows frontend origin with credentials |
| Rate Limiter | Global | 20 requests/second per client |
| Logger | Global | Structured logging via slog |
| Auth | Protected routes | Cookie or API key → user context |
| TeamAccess | `/api/teams/:teamSlug/*` | Resolves slug, verifies membership, sets team role |
| ProjectAccess | `/api/projects/:projectId/*` | Verifies team membership for the project |
| RequireRole | Specific routes | Enforces minimum role (owner/admin/member) |

---

## Error Responses

```json
// 400 Bad Request
{ "error": "Title is required" }

// 401 Unauthorized
{ "error": "Authentication required" }

// 404 Not Found
{ "error": "Node not found" }

// 500 Internal Server Error
{ "error": "Internal server error" }
```

---

## Validation

All inputs are validated with Go struct tags (`validate`):

| Endpoint | Validated Fields |
|---|---|
| Register | email (required, email), password (min 8), displayName (required) |
| Login | email (required), password (required) |
| Create Team | name (required), slug (required, alphanum+hyphen) |
| Invite Member | email (required, email), role (oneof: owner/admin/member/viewer) |
| Create Project | name (required) |
| Create Node | type (required), title (required) |
| Update Node | all fields optional, validated when present |
| Batch Positions | positions array with id, x, y required |
| Create Edge | sourceId (required), targetId (required) |
| Update Edge | edgeType and label optional |
