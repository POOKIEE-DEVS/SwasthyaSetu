package api

import (
	"errors"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"os"
	"path"
	"strings"
)

// The frontend is a Next.js static export (trailingSlash: true), so a page
// like /patient/ is the file patient/index.html.
//
//   - A file is served as is.
//   - A directory serves its index.html; without the trailing slash it
//     redirects to add one, so relative links resolve.
//   - Anything else gets 404.html with status 404.
//
// Files under /_next/static/ have content hashes in their names and are
// cached for a year; everything else is revalidated on every visit, so a
// new deploy shows up at once.

func init() {
	// Types the minimal production image has no system table for.
	for ext, contentType := range map[string]string{
		".webmanifest": "application/manifest+json",
		".woff2":       "font/woff2",
		".woff":        "font/woff",
		".ttf":         "font/ttf",
		".otf":         "font/otf",
		".ico":         "image/x-icon",
		".txt":         "text/plain; charset=utf-8",
		".map":         "application/json",
		".mp4":         "video/mp4",
		".webm":        "video/webm",
		".mp3":         "audio/mpeg",
		".wav":         "audio/wav",
	} {
		_ = mime.AddExtensionType(ext, contentType)
	}
}

type staticSite struct {
	root *os.Root // cannot be escaped with ../ or symlinks
}

func newStatic(dir string) (*staticSite, error) {
	info, err := os.Stat(dir)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, errors.New("not a directory")
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, err
	}
	return &staticSite{root: root}, nil
}

func (s *staticSite) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		writeJSON(w, http.StatusMethodNotAllowed, detail{"Method Not Allowed"})
		return
	}
	name := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
	if name == "" {
		name = "."
	}
	info, err := s.root.Stat(name)
	if err == nil && info.Mode().IsRegular() {
		s.serveFile(w, r, name, info, http.StatusOK)
		return
	}
	if err == nil && info.IsDir() {
		index := path.Join(name, "index.html")
		if indexInfo, err := s.root.Stat(index); err == nil && indexInfo.Mode().IsRegular() {
			if !strings.HasSuffix(r.URL.Path, "/") {
				// Built from the cleaned name: "//host" must never become a
				// redirect to another site.
				target := "/" + name + "/"
				if r.URL.RawQuery != "" {
					target += "?" + r.URL.RawQuery
				}
				http.Redirect(w, r, target, http.StatusTemporaryRedirect)
				return
			}
			s.serveFile(w, r, index, indexInfo, http.StatusOK)
			return
		}
	}
	if info, err := s.root.Stat("404.html"); err == nil && info.Mode().IsRegular() {
		s.serveFile(w, r, "404.html", info, http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusNotFound, detail{"Not Found"})
}

func (s *staticSite) serveFile(w http.ResponseWriter, r *http.Request, name string, info fs.FileInfo, status int) {
	f, err := s.root.Open(name)
	if err != nil {
		writeJSON(w, http.StatusNotFound, detail{"Not Found"})
		return
	}
	defer f.Close()
	h := w.Header()
	if contentType := mime.TypeByExtension(path.Ext(name)); contentType != "" {
		h.Set("Content-Type", contentType)
	}
	if strings.HasPrefix(name, "_next/static/") {
		h.Set("Cache-Control", "public, max-age=31536000, immutable")
	} else {
		h.Set("Cache-Control", "no-cache")
	}
	if status == http.StatusOK {
		// Handles HEAD, ranges and If-Modified-Since.
		http.ServeContent(w, r, name, info.ModTime(), f)
		return
	}
	w.WriteHeader(status)
	if r.Method != http.MethodHead {
		_, _ = io.Copy(w, f)
	}
}
