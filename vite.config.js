import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  build: {
    manifest: "manifest.json",
    outDir: "public/build",
    emptyOutDir: true,
    rollupOptions: {
      input: "resources/js/app.jsx",
    },
  },
});
