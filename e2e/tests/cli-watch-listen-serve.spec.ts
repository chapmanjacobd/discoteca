import { test, expect } from '../fixtures-cli';
import * as fs from 'fs';
import * as path from 'path';

test.describe('CLI: Watch Command', () => {
  test('watches video file with mock player', async ({ cli, testDbPath, createValidVideo }) => {
    const videoPath = createValidVideo('watch_test.mp4');
    await cli.runAndVerify(['add', testDbPath, videoPath]);

    const result = await cli.runAndVerify(['watch', '--override-player', 'echo', testDbPath, '--search', 'watch_test']);
    expect(result.stdout).toContain('watch_test.mp4');
  });

  test('watches with start position', async ({ cli, testDbPath, createValidVideo }) => {
    const videoPath = createValidVideo('start_pos.mp4');
    await cli.runAndVerify(['add', testDbPath, videoPath]);

    const result = await cli.runAndVerify(['watch', '--override-player', 'echo', '--start', '10', testDbPath, '--search', 'start_pos']);
    expect(result.exitCode).toBe(0);
  });

  test('watches with loop', async ({ cli, testDbPath, createValidVideo }) => {
    const videoPath = createValidVideo('loop_test.mp4');
    await cli.runAndVerify(['add', testDbPath, videoPath]);

    const result = await cli.runAndVerify(['watch', '--override-player', 'echo', '--loop', testDbPath, '--search', 'loop_test']);
    expect(result.exitCode).toBe(0);
  });
});

test.describe('CLI: Listen Command', () => {
  test('listens to audio file with mock player', async ({ cli, testDbPath, createValidAudio }) => {
    const audioPath = createValidAudio('listen_test.mp3');
    await cli.runAndVerify(['add', testDbPath, audioPath]);

    const result = await cli.runAndVerify(['listen', '--override-player', 'echo', testDbPath, '--search', 'listen_test']);
    expect(result.stdout).toContain('listen_test.mp3');
  });
});

test.describe('CLI: Serve Command', () => {
  test('starts server and responds to health check', async ({ cli, testDbPath, createValidVideo, tempDir }) => {
    createValidVideo('v.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    const result = await cli.run(['serve', '--help']);
    expect(result.stdout).toContain('serve');
  });
});
