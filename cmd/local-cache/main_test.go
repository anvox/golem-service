package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGetCacheDir(t *testing.T) {
	home := os.Getenv("HOME")
	if home == "" {
		home = "."
	}

	t.Run("explicit custom cache dir", func(t *testing.T) {
		got := getCacheDir("/tmp/custom-cache")
		want := "/tmp/custom-cache"
		if got != want {
			t.Errorf("getCacheDir() = %q, want %q", got, want)
		}
	})

	t.Run("expand tilde", func(t *testing.T) {
		got := getCacheDir("~/my-cache")
		want := filepath.Join(home, "my-cache")
		if got != want {
			t.Errorf("getCacheDir() = %q, want %q", got, want)
		}
	})

	t.Run("default with GITHUB_REPOSITORY env var", func(t *testing.T) {
		origRepo := os.Getenv("GITHUB_REPOSITORY")
		defer os.Setenv("GITHUB_REPOSITORY", origRepo)

		os.Setenv("GITHUB_REPOSITORY", "anvox/golem-service")
		got := getCacheDir("")
		want := filepath.Join(home, "runner-cache", "anvox/golem-service")
		if got != want {
			t.Errorf("getCacheDir() = %q, want %q", got, want)
		}
	})
}

func TestFindLatestCache(t *testing.T) {
	tempDir := t.TempDir()

	f1 := filepath.Join(tempDir, "yarn-cache-1.tar.zst")
	f2 := filepath.Join(tempDir, "yarn-cache-2.tar.zst")
	f3 := filepath.Join(tempDir, "other-cache-3.tar.zst")

	if err := os.WriteFile(f1, []byte("cache1"), 0644); err != nil {
		t.Fatal(err)
	}
	time.Sleep(10 * time.Millisecond)

	if err := os.WriteFile(f2, []byte("cache2"), 0644); err != nil {
		t.Fatal(err)
	}
	time.Sleep(10 * time.Millisecond)

	if err := os.WriteFile(f3, []byte("other3"), 0644); err != nil {
		t.Fatal(err)
	}

	latest := findLatestCache(tempDir, "yarn-cache-")
	if latest != f2 {
		t.Errorf("findLatestCache() = %q, want %q", latest, f2)
	}

	latestOther := findLatestCache(tempDir, "other-cache-")
	if latestOther != f3 {
		t.Errorf("findLatestCache() = %q, want %q", latestOther, f3)
	}

	latestMissing := findLatestCache(tempDir, "nonexistent-")
	if latestMissing != "" {
		t.Errorf("findLatestCache() = %q, want empty string", latestMissing)
	}
}

func TestWriteGithubOutput(t *testing.T) {
	tempFile := filepath.Join(t.TempDir(), "github_output")
	origOutput := os.Getenv("GITHUB_OUTPUT")
	defer os.Setenv("GITHUB_OUTPUT", origOutput)

	os.Setenv("GITHUB_OUTPUT", tempFile)
	writeGithubOutput("cache-hit", "true")

	content, err := os.ReadFile(tempFile)
	if err != nil {
		t.Fatalf("Failed to read GITHUB_OUTPUT file: %v", err)
	}

	want := "cache-hit=true\n"
	if string(content) != want {
		t.Errorf("GITHUB_OUTPUT content = %q, want %q", string(content), want)
	}
}

func TestSaveAndRestoreMultipleDirectories(t *testing.T) {
	cacheDir := t.TempDir()
	workDir := t.TempDir()

	dirA := filepath.Join(workDir, "dirA")
	dirB := filepath.Join(workDir, "dirB", "sub")
	if err := os.MkdirAll(dirA, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dirB, 0755); err != nil {
		t.Fatal(err)
	}

	fileA := filepath.Join(dirA, "fileA.txt")
	fileB := filepath.Join(dirB, "fileB.txt")

	if err := os.WriteFile(fileA, []byte("content A"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fileB, []byte("content B"), 0644); err != nil {
		t.Fatal(err)
	}

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origDir)

	if err := os.Chdir(workDir); err != nil {
		t.Fatal(err)
	}

	pathsToSave := []string{"dirA", "dirB/sub", "nonexistent-dir"}
	if err := runSave("multi-test-123", pathsToSave, cacheDir); err != nil {
		t.Fatalf("runSave failed: %v", err)
	}

	// Test saving again when cache file already exists -> should skip
	if err := runSave("multi-test-123", pathsToSave, cacheDir); err != nil {
		t.Fatalf("runSave failed when cache exists: %v", err)
	}

	// Remove source files
	if err := os.RemoveAll(filepath.Join(workDir, "dirA")); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(workDir, "dirB")); err != nil {
		t.Fatal(err)
	}

	// Restore
	if err := runRestore("multi-test-123", "", cacheDir); err != nil {
		t.Fatalf("runRestore failed: %v", err)
	}

	// Verify restored contents and directory structure
	contentA, err := os.ReadFile(fileA)
	if err != nil {
		t.Fatalf("Failed to read restored file A: %v", err)
	}
	if string(contentA) != "content A" {
		t.Errorf("file A content = %q, want %q", string(contentA), "content A")
	}

	contentB, err := os.ReadFile(fileB)
	if err != nil {
		t.Fatalf("Failed to read restored file B: %v", err)
	}
	if string(contentB) != "content B" {
		t.Errorf("file B content = %q, want %q", string(contentB), "content B")
	}
}
