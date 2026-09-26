import { readFileSync } from 'node:fs';
import { homedir } from 'node:os';
import { join } from 'node:path';
import { svelte } from '@sveltejs/vite-plugin-svelte';
import { defineConfig } from 'vite';

// `core serve` guards /api behind a Host check and a per-install token kept in
// ~/.fleet/launch-token. In dev, Vite serves index.html itself (no injected
// token), so the proxy adds the header and rewrites Host to the target.
function launchToken() {
  try {
    return readFileSync(join(homedir(), '.fleet', 'launch-token'), 'utf8').trim();
  } catch {
    return '';
  }
}

export default defineConfig({
  plugins: [svelte()],
  server: {
    port: 5174,
    strictPort: false,
    proxy: {
      '/api': {
        target: 'http://127.0.0.1:8787',
        changeOrigin: true,
        headers: launchToken() ? { Authorization: `Bearer ${launchToken()}` } : {}
      }
    }
  }
});
