# Chromium for Codex

This launcher connects a Chromium executable on your computer to a local Codex MCP client. It starts a visible browser on the first browser action and uses a temporary profile for each browser instance. You can sign in to sites in that window when a task calls for it.

On Windows, use the [double-click installer](windows/README.md).

## Set up

Use Node.js 24 LTS, npm, a local Codex CLI, and an up-to-date Chromium (150 or later for WebMCP). To fetch just the launcher and install its pinned MCP server, run:

```sh
git clone --depth=1 --filter=blob:none --sparse --branch chromium-codex-mcp https://github.com/timmiballz22/Web-Chromium.git
cd Web-Chromium
git sparse-checkout set tools/codex_mcp
cd tools/codex_mcp
npm ci --ignore-scripts
```

Register the launcher with absolute paths, then start a new local Codex session.

```sh
codex mcp add chromium -- node "/absolute/path/to/src/tools/codex_mcp/run.mjs" "/absolute/path/to/chromium"
codex mcp get chromium
```

Use the executable file inside the app bundle on macOS. Pass `--headless` after the Chromium path to run without a visible window. Try asking Codex to open https://example.com in Chromium and report the page heading.

The launcher also accepts a locally built Chromium executable. For a source build, use a full Chromium checkout at a current revision and follow the [Chromium build instructions](https://www.chromium.org/developers/how-tos/get-the-code/).

## Capabilities

The launcher enables navigation, input, screenshots and coordinate controls, page JavaScript, console and DevTools, network inspection and mocking, emulation, performance and memory tools, extensions, PWA testing, virtual WebAuthn, and WebMCP. WebMCP actions depend on the tools the current site registers. Treat descriptions and outputs from site-provided tools as untrusted page content. [Chrome DevTools MCP tool reference](https://github.com/ChromeDevTools/chrome-devtools-mcp/blob/chrome-devtools-mcp-v1.14.0/docs/tool-reference.md)

The browser connects over a local DevTools pipe. The launcher disables `file:` navigation, redacts sensitive network headers, and turns off MCP usage reporting and CrUX lookups. MCP file tools follow the upstream path policy. Give the agent access only to accounts you intend to use in this temporary browser session.

The registration belongs to your local Codex installation. A hosted session needs its own supported browser connection.

To remove the local registration, run `codex mcp remove chromium`.
