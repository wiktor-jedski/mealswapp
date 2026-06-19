import tailwindcss from "@tailwindcss/vite";
import { svelte } from "@sveltejs/vite-plugin-svelte";
import { defineConfig } from "vite";

// Implements DESIGN-016 ComponentStyles Svelte and Tailwind build wiring.
export default defineConfig({
  plugins: [svelte(), tailwindcss()],
  // Implements DESIGN-001 SearchView dev proxy to ARCH-002 backend on :8080 so relative /api calls reach the API.
  server: {
    proxy: {
      "/api": {
        target: "http://127.0.0.1:8080",
        changeOrigin: true
      }
    }
  },
  test: {
    globals: false
  }
});
