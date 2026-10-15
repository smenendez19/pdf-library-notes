import { ref, watch } from 'vue';
import { fetchCatalog } from '../services/catalogService';
import type { CatalogDocument } from '../types/catalog';

export function useCatalog() {
  const page = ref(1);
  const itemsPerPage = ref(12);
  const search = ref('');
  const debouncedSearch = ref('');
  const documents = ref<CatalogDocument[]>([]);
  const totalPages = ref(1);
  const totalItems = ref(0);
  const isLoading = ref(false);
  const error = ref('');

  watch(search, (value, _oldValue, onCleanup) => {
    const timer = setTimeout(() => {
      debouncedSearch.value = value;
    }, 300);
    onCleanup(() => clearTimeout(timer));
  });

  async function load(): Promise<void> {
    isLoading.value = true;
    error.value = '';
    try {
      const response = await fetchCatalog({
        page: page.value,
        items: itemsPerPage.value,
        search: debouncedSearch.value,
      });
      documents.value = response.data;
      totalPages.value = response.max_pages;
      totalItems.value = response.max_items;
      if (page.value !== response.page) {
        page.value = response.page;
      }
    } catch {
      error.value = 'No se pudo cargar el catálogo. Inténtalo de nuevo.';
      documents.value = [];
    } finally {
      isLoading.value = false;
    }
  }

  watch([page, itemsPerPage, debouncedSearch], (newValues, oldValues) => {
    const newSearch = newValues[2];
    const oldSearch = oldValues[2];
    const newPage = newValues[0];
    if (newSearch !== oldSearch && newPage !== 1) {
      page.value = 1;
      return;
    }
    void load();
  });

  void load();

  return {
    page,
    itemsPerPage,
    search,
    debouncedSearch,
    documents,
    totalPages,
    totalItems,
    isLoading,
    error,
    load,
  };
}
