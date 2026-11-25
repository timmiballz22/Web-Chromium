//go:build windows

package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"syscall"
	"time"
)

//go:embed runtime.zip
var payload []byte

const (
	setupVersion    = "1.0.0"
	serverName      = "web_chromium_mcp"
	minBrowserMajor = 150
)

type setup struct {
	root, local, browser, codex string
}

func main() {
	runtime.LockOSThread()
	if err := run(); err != nil {
		dialog("Setup could not finish.\r\n\r\n"+err.Error(), 0x10)
		os.Exit(1)
	}
}

func run() error {
	flags := flag.NewFlagSet("setup", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	browserOverride := flags.String("browser", "", "absolute path to Chromium or Chrome")
	if err := flags.Parse(os.Args[1:]); err != nil || flags.NArg() != 0 {
		return errors.New("Double-click Setup, or use --browser followed by the full path to Chromium.")
	}
	local := os.Getenv("LOCALAPPDATA")
	if !filepath.IsAbs(local) {
		return errors.New("Windows did not provide a local application data directory.")
	}
	s := setup{root: filepath.Join(local, "WebChromiumMCP"), local: local, codex: findCodex(local)}
	if *browserOverride != "" {
		if !filepath.IsAbs(*browserOverride) || browserMajor(*browserOverride) < minBrowserMajor {
			return errors.New("Choose the full path to a Chromium or Chrome executable at version 150 or later.")
		}
		s.browser = filepath.Clean(*browserOverride)
	} else {
		s.browser = findBrowser(local)
	}

	prompt := "Connect a separate Chromium browser profile to local Codex?\r\n\r\nCodex will be able to see and operate pages in that profile. You can sign in there when needed."
	var downloads []string
	if s.browser == "" {
		downloads = append(downloads, "Google Chrome")
	}
	if s.codex == "" {
		downloads = append(downloads, "OpenAI Codex CLI")
	}
	if len(downloads) != 0 {
		prompt += "\r\n\r\nSetup will use Windows Package Manager to install " + strings.Join(downloads, " and ") + " for this Windows user."
	}
	// OK and Cancel, with Cancel as the default button.
	if dialog(prompt, 0x121) != 1 {
		return nil
	}
	if err := showProgress(s.install); err != nil {
		return err
	}
	dialog("Browser tools are ready in local Codex.\r\n\r\nStart or restart Codex on this PC and ask it to open https://example.com with web_chromium_mcp.", 0x40)
	return nil
}

func findBrowser(local string) string {
	roots := []string{local, os.Getenv("ProgramW6432"), os.Getenv("ProgramFiles"), os.Getenv("ProgramFiles(x86)")}
	for _, vendor := range []string{"Chromium", filepath.Join("Google", "Chrome")} {
		for _, root := range roots {
			if root == "" {
				continue
			}
			path := filepath.Join(root, vendor, "Application", "chrome.exe")
			if browserMajor(path) >= minBrowserMajor {
				return path
			}
		}
	}
	for _, name := range []string{"chromium.exe", "chrome.exe"} {
		if path, err := exec.LookPath(name); err == nil && browserMajor(path) >= minBrowserMajor {
			return path
		}
	}
	return ""
}

func findCodex(local string) string {
	if custom := os.Getenv("CODEX_INSTALL_DIR"); filepath.IsAbs(custom) {
		if path := filepath.Join(custom, "codex.exe"); fileExists(path) {
			return path
		}
	}
	for _, path := range []string{
		filepath.Join(local, "Programs", "OpenAI", "Codex", "bin", "codex.exe"),
		filepath.Join(local, "Microsoft", "WinGet", "Links", "codex.exe"),
		filepath.Join(local, "Microsoft", "WindowsApps", "codex.exe"),
	} {
		if fileExists(path) {
			return path
		}
	}
	path, _ := exec.LookPath("codex.exe")
	return path
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func (s setup) install(update func(string)) error {
	if s.browser == "" {
		update("Installing current Google Chrome through Windows Package Manager…")
		err := s.winget("Google.Chrome.EXE")
		s.browser = findBrowser(s.local)
		if s.browser == "" {
			return installError("Google Chrome did not become available.", err)
		}
	}
	if s.codex == "" {
		update("Installing OpenAI Codex CLI through Windows Package Manager…")
		err := s.winget("OpenAI.Codex")
		s.codex = findCodex(s.local)
		if s.codex == "" {
			return installError("OpenAI Codex CLI did not become available.", err)
		}
	}
	update("Installing the bundled browser tools…")
	dir, err := installBundle(s.root, setupVersion+"-"+runtime.GOARCH, payload)
	if err != nil {
		return fmt.Errorf("install the bundled browser tools: %w", err)
	}
	update("Registering browser tools with local Codex…")
	return s.register(filepath.Join(dir, "node.exe"), filepath.Join(dir, "run.mjs"))
}

func installError(message string, err error) error {
	if err == nil {
		return errors.New(message + " Install it for this Windows user, then rerun Setup.")
	}
	return fmt.Errorf("%s %w", message, err)
}

func (s setup) winget(id string) error {
	path := filepath.Join(s.local, "Microsoft", "WindowsApps", "winget.exe")
	if !fileExists(path) {
		path, _ = exec.LookPath("winget.exe")
	}
	if path == "" {
		return errors.New("Windows Package Manager was unavailable. Install or update App Installer from the Microsoft Store, then rerun Setup.")
	}
	out, err := command(15*time.Minute, path, "install", "--id", id, "--exact", "--source", "winget",
		"--scope", "user", "--silent", "--disable-interactivity", "--accept-source-agreements", "--accept-package-agreements")
	if err != nil {
		return fmt.Errorf("WinGet install for %s failed (%v).\r\n%s", id, err, tail(out))
	}
	return nil
}

type registration struct {
	Enabled   bool `json:"enabled"`
	Transport struct {
		Command string   `json:"command"`
		Args    []string `json:"args"`
	} `json:"transport"`
}

func (s setup) existingRegistration() (*registration, error) {
	out, err := command(30*time.Second, s.codex, "mcp", "get", "--json", serverName)
	if err != nil {
		if strings.Contains(string(out), "No MCP server named '"+serverName+"' found.") {
			return nil, nil
		}
		// Query output may include settings for an existing MCP server.
		return nil, fmt.Errorf("Codex could not read the browser registration (%v). Check that the Codex CLI runs for this Windows user.", err)
	}
	var entry registration
	if err := json.Unmarshal(out, &entry); err != nil {
		return nil, fmt.Errorf("read the Codex browser registration: %w", err)
	}
	return &entry, nil
}

func (s setup) register(node, launcher string) error {
	entry, err := s.existingRegistration()
	if err != nil {
		return err
	}
	args := []string{launcher, s.browser}
	if entry != nil {
		if strings.EqualFold(entry.Transport.Command, node) && slices.Equal(entry.Transport.Args, args) {
			if !entry.Enabled {
				return errors.New("The browser registration is disabled in Codex. Enable it in your local Codex settings and run Setup again.")
			}
			return nil
		}
		if !s.owns(entry) {
			return errors.New("Codex already has a different MCP server named " + serverName + ". Rename that entry before running Setup.")
		}
	}
	_, err = command(30*time.Second, s.codex, "mcp", "add", serverName, "--", node, launcher, s.browser)
	if err != nil {
		return fmt.Errorf("Codex could not register the browser tools (%v). Check the local Codex configuration and rerun Setup.", err)
	}
	entry, err = s.existingRegistration()
	if err != nil {
		return err
	}
	if entry == nil || !entry.Enabled || !strings.EqualFold(entry.Transport.Command, node) || !slices.Equal(entry.Transport.Args, args) {
		return errors.New("Codex did not enable the expected browser registration. Check your local Codex settings and rerun Setup.")
	}
	return nil
}

func (s setup) owns(entry *registration) bool {
	if !entry.Enabled || len(entry.Transport.Args) != 2 {
		return false
	}
	dir := filepath.Dir(entry.Transport.Command)
	return strings.EqualFold(filepath.Dir(dir), s.root) &&
		strings.EqualFold(filepath.Base(entry.Transport.Command), "node.exe") &&
		strings.EqualFold(entry.Transport.Args[0], filepath.Join(dir, "run.mjs"))
}

func command(timeout time.Duration, path string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, path, args...)
	cmd.Dir = os.Getenv("LOCALAPPDATA")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	out, err := cmd.CombinedOutput()
	if ctx.Err() != nil {
		return out, ctx.Err()
	}
	return out, err
}

func tail(out []byte) string {
	lines := strings.FieldsFunc(string(out), func(r rune) bool { return r == '\r' || r == '\n' })
	if len(lines) > 4 {
		lines = lines[len(lines)-4:]
	}
	s := []rune(strings.Join(lines, "\r\n"))
	if len(s) > 700 {
		s = s[len(s)-700:]
	}
	return string(s)
}
