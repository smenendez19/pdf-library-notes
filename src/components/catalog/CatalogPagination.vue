<script setup lang="ts">
import { computed } from 'vue';

const props = defineProps<{ page: number; totalPages: number }>();

const emit = defineEmits<{ 'update:page': [page: number] }>();

const safeTotal = computed(() => Math.max(1, props.totalPages));

function clamp(value: number): number {
  return Math.min(safeTotal.value, Math.max(1, value));
}

function goTo(value: number): void {
  const target = clamp(value);
  if (target !== props.page) {
    emit('update:page', target);
  }
}
</script>

<template>
  <nav
    class="pagination"
    aria-label="Paginación del catálogo"
  >
    <button
      v-if="safeTotal > 1"
      type="button"
      aria-label="Ir a la primera página"
      :disabled="page <= 1"
      @click="goTo(1)"
    >
      «
    </button>
    <button
      type="button"
      aria-label="Ir a la página anterior"
      :disabled="page <= 1"
      @click="goTo(page - 1)"
    >
      Anterior
    </button>
    <span class="divider" aria-hidden="true" />
    <span aria-live="polite">Página {{ page }} de {{ safeTotal }}</span>
    <span class="divider" aria-hidden="true" />
    <button
      type="button"
      aria-label="Ir a la página siguiente"
      :disabled="page >= safeTotal"
      @click="goTo(page + 1)"
    >
      Siguiente
    </button>
    <button
      v-if="safeTotal > 1"
      type="button"
      aria-label="Ir a la última página"
      :disabled="page >= safeTotal"
      @click="goTo(safeTotal)"
    >
      »
    </button>
  </nav>
</template>

<style scoped>
.pagination {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  padding: 1rem 0;
  border-top: 1px solid var(--reader-border);
}

.pagination button {
  padding: 0.4rem 0.9rem;
  border: 1px solid var(--reader-border);
  border-radius: 0.5rem;
  background: var(--reader-surface);
  color: var(--reader-heading);
  font: inherit;
  cursor: pointer;
  transition: background-color 0.15s ease, border-color 0.15s ease, color 0.15s ease;
}

.pagination button:hover:not(:disabled) {
  border-color: var(--reader-muted);
  background: var(--reader-canvas);
}

.pagination button:focus-visible {
  outline: 3px solid var(--reader-muted);
  outline-offset: 2px;
}

.pagination button:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.divider {
  width: 1px;
  height: 1.5rem;
  background: var(--reader-border);
}
</style>
