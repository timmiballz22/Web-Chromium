# Web-Chromium MCP Setup for Windows

Download the setup EXE for your processor, double-click it, and select **OK**. Most Intel and AMD PCs use **x64**. Choose **ARM64** for Windows on Arm. Windows shows the processor type under **Settings → System → About**.

Setup reuses a compatible Chromium or Chrome installation and an existing Codex CLI. If either is missing, it offers to install the official packages through WinGet. The installer itself is unsigned and Windows may show a SmartScreen prompt.

Start or restart **local Codex** on the same Windows PC. Ask it to open `https://example.com` with `web_chromium_mcp` and report the heading. Codex may ask you to sign in on first use. The browser opens when Codex first invokes a browser action. Sign in to websites in that browser if your task requires it.

A Chromium build in a custom directory can be selected with the optional `--browser` argument.

```powershell
.\Web-Chromium-MCP-Setup-x64.exe --browser "C:\path\to\chrome.exe"
```

## What setup installs

- A native Windows x64 or ARM64 setup program deploys the pinned Node.js runtime, license files, and Chrome DevTools MCP 1.14.0 to a versioned directory under `%LOCALAPPDATA%\WebChromiumMCP`. Setup needs no global Node.js, npm, development checkout, or PATH changes of its own.
- The setup finds Chromium first, then Chrome. It checks the executable's Windows version resource for version 150 or later. On a fresh PC the fallback is the regular Google Chrome stable package, which can receive browser security updates. WinGet validates the publisher package hashes. It can install the Codex CLI and its package dependencies when the CLI is missing.
- Setup runs `codex mcp add` for the current Windows user and reads the registration back. Codex starts the bundled `node.exe` directly over stdio. This avoids the Windows shell and npm shim at runtime. The MCP server launches the browser through a local DevTools pipe into a temporary profile.
- The launcher enables the 63 tools in its configured tool set, including coordinate input, browser JavaScript, network inspection and mocking, screenshots, performance and memory, extension and PWA tools, virtual WebAuthn, and WebMCP. Website-provided WebMCP tools vary by page. Codex and browser policies still apply, and the launcher uses the separate profile's sessions. This local configuration does not attach a hosted ChatGPT conversation to your PC.

## Troubleshooting and removal

If setup needs WinGet and it is unavailable, install or update Microsoft's **App Installer**, then rerun setup. You can also install Chromium or Chrome and the Codex CLI yourself before rerunning the same EXE.

To disconnect the browser, run `codex mcp remove web_chromium_mcp` and restart Codex. To remove the bundled runtime, close any browser sessions started through it and delete `%LOCALAPPDATA%\WebChromiumMCP`. Browser and Codex installations are managed by Windows separately.

## Build

Use Go 1.26 or later, Python 3.11 or later, and Node.js with npm. From this directory run `python build.py`. The build uses the launcher's npm lockfile and checks both Windows Node executables against pinned SHA-256 hashes. It cross-compiles the native setup binaries and writes them with their checksums to `dist/`.

Upstream references for the implementation include the [Codex MCP configuration](https://developers.openai.com/codex/mcp), [Chrome DevTools MCP tool reference](https://github.com/ChromeDevTools/chrome-devtools-mcp/blob/chrome-devtools-mcp-v1.14.0/docs/tool-reference.md), [Chrome DevTools MCP Windows troubleshooting](https://github.com/ChromeDevTools/chrome-devtools-mcp/blob/chrome-devtools-mcp-v1.14.0/docs/troubleshooting.md#windows-10-error-during-discovery-for-mcp-server-chrome-devtools-mcp-error--32000-connection-closed), [WinGet install command](https://learn.microsoft.com/windows/package-manager/winget/install), and [Chromium Windows build prerequisites](https://chromium.googlesource.com/chromium/src/+/main/docs/windows_build_instructions.md).
