package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSandbox_PathEscapeAndCRUD(t *testing.T) {
	dir := t.TempDir()
	sb, err := NewSandbox(dir)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := sb.Resolve("../etc/passwd"); err == nil {
		t.Fatal("expected escape to fail")
	}

	if err := sb.WriteFile("pkg/a.go", "package pkg\n// line2\n// line3\n"); err != nil {
		t.Fatal(err)
	}
	got, err := sb.ReadFile("pkg/a.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "package pkg") {
		t.Fatalf("read: %s", got)
	}

	if err := sb.ReplaceLines("pkg/a.go", 2, 2, "// replaced"); err != nil {
		t.Fatal(err)
	}
	got, _ = sb.ReadFile("pkg/a.go")
	if !strings.Contains(got, "// replaced") {
		t.Fatalf("replace lines failed: %s", got)
	}

	if err := sb.ReplaceText("pkg/a.go", "// replaced", "// once", false); err != nil {
		t.Fatal(err)
	}

	list, err := sb.ListDir(".", true)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(list, "pkg/a.go") && !strings.Contains(list, filepath.Join("pkg", "a.go")) {
		t.Fatalf("list: %s", list)
	}

	res, err := sb.RunBash(context.Background(), "echo hi", 0)
	if err != nil {
		t.Fatal(err)
	}
	if res.ExitCode != 0 || !strings.Contains(res.Stdout, "hi") {
		t.Fatalf("bash: %+v", res)
	}

	out, err := sb.Call(context.Background(), "write_file", map[string]any{
		"path":    "b.txt",
		"content": "hello",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "b.txt") {
		t.Fatalf("call out: %s", out)
	}
	raw, _ := os.ReadFile(filepath.Join(dir, "b.txt"))
	if string(raw) != "hello" {
		t.Fatalf("file content %q", raw)
	}
}
