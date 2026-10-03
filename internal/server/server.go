// Package server exposes the HTTP API, public file links and the embedded admin UI.
package server

import (
	"bytes"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gitbobobo/uiv/internal/config"
	"github.com/gitbobobo/uiv/internal/media"
	"github.com/gitbobobo/uiv/internal/store"
)

type Server struct {
	cfg     config.Config
	store   *store.Store
	version string
}

// New returns the root handler. static holds the built admin UI (index.html at its root).
func New(cfg config.Config, st *store.Store, static fs.FS, version string) http.Handler {
	s := &Server{cfg: cfg, store: st, version: version}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", s.health)
	mux.Handle("GET /api/files", s.auth(s.list))
	mux.Handle("POST /api/files", s.auth(s.upload))
	mux.Handle("DELETE /api/files/{id}", s.auth(s.delete))
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotFound, "no such API endpoint")
	})
	mux.HandleFunc("GET /f/{name}", s.serveFile)
	mux.Handle("/", staticHandler(static))
	return mux
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"version": s.version})
}

func (s *Server) auth(next http.HandlerFunc) http.Handler {
	want := []byte("Bearer " + s.cfg.Token)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := []byte(r.Header.Get("Authorization"))
		if subtle.ConstantTimeCompare(got, want) != 1 {
			w.Header().Set("WWW-Authenticate", `Bearer realm="uiv"`)
			writeError(w, http.StatusUnauthorized, "missing or wrong token: send Authorization: Bearer <UIV_TOKEN>")
			return
		}
		next(w, r)
	})
}

type fileJSON struct {
	ID        string    `json:"id"`
	URL       string    `json:"url"`
	Markdown  string    `json:"markdown"`
	Name      string    `json:"name"`
	Size      int64     `json:"size"`
	Type      string    `json:"type"`
	CreatedAt time.Time `json:"created_at"`
}

func (s *Server) view(base string, f store.File) fileJSON {
	url := base + "/f/" + f.Filename()
	alt := markdownEscaper.Replace(f.Name)
	md := fmt.Sprintf("[%s](%s)", alt, url)
	if media.IsImage(f.Type) {
		md = "!" + md
	}
	return fileJSON{ID: f.ID, URL: url, Markdown: md, Name: f.Name, Size: f.Size, Type: f.Type, CreatedAt: f.CreatedAt}
}

var markdownEscaper = strings.NewReplacer(`\`, `\\`, "[", `\[`, "]", `\]`, "\r", " ", "\n", " ")

func (s *Server) list(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 200 {
			writeError(w, http.StatusBadRequest, "limit must be between 1 and 200")
			return
		}
		limit = n
	}
	files, next, err := s.store.List(r.Context(), r.URL.Query().Get("cursor"), limit)
	if err != nil {
		if errors.Is(err, store.ErrInvalidCursor) {
			writeError(w, http.StatusBadRequest, "invalid cursor")
			return
		}
		s.internalError(w, "list files", err)
		return
	}
	base := s.baseURL(r)
	out := make([]fileJSON, 0, len(files))
	for _, f := range files {
		out = append(out, s.view(base, f))
	}
	writeJSON(w, http.StatusOK, map[string]any{"files": out, "next_cursor": next, "base_url": base})
}

func (s *Server) upload(w http.ResponseWriter, r *http.Request) {
	// Leave room for multipart boundaries and headers around the file itself.
	r.Body = http.MaxBytesReader(w, r.Body, s.cfg.MaxSize+1<<20)
	mr, err := r.MultipartReader()
	if err != nil {
		writeError(w, http.StatusBadRequest, `expected multipart/form-data with a "file" field, e.g. curl -F file=@shot.png`)
		return
	}
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			writeError(w, http.StatusBadRequest, `missing "file" field`)
			return
		}
		if err != nil {
			s.bodyError(w, err)
			return
		}
		if part.FormName() == "file" {
			s.saveUpload(w, r, part)
			return
		}
		part.Close()
	}
}

func (s *Server) saveUpload(w http.ResponseWriter, r *http.Request, part *multipart.Part) {
	head := make([]byte, media.SniffLen)
	n, err := io.ReadFull(part, head)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		s.bodyError(w, err)
		return
	}
	head = head[:n]
	if n == 0 {
		writeError(w, http.StatusBadRequest, "file is empty")
		return
	}
	mime, ext, ok := media.Detect(head)
	if !ok {
		writeError(w, http.StatusUnsupportedMediaType, fmt.Sprintf("unsupported file type %s: only images and videos are accepted", mime))
		return
	}

	tmp, err := s.store.TempFile()
	if err != nil {
		s.internalError(w, "create temp file", err)
		return
	}
	defer os.Remove(tmp.Name())
	body := io.MultiReader(bytes.NewReader(head), io.LimitReader(part, s.cfg.MaxSize-int64(n)+1))
	size, copyErr := io.Copy(tmp, body)
	closeErr := tmp.Close()
	if copyErr != nil {
		s.bodyError(w, copyErr)
		return
	}
	if size > s.cfg.MaxSize {
		writeError(w, http.StatusRequestEntityTooLarge, fmt.Sprintf("file exceeds the %d byte limit", s.cfg.MaxSize))
		return
	}
	if closeErr != nil {
		s.internalError(w, "write upload", closeErr)
		return
	}

	f, err := s.store.Commit(r.Context(), tmp.Name(), store.File{
		Ext:  ext,
		Name: cleanName(part.FileName(), ext),
		Size: size,
		Type: mime,
	})
	if err != nil {
		s.internalError(w, "commit upload", err)
		return
	}
	slog.Info("uploaded", "id", f.ID, "name", f.Name, "type", f.Type, "size", f.Size)
	writeJSON(w, http.StatusCreated, s.view(s.baseURL(r), f))
}

// cleanName keeps only the base name the client sent, for display in the admin UI.
func cleanName(name, ext string) string {
	name = strings.ReplaceAll(name, `\`, "/")
	name = strings.TrimSpace(path.Base(name))
	if name == "." || name == "/" || name == "" || !utf8.ValidString(name) {
		return "upload." + ext
	}
	if r := []rune(name); len(r) > 200 {
		name = string(r[:200])
	}
	return name
}

func (s *Server) delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	err := s.store.Delete(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "file not found")
		return
	}
	if err != nil {
		s.internalError(w, "delete file", err)
		return
	}
	slog.Info("deleted", "id", id)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) serveFile(w http.ResponseWriter, r *http.Request) {
	id, ext, _ := strings.Cut(r.PathValue("name"), ".")
	f, err := s.store.Get(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) || (err == nil && ext != f.Ext) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		s.internalError(w, "get file", err)
		return
	}
	file, err := os.Open(s.store.Path(f))
	if err != nil {
		s.internalError(w, "open file", err)
		return
	}
	defer file.Close()

	h := w.Header()
	h.Set("Content-Type", f.Type)
	h.Set("X-Content-Type-Options", "nosniff")
	// IDs are never reused, so a link's content never changes.
	h.Set("Cache-Control", "public, max-age=31536000, immutable")
	if f.Type == "image/svg+xml" {
		// SVG can carry scripts; sandbox it so it cannot run code on this origin.
		h.Set("Content-Security-Policy", "sandbox")
	}
	http.ServeContent(w, r, f.Filename(), f.CreatedAt, file)
}

// baseURL is the externally visible origin. Without UIV_PUBLIC_URL it is derived from the request,
// so links match whatever address (e.g. a frp endpoint) the client used to reach the server.
func (s *Server) baseURL(r *http.Request) string {
	if s.cfg.PublicURL != "" {
		return s.cfg.PublicURL
	}
	proto := firstValue(r.Header.Get("X-Forwarded-Proto"))
	if proto != "http" && proto != "https" {
		proto = "http"
		if r.TLS != nil {
			proto = "https"
		}
	}
	host := firstValue(r.Header.Get("X-Forwarded-Host"))
	if host == "" {
		host = r.Host
	}
	return proto + "://" + host
}

func firstValue(v string) string {
	first, _, _ := strings.Cut(v, ",")
	return strings.TrimSpace(first)
}

func staticHandler(static fs.FS) http.Handler {
	files := http.FileServerFS(static)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/assets/") {
			// Vite puts a content hash in every asset filename.
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			w.Header().Set("Cache-Control", "no-cache")
		}
		files.ServeHTTP(w, r)
	})
}

func (s *Server) bodyError(w http.ResponseWriter, err error) {
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		writeError(w, http.StatusRequestEntityTooLarge, fmt.Sprintf("file exceeds the %d byte limit", s.cfg.MaxSize))
		return
	}
	writeError(w, http.StatusBadRequest, "could not read upload: "+err.Error())
}

func (s *Server) internalError(w http.ResponseWriter, action string, err error) {
	slog.Error(action, "err", err)
	writeError(w, http.StatusInternalServerError, "internal error")
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
