'use client';

import { useEffect, useRef, useCallback, forwardRef, useImperativeHandle } from 'react';
import cytoscape, { type Core, type EventObject } from 'cytoscape';
import fcose from 'cytoscape-fcose';
import edgehandles from 'cytoscape-edgehandles';
import { cytoscapeStylesheet } from '@/lib/cytoscape/styles';
import { getFcoseLayout, getPresetLayout } from '@/lib/cytoscape/layouts';
import { getChildNodes, getDescendantNodes, getDescendantIdSet } from '@/lib/cytoscape/groupHelpers';
import type { GraphNode, GraphEdge } from '@/types/graph';

// Register extensions once
let extensionsRegistered = false;
if (!extensionsRegistered) {
  cytoscape.use(fcose);
  cytoscape.use(edgehandles);
  extensionsRegistered = true;
}

const PORT_SIZE = 20;

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
    const portOverlayRef = useRef<HTMLDivElement>(null);
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
        canConnect: (sourceNode, targetNode) => !sourceNode.same(targetNode),
        edgeParams: () => ({}),
        hoverDelay: 0,
        snap: true,
        snapThreshold: 50,
        snapFrequency: 15,
        noEdgeEventsInDraw: true,
        disableBrowserGestures: true,
        handleNodes: 'DONOTMATCHANYTHING',
      });
      eh.enable();
      ehRef.current = eh;

      // --- HTML port overlay for edge creation ---
      const overlay = portOverlayRef.current;
      let portSource: cytoscape.NodeSingular | null = null;
      let portHideTimer: ReturnType<typeof setTimeout> | null = null;

      function showPorts(node: cytoscape.NodeSingular) {
        if (!overlay) return;
        portSource = node;
        if (portHideTimer) { clearTimeout(portHideTimer); portHideTimer = null; }
        positionPorts(node);
        overlay.style.display = 'block';
      }

      function hidePorts(delay = 150) {
        portHideTimer = setTimeout(() => {
          if (overlay) overlay.style.display = 'none';
          portSource = null;
        }, delay);
      }

      function positionPorts(node: cytoscape.NodeSingular) {
        if (!overlay) return;
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
          const el = overlay.querySelector(`.${p.cls}`) as HTMLElement | null;
          if (el) {
            el.style.left = `${p.x}px`;
            el.style.top = `${p.y}px`;
          }
        }
      }

      // Port mousedown: start edge drawing
      if (overlay) {
        overlay.addEventListener('mousedown', (e: MouseEvent) => {
          if (!(e.target as HTMLElement).classList.contains('port-dot')) return;
          e.preventDefault();
          e.stopPropagation();
          if (portSource) {
            if (overlay) overlay.style.display = 'none';
            eh.start(portSource);
            // Cytoscape never saw tapstart, so tapend won't fire.
            // Listen for mouseup to manually stop edgehandles.
            const onMouseUp = () => {
              document.removeEventListener('mouseup', onMouseUp);
              eh.stop();
            };
            document.addEventListener('mouseup', onMouseUp);
          }
        });
        // Keep ports visible while hovering overlay
        overlay.addEventListener('mouseenter', () => {
          if (portHideTimer) { clearTimeout(portHideTimer); portHideTimer = null; }
        });
        overlay.addEventListener('mouseleave', () => {
          if (!eh.active) hidePorts(100);
        });
      }

      // Show ports when hovering a node
      cy.on('mouseover', 'node', (e) => {
        const node = e.target;
        if (eh.active || node.grabbed() || isResizing) return;
        if (node.data('nodeType') === 'GROUP' && !node.hasClass('group-collapsed')) return;
        showPorts(node);
      });

      cy.on('mouseout', 'node', () => {
        if (!eh.active) hidePorts();
      });

      // Reposition ports on pan/zoom
      cy.on('pan zoom', () => {
        if (portSource && overlay?.style.display === 'block') positionPorts(portSource);
      });

      // After edge drawing ends, clean up
      cy.on('ehstop ehcancel', () => {
        if (overlay) overlay.style.display = 'none';
        portSource = null;
      });

      // Hide ports when node drag starts + setup GROUP drag state
      cy.on('grab', 'node', (e) => {
        const node = e.target;
        if (overlay) overlay.style.display = 'none';
        portSource = null;

        if (node.data('nodeType') === 'GROUP') {
          // GROUP grabbed: collect all descendants + their position offsets
          const groupPos = node.position();
          const descendants = getDescendantNodes(cy, node.id());
          const childOffsets = new Map<string, { dx: number; dy: number }>();
          descendants.forEach((d: cytoscape.NodeSingular) => {
            const dPos = d.position();
            childOffsets.set(d.id(), {
              dx: dPos.x - groupPos.x,
              dy: dPos.y - groupPos.y,
            });
          });
          groupDragState = { groupId: node.id(), childOffsets };
          dragDescendantIds = getDescendantIdSet(cy, node.id());
        } else {
          groupDragState = null;
          dragDescendantIds = null;
        }
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

      // Highlight GROUP nodes when dragging a node over them + move GROUP children in lockstep
      cy.on('drag', 'node', (evt: EventObject) => {
        const node = evt.target;
        const cursorPos = evt.position;

        // Drop-target highlight: find innermost (smallest) GROUP under cursor
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

        // GROUP drag: move all descendants in lockstep
        if (groupDragState && groupDragState.groupId === node.id()) {
          const groupPos = node.position();
          groupDragState.childOffsets.forEach((offset, childId) => {
            const child = cy.getElementById(childId);
            if (child.length) {
              child.position({
                x: groupPos.x + offset.dx,
                y: groupPos.y + offset.dy,
              });
            }
          });
        }
      });

      cy.on('dragfree', 'node', (evt: EventObject) => {
        cy.nodes('.drop-target').removeClass('drop-target');
        const cb = callbacksRef.current;
        const node = evt.target;
        const positionsToSave: Array<{ id: string; x: number; y: number }> = [];

        if (groupDragState && groupDragState.groupId === node.id()) {
          // GROUP was dragged: save GROUP + all descendant positions
          const gPos = node.position();
          positionsToSave.push({ id: node.id(), x: gPos.x, y: gPos.y });
          groupDragState.childOffsets.forEach((_offset, childId) => {
            const child = cy.getElementById(childId);
            if (child.length) {
              const p = child.position();
              positionsToSave.push({ id: childId, x: p.x, y: p.y });
            }
          });
          groupDragState = null;
        } else {
          // Regular node (or child of GROUP) dragged
          const dropPos = evt.position;

          // Find innermost GROUP under drop position
          const groupNodes = cy.nodes().filter(
            (n) => n.data('nodeType') === 'GROUP' && n.id() !== node.id() && !n.hasClass('group-collapsed') && !dragDescendantIds?.has(n.id()),
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

          const currentParentId = (node.data('parentId') as string) || null;

          if (currentParentId) {
            // Node has a parent GROUP — check if still inside bounds
            const parentNode = cy.getElementById(currentParentId);
            if (parentNode.length && parentNode.data('nodeType') === 'GROUP' && !parentNode.hasClass('group-collapsed')) {
              const bb = parentNode.boundingBox({});
              const nodePos = node.position();
              const isInside = nodePos.x >= bb.x1 && nodePos.x <= bb.x2 && nodePos.y >= bb.y1 && nodePos.y <= bb.y2;
              if (!isInside) {
                // Dragged outside parent → detach or reparent
                cb.onNodeDropOnGroup?.(node.id(), targetGroup);
              } else if (targetGroup && targetGroup !== currentParentId) {
                // Dropped inside a different GROUP
                cb.onNodeDropOnGroup?.(node.id(), targetGroup);
              }
            }
          } else {
            // Node has no parent — check if dropped on a GROUP
            if (targetGroup) {
              const group = cy.getElementById(targetGroup);
              const groupBB = group.boundingBox({});
              const nodePos = node.position();
              const pad = 25;
              const nw = node.width() / 2;
              const nh = node.height() / 2;
              const fitsInside =
                nodePos.x - nw >= groupBB.x1 + pad &&
                nodePos.x + nw <= groupBB.x2 - pad &&
                nodePos.y - nh >= groupBB.y1 + pad &&
                nodePos.y + nh <= groupBB.y2 - pad;
              if (fitsInside) {
                const clampedX = Math.max(groupBB.x1 + pad + nw, Math.min(groupBB.x2 - pad - nw, nodePos.x));
                const clampedY = Math.max(groupBB.y1 + pad + nh, Math.min(groupBB.y2 - pad - nh, nodePos.y));
                node.position({ x: clampedX, y: clampedY });
              }
              cb.onNodeDropOnGroup?.(node.id(), targetGroup);
            }
          }

          // Save dragged node position
          const p = node.position();
          positionsToSave.push({ id: node.id(), x: p.x, y: p.y });
        }

        dragDescendantIds = null;
        if (positionsToSave.length > 0) {
          cb.onNodeDragEnd?.(positionsToSave);
        }
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

      // GROUP drag state: when a GROUP is grabbed, store descendants + position offsets
      let groupDragState: {
        groupId: string;
        childOffsets: Map<string, { dx: number; dy: number }>;
      } | null = null;
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

        const bb = resizeTarget.boundingBox({ includeLabels: false, includeOverlays: false });
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

        // Enforce minimum dimensions (dynamic: clamp to children bounding box)
        let minW = 80;
        let minH = 50;
        if (resizeTarget.data('nodeType') === 'GROUP' && !resizeTarget.hasClass('group-collapsed')) {
          const children = getChildNodes(cy, resizeTarget.id());
          if (children.length > 0) {
            const PAD = 30;
            let cxMin = Infinity, cyMin = Infinity, cxMax = -Infinity, cyMax = -Infinity;
            children.forEach((c: cytoscape.NodeSingular) => {
              const pos = c.position();
              const hw = c.width() / 2;
              const hh = c.height() / 2;
              cxMin = Math.min(cxMin, pos.x - hw);
              cyMin = Math.min(cyMin, pos.y - hh);
              cxMax = Math.max(cxMax, pos.x + hw);
              cyMax = Math.max(cyMax, pos.y + hh);
            });
            minW = Math.max(minW, cxMax - cxMin + PAD * 2);
            minH = Math.max(minH, cyMax - cyMin + PAD * 2);
          }
        }
        if (x2 - x1 < minW) {
          if (moveLeft) x1 = x2 - minW;
          else x2 = x1 + minW;
        }
        if (y2 - y1 < minH) {
          if (moveTop) y1 = y2 - minH;
          else y2 = y1 + minH;
        }

        const newW = x2 - x1;
        const newH = y2 - y1;
        const newCenterX = (x1 + x2) / 2;
        const newCenterY = (y1 + y2) / 2;

        // GROUP is a regular node with explicit width/height — set directly
        resizeTarget.data('width', newW);
        resizeTarget.data('height', newH);
        resizeTarget.position({ x: newCenterX, y: newCenterY });
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

        const finalW = resizeTarget.data('width') as number;
        const finalH = resizeTarget.data('height') as number;
        callbacksRef.current.onNodeResize?.(resizeTarget.id(), finalW, finalH);
      }

      container.addEventListener('mousedown', onResizeMouseDown);
      container.addEventListener('mousemove', onResizeMouseMove);
      container.addEventListener('mouseup', onResizeEnd);
      container.addEventListener('mouseleave', onResizeEnd);

      return () => {
        if (portHideTimer) clearTimeout(portHideTimer);
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

        // Remove stale elements
        cy.nodes().forEach((n) => {
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
            title: node.title,
            nodeType: node.type,
            status: node.status,
          };
          // Store parentId as regular data (NOT Cytoscape compound 'parent')
          data.parentId = node.parentId || null;

          if (node.type === 'GROUP') {
            const childCount = nodes.filter((n) => n.parentId === node.id).length;
            data.childCount = childCount;
            data.width = Math.max(node.width ?? 160, 160);
            data.height = Math.max(node.height ?? 100, 100);
          }

          if (existing.length) {
            existing.data(data);
          } else {
            cy.add({
              group: 'nodes',
              data: { ...data },
              position: hasPositions
                ? { x: node.positionX, y: node.positionY }
                : undefined,
            });
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
        const parentId = n.data('parentId') as string | null;
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
          const layout = cy.layout(getFcoseLayout());
          layout.on('layoutstop', () => {
            // After fcose: reposition GROUP nodes to encompass their children
            cy.nodes('[nodeType="GROUP"]').forEach((g) => {
              if (g.hasClass('group-collapsed')) return;
              const children = cy.nodes().filter((n: cytoscape.NodeSingular) => n.data('parentId') === g.id());
              if (children.length === 0) return;
              const PAD = 40;
              const NODE_HALF = 40;
              let minX = Infinity, minY = Infinity, maxX = -Infinity, maxY = -Infinity;
              children.forEach((c: cytoscape.NodeSingular) => {
                const pos = c.position();
                minX = Math.min(minX, pos.x - NODE_HALF);
                maxX = Math.max(maxX, pos.x + NODE_HALF);
                minY = Math.min(minY, pos.y - NODE_HALF);
                maxY = Math.max(maxY, pos.y + NODE_HALF);
              });
              const w = Math.max(160, maxX - minX + PAD * 2);
              const h = Math.max(100, maxY - minY + PAD * 2);
              g.data('width', w);
              g.data('height', h);
              g.position({ x: (minX + maxX) / 2, y: (minY + maxY) / 2 });
            });
          });
          layout.run();
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
      <div className="relative h-full w-full" style={{ minHeight: '400px' }}>
        <div
          ref={containerRef}
          className="h-full w-full"
        />
        {/* Port overlay for edge creation — 4 dots around hovered node */}
        <div
          ref={portOverlayRef}
          style={{ display: 'none', position: 'absolute', top: 0, left: 0, pointerEvents: 'none' }}
        >
          {['port-top', 'port-right', 'port-bottom', 'port-left'].map((cls) => (
            <div
              key={cls}
              className={`port-dot ${cls}`}
              style={{
                position: 'absolute',
                width: PORT_SIZE,
                height: PORT_SIZE,
                borderRadius: '50%',
                background: '#3b82f6',
                border: '2px solid #1d4ed8',
                cursor: 'crosshair',
                pointerEvents: 'auto',
                transition: 'transform 0.1s ease',
              }}
              onMouseEnter={(e) => {
                (e.currentTarget as HTMLElement).style.transform = 'scale(1.3)';
              }}
              onMouseLeave={(e) => {
                (e.currentTarget as HTMLElement).style.transform = 'scale(1)';
              }}
            />
          ))}
        </div>
      </div>
    );
  },
);
