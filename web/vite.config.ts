import { fileURLToPath, URL } from "node:url";

import tailwindcss from "@tailwindcss/vite";
import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";

const srcPath = fileURLToPath(new URL("./src", import.meta.url));

export default defineConfig({
  plugins: [react(), tailwindcss()],
  publicDir: "src/public",
  resolve: {
    alias: {
      "@": srcPath,
    },
  },
  server: {
    port: 2233,
    proxy: {
      "/admin/api": {
        target: "http://localhost:2232",
        changeOrigin: true,
      },
    },
  },
});
