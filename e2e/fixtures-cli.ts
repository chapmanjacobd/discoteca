import { test as base, expect } from '@playwright/test';
import { CLIRunner, createTempDir, cleanupTempDir } from './utils/cli-runner';
import * as path from 'path';
import * as fs from 'fs';

// Extended test fixture with CLI runner
export const test = base.extend<{
  cli: CLIRunner;
  tempDir: string;
  testDbPath: string;
  createValidVideo: (name: string) => string;
  createValidAudio: (name: string) => string;
  createValidImage: (name: string) => string;
  createValidDocument: (name: string) => string;
  createValidVtt: (name: string, content?: string) => string;
}>({
  // CLI runner instance
  cli: async ({}, use) => {
    const binaryPath = process.env.DISCO_BINARY || path.join(__dirname, '../disco');
    const verbose = process.env.VERBOSE === 'true' || process.env.DEBUG === 'true';
    const cli = new CLIRunner({ binaryPath, verbose });
    await use(cli);
  },

  // Temporary directory for test files (unique per test)
  tempDir: async ({}, use) => {
    const dir = createTempDir();
    await use(dir);
    cleanupTempDir(dir);
  },

  // Test database path (created in separate temp dir to avoid scanning it)
  testDbPath: async ({}, use) => {
    const dbDir = createTempDir();
    const dbPath = path.join(dbDir, 'test.db');
    await use(dbPath);
    cleanupTempDir(dbDir);
  },

  // Helper to create a valid video file by copying from fixtures
  createValidVideo: async ({ tempDir }, use) => {
    const createVideo = (name: string): string => {
      const fixturePath = path.join(__dirname, 'fixtures/media/videos/test_video1.mp4');
      const targetPath = path.join(tempDir, name);
      if (!fs.existsSync(fixturePath)) {
        throw new Error(`Required fixture missing: ${fixturePath}. Run 'make e2e-init' to generate real media files.`);
      }
      fs.copyFileSync(fixturePath, targetPath);
      return targetPath;
    };
    await use(createVideo);
  },

  // Helper to create a valid audio file by copying from fixtures
  createValidAudio: async ({ tempDir }, use) => {
    const createAudio = (name: string): string => {
      const fixturePath = path.join(__dirname, 'fixtures/media/audio/test_audio1.mp3');
      const targetPath = path.join(tempDir, name);
      if (!fs.existsSync(fixturePath)) {
        throw new Error(`Required fixture missing: ${fixturePath}. Run 'make e2e-init' to generate real media files.`);
      }
      fs.copyFileSync(fixturePath, targetPath);
      return targetPath;
    };
    await use(createAudio);
  },

  // Helper to create a valid image file by copying from fixtures
  createValidImage: async ({ tempDir }, use) => {
    const createImage = (name: string): string => {
      const fixturePath = path.join(__dirname, 'fixtures/media/images/test_image1.png');
      const targetPath = path.join(tempDir, name);
      if (!fs.existsSync(fixturePath)) {
        throw new Error(`Required fixture missing: ${fixturePath}. Run 'make e2e-init' to generate real media files.`);
      }
      fs.copyFileSync(fixturePath, targetPath);
      return targetPath;
    };
    await use(createImage);
  },

  // Helper to create a valid document file (PDF, EPUB) by copying from fixtures
  createValidDocument: async ({ tempDir }, use) => {
    const createDocument = (name: string): string => {
      const ext = path.extname(name).toLowerCase();
      let fixtureName = 'test-document.pdf';
      if (ext === '.epub') fixtureName = 'test-book.epub';

      const fixturePath = path.join(__dirname, 'fixtures/media/documents', fixtureName);
      const targetPath = path.join(tempDir, name);
      if (!fs.existsSync(fixturePath)) {
        throw new Error(`Required fixture missing: ${fixturePath}. Run 'make e2e-init' to generate real media files.`);
      }
      fs.copyFileSync(fixturePath, targetPath);
      return targetPath;
    };
    await use(createDocument);
  },

  // Helper to create valid VTT subtitle files
  createValidVtt: async ({ tempDir }, use) => {
    const createVtt = (name: string, content?: string): string => {
      const filePath = path.join(tempDir, name);
      const vttContent = content || `WEBVTT

00:00:01.000 --> 00:00:03.000
Sample subtitle line 1

00:00:04.000 --> 00:00:06.000
Sample subtitle line 2
`;
      fs.writeFileSync(filePath, vttContent, 'utf-8');
      return filePath;
    };
    await use(createVtt);
  },
});

export { expect };
