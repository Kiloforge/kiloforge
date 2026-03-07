package jsonfile

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"crelay/internal/core/domain"
)

func TestProjectStore_LoadEmpty(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store, err := LoadProjectStore(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store.Version != 1 {
		t.Errorf("expected version 1, got %d", store.Version)
	}
	if len(store.Projects) != 0 {
		t.Errorf("expected empty projects, got %d", len(store.Projects))
	}
}

func TestProjectStore_SaveAndLoad(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store, _ := LoadProjectStore(dir)

	now := time.Now().Truncate(time.Second)
	p := domain.Project{
		Slug:         "myproject",
		RepoName:     "myproject",
		ProjectDir:   "/home/user/myproject",
		OriginRemote: "git@github.com:user/myproject.git",
		RegisteredAt: now,
		Active:       true,
	}
	if err := store.Add(p); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := store.Save(); err != nil {
		t.Fatalf("save error: %v", err)
	}

	loaded, err := LoadProjectStore(dir)
	if err != nil {
		t.Fatalf("load error: %v", err)
	}

	got, ok := loaded.Get("myproject")
	if !ok {
		t.Fatal("expected project 'myproject' to exist")
	}
	if got.ProjectDir != "/home/user/myproject" {
		t.Errorf("expected project dir '/home/user/myproject', got %q", got.ProjectDir)
	}
	if got.OriginRemote != "git@github.com:user/myproject.git" {
		t.Errorf("expected origin remote 'git@github.com:user/myproject.git', got %q", got.OriginRemote)
	}
	if !got.RegisteredAt.Equal(now) {
		t.Errorf("expected registered_at %v, got %v", now, got.RegisteredAt)
	}
}

func TestProjectStore_AddDuplicate(t *testing.T) {
	t.Parallel()

	store, _ := LoadProjectStore(t.TempDir())

	p := domain.Project{Slug: "dup", RepoName: "dup", ProjectDir: "/a"}
	if err := store.Add(p); err != nil {
		t.Fatalf("first add should succeed: %v", err)
	}

	err := store.Add(domain.Project{Slug: "dup", RepoName: "dup", ProjectDir: "/b"})
	if err == nil {
		t.Fatal("expected error on duplicate add")
	}
}

func TestProjectStore_FindByDir(t *testing.T) {
	t.Parallel()

	store, _ := LoadProjectStore(t.TempDir())
	_ = store.Add(domain.Project{Slug: "proj1", ProjectDir: "/home/user/proj1"})
	_ = store.Add(domain.Project{Slug: "proj2", ProjectDir: "/home/user/proj2"})

	got, ok := store.FindByDir("/home/user/proj1")
	if !ok {
		t.Fatal("expected to find project by dir")
	}
	if got.Slug != "proj1" {
		t.Errorf("expected slug 'proj1', got %q", got.Slug)
	}

	_, ok = store.FindByDir("/home/user/nope")
	if ok {
		t.Fatal("expected not to find project for unknown dir")
	}
}

func TestProjectStore_List(t *testing.T) {
	t.Parallel()

	store, _ := LoadProjectStore(t.TempDir())
	_ = store.Add(domain.Project{Slug: "a", ProjectDir: "/a"})
	_ = store.Add(domain.Project{Slug: "b", ProjectDir: "/b"})

	list := store.List()
	if len(list) != 2 {
		t.Fatalf("expected 2 projects, got %d", len(list))
	}
}

func TestEnsureProjectDir(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := EnsureProjectDir(dir, "myproj"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	logsDir := filepath.Join(dir, "projects", "myproj", "logs")
	info, err := os.Stat(logsDir)
	if err != nil {
		t.Fatalf("expected logs dir to exist: %v", err)
	}
	if !info.IsDir() {
		t.Error("expected logs path to be a directory")
	}
}
