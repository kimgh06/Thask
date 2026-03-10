# Architecture

## Overview

Thask is a **Next.js 15 full-stack application** that visualizes product flows, tasks, and bugs as a linked node graph. It uses a layered architecture with clear separation of concerns.

```
┌──────────────────────────────────────────────────────────┐
│  UI Layer (React Components)                             │
│  CytoscapeCanvas · GraphToolbar · NodeDetailPanel · ...  │
├──────────────────────────────────────────────────────────┤
│  State Layer (Zustand + React Query)                     │
│  useAuthStore · useGraphStore · useUndoStore             │
├──────────────────────────────────────────────────────────┤
│  Hook Layer (Business Logic)                             │
│  useGraphData · useNodeDetail · useUndoRedo              │
├──────────────────────────────────────────────────────────┤
│  API Layer (Next.js Route Handlers)                      │
│  /api/auth/* · /api/teams/* · /api/projects/*            │
├──────────────────────────────────────────────────────────┤
│  Data Layer (Drizzle ORM + PostgreSQL 17)                │
│  users · teams · projects · nodes · edges · nodeHistory  │
└──────────────────────────────────────────────────────────┘
```

---

## Data Flow

```
Stores (Zustand)                 Hooks (React Query)
  useAuthStore ──┐                 useGraphData ──┐
  useGraphStore ─┤                 useNodeDetail ─┤
  useUndoStore ──┘                 useUndoRedo ───┘
        │                                │
        ▼                                ▼
   ┌──────────────────────────────────────────┐
   │         Components (UI Layer)            │
   │  ProjectGraphPage                        │
   │   ├─ CytoscapeCanvas (graph render)      │
   │   ├─ GraphToolbar (controls)             │
   │   ├─ GraphMinimap (overview)             │
   │   ├─ NodeDetailPanel (editor)            │
   │   ├─ AddNodeModal (create)               │
   │   ├─ EdgeColorPopover (edge type)        │
   │   └─ ConfirmDialog (delete)              │
   └──────────────┬───────────────────────────┘
                  │ fetch / mutate
                  ▼
   ┌──────────────────────────────────────────┐
   │         API Routes (Backend)             │
   │  POST /api/auth/login                    │
   │  GET  /api/projects/[id]/nodes           │
   │  PATCH /api/projects/[id]/nodes/[nodeId] │
   │  ...22 endpoints total                   │
   └──────────────┬───────────────────────────┘
                  │ query
                  ▼
   ┌──────────────────────────────────────────┐
   │         PostgreSQL (Drizzle ORM)         │
   │  8 tables, 5 enums, 12+ indexes         │
   └──────────────────────────────────────────┘
```

---

## Directory Structure

```
src/
├── app/                              # Next.js App Router
│   ├── (auth)/                       # Auth route group
│   │   ├── login/page.tsx
│   │   └── register/page.tsx
│   ├── (dashboard)/                  # Protected route group
│   │   ├── layout.tsx                # Sidebar + Header layout
│   │   ├── dashboard/page.tsx        # Dashboard index
│   │   └── dashboard/
│   │       └── [teamSlug]/
│   │           ├── page.tsx          # Team view
│   │           └── [projectId]/
│   │               └── page.tsx      # Graph editor (main page)
│   ├── api/                          # REST API (22 endpoints)
│   │   ├── auth/                     # login, register, me, logout
│   │   ├── teams/                    # CRUD + members + projects
│   │   └── projects/                 # nodes, edges, positions, impact
│   ├── layout.tsx                    # Root layout with Providers
│   └── page.tsx                      # Landing redirect
│
├── components/
│   ├── auth/                         # LoginForm, RegisterForm
│   ├── graph/                        # CytoscapeCanvas, GraphToolbar,
│   │                                 # AddNodeModal, EdgeColorPopover,
│   │                                 # GraphMinimap
│   ├── layout/                       # Header, Sidebar, Providers
│   ├── panels/                       # NodeDetailPanel
│   └── ui/                           # ConfirmDialog
│
├── hooks/
│   ├── useGraphData.ts               # Nodes/edges CRUD (React Query)
│   ├── useNodeDetail.ts              # Selected node detail + history
│   ├── useUndoRedo.ts                # Undo/redo execution
│   └── useEdgePopover.ts             # Edge popover positioning
│
├── stores/
│   ├── useAuthStore.ts               # User session state
│   ├── useGraphStore.ts              # Selection, filters, impact mode
│   └── useUndoStore.ts               # Undo/redo stacks (max 50)
│
├── types/
│   ├── api.ts                        # ApiResponse<T>
│   ├── auth.ts                       # AuthUser, Team, Project
│   ├── graph.ts                      # GraphNode, GraphEdge, enums
│   ├── undo.ts                       # UndoEntry (10 action types)
│   ├── cytoscape-edgehandles.d.ts
│   └── cytoscape-fcose.d.ts
│
├── lib/
│   ├── auth/
│   │   ├── guard.ts                  # requireAuth() API guard
│   │   ├── session.ts                # Token + cookie management
│   │   └── password.ts               # bcrypt hash/verify
│   ├── cytoscape/
│   │   ├── styles.ts                 # 60+ style rules
│   │   ├── layouts.ts                # fCOSE + preset layouts
│   │   ├── groupHelpers.ts           # Child/descendant queries
│   │   └── impact.ts                 # Impact mode activation
│   ├── db/
│   │   ├── index.ts                  # Drizzle connection
│   │   └── schema.ts                 # 8 tables, 5 enums
│   ├── utils.ts                      # cn() class helper
│   ├── validators.ts                 # 10+ Zod schemas
│   └── waterfall.ts                  # Status propagation logic
│
└── middleware.ts                     # Auth redirect middleware
```

---

## Key Modules

### Stores (Zustand)

| Store | Responsibility |
|---|---|
| `useAuthStore` | User session, login state, `logout()` |
| `useGraphStore` | Node selection, type/status filters, impact mode, collapsed groups |
| `useUndoStore` | Undo/redo stacks with max 50 entries |

### Hooks

| Hook | Responsibility |
|---|---|
| `useGraphData(projectId)` | React Query: nodes/edges CRUD, batch positions, group/ungroup |
| `useNodeDetail(projectId, nodeId, nodes, edges)` | Optimistic node detail + async history fetch |
| `useUndoRedo(projectId)` | Execute undo/redo from store stacks |
| `useEdgePopover(graphRef)` | Edge click popover positioning |

### Auth Flow

```
Browser → middleware.ts (redirect check)
       → API route → requireAuth() → validateSession()
                                       → sessions table (token lookup)
                                       → users table (join)
                                       → return AuthContext { userId, email, displayName }
```

- Session tokens: 32-byte hex, 7-day expiry
- Storage: HttpOnly cookie (`thask_session`)
- Passwords: bcrypt with 12 salt rounds

### Access Control

```
User → TeamMembers → Teams → Projects → Nodes/Edges
```

Every API route verifies:
1. Valid session (`requireAuth()`)
2. Team membership for the project (`verifyProjectAccess()`)
