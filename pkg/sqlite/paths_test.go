package sqlite

import (
	"context"
	"path/filepath"
	"testing"
)

func TestSidecarDirFileDB(t *testing.T) {
	db := &DB{cfg: Config{DSN: `C:\tmp\anyclaw.db`}}

	got := db.SidecarDir("vec")
	want := `C:\tmp\anyclaw.vec`
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestSidecarDirInMemoryDB(t *testing.T) {
	db := &DB{cfg: Config{DSN: ":memory:"}}

	if got := db.SidecarDir("vec"); got != "" {
		t.Fatalf("expected empty sidecar dir for in-memory db, got %q", got)
	}
}

func TestSidecarDirNilAndMemoryVariants(t *testing.T) {
	var nilDB *DB
	if got := nilDB.SidecarDir("vec"); got != "" {
		t.Fatalf("expected empty sidecar dir for nil db, got %q", got)
	}

	blank := &DB{cfg: Config{DSN: "   "}}
	if got := blank.SidecarDir("vec"); got != "" {
		t.Fatalf("expected empty sidecar dir for blank dsn, got %q", got)
	}

	mem := &DB{cfg: Config{DSN: "file:memdb1?mode=memory&cache=shared"}}
	if got := mem.SidecarDir("vec"); got != "" {
		t.Fatalf("expected empty sidecar dir for mode=memory dsn, got %q", got)
	}
}

func TestSidecarDirFileDSNVariants(t *testing.T) {
	db := &DB{cfg: Config{DSN: `file:C:\tmp\anyclaw.db?cache=shared`}}

	if got := db.SidecarDir(""); got != `C:\tmp\anyclaw` {
		t.Fatalf("expected base sidecar path %q, got %q", `C:\tmp\anyclaw`, got)
	}
	if got := db.SidecarDir("vec"); got != `C:\tmp\anyclaw.vec` {
		t.Fatalf("expected vec sidecar path %q, got %q", `C:\tmp\anyclaw.vec`, got)
	}
}

func TestSidecarDirForSQLDBFileDB(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "anyclaw.db")
	db, err := Open(DefaultConfig(dbPath))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() {
		_ = db.Close()
	}()

	got := SidecarDirForSQLDB(context.Background(), db.DB, "vec")
	want := filepath.Join(filepath.Dir(dbPath), "anyclaw.vec")
	if got != want {
		t.Fatalf("expected sidecar path %q, got %q", want, got)
	}
}

func TestSidecarDirForSQLDBInMemoryDB(t *testing.T) {
	db, err := Open(InMemoryConfig())
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() {
		_ = db.Close()
	}()

	if got := SidecarDirForSQLDB(context.Background(), db.DB, "vec"); got != "" {
		t.Fatalf("expected empty sidecar path for in-memory sql db, got %q", got)
	}
}
