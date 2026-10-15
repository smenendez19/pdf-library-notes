import type { StickyNote } from "./useStickyNotes";

export const DARK_MODE_STORAGE_KEY = "pdf-notes:dark-mode";
export const NOTES_STORAGE_KEY_PREFIX = "pdf-notes:notes:";

function getStorage(): Storage | null {
  if (typeof window === "undefined") return null;
  try {
    return window.localStorage;
  } catch {
    return null;
  }
}

function notesStorageKey(documentKey: string): string | null {
  if (!documentKey) return null;
  return `${NOTES_STORAGE_KEY_PREFIX}${encodeURIComponent(documentKey)}`;
}

function isStoredNote(value: unknown): value is StickyNote {
  if (!value || typeof value !== "object") return false;
  const note = value as Record<string, unknown>;
  return (
    Number.isInteger(note.id) &&
    (note.id as number) > 0 &&
    typeof note.page === "number" &&
    Number.isFinite(note.page) &&
    typeof note.content === "string" &&
    typeof note.color === "string" &&
    typeof note.x === "number" &&
    Number.isFinite(note.x) &&
    typeof note.y === "number" &&
    Number.isFinite(note.y) &&
    typeof note.width === "number" &&
    Number.isFinite(note.width) &&
    typeof note.height === "number" &&
    Number.isFinite(note.height) &&
    typeof note.visible === "boolean"
  );
}

export function useReaderPersistence() {
  function readDarkMode(): boolean {
    const storage = getStorage();
    if (!storage) return false;

    try {
      const value: unknown = JSON.parse(storage.getItem(DARK_MODE_STORAGE_KEY) ?? "null");
      return typeof value === "boolean" ? value : false;
    } catch {
      return false;
    }
  }

  function writeDarkMode(isDarkMode: boolean): void {
    const storage = getStorage();
    if (!storage || typeof isDarkMode !== "boolean") return;

    try {
      storage.setItem(DARK_MODE_STORAGE_KEY, JSON.stringify(isDarkMode));
    } catch {
      // localStorage may be unavailable or full.
    }
  }

  function readNotes(documentKey: string): StickyNote[] {
    const key = notesStorageKey(documentKey);
    const storage = getStorage();
    if (!key || !storage) return [];

    try {
      const value: unknown = JSON.parse(storage.getItem(key) ?? "null");
      return Array.isArray(value)
        ? value.filter(isStoredNote).map((note) => ({ ...note }))
        : [];
    } catch {
      return [];
    }
  }

  function writeNotes(documentKey: string, notes: ReadonlyArray<StickyNote>): void {
    const key = notesStorageKey(documentKey);
    const storage = getStorage();
    if (!key || !storage) return;

    try {
      const validNotes = notes.filter(isStoredNote).map((note) => ({ ...note }));
      storage.setItem(key, JSON.stringify(validNotes));
    } catch {
      // localStorage may be unavailable, full, or receive unserializable data.
    }
  }

  return {
    readDarkMode,
    writeDarkMode,
    readNotes,
    writeNotes,
  };
}
