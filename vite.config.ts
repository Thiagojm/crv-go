import { defineConfig } from 'vite';
import { svelte } from '@sveltejs/vite-plugin-svelte';
import tailwindcss from '@tailwindcss/vite';
import { viteSingleFile } from 'vite-plugin-singlefile';
// Single-file dist/ is embedded by main.go. Do not import farsight/ into the bundle.
export default defineConfig({
  plugins: [svelte(), tailwindcss(), viteSingleFile()],
  build: { assetsInlineLimit: 2000000, outDir: 'dist', emptyOutDir: true },
  server: { host: '127.0.0.1' },
});
