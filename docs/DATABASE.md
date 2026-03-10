# Database Schema

PostgreSQL 17 with Drizzle ORM.

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

**Self-reference:** `nodes.parentId → nodes.id` (GROUP containment)

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
| id | uuid | PK, default gen |
| email | text | UNIQUE, NOT NULL |
| displayName | text | NOT NULL |
| passwordHash | text | NOT NULL |
| createdAt | timestamp | default now() |
| updatedAt | timestamp | default now() |

### sessions

| Column | Type | Constraints |
|---|---|---|
| id | uuid | PK |
| userId | uuid | FK → users |
| token | text | UNIQUE |
| expiresAt | timestamp | NOT NULL |
| createdAt | timestamp | default now() |

Index: `idx_sessions_user_id (userId)`

### teams

| Column | Type | Constraints |
|---|---|---|
| id | uuid | PK |
| name | text | NOT NULL |
| slug | text | UNIQUE, NOT NULL |
| createdBy | uuid | FK → users |
| createdAt | timestamp | default now() |
| updatedAt | timestamp | default now() |

### teamMembers

| Column | Type | Constraints |
|---|---|---|
| id | uuid | PK |
| teamId | uuid | FK → teams (CASCADE) |
| userId | uuid | FK → users (CASCADE) |
| role | team_role | default: member |
| joinedAt | timestamp | default now() |

Unique: `(teamId, userId)`

### projects

| Column | Type | Constraints |
|---|---|---|
| id | uuid | PK |
| teamId | uuid | FK → teams (CASCADE) |
| name | text | NOT NULL |
| description | text | |
| createdBy | uuid | FK → users |
| createdAt | timestamp | default now() |
| updatedAt | timestamp | default now() |

Index: `idx_projects_team_id (teamId)`

### nodes

| Column | Type | Constraints |
|---|---|---|
| id | uuid | PK |
| projectId | uuid | FK → projects (CASCADE) |
| type | node_type | NOT NULL |
| title | text | NOT NULL |
| description | text | |
| status | node_status | default: IN_PROGRESS |
| assigneeId | uuid | FK → users, nullable |
| tags | text[] | default: {} |
| metadata | jsonb | |
| parentId | uuid | nullable (GROUP containment) |
| positionX | double | canvas X position |
| positionY | double | canvas Y position |
| width | double | GROUP width |
| height | double | GROUP height |
| createdAt | timestamp | default now() |
| updatedAt | timestamp | default now() |

Indexes:
- `idx_nodes_project_id (projectId)`
- `idx_nodes_updated_at (projectId, updatedAt)`
- `idx_nodes_status (projectId, status)`
- `idx_nodes_type (projectId, type)`
- `idx_nodes_assignee (assigneeId)`
- `idx_nodes_parent_id (parentId)`

### edges

| Column | Type | Constraints |
|---|---|---|
| id | uuid | PK |
| projectId | uuid | FK → projects (CASCADE) |
| sourceId | uuid | FK → nodes (CASCADE) |
| targetId | uuid | FK → nodes (CASCADE) |
| edgeType | edge_type | default: related |
| label | text | |
| createdAt | timestamp | default now() |

Constraints:
- Unique: `(sourceId, targetId, edgeType)`
- Check: `sourceId != targetId` (no self-loops)

Indexes:
- `idx_edges_project_id (projectId)`
- `idx_edges_source (sourceId)`
- `idx_edges_target (targetId)`

### nodeHistory

| Column | Type | Constraints |
|---|---|---|
| id | uuid | PK |
| nodeId | uuid | FK → nodes (CASCADE) |
| projectId | uuid | FK → projects (CASCADE) |
| userId | uuid | FK → users |
| action | history_action | NOT NULL |
| fieldName | text | nullable |
| oldValue | text | nullable |
| newValue | text | nullable |
| createdAt | timestamp | default now() |

Indexes:
- `idx_node_history_project_recent (projectId, createdAt)`
- `idx_node_history_node_id (nodeId, createdAt)`

---

## Relations Summary

| Relation | Type | ON DELETE |
|---|---|---|
| sessions → users | M:1 | — |
| teams → users (createdBy) | M:1 | — |
| teamMembers → teams | M:1 | CASCADE |
| teamMembers → users | M:1 | CASCADE |
| projects → teams | M:1 | CASCADE |
| projects → users (createdBy) | M:1 | — |
| nodes → projects | M:1 | CASCADE |
| nodes → users (assigneeId) | M:1 | SET NULL |
| nodes → nodes (parentId) | self-ref | — |
| edges → projects | M:1 | CASCADE |
| edges → nodes (sourceId) | M:1 | CASCADE |
| edges → nodes (targetId) | M:1 | CASCADE |
| nodeHistory → nodes | M:1 | CASCADE |
| nodeHistory → projects | M:1 | CASCADE |
| nodeHistory → users | M:1 | — |
