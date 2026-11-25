package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// installBundle publishes a complete, versioned runtime together with a
// content marker. An interrupted setup can be rerun without partial files.
func installBundle(root, version string, payload []byte) (string, error) {
	target := filepath.Join(root, version)
	marker := fmt.Sprintf("%x\n", sha256.Sum256(payload))
	if ok := installedBundle(target, marker); ok {
		return target, nil
	}
	if err := os.MkdirAll(root, 0700); err != nil {
		return "", err
	}
	staging, err := os.MkdirTemp(root, ".install-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(staging)

	archive, err := zip.NewReader(bytes.NewReader(payload), int64(len(payload)))
	if err != nil {
		return "", err
	}
	for _, entry := range archive.File {
		if err := extractFile(staging, entry); err != nil {
			return "", err
		}
	}
	if err := os.WriteFile(filepath.Join(staging, ".bundle-sha256"), []byte(marker), 0600); err != nil {
		return "", err
	}
	if err := os.Rename(staging, target); err != nil && !installedBundle(target, marker) {
		return "", fmt.Errorf("publish runtime at %s: %w", target, err)
	}
	return target, nil
}

func installedBundle(dir, marker string) bool {
	value, err := os.ReadFile(filepath.Join(dir, ".bundle-sha256"))
	if err != nil || string(value) != marker {
		return false
	}
	for _, name := range []string{"node.exe", "run.mjs", "node_modules/chrome-devtools-mcp/out/Default/gen/src/bin/chrome-devtools-mcp.js"} {
		if info, err := os.Stat(filepath.Join(dir, filepath.FromSlash(name))); err != nil || !info.Mode().IsRegular() {
			return false
		}
	}
	return true
}

func extractFile(dir string, entry *zip.File) error {
	if !filepath.IsLocal(filepath.FromSlash(entry.Name)) || strings.ContainsAny(entry.Name, ":\\") || !entry.Mode().IsRegular() {
		return fmt.Errorf("invalid runtime entry %q", entry.Name)
	}
	target := filepath.Join(dir, filepath.FromSlash(entry.Name))
	if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
		return err
	}
	input, err := entry.Open()
	if err != nil {
		return err
	}
	defer input.Close()
	output, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(output, input)
	closeErr := output.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}
