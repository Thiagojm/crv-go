package app

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/Thiagojm/crv-go/internal/backup"
	exportdoc "github.com/Thiagojm/crv-go/internal/export"
	"github.com/Thiagojm/crv-go/internal/session"
	"github.com/Thiagojm/crv-go/internal/store"
	"github.com/go-chi/chi/v5"
)

func (s *Server) mountExportRoutes(r chi.Router) {
	r.Get("/api/exports/csv", s.requireAuth(s.handleExportCSV))
	r.Get("/api/exports/sessions/{id}/print", s.requireAuth(s.handleExportPrint))
	r.Get("/api/backup", s.requireAuthExclusive(s.handleBackupDownload))
	r.Post("/api/backup/restore", s.requireAuthExclusiveMutating(s.handleBackupRestore))
}

func (s *Server) maintenanceBlocked(w http.ResponseWriter) bool {
	if atomic.LoadInt32(&s.maintaining) != 0 {
		writeErr(w, http.StatusServiceUnavailable, "maintenance", "manutenção em andamento; tente novamente em instantes")
		return true
	}
	return false
}

func (s *Server) handleExportCSV(w http.ResponseWriter, r *http.Request) {
	if s.maintenanceBlocked(w) {
		return
	}
	q := r.URL.Query()
	var states []string
	if raw := strings.TrimSpace(q.Get("state")); raw != "" {
		for _, part := range strings.Split(raw, ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				states = append(states, part)
			}
		}
	}
	rows, err := exportdoc.ListReports(s.Store, store.HistoryFilter{
		States: states,
		From:   strings.TrimSpace(q.Get("from")),
		To:     strings.TrimSpace(q.Get("to")),
	})
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid_filter", err.Error())
		return
	}
	raw, err := exportdoc.BuildCSV(rows)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "export_failed", "falha ao gerar CSV")
		return
	}
	name := "crv-historico-" + time.Now().UTC().Format("20060102T150405Z") + ".csv"
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+name+`"`)
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(raw)
}

func (s *Server) handleExportPrint(w http.ResponseWriter, r *http.Request) {
	if s.maintenanceBlocked(w) {
		return
	}
	id := chi.URLParam(r, "id")
	row, err := s.Store.GetSession(id)
	if err != nil || row == nil {
		http.NotFound(w, r)
		return
	}
	rep, err := exportdoc.BuildSessionReport(row, s.Store)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "export_failed", "falha ao montar relatório")
		return
	}
	html, err := exportdoc.RenderSessionHTML(rep)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "export_failed", "falha ao gerar HTML de impressão")
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(html)
}

func (s *Server) handleBackupDownload(w http.ResponseWriter, r *http.Request) {
	out := backup.DefaultBackupPath(s.Store.DataDir)
	if err := backup.Create(s.Store, out); err != nil {
		writeErr(w, http.StatusBadRequest, "backup_failed", err.Error())
		return
	}
	f, err := os.Open(out)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "backup_failed", "falha ao abrir o arquivo de backup")
		return
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "backup_failed", "falha ao ler o arquivo de backup")
		return
	}
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filepath.Base(out)+`"`)
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-CRV-Backup-Note", "Contém atribuições ocultas de sessões.")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", info.Size()))
	_, _ = io.Copy(w, f)
}

func (s *Server) handleBackupRestore(w http.ResponseWriter, r *http.Request) {
	// Exclusive dataMu is held by requireAuthExclusiveMutating, draining in-flight
	// shared handlers. maintaining blocks new auth attempts for the whole swap.
	atomic.StoreInt32(&s.maintaining, 1)
	defer atomic.StoreInt32(&s.maintaining, 0)

	active, err := s.Store.ActiveSession()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "store_error", "falha ao consultar sessão ativa")
		return
	}
	if active != nil {
		writeErr(w, http.StatusConflict, "session_active", "restauração indisponível enquanto há sessão em andamento")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, catalogMaxArchiveBytes+1)
	ct := r.Header.Get("Content-Type")
	confirm := false
	var reader io.Reader
	switch {
	case strings.HasPrefix(ct, "multipart/form-data"):
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid_multipart", "formulário multipart inválido")
			return
		}
		confirm = strings.EqualFold(r.FormValue("confirm"), "true") || r.FormValue("confirm") == "1"
		f, _, err := r.FormFile("archive")
		if err != nil {
			writeErr(w, http.StatusBadRequest, "missing_archive", "arquivo ZIP de backup ausente")
			return
		}
		defer f.Close()
		reader = f
	default:
		writeErr(w, http.StatusBadRequest, "invalid_multipart", "envie multipart com archive e confirm=true")
		return
	}
	if !confirm {
		writeErr(w, http.StatusBadRequest, "confirm_required", "confirmação explícita necessária para substituir os dados")
		return
	}

	dataDir := s.Store.DataDir
	uploadDir := filepath.Join(dataDir, "backups")
	if err := os.MkdirAll(uploadDir, 0o755); err != nil {
		writeErr(w, http.StatusInternalServerError, "restore_failed", "falha ao preparar pasta de backups")
		return
	}
	uploadPath := filepath.Join(uploadDir, "restore-upload-"+time.Now().UTC().Format("20060102T150405Z")+".zip")
	uf, err := os.OpenFile(uploadPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "restore_failed", "falha ao gravar upload")
		return
	}
	if _, err := io.Copy(uf, reader); err != nil {
		_ = uf.Close()
		_ = os.Remove(uploadPath)
		writeErr(w, http.StatusBadRequest, "restore_failed", "falha ao ler o arquivo enviado")
		return
	}
	_ = uf.Close()

	staging := backup.DefaultStagingDir(dataDir)
	if err := backup.ValidateArchive(uploadPath, staging); err != nil {
		_ = os.Remove(uploadPath)
		writeErr(w, http.StatusBadRequest, "invalid_backup", err.Error())
		return
	}

	prePath := filepath.Join(uploadDir, "pre-restore-"+time.Now().UTC().Format("20060102T150405Z")+".zip")
	if err := backup.Create(s.Store, prePath); err != nil {
		_ = os.RemoveAll(staging)
		writeErr(w, http.StatusInternalServerError, "prebackup_failed", "falha ao criar cópia de segurança pré-restauração: "+err.Error())
		return
	}
	if err := backup.PrepareSwap(dataDir, staging, prePath); err != nil {
		_ = os.RemoveAll(staging)
		writeErr(w, http.StatusInternalServerError, "restore_failed", err.Error())
		return
	}

	if err := backup.ApplySwap(dataDir, func() error {
		if s.Store != nil {
			return s.Store.Close()
		}
		return nil
	}); err != nil {
		_ = backup.RollbackSwap(dataDir)
		// Best-effort reopen previous data.
		_ = s.reopenAfterRestore(dataDir)
		writeErr(w, http.StatusInternalServerError, "restore_failed", "falha na troca dos dados: "+err.Error())
		return
	}

	if err := s.reopenAfterRestore(dataDir); err != nil {
		_ = backup.RollbackSwap(dataDir)
		_ = s.reopenAfterRestore(dataDir)
		writeErr(w, http.StatusInternalServerError, "restore_failed", "falha ao reabrir dados restaurados: "+err.Error())
		return
	}
	if err := backup.CommitSwap(dataDir); err != nil {
		writeErr(w, http.StatusInternalServerError, "restore_failed", "dados restaurados, mas falha ao limpar staging: "+err.Error())
		return
	}

	sec, err := NewSecurity()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "restore_failed", "restauração ok, mas falha ao renovar credenciais")
		return
	}
	s.Security = sec

	writeJSON(w, map[string]any{
		"ok":             true,
		"preBackupPath":  prePath,
		"bootstrapToken": sec.BootstrapToken(),
		"message":        "Restauração concluída. A cópia anterior está em " + prePath + ". Reautentique esta aba; sessões ativas restauradas ficam pausadas.",
		"ready":          s.Ready,
		"phase":          "4",
	})
}

func (s *Server) reopenAfterRestore(dataDir string) error {
	st, err := store.Open(dataDir)
	if err != nil {
		return err
	}
	if err := st.PauseNonterminal(); err != nil {
		_ = st.Close()
		return err
	}
	report, ready, initErr := EnsureCatalog(st, s.CatalogDir)
	s.Store = st
	s.Sessions = session.New(st)
	s.Report = report
	s.Ready = ready
	s.InitError = initErr
	if !ready && initErr == "" {
		s.InitError = report.Error
	}
	return nil
}
