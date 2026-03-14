export interface Command {
	description: string;
	execute(): Promise<void>;
	undo(): Promise<void>;
}

class UndoStack {
	past = $state<Command[]>([]);
	future = $state<Command[]>([]);

	get canUndo() {
		return this.past.length > 0;
	}

	get canRedo() {
		return this.future.length > 0;
	}

	async run(cmd: Command) {
		await cmd.execute();
		this.past = [...this.past, cmd];
		this.future = [];
	}

	/** Record a command that was already executed (e.g. drag) */
	record(cmd: Command) {
		this.past = [...this.past, cmd];
		this.future = [];
	}

	async undo() {
		if (this.past.length === 0) return;
		const cmd = this.past[this.past.length - 1];
		this.past = this.past.slice(0, -1);
		await cmd.undo();
		this.future = [...this.future, cmd];
	}

	async redo() {
		if (this.future.length === 0) return;
		const cmd = this.future[this.future.length - 1];
		this.future = this.future.slice(0, -1);
		await cmd.execute();
		this.past = [...this.past, cmd];
	}

	clear() {
		this.past = [];
		this.future = [];
	}
}

export const undoStack = new UndoStack();
