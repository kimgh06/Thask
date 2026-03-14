import type cytoscape from 'cytoscape';

interface Edgehandles {
	active: boolean;
	start: (node: cytoscape.NodeSingular) => void;
	stop: () => void;
}

interface PortOverlayOptions {
	isResizing: () => boolean;
	isOnGroupBorder: (pos: { x: number; y: number }, node: cytoscape.NodeSingular) => boolean;
}

const PORT_SIZE = 20;

export function attachPortOverlay(
	cy: cytoscape.Core,
	portOverlay: HTMLDivElement,
	eh: Edgehandles,
	options: PortOverlayOptions,
): { getPortSource: () => cytoscape.NodeSingular | null; cleanup: () => void } {
	let portSource: cytoscape.NodeSingular | null = null;
	let portHideTimer: ReturnType<typeof setTimeout> | null = null;

	function showPorts(node: cytoscape.NodeSingular) {
		portSource = node;
		if (portHideTimer) {
			clearTimeout(portHideTimer);
			portHideTimer = null;
		}
		positionPorts(node);
		portOverlay.style.display = 'block';
	}

	function hidePorts(delay = 150) {
		portHideTimer = setTimeout(() => {
			portOverlay.style.display = 'none';
			portSource = null;
		}, delay);
	}

	function positionPorts(node: cytoscape.NodeSingular) {
		const rbb = node.renderedBoundingBox({ includeLabels: false, includeOverlays: false });
		const cxR = (rbb.x1 + rbb.x2) / 2;
		const cyR = (rbb.y1 + rbb.y2) / 2;
		const half = PORT_SIZE / 2;

		const positions = [
			{ cls: 'port-top', x: cxR - half, y: rbb.y1 - PORT_SIZE },
			{ cls: 'port-right', x: rbb.x2, y: cyR - half },
			{ cls: 'port-bottom', x: cxR - half, y: rbb.y2 },
			{ cls: 'port-left', x: rbb.x1 - PORT_SIZE, y: cyR - half },
		];

		for (const p of positions) {
			const el = portOverlay.querySelector(`.${p.cls}`) as HTMLElement | null;
			if (el) {
				el.style.left = `${p.x}px`;
				el.style.top = `${p.y}px`;
			}
		}
	}

	// DOM listeners on port overlay
	function onPortMouseDown(e: MouseEvent) {
		if (!(e.target as HTMLElement).classList.contains('port-dot')) return;
		e.preventDefault();
		e.stopPropagation();
		if (portSource) {
			portOverlay.style.display = 'none';
			eh.start(portSource);
			const onMouseUp = () => {
				document.removeEventListener('mouseup', onMouseUp);
				eh.stop();
			};
			document.addEventListener('mouseup', onMouseUp);
		}
	}

	function onPortMouseEnter() {
		if (portHideTimer) {
			clearTimeout(portHideTimer);
			portHideTimer = null;
		}
	}

	function onPortMouseLeave() {
		if (!eh.active) hidePorts(100);
	}

	portOverlay.addEventListener('mousedown', onPortMouseDown);
	portOverlay.addEventListener('mouseenter', onPortMouseEnter);
	portOverlay.addEventListener('mouseleave', onPortMouseLeave);

	// Cytoscape node hover → show/hide ports
	cy.on('mouseover', 'node', (e) => {
		const node = e.target as cytoscape.NodeSingular;
		if (eh.active || node.grabbed() || options.isResizing()) return;
		if (node.data('nodeType') === 'GROUP' && !node.hasClass('group-collapsed')) {
			if (!options.isOnGroupBorder(e.position, node)) return;
		}
		showPorts(node);
	});

	cy.on('mouseout', 'node', () => {
		if (!eh.active) hidePorts();
	});

	cy.on('pan zoom', () => {
		if (portSource && portOverlay.style.display === 'block') positionPorts(portSource);
	});

	cy.on('ehstop ehcancel', () => {
		portOverlay.style.display = 'none';
		portSource = null;
	});

	return {
		getPortSource: () => portSource,
		cleanup: () => {
			if (portHideTimer) clearTimeout(portHideTimer);
			portOverlay.removeEventListener('mousedown', onPortMouseDown);
			portOverlay.removeEventListener('mouseenter', onPortMouseEnter);
			portOverlay.removeEventListener('mouseleave', onPortMouseLeave);
		},
	};
}
