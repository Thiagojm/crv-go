package catalog

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Thiagojm/crv-go/internal/store"
)

const (
	maxImageBytes   = 20 << 20 // 20 MiB
	maxPixels       = 40_000_000
	minEligible     = 4
	expectedFormat  = "crv-local-image-inventory-v1"
)

type SourceImage struct {
	LocalPath    string `json:"local_path"`
	Filename     string `json:"filename"`
	SourceURL    string `json:"source_url"`
	OptimizedSrc string `json:"optimized_src"`
	Alt          string `json:"alt"`
}

type SourceRecord struct {
	ID             string          `json:"id"`
	Pool           string          `json:"pool"`
	Index          int             `json:"index"`
	Label          string          `json:"label"`
	TargetSpecific string          `json:"target_specific"`
	Description    string          `json:"description"`
	Credit         string          `json:"credit"`
	Empty          bool            `json:"empty,omitempty"`
	Images         []SourceImage   `json:"images"`
	SAMAttributes  json.RawMessage `json:"sam_attributes"`
}

type InventoryImage struct {
	SHA256  string         `json:"sha256"`
	Paths   []string       `json:"paths"`
	Sources []SourceRecord `json:"sources"`
}

type InventorySummary struct {
	PoolEntries               int `json:"poolEntries"`
	Files                     int `json:"files"`
	UniqueHashes              int `json:"uniqueHashes"`
	DuplicateFileCopies       int `json:"duplicateFileCopies"`
	SingleImageCandidates     int `json:"singleImageCandidates"`
	EntriesWithoutImages      int `json:"entriesWithoutImages"`
	EntriesWithMultipleImages int `json:"entriesWithMultipleImages"`
	UnreferencedImages        int `json:"unreferencedImages"`
}

type Inventory struct {
	Format         string            `json:"format"`
	Note           string            `json:"note"`
	Summary        *InventorySummary `json:"summary"`
	Provenance     json.RawMessage   `json:"provenance"`
	Images         []InventoryImage  `json:"images"`
	ReviewEntries  json.RawMessage   `json:"reviewEntries"`
	SourceMetadata json.RawMessage   `json:"sourceMetadata"`
	SourceManifest json.RawMessage   `json:"sourceManifest"`
}

type Exclusion struct {
	SHA256 string `json:"sha256,omitempty"`
	Path   string `json:"path,omitempty"`
	Reason string `json:"reason"`
}

type Report struct {
	SourceLabel       string            `json:"sourceLabel"`
	InventoryImages   int               `json:"inventoryImages"`
	EligibleCount     int               `json:"eligibleCount"`
	ExcludedCount     int               `json:"excludedCount"`
	MultiImageSources int               `json:"multiImageSources"`
	EmptySources      int               `json:"emptySources"`
	Summary           *InventorySummary `json:"summary,omitempty"`
	Provenance        json.RawMessage   `json:"provenance,omitempty"`
	ReviewEntries     json.RawMessage   `json:"reviewEntries,omitempty"`
	SourceMetadata    json.RawMessage   `json:"sourceMetadata,omitempty"`
	Exclusions        []Exclusion       `json:"exclusions"`
	Ready             bool              `json:"ready"`
	Error             string            `json:"error,omitempty"`
}

type InstallResult struct {
	RevisionID int64
	Report     Report
}

func EnsureInstalled(st *store.Store, catalogDir string) (*InstallResult, error) {
	active, err := st.ActiveCatalog()
	if err != nil {
		return nil, err
	}
	if active != nil && active.EligibleCount >= minEligible {
		if err := verifyRevisionFiles(st, active.RevisionID); err == nil {
			var report Report
			_ = json.Unmarshal([]byte(active.ReportJSON), &report)
			report.Ready = true
			report.EligibleCount = active.EligibleCount
			report.ExcludedCount = active.ExcludedCount
			report.SourceLabel = active.SourceLabel
			fillCountsFromSummary(&report)
			return &InstallResult{RevisionID: active.RevisionID, Report: report}, nil
		}
		// Broken activation (e.g. DB commit without files): drop it and reinstall if possible.
		_ = st.DeactivateRevision(active.RevisionID)
	}
	if catalogDir == "" {
		return &InstallResult{Report: Report{Error: "diretório do banco não configurado", Ready: false}}, errors.New("diretório do banco não configurado")
	}
	if _, err := os.Stat(catalogDir); err != nil {
		if active != nil {
			return &InstallResult{Report: Report{
				Error: "revisão instalada incompleta e banco de distribuição ausente",
				Ready: false,
			}}, errors.New("revisão instalada incompleta e banco de distribuição ausente")
		}
		return &InstallResult{Report: Report{Error: "banco incluído ausente ou ilegível", Ready: false}}, fmt.Errorf("read catalog: %w", err)
	}
	return Install(st, catalogDir)
}

func Install(st *store.Store, catalogDir string) (*InstallResult, error) {
	catalogDir = filepath.Clean(catalogDir)
	invPath := filepath.Join(catalogDir, "catalog-unified.json")
	raw, err := os.ReadFile(invPath)
	if err != nil {
		return &InstallResult{Report: Report{Error: "banco incluído ausente ou ilegível", Ready: false}}, fmt.Errorf("read catalog: %w", err)
	}
	var inv Inventory
	if err := json.Unmarshal(raw, &inv); err != nil {
		return &InstallResult{Report: Report{Error: "catálogo malformado", Ready: false}}, fmt.Errorf("parse catalog: %w", err)
	}
	if inv.Format != expectedFormat {
		msg := "formato de catálogo não suportado"
		if inv.Format != "" {
			msg = msg + ": " + inv.Format
		}
		return &InstallResult{Report: Report{Error: msg, Ready: false}}, errors.New(msg)
	}

	report := Report{
		SourceLabel:     "bundled",
		InventoryImages: len(inv.Images),
		Summary:         inv.Summary,
		Provenance:      inv.Provenance,
		ReviewEntries:   inv.ReviewEntries,
		SourceMetadata:  inv.SourceMetadata,
	}
	fillCountsFromSummary(&report)
	if report.MultiImageSources == 0 && report.EmptySources == 0 {
		report.MultiImageSources, report.EmptySources = countReviewEntryKinds(inv.ReviewEntries)
	}

	seenHash := map[string]int{}
	for _, img := range inv.Images {
		h := strings.ToLower(img.SHA256)
		seenHash[h]++
	}
	for h, n := range seenHash {
		if h != "" && n > 1 {
			msg := "identidades duplicadas no catálogo"
			report.Ready = false
			report.Error = msg
			report.Exclusions = append(report.Exclusions, Exclusion{SHA256: h, Reason: fmt.Sprintf("hash repetido %d vezes", n)})
			return &InstallResult{Report: report}, errors.New(msg)
		}
	}

	stageRoot := filepath.Join(st.DataDir, "catalog-staging")
	_ = os.RemoveAll(stageRoot)
	if err := os.MkdirAll(filepath.Join(stageRoot, "originals"), 0o755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Join(stageRoot, "display"), 0o755); err != nil {
		return nil, err
	}

	var eligible []store.ImageRow
	for _, img := range inv.Images {
		exclusionsBefore := len(report.Exclusions)
		row, ex := validateImage(catalogDir, stageRoot, img)
		if ex != nil {
			report.Exclusions = append(report.Exclusions, *ex)
			continue
		}
		if row == nil {
			if len(report.Exclusions) == exclusionsBefore {
				report.Exclusions = append(report.Exclusions, Exclusion{SHA256: img.SHA256, Reason: "candidato inválido"})
			}
			continue
		}
		eligible = append(eligible, *row)
	}

	report.EligibleCount = len(eligible)
	report.ExcludedCount = len(report.Exclusions)
	if len(eligible) < minEligible {
		report.Ready = false
		report.Error = fmt.Sprintf("são necessárias pelo menos %d imagens elegíveis", minEligible)
		_ = os.RemoveAll(stageRoot)
		return &InstallResult{Report: report}, errors.New(report.Error)
	}

	revPending := filepath.Join(st.DataDir, "catalog", "rev-pending")
	_ = os.RemoveAll(revPending)
	if err := os.MkdirAll(revPending, 0o755); err != nil {
		return nil, err
	}
	if err := copyDir(filepath.Join(stageRoot, "originals"), filepath.Join(revPending, "originals")); err != nil {
		_ = os.RemoveAll(revPending)
		return nil, err
	}
	if err := copyDir(filepath.Join(stageRoot, "display"), filepath.Join(revPending, "display")); err != nil {
		_ = os.RemoveAll(revPending)
		return nil, err
	}
	_ = os.RemoveAll(stageRoot)

	exRows := make([]store.ExclusionRow, 0, len(report.Exclusions))
	for _, e := range report.Exclusions {
		exRows = append(exRows, store.ExclusionRow{SHA256: e.SHA256, Path: e.Path, Reason: e.Reason})
	}

	report.Ready = false
	id, err := st.InsertRevisionInactive(report.SourceLabel, eligible, exRows, report)
	if err != nil {
		_ = os.RemoveAll(revPending)
		return nil, err
	}
	finalDir := store.RevisionDir(st.DataDir, id)
	_ = os.RemoveAll(finalDir)
	if err := os.Rename(revPending, finalDir); err != nil {
		_ = st.DeleteRevision(id)
		_ = os.RemoveAll(revPending)
		_ = os.RemoveAll(finalDir)
		report.Error = "falha ao finalizar arquivos do catálogo"
		return &InstallResult{Report: report}, err
	}
	if err := verifyRevisionFiles(st, id); err != nil {
		_ = st.DeleteRevision(id)
		_ = os.RemoveAll(finalDir)
		report.Error = "revisão instalada incompleta"
		return &InstallResult{Report: report}, err
	}

	report.Ready = true
	if err := st.MarkRevisionActive(id, report); err != nil {
		_ = st.DeleteRevision(id)
		_ = os.RemoveAll(finalDir)
		report.Ready = false
		report.Error = "falha ao ativar catálogo"
		return &InstallResult{Report: report}, err
	}
	return &InstallResult{RevisionID: id, Report: report}, nil
}

func fillCountsFromSummary(report *Report) {
	if report.Summary == nil {
		return
	}
	report.MultiImageSources = report.Summary.EntriesWithMultipleImages
	report.EmptySources = report.Summary.EntriesWithoutImages
}

func countReviewEntryKinds(raw json.RawMessage) (multi, empty int) {
	if len(raw) == 0 {
		return 0, 0
	}
	var entries []struct {
		Images []json.RawMessage `json:"images"`
	}
	if err := json.Unmarshal(raw, &entries); err != nil {
		return 0, 0
	}
	for _, e := range entries {
		switch len(e.Images) {
		case 0:
			empty++
		case 1:
		default:
			multi++
		}
	}
	return multi, empty
}

func verifyRevisionFiles(st *store.Store, revisionID int64) error {
	images, err := st.RevisionImages(revisionID)
	if err != nil {
		return err
	}
	if len(images) < minEligible {
		return fmt.Errorf("revisão com poucas imagens: %d", len(images))
	}
	root := store.RevisionDir(st.DataDir, revisionID)
	for _, img := range images {
		for _, rel := range []string{img.OriginalRelpath, img.DisplayRelpath} {
			if rel == "" {
				return errors.New("caminho de imagem ausente na revisão")
			}
			p := filepath.Join(root, filepath.FromSlash(rel))
			info, err := os.Lstat(p)
			if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Size() <= 0 {
				return fmt.Errorf("arquivo de revisão ausente ou inválido: %s", rel)
			}
		}
	}
	return nil
}

func validateImage(catalogDir, stageRoot string, img InventoryImage) (row *store.ImageRow, ex *Exclusion) {
	if !isHexSHA256(img.SHA256) {
		return nil, &Exclusion{SHA256: img.SHA256, Reason: "hash inválido"}
	}
	var singleSources []SourceRecord
	var multi, empty int
	for _, src := range img.Sources {
		switch len(src.Images) {
		case 0:
			empty++
		case 1:
			singleSources = append(singleSources, src)
		default:
			multi++
		}
	}
	if len(singleSources) == 0 {
		reason := "sem registro de fonte com uma única imagem"
		if multi > 0 {
			reason = "apenas fontes multi-imagem"
		} else if empty > 0 {
			reason = "apenas fontes sem imagem"
		}
		return nil, &Exclusion{SHA256: img.SHA256, Reason: reason}
	}

	rel, err := pickPath(img, singleSources)
	if err != nil {
		return nil, &Exclusion{SHA256: img.SHA256, Reason: err.Error()}
	}
	if err := assertSafeRelPath(rel); err != nil {
		return nil, &Exclusion{SHA256: img.SHA256, Path: rel, Reason: err.Error()}
	}
	abs := filepath.Join(catalogDir, filepath.FromSlash(rel))
	if err := assertNoSymlinkAncestors(catalogDir, abs); err != nil {
		return nil, &Exclusion{SHA256: img.SHA256, Path: rel, Reason: err.Error()}
	}
	info, err := os.Lstat(abs)
	if err != nil {
		return nil, &Exclusion{SHA256: img.SHA256, Path: rel, Reason: "arquivo ausente"}
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, &Exclusion{SHA256: img.SHA256, Path: rel, Reason: "atalho/link rejeitado"}
	}
	if !info.Mode().IsRegular() {
		return nil, &Exclusion{SHA256: img.SHA256, Path: rel, Reason: "não é arquivo regular"}
	}
	if info.Size() > maxImageBytes {
		return nil, &Exclusion{SHA256: img.SHA256, Path: rel, Reason: "arquivo excede 20 MiB"}
	}
	if info.Size() <= 0 {
		return nil, &Exclusion{SHA256: img.SHA256, Path: rel, Reason: "arquivo vazio"}
	}

	f, err := os.Open(abs)
	if err != nil {
		return nil, &Exclusion{SHA256: img.SHA256, Path: rel, Reason: "falha ao abrir"}
	}
	defer f.Close()
	h := sha256.New()
	limited := io.LimitReader(f, maxImageBytes+1)
	n, err := io.Copy(h, limited)
	if err != nil {
		return nil, &Exclusion{SHA256: img.SHA256, Path: rel, Reason: "falha ao ler"}
	}
	if n > maxImageBytes {
		return nil, &Exclusion{SHA256: img.SHA256, Path: rel, Reason: "arquivo excede 20 MiB"}
	}
	sum := hex.EncodeToString(h.Sum(nil))
	if !strings.EqualFold(sum, img.SHA256) {
		return nil, &Exclusion{SHA256: img.SHA256, Path: rel, Reason: "hash divergente"}
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return nil, &Exclusion{SHA256: img.SHA256, Path: rel, Reason: "falha ao reler"}
	}
	cfg, format, err := image.DecodeConfig(io.LimitReader(f, maxImageBytes))
	if err != nil {
		return nil, &Exclusion{SHA256: img.SHA256, Path: rel, Reason: "decodificação falhou"}
	}
	if format != "jpeg" && format != "gif" {
		return nil, &Exclusion{SHA256: img.SHA256, Path: rel, Reason: "formato não suportado: " + format}
	}
	if cfg.Width <= 0 || cfg.Height <= 0 || int64(cfg.Width)*int64(cfg.Height) > maxPixels {
		return nil, &Exclusion{SHA256: img.SHA256, Path: rel, Reason: "dimensões inválidas ou acima de 40 MP"}
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return nil, &Exclusion{SHA256: img.SHA256, Path: rel, Reason: "falha ao reler"}
	}
	decoded, _, err := image.Decode(io.LimitReader(f, maxImageBytes))
	if err != nil {
		return nil, &Exclusion{SHA256: img.SHA256, Path: rel, Reason: "decodificação da imagem falhou"}
	}

	origName := strings.ToLower(img.SHA256) + filepath.Ext(rel)
	dispName := strings.ToLower(img.SHA256) + ".png"
	origDst := filepath.Join(stageRoot, "originals", origName)
	dispDst := filepath.Join(stageRoot, "display", dispName)
	if err := copyFile(abs, origDst); err != nil {
		return nil, &Exclusion{SHA256: img.SHA256, Path: rel, Reason: "falha ao copiar original"}
	}
	if err := writePNG(dispDst, decoded); err != nil {
		return nil, &Exclusion{SHA256: img.SHA256, Path: rel, Reason: "falha ao gerar display PNG"}
	}

	primary := choosePrimary(singleSources)
	sourcesJSON, _ := json.Marshal(img.Sources)
	return &store.ImageRow{
		SHA256:          strings.ToLower(img.SHA256),
		OriginalRelpath: filepath.ToSlash(filepath.Join("originals", origName)),
		DisplayRelpath:  filepath.ToSlash(filepath.Join("display", dispName)),
		Width:           cfg.Width,
		Height:          cfg.Height,
		Bytes:           info.Size(),
		PrimarySourceID: primary.ID,
		SourcesJSON:     string(sourcesJSON),
	}, nil
}

func pickPath(img InventoryImage, singles []SourceRecord) (string, error) {
	candidates := map[string]bool{}
	for _, p := range img.Paths {
		candidates[filepath.ToSlash(p)] = true
	}
	for _, s := range singles {
		for _, im := range s.Images {
			if im.LocalPath != "" {
				candidates[filepath.ToSlash(im.LocalPath)] = true
			}
		}
	}
	if len(candidates) == 0 {
		return "", errors.New("caminho local ausente")
	}
	list := make([]string, 0, len(candidates))
	for p := range candidates {
		list = append(list, p)
	}
	sort.Strings(list)
	return list[0], nil
}

func choosePrimary(sources []SourceRecord) SourceRecord {
	order := map[string]int{"A": 0, "B": 1, "C": 2}
	best := sources[0]
	for _, s := range sources[1:] {
		bp, okB := order[best.Pool]
		sp, okS := order[s.Pool]
		if !okB {
			bp = 99
		}
		if !okS {
			sp = 99
		}
		if sp < bp || (sp == bp && s.ID < best.ID) {
			best = s
		}
	}
	return best
}

func assertSafeRelPath(rel string) error {
	rel = filepath.ToSlash(rel)
	if rel == "" || strings.HasPrefix(rel, "/") || strings.Contains(rel, ":") {
		return errors.New("caminho absoluto rejeitado")
	}
	clean := filepath.ToSlash(filepath.Clean(rel))
	if clean == ".." || strings.HasPrefix(clean, "../") || strings.Contains(clean, "/../") {
		return errors.New("caminho com travessia rejeitado")
	}
	if strings.Contains(rel, "\\") {
		return errors.New("separador inválido")
	}
	return nil
}

// assertNoSymlinkAncestors rejects symlinks on the file or any directory from catalog root to it.
func assertNoSymlinkAncestors(catalogDir, absFile string) error {
	catalogDir = filepath.Clean(catalogDir)
	absFile = filepath.Clean(absFile)
	rel, err := filepath.Rel(catalogDir, absFile)
	if err != nil || strings.HasPrefix(rel, "..") {
		return errors.New("caminho fora do catálogo")
	}
	cur := catalogDir
	parts := strings.Split(rel, string(os.PathSeparator))
	for i, part := range parts {
		if part == "" || part == "." {
			continue
		}
		cur = filepath.Join(cur, part)
		info, err := os.Lstat(cur)
		if err != nil {
			if i == len(parts)-1 {
				return nil // final missing handled by caller
			}
			return errors.New("caminho intermediário ausente")
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return errors.New("atalho/link em ancestral rejeitado")
		}
	}
	return nil
}

func isHexSHA256(s string) bool {
	if len(s) != 64 {
		return false
	}
	_, err := hex.DecodeString(s)
	return err == nil
}

func writePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := png.Encoder{CompressionLevel: png.BestSpeed}
	return enc.Encode(f, img)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func copyDir(src, dst string) error {
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if err := copyFile(filepath.Join(src, e.Name()), filepath.Join(dst, e.Name())); err != nil {
			return err
		}
	}
	return nil
}
