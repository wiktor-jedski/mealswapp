import tailwindcss from "@tailwindcss/vite";
import { svelte } from "@sveltejs/vite-plugin-svelte";
import { defineConfig } from "vite";

const apiTarget = process.env.MEALSWAPP_VITE_API_TARGET ?? "http://127.0.0.1:8080";
let dropTask282ImportResponse = process.env.MEALSWAPP_TASK282_DROP_IMPORT_RESPONSE_ONCE === "1";

// Implements DESIGN-016 ComponentStyles Svelte and Tailwind build wiring.
export default defineConfig({
  plugins: [svelte(), tailwindcss()],
  // Implements DESIGN-001 SearchView dev proxy to ARCH-002 backend on :8080 so relative /api calls reach the API.
  server: {
    proxy: {
      "/api": {
        target: apiTarget,
        changeOrigin: true,
        configure(proxy) {
          proxy.on("proxyRes", (_proxyResponse, request, response) => {
            if (
              dropTask282ImportResponse &&
              request.method === "POST" &&
              request.url?.split("?")[0] === "/api/v1/admin/imports"
            ) {
              dropTask282ImportResponse = false;
              response.destroy();
            }
          });
        }
      }
    }
  },
  test: {
    globals: false
  }
});
