<script setup lang="ts">
import { useRouter } from 'vue-router';
import CatalogGrid from '../components/catalog/CatalogGrid.vue';
import CatalogLoadingModal from '../components/catalog/CatalogLoadingModal.vue';
import CatalogPagination from '../components/catalog/CatalogPagination.vue';
import CatalogSearch from '../components/catalog/CatalogSearch.vue';
import ThemeToggleButton from '../components/common/ThemeToggleButton.vue';
import { useCatalog } from '../composables/useCatalog';
import { useTheme } from '../composables/useTheme';
import type { CatalogDocument } from '../types/catalog';

const router = useRouter();
const { page, search, documents, totalPages, isLoading, error, load } = useCatalog();
const { isDarkMode, toggleTheme } = useTheme();

function openDocument(doc: CatalogDocument): void {
  void router.push({ name: 'reader', query: { doc: doc.fileUrl, title: doc.title } });
}

function openById(id: string): void {
  const doc = documents.value.find((item) => item.id === id);
  if (doc) {
    openDocument(doc);
  }
}
</script>

<template>
  <main class="catalog-page" :class="{ 'catalog-page--dark': isDarkMode }">
    <header class="catalog-header">
      <h1>Library PDF reader</h1>
      <ThemeToggleButton :is-dark-mode="isDarkMode" @toggle="toggleTheme" />
    </header>
    <CatalogSearch v-model="search" />
    <CatalogLoadingModal :visible="isLoading" />
    <div v-if="error" class="catalog-error">
      <p role="alert">{{ error }}</p>
      <button type="button" @click="load">Reintentar</button>
    </div>
    <CatalogGrid :documents="documents" @open="openById" />
    <CatalogPagination :page="page" :total-pages="totalPages" @update:page="page = $event" />
  </main>
</template>

<style scoped>
.catalog-page {
  --reader-background: #f4f7fb;
  --reader-surface: #ffffff;
  --reader-border: #dbe4ee;
  --reader-heading: #172033;
  --reader-text: #334155;
  --reader-muted: #64748b;
  --reader-canvas: #e2e8f0;
  min-height: 100svh;
  box-sizing: border-box;
  padding: 2rem 1.5rem;
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
  background: var(--reader-background);
  color: var(--reader-text);
  transition: background-color 0.2s ease, color 0.2s ease;
}

.catalog-page--dark {
  --reader-background: #20252d;
  --reader-surface: #2b313b;
  --reader-border: #46515f;
  --reader-heading: #f8fafc;
  --reader-text: #f8fafc;
  --reader-muted: #cbd5e1;
  --reader-canvas: #161b22;
}

.catalog-header {
  display: flex;
  align-items: center;
  gap: 1rem;
}

.catalog-header h1 {
  flex: 1;
  margin: 0;
  text-align: center;
  color: var(--reader-heading);
  font-size: clamp(1.5rem, 2.5vw, 2rem);
  line-height: 1.2;
  letter-spacing: -0.025em;
}

.catalog-error {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.catalog-error p {
  margin: 0;
}

.catalog-error button {
  padding: 0.4rem 0.9rem;
  border: 1px solid var(--reader-border);
  border-radius: 0.5rem;
  background: var(--reader-surface);
  color: var(--reader-heading);
  font: inherit;
  cursor: pointer;
}

@media (max-width: 640px) {
  .catalog-page {
    padding: 1.25rem 0.75rem;
  }
}
</style>
