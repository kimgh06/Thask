export interface Point {
	x: number;
	y: number;
}

export interface ObstacleRect {
	x1: number;
	y1: number;
	x2: number;
	y2: number;
}

export interface GridRouteOptions {
	gridSize?: number;
	paddingCells?: number;
	maxExpansions?: number;
	bendPenalty?: number;
	backtrackPenalty?: number;
}

interface Cell {
	x: number;
	y: number;
}

interface SearchState extends Cell {
	dir: number;
}

interface QueueItem extends SearchState {
	f: number;
}

const DIRS: Cell[] = [
	{ x: 1, y: 0 },
	{ x: 1, y: 1 },
	{ x: 0, y: 1 },
	{ x: -1, y: 1 },
	{ x: -1, y: 0 },
	{ x: -1, y: -1 },
	{ x: 0, y: -1 },
	{ x: 1, y: -1 },
];

const SQRT2 = Math.SQRT2;

class MinHeap<T extends { f: number }> {
	private items: T[] = [];

	get length(): number {
		return this.items.length;
	}

	push(item: T): void {
		this.items.push(item);
		this.bubbleUp(this.items.length - 1);
	}

	pop(): T | undefined {
		const first = this.items[0];
		const last = this.items.pop();
		if (!first || !last) return first;
		if (this.items.length > 0) {
			this.items[0] = last;
			this.bubbleDown(0);
		}
		return first;
	}

	private bubbleUp(index: number): void {
		while (index > 0) {
			const parent = Math.floor((index - 1) / 2);
			if (this.items[parent].f <= this.items[index].f) break;
			[this.items[parent], this.items[index]] = [this.items[index], this.items[parent]];
			index = parent;
		}
	}

	private bubbleDown(index: number): void {
		while (true) {
			const left = index * 2 + 1;
			const right = left + 1;
			let smallest = index;
			if (left < this.items.length && this.items[left].f < this.items[smallest].f) smallest = left;
			if (right < this.items.length && this.items[right].f < this.items[smallest].f) smallest = right;
			if (smallest === index) break;
			[this.items[index], this.items[smallest]] = [this.items[smallest], this.items[index]];
			index = smallest;
		}
	}
}

function cellKey(cell: Cell): string {
	return `${cell.x},${cell.y}`;
}

function stateKey(state: SearchState): string {
	return `${state.x},${state.y},${state.dir}`;
}

function toCell(point: Point, gridSize: number): Cell {
	return {
		x: Math.round(point.x / gridSize),
		y: Math.round(point.y / gridSize),
	};
}

function toPoint(cell: Cell, gridSize: number): Point {
	return {
		x: cell.x * gridSize,
		y: cell.y * gridSize,
	};
}

function octileDistance(a: Cell, b: Cell): number {
	const dx = Math.abs(a.x - b.x);
	const dy = Math.abs(a.y - b.y);
	const diagonal = Math.min(dx, dy);
	const straight = Math.max(dx, dy) - diagonal;
	return diagonal * SQRT2 + straight;
}

function directionCost(fromDir: number, toDir: number, bendPenalty: number, backtrackPenalty: number): number {
	if (fromDir < 0 || fromDir === toDir) return 0;
	const diff = Math.abs(fromDir - toDir);
	const turn = Math.min(diff, 8 - diff);
	if (turn === 4) return bendPenalty + backtrackPenalty;
	return bendPenalty;
}

function buildBlockedCells(
	obstacles: ObstacleRect[],
	gridSize: number,
	start: Cell,
	goal: Cell,
): Set<string> {
	const blocked = new Set<string>();

	for (const rect of obstacles) {
		const minX = Math.floor(rect.x1 / gridSize);
		const maxX = Math.ceil(rect.x2 / gridSize);
		const minY = Math.floor(rect.y1 / gridSize);
		const maxY = Math.ceil(rect.y2 / gridSize);

		for (let x = minX; x <= maxX; x += 1) {
			for (let y = minY; y <= maxY; y += 1) {
				blocked.add(cellKey({ x, y }));
			}
		}
	}

	blocked.delete(cellKey(start));
	blocked.delete(cellKey(goal));
	return blocked;
}

function simplifyCells(cells: Cell[]): Cell[] {
	if (cells.length <= 2) return cells;

	const simplified: Cell[] = [cells[0]];
	let prevDir = {
		x: Math.sign(cells[1].x - cells[0].x),
		y: Math.sign(cells[1].y - cells[0].y),
	};

	for (let i = 1; i < cells.length - 1; i += 1) {
		const nextDir = {
			x: Math.sign(cells[i + 1].x - cells[i].x),
			y: Math.sign(cells[i + 1].y - cells[i].y),
		};
		if (nextDir.x !== prevDir.x || nextDir.y !== prevDir.y) {
			simplified.push(cells[i]);
			prevDir = nextDir;
		}
	}

	simplified.push(cells[cells.length - 1]);
	return simplified;
}

function reconstruct(cameFrom: Map<string, string>, endKey: string): Cell[] {
	const states: SearchState[] = [];
	let key: string | undefined = endKey;

	while (key) {
		const [x, y, dir] = key.split(',').map(Number);
		states.push({ x, y, dir });
		key = cameFrom.get(key);
	}

	states.reverse();
	return simplifyCells(states.map(({ x, y }) => ({ x, y })));
}

export function routeGrid8(startPoint: Point, goalPoint: Point, obstacles: ObstacleRect[], options: GridRouteOptions = {}): Point[] | null {
	const gridSize = options.gridSize ?? 24;
	const paddingCells = options.paddingCells ?? 6;
	const maxExpansions = options.maxExpansions ?? 80_000;
	const bendPenalty = options.bendPenalty ?? 0.03;
	const backtrackPenalty = options.backtrackPenalty ?? 0.01;
	const start = toCell(startPoint, gridSize);
	const goal = toCell(goalPoint, gridSize);
	const blocked = buildBlockedCells(obstacles, gridSize, start, goal);

	const minX = Math.min(start.x, goal.x, ...obstacles.map((r) => Math.floor(r.x1 / gridSize))) - paddingCells;
	const maxX = Math.max(start.x, goal.x, ...obstacles.map((r) => Math.ceil(r.x2 / gridSize))) + paddingCells;
	const minY = Math.min(start.y, goal.y, ...obstacles.map((r) => Math.floor(r.y1 / gridSize))) - paddingCells;
	const maxY = Math.max(start.y, goal.y, ...obstacles.map((r) => Math.ceil(r.y2 / gridSize))) + paddingCells;

	const open = new MinHeap<QueueItem>();
	const cameFrom = new Map<string, string>();
	const gScore = new Map<string, number>();
	const startState: SearchState = { ...start, dir: -1 };
	const startKey = stateKey(startState);

	gScore.set(startKey, 0);
	open.push({ ...startState, f: octileDistance(start, goal) });

	let expansions = 0;
	while (open.length > 0 && expansions < maxExpansions) {
		const current = open.pop();
		if (!current) break;

		const currentKey = stateKey(current);
		const currentG = gScore.get(currentKey);
		if (currentG === undefined) continue;
		if (current.x === goal.x && current.y === goal.y) {
			return reconstruct(cameFrom, currentKey).map((cell) => toPoint(cell, gridSize));
		}

		expansions += 1;
		for (let dir = 0; dir < DIRS.length; dir += 1) {
			const delta = DIRS[dir];
			const next = { x: current.x + delta.x, y: current.y + delta.y };
			if (next.x < minX || next.x > maxX || next.y < minY || next.y > maxY) continue;
			if (blocked.has(cellKey(next))) continue;
			if (
				delta.x !== 0 &&
				delta.y !== 0 &&
				(blocked.has(cellKey({ x: current.x + delta.x, y: current.y })) ||
					blocked.has(cellKey({ x: current.x, y: current.y + delta.y })))
			) {
				continue;
			}

			const stepCost = delta.x !== 0 && delta.y !== 0 ? SQRT2 : 1;
			const nextState: SearchState = { ...next, dir };
			const nextKey = stateKey(nextState);
			const tentativeG = currentG + stepCost + directionCost(current.dir, dir, bendPenalty, backtrackPenalty);
			if (tentativeG >= (gScore.get(nextKey) ?? Number.POSITIVE_INFINITY)) continue;

			cameFrom.set(nextKey, currentKey);
			gScore.set(nextKey, tentativeG);
			open.push({
				...nextState,
				f: tentativeG + octileDistance(next, goal),
			});
		}
	}

	return null;
}
