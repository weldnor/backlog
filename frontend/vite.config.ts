import { defineConfig } from "vitest/config";
import react from "@vitejs/plugin-react";

// The build output lands directly in internal/browse/web/, the directory the
// Go binary embeds with `go:embed all:web`. emptyOutDir wipes that directory
// on every build, which is why the vendored Archivo font lives under src/
// (imported by style.css) rather than only in the output. base: "./" keeps
// every asset reference relative so the page works when served from "/".
export default defineConfig({
  plugins: [react()],
  base: "./",
  build: {
    outDir: "../internal/browse/web",
    emptyOutDir: true,
  },
  test: {
    environment: "jsdom",
    globals: true,
    setupFiles: "./src/test-setup.ts",
  },
});
