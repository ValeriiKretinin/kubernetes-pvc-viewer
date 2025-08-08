package agent

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/valeriikretinin/kubernetes-pvc-viewer/internal/fsutil"
)

type HTTPServer struct {
	Router   *chi.Mux
	DataRoot string
	ReadOnly bool
}

func NewHTTPServer(dataRoot string, readOnly bool) *HTTPServer {
	s := &HTTPServer{Router: chi.NewRouter(), DataRoot: dataRoot, ReadOnly: readOnly}
	s.routes()
	return s
}

func (s *HTTPServer) routes() {
	s.Router.Get("/v1/tree", s.handleTree)
	s.Router.Get("/v1/file", s.handleGetFile)
	s.Router.Delete("/v1/file", s.handleDelete)
	s.Router.Post("/v1/upload", s.handleUpload)
}

type TreeEntry struct {
	Name  string    `json:"name"`
	Path  string    `json:"path"`
	IsDir bool      `json:"isDir"`
	Size  int64     `json:"size"`
	Mod   time.Time `json:"mod"`
}

func (s *HTTPServer) handleTree(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	p := q.Get("path")
	limit := intFromQuery(q.Get("limit"), 200)
	if limit <= 0 || limit > 1000 {
		limit = 200
	}
	offset := intFromQuery(q.Get("offset"), 0)
	full, err := fsutil.JoinSecure(s.DataRoot, p)
	if err != nil {
		http.Error(w, "bad path", http.StatusBadRequest)
		return
	}

	f, err := os.Open(full)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		http.Error(w, "stat error", http.StatusInternalServerError)
		return
	}
	if !fi.IsDir() {
		http.Error(w, "not a directory", http.StatusBadRequest)
		return
	}

	entries, err := f.Readdir(0)
	if err != nil {
		http.Error(w, "read dir error", http.StatusInternalServerError)
		return
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	end := offset + limit
	if offset > len(entries) {
		offset = len(entries)
	}
	if end > len(entries) {
		end = len(entries)
	}
	page := entries[offset:end]
	out := make([]TreeEntry, 0, len(page))
	for _, e := range page {
		out = append(out, TreeEntry{
			Name:  e.Name(),
			Path:  filepath.Join(p, e.Name()),
			IsDir: e.IsDir(),
			Size:  e.Size(),
			Mod:   e.ModTime(),
		})
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Total-Count", strconv.Itoa(len(entries)))
	_ = json.NewEncoder(w).Encode(out)
}

func (s *HTTPServer) handleGetFile(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	p := q.Get("path")
	full, err := fsutil.JoinSecure(s.DataRoot, p)
	if err != nil {
		http.Error(w, "bad path", http.StatusBadRequest)
		return
	}

	f, err := os.Open(full)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		http.Error(w, "stat error", http.StatusInternalServerError)
		return
	}
	if fi.IsDir() {
		http.Error(w, "is a directory", http.StatusBadRequest)
		return
	}

	w.Header().Set("Accept-Ranges", "bytes")
	etag := fmt.Sprintf("\"%x-%x\"", fi.ModTime().UnixNano(), fi.Size())
	w.Header().Set("ETag", etag)
	if inm := r.Header.Get("If-None-Match"); inm != "" && inm == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	// Content-Type best-effort
	if ct := mimeByName(p); ct != "" {
		w.Header().Set("Content-Type", ct)
	}

	// Range support
	if rng := r.Header.Get("Range"); rng != "" {
		start, end, ok := parseRange(rng, fi.Size())
		if !ok {
			http.Error(w, "invalid range", http.StatusRequestedRangeNotSatisfiable)
			return
		}
		if _, err := f.Seek(start, io.SeekStart); err != nil {
			http.Error(w, "seek", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end-1, fi.Size()))
		w.WriteHeader(http.StatusPartialContent)
		_, _ = io.CopyN(w, f, end-start)
		return
	}
	w.Header().Set("Content-Length", strconv.FormatInt(fi.Size(), 10))
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, f)
}

func (s *HTTPServer) handleDelete(w http.ResponseWriter, r *http.Request) {
	if s.ReadOnly {
		http.Error(w, "read-only", http.StatusForbidden)
		return
	}
	q := r.URL.Query()
	p := q.Get("path")
	full, err := fsutil.JoinSecure(s.DataRoot, p)
	if err != nil {
		http.Error(w, "bad path", http.StatusBadRequest)
		return
	}
	if err := os.RemoveAll(full); err != nil {
		http.Error(w, "delete failed", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *HTTPServer) handleUpload(w http.ResponseWriter, r *http.Request) {
	if s.ReadOnly {
		http.Error(w, "read-only", http.StatusForbidden)
		return
	}
	if err := r.ParseMultipartForm(32 << 20); err != nil { // 32 MiB
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	q := r.URL.Query()
	dir := q.Get("path")
	fullDir, err := fsutil.JoinSecure(s.DataRoot, dir)
	if err != nil {
		http.Error(w, "bad path", http.StatusBadRequest)
		return
	}
	if err := os.MkdirAll(fullDir, 0o755); err != nil {
		http.Error(w, "mkdir", http.StatusInternalServerError)
		return
	}
	files := r.MultipartForm.File["file"]
	if len(files) == 0 {
		http.Error(w, "no file", http.StatusBadRequest)
		return
	}
	for _, fh := range files {
		src, err := fh.Open()
		if err != nil {
			http.Error(w, "open", http.StatusBadRequest)
			return
		}
		defer src.Close()
		name := filepath.Base(fh.Filename)
		dstPath, err := fsutil.JoinSecure(fullDir, name)
		if err != nil {
			http.Error(w, "bad name", http.StatusBadRequest)
			return
		}
		dst, err := os.OpenFile(dstPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
		if err != nil {
			http.Error(w, "write", http.StatusInternalServerError)
			return
		}
		if _, err := io.Copy(dst, src); err != nil {
			_ = dst.Close()
			http.Error(w, "copy", http.StatusInternalServerError)
			return
		}
		_ = dst.Close()
	}
	w.WriteHeader(http.StatusCreated)
}

func parseRange(h string, size int64) (start, end int64, ok bool) {
	const prefix = "bytes="
	if len(h) < len(prefix) || h[:len(prefix)] != prefix {
		return 0, 0, false
	}
	spec := h[len(prefix):]
	dash := -1
	for i := 0; i < len(spec); i++ {
		if spec[i] == '-' {
			dash = i
			break
		}
	}
	if dash == -1 {
		return 0, 0, false
	}
	a, b := spec[:dash], spec[dash+1:]
	if a == "" && b == "" {
		return 0, 0, false
	}
	if a == "" {
		n, perr := strconv.ParseInt(b, 10, 64)
		if perr != nil || n <= 0 {
			return 0, 0, false
		}
		if n > size {
			n = size
		}
		return size - n, size, true
	}
	s, perr := strconv.ParseInt(a, 10, 64)
	if perr != nil || s < 0 || s >= size {
		return 0, 0, false
	}
	if b == "" {
		return s, size, true
	}
	e, perr := strconv.ParseInt(b, 10, 64)
	if perr != nil || e < s {
		return 0, 0, false
	}
	if e >= size {
		e = size - 1
	}
	return s, e + 1, true
}

func mimeByName(name string) string {
	ext := filepath.Ext(name)
	switch ext {
	case ".txt", ".log":
		return "text/plain; charset=utf-8"
	case ".json":
		return "application/json"
	case ".yaml", ".yml":
		return "application/x-yaml"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".pdf":
		return "application/pdf"
	default:
		return "application/octet-stream"
	}
}

func intFromQuery(v string, def int) int {
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}
