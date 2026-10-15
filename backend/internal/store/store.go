package store

import (
	"database/sql"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// Store wraps the SQLite connection pool.
type Store struct {
	DB *sql.DB
}

// Open opens (creating parent dirs if needed) the SQLite database at path,
// enables WAL + foreign keys, and creates the schema if it does not exist.
func Open(path string) (*Store, error) {
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}

	for _, pragma := range []string{
		`PRAGMA journal_mode=WAL`,
		`PRAGMA foreign_keys=ON`,
	} {
		if _, err := db.Exec(pragma); err != nil {
			_ = db.Close()
			return nil, err
		}
	}

	if err := migrate(db); err != nil {
		_ = db.Close()
		return nil, err
	}

	return &Store{DB: db}, nil
}

// Close closes the underlying database.
func (s *Store) Close() error {
	return s.DB.Close()
}

func migrate(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS documents(
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			author TEXT,
			editorial TEXT,
			description TEXT,
			storage_key TEXT NOT NULL UNIQUE,
			original_filename TEXT,
			mime_type TEXT,
			size_bytes INTEGER,
			sha256 TEXT UNIQUE,
			upload_date TEXT,
			created_at TEXT,
			updated_at TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS document_tags(
			document_id TEXT NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
			tag TEXT NOT NULL,
			PRIMARY KEY(document_id, tag)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_document_tags_tag ON document_tags(tag)`,
		`CREATE TABLE IF NOT EXISTS notes(
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			document_id TEXT NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
			page INTEGER NOT NULL,
			content TEXT NOT NULL DEFAULT '',
			color TEXT NOT NULL DEFAULT '#fff9db',
			x REAL NOT NULL DEFAULT 0,
			y REAL NOT NULL DEFAULT 0,
			width REAL NOT NULL DEFAULT 200,
			height REAL NOT NULL DEFAULT 150,
			visible INTEGER NOT NULL DEFAULT 1,
			created_at TEXT,
			updated_at TEXT
		)`,
		`CREATE INDEX IF NOT EXISTS idx_notes_document_id ON notes(document_id)`,
		`CREATE TABLE IF NOT EXISTS document_state(
			document_id TEXT PRIMARY KEY REFERENCES documents(id) ON DELETE CASCADE,
			current_page INTEGER DEFAULT 1,
			notes_version INTEGER DEFAULT 0,
			updated_at TEXT
		)`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return err
		}
	}
	return ensureCoverColumn(db)
}

// ensureCoverColumn adds documents.cover_key on pre-existing databases.
func ensureCoverColumn(db *sql.DB) error {
	rows, err := db.Query(`PRAGMA table_info(documents)`)
	if err != nil {
		return err
	}
	defer rows.Close()
	var cid int
	var name, ctype string
	var notnull int
	var dflt sql.NullString
	var pk int
	for rows.Next() {
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return err
		}
		if name == "cover_key" {
			return rows.Err()
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	_, err = db.Exec(`ALTER TABLE documents ADD COLUMN cover_key TEXT`)
	return err
}
