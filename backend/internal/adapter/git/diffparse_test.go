package git

import (
	"testing"

	"kiloforge/internal/core/domain"
)

func intPtr(v int) *int { return &v }

func TestParseUnifiedDiff(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantFiles int
		check     func(t *testing.T, files []domain.FileDiff)
	}{
		{
			name:      "empty diff",
			input:     "",
			wantFiles: 0,
		},
		{
			name: "modified file with single hunk",
			input: `diff --git a/foo.go b/foo.go
index abc1234..def5678 100644
--- a/foo.go
+++ b/foo.go
@@ -1,3 +1,4 @@
 package foo

+// Added comment
 func Bar() {}
`,
			wantFiles: 1,
			check: func(t *testing.T, files []domain.FileDiff) {
				f := files[0]
				if f.Path != "foo.go" {
					t.Errorf("path = %q, want %q", f.Path, "foo.go")
				}
				if f.Status != domain.FileStatusModified {
					t.Errorf("status = %q, want %q", f.Status, domain.FileStatusModified)
				}
				if f.Insertions != 1 || f.Deletions != 0 {
					t.Errorf("stats = +%d/-%d, want +1/-0", f.Insertions, f.Deletions)
				}
				if len(f.Hunks) != 1 {
					t.Fatalf("hunks = %d, want 1", len(f.Hunks))
				}
				h := f.Hunks[0]
				if h.OldStart != 1 || h.OldLines != 3 || h.NewStart != 1 || h.NewLines != 4 {
					t.Errorf("hunk header = -%d,%d +%d,%d, want -1,3 +1,4", h.OldStart, h.OldLines, h.NewStart, h.NewLines)
				}
				if len(h.Lines) != 4 {
					t.Fatalf("lines = %d, want 4", len(h.Lines))
				}
				// Check the added line
				added := h.Lines[2]
				if added.Type != domain.DiffLineAdd {
					t.Errorf("line[2] type = %q, want %q", added.Type, domain.DiffLineAdd)
				}
				if added.Content != "// Added comment" {
					t.Errorf("line[2] content = %q, want %q", added.Content, "// Added comment")
				}
				if added.OldNo != nil {
					t.Errorf("line[2] old_no = %v, want nil", *added.OldNo)
				}
				if added.NewNo == nil || *added.NewNo != 3 {
					t.Errorf("line[2] new_no = %v, want 3", added.NewNo)
				}
			},
		},
		{
			name: "new file",
			input: `diff --git a/new.go b/new.go
new file mode 100644
index 0000000..abc1234
--- /dev/null
+++ b/new.go
@@ -0,0 +1,3 @@
+package new
+
+func Hello() {}
`,
			wantFiles: 1,
			check: func(t *testing.T, files []domain.FileDiff) {
				f := files[0]
				if f.Path != "new.go" {
					t.Errorf("path = %q, want %q", f.Path, "new.go")
				}
				if f.Status != domain.FileStatusAdded {
					t.Errorf("status = %q, want %q", f.Status, domain.FileStatusAdded)
				}
				if f.Insertions != 3 {
					t.Errorf("insertions = %d, want 3", f.Insertions)
				}
			},
		},
		{
			name: "deleted file",
			input: `diff --git a/old.go b/old.go
deleted file mode 100644
index abc1234..0000000
--- a/old.go
+++ /dev/null
@@ -1,2 +0,0 @@
-package old
-func Gone() {}
`,
			wantFiles: 1,
			check: func(t *testing.T, files []domain.FileDiff) {
				f := files[0]
				if f.Path != "old.go" {
					t.Errorf("path = %q, want %q", f.Path, "old.go")
				}
				if f.Status != domain.FileStatusDeleted {
					t.Errorf("status = %q, want %q", f.Status, domain.FileStatusDeleted)
				}
				if f.Deletions != 2 {
					t.Errorf("deletions = %d, want 2", f.Deletions)
				}
			},
		},
		{
			name: "renamed file",
			input: `diff --git a/old_name.go b/new_name.go
similarity index 95%
rename from old_name.go
rename to new_name.go
index abc1234..def5678 100644
--- a/old_name.go
+++ b/new_name.go
@@ -1,3 +1,3 @@
 package pkg

-func OldName() {}
+func NewName() {}
`,
			wantFiles: 1,
			check: func(t *testing.T, files []domain.FileDiff) {
				f := files[0]
				if f.Path != "new_name.go" {
					t.Errorf("path = %q, want %q", f.Path, "new_name.go")
				}
				if f.OldPath != "old_name.go" {
					t.Errorf("old_path = %q, want %q", f.OldPath, "old_name.go")
				}
				if f.Status != domain.FileStatusRenamed {
					t.Errorf("status = %q, want %q", f.Status, domain.FileStatusRenamed)
				}
			},
		},
		{
			name: "binary file",
			input: `diff --git a/image.png b/image.png
new file mode 100644
index 0000000..abc1234
Binary files /dev/null and b/image.png differ
`,
			wantFiles: 1,
			check: func(t *testing.T, files []domain.FileDiff) {
				f := files[0]
				if f.Path != "image.png" {
					t.Errorf("path = %q, want %q", f.Path, "image.png")
				}
				if !f.IsBinary {
					t.Error("expected is_binary = true")
				}
				if len(f.Hunks) != 0 {
					t.Errorf("hunks = %d, want 0 for binary", len(f.Hunks))
				}
			},
		},
		{
			name: "multiple files",
			input: `diff --git a/a.go b/a.go
index abc..def 100644
--- a/a.go
+++ b/a.go
@@ -1,2 +1,3 @@
 package a
+// new
 func A() {}
diff --git a/b.go b/b.go
index abc..def 100644
--- a/b.go
+++ b/b.go
@@ -1,2 +1,2 @@
 package b
-func B() {}
+func B2() {}
`,
			wantFiles: 2,
			check: func(t *testing.T, files []domain.FileDiff) {
				if files[0].Path != "a.go" {
					t.Errorf("file[0].path = %q, want %q", files[0].Path, "a.go")
				}
				if files[1].Path != "b.go" {
					t.Errorf("file[1].path = %q, want %q", files[1].Path, "b.go")
				}
				if files[0].Insertions != 1 || files[0].Deletions != 0 {
					t.Errorf("file[0] stats = +%d/-%d, want +1/-0", files[0].Insertions, files[0].Deletions)
				}
				if files[1].Insertions != 1 || files[1].Deletions != 1 {
					t.Errorf("file[1] stats = +%d/-%d, want +1/-1", files[1].Insertions, files[1].Deletions)
				}
			},
		},
		{
			name: "multiple hunks in one file",
			input: `diff --git a/multi.go b/multi.go
index abc..def 100644
--- a/multi.go
+++ b/multi.go
@@ -1,3 +1,4 @@
 package multi

+// first addition
 func A() {}
@@ -10,3 +11,4 @@
 func B() {}

+// second addition
 func C() {}
`,
			wantFiles: 1,
			check: func(t *testing.T, files []domain.FileDiff) {
				if len(files[0].Hunks) != 2 {
					t.Fatalf("hunks = %d, want 2", len(files[0].Hunks))
				}
				if files[0].Hunks[0].OldStart != 1 {
					t.Errorf("hunk[0] old_start = %d, want 1", files[0].Hunks[0].OldStart)
				}
				if files[0].Hunks[1].OldStart != 10 {
					t.Errorf("hunk[1] old_start = %d, want 10", files[0].Hunks[1].OldStart)
				}
			},
		},
		{
			name: "no newline at EOF marker",
			input: `diff --git a/noeol.go b/noeol.go
index abc..def 100644
--- a/noeol.go
+++ b/noeol.go
@@ -1,2 +1,2 @@
 package noeol
-func Old() {}
\ No newline at end of file
+func New() {}
\ No newline at end of file
`,
			wantFiles: 1,
			check: func(t *testing.T, files []domain.FileDiff) {
				f := files[0]
				// The "\ No newline at end of file" markers should be skipped
				if f.Insertions != 1 || f.Deletions != 1 {
					t.Errorf("stats = +%d/-%d, want +1/-1", f.Insertions, f.Deletions)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files, err := ParseUnifiedDiff(tt.input)
			if err != nil {
				t.Fatalf("ParseUnifiedDiff() error = %v", err)
			}
			if len(files) != tt.wantFiles {
				t.Fatalf("got %d files, want %d", len(files), tt.wantFiles)
			}
			if tt.check != nil {
				tt.check(t, files)
			}
		})
	}
}
