import { ref } from "vue";
import { getDocument, GlobalWorkerOptions } from "pdfjs-dist";
import workerSrc from "pdfjs-dist/build/pdf.worker.mjs?url";

if (!GlobalWorkerOptions.workerSrc) {
  GlobalWorkerOptions.workerSrc = workerSrc;
}

export interface PdfUploadMetadata {
  title?: string;
  author?: string;
  editorial?: string;
}

export interface PreparedPdfFile extends PdfUploadMetadata {
  suggestedTitle: string;
}

export function parseTitleFromFilename(name: string): string {
  const withoutExtension = name.replace(/\.pdf$/i, "");
  return withoutExtension
    .replace(/[_-]+/g, " ")
    .replace(/\s+/g, " ")
    .trim();
}

function toCleanString(value: unknown): string | undefined {
  if (typeof value !== "string") return undefined;
  const trimmed = value.trim();
  return trimmed ? trimmed : undefined;
}

async function toFileData(file: File): Promise<Uint8Array> {
  return new Uint8Array(await file.arrayBuffer());
}

export async function extractPdfMetadata(
  file: File,
): Promise<PdfUploadMetadata> {
  try {
    const data = await toFileData(file);
    const doc = await getDocument({ data }).promise;
    try {
      const { info, metadata } = await doc.getMetadata();
      const infoRecord = (info ?? {}) as Record<string, unknown>;
      const dc = (key: string): string | undefined => {
        try {
          const holder = metadata as unknown as {
            get?: (name: string) => unknown;
          } | null;
          return toCleanString(holder?.get?.(key));
        } catch {
          return undefined;
        }
      };
      const title =
        toCleanString(infoRecord.Title) ?? dc("dc:title");
      const author =
        toCleanString(infoRecord.Author) ?? dc("dc:creator");
      const editorial =
        toCleanString(infoRecord.Producer) ??
        toCleanString(infoRecord.Creator) ??
        dc("dc:publisher");
      const result: PdfUploadMetadata = {};
      if (title) result.title = title;
      if (author) result.author = author;
      if (editorial) result.editorial = editorial;
      return result;
    } finally {
      await doc.destroy().catch(() => {});
    }
  } catch {
    return {};
  }
}

export async function renderFirstPageCover(
  file: File,
  scale = 0.6,
): Promise<string> {
  let doc: Awaited<ReturnType<typeof getDocument>["promise"]> | null = null;
  try {
    const data = await toFileData(file);
    doc = await getDocument({ data }).promise;
    const page = await doc.getPage(1);
    try {
      const viewport = page.getViewport({ scale });
      const canvas = document.createElement("canvas");
      canvas.width = Math.floor(viewport.width);
      canvas.height = Math.floor(viewport.height);
      const context = canvas.getContext("2d");
      if (!context) {
        throw new Error("El navegador no pudo crear el contexto 2D.");
      }
      await page.render({ canvas, canvasContext: context, viewport }).promise;
      return canvas.toDataURL("image/png");
    } finally {
      page.cleanup();
    }
  } catch (error) {
    if (error instanceof Error && error.message.startsWith("No se pudo")) {
      throw error;
    }
    const detail = error instanceof Error ? `: ${error.message}` : "";
    throw new Error(`No se pudo generar la portada del PDF${detail}`);
  } finally {
    await doc?.destroy().catch(() => {});
  }
}

function assertIsPdf(file: File): void {
  const isPdfMime = file.type === "application/pdf";
  const isPdfExtension = file.name.toLowerCase().endsWith(".pdf");
  if (!isPdfMime && !isPdfExtension) {
    throw new Error("El archivo debe ser un PDF válido.");
  }
  if (file.size <= 0) {
    throw new Error("El archivo está vacío.");
  }
}

export function usePdfUpload() {
  const selectedFile = ref<File | null>(null);
  const coverDataUrl = ref<string>("");
  const isParsing = ref(false);
  const parseError = ref<string>("");

  async function prepareFile(file: File): Promise<PreparedPdfFile> {
    assertIsPdf(file);
    selectedFile.value = file;
    coverDataUrl.value = "";
    parseError.value = "";
    isParsing.value = true;
    const filenameTitle = parseTitleFromFilename(file.name);
    try {
      const [metadata, cover] = await Promise.all([
        extractPdfMetadata(file).catch(() => ({}) as PdfUploadMetadata),
        renderFirstPageCover(file),
      ]);
      coverDataUrl.value = cover;
      return {
        suggestedTitle: metadata.title ?? filenameTitle,
        ...(metadata.author ? { author: metadata.author } : {}),
        ...(metadata.editorial ? { editorial: metadata.editorial } : {}),
      };
    } catch (error) {
      parseError.value =
        error instanceof Error
          ? error.message
          : "No se pudo generar la portada del PDF.";
      const metadata = await extractPdfMetadata(file).catch(
        () => ({}) as PdfUploadMetadata,
      );
      return {
        suggestedTitle: metadata.title ?? filenameTitle,
        ...(metadata.author ? { author: metadata.author } : {}),
        ...(metadata.editorial ? { editorial: metadata.editorial } : {}),
      };
    } finally {
      isParsing.value = false;
    }
  }

  return {
    selectedFile,
    coverDataUrl,
    isParsing,
    parseError,
    prepareFile,
  };
}
