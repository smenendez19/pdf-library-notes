<script setup lang="ts">
import { computed } from "vue";
import type { StickyNote } from "../../composables/useStickyNotes";
import DraggableStickyNote from "./DraggableStickyNote.vue";

const props = defineProps<{
  page: number;
  notes: StickyNote[];
}>();

const emit = defineEmits<{
  move: [payload: { id: number; x: number; y: number }];
  resize: [payload: { id: number; width: number; height: number }];
}>();

const visible = computed(() =>
  props.notes.filter((note) => note.page === props.page && note.visible),
);

function forwardMove(payload: { id: number; x: number; y: number }): void {
  emit("move", payload);
}

function forwardResize(payload: {
  id: number;
  width: number;
  height: number;
}): void {
  emit("resize", payload);
}
</script>

<template>
  <div class="sticky-overlay" aria-hidden="false">
    <DraggableStickyNote
      v-for="note in visible"
      :key="note.id"
      :note="note"
      @move="forwardMove"
      @resize="forwardResize"
    />
  </div>
</template>

<style scoped>
.sticky-overlay {
  position: absolute;
  inset: 0;
  pointer-events: none;
  z-index: 2;
  overflow: visible;
}
</style>
