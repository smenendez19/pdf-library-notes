<script setup lang="ts">
defineProps<{ label?: string }>();

defineEmits<{ click: [] }>();
</script>

<template>
  <button class="upload-pdf-button" type="button" @click="$emit('click')">
    <svg viewBox="0 0 24 24" aria-hidden="true" focusable="false">
      <path d="M12 16V4m0 0 4 4m-4-4-4 4" />
      <path d="M4 16v3a1 1 0 0 0 1 1h14a1 1 0 0 0 1-1v-3" />
    </svg>
    <span>{{ label ?? 'Subir PDF' }}</span>
  </button>
</template>

<style scoped>
.upload-pdf-button {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.6rem 1.1rem;
  border: 1px solid var(--reader-border, #dbe4ee);
  border-radius: 0.6rem;
  background: var(--reader-surface, #ffffff);
  color: var(--reader-heading, #172033);
  cursor: pointer;
  font: inherit;
  font-weight: 600;
  transition: background-color 0.2s ease, border-color 0.2s ease;
}

.upload-pdf-button svg {
  width: 1.2rem;
  height: 1.2rem;
  fill: none;
  stroke: currentColor;
  stroke-linecap: round;
  stroke-linejoin: round;
  stroke-width: 2;
}

.upload-pdf-button:hover {
  border-color: var(--reader-muted, #64748b);
  background: var(--reader-border, #dbe4ee);
}

.upload-pdf-button:focus-visible {
  outline: 3px solid var(--reader-muted, #64748b);
  outline-offset: 3px;
}
</style>
