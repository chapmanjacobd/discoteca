import { test, expect } from '../fixtures-cli';
import * as fs from 'fs';
import * as path from 'path';

test.describe('CLI: Sort Commands', () => {
  test('regex-sort sorts media', async ({ cli, tempDir, testDbPath, createValidVideo }) => {
    createValidVideo('movie_2023.mp4');
    createValidVideo('movie_2022.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    const result = await cli.runAndVerify(['regex-sort', testDbPath]);
    expect(result.exitCode).toBe(0);
  });

  test('cluster-sort groups items', async ({ cli, tempDir, testDbPath, createValidVideo }) => {
    createValidVideo('movie_part1.mp4');
    createValidVideo('movie_part2.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    const result = await cli.runAndVerify(['cluster-sort', testDbPath]);
    expect(result.exitCode).toBe(0);
  });
});

test.describe('CLI: Sample-Hash Command', () => {
  test('calculates hash for file', async ({ cli, createValidVideo }) => {
    const videoPath = createValidVideo('hash_test.mp4');
    const result = await cli.runAndVerify(['sample-hash', videoPath]);
    expect(result.stdout).toBeTruthy();
  });

  test('calculates hash as JSON', async ({ cli, createValidVideo }) => {
    const videoPath = createValidVideo('hash_json.mp4');
    const result = await cli.runJson<any[]>(['sample-hash', '-j', videoPath]);
    expect(result[0].hash).toBeTruthy();
  });
});

test.describe('CLI: MPV Controls (Fail gracefully)', () => {
  test('now command fails without mpv', async ({ cli }) => {
    const result = await cli.run(['now', '--mpv-socket', '/tmp/nonexistent-socket']);
    expect(result.exitCode).not.toBe(0);
  });

  test('next command succeeds even without mpv', async ({ cli }) => {
    const result = await cli.run(['next', '--mpv-socket', '/tmp/nonexistent-socket']);
    expect(result.exitCode).toBe(0);
  });
});
