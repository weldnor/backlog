import { defineConfig } from "vitest/config";
import react from "@vitejs/plugin-react";

// The build output lands directly in internal/browse/web/, the directory the
// Go binary embeds with `go:embed all:web`. emptyOutDir wipes that directory
// on every build, so nothing may be hand-placed there: the vendored Poppins
// woff2 files come from @fontsource/poppins through style.css and are emitted
// as hashed assets. base: "./" keeps
// every asset reference relative so the page works when served from "/".
export default defineConfig({
  plugins: [react()],
  base: "./",
  // `npm run dev` talks to a running `backlog browse` (default port 4173)
  // for the API; override with BACKLOG_API=http://host:port.
  server: {
    proxy: {
      "/api": process.env.BACKLOG_API ?? "http://127.0.0.1:4173",
    },
  },
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
