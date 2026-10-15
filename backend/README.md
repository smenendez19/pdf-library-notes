# pdfnotes-backend

Backend de catálogo PDF + notas adhesivas sincronizadas. Go 1.26, solo stdlib + `modernc.org/sqlite` (SQLite puro Go, sin cgo).

## Requisitos

- Go 1.26 (`export GOROOT=/usr/lib/go-1.26 PATH=/usr/lib/go-1.26/bin:$PATH` si no es el Go por defecto).
- Funciona en Raspberry Pi (binario estático, sin cgo ni dependencias del sistema).

## Configuración (env)

| Variable           | Default           | Descripción                              |
|--------------------|-------------------|------------------------------------------|
| `PORT`             | `8080`            | Puerto HTTP                              |
| `STORAGE_ROOT`     | `./data/files`    | Directorio de PDFs (dedup por SHA-256)   |
| `DATABASE_PATH`    | `./data/catalog.db` | Ruta del SQLite                        |
| `ALLOWED_ORIGINS`  | `*`               | CORS, lista separada por comas           |
| `MAX_UPLOAD_BYTES` | `104857600`       | Límite de upload en bytes (100 MB)       |

Ver `backend/.env.example` (mismos defaults).

## Cómo correr

```bash
export GOROOT=/usr/lib/go-1.26 PATH=/usr/lib/go-1.26/bin:$PATH
cd backend
go vet ./... && go build ./...
PORT=8080 STORAGE_ROOT=./data/files DATABASE_PATH=./data/catalog.db go run ./cmd/server
```

## Contrato de endpoints

Todos los errores son JSON: `{"error":"mensaje"}`.

### `GET /api/v1/health`

```bash
curl localhost:8080/api/v1/health
# {"status":"ok","time":"...","version":"v0.1.0"}
```

### `GET /api/v1/documents?page=1&items=10&search=&tags=a,b&author=&editorial=&date_from=2026-01-01&date_to=2026-12-31`

Lista paginada (`items` máx 50). Respuesta: `{data:[{id,title,tags,upload_date,author,editorial,description,fileUrl}],page,items,max_pages,max_items}`.

### `GET /api/v1/documents/{id}`

Detalle + `size_bytes`, `mime_type`. 404 si no existe.

### `GET /api/v1/documents/{id}/content`

PDF crudo con soporte `Range` (vía `http.ServeContent`). Nunca gzipped.

### `POST /api/v1/documents/upload` (multipart)

```bash
curl -F "file=@doc.pdf;type=application/pdf" -F "title=Mi doc" -F "tags=a,b" localhost:8080/api/v1/documents/upload
# 201 {"id":"...","title":"...","tags":[],"upload_date":"2026-10-14",...,"fileUrl":"/api/v1/documents/<id>/content","size_bytes":123,"sha256":"..."}
```

Campos: `file` (requerido, `.pdf` válido), `title` (default: nombre del archivo), `author`, `editorial`, `description`, `upload_date` (`YYYY-MM-DD`, default hoy), `tags` (CSV o array JSON). Dedup por SHA-256: si el contenido ya existe devuelve la fila existente (201). Acepta `Content-Encoding: gzip` en el request. 413 si supera 100 MB.

### `POST /api/v1/notes/save` (JSON)

Guardado delta: requiere `Content-Type: application/json`.

```bash
curl -X POST localhost:8080/api/v1/notes/save -H 'Content-Type: application/json' -d '{
  "document_id": "<id>",
  "current_page": 3,
  "upsert": [{"id": 0, "page": 3, "content": "hola", "color": "#fff9db",
              "x": 12, "y": 10, "width": 200, "height": 150, "visible": true}],
  "delete_ids": []
}'
# {"document_id":"<id>","current_page":3,"notes_version":1,
#  "saved":[{"client_id":0,"id":1}],"deleted":[]}
```

Reglas: el documento debe existir (404 si no); `id<=0` = nueva (el servidor asigna `AUTOINCREMENT` y lo devuelve en `saved`); upsert con `id>0` actualiza si pertenece al documento, se crea con ese id si no existe, 400 si pertenece a otro documento; `page>=1` (si no, 400); `content` se trunca a 1000; `color` debe ser `#` + 6 hex (si no, default `#fff9db`); `x/y` clamp a `[-100,200]`; `width` clamp a `[140,480]` (default 200), `height` clamp a `[100,480]` (default 150); máx 500 items en `upsert` y 500 en `delete_ids` (400 si excede). Todo (upserts + deletes + `current_page` + `notes_version+1`) en una transacción.

### `GET /api/v1/documents/{id}/notes?since_version=N`

```bash
curl "localhost:8080/api/v1/documents/<id>/notes?since_version=0"
# {"document_id":"<id>","notes_version":1,"current_page":3,
#  "notes":[{"id":1,"page":3,"content":"hola",...}],"deleted_tombstones":[]}
```

Sin `since_version` devuelve todo; si `since_version >= notes_version` devuelve `notes:[]` (polling barato). `notes` ordenadas por id. 404 si el doc no existe.

### `GET /api/v1/documents/{id}/state`

```bash
curl localhost:8080/api/v1/documents/<id>/state
# {"document_id":"<id>","current_page":3,"notes_version":1,"updated_at":"..."}
```

## Estrategia gzip

- Respuestas JSON: se comprimen con gzip solo si el cliente envía `Accept-Encoding: gzip` (middleware `gzipJSON`).
- `/content` se sirve crudo (nunca gzipped) con `Range` para el visor PDF.
- El upload acepta requests con `Content-Encoding: gzip`.

## Límites

- Upload: 100 MB (`MAX_UPLOAD_BYTES`).
- `upsert` / `delete_ids`: 500 items cada uno por request.
- Notas: `id<=0` = nueva; `content` máx 1000 caracteres; `page>=1`.

## Nota de portabilidad

`backend/` es movible: módulo propio (`pdfnotes-backend`), única dependencia externa `modernc.org/sqlite` (puro Go, sin cgo). Basta copiar la carpeta y ajustar las 5 env vars.
