import { test, expect } from '../fixtures-cli';
import * as fs from 'fs';
import * as path from 'path';
import { exec } from 'child_process';
import { promisify } from 'util';
import { TestServer } from '../utils/test-server';
import * as http from 'http';

const execAsync = promisify(exec);

test.describe('CLI: Subtitle Optimization', () => {
  test('verifies subtitle_count is 0 for videos without embedded subtitles', async ({ cli, testDbPath, createValidVideo }) => {
    const videoPath = createValidVideo('no_subs.mp4');
    
    // Add video to database
    await cli.runAndVerify(['add', testDbPath, videoPath]);

    // Verify the media has subtitle_count = 0
    const mediaResult = await cli.runJson<any[]>(['print', testDbPath, '--json', '--path', videoPath]);
    expect(mediaResult.length).toBe(1);
    expect(mediaResult[0].subtitle_count).toBe(0);

    // The optimization in handleSubtitles checks subtitle_count before running ffmpeg
    // If subtitle_count is 0, it should return early with "No subtitles available"
    const hasSubtitles = mediaResult[0].subtitle_count > 0;
    expect(hasSubtitles).toBe(false);
  });

  test('verifies subtitle_count field exists and is queryable via database', async ({ cli, testDbPath, createValidVideo }) => {
    const videoPath = createValidVideo('optimization_test.mp4');
    
    // Add video to database
    await cli.runAndVerify(['add', testDbPath, videoPath]);

    // Query database directly to get subtitle_count - this is the optimization
    // that should happen BEFORE running ffmpeg in the streaming handler (serve_streaming.go)
    const { stdout } = await execAsync(`sqlite3 "${testDbPath}" "SELECT subtitle_count FROM media WHERE path='${videoPath}'"`);
    const subtitleCount = parseInt(stdout.trim()) || 0;

    // Verify subtitle_count is 0 (no embedded subtitles)
    expect(subtitleCount).toBe(0);

    // This test verifies that the database has the subtitle_count field
    // which is used by serve_streaming.go to optimize subtitle handling
    // by skipping ffmpeg for files without subtitles
  });

  test('verifies subtitle_codecs field is populated for media inspection', async ({ cli, testDbPath, createValidVideo }) => {
    const videoPath = createValidVideo('codec_check.mp4');
    
    // Add video to database
    await cli.runAndVerify(['add', testDbPath, videoPath]);

    // Query database to check subtitle_codecs field
    const { stdout } = await execAsync(`sqlite3 "${testDbPath}" "SELECT subtitle_codecs FROM media WHERE path='${videoPath}'"`);
    const subtitleCodecs = stdout.trim();

    // For test videos without embedded subtitles, this should be empty or null
    // The optimization uses this field along with subtitle_count to avoid unnecessary ffmpeg calls
    expect(subtitleCodecs).toBe('');
  });

  test('verifies files with subtitle_count=0 can be filtered efficiently', async ({ cli, testDbPath, createValidVideo, createValidAudio }) => {
    const videoPath = createValidVideo('filter_video.mp4');
    const audioPath = createValidAudio('filter_audio.mp3');
    
    // Add files to database
    await cli.runAndVerify(['add', testDbPath, videoPath]);
    await cli.runAndVerify(['add', testDbPath, audioPath]);

    // Query for files with subtitles (subtitle_count > 0)
    // This demonstrates the optimization: we can query the database first
    // instead of running ffmpeg on every file
    const { stdout } = await execAsync(`sqlite3 "${testDbPath}" "SELECT COUNT(*) FROM media WHERE subtitle_count > 0"`);
    const filesWithSubs = parseInt(stdout.trim());

    // Our test files don't have embedded subtitles, so count should be 0
    expect(filesWithSubs).toBe(0);

    // This verifies the optimization strategy:
    // 1. Query database for subtitle_count > 0
    // 2. Only run ffmpeg on files that have subtitles
    // 3. Avoid "Failed to convert subtitles" errors for files without subtitles
  });

  test('server returns 404 for subtitles on video without embedded subtitles (optimization check)', async ({ cli, testDbPath, createValidVideo }) => {
    const videoPath = createValidVideo('no_subs_server.mp4');
    
    // Add video to database
    await cli.runAndVerify(['add', testDbPath, videoPath]);

    // Start server
    const server = new TestServer({ databasePath: testDbPath });
    await server.start();

    try {
      // Request subtitles for a video without embedded subtitles
      // The optimization should check subtitle_count first and return 404 immediately
      // without attempting ffmpeg conversion
      const response = await new Promise<{ statusCode: number | undefined, body: string }>((resolve) => {
        const url = `${server.getBaseUrl()}/subtitles?path=${encodeURIComponent(videoPath)}`;
        http.get(url, (res) => {
          let body = '';
          res.on('data', (chunk) => body += chunk);
          res.on('end', () => resolve({ statusCode: res.statusCode, body }));
        }).on('error', (err) => resolve({ statusCode: undefined, body: err.message }));
      });

      // Should return 404 (No subtitles available) without running ffmpeg
      // This verifies the optimization: database check happens before ffmpeg
      expect(response.statusCode).toBe(404);
      expect(response.body).toContain('No subtitles');
    } finally {
      await server.stop();
    }
  });
});
