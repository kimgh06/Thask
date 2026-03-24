# Thask API v1 Quickstart

Build integrations on top of Thask's graph-based project management API. This guide covers authentication, core concepts, and practical examples to get you started in minutes.

## Quick Start

### 1. Get Your API Key

Create an API key from the Thask web UI, or use the API directly:

```bash
curl -X POST http://localhost:7244/api/auth/api-keys \
  -H "Authorization: Bearer <existing-key>" \
  -H "Content-Type: application/json" \
  -d '{"name": "My Integration", "expiresIn": 90}'
```

Response (save the `key` field — it's only returned once):

```json
{
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "name": "My Integration",
    "keyPrefix": "thsk_abc123...",
    "key": "thsk_abc123...",
    "expiresAt": "2026-06-24T00:00:00Z",
    "createdAt": "2026-03-24T00:00:00Z"
  }
}
```

### 2. Make Your First API Call

Test authentication by fetching the API version:

```bash
curl http://localhost:7244/api/v1 \
  -H "Authorization: Bearer thsk_your_api_key_here"
```

Response:

```json
{
  "version": "1.0.0",
  "status": "ok"
}
```

### 3. List Your Teams

```bash
curl http://localhost:7244/api/v1/teams \
  -H "Authorization: Bearer thsk_your_api_key_here"
```

Response:

```json
{
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "name": "Product Team",
      "slug": "product-team",
      "createdBy": "660e8400-e29b-41d4-a716-446655440001",
      "createdAt": "2026-03-01T10:00:00Z"
    }
  ]
}
```

## Authentication

All API v1 endpoints require Bearer token authentication using API keys.

**Base URL:** `http://localhost:7244/api/v1`

**Header format:**

```
Authorization: Bearer thsk_your_key_here
```

API keys:
- Start with `thsk_` prefix
- Created via web UI or `POST /api/auth/api-keys`
- Maximum 10 keys per user
- Optional expiration (1-365 days)

Response format includes `X-API-Version: v1` header on all responses.

## Core Concepts

### Nodes

Nodes represent items in your project graph. Each node has:

- **Type:** FLOW, BRANCH, TASK, BUG, API, UI, GROUP
- **Status:** PASS, FAIL, IN_PROGRESS, BLOCKED
- **Positioning:** positionX, positionY (canvas coordinates)
- **Hierarchy:** parentId (for nesting within groups)
- **Metadata:** title, description, tags, assigneeId

### Edges

Edges define relationships between nodes:

- **depends_on** — Node A depends on Node B completing
- **blocks** — Node A blocks Node B from starting
- **related** — Informational relationship
- **parent_child** — Hierarchical (for group membership)
- **triggers** — Node A triggers Node B

### Graphs

A complete graph contains all nodes and edges for a project. Graphs are:
- Not paginated (full snapshot on fetch)
- Queryable via single endpoint
- Exportable to JSON
- Importable with replace/merge strategies

### Projects

Projects are the top-level container for graphs. Each project:
- Belongs to exactly one team
- Contains nodes and edges
- Has sharing settings (viewers/editors)
- Tracks creation and update timestamps

## Common Operations

### Teams

**List teams:**

```bash
curl http://localhost:7244/api/v1/teams \
  -H "Authorization: Bearer thsk_your_key_here"
```

**Get team by slug:**

```bash
curl http://localhost:7244/api/v1/teams/product-team \
  -H "Authorization: Bearer thsk_your_key_here"
```

**List team members:**

```bash
curl http://localhost:7244/api/v1/teams/product-team/members \
  -H "Authorization: Bearer thsk_your_key_here"
```

**List team projects:**

```bash
curl http://localhost:7244/api/v1/teams/product-team/projects \
  -H "Authorization: Bearer thsk_your_key_here"
```

### Projects

**Get project:**

```bash
curl http://localhost:7244/api/v1/projects/550e8400-e29b-41d4-a716-446655440000 \
  -H "Authorization: Bearer thsk_your_key_here"
```

**Update project:**

```bash
curl -X PATCH http://localhost:7244/api/v1/projects/550e8400-e29b-41d4-a716-446655440000 \
  -H "Authorization: Bearer thsk_your_key_here" \
  -H "Content-Type: application/json" \
  -d '{"name": "Updated Project Name"}'
```

### Nodes

**List all nodes in project:**

```bash
curl http://localhost:7244/api/v1/projects/550e8400-e29b-41d4-a716-446655440000/nodes \
  -H "Authorization: Bearer thsk_your_key_here"
```

**Get single node:**

```bash
curl http://localhost:7244/api/v1/projects/550e8400-e29b-41d4-a716-446655440000/nodes/660e8400-e29b-41d4-a716-446655440001 \
  -H "Authorization: Bearer thsk_your_key_here"
```

Response:

```json
{
  "data": {
    "id": "660e8400-e29b-41d4-a716-446655440001",
    "projectId": "550e8400-e29b-41d4-a716-446655440000",
    "type": "TASK",
    "title": "Implement authentication",
    "description": "Add OAuth provider support",
    "status": "IN_PROGRESS",
    "assigneeId": "770e8400-e29b-41d4-a716-446655440002",
    "tags": ["backend", "security"],
    "parentId": null,
    "positionX": 100,
    "positionY": 200,
    "width": 200,
    "height": 100,
    "createdAt": "2026-03-20T14:00:00Z",
    "updatedAt": "2026-03-24T09:30:00Z"
  }
}
```

**Create node:**

```bash
curl -X POST http://localhost:7244/api/v1/projects/550e8400-e29b-41d4-a716-446655440000/nodes \
  -H "Authorization: Bearer thsk_your_key_here" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "TASK",
    "title": "Add caching layer",
    "description": "Redis integration for performance",
    "status": "BLOCKED",
    "positionX": 300,
    "positionY": 150,
    "tags": ["backend"]
  }'
```

**Update node:**

```bash
curl -X PATCH http://localhost:7244/api/v1/projects/550e8400-e29b-41d4-a716-446655440000/nodes/660e8400-e29b-41d4-a716-446655440001 \
  -H "Authorization: Bearer thsk_your_key_here" \
  -H "Content-Type: application/json" \
  -d '{
    "status": "PASS",
    "title": "Implement authentication (Complete)"
  }'
```

**Delete node:**

```bash
curl -X DELETE http://localhost:7244/api/v1/projects/550e8400-e29b-41d4-a716-446655440000/nodes/660e8400-e29b-41d4-a716-446655440001 \
  -H "Authorization: Bearer thsk_your_key_here"
```

### Edges

**List all edges in project:**

```bash
curl http://localhost:7244/api/v1/projects/550e8400-e29b-41d4-a716-446655440000/edges \
  -H "Authorization: Bearer thsk_your_key_here"
```

**Create edge:**

```bash
curl -X POST http://localhost:7244/api/v1/projects/550e8400-e29b-41d4-a716-446655440000/edges \
  -H "Authorization: Bearer thsk_your_key_here" \
  -H "Content-Type: application/json" \
  -d '{
    "sourceId": "660e8400-e29b-41d4-a716-446655440001",
    "targetId": "770e8400-e29b-41d4-a716-446655440003",
    "edgeType": "depends_on",
    "label": "Backend ready before frontend"
  }'
```

**Update edge:**

```bash
curl -X PATCH http://localhost:7244/api/v1/projects/550e8400-e29b-41d4-a716-446655440000/edges/880e8400-e29b-41d4-a716-446655440004 \
  -H "Authorization: Bearer thsk_your_key_here" \
  -H "Content-Type: application/json" \
  -d '{"edgeType": "blocks"}'
```

**Delete edge:**

```bash
curl -X DELETE http://localhost:7244/api/v1/projects/550e8400-e29b-41d4-a716-446655440000/edges/880e8400-e29b-41d4-a716-446655440004 \
  -H "Authorization: Bearer thsk_your_key_here"
```

### Graph Operations

**Get full graph:**

```bash
curl http://localhost:7244/api/v1/projects/550e8400-e29b-41d4-a716-446655440000/graph \
  -H "Authorization: Bearer thsk_your_key_here"
```

Response includes all nodes and edges:

```json
{
  "data": {
    "nodes": [
      {
        "id": "660e8400-e29b-41d4-a716-446655440001",
        "type": "TASK",
        "title": "Backend API",
        ...
      }
    ],
    "edges": [
      {
        "id": "880e8400-e29b-41d4-a716-446655440004",
        "sourceId": "660e8400-e29b-41d4-a716-446655440001",
        "targetId": "770e8400-e29b-41d4-a716-446655440003",
        "edgeType": "depends_on"
      }
    ]
  }
}
```

**Import graph (replace):**

```bash
curl -X POST http://localhost:7244/api/v1/projects/550e8400-e29b-41d4-a716-446655440000/graph/import \
  -H "Authorization: Bearer thsk_your_key_here" \
  -H "Content-Type: application/json" \
  -d '{
    "strategy": "replace",
    "nodes": [...],
    "edges": [...]
  }'
```

**Import graph (merge):**

```bash
curl -X POST http://localhost:7244/api/v1/projects/550e8400-e29b-41d4-a716-446655440000/graph/import \
  -H "Authorization: Bearer thsk_your_key_here" \
  -H "Content-Type: application/json" \
  -d '{
    "strategy": "merge",
    "nodes": [...],
    "edges": [...]
  }'
```

**Apply auto-layout:**

```bash
curl -X POST http://localhost:7244/api/v1/projects/550e8400-e29b-41d4-a716-446655440000/graph/layout \
  -H "Authorization: Bearer thsk_your_key_here" \
  -H "Content-Type: application/json" \
  -d '{"algorithm": "fcose"}'
```

### Impact Analysis

**Get impact (changed nodes and downstream dependencies):**

```bash
curl http://localhost:7244/api/v1/projects/550e8400-e29b-41d4-a716-446655440000/impact \
  -H "Authorization: Bearer thsk_your_key_here"
```

Response:

```json
{
  "data": {
    "changedNodeIds": ["660e8400-e29b-41d4-a716-446655440001"],
    "affectedNodeIds": [
      "660e8400-e29b-41d4-a716-446655440001",
      "770e8400-e29b-41d4-a716-446655440003",
      "880e8400-e29b-41d4-a716-446655440005"
    ]
  }
}
```

### Sharing

**Get sharing settings:**

```bash
curl http://localhost:7244/api/v1/projects/550e8400-e29b-41d4-a716-446655440000/sharing \
  -H "Authorization: Bearer thsk_your_key_here"
```

**Update sharing (admin+ only):**

```bash
curl -X PUT http://localhost:7244/api/v1/projects/550e8400-e29b-41d4-a716-446655440000/sharing \
  -H "Authorization: Bearer thsk_your_key_here" \
  -H "Content-Type: application/json" \
  -d '{
    "isPublic": true,
    "defaultRole": "viewer"
  }'
```

## Idempotency

For mutation operations (POST, PATCH, DELETE), use the `Idempotency-Key` header to ensure safe retries:

```bash
curl -X POST http://localhost:7244/api/v1/projects/550e8400-e29b-41d4-a716-446655440000/nodes \
  -H "Authorization: Bearer thsk_your_key_here" \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: create-task-2026-03-24-unique-id" \
  -d '{
    "type": "TASK",
    "title": "Add caching layer"
  }'
```

Key rules:

- **Max length:** 256 characters
- **TTL:** 24 hours
- **Replay response:** Includes `X-Idempotency-Replayed: true` header

Idempotency ensures:
- Duplicate requests return the same response
- Safe to retry without side effects
- Network timeouts don't cause double-creation

## Error Handling

All non-2xx responses follow a consistent error format:

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Failed to validate request",
    "details": [
      {
        "field": "title",
        "reason": "title is required"
      },
      {
        "field": "type",
        "reason": "type must be one of: FLOW, BRANCH, TASK, BUG, API, UI, GROUP"
      }
    ]
  }
}
```

### Error Codes

| Code | HTTP | Meaning |
|---|---|---|
| `AUTHENTICATION_REQUIRED` | 401 | Missing or invalid API key |
| `FORBIDDEN` | 403 | Authenticated but lacking permission |
| `NOT_FOUND` | 404 | Resource does not exist |
| `VALIDATION_ERROR` | 400 | Request validation failed (details provided) |
| `CONFLICT` | 409 | Resource conflict (e.g., duplicate edge) |
| `RATE_LIMITED` | 429 | Rate limit exceeded (wait before retry) |
| `BODY_TOO_LARGE` | 413 | Request body exceeds 1MB |
| `INTERNAL_ERROR` | 500 | Server error (contact support) |

### Example Error Responses

**Missing authentication:**

```bash
curl http://localhost:7244/api/v1/teams
```

Response (401):

```json
{
  "error": {
    "code": "AUTHENTICATION_REQUIRED",
    "message": "API key required"
  }
}
```

**Invalid node type:**

```bash
curl -X POST http://localhost:7244/api/v1/projects/550e8400-e29b-41d4-a716-446655440000/nodes \
  -H "Authorization: Bearer thsk_your_key_here" \
  -H "Content-Type: application/json" \
  -d '{"type": "INVALID", "title": "Task"}'
```

Response (400):

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Failed to validate request",
    "details": [
      {
        "field": "type",
        "reason": "type must be one of: FLOW, BRANCH, TASK, BUG, API, UI, GROUP"
      }
    ]
  }
}
```

**Insufficient permissions:**

```bash
curl -X PUT http://localhost:7244/api/v1/projects/550e8400-e29b-41d4-a716-446655440000/sharing \
  -H "Authorization: Bearer thsk_your_key_here" \
  -H "Content-Type: application/json" \
  -d '{"isPublic": true}'
```

Response (403):

```json
{
  "error": {
    "code": "FORBIDDEN",
    "message": "Only admins can change sharing settings"
  }
}
```

## Interactive API Documentation

Explore the complete API specification and test endpoints in your browser:

**Scalar UI (recommended):**

```
http://localhost:7244/api/v1/docs
```

Navigate to this URL in your browser to:
- Browse all endpoints and schemas
- See detailed parameter and response documentation
- Test requests directly with your API key
- View example request/response payloads

**OpenAPI 3.1 Specification:**

```
http://localhost:7244/api/v1/openapi.yaml
```

Download or reference the raw OpenAPI spec for:
- Integration with tools like Postman, Insomnia, or OpenAPI generators
- Automated client generation
- Documentation automation

## Rate Limits & Constraints

- **Rate limit:** 100 requests per minute per API key
- **Body size limit:** 1MB max request body
- **Graph size:** No hard limit; import/export transactions ensure consistency
- **Key storage:** Store keys securely; they cannot be retrieved after creation

## What's NOT on v1

The v1 API is focused on graph operations. These features are web-UI only:

- User authentication (register/login/logout)
- Team management writes (create/delete teams)
- Batch node/edge operations
- Server-sent events (SSE) for real-time updates
- Shared project routes (use direct endpoints instead)
- Embed routes

For team operations, contact an admin or use the web UI.

## Next Steps

1. **Explore the Scalar UI** at `/api/v1/docs` to discover all endpoints
2. **Read the full API reference** in [docs/API.md](../../docs/API.md)
3. **Check the OpenAPI spec** at `/api/v1/openapi.yaml` for integration tools
4. **Join the community** for support and feature requests
