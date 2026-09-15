import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// envPrefix exposes GREETING_* and VITE_* vars from the build environment to
// import.meta.env, so GREETING_TAG is baked at build time with no VITE_ copy.
export default defineConfig({
  plugins: [react()],
  root: "client",
  build: {
    outDir: "../dist",
    emptyOutDir: true,
  },
  envPrefix: ["GREETING_", "VITE_"],
});
