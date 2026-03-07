package project

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"crelay/internal/core/domain"
)

func TestRegistry_LoadEmpty(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	reg, err := LoadRegistry(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reg.Version != 1 {
		t.Errorf("expected version 1, got %d", reg.Version)
	}
	if len(reg.Projects) != 0 {
		t.Errorf("expected empty projects, got %d", len(reg.Projects))
	}
}

func TestRegistry_SaveAndLoad(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	reg := &Registry{
		Version:  1,
		Projects: map[string]domain.Project{},
	}

	now := time.Now().Truncate(time.Second)
	p := domain.Project{
		Slug:         "myproject",
		RepoName:     "myproject",
		ProjectDir:   "/home/user/myproject",
		OriginRemote: "git@github.com:user/myproject.git",
		RegisteredAt: now,
		Active:       true,
	}
	if err := reg.Add(p); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := reg.Save(dir); err != nil {
		t.Fatalf("save error: %v", err)
	}

	loaded, err := LoadRegistry(dir)
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

func TestRegistry_AddDuplicate(t *testing.T) {
	t.Parallel()

	reg := &Registry{
		Version:  1,
		Projects: map[string]domain.Project{},
	}

	p := domain.Project{Slug: "dup", RepoName: "dup", ProjectDir: "/a"}
	if err := reg.Add(p); err != nil {
		t.Fatalf("first add should succeed: %v", err)
	}

	err := reg.Add(domain.Project{Slug: "dup", RepoName: "dup", ProjectDir: "/b"})
	if err == nil {
		t.Fatal("expected error on duplicate add")
	}
}

func TestRegistry_FindByDir(t *testing.T) {
	t.Parallel()

	reg := &Registry{
		Version:  1,
		Projects: map[string]domain.Project{},
	}
	_ = reg.Add(domain.Project{Slug: "proj1", ProjectDir: "/home/user/proj1"})
	_ = reg.Add(domain.Project{Slug: "proj2", ProjectDir: "/home/user/proj2"})

	got, ok := reg.FindByDir("/home/user/proj1")
	if !ok {
		t.Fatal("expected to find project by dir")
	}
	if got.Slug != "proj1" {
		t.Errorf("expected slug 'proj1', got %q", got.Slug)
	}

	_, ok = reg.FindByDir("/home/user/nope")
	if ok {
		t.Fatal("expected not to find project for unknown dir")
	}
}

func TestRegistry_List(t *testing.T) {
	t.Parallel()

	reg := &Registry{
		Version:  1,
		Projects: map[string]domain.Project{},
	}
	_ = reg.Add(domain.Project{Slug: "a", ProjectDir: "/a"})
	_ = reg.Add(domain.Project{Slug: "b", ProjectDir: "/b"})

	list := reg.List()
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
