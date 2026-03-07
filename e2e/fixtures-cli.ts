import { test as base, expect } from '@playwright/test';
import { CLIRunner, createTempDir, cleanupTempDir } from './utils/cli-runner';
import * as path from 'path';
import * as fs from 'fs';

// Extended test fixture with CLI runner
export const test = base.extend<{
  cli: CLIRunner;
  tempDir: string;
  testDbPath: string;
  createDummyFile: (name: string, size?: number) => string;
  createDummyVideo: (name: string) => string;
  createDummyAudio: (name: string) => string;
  createDummyImage: (name: string) => string;
  createDummyDocument: (name: string) => string;
  createDummyVtt: (name: string, content?: string) => string;
}>({
  // CLI runner instance
  cli: async ({}, use) => {
    const binaryPath = process.env.DISCO_BINARY || path.join(__dirname, '../disco');
    const cli = new CLIRunner({ binaryPath });
    await use(cli);
  },

  // Temporary directory for test files
  tempDir: async ({}, use) => {
    const dir = createTempDir();
    await use(dir);
    cleanupTempDir(dir);
  },

  // Test database path (created in temp dir)
  testDbPath: async ({ tempDir }, use) => {
    const dbPath = path.join(tempDir, 'test.db');
    await use(dbPath);
  },

  // Helper to create dummy files
  createDummyFile: async ({ tempDir }, use) => {
    const createFile = (name: string, size: number = 1024): string => {
      const filePath = path.join(tempDir, name);
      const buffer = Buffer.alloc(size, Math.random() * 255);
      fs.writeFileSync(filePath, buffer);
      return filePath;
    };
    await use(createFile);
  },

  // Helper to create dummy video files
  createDummyVideo: async ({ tempDir }, use) => {
    const createVideo = (name: string): string => {
      const filePath = path.join(tempDir, name);
      // Minimal MP4 header (ftyp box)
      const header = Buffer.from([
        0x00, 0x00, 0x00, 0x20, // box size
        0x66, 0x74, 0x79, 0x70, // 'ftyp'
        0x69, 0x73, 0x6F, 0x6D, // 'isom'
        0x00, 0x00, 0x00, 0x01, // version
        0x69, 0x73, 0x6F, 0x6D, // 'isom'
        0x61, 0x76, 0x63, 0x31, // 'avc1'
        0x6D, 0x70, 0x34, 0x32, // 'mp42'
      ]);
      fs.writeFileSync(filePath, header);
      return filePath;
    };
    await use(createVideo);
  },

  // Helper to create dummy audio files
  createDummyAudio: async ({ tempDir }, use) => {
    const createAudio = (name: string): string => {
      const filePath = path.join(tempDir, name);
      // Minimal MP3 header (ID3 tag)
      const header = Buffer.from([
        0x49, 0x44, 0x33, // 'ID3'
        0x04, 0x00, // version
        0x00, // flags
        0x00, 0x00, 0x00, 0x00, // size
      ]);
      fs.writeFileSync(filePath, header);
      return filePath;
    };
    await use(createAudio);
  },

  // Helper to create dummy image files
  createDummyImage: async ({ tempDir }, use) => {
    const createImage = (name: string): string => {
      const filePath = path.join(tempDir, name);
      let header: Buffer;

      if (name.endsWith('.png')) {
        // PNG signature
        header = Buffer.from([
          0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A,
        ]);
      } else if (name.endsWith('.gif')) {
        // GIF signature
        header = Buffer.from([0x47, 0x49, 0x46, 0x38, 0x39, 0x61]);
      } else {
        // JPEG signature (default)
        header = Buffer.from([0xFF, 0xD8, 0xFF, 0xE0]);
      }

      fs.writeFileSync(filePath, header);
      return filePath;
    };
    await use(createImage);
  },

  // Helper to create dummy document files (PDF, EPUB)
  createDummyDocument: async ({ tempDir }, use) => {
    const createDocument = (name: string): string => {
      const filePath = path.join(tempDir, name);
      let header: Buffer;

      if (name.endsWith('.pdf')) {
        // PDF signature
        header = Buffer.from([0x25, 0x50, 0x44, 0x46, 0x2D]); // '%PDF-'
      } else if (name.endsWith('.epub')) {
        // EPUB is a ZIP file
        header = Buffer.from([0x50, 0x4B, 0x03, 0x04]); // PK
      } else {
        // Generic text
        header = Buffer.from('Dummy document content');
      }

      fs.writeFileSync(filePath, header);
      return filePath;
    };
    await use(createDocument);
  },

  // Helper to create dummy VTT subtitle files
  createDummyVtt: async ({ tempDir }, use) => {
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
