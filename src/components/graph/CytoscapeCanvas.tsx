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
  focusNode: (nodeId: string) => void;
}

interface CytoscapeCanvasProps {
  nodes: GraphNode[];
  edges: GraphEdge[];
  collapsedGroupIds?: string[];
  selectedNodeIds?: string[];
  onNodeSelect?: (nodeId: string | null) => void;
  onNodeToggleSelect?: (nodeId: string) => void;
  onNodeDragEnd?: (positions: Array<{ id: string; x: number; y: number }>) => void;
  onConnectEnd?: (sourceId: string, targetId: string, renderedPosition: { x: number; y: number }) => void;
  onEdgeClick?: (edgeId: string, renderedPosition: { x: number; y: number }) => void;
  onNodeDropOnGroup?: (nodeId: string, groupId: string | null) => void;
  onToggleGroupCollapse?: (groupId: string) => void;
  onNodeResize?: (nodeId: string, width: number, height: number) => void;
}

export const CytoscapeCanvas = forwardRef<CytoscapeCanvasHandle, CytoscapeCanvasProps>(
  function CytoscapeCanvas(
    { nodes, edges, collapsedGroupIds = [], selectedNodeIds = [], onNodeSelect, onNodeToggleSelect, onNodeDragEnd, onConnectEnd, onEdgeClick, onNodeDropOnGroup, onToggleGroupCollapse, onNodeResize },
    ref,
  ) {
    const containerRef = useRef<HTMLDivElement>(null);
    const cyRef = useRef<Core | null>(null);
    const ehRef = useRef<ReturnType<Core['edgehandles']> | null>(null);
    const initialLayoutDone = useRef(false);

    // Store latest callbacks in refs
    const callbacksRef = useRef({ onNodeSelect, onNodeToggleSelect, onNodeDragEnd, onConnectEnd, onEdgeClick, onNodeDropOnGroup, onToggleGroupCollapse, onNodeResize });
    callbacksRef.current = { onNodeSelect, onNodeToggleSelect, onNodeDragEnd, onConnectEnd, onEdgeClick, onNodeDropOnGroup, onToggleGroupCollapse, onNodeResize };

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
      focusNode: (nodeId: string) => {
        const cy = cyRef.current;
        if (!cy) return;
        const node = cy.getElementById(nodeId);
        if (!node.length) return;
        cy.animate({ center: { eles: node }, zoom: 1.5 }, { duration: 400 });
        cy.nodes().removeClass('search-highlight');
        node.addClass('search-highlight');
        setTimeout(() => node.removeClass('search-highlight'), 2000);
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
      // Allow edge creation from compound parent nodes (GROUP with children)
      // Default canStartOn blocks isParent() nodes
      (eh as ReturnType<Core['edgehandles']> & { canStartOn: (n: cytoscape.NodeSingular) => boolean }).canStartOn =
        (node) => node.isNode() && node.id() !== HANDLE_ID && node.id() !== RESIZE_HANDLE_ID;
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
        if (eh.active) return;
        if (node.grabbed()) return;
        if (isResizing) return;
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
        if (hoverSource) {
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

      // Node selection (Shift+click = toggle multi-select)
      cy.on('tap', 'node', (evt: EventObject) => {
        if (evt.target.id() === HANDLE_ID || evt.target.id() === RESIZE_HANDLE_ID) return;
        const cb = callbacksRef.current;
        if (evt.originalEvent.shiftKey) {
          cb.onNodeToggleSelect?.(evt.target.id());
        } else {
          cb.onNodeSelect?.(evt.target.id());
        }
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
        const cursorPos = evt.position;
        // Find the innermost (smallest) GROUP under cursor for nested GROUP support
        let innerTarget: string | null = null;
        let innerTargetArea = Infinity;
        cy.nodes('[nodeType="GROUP"]').forEach((g) => {
          g.removeClass('drop-target');
          if (g.id() === node.id() || g.hasClass('group-collapsed') || dragDescendantIds?.has(g.id())) return;
          const bb = g.boundingBox({});
          if (cursorPos.x >= bb.x1 && cursorPos.x <= bb.x2 && cursorPos.y >= bb.y1 && cursorPos.y <= bb.y2) {
            const area = (bb.x2 - bb.x1) * (bb.y2 - bb.y1);
            if (area < innerTargetArea) {
              innerTarget = g.id();
              innerTargetArea = area;
            }
          }
        });
        if (innerTarget) {
          cy.getElementById(innerTarget).addClass('drop-target');
        }
      });

      cy.on('dragfree', 'node', (evt: EventObject) => {
        if (evt.target.id() === HANDLE_ID || evt.target.id() === RESIZE_HANDLE_ID) return;
        cy.nodes('.drop-target').removeClass('drop-target');
        const cb = callbacksRef.current;
        const node = evt.target;

        // Collect positions: compound parent → all descendants; otherwise → single node
        const positionsToSave: Array<{ id: string; x: number; y: number }> = [];
        if (node.isParent()) {
          node.descendants().forEach((d: cytoscape.NodeSingular) => {
            const p = d.position();
            positionsToSave.push({ id: d.id(), x: p.x, y: p.y });
          });
        } else {
          const p = node.position();
          positionsToSave.push({ id: node.id(), x: p.x, y: p.y });
        }
        if (positionsToSave.length > 0) {
          cb.onNodeDragEnd?.(positionsToSave);
        }

        // Find GROUP node under cursor drop position (skip self, collapsed, descendants)
        const dropPos = evt.position;
        const groupNodes = cy.nodes().filter(
          (n) => n.data('nodeType') === 'GROUP' && n.id() !== node.id() && n.id() !== HANDLE_ID && !n.hasClass('group-collapsed') && !dragDescendantIds?.has(n.id()),
        );
        let targetGroup: string | null = null;
        let targetGroupArea = Infinity;
        groupNodes.forEach((g) => {
          const bb = g.boundingBox({});
          if (dropPos.x >= bb.x1 && dropPos.x <= bb.x2 && dropPos.y >= bb.y1 && dropPos.y <= bb.y2) {
            const area = (bb.x2 - bb.x1) * (bb.y2 - bb.y1);
            if (area < targetGroupArea) {
              targetGroup = g.id();
              targetGroupArea = area;
            }
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

      // Temporary unparent state — keeps GROUP fixed while child is dragged
      let dragOriginalParent: string | null = null;
      let dragOriginalMinDims: { w: number | undefined; h: number | undefined } | null = null;
      let dragDescendantIds: Set<string> | null = null;

      // --- 8-directional resize for GROUP nodes ---
      type ResizeZone = 'n' | 's' | 'e' | 'w' | 'ne' | 'nw' | 'se' | 'sw';
      const ZONE_CURSORS: Record<ResizeZone, string> = {
        nw: 'nwse-resize', se: 'nwse-resize',
        ne: 'nesw-resize', sw: 'nesw-resize',
        n: 'ns-resize', s: 'ns-resize',
        e: 'ew-resize', w: 'ew-resize',
      };

      let resizeTarget: cytoscape.NodeSingular | null = null;
      let isResizing = false;
      let resizeZone: ResizeZone | null = null;
      let resizeStartPos = { x: 0, y: 0 };
      let resizeStartBB = { x1: 0, y1: 0, x2: 0, y2: 0 };

      function detectResizeZone(e: MouseEvent, node: cytoscape.NodeSingular): ResizeZone | null {
        if (node.hasClass('group-collapsed') || node.hasClass('filter-hidden')) return null;
        const bb = node.renderedBoundingBox({});
        const mx = e.offsetX;
        const my = e.offsetY;
        const T = 10;

        const inX = mx >= bb.x1 - T && mx <= bb.x2 + T;
        const inY = my >= bb.y1 - T && my <= bb.y2 + T;
        if (!inX || !inY) return null;

        const nearTop = Math.abs(my - bb.y1) <= T;
        const nearBottom = Math.abs(my - bb.y2) <= T;
        const nearLeft = Math.abs(mx - bb.x1) <= T;
        const nearRight = Math.abs(mx - bb.x2) <= T;

        if (nearTop && nearLeft) return 'nw';
        if (nearTop && nearRight) return 'ne';
        if (nearBottom && nearLeft) return 'sw';
        if (nearBottom && nearRight) return 'se';
        if (nearTop) return 'n';
        if (nearBottom) return 's';
        if (nearLeft) return 'w';
        if (nearRight) return 'e';
        return null;
      }

      const container = cy.container()!;

      function onResizeMouseDown(e: MouseEvent) {
        if (!resizeTarget || eh.active) return;
        const zone = detectResizeZone(e, resizeTarget);
        if (!zone) return;

        e.preventDefault();
        e.stopPropagation();
        isResizing = true;
        resizeZone = zone;
        container.style.cursor = ZONE_CURSORS[zone];
        cy.panningEnabled(false);
        cy.boxSelectionEnabled(false);
        resizeTarget.ungrabify();
        resizeTarget.addClass('resizing');

        const bb = resizeTarget.boundingBox({});
        resizeStartBB = { x1: bb.x1, y1: bb.y1, x2: bb.x2, y2: bb.y2 };
        resizeStartPos = { x: e.clientX, y: e.clientY };
      }

      function onResizeMouseMove(e: MouseEvent) {
        if (!isResizing) {
          if (eh.active) return;
          // Scan all visible GROUP nodes for resize zone proximity
          let foundTarget: cytoscape.NodeSingular | null = null;
          let foundZone: ResizeZone | null = null;
          cy.nodes('[nodeType="GROUP"]').forEach((g) => {
            if (foundZone) return;
            const zone = detectResizeZone(e, g);
            if (zone) {
              foundTarget = g;
              foundZone = zone;
            }
          });
          resizeTarget = foundTarget;
          container.style.cursor = foundZone ? ZONE_CURSORS[foundZone] : '';
          return;
        }
        if (!resizeTarget || !resizeZone) return;

        const zoom = cy.zoom();
        const deltaX = (e.clientX - resizeStartPos.x) / zoom;
        const deltaY = (e.clientY - resizeStartPos.y) / zoom;

        const moveLeft = resizeZone.includes('w');
        const moveRight = resizeZone.includes('e');
        const moveTop = resizeZone.includes('n');
        const moveBottom = resizeZone.includes('s');

        let { x1, y1, x2, y2 } = resizeStartBB;
        if (moveLeft) x1 += deltaX;
        if (moveRight) x2 += deltaX;
        if (moveTop) y1 += deltaY;
        if (moveBottom) y2 += deltaY;

        // Enforce minimum dimensions
        const MIN_W = 80;
        const MIN_H = 50;
        if (x2 - x1 < MIN_W) {
          if (moveLeft) x1 = x2 - MIN_W;
          else x2 = x1 + MIN_W;
        }
        if (y2 - y1 < MIN_H) {
          if (moveTop) y1 = y2 - MIN_H;
          else y2 = y1 + MIN_H;
        }

        const newW = x2 - x1;
        const newH = y2 - y1;
        const newCenterX = (x1 + x2) / 2;
        const newCenterY = (y1 + y2) / 2;

        if (resizeTarget.isParent()) {
          resizeTarget.data('minWidth', newW);
          resizeTarget.data('minHeight', newH);
          // Shift children so parent bounding box matches desired position
          const actualBB = resizeTarget.boundingBox({});
          const shiftX = newCenterX - (actualBB.x1 + actualBB.x2) / 2;
          const shiftY = newCenterY - (actualBB.y1 + actualBB.y2) / 2;
          if (Math.abs(shiftX) > 0.5 || Math.abs(shiftY) > 0.5) {
            resizeTarget.children().forEach((child) => {
              const pos = child.position();
              child.position({ x: pos.x + shiftX, y: pos.y + shiftY });
            });
          }
        } else {
          resizeTarget.style({ width: newW, height: newH });
          resizeTarget.position({ x: newCenterX, y: newCenterY });
        }
      }

      function onResizeEnd() {
        if (!isResizing || !resizeTarget) return;
        isResizing = false;
        resizeZone = null;
        container.style.cursor = '';
        cy.panningEnabled(true);
        cy.boxSelectionEnabled(true);
        resizeTarget.grabify();
        resizeTarget.removeClass('resizing');

        const bb = resizeTarget.boundingBox({});
        const finalW = bb.x2 - bb.x1;
        const finalH = bb.y2 - bb.y1;
        callbacksRef.current.onNodeResize?.(resizeTarget.id(), finalW, finalH);
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
        const title = g.data('title') as string ?? '';
        if (collapsedGroupIds.includes(g.id())) {
          g.addClass('group-collapsed');
          g.data('label', childCount > 0 ? `${title} (${childCount})` : title);
        } else {
          g.removeClass('group-collapsed');
          g.data('label', title);
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
      // Hide/show edges connected to collapsed children
      cy.edges().forEach((e) => {
        if (e.source().hasClass('collapsed-child') || e.target().hasClass('collapsed-child')) {
          e.addClass('collapsed-edge');
        } else {
          e.removeClass('collapsed-edge');
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

    // Apply multi-select highlighting
    useEffect(() => {
      const cy = cyRef.current;
      if (!cy) return;
      cy.nodes().removeClass('multi-selected');
      if (selectedNodeIds.length > 1) {
        selectedNodeIds.forEach((id) => {
          const node = cy.getElementById(id);
          if (node.length) node.addClass('multi-selected');
        });
      }
    }, [selectedNodeIds]);

    return (
      <div
        ref={containerRef}
        className="h-full w-full"
        style={{ minHeight: '400px' }}
      />
    );
  },
);
