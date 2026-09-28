package app

import (
	"context"
	"encoding/json"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Thiagojm/crv-go/internal/catalog"
	"github.com/Thiagojm/crv-go/internal/session"
	"github.com/Thiagojm/crv-go/internal/store"
	"github.com/go-chi/chi/v5"
)

type Server struct {
	Store      *store.Store
	Security   *Security
	Sessions   *session.Service
	UI         fs.FS
	CatalogDir string
	Port       int
	Report     catalog.Report
	Ready      bool
	InitError  string
	OnListen   func(port int, url string)

	mu        sync.Mutex
	httpSrv   *http.Server
	shutdown  chan struct{}
	once      sync.Once
	pauseOnce sync.Once
}

type readinessDTO struct {
	Ready         bool   `json:"ready"`
	Phase         string `json:"phase"`
	SessionsOpen  bool   `json:"sessionsOpen"`
	EligibleCount int    `json:"eligibleCount"`
	ExcludedCount int    `json:"excludedCount"`
	CatalogLabel  string `json:"catalogLabel,omitempty"`
	Error         string `json:"error,omitempty"`
	Message       string `json:"message"`
}

func (s *Server) Handler() http.Handler {
	s.pauseOnce.Do(func() {
		if s.Store != nil {
			_ = s.Store.PauseNonterminal()
		}
	})
	r := chi.NewRouter()
	r.Use(s.securityMiddleware)
	r.Get("/api/bootstrap", s.handleBootstrap)
	r.Get("/api/ready", s.requireAuth(s.handleReady))
	r.Get("/api/catalog/summary", s.requireAuth(s.handleCatalogSummary))
	r.Get("/api/settings", s.requireAuth(s.handleSettings))
	r.Post("/api/shutdown", s.requireAuthMutating(s.handleShutdown))
	s.mountSessionRoutes(r)
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}
		s.serveUI(w, r)
	})
	return r
}

func (s *Server) securityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		if !isLoopbackHost(r.Host) {
			http.Error(w, "host rejeitado", http.StatusForbidden)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/") && r.Method != http.MethodGet && r.Method != http.MethodHead {
			if !originAllowed(r, s.Port) {
				http.Error(w, "origin rejeitada", http.StatusForbidden)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) requireAuth(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !cookieOK(r, s.Security.SessionCookieValue()) {
			http.Error(w, "não autenticado", http.StatusUnauthorized)
			return
		}
		h(w, r)
	}
}

func (s *Server) requireAuthMutating(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !cookieOK(r, s.Security.SessionCookieValue()) {
			http.Error(w, "não autenticado", http.StatusUnauthorized)
			return
		}
		if !csrfOK(r, s.Security.CSRFToken()) {
			http.Error(w, "CSRF inválido", http.StatusForbidden)
			return
		}
		h(w, r)
	}
}

func (s *Server) handleBootstrap(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if !s.Security.ConsumeBootstrap(token) {
		http.Error(w, "bootstrap inválido", http.StatusForbidden)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "crv_session",
		Value:    s.Security.SessionCookieValue(),
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   false,
	})
	writeJSON(w, map[string]string{"csrf": s.Security.CSRFToken()})
}

func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	dto := s.readiness()
	writeJSON(w, map[string]any{
		"ready":         dto.Ready,
		"phase":         dto.Phase,
		"sessionsOpen":  dto.SessionsOpen,
		"eligibleCount": dto.EligibleCount,
		"excludedCount": dto.ExcludedCount,
		"catalogLabel":  dto.CatalogLabel,
		"error":         dto.Error,
		"message":       dto.Message,
		"csrf":          s.Security.CSRFToken(),
	})
}

func (s *Server) handleCatalogSummary(w http.ResponseWriter, r *http.Request) {
	dto := s.readiness()
	writeJSON(w, map[string]any{
		"ready":         dto.Ready,
		"eligibleCount": dto.EligibleCount,
		"excludedCount": dto.ExcludedCount,
		"catalogLabel":  dto.CatalogLabel,
		"error":         dto.Error,
		"exclusions":    truncateExclusions(s.Report.Exclusions, 50),
		"sessionsOpen":  s.sessionsOpen(),
		"phase":         "2",
	})
}

func (s *Server) handleSettings(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]any{
		"theme":            "system",
		"durationMinutes":  10,
		"catalog":          s.readiness(),
		"sessionsOpen":     s.sessionsOpen(),
		"phase":            "2",
		"historyAvailable": false,
		"exportsAvailable": false,
		"message":          "Sessões cegas estão ativas. Histórico, estatísticas e exportações chegam nas fases seguintes.",
	})
}

func (s *Server) handleShutdown(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]string{"status": "shutting_down"})
	s.once.Do(func() {
		go func() {
			time.Sleep(150 * time.Millisecond)
			s.TriggerShutdown()
		}()
	})
}

func (s *Server) sessionsOpen() bool {
	return s.Ready && s.Report.EligibleCount >= 4
}

func (s *Server) readiness() readinessDTO {
	msg := "Catálogo pronto. Você pode iniciar uma sessão."
	if !s.Ready {
		msg = "Catálogo não está pronto. Use reparo/importação quando disponível."
		if s.InitError != "" {
			msg = s.InitError
		}
	}
	return readinessDTO{
		Ready:         s.Ready,
		Phase:         "2",
		SessionsOpen:  s.sessionsOpen(),
		EligibleCount: s.Report.EligibleCount,
		ExcludedCount: s.Report.ExcludedCount,
		CatalogLabel:  s.Report.SourceLabel,
		Error:         s.Report.Error,
		Message:       msg,
	}
}

func (s *Server) serveUI(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/")
	if path == "" {
		path = "index.html"
	}
	// No static bank or guessed image routes.
	if strings.HasPrefix(path, "farsight/") || strings.HasPrefix(path, "images/") || strings.Contains(path, "..") {
		http.NotFound(w, r)
		return
	}
	if s.UI == nil {
		http.Error(w, "UI não embutida — execute npm run build antes do go build", http.StatusServiceUnavailable)
		return
	}
	data, err := fs.ReadFile(s.UI, path)
	if err != nil {
		data, err = fs.ReadFile(s.UI, "index.html")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(data)
		return
	}
	if strings.HasSuffix(path, ".html") {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
	}
	_, _ = w.Write(data)
}

func (s *Server) ListenAndServe(ctx context.Context, openBrowser bool) error {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	s.Port = ln.Addr().(*net.TCPAddr).Port
	s.shutdown = make(chan struct{})
	s.httpSrv = &http.Server{Handler: s.Handler()}

	token := s.Security.BootstrapToken()
	base := "http://127.0.0.1:" + strconv.Itoa(s.Port)
	url := base + "/#bootstrap=" + token
	if s.OnListen != nil {
		s.OnListen(s.Port, base)
	}
	log.Printf("CRV Go escutando em %s", base)
	if openBrowser {
		if err := openBrowserURL(url); err != nil {
			log.Printf("não foi possível abrir o navegador: %v", err)
			log.Printf("abra manualmente: %s", url)
		}
	} else {
		log.Printf("URL local: %s", url)
	}

	errCh := make(chan error, 1)
	go func() { errCh <- s.httpSrv.Serve(ln) }()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.httpSrv.Shutdown(shutdownCtx)
		return ctx.Err()
	case <-s.shutdown:
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.httpSrv.Shutdown(shutdownCtx)
	case err := <-errCh:
		if err == http.ErrServerClosed {
			return nil
		}
		return err
	}
}

func (s *Server) TriggerShutdown() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.shutdown != nil {
		select {
		case <-s.shutdown:
		default:
			close(s.shutdown)
		}
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(true)
	_ = enc.Encode(v)
}

func truncateExclusions(in []catalog.Exclusion, n int) []catalog.Exclusion {
	if len(in) <= n {
		return in
	}
	return in[:n]
}

func openBrowserURL(url string) error {
	switch runtime.GOOS {
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "linux":
		return exec.Command("xdg-open", url).Start()
	default:
		return exec.Command("xdg-open", url).Start()
	}
}

// EnsureCatalog loads or installs the bundled bank into the store.
// A verified installed revision is reused even when the distribution folder is gone.
func EnsureCatalog(st *store.Store, catalogDir string) (catalog.Report, bool, string) {
	res, err := catalog.EnsureInstalled(st, catalogDir)
	if err != nil {
		if res != nil {
			msg := res.Report.Error
			if msg == "" {
				msg = err.Error()
			}
			return res.Report, false, msg
		}
		return catalog.Report{Error: err.Error(), Ready: false}, false, err.Error()
	}
	return res.Report, res.Report.Ready, ""
}
