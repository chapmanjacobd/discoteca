import { defineConfig } from 'vite';

export default defineConfig({
  // Relative asset URLs keep the build usable when the app is mounted below a
  // reverse-proxy prefix such as /disco.
  base: './',
  build: {
    outDir: 'dist',
    emptyOutDir: true,
    target: 'esnext',
    assetsInlineLimit: 0,
    chunkSizeWarningLimit: 1000,
    rollupOptions: {
      output: {
        entryFileNames: `[name].js`,
        chunkFileNames: `[name].js`,
        assetFileNames: `[name].[ext]`
      }
    }
  },
  server: {
    port: 5173
  }
});
