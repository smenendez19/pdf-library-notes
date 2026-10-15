<script setup lang="ts">
import { nextTick, useTemplateRef, watch } from 'vue';

const props = defineProps<{ open: boolean; title: string; isDeleting?: boolean; error?: string }>();

const emit = defineEmits<{
  close: [];
  confirm: [];
}>();

const dialogRef = useTemplateRef<HTMLElement>('dialogRef');

function onOverlayClick(event: MouseEvent): void {
  if (event.target === event.currentTarget) emit('close');
}

function onEscape(event: KeyboardEvent): void {
  if (event.key === 'Escape' && !props.isDeleting) emit('close');
}

function onCancel(): void {
  emit('close');
}

function onConfirm(): void {
  emit('confirm');
}

watch(
  () => props.open,
  (isOpen) => {
    if (isOpen) {
      void nextTick(() => dialogRef.value?.focus());
    }
  },
);
</script>

<template>
  <Teleport to="body">
    <div v-if="open" class="delete-overlay" @click="onOverlayClick" @keydown="onEscape">
      <div
        ref="dialogRef"
        class="delete-dialog"
        role="dialog"
        aria-modal="true"
        aria-labelledby="delete-document-title"
        tabindex="-1"
      >
        <h2 id="delete-document-title">Eliminar documento</h2>
        <p class="delete-text">¿Eliminar "{{ title }}"? Esta acción no se puede deshacer.</p>
        <p v-if="error" class="delete-error" role="alert">{{ error }}</p>
        <div class="delete-actions">
          <button type="button" class="delete-cancel" :disabled="isDeleting" @click="onCancel">Cancelar</button>
          <button type="button" class="delete-confirm" :disabled="isDeleting" @click="onConfirm">
            {{ isDeleting ? 'Eliminando…' : 'Eliminar' }}
          </button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.delete-overlay {
  position: fixed;
  inset: 0;
  z-index: 50;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 1rem;
  background: rgb(15 23 42 / 0.45);
}

.delete-dialog {
  width: min(28rem, 100%);
  max-height: 90vh;
  overflow-y: auto;
  padding: 1.5rem;
  border: 1px solid var(--reader-border, #dbe4ee);
  border-radius: 0.9rem;
  background: var(--reader-surface, #ffffff);
  color: var(--reader-text, #334155);
  box-shadow: 0 12px 32px rgb(15 23 42 / 0.25);
}

.delete-dialog:focus {
  outline: none;
}

.delete-dialog:focus-visible {
  outline: 3px solid var(--reader-muted, #64748b);
  outline-offset: 2px;
}

.delete-dialog h2 {
  margin: 0 0 0.75rem;
  text-align: center;
  color: var(--reader-heading, #172033);
  font-size: 1.25rem;
}

.delete-text {
  margin: 0 0 1rem;
  overflow-wrap: break-word;
}

.delete-error {
  color: #b91c1c;
  font-size: 0.9rem;
}

.delete-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.6rem;
  margin-top: 0.5rem;
}

.delete-cancel,
.delete-confirm {
  padding: 0.6rem 1.1rem;
  border-radius: 0.6rem;
  cursor: pointer;
  font: inherit;
  font-weight: 600;
}

.delete-cancel {
  border: 1px solid var(--reader-border, #dbe4ee);
  background: transparent;
  color: var(--reader-heading, #172033);
}

.delete-confirm {
  border: 1px solid #b91c1c;
  background: #b91c1c;
  color: #ffffff;
}

.delete-confirm:disabled,
.delete-cancel:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.delete-cancel:focus-visible,
.delete-confirm:focus-visible {
  outline: 3px solid var(--reader-muted, #64748b);
  outline-offset: 2px;
}

@media (max-width: 480px) {
  .delete-dialog {
    padding: 1rem;
  }

  .delete-actions {
    flex-direction: column-reverse;
  }

  .delete-cancel,
  .delete-confirm {
    width: 100%;
  }
}
</style>
