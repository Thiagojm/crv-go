package app

import (
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/Thiagojm/crv-go/internal/catalog"
	"github.com/go-chi/chi/v5"
)

// catalogMaxArchiveBytes mirrors the import package compressed bound (1 GiB).
const catalogMaxArchiveBytes = 1 << 30

func (s *Server) mountCatalogRoutes(r chi.Router) {
	r.Post("/api/catalog/import/folder", s.requireAuthMutating(s.handleCatalogImportFolder))
	r.Post("/api/catalog/import/zip", s.requireAuthMutating(s.handleCatalogImportZip))
	r.Post("/api/catalog/repair", s.requireAuthMutating(s.handleCatalogRepair))
}

func (s *Server) handleCatalogImportFolder(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Path string `json:"path"`
	}
	if err := readJSON(w, r, &in); err != nil {
		return
	}
	in.Path = strings.TrimSpace(in.Path)
	if in.Path == "" {
		writeErr(w, http.StatusBadRequest, "missing_path", "caminho da pasta ausente")
		return
	}
	if !filepath.IsAbs(in.Path) {
		writeErr(w, http.StatusBadRequest, "relative_path", "o caminho da pasta deve ser absoluto")
		return
	}
	label := "import-folder"
	if base := filepath.Base(in.Path); base != "" && base != "." && base != string(filepath.Separator) {
		label = "import-folder:" + base
	}
	s.runCatalogReplace(w, func() (*catalog.InstallResult, error) {
		return catalog.ImportFolder(s.Store, in.Path, label)
	})
}

func (s *Server) handleCatalogImportZip(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, catalogMaxArchiveBytes+1)
	ct := r.Header.Get("Content-Type")
	var reader io.Reader
	switch {
	case strings.HasPrefix(ct, "multipart/form-data"):
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid_multipart", "formulário multipart inválido")
			return
		}
		f, _, err := r.FormFile("archive")
		if err != nil {
			writeErr(w, http.StatusBadRequest, "missing_archive", "arquivo ZIP ausente")
			return
		}
		defer f.Close()
		reader = f
	default:
		reader = r.Body
	}
	s.runCatalogReplace(w, func() (*catalog.InstallResult, error) {
		return catalog.ImportZip(s.Store, reader, "import-zip")
	})
}

func (s *Server) handleCatalogRepair(w http.ResponseWriter, r *http.Request) {
	s.runCatalogReplace(w, func() (*catalog.InstallResult, error) {
		return catalog.Repair(s.Store, s.CatalogDir)
	})
}

func (s *Server) runCatalogReplace(w http.ResponseWriter, fn func() (*catalog.InstallResult, error)) {
	s.catalogMu.Lock()
	defer s.catalogMu.Unlock()

	active, err := s.Store.ActiveSession()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "store_error", "falha ao consultar sessão ativa")
		return
	}
	if active != nil {
		writeErr(w, http.StatusConflict, "session_active", "substitução indisponível enquanto há sessão em andamento")
		return
	}

	res, err := fn()
	if err != nil {
		msg := err.Error()
		if res != nil && res.Report.Error != "" {
			msg = res.Report.Error
		}
		status := http.StatusBadRequest
		writeJSONStatus(w, status, map[string]any{
			"error":   "catalog_import_failed",
			"message": msg,
			"report":  reportOrEmpty(res),
			"ready":   s.Ready,
		})
		return
	}
	s.Report = res.Report
	s.Ready = res.Report.Ready
	writeJSON(w, map[string]any{
		"report":     res.Report,
		"revisionId": res.RevisionID,
		"ready":      s.Ready,
	})
}

func reportOrEmpty(res *catalog.InstallResult) catalog.Report {
	if res == nil {
		return catalog.Report{}
	}
	return res.Report
}

func writeJSONStatus(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(true)
	_ = enc.Encode(v)
}
