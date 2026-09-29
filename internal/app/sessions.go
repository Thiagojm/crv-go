package app

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Thiagojm/crv-go/internal/catalog"
	"github.com/Thiagojm/crv-go/internal/session"
	"github.com/Thiagojm/crv-go/internal/store"
	"github.com/go-chi/chi/v5"
)

const maxBody = session.MaxRecordBytes

type sessionDTO struct {
	ID                  string          `json:"id"`
	Code                string          `json:"code"`
	State               string          `json:"state"`
	Revision            int             `json:"revision"`
	Paused              bool            `json:"paused"`
	YouHoldLease        bool            `json:"youHoldLease"`
	Step                int             `json:"step"`
	CollectionMs        int             `json:"collectionMs"`
	ChoiceMs            int             `json:"choiceMs"`
	TimingSeq           int             `json:"timingSeq"`
	HelpVersion         string          `json:"helpVersion"`
	Record              json.RawMessage `json:"record"`
	Alternatives        []altDTO        `json:"alternatives,omitempty"`
	TentativeChoice     *string         `json:"tentativeChoice,omitempty"`
	TentativeConfidence *int            `json:"tentativeConfidence,omitempty"`
	Choice              *string         `json:"choice,omitempty"`
	Correct             *string         `json:"correct,omitempty"`
	Hit                 *bool           `json:"hit,omitempty"`
	Feedback            *feedbackDTO    `json:"feedback,omitempty"`
	Comment             string          `json:"comment,omitempty"`
	AbandonedAfterAlts  bool            `json:"abandonedAfterAlts,omitempty"`
	CreatedAt           string          `json:"createdAt"`
}

type altDTO struct {
	Position string `json:"position"`
	URL      string `json:"url"`
}

type feedbackDTO struct {
	Position    string          `json:"position"`
	Label       string          `json:"label"`
	Description string          `json:"description"`
	Credit      string          `json:"credit"`
	SAM         json.RawMessage `json:"sam,omitempty"`
	Sources     []sourceDTO     `json:"sources,omitempty"`
}

type sourceDTO struct {
	ID          string `json:"id"`
	Pool        string `json:"pool"`
	Label       string `json:"label"`
	Description string `json:"description"`
	Credit      string `json:"credit"`
}

type sessionEnvelope struct {
	Session    sessionDTO `json:"session"`
	LeaseToken string     `json:"leaseToken,omitempty"`
}

func (s *Server) mountSessionRoutes(r chi.Router) {
	r.Post("/api/sessions", s.requireAuthMutating(s.handleCreateSession))
	r.Get("/api/sessions/current", s.requireAuth(s.handleCurrentSession))
	r.Get("/api/sessions/{id}", s.requireAuth(s.handleGetSession))
	r.Put("/api/sessions/{id}/record", s.requireAuthMutating(s.handleSaveRecord))
	r.Post("/api/sessions/{id}/lock", s.requireAuthMutating(s.handleLock))
	r.Put("/api/sessions/{id}/choice", s.requireAuthMutating(s.handleTentative))
	r.Post("/api/sessions/{id}/confirm", s.requireAuthMutating(s.handleConfirm))
	r.Post("/api/sessions/{id}/abandon", s.requireAuthMutating(s.handleAbandon))
	r.Put("/api/sessions/{id}/comment", s.requireAuthMutating(s.handleComment))
	r.Post("/api/sessions/{id}/lease", s.requireAuthMutating(s.handleLease))
	r.Post("/api/sessions/{id}/timing", s.requireAuthMutating(s.handleTiming))
	r.Post("/api/sessions/{id}/pause", s.requireAuthMutating(s.handlePause))
	r.Post("/api/sessions/{id}/resume", s.requireAuthMutating(s.handleResume))
	r.Post("/api/sessions/{id}/example", s.requireAuthMutating(s.handleExample))
	r.Post("/api/sessions/{id}/loaded", s.requireAuthMutating(s.handleLoaded))
	r.Get("/api/sessions/{id}/images/{pos}", s.requireAuth(s.handleSessionImage))
}

func (s *Server) svc() *session.Service {
	if s.Sessions == nil {
		s.Sessions = session.New(s.Store)
	}
	return s.Sessions
}

func (s *Server) handleCreateSession(w http.ResponseWriter, r *http.Request) {
	if !s.Ready {
		writeErr(w, http.StatusConflict, "catalog_not_ready", "catálogo não está pronto")
		return
	}
	var in struct {
		OperationID   string `json:"operationId"`
		Disposition   string `json:"disposition"`
		Concentration string `json:"concentration"`
	}
	if err := readJSON(w, r, &in); err != nil {
		return
	}
	s.catalogMu.Lock()
	defer s.catalogMu.Unlock()
	row, lease, err := s.svc().Create(session.CreateInput{
		OperationID: in.OperationID, Disposition: in.Disposition, Concentration: in.Concentration,
	})
	if err != nil {
		writeSessionErr(w, err)
		return
	}
	writeJSON(w, sessionEnvelope{Session: s.toDTO(row, lease), LeaseToken: lease})
}

func (s *Server) handleCurrentSession(w http.ResponseWriter, r *http.Request) {
	row, err := s.svc().Current()
	if err != nil {
		writeSessionErr(w, err)
		return
	}
	if row == nil {
		writeJSON(w, map[string]any{"session": nil})
		return
	}
	lease := r.Header.Get("X-Lease-Token")
	writeJSON(w, sessionEnvelope{Session: s.toDTO(row, lease)})
}

func (s *Server) handleGetSession(w http.ResponseWriter, r *http.Request) {
	row, err := s.svc().Get(chi.URLParam(r, "id"))
	if err != nil {
		writeSessionErr(w, err)
		return
	}
	writeJSON(w, sessionEnvelope{Session: s.toDTO(row, r.Header.Get("X-Lease-Token"))})
}

func (s *Server) handleSaveRecord(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ExpectedRevision int            `json:"expectedRevision"`
		Record           session.Record `json:"record"`
	}
	if err := readJSON(w, r, &in); err != nil {
		return
	}
	row, err := s.svc().SaveRecord(chi.URLParam(r, "id"), leaseOf(r), in.ExpectedRevision, in.Record)
	if err != nil {
		writeSessionErr(w, err)
		return
	}
	writeJSON(w, sessionEnvelope{Session: s.toDTO(row, leaseOf(r))})
}

func (s *Server) handleLock(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ExpectedRevision int `json:"expectedRevision"`
	}
	if err := readJSON(w, r, &in); err != nil {
		return
	}
	row, err := s.svc().Lock(chi.URLParam(r, "id"), leaseOf(r), in.ExpectedRevision)
	if err != nil {
		writeSessionErr(w, err)
		return
	}
	writeJSON(w, sessionEnvelope{Session: s.toDTO(row, leaseOf(r))})
}

func (s *Server) handleTentative(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ExpectedRevision int    `json:"expectedRevision"`
		Choice           string `json:"choice"`
		Confidence       *int   `json:"confidence"`
	}
	if err := readJSON(w, r, &in); err != nil {
		return
	}
	row, err := s.svc().Tentative(chi.URLParam(r, "id"), leaseOf(r), in.ExpectedRevision, in.Choice, in.Confidence)
	if err != nil {
		writeSessionErr(w, err)
		return
	}
	writeJSON(w, sessionEnvelope{Session: s.toDTO(row, leaseOf(r))})
}

func (s *Server) handleConfirm(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ExpectedRevision int    `json:"expectedRevision"`
		Choice           string `json:"choice"`
		Confidence       *int   `json:"confidence"`
	}
	if err := readJSON(w, r, &in); err != nil {
		return
	}
	row, err := s.svc().Confirm(chi.URLParam(r, "id"), leaseOf(r), in.ExpectedRevision, in.Choice, in.Confidence)
	if err != nil {
		writeSessionErr(w, err)
		return
	}
	writeJSON(w, sessionEnvelope{Session: s.toDTO(row, leaseOf(r))})
}

func (s *Server) handleAbandon(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ExpectedRevision int `json:"expectedRevision"`
	}
	if err := readJSON(w, r, &in); err != nil {
		return
	}
	row, err := s.svc().Abandon(chi.URLParam(r, "id"), leaseOf(r), in.ExpectedRevision)
	if err != nil {
		writeSessionErr(w, err)
		return
	}
	writeJSON(w, sessionEnvelope{Session: s.toDTO(row, leaseOf(r))})
}

func (s *Server) handleComment(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Comment string `json:"comment"`
	}
	if err := readJSON(w, r, &in); err != nil {
		return
	}
	row, err := s.svc().Comment(chi.URLParam(r, "id"), in.Comment)
	if err != nil {
		writeSessionErr(w, err)
		return
	}
	writeJSON(w, sessionEnvelope{Session: s.toDTO(row, leaseOf(r))})
}

func (s *Server) handleLease(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Token    string `json:"token"`
		Transfer bool   `json:"transfer"`
	}
	if err := readJSON(w, r, &in); err != nil {
		return
	}
	if in.Token == "" {
		in.Token = leaseOf(r)
	}
	token, row, err := s.svc().Heartbeat(chi.URLParam(r, "id"), in.Token, in.Transfer)
	if err != nil {
		writeSessionErr(w, err)
		return
	}
	writeJSON(w, sessionEnvelope{Session: s.toDTO(row, token), LeaseToken: token})
}

func (s *Server) handleTiming(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Seq     int `json:"seq"`
		DeltaMs int `json:"deltaMs"`
	}
	if err := readJSON(w, r, &in); err != nil {
		return
	}
	row, err := s.svc().Timing(chi.URLParam(r, "id"), leaseOf(r), in.Seq, in.DeltaMs)
	if err != nil {
		writeSessionErr(w, err)
		return
	}
	writeJSON(w, map[string]any{
		"collectionMs": row.CollectionMs,
		"choiceMs":     row.ChoiceMs,
		"timingSeq":    row.TimingSeq,
	})
}

func (s *Server) handlePause(w http.ResponseWriter, r *http.Request) {
	_ = r.Body.Close()
	s.setPaused(w, r, true)
}

func (s *Server) handleResume(w http.ResponseWriter, r *http.Request) {
	_ = r.Body.Close()
	s.setPaused(w, r, false)
}

func (s *Server) setPaused(w http.ResponseWriter, r *http.Request, paused bool) {
	row, err := s.svc().SetPaused(chi.URLParam(r, "id"), leaseOf(r), paused)
	if err != nil {
		writeSessionErr(w, err)
		return
	}
	writeJSON(w, sessionEnvelope{Session: s.toDTO(row, leaseOf(r))})
}

func (s *Server) handleExample(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Step int `json:"step"`
	}
	if err := readJSON(w, r, &in); err != nil {
		return
	}
	row, err := s.svc().RecordExample(chi.URLParam(r, "id"), leaseOf(r), in.Step)
	if err != nil {
		writeSessionErr(w, err)
		return
	}
	writeJSON(w, sessionEnvelope{Session: s.toDTO(row, leaseOf(r))})
}

func (s *Server) handleLoaded(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Positions []string `json:"positions"`
	}
	if err := readJSON(w, r, &in); err != nil {
		return
	}
	if err := s.svc().RecordLoaded(chi.URLParam(r, "id"), in.Positions); err != nil {
		writeSessionErr(w, err)
		return
	}
	writeJSON(w, map[string]string{"status": "ok"})
}

func (s *Server) handleSessionImage(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	pos := strings.ToUpper(chi.URLParam(r, "pos"))
	row, err := s.svc().Get(id)
	if err != nil || row == nil || !session.ImagesAuthorized(row) {
		http.NotFound(w, r)
		return
	}
	sha := session.SHAForPos(row, pos)
	if sha == "" {
		http.NotFound(w, r)
		return
	}
	img, err := s.Store.Image(row.CatalogRevisionID, sha)
	if err != nil || img == nil {
		http.Error(w, "imagem ausente — reparo necessário", http.StatusServiceUnavailable)
		return
	}
	path := filepath.Join(store.RevisionDir(s.Store.DataDir, row.CatalogRevisionID), filepath.FromSlash(img.DisplayRelpath))
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() {
		http.Error(w, "imagem ausente — reparo necessário", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-store")
	http.ServeFile(w, r, path)
}

func (s *Server) toDTO(row *store.SessionRow, lease string) sessionDTO {
	rec := row.RecordJSON
	if row.LockedRecordJSON != "" && row.State != "collecting" {
		rec = row.LockedRecordJSON
	}
	dto := sessionDTO{
		ID:           row.ID,
		Code:         row.Code,
		State:        row.State,
		Revision:     row.Revision,
		Paused:       row.Paused,
		YouHoldLease: session.HoldsLease(row, lease, time.Now().UTC()),
		Step:         row.Step,
		CollectionMs: row.CollectionMs,
		ChoiceMs:     row.ChoiceMs,
		TimingSeq:    row.TimingSeq,
		HelpVersion:  row.HelpVersion,
		Record:       json.RawMessage(rec),
		Comment:      row.Comment,
		CreatedAt:    row.CreatedAt,
	}
	if session.ImagesAuthorized(row) {
		dto.Alternatives = []altDTO{
			{Position: "A", URL: "/api/sessions/" + row.ID + "/images/A"},
			{Position: "B", URL: "/api/sessions/" + row.ID + "/images/B"},
			{Position: "C", URL: "/api/sessions/" + row.ID + "/images/C"},
			{Position: "D", URL: "/api/sessions/" + row.ID + "/images/D"},
		}
		if row.TentativeChoice != "" {
			c := row.TentativeChoice
			dto.TentativeChoice = &c
			dto.TentativeConfidence = row.TentativeConfidence
		}
	}
	if row.State == "abandoned" {
		dto.AbandonedAfterAlts = row.AbandonedAfterAlts
	}
	if row.State == "completed" {
		ch := row.ConfirmedChoice
		cr := session.CorrectPos(row)
		dto.Choice = &ch
		dto.Correct = &cr
		hit := row.Hit != nil && *row.Hit == 1
		dto.Hit = &hit
		dto.TentativeConfidence = nil
		if row.ConfirmedConfidence != nil {
			dto.TentativeConfidence = row.ConfirmedConfidence
		}
		dto.Feedback = s.feedbackFor(row)
	}
	return dto
}

func (s *Server) feedbackFor(row *store.SessionRow) *feedbackDTO {
	img, err := s.Store.Image(row.CatalogRevisionID, row.TargetSHA)
	if err != nil || img == nil {
		return &feedbackDTO{Position: session.CorrectPos(row), Label: "Alvo"}
	}
	var sources []catalog.SourceRecord
	_ = json.Unmarshal([]byte(img.SourcesJSON), &sources)
	fb := &feedbackDTO{Position: session.CorrectPos(row)}
	for _, src := range sources {
		fb.Sources = append(fb.Sources, sourceDTO{
			ID: src.ID, Pool: src.Pool, Label: src.Label,
			Description: src.Description, Credit: src.Credit,
		})
		if src.ID == img.PrimarySourceID || fb.Label == "" {
			fb.Label = src.Label
			if fb.Label == "" {
				fb.Label = src.ID
			}
			fb.Description = src.Description
			fb.Credit = src.Credit
			fb.SAM = src.SAMAttributes
		}
	}
	if fb.Label == "" {
		fb.Label = "Alvo"
	}
	return fb
}

func leaseOf(r *http.Request) string {
	return r.Header.Get("X-Lease-Token")
}

func readJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBody)
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(dst); err != nil {
		if errors.Is(err, io.EOF) {
			writeErr(w, http.StatusBadRequest, "invalid_json", "corpo ausente")
			return err
		}
		writeErr(w, http.StatusBadRequest, "invalid_json", "pedido inválido")
		return err
	}
	return nil
}

func writeErr(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": code, "message": msg})
}

func writeSessionErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, session.ErrNotFound):
		writeErr(w, http.StatusNotFound, "not_found", "sessão não encontrada")
	case errors.Is(err, session.ErrActiveExists):
		writeErr(w, http.StatusConflict, "active_exists", "já existe uma sessão em andamento")
	case errors.Is(err, session.ErrConflict):
		writeErr(w, http.StatusConflict, "revision_conflict", "há uma versão mais nova no servidor")
	case errors.Is(err, session.ErrChoiceConflict):
		writeErr(w, http.StatusConflict, "choice_conflict", "a escolha já foi confirmada de outro modo")
	case errors.Is(err, session.ErrLease):
		writeErr(w, http.StatusConflict, "lease_held", "outra aba está editando esta sessão")
	case errors.Is(err, session.ErrPaused):
		writeErr(w, http.StatusConflict, "paused", "retome a sessão para editar")
	case errors.Is(err, session.ErrBadState):
		writeErr(w, http.StatusConflict, "bad_state", "operação não permitida neste estado")
	case errors.Is(err, session.ErrCatalogNotReady), errors.Is(err, session.ErrTooFewImages):
		writeErr(w, http.StatusConflict, "catalog_not_ready", "catálogo não está pronto")
	case errors.Is(err, session.ErrRand):
		writeErr(w, http.StatusServiceUnavailable, "random_failed", "não foi possível sortear a sessão")
	case errors.Is(err, session.ErrValidation):
		writeErr(w, http.StatusBadRequest, "invalid_record", "registro inválido")
	default:
		if err != nil && strings.Contains(err.Error(), "excede") || (err != nil && strings.Contains(err.Error(), "inválid")) {
			writeErr(w, http.StatusBadRequest, "invalid_record", "registro inválido")
			return
		}
		writeErr(w, http.StatusInternalServerError, "internal", "falha interna")
	}
}
