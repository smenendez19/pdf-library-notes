# PDF Notes prototype

## Descripcion

Prototipo de App Web de un catalogo de libros en formato PDF que te permite leerlos, realizar notas y guardar progresos, clasificar libros y cargar nuevos libros utilizando un sistema de archivos.

## Link web

Lector PDF: <http://localhost:5173/reader>

## Caracteristicas

- Catalogo de libros por portada con filtros por etiquetas, autores, editoriales.
- Lector de PDF con PDF.js directo.
- Notas tipo postit por página, mantenidas solo en memoria.
- Controles de página, zoom, búsqueda e impresión, además de modo oscuro.
- Sincronizacion por perfil.
- Sistema de agregado de nuevos libros PDF.

## Tecnologias

Frontend:

- Vue 3
- Vite
- TypeScript

## Arquitectura

## Ejecucion manual

1. Instalar pnpm (Alternativa mas eficiente que npm ya que utiliza symlinks para las dependencias y es mas seguro)
    npx get-pnpm
2. Si estas en WSL (Windows 11) configurar el store-dir:
    pnpm config set store-dir /mnt/e/.pnpm-store --global
3. Ejecutar la instalacion
    pnpm install
