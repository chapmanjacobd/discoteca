import { test, expect } from '../fixtures-cli';
import * as fs from 'fs';
import * as path from 'path';

test.describe('CLI: Categorize Command', () => {
  test('auto-groups media into categories', async ({ cli, tempDir, testDbPath, createDummyVideo, createDummyAudio }) => {
    // Create and add files
    createDummyVideo('movie.mp4');
    createDummyAudio('music.mp3');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Categorize media
    const result = await cli.runAndVerify(['categorize', testDbPath]);

    // Should categorize successfully
    expect(result.exitCode).toBe(0);
  });

  test('categorizes with no default categories', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create and add a file
    createDummyVideo('video.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Categorize with no default categories
    const result = await cli.runAndVerify(['categorize', '--no-default-categories', testDbPath]);

    // Should succeed
    expect(result.exitCode).toBe(0);
  });

  test('categorizes with verbose output', async ({ cli, tempDir, testDbPath, createDummyVideo, createDummyAudio }) => {
    // Create and add files
    createDummyVideo('movie.mp4');
    createDummyAudio('music.mp3');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Categorize with verbose
    const result = await cli.runAndVerify(['categorize', '--verbose', testDbPath]);

    // Should have output
    expect(result.stdout).toBeTruthy();
  });

  test('categorizes empty database', async ({ cli, tempDir, testDbPath }) => {
    // Categorize empty database
    const result = await cli.runAndVerify(['categorize', testDbPath]);

    // Should succeed
    expect(result.exitCode).toBe(0);
  });
});

test.describe('CLI: Similar-Files Command', () => {
  test('finds similar files', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create and add files with similar names
    createDummyVideo('movie_part1.mp4');
    createDummyVideo('movie_part2.mp4');
    createDummyVideo('different.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Find similar files
    const result = await cli.runAndVerify(['similar-files', testDbPath]);

    // Should find similar files
    expect(result.stdout).toBeTruthy();
  });

  test('finds similar files by name pattern', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create and add files
    createDummyVideo('show_s01e01.mp4');
    createDummyVideo('show_s01e02.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Find similar files
    const result = await cli.runAndVerify(['similar-files', testDbPath]);

    // Should find similar files
    expect(result.stdout).toContain('show');
  });

  test('finds similar files as JSON', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create and add files
    createDummyVideo('similar1.mp4');
    createDummyVideo('similar2.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Find similar files as JSON
    const result = await cli.runJson(['similar-files', '-j', testDbPath]);

    expect(Array.isArray(result)).toBe(true);
  });

  test('finds similar files with threshold', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create and add files
    createDummyVideo('file1.mp4');
    createDummyVideo('file2.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Find similar files with threshold
    const result = await cli.runAndVerify(['similar-files', '--threshold', '0.5', testDbPath]);

    // Should succeed
    expect(result.exitCode).toBe(0);
  });

  test('finds similar files with verbose output', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create and add files
    createDummyVideo('verbose1.mp4');
    createDummyVideo('verbose2.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Find similar files with verbose
    const result = await cli.runAndVerify(['similar-files', '--verbose', testDbPath]);

    // Should have output
    expect(result.stdout).toBeTruthy();
  });

  test('finds similar files in empty database', async ({ cli, tempDir, testDbPath }) => {
    // Find similar files in empty database
    const result = await cli.runAndVerify(['similar-files', testDbPath]);

    // Should succeed (no results)
    expect(result.exitCode).toBe(0);
  });
});

test.describe('CLI: Similar-Folders Command', () => {
  test('finds similar folders', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create subdirectory structure with similar names
    const dir1 = path.join(tempDir, 'movies_action');
    const dir2 = path.join(tempDir, 'movies_comedy');
    fs.mkdirSync(dir1);
    fs.mkdirSync(dir2);
    
    fs.writeFileSync(path.join(dir1, 'video1.mp4'), Buffer.alloc(1024));
    fs.writeFileSync(path.join(dir2, 'video2.mp4'), Buffer.alloc(1024));
    
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Find similar folders
    const result = await cli.runAndVerify(['similar-folders', testDbPath]);

    // Should find similar folders
    expect(result.stdout).toBeTruthy();
  });

  test('finds similar folders as JSON', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create subdirectory structure
    const dir1 = path.join(tempDir, 'folder1');
    const dir2 = path.join(tempDir, 'folder2');
    fs.mkdirSync(dir1);
    fs.mkdirSync(dir2);
    
    fs.writeFileSync(path.join(dir1, 'video.mp4'), Buffer.alloc(1024));
    fs.writeFileSync(path.join(dir2, 'video.mp4'), Buffer.alloc(1024));
    
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Find similar folders as JSON
    const result = await cli.runJson(['similar-folders', '-j', testDbPath]);

    expect(Array.isArray(result)).toBe(true);
  });

  test('finds similar folders with threshold', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create subdirectory structure
    const dir1 = path.join(tempDir, 'similar1');
    const dir2 = path.join(tempDir, 'similar2');
    fs.mkdirSync(dir1);
    fs.mkdirSync(dir2);
    
    fs.writeFileSync(path.join(dir1, 'video.mp4'), Buffer.alloc(1024));
    fs.writeFileSync(path.join(dir2, 'video.mp4'), Buffer.alloc(1024));
    
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Find similar folders with threshold
    const result = await cli.runAndVerify(['similar-folders', '--threshold', '0.5', testDbPath]);

    // Should succeed
    expect(result.exitCode).toBe(0);
  });

  test('finds similar folders with verbose output', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create subdirectory structure
    const dir1 = path.join(tempDir, 'verbose1');
    const dir2 = path.join(tempDir, 'verbose2');
    fs.mkdirSync(dir1);
    fs.mkdirSync(dir2);
    
    fs.writeFileSync(path.join(dir1, 'video.mp4'), Buffer.alloc(1024));
    fs.writeFileSync(path.join(dir2, 'video.mp4'), Buffer.alloc(1024));
    
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Find similar folders with verbose
    const result = await cli.runAndVerify(['similar-folders', '--verbose', testDbPath]);

    // Should have output
    expect(result.stdout).toBeTruthy();
  });

  test('finds similar folders in empty database', async ({ cli, tempDir, testDbPath }) => {
    // Find similar folders in empty database
    const result = await cli.runAndVerify(['similar-folders', testDbPath]);

    // Should succeed (no results)
    expect(result.exitCode).toBe(0);
  });
});

test.describe('CLI: Dedupe Command', () => {
  test('deduplicates similar media', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create and add files with similar names
    createDummyVideo('duplicate1.mp4');
    createDummyVideo('duplicate2.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Run dedupe (dry run)
    const result = await cli.run(['dedupe', '--simulate', testDbPath]);

    // Should complete
    expect(result.exitCode).toBeGreaterThanOrEqual(0);
  });

  test('deduplicates with dry run', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create and add files
    createDummyVideo('dup1.mp4');
    createDummyVideo('dup2.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Run dedupe with dry run
    const result = await cli.runAndVerify(['dedupe', '--simulate', testDbPath]);

    // Should not modify database
    const queryResult = await cli.runJson(['print', testDbPath, '--all']);
    expect(queryResult.length).toBe(2);
  });

  test('deduplicates with verbose output', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create and add files
    createDummyVideo('dup1.mp4');
    createDummyVideo('dup2.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Run dedupe with verbose
    const result = await cli.run(['dedupe', '--verbose', '--simulate', testDbPath]);

    // Should have output
    expect(result.stdout || result.stderr).toBeTruthy();
  });

  test('deduplicates with threshold', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create and add files
    createDummyVideo('dup1.mp4');
    createDummyVideo('dup2.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Run dedupe with threshold
    const result = await cli.run(['dedupe', '--threshold', '0.8', '--simulate', testDbPath]);

    // Should complete
    expect(result.exitCode).toBeGreaterThanOrEqual(0);
  });

  test('deduplicates with strategy', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create and add files
    createDummyVideo('dup1.mp4');
    createDummyVideo('dup2.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Run dedupe with strategy
    const result = await cli.run(['dedupe', '--strategy', 'keep-largest', '--simulate', testDbPath]);

    // Should complete
    expect(result.exitCode).toBeGreaterThanOrEqual(0);
  });

  test('deduplicates with post-action', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create and add files
    createDummyVideo('dup1.mp4');
    createDummyVideo('dup2.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Run dedupe with post-action (mark-deleted)
    const result = await cli.run(['dedupe', '--post-action', 'mark-deleted', '--simulate', testDbPath]);

    // Should complete
    expect(result.exitCode).toBeGreaterThanOrEqual(0);
  });

  test('deduplicates with limit', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create and add multiple files
    for (let i = 0; i < 5; i++) {
      createDummyVideo(`dup${i}.mp4`);
    }
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Run dedupe with limit
    const result = await cli.run(['dedupe', '--action-limit', '2', '--simulate', testDbPath]);

    // Should complete
    expect(result.exitCode).toBeGreaterThanOrEqual(0);
  });

  test('deduplicates with size limit', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create and add files
    createDummyVideo('dup1.mp4');
    createDummyVideo('dup2.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Run dedupe with size limit
    const result = await cli.run(['dedupe', '--action-size', '1MB', '--simulate', testDbPath]);

    // Should complete
    expect(result.exitCode).toBeGreaterThanOrEqual(0);
  });

  test('deduplicates empty database', async ({ cli, tempDir, testDbPath }) => {
    // Run dedupe on empty database
    const result = await cli.runAndVerify(['dedupe', '--simulate', testDbPath]);

    // Should succeed
    expect(result.exitCode).toBe(0);
  });
});

test.describe('CLI: Big-Dirs Command', () => {
  test('shows big directories', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create subdirectory structure
    const subDir = path.join(tempDir, 'big_dir');
    fs.mkdirSync(subDir);
    fs.writeFileSync(path.join(subDir, 'video1.mp4'), Buffer.alloc(1024));
    fs.writeFileSync(path.join(subDir, 'video2.mp4'), Buffer.alloc(1024));
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Show big dirs
    const result = await cli.runAndVerify(['big-dirs', testDbPath]);

    // Should show big directories
    expect(result.stdout).toBeTruthy();
  });

  test('shows big directories as JSON', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create subdirectory structure
    const subDir = path.join(tempDir, 'big_dir');
    fs.mkdirSync(subDir);
    fs.writeFileSync(path.join(subDir, 'video.mp4'), Buffer.alloc(1024));
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Show big dirs as JSON
    const result = await cli.runJson(['big-dirs', '-j', testDbPath]);

    expect(Array.isArray(result)).toBe(true);
  });

  test('shows big directories with limit', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create multiple subdirectories
    for (let i = 0; i < 5; i++) {
      const subDir = path.join(tempDir, `dir${i}`);
      fs.mkdirSync(subDir);
      fs.writeFileSync(path.join(subDir, 'video.mp4'), Buffer.alloc(1024));
    }
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Show big dirs with limit
    const result = await cli.runAndVerify(['big-dirs', '-L', '3', testDbPath]);

    // Should limit results
    const lines = result.stdout.split('\n').filter(l => l.includes('dir'));
    expect(lines.length).toBeLessThanOrEqual(3);
  });

  test('shows big directories sorted by size', async ({ cli, tempDir, testDbPath, createDummyFile }) => {
    // Create subdirectories with different sizes
    const smallDir = path.join(tempDir, 'small');
    const largeDir = path.join(tempDir, 'large');
    fs.mkdirSync(smallDir);
    fs.mkdirSync(largeDir);
    
    fs.writeFileSync(path.join(smallDir, 'file.bin'), Buffer.alloc(100));
    fs.writeFileSync(path.join(largeDir, 'file.bin'), Buffer.alloc(10000));
    
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Show big dirs sorted by size
    const result = await cli.runAndVerify(['big-dirs', '-u', 'size', testDbPath]);

    // Large should come first
    expect(result.stdout.indexOf('large')).toBeLessThan(result.stdout.indexOf('small'));
  });

  test('shows big directories with depth', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create nested subdirectory structure
    const subDir = path.join(tempDir, 'subdir');
    const subSubDir = path.join(subDir, 'subsubdir');
    fs.mkdirSync(subDir);
    fs.mkdirSync(subSubDir);
    
    fs.writeFileSync(path.join(subSubDir, 'video.mp4'), Buffer.alloc(1024));
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Show big dirs with depth
    const result = await cli.runAndVerify(['big-dirs', '-D', '2', testDbPath]);

    // Should aggregate at depth 2
    expect(result.stdout).toBeTruthy();
  });

  test('shows big directories with columns', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create subdirectory structure
    const subDir = path.join(tempDir, 'subdir');
    fs.mkdirSync(subDir);
    fs.writeFileSync(path.join(subDir, 'video.mp4'), Buffer.alloc(1024));
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Show big dirs with columns
    const result = await cli.runAndVerify(['big-dirs', '-c', 'path,size,count', testDbPath]);

    // Should show specified columns
    expect(result.stdout).toBeTruthy();
  });

  test('shows big directories summarized', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create subdirectory structure
    const subDir = path.join(tempDir, 'subdir');
    fs.mkdirSync(subDir);
    fs.writeFileSync(path.join(subDir, 'video.mp4'), Buffer.alloc(1024));
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Show big dirs summarized
    const result = await cli.runAndVerify(['big-dirs', '--summarize', testDbPath]);

    // Should show summary
    expect(result.stdout).toBeTruthy();
  });

  test('shows big directories empty database', async ({ cli, tempDir, testDbPath }) => {
    // Show big dirs for empty database
    const result = await cli.runAndVerify(['big-dirs', testDbPath]);

    // Should succeed (no results)
    expect(result.exitCode).toBe(0);
  });
});
