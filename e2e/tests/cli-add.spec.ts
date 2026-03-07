import { test, expect } from '../fixtures-cli';
import * as fs from 'fs';
import * as path from 'path';

test.describe('CLI: Add Command', () => {
  test.describe.configure({ mode: 'serial' });
  test('adds a single video file to database', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create a dummy video file
    const videoPath = createDummyVideo('test_video.mp4');

    // Run add command
    const result = await cli.runAndVerify(['add', testDbPath, videoPath]);

    // Verify command succeeded
    expect(result.exitCode).toBe(0);
    expect(result.stdout).toContain('Processed');

    // Verify file is in database
    const queryResult = await cli.runJson(['print', testDbPath, '--search', 'test_video', '--json']);
    expect(Array.isArray(queryResult)).toBe(true);
    expect(queryResult.length).toBeGreaterThan(0);
  });

  test('adds multiple files from directory', async ({ cli, tempDir, testDbPath, createDummyVideo, createDummyAudio }) => {
    // Create multiple dummy files
    createDummyVideo('video1.mp4');
    createDummyVideo('video2.mp4');
    createDummyAudio('audio1.mp3');

    // Run add command on directory
    const result = await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Verify command succeeded
    expect(result.exitCode).toBe(0);

    // Verify files are in database
    const queryResult = await cli.runJson(['print', testDbPath, '--all', '--json']);
    expect(Array.isArray(queryResult)).toBe(true);
    expect(queryResult.length).toBeGreaterThanOrEqual(3);
  });

  test('adds files with video-only filter', async ({ cli, tempDir, testDbPath, createDummyVideo, createDummyAudio }) => {
    // Create mixed file types
    createDummyVideo('video.mp4');
    createDummyAudio('audio.mp3');

    // Run add command with video-only filter
    const result = await cli.runAndVerify(['add', '--video-only', testDbPath, tempDir]);

    // Verify only video was added
    const queryResult = await cli.runJson(['print', testDbPath, '--all', '--json']);
    expect(Array.isArray(queryResult)).toBe(true);
    expect(queryResult.length).toBe(1);
    expect(queryResult[0].path).toContain('video.mp4');
  });

  test('adds files with audio-only filter', async ({ cli, tempDir, testDbPath, createDummyVideo, createDummyAudio }) => {
    // Create mixed file types
    createDummyVideo('video.mp4');
    createDummyAudio('audio.mp3');

    // Run add command with audio-only filter
    const result = await cli.runAndVerify(['add', '--audio-only', testDbPath, tempDir]);

    // Verify only audio was added
    const queryResult = await cli.runJson(['print', testDbPath, '--all', '--json']);
    expect(Array.isArray(queryResult)).toBe(true);
    expect(queryResult.length).toBe(1);
    expect(queryResult[0].path).toContain('audio.mp3');
  });

  test('adds files with image-only filter', async ({ cli, tempDir, testDbPath, createDummyImage }) => {
    // Create image files
    createDummyImage('image.jpg');
    createDummyImage('photo.png');

    // Run add command with image-only filter
    const result = await cli.runAndVerify(['add', '--image-only', testDbPath, tempDir]);

    // Verify images were added
    const queryResult = await cli.runJson(['print', testDbPath, '--all', '--json']);
    expect(Array.isArray(queryResult)).toBe(true);
    expect(queryResult.length).toBeGreaterThanOrEqual(2);
  });

  test('adds files with text-only filter', async ({ cli, tempDir, testDbPath, createDummyDocument }) => {
    // Create document files
    createDummyDocument('book.epub');
    createDummyDocument('document.pdf');

    // Run add command with text-only filter
    const result = await cli.runAndVerify(['add', '--text-only', testDbPath, tempDir]);

    // Verify documents were added
    const queryResult = await cli.runJson(['print', testDbPath, '--all', '--json']);
    expect(Array.isArray(queryResult)).toBe(true);
    expect(queryResult.length).toBeGreaterThanOrEqual(2);
  });

  test('adds files with extension filter', async ({ cli, tempDir, testDbPath, createDummyVideo, createDummyAudio }) => {
    // Create files with different extensions
    createDummyVideo('video.mp4');
    createDummyVideo('video.mkv');
    createDummyAudio('audio.mp3');

    // Run add command with extension filter
    const result = await cli.runAndVerify(['add', '--ext', '.mp4', testDbPath, tempDir]);

    // Verify only .mp4 was added
    const queryResult = await cli.runJson(['print', testDbPath, '--all', '--json']);
    expect(Array.isArray(queryResult)).toBe(true);
    expect(queryResult.length).toBe(1);
    expect(queryResult[0].path).toContain('.mp4');
  });

  test('adds files with exclude pattern', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create files
    createDummyVideo('keep_this.mp4');
    createDummyVideo('exclude_this.mp4');

    // Run add command with exclude pattern
    const result = await cli.runAndVerify(['add', '--exclude', 'exclude', testDbPath, tempDir]);

    // Verify excluded file was not added
    const queryResult = await cli.runJson(['print', testDbPath, '--all', '--json']);
    expect(Array.isArray(queryResult)).toBe(true);
    expect(queryResult.length).toBe(1);
    expect(queryResult[0].path).toContain('keep_this');
  });

  test('adds files with include pattern', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create files
    createDummyVideo('important_video.mp4');
    createDummyVideo('other_video.mp4');

    // Run add command with include pattern
    const result = await cli.runAndVerify(['add', '--include', 'important', testDbPath, tempDir]);

    // Verify only included file was added
    const queryResult = await cli.runJson(['print', testDbPath, '--all', '--json']);
    expect(Array.isArray(queryResult)).toBe(true);
    expect(queryResult.length).toBe(1);
    expect(queryResult[0].path).toContain('important');
  });

  test('adds files with size filter', async ({ cli, tempDir, testDbPath, createDummyFile }) => {
    // Create files of different sizes
    createDummyFile('small.bin', 100); // 100 bytes
    createDummyFile('large.bin', 10000); // 10KB

    // Run add command with size filter
    const result = await cli.runAndVerify(['add', '--size', '>1KB', testDbPath, tempDir]);

    // Verify only large file was added
    const queryResult = await cli.runJson(['print', testDbPath, '--all', '--json']);
    expect(Array.isArray(queryResult)).toBe(true);
    expect(queryResult.length).toBe(1);
    expect(queryResult[0].path).toContain('large.bin');
  });

  test('handles non-existent path gracefully', async ({ cli, tempDir, testDbPath }) => {
    const nonExistentPath = path.join(tempDir, 'does_not_exist.mp4');

    // Run add command with non-existent path
    const result = await cli.run(['add', testDbPath, nonExistentPath]);

    // Should fail gracefully
    expect(result.exitCode).not.toBe(0);
    expect(result.stderr).toContain('not found');
  });

  test('handles empty directory', async ({ cli, tempDir, testDbPath }) => {
    // Create empty subdirectory
    const emptyDir = path.join(tempDir, 'empty');
    fs.mkdirSync(emptyDir);

    // Run add command on empty directory
    const result = await cli.run(['add', testDbPath, emptyDir]);

    // Should succeed but add nothing
    expect(result.exitCode).toBe(0);
  });

  test('adds files with regex pattern', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create files
    createDummyVideo('movie_2023.mp4');
    createDummyVideo('show_2022.mp4');

    // Run add command with regex pattern
    const result = await cli.runAndVerify(['add', '--regex', '2023', testDbPath, tempDir]);

    // Verify only matching file was added
    const queryResult = await cli.runJson(['print', testDbPath, '--all', '--json']);
    expect(Array.isArray(queryResult)).toBe(true);
    expect(queryResult.length).toBe(1);
    expect(queryResult[0].path).toContain('2023');
  });

  test('adds files with path-contains filter', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create subdirectory with files
    const subDir = path.join(tempDir, 'movies');
    fs.mkdirSync(subDir);
    const videoPath = path.join(subDir, 'video.mp4');
    fs.writeFileSync(videoPath, Buffer.alloc(1024));

    // Run add command with path-contains filter
    const result = await cli.runAndVerify(['add', '--path-contains', 'movies', testDbPath, tempDir]);

    // Verify file in movies directory was added
    const queryResult = await cli.runJson(['print', testDbPath, '--all', '--json']);
    expect(Array.isArray(queryResult)).toBe(true);
    expect(queryResult.length).toBe(1);
    expect(queryResult[0].path).toContain('movies');
  });

  test('adds files with mime-type filter', async ({ cli, tempDir, testDbPath, createDummyVideo, createDummyAudio }) => {
    // Create files
    createDummyVideo('video.mp4');
    createDummyAudio('audio.mp3');

    // Run add command with mime-type filter
    const result = await cli.runAndVerify(['add', '--mime-type', 'video', testDbPath, tempDir]);

    // Verify only video was added
    const queryResult = await cli.runJson(['print', testDbPath, '--all', '--json']);
    expect(Array.isArray(queryResult)).toBe(true);
    expect(queryResult.length).toBe(1);
  });

  test('adds files with verbose output', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create file
    const videoPath = createDummyVideo('verbose_test.mp4');

    // Run add command with verbose flag
    const result = await cli.runAndVerify(['add', '--verbose', testDbPath, videoPath]);

    // Should have verbose output
    expect(result.stdout).toBeTruthy();
  });

  test('dry run does not modify database', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create file
    const videoPath = createDummyVideo('dry_run.mp4');

    // Run add command with simulate flag
    const result = await cli.runAndVerify(['add', '--simulate', testDbPath, videoPath]);

    // Verify database is empty
    const queryResult = await cli.runJson(['print', testDbPath, '--all', '--json']);
    expect(Array.isArray(queryResult)).toBe(true);
    expect(queryResult.length).toBe(0);
  });

  test('no-confirm flag skips confirmation', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create file
    const videoPath = createDummyVideo('no_confirm.mp4');

    // Run add command with no-confirm flag
    const result = await cli.runAndVerify(['add', '--no-confirm', '-y', testDbPath, videoPath]);

    // Should succeed without hanging
    expect(result.exitCode).toBe(0);
  });
});
