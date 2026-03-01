'use client';

import { useEffect, useRef, useCallback, forwardRef, useImperativeHandle } from 'react';
import cytoscape, { type Core, type EventObject } from 'cytoscape';
import fcose from 'cytoscape-fcose';
import edgehandles from 'cytoscape-edgehandles';
import { cytoscapeStylesheet } from '@/lib/cytoscape/styles';
import { getFcoseLayout, getPresetLayout } from '@/lib/cytoscape/layouts';
import type { GraphNode, GraphEdge } from '@/types/graph';

// Register extensions once
let extensionsRegistered = false;
if (!extensionsRegistered) {
  cytoscape.use(fcose);
  cytoscape.use(edgehandles);
  extensionsRegistered = true;
}

const HANDLE_ID = '__eh-handle__';
const RESIZE_HANDLE_ID = '__resize-handle__';

export interface CytoscapeCanvasHandle {
  getCy: () => Core | null;
  runLayout: () => void;
  fitToView: () => void;
  zoomIn: () => void;
  zoomOut: () => void;
}

interface CytoscapeCanvasProps {
  nodes: GraphNode[];
  edges: GraphEdge[];
  collapsedGroupIds?: string[];
  onNodeSelect?: (nodeId: string | null) => void;
  onNodeDragEnd?: (nodeId: string, x: number, y: number) => void;
  onConnectEnd?: (sourceId: string, targetId: string, renderedPosition: { x: number; y: number }) => void;
  onEdgeClick?: (edgeId: string, renderedPosition: { x: number; y: number }) => void;
  onNodeDropOnGroup?: (nodeId: string, groupId: string | null) => void;
  onToggleGroupCollapse?: (groupId: string) => void;
  onNodeResize?: (nodeId: string, width: number, height: number) => void;
}

export const CytoscapeCanvas = forwardRef<CytoscapeCanvasHandle, CytoscapeCanvasProps>(
  function CytoscapeCanvas(
    { nodes, edges, collapsedGroupIds = [], onNodeSelect, onNodeDragEnd, onConnectEnd, onEdgeClick, onNodeDropOnGroup, onToggleGroupCollapse, onNodeResize },
    ref,
  ) {
    const containerRef = useRef<HTMLDivElement>(null);
    const cyRef = useRef<Core | null>(null);
    const ehRef = useRef<ReturnType<Core['edgehandles']> | null>(null);
    const initialLayoutDone = useRef(false);

    // Store latest callbacks in refs
    const callbacksRef = useRef({ onNodeSelect, onNodeDragEnd, onConnectEnd, onEdgeClick, onNodeDropOnGroup, onToggleGroupCollapse, onNodeResize });
    callbacksRef.current = { onNodeSelect, onNodeDragEnd, onConnectEnd, onEdgeClick, onNodeDropOnGroup, onToggleGroupCollapse, onNodeResize };

    const runLayout = useCallback(() => {
      const cy = cyRef.current;
      if (!cy || cy.nodes().length === 0) return;
      cy.layout(getFcoseLayout()).run();
    }, []);

    useImperativeHandle(ref, () => ({
      getCy: () => cyRef.current,
      runLayout,
      fitToView: () => cyRef.current?.fit(undefined, 50),
      zoomIn: () => {
        const cy = cyRef.current;
        if (cy) cy.zoom({ level: cy.zoom() * 1.2, renderedPosition: { x: cy.width() / 2, y: cy.height() / 2 } });
      },
      zoomOut: () => {
        const cy = cyRef.current;
        if (cy) cy.zoom({ level: cy.zoom() / 1.2, renderedPosition: { x: cy.width() / 2, y: cy.height() / 2 } });
      },
    }));

    // Initialize Cytoscape once
    useEffect(() => {
      if (!containerRef.current) return;

      const cy = cytoscape({
        container: containerRef.current,
        style: cytoscapeStylesheet,
        layout: { name: 'preset' },
        minZoom: 0.2,
        maxZoom: 4,
        wheelSensitivity: 0.3,
      });

      cyRef.current = cy;

      // Initialize edgehandles (non-drawMode; we trigger start() manually)
      const eh = cy.edgehandles({
        canConnect: (sourceNode, targetNode) =>
          !sourceNode.same(targetNode) && targetNode.id() !== HANDLE_ID && targetNode.id() !== RESIZE_HANDLE_ID,
        edgeParams: () => ({}),
        hoverDelay: 150,
        snap: true,
        snapThreshold: 50,
        snapFrequency: 15,
        noEdgeEventsInDraw: true,
        disableBrowserGestures: true,
      });
      eh.enable();
      ehRef.current = eh;

      // --- Custom hover handle ---
      // Add an invisible handle node used as a drag point
      cy.add({
        group: 'nodes',
        data: { id: HANDLE_ID },
        classes: 'eh-custom-handle',
        position: { x: 0, y: 0 },
      });
      const handleNode = cy.getElementById(HANDLE_ID);
      // Apply styles directly (override base node 80x80)
      handleNode.style({
        'width': 14,
        'height': 14,
        'shape': 'ellipse',
        'background-color': '#3b82f6',
        'border-width': 2,
        'border-color': '#1d4ed8',
        'overlay-opacity': 0,
        'label': '',
        'display': 'none',
      });
      handleNode.ungrabify();
      handleNode.unselectify();

      let hoverSource: cytoscape.NodeSingular | null = null;
      let handleHovered = false;
      let hideTimeout: ReturnType<typeof setTimeout> | null = null;

      function positionHandleNearestSide(node: cytoscape.NodeSingular, mousePos: { x: number; y: number }) {
        if (hideTimeout) { clearTimeout(hideTimeout); hideTimeout = null; }
        hoverSource = node;
        const bb = node.boundingBox({});
        const cx = (bb.x1 + bb.x2) / 2;
        const cy2 = (bb.y1 + bb.y2) / 2;

        // Distance from mouse to each side
        const dTop = Math.abs(mousePos.y - bb.y1);
        const dBottom = Math.abs(mousePos.y - bb.y2);
        const dLeft = Math.abs(mousePos.x - bb.x1);
        const dRight = Math.abs(mousePos.x - bb.x2);
        const min = Math.min(dTop, dBottom, dLeft, dRight);

        let hx: number, hy: number;
        if (min === dTop) { hx = cx; hy = bb.y1 - 4; }
        else if (min === dBottom) { hx = cx; hy = bb.y2 + 4; }
        else if (min === dLeft) { hx = bb.x1 - 4; hy = cy2; }
        else { hx = bb.x2 + 4; hy = cy2; }

        handleNode.position({ x: hx, y: hy });
        handleNode.style('display', 'element');
      }

      function hideHandle(delay = 200) {
        hideTimeout = setTimeout(() => {
          if (!handleHovered && !eh.active) {
            handleNode.style('display', 'none');
            hoverSource = null;
          }
        }, delay);
      }

      // Position handle on nearest side based on mouse position
      cy.on('mousemove', 'node', (e) => {
        const node = e.target;
        if (node.id() === HANDLE_ID || node.id() === RESIZE_HANDLE_ID) return;
        if (node.data('nodeType') === 'GROUP') return;
        if (eh.active) return;
        if (node.grabbed()) return;
        positionHandleNearestSide(node, e.position);
      });

      cy.on('mouseout', 'node', (e) => {
        const node = e.target;
        if (node.id() === HANDLE_ID) return;
        if (eh.active) return;
        hideHandle();
      });

      // Track handle hover state
      cy.on('mouseover', `.eh-custom-handle`, () => {
        handleHovered = true;
        if (hideTimeout) { clearTimeout(hideTimeout); hideTimeout = null; }
      });

      cy.on('mouseout', `.eh-custom-handle`, () => {
        handleHovered = false;
        if (!eh.active) hideHandle();
      });

      // Hide edge handle when node drag starts + temporarily unparent for GROUP stability
      cy.on('grab', 'node', (e) => {
        const node = e.target;
        if (node.id() === HANDLE_ID || node.id() === RESIZE_HANDLE_ID) return;
        handleNode.style('display', 'none');
        hoverSource = null;

        // If node is inside a GROUP, temporarily unparent so GROUP stays fixed
        const parentId = node.data('parent');
        if (parentId) {
          const parentNode = cy.getElementById(parentId);
          if (parentNode.length && parentNode.data('nodeType') === 'GROUP' && !parentNode.hasClass('group-collapsed')) {
            dragOriginalParent = parentId;
            dragOriginalMinDims = {
              w: parentNode.data('minWidth'),
              h: parentNode.data('minHeight'),
            };

            // Lock GROUP to current size and position before unparenting
            const bb = parentNode.boundingBox({});
            const bbW = bb.x2 - bb.x1;
            const bbH = bb.y2 - bb.y1;
            const groupCenter = { x: (bb.x1 + bb.x2) / 2, y: (bb.y1 + bb.y2) / 2 };
            parentNode.data('minWidth', bbW);
            parentNode.data('minHeight', bbH);

            node.move({ parent: null });

            // Empty GROUP: set position + style to prevent jump
            if (!parentNode.isParent()) {
              parentNode.position(groupCenter);
              parentNode.style({ width: bbW, height: bbH });
            }
          }
        }

        // Precompute descendant IDs for cycle prevention during GROUP nesting
        dragDescendantIds = node.data('nodeType') === 'GROUP' && node.isParent()
          ? new Set(node.descendants().map((d: cytoscape.NodeSingular) => d.id()))
          : null;
      });

      // Start edge drawing when handle is tapped/dragged
      cy.on('tapstart', `.eh-custom-handle`, () => {
        if (hoverSource && eh.canStartOn(hoverSource)) {
          handleNode.style('display', 'none');
          eh.start(hoverSource);
        }
      });

      // After edge drawing ends, clean up
      cy.on('ehstop ehcancel', () => {
        handleNode.style('display', 'none');
        hoverSource = null;
        handleHovered = false;
      });

      // On edge complete: remove temp edge, call callback
      cy.on('ehcomplete', (_event, sourceNode, targetNode, addedEdge) => {
        addedEdge.remove();

        const cb = callbacksRef.current;
        const renderedPos = targetNode.renderedPosition();
        cb.onConnectEnd?.(sourceNode.id(), targetNode.id(), { x: renderedPos.x, y: renderedPos.y });
      });

      // Node selection
      cy.on('tap', 'node', (evt: EventObject) => {
        if (evt.target.id() === HANDLE_ID || evt.target.id() === RESIZE_HANDLE_ID) return;
        const cb = callbacksRef.current;
        cb.onNodeSelect?.(evt.target.id());
      });

      cy.on('tap', (evt: EventObject) => {
        if (evt.target === cy) {
          const cb = callbacksRef.current;
          cb.onNodeSelect?.(null);
        }
      });

      // Highlight GROUP nodes when dragging a node over them
      cy.on('drag', 'node', (evt: EventObject) => {
        const node = evt.target;
        if (node.id() === HANDLE_ID || node.id() === RESIZE_HANDLE_ID) return;
        const pos = node.position();
        cy.nodes('[nodeType="GROUP"]').forEach((g) => {
          // Skip self, collapsed, and descendants (cycle prevention)
          if (g.id() === node.id() || g.hasClass('group-collapsed') || dragDescendantIds?.has(g.id())) {
            g.removeClass('drop-target');
            return;
          }
          const bb = g.boundingBox({});
          if (pos.x >= bb.x1 && pos.x <= bb.x2 && pos.y >= bb.y1 && pos.y <= bb.y2) {
            g.addClass('drop-target');
          } else {
            g.removeClass('drop-target');
          }
        });
      });

      cy.on('dragfree', 'node', (evt: EventObject) => {
        if (evt.target.id() === HANDLE_ID || evt.target.id() === RESIZE_HANDLE_ID) return;
        cy.nodes('.drop-target').removeClass('drop-target');
        const cb = callbacksRef.current;
        const node = evt.target;
        const pos = node.position();
        cb.onNodeDragEnd?.(node.id(), pos.x, pos.y);

        // Find GROUP node under drop position (skip self, collapsed, descendants)
        const groupNodes = cy.nodes().filter(
          (n) => n.data('nodeType') === 'GROUP' && n.id() !== node.id() && n.id() !== HANDLE_ID && !n.hasClass('group-collapsed') && !dragDescendantIds?.has(n.id()),
        );
        let targetGroup: string | null = null;
        groupNodes.forEach((g) => {
          if (targetGroup) return;
          const bb = g.boundingBox({});
          if (pos.x >= bb.x1 && pos.x <= bb.x2 && pos.y >= bb.y1 && pos.y <= bb.y2) {
            targetGroup = g.id();
          }
        });

        if (dragOriginalParent) {
          // Node was temporarily unparented from a GROUP during grab
          if (targetGroup === dragOriginalParent) {
            // Dropped back inside original group → re-parent (no API call)
            node.move({ parent: dragOriginalParent });
          } else {
            // Moved to different group or out of all groups → notify server
            cb.onNodeDropOnGroup?.(node.id(), targetGroup);
          }

          // Restore original GROUP dimensions
          const originalGroup = cy.getElementById(dragOriginalParent);
          if (originalGroup.length) {
            if (dragOriginalMinDims?.w !== undefined) {
              originalGroup.data('minWidth', dragOriginalMinDims.w);
            } else {
              originalGroup.removeData('minWidth');
            }
            if (dragOriginalMinDims?.h !== undefined) {
              originalGroup.data('minHeight', dragOriginalMinDims.h);
            } else {
              originalGroup.removeData('minHeight');
            }
          }

          dragOriginalParent = null;
          dragOriginalMinDims = null;
        } else {
          // Normal case: node was not inside any group
          const currentParent = node.data('parent') || null;
          if (targetGroup !== currentParent) {
            cb.onNodeDropOnGroup?.(node.id(), targetGroup);
          }
        }
        dragDescendantIds = null;
      });

      // Edge click for color change
      cy.on('tap', 'edge', (evt: EventObject) => {
        const cb = callbacksRef.current;
        const edge = evt.target;
        const renderedPos = edge.renderedMidpoint();
        cb.onEdgeClick?.(edge.id(), { x: renderedPos.x, y: renderedPos.y });
      });

      // Double-click GROUP to toggle collapse/expand
      cy.on('dbltap', 'node[nodeType="GROUP"]', (evt: EventObject) => {
        const cb = callbacksRef.current;
        cb.onToggleGroupCollapse?.(evt.target.id());
      });

      // --- Resize handle for GROUP nodes ---
      cy.add({
        group: 'nodes',
        data: { id: RESIZE_HANDLE_ID },
        classes: 'resize-handle',
        position: { x: 0, y: 0 },
      });
      const resizeHandle = cy.getElementById(RESIZE_HANDLE_ID);
      resizeHandle.style('display', 'none');
      resizeHandle.ungrabify();
      resizeHandle.unselectify();

      let resizeTarget: cytoscape.NodeSingular | null = null;
      let isResizing = false;
      let resizeStartPos = { x: 0, y: 0 };
      let resizeStartDims = { w: 0, h: 0 };
      let resizeStartCenter = { x: 0, y: 0 };

      // Temporary unparent state — keeps GROUP fixed while child is dragged
      let dragOriginalParent: string | null = null;
      let dragOriginalMinDims: { w: number | undefined; h: number | undefined } | null = null;
      let dragDescendantIds: Set<string> | null = null;

      function showResizeHandle(groupNode: cytoscape.NodeSingular) {
        if (isResizing || eh.active) return;
        if (groupNode.hasClass('group-collapsed')) return;
        resizeTarget = groupNode;
        const bb = groupNode.boundingBox({});
        resizeHandle.position({ x: bb.x2, y: bb.y2 });
        resizeHandle.style('display', 'element');
      }

      function hideResizeHandle() {
        if (isResizing) return;
        resizeHandle.style('display', 'none');
        resizeTarget = null;
      }

      cy.on('select', 'node[nodeType="GROUP"]', (e) => {
        showResizeHandle(e.target);
      });
      cy.on('unselect', 'node[nodeType="GROUP"]', () => {
        if (!isResizing) hideResizeHandle();
      });

      // Update resize handle position during GROUP drag
      cy.on('drag', 'node[nodeType="GROUP"]', (e) => {
        if (!resizeTarget || resizeTarget.id() !== e.target.id()) return;
        if (isResizing) return;
        const bb = e.target.boundingBox({});
        resizeHandle.position({ x: bb.x2, y: bb.y2 });
      });

      const container = cy.container()!;

      function isNearResizeHandle(e: MouseEvent): boolean {
        if (!resizeTarget || resizeHandle.style('display') === 'none') return false;
        const handleRenderedPos = resizeHandle.renderedPosition();
        const dx = e.offsetX - handleRenderedPos.x;
        const dy = e.offsetY - handleRenderedPos.y;
        return Math.sqrt(dx * dx + dy * dy) <= 14;
      }

      function onResizeMouseDown(e: MouseEvent) {
        if (!resizeTarget || !isNearResizeHandle(e)) return;

        e.preventDefault();
        e.stopPropagation();
        isResizing = true;
        container.style.cursor = 'nwse-resize';
        cy.panningEnabled(false);
        cy.boxSelectionEnabled(false);
        resizeTarget.ungrabify();
        resizeTarget.addClass('resizing');

        const bb = resizeTarget.boundingBox({});
        resizeStartDims = { w: bb.x2 - bb.x1, h: bb.y2 - bb.y1 };
        resizeStartPos = { x: e.clientX, y: e.clientY };
        resizeStartCenter = { ...resizeTarget.position() };
      }

      function onResizeMouseMove(e: MouseEvent) {
        if (!isResizing) {
          // Cursor feedback on hover
          container.style.cursor = isNearResizeHandle(e) ? 'nwse-resize' : '';
          return;
        }
        if (!resizeTarget) return;
        const zoom = cy.zoom();
        const deltaX = (e.clientX - resizeStartPos.x) / zoom;
        const deltaY = (e.clientY - resizeStartPos.y) / zoom;
        // 2x delta because width/height expand from center (half goes each direction)
        const newW = Math.max(80, resizeStartDims.w + 2 * deltaX);
        const newH = Math.max(50, resizeStartDims.h + 2 * deltaY);

        if (resizeTarget.isParent()) {
          resizeTarget.data('minWidth', newW);
          resizeTarget.data('minHeight', newH);
        } else {
          resizeTarget.style({ width: newW, height: newH });
          // Shift position to anchor top-left corner
          resizeTarget.position({
            x: resizeStartCenter.x + (newW - resizeStartDims.w) / 2,
            y: resizeStartCenter.y + (newH - resizeStartDims.h) / 2,
          });
        }
        const bb = resizeTarget.boundingBox({});
        resizeHandle.position({ x: bb.x2, y: bb.y2 });
      }

      function onResizeEnd() {
        if (!isResizing || !resizeTarget) return;
        isResizing = false;
        container.style.cursor = '';
        cy.panningEnabled(true);
        cy.boxSelectionEnabled(true);
        resizeTarget.grabify();
        resizeTarget.removeClass('resizing');

        const bb = resizeTarget.boundingBox({});
        const finalW = bb.x2 - bb.x1;
        const finalH = bb.y2 - bb.y1;
        callbacksRef.current.onNodeResize?.(resizeTarget.id(), finalW, finalH);

        resizeHandle.position({ x: bb.x2, y: bb.y2 });
      }

      container.addEventListener('mousedown', onResizeMouseDown);
      container.addEventListener('mousemove', onResizeMouseMove);
      container.addEventListener('mouseup', onResizeEnd);
      container.addEventListener('mouseleave', onResizeEnd);

      return () => {
        if (hideTimeout) clearTimeout(hideTimeout);
        container.removeEventListener('mousedown', onResizeMouseDown);
        container.removeEventListener('mousemove', onResizeMouseMove);
        container.removeEventListener('mouseup', onResizeEnd);
        container.removeEventListener('mouseleave', onResizeEnd);
        eh.destroy();
        cy.destroy();
        cyRef.current = null;
        ehRef.current = null;
        initialLayoutDone.current = false;
      };
    // eslint-disable-next-line react-hooks/exhaustive-deps
    }, []);

    // Sync elements when data changes
    useEffect(() => {
      const cy = cyRef.current;
      if (!cy) return;

      const hasPositions = nodes.some((n) => n.positionX !== 0 || n.positionY !== 0);

      cy.batch(() => {
        const newNodeIds = new Set(nodes.map((n) => n.id));
        const newEdgeIds = new Set(edges.map((e) => e.id));

        // Remove stale elements (skip handle node)
        cy.nodes().forEach((n) => {
          if (n.id() === HANDLE_ID || n.id() === RESIZE_HANDLE_ID) return;
          if (!newNodeIds.has(n.id())) n.remove();
        });
        cy.edges().forEach((e) => {
          if (!newEdgeIds.has(e.id())) e.remove();
        });

        // Add or update nodes (parents first so compound structure works)
        const sorted = [...nodes].sort((a, b) => {
          if (!a.parentId && b.parentId) return -1;
          if (a.parentId && !b.parentId) return 1;
          return 0;
        });

        sorted.forEach((node) => {
          const existing = cy.getElementById(node.id);
          const data: Record<string, unknown> = {
            id: node.id,
            label: node.title,
            nodeType: node.type,
            status: node.status,
          };
          if (node.type === 'GROUP') {
            const childCount = nodes.filter((n) => n.parentId === node.id).length;
            data.childCount = childCount;
            if (node.width && node.height) {
              data.minWidth = node.width;
              data.minHeight = node.height;
            }
          }

          if (existing.length) {
            existing.data(data);
            // Sync parent via move() — data() is unreliable for topology changes within batch
            const currentParent = existing.data('parent') || null;
            const newParent = node.parentId || null;
            if (currentParent !== newParent) {
              existing.move({ parent: newParent });
            }
            if (node.type === 'GROUP' && node.width && node.height && !existing.isParent()) {
              existing.style({ width: node.width, height: node.height });
            }
          } else {
            // New nodes: include parent in data for cy.add()
            const addData = { ...data };
            if (node.parentId) {
              addData.parent = node.parentId;
            }
            cy.add({
              group: 'nodes',
              data: addData,
              position: hasPositions
                ? { x: node.positionX, y: node.positionY }
                : undefined,
            });
            if (node.type === 'GROUP' && node.width && node.height) {
              const added = cy.getElementById(node.id);
              if (!added.isParent()) {
                added.style({ width: node.width, height: node.height });
              }
            }
          }
        });

        // Add or update edges
        edges.forEach((edge) => {
          const existing = cy.getElementById(edge.id);
          const data = {
            id: edge.id,
            source: edge.sourceId,
            target: edge.targetId,
            label: edge.label ?? '',
            edgeType: edge.edgeType,
          };

          if (existing.length) {
            existing.data(data);
          } else {
            cy.add({ group: 'edges', data });
          }
        });
      });

      // Apply collapse state with child count label
      cy.nodes('[nodeType="GROUP"]').forEach((g) => {
        const childCount = g.data('childCount') ?? 0;
        if (collapsedGroupIds.includes(g.id())) {
          g.addClass('group-collapsed');
          g.data('label', `${g.data('label').replace(/ \(\d+\)$/, '')} (${childCount})`);
        } else {
          g.removeClass('group-collapsed');
          g.data('label', g.data('label').replace(/ \(\d+\)$/, ''));
        }
      });
      cy.nodes().forEach((n) => {
        const parentId = n.data('parent');
        if (parentId && collapsedGroupIds.includes(parentId)) {
          n.addClass('collapsed-child');
        } else {
          n.removeClass('collapsed-child');
        }
      });

      // Run layout on first load or if no saved positions
      if (!initialLayoutDone.current && nodes.length > 0) {
        if (hasPositions) {
          cy.layout(getPresetLayout()).run();
        } else {
          cy.layout(getFcoseLayout()).run();
        }
        initialLayoutDone.current = true;
      }
    }, [nodes, edges, collapsedGroupIds]);

    return (
      <div
        ref={containerRef}
        className="h-full w-full"
        style={{ minHeight: '400px' }}
      />
    );
  },
);
