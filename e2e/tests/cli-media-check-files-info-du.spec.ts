import { test, expect } from '../fixtures-cli';
import * as fs from 'fs';
import * as path from 'path';

test.describe('CLI: Media Check and Info Commands', () => {
  test('files-info shows information about added files', async ({ cli, testDbPath, createValidVideo }) => {
    const videoPath = createValidVideo('info_test.mp4');
    await cli.runAndVerify(['add', testDbPath, videoPath]);

    const result = await cli.runAndVerify(['files-info', testDbPath, '--search', 'info_test']);
    expect(result.stdout).toContain('info_test.mp4');
  });

  test('disk-usage shows aggregation', async ({ cli, tempDir, testDbPath, createValidVideo }) => {
    const subDir = path.join(tempDir, 'du_dir');
    fs.mkdirSync(subDir);
    const videoPath = path.join(subDir, 'v1.mp4');
    // Using a valid video here ensures it's detected and has a real size
    const fixturePath = path.join(__dirname, '../fixtures/media/videos/test_video1.mp4');
    fs.copyFileSync(fixturePath, videoPath);
    
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    const result = await cli.runAndVerify(['disk-usage', testDbPath]);
    expect(result.stdout).toContain('du_dir');
  });

  test('media-check runs on files', async ({ cli, testDbPath, createValidVideo }) => {
    const videoPath = createValidVideo('check_test.mp4');
    await cli.runAndVerify(['add', testDbPath, videoPath]);

    const result = await cli.runAndVerify(['media-check', testDbPath, '--search', 'check_test']);
    expect(result.stdout).toContain('check_test.mp4');
    expect(result.stdout).toMatch(/\d+\.\d+%/);
  });
});

test.describe('CLI: Disk Usage Deep Dive', () => {
  test('shows disk usage by depth', async ({ cli, tempDir, testDbPath, createValidVideo }) => {
    const d1 = path.join(tempDir, 'depth1');
    const d2 = path.join(d1, 'depth2');
    fs.mkdirSync(d1);
    fs.mkdirSync(d2);
    const videoPath = path.join(d2, 'f.mp4');
    const fixturePath = path.join(__dirname, '../fixtures/media/videos/test_video1.mp4');
    fs.copyFileSync(fixturePath, videoPath);
    
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    const result = await cli.runAndVerify(['disk-usage', '-D', '10', testDbPath]);
    expect(result.stdout).toContain('depth1');
    expect(result.stdout).toContain('depth2');
  });

  test('groups by extension', async ({ cli, tempDir, testDbPath, createValidVideo, createValidAudio }) => {
    createValidVideo('v.mp4');
    createValidAudio('a.mp3');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    const result = await cli.runAndVerify(['disk-usage', '--group-by-extensions', testDbPath]);
    expect(result.stdout).toContain('.mp4');
    expect(result.stdout).toContain('.mp3');
  });
});
