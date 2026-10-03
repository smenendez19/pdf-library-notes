<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import ThemeToggleButton from '../components/common/ThemeToggleButton.vue'
import NotesListPanel from '../components/notes/NotesListPanel.vue'
import StickyNotesOverlay from '../components/notes/StickyNotesOverlay.vue'
import PdfReader from '../components/PdfReader.vue'
import { useTheme } from '../composables/useTheme'
import { useReaderPersistence } from '../composables/useReaderPersistence'
import { useStickyNotes, type StickyNote } from '../composables/useStickyNotes'

const DEFAULT_DOCUMENT_TITLE = 'Automate the Boring Stuff with Python'
const DEFAULT_DOCUMENT_SOURCE = '/automatetheboringstuffwithpython.pdf'

const route = useRoute()
const documentTitle = computed(() => {
  const title = route.query.title
  return typeof title === 'string' && title.trim() !== '' ? title : DEFAULT_DOCUMENT_TITLE
})
const documentSource = computed(() => {
  const doc = route.query.doc
  return typeof doc === 'string' && doc.trim() !== '' && doc.startsWith('/')
    ? doc
    : DEFAULT_DOCUMENT_SOURCE
})
const currentPage = ref(1)
const { readNotes, writeNotes } = useReaderPersistence()
const { isDarkMode, toggleTheme } = useTheme()
const pdfReaderRef = ref<InstanceType<typeof PdfReader> | null>(null)

const { notes, createNote, updateNote, moveNote, resizeNote, toggleVisibility, removeNote } =
  useStickyNotes({ initialNotes: readNotes(documentSource.value) })

watch(
  notes,
  (value) => {
    writeNotes(documentSource.value, value)
  },
  { deep: true },
)

function onCreate(payload: { page: number; content: string; color: string }): void {
  createNote(payload.page, { content: payload.content, color: payload.color })
}

function onUpdate(id: number, patch: Partial<StickyNote>): void {
  updateNote(id, patch)
}

function onDelete(id: number): void {
  removeNote(id)
}

function onToggleVisibility(id: number): void {
  toggleVisibility(id)
}

function onGoToPage(page: number): void {
  currentPage.value = page
  const target = pdfReaderRef.value as unknown as {
    goToPage?: (page: number) => void
    setPage?: (page: number) => void
  } | null
  target?.goToPage?.(page) ?? target?.setPage?.(page)
}

function onMove(payload: { id: number; x: number; y: number }): void {
  moveNote(payload.id, payload.x, payload.y)
}

function onResize(payload: { id: number; width: number; height: number }): void {
  resizeNote(payload.id, payload.width, payload.height)
}
</script>

<template>
  <main class="reader-page" :class="{ 'reader-page--dark': isDarkMode }">
    <section class="reader-surface" aria-labelledby="document-title">
      <header class="reader-header">
        <div>
          <p class="reader-kicker">PDF reader</p>
          <h1 id="document-title">{{ documentTitle }}</h1>
          <p class="reader-meta">PDF document <span aria-hidden="true">·</span> Local file</p>
        </div>
        <div class="reader-actions">
          <RouterLink class="back-link" to="/catalog">← Volver al catálogo</RouterLink>
          <ThemeToggleButton :is-dark-mode="isDarkMode" @toggle="toggleTheme" />
        </div>
      </header>

      <div class="reader-layout">
        <NotesListPanel
          :notes="notes"
          :current-page="currentPage"
          @create="onCreate"
          @update="onUpdate"
          @delete="onDelete"
          @toggle-visibility="onToggleVisibility"
          @go-to-page="onGoToPage"
        />
        <div class="document-panel">
          <PdfReader ref="pdfReaderRef" :src="documentSource" @page-change="currentPage = $event">
            <template #page-overlay="{ page }">
              <StickyNotesOverlay :page="page" :notes="notes" @move="onMove" @resize="onResize" />
            </template>
          </PdfReader>
        </div>
      </div>
    </section>
  </main>
</template>

<style scoped>
.reader-page {
  --reader-background: #f4f7fb;
  --reader-surface: #ffffff;
  --reader-border: #dbe4ee;
  --reader-heading: #172033;
  --reader-text: #334155;
  --reader-muted: #64748b;
  --reader-canvas: #e2e8f0;
  height: 100svh;
  min-height: 100svh;
  padding: 1rem;
  box-sizing: border-box;
  display: flex;
  flex-direction: column;
  background: var(--reader-background);
  color: var(--reader-text);
  overflow: hidden;
  transition: background-color 0.2s ease, color 0.2s ease;
}

.reader-page--dark {
  --reader-background: #20252d;
  --reader-surface: #2b313b;
  --reader-border: #46515f;
  --reader-heading: #f8fafc;
  --reader-text: #f8fafc;
  --reader-muted: #cbd5e1;
  --reader-canvas: #161b22;
}

.reader-surface {
  width: 100%;
  height: 100%;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.reader-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 1.5rem;
  padding: 0 0 1.5rem;
}

.reader-kicker {
  margin: 0 0 0.5rem;
  color: var(--reader-muted);
  font-size: 0.75rem;
  font-weight: 700;
  letter-spacing: 0.12em;
  text-transform: uppercase;
}

h1 {
  margin: 0;
  color: var(--reader-heading);
  font-size: clamp(1.5rem, 2.5vw, 2rem);
  line-height: 1.2;
  letter-spacing: -0.025em;
}

.reader-meta {
  margin: 0.5rem 0 0;
  color: var(--reader-muted);
  font-size: 0.875rem;
}

.reader-actions {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  flex-shrink: 0;
}

.back-link {
  color: var(--reader-muted);
  font-size: 0.875rem;
  font-weight: 600;
  text-decoration: none;
  white-space: nowrap;
}

.back-link:hover {
  color: var(--reader-heading);
  text-decoration: underline;
}

.reader-layout {
  flex: 1;
  height: 100%;
  min-height: 0;
  display: grid;
  grid-template-columns: minmax(220px, 280px) minmax(0, 1fr);
  align-items: stretch;
  gap: 1rem;
  overflow: hidden;
}

.document-panel {
  min-width: 0;
  min-height: 0;
  height: 100%;
  overflow: hidden;
  border: 1px solid var(--reader-border);
  border-radius: 0.75rem;
  background: var(--reader-surface);
  box-shadow: 0 10px 30px rgb(15 23 42 / 12%);
}

.document-panel:focus-within {
  outline: 3px solid rgb(59 130 246 / 35%);
  outline-offset: 3px;
}

@media (max-width: 800px) {
  .reader-layout {
    grid-template-columns: 1fr;
    grid-template-rows: minmax(0, 1fr) minmax(0, 2fr);
  }
}

@media (max-width: 640px) {
  .reader-page {
    padding: 0.75rem 0.5rem;
  }

  .reader-header {
    gap: 1rem;
  }

  .document-panel {
    border-radius: 0.5rem;
  }
}
</style>
