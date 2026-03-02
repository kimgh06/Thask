'use client';

import { useState, type RefObject } from 'react';
import type { CytoscapeCanvasHandle } from '@/components/graph/CytoscapeCanvas';

interface EdgePopoverState {
  visible: boolean;
  position: { x: number; y: number };
  edgeId: string;
}

const INITIAL_STATE: EdgePopoverState = { visible: false, position: { x: 0, y: 0 }, edgeId: '' };

export function useEdgePopover(graphRef: RefObject<CytoscapeCanvasHandle | null>) {
  const [edgeColorPopover, setEdgeColorPopover] = useState<EdgePopoverState>(INITIAL_STATE);

  function showPopover(edgeId: string, renderedPosition: { x: number; y: number }) {
    const container = graphRef.current?.getCy()?.container();
    const rect = container?.getBoundingClientRect();
    const pageX = (rect?.left ?? 0) + renderedPosition.x;
    const pageY = (rect?.top ?? 0) + renderedPosition.y;

    setEdgeColorPopover({
      visible: true,
      position: { x: pageX + 10, y: pageY - 10 },
      edgeId,
    });
  }

  function hidePopover() {
    setEdgeColorPopover((p) => ({ ...p, visible: false }));
  }

  return { edgeColorPopover, showPopover, hidePopover };
}
