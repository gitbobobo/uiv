// Package store keeps uploaded files on disk and their metadata in SQLite.
package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

var (
	ErrNotFound      = errors.New("not found")
	ErrInvalidCursor = errors.New("invalid cursor")
)

type File struct {
	ID        string
	Ext       string
	Name      string
	Size      int64
	Type      string
	CreatedAt time.Time
}

// Filename is the on-disk and public name: "{id}.{ext}".
func (f File) Filename() string { return f.ID + "." + f.Ext }

type Store struct {
	db       *sql.DB
	filesDir string
	tmpDir   string
}

func Open(dataDir string) (*Store, error) {
	s := &Store{
		filesDir: filepath.Join(dataDir, "files"),
		tmpDir:   filepath.Join(dataDir, "tmp"),
	}
	// Leftover temp files are failed uploads from a previous run.
	if err := os.RemoveAll(s.tmpDir); err != nil {
		return nil, err
	}
	for _, dir := range []string{s.filesDir, s.tmpDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
	}
	dsn := "file:" + filepath.ToSlash(filepath.Join(dataDir, "uiv.db")) + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS files (
		id TEXT PRIMARY KEY,
		ext TEXT NOT NULL,
		name TEXT NOT NULL,
		size INTEGER NOT NULL,
		type TEXT NOT NULL,
		created_at INTEGER NOT NULL
	);
	CREATE INDEX IF NOT EXISTS files_created ON files (created_at DESC, id DESC);`); err != nil {
		db.Close()
		return nil, fmt.Errorf("init schema: %w", err)
	}
	s.db = db
	return s, nil
}

func (s *Store) Close() error { return s.db.Close() }

// TempFile creates a scratch file on the same volume as the final location so Commit can rename it.
func (s *Store) TempFile() (*os.File, error) {
	return os.CreateTemp(s.tmpDir, "upload-*")
}

// Commit moves a finished temp file into place and records its metadata.
func (s *Store) Commit(ctx context.Context, tmpPath string, f File) (File, error) {
	f.ID = newID()
	f.CreatedAt = time.Now().UTC().Truncate(time.Millisecond)
	dst := s.Path(f)
	if err := os.Rename(tmpPath, dst); err != nil {
		return File{}, err
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO files (id, ext, name, size, type, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		f.ID, f.Ext, f.Name, f.Size, f.Type, f.CreatedAt.UnixMilli())
	if err != nil {
		os.Remove(dst)
		return File{}, err
	}
	return f, nil
}

func (s *Store) Path(f File) string { return filepath.Join(s.filesDir, f.Filename()) }

func (s *Store) Get(ctx context.Context, id string) (File, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, ext, name, size, type, created_at FROM files WHERE id = ?`, id)
	f, err := scan(row)
	if errors.Is(err, sql.ErrNoRows) {
		return File{}, ErrNotFound
	}
	return f, err
}

// List returns files newest first. cursor is the opaque value returned by a previous call; next is
// empty when there are no more files.
func (s *Store) List(ctx context.Context, cursor string, limit int) (files []File, next string, err error) {
	query := `SELECT id, ext, name, size, type, created_at FROM files`
	args := []any{}
	if cursor != "" {
		ms, id, ok := parseCursor(cursor)
		if !ok {
			return nil, "", ErrInvalidCursor
		}
		query += ` WHERE (created_at, id) < (?, ?)`
		args = append(args, ms, id)
	}
	query += ` ORDER BY created_at DESC, id DESC LIMIT ?`
	args = append(args, limit+1)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	for rows.Next() {
		f, err := scan(rows)
		if err != nil {
			return nil, "", err
		}
		files = append(files, f)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}
	if len(files) > limit {
		files = files[:limit]
		last := files[len(files)-1]
		next = strconv.FormatInt(last.CreatedAt.UnixMilli(), 10) + "." + last.ID
	}
	return files, next, nil
}

func (s *Store) Delete(ctx context.Context, id string) error {
	f, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx, `DELETE FROM files WHERE id = ?`, id); err != nil {
		return err
	}
	if err := os.Remove(s.Path(f)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

type scanner interface{ Scan(...any) error }

func scan(r scanner) (File, error) {
	var f File
	var ms int64
	if err := r.Scan(&f.ID, &f.Ext, &f.Name, &f.Size, &f.Type, &ms); err != nil {
		return File{}, err
	}
	f.CreatedAt = time.UnixMilli(ms).UTC()
	return f, nil
}

func parseCursor(c string) (int64, string, bool) {
	msStr, id, ok := strings.Cut(c, ".")
	if !ok || id == "" {
		return 0, "", false
	}
	ms, err := strconv.ParseInt(msStr, 10, 64)
	return ms, id, err == nil
}

const idAlphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

// newID returns 10 random base62 characters (~59 bits), enough that public links cannot be guessed.
func newID() string {
	b := make([]byte, 10)
	buf := make([]byte, 1)
	for i := 0; i < len(b); {
		if _, err := io.ReadFull(rand.Reader, buf); err != nil {
			panic(err)
		}
		// Reject values >= 248 so every character is equally likely.
		if buf[0] >= 248 {
			continue
		}
		b[i] = idAlphabet[buf[0]%62]
		i++
	}
	return string(b)
}
