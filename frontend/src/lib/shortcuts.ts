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
}

export function createKeydownHandler(actions: ShortcutActions): (e: KeyboardEvent) => void {
	return (e: KeyboardEvent) => {
		const tag = (e.target as HTMLElement)?.tagName;
		if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT') return;
		if ((e.target as HTMLElement)?.isContentEditable) return;

		const mod = e.ctrlKey || e.metaKey;
		const key = e.key.toLowerCase();

		if (e.key === 'Delete' || e.key === 'Backspace') {
			e.preventDefault();
			actions.deleteSelection();
			return;
		}

		if (e.key === 'Escape') {
			actions.escape();
			return;
		}

		if (mod && !e.shiftKey && key === 'z') {
			e.preventDefault();
			actions.undo();
			return;
		}
		if ((mod && e.shiftKey && key === 'z') || (mod && key === 'y')) {
			e.preventDefault();
			actions.redo();
			return;
		}
		if (mod && key === 'a') {
			e.preventDefault();
			actions.selectAll();
			return;
		}

		if (mod || e.altKey) return;

		switch (key) {
			case 'n':
				actions.addNode();
				break;
			case 'g':
				actions.addGroup();
				break;
			case '+':
			case '=':
				actions.zoomIn();
				break;
			case '-':
				actions.zoomOut();
				break;
			case '0':
				actions.fitView();
				break;
			case 'l':
				actions.runLayout();
				break;
			case 'i':
				actions.toggleImpact();
				break;
			default:
				return;
		}
	};
}
