import { computed, ref, type ComputedRef, type Ref } from "vue";

export interface StickyNote {
  id: number;
  page: number;
  content: string;
  color: string;
  x: number;
  y: number;
  width: number;
  height: number;
  visible: boolean;
}

export const DEFAULT_COLOR = "#fff9db";
export const MAX_CONTENT_LENGTH = 1000;
export const DEFAULT_WIDTH = 200;
export const DEFAULT_HEIGHT = 150;
export const MIN_WIDTH = 140;
export const MIN_HEIGHT = 100;
export const MAX_WIDTH = 480;
export const MAX_HEIGHT = 480;
export const EXTENDED_MIN = -100;
export const EXTENDED_MAX = 200;

const COLOR_PATTERN = /^#[\da-f]{6}$/i;

export function noteColor(note: Pick<StickyNote, "color">): string {
  return COLOR_PATTERN.test(note.color) ? note.color : DEFAULT_COLOR;
}

function clampPosition(value: number): number {
  if (!Number.isFinite(value)) return 0;
  return Math.min(EXTENDED_MAX, Math.max(EXTENDED_MIN, value));
}

function clampSize(value: number, min: number, max: number): number {
  if (!Number.isFinite(value)) return min;
  return Math.min(max, Math.max(min, value));
}

function normalizeContent(content: string): string {
  return content.slice(0, MAX_CONTENT_LENGTH);
}

export interface UseStickyNotesOptions {
  initialNotes?: ReadonlyArray<StickyNote>;
}

export function useStickyNotes(options: UseStickyNotesOptions = {}) {
  const initialNotes = options.initialNotes ?? [];
  const notes: Ref<StickyNote[]> = ref(initialNotes.map((note) => ({ ...note })));
  const maxInitialId = notes.value.reduce(
    (maxId, note) => (Number.isFinite(note.id) ? Math.max(maxId, note.id) : maxId),
    0,
  );
  let nextId = maxInitialId + 1;

  function createNote(
    page: number,
    partial?: Partial<Pick<StickyNote, "content" | "color" | "x" | "y" | "width" | "height">>,
  ): StickyNote {
    const count = notes.value.length;
    const note: StickyNote = {
      id: nextId++,
      page,
      content: normalizeContent(partial?.content ?? ""),
      color: partial?.color ?? DEFAULT_COLOR,
      x: partial?.x ?? 12 + (count % 5) * 6,
      y: partial?.y ?? 10 + (count % 5) * 6,
      width: clampSize(partial?.width ?? DEFAULT_WIDTH, MIN_WIDTH, MAX_WIDTH),
      height: clampSize(partial?.height ?? DEFAULT_HEIGHT, MIN_HEIGHT, MAX_HEIGHT),
      visible: true,
    };
    notes.value.push(note);
    return note;
  }

  function updateNote(id: number, patch: Partial<StickyNote>): void {
    const note = notes.value.find((candidate) => candidate.id === id);
    if (!note) return;
    if (patch.content !== undefined) {
      note.content = normalizeContent(patch.content);
    }
    if (patch.color !== undefined) note.color = patch.color;
    if (patch.page !== undefined) note.page = patch.page;
    if (patch.x !== undefined) note.x = clampPosition(patch.x);
    if (patch.y !== undefined) note.y = clampPosition(patch.y);
    if (patch.width !== undefined)
      note.width = clampSize(patch.width, MIN_WIDTH, MAX_WIDTH);
    if (patch.height !== undefined)
      note.height = clampSize(patch.height, MIN_HEIGHT, MAX_HEIGHT);
    if (patch.visible !== undefined) note.visible = patch.visible;
  }

  function moveNote(id: number, x: number, y: number): void {
    const note = notes.value.find((candidate) => candidate.id === id);
    if (!note) return;
    note.x = clampPosition(x);
    note.y = clampPosition(y);
  }

  function resizeNote(id: number, width: number, height: number): void {
    const note = notes.value.find((candidate) => candidate.id === id);
    if (!note) return;
    if (!Number.isFinite(width) || !Number.isFinite(height)) return;
    note.width = clampSize(width, MIN_WIDTH, MAX_WIDTH);
    note.height = clampSize(height, MIN_HEIGHT, MAX_HEIGHT);
  }

  function toggleVisibility(id: number): void {
    const note = notes.value.find((candidate) => candidate.id === id);
    if (!note) return;
    note.visible = !note.visible;
  }

  function removeNote(id: number): void {
    notes.value = notes.value.filter((candidate) => candidate.id !== id);
  }

  function notesForPage(page: number): ComputedRef<StickyNote[]> {
    return computed(() => notes.value.filter((note) => note.page === page));
  }

  function visibleNotesForPage(page: number): ComputedRef<StickyNote[]> {
    return computed(() =>
      notes.value.filter((note) => note.page === page && note.visible),
    );
  }

  return {
    notes,
    createNote,
    updateNote,
    moveNote,
    resizeNote,
    toggleVisibility,
    removeNote,
    notesForPage,
    visibleNotesForPage,
    noteColor,
  };
}
