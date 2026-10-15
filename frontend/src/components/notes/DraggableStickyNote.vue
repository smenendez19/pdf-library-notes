<script setup lang="ts">
import DOMPurify from "dompurify";
import { marked } from "marked";
import { computed, ref } from "vue";
import {
  DEFAULT_HEIGHT,
  DEFAULT_WIDTH,
  MAX_HEIGHT,
  MAX_WIDTH,
  MIN_HEIGHT,
  MIN_WIDTH,
  noteColor,
  type StickyNote,
} from "../../composables/useStickyNotes";

const props = defineProps<{
  note: StickyNote;
  scale?: number;
}>();

const emit = defineEmits<{
  move: [payload: { id: number; x: number; y: number }];
  resize: [payload: { id: number; width: number; height: number }];
}>();

const cardElement = ref<HTMLElement | null>(null);

let dragging = false;
let startClientX = 0;
let startClientY = 0;
let startNoteX = 0;
let startNoteY = 0;
let originX = 0;
let originY = 0;

let resizing = false;
let resizeStartClientX = 0;
let resizeStartClientY = 0;
let resizeStartWidth = 0;
let resizeStartHeight = 0;

interface PositionLimits {
  minX: number;
  maxX: number;
  minY: number;
  maxY: number;
}

interface SizeLimits {
  maxWidth: number;
  maxHeight: number;
}

const renderedMarkdown = computed(() =>
  DOMPurify.sanitize(marked.parse(props.note.content) as string),
);

const ariaLabel = computed(() => {
  const excerpt = props.note.content.trim().slice(0, 80) || "Nota vacía";
  return `Nota en página ${props.note.page}: ${excerpt}`;
});

const cardStyle = computed(() => ({
  left: `${props.note.x}%`,
  top: `${props.note.y}%`,
  backgroundColor: noteColor(props.note),
  width: `${props.note.width ?? DEFAULT_WIDTH}px`,
  height: `${props.note.height ?? DEFAULT_HEIGHT}px`,
}));

function getPageElement(): HTMLElement | null {
  const overlay = cardElement.value?.parentElement;
  if (!overlay) return null;
  if (overlay.offsetParent instanceof HTMLElement)
    return overlay.offsetParent;
  return overlay.parentElement instanceof HTMLElement
    ? overlay.parentElement
    : null;
}

function getPageRect(): DOMRect | null {
  const page = getPageElement();
  if (!page) return null;
  const rect = page.getBoundingClientRect();
  if (
    !Number.isFinite(rect.width) ||
    !Number.isFinite(rect.height) ||
    rect.width <= 0 ||
    rect.height <= 0
  ) {
    return null;
  }
  return rect;
}

function getViewportElement(): HTMLElement | null {
  const fromClosest = cardElement.value?.closest(".pdf-reader__viewport");
  if (fromClosest instanceof HTMLElement) return fromClosest;
  if (typeof document !== "undefined") {
    const queried = document.querySelector(".pdf-reader__viewport");
    if (queried instanceof HTMLElement) return queried;
  }
  return null;
}

function getViewportLimits(): PositionLimits {
  const fallback: PositionLimits = { minX: 0, maxX: 100, minY: 0, maxY: 100 };
  const pageRect = getPageRect();
  if (!pageRect) return fallback;
  const noteRect = cardElement.value?.getBoundingClientRect();
  if (
    !noteRect ||
    !Number.isFinite(noteRect.width) ||
    !Number.isFinite(noteRect.height)
  ) {
    return fallback;
  }
  const viewportElement = getViewportElement();
  const viewportRect = viewportElement
    ? viewportElement.getBoundingClientRect()
    : getPageRect();
  if (
    !viewportRect ||
    !Number.isFinite(viewportRect.left) ||
    !Number.isFinite(viewportRect.right) ||
    !Number.isFinite(viewportRect.top) ||
    !Number.isFinite(viewportRect.bottom)
  ) {
    return fallback;
  }
  const clampAbsolute = (value: number): number => {
    if (!Number.isFinite(value)) return 0;
    return Math.min(200, Math.max(-100, value));
  };
  return {
    minX: clampAbsolute(
      ((viewportRect.left - pageRect.left) / pageRect.width) * 100,
    ),
    maxX: clampAbsolute(
      ((viewportRect.right - noteRect.width - pageRect.left) /
        pageRect.width) *
        100,
    ),
    minY: clampAbsolute(
      ((viewportRect.top - pageRect.top) / pageRect.height) * 100,
    ),
    maxY: clampAbsolute(
      ((viewportRect.bottom - noteRect.height - pageRect.top) /
        pageRect.height) *
        100,
    ),
  };
}

function getSizeLimits(): SizeLimits {
  const pageRect = getPageRect();
  if (!pageRect) return { maxWidth: MAX_WIDTH, maxHeight: MAX_HEIGHT };

  let horizontalBorder = 0;
  let verticalBorder = 0;
  const card = cardElement.value;
  if (card && typeof window !== "undefined") {
    const style = window.getComputedStyle(card);
    if (style.boxSizing !== "border-box") {
      horizontalBorder =
        parseFloat(style.borderLeftWidth) + parseFloat(style.borderRightWidth);
      verticalBorder =
        parseFloat(style.borderTopWidth) + parseFloat(style.borderBottomWidth);
      if (!Number.isFinite(horizontalBorder)) horizontalBorder = 0;
      if (!Number.isFinite(verticalBorder)) verticalBorder = 0;
    }
  }

  let baseWidth = pageRect.width;
  const viewportElement = getViewportElement();
  if (viewportElement) {
    const viewportRect = viewportElement.getBoundingClientRect();
    if (
      Number.isFinite(viewportRect.width) &&
      viewportRect.width > 0
    ) {
      baseWidth = Math.max(pageRect.width, viewportRect.width);
    }
  }

  return {
    maxWidth: Math.min(
      MAX_WIDTH,
      Math.max(0, baseWidth - horizontalBorder),
    ),
    maxHeight: Math.min(
      MAX_HEIGHT,
      Math.max(0, pageRect.height - verticalBorder),
    ),
  };
}

function clampNotePosition(value: number, min: number, max: number): number {
  const lowerBound = Number.isFinite(min) ? min : 0;
  const upperBound = Number.isFinite(max) ? max : 100;
  if (!Number.isFinite(value)) return lowerBound;
  return Math.min(upperBound, Math.max(lowerBound, value));
}

function clampSize(value: number, min: number, max: number): number {
  if (!Number.isFinite(value)) return min;
  return Math.min(max, Math.max(min, value));
}

function onHandlePointerDown(event: PointerEvent): void {
  if (event.pointerType === "mouse" && event.button !== 0) return;
  const handle = event.currentTarget as HTMLElement | null;
  handle?.setPointerCapture(event.pointerId);
  dragging = true;
  startClientX = event.clientX;
  startClientY = event.clientY;
  const { minX, maxX, minY, maxY } = getViewportLimits();
  startNoteX = clampNotePosition(props.note.x, minX, maxX);
  startNoteY = clampNotePosition(props.note.y, minY, maxY);
  originX = startNoteX;
  originY = startNoteY;
  event.preventDefault();
}

function onHandlePointerMove(event: PointerEvent): void {
  if (!dragging) return;
  const rect = getPageRect();
  if (!rect) return;
  const { minX, maxX, minY, maxY } = getViewportLimits();
  const nextX = clampNotePosition(
    startNoteX + ((event.clientX - startClientX) / rect.width) * 100,
    minX,
    maxX,
  );
  const nextY = clampNotePosition(
    startNoteY + ((event.clientY - startClientY) / rect.height) * 100,
    minY,
    maxY,
  );
  emit("move", { id: props.note.id, x: nextX, y: nextY });
}

function stopDrag(event: PointerEvent): void {
  if (!dragging) return;
  dragging = false;
  const handle = event.currentTarget as HTMLElement | null;
  if (handle?.hasPointerCapture(event.pointerId))
    handle.releasePointerCapture(event.pointerId);
}

function onResizePointerDown(event: PointerEvent): void {
  if (event.pointerType === "mouse" && event.button !== 0) return;
  const handle = event.currentTarget as HTMLElement | null;
  handle?.setPointerCapture(event.pointerId);
  resizing = true;
  resizeStartClientX = event.clientX;
  resizeStartClientY = event.clientY;
  resizeStartWidth = props.note.width ?? DEFAULT_WIDTH;
  resizeStartHeight = props.note.height ?? DEFAULT_HEIGHT;
  event.preventDefault();
  event.stopPropagation();
}

function onResizePointerMove(event: PointerEvent): void {
  if (!resizing) return;
  const { maxWidth, maxHeight } = getSizeLimits();
  const nextWidth = clampSize(
    resizeStartWidth + (event.clientX - resizeStartClientX),
    MIN_WIDTH,
    maxWidth,
  );
  const nextHeight = clampSize(
    resizeStartHeight + (event.clientY - resizeStartClientY),
    MIN_HEIGHT,
    maxHeight,
  );
  emit("resize", { id: props.note.id, width: nextWidth, height: nextHeight });
}

function stopResize(event: PointerEvent): void {
  if (!resizing) return;
  resizing = false;
  const handle = event.currentTarget as HTMLElement | null;
  if (handle?.hasPointerCapture(event.pointerId))
    handle.releasePointerCapture(event.pointerId);
}

function onCardKeydown(event: KeyboardEvent): void {
  if (event.ctrlKey) {
    const step = 10;
    const currentWidth = props.note.width ?? DEFAULT_WIDTH;
    const currentHeight = props.note.height ?? DEFAULT_HEIGHT;
    const { maxWidth, maxHeight } = getSizeLimits();
    const boundedWidth = clampSize(currentWidth, MIN_WIDTH, maxWidth);
    const boundedHeight = clampSize(currentHeight, MIN_HEIGHT, maxHeight);
    switch (event.key) {
      case "ArrowLeft":
        event.preventDefault();
        emit("resize", {
          id: props.note.id,
          width: clampSize(currentWidth - step, MIN_WIDTH, maxWidth),
          height: boundedHeight,
        });
        break;
      case "ArrowRight":
        event.preventDefault();
        emit("resize", {
          id: props.note.id,
          width: clampSize(currentWidth + step, MIN_WIDTH, maxWidth),
          height: boundedHeight,
        });
        break;
      case "ArrowUp":
        event.preventDefault();
        emit("resize", {
          id: props.note.id,
          width: boundedWidth,
          height: clampSize(currentHeight - step, MIN_HEIGHT, maxHeight),
        });
        break;
      case "ArrowDown":
        event.preventDefault();
        emit("resize", {
          id: props.note.id,
          width: boundedWidth,
          height: clampSize(currentHeight + step, MIN_HEIGHT, maxHeight),
        });
        break;
    }
    return;
  }
  const step = event.shiftKey ? 5 : 1;
  const { minX, maxX, minY, maxY } = getViewportLimits();
  switch (event.key) {
    case "ArrowLeft":
      event.preventDefault();
      emit("move", {
        id: props.note.id,
        x: clampNotePosition(props.note.x - step, minX, maxX),
        y: clampNotePosition(props.note.y, minY, maxY),
      });
      break;
    case "ArrowRight":
      event.preventDefault();
      emit("move", {
        id: props.note.id,
        x: clampNotePosition(props.note.x + step, minX, maxX),
        y: clampNotePosition(props.note.y, minY, maxY),
      });
      break;
    case "ArrowUp":
      event.preventDefault();
      emit("move", {
        id: props.note.id,
        x: clampNotePosition(props.note.x, minX, maxX),
        y: clampNotePosition(props.note.y - step, minY, maxY),
      });
      break;
    case "ArrowDown":
      event.preventDefault();
      emit("move", {
        id: props.note.id,
        x: clampNotePosition(props.note.x, minX, maxX),
        y: clampNotePosition(props.note.y + step, minY, maxY),
      });
      break;
    case "Escape":
      if (dragging) {
        event.preventDefault();
        dragging = false;
        const limits = getViewportLimits();
        emit("move", {
          id: props.note.id,
          x: clampNotePosition(originX, limits.minX, limits.maxX),
          y: clampNotePosition(originY, limits.minY, limits.maxY),
        });
      }
      break;
  }
}
</script>

<template>
  <div
    ref="cardElement"
    class="sticky-note"
    :style="cardStyle"
    tabindex="0"
    role="dialog"
    :aria-label="ariaLabel"
    @keydown="onCardKeydown"
  >
    <div
      class="drag-handle"
      aria-label="Mover nota"
      title="Mover"
      @pointerdown="onHandlePointerDown"
      @pointermove="onHandlePointerMove"
      @pointerup="stopDrag"
      @pointercancel="stopDrag"
    >
      <span aria-hidden="true">⠿</span>
    </div>
    <div class="sticky-note__content" v-html="renderedMarkdown"></div>
    <div
      class="resize-handle"
      aria-label="Redimensionar nota"
      title="Redimensionar"
      @pointerdown="onResizePointerDown"
      @pointermove="onResizePointerMove"
      @pointerup="stopResize"
      @pointercancel="stopResize"
    ></div>
  </div>
</template>

<style scoped>
.sticky-note {
  position: absolute;
  box-sizing: border-box;
  width: 200px;
  max-width: min(70%, 100%);
  min-width: min(140px, 100%);
  max-height: 100%;
  min-height: min(100px, 100%);
  display: flex;
  flex-direction: column;
  pointer-events: auto;
  border: 1px solid #f1d58a;
  border-radius: 0.35rem;
  box-shadow: 0 2px 8px rgb(15 23 42 / 20%);
  color: #1f2937;
  font-size: 0.875rem;
}

.drag-handle {
  touch-action: none;
  cursor: grab;
  user-select: none;
  display: flex;
  align-items: center;
  gap: 0.35rem;
  padding: 0.25rem 0.5rem;
  border-bottom: 1px solid rgb(15 23 42 / 10%);
  background: rgb(255 255 255 / 35%);
  font-size: 0.75rem;
  font-weight: 600;
}

.drag-handle:active {
  cursor: grabbing;
}

.sticky-note__content {
  flex: 1;
  min-height: 0;
  padding: 0.5rem;
  overflow: auto;
  overflow-wrap: anywhere;
}

.resize-handle {
  position: absolute;
  right: 2px;
  bottom: 2px;
  width: 16px;
  height: 16px;
  cursor: nwse-resize;
  touch-action: none;
  background: linear-gradient(
    135deg,
    transparent 50%,
    rgb(15 23 42 / 45%) 50%
  );
  border-bottom-right-radius: 0.3rem;
}
</style>
