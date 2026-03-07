import { test, expect } from '../fixtures-cli';
import * as fs from 'fs';
import * as path from 'path';

test.describe('CLI: History Commands', () => {
  test.describe.configure({ mode: 'serial' });
  test('adds file to history', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create and add a file
    const videoPath = createDummyVideo('history_test.mp4');
    await cli.runAndVerify(['add', testDbPath, videoPath]);

    // Add to history
    const result = await cli.runAndVerify(['history-add', testDbPath, videoPath]);

    // Verify command succeeded
    expect(result.exitCode).toBe(0);

    // Verify file is in history
    const queryResult = await cli.runJson(['search-db', testDbPath, 'history', '-w', `media_path = '${videoPath}'`]);
    expect(Array.isArray(queryResult)).toBe(true);
    expect(queryResult.length).toBeGreaterThan(0);
  });

  test('adds file to history with done flag', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create and add a file
    const videoPath = createDummyVideo('history_done_test.mp4');
    await cli.runAndVerify(['add', testDbPath, videoPath]);

    // Add to history with done flag
    const result = await cli.runAndVerify(['history-add', '--done', testDbPath, videoPath]);

    // Verify file is in history with done = 1
    const queryResult = await cli.runJson(['search-db', testDbPath, 'history', '-w', `media_path = '${videoPath}'`]);
    expect(Array.isArray(queryResult)).toBe(true);
    expect(queryResult.length).toBeGreaterThan(0);
    expect(queryResult[0].done).toBe(1);
  });

  test('adds file to history with playhead', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create and add a file
    const videoPath = createDummyVideo('history_playhead_test.mp4');
    await cli.runAndVerify(['add', testDbPath, videoPath]);

    // Add to history with playhead
    const result = await cli.runAndVerify(['history-add', '--playhead', '60', testDbPath, videoPath]);

    // Verify command succeeded
    expect(result.exitCode).toBe(0);
  });

  test('shows history', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create and add files
    const videoPath1 = createDummyVideo('history1.mp4');
    const videoPath2 = createDummyVideo('history2.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Add to history
    await cli.runAndVerify(['history-add', testDbPath, videoPath1]);
    await cli.runAndVerify(['history-add', testDbPath, videoPath2]);

    // Show history
    const result = await cli.runAndVerify(['history', testDbPath]);

    // Should show history entries
    expect(result.stdout).toBeTruthy();
  });

  test('shows history as JSON', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create and add a file
    const videoPath = createDummyVideo('history_json.mp4');
    await cli.runAndVerify(['add', testDbPath, videoPath]);

    // Add to history
    await cli.runAndVerify(['history-add', testDbPath, videoPath]);

    // Show history as JSON
    const result = await cli.runJson(['history', '-j', testDbPath]);

    expect(Array.isArray(result)).toBe(true);
  });

  test('imports mpv watchlater files', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create and add a file
    const videoPath = createDummyVideo('watchlater_test.mp4');
    await cli.runAndVerify(['add', testDbPath, videoPath]);

    // Create mpv watchlater directory and file
    const watchlaterDir = path.join(tempDir, 'mpv_watchlater');
    fs.mkdirSync(watchlaterDir);
    
    // Create watchlater file with hash of path
    const crypto = require('crypto');
    const hash = crypto.createHash('md5').update(videoPath).digest('hex');
    const watchlaterContent = `pause=no
pos=123.456
`;
    fs.writeFileSync(path.join(watchlaterDir, hash), watchlaterContent);

    // Import watchlater
    const result = await cli.runAndVerify(['mpv-watchlater', testDbPath, watchlaterDir]);

    // Should import successfully
    expect(result.exitCode).toBe(0);
  });

  test('handles non-existent file for history', async ({ cli, tempDir, testDbPath }) => {
    const nonExistentPath = path.join(tempDir, 'not_exists.mp4');

    // Try to add non-existent file to history
    const result = await cli.run(['history-add', testDbPath, nonExistentPath]);

    // Should fail gracefully
    expect(result.exitCode).not.toBe(0);
  });
});

test.describe('CLI: Stats Command', () => {
  test('shows library statistics', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Add files
    createDummyVideo('video1.mp4');
    createDummyVideo('video2.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Show stats
    const result = await cli.runAndVerify(['stats', testDbPath]);

    // Should show statistics
    expect(result.stdout).toBeTruthy();
    expect(result.stdout).toMatch(/(total|count|size|duration)/i);
  });

  test('shows statistics as JSON', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Add files
    createDummyVideo('video1.mp4');
    createDummyVideo('video2.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Show stats as JSON
    const result = await cli.runJson(['stats', '-j', testDbPath]);

    expect(typeof result).toBe('object');
    expect(result).toHaveProperty('total_count');
  });

  test('shows statistics by category', async ({ cli, tempDir, testDbPath, createDummyVideo, createDummyAudio }) => {
    // Add files of different types
    createDummyVideo('video.mp4');
    createDummyAudio('audio.mp3');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Show stats by category
    const result = await cli.runAndVerify(['stats', '--by-category', testDbPath]);

    // Should show category breakdown
    expect(result.stdout).toBeTruthy();
  });

  test('shows statistics by extension', async ({ cli, tempDir, testDbPath, createDummyVideo, createDummyAudio }) => {
    // Add files of different types
    createDummyVideo('video.mp4');
    createDummyAudio('audio.mp3');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Show stats by extension
    const result = await cli.runAndVerify(['stats', '--by-extension', testDbPath]);

    // Should show extension breakdown
    expect(result.stdout).toBeTruthy();
  });

  test('shows statistics by size buckets', async ({ cli, tempDir, testDbPath, createDummyFile }) => {
    // Add files of different sizes
    createDummyFile('small.bin', 100);
    createDummyFile('medium.bin', 5000);
    createDummyFile('large.bin', 10000);
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Show stats by size
    const result = await cli.runAndVerify(['stats', '--by-size', testDbPath]);

    // Should show size buckets
    expect(result.stdout).toBeTruthy();
  });

  test('shows statistics with time frequency', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Add files
    createDummyVideo('video1.mp4');
    createDummyVideo('video2.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Show stats with frequency
    const result = await cli.runAndVerify(['stats', '-f', 'monthly', testDbPath]);

    // Should show time-based stats
    expect(result.stdout).toBeTruthy();
  });

  test('shows statistics for empty database', async ({ cli, tempDir, testDbPath }) => {
    // Show stats for empty database
    const result = await cli.runAndVerify(['stats', testDbPath]);

    // Should show zero statistics
    expect(result.stdout).toBeTruthy();
  });
});

test.describe('CLI: Playlists Command', () => {
  test('lists playlists', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Add files (creates playlist automatically)
    createDummyVideo('video.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // List playlists
    const result = await cli.runAndVerify(['playlists', testDbPath]);

    // Should list playlists
    expect(result.stdout).toBeTruthy();
  });

  test('lists playlists as JSON', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Add files
    createDummyVideo('video.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // List playlists as JSON
    const result = await cli.runJson(['playlists', '-j', testDbPath]);

    expect(Array.isArray(result)).toBe(true);
  });

  test('lists playlists with columns', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Add files
    createDummyVideo('video.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // List playlists with specific columns
    const result = await cli.runAndVerify(['playlists', '-c', 'path,count', testDbPath]);

    // Should show specified columns
    expect(result.stdout).toBeTruthy();
  });

  test('lists playlists summarized', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Add files
    createDummyVideo('video.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // List playlists summarized
    const result = await cli.runAndVerify(['playlists', '--summarize', testDbPath]);

    // Should show summary
    expect(result.stdout).toBeTruthy();
  });

  test('lists empty playlists', async ({ cli, tempDir, testDbPath }) => {
    // List playlists for empty database
    const result = await cli.runAndVerify(['playlists', testDbPath]);

    // Should succeed (may show no results)
    expect(result.exitCode).toBe(0);
  });
});

test.describe('CLI: Optimize Command', () => {
  test('optimizes database', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Add files to create database content
    createDummyVideo('video.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Optimize database
    const result = await cli.runAndVerify(['optimize', testDbPath]);

    // Should optimize successfully
    expect(result.exitCode).toBe(0);
    expect(result.stdout).toMatch(/(vacuum|analyze|optimize)/i);
  });

  test('optimizes database with vacuum only', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Add files
    createDummyVideo('video.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Optimize with vacuum only
    const result = await cli.runAndVerify(['optimize', '--vacuum', testDbPath]);

    // Should vacuum successfully
    expect(result.exitCode).toBe(0);
  });

  test('optimizes database with analyze only', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Add files
    createDummyVideo('video.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Optimize with analyze only
    const result = await cli.runAndVerify(['optimize', '--analyze', testDbPath]);

    // Should analyze successfully
    expect(result.exitCode).toBe(0);
  });

  test('optimizes FTS tables', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Add files
    createDummyVideo('video.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Optimize FTS
    const result = await cli.runAndVerify(['optimize', '--fts', testDbPath]);

    // Should optimize FTS successfully
    expect(result.exitCode).toBe(0);
  });

  test('optimizes empty database', async ({ cli, tempDir, testDbPath }) => {
    // Optimize empty database
    const result = await cli.runAndVerify(['optimize', testDbPath]);

    // Should succeed
    expect(result.exitCode).toBe(0);
  });
});

test.describe('CLI: Repair Command', () => {
  test('repairs database', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Add files
    createDummyVideo('video.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Repair database
    const result = await cli.runAndVerify(['repair', testDbPath]);

    // Should repair successfully
    expect(result.exitCode).toBe(0);
  });

  test('repairs database with verbose output', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Add files
    createDummyVideo('video.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Repair with verbose
    const result = await cli.runAndVerify(['repair', '--verbose', testDbPath]);

    // Should repair successfully with output
    expect(result.exitCode).toBe(0);
    expect(result.stdout).toBeTruthy();
  });
});
