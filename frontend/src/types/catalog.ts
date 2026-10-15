export interface CatalogDocument {
  id: string;
  title: string;
  tags: string[];
  image?: string;
  coverUrl?: string | null;
  upload_date: string;
  author: string;
  editorial: string;
  description?: string;
  fileUrl: string;
}

export interface UploadDocumentPayload {
  file: File;
  title: string;
  author?: string;
  editorial?: string;
  description?: string;
  tags: string[];
  coverDataUrl?: string;
}

export interface UploadDocumentResponse {
  id: string;
  title: string;
  tags: string[];
  upload_date: string;
  author: string;
  editorial: string;
  description?: string;
  fileUrl: string;
  size_bytes?: number;
  sha256?: string;
  coverUrl?: string | null;
}
export interface CatalogQueryParams {
  page?: number;
  items?: number;
  search?: string;
}

export interface PaginatedCatalogResponse {
  data: CatalogDocument[];
  page: number;
  items: number;
  max_pages: number;
  max_items: number;
}
