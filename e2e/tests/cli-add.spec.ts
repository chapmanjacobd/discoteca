import { test, expect } from '../fixtures-cli';
import * as fs from 'fs';
import * as path from 'path';

test.describe('CLI: Add Command', () => {
  test('adds a single video file to database', async ({ cli, testDbPath, createValidVideo }) => {
    const videoPath = createValidVideo('test_video.mp4');
    const result = await cli.runAndVerify(['add', testDbPath, videoPath]);

    expect(result.stdout).toContain('Processed 1/1 files');

    const queryResult = await cli.runJson<any[]>(['print', testDbPath, '--search', 'test_video', '--json']);
    expect(queryResult.length).toBe(1);
    expect(queryResult[0].path).toBe(videoPath);
    expect(queryResult[0].type).toBe('video');
  });

  test('adds multiple files from directory', async ({ cli, tempDir, testDbPath, createValidVideo, createValidAudio }) => {
    const v1 = createValidVideo('video1.mp4');
    const v2 = createValidVideo('video2.mp4');
    const a1 = createValidAudio('audio1.mp3');

    const result = await cli.runAndVerify(['add', testDbPath, tempDir]);
    expect(result.stdout).toContain('Processed 3/3 files');

    const queryResult = await cli.runJson<any[]>(['print', testDbPath, '--all', '--json']);
    const paths = queryResult.map(m => m.path);
    expect(paths).toContain(v1);
    expect(paths).toContain(v2);
    expect(paths).toContain(a1);
  });

  test('adds files with video-only filter', async ({ cli, tempDir, testDbPath, createValidVideo, createValidAudio }) => {
    createValidVideo('video.mp4');
    createValidAudio('audio.mp3');

    await cli.runAndVerify(['add', '--video-only', testDbPath, tempDir]);

    const queryResult = await cli.runJson<any[]>(['print', testDbPath, '--all', '--json']);
    expect(queryResult.length).toBe(1);
    expect(queryResult[0].path).toContain('video.mp4');
    expect(queryResult[0].type).toBe('video');
  });

  test('adds files with audio-only filter', async ({ cli, tempDir, testDbPath, createValidVideo, createValidAudio }) => {
    createValidVideo('video.mp4');
    createValidAudio('audio.mp3');

    await cli.runAndVerify(['add', '--audio-only', testDbPath, tempDir]);

    const queryResult = await cli.runJson<any[]>(['print', testDbPath, '--all', '--json']);
    expect(queryResult.length).toBe(1);
    expect(queryResult[0].path).toContain('audio.mp3');
    expect(queryResult[0].type).toBe('audio');
  });

  test('adds files with image-only filter', async ({ cli, tempDir, testDbPath, createValidImage }) => {
    createValidImage('image.jpg');
    createValidImage('photo.png');

    await cli.runAndVerify(['add', '--image-only', testDbPath, tempDir]);

    const queryResult = await cli.runJson<any[]>(['print', testDbPath, '--all', '--json']);
    expect(queryResult.length).toBeGreaterThanOrEqual(2);
    expect(queryResult.every(m => m.type === 'image')).toBe(true);
  });

  test('adds files with text-only filter', async ({ cli, tempDir, testDbPath, createValidDocument }) => {
    createValidDocument('book.epub');
    createValidDocument('document.pdf');

    await cli.runAndVerify(['add', '--text-only', testDbPath, tempDir]);

    const queryResult = await cli.runJson<any[]>(['print', testDbPath, '--all', '--json']);
    expect(queryResult.length).toBeGreaterThanOrEqual(2);
    expect(queryResult.every(m => m.type === 'text')).toBe(true);
  });

  test('adds files with extension filter', async ({ cli, tempDir, testDbPath, createValidVideo, createValidAudio }) => {
    createValidVideo('video.mp4');
    createValidVideo('video.mkv');
    createValidAudio('audio.mp3');

    await cli.runAndVerify(['add', '--ext', '.mp4', testDbPath, tempDir]);

    const queryResult = await cli.runJson<any[]>(['print', testDbPath, '--all', '--json']);
    expect(queryResult.length).toBe(1);
    expect(queryResult[0].path).toContain('.mp4');
  });

  test('adds files with exclude pattern', async ({ cli, tempDir, testDbPath, createValidVideo }) => {
    createValidVideo('keep_this.mp4');
    createValidVideo('exclude_this.mp4');

    await cli.runAndVerify(['add', '--exclude', 'exclude', testDbPath, tempDir]);

    const queryResult = await cli.runJson<any[]>(['print', testDbPath, '--all', '--json']);
    expect(queryResult.length).toBe(1);
    expect(queryResult[0].path).toContain('keep_this');
  });

  test('adds files with include pattern', async ({ cli, tempDir, testDbPath, createValidVideo }) => {
    createValidVideo('important_video.mp4');
    createValidVideo('other_video.mp4');

    await cli.runAndVerify(['add', '--include', 'important', testDbPath, tempDir]);

    const queryResult = await cli.runJson<any[]>(['print', testDbPath, '--all', '--json']);
    expect(queryResult.length).toBe(1);
    expect(queryResult[0].path).toContain('important');
  });

  test('adds files with size filter', async ({ cli, tempDir, testDbPath, createValidVideo }) => {
    // Create files of different sizes using real fixtures
    const small = path.join(tempDir, 'small.mp4');
    const large = path.join(tempDir, 'large.mp4');
    
    const videoFixture = path.join(__dirname, '../fixtures/media/videos/test_video1.mp4');
    const clipFixture = path.join(__dirname, '../fixtures/media/videos/test_clip1.mp4');
    
    fs.copyFileSync(clipFixture, small);
    fs.copyFileSync(videoFixture, large);

    // Note: ensure the sizes in init-db.sh are such that video1 > clip1
    // or just use any real files and adjust the filter if needed.
    // video1 is 10s, clip1 is 5s, so video1 should be larger.

    await cli.runAndVerify(['add', '--size', '>100KB', testDbPath, tempDir]);

    const queryResult = await cli.runJson<any[]>(['print', testDbPath, '--all', '--json']);
    expect(queryResult.length).toBeGreaterThan(0);
  });

  test('handles non-existent path gracefully', async ({ cli, tempDir, testDbPath }) => {
    const nonExistentPath = path.join(tempDir, 'does_not_exist.mp4');
    const result = await cli.run(['add', testDbPath, nonExistentPath]);

    expect(result.exitCode).not.toBe(0);
    expect(result.stderr).toContain('no such file or directory');
  });

  test('handles empty directory', async ({ cli, tempDir, testDbPath }) => {
    const emptyDir = path.join(tempDir, 'empty');
    fs.mkdirSync(emptyDir);

    const result = await cli.run(['add', testDbPath, emptyDir]);
    expect(result.exitCode).toBe(0);

    const queryResult = await cli.runJson<any[]>(['print', testDbPath, '--all', '--json']);
    expect(queryResult.length).toBe(0);
  });

  test('adds files with regex pattern', async ({ cli, tempDir, testDbPath, createValidVideo }) => {
    createValidVideo('movie_2023.mp4');
    createValidVideo('show_2022.mp4');

    await cli.runAndVerify(['add', '--regex', '2023', testDbPath, tempDir]);

    const queryResult = await cli.runJson<any[]>(['print', testDbPath, '--all', '--json']);
    expect(queryResult.length).toBe(1);
    expect(queryResult[0].path).toContain('2023');
  });

  test('adds files with path-contains filter', async ({ cli, tempDir, testDbPath, createValidVideo }) => {
    const subDir = path.join(tempDir, 'movies');
    fs.mkdirSync(subDir);
    const videoPath = path.join(subDir, 'video.mp4');
    const fixturePath = path.join(__dirname, '../fixtures/media/videos/test_video1.mp4');
    fs.copyFileSync(fixturePath, videoPath);

    await cli.runAndVerify(['add', '--path-contains', 'movies', testDbPath, tempDir]);

    const queryResult = await cli.runJson<any[]>(['print', testDbPath, '--all', '--json']);
    expect(queryResult.length).toBe(1);
    expect(queryResult[0].path).toContain('movies');
  });

  test('adds files with mime-type filter', async ({ cli, tempDir, testDbPath, createValidVideo, createValidAudio }) => {
    createValidVideo('video.mp4');
    createValidAudio('audio.mp3');

    await cli.runAndVerify(['add', '--mime-type', 'video', testDbPath, tempDir]);

    const queryResult = await cli.runJson<any[]>(['print', testDbPath, '--all', '--json']);
    expect(queryResult.length).toBe(1);
    expect(queryResult[0].type).toBe('video');
  });

  test('adds files with verbose output', async ({ cli, testDbPath, createValidVideo }) => {
    const videoPath = createValidVideo('verbose_test.mp4');
    const result = await cli.runAndVerify(['add', '--verbose', testDbPath, videoPath]);
    expect(result.stdout).toBeTruthy();
  });

  test('dry run does not modify database', async ({ cli, testDbPath, createValidVideo }) => {
    const videoPath = createValidVideo('dry_run.mp4');
    await cli.runAndVerify(['add', '--simulate', testDbPath, videoPath]);

    const queryResult = await cli.runJson<any[]>(['print', testDbPath, '--all', '--json']);
    expect(queryResult.length).toBe(0);
  });

  test('no-confirm flag skips confirmation', async ({ cli, testDbPath, createValidVideo }) => {
    const videoPath = createValidVideo('no_confirm.mp4');
    const result = await cli.runAndVerify(['add', '--no-confirm', '-y', testDbPath, videoPath]);
    expect(result.exitCode).toBe(0);
  });
});
