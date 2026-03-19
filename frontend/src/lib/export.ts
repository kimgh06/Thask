import type cytoscape from 'cytoscape';
import type { GraphNode, GraphEdge, GraphData } from '$lib/types';
import { api } from '$lib/api';

function downloadBlob(blob: Blob, filename: string): void {
	const url = URL.createObjectURL(blob);
	const a = document.createElement('a');
	a.href = url;
	a.download = filename;
	a.click();
	URL.revokeObjectURL(url);
}

export async function exportPNG(cy: cytoscape.Core, filename = 'graph.png'): Promise<void> {
	const blob: Blob = await cy.png({ output: 'blob-promise', bg: '#131214', full: true, scale: 2 });
	downloadBlob(blob, filename);
}

export function exportJSON(nodes: GraphNode[], edges: GraphEdge[], filename = 'graph.json'): void {
	const data = { nodes, edges, exportedAt: new Date().toISOString() };
	const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' });
	downloadBlob(blob, filename);
}

export async function importJSON(
	projectId: string,
	mode: 'replace' | 'merge',
	existingNodes?: GraphNode[],
): Promise<{ nodes: GraphNode[]; edges: GraphEdge[] } | null> {
	return new Promise((resolve) => {
		const input = document.createElement('input');
		input.type = 'file';
		input.accept = '.json';
		input.onchange = async () => {
			const file = input.files?.[0];
			if (!file) { resolve(null); return; }
			try {
				const text = await file.text();
				const parsed = JSON.parse(text);
				const nodes: GraphNode[] = parsed.nodes ?? [];
				const edges: GraphEdge[] = parsed.edges ?? [];

				// Merge mode: offset imported nodes to the right of existing graph
				if (mode === 'merge' && existingNodes && existingNodes.length > 0 && nodes.length > 0) {
					const nodeRight = (n: GraphNode) => n.positionX + (n.width ?? 72) / 2;
					const nodeLeft = (n: GraphNode) => n.positionX - (n.width ?? 72) / 2;
					const existingRightEdge = Math.max(...existingNodes.map(nodeRight));
					const importLeftEdge = Math.min(...nodes.map(nodeLeft));
					const offsetX = existingRightEdge - importLeftEdge + 200;
					for (const node of nodes) {
						node.positionX += offsetX;
					}
				}

				const res = await api.post<GraphData>(`/api/projects/${projectId}/graph/import`, { mode, nodes, edges });
				if (res.error || !res.data) {
					console.error('Import failed:', res.error);
					resolve(null);
					return;
				}
				resolve({ nodes: res.data.nodes ?? [], edges: res.data.edges ?? [] });
			} catch (e) {
				console.error('Import failed:', e);
				resolve(null);
			}
		};
		input.click();
	});
}
