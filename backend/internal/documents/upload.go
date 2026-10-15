package documents

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"path"
	"path/filepath"
	"strings"
	"time"

	"pdfnotes-backend/internal/config"
	"pdfnotes-backend/internal/storage"
)

const (
	// maxUploadOverhead is the headroom over config.DefaultMaxUploadBytes
	// granted to the request body so multipart framing does not eat into
	// the 100 MiB file budget.
	maxUploadOverhead = 1 << 20
	// multipartMaxMemory caps the RAM used by ParseMultipartForm; the rest
	// of the upload spills to temporary files on disk.
	multipartMaxMemory = 32 << 20
	// maxCoverBytes caps the decoded cover image (PNG/JPEG) at 2 MiB.
	maxCoverBytes = 2 << 20
)

type uploadResponse struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Tags        []string `json:"tags"`
	UploadDate  string   `json:"upload_date"`
	Author      string   `json:"author"`
	Editorial   string   `json:"editorial"`
	Description string   `json:"description"`
	FileURL     string   `json:"fileUrl"`
	CoverURL    *string  `json:"coverUrl"`
	SizeBytes   int64    `json:"size_bytes"`
	SHA256      string   `json:"sha256"`
}

// Upload handles POST /api/v1/documents/upload (multipart/form-data).
// It stores the PDF atomically, deduplicates content by SHA-256 and
// registers the metadata in SQLite.
func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	// The request itself may arrive gzipped: unwrap it before parsing so
	// curl --compressed (or any gzip uploader) keeps working. Stored
	// responses are still served raw, never gzipped.
	if strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
		zr, err := gzip.NewReader(r.Body)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid gzip body")
			return
		}
		defer zr.Close()
		r.Body = zr
	}
	r.Body = http.MaxBytesReader(w, r.Body, config.DefaultMaxUploadBytes+maxUploadOverhead)
	if err := r.ParseMultipartForm(multipartMaxMemory); err != nil {
		if strings.Contains(err.Error(), "request body too large") {
			writeError(w, http.StatusRequestEntityTooLarge, "upload exceeds the 100 MB limit")
			return
		}
		writeError(w, http.StatusBadRequest, "invalid multipart body")
		return
	}

	f, fh, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "field 'file' is required")
		return
	}
	defer f.Close()

	original := sanitizeFilename(fh.Filename)
	if strings.ToLower(filepath.Ext(original)) != ".pdf" {
		writeError(w, http.StatusBadRequest, "only .pdf files are accepted")
		return
	}
	head := make([]byte, 512)
	n, _ := io.ReadFull(f, head)
	head = head[:n]
	if !strings.Contains(http.DetectContentType(head), "application/pdf") {
		writeError(w, http.StatusBadRequest, "file is not a valid PDF")
		return
	}

	title := strings.TrimSpace(r.FormValue("title"))
	if title == "" {
		title = strings.TrimSuffix(original, filepath.Ext(original))
	}
	uploadDate := strings.TrimSpace(r.FormValue("upload_date"))
	if uploadDate == "" {
		uploadDate = time.Now().UTC().Format("2006-01-02")
	} else if _, err := time.Parse("2006-01-02", uploadDate); err != nil {
		writeError(w, http.StatusBadRequest, "upload_date must be YYYY-MM-DD")
		return
	}

	// Stream the upload while hashing so no file-sized buffer is retained.
	sum := sha256.New()
	if _, err := io.Copy(sum, io.MultiReader(bytes.NewReader(head), f)); err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	sha := hex.EncodeToString(sum.Sum(nil))
	key := storage.KeyFor(sha)

	// Optional cover: invalid or oversized covers are ignored so the
	// PDF upload still succeeds.
	coverBytes, hasCover := parseCover(r)

	// Content already catalogued: reuse the existing row and bytes.
	if existing, err := h.svc.GetBySHA(sha); err == nil {
		if hasCover && strings.TrimSpace(existing.CoverKey) == "" {
			if _, _, err := storage.SaveCoverAtomic(h.storageRoot, sha, bytes.NewReader(coverBytes)); err == nil {
				coverKey := storage.CoverKeyFor(sha)
				_ = h.svc.SetCoverKey(existing.ID, coverKey)
				existing.CoverKey = coverKey
			}
		}
		writeJSON(w, http.StatusCreated, toUploadResponse(existing))
		return
	}

	// Reuse the stored bytes when the file is already on disk (dedup);
	// otherwise persist the upload atomically.
	reused := false
	if ef, err := storage.Open(h.storageRoot, key); err == nil {
		ef.Close()
		reused = true
	}
	var size int64
	if reused {
		ef, err := storage.Open(h.storageRoot, key)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}
		st, err := ef.Stat()
		_ = ef.Close()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}
		size = st.Size()
	} else {
		if _, err := f.Seek(0, io.SeekStart); err != nil {
			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}
		if _, size, err = storage.SaveAtomic(h.storageRoot, sha, f); err != nil {
			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}
	}

	// Persist the cover bytes deduplicated by the PDF sha. A cover file
	// already on disk is shared, never overwritten destructively here
	// beyond the atomic replace of identical content.
	coverKey := ""
	coverReused := false
	if hasCover {
		if ef, err := storage.OpenCover(h.storageRoot, storage.CoverKeyFor(sha)); err == nil {
			ef.Close()
			coverReused = true
			coverKey = storage.CoverKeyFor(sha)
		} else if _, _, err := storage.SaveCoverAtomic(h.storageRoot, sha, bytes.NewReader(coverBytes)); err == nil {
			coverKey = storage.CoverKeyFor(sha)
		} else {
			hasCover = false
		}
	}

	doc, err := h.svc.Create(CreateInput{
		Title:            title,
		Author:           strings.TrimSpace(r.FormValue("author")),
		Editorial:        strings.TrimSpace(r.FormValue("editorial")),
		Description:      strings.TrimSpace(r.FormValue("description")),
		UploadDate:       uploadDate,
		OriginalFilename: original,
		MimeType:         "application/pdf",
		StorageKey:       key,
		CoverKey:         coverKey,
		SHA256:           sha,
		SizeBytes:        size,
		Tags:             parseTags(r.FormValue("tags")),
	})
	if err != nil {
		// Never delete deduplicated bytes owned by another document.
		if !reused {
			_ = storage.Delete(h.storageRoot, key)
		}
		if hasCover && !coverReused && coverKey != "" {
			_ = storage.DeleteCover(h.storageRoot, coverKey)
		}
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusCreated, toUploadResponse(doc))
}

func toUploadResponse(d Doc) uploadResponse {
	tags := d.Tags
	if tags == nil {
		tags = []string{}
	}
	return uploadResponse{
		ID:          d.ID,
		Title:       d.Title,
		Tags:        tags,
		UploadDate:  d.UploadDate,
		Author:      d.Author,
		Editorial:   d.Editorial,
		Description: d.Description,
		FileURL:     "/api/v1/documents/" + d.ID + "/content",
		CoverURL:    coverURLFor(d),
		SizeBytes:   d.SizeBytes,
		SHA256:      d.SHA256,
	}
}

// parseCover reads the optional multipart field 'cover' as raw PNG/JPEG
// bytes or as a dataURL (data:image/png|jpeg;base64,...). It returns
// ok=false when absent or softly invalid so the upload can continue
// without a cover.
func parseCover(r *http.Request) ([]byte, bool) {
	if cf, _, err := r.FormFile("cover"); err == nil {
		defer cf.Close()
		b, err := io.ReadAll(io.LimitReader(cf, maxCoverBytes+1))
		if err != nil || len(b) == 0 || len(b) > maxCoverBytes || !isCoverImage(b) {
			return nil, false
		}
		return b, true
	}
	raw := strings.TrimSpace(r.FormValue("cover"))
	if raw == "" {
		return nil, false
	}
	b, ok := decodeCoverDataURL(raw)
	if !ok || len(b) == 0 || len(b) > maxCoverBytes || !isCoverImage(b) {
		return nil, false
	}
	return b, true
}

// decodeCoverDataURL decodes data:image/png|jpeg;base64,... payloads.
func decodeCoverDataURL(raw string) ([]byte, bool) {
	if !strings.HasPrefix(strings.ToLower(raw), "data:image/") {
		return nil, false
	}
	comma := strings.Index(raw, ",")
	if comma < 0 {
		return nil, false
	}
	meta := strings.ToLower(strings.TrimSpace(raw[:comma]))
	if !strings.Contains(meta, ";base64") {
		return nil, false
	}
	mime := strings.TrimPrefix(meta, "data:")
	if i := strings.Index(mime, ";"); i >= 0 {
		mime = mime[:i]
	}
	if mime != "image/png" && mime != "image/jpeg" && mime != "image/jpg" {
		return nil, false
	}
	b, err := base64.StdEncoding.DecodeString(strings.TrimSpace(raw[comma+1:]))
	if err != nil {
		if b, err = base64.RawStdEncoding.DecodeString(strings.TrimSpace(raw[comma+1:])); err != nil {
			return nil, false
		}
	}
	return b, true
}

// isCoverImage reports whether b holds PNG or JPEG magic bytes.
func isCoverImage(b []byte) bool {
	if len(b) >= 8 && b[0] == 0x89 && b[1] == 0x50 && b[2] == 0x4E && b[3] == 0x47 &&
		b[4] == 0x0D && b[5] == 0x0A && b[6] == 0x1A && b[7] == 0x0A {
		return true
	}
	if len(b) >= 3 && b[0] == 0xFF && b[1] == 0xD8 && b[2] == 0xFF {
		return true
	}
	return false
}

// sanitizeFilename keeps only the base name so paths from upload clients
// cannot escape the storage root.
func sanitizeFilename(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, "\\", "/")
	return strings.TrimSpace(path.Base(name))
}

// parseTags accepts a comma-separated list or a JSON array of strings.
func parseTags(raw string) []string {
	out := []string{}
	if raw = strings.TrimSpace(raw); raw == "" {
		return out
	}
	if strings.HasPrefix(raw, "[") {
		var arr []string
		if err := json.Unmarshal([]byte(raw), &arr); err == nil {
			for _, t := range arr {
				if t = strings.TrimSpace(t); t != "" {
					out = append(out, t)
				}
			}
			return out
		}
	}
	for _, p := range strings.Split(raw, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
