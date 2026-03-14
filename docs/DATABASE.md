# Database Schema

PostgreSQL 17 with pgx/v5 (raw SQL queries). Migrations in `backend/migrations/001_initial.sql`.

---

## ER Diagram

```
Users ◄──── Sessions
  │
  ├──── TeamMembers ────► Teams
  │                         │
  │                         └──── Projects
  │                                 │
  │                          ┌──────┴──────┐
  │                          │             │
  │                        Nodes ◄──── Edges
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

---

## Relations Summary

| Relation | Type | ON DELETE |
|---|---|---|
| sessions → users | M:1 | — |
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
