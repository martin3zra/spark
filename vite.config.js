import { fileURLToPath, URL } from "node:url";

import tailwindcss from "@tailwindcss/vite";
import react from "@vitejs/plugin-react";
import laravel from "laravel-vite-plugin";
import { defineConfig } from "vite";

export default defineConfig({
  server: {
    host: "localhost",
    // Stable origin so the dev "hot" URL is browser-reachable (not [::]).
    hmr: { host: "localhost" },
  },
  plugins: [
    laravel({
      input: ["resources/js/app.jsx", "resources/css/app.css"],
      publicDirectory: "public",
      buildDirectory: "build",
      refresh: true,
    }),
    react(),
    tailwindcss(),
  ],
  esbuild: { jsx: "automatic" },
  resolve: {
    alias: {
      "@": fileURLToPath(new URL("./resources/js", import.meta.url)),
    },
  },
});
