'use client';

import { useEffect, useRef, useCallback } from 'react';
import type { Core } from 'cytoscape';

const HANDLE_IDS = new Set(['__eh-handle__', '__resize-handle__']);

const NODE_COLORS: Record<string, string> = {
  FLOW: '#3b82f6',
  BRANCH: '#8b5cf6',
  TASK: '#06b6d4',
  BUG: '#ef4444',
  API: '#f97316',
  UI: '#10b981',
  GROUP: '#64748b',
};

interface MinimapTransform {
  scale: number;
  offsetX: number;
  offsetY: number;
}

interface GraphMinimapProps {
  getCy: () => Core | null;
  width?: number;
  height?: number;
}

export function GraphMinimap({ getCy, width = 200, height = 150 }: GraphMinimapProps) {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const animFrameRef = useRef(0);
  const isDraggingRef = useRef(false);
  const transformRef = useRef<MinimapTransform | null>(null);

  const draw = useCallback(() => {
    const cy = getCy();
    const canvas = canvasRef.current;
    if (!cy || !canvas) return;

    const ctx = canvas.getContext('2d');
    if (!ctx) return;

    const dpr = window.devicePixelRatio || 1;
    canvas.width = width * dpr;
    canvas.height = height * dpr;
    ctx.scale(dpr, dpr);

    // Background
    ctx.fillStyle = '#f8fafc';
    ctx.fillRect(0, 0, width, height);

    // Visible nodes only (exclude handles, collapsed children, filtered nodes)
    const visibleNodes = cy.nodes().filter(
      (n) =>
        !HANDLE_IDS.has(n.id()) &&
        !n.hasClass('collapsed-child') &&
        !n.hasClass('filter-hidden') &&
        n.style('display') !== 'none',
    );

    if (visibleNodes.length === 0) {
      transformRef.current = null;
      return;
    }

    // Bounding box of all visible nodes
    const bb = visibleNodes.boundingBox({});
    const pad = 30;
    const graphW = bb.w + pad * 2;
    const graphH = bb.h + pad * 2;

    const scale = Math.min(width / graphW, height / graphH);
    const offsetX = (width - graphW * scale) / 2 - (bb.x1 - pad) * scale;
    const offsetY = (height - graphH * scale) / 2 - (bb.y1 - pad) * scale;
    transformRef.current = { scale, offsetX, offsetY };

    // Draw edges (skip collapsed/filtered)
    cy.edges().forEach((edge) => {
      if (edge.hasClass('collapsed-edge')) return;
      const src = edge.source();
      const tgt = edge.target();
      if (src.hasClass('collapsed-child') || tgt.hasClass('collapsed-child')) return;
      if (src.hasClass('filter-hidden') || tgt.hasClass('filter-hidden')) return;
      if (HANDLE_IDS.has(src.id()) || HANDLE_IDS.has(tgt.id())) return;

      const sp = src.position();
      const tp = tgt.position();
      ctx.strokeStyle = '#cbd5e1';
      ctx.lineWidth = 0.5;
      ctx.beginPath();
      ctx.moveTo(sp.x * scale + offsetX, sp.y * scale + offsetY);
      ctx.lineTo(tp.x * scale + offsetX, tp.y * scale + offsetY);
      ctx.stroke();
    });

    // Draw nodes
    visibleNodes.forEach((node) => {
      const pos = node.position();
      const nx = pos.x * scale + offsetX;
      const ny = pos.y * scale + offsetY;
      const nw = Math.max(8, node.width() * scale);
      const nh = Math.max(8, node.height() * scale);
      const nodeType = node.data('nodeType') as string;

      if (nodeType === 'GROUP') {
        // GROUP: dashed outline only (matches main graph style)
        ctx.strokeStyle = NODE_COLORS.GROUP ?? '#64748b';
        ctx.setLineDash([3, 2]);
        ctx.lineWidth = 1;
        ctx.strokeRect(nx - nw / 2, ny - nh / 2, nw, nh);
        ctx.setLineDash([]);
      } else {
        // Regular nodes: filled rounded rectangle
        ctx.fillStyle = NODE_COLORS[nodeType] ?? '#94a3b8';
        ctx.beginPath();
        ctx.roundRect(nx - nw / 2, ny - nh / 2, nw, nh, 2);
        ctx.fill();
      }
    });

    // Draw viewport rectangle with clamping
    const ext = cy.extent();
    const rawVx = ext.x1 * scale + offsetX;
    const rawVy = ext.y1 * scale + offsetY;
    const rawVw = (ext.x2 - ext.x1) * scale;
    const rawVh = (ext.y2 - ext.y1) * scale;

    const vx = Math.max(0, rawVx);
    const vy = Math.max(0, rawVy);
    const vw = Math.min(width - vx, rawVw - (vx - rawVx));
    const vh = Math.min(height - vy, rawVh - (vy - rawVy));

    if (vw > 0 && vh > 0) {
      ctx.strokeStyle = '#3b82f6';
      ctx.lineWidth = 2;
      ctx.strokeRect(vx, vy, vw, vh);
      ctx.fillStyle = 'rgba(59, 130, 246, 0.12)';
      ctx.fillRect(vx, vy, vw, vh);
    }
  }, [getCy, width, height]);

  // Bind cy events with polling for cy readiness
  useEffect(() => {
    let pollTimer: ReturnType<typeof setInterval> | null = null;
    let cleanup: (() => void) | null = null;

    function bindAndDraw(cy: Core) {
      function scheduleRedraw() {
        cancelAnimationFrame(animFrameRef.current);
        animFrameRef.current = requestAnimationFrame(draw);
      }

      cy.on('viewport add remove position layoutstop', scheduleRedraw);
      scheduleRedraw();

      cleanup = () => {
        cy.off('viewport add remove position layoutstop', scheduleRedraw);
        cancelAnimationFrame(animFrameRef.current);
      };
    }

    const cy = getCy();
    if (cy) {
      bindAndDraw(cy);
    } else {
      // Poll until cy is ready
      pollTimer = setInterval(() => {
        const ready = getCy();
        if (ready) {
          if (pollTimer) clearInterval(pollTimer);
          pollTimer = null;
          bindAndDraw(ready);
        }
      }, 100);
    }

    return () => {
      if (pollTimer) clearInterval(pollTimer);
      cleanup?.();
    };
  }, [getCy, draw]);

  const handlePointerEvent = useCallback(
    (e: React.MouseEvent<HTMLCanvasElement>) => {
      const cy = getCy();
      const canvas = canvasRef.current;
      const t = transformRef.current;
      if (!cy || !canvas || !t) return;

      const rect = canvas.getBoundingClientRect();
      const mx = e.clientX - rect.left;
      const my = e.clientY - rect.top;

      // Convert minimap coords → graph coords
      const graphX = (mx - t.offsetX) / t.scale;
      const graphY = (my - t.offsetY) / t.scale;

      cy.pan({
        x: cy.width() / 2 - graphX * cy.zoom(),
        y: cy.height() / 2 - graphY * cy.zoom(),
      });
    },
    [getCy],
  );

  return (
    <div className="absolute bottom-4 right-4 z-10 overflow-hidden rounded-lg border border-gray-200 bg-white shadow-md">
      <canvas
        ref={canvasRef}
        style={{ width, height, cursor: 'crosshair', display: 'block' }}
        onMouseDown={(e) => {
          isDraggingRef.current = true;
          handlePointerEvent(e);
        }}
        onMouseMove={(e) => {
          if (isDraggingRef.current) handlePointerEvent(e);
        }}
        onMouseUp={() => { isDraggingRef.current = false; }}
        onMouseLeave={() => { isDraggingRef.current = false; }}
      />
    </div>
  );
}
