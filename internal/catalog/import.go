package catalog

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/Thiagojm/crv-go/internal/store"
)

const (
	maxArchiveBytes    = 1 << 30 // 1 GiB compressed input
	maxZipUncompressed = 1 << 30 // 1 GiB uncompressed total
	maxZipFiles        = 10_000
	maxZipFileBytes    = 20 << 20 // 20 MiB per file
)

// ExtractZip unpacks a ZIP into dest with path/link/size bounds. Does not modify the reader source beyond reading.
func ExtractZip(r io.Reader, dest string) error {
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dest, ".archive-*.zip")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
	}()

	n, err := io.Copy(tmp, io.LimitReader(r, maxArchiveBytes+1))
	if err != nil {
		return fmt.Errorf("leitura do ZIP: %w", err)
	}
	if n > maxArchiveBytes {
		return errors.New("arquivo ZIP excede o tamanho máximo permitido")
	}
	if _, err := tmp.Seek(0, io.SeekStart); err != nil {
		return err
	}
	zr, err := zip.NewReader(tmp, n)
	if err != nil {
		return errors.New("arquivo ZIP inválido ou corrompido")
	}

	var files int
	var total uint64
	destClean := filepath.Clean(dest)

	for _, f := range zr.File {
		rel, isDir, err := safeZipEntry(f.Name)
		if err != nil {
			return err
		}
		mode := f.Mode()
		if mode&os.ModeSymlink != 0 {
			return errors.New("atalho/link rejeitado no arquivo ZIP")
		}
		target := filepath.Join(destClean, filepath.FromSlash(rel))
		if err := assertUnderRoot(destClean, target); err != nil {
			return err
		}
		if isDir || strings.HasSuffix(f.Name, "/") || f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}
		if mode != 0 && !mode.IsRegular() && mode&os.ModeType != 0 {
			return errors.New("entrada especial rejeitada no arquivo ZIP")
		}
		files++
		if files > maxZipFiles {
			return errors.New("arquivo ZIP excede o número máximo de arquivos")
		}
		if f.UncompressedSize64 > maxZipFileBytes {
			return errors.New("arquivo no ZIP excede 20 MiB")
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := writeZipFile(f, target); err != nil {
			return err
		}
		info, err := os.Lstat(target)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			_ = os.Remove(target)
			return errors.New("atalho/link rejeitado no arquivo ZIP")
		}
		total += uint64(info.Size())
		if total > maxZipUncompressed {
			return errors.New("arquivo ZIP excede o tamanho descomprimido máximo")
		}
	}
	return nil
}

func writeZipFile(f *zip.File, target string) error {
	rc, err := f.Open()
	if err != nil {
		return errors.New("arquivo ZIP inválido ou corrompido")
	}
	defer rc.Close()

	out, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()

	written, err := io.Copy(out, io.LimitReader(rc, maxZipFileBytes+1))
	if err != nil {
		_ = os.Remove(target)
		return err
	}
	if written > maxZipFileBytes {
		_ = os.Remove(target)
		return errors.New("arquivo no ZIP excede 20 MiB")
	}
	if f.UncompressedSize64 > 0 && uint64(written) > f.UncompressedSize64 {
		_ = os.Remove(target)
		return errors.New("tamanho descomprimido inconsistente no ZIP")
	}
	return nil
}

func safeZipEntry(name string) (rel string, isDir bool, err error) {
	name = strings.ReplaceAll(name, "\\", "/")
	if name == "" {
		return "", false, errors.New("caminho inseguro no arquivo ZIP")
	}
	if strings.HasPrefix(name, "/") || strings.HasPrefix(name, "//") {
		return "", false, errors.New("caminho absoluto rejeitado no ZIP")
	}
	if len(name) >= 2 && name[1] == ':' {
		return "", false, errors.New("caminho absoluto rejeitado no ZIP")
	}
	if strings.Contains(name, ":") {
		return "", false, errors.New("caminho absoluto rejeitado no ZIP")
	}
	isDir = strings.HasSuffix(name, "/")
	clean := path.Clean(name)
	if clean == "." || clean == "" {
		return "", false, errors.New("caminho inseguro no arquivo ZIP")
	}
	if clean == ".." || strings.HasPrefix(clean, "../") {
		return "", false, errors.New("caminho com travessia rejeitado no ZIP")
	}
	return clean, isDir, nil
}

func assertUnderRoot(root, target string) error {
	root = filepath.Clean(root)
	target = filepath.Clean(target)
	rel, err := filepath.Rel(root, target)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return errors.New("caminho com travessia rejeitado no ZIP")
	}
	return nil
}

// ImportFolder installs a local catalog folder without modifying the source tree.
func ImportFolder(st *store.Store, folder, label string) (*InstallResult, error) {
	folder = filepath.Clean(folder)
	if folder == "" || folder == "." {
		return &InstallResult{Report: Report{Error: "pasta de importação inválida", Ready: false}}, errors.New("pasta de importação inválida")
	}
	info, err := os.Lstat(folder)
	if err != nil {
		return &InstallResult{Report: Report{Error: "pasta de importação ausente ou ilegível", Ready: false}}, fmt.Errorf("stat folder: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return &InstallResult{Report: Report{Error: "atalho/link rejeitado na pasta de importação", Ready: false}}, errors.New("atalho/link rejeitado na pasta de importação")
	}
	if !info.IsDir() {
		return &InstallResult{Report: Report{Error: "caminho de importação não é uma pasta", Ready: false}}, errors.New("caminho de importação não é uma pasta")
	}
	if label == "" {
		label = "import-folder"
	}
	return InstallLabeled(st, folder, label)
}

// ImportZip extracts a ZIP to a temp directory under the store data dir, installs it, then removes the temp tree.
func ImportZip(st *store.Store, r io.Reader, label string) (*InstallResult, error) {
	if label == "" {
		label = "import-zip"
	}
	tmpRoot := filepath.Join(st.DataDir, "catalog-import-tmp")
	_ = os.RemoveAll(tmpRoot)
	if err := os.MkdirAll(tmpRoot, 0o755); err != nil {
		return nil, err
	}
	defer func() { _ = os.RemoveAll(tmpRoot) }()

	extractDir := filepath.Join(tmpRoot, "pkg")
	if err := ExtractZip(r, extractDir); err != nil {
		msg := err.Error()
		return &InstallResult{Report: Report{Error: msg, Ready: false}}, err
	}
	pkgDir, err := resolvePackageRoot(extractDir)
	if err != nil {
		return &InstallResult{Report: Report{Error: err.Error(), Ready: false}}, err
	}
	return InstallLabeled(st, pkgDir, label)
}

func resolvePackageRoot(extractDir string) (string, error) {
	inv := filepath.Join(extractDir, "catalog-unified.json")
	if _, err := os.Stat(inv); err == nil {
		return extractDir, nil
	}
	entries, err := os.ReadDir(extractDir)
	if err != nil {
		return "", errors.New("pacote ZIP ilegível")
	}
	var onlyDir string
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		if !e.IsDir() {
			return "", errors.New("catalog-unified.json ausente no ZIP")
		}
		if onlyDir != "" {
			return "", errors.New("catalog-unified.json ausente no ZIP")
		}
		onlyDir = filepath.Join(extractDir, name)
	}
	if onlyDir == "" {
		return "", errors.New("catalog-unified.json ausente no ZIP")
	}
	if _, err := os.Stat(filepath.Join(onlyDir, "catalog-unified.json")); err != nil {
		return "", errors.New("catalog-unified.json ausente no ZIP")
	}
	return onlyDir, nil
}
