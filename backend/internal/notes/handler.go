package notes

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"pdfnotes-backend/internal/store"
)

// Handler serves the notes sync endpoints.
type Handler struct {
	svc *Service
}

// NewHandler returns a Handler backed by s.
func NewHandler(s *store.Store) *Handler {
	return &Handler{svc: NewService(s)}
}

// Register mounts the notes routes on mux.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/notes/save", h.save)
	mux.HandleFunc("GET /api/v1/documents/{id}/notes", h.list)
	mux.HandleFunc("GET /api/v1/documents/{id}/state", h.state)
}

type upsertItem struct {
	ID      int64   `json:"id"`
	Page    int     `json:"page"`
	Content string  `json:"content"`
	Color   string  `json:"color"`
	X       float64 `json:"x"`
	Y       float64 `json:"y"`
	Width   float64 `json:"width"`
	Height  float64 `json:"height"`
	Visible bool    `json:"visible"`
}

type saveRequest struct {
	DocumentID  string       `json:"document_id"`
	CurrentPage int          `json:"current_page"`
	Upsert      []upsertItem `json:"upsert"`
	DeleteIDs   []int64      `json:"delete_ids"`
}

type savedItem struct {
	ClientID int64 `json:"client_id"`
	ID       int64 `json:"id"`
}

type saveResponse struct {
	DocumentID   string      `json:"document_id"`
	CurrentPage  int         `json:"current_page"`
	NotesVersion int64       `json:"notes_version"`
	Saved        []savedItem `json:"saved"`
	Deleted      []int64     `json:"deleted"`
}

func (h *Handler) save(w http.ResponseWriter, r *http.Request) {
	if ct := r.Header.Get("Content-Type"); !strings.Contains(strings.ToLower(ct), "application/json") {
		writeError(w, http.StatusBadRequest, "Content-Type must be application/json")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)
	var req saveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	upserts := make([]ClientNote, 0, len(req.Upsert))
	for _, u := range req.Upsert {
		upserts = append(upserts, ClientNote{
			ID: u.ID, Page: u.Page, Content: u.Content, Color: u.Color,
			X: u.X, Y: u.Y, Width: u.Width, Height: u.Height, Visible: u.Visible,
		})
	}
	res, err := h.svc.Save(req.DocumentID, req.CurrentPage, upserts, req.DeleteIDs)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	saved := make([]savedItem, 0, len(res.Saved))
	for _, m := range res.Saved {
		saved = append(saved, savedItem{ClientID: m.ClientID, ID: m.ID})
	}
	writeJSON(w, http.StatusOK, saveResponse{
		DocumentID:   res.DocumentID,
		CurrentPage:  res.CurrentPage,
		NotesVersion: res.NotesVersion,
		Saved:        saved,
		Deleted:      res.Deleted,
	})
}

type noteItem struct {
	ID      int64   `json:"id"`
	Page    int     `json:"page"`
	Content string  `json:"content"`
	Color   string  `json:"color"`
	X       float64 `json:"x"`
	Y       float64 `json:"y"`
	Width   float64 `json:"width"`
	Height  float64 `json:"height"`
	Visible bool    `json:"visible"`
}

type listResponse struct {
	DocumentID        string     `json:"document_id"`
	NotesVersion      int64      `json:"notes_version"`
	CurrentPage       int        `json:"current_page"`
	Notes             []noteItem `json:"notes"`
	DeletedTombstones []int64    `json:"deleted_tombstones"`
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var since *int64
	if raw := strings.TrimSpace(r.URL.Query().Get("since_version")); raw != "" {
		n, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || n < 0 {
			writeError(w, http.StatusBadRequest, "since_version must be a non-negative integer")
			return
		}
		since = &n
	}
	notes, st, err := h.svc.List(id, since)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	items := make([]noteItem, 0, len(notes))
	for _, n := range notes {
		items = append(items, noteItem{
			ID: n.ID, Page: n.Page, Content: n.Content, Color: n.Color,
			X: n.X, Y: n.Y, Width: n.Width, Height: n.Height, Visible: n.Visible,
		})
	}
	writeJSON(w, http.StatusOK, listResponse{
		DocumentID:        id,
		NotesVersion:      st.NotesVersion,
		CurrentPage:       st.CurrentPage,
		Notes:             items,
		DeletedTombstones: []int64{},
	})
}

type stateResponse struct {
	DocumentID   string `json:"document_id"`
	CurrentPage  int    `json:"current_page"`
	NotesVersion int64  `json:"notes_version"`
	UpdatedAt    string `json:"updated_at"`
}

func (h *Handler) state(w http.ResponseWriter, r *http.Request) {
	st, err := h.svc.GetState(r.PathValue("id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, stateResponse{
		DocumentID:   st.DocumentID,
		CurrentPage:  st.CurrentPage,
		NotesVersion: st.NotesVersion,
		UpdatedAt:    st.UpdatedAt,
	})
}

func writeServiceError(w http.ResponseWriter, err error) {
	var se *StatusError
	if errors.As(err, &se) {
		writeError(w, se.Code, se.Msg)
		return
	}
	writeError(w, http.StatusInternalServerError, "internal server error")
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
