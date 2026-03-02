import type { GraphNode, GraphEdge, EdgeType } from './graph';

export type UndoEntry =
  | {
      type: 'addNode';
      nodeId: string;
      nodeData: GraphNode;
    }
  | {
      type: 'deleteNode';
      node: GraphNode;
      edges: GraphEdge[];
    }
  | {
      type: 'updateNode';
      nodeId: string;
      prevData: Partial<GraphNode>;
      nextData: Partial<GraphNode>;
    }
  | {
      type: 'addEdge';
      edgeId: string;
      edgeData: GraphEdge;
    }
  | {
      type: 'deleteEdge';
      edge: GraphEdge;
    }
  | {
      type: 'updateEdgeType';
      edgeId: string;
      prevType: EdgeType;
      nextType: EdgeType;
    }
  | {
      type: 'updateEdgeLabel';
      edgeId: string;
      prevLabel: string | null;
      nextLabel: string | null;
    }
  | {
      type: 'dropOnGroup';
      nodeId: string;
      prevParentId: string | null;
      nextParentId: string | null;
    };
