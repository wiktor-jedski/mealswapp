import tailwindcss from "@tailwindcss/vite";
import { svelte } from "@sveltejs/vite-plugin-svelte";
import { defineConfig } from "vite";

// Implements DESIGN-016 ComponentStyles Svelte and Tailwind build wiring.
export default defineConfig({
  plugins: [svelte(), tailwindcss()],
  test: {
    globals: false
  }
});
