Double-click the EXE for your Windows processor and select **OK**.

- **x64** covers most Intel and AMD PCs.
- **ARM64** covers Windows PCs with an Arm processor.

Setup reuses Chromium or Chrome and Codex CLI when available. With your approval it uses WinGet for missing prerequisites. Start or restart local Codex and ask it to open https://example.com using `web_chromium_mcp`.

The browser runs in a separate temporary profile with 63 MCP tools. These unsigned setup binaries may trigger Windows SmartScreen. The attached checksum file contains SHA-256 hashes for both installers.

[Setup guide and build instructions](https://github.com/timmiballz22/Web-Chromium/tree/chromium-codex-mcp/tools/codex_mcp/windows)
