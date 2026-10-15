<script setup lang="ts">
import { computed, nextTick, ref, useTemplateRef, watch } from 'vue';
import type { CatalogDocument } from '../../types/catalog';
import { uploadDocument } from '../../services/catalogService';
import { usePdfUpload } from '../../composables/usePdfUpload';

const props = defineProps<{ open: boolean }>();

const emit = defineEmits<{
  close: [];
  uploaded: [doc: CatalogDocument];
}>();

const { coverDataUrl, isParsing, parseError, prepareFile } = usePdfUpload();

const dialogRef = useTemplateRef<HTMLElement>('dialogRef');
const fileInputRef = useTemplateRef<HTMLInputElement>('fileInputRef');

const selectedFile = ref<File | null>(null);
const isDragging = ref(false);
const fileError = ref('');
const title = ref('');
const titleEdited = ref(false);
const author = ref('');
const editorial = ref('');
const description = ref('');
const tagsInput = ref('');
const isUploading = ref(false);
const error = ref('');

const tags = computed<string[]>(() => {
  const seen = new Set<string>();
  for (const part of tagsInput.value.split(',')) {
    const tag = part.trim();
    if (tag && !seen.has(tag)) seen.add(tag);
  }
  return [...seen];
});

const canSubmit = computed(
  () => selectedFile.value !== null && title.value.trim() !== '' && !isUploading.value && !isParsing.value,
);

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

function resetState(): void {
  selectedFile.value = null;
  isDragging.value = false;
  fileError.value = '';
  title.value = '';
  titleEdited.value = false;
  author.value = '';
  editorial.value = '';
  description.value = '';
  tagsInput.value = '';
  isUploading.value = false;
  error.value = '';
}

function isPdfFile(file: File): boolean {
  return file.type === 'application/pdf' || file.name.toLowerCase().endsWith('.pdf');
}

async function handleFile(file: File): Promise<void> {
  fileError.value = '';
  error.value = '';
  if (!isPdfFile(file)) {
    fileError.value = 'El archivo debe ser un PDF válido.';
    return;
  }
  if (file.size <= 0) {
    fileError.value = 'El archivo está vacío.';
    return;
  }
  selectedFile.value = file;
  try {
    const prepared = await prepareFile(file);
    if (!titleEdited.value) title.value = prepared.suggestedTitle;
    if (prepared.author) author.value = prepared.author;
    if (prepared.editorial) editorial.value = prepared.editorial;
  } catch {
    // prepareFile no lanza por validación aquí (ya validado); el aviso
    // no bloqueante de portada/metadata vive en parseError.
  }
}

function onInputChange(event: Event): void {
  const input = event.target as HTMLInputElement;
  const file = input.files?.[0];
  if (file) void handleFile(file);
  input.value = '';
}

function onDrop(event: DragEvent): void {
  isDragging.value = false;
  const file = event.dataTransfer?.files?.[0];
  if (file) void handleFile(file);
}

function openPicker(): void {
  fileInputRef.value?.click();
}

function onDropzoneKeydown(event: KeyboardEvent): void {
  if (event.key === 'Enter' || event.key === ' ') {
    event.preventDefault();
    openPicker();
  }
}

function onOverlayClick(event: MouseEvent): void {
  if (event.target === event.currentTarget) emit('close');
}

function onEscape(event: KeyboardEvent): void {
  if (event.key === 'Escape') emit('close');
}

async function onSubmit(): Promise<void> {
  if (!canSubmit.value || !selectedFile.value) return;
  isUploading.value = true;
  error.value = '';
  try {
    const doc = await uploadDocument({
      file: selectedFile.value,
      title: title.value.trim(),
      ...(author.value.trim() ? { author: author.value.trim() } : {}),
      ...(editorial.value.trim() ? { editorial: editorial.value.trim() } : {}),
      ...(description.value.trim() ? { description: description.value.trim() } : {}),
      tags: tags.value,
      ...(coverDataUrl.value ? { coverDataUrl: coverDataUrl.value } : {}),
    });
    emit('uploaded', doc);
    resetState();
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Error al subir el documento.';
  } finally {
    isUploading.value = false;
  }
}

function onCancel(): void {
  emit('close');
}

watch(
  () => props.open,
  (isOpen) => {
    if (isOpen) {
      void nextTick(() => dialogRef.value?.focus());
    } else {
      resetState();
    }
  },
);
</script>

<template>
  <Teleport to="body">
    <div v-if="open" class="upload-overlay" @click="onOverlayClick" @keydown="onEscape">
      <div
        ref="dialogRef"
        class="upload-dialog"
        role="dialog"
        aria-modal="true"
        aria-labelledby="upload-pdf-title"
        tabindex="-1"
      >
        <div class="upload-header">
          <h2 id="upload-pdf-title">Subir PDF</h2>
          <button class="upload-close" type="button" aria-label="Cerrar" @click="onCancel">✕</button>
        </div>

        <div
          class="upload-dropzone"
          :class="{ dragging: isDragging }"
          role="button"
          tabindex="0"
          aria-label="Seleccionar archivo PDF"
          @click="openPicker"
          @keydown="onDropzoneKeydown"
          @dragover.prevent="isDragging = true"
          @dragleave="isDragging = false"
          @drop.prevent="onDrop"
        >
          <p>Arrastra tu PDF aquí o haz clic para buscarlo</p>
          <p class="upload-hint">Solo archivos .pdf</p>
          <input
            ref="fileInputRef"
            type="file"
            accept="application/pdf,.pdf"
            class="upload-hidden-input"
            tabindex="-1"
            aria-hidden="true"
            @change="onInputChange"
          />
        </div>
        <p v-if="fileError" class="upload-error" role="alert">{{ fileError }}</p>

        <div v-if="selectedFile" class="upload-file-info">
          <div class="upload-file-meta">
            <strong>{{ selectedFile.name }}</strong>
            <span>{{ formatSize(selectedFile.size) }}</span>
          </div>
          <p v-if="isParsing" class="upload-hint" role="status">Generando portada…</p>
          <figure v-if="coverDataUrl" class="upload-cover">
            <img :src="coverDataUrl" alt="Portada del PDF" />
            <figcaption>Portada: primera página</figcaption>
          </figure>
          <p v-else-if="parseError" class="upload-warning" role="status">{{ parseError }}</p>
        </div>

        <form class="upload-form" @submit.prevent="onSubmit">
          <label class="upload-field">
            <span>Título *</span>
            <input v-model="title" type="text" required @input="titleEdited = true" />
          </label>
          <label class="upload-field">
            <span>Autor</span>
            <input v-model="author" type="text" />
          </label>
          <label class="upload-field">
            <span>Editorial</span>
            <input v-model="editorial" type="text" />
          </label>
          <label class="upload-field">
            <span>Descripción</span>
            <textarea v-model="description" rows="3" />
          </label>
          <label class="upload-field">
            <span>Etiquetas</span>
            <input v-model="tagsInput" type="text" placeholder="separa por comas" />
          </label>
          <div v-if="tags.length" class="upload-chips">
            <span v-for="tag in tags" :key="tag" class="upload-chip">{{ tag }}</span>
          </div>

          <p v-if="error" class="upload-error" role="alert">{{ error }}</p>

          <div class="upload-actions">
            <button type="button" class="upload-cancel" @click="onCancel">Cancelar</button>
            <button type="submit" class="upload-submit" :disabled="!canSubmit">
              {{ isUploading ? 'Subiendo…' : 'Subir' }}
            </button>
          </div>
        </form>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.upload-overlay {
  position: fixed;
  inset: 0;
  z-index: 50;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 1rem;
  background: rgb(15 23 42 / 0.45);
}

.upload-dialog {
  width: min(32rem, 100%);
  max-height: 90vh;
  overflow-y: auto;
  padding: 1.5rem;
  border: 1px solid var(--reader-border, #dbe4ee);
  border-radius: 0.9rem;
  background: var(--reader-surface, #ffffff);
  color: var(--reader-text, #334155);
  box-shadow: 0 12px 32px rgb(15 23 42 / 0.25);
}

.upload-dialog:focus {
  outline: none;
}

.upload-dialog:focus-visible {
  outline: 3px solid var(--reader-muted, #64748b);
  outline-offset: 2px;
}

.upload-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 1rem;
}

.upload-header h2 {
  margin: 0;
  flex: 1;
  text-align: center;
  color: var(--reader-heading, #172033);
  font-size: 1.25rem;
}

.upload-close {
  display: grid;
  place-items: center;
  width: 2rem;
  height: 2rem;
  padding: 0;
  border: 1px solid var(--reader-border, #dbe4ee);
  border-radius: 50%;
  background: transparent;
  color: var(--reader-heading, #172033);
  cursor: pointer;
  font: inherit;
}

.upload-close:focus-visible {
  outline: 3px solid var(--reader-muted, #64748b);
  outline-offset: 2px;
}

.upload-dropzone {
  padding: 1.5rem 1rem;
  border: 2px dashed var(--reader-border, #dbe4ee);
  border-radius: 0.75rem;
  text-align: center;
  cursor: pointer;
  background: var(--reader-background, #f6f9fc);
}

.upload-dropzone.dragging {
  border-color: var(--reader-muted, #64748b);
  border-style: solid;
}

.upload-dropzone:focus-visible {
  outline: 3px solid var(--reader-muted, #64748b);
  outline-offset: 2px;
}

.upload-hint {
  margin: 0.25rem 0 0;
  font-size: 0.85rem;
  color: var(--reader-muted, #64748b);
}

.upload-hidden-input {
  display: none;
}

.upload-error {
  color: #b91c1c;
  font-size: 0.9rem;
}

.upload-warning {
  color: var(--reader-muted, #64748b);
  font-size: 0.9rem;
}

.upload-file-info {
  margin-top: 1rem;
}

.upload-file-meta {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 0.75rem;
}

.upload-cover {
  margin: 0.75rem 0 0;
  text-align: center;
}

.upload-cover img {
  display: block;
  margin-inline: auto;
  max-width: 100%;
  max-height: 14rem;
  border: 1px solid var(--reader-border, #dbe4ee);
  border-radius: 0.5rem;
}

.upload-cover figcaption {
  margin-top: 0.35rem;
  font-size: 0.85rem;
  color: var(--reader-muted, #64748b);
}

.upload-form {
  display: grid;
  gap: 0.75rem;
  margin-top: 1rem;
}

.upload-field {
  display: grid;
  gap: 0.3rem;
  font-size: 0.9rem;
}

.upload-field input,
.upload-field textarea {
  padding: 0.55rem 0.7rem;
  border: 1px solid var(--reader-border, #dbe4ee);
  border-radius: 0.5rem;
  background: var(--reader-surface, #ffffff);
  color: var(--reader-text, #334155);
  font: inherit;
}

.upload-field input:focus-visible,
.upload-field textarea:focus-visible {
  outline: 3px solid var(--reader-muted, #64748b);
  outline-offset: 1px;
}

.upload-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 0.4rem;
}

.upload-chip {
  padding: 0.2rem 0.65rem;
  border: 1px solid var(--reader-border, #dbe4ee);
  border-radius: 999px;
  background: var(--reader-background, #f6f9fc);
  font-size: 0.8rem;
}

.upload-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.6rem;
  margin-top: 0.5rem;
}

.upload-cancel,
.upload-submit {
  padding: 0.6rem 1.1rem;
  border-radius: 0.6rem;
  cursor: pointer;
  font: inherit;
  font-weight: 600;
}

.upload-cancel {
  border: 1px solid var(--reader-border, #dbe4ee);
  background: transparent;
  color: var(--reader-heading, #172033);
}

.upload-submit {
  border: 1px solid var(--reader-heading, #172033);
  background: var(--reader-heading, #172033);
  color: var(--reader-surface, #ffffff);
}

.upload-submit:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.upload-cancel:focus-visible,
.upload-submit:focus-visible {
  outline: 3px solid var(--reader-muted, #64748b);
  outline-offset: 2px;
}

@media (max-width: 480px) {
  .upload-dialog {
    padding: 1rem;
  }

  .upload-actions {
    flex-direction: column-reverse;
  }

  .upload-cancel,
  .upload-submit {
    width: 100%;
  }
}
</style>
