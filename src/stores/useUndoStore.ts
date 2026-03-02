'use client';

import { create } from 'zustand';
import type { UndoEntry } from '@/types/undo';

const MAX_STACK = 50;

interface UndoState {
  undoStack: UndoEntry[];
  redoStack: UndoEntry[];
  canUndo: boolean;
  canRedo: boolean;
  pushUndo: (entry: UndoEntry) => void;
  pushUndoFromRedo: (entry: UndoEntry) => void;
  popUndo: () => UndoEntry | null;
  popRedo: () => UndoEntry | null;
  pushRedo: (entry: UndoEntry) => void;
  clearStacks: () => void;
}

export const useUndoStore = create<UndoState>()((set, get) => ({
  undoStack: [],
  redoStack: [],
  canUndo: false,
  canRedo: false,

  pushUndo: (entry) =>
    set((s) => {
      const stack = [...s.undoStack, entry].slice(-MAX_STACK);
      return { undoStack: stack, redoStack: [], canUndo: true, canRedo: false };
    }),

  pushUndoFromRedo: (entry) =>
    set((s) => {
      const stack = [...s.undoStack, entry].slice(-MAX_STACK);
      return { undoStack: stack, canUndo: true };
    }),

  popUndo: () => {
    const { undoStack } = get();
    if (undoStack.length === 0) return null;
    const entry = undoStack[undoStack.length - 1]!;
    set({
      undoStack: undoStack.slice(0, -1),
      canUndo: undoStack.length > 1,
    });
    return entry;
  },

  popRedo: () => {
    const { redoStack } = get();
    if (redoStack.length === 0) return null;
    const entry = redoStack[redoStack.length - 1]!;
    set({
      redoStack: redoStack.slice(0, -1),
      canRedo: redoStack.length > 1,
    });
    return entry;
  },

  pushRedo: (entry) =>
    set((s) => ({
      redoStack: [...s.redoStack, entry],
      canRedo: true,
    })),

  clearStacks: () =>
    set({ undoStack: [], redoStack: [], canUndo: false, canRedo: false }),
}));
