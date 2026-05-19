package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func inTempDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(orig) })
	return dir
}

func captureOutput(t *testing.T, fn func() error) (string, error) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	oldStdout := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = oldStdout }()

	fnErr := fn()
	w.Close()

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String(), fnErr
}

func assertSymlink(t *testing.T, name, target string) {
	t.Helper()
	info, err := os.Lstat(name)
	if err != nil {
		t.Fatalf("%s does not exist", name)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("%s is not a symlink", name)
	}
	got, err := os.Readlink(name)
	if err != nil {
		t.Fatal(err)
	}
	if got != target {
		t.Fatalf("%s points to %q, want %q", name, got, target)
	}
}

func assertRegularFile(t *testing.T, name string) {
	t.Helper()
	info, err := os.Lstat(name)
	if err != nil {
		t.Fatalf("%s does not exist", name)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("%s is a symlink, expected regular file", name)
	}
	if !info.Mode().IsRegular() {
		t.Fatalf("%s is not a regular file", name)
	}
}

func assertFileContent(t *testing.T, name, want string) {
	t.Helper()
	data, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != want {
		t.Fatalf("%s content = %q, want %q", name, string(data), want)
	}
}

func TestNoFilesExist(t *testing.T) {
	inTempDir(t)

	output, err := captureOutput(t, func() error { return run(false) })
	if err != nil {
		t.Fatal(err)
	}

	assertRegularFile(t, "AGENTS.md")
	assertSymlink(t, "CLAUDE.md", "AGENTS.md")
	assertSymlink(t, "GEMINI.md", "AGENTS.md")
	assertFileContent(t, "AGENTS.md", "")

	if !strings.Contains(output, "created AGENTS.md") {
		t.Fatalf("expected 'created AGENTS.md' in output, got: %s", output)
	}
}

func TestOnlyAgentsMdExists(t *testing.T) {
	inTempDir(t)
	os.WriteFile("AGENTS.md", []byte("hello"), 0644)

	output, err := captureOutput(t, func() error { return run(false) })
	if err != nil {
		t.Fatal(err)
	}

	assertRegularFile(t, "AGENTS.md")
	assertSymlink(t, "CLAUDE.md", "AGENTS.md")
	assertSymlink(t, "GEMINI.md", "AGENTS.md")
	assertFileContent(t, "AGENTS.md", "hello")

	if !strings.Contains(output, "CLAUDE.md → AGENTS.md") {
		t.Fatalf("expected symlink message in output, got: %s", output)
	}
}

func TestOnlyClaudeMdExists(t *testing.T) {
	inTempDir(t)
	os.WriteFile("CLAUDE.md", []byte("content"), 0644)

	_, err := captureOutput(t, func() error { return run(false) })
	if err != nil {
		t.Fatal(err)
	}

	assertRegularFile(t, "AGENTS.md")
	assertSymlink(t, "CLAUDE.md", "AGENTS.md")
	assertSymlink(t, "GEMINI.md", "AGENTS.md")
	assertFileContent(t, "AGENTS.md", "content")
}

func TestAllThreeIdenticalContent(t *testing.T) {
	inTempDir(t)
	for _, f := range allFiles {
		os.WriteFile(f, []byte("same"), 0644)
	}

	_, err := captureOutput(t, func() error { return run(false) })
	if err != nil {
		t.Fatal(err)
	}

	assertRegularFile(t, "AGENTS.md")
	assertSymlink(t, "CLAUDE.md", "AGENTS.md")
	assertSymlink(t, "GEMINI.md", "AGENTS.md")
	assertFileContent(t, "AGENTS.md", "same")
}

func TestAlreadyCorrectState(t *testing.T) {
	inTempDir(t)
	os.WriteFile("AGENTS.md", []byte("ok"), 0644)
	os.Symlink("AGENTS.md", "CLAUDE.md")
	os.Symlink("AGENTS.md", "GEMINI.md")

	output, err := captureOutput(t, func() error { return run(false) })
	if err != nil {
		t.Fatal(err)
	}

	if output != "" {
		t.Fatalf("expected no output for already-correct state, got: %q", output)
	}
}

func TestWrongSymlinkTarget(t *testing.T) {
	inTempDir(t)
	os.WriteFile("AGENTS.md", []byte("ok"), 0644)
	os.WriteFile("other.md", []byte("x"), 0644)
	os.Symlink("other.md", "CLAUDE.md")
	os.Symlink("AGENTS.md", "GEMINI.md")

	_, err := captureOutput(t, func() error { return run(false) })
	if err != nil {
		t.Fatal(err)
	}

	assertSymlink(t, "CLAUDE.md", "AGENTS.md")
	assertSymlink(t, "GEMINI.md", "AGENTS.md")
}

func TestCheckCorrectState(t *testing.T) {
	inTempDir(t)
	os.WriteFile("AGENTS.md", []byte("ok"), 0644)
	os.Symlink("AGENTS.md", "CLAUDE.md")
	os.Symlink("AGENTS.md", "GEMINI.md")

	output, err := captureOutput(t, func() error { return run(true) })
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if output != "" {
		t.Fatalf("expected no output, got: %q", output)
	}
}

func TestCheckIncorrectState(t *testing.T) {
	inTempDir(t)
	os.WriteFile("AGENTS.md", []byte("ok"), 0644)
	// CLAUDE.md missing, GEMINI.md missing

	output, err := captureOutput(t, func() error { return run(true) })
	if err == nil {
		t.Fatal("expected error for incorrect state")
	}

	if !strings.Contains(output, "CLAUDE.md is missing") {
		t.Fatalf("expected missing message, got: %s", output)
	}

	// Verify nothing was changed
	if _, err := os.Lstat("CLAUDE.md"); err == nil {
		t.Fatal("CLAUDE.md should not have been created in check mode")
	}
}

func TestConflictResolution(t *testing.T) {
	inTempDir(t)
	os.WriteFile("AGENTS.md", []byte("content-a"), 0644)
	os.WriteFile("CLAUDE.md", []byte("content-c"), 0644)

	// Simulate stdin input: choose option 2 (CLAUDE.md)
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	w.WriteString("2\n")
	w.Close()
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	_, err := captureOutput(t, func() error { return run(false) })
	if err != nil {
		t.Fatal(err)
	}

	assertRegularFile(t, "AGENTS.md")
	assertSymlink(t, "CLAUDE.md", "AGENTS.md")
	assertSymlink(t, "GEMINI.md", "AGENTS.md")
	assertFileContent(t, "AGENTS.md", "content-c")
}

func TestOnlyGeminiMdExists(t *testing.T) {
	inTempDir(t)
	os.WriteFile("GEMINI.md", []byte("gemini-content"), 0644)

	_, err := captureOutput(t, func() error { return run(false) })
	if err != nil {
		t.Fatal(err)
	}

	assertRegularFile(t, "AGENTS.md")
	assertSymlink(t, "CLAUDE.md", "AGENTS.md")
	assertSymlink(t, "GEMINI.md", "AGENTS.md")
	assertFileContent(t, "AGENTS.md", "gemini-content")
}

func TestSymlinksResolveToAgentsContent(t *testing.T) {
	inTempDir(t)
	os.WriteFile("AGENTS.md", []byte("shared content"), 0644)

	captureOutput(t, func() error { return run(false) })

	// Reading through symlinks should give AGENTS.md content
	for _, f := range symlinkFiles {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != "shared content" {
			t.Fatalf("%s content = %q, want %q", f, string(data), "shared content")
		}
	}
}

func TestCheckConflictNoPanic(t *testing.T) {
	inTempDir(t)
	os.WriteFile("AGENTS.md", []byte("aaa"), 0644)
	os.WriteFile("CLAUDE.md", []byte("bbb"), 0644)

	output, err := captureOutput(t, func() error { return run(true) })
	if err == nil {
		t.Fatal("expected error for conflict in check mode")
	}
	if !strings.Contains(output, "conflict") {
		t.Fatalf("expected conflict message, got: %s", output)
	}

	// Files should be unchanged
	assertFileContent(t, "AGENTS.md", "aaa")
	assertFileContent(t, "CLAUDE.md", "bbb")
}

func TestAbsoluteSymlinkPath(t *testing.T) {
	// Verify symlinks are relative, not absolute
	inTempDir(t)
	os.WriteFile("AGENTS.md", []byte("test"), 0644)

	captureOutput(t, func() error { return run(false) })

	target, _ := os.Readlink("CLAUDE.md")
	if filepath.IsAbs(target) {
		t.Fatalf("symlink should be relative, got absolute: %s", target)
	}
}
