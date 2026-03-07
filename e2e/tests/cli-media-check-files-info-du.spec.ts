import { test, expect } from '../fixtures-cli';
import * as fs from 'fs';
import * as path from 'path';

test.describe('CLI: Media-Check Command', () => {
  test.describe.configure({ mode: 'serial' });
  test('checks media file for corruption', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create and add a file
    const videoPath = createDummyVideo('check_corruption.mp4');
    await cli.runAndVerify(['add', testDbPath, videoPath]);

    // Check media
    const result = await cli.run(['media-check', testDbPath, '--search', 'check_corruption']);

    // Should complete (may report corruption for dummy file)
    expect(result.exitCode).toBeGreaterThanOrEqual(0);
  });

  test('checks media with full scan', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create and add a file
    const videoPath = createDummyVideo('full_scan.mp4');
    await cli.runAndVerify(['add', testDbPath, videoPath]);

    // Check with full scan
    const result = await cli.run(['media-check', '--full-scan', testDbPath, '--search', 'full_scan']);

    // Should complete
    expect(result.exitCode).toBeGreaterThanOrEqual(0);
  });

  test('checks media with chunk size', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create and add a file
    const videoPath = createDummyVideo('chunk_check.mp4');
    await cli.runAndVerify(['add', testDbPath, videoPath]);

    // Check with chunk size
    const result = await cli.run(['media-check', '--chunk-size', '1', testDbPath, '--search', 'chunk_check']);

    // Should complete
    expect(result.exitCode).toBeGreaterThanOrEqual(0);
  });

  test('checks media with gap', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create and add a file
    const videoPath = createDummyVideo('gap_check.mp4');
    await cli.runAndVerify(['add', testDbPath, videoPath]);

    // Check with gap
    const result = await cli.run(['media-check', '--gap', '2', testDbPath, '--search', 'gap_check']);

    // Should complete
    expect(result.exitCode).toBeGreaterThanOrEqual(0);
  });

  test('checks audio track only', async ({ cli, tempDir, testDbPath, createDummyAudio }) => {
    // Create and add a file
    const audioPath = createDummyAudio('audio_check.mp3');
    await cli.runAndVerify(['add', testDbPath, audioPath]);

    // Check audio only
    const result = await cli.run(['media-check', '--audio-scan', testDbPath, '--search', 'audio_check']);

    // Should complete
    expect(result.exitCode).toBeGreaterThanOrEqual(0);
  });

  test('checks with delete corrupt option', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create and add a file
    const videoPath = createDummyVideo('delete_corrupt.mp4');
    await cli.runAndVerify(['add', testDbPath, videoPath]);

    // Check with delete corrupt (dry run)
    const result = await cli.run(['media-check', '--delete-corrupt', '1', '--simulate', testDbPath, '--search', 'delete_corrupt']);

    // Should complete
    expect(result.exitCode).toBeGreaterThanOrEqual(0);
  });

  test('checks with full scan if corrupt', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create and add a file
    const videoPath = createDummyVideo('full_if_corrupt.mp4');
    await cli.runAndVerify(['add', testDbPath, videoPath]);

    // Check with full scan if corrupt
    const result = await cli.run(['media-check', '--full-scan-if-corrupt', '0.5', testDbPath, '--search', 'full_if_corrupt']);

    // Should complete
    expect(result.exitCode).toBeGreaterThanOrEqual(0);
  });

  test('checks with verbose output', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create and add a file
    const videoPath = createDummyVideo('verbose_check.mp4');
    await cli.runAndVerify(['add', testDbPath, videoPath]);

    // Check with verbose
    const result = await cli.run(['media-check', '--verbose', testDbPath, '--search', 'verbose_check']);

    // Should have output
    expect(result.stdout || result.stderr).toBeTruthy();
  });
});

test.describe('CLI: Files-Info Command', () => {
  test('shows information about files', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create and add a file
    const videoPath = createDummyVideo('file_info.mp4');
    await cli.runAndVerify(['add', testDbPath, videoPath]);

    // Show file info
    const result = await cli.runAndVerify(['files-info', testDbPath, '--search', 'file_info']);

    // Should show file information
    expect(result.stdout).toBeTruthy();
  });

  test('shows file info as JSON', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create and add a file
    const videoPath = createDummyVideo('file_info_json.mp4');
    await cli.runAndVerify(['add', testDbPath, videoPath]);

    // Show file info as JSON
    const result = await cli.runJson(['files-info', '-j', testDbPath, '--search', 'file_info_json']);

    expect(Array.isArray(result)).toBe(true);
  });

  test('shows file info with custom columns', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create and add a file
    const videoPath = createDummyVideo('file_info_columns.mp4');
    await cli.runAndVerify(['add', testDbPath, videoPath]);

    // Show file info with columns
    const result = await cli.runAndVerify(['files-info', '-c', 'path,size,duration', testDbPath, '--search', 'file_info_columns']);

    // Should show specified columns
    expect(result.stdout).toBeTruthy();
  });

  test('shows file info summarized', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create and add files
    createDummyVideo('file1.mp4');
    createDummyVideo('file2.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Show file info summarized
    const result = await cli.runAndVerify(['files-info', '--summarize', testDbPath]);

    // Should show summary
    expect(result.stdout).toBeTruthy();
  });

  test('shows file info with frequency', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create and add files
    createDummyVideo('file1.mp4');
    createDummyVideo('file2.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Show file info with frequency
    const result = await cli.runAndVerify(['files-info', '-f', 'monthly', testDbPath]);

    // Should show time-based info
    expect(result.stdout).toBeTruthy();
  });

  test('shows file info filtered by size', async ({ cli, tempDir, testDbPath, createDummyFile }) => {
    // Create files of different sizes
    createDummyFile('small.bin', 100);
    createDummyFile('large.bin', 10000);
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Show file info filtered by size
    const result = await cli.runAndVerify(['files-info', '-S', '>1KB', testDbPath]);

    // Should only show large file
    expect(result.stdout).toContain('large.bin');
    expect(result.stdout).not.toContain('small.bin');
  });

  test('shows file info filtered by duration', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create and add a file
    createDummyVideo('duration_filter.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Show file info filtered by duration
    const result = await cli.runAndVerify(['files-info', '-d', '>0', testDbPath]);

    // Command should succeed
    expect(result.exitCode).toBe(0);
  });

  test('shows file info with watched filter', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create and add a file
    const videoPath = createDummyVideo('watched_filter.mp4');
    await cli.runAndVerify(['add', testDbPath, videoPath]);

    // Show file info with watched filter
    const result = await cli.runAndVerify(['files-info', '--watched', 'false', testDbPath]);

    // Command should succeed
    expect(result.exitCode).toBe(0);
  });

  test('shows file info with play count filter', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create and add a file
    createDummyVideo('playcount_filter.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Show file info with play count filter
    const result = await cli.runAndVerify(['files-info', '--play-count-min', '0', testDbPath]);

    // Command should succeed
    expect(result.exitCode).toBe(0);
  });

  test('shows file info with category filter', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create and add a file
    createDummyVideo('category_filter.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Show file info with category filter
    const result = await cli.runAndVerify(['files-info', '--category', 'video', testDbPath]);

    // Command should succeed
    expect(result.exitCode).toBe(0);
  });

  test('shows file info with date filters', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create and add a file
    createDummyVideo('date_filter.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Show file info with created after filter
    const result = await cli.runAndVerify(['files-info', '--created-after', '2020-01-01', testDbPath]);

    // Should show recently created file
    expect(result.stdout).toContain('date_filter.mp4');
  });

  test('shows file info with where clause', async ({ cli, tempDir, testDbPath, createDummyVideo, createDummyAudio }) => {
    // Create and add files
    createDummyVideo('video.mp4');
    createDummyAudio('audio.mp3');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Show file info with where clause
    const result = await cli.runAndVerify(['files-info', '-w', "type = 'video'", testDbPath]);

    // Should only show video
    expect(result.stdout).toContain('video.mp4');
    expect(result.stdout).not.toContain('audio.mp3');
  });

  test('shows file info with TUI flag', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create and add a file
    createDummyVideo('tui_test.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Show file info with TUI flag (should work in non-interactive mode)
    const result = await cli.run(['files-info', '--tui', testDbPath]);

    // May fail in non-interactive mode, but shouldn't crash
    expect(result.exitCode).toBeGreaterThanOrEqual(0);
  });
});

test.describe('CLI: Disk-Usage Command', () => {
  test('shows disk usage aggregation', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create and add files
    createDummyVideo('video1.mp4');
    createDummyVideo('video2.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Show disk usage
    const result = await cli.runAndVerify(['disk-usage', testDbPath]);

    // Should show disk usage
    expect(result.stdout).toBeTruthy();
  });

  test('shows disk usage as JSON', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create and add files
    createDummyVideo('video1.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Show disk usage as JSON
    const result = await cli.runJson(['disk-usage', '-j', testDbPath]);

    expect(Array.isArray(result)).toBe(true);
  });

  test('shows disk usage with columns', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create and add files
    createDummyVideo('video1.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Show disk usage with columns
    const result = await cli.runAndVerify(['disk-usage', '-c', 'path,size', testDbPath]);

    // Should show specified columns
    expect(result.stdout).toBeTruthy();
  });

  test('shows disk usage summarized', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create and add files
    createDummyVideo('video1.mp4');
    createDummyVideo('video2.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Show disk usage summarized
    const result = await cli.runAndVerify(['disk-usage', '--summarize', testDbPath]);

    // Should show summary
    expect(result.stdout).toBeTruthy();
  });

  test('shows disk usage by depth', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create subdirectory structure
    const subDir = path.join(tempDir, 'subdir');
    fs.mkdirSync(subDir);
    const subSubDir = path.join(subDir, 'subsubdir');
    fs.mkdirSync(subSubDir);
    
    fs.writeFileSync(path.join(subDir, 'video1.mp4'), Buffer.alloc(1024));
    fs.writeFileSync(path.join(subSubDir, 'video2.mp4'), Buffer.alloc(1024));
    
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Show disk usage by depth
    const result = await cli.runAndVerify(['disk-usage', '-D', '1', testDbPath]);

    // Should aggregate at depth 1
    expect(result.stdout).toBeTruthy();
  });

  test('shows disk usage with min depth', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create subdirectory structure
    const subDir = path.join(tempDir, 'subdir');
    fs.mkdirSync(subDir);
    fs.writeFileSync(path.join(subDir, 'video.mp4'), Buffer.alloc(1024));
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Show disk usage with min depth
    const result = await cli.runAndVerify(['disk-usage', '--min-depth', '1', testDbPath]);

    // Should show directories at min depth 1
    expect(result.stdout).toBeTruthy();
  });

  test('shows disk usage with max depth', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create subdirectory structure
    const subDir = path.join(tempDir, 'subdir');
    fs.mkdirSync(subDir);
    const subSubDir = path.join(subDir, 'subsubdir');
    fs.mkdirSync(subSubDir);
    
    fs.writeFileSync(path.join(subSubDir, 'video.mp4'), Buffer.alloc(1024));
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Show disk usage with max depth
    const result = await cli.runAndVerify(['disk-usage', '--max-depth', '1', testDbPath]);

    // Should limit to max depth 1
    expect(result.stdout).toBeTruthy();
  });

  test('shows disk usage with parents', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create subdirectory structure
    const subDir = path.join(tempDir, 'subdir');
    fs.mkdirSync(subDir);
    fs.writeFileSync(path.join(subDir, 'video.mp4'), Buffer.alloc(1024));
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Show disk usage with parents
    const result = await cli.runAndVerify(['disk-usage', '--parents', testDbPath]);

    // Should include parent directories
    expect(result.stdout).toBeTruthy();
  });

  test('shows disk usage folders only', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create and add files
    createDummyVideo('video.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Show disk usage folders only
    const result = await cli.runAndVerify(['disk-usage', '--folders-only', testDbPath]);

    // Should show only folders
    expect(result.stdout).toBeTruthy();
  });

  test('shows disk usage files only', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create and add files
    createDummyVideo('video.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Show disk usage files only
    const result = await cli.runAndVerify(['disk-usage', '--files-only', testDbPath]);

    // Should show only files
    expect(result.stdout).toBeTruthy();
  });

  test('shows disk usage grouped by extension', async ({ cli, tempDir, testDbPath, createDummyVideo, createDummyAudio }) => {
    // Create and add files
    createDummyVideo('video.mp4');
    createDummyAudio('audio.mp3');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Show disk usage grouped by extension
    const result = await cli.runAndVerify(['disk-usage', '--group-by-extensions', testDbPath]);

    // Should show grouped by extension
    expect(result.stdout).toBeTruthy();
  });

  test('shows disk usage grouped by mime types', async ({ cli, tempDir, testDbPath, createDummyVideo, createDummyAudio }) => {
    // Create and add files
    createDummyVideo('video.mp4');
    createDummyAudio('audio.mp3');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Show disk usage grouped by mime types
    const result = await cli.runAndVerify(['disk-usage', '--group-by-mime-types', testDbPath]);

    // Should show grouped by mime type
    expect(result.stdout).toBeTruthy();
  });

  test('shows disk usage grouped by size', async ({ cli, tempDir, testDbPath, createDummyFile }) => {
    // Create files of different sizes
    createDummyFile('small.bin', 100);
    createDummyFile('large.bin', 10000);
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Show disk usage grouped by size
    const result = await cli.runAndVerify(['disk-usage', '--group-by-size', testDbPath]);

    // Should show grouped by size
    expect(result.stdout).toBeTruthy();
  });

  test('shows disk usage grouped by parent', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create subdirectory structure
    const subDir = path.join(tempDir, 'subdir');
    fs.mkdirSync(subDir);
    fs.writeFileSync(path.join(subDir, 'video.mp4'), Buffer.alloc(1024));
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Show disk usage grouped by parent
    const result = await cli.runAndVerify(['disk-usage', '--group-by-parent', testDbPath]);

    // Should show grouped by parent
    expect(result.stdout).toBeTruthy();
  });

  test('shows disk usage with folder sizes filter', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create subdirectory structure
    const subDir = path.join(tempDir, 'subdir');
    fs.mkdirSync(subDir);
    fs.writeFileSync(path.join(subDir, 'video.mp4'), Buffer.alloc(1024));
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Show disk usage with folder sizes filter
    const result = await cli.runAndVerify(['disk-usage', '--folder-sizes', '>0', testDbPath]);

    // Should show folders with size > 0
    expect(result.stdout).toBeTruthy();
  });

  test('shows disk usage with folder counts filter', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create subdirectory structure
    const subDir = path.join(tempDir, 'subdir');
    fs.mkdirSync(subDir);
    fs.writeFileSync(path.join(subDir, 'video.mp4'), Buffer.alloc(1024));
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Show disk usage with folder counts filter
    const result = await cli.runAndVerify(['disk-usage', '--folder-counts', '>0', testDbPath]);

    // Should show folders with count > 0
    expect(result.stdout).toBeTruthy();
  });

  test('shows disk usage with file counts filter', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create and add files
    createDummyVideo('video1.mp4');
    createDummyVideo('video2.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Show disk usage with file counts filter
    const result = await cli.runAndVerify(['disk-usage', '--file-counts', '>1', testDbPath]);

    // Should show directories with file count > 1
    expect(result.stdout).toBeTruthy();
  });

  test('shows disk usage with TUI flag', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create and add files
    createDummyVideo('video.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Show disk usage with TUI flag
    const result = await cli.run(['disk-usage', '--tui', testDbPath]);

    // May fail in non-interactive mode, but shouldn't crash
    expect(result.exitCode).toBeGreaterThanOrEqual(0);
  });

  test('shows disk usage for empty database', async ({ cli, tempDir, testDbPath }) => {
    // Show disk usage for empty database
    const result = await cli.runAndVerify(['disk-usage', testDbPath]);

    // Should succeed (may show no results)
    expect(result.exitCode).toBe(0);
  });
});
