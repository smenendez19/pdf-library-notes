<script setup lang="ts">
import type { CatalogDocument } from '../../types/catalog';
import CatalogCard from './CatalogCard.vue';

defineProps<{ documents: CatalogDocument[] }>();

defineEmits<{ open: [id: string] }>();
</script>

<template>
  <p
    v-if="documents.length === 0"
    class="empty"
  >
    No se encontraron documentos.
  </p>
  <div
    v-else
    class="catalog-grid"
  >
    <CatalogCard
      v-for="doc in documents"
      :key="doc.id"
      :document="doc"
      @open="$emit('open', $event)"
    />
  </div>
</template>

<style scoped>
.catalog-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
  gap: 1rem;
}

.empty {
  margin: 0;
  padding: 2rem 1rem;
  text-align: center;
  color: var(--reader-muted, #64748b);
}
</style>
