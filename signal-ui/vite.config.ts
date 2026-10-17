
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      "/api/investigations": {
        target: "http://localhost:8090",
        changeOrigin: true,
      },
      "/api/settings": {
        target: "http://localhost:8090",
        changeOrigin: true,
      },
      "/api/failure-scenario": {
        target: "http://localhost:8080",
        changeOrigin: true,
        rewrite: () => "/api/admin/failure-scenario",
      },
    },
  },
});
