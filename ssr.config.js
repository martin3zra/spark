import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";

export default defineConfig({
  plugins: [react()],
  esbuild: { jsx: "automatic" },
  build: {
    ssr: true,
    outDir: "bootstrap/ssr",
    emptyOutDir: true,
    rollupOptions: {
      input: "resources/js/ssr.jsx",
      output: { entryFileNames: "ssr.js" },
    },
  },
});
