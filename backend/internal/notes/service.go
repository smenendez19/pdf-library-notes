package notes

import (
	"database/sql"
	"math"
	"net/http"
	"regexp"
	"strings"
	"time"

	"pdfnotes-backend/internal/store"
)

// Limits and defaults mirrored from frontend/src/composables/useStickyNotes.ts.
const (
	MaxUpsertItems = 500
	MaxDeleteItems = 500
	MaxContentLen  = 1000

	DefaultColor  = "#fff9db"
	DefaultWidth  = 200
	DefaultHeight = 150

	MinWidth  = 140
	MaxWidth  = 480
	MinHeight = 100
	MaxHeight = 480

	MinPos = -100
	MaxPos = 200
)

var colorRe = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

// StatusError carries an HTTP status with a client-facing message.
type StatusError struct {
	Code int
	Msg  string
}

func (e *StatusError) Error() string { return e.Msg }

func badRequest(msg string) *StatusError {
	return &StatusError{Code: http.StatusBadRequest, Msg: msg}
}

func notFound(msg string) *StatusError {
	return &StatusError{Code: http.StatusNotFound, Msg: msg}
}

// Note is one sticky note row.
type Note struct {
	ID       int64
	Page     int
	Content  string
	Color    string
	X        float64
	Y        float64
	Width    float64
	Height   float64
	Visible  bool
}

// ClientNote is one upsert item as sent by the client.
type ClientNote struct {
	ID      int64
	Page    int
	Content string
	Color   string
	X       float64
	Y       float64
	Width   float64
	Height  float64
	Visible bool
}

// SavedMapping maps the client-sent id to the stored id.
type SavedMapping struct {
	ClientID int64
	ID       int64
}

// SaveResult is the outcome of a Save call.
type SaveResult struct {
	DocumentID   string
	CurrentPage  int
	NotesVersion int64
	Saved        []SavedMapping
	Deleted      []int64
}

// State is the current reading/progress state of a document.
type State struct {
	DocumentID   string
	CurrentPage  int
	NotesVersion int64
	UpdatedAt    string
}

// Service persists notes and document state.
type Service struct {
	store *store.Store
}

// NewService returns a notes Service backed by s.
func NewService(s *store.Store) *Service {
	return &Service{store: s}
}

// Save applies upserts + deletes and bumps notes_version in one transaction.
// Items with id<=0 are created (the server assigns the id); items with id>0
// update the note when it belongs to documentID, are created with that id
// when it does not exist yet, and are rejected when the id belongs to
// another document.
func (s *Service) Save(documentID string, currentPage int, upserts []ClientNote, deleteIDs []int64) (SaveResult, error) {
	if strings.TrimSpace(documentID) == "" {
		return SaveResult{}, badRequest("document_id is required")
	}
	if len(upserts) > MaxUpsertItems {
		return SaveResult{}, badRequest("upsert exceeds the 500 item limit")
	}
	if len(deleteIDs) > MaxDeleteItems {
		return SaveResult{}, badRequest("delete_ids exceeds the 500 item limit")
	}
	for _, u := range upserts {
		if u.Page < 1 {
			return SaveResult{}, badRequest("note page must be >= 1")
		}
	}
	if currentPage < 1 {
		currentPage = 1
	}

	tx, err := s.store.DB.Begin()
	if err != nil {
		return SaveResult{}, err
	}
	defer tx.Rollback()

	var dummy int
	if err := tx.QueryRow(`SELECT 1 FROM documents WHERE id = ?`, documentID).Scan(&dummy); err != nil {
		if err == sql.ErrNoRows {
			return SaveResult{}, notFound("document not found")
		}
		return SaveResult{}, err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	saved := make([]SavedMapping, 0, len(upserts))

	for _, u := range upserts {
		n := normalize(u)
		if n.ID <= 0 {
			res, err := tx.Exec(`INSERT INTO notes(document_id, page, content, color, x, y, width, height, visible, created_at, updated_at)
				VALUES(?,?,?,?,?,?,?,?,?,?,?)`,
				documentID, n.Page, n.Content, n.Color, n.X, n.Y, n.Width, n.Height, boolToInt(n.Visible), now, now)
			if err != nil {
				return SaveResult{}, err
			}
			id, err := res.LastInsertId()
			if err != nil {
				return SaveResult{}, err
			}
			saved = append(saved, SavedMapping{ClientID: u.ID, ID: id})
			continue
		}
		var owner string
		err := tx.QueryRow(`SELECT document_id FROM notes WHERE id = ?`, n.ID).Scan(&owner)
		switch {
		case err == sql.ErrNoRows:
			if _, err := tx.Exec(`INSERT INTO notes(id, document_id, page, content, color, x, y, width, height, visible, created_at, updated_at)
				VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`,
				n.ID, documentID, n.Page, n.Content, n.Color, n.X, n.Y, n.Width, n.Height, boolToInt(n.Visible), now, now); err != nil {
				return SaveResult{}, err
			}
			saved = append(saved, SavedMapping{ClientID: u.ID, ID: n.ID})
		case err != nil:
			return SaveResult{}, err
		case owner != documentID:
			return SaveResult{}, badRequest("note id belongs to another document")
		default:
			if _, err := tx.Exec(`UPDATE notes SET page=?, content=?, color=?, x=?, y=?, width=?, height=?, visible=?, updated_at=? WHERE id=?`,
				n.Page, n.Content, n.Color, n.X, n.Y, n.Width, n.Height, boolToInt(n.Visible), now, n.ID); err != nil {
				return SaveResult{}, err
			}
			saved = append(saved, SavedMapping{ClientID: u.ID, ID: n.ID})
		}
	}

	for _, id := range deleteIDs {
		if _, err := tx.Exec(`DELETE FROM notes WHERE id = ? AND document_id = ?`, id, documentID); err != nil {
			return SaveResult{}, err
		}
	}
	deleted := append([]int64{}, deleteIDs...)
	if deleted == nil {
		deleted = []int64{}
	}

	if _, err := tx.Exec(`INSERT INTO document_state(document_id, current_page, notes_version, updated_at)
		VALUES(?,?,1,?)
		ON CONFLICT(document_id) DO UPDATE SET current_page=excluded.current_page,
			notes_version=document_state.notes_version+1, updated_at=excluded.updated_at`,
		documentID, currentPage, now); err != nil {
		return SaveResult{}, err
	}

	var version int64
	if err := tx.QueryRow(`SELECT notes_version FROM document_state WHERE document_id = ?`, documentID).Scan(&version); err != nil {
		return SaveResult{}, err
	}

	if err := tx.Commit(); err != nil {
		return SaveResult{}, err
	}

	return SaveResult{
		DocumentID:   documentID,
		CurrentPage:  currentPage,
		NotesVersion: version,
		Saved:        saved,
		Deleted:      deleted,
	}, nil
}

// List returns all notes of documentID ordered by id. When sinceVersion is
// not nil and is >= the current notes_version, it returns no notes so
// clients can poll cheaply.
func (s *Service) List(documentID string, sinceVersion *int64) ([]Note, State, error) {
	if err := s.ensureDocument(documentID); err != nil {
		return nil, State{}, err
	}
	st, err := s.getState(documentID)
	if err != nil {
		return nil, State{}, err
	}
	if sinceVersion != nil && *sinceVersion >= st.NotesVersion {
		return []Note{}, st, nil
	}
	rows, err := s.store.DB.Query(`SELECT id, page, content, color, x, y, width, height, visible
		FROM notes WHERE document_id = ? ORDER BY id`, documentID)
	if err != nil {
		return nil, State{}, err
	}
	defer rows.Close()
	notes := []Note{}
	for rows.Next() {
		var n Note
		var visible int
		if err := rows.Scan(&n.ID, &n.Page, &n.Content, &n.Color, &n.X, &n.Y, &n.Width, &n.Height, &visible); err != nil {
			return nil, State{}, err
		}
		n.Visible = visible != 0
		notes = append(notes, n)
	}
	if err := rows.Err(); err != nil {
		return nil, State{}, err
	}
	return notes, st, nil
}

// GetState returns the current state of documentID, creating the row with
// defaults when it does not exist yet.
func (s *Service) GetState(documentID string) (State, error) {
	if err := s.ensureDocument(documentID); err != nil {
		return State{}, err
	}
	return s.getState(documentID)
}

func (s *Service) ensureDocument(documentID string) error {
	var dummy int
	if err := s.store.DB.QueryRow(`SELECT 1 FROM documents WHERE id = ?`, documentID).Scan(&dummy); err != nil {
		if err == sql.ErrNoRows {
			return notFound("document not found")
		}
		return err
	}
	return nil
}

func (s *Service) getState(documentID string) (State, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := s.store.DB.Exec(`INSERT OR IGNORE INTO document_state(document_id, current_page, notes_version, updated_at)
		VALUES(?,1,0,?)`, documentID, now); err != nil {
		return State{}, err
	}
	var st State
	st.DocumentID = documentID
	if err := s.store.DB.QueryRow(`SELECT current_page, notes_version, COALESCE(updated_at,'')
		FROM document_state WHERE document_id = ?`, documentID).Scan(&st.CurrentPage, &st.NotesVersion, &st.UpdatedAt); err != nil {
		return State{}, err
	}
	return st, nil
}

// normalize applies the frontend (useStickyNotes) constraints: content is
// truncated to 1000 runes, invalid colors fall back to the default,
// positions clamp to [-100,200] (non-finite becomes 0) and sizes clamp to
// their ranges (missing/non-finite becomes the default).
func normalize(n ClientNote) ClientNote {
	if r := []rune(n.Content); len(r) > MaxContentLen {
		n.Content = string(r[:MaxContentLen])
	}
	if !colorRe.MatchString(n.Color) {
		n.Color = DefaultColor
	}
	n.X = clampPos(n.X)
	n.Y = clampPos(n.Y)
	n.Width = normalizeSize(n.Width, MinWidth, MaxWidth, DefaultWidth)
	n.Height = normalizeSize(n.Height, MinHeight, MaxHeight, DefaultHeight)
	return n
}

func clampPos(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	return math.Min(MaxPos, math.Max(MinPos, v))
}

func normalizeSize(v float64, min, max, def float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) || v == 0 {
		return def
	}
	return math.Min(max, math.Max(min, v))
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
