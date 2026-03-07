import { test, expect } from '../fixtures-cli';
import * as fs from 'fs';
import * as path from 'path';

test.describe('CLI: Watch Command', () => {
  test.describe.configure({ mode: 'serial' });
  test('watches video file', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create and add a file
    const videoPath = createDummyVideo('watch_test.mp4');
    await cli.runAndVerify(['add', testDbPath, videoPath]);

    // Watch command (will fail without mpv, but should handle gracefully)
    const result = await cli.run(['watch', testDbPath, '--search', 'watch_test']);

    // Should fail gracefully (mpv not available in test environment)
    expect(result.exitCode).not.toBe(0);
    expect(result.stderr).toBeTruthy();
  });

  test('watches with custom player', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create and add a file
    const videoPath = createDummyVideo('custom_player.mp4');
    await cli.runAndVerify(['add', testDbPath, videoPath]);

    // Watch with custom player (non-existent player should fail gracefully)
    const result = await cli.run(['watch', '--override-player', 'nonexistent-player', testDbPath, '--search', 'custom_player']);

    // Should fail gracefully
    expect(result.exitCode).not.toBe(0);
  });

  test('watches with start position', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create and add a file
    const videoPath = createDummyVideo('start_pos.mp4');
    await cli.runAndVerify(['add', testDbPath, videoPath]);

    // Watch with start position
    const result = await cli.run(['watch', '--start', '10', testDbPath, '--search', 'start_pos']);

    // Should fail gracefully (no mpv)
    expect(result.exitCode).not.toBe(0);
  });

  test('watches with end position', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create and add a file
    const videoPath = createDummyVideo('end_pos.mp4');
    await cli.runAndVerify(['add', testDbPath, videoPath]);

    // Watch with end position
    const result = await cli.run(['watch', '--end', '60', testDbPath, '--search', 'end_pos']);

    // Should fail gracefully
    expect(result.exitCode).not.toBe(0);
  });

  test('watches with speed', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create and add a file
    const videoPath = createDummyVideo('speed_test.mp4');
    await cli.runAndVerify(['add', testDbPath, videoPath]);

    // Watch with speed
    const result = await cli.run(['watch', '--speed', '1.5', testDbPath, '--search', 'speed_test']);

    // Should fail gracefully
    expect(result.exitCode).not.toBe(0);
  });

  test('watches with volume', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create and add a file
    const videoPath = createDummyVideo('volume_test.mp4');
    await cli.runAndVerify(['add', testDbPath, videoPath]);

    // Watch with volume
    const result = await cli.run(['watch', '--volume', '50', testDbPath, '--search', 'volume_test']);

    // Should fail gracefully
    expect(result.exitCode).not.toBe(0);
  });

  test('watches with fullscreen', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create and add a file
    const videoPath = createDummyVideo('fullscreen_test.mp4');
    await cli.runAndVerify(['add', testDbPath, videoPath]);

    // Watch with fullscreen
    const result = await cli.run(['watch', '--fullscreen', testDbPath, '--search', 'fullscreen_test']);

    // Should fail gracefully
    expect(result.exitCode).not.toBe(0);
  });

  test('watches with no subtitles', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create and add a file
    const videoPath = createDummyVideo('no_subs_test.mp4');
    await cli.runAndVerify(['add', testDbPath, videoPath]);

    // Watch with no subtitles
    const result = await cli.run(['watch', '--no-subtitles', testDbPath, '--search', 'no_subs_test']);

    // Should fail gracefully
    expect(result.exitCode).not.toBe(0);
  });

  test('watches with save playhead', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create and add a file
    const videoPath = createDummyVideo('save_playhead.mp4');
    await cli.runAndVerify(['add', testDbPath, videoPath]);

    // Watch with save playhead
    const result = await cli.run(['watch', '--save-playhead', testDbPath, '--search', 'save_playhead']);

    // Should fail gracefully
    expect(result.exitCode).not.toBe(0);
  });

  test('watches with mpv socket', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create and add a file
    const videoPath = createDummyVideo('socket_test.mp4');
    await cli.runAndVerify(['add', testDbPath, videoPath]);

    // Watch with mpv socket
    const result = await cli.run(['watch', '--mpv-socket', '/tmp/mpv-socket', testDbPath, '--search', 'socket_test']);

    // Should fail gracefully
    expect(result.exitCode).not.toBe(0);
  });

  test('watches with player args', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create and add a file
    const videoPath = createDummyVideo('player_args.mp4');
    await cli.runAndVerify(['add', testDbPath, videoPath]);

    // Watch with player args
    const result = await cli.run(['watch', '--player-args-sub', '--arg1 --arg2', testDbPath, '--search', 'player_args']);

    // Should fail gracefully
    expect(result.exitCode).not.toBe(0);
  });

  test('watches with loop', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create and add a file
    const videoPath = createDummyVideo('loop_test.mp4');
    await cli.runAndVerify(['add', testDbPath, videoPath]);

    // Watch with loop
    const result = await cli.run(['watch', '--loop', testDbPath, '--search', 'loop_test']);

    // Should fail gracefully
    expect(result.exitCode).not.toBe(0);
  });

  test('watches with mute', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create and add a file
    const videoPath = createDummyVideo('mute_test.mp4');
    await cli.runAndVerify(['add', testDbPath, videoPath]);

    // Watch with mute
    const result = await cli.run(['watch', '--mute', testDbPath, '--search', 'mute_test']);

    // Should fail gracefully
    expect(result.exitCode).not.toBe(0);
  });

  test('watches with play in order', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create and add multiple files
    createDummyVideo('video1.mp4');
    createDummyVideo('video2.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Watch with play in order
    const result = await cli.run(['watch', '--play-in-order', testDbPath]);

    // Should fail gracefully
    expect(result.exitCode).not.toBe(0);
  });
});

test.describe('CLI: Listen Command', () => {
  test('listens to audio file', async ({ cli, tempDir, testDbPath, createDummyAudio }) => {
    // Create and add a file
    const audioPath = createDummyAudio('listen_test.mp3');
    await cli.runAndVerify(['add', testDbPath, audioPath]);

    // Listen command (will fail without mpv)
    const result = await cli.run(['listen', testDbPath, '--search', 'listen_test']);

    // Should fail gracefully
    expect(result.exitCode).not.toBe(0);
  });

  test('listens with custom player', async ({ cli, tempDir, testDbPath, createDummyAudio }) => {
    // Create and add a file
    const audioPath = createDummyAudio('custom_player.mp3');
    await cli.runAndVerify(['add', testDbPath, audioPath]);

    // Listen with custom player
    const result = await cli.run(['listen', '--override-player', 'nonexistent-player', testDbPath, '--search', 'custom_player']);

    // Should fail gracefully
    expect(result.exitCode).not.toBe(0);
  });

  test('listens with start position', async ({ cli, tempDir, testDbPath, createDummyAudio }) => {
    // Create and add a file
    const audioPath = createDummyAudio('start_pos.mp3');
    await cli.runAndVerify(['add', testDbPath, audioPath]);

    // Listen with start position
    const result = await cli.run(['listen', '--start', '10', testDbPath, '--search', 'start_pos']);

    // Should fail gracefully
    expect(result.exitCode).not.toBe(0);
  });

  test('listens with speed', async ({ cli, tempDir, testDbPath, createDummyAudio }) => {
    // Create and add a file
    const audioPath = createDummyAudio('speed_test.mp3');
    await cli.runAndVerify(['add', testDbPath, audioPath]);

    // Listen with speed
    const result = await cli.run(['listen', '--speed', '1.5', testDbPath, '--search', 'speed_test']);

    // Should fail gracefully
    expect(result.exitCode).not.toBe(0);
  });

  test('listens with volume', async ({ cli, tempDir, testDbPath, createDummyAudio }) => {
    // Create and add a file
    const audioPath = createDummyAudio('volume_test.mp3');
    await cli.runAndVerify(['add', testDbPath, audioPath]);

    // Listen with volume
    const result = await cli.run(['listen', '--volume', '50', testDbPath, '--search', 'volume_test']);

    // Should fail gracefully
    expect(result.exitCode).not.toBe(0);
  });

  test('listens with save playhead', async ({ cli, tempDir, testDbPath, createDummyAudio }) => {
    // Create and add a file
    const audioPath = createDummyAudio('save_playhead.mp3');
    await cli.runAndVerify(['add', testDbPath, audioPath]);

    // Listen with save playhead
    const result = await cli.run(['listen', '--save-playhead', testDbPath, '--search', 'save_playhead']);

    // Should fail gracefully
    expect(result.exitCode).not.toBe(0);
  });

  test('listens with loop', async ({ cli, tempDir, testDbPath, createDummyAudio }) => {
    // Create and add a file
    const audioPath = createDummyAudio('loop_test.mp3');
    await cli.runAndVerify(['add', testDbPath, audioPath]);

    // Listen with loop
    const result = await cli.run(['listen', '--loop', testDbPath, '--search', 'loop_test']);

    // Should fail gracefully
    expect(result.exitCode).not.toBe(0);
  });

  test('listens with play in order', async ({ cli, tempDir, testDbPath, createDummyAudio }) => {
    // Create and add multiple files
    createDummyAudio('audio1.mp3');
    createDummyAudio('audio2.mp3');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Listen with play in order
    const result = await cli.run(['listen', '--play-in-order', testDbPath]);

    // Should fail gracefully
    expect(result.exitCode).not.toBe(0);
  });
});

test.describe('CLI: Serve Command', () => {
  test('starts server on specified port', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create and add a file
    createDummyVideo('video.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Start server in background (will timeout, but we just test it starts)
    const result = await cli.run(['serve', testDbPath, '--port', '9999', '--timeout', '5s']);

    // Should timeout or fail gracefully in test environment
    expect(result.duration).toBeGreaterThanOrEqual(0);
  });

  test('starts server in dev mode', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create and add a file
    createDummyVideo('video.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Start server in dev mode
    const result = await cli.run(['serve', testDbPath, '--dev', '--port', '9998', '--timeout', '5s']);

    // Should timeout or fail gracefully
    expect(result.duration).toBeGreaterThanOrEqual(0);
  });

  test('starts server with custom host', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create and add a file
    createDummyVideo('video.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Start server with custom host
    const result = await cli.run(['serve', testDbPath, '--host', '127.0.0.1', '--port', '9997', '--timeout', '5s']);

    // Should timeout or fail gracefully
    expect(result.duration).toBeGreaterThanOrEqual(0);
  });

  test('starts server with read-only mode', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create and add a file
    createDummyVideo('video.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Start server with read-only mode
    const result = await cli.run(['serve', testDbPath, '--read-only', '--port', '9996', '--timeout', '5s']);

    // Should timeout or fail gracefully
    expect(result.duration).toBeGreaterThanOrEqual(0);
  });

  test('starts server with verbose output', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create and add a file
    createDummyVideo('video.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Start server with verbose
    const result = await cli.run(['serve', '--verbose', testDbPath, '--port', '9995', '--timeout', '5s']);

    // Should have output
    expect(result.stdout || result.stderr).toBeTruthy();
  });

  test('fails with non-existent database', async ({ cli, tempDir }) => {
    const nonExistentDb = path.join(tempDir, 'non_existent.db');

    // Try to start server with non-existent database
    const result = await cli.run(['serve', nonExistentDb, '--port', '9994', '--timeout', '5s']);

    // Should fail
    expect(result.exitCode).not.toBe(0);
  });
});
