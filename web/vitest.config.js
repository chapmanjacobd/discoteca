import { defineConfig } from 'vitest/config';
import os from 'os';

export default defineConfig({
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: ['./tests/setup.js'],
    css: false,
    maxThreads: os.cpus().length,
    minThreads: Math.min(os.cpus().length, 4),
    sequence: {
      concurrent: false,
    },
    include: ['tests/**/*.test.js', 'src/**/*.test.ts'],
  },
});
