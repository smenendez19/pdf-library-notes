<script setup lang="ts">
defineProps<{ modelValue: string }>();

defineEmits<{ 'update:modelValue': [value: string] }>();
</script>

<template>
  <div class="catalog-search">
    <input
      type="search"
      class="catalog-search-input"
      :value="modelValue"
      aria-label="Buscar documentos"
      placeholder="Buscar por título, autor o etiqueta…"
      @input="$emit('update:modelValue', ($event.target as HTMLInputElement).value)"
    />
    <button
      v-if="modelValue"
      type="button"
      class="catalog-search-clear"
      aria-label="Limpiar búsqueda"
      @click="$emit('update:modelValue', '')"
    >
      ×
    </button>
  </div>
</template>

<style scoped>
.catalog-search {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.catalog-search-input {
  flex: 1;
  padding: 0.5rem 0.75rem;
  font: inherit;
  border: 1px solid var(--reader-border, #dbe4ee);
  border-radius: 0.5rem;
  background: var(--reader-surface, #ffffff);
  color: inherit;
}

.catalog-search-clear {
  padding: 0.25rem 0.6rem;
  font: inherit;
  font-size: 1.1rem;
  line-height: 1;
  border: 1px solid var(--reader-border, #dbe4ee);
  border-radius: 0.5rem;
  background: transparent;
  color: inherit;
  cursor: pointer;
}
</style>
