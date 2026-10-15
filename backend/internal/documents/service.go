package documents

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"pdfnotes-backend/internal/storage"
	"pdfnotes-backend/internal/store"
)

// Doc holds one catalog row with its tags.
type Doc struct {
	ID               string
	Title            string
	Author           string
	Editorial        string
	Description      string
	UploadDate       string
	StorageKey       string
	CoverKey         string
	OriginalFilename string
	MimeType         string
	SizeBytes        int64
	SHA256           string
	Tags             []string
}

// Filter carries the list query parameters.
type Filter struct {
	Page      int
	Items     int
	Search    string
	Tags      []string
	Author    string
	Editorial string
	DateFrom  string
	DateTo    string
}

// ListResult is a paginated document list.
type ListResult struct {
	Items     []Doc
	Page      int
	ItemsPage int
	MaxPages  int
	MaxItems  int
}

// Service queries documents from the store.
type Service struct {
	store       *store.Store
	storageRoot string
}

// NewService returns a document Service backed by s.
func NewService(s *store.Store) *Service {
	return &Service{store: s}
}

// List returns the paginated, filtered document list ordered by
// upload_date DESC, rowid DESC.
func (s *Service) List(f Filter) (ListResult, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.Items < 1 {
		f.Items = 10
	}
	if f.Items > 50 {
		f.Items = 50
	}

	where, args := buildWhere(f)

	var total int
	countQ := `SELECT COUNT(*) FROM documents d` + where
	if err := s.store.DB.QueryRow(countQ, args...).Scan(&total); err != nil {
		return ListResult{}, err
	}

	maxPages := 1
	if total > 0 {
		maxPages = (total + f.Items - 1) / f.Items
	}
	if f.Page > maxPages {
		f.Page = maxPages
	}
	offset := (f.Page - 1) * f.Items

	q := `SELECT d.id, d.title,
		COALESCE(d.author,''), COALESCE(d.editorial,''), COALESCE(d.description,''),
		COALESCE(d.upload_date,''), d.storage_key,
		COALESCE(d.cover_key,''),
		COALESCE(d.original_filename,''), COALESCE(d.mime_type,''),
		COALESCE(d.size_bytes,0)
		FROM documents d` + where + `
		ORDER BY d.upload_date DESC, d.rowid DESC
		LIMIT ? OFFSET ?`
	qArgs := append(append([]any{}, args...), f.Items, offset)
	rows, err := s.store.DB.Query(q, qArgs...)
	if err != nil {
		return ListResult{}, err
	}
	defer rows.Close()

	docs := make([]Doc, 0)
	for rows.Next() {
		var d Doc
		if err := rows.Scan(&d.ID, &d.Title, &d.Author, &d.Editorial,
			&d.Description, &d.UploadDate, &d.StorageKey, &d.CoverKey,
			&d.OriginalFilename, &d.MimeType, &d.SizeBytes); err != nil {
			return ListResult{}, err
		}
		docs = append(docs, d)
	}
	if err := rows.Err(); err != nil {
		return ListResult{}, err
	}

	if err := s.loadTags(docs); err != nil {
		return ListResult{}, err
	}

	return ListResult{
		Items:     docs,
		Page:      f.Page,
		ItemsPage: f.Items,
		MaxPages:  maxPages,
		MaxItems:  total,
	}, nil
}

// GetByID returns a single document with its tags, or sql.ErrNoRows.
func (s *Service) GetByID(id string) (Doc, error) {
	var d Doc
	err := s.store.DB.QueryRow(`SELECT d.id, d.title,
		COALESCE(d.author,''), COALESCE(d.editorial,''), COALESCE(d.description,''),
		COALESCE(d.upload_date,''), d.storage_key,
		COALESCE(d.cover_key,''),
		COALESCE(d.original_filename,''), COALESCE(d.mime_type,''),
		COALESCE(d.size_bytes,0), COALESCE(d.sha256,'')
		FROM documents d WHERE d.id = ?`, id).Scan(
		&d.ID, &d.Title, &d.Author, &d.Editorial, &d.Description,
		&d.UploadDate, &d.StorageKey, &d.CoverKey, &d.OriginalFilename, &d.MimeType, &d.SizeBytes, &d.SHA256)
	if err != nil {
		return Doc{}, err
	}
	docs := []Doc{d}
	if err := s.loadTags(docs); err != nil {
		return Doc{}, err
	}
	return docs[0], nil
}

// GetBySHA returns the document with the given sha256 hex digest,
// or sql.ErrNoRows when no document holds that content.
func (s *Service) GetBySHA(sha string) (Doc, error) {
	var id string
	if err := s.store.DB.QueryRow(`SELECT id FROM documents WHERE sha256 = ?`, sha).Scan(&id); err != nil {
		return Doc{}, err
	}
	return s.GetByID(id)
}

// CreateInput carries the fields needed to register an uploaded document.
type CreateInput struct {
	Title            string
	Author           string
	Editorial        string
	Description      string
	UploadDate       string
	OriginalFilename string
	MimeType         string
	StorageKey       string
	CoverKey         string
	SHA256           string
	SizeBytes        int64
	Tags             []string
}

// Create inserts a document with its tags and initial state in one
// transaction and returns the stored row.
func (s *Service) Create(in CreateInput) (Doc, error) {
	id, err := newDocID()
	if err != nil {
		return Doc{}, err
	}
	now := time.Now().UTC().Format(time.RFC3339)

	tx, err := s.store.DB.Begin()
	if err != nil {
		return Doc{}, err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`INSERT INTO documents(id, title, author, editorial,
		description, storage_key, cover_key, original_filename, mime_type, size_bytes,
		sha256, upload_date, created_at, updated_at)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		id, in.Title, in.Author, in.Editorial, in.Description,
		in.StorageKey, nullIfEmpty(in.CoverKey), in.OriginalFilename, in.MimeType, in.SizeBytes,
		in.SHA256, in.UploadDate, now, now); err != nil {
		return Doc{}, err
	}
	for _, t := range in.Tags {
		if _, err := tx.Exec(`INSERT OR IGNORE INTO document_tags(document_id, tag) VALUES(?,?)`, id, t); err != nil {
			return Doc{}, err
		}
	}
	if _, err := tx.Exec(`INSERT INTO document_state(document_id, current_page, notes_version, updated_at)
		VALUES(?,?,?,?)`, id, 1, 0, now); err != nil {
		return Doc{}, err
	}
	if err := tx.Commit(); err != nil {
		return Doc{}, err
	}

	tags := append([]string{}, in.Tags...)
	if tags == nil {
		tags = []string{}
	}
	return Doc{
		ID:               id,
		Title:            in.Title,
		Author:           in.Author,
		Editorial:        in.Editorial,
		Description:      in.Description,
		UploadDate:       in.UploadDate,
		StorageKey:       in.StorageKey,
		CoverKey:         in.CoverKey,
		OriginalFilename: in.OriginalFilename,
		MimeType:         in.MimeType,
		SizeBytes:        in.SizeBytes,
		SHA256:           in.SHA256,
		Tags:             tags,
	}, nil
}

// SetCoverKey assigns a cover key to a document that has none.
func (s *Service) SetCoverKey(id, coverKey string) error {
	_, err := s.store.DB.Exec(`UPDATE documents SET cover_key = ?, updated_at = ? WHERE id = ? AND (cover_key IS NULL OR cover_key = '')`,
		coverKey, time.Now().UTC().Format(time.RFC3339), id)
	return err
}

// DeleteByID removes the document row and deletes its stored files only
// when no other document references them. It returns the deleted Doc,
// or sql.ErrNoRows when id is empty or unknown.
func (s *Service) DeleteByID(id string) (Doc, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return Doc{}, sql.ErrNoRows
	}
	d, err := s.GetByID(id)
	if err != nil {
		return Doc{}, err
	}

	tx, err := s.store.DB.Begin()
	if err != nil {
		return Doc{}, err
	}
	defer tx.Rollback()

	res, err := tx.Exec(`DELETE FROM documents WHERE id = ?`, id)
	if err != nil {
		return Doc{}, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return Doc{}, err
	}
	if n == 0 {
		return Doc{}, sql.ErrNoRows
	}
	if err := tx.Commit(); err != nil {
		return Doc{}, err
	}

	var pdfRefs int
	if err := s.store.DB.QueryRow(`SELECT COUNT(*) FROM documents WHERE storage_key = ?`, d.StorageKey).Scan(&pdfRefs); err != nil {
		return Doc{}, err
	}
	if pdfRefs == 0 {
		_ = storage.Delete(s.storageRoot, d.StorageKey)
	}
	if strings.TrimSpace(d.CoverKey) != "" {
		var coverRefs int
		if err := s.store.DB.QueryRow(`SELECT COUNT(*) FROM documents WHERE cover_key = ?`, d.CoverKey).Scan(&coverRefs); err != nil {
			return Doc{}, err
		}
		if coverRefs == 0 {
			_ = storage.DeleteCover(s.storageRoot, d.CoverKey)
		}
	}
	return d, nil
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// newDocID returns 16 crypto/rand bytes hex-encoded (32 hex chars).
func newDocID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

func buildWhere(f Filter) (string, []any) {
	var conds []string
	var args []any

	if s := strings.TrimSpace(f.Search); s != "" {
		like := "%" + s + "%"
		conds = append(conds, `(d.title LIKE ? COLLATE NOCASE OR d.author LIKE ? COLLATE NOCASE OR d.editorial LIKE ? COLLATE NOCASE)`)
		args = append(args, like, like, like)
	}
	if len(f.Tags) > 0 {
		holders := strings.Repeat("?,", len(f.Tags))
		holders = strings.TrimSuffix(holders, ",")
		conds = append(conds, fmt.Sprintf(`d.id IN (SELECT t.document_id FROM document_tags t WHERE t.tag IN (%s))`, holders))
		for _, t := range f.Tags {
			args = append(args, t)
		}
	}
	if a := strings.TrimSpace(f.Author); a != "" {
		conds = append(conds, `d.author LIKE ? COLLATE NOCASE`)
		args = append(args, "%"+a+"%")
	}
	if e := strings.TrimSpace(f.Editorial); e != "" {
		conds = append(conds, `d.editorial LIKE ? COLLATE NOCASE`)
		args = append(args, "%"+e+"%")
	}
	if d := strings.TrimSpace(f.DateFrom); d != "" {
		conds = append(conds, `substr(d.upload_date,1,10) >= ?`)
		args = append(args, d)
	}
	if d := strings.TrimSpace(f.DateTo); d != "" {
		conds = append(conds, `substr(d.upload_date,1,10) <= ?`)
		args = append(args, d)
	}

	if len(conds) == 0 {
		return "", nil
	}
	return " WHERE " + strings.Join(conds, " AND "), args
}

func (s *Service) loadTags(docs []Doc) error {
	for i := range docs {
		docs[i].Tags = []string{}
	}
	if len(docs) == 0 {
		return nil
	}
	holders := strings.Repeat("?,", len(docs))
	holders = strings.TrimSuffix(holders, ",")
	args := make([]any, 0, len(docs))
	for _, d := range docs {
		args = append(args, d.ID)
	}
	rows, err := s.store.DB.Query(
		fmt.Sprintf(`SELECT document_id, tag FROM document_tags WHERE document_id IN (%s) ORDER BY tag`, holders), args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	byID := make(map[string][]string, len(docs))
	for rows.Next() {
		var id, tag string
		if err := rows.Scan(&id, &tag); err != nil {
			return err
		}
		byID[id] = append(byID[id], tag)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for i := range docs {
		if tags, ok := byID[docs[i].ID]; ok {
			docs[i].Tags = tags
		}
	}
	return nil
}

// IsNotFound reports whether err is a missing-row error.
func IsNotFound(err error) bool {
	return err == sql.ErrNoRows
}
