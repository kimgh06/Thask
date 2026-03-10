declare module 'cytoscape-edgehandles' {
  import cytoscape from 'cytoscape';

  interface EdgeHandlesOptions {
    canConnect?: (sourceNode: cytoscape.NodeSingular, targetNode: cytoscape.NodeSingular) => boolean;
    edgeParams?: (
      sourceNode: cytoscape.NodeSingular,
      targetNode: cytoscape.NodeSingular,
    ) => Record<string, unknown>;
    hoverDelay?: number;
    snap?: boolean;
    snapThreshold?: number;
    snapFrequency?: number;
    noEdgeEventsInDraw?: boolean;
    disableBrowserGestures?: boolean;
    handleNodes?: string;
  }

  interface EdgeHandlesInstance {
    active: boolean;
    enable(): void;
    disable(): void;
    enableDrawMode(): void;
    disableDrawMode(): void;
    canStartOn(node: cytoscape.NodeSingular): boolean;
    start(sourceNode: cytoscape.NodeSingular): void;
    stop(): void;
    destroy(): void;
  }

  const edgehandles: cytoscape.Ext;
  export default edgehandles;

  declare global {
    namespace cytoscape {
      interface Core {
        edgehandles(options?: EdgeHandlesOptions): EdgeHandlesInstance;
      }
    }
  }
}
