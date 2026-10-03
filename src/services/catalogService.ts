import type { CatalogDocument, CatalogQueryParams, PaginatedCatalogResponse } from '../types/catalog';

export const API_BASE_URL = import.meta.env.VITE_API_URL as string | undefined;

const REAL_FILE_URL = '/automatetheboringstuffwithpython.pdf';

const SYNTHETIC_TITLES: Array<{
  title: string;
  author: string;
  editorial: string;
  tags: string[];
}> = [
  { title: 'Python Crash Course', author: 'Eric Matthes', editorial: 'No Starch Press', tags: ['python', 'programación'] },
  { title: 'Fluent Python', author: 'Luciano Ramalho', editorial: "O'Reilly Media", tags: ['python', 'avanzado'] },
  { title: 'Eloquent JavaScript', author: 'Marijn Haverbeke', editorial: 'No Starch Press', tags: ['javascript', 'programación'] },
  { title: 'You Don’t Know JS: Scope & Closures', author: 'Kyle Simpson', editorial: "O'Reilly Media", tags: ['javascript', 'avanzado'] },
  { title: 'Clean Code', author: 'Robert C. Martin', editorial: 'Prentice Hall', tags: ['calidad', 'programación'] },
  { title: 'The Pragmatic Programmer', author: 'David Thomas', editorial: 'Addison-Wesley', tags: ['carrera', 'programación'] },
  { title: 'Design Patterns', author: 'Erich Gamma', editorial: 'Addison-Wesley', tags: ['patrones', 'arquitectura'] },
  { title: 'Refactoring', author: 'Martin Fowler', editorial: 'Addison-Wesley', tags: ['calidad', 'refactorización'] },
  { title: 'Introduction to Algorithms', author: 'Thomas H. Cormen', editorial: 'MIT Press', tags: ['algoritmos', 'teoría'] },
  { title: 'Grokking Algorithms', author: 'Aditya Bhargava', editorial: 'Manning', tags: ['algoritmos', 'ilustrado'] },
  { title: 'Learning Vue 3', author: 'Marina Mosti', editorial: 'Packt', tags: ['vue', 'frontend'] },
  { title: 'The Vue.js Handbook', author: 'Flavio Copes', editorial: 'Independiente', tags: ['vue', 'frontend'] },
  { title: 'CSS in Depth', author: 'Keith J. Grant', editorial: 'Manning', tags: ['css', 'frontend'] },
  { title: 'SQL Antipatterns', author: 'Bill Karwin', editorial: 'Pragmatic Bookshelf', tags: ['sql', 'bases de datos'] },
  { title: 'Designing Data-Intensive Applications', author: 'Martin Kleppmann', editorial: "O'Reilly Media", tags: ['datos', 'arquitectura'] },
  { title: 'Fundamentals of Data Engineering', author: 'Joe Reis', editorial: "O'Reilly Media", tags: ['datos', 'ingeniería'] },
  { title: 'Hands-On Machine Learning', author: 'Aurélien Géron', editorial: "O'Reilly Media", tags: ['machine learning', 'python'] },
  { title: 'Deep Learning with Python', author: 'François Chollet', editorial: 'Manning', tags: ['deep learning', 'python'] },
  { title: 'The Linux Command Line', author: 'William Shotts', editorial: 'No Starch Press', tags: ['linux', 'terminal'] },
  { title: 'Git Pro', author: 'Scott Chacon', editorial: 'Apress', tags: ['git', 'herramientas'] },
  { title: 'Docker Deep Dive', author: 'Nigel Poulton', editorial: 'Independiente', tags: ['docker', 'devops'] },
  { title: 'Site Reliability Engineering', author: 'Betsy Beyer', editorial: "O'Reilly Media", tags: ['devops', 'sre'] },
  { title: 'Accessibility for Everyone', author: 'Laura Kalbag', editorial: 'A Book Apart', tags: ['accesibilidad', 'frontend'] },
];

function toUploadDate(index: number): string {
  const month = String((index % 12) + 1).padStart(2, '0');
  const day = String((index % 28) + 1).padStart(2, '0');
  return `2023-${month}-${day}`;
}

const MOCK_CATALOG: CatalogDocument[] = [
  {
    id: 'automatetheboringstuff',
    title: 'Automate the Boring Stuff with Python',
    tags: ['python', 'automatización', 'programación'],
    upload_date: '2019-11-12',
    author: 'Al Sweigart',
    editorial: 'No Starch Press',
    description: 'Guía práctica de programación en Python para automatizar tareas cotidianas.',
    fileUrl: REAL_FILE_URL,
  },
  ...SYNTHETIC_TITLES.map((entry, index) => ({
    id: `doc-${index + 2}`,
    title: entry.title,
    tags: entry.tags,
    upload_date: toUploadDate(index),
    author: entry.author,
    editorial: entry.editorial,
    description: `Documento de ejemplo: ${entry.title}.`,
    fileUrl: REAL_FILE_URL,
  })),
];

function normalizePage(page: number | undefined): number {
  if (typeof page !== 'number' || !Number.isFinite(page)) return 1;
  return Math.max(1, Math.floor(page));
}

function normalizeItems(items: number | undefined): number {
  if (typeof items !== 'number' || !Number.isFinite(items)) return 12;
  return Math.min(50, Math.max(1, Math.floor(items)));
}

function matchesSearch(document: CatalogDocument, search: string): boolean {
  const query = search.trim().toLowerCase();
  if (!query) return true;
  const haystack = [document.title, document.author, document.editorial, ...document.tags]
    .join(' ')
    .toLowerCase();
  return haystack.includes(query);
}

function paginate(documents: CatalogDocument[], page: number, items: number): PaginatedCatalogResponse {
  const max_items = documents.length;
  const max_pages = Math.max(1, Math.ceil(max_items / items));
  const safePage = Math.min(page, max_pages);
  const data = documents.slice((safePage - 1) * items, safePage * items);
  return { data, page: safePage, items, max_pages, max_items };
}

function delay(ms: number): Promise<void> {
  return new Promise((resolve) => {
    setTimeout(resolve, ms);
  });
}

export async function fetchCatalog(params: CatalogQueryParams = {}): Promise<PaginatedCatalogResponse> {
  const page = normalizePage(params.page);
  const items = normalizeItems(params.items);
  const search = params.search ?? '';

  // TODO: cuando el backend esté disponible, usar esta rama como fuente principal
  // y mantener el mock solo como fallback.
  if (API_BASE_URL) {
    try {
      const query = new URLSearchParams({
        page: String(page),
        items: String(items),
        ...(search ? { search } : {}),
      });
      const response = await fetch(`${API_BASE_URL}/catalog?${query.toString()}`);
      if (response.ok) {
        return (await response.json()) as PaginatedCatalogResponse;
      }
    } catch {
      // Sin red o backend no disponible: continuar con el mock local.
    }
  }

  const filtered = MOCK_CATALOG.filter((document) => matchesSearch(document, search));
  await delay(250);
  return paginate(filtered, page, items);
}
