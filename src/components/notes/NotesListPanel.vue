<script setup lang="ts">
import DOMPurify from "dompurify";
import { marked } from "marked";
import { computed, onBeforeUnmount, ref, watch } from "vue";
import {
  DEFAULT_COLOR,
  MAX_CONTENT_LENGTH,
  noteColor,
  type StickyNote,
} from "../../composables/useStickyNotes";

type NotesFilter = "todas" | "esta página" | "ocultas";

const props = defineProps<{
  notes: StickyNote[];
  currentPage: number;
}>();

const emit = defineEmits<{
  create: [payload: { page: number; content: string; color: string }];
  update: [id: number, patch: Partial<StickyNote>];
  delete: [id: number];
  "toggle-visibility": [id: number];
  "go-to-page": [page: number];
}>();

const filter = ref<NotesFilter>("todas");
const draft = ref("");
const draftColor = ref(DEFAULT_COLOR);
const editingId = ref<number | null>(null);
const isEditorOpen = ref(false);
const isPreviewOpen = ref(false);
const selectedNoteId = ref<number | null>(null);

const draftPreview = computed(() =>
  DOMPurify.sanitize(marked.parse(draft.value) as string),
);

const filteredNotes = computed(() => {
  if (filter.value === "esta página")
    return props.notes.filter((note) => note.page === props.currentPage);
  if (filter.value === "ocultas")
    return props.notes.filter((note) => !note.visible);
  return props.notes;
});

const displayNumberById = computed(
  () => new Map(props.notes.map((note, index) => [note.id, index + 1])),
);

const selectedNote = computed<StickyNote | null>(() => {
  if (selectedNoteId.value === null) return null;
  return props.notes.find((note) => note.id === selectedNoteId.value) ?? null;
});

const selectedNotePreview = computed(() =>
  selectedNote.value
    ? DOMPurify.sanitize(marked.parse(selectedNote.value.content) as string)
    : "",
);

const emptyMessage = computed(() => {
  if (!props.notes.length) return "No hay notas todavía. Crea la primera en la página actual.";
  if (filter.value === "esta página")
    return `No hay notas en la página ${props.currentPage}.`;
  if (filter.value === "ocultas") return "No hay notas ocultas.";
  return "No hay notas.";
});

function openNewNote(): void {
  draft.value = "";
  draftColor.value = DEFAULT_COLOR;
  editingId.value = null;
  isEditorOpen.value = true;
  isPreviewOpen.value = false;
}

function openNoteForEditing(note: StickyNote): void {
  draft.value = note.content;
  draftColor.value = noteColor(note);
  editingId.value = note.id;
  isEditorOpen.value = true;
  isPreviewOpen.value = false;
}

function openNoteOptions(note: StickyNote): void {
  selectedNoteId.value = note.id;
}

function closeNoteOptions(): void {
  selectedNoteId.value = null;
}

function handleEscape(event: KeyboardEvent): void {
  if (event.key === "Escape") closeNoteOptions();
}

function updateEscapeListener(isOpen: boolean): void {
  if (typeof document === "undefined") return;
  if (isOpen) {
    document.addEventListener("keydown", handleEscape);
  } else {
    document.removeEventListener("keydown", handleEscape);
  }
}

function goToSelectedNote(): void {
  const note = selectedNote.value;
  if (!note) return;
  emit("go-to-page", note.page);
  closeNoteOptions();
}

function editSelectedNote(): void {
  const note = selectedNote.value;
  if (!note) return;
  closeNoteOptions();
  openNoteForEditing(note);
}

function toggleSelectedNoteVisibility(): void {
  const note = selectedNote.value;
  if (!note) return;
  emit("toggle-visibility", note.id);
  closeNoteOptions();
}

function deleteSelectedNote(): void {
  const note = selectedNote.value;
  if (!note) return;
  closeNoteOptions();
  emit("delete", note.id);
}

function cancelEditing(): void {
  draft.value = "";
  draftColor.value = DEFAULT_COLOR;
  editingId.value = null;
  isEditorOpen.value = false;
  isPreviewOpen.value = false;
}

function saveNote(): void {
  if (!draft.value.trim()) return;
  const content = draft.value.slice(0, MAX_CONTENT_LENGTH);
  const color = draftColor.value;
  if (editingId.value === null) {
    emit("create", { page: props.currentPage, content, color });
  } else {
    emit("update", editingId.value, { content, color });
  }
  cancelEditing();
}

watch(
  () => props.currentPage,
  () => {
    if (editingId.value === null) cancelEditing();
  },
);

watch(selectedNoteId, (noteId) => {
  updateEscapeListener(noteId !== null);
});

watch(selectedNote, (note) => {
  if (!note) closeNoteOptions();
});

onBeforeUnmount(() => updateEscapeListener(false));
</script>

<template>
  <section class="notes-inventory" aria-labelledby="notes-inventory-title">
    <div class="notes-inventory__header">
      <div>
        <h2 id="notes-inventory-title" class="notes-inventory__title">
          Notas
        </h2>
        <p class="notes-inventory__page">Página actual: {{ currentPage }}</p>
      </div>
      <button
        class="new-note-button"
        type="button"
        aria-label="Nueva nota"
        @click="openNewNote"
      >
        <svg aria-hidden="true" focusable="false" viewBox="0 0 24 24">
          <path d="M12 5v14M5 12h14" />
        </svg>
      </button>
    </div>

    <form v-if="isEditorOpen" class="note-editor" @submit.prevent="saveNote">
      <label class="note-editor__label" for="note-content"
        >Contenido de la nota</label
      >
      <textarea
        id="note-content"
        class="note-editor__textarea"
        v-model="draft"
        :maxlength="MAX_CONTENT_LENGTH"
        rows="6"
        placeholder="Escribe una nota"
      ></textarea>
      <label class="note-editor__label" for="note-color"
        >Color de la nota</label
      >
      <input
        id="note-color"
        v-model="draftColor"
        type="color"
        aria-label="Color de la nota"
      />
      <div class="note-editor__meta">
        <span aria-live="polite"
          >{{ draft.length }}/{{ MAX_CONTENT_LENGTH }} caracteres</span
        >
      </div>
      <button
        type="button"
        :aria-expanded="isPreviewOpen"
        @click="isPreviewOpen = !isPreviewOpen"
      >
        {{ isPreviewOpen ? "Ocultar vista previa" : "Vista previa" }}
      </button>
      <div
        v-if="isPreviewOpen"
        class="note-preview"
        aria-label="Vista previa de la nota"
      >
        <div v-html="draftPreview"></div>
      </div>
      <div class="note-editor__actions">
        <button
          type="submit"
          :disabled="!draft.trim()"
          :aria-label="
            editingId === null ? 'Guardar nota' : 'Guardar cambios'
          "
        >
          Guardar
        </button>
        <button type="button" aria-label="Cancelar edición" @click="cancelEditing">
          Cancelar
        </button>
      </div>
    </form>

    <div
      class="notes-filter"
      role="group"
      aria-label="Filtrar notas"
    >
      <button
        type="button"
        :aria-pressed="filter === 'todas'"
        :class="{ 'is-active': filter === 'todas' }"
        @click="filter = 'todas'"
      >
        Todas
      </button>
      <button
        type="button"
        :aria-pressed="filter === 'esta página'"
        :class="{ 'is-active': filter === 'esta página' }"
        @click="filter = 'esta página'"
      >
        Esta página
      </button>
      <button
        type="button"
        :aria-pressed="filter === 'ocultas'"
        :class="{ 'is-active': filter === 'ocultas' }"
        @click="filter = 'ocultas'"
      >
        Ocultas
      </button>
    </div>

    <p v-if="!filteredNotes.length" class="notes-inventory__empty">
      {{ emptyMessage }}
    </p>

    <ul v-else class="notes-list" aria-label="Inventario de notas">
      <li
        v-for="note in filteredNotes"
        :key="note.id"
        class="note-row"
        :style="{ backgroundColor: noteColor(note) }"
      >
        <p class="note-row__title">
          Nota {{ displayNumberById.get(note.id) ?? 0 }}
        </p>
        <span class="note-row__badge">Pág. {{ note.page }}</span>
        <button
          class="note-row__options"
          type="button"
          :aria-label="`Opciones de Nota ${displayNumberById.get(note.id) ?? 0}`"
          @click="openNoteOptions(note)"
        >
          <svg aria-hidden="true" focusable="false" viewBox="0 0 24 24">
            <path d="m9 5 7 7-7 7" />
          </svg>
        </button>
      </li>
    </ul>
  </section>

  <Teleport to="body">
    <div
      v-if="selectedNote"
      class="notes-modal"
      role="presentation"
      @click.self="closeNoteOptions"
    >
      <div
        class="notes-modal__dialog"
        role="dialog"
        aria-modal="true"
        aria-labelledby="selected-note-title"
        :style="{ backgroundColor: noteColor(selectedNote) }"
        @keydown.esc="closeNoteOptions"
      >
        <div class="notes-modal__header">
          <div>
            <h2 id="selected-note-title" class="notes-modal__title">
              Nota {{ displayNumberById.get(selectedNote.id) ?? 0 }}
            </h2>
            <p class="notes-modal__page">Página {{ selectedNote.page }}</p>
          </div>
          <button
            class="notes-modal__close"
            type="button"
            aria-label="Cerrar opciones de la nota"
            @click="closeNoteOptions"
          >
            <span aria-hidden="true">×</span>
          </button>
        </div>
        <div
          class="notes-modal__preview"
          aria-label="Vista previa completa de la nota"
        >
          <div v-html="selectedNotePreview"></div>
        </div>
        <div class="notes-modal__actions">
          <button type="button" @click="goToSelectedNote">Ir a página</button>
          <button type="button" @click="editSelectedNote">Editar</button>
          <button type="button" @click="toggleSelectedNoteVisibility">
            {{ selectedNote.visible ? "Ocultar" : "Mostrar" }}
          </button>
          <button type="button" @click="deleteSelectedNote">Eliminar</button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.notes-inventory {
  height: 100%;
  min-height: 0;
  box-sizing: border-box;
  overflow: auto;
  padding: 1rem;
  border: 1px solid var(--reader-border, #dbe4ee);
  border-radius: 0.5rem;
  background: var(--reader-surface, #ffffff);
  color: var(--reader-text, #334155);
}

.notes-inventory__header,
.note-editor__meta,
.note-editor__actions,
.notes-modal__header,
.notes-modal__actions {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.notes-inventory__header {
  justify-content: space-between;
  gap: 1rem;
}

.notes-inventory__title,
.notes-modal__title {
  margin: 0;
  color: var(--reader-heading, #172033);
  font-size: 1.1rem;
}

.notes-inventory__page,
.notes-inventory__empty,
.note-editor__meta {
  color: var(--reader-muted, #64748b);
  font-size: 0.875rem;
}

.notes-inventory__page {
  margin: 0.25rem 0 0;
}

.notes-inventory__empty {
  margin: 1rem 0 0;
}

.note-editor {
  display: grid;
  gap: 0.5rem;
  margin-top: 1rem;
}

.note-editor__label {
  color: var(--reader-heading, #172033);
  font-weight: 600;
}

.note-editor__textarea {
  width: 100%;
  box-sizing: border-box;
  resize: vertical;
  border: 1px solid var(--reader-border, #cbd5e1);
  border-radius: 0.35rem;
  padding: 0.6rem;
  font: inherit;
  background: var(--reader-surface, #ffffff);
  color: var(--reader-text, #334155);
}

.note-editor__meta {
  justify-content: space-between;
  flex-wrap: wrap;
}

.note-preview {
  padding: 0.75rem;
  border: 1px solid var(--reader-border, #e2e8f0);
  border-radius: 0.35rem;
  background: color-mix(in srgb, var(--reader-surface, #ffffff) 88%, var(--reader-text, #334155));
}

.note-editor__actions {
  justify-content: flex-end;
}

.notes-filter {
  display: flex;
  gap: 0.5rem;
  margin-top: 1rem;
}

.notes-filter .is-active {
  outline: 2px solid var(--reader-focus, #2563eb);
  outline-offset: 1px;
}

.notes-list {
  display: grid;
  gap: 0.75rem;
  margin-top: 1rem;
  padding: 0;
  list-style: none;
}

.note-row {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  min-height: 2.75rem;
  box-sizing: border-box;
  padding: 0.75rem;
  border: 1px solid rgb(31 41 55 / 18%);
  border-radius: 0.35rem;
  color: #1f2937;
}

.note-row__badge {
  display: inline-block;
  padding: 0.1rem 0.5rem;
  border-radius: 999px;
  background: rgb(255 255 255 / 65%);
  font-size: 0.75rem;
  font-weight: 700;
}

.note-row__title {
  flex: 1;
  margin: 0;
  overflow-wrap: anywhere;
  color: #1f2937;
  font-size: 0.875rem;
  font-weight: 600;
}

.note-row__options {
  display: inline-grid;
  width: 2rem;
  height: 2rem;
  min-height: 0;
  flex: 0 0 auto;
  place-items: center;
  padding: 0;
  border-color: rgb(31 41 55 / 20%);
  background: rgb(255 255 255 / 55%);
  color: #1f2937;
}

.note-row__options svg {
  width: 1.15rem;
  height: 1.15rem;
  fill: none;
  stroke: currentColor;
  stroke-linecap: round;
  stroke-linejoin: round;
  stroke-width: 2;
}

.notes-modal {
  position: fixed;
  z-index: 1000;
  inset: 0;
  display: grid;
  place-items: center;
  padding: 1rem;
  background: rgb(15 23 42 / 60%);
}

.notes-modal__dialog {
  width: min(100%, 36rem);
  max-height: min(90vh, 42rem);
  box-sizing: border-box;
  overflow: auto;
  padding: 1rem;
  border: 1px solid rgb(31 41 55 / 22%);
  border-radius: 0.5rem;
  color: #1f2937;
}

.notes-modal__header {
  justify-content: space-between;
  align-items: flex-start;
}

.notes-modal__title {
  color: #1f2937;
}

.notes-modal__page {
  margin: 0.25rem 0 0;
  font-size: 0.875rem;
}

.notes-modal__close {
  display: inline-grid;
  width: 2rem;
  height: 2rem;
  min-height: 0;
  place-items: center;
  padding: 0;
  background: rgb(255 255 255 / 55%);
  color: #1f2937;
  font-size: 1.4rem;
  line-height: 1;
}

.notes-modal__preview {
  max-height: 50vh;
  margin-top: 1rem;
  overflow: auto;
  padding: 0.75rem;
  border: 1px solid rgb(31 41 55 / 18%);
  border-radius: 0.35rem;
  background: rgb(255 255 255 / 70%);
}

.notes-modal__actions {
  justify-content: flex-end;
  flex-wrap: wrap;
  margin-top: 1rem;
}

.new-note-button {
  display: inline-grid;
  width: 2.25rem;
  height: 2.25rem;
  min-height: 0;
  place-items: center;
  padding: 0;
  border-radius: 50%;
}

.new-note-button svg {
  width: 1.25rem;
  height: 1.25rem;
  fill: none;
  stroke: currentColor;
  stroke-linecap: round;
  stroke-width: 2;
}

.notes-inventory button,
.notes-modal button {
  min-height: 2rem;
  padding: 0.25rem 0.6rem;
  border: 1px solid var(--reader-border, #cbd5e1);
  border-radius: 0.35rem;
  background: var(--reader-surface, #ffffff);
  color: var(--reader-heading, #172033);
  cursor: pointer;
}

.notes-inventory button:disabled,
.notes-modal button:disabled {
  cursor: not-allowed;
  opacity: 0.5;
}

.notes-inventory button:focus-visible,
.notes-modal button:focus-visible {
  outline: 3px solid var(--reader-focus, #2563eb);
  outline-offset: 2px;
}
</style>
