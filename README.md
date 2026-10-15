# PDF Library Notes

App web de catálogo de libros en PDF: explora libros por portada, léelos en el navegador, toma notas tipo post-it por página y guarda tu progreso de lectura. Permite clasificar por etiquetas/autores/editoriales y cargar nuevos PDFs.

## Capturas

> Placeholders: añade las imágenes reales en `docs/screenshots/`.

![Catálogo de libros](docs/screenshots/catalog.png)
![Lector PDF con notas](docs/screenshots/reader.png)
![Agregar nuevo PDF](docs/screenshots/upload.png)

## Tecnologías

- **Frontend:** Vue 3, Vite, TypeScript, `pdfjs-dist`, `vue-router`, `marked`, `dompurify`.
- **Backend:** Go 1.26 (solo stdlib + `modernc.org/sqlite`, SQLite puro Go sin cgo).
- **Tooling:** pnpm, Docker Compose.

## Estructura del repo

```text
backend/    API en Go + SQLite (catálogo, PDFs, notas y progreso)
frontend/   Cliente Vue 3 + Vite (catálogo, lector PDF.js, notas)
```

## Documentación

- Backend (endpoints y configuración): [backend/README.md](backend/README.md)
- Frontend (instalación y uso): [frontend/README.md](frontend/README.md)

## Arranque rápido

### Manual

```bash
# Backend (http://localhost:8080)
cd backend
go run ./cmd/server

# Frontend (http://localhost:5173)
cd frontend
pnpm install
VITE_API_URL=http://localhost:8080/api/v1 pnpm dev
```

### Docker Compose

```bash
docker compose up --build
```

- Backend: <http://localhost:8080>
- Frontend: <http://localhost:8081>
