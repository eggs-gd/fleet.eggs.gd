import { svelte } from '@sveltejs/vite-plugin-svelte';
import { defineConfig } from 'vite';

export default defineConfig({
  plugins: [svelte()],
  server: {
    port: 5174,
    strictPort: false,
    proxy: {
      '/api': 'http://127.0.0.1:8787'
    }
  }
});
