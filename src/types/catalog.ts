export interface CatalogDocument {
  id: string;
  title: string;
  tags: string[];
  image?: string;
  upload_date: string;
  author: string;
  editorial: string;
  description?: string;
  fileUrl: string;
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
