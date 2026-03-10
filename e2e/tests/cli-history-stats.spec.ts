import { test, expect } from '../fixtures-cli';
import * as fs from 'fs';
import * as path from 'path';

test.describe('CLI: History Commands', () => {
  test('adds file to history', async ({ cli, testDbPath, createValidVideo }) => {
    const videoPath = createValidVideo('history_test.mp4');
    await cli.runAndVerify(['add', testDbPath, videoPath]);

    await cli.runAndVerify(['history-add', testDbPath, videoPath]);

    const queryResult = await cli.runJson<any[]>(['print', testDbPath, '--all', '--json', '--played']);
    expect(queryResult.length).toBe(1);
  });

  test('adds file to history with done flag', async ({ cli, testDbPath, createValidVideo }) => {
    const videoPath = createValidVideo('history_done.mp4');
    await cli.runAndVerify(['add', testDbPath, videoPath]);

    await cli.runAndVerify(['history-add', '--done', testDbPath, videoPath]);

    const queryResult = await cli.runJson<any[]>(['print', testDbPath, '--all', '--json', '--completed']);
    expect(queryResult.length).toBe(1);
  });

  test('imports mpv watchlater files', async ({ cli, tempDir, testDbPath, createValidVideo }) => {
    const videoPath = createValidVideo('watchlater.mp4');
    await cli.runAndVerify(['add', testDbPath, videoPath]);

    const watchlaterDir = path.join(tempDir, 'mpv_watchlater');
    fs.mkdirSync(watchlaterDir);
    
    const crypto = require('crypto');
    const hash = crypto.createHash('md5').update(videoPath).digest('hex').toUpperCase();
    fs.writeFileSync(path.join(watchlaterDir, hash), 'start=123.456\n');

    await cli.runAndVerify(['mpv-watchlater', testDbPath, '--watch-later-dir', watchlaterDir]);

    const queryResult = await cli.runJson<any[]>(['print', testDbPath, '--all', '--json', '--played']);
    expect(queryResult.length).toBe(1);
    expect(queryResult[0].playhead).toBeCloseTo(123.456, 0);
  });
});

test.describe('CLI: Stats Command', () => {
  test('shows library statistics', async ({ cli, tempDir, testDbPath, createValidVideo }) => {
    createValidVideo('v1.mp4');
    createValidVideo('v2.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    const result = await cli.runAndVerify(['stats', 'created', testDbPath]);
    expect(result.stdout).toMatch(/\d+/);
  });

  test('shows statistics as JSON', async ({ cli, tempDir, testDbPath, createValidVideo }) => {
    createValidVideo('v1.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    const result = await cli.runJson<any>(['stats', 'created', '-j', testDbPath]);
    expect(result).toBeTruthy();
  });
});

test.describe('CLI: Optimize and Repair', () => {
  test('optimizes database', async ({ cli, tempDir, testDbPath, createValidVideo }) => {
    createValidVideo('v.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    const result = await cli.runAndVerify(['optimize', testDbPath]);
    expect(result.exitCode).toBe(0);
  });

  test('repairs database', async ({ cli, tempDir, testDbPath, createValidVideo }) => {
    createValidVideo('v.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    const result = await cli.runAndVerify(['repair', testDbPath]);
    expect(result.exitCode).toBe(0);
  });
});
