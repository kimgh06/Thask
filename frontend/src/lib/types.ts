export type NodeType = 'FLOW' | 'BRANCH' | 'TASK' | 'BUG' | 'API' | 'UI' | 'GROUP';
export type NodeStatus = 'PASS' | 'FAIL' | 'IN_PROGRESS' | 'BLOCKED';
export type EdgeType = 'depends_on' | 'blocks' | 'related' | 'parent_child' | 'triggers';
export type TeamRole = 'owner' | 'admin' | 'member' | 'viewer';

export interface User {
	id: string;
	email: string;
	displayName: string;
}

export interface Team {
	id: string;
	name: string;
	slug: string;
	createdBy: string;
	createdAt: string;
	updatedAt: string;
	projects?: Project[];
}

export interface Project {
	id: string;
	teamId: string;
	name: string;
	description: string | null;
	createdBy: string;
	linkSharing: 'off' | 'viewer' | 'editor';
	createdAt: string;
	updatedAt: string;
}

export interface GraphNode {
	id: string;
	projectId: string;
	type: NodeType;
	title: string;
	description: string | null;
	status: NodeStatus;
	assigneeId: string | null;
	tags: string[];
	metadata: Record<string, unknown>;
	parentId: string | null;
	positionX: number;
	positionY: number;
	width: number | null;
	height: number | null;
	createdAt: string;
	updatedAt: string;
	createdBy?: string;
	creatorEmail?: string;
}

export interface GraphEdge {
	id: string;
	projectId: string;
	sourceId: string;
	targetId: string;
	edgeType: EdgeType;
	label: string | null;
	sourcePort: string;
	targetPort: string;
	createdAt: string;
}

export interface GraphData {
	nodes: GraphNode[];
	edges: GraphEdge[];
}

export interface NodeHistoryEntry {
	id: string;
	action: string;
	fieldName: string | null;
	oldValue: string | null;
	newValue: string | null;
	createdAt: string;
	userName: string;
}

export interface NodeDetail extends GraphNode {
	connectedEdges: GraphEdge[];
	connectedNodeIds: string[];
	history: NodeHistoryEntry[];
}

export interface StatusChange {
	nodeId: string;
	oldStatus: NodeStatus;
	newStatus: NodeStatus;
}

export interface NodeUpdateResult {
	node: GraphNode;
	propagated: StatusChange[];
}

export interface TeamMember {
	id: string;
	teamId: string;
	userId: string;
	role: TeamRole;
	joinedAt: string;
	user: User;
}

export type APIKeyKind = 'user_interactive' | 'agent' | 'service';

export interface APIKeyPermissions {
	read: boolean;
	write_structural: boolean;
	write_semantic: boolean;
	write_meta: boolean;
	verify: boolean;
	delete: boolean;
	suggest: boolean;
}

export interface APIKey {
	id: string;
	name: string;
	keyPrefix: string;
	kind: APIKeyKind;
	permissions: APIKeyPermissions;
	lastUsedAt: string | null;
	expiresAt: string | null;
	createdAt: string;
}

export interface APIKeyCreated extends APIKey {
	key: string;
}

export type ProjectRole = 'editor' | 'viewer';

export interface ProjectMember {
	id: string;
	projectId: string;
	userId: string;
	role: ProjectRole;
	createdAt: string;
	user: User;
}

export interface SharingInfo {
	linkSharing: 'off' | 'viewer' | 'editor';
	shareUrl: string | null;
	members: ProjectMember[];
}

export interface ImpactResult {
	changedNodes: GraphNode[];
	impactedNodes: GraphNode[];
	failNodes: GraphNode[];
	impactEdges: GraphEdge[];
}
