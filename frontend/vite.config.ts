import tailwindcss from "@tailwindcss/vite";
import { svelte } from "@sveltejs/vite-plugin-svelte";
import { defineConfig } from "vite";

const apiTarget = process.env.MEALSWAPP_VITE_API_TARGET ?? "http://127.0.0.1:8080";
let dropTask282ImportResponse = process.env.MEALSWAPP_TASK282_DROP_IMPORT_RESPONSE_ONCE === "1";
let corruptTask294ManualItemResponse = process.env.MEALSWAPP_TASK294_CORRUPT_MANUAL_ITEM_RESPONSE_ONCE === "1";

// Implements DESIGN-016 ComponentStyles Svelte and Tailwind build wiring.
export default defineConfig({
  plugins: [svelte(), tailwindcss()],
  // Implements DESIGN-001 SearchView dev proxy to ARCH-002 backend on :8080 so relative /api calls reach the API.
  server: {
    proxy: {
      "/api": {
        target: apiTarget,
        changeOrigin: true,
        selfHandleResponse: true,
        configure(proxy) {
          proxy.on("proxyRes", (proxyResponse, request, response) => {
            const isDroppedImport = dropTask282ImportResponse &&
              request.method === "POST" &&
              request.url?.split("?")[0] === "/api/v1/admin/imports";
            const isCorruptedManualItem = corruptTask294ManualItemResponse &&
              request.method === "POST" &&
              request.url?.split("?")[0] === "/api/v1/admin/items" &&
              proxyResponse.statusCode !== undefined &&
              proxyResponse.statusCode >= 200 && proxyResponse.statusCode < 300;

            if (isDroppedImport) dropTask282ImportResponse = false;
            if (isCorruptedManualItem) corruptTask294ManualItemResponse = false;

            const chunks: Buffer[] = [];
            proxyResponse.on("data", (chunk: Buffer) => chunks.push(chunk));
            proxyResponse.on("end", () => {
              if (isDroppedImport) {
                response.destroy();
                return;
              }
              const body = isCorruptedManualItem ? Buffer.from("{") : Buffer.concat(chunks);
              const headers = { ...proxyResponse.headers, "content-length": String(body.length) };
              delete headers["transfer-encoding"];
              response.writeHead(proxyResponse.statusCode ?? 502, headers);
              response.end(body);
            });
            proxyResponse.resume();
          });
        }
      }
    }
  },
  test: {
    globals: false
  }
});
