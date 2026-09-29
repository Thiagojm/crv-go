package backup

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
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

	"github.com/Thiagojm/crv-go/internal/store"
)

// FormatVersion is the only backup archive format this package produces and accepts.
const FormatVersion = "crv-backup-v1"

const (
	maxArchiveBytes    = 1 << 30 // 1 GiB compressed
	maxZipUncompressed = 1 << 30 // 1 GiB uncompressed total
	maxZipFiles        = 10_000
	maxZipFileBytes    = 20 << 20 // 20 MiB per file

	manifestNote = "Contém atribuições ocultas de sessões."
)

// Manifest is the versioned inventory written as manifest.json inside a backup ZIP.
type Manifest struct {
	Format      string            `json:"format"`
	CreatedAt   string            `json:"createdAt"`
	Files       map[string]string `json:"files"`
	RevisionIDs []int64           `json:"revisionIds"`
	Note        string            `json:"note"`
}

// SnapshotDB writes a consistent SQLite snapshot of st into destFile via VACUUM INTO.
// destFile must be an absolute path; its parent directory is created if needed.
// The destination must not already exist. Safe on the live connection with MaxOpenConns(1).
func SnapshotDB(st *store.Store, destFile string) error {
	if st == nil || st.DB == nil {
		return errors.New("armazenamento indisponível para snapshot")
	}
	destFile = filepath.Clean(destFile)
	if !filepath.IsAbs(destFile) {
		abs, err := filepath.Abs(destFile)
		if err != nil {
			return fmt.Errorf("caminho do snapshot: %w", err)
		}
		destFile = abs
	}
	if err := os.MkdirAll(filepath.Dir(destFile), 0o755); err != nil {
		return err
	}
	if _, err := os.Stat(destFile); err == nil {
		return errors.New("destino do snapshot já existe")
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	// VACUUM INTO requires a quoted literal path (not a bound parameter).
	escaped := strings.ReplaceAll(destFile, "'", "''")
	if _, err := st.DB.Exec("VACUUM INTO '" + escaped + "'"); err != nil {
		return fmt.Errorf("falha ao criar snapshot do banco: %w", err)
	}
	return nil
}

// Create writes a crv-backup-v1 ZIP to outPath. It builds outPath+".partial" first and
// renames only after size/hash checks succeed, leaving existing data intact on failure.
func Create(st *store.Store, outPath string) error {
	if st == nil || st.DB == nil {
		return errors.New("armazenamento indisponível para backup")
	}
	outPath = filepath.Clean(outPath)
	if outPath == "" || outPath == "." {
		return errors.New("caminho de backup inválido")
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}

	partial := outPath + ".partial"
	_ = os.Remove(partial)

	tmpRoot, err := os.MkdirTemp(filepath.Dir(outPath), "crv-backup-build-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(tmpRoot) }()

	snapPath := filepath.Join(tmpRoot, "crv.sqlite")
	if err := SnapshotDB(st, snapPath); err != nil {
		return err
	}

	revIDs, err := referencedRevisionIDs(st)
	if err != nil {
		return err
	}

	prefs, err := dumpPreferences(st)
	if err != nil {
		return err
	}
	prefsPath := filepath.Join(tmpRoot, "preferences.json")
	prefsRaw, err := json.Marshal(prefs)
	if err != nil {
		return err
	}
	if err := os.WriteFile(prefsPath, prefsRaw, 0o644); err != nil {
		return err
	}

	zf, err := os.OpenFile(partial, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	zw := zip.NewWriter(zf)

	files := make(map[string]string)
	var uncompressed uint64
	var fileCount int

	addFile := func(zipName, srcPath string) error {
		zipName = path.Clean(strings.ReplaceAll(zipName, "\\", "/"))
		if zipName == "." || zipName == "" || strings.HasPrefix(zipName, "../") || zipName == ".." {
			return errors.New("caminho inseguro no backup")
		}
		info, err := os.Lstat(srcPath)
		if err != nil {
			return fmt.Errorf("arquivo ausente no backup (%s): %w", zipName, err)
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return fmt.Errorf("entrada especial rejeitada no backup: %s", zipName)
		}
		if info.Size() > maxZipFileBytes {
			return errors.New("backup excede o limite de 20 MiB por arquivo")
		}
		fileCount++
		if fileCount > maxZipFiles {
			return errors.New("backup excede o número máximo de arquivos")
		}
		uncompressed += uint64(info.Size())
		if uncompressed > maxZipUncompressed {
			return errors.New("backup excede o tamanho descomprimido máximo de 1 GiB")
		}
		sum, err := sha256File(srcPath)
		if err != nil {
			return err
		}
		if err := writeZipFileFromPath(zw, zipName, srcPath); err != nil {
			return err
		}
		files[zipName] = sum
		return nil
	}

	finalizeFail := func(err error) error {
		_ = zw.Close()
		_ = zf.Close()
		_ = os.Remove(partial)
		return err
	}

	if err := addFile("crv.sqlite", snapPath); err != nil {
		return finalizeFail(err)
	}
	if err := addFile("preferences.json", prefsPath); err != nil {
		return finalizeFail(err)
	}

	for _, id := range revIDs {
		if err := verifyLiveRevisionFiles(st, id); err != nil {
			return finalizeFail(err)
		}
		revDir := store.RevisionDir(st.DataDir, id)
		prefix := fmt.Sprintf("catalog/rev-%d", id)
		err = filepath.Walk(revDir, func(p string, fi os.FileInfo, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if fi.IsDir() {
				return nil
			}
			if fi.Mode()&os.ModeSymlink != 0 || !fi.Mode().IsRegular() {
				return fmt.Errorf("entrada especial rejeitada no catálogo: %s", p)
			}
			rel, err := filepath.Rel(revDir, p)
			if err != nil {
				return err
			}
			zipName := path.Join(prefix, filepath.ToSlash(rel))
			return addFile(zipName, p)
		})
		if err != nil {
			return finalizeFail(err)
		}
	}

	man := Manifest{
		Format:      FormatVersion,
		CreatedAt:   time.Now().UTC().Format(time.RFC3339),
		Files:       files,
		RevisionIDs: revIDs,
		Note:        manifestNote,
	}
	manRaw, err := json.MarshalIndent(man, "", "  ")
	if err != nil {
		return finalizeFail(err)
	}
	if uint64(len(manRaw)) > maxZipFileBytes {
		return finalizeFail(errors.New("backup excede o limite de 20 MiB por arquivo"))
	}
	fileCount++
	if fileCount > maxZipFiles {
		return finalizeFail(errors.New("backup excede o número máximo de arquivos"))
	}
	uncompressed += uint64(len(manRaw))
	if uncompressed > maxZipUncompressed {
		return finalizeFail(errors.New("backup excede o tamanho descomprimido máximo de 1 GiB"))
	}
	w, err := zw.CreateHeader(&zip.FileHeader{
		Name:     "manifest.json",
		Method:   zip.Deflate,
		Modified: time.Now().UTC(),
	})
	if err != nil {
		return finalizeFail(err)
	}
	if _, err := w.Write(manRaw); err != nil {
		return finalizeFail(err)
	}

	if err := zw.Close(); err != nil {
		_ = zf.Close()
		_ = os.Remove(partial)
		return err
	}
	if err := zf.Close(); err != nil {
		_ = os.Remove(partial)
		return err
	}

	fi, err := os.Stat(partial)
	if err != nil {
		_ = os.Remove(partial)
		return err
	}
	if fi.Size() > maxArchiveBytes {
		_ = os.Remove(partial)
		return errors.New("backup excede o tamanho comprimido máximo de 1 GiB")
	}

	_ = os.Remove(outPath)
	if err := os.Rename(partial, outPath); err != nil {
		_ = os.Remove(partial)
		return fmt.Errorf("falha ao finalizar arquivo de backup: %w", err)
	}
	return nil
}

// DefaultBackupPath suggests dataDir/backups/crv-backup-<UTC timestamp>.zip for the app layer.
func DefaultBackupPath(dataDir string) string {
	ts := time.Now().UTC().Format("20060102T150405Z")
	return filepath.Join(dataDir, "backups", "crv-backup-"+ts+".zip")
}

func sha256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func writeZipFileFromPath(zw *zip.Writer, name, srcPath string) error {
	info, err := os.Stat(srcPath)
	if err != nil {
		return err
	}
	hdr, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	hdr.Name = name
	hdr.Method = zip.Deflate
	w, err := zw.CreateHeader(hdr)
	if err != nil {
		return err
	}
	f, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(w, f)
	return err
}

func dumpPreferences(st *store.Store) (map[string]string, error) {
	rows, err := st.DB.Query(`SELECT key, value FROM preferences ORDER BY key`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]string)
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		out[k] = v
	}
	return out, rows.Err()
}

// verifyLiveRevisionFiles ensures every original/display path referenced by the
// live database for revID exists before the archive is finalized.
func verifyLiveRevisionFiles(st *store.Store, revID int64) error {
	imgs, err := st.RevisionImages(revID)
	if err != nil {
		return err
	}
	root := store.RevisionDir(st.DataDir, revID)
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return fmt.Errorf("revisão de catálogo ausente: rev-%d", revID)
	}
	if len(imgs) == 0 {
		return fmt.Errorf("revisão sem imagens no banco: rev-%d", revID)
	}
	for _, img := range imgs {
		for _, rel := range []string{img.OriginalRelpath, img.DisplayRelpath} {
			if rel == "" {
				return fmt.Errorf("caminho de imagem ausente na revisão rev-%d", revID)
			}
			p := filepath.Join(root, filepath.FromSlash(rel))
			fi, err := os.Lstat(p)
			if err != nil || fi.Mode()&os.ModeSymlink != 0 || !fi.Mode().IsRegular() || fi.Size() <= 0 {
				return fmt.Errorf("arquivo de catálogo ausente no backup (rev-%d/%s)", revID, rel)
			}
		}
	}
	return nil
}

func referencedRevisionIDs(st *store.Store) ([]int64, error) {
	set := make(map[int64]struct{})
	active, err := st.ActiveCatalog()
	if err != nil {
		return nil, err
	}
	if active != nil {
		set[active.RevisionID] = struct{}{}
	}
	rows, err := st.DB.Query(`SELECT DISTINCT catalog_revision_id FROM sessions WHERE catalog_revision_id IS NOT NULL`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		if id > 0 {
			set[id] = struct{}{}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(set))
	for id := range set {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids, nil
}
