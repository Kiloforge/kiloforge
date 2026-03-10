package sqlite

import (
	"database/sql"
	"testing"
	"time"

	"kiloforge/internal/core/domain"
)

func TestProjectStore_AddAndGet(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	store := NewProjectStore(db)

	p := domain.Project{
		Slug:         "myproject",
		RepoName:     "myproject",
		ProjectDir:   "/tmp/myproject",
		OriginRemote: "https://github.com/test/myproject.git",
		RegisteredAt: time.Now().Truncate(time.Second),
		Active:       true,
	}

	if err := store.Add(p); err != nil {
		t.Fatalf("Add: %v", err)
	}

	got, err := store.Get("myproject")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Slug != p.Slug {
		t.Errorf("Slug: want %q, got %q", p.Slug, got.Slug)
	}
	if got.OriginRemote != p.OriginRemote {
		t.Errorf("OriginRemote: want %q, got %q", p.OriginRemote, got.OriginRemote)
	}
	if !got.Active {
		t.Error("Active: want true")
	}
}

func TestProjectStore_List(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	store := NewProjectStore(db)

	store.Add(domain.Project{Slug: "beta", RepoName: "beta", ProjectDir: "/b", RegisteredAt: time.Now(), Active: true})
	store.Add(domain.Project{Slug: "alpha", RepoName: "alpha", ProjectDir: "/a", RegisteredAt: time.Now(), Active: true})

	list := store.List()
	if len(list) != 2 {
		t.Fatalf("List: want 2, got %d", len(list))
	}
	if list[0].Slug != "alpha" {
		t.Errorf("List[0].Slug: want alpha, got %q", list[0].Slug)
	}
}

func TestProjectStore_Remove(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	store := NewProjectStore(db)

	store.Add(domain.Project{Slug: "rm-me", RepoName: "rm-me", ProjectDir: "/rm", RegisteredAt: time.Now(), Active: true})

	if err := store.Remove("rm-me"); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if _, err := store.Get("rm-me"); err == nil {
		t.Error("Get after Remove: should not find")
	}
}

func TestProjectStore_FindByRepoName(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	store := NewProjectStore(db)

	store.Add(domain.Project{Slug: "slug1", RepoName: "target-repo", ProjectDir: "/x", RegisteredAt: time.Now(), Active: true})

	got, ok := store.FindByRepoName("target-repo")
	if !ok {
		t.Fatal("FindByRepoName: not found")
	}
	if got.Slug != "slug1" {
		t.Errorf("Slug: want slug1, got %q", got.Slug)
	}
}

func TestProjectStore_MirrorDir_RoundTrip(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	store := NewProjectStore(db)

	p := domain.Project{
		Slug:       "mirror-rt",
		RepoName:   "mirror-rt",
		ProjectDir: "/tmp/mirror-rt",
		MirrorDir:  "/home/user/projects/mirror-rt",
		RegisteredAt: time.Now().Truncate(time.Second),
		Active:     true,
	}
	if err := store.Add(p); err != nil {
		t.Fatalf("Add: %v", err)
	}

	got, err := store.Get("mirror-rt")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.MirrorDir != p.MirrorDir {
		t.Errorf("MirrorDir: want %q, got %q", p.MirrorDir, got.MirrorDir)
	}

	// Also verify through List.
	list := store.List()
	if len(list) != 1 || list[0].MirrorDir != p.MirrorDir {
		t.Errorf("List MirrorDir: want %q, got %q", p.MirrorDir, list[0].MirrorDir)
	}

	// FindByRepoName.
	found, ok := store.FindByRepoName("mirror-rt")
	if !ok {
		t.Fatal("FindByRepoName: not found")
	}
	if found.MirrorDir != p.MirrorDir {
		t.Errorf("FindByRepoName MirrorDir: want %q, got %q", p.MirrorDir, found.MirrorDir)
	}

	// FindByDir.
	found2, ok := store.FindByDir("/tmp/mirror-rt")
	if !ok {
		t.Fatal("FindByDir: not found")
	}
	if found2.MirrorDir != p.MirrorDir {
		t.Errorf("FindByDir MirrorDir: want %q, got %q", p.MirrorDir, found2.MirrorDir)
	}
}

func TestProjectStore_SaveIsNoop(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	store := NewProjectStore(db)
	if err := store.Save(); err != nil {
		t.Errorf("Save: %v", err)
	}
}

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}
