import { fileURLToPath, URL } from "node:url";

import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      "@": fileURLToPath(new URL("./resources/js", import.meta.url)),
    },
  },
  build: {
    ssr: true,
    outDir: "bootstrap/ssr",
    emptyOutDir: true,
    rollupOptions: {
      input: "resources/js/ssr.tsx",
      output: { entryFileNames: "ssr.js" },
    },
  },
});
