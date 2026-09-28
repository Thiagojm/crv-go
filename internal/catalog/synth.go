package catalog

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
)

// WriteSynthetic writes n tiny unique JPEGs and catalog-unified.json into dir.
func WriteSynthetic(dir string, n int) ([]string, error) {
	if n < 4 {
		n = 4
	}
	if err := os.MkdirAll(filepath.Join(dir, "images"), 0o755); err != nil {
		return nil, err
	}
	inv := Inventory{
		Format:  expectedFormat,
		Summary: &InventorySummary{PoolEntries: n, Files: n, UniqueHashes: n, SingleImageCandidates: n},
	}
	hashes := make([]string, 0, n)
	for i := 0; i < n; i++ {
		rel := fmt.Sprintf("images/t%02d.jpg", i)
		hash, err := writeJPEGFile(filepath.Join(dir, rel), 8, 8, byte(40+i*3))
		if err != nil {
			return nil, err
		}
		hashes = append(hashes, hash)
		inv.Images = append(inv.Images, InventoryImage{
			SHA256: hash,
			Paths:  []string{rel},
			Sources: []SourceRecord{{
				ID:          fmt.Sprintf("A-T%02d", i),
				Pool:        "A",
				Index:       i,
				Label:       fmt.Sprintf("Alvo sintético %02d", i),
				Description: "Imagem sintética de teste; sem valor experimental.",
				Credit:      "fixture",
				Images:      []SourceImage{{LocalPath: rel}},
			}},
		})
	}
	raw, err := json.Marshal(inv)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(dir, "catalog-unified.json"), raw, 0o644); err != nil {
		return nil, err
	}
	return hashes, nil
}

func writeJPEGFile(path string, w, h int, salt byte) (string, error) {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x + int(salt)), G: uint8(y + int(salt)), B: salt, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90}); err != nil {
		return "", err
	}
	sum := sha256.Sum256(buf.Bytes())
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		return "", err
	}
	return hex.EncodeToString(sum[:]), nil
}
