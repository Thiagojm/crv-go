package export

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"sort"
	"strings"

	"github.com/Thiagojm/crv-go/internal/session"
)

//go:embed session.html
var sessionTmplFS embed.FS

var sessionTmpl = template.Must(template.New("session.html").ParseFS(sessionTmplFS, "session.html"))

type sessionHTMLView struct {
	SessionReport
	StateLabel       string
	IsCompleted      bool
	HitLabel         string
	Disposition      string
	Concentration    string
	GroupsText       string
	AOL1, Sensory    string
	AOL2, Forms      string
	Dimensions       string
	Positions        string
	Spatial, AOL3    string
	SummaryLines     []string
	RecordConfidence *int
	IdeogramSVG      template.HTML
	SketchSVG        template.HTML
	// TargetImageURL must be template.URL so html/template does not replace
	// data:image/...;base64 URIs with #ZgotmplZ in src attributes.
	TargetImageURL template.URL
}

// RenderSessionHTML builds a print-ready HTML document for one session.
func RenderSessionHTML(rep SessionReport) ([]byte, error) {
	rec, err := session.ParseRecord(rep.RecordJSON)
	if err != nil {
		rec = session.EmptyRecord()
	}
	v := sessionHTMLView{
		SessionReport:    rep,
		StateLabel:       stateLabelPT(rep.State),
		IsCompleted:      rep.State == "completed",
		Disposition:      rec.Disposition,
		Concentration:    rec.Concentration,
		GroupsText:       formatGroups(rec.Groups),
		AOL1:             rec.AOL1,
		Sensory:          rec.Sensory,
		AOL2:             rec.AOL2,
		Forms:            rec.Forms,
		Dimensions:       rec.Dimensions,
		Positions:        rec.Positions,
		Spatial:          rec.Spatial,
		AOL3:             rec.AOL3,
		SummaryLines:     nonEmptySummary(rec.Summary),
		RecordConfidence: rec.Confidence,
		IdeogramSVG:      template.HTML(strokesToSVG(rec.Drawings.Ideogram)),
		SketchSVG:        template.HTML(strokesToSVG(rec.Drawings.Sketch)),
		TargetImageURL:   template.URL(rep.TargetImageDataURI),
	}
	if rep.Hit != nil {
		if *rep.Hit {
			v.HitLabel = "acerto"
		} else {
			v.HitLabel = "erro"
		}
	}
	var buf bytes.Buffer
	if err := sessionTmpl.Execute(&buf, v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func stateLabelPT(state string) string {
	switch state {
	case "collecting":
		return "em coleta"
	case "locked":
		return "bloqueada"
	case "completed":
		return "concluída"
	case "abandoned":
		return "abandonada"
	default:
		return state
	}
}

func nonEmptySummary(lines []string) []string {
	var out []string
	for _, s := range lines {
		s = strings.TrimSpace(s)
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

func formatGroups(groups map[string]session.GroupValue) string {
	if len(groups) == 0 {
		return ""
	}
	keys := make([]string, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		gv := groups[k]
		if len(gv.IDs) == 0 && strings.TrimSpace(gv.Note) == "" {
			continue
		}
		fmt.Fprintf(&b, "%s: %s", k, strings.Join(gv.IDs, ", "))
		if note := strings.TrimSpace(gv.Note); note != "" {
			fmt.Fprintf(&b, " — %s", note)
		}
		b.WriteByte('\n')
	}
	return strings.TrimRight(b.String(), "\n")
}

func strokesToSVG(strokes []session.Stroke) string {
	var b strings.Builder
	// Explicit width/height (not only viewBox + width:100%) so Chromium print/PDF
	// keeps a non-zero layout box for the polylines.
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %.0f %.0f" width="%.0f" height="%.0f" style="width:100%%;height:auto;max-width:1000px" preserveAspectRatio="xMidYMid meet">`,
		session.LogicalW, session.LogicalH, session.LogicalW, session.LogicalH)
	for _, st := range strokes {
		if len(st.Points) == 0 {
			continue
		}
		pts := make([]string, len(st.Points))
		for i, p := range st.Points {
			pts[i] = fmt.Sprintf("%g,%g", p.X, p.Y)
		}
		w := st.Width
		if w <= 0 {
			w = 2
		}
		fmt.Fprintf(&b, `<polyline points="%s" fill="none" stroke="#111" stroke-width="%g" stroke-linecap="round" stroke-linejoin="round"/>`,
			strings.Join(pts, " "), w)
	}
	b.WriteString(`</svg>`)
	return b.String()
}
