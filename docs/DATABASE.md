# Database Schema

PostgreSQL 17 with pgx/v5 (raw SQL queries). Migrations in `backend/migrations/001_initial.sql`.

---

## ER Diagram

```
Users ◄──── Sessions
  │
  ├──── ApiKeys
  │
  ├──── TeamMembers ────► Teams
  │                         │
  │                         └──── Projects ◄────┐
  │                                 │            │
  │                          ┌──────┴──────┐     │
  │                          │             │     │
  │                        Nodes ◄──── Edges   ProjectMembers
  │                          │
  │                     NodeHistory
  │                          │
  └──────────────────────────┘ (assigneeId, userId)
```

**Self-reference:** `nodes.parent_id → nodes.id` (GROUP containment)

---

## Enums

```sql
team_role:      owner | admin | member | viewer
node_type:      FLOW | BRANCH | TASK | BUG | API | UI | GROUP
node_status:    PASS | FAIL | IN_PROGRESS | BLOCKED
edge_type:      depends_on | blocks | related | parent_child | triggers
history_action: created | updated | deleted | status_changed
```

---

## Tables

### users

| Column | Type | Constraints |
|---|---|---|
| id | uuid | PK, default gen_random_uuid() |
| email | text | UNIQUE, NOT NULL |
| display_name | text | NOT NULL |
| password_hash | text | NOT NULL |
| created_at | timestamptz | default now() |
| updated_at | timestamptz | default now() |

### sessions

| Column | Type | Constraints |
|---|---|---|
| id | uuid | PK, default gen_random_uuid() |
| user_id | uuid | FK → users, NOT NULL |
| token | text | UNIQUE, NOT NULL |
| expires_at | timestamptz | NOT NULL |
| created_at | timestamptz | default now() |

Index: `idx_sessions_user_id (user_id)`

### teams

| Column | Type | Constraints |
|---|---|---|
| id | uuid | PK, default gen_random_uuid() |
| name | text | NOT NULL |
| slug | text | UNIQUE, NOT NULL |
| created_by | uuid | FK → users |
| created_at | timestamptz | default now() |
| updated_at | timestamptz | default now() |

### team_members

| Column | Type | Constraints |
|---|---|---|
| id | uuid | PK, default gen_random_uuid() |
| team_id | uuid | FK → teams (CASCADE), NOT NULL |
| user_id | uuid | FK → users (CASCADE), NOT NULL |
| role | team_role | default: 'member' |
| joined_at | timestamptz | default now() |

Unique: `(team_id, user_id)`

### projects

| Column | Type | Constraints |
|---|---|---|
| id | uuid | PK, default gen_random_uuid() |
| team_id | uuid | FK → teams (CASCADE), NOT NULL |
| name | text | NOT NULL |
| description | text | |
| created_by | uuid | FK → users |
| created_at | timestamptz | default now() |
| updated_at | timestamptz | default now() |

Index: `idx_projects_team_id (team_id)`

> **Added in migration 003:** `link_sharing` (TEXT, default `'off'`), `share_token` (TEXT, nullable), `share_token_hash` (TEXT, UNIQUE, nullable)

### nodes

| Column | Type | Constraints |
|---|---|---|
| id | uuid | PK, default gen_random_uuid() |
| project_id | uuid | FK → projects (CASCADE), NOT NULL |
| type | node_type | NOT NULL |
| title | text | NOT NULL |
| description | text | |
| status | node_status | default: 'IN_PROGRESS' |
| assignee_id | uuid | FK → users (SET NULL) |
| tags | text[] | default: '{}' |
| metadata | jsonb | default: '{}' |
| parent_id | uuid | nullable (GROUP containment) |
| position_x | double precision | default: 0 |
| position_y | double precision | default: 0 |
| width | double precision | |
| height | double precision | |
| created_at | timestamptz | default now() |
| updated_at | timestamptz | default now() |

Indexes:
- `idx_nodes_project_id (project_id)`
- `idx_nodes_updated_at (project_id, updated_at)`
- `idx_nodes_status (project_id, status)`
- `idx_nodes_type (project_id, type)`
- `idx_nodes_assignee (assignee_id)`
- `idx_nodes_parent_id (parent_id)`

### edges

| Column | Type | Constraints |
|---|---|---|
| id | uuid | PK, default gen_random_uuid() |
| project_id | uuid | FK → projects (CASCADE), NOT NULL |
| source_id | uuid | FK → nodes (CASCADE), NOT NULL |
| target_id | uuid | FK → nodes (CASCADE), NOT NULL |
| edge_type | edge_type | default: 'related' |
| label | text | |
| created_at | timestamptz | default now() |

Constraints:
- Unique: `(source_id, target_id, edge_type)`
- Check: `source_id != target_id` (no self-loops)

Indexes:
- `idx_edges_project_id (project_id)`
- `idx_edges_source (source_id)`
- `idx_edges_target (target_id)`

### node_history

| Column | Type | Constraints |
|---|---|---|
| id | uuid | PK, default gen_random_uuid() |
| node_id | uuid | FK → nodes (CASCADE), NOT NULL |
| project_id | uuid | FK → projects (CASCADE), NOT NULL |
| user_id | uuid | FK → users, NOT NULL |
| action | history_action | NOT NULL |
| field_name | text | |
| old_value | text | |
| new_value | text | |
| created_at | timestamptz | default now() |

Indexes:
- `idx_node_history_project_recent (project_id, created_at)`
- `idx_node_history_node_id (node_id, created_at)`

### api_keys

API keys for programmatic access (CLI, MCP, CI/CD).

| Column | Type | Constraints |
|---|---|---|
| `id` | UUID | PK, default `gen_random_uuid()` |
| `user_id` | UUID | FK → `users(id)` ON DELETE CASCADE |
| `name` | TEXT | NOT NULL |
| `key_prefix` | CHAR(12) | NOT NULL — first 12 chars for display |
| `key_hash` | TEXT | NOT NULL — SHA256 hash |
| `last_used_at` | TIMESTAMPTZ | nullable |
| `expires_at` | TIMESTAMPTZ | nullable (NULL = no expiration) |
| `created_at` | TIMESTAMPTZ | NOT NULL, default `now()` |

**Indexes:**
- `idx_api_keys_user` — `user_id`
- `idx_api_keys_hash` — `key_hash`

### project_members

Per-project access control for individual users.

| Column | Type | Constraints |
|---|---|---|
| `id` | UUID | PK, default `gen_random_uuid()` |
| `project_id` | UUID | FK → `projects(id)` ON DELETE CASCADE |
| `user_id` | UUID | FK → `users(id)` ON DELETE CASCADE |
| `role` | TEXT | CHECK (`editor`, `viewer`), DEFAULT `viewer` |
| `created_at` | TIMESTAMPTZ | NOT NULL, default `now()` |

**Constraints:** UNIQUE(`project_id`, `user_id`)

---

## Relations Summary

| Relation | Type | ON DELETE |
|---|---|---|
| sessions → users | M:1 | — |
| api_keys → users | M:1 | CASCADE |
| teams → users (created_by) | M:1 | — |
| team_members → teams | M:1 | CASCADE |
| team_members → users | M:1 | CASCADE |
| projects → teams | M:1 | CASCADE |
| projects → users (created_by) | M:1 | — |
| nodes → projects | M:1 | CASCADE |
| nodes → users (assignee_id) | M:1 | SET NULL |
| nodes → nodes (parent_id) | self-ref | — |
| edges → projects | M:1 | CASCADE |
| edges → nodes (source_id) | M:1 | CASCADE |
| edges → nodes (target_id) | M:1 | CASCADE |
| node_history → nodes | M:1 | CASCADE |
| node_history → projects | M:1 | CASCADE |
| node_history → users | M:1 | — |

---

## Migrations

Migrations are plain SQL files in `backend/migrations/`. They run automatically on server startup via `main.go`.

```bash
# Migration is applied automatically when backend starts.
# To manually inspect:
cat backend/migrations/001_initial.sql
```

---

## Provenance & Audit (Migrations 006~010)

To prevent AI agents from silently writing hallucinated content into the graph,
v0.5.9 adds a unified audit pipeline and per-API-key permissions. Six orthogonal
dimensions are recorded for every write event.

### `audit_log` (migration 006)

Single source of truth for "who/how/when/why did this happen". Replaces
`node_history` over a deprecation window.

| Column | Type | Notes |
|---|---|---|
| `id` | UUID | PK |
| `project_id` | UUID | FK → projects, CASCADE |
| `entity_type` | TEXT | `node` / `edge` / `project` / `graph` / `suggestion` |
| `entity_id` | UUID | NULL for graph-level events |
| `action` | TEXT | `created` / `updated` / `deleted` / `imported` / `verified` / `suggested` / `suggestion_decided` / `write_denied` / `status_changed` / `layout_computed` |
| `field_name` | TEXT | NULL for non-field events |
| `old_value`, `new_value` | TEXT | optional |
| `mutation_kind` | TEXT | `structural` / `semantic` / `meta` — gates permission checks |
| `user_id` | UUID | FK → users (NULL for anonymous shared) |
| `api_key_id` | UUID | FK → api_keys (NULL for cookie auth) |
| `actor_kind` | TEXT | `user_interactive` / `agent` / `service` / `scanner` / `system` |
| `client_type`, `client_version` | TEXT | Parsed from `X-Thask-Client` |
| `agent_model`, `agent_session_id` | TEXT | Only meaningful when actor_kind=agent |
| `trigger` | TEXT | `manual` / `import` / `scan_run` / `propagation` / `layout` / `batch` |
| `batch_id` | UUID | Groups related events (one import = many rows + one batch_id) |
| `code_commit`, `source_path`, `confidence` | TEXT | Optional evidence (agent self-report) |
| `metadata` | JSONB | Open-ended |
| `created_at` | TIMESTAMPTZ | DEFAULT now() |

Indexes: `(project_id, created_at DESC)`, `(entity_type, entity_id)`,
`(actor_kind, agent_model)`, partial `(batch_id) WHERE batch_id IS NOT NULL`.

### Node provenance columns (migration 007)

Added to the `nodes` table — snapshot view of "current trust state".

| Column | Type | Purpose |
|---|---|---|
| `description_source` | TEXT | `human` / `agent` / `scanner` / `import` / `unknown` |
| `description_authored_by` | UUID → users | Who wrote it |
| `description_authored_at` | TIMESTAMPTZ | When |
| `description_agent_model` | TEXT | NULL for human writes; e.g. `claude-opus-4-7` |
| `last_verified_at` | TIMESTAMPTZ | Last "still correct" check |
| `last_verified_by` | UUID → users | Human who pressed verify |
| `last_verified_commit` | TEXT | Git SHA verified against |
| `field_provenance` | JSONB | Per-field source/author for non-description fields |

Indexed partial scan for stale agent content:
`idx_nodes_unverified (project_id, last_verified_at) WHERE description_source = 'agent'`.

### API key kind + permissions (migration 008)

| Column | Type | Notes |
|---|---|---|
| `kind` | TEXT | `user_interactive` / `agent` / `service`. Defaults to `user_interactive`. Existing keys backfilled. |
| `permissions` | JSONB | `{ read, write_structural, write_semantic, write_meta, verify, delete, suggest }`. Defaults to all-true for backwards compatibility; `agent` kind UI preset sets `write_semantic=false`, `verify=false`. |

### `node_suggestions` (migration 009)

Agent-proposed updates that need human approval before touching the graph.

| Column | Type | Notes |
|---|---|---|
| `id` | UUID | PK |
| `project_id` | UUID | FK → projects, CASCADE |
| `node_id` | UUID | FK → nodes, CASCADE |
| `field_name` | TEXT | DEFAULT `description` |
| `proposed_value` | TEXT | Required |
| `current_value` | TEXT | Snapshot of node field at time of proposal |
| `rationale` | TEXT | Why the change is needed |
| `evidence` | JSONB | `{ codeCommit, sourcePaths[], confidence }` |
| `proposed_by` | UUID → users | Owner of the agent key |
| `agent_model`, `agent_session_id` | TEXT | Which agent proposed |
| `status` | TEXT | `pending` / `accepted` / `rejected` / `expired` / `superseded` |
| `decided_by`, `decided_at`, `decided_reason` | meta | Filled by the reviewer |
| `created_at` | TIMESTAMPTZ | DEFAULT now() |

### Backfill (migration 010)

One-time `INSERT INTO audit_log SELECT FROM node_history` to preserve historical
writes. Pre-migration rows carry `actor_kind='user_interactive'`, NULL channel
fields, and `metadata.backfilled_from='node_history'`. Re-running is a no-op
because the migration short-circuits when `audit_log` is non-empty.
