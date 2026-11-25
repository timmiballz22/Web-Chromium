// Launch the pinned DevTools MCP CLI in this process so stdio and signals
// go directly between the MCP client and server.
import { accessSync, constants, statSync } from 'node:fs';
import { isAbsolute } from 'node:path';
import { fileURLToPath } from 'node:url';

const [executable, mode, ...extra] = process.argv.slice(2);
if (!executable || !isAbsolute(executable) ||
    (mode !== undefined && mode !== '--headless') || extra.length > 0) {
  console.error('Usage: node run.mjs <absolute-path-to-chromium> [--headless]');
  process.exit(2);
}

try {
  if (!statSync(executable).isFile()) {
    throw new Error('Expected a browser executable file');
  }
  accessSync(executable, constants.X_OK);
} catch (error) {
  console.error(`Cannot use the Chromium executable: ${error.message}`);
  process.exit(2);
}

const cli = new URL(
  './node_modules/chrome-devtools-mcp/out/Default/gen/src/bin/chrome-devtools-mcp.js',
  import.meta.url,
);
const options = [
  `--executable-path=${executable}`,
  '--isolated',
  '--no-usage-statistics',
  '--no-performance-crux',
  '--no-file-navigations',
  '--redact-network-headers',
  '--category-network-mocks',
  '--category-extensions',
  '--category-experimental-third-party',
  '--category-experimental-webmcp',
  '--chrome-arg=--enable-features=WebMCP',
  '--category-pwa',
  '--category-webauthn',
  '--experimental-devtools',
  '--experimental-vision',
  '--memory-debugging',
];
if (mode) {
  options.push(mode);
}

process.argv.splice(1, process.argv.length - 1, fileURLToPath(cli), ...options);
await import(cli.href);
