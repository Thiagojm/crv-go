package backup

// Startup recovery (call before store.Open):
//
//	if err := backup.ResolveInterrupted(dataDir); err != nil { ... }

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Thiagojm/crv-go/internal/catalog"

	_ "modernc.org/sqlite"
)

// Marker stages for durable restore swap recovery.
const (
	StageValidated  = "validated"
	StagePreBackup  = "prebackup"
	StageLiveMoved  = "live_moved"
	StageNewInPlace = "new_inplace"
	StageCommitted  = "committed"
)

// Marker is the durable restore-control file kept outside swapped payload.
type Marker struct {
	Stage        string   `json:"stage"`
	PreBackup    string   `json:"preBackup"`
	Staging      string   `json:"staging"`
	OldDir       string   `json:"oldDir"`
	OldFiles     []string `json:"oldFiles,omitempty"`
	BackupFormat string   `json:"backupFormat"`
}

// Live payload names swapped during restore (relative to dataDir).
var livePayloadFiles = []string{
	"crv.sqlite",
	"crv.sqlite-wal",
	"crv.sqlite-shm",
}

const liveCatalogDir = "catalog"

// MarkerPath returns dataDir/restore-marker.json.
func MarkerPath(dataDir string) string {
	return filepath.Join(dataDir, "restore-marker.json")
}

// DefaultStagingDir returns dataDir/restore-staging.
func DefaultStagingDir(dataDir string) string {
	return filepath.Join(dataDir, "restore-staging")
}

// LivePayloadExists reports whether any swappable live payload is present under dataDir.
func LivePayloadExists(dataDir string) bool {
	for _, name := range livePayloadFiles {
		if _, err := os.Stat(filepath.Join(dataDir, name)); err == nil {
			return true
		}
	}
	if st, err := os.Stat(filepath.Join(dataDir, liveCatalogDir)); err == nil && st.IsDir() {
		return true
	}
	return false
}

// ValidateArchive extracts zipPath into stagingDir (emptied first), verifies
// format/hashes/paths and PRAGMA integrity_check on the staged database.
func ValidateArchive(zipPath, stagingDir string) error {
	zipPath = filepath.Clean(zipPath)
	stagingDir = filepath.Clean(stagingDir)
	if zipPath == "" || stagingDir == "" {
		return errors.New("caminho de restauração inválido")
	}
	info, err := os.Stat(zipPath)
	if err != nil {
		return errors.New("arquivo de backup ausente ou ilegível")
	}
	if !info.Mode().IsRegular() {
		return errors.New("arquivo de backup inválido")
	}
	if info.Size() > maxArchiveBytes {
		return errors.New("arquivo ZIP excede o tamanho máximo permitido")
	}

	_ = os.RemoveAll(stagingDir)
	if err := os.MkdirAll(stagingDir, 0o755); err != nil {
		return err
	}

	f, err := os.Open(zipPath)
	if err != nil {
		return errors.New("arquivo de backup ausente ou ilegível")
	}
	defer f.Close()

	if err := catalog.ExtractZip(f, stagingDir); err != nil {
		_ = os.RemoveAll(stagingDir)
		return err
	}

	man, err := readAndVerifyManifest(stagingDir)
	if err != nil {
		_ = os.RemoveAll(stagingDir)
		return err
	}

	dbPath := filepath.Join(stagingDir, "crv.sqlite")
	if err := integrityCheckDB(dbPath); err != nil {
		_ = os.RemoveAll(stagingDir)
		return err
	}

	catDir := filepath.Join(stagingDir, liveCatalogDir)
	if st, err := os.Stat(catDir); err != nil || !st.IsDir() {
		_ = os.RemoveAll(stagingDir)
		return errors.New("catálogo ausente no backup")
	}

	required, err := stagedRequiredRevisionIDs(dbPath)
	if err != nil {
		_ = os.RemoveAll(stagingDir)
		return err
	}
	manifestSet := make(map[int64]struct{}, len(man.RevisionIDs))
	for _, id := range man.RevisionIDs {
		manifestSet[id] = struct{}{}
	}
	for _, id := range required {
		if _, ok := manifestSet[id]; !ok {
			_ = os.RemoveAll(stagingDir)
			return fmt.Errorf("revisão referenciada ausente do manifesto: rev-%d", id)
		}
	}
	// Verify every revision the staged DB needs (active + session-referenced).
	for _, id := range required {
		rev := filepath.Join(catDir, fmt.Sprintf("rev-%d", id))
		if st, err := os.Stat(rev); err != nil || !st.IsDir() {
			_ = os.RemoveAll(stagingDir)
			return fmt.Errorf("revisão ausente no backup: rev-%d", id)
		}
	}
	if err := verifyStagedCatalogFiles(stagingDir, required); err != nil {
		_ = os.RemoveAll(stagingDir)
		return err
	}
	return nil
}

// PrepareSwap records a validated→prebackup marker after confirming preBackupZip exists.
// stagingDir must already contain a ValidateArchive result.
func PrepareSwap(dataDir, stagingDir, preBackupZip string) error {
	dataDir = filepath.Clean(dataDir)
	stagingDir = filepath.Clean(stagingDir)
	preBackupZip = filepath.Clean(preBackupZip)
	if dataDir == "" || stagingDir == "" || preBackupZip == "" {
		return errors.New("parâmetros de restauração inválidos")
	}
	if _, err := os.Stat(preBackupZip); err != nil {
		return errors.New("backup pré-restauração ausente ou ilegível")
	}
	if _, err := os.Stat(filepath.Join(stagingDir, "crv.sqlite")); err != nil {
		return errors.New("staging de restauração incompleto")
	}
	if _, err := os.Stat(filepath.Join(stagingDir, "manifest.json")); err != nil {
		return errors.New("staging de restauração incompleto")
	}

	man, err := readManifestFile(filepath.Join(stagingDir, "manifest.json"))
	if err != nil {
		return err
	}

	m := Marker{
		Stage:        StageValidated,
		PreBackup:    preBackupZip,
		Staging:      stagingDir,
		BackupFormat: man.Format,
	}
	if err := writeMarker(dataDir, m); err != nil {
		return err
	}
	m.Stage = StagePreBackup
	return writeMarker(dataDir, m)
}

// ApplySwap closes DB handles via stCloser, moves live payload aside, then installs
// staged crv.sqlite and catalog/ into dataDir. Caller must reopen the store afterward.
func ApplySwap(dataDir string, stCloser func() error) error {
	dataDir = filepath.Clean(dataDir)
	m, err := readMarker(dataDir)
	if err != nil {
		return err
	}
	if m.Stage != StagePreBackup && m.Stage != StageValidated {
		return fmt.Errorf("estado de restauração inesperado: %s", m.Stage)
	}
	if m.Staging == "" {
		return errors.New("staging de restauração ausente no marcador")
	}
	if _, err := os.Stat(filepath.Join(m.Staging, "crv.sqlite")); err != nil {
		return errors.New("staging de restauração incompleto")
	}
	if st, err := os.Stat(filepath.Join(m.Staging, liveCatalogDir)); err != nil || !st.IsDir() {
		return errors.New("staging de restauração incompleto")
	}

	if stCloser != nil {
		if err := stCloser(); err != nil {
			return fmt.Errorf("falha ao fechar banco antes da troca: %w", err)
		}
	}

	oldDir := filepath.Join(dataDir, "restore-old-"+time.Now().UTC().Format("20060102T150405Z"))
	if err := os.MkdirAll(oldDir, 0o755); err != nil {
		return err
	}
	m.OldFiles = nil
	for _, name := range append(livePayloadFiles, liveCatalogDir) {
		if _, err := os.Lstat(filepath.Join(dataDir, name)); err == nil {
			m.OldFiles = append(m.OldFiles, name)
		} else if !errors.Is(err, os.ErrNotExist) {
			_ = os.RemoveAll(oldDir)
			return err
		}
	}
	// Persist live_moved + OldDir before any live payload move so an interruption
	// cannot leave the old database aside while ResolveInterrupted still treats
	// the marker as a pre-swap abort.
	m.OldDir = oldDir
	m.Stage = StageLiveMoved
	if err := writeMarker(dataDir, *m); err != nil {
		_ = os.RemoveAll(oldDir)
		return err
	}
	if err := moveLivePayloadFn(dataDir, oldDir); err != nil {
		if undoErr := moveLivePayloadFn(oldDir, dataDir); undoErr != nil {
			// Keep live_moved + OldDir so ResolveInterrupted can recover.
			return fmt.Errorf("falha ao mover dados vivos e ao desfazer (%v): %w", undoErr, err)
		}
		_ = os.RemoveAll(oldDir)
		m.Stage = StagePreBackup
		m.OldDir = ""
		m.OldFiles = nil
		_ = writeMarker(dataDir, *m)
		return err
	}

	if err := moveFile(filepath.Join(m.Staging, "crv.sqlite"), filepath.Join(dataDir, "crv.sqlite")); err != nil {
		return err
	}
	if err := moveFile(filepath.Join(m.Staging, liveCatalogDir), filepath.Join(dataDir, liveCatalogDir)); err != nil {
		return err
	}
	m.Stage = StageNewInPlace
	return writeMarker(dataDir, *m)
}

// CommitSwap removes OldDir and staging, marks committed, then deletes the marker.
func CommitSwap(dataDir string) error {
	dataDir = filepath.Clean(dataDir)
	m, err := readMarker(dataDir)
	if err != nil {
		return err
	}
	if m.OldDir != "" {
		_ = os.RemoveAll(m.OldDir)
	}
	if m.Staging != "" {
		_ = os.RemoveAll(m.Staging)
	}
	m.Stage = StageCommitted
	if err := writeMarker(dataDir, *m); err != nil {
		return err
	}
	return os.Remove(MarkerPath(dataDir))
}

// RollbackSwap restores OldDir payload when a swap was interrupted or new data is bad.
func RollbackSwap(dataDir string) error {
	dataDir = filepath.Clean(dataDir)
	m, err := readMarker(dataDir)
	if err != nil {
		return err
	}
	switch m.Stage {
	case StageLiveMoved, StageNewInPlace:
		// continue
	default:
		return fmt.Errorf("rollback indisponível no estado %s", m.Stage)
	}
	if m.OldDir == "" {
		return errors.New("cópia anterior ausente para rollback")
	}
	if _, err := os.Stat(m.OldDir); err != nil {
		return errors.New("cópia anterior ausente para rollback")
	}

	oldFiles := make(map[string]bool, len(m.OldFiles))
	for _, name := range m.OldFiles {
		oldFiles[name] = true
	}
	// Older markers lack OldFiles. Preserve live entries absent from OldDir.
	if m.OldFiles == nil {
		for _, name := range append(livePayloadFiles, liveCatalogDir) {
			for _, root := range []string{m.OldDir, dataDir} {
				if _, err := os.Lstat(filepath.Join(root, name)); err == nil {
					oldFiles[name] = true
				} else if !errors.Is(err, os.ErrNotExist) {
					return err
				}
			}
		}
	}
	removeLive := func(name string) error {
		p := filepath.Join(dataDir, name)
		if name == liveCatalogDir {
			return os.RemoveAll(p)
		}
		err := os.Remove(p)
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	for _, name := range append(livePayloadFiles, liveCatalogDir) {
		oldPath := filepath.Join(m.OldDir, name)
		livePath := filepath.Join(dataDir, name)
		if _, err := os.Lstat(oldPath); err == nil {
			if err := removeLive(name); err != nil {
				return fmt.Errorf("falha ao remover dados parciais: %w", err)
			}
			if err := moveFile(oldPath, livePath); err != nil {
				return fmt.Errorf("falha ao restaurar dados anteriores: %w", err)
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		} else if !oldFiles[name] {
			if err := removeLive(name); err != nil {
				return fmt.Errorf("falha ao remover dados novos: %w", err)
			}
		} else if _, err := os.Lstat(livePath); err != nil {
			return fmt.Errorf("dado anterior ausente (%s): %w", name, err)
		}
	}
	_ = os.RemoveAll(m.OldDir)
	if m.Staging != "" {
		_ = os.RemoveAll(m.Staging)
	}
	return os.Remove(MarkerPath(dataDir))
}

// ResolveInterrupted recovers an interrupted restore before store.Open.
// validated/prebackup: discard staging and marker (live untouched).
// live_moved/new_inplace: commit if new payload is intact, otherwise roll back.
func ResolveInterrupted(dataDir string) error {
	dataDir = filepath.Clean(dataDir)
	m, err := readMarker(dataDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	switch m.Stage {
	case StageValidated, StagePreBackup:
		if m.Staging != "" {
			_ = os.RemoveAll(m.Staging)
		}
		_ = os.Remove(MarkerPath(dataDir))
		return nil
	case StageCommitted:
		_ = os.Remove(MarkerPath(dataDir))
		return nil
	case StageLiveMoved, StageNewInPlace:
		dbPath := filepath.Join(dataDir, "crv.sqlite")
		catOK := false
		if st, err := os.Stat(filepath.Join(dataDir, liveCatalogDir)); err == nil && st.IsDir() {
			catOK = true
		}
		if catOK && integrityCheckDB(dbPath) == nil {
			return CommitSwap(dataDir)
		}
		return RollbackSwap(dataDir)
	default:
		// Unknown stage: prefer rollback if OldDir exists, else clear marker carefully.
		if m.OldDir != "" {
			if _, err := os.Stat(m.OldDir); err == nil {
				return RollbackSwap(dataDir)
			}
		}
		if m.Staging != "" {
			_ = os.RemoveAll(m.Staging)
		}
		_ = os.Remove(MarkerPath(dataDir))
		return fmt.Errorf("marcador de restauração em estado desconhecido: %s", m.Stage)
	}
}

func readAndVerifyManifest(stagingDir string) (*Manifest, error) {
	manPath := filepath.Join(stagingDir, "manifest.json")
	man, err := readManifestFile(manPath)
	if err != nil {
		return nil, err
	}
	if man.Format != FormatVersion {
		return nil, errors.New("formato de backup não suportado")
	}
	if man.Files == nil {
		return nil, errors.New("manifesto de backup inválido")
	}
	if _, ok := man.Files["crv.sqlite"]; !ok {
		return nil, errors.New("manifesto de backup sem crv.sqlite")
	}
	for name, want := range man.Files {
		if err := assertSafeManifestPath(name); err != nil {
			return nil, err
		}
		p := filepath.Join(stagingDir, filepath.FromSlash(name))
		if err := assertUnderRoot(stagingDir, p); err != nil {
			return nil, err
		}
		info, err := os.Lstat(p)
		if err != nil {
			return nil, fmt.Errorf("arquivo do manifesto ausente: %s", name)
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return nil, fmt.Errorf("entrada especial no backup: %s", name)
		}
		got, err := sha256File(p)
		if err != nil {
			return nil, err
		}
		if !strings.EqualFold(got, want) {
			return nil, fmt.Errorf("hash divergente no backup: %s", name)
		}
	}
	return man, nil
}

func readManifestFile(path string) (*Manifest, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.New("manifesto de backup ausente ou ilegível")
	}
	var man Manifest
	if err := json.Unmarshal(raw, &man); err != nil {
		return nil, errors.New("manifesto de backup inválido")
	}
	return &man, nil
}

func assertSafeManifestPath(name string) error {
	name = strings.ReplaceAll(name, "\\", "/")
	if name == "" {
		return errors.New("caminho inseguro no manifesto")
	}
	if strings.HasPrefix(name, "/") || strings.HasPrefix(name, "//") {
		return errors.New("caminho absoluto rejeitado no manifesto")
	}
	if len(name) >= 2 && name[1] == ':' {
		return errors.New("caminho absoluto rejeitado no manifesto")
	}
	if strings.Contains(name, ":") {
		return errors.New("caminho absoluto rejeitado no manifesto")
	}
	clean := path.Clean(name)
	if clean == "." || clean == "" || clean == ".." || strings.HasPrefix(clean, "../") {
		return errors.New("caminho com travessia rejeitado no manifesto")
	}
	return nil
}

func assertUnderRoot(root, target string) error {
	root = filepath.Clean(root)
	target = filepath.Clean(target)
	rel, err := filepath.Rel(root, target)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return errors.New("caminho com travessia rejeitado no manifesto")
	}
	return nil
}

// verifyStagedCatalogFiles ensures every original/display path referenced by the
// staged database for the listed revisions exists as a regular non-empty file.
func verifyStagedCatalogFiles(stagingDir string, revIDs []int64) error {
	dbPath := filepath.Join(stagingDir, "crv.sqlite")
	db, err := sql.Open("sqlite", dbPath+"?mode=ro&_pragma=busy_timeout(5000)")
	if err != nil {
		return errors.New("não foi possível abrir o banco do backup")
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	for _, id := range revIDs {
		rows, err := db.Query(`SELECT original_relpath, display_relpath FROM catalog_images WHERE revision_id = ?`, id)
		if err != nil {
			return fmt.Errorf("falha ao ler imagens da revisão rev-%d no backup", id)
		}
		root := filepath.Join(stagingDir, liveCatalogDir, fmt.Sprintf("rev-%d", id))
		var count int
		for rows.Next() {
			var orig, disp string
			if err := rows.Scan(&orig, &disp); err != nil {
				_ = rows.Close()
				return err
			}
			count++
			for _, rel := range []string{orig, disp} {
				if rel == "" {
					_ = rows.Close()
					return fmt.Errorf("caminho de imagem ausente no backup (rev-%d)", id)
				}
				p := filepath.Join(root, filepath.FromSlash(rel))
				info, err := os.Lstat(p)
				if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Size() <= 0 {
					_ = rows.Close()
					return fmt.Errorf("arquivo de catálogo ausente no backup (rev-%d/%s)", id, rel)
				}
			}
		}
		err = rows.Err()
		_ = rows.Close()
		if err != nil {
			return err
		}
		if count == 0 {
			return fmt.Errorf("revisão sem imagens no backup: rev-%d", id)
		}
	}
	return nil
}

func integrityCheckDB(dbPath string) error {
	info, err := os.Stat(dbPath)
	if err != nil || !info.Mode().IsRegular() {
		return errors.New("banco do backup ausente ou inválido")
	}
	db, err := sql.Open("sqlite", dbPath+"?_pragma=busy_timeout(5000)")
	if err != nil {
		return errors.New("não foi possível abrir o banco do backup")
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	var result string
	if err := db.QueryRow(`PRAGMA integrity_check`).Scan(&result); err != nil {
		return errors.New("falha na verificação de integridade do banco")
	}
	if result != "ok" {
		return errors.New("banco do backup falhou na verificação de integridade")
	}
	return nil
}

func writeMarker(dataDir string, m Marker) error {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	path := MarkerPath(dataDir)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func readMarker(dataDir string) (*Marker, error) {
	raw, err := os.ReadFile(MarkerPath(dataDir))
	if err != nil {
		return nil, err
	}
	var m Marker
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, errors.New("marcador de restauração inválido")
	}
	return &m, nil
}

// moveLivePayloadFn is the live-payload mover; tests may replace it.
var moveLivePayloadFn = moveLivePayload

// stagedRequiredRevisionIDs returns active and session-referenced catalog
// revision IDs from a staged (read-only) SQLite path.
func stagedRequiredRevisionIDs(dbPath string) ([]int64, error) {
	db, err := sql.Open("sqlite", dbPath+"?mode=ro&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, errors.New("não foi possível abrir o banco do backup")
	}
	defer db.Close()
	db.SetMaxOpenConns(1)

	set := make(map[int64]struct{})
	var active sql.NullInt64
	err = db.QueryRow(`SELECT id FROM catalog_revisions WHERE active = 1 LIMIT 1`).Scan(&active)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, errors.New("falha ao ler revisão ativa do backup")
	}
	if active.Valid && active.Int64 > 0 {
		set[active.Int64] = struct{}{}
	}
	rows, err := db.Query(`SELECT DISTINCT catalog_revision_id FROM sessions WHERE catalog_revision_id IS NOT NULL`)
	if err != nil {
		return nil, errors.New("falha ao ler revisões referenciadas pelo backup")
	}
	defer rows.Close()
	for rows.Next() {
		var id sql.NullInt64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		if id.Valid && id.Int64 > 0 {
			set[id.Int64] = struct{}{}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(set) == 0 {
		return nil, errors.New("backup sem revisão de catálogo ativa ou referenciada")
	}
	ids := make([]int64, 0, len(set))
	for id := range set {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids, nil
}

func moveLivePayload(fromDir, toDir string) error {
	if err := os.MkdirAll(toDir, 0o755); err != nil {
		return err
	}
	for _, name := range livePayloadFiles {
		src := filepath.Join(fromDir, name)
		if _, err := os.Lstat(src); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return err
		}
		dst := filepath.Join(toDir, name)
		if err := moveFile(src, dst); err != nil {
			return err
		}
	}
	srcCat := filepath.Join(fromDir, liveCatalogDir)
	if st, err := os.Lstat(srcCat); err == nil && st.IsDir() {
		dstCat := filepath.Join(toDir, liveCatalogDir)
		if err := moveFile(srcCat, dstCat); err != nil {
			return err
		}
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func moveFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	// Cross-device fallback: copy then remove.
	info, err := os.Lstat(src)
	if err != nil {
		return err
	}
	if info.IsDir() {
		if err := copyDir(src, dst); err != nil {
			return err
		}
		return os.RemoveAll(src)
	}
	if err := copyFile(src, dst); err != nil {
		return err
	}
	return os.Remove(src)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return errors.New("atalho/link rejeitado na cópia")
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return copyFile(p, target)
	})
}
