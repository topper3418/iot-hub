// Directory: frontend/
// Modified: 2026-04-08
// Description: Vite build configuration. Enables the React plugin and proxies /api to the Go backend in dev mode.
// Uses: none
// Used by: npm run build (scripts/build.sh)

import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      '/api': 'http://localhost:8080'
    }
  }
});
