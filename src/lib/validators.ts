import { z } from 'zod';

// Auth
export const registerSchema = z.object({
  email: z.string().email('Invalid email address'),
  password: z.string().min(8, 'Password must be at least 8 characters'),
  displayName: z.string().min(1, 'Display name is required').max(100),
});

export const loginSchema = z.object({
  email: z.string().email('Invalid email address'),
  password: z.string().min(1, 'Password is required'),
});

// Teams
export const createTeamSchema = z.object({
  name: z.string().min(1, 'Team name is required').max(100),
  slug: z
    .string()
    .min(1)
    .max(100)
    .regex(/^[a-z0-9-]+$/, 'Slug must be lowercase alphanumeric with dashes'),
});

export const inviteMemberSchema = z.object({
  email: z.string().email('Invalid email address'),
  role: z.enum(['admin', 'member', 'viewer']).default('member'),
});

// Projects
export const createProjectSchema = z.object({
  name: z.string().min(1, 'Project name is required').max(200),
  description: z.string().max(2000).optional(),
});

export const updateProjectSchema = z.object({
  name: z.string().min(1).max(200).optional(),
  description: z.string().max(2000).optional(),
});

// Nodes
export const createNodeSchema = z.object({
  type: z.enum(['FLOW', 'BRANCH', 'TASK', 'BUG', 'API', 'UI', 'GROUP']),
  title: z.string().min(1, 'Title is required').max(300),
  description: z.string().max(5000).optional(),
  status: z.enum(['PASS', 'FAIL', 'IN_PROGRESS', 'BLOCKED']).default('IN_PROGRESS'),
  assigneeId: z.string().uuid().optional().nullable(),
  tags: z.array(z.string().max(50)).max(20).default([]),
  metadata: z.record(z.unknown()).default({}),
  positionX: z.number().default(0),
  positionY: z.number().default(0),
  width: z.number().min(80).optional(),
  height: z.number().min(50).optional(),
});

export const updateNodeSchema = z.object({
  type: z.enum(['FLOW', 'BRANCH', 'TASK', 'BUG', 'API', 'UI', 'GROUP']).optional(),
  title: z.string().min(1).max(300).optional(),
  description: z.string().max(5000).optional().nullable(),
  status: z.enum(['PASS', 'FAIL', 'IN_PROGRESS', 'BLOCKED']).optional(),
  assigneeId: z.string().uuid().optional().nullable(),
  tags: z.array(z.string().max(50)).max(20).optional(),
  metadata: z.record(z.unknown()).optional(),
  parentId: z.string().uuid().optional().nullable(),
  width: z.number().min(80).optional().nullable(),
  height: z.number().min(50).optional().nullable(),
});

export const batchPositionSchema = z.object({
  positions: z.array(
    z.object({
      id: z.string().uuid(),
      x: z.number(),
      y: z.number(),
      width: z.number().min(80).optional(),
      height: z.number().min(50).optional(),
    }),
  ),
});

// Edges
export const createEdgeSchema = z.object({
  sourceId: z.string().uuid(),
  targetId: z.string().uuid(),
  edgeType: z.enum(['depends_on', 'blocks', 'related', 'parent_child', 'triggers']).default('related'),
  label: z.string().max(100).optional(),
});

export const updateEdgeSchema = z.object({
  edgeType: z.enum(['depends_on', 'blocks', 'related', 'parent_child', 'triggers']).optional(),
  label: z.string().max(100).optional().nullable(),
});
