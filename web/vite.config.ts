import tailwindcss from "@tailwindcss/vite";
import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";

const backend = process.env.UIV_BACKEND ?? "http://localhost:8080";

export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    proxy: {
      // Keep the browser's Host header so the backend derives links from the dev address.
      "/api": { target: backend, changeOrigin: false },
      "/f": { target: backend, changeOrigin: false },
    },
  },
});
