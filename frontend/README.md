# PDF Library Notes — Frontend

Cliente Vue 3 del catálogo de libros PDF: explora el catálogo por portada, lee PDFs en el navegador y toma notas tipo post-it por página.

## Características principales

- Catálogo por portada con filtros por etiquetas, autores y editoriales.
- Lector PDF con PDF.js (`pdfjs-dist` 5.4.530) directo en el navegador.
- Notas tipo post-it por página (arrastrables, de colores).
- Controles de página, zoom, búsqueda en el documento e impresión.
- Modo oscuro.
- Agregar nuevos PDFs al catálogo desde la interfaz.
- `VITE_API_URL` con fallback a datos mock locales si el backend no está disponible.

## Instalación (pnpm)

Requiere Node 20+ y pnpm 9+.

```bash
cd frontend
pnpm install
```

> Nota WSL: en Windows con WSL conviene mover el store fuera del disco de Windows para evitar lentitud:
>
> ```bash
> pnpm config set store-dir /mnt/e/.pnpm-store --global
> ```

## Uso

```bash
pnpm dev      # desarrollo (Vite, http://localhost:5173)
pnpm build    # compila a dist/
pnpm preview  # sirve el build local
```

Variable de entorno:

| Variable       | Descripción                                              |
| -------------- | -------------------------------------------------------- |
| `VITE_API_URL` | URL base del backend (p. ej. `http://localhost:8080/api/v1`). Si no se define, el catálogo usa datos mock locales. |

Ejemplo con backend local:

```bash
VITE_API_URL=http://localhost:8080/api/v1 pnpm dev
```

## Scripts

| Script         | Comando                  | Descripción              |
| -------------- | ------------------------ | ------------------------ |
| `pnpm dev`     | `vite`                   | Servidor de desarrollo   |
| `pnpm build`   | `vue-tsc -b && vite build` | Chequeo de tipos + build |
| `pnpm preview` | `vite preview`           | Previsualiza `dist/`     |

El contrato de endpoints vive en [../backend/README.md](../backend/README.md).
