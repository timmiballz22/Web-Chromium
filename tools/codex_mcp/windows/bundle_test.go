package main

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func testArchive(t *testing.T, names ...string) []byte {
	t.Helper()
	var data bytes.Buffer
	writer := zip.NewWriter(&data)
	for _, name := range names {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write([]byte("fixture")); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return data.Bytes()
}

func TestInstallBundleRerun(t *testing.T) {
	root := t.TempDir()
	payload := testArchive(t, "node.exe", "run.mjs", "node_modules/chrome-devtools-mcp/out/Default/gen/src/bin/chrome-devtools-mcp.js")
	target, err := installBundle(root, "1.0.0", payload)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(target, "run.mjs")
	if data, err := os.ReadFile(path); err != nil || string(data) != "fixture" {
		t.Fatalf("installed launcher = %q, %v", data, err)
	}
	if again, err := installBundle(root, "1.0.0", payload); err != nil || again != target {
		t.Fatalf("rerun = %q, %v", again, err)
	}
}

func TestInstallBundleRejectsEscapingEntry(t *testing.T) {
	root := t.TempDir()
	if _, err := installBundle(root, "1.0.0", testArchive(t, "run.mjs", "../escape")); err == nil {
		t.Fatal("expected archive path validation to fail")
	}
	if entries, err := os.ReadDir(root); err != nil || len(entries) != 0 {
		t.Fatalf("failed setup published files: %v, %v", entries, err)
	}
}
