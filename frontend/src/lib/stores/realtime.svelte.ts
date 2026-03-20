class RealtimeStore {
	connected = $state(false);
	private source: EventSource | null = null;
	private onGraphChange: (() => void) | null = null;
	private debounceTimer: ReturnType<typeof setTimeout> | null = null;

	connect(projectId: string, onGraphChange: () => void) {
		this.disconnect();
		this.onGraphChange = onGraphChange;

		// Use SvelteKit proxy (same origin) so cookies are sent automatically
		const url = `/api/projects/${projectId}/events`;
		this.source = new EventSource(url);

		this.source.addEventListener('connected', () => {
			this.connected = true;
		});

		const eventTypes = [
			'node.created',
			'node.updated',
			'node.deleted',
			'edge.created',
			'edge.updated',
			'edge.deleted',
			'graph.layout',
			'graph.import',
		];

		for (const type of eventTypes) {
			this.source.addEventListener(type, () => {
				this.debouncedRefresh();
			});
		}

		this.source.onerror = () => {
			this.connected = false;
		};
	}

	private debouncedRefresh() {
		if (this.debounceTimer) {
			clearTimeout(this.debounceTimer);
		}
		this.debounceTimer = setTimeout(() => {
			this.onGraphChange?.();
			this.debounceTimer = null;
		}, 300);
	}

	disconnect() {
		if (this.debounceTimer) {
			clearTimeout(this.debounceTimer);
			this.debounceTimer = null;
		}
		this.source?.close();
		this.source = null;
		this.connected = false;
	}
}

export const realtimeStore = new RealtimeStore();
