export interface ShortcutActions {
	deleteSelection: () => void;
	escape: () => void;
	undo: () => void;
	redo: () => void;
	selectAll: () => void;
	addNode: () => void;
	addGroup: () => void;
	zoomIn: () => void;
	zoomOut: () => void;
	fitView: () => void;
	runLayout: () => void;
	toggleImpact: () => void;
	toggleAnalysis: () => void;
}

export function createKeydownHandler(actions: ShortcutActions): (e: KeyboardEvent) => void {
	return (e: KeyboardEvent) => {
		const tag = (e.target as HTMLElement)?.tagName;
		if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT') return;
		if ((e.target as HTMLElement)?.isContentEditable) return;

		const mod = e.ctrlKey || e.metaKey;
		const code = e.code;

		if (code === 'Delete' || code === 'Backspace') {
			e.preventDefault();
			actions.deleteSelection();
			return;
		}

		if (code === 'Escape') {
			actions.escape();
			return;
		}

		if (mod && !e.shiftKey && code === 'KeyZ') {
			e.preventDefault();
			actions.undo();
			return;
		}
		if ((mod && e.shiftKey && code === 'KeyZ') || (mod && code === 'KeyY')) {
			e.preventDefault();
			actions.redo();
			return;
		}
		if (mod && code === 'KeyA') {
			e.preventDefault();
			actions.selectAll();
			return;
		}

		if (mod || e.altKey) return;

		switch (code) {
			case 'KeyN':
				actions.addNode();
				break;
			case 'KeyG':
				actions.addGroup();
				break;
			case 'Equal':
				actions.zoomIn();
				break;
			case 'Minus':
				actions.zoomOut();
				break;
			case 'Digit0':
				actions.fitView();
				break;
			case 'KeyL':
				actions.runLayout();
				break;
			case 'KeyI':
				actions.toggleImpact();
				break;
			case 'KeyA':
				if (!e.shiftKey) return;
				actions.toggleAnalysis();
				break;
			default:
				return;
		}
	};
}
