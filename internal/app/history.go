package app

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/Thiagojm/crv-go/internal/store"
	"github.com/go-chi/chi/v5"
)

type historyItemDTO struct {
	ID                 string `json:"id"`
	Code               string `json:"code"`
	State              string `json:"state"`
	CreatedAt          string `json:"createdAt"`
	CollectionMs       int    `json:"collectionMs"`
	ChoiceMs           int    `json:"choiceMs"`
	AbandonedAfterAlts bool   `json:"abandonedAfterAlts,omitempty"`
	ConfirmedChoice    string `json:"confirmedChoice,omitempty"`
	Hit                *bool  `json:"hit,omitempty"`
	CompletedAt        string `json:"completedAt,omitempty"`
	AbandonedAt        string `json:"abandonedAt,omitempty"`
}

type historyPageDTO struct {
	Items []historyItemDTO `json:"items"`
	Total int              `json:"total"`
}

type chartPointDTO struct {
	N           int     `json:"n"`
	Hits        int     `json:"hits"`
	Reference   float64 `json:"reference"`
	Code        string  `json:"code"`
	CompletedAt string  `json:"completedAt"`
	Hit         bool    `json:"hit"`
}

type statsDTO struct {
	Initiated        int             `json:"initiated"`
	Active           int             `json:"active"`
	Completed        int             `json:"completed"`
	AbandonedBefore  int             `json:"abandonedBefore"`
	AbandonedAfter   int             `json:"abandonedAfter"`
	ConfirmedChoices int             `json:"confirmedChoices"`
	Hits             int             `json:"hits"`
	HitRate          *float64        `json:"hitRate"`
	Chart            []chartPointDTO `json:"chart"`
}

func (s *Server) mountHistoryRoutes(r chi.Router) {
	r.Get("/api/history", s.requireAuth(s.handleHistory))
	r.Get("/api/statistics", s.requireAuth(s.handleStatistics))
}

func (s *Server) handleHistory(w http.ResponseWriter, r *http.Request) {
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
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	page, err := s.Store.ListHistory(store.HistoryFilter{
		States: states,
		From:   strings.TrimSpace(q.Get("from")),
		To:     strings.TrimSpace(q.Get("to")),
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid_filter", err.Error())
		return
	}
	items := make([]historyItemDTO, 0, len(page.Items))
	for _, it := range page.Items {
		dto := historyItemDTO{
			ID: it.ID, Code: it.Code, State: it.State, CreatedAt: it.CreatedAt,
			CollectionMs: it.CollectionMs, ChoiceMs: it.ChoiceMs,
			AbandonedAfterAlts: it.AbandonedAfterAlts,
			ConfirmedChoice:    it.ConfirmedChoice,
			CompletedAt:        it.CompletedAt,
			AbandonedAt:        it.AbandonedAt,
		}
		if it.Hit != nil {
			v := *it.Hit == 1
			dto.Hit = &v
		}
		items = append(items, dto)
	}
	writeJSON(w, historyPageDTO{Items: items, Total: page.Total})
}

func (s *Server) handleStatistics(w http.ResponseWriter, r *http.Request) {
	stats, err := s.Store.Stats()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal", "falha ao calcular estatísticas")
		return
	}
	chart := make([]chartPointDTO, 0, len(stats.Chart))
	for _, p := range stats.Chart {
		chart = append(chart, chartPointDTO{
			N: p.N, Hits: p.Hits, Reference: p.Reference,
			Code: p.Code, CompletedAt: p.CompletedAt, Hit: p.Hit,
		})
	}
	writeJSON(w, statsDTO{
		Initiated:        stats.Initiated,
		Active:           stats.Active,
		Completed:        stats.Completed,
		AbandonedBefore:  stats.AbandonedBefore,
		AbandonedAfter:   stats.AbandonedAfter,
		ConfirmedChoices: stats.ConfirmedChoices,
		Hits:             stats.Hits,
		HitRate:          stats.HitRate,
		Chart:            chart,
	})
}
