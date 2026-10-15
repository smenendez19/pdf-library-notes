<script setup lang="ts">
defineProps<{ isDarkMode: boolean }>();

defineEmits<{ toggle: [] }>();
</script>

<template>
  <button
    class="theme-toggle"
    type="button"
    :aria-pressed="isDarkMode"
    :aria-label="isDarkMode ? 'Activar modo claro' : 'Activar modo oscuro'"
    @click="$emit('toggle')"
  >
    <svg v-if="isDarkMode" viewBox="0 0 24 24" aria-hidden="true" focusable="false">
      <circle cx="12" cy="12" r="4" />
      <path d="M12 2v2M12 20v2M4.93 4.93l1.41 1.41M17.66 17.66l1.41 1.41M2 12h2M20 12h2M4.93 19.07l1.41-1.41M17.66 6.34l1.41-1.41" />
    </svg>
    <svg v-else viewBox="0 0 24 24" aria-hidden="true" focusable="false">
      <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79Z" />
    </svg>
  </button>
</template>

<style scoped>
.theme-toggle {
  width: 2.5rem;
  height: 2.5rem;
  display: grid;
  place-items: center;
  padding: 0;
  border: 1px solid var(--reader-border, #dbe4ee);
  border-radius: 50%;
  background: var(--reader-surface, #ffffff);
  color: var(--reader-heading, #172033);
  cursor: pointer;
  font: inherit;
  font-weight: 600;
  transition: background-color 0.2s ease, color 0.2s ease, border-color 0.2s ease;
}

.theme-toggle svg {
  width: 1.25rem;
  height: 1.25rem;
  fill: none;
  stroke: currentColor;
  stroke-linecap: round;
  stroke-linejoin: round;
  stroke-width: 2;
}

.theme-toggle:hover {
  border-color: var(--reader-muted, #64748b);
  background: var(--reader-border, #dbe4ee);
}

.theme-toggle:focus-visible {
  outline: 3px solid var(--reader-muted, #64748b);
  outline-offset: 3px;
}
</style>
