import type { CatalogDocument, CatalogQueryParams, PaginatedCatalogResponse, UploadDocumentPayload, UploadDocumentResponse } from '../types/catalog';

export const API_BASE_URL = import.meta.env.VITE_API_URL as string | undefined;

export function resolveDocumentUrl(fileUrl: string): string {
  if (fileUrl.startsWith('http://') || fileUrl.startsWith('https://')) {
    return fileUrl;
  }
  if (fileUrl.startsWith('/') && API_BASE_URL) {
    try {
      return new URL(fileUrl, API_BASE_URL).href;
    } catch {
      return fileUrl;
    }
  }
  return fileUrl;
}

function normalizePage(page: number | undefined): number {
  if (typeof page !== 'number' || !Number.isFinite(page)) return 1;
  return Math.max(1, Math.floor(page));
}

function normalizeItems(items: number | undefined): number {
  if (typeof items !== 'number' || !Number.isFinite(items)) return 12;
  return Math.min(50, Math.max(1, Math.floor(items)));
}

export async function fetchCatalog(params: CatalogQueryParams = {}): Promise<PaginatedCatalogResponse> {
  const page = normalizePage(params.page);
  const items = normalizeItems(params.items);
  const search = params.search ?? '';

  if (!API_BASE_URL) {
    return { data: [], page, items, max_pages: 1, max_items: 0 };
  }

  const query = new URLSearchParams({
    page: String(page),
    items: String(items),
    ...(search ? { search } : {}),
  });
  const response = await fetch(`${API_BASE_URL}/documents?${query.toString()}`);
  if (!response.ok) {
    let message = 'No se pudo cargar el catálogo.';
    try {
      const data = (await response.json()) as { error?: string };
      if (data?.error) message = data.error;
    } catch {
      // Mantener mensaje genérico si el cuerpo no es JSON.
    }
    throw new Error(message);
  }
  const paginated = (await response.json()) as PaginatedCatalogResponse;
  return {
    ...paginated,
    data: paginated.data.map((doc) => ({
      ...doc,
      image: doc.coverUrl ? resolveDocumentUrl(doc.coverUrl) : doc.image,
    })),
  };
}

export async function uploadDocument(payload: UploadDocumentPayload): Promise<CatalogDocument> {
  const isPdfMime = payload.file.type === 'application/pdf';
  const isPdfExtension = payload.file.name.toLowerCase().endsWith('.pdf');
  if (!isPdfMime && !isPdfExtension) {
    throw new Error('El archivo debe ser un PDF válido.');
  }
  if (payload.file.size <= 0) {
    throw new Error('El archivo está vacío.');
  }
  if (!API_BASE_URL) {
    throw new Error('Backend no disponible');
  }

  const formData = new FormData();
  formData.append('file', payload.file);
  formData.append('title', payload.title);
  if (payload.author !== undefined) formData.append('author', payload.author);
  if (payload.editorial !== undefined) formData.append('editorial', payload.editorial);
  if (payload.description !== undefined) formData.append('description', payload.description);
  formData.append('tags', payload.tags.join(','));
  if (payload.coverDataUrl) {
    formData.append('cover', payload.coverDataUrl);
  }

  const response = await fetch(`${API_BASE_URL}/documents/upload`, {
    method: 'POST',
    body: formData,
  });
  if (!response.ok) {
    let message = 'Error al subir el documento.';
    try {
      const data = (await response.json()) as { error?: string };
      if (data?.error) message = data.error;
    } catch {
      // Mantener mensaje genérico si el cuerpo no es JSON.
    }
    throw new Error(message);
  }

  const uploaded = (await response.json()) as UploadDocumentResponse;
  return {
    id: uploaded.id,
    title: uploaded.title,
    tags: uploaded.tags,
    upload_date: uploaded.upload_date,
    author: uploaded.author,
    editorial: uploaded.editorial,
    description: uploaded.description,
    fileUrl: uploaded.fileUrl,
    coverUrl: uploaded.coverUrl ?? null,
    image: uploaded.coverUrl ? resolveDocumentUrl(uploaded.coverUrl) : undefined,
  };
}

export async function deleteDocument(id: string): Promise<void> {
  if (!API_BASE_URL) {
    throw new Error('Backend no disponible');
  }
  const response = await fetch(`${API_BASE_URL}/documents/${encodeURIComponent(id)}`, {
    method: 'DELETE',
  });
  if (response.ok) {
    return;
  }
  if (response.status === 404) {
    throw new Error('El documento no existe o ya fue eliminado.');
  }
  let message = 'Error al eliminar el documento.';
  try {
    const data = (await response.json()) as { error?: string };
    if (data?.error) message = data.error;
  } catch {
    // Mantener mensaje genérico si el cuerpo no es JSON.
  }
  throw new Error(message);
}
