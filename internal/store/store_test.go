package store

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpenMigrateAndActivate(t *testing.T) {
	dir := t.TempDir()
	st, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	if _, err := os.Stat(filepath.Join(dir, "crv.sqlite")); err != nil {
		t.Fatal(err)
	}
	active, err := st.ActiveCatalog()
	if err != nil {
		t.Fatal(err)
	}
	if active != nil {
		t.Fatal("expected no active catalog")
	}

	id, err := st.ActivateRevision("test", []ImageRow{{
		SHA256: "aa", OriginalRelpath: "originals/aa.jpg", DisplayRelpath: "display/aa.png",
		Width: 1, Height: 1, Bytes: 10, SourcesJSON: "[]",
	}}, nil, map[string]any{"ready": true})
	if err != nil {
		t.Fatal(err)
	}
	active, err = st.ActiveCatalog()
	if err != nil || active == nil || active.RevisionID != id {
		t.Fatalf("active=%v err=%v", active, err)
	}

	_ = st.Close()
	st2, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer st2.Close()
	active2, err := st2.ActiveCatalog()
	if err != nil || active2 == nil || active2.RevisionID != id {
		t.Fatalf("persist failed: %v %v", active2, err)
	}
	var ver int
	if err := st2.DB.QueryRow(`SELECT COUNT(1) FROM schema_migrations`).Scan(&ver); err != nil || ver < 1 {
		t.Fatalf("migrations missing: %v %d", err, ver)
	}
}
