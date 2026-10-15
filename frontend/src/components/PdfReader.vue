<script setup lang="ts">
import {
  nextTick,
  onBeforeUnmount,
  onMounted,
  reactive,
  ref,
  shallowRef,
  watch,
  computed,
} from "vue";
import {
  getDocument,
  GlobalWorkerOptions,
  type PDFDocumentProxy,
  type PDFPageProxy,
  type RenderTask,
} from "pdfjs-dist";
import workerSrc from "pdfjs-dist/build/pdf.worker.mjs?url";

GlobalWorkerOptions.workerSrc = workerSrc;

const props = defineProps<{ src: string }>();
const emit = defineEmits<{ "page-change": [page: number] }>();

const canvasContainer = ref<HTMLElement | null>(null);
const documentProxy = shallowRef<PDFDocumentProxy | null>(null);
const totalPages = ref(0);
const currentPage = ref(1);
const virtualStart = ref(1);
const virtualEnd = ref(0);
const estimatedPageHeight = ref(900);
const pageInput = ref("1");
const scale = ref(1);
const searchQuery = ref("");
const searchStatus = ref("");
const isLoading = ref(true);
const isInitialRenderDone = ref(false);
const errorMessage = ref("");
const pageElements = new Map<number, HTMLElement>();
const pageHeightCache = new Map<number, number>();
const pageMetricsVersion = ref(0);
const canvases = new Map<number, HTMLCanvasElement>();
const visiblePages = new Set<number>();
const preloadPages = new Set<number>();
const renderedPages = reactive(new Set<number>());
const failedPages = reactive(new Set<number>());
type CachedTextItem = {
  str: string;
  width: number;
  height: number;
  transform: [number, number, number, number, number, number];
};
type CachedPageText = { items: CachedTextItem[] };
type SearchOccurrence = {
  page: number;
  rect: [number, number, number, number];
};
const textCache = new Map<number, CachedPageText>();
const searchOccurrences = ref<SearchOccurrence[]>([]);
const activeOccurrenceIndex = ref(-1);
const activeOccurrence = computed(
  () => searchOccurrences.value[activeOccurrenceIndex.value] ?? null,
);
const renderTasks = new Map<number, RenderTask>();
const renderPromises = new Map<number, Promise<void>>();
type RenderRequest = {
  page: number;
  resolve: () => void;
  reject: (error: unknown) => void;
  promise: Promise<void>;
};
const renderQueue: RenderRequest[] = [];
let activeRenderCount = 0;
const MAX_CONCURRENT_RENDERS = 2;
const PRELOAD_MARGIN = 350;
const RETAIN_MARGIN = PRELOAD_MARGIN * 2;
const VIRTUAL_RADIUS = 2;
let loadingTask: ReturnType<typeof getDocument> | null = null;
let observer: IntersectionObserver | null = null;
let observerRefreshPending = false;
let loadVersion = 0;
let renderVersion = 0;
let zoomVersion = 0;
let automaticPageDetectionEnabled = false;

const isNormalCancellation = (error: unknown) => {
  const name = error instanceof Error ? error.name : "";
  return name === "RenderingCancelledException" || name === "AbortException";
};

const getErrorMessage = (error: unknown) =>
  error instanceof Error && error.message ? error.message : "Error desconocido";

const setPageElement = (page: number, element: unknown) => {
  if (element instanceof HTMLElement) {
    pageElements.set(page, element);
    measurePage(page);
  } else pageElements.delete(page);
};

const getPageHeight = (page: number) =>
  pageHeightCache.get(page) ?? estimatedPageHeight.value;

const measurePage = (page: number) => {
  const element = pageElements.get(page);
  if (!element) return;
  const height = element.getBoundingClientRect().height + 16;
  if (height < 16 || Math.abs((pageHeightCache.get(page) ?? 0) - height) < 1)
    return;
  pageHeightCache.set(page, height);
  pageMetricsVersion.value += 1;
};

const measureMountedPages = () => {
  for (const page of pageElements.keys()) measurePage(page);
};

const setCanvas = (page: number, element: unknown) => {
  if (element instanceof HTMLCanvasElement) {
    canvases.set(page, element);
    if (!renderedPages.has(page) && !failedPages.has(page))
      requestActivePages([page]);
  } else canvases.delete(page);
};

const cancelRenders = () => {
  renderVersion += 1;
  for (const task of renderTasks.values()) task.cancel();
  renderTasks.clear();
  for (const request of renderQueue) request.resolve();
  renderQueue.length = 0;
  renderPromises.clear();
};

const releasePage = (page: number) => {
  const canvas = canvases.get(page);
  if (canvas) {
    const context = canvas.getContext("2d");
    context?.clearRect(0, 0, canvas.width, canvas.height);
    canvas.width = 0;
    canvas.height = 0;
    canvas.style.removeProperty("width");
    canvas.style.removeProperty("height");
  }
  renderedPages.delete(page);
  failedPages.delete(page);
  pageElements.delete(page);
  void documentProxy.value
    ?.getPage(page)
    .then((loadedPage) => loadedPage.cleanup())
    .catch(() => {});
};

const cancelPageRender = (page: number) => {
  renderTasks.get(page)?.cancel();
  renderTasks.delete(page);
  for (let index = renderQueue.length - 1; index >= 0; index -= 1) {
    if (renderQueue[index].page !== page) continue;
    const [request] = renderQueue.splice(index, 1);
    renderPromises.delete(page);
    request.resolve();
  }
};

const virtualPages = computed(() => {
  const pages: number[] = [];
  for (let page = virtualStart.value; page <= virtualEnd.value; page += 1)
    pages.push(page);
  return pages;
});

const hasPendingPages = computed(() =>
  virtualPages.value.some(
    (page) => !renderedPages.has(page) && !failedPages.has(page),
  ),
);

const showLoadingOverlay = computed(() =>
  isLoading.value ? true : !isInitialRenderDone.value ? hasPendingPages.value : false,
);

const getSpacerHeight = (firstPage: number, lastPage: number) => {
  void pageMetricsVersion.value;
  if (firstPage > lastPage) return 0;
  let height = 0;
  for (let page = firstPage; page <= lastPage; page += 1)
    height += getPageHeight(page);
  return height;
};

const updateVirtualWindow = (page: number) => {
  if (!totalPages.value) {
    return;
  }
  measureMountedPages();
  const nextStart = Math.max(
    1,
    Math.min(page - VIRTUAL_RADIUS, totalPages.value),
  );
  const nextEnd = Math.min(totalPages.value, nextStart + VIRTUAL_RADIUS * 2);
  if (nextStart === virtualStart.value && nextEnd === virtualEnd.value) return;
  virtualStart.value = nextStart;
  virtualEnd.value = nextEnd;
  const mountedPages = new Set(
    Array.from(
      { length: nextEnd - nextStart + 1 },
      (_, index) => nextStart + index,
    ),
  );
  for (const pageNumber of [...pageElements.keys()]) {
    if (!mountedPages.has(pageNumber)) pageElements.delete(pageNumber);
  }
  const trackedPages = new Set([
    ...canvases.keys(),
    ...renderTasks.keys(),
    ...renderPromises.keys(),
    ...renderedPages,
  ]);
  for (const pageNumber of trackedPages) {
    if (!mountedPages.has(pageNumber)) {
      cancelPageRender(pageNumber);
      releasePage(pageNumber);
      canvases.delete(pageNumber);
    }
  }
  scheduleObservePages();
};

const updateVirtualWindowFromScroll = () => {
  const container = canvasContainer.value;
  if (!container || !totalPages.value) return;
  measureMountedPages();
  const center = container.scrollTop + container.clientHeight / 2;
  const viewportCenter =
    container.getBoundingClientRect().top + container.clientHeight / 2;
  const mountedRects = [...pageElements.entries()]
    .map(([page, element]) => ({ page, rect: element.getBoundingClientRect() }))
    .filter(
      ({ rect }) => Number.isFinite(rect.top) && Number.isFinite(rect.bottom),
    );
  const firstMountedTop = Math.min(...mountedRects.map(({ rect }) => rect.top));
  const lastMountedBottom = Math.max(
    ...mountedRects.map(({ rect }) => rect.bottom),
  );
  const mountedPage =
    mountedRects.length &&
    viewportCenter >= firstMountedTop &&
    viewportCenter <= lastMountedBottom
      ? mountedRects.reduce((closest, candidate) => {
          const closestDistance = Math.abs(
            (closest.rect.top + closest.rect.bottom) / 2 - viewportCenter,
          );
          const candidateDistance = Math.abs(
            (candidate.rect.top + candidate.rect.bottom) / 2 - viewportCenter,
          );
          return candidateDistance < closestDistance ? candidate : closest;
        }).page
      : undefined;
  const page =
    mountedPage ??
    Math.min(
      totalPages.value,
      Math.max(1, Math.floor(center / estimatedPageHeight.value) + 1),
    );
  updateVirtualWindow(page);
};

const isPageNearViewport = (page: number, margin: number) => {
  const root = canvasContainer.value;
  const element = pageElements.get(page);
  if (!root || !element) return page === currentPage.value;
  const rootRect = root.getBoundingClientRect();
  const pageRect = element.getBoundingClientRect();
  return (
    pageRect.bottom >= rootRect.top - margin &&
    pageRect.top <= rootRect.bottom + margin
  );
};

const pruneRenderedPages = () => {
  for (const page of [...renderedPages]) {
    if (page !== currentPage.value && !isPageNearViewport(page, RETAIN_MARGIN))
      releasePage(page);
  }
  for (let index = renderQueue.length - 1; index >= 0; index -= 1) {
    const request = renderQueue[index];
    if (
      request.page !== currentPage.value &&
      !isPageNearViewport(request.page, PRELOAD_MARGIN)
    ) {
      renderQueue.splice(index, 1);
      renderPromises.delete(request.page);
      request.resolve();
    }
  }
};

const updateCurrentPage = () => {
  const container = canvasContainer.value;
  if (!automaticPageDetectionEnabled || !container || !visiblePages.size)
    return;
  const containerRect = container.getBoundingClientRect();
  const center = containerRect.top + containerRect.height / 2;
  let closest = currentPage.value;
  let distance = Number.POSITIVE_INFINITY;
  for (const page of visiblePages) {
    const element = pageElements.get(page);
    if (!element) continue;
    const rect = element.getBoundingClientRect();
    const nextDistance = Math.abs((rect.top + rect.bottom) / 2 - center);
    if (nextDistance < distance) {
      distance = nextDistance;
      closest = page;
    }
  }
  if (closest !== currentPage.value) {
    cancelRenders();
    currentPage.value = closest;
    pageInput.value = String(closest);
    updateVirtualWindow(closest);
    emit("page-change", closest);
  }
};

const observePages = async () => {
  await nextTick();
  observer?.disconnect();
  visiblePages.clear();
  preloadPages.clear();
  const root = canvasContainer.value;
  if (!root) return;
  observer = new IntersectionObserver(
    (entries) => {
      for (const entry of entries) {
        const page = Number((entry.target as HTMLElement).dataset.page);
        if (entry.isIntersecting) {
          preloadPages.add(page);
        } else preloadPages.delete(page);
      }
      const rootRect = root.getBoundingClientRect();
      visiblePages.clear();
      for (const [page, element] of pageElements) {
        const rect = element.getBoundingClientRect();
        if (rect.bottom > rootRect.top && rect.top < rootRect.bottom)
          visiblePages.add(page);
      }
      updateCurrentPage();
      pruneRenderedPages();
      requestActivePages(virtualPages.value);
    },
    { root, rootMargin: `${PRELOAD_MARGIN}px 0px`, threshold: 0.01 },
  );
  for (const element of pageElements.values()) observer.observe(element);
  preloadPages.add(currentPage.value);
  requestActivePages(virtualPages.value);
};

const scheduleObservePages = () => {
  if (observerRefreshPending) return;
  observerRefreshPending = true;
  void nextTick().then(async () => {
    observerRefreshPending = false;
    await observePages();
  });
};

const performRender = async (pageNumber: number): Promise<void> => {
  const version = renderVersion;
  const loadedDocument = documentProxy.value;
  await nextTick();
  const canvas = canvases.get(pageNumber);
  if (!loadedDocument || !canvas) {
    renderPromises.delete(pageNumber);
    return;
  }
  const page: PDFPageProxy = await loadedDocument.getPage(pageNumber);
  if (version !== renderVersion || documentProxy.value !== loadedDocument)
    return;
  const viewport = page.getViewport({ scale: scale.value });
  const devicePixelRatio = window.devicePixelRatio || 1;
  canvas.width = Math.floor(viewport.width * devicePixelRatio);
  canvas.height = Math.floor(viewport.height * devicePixelRatio);
  canvas.style.width = `${viewport.width}px`;
  canvas.style.height = `${viewport.height}px`;
  const context = canvas.getContext("2d");
  if (!context) return;
  context.setTransform(devicePixelRatio, 0, 0, devicePixelRatio, 0, 0);
  const task = page.render({ canvas, canvasContext: context, viewport });
  renderTasks.set(pageNumber, task);
  try {
    await task.promise;
    if (version !== renderVersion) return;
    renderedPages.add(pageNumber);
    failedPages.delete(pageNumber);
  } catch (error) {
    if (isNormalCancellation(error)) return;
    failedPages.add(pageNumber);
    throw error;
  } finally {
    if (renderTasks.get(pageNumber) === task) renderTasks.delete(pageNumber);
  }
};

const pumpRenderQueue = () => {
  while (activeRenderCount < MAX_CONCURRENT_RENDERS && renderQueue.length) {
    renderQueue.sort((left, right) => {
      const priority = (page: number) =>
        page === currentPage.value ? 0 : visiblePages.has(page) ? 1 : 2;
      return priority(left.page) - priority(right.page);
    });
    const request = renderQueue.shift();
    if (!request) return;
    activeRenderCount += 1;
    void performRender(request.page)
      .then(request.resolve, request.reject)
      .finally(() => {
        activeRenderCount -= 1;
        if (renderPromises.get(request.page) === request.promise)
          renderPromises.delete(request.page);
        pumpRenderQueue();
      });
  }
};

const renderPage = (pageNumber: number): Promise<void> => {
  if (renderedPages.has(pageNumber)) return Promise.resolve();
  const existing = renderPromises.get(pageNumber);
  if (existing) return existing;
  let resolveRequest!: () => void;
  let rejectRequest!: (error: unknown) => void;
  const promise = new Promise<void>((resolve, reject) => {
    resolveRequest = resolve;
    rejectRequest = reject;
  });
  renderPromises.set(pageNumber, promise);
  renderQueue.push({
    page: pageNumber,
    resolve: resolveRequest,
    reject: rejectRequest,
    promise,
  });
  pumpRenderQueue();
  return promise;
};

const requestActivePages = (extraPages: readonly number[] = []) => {
  const pages = new Set([
    ...preloadPages,
    ...visiblePages,
    currentPage.value,
    ...extraPages,
  ]);
  const pending: Promise<void>[] = [];
  for (const page of pages) {
    if (!renderedPages.has(page) && !failedPages.has(page)) {
      pending.push(
        renderPage(page).catch((error) => {
          if (!isNormalCancellation(error))
            errorMessage.value = `No se pudo renderizar la página ${page}: ${getErrorMessage(error)}`;
        }),
      );
    }
  }
  return pending;
};

const renderActivePages = (extraPages: readonly number[] = []) => {
  for (const page of [...renderedPages]) releasePage(page);
  failedPages.clear();
  return requestActivePages(extraPages);
};

const handleWheel = (event: WheelEvent) => {
  if (!event.ctrlKey) return;
  event.preventDefault();
  changeZoom(event.deltaY < 0 ? 0.1 : -0.1);
};

const loadDocument = async () => {
  const version = ++loadVersion;
  automaticPageDetectionEnabled = false;
  cancelRenders();
  observer?.disconnect();
  observer = null;
  await loadingTask?.destroy();
  await documentProxy.value?.cleanup();
  loadingTask = null;
  documentProxy.value = null;
  totalPages.value = 0;
  virtualStart.value = 1;
  virtualEnd.value = 0;
  textCache.clear();
  searchOccurrences.value = [];
  activeOccurrenceIndex.value = -1;
  renderedPages.clear();
  failedPages.clear();
  visiblePages.clear();
  preloadPages.clear();
  pageElements.clear();
  pageHeightCache.clear();
  pageMetricsVersion.value += 1;
  canvases.clear();
  isInitialRenderDone.value = false;
  isLoading.value = true;
  errorMessage.value = "";
  try {
    const response = await fetch(new URL(props.src, window.location.href).href);
    if (!response.ok)
      throw new Error(`El recurso respondió con HTTP ${response.status}`);
    const data = new Uint8Array(await response.arrayBuffer());
    if (version !== loadVersion) return;
    const task = getDocument({ data });
    loadingTask = task;
    const loadedDocument = await task.promise;
    if (version !== loadVersion) return;
    documentProxy.value = loadedDocument;
    totalPages.value = loadedDocument.numPages;
    currentPage.value = 1;
    pageInput.value = "1";
    const firstPage = await loadedDocument.getPage(1);
    estimatedPageHeight.value =
      firstPage.getViewport({ scale: scale.value }).height + 16;
    updateVirtualWindow(1);
    await observePages();
  } catch (error) {
    if (version === loadVersion && !isNormalCancellation(error))
      errorMessage.value = `No se pudo cargar el PDF: ${getErrorMessage(error)}`;
  } finally {
    if (version === loadVersion) isLoading.value = false;
  }
};

const navigateToPage = async (
  page: number,
  behavior: ScrollBehavior = "smooth",
) => {
  if (!totalPages.value) return;
  const nextPage = Math.min(Math.max(page, 1), totalPages.value);
  automaticPageDetectionEnabled = true;
  if (nextPage !== currentPage.value) cancelRenders();
  currentPage.value = nextPage;
  pageInput.value = String(nextPage);
  emit("page-change", nextPage);
  updateVirtualWindow(nextPage);
  preloadPages.add(nextPage);
  await nextTick();
  requestActivePages();
  await renderPage(nextPage);
  await nextTick();
  const container = canvasContainer.value;
  const element = pageElements.get(nextPage);
  if (!container || !element) return;
  const containerRect = container.getBoundingClientRect();
  const pageRect = element.getBoundingClientRect();
  container.scrollTo({
    top: Math.max(0, container.scrollTop + pageRect.top - containerRect.top),
    behavior,
  });
};

const setPage = (page: number) => void navigateToPage(page);

const submitPage = () => {
  const requestedPage = Number.parseInt(pageInput.value, 10);
  if (Number.isFinite(requestedPage)) void setPage(requestedPage);
  else pageInput.value = String(currentPage.value);
};

type ZoomAnchor = { page: number; offset: number };

const captureZoomAnchor = (): ZoomAnchor | null => {
  const container = canvasContainer.value;
  const element = pageElements.get(currentPage.value);
  if (!container || !element) return null;
  return {
    page: currentPage.value,
    offset:
      element.getBoundingClientRect().top -
      container.getBoundingClientRect().top,
  };
};

const restoreZoomAnchor = (anchor: ZoomAnchor | null) => {
  if (!anchor) return;
  const container = canvasContainer.value;
  const element = pageElements.get(anchor.page);
  if (!container || !element) return;
  const containerRect = container.getBoundingClientRect();
  const pageRect = element.getBoundingClientRect();
  const nextScrollTop =
    container.scrollTop + (pageRect.top - containerRect.top) - anchor.offset;
  const maxScrollTop = Math.max(
    0,
    container.scrollHeight - container.clientHeight,
  );
  container.scrollTop = Math.min(maxScrollTop, Math.max(0, nextScrollTop));
};

const rescalePageHeightCache = (factor: number) => {
  for (const [page, height] of pageHeightCache)
    pageHeightCache.set(page, Math.max(16, height * factor));
  pageMetricsVersion.value += 1;
};

const applyZoom = async (nextScale: number) => {
  const previousScale = scale.value;
  const targetScale = Math.min(3, Math.max(0.5, nextScale));
  if (targetScale === previousScale) return;
  const version = ++zoomVersion;
  const anchor = captureZoomAnchor();
  measureMountedPages();
  const factor = targetScale / previousScale;
  scale.value = targetScale;
  rescalePageHeightCache(factor);
  estimatedPageHeight.value = Math.max(
    160,
    (estimatedPageHeight.value - 16) * factor + 16,
  );
  cancelRenders();
  const renders = renderActivePages(virtualPages.value);
  await nextTick();
  if (version !== zoomVersion) return;
  restoreZoomAnchor(anchor);
  await Promise.allSettled(renders);
  if (version !== zoomVersion) return;
  measureMountedPages();
  restoreZoomAnchor(anchor);
};

const changeZoom = (amount: number) => void applyZoom(scale.value + amount);

const resetZoom = () => applyZoom(1);

const findText = async () => {
  const loadedDocument = documentProxy.value;
  const query = searchQuery.value.trim().toLowerCase();
  if (!loadedDocument || !query) {
    searchStatus.value = "";
    searchOccurrences.value = [];
    activeOccurrenceIndex.value = -1;
    return;
  }
  searchStatus.value = "Buscando…";
  const occurrences: SearchOccurrence[] = [];
  for (
    let pageNumber = 1;
    pageNumber <= loadedDocument.numPages;
    pageNumber += 1
  ) {
    let pageText = textCache.get(pageNumber);
    if (!pageText) {
      const page = await loadedDocument.getPage(pageNumber);
      const content = await page.getTextContent();
      const items: CachedTextItem[] = [];
      for (const item of content.items) {
        const candidate = item as unknown as Record<string, unknown>;
        if (typeof candidate.str !== "string") continue;
        const transform = Array.isArray(candidate.transform)
          ? candidate.transform
          : [1, 0, 0, 1, 0, 0];
        items.push({
          str: candidate.str,
          width: typeof candidate.width === "number" ? candidate.width : 0,
          height: typeof candidate.height === "number" ? candidate.height : 0,
          transform: transform.slice(0, 6) as CachedTextItem["transform"],
        });
      }
      pageText = { items };
      textCache.set(pageNumber, pageText);
    }
    const page = await loadedDocument.getPage(pageNumber);
    const viewport = page.getViewport({ scale: 1 });
    for (const item of pageText.items) {
      const lowerText = item.str.toLowerCase();
      let start = lowerText.indexOf(query);
      while (start !== -1) {
        const transform = item.transform;
        const x = transform[4];
        const y = transform[5];
        const itemRect = viewport.convertToViewportRectangle([
          x,
          y,
          x + item.width,
          y + (item.height || Math.abs(transform[3]) || 1),
        ]);
        const left = Math.min(itemRect[0], itemRect[2]);
        const top = Math.min(itemRect[1], itemRect[3]);
        const itemWidth = Math.abs(itemRect[2] - itemRect[0]);
        const ratioStart = start / Math.max(item.str.length, 1);
        const ratioWidth = query.length / Math.max(item.str.length, 1);
        occurrences.push({
          page: pageNumber,
          rect: [
            left + itemWidth * ratioStart,
            top,
            Math.max(2, itemWidth * ratioWidth),
            Math.max(4, Math.abs(itemRect[3] - itemRect[1])),
          ],
        });
        start = lowerText.indexOf(query, start + query.length);
      }
    }
  }
  searchOccurrences.value = occurrences;
  activeOccurrenceIndex.value = occurrences.length ? 0 : -1;
  if (!occurrences.length) {
    searchStatus.value = "No encontrado";
    return;
  }
  await showActiveOccurrence();
};

const showActiveOccurrence = async () => {
  const occurrence = activeOccurrence.value;
  if (!occurrence) return;
  searchStatus.value = `${activeOccurrenceIndex.value + 1} de ${searchOccurrences.value.length} · Página ${occurrence.page}`;
  await navigateToPage(occurrence.page, "auto");
  await nextTick();
  const container = canvasContainer.value;
  const highlight = pageElements
    .get(occurrence.page)
    ?.querySelector<HTMLElement>(".pdf-reader__highlight");
  if (!container || !highlight) return;
  const containerRect = container.getBoundingClientRect();
  const highlightRect = highlight.getBoundingClientRect();
  container.scrollTo({
    top: Math.max(
      0,
      container.scrollTop +
        highlightRect.top -
        containerRect.top -
        (container.clientHeight - highlightRect.height) / 2,
    ),
    behavior: "smooth",
  });
};

const moveOccurrence = (direction: -1 | 1) => {
  if (!searchOccurrences.value.length) return;
  activeOccurrenceIndex.value =
    (activeOccurrenceIndex.value + direction + searchOccurrences.value.length) %
    searchOccurrences.value.length;
  void showActiveOccurrence();
};

const getHighlightStyle = (occurrence: SearchOccurrence | null) => {
  if (!occurrence) return {};
  const [left, top, width, height] = occurrence.rect;
  return {
    left: `${left * scale.value}px`,
    top: `${top * scale.value}px`,
    width: `${width * scale.value}px`,
    height: `${height * scale.value}px`,
  };
};

const retry = () => void loadDocument();
const handleScroll = () => {
  const root = canvasContainer.value;
  if (!root) return;
  automaticPageDetectionEnabled = true;
  updateVirtualWindowFromScroll();
  const rootRect = root.getBoundingClientRect();
  visiblePages.clear();
  for (const [page, element] of pageElements) {
    const rect = element.getBoundingClientRect();
    if (rect.bottom > rootRect.top && rect.top < rootRect.bottom)
      visiblePages.add(page);
  }
  updateCurrentPage();
  pruneRenderedPages();
  requestActivePages(virtualPages.value);
};

watch(
  () => props.src,
  () => void loadDocument(),
  { immediate: true },
);
watch(hasPendingPages, (pending) => {
  if (!pending && !isLoading.value) isInitialRenderDone.value = true;
});
onMounted(() => void observePages());
onBeforeUnmount(() => {
  ++loadVersion;
  observer?.disconnect();
  cancelRenders();
  void loadingTask?.destroy();
  void documentProxy.value?.cleanup();
});

defineExpose({ goToPage: (page: number) => void navigateToPage(page), setPage, currentPage });
</script>

<template>
  <div class="pdf-reader">
    <div class="pdf-reader__toolbar" aria-label="Controles del PDF">
      <button
        type="button"
        :disabled="currentPage <= 1"
        @click="void setPage(currentPage - 1)"
      >
        Anterior
      </button>
      <form class="page-control" @submit.prevent="submitPage">
        <label for="pdf-page">Página</label>
        <input
          id="pdf-page"
          v-model="pageInput"
          type="number"
          min="1"
          :max="totalPages"
          aria-label="Página actual"
        />
        <span>/ {{ totalPages || "—" }}</span>
      </form>
      <button
        type="button"
        :disabled="currentPage >= totalPages"
        @click="void setPage(currentPage + 1)"
      >
        Siguiente
      </button>
      <span class="toolbar-separator" aria-hidden="true"></span>
      <button type="button" aria-label="Reducir zoom" @click="changeZoom(-0.1)">
        −
      </button>
      <span>{{ Math.round(scale * 100) }}%</span>
      <button type="button" aria-label="Aumentar zoom" @click="changeZoom(0.1)">
        +
      </button>
      <button type="button" @click="void resetZoom()">Ajustar</button>
      <form class="search-control" @submit.prevent="void findText()">
        <input
          id="pdf-search"
          v-model="searchQuery"
          type="search"
          placeholder="Texto…"
          aria-label="Buscar texto"
        />
        <button type="submit">Buscar</button>
      </form>
      <button
        type="button"
        aria-label="Ocurrencia anterior"
        :disabled="!searchOccurrences.length"
        @click="moveOccurrence(-1)"
      >
        ←
      </button>
      <button
        type="button"
        aria-label="Ocurrencia siguiente"
        :disabled="!searchOccurrences.length"
        @click="moveOccurrence(1)"
      >
        →
      </button>
    </div>
    <p v-if="searchStatus" class="reader-status" role="status">
      {{ searchStatus }}
    </p>
    <p
      v-if="errorMessage"
      class="reader-status reader-status--error"
      role="alert"
    >
      {{ errorMessage }}
      <button type="button" @click="retry">Reintentar</button>
    </p>
    <div class="pdf-reader__viewport">
      <div
        ref="canvasContainer"
        class="pdf-reader__canvas-container"
        tabindex="0"
        @wheel="handleWheel"
        @scroll="handleScroll"
      >
        <div
          v-if="totalPages"
          class="pdf-reader__spacer"
          :style="{ height: `${getSpacerHeight(1, virtualStart - 1)}px` }"
          aria-hidden="true"
        ></div>
        <div
          v-for="page in virtualPages"
          :key="page"
          :ref="(element) => setPageElement(page, element)"
          :data-page="page"
          class="pdf-reader__page"
          :style="{
            minHeight: renderedPages.has(page)
              ? undefined
              : `${Math.max(0, estimatedPageHeight - 16)}px`,
          }"
        >
          <canvas
            :ref="(element) => setCanvas(page, element)"
            class="pdf-reader__canvas"
          ></canvas>
          <span
            v-if="activeOccurrence?.page === page"
            class="pdf-reader__highlight"
            :style="getHighlightStyle(activeOccurrence)"
            aria-hidden="true"
          ></span>
          <slot name="page-overlay" :page="page" :scale="scale" />
          <span v-if="!renderedPages.has(page)" class="pdf-reader__placeholder"
            >Página {{ page }}</span
          >
        </div>
        <div
          v-if="totalPages"
          class="pdf-reader__spacer"
          :style="{
            height: `${getSpacerHeight(virtualEnd + 1, totalPages)}px`,
          }"
          aria-hidden="true"
        ></div>
      </div>
      <div
        v-if="showLoadingOverlay"
        class="pdf-reader__loading pdf-reader__loading--blocking"
        role="status"
      >
        <span class="pdf-reader__loading-pill">
          <span class="pdf-reader__loading-spinner" aria-hidden="true"></span>
          Cargando documento…
        </span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.pdf-reader {
  width: 100%;
  height: 100%;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.pdf-reader__toolbar {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 0.5rem;
  padding: 0.75rem;
  border-bottom: 1px solid var(--reader-border, #dbe4ee);
  background: var(--reader-surface, #f8fafc);
}
button,
input {
  min-height: 2rem;
  border: 1px solid var(--reader-border, #cbd5e1);
  border-radius: 0.35rem;
  background: var(--reader-surface, #fff);
  color: var(--reader-heading, #172033);
}
button {
  padding: 0.25rem 0.6rem;
  cursor: pointer;
}
button:disabled {
  cursor: not-allowed;
  opacity: 0.5;
}
input {
  width: 4rem;
  padding: 0.25rem 0.4rem;
}
.page-control,
.search-control {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
}
.search-control input {
  width: 8rem;
}
.toolbar-separator {
  width: 1px;
  height: 1.5rem;
  margin: 0 0.25rem;
  background: var(--reader-border, #cbd5e1);
}
.reader-status {
  margin: 0;
  padding: 0.5rem 0.75rem;
  color: var(--reader-muted, #475569);
  font-size: 0.875rem;
}
.reader-status--error {
  color: var(--reader-muted, #b91c1c);
}
.pdf-reader__viewport {
  position: relative;
  flex: 1;
  min-height: 0;
  display: grid;
  grid-template-rows: minmax(0, 1fr);
}
.pdf-reader__canvas-container {
  flex: 1;
  height: 100%;
  min-height: 0;
  overflow: auto;
  box-sizing: border-box;
  padding: 1rem;
  background: var(--reader-canvas, #e2e8f0);
}
.pdf-reader__loading {
  position: absolute;
  inset: 0;
  display: grid;
  place-items: center;
  pointer-events: none;
  z-index: 1;
}
.pdf-reader__loading--blocking {
  background: color-mix(
    in srgb,
    var(--reader-canvas, #e2e8f0) 40%,
    transparent
  );
  backdrop-filter: blur(1px);
  pointer-events: auto;
  z-index: 2;
}
.pdf-reader__loading-pill {
  display: inline-flex;
  align-items: center;
  gap: 0.45rem;
  padding: 0.4rem 0.85rem;
  border: 1px solid var(--reader-border, #dbe4ee);
  border-radius: 999px;
  background: var(--reader-surface, #fff);
  color: var(--reader-muted, #64748b);
  font-size: 0.875rem;
  white-space: nowrap;
  box-shadow: 0 2px 8px
    color-mix(in srgb, var(--reader-heading, #0f172a) 15%, transparent);
}
.pdf-reader__loading-spinner {
  width: 0.75rem;
  height: 0.75rem;
  border: 2px solid var(--reader-border, #cbd5e1);
  border-top-color: var(--reader-muted, #64748b);
  border-radius: 50%;
  animation: pdf-reader-spin 0.8s linear infinite;
}
@keyframes pdf-reader-spin {
  to {
    transform: rotate(360deg);
  }
}
.pdf-reader__page {
  position: relative;
  display: flex;
  justify-content: center;
  align-items: center;
  width: fit-content;
  min-height: 160px;
  margin: 0 auto 1rem;
  overflow: visible;
}
.pdf-reader__spacer {
  width: 100%;
  pointer-events: none;
}
.pdf-reader__canvas {
  display: block;
  max-width: none;
  background: #fff;
  box-shadow: 0 2px 8px
    color-mix(in srgb, var(--reader-heading, #0f172a) 20%, transparent);
}
.pdf-reader__highlight {
  position: absolute;
  z-index: 3;
  pointer-events: none;
  background: rgb(250 204 21 / 55%);
  outline: 2px solid rgb(202 138 4 / 75%);
}
.pdf-reader__placeholder {
  position: absolute;
  color: var(--reader-muted, #64748b);
  font-size: 0.875rem;
}
@media (max-width: 600px) {
  .pdf-reader {
    height: 100%;
    min-height: 400px;
  }
  .pdf-reader__toolbar {
    gap: 0.35rem;
  }
  .search-control {
    width: 100%;
  }
  .search-control input {
    flex: 1;
  }
}
</style>
