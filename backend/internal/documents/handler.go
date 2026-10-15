package documents

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"pdfnotes-backend/internal/storage"
	"pdfnotes-backend/internal/store"
)

// Handler serves the document catalog endpoints.
type Handler struct {
	svc         *Service
	storageRoot string
}

// NewHandler returns a Handler backed by s serving files from storageRoot.
func NewHandler(s *store.Store, storageRoot string) *Handler {
	svc := NewService(s)
	svc.storageRoot = storageRoot
	return &Handler{svc: svc, storageRoot: storageRoot}
}

// Register mounts the document routes on mux.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/documents", h.list)
	mux.HandleFunc("GET /api/v1/documents/{id}", h.get)
	mux.HandleFunc("GET /api/v1/documents/{id}/content", h.content)
	mux.HandleFunc("GET /api/v1/documents/{id}/cover", h.cover)
	mux.HandleFunc("DELETE /api/v1/documents/{id}", h.remove)
}

type listItem struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Tags        []string `json:"tags"`
	UploadDate  string   `json:"upload_date"`
	Author      string   `json:"author"`
	Editorial   string   `json:"editorial"`
	Description string   `json:"description"`
	FileURL     string   `json:"fileUrl"`
	CoverURL    *string  `json:"coverUrl"`
}

type detailItem struct {
	listItem
	SizeBytes int64  `json:"size_bytes"`
	MimeType  string `json:"mime_type"`
}

type listResponse struct {
	Data     []listItem `json:"data"`
	Page     int        `json:"page"`
	Items    int        `json:"items"`
	MaxPages int        `json:"max_pages"`
	MaxItems int        `json:"max_items"`
}

func toListItem(d Doc) listItem {
	return listItem{
		ID:          d.ID,
		Title:       d.Title,
		Tags:        d.Tags,
		UploadDate:  d.UploadDate,
		Author:      d.Author,
		Editorial:   d.Editorial,
		Description: d.Description,
		FileURL:     "/api/v1/documents/" + d.ID + "/content",
		CoverURL:    coverURLFor(d),
	}
}

// coverURLFor returns the cover endpoint URL, or nil when the document
// has no cover so it serializes as JSON null.
func coverURLFor(d Doc) *string {
	if strings.TrimSpace(d.CoverKey) == "" {
		return nil
	}
	u := "/api/v1/documents/" + d.ID + "/cover"
	return &u
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	res, err := h.svc.List(Filter{
		Page:      parsePositive(q.Get("page"), 1),
		Items:     parsePositive(q.Get("items"), 10),
		Search:    strings.TrimSpace(q.Get("search")),
		Tags:      splitCSV(q.Get("tags")),
		Author:    strings.TrimSpace(q.Get("author")),
		Editorial: strings.TrimSpace(q.Get("editorial")),
		DateFrom:  strings.TrimSpace(q.Get("date_from")),
		DateTo:    strings.TrimSpace(q.Get("date_to")),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	items := make([]listItem, 0, len(res.Items))
	for _, d := range res.Items {
		items = append(items, toListItem(d))
	}
	writeJSON(w, http.StatusOK, listResponse{
		Data:     items,
		Page:     res.Page,
		Items:    res.ItemsPage,
		MaxPages: res.MaxPages,
		MaxItems: res.MaxItems,
	})
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	d, err := h.svc.GetByID(id)
	if err != nil {
		if IsNotFound(err) {
			writeError(w, http.StatusNotFound, "document not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	item := toListItem(d)
	writeJSON(w, http.StatusOK, detailItem{
		listItem:  item,
		SizeBytes: d.SizeBytes,
		MimeType:  d.MimeType,
	})
}

func (h *Handler) remove(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := h.svc.DeleteByID(id); err != nil {
		if IsNotFound(err) {
			writeError(w, http.StatusNotFound, "document not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) content(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	d, err := h.svc.GetByID(id)
	if err != nil {
		if IsNotFound(err) {
			writeError(w, http.StatusNotFound, "document not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	f, err := storage.Open(h.storageRoot, d.StorageKey)
	if err != nil {
		if os.IsNotExist(err) {
			writeError(w, http.StatusNotFound, "file not found")
			return
		}
		// Treat invalid keys / missing files as 404 to avoid leaking internals.
		if strings.Contains(err.Error(), "invalid storage key") {
			writeError(w, http.StatusNotFound, "file not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	defer f.Close()

	st, err := f.Stat()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	name := d.OriginalFilename
	if strings.TrimSpace(name) == "" {
		name = d.ID + ".pdf"
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `inline; filename="`+strings.ReplaceAll(name, `"`, "_")+`"`)
	// NOTE: responses here are served raw (never gzipped); Range requests are
	// passed straight through to ServeContent which sets Accept-Ranges.
	http.ServeContent(w, r, name, st.ModTime(), f)
}

func (h *Handler) cover(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	d, err := h.svc.GetByID(id)
	if err != nil {
		if IsNotFound(err) {
			writeError(w, http.StatusNotFound, "document not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if strings.TrimSpace(d.CoverKey) == "" {
		writeError(w, http.StatusNotFound, "cover not found")
		return
	}
	f, err := storage.OpenCover(h.storageRoot, d.CoverKey)
	if err != nil {
		if os.IsNotExist(err) || strings.Contains(err.Error(), "invalid storage key") {
			writeError(w, http.StatusNotFound, "cover not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	defer f.Close()

	st, err := f.Stat()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	head := make([]byte, 512)
	n, _ := f.Read(head)
	head = head[:n]
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	w.Header().Set("Content-Type", http.DetectContentType(head))
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	http.ServeContent(w, r, id+".cover", st.ModTime(), f)
}

func parsePositive(s string, def int) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 {
		return def
	}
	return n
}

func splitCSV(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
