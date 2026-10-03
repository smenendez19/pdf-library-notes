<script setup lang="ts">
import { computed } from 'vue';
import type { CatalogDocument } from '../../types/catalog';

const props = defineProps<{ document: CatalogDocument }>();

const emit = defineEmits<{ open: [id: string] }>();

const initial = computed(() => props.document.title.charAt(0).toUpperCase() || '?');

function handleOpen(): void {
  emit('open', props.document.id);
}

function handleKeydown(event: KeyboardEvent): void {
  if (event.key === 'Enter') {
    handleOpen();
  }
}
</script>

<template>
  <article
    class="card"
    tabindex="0"
    role="button"
    :aria-label="document.title"
    @click="handleOpen"
    @keydown="handleKeydown"
  >
    <div class="cover">
      <img
        v-if="document.image"
        :src="document.image"
        :alt="document.title"
        loading="lazy"
      />
      <div
        v-else
        class="cover-placeholder"
        aria-hidden="true"
      >
        {{ initial }}
      </div>
      <div class="card-overlay">
        <div class="tags">
          <span
            v-for="tag in document.tags"
            :key="tag"
            class="chip"
          >
            {{ tag }}
          </span>
        </div>
        <p class="meta">{{ document.author }} · {{ document.editorial }} · {{ document.upload_date }}</p>
        <p class="desc">{{ document.description ?? 'Sin descripción disponible.' }}</p>
      </div>
    </div>
    <h3 class="title">{{ document.title }}</h3>
    <p class="author">{{ document.author }}</p>
  </article>
</template>

<style scoped>
.card {
  border: 1px solid var(--reader-border, #dbe4ee);
  border-radius: 0.75rem;
  overflow: hidden;
  background: var(--reader-surface, #ffffff);
  cursor: pointer;
}

.card:focus-visible {
  outline: 3px solid var(--reader-muted, #64748b);
  outline-offset: 2px;
}

.cover {
  position: relative;
  aspect-ratio: 3 / 4;
  overflow: hidden;
  background: var(--reader-border, #dbe4ee);
}

.cover img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

.cover-placeholder {
  width: 100%;
  height: 100%;
  display: grid;
  place-items: center;
  font-size: 3rem;
  font-weight: 700;
  color: var(--reader-muted, #64748b);
}

.card-overlay {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  justify-content: flex-end;
  gap: 0.5rem;
  padding: 0.75rem;
  background: rgb(0 0 0 / 0.75);
  color: #fff;
  opacity: 0;
  transition: opacity 0.2s ease;
  pointer-events: none;
  overflow: hidden;
}

.card:hover .card-overlay,
.card:focus .card-overlay,
.card:focus-within .card-overlay {
  opacity: 1;
}

.tags {
  display: flex;
  flex-wrap: wrap;
  gap: 0.25rem;
}

.chip {
  padding: 0.1rem 0.5rem;
  border-radius: 999px;
  background: rgb(255 255 255 / 0.2);
  font-size: 0.75rem;
}

.meta {
  margin: 0;
  font-size: 0.8rem;
}

.desc {
  margin: 0;
  font-size: 0.8rem;
  display: -webkit-box;
  -webkit-line-clamp: 4;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.title {
  margin: 0.5rem 0.75rem 0;
  font-size: 0.95rem;
}

.author {
  margin: 0.25rem 0.75rem 0.75rem;
  font-size: 0.85rem;
  color: var(--reader-muted, #64748b);
}
</style>
