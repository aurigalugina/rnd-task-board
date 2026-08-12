import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

export default defineConfig({
  plugins: [sveltekit()],
  server: {
    host: true,
    proxy: {
      // Default localhost:8080 buat dev lokal biasa (npm run dev di host).
      // Di docker-compose.dev.yml, frontend jalan di container sendiri jadi
      // "localhost" bukan backend container -- di-override via env ke
      // "http://backend:8080" (nama service di docker network).
      // ws:true supaya WebSocket chat (/api/v1/chat/ws/...) ikut diproksikan ke
      // backend, bukan cuma request HTTP biasa (Vision §6 chat streaming).
      '/api': {
        target: process.env.VITE_API_PROXY_TARGET ?? 'http://localhost:8080',
        ws: true,
        changeOrigin: true
      }
    }
  }
});
