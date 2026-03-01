export function getFcoseLayout() {
  return {
    name: 'fcose' as const,
    quality: 'default' as const,
    randomize: true,
    animate: true,
    animationDuration: 500,
    fit: true,
    padding: 50,
    nodeRepulsion: 8000,
    idealEdgeLength: 120,
    edgeElasticity: 0.45,
    nestingFactor: 0.1,
    gravity: 0.25,
    gravityRange: 3.8,
    gravityCompound: 1.0,
    gravityRangeCompound: 1.5,
    numIter: 2500,
    tile: true,
  };
}

export function getPresetLayout() {
  return {
    name: 'preset' as const,
    fit: true,
    padding: 50,
  };
}
