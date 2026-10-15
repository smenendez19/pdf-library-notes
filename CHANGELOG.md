# Changelog

## [0.0.1] - 2026-10-15

Versión inicial del prototipo PDF Library Notes.

### Añadido
- Catálogo de PDFs por portada con filtros por etiquetas, autores, editoriales y búsqueda.
- Lector PDF en navegador (PDF.js) con paginación, zoom, búsqueda e impresión.
- Notas tipo post-it por página (arrastrables, de colores) con sincronización y guardado de progreso.
- Subida de PDFs con deduplicación por SHA-256.
- API Go + SQLite (documentos, notas, estado, health) y despliegue con Docker Compose.
- Modo oscuro y fallback a datos mock si el backend no está disponible.
