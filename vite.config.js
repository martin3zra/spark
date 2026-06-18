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
      input: ["resources/js/app.jsx"],
      publicDirectory: "public",
      buildDirectory: "build",
      refresh: true,
    }),
    react(),
  ],
  esbuild: { jsx: "automatic" },
});
