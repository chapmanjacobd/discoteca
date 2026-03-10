import { test, expect } from '../fixtures-cli';
import * as fs from 'fs';
import * as path from 'path';

test.describe('CLI: Categorize Command', () => {
  test('auto-groups media into categories', async ({ cli, tempDir, testDbPath, createValidVideo, createValidAudio }) => {
    createValidVideo('movie.mp4');
    createValidAudio('music.mp3');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    await cli.runAndVerify(['categorize', testDbPath]);

    const queryResult = await cli.runJson<any[]>(['print', testDbPath, '--all', '--json']);
    expect(Array.isArray(queryResult)).toBe(true);
  });

  test('categorizes with no default categories', async ({ cli, tempDir, testDbPath, createValidVideo }) => {
    createValidVideo('video.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    const result = await cli.runAndVerify(['categorize', '--no-default-categories', testDbPath]);
    expect(result.exitCode).toBe(0);
  });
});

test.describe('CLI: Similar-Files Command', () => {
  test('finds similar files', async ({ cli, tempDir, testDbPath, createValidVideo }) => {
    createValidVideo('movie_part1.mp4');
    createValidVideo('movie_part2.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    const result = await cli.runAndVerify(['similar-files', testDbPath]);
    expect(result.stdout).toContain('movie_part1');
    expect(result.stdout).toContain('movie_part2');
  });

  test('finds similar files as JSON', async ({ cli, tempDir, testDbPath, createValidVideo }) => {
    createValidVideo('similar1.mp4');
    createValidVideo('similar2.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    const result = await cli.runJson<any[]>(['similar-files', '-j', testDbPath]);
    expect(result.length).toBeGreaterThan(0);
  });
});

test.describe('CLI: Similar-Folders Command', () => {
  test('finds similar folders', async ({ cli, tempDir, testDbPath, createValidVideo }) => {
    const dir1 = path.join(tempDir, 'movies_action');
    const dir2 = path.join(tempDir, 'movies_comedy');
    fs.mkdirSync(dir1);
    fs.mkdirSync(dir2);
    
    // Use helpers to create real media files in subdirs
    const v1Path = path.join(dir1, 'v1.mp4');
    const v2Path = path.join(dir2, 'v2.mp4');
    
    // We need to manually copy because the helper uses tempDir by default
    // Let's improve the helper or just copy manually from the known fixture path
    const fixturePath = path.join(__dirname, '../fixtures/media/videos/test_video1.mp4');
    fs.copyFileSync(fixturePath, v1Path);
    fs.copyFileSync(fixturePath, v2Path);
    
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    const result = await cli.runAndVerify(['similar-folders', testDbPath]);
    expect(result.stdout).toBeTruthy();
  });
});

test.describe('CLI: Dedupe Command', () => {
  test('deduplicates similar media (simulate)', async ({ cli, tempDir, testDbPath, createValidVideo }) => {
    const dir1 = path.join(tempDir, 'dir1');
    const dir2 = path.join(tempDir, 'dir2');
    fs.mkdirSync(dir1);
    fs.mkdirSync(dir2);
    
    const f1Path = path.join(dir1, 'movie.mp4');
    const f2Path = path.join(dir2, 'movie.mp4');
    const fixturePath = path.join(__dirname, '../fixtures/media/videos/test_video1.mp4');
    fs.copyFileSync(fixturePath, f1Path);
    fs.copyFileSync(fixturePath, f2Path);
    
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    const result = await cli.runAndVerify(['dedupe', '--filesystem', '-y', '--simulate', testDbPath]);
    expect(result.stdout).toContain('movie.mp4');
  });
});

test.describe('CLI: Big-Dirs Command', () => {
  test('shows big directories', async ({ cli, tempDir, testDbPath, createValidVideo }) => {
    const subDir = path.join(tempDir, 'big_dir');
    fs.mkdirSync(subDir);
    createValidVideo('big_dir/v1.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    const result = await cli.runAndVerify(['big-dirs', testDbPath]);
    expect(result.stdout).toContain('big_dir');
  });

  test('shows big directories sorted by size', async ({ cli, tempDir, testDbPath, createValidVideo }) => {
    const smallDir = path.join(tempDir, 'small');
    const largeDir = path.join(tempDir, 'large');
    fs.mkdirSync(smallDir);
    fs.mkdirSync(largeDir);
    
    // Use real media files
    const fixturePath = path.join(__dirname, '../fixtures/media/videos/test_video1.mp4');
    fs.copyFileSync(fixturePath, path.join(smallDir, 'f.mp4'));
    // For large one, we can just copy it multiple times or use a different fixture if available
    // But even same file in different dirs is fine for testing big-dirs
    fs.copyFileSync(fixturePath, path.join(largeDir, 'f.mp4'));
    
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    const result = await cli.runAndVerify(['big-dirs', '--sort-by', 'size', testDbPath]);
    expect(result.stdout).toBeTruthy();
  });
});
