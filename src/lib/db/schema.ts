import {
  pgTable,
  uuid,
  text,
  timestamp,
  doublePrecision,
  jsonb,
  pgEnum,
  unique,
  check,
  index,
} from 'drizzle-orm/pg-core';
import { sql } from 'drizzle-orm';

// Enums
export const teamRoleEnum = pgEnum('team_role', ['owner', 'admin', 'member', 'viewer']);
export const nodeTypeEnum = pgEnum('node_type', ['FLOW', 'BRANCH', 'TASK', 'BUG', 'API', 'UI', 'GROUP']);
export const nodeStatusEnum = pgEnum('node_status', ['PASS', 'FAIL', 'IN_PROGRESS', 'BLOCKED']);
export const edgeTypeEnum = pgEnum('edge_type', [
  'depends_on',
  'blocks',
  'related',
  'parent_child',
  'triggers',
]);
export const historyActionEnum = pgEnum('history_action', [
  'created',
  'updated',
  'deleted',
  'status_changed',
]);

// Users
export const users = pgTable('users', {
  id: uuid('id').defaultRandom().primaryKey(),
  email: text('email').notNull().unique(),
  displayName: text('display_name').notNull(),
  passwordHash: text('password_hash').notNull(),
  createdAt: timestamp('created_at', { withTimezone: true }).defaultNow().notNull(),
  updatedAt: timestamp('updated_at', { withTimezone: true }).defaultNow().notNull(),
});

// Sessions
export const sessions = pgTable(
  'sessions',
  {
    id: uuid('id').defaultRandom().primaryKey(),
    userId: uuid('user_id')
      .notNull()
      .references(() => users.id, { onDelete: 'cascade' }),
    token: text('token').notNull().unique(),
    expiresAt: timestamp('expires_at', { withTimezone: true }).notNull(),
    createdAt: timestamp('created_at', { withTimezone: true }).defaultNow().notNull(),
  },
  (table) => [index('idx_sessions_user_id').on(table.userId)],
);

// Teams
export const teams = pgTable('teams', {
  id: uuid('id').defaultRandom().primaryKey(),
  name: text('name').notNull(),
  slug: text('slug').notNull().unique(),
  createdBy: uuid('created_by')
    .notNull()
    .references(() => users.id),
  createdAt: timestamp('created_at', { withTimezone: true }).defaultNow().notNull(),
  updatedAt: timestamp('updated_at', { withTimezone: true }).defaultNow().notNull(),
});

// Team members
export const teamMembers = pgTable(
  'team_members',
  {
    id: uuid('id').defaultRandom().primaryKey(),
    teamId: uuid('team_id')
      .notNull()
      .references(() => teams.id, { onDelete: 'cascade' }),
    userId: uuid('user_id')
      .notNull()
      .references(() => users.id, { onDelete: 'cascade' }),
    role: teamRoleEnum('role').notNull().default('member'),
    joinedAt: timestamp('joined_at', { withTimezone: true }).defaultNow().notNull(),
  },
  (table) => [unique('uq_team_user').on(table.teamId, table.userId)],
);

// Projects
export const projects = pgTable(
  'projects',
  {
    id: uuid('id').defaultRandom().primaryKey(),
    teamId: uuid('team_id')
      .notNull()
      .references(() => teams.id, { onDelete: 'cascade' }),
    name: text('name').notNull(),
    description: text('description'),
    createdBy: uuid('created_by')
      .notNull()
      .references(() => users.id),
    createdAt: timestamp('created_at', { withTimezone: true }).defaultNow().notNull(),
    updatedAt: timestamp('updated_at', { withTimezone: true }).defaultNow().notNull(),
  },
  (table) => [index('idx_projects_team_id').on(table.teamId)],
);

// Nodes
export const nodes = pgTable(
  'nodes',
  {
    id: uuid('id').defaultRandom().primaryKey(),
    projectId: uuid('project_id')
      .notNull()
      .references(() => projects.id, { onDelete: 'cascade' }),
    type: nodeTypeEnum('type').notNull(),
    title: text('title').notNull(),
    description: text('description'),
    status: nodeStatusEnum('status').notNull().default('IN_PROGRESS'),
    assigneeId: uuid('assignee_id').references(() => users.id, { onDelete: 'set null' }),
    tags: text('tags').array().default(sql`'{}'::text[]`),
    metadata: jsonb('metadata').default({}),
    parentId: uuid('parent_id'),
    positionX: doublePrecision('position_x').default(0),
    positionY: doublePrecision('position_y').default(0),
    width: doublePrecision('width'),
    height: doublePrecision('height'),
    createdAt: timestamp('created_at', { withTimezone: true }).defaultNow().notNull(),
    updatedAt: timestamp('updated_at', { withTimezone: true }).defaultNow().notNull(),
  },
  (table) => [
    index('idx_nodes_project_id').on(table.projectId),
    index('idx_nodes_updated_at').on(table.projectId, table.updatedAt),
    index('idx_nodes_status').on(table.projectId, table.status),
    index('idx_nodes_type').on(table.projectId, table.type),
    index('idx_nodes_assignee').on(table.assigneeId),
    index('idx_nodes_parent_id').on(table.parentId),
  ],
);

// Edges
export const edges = pgTable(
  'edges',
  {
    id: uuid('id').defaultRandom().primaryKey(),
    projectId: uuid('project_id')
      .notNull()
      .references(() => projects.id, { onDelete: 'cascade' }),
    sourceId: uuid('source_id')
      .notNull()
      .references(() => nodes.id, { onDelete: 'cascade' }),
    targetId: uuid('target_id')
      .notNull()
      .references(() => nodes.id, { onDelete: 'cascade' }),
    edgeType: edgeTypeEnum('edge_type').notNull().default('related'),
    label: text('label'),
    createdAt: timestamp('created_at', { withTimezone: true }).defaultNow().notNull(),
  },
  (table) => [
    unique('uq_edge').on(table.sourceId, table.targetId, table.edgeType),
    check('chk_no_self_ref', sql`${table.sourceId} != ${table.targetId}`),
    index('idx_edges_project_id').on(table.projectId),
    index('idx_edges_source').on(table.sourceId),
    index('idx_edges_target').on(table.targetId),
  ],
);

// Node history (audit log)
export const nodeHistory = pgTable(
  'node_history',
  {
    id: uuid('id').defaultRandom().primaryKey(),
    nodeId: uuid('node_id')
      .notNull()
      .references(() => nodes.id, { onDelete: 'cascade' }),
    projectId: uuid('project_id')
      .notNull()
      .references(() => projects.id, { onDelete: 'cascade' }),
    userId: uuid('user_id')
      .notNull()
      .references(() => users.id),
    action: historyActionEnum('action').notNull(),
    fieldName: text('field_name'),
    oldValue: text('old_value'),
    newValue: text('new_value'),
    createdAt: timestamp('created_at', { withTimezone: true }).defaultNow().notNull(),
  },
  (table) => [
    index('idx_node_history_project_recent').on(table.projectId, table.createdAt),
    index('idx_node_history_node_id').on(table.nodeId, table.createdAt),
  ],
);
