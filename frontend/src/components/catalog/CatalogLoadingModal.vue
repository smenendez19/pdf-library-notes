<script setup lang="ts">
defineProps<{ visible: boolean }>();
</script>

<template>
  <div v-if="visible" class="catalog-loading-overlay">
    <div class="catalog-loading-pill" role="status" aria-live="polite">
      <span class="catalog-loading-spinner" aria-hidden="true" />
      <span>Cargando catálogo…</span>
    </div>
  </div>
</template>

<style scoped>
.catalog-loading-overlay {
  position: fixed;
  inset: 0;
  z-index: 50;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 1rem;
  background: rgb(15 23 42 / 0.45);
  backdrop-filter: blur(2px);
}

.catalog-loading-pill {
  display: inline-flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.9rem 1.5rem;
  border: 1px solid var(--reader-border, #dbe4ee);
  border-radius: 999px;
  background: var(--reader-surface, #ffffff);
  color: var(--reader-heading, #172033);
  box-shadow: 0 12px 32px rgb(15 23 42 / 0.25);
}

.catalog-loading-spinner {
  width: 1.25rem;
  height: 1.25rem;
  border-radius: 50%;
  border: 3px solid var(--reader-border, #dbe4ee);
  border-top-color: var(--reader-muted, #64748b);
  animation: catalog-loading-spin 0.8s linear infinite;
}

@keyframes catalog-loading-spin {
  to {
    transform: rotate(360deg);
  }
}
</style>
