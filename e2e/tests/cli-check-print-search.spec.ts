import { test, expect } from '../fixtures-cli';
import * as fs from 'fs';
import * as path from 'path';

test.describe('CLI: Check Command', () => {
  test.describe.configure({ mode: 'serial' });
  test('marks missing files as deleted', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create and add a file
    const videoPath = createDummyVideo('to_delete.mp4');
    await cli.runAndVerify(['add', testDbPath, videoPath]);

    // Verify file is in database
    let queryResult = await cli.runJson(['print', testDbPath, '--all', '--json']);
    expect(queryResult.length).toBe(1);

    // Delete the physical file
    fs.unlinkSync(videoPath);

    // Run check command
    const checkResult = await cli.runAndVerify(['check', testDbPath, tempDir]);

    // Verify file is marked as deleted
    queryResult = await cli.runJson(['print', testDbPath, '--all', '--json']);
    expect(queryResult.length).toBe(1);
    expect(queryResult[0].time_deleted).toBeGreaterThan(0);
  });

  test('dry run does not mark files', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create and add a file
    const videoPath = createDummyVideo('dry_run_check.mp4');
    await cli.runAndVerify(['add', testDbPath, videoPath]);

    // Delete the physical file
    fs.unlinkSync(videoPath);

    // Run check command with dry-run flag
    await cli.runAndVerify(['check', '--dry-run', testDbPath, tempDir]);

    // Verify file is NOT marked as deleted
    const queryResult = await cli.runJson(['print', testDbPath, '--all', '--json']);
    expect(queryResult.length).toBe(1);
    expect(queryResult[0].time_deleted).toBe(0);
  });

  test('checks specific paths', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create subdirectories with files
    const dir1 = path.join(tempDir, 'dir1');
    const dir2 = path.join(tempDir, 'dir2');
    fs.mkdirSync(dir1);
    fs.mkdirSync(dir2);

    const video1 = createDummyVideo('video1.mp4');
    const video2 = path.join(dir2, 'video2.mp4');
    fs.writeFileSync(video2, Buffer.alloc(1024));

    // Add both files
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Delete one file
    fs.unlinkSync(video1);

    // Check only dir2 (should not mark video1 as deleted)
    await cli.runAndVerify(['check', testDbPath, dir2]);

    // Verify video1 is NOT marked as deleted (not in checked path)
    const queryResult = await cli.runJson(['print', testDbPath, '--all', '--json']);
    const deletedCount = queryResult.filter((m: any) => m.time_deleted > 0).length;
    expect(deletedCount).toBe(0);
  });

  test('handles non-existent database', async ({ cli, tempDir }) => {
    const nonExistentDb = path.join(tempDir, 'non_existent.db');

    const result = await cli.run(['check', nonExistentDb, tempDir]);

    // Should fail gracefully
    expect(result.exitCode).not.toBe(0);
  });
});

test.describe('CLI: Print Command', () => {
  test('prints all media', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Add files
    createDummyVideo('video1.mp4');
    createDummyVideo('video2.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Print all
    const result = await cli.runAndVerify(['print', testDbPath, '--all']);

    // Should contain both files
    expect(result.stdout).toContain('video1.mp4');
    expect(result.stdout).toContain('video2.mp4');
  });

  test('prints media as JSON', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Add file
    createDummyVideo('json_test.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Print as JSON
    const result = await cli.runJson(['print', testDbPath, '--all']);

    expect(Array.isArray(result)).toBe(true);
    expect(result.length).toBe(1);
    expect(result[0].path).toContain('json_test.mp4');
  });

  test('prints media with custom columns', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Add file
    createDummyVideo('columns_test.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Print with specific columns
    const result = await cli.runAndVerify(['print', testDbPath, '--all', '-c', 'path,size']);

    // Should only show path and size
    expect(result.stdout).toContain('columns_test.mp4');
  });

  test('prints media sorted by size', async ({ cli, tempDir, testDbPath, createDummyFile }) => {
    // Add files of different sizes
    createDummyFile('small.bin', 100);
    createDummyFile('large.bin', 10000);
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Print sorted by size
    const result = await cli.runAndVerify(['print', testDbPath, '--all', '-u', 'size']);

    // Verify order (large should come first in descending order)
    const lines = result.stdout.split('\n').filter(l => l.includes('.bin'));
    expect(lines[0]).toContain('large.bin');
  });

  test('prints media sorted by size reversed', async ({ cli, tempDir, testDbPath, createDummyFile }) => {
    // Add files of different sizes
    createDummyFile('small.bin', 100);
    createDummyFile('large.bin', 10000);
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Print sorted by size reversed
    const result = await cli.runAndVerify(['print', testDbPath, '--all', '-u', 'size', '-V']);

    // Verify order (small should come first in ascending order)
    const lines = result.stdout.split('\n').filter(l => l.includes('.bin'));
    expect(lines[0]).toContain('small.bin');
  });

  test('prints media with limit', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Add multiple files
    createDummyVideo('video1.mp4');
    createDummyVideo('video2.mp4');
    createDummyVideo('video3.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Print with limit
    const result = await cli.runAndVerify(['print', testDbPath, '-L', '2']);

    // Should only show 2 files
    const lines = result.stdout.split('\n').filter(l => l.includes('.mp4'));
    expect(lines.length).toBeLessThanOrEqual(2);
  });

  test('prints media with offset', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Add multiple files
    createDummyVideo('video1.mp4');
    createDummyVideo('video2.mp4');
    createDummyVideo('video3.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Print with offset
    const result = await cli.runAndVerify(['print', testDbPath, '--all', '--offset', '2']);

    // Should skip first 2 files
    const lines = result.stdout.split('\n').filter(l => l.includes('.mp4'));
    expect(lines.length).toBe(1);
  });

  test('prints media filtered by search', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Add files
    createDummyVideo('movie_action.mp4');
    createDummyVideo('show_comedy.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Search for "movie"
    const result = await cli.runAndVerify(['print', testDbPath, '--search', 'movie']);

    // Should only show movie
    expect(result.stdout).toContain('movie_action');
    expect(result.stdout).not.toContain('show_comedy');
  });

  test('prints media filtered by size range', async ({ cli, tempDir, testDbPath, createDummyFile }) => {
    // Add files of different sizes
    createDummyFile('small.bin', 100);
    createDummyFile('medium.bin', 5000);
    createDummyFile('large.bin', 10000);
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Filter by size range
    const result = await cli.runAndVerify(['print', testDbPath, '--all', '-S', '>1KB']);

    // Should only show medium and large
    expect(result.stdout).toContain('medium.bin');
    expect(result.stdout).toContain('large.bin');
    expect(result.stdout).not.toContain('small.bin');
  });

  test('prints media filtered by duration range', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Add file
    createDummyVideo('duration_test.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Filter by duration (should work even without duration metadata)
    const result = await cli.runAndVerify(['print', testDbPath, '--all', '-d', '>0']);

    // Command should succeed
    expect(result.exitCode).toBe(0);
  });

  test('prints media with random order', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Add multiple files
    createDummyVideo('video1.mp4');
    createDummyVideo('video2.mp4');
    createDummyVideo('video3.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Print in random order
    const result = await cli.runAndVerify(['print', testDbPath, '--all', '-r']);

    // Should show all files
    expect(result.stdout).toContain('video1.mp4');
    expect(result.stdout).toContain('video2.mp4');
    expect(result.stdout).toContain('video3.mp4');
  });

  test('prints media summarized', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Add files
    createDummyVideo('video1.mp4');
    createDummyVideo('video2.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Print summarized
    const result = await cli.runAndVerify(['print', testDbPath, '--all', '--summarize']);

    // Should show summary statistics
    expect(result.stdout).toBeTruthy();
  });

  test('prints big directories', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create subdirectory structure
    const subDir = path.join(tempDir, 'subdir');
    fs.mkdirSync(subDir);
    createDummyVideo('subdir/video.mp4');
    createDummyVideo('root_video.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Print big dirs
    const result = await cli.runAndVerify(['print', testDbPath, '-B']);

    // Should show directory aggregation
    expect(result.stdout).toBeTruthy();
  });

  test('prints media grouped by extension', async ({ cli, tempDir, testDbPath, createDummyVideo, createDummyAudio }) => {
    // Add files of different types
    createDummyVideo('video.mp4');
    createDummyAudio('audio.mp3');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Print grouped by extension
    const result = await cli.runAndVerify(['print', testDbPath, '--all', '--group-by-extensions']);

    // Should show grouped results
    expect(result.stdout).toBeTruthy();
  });

  test('prints media with natural sort', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Add files with numbers in names
    createDummyVideo('video1.mp4');
    createDummyVideo('video10.mp4');
    createDummyVideo('video2.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Print with natural sort
    const result = await cli.runAndVerify(['print', testDbPath, '--all', '-n']);

    // Natural sort should order: video1, video2, video10
    const lines = result.stdout.split('\n').filter(l => l.includes('video'));
    expect(lines.map(l => l.match(/video(\d+)/)?.[1])).toEqual(['1', '2', '10']);
  });

  test('prints only existing files', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Add file
    const videoPath = createDummyVideo('exists_test.mp4');
    await cli.runAndVerify(['add', testDbPath, videoPath]);

    // Delete the file
    fs.unlinkSync(videoPath);

    // Print only existing
    const result = await cli.runAndVerify(['print', testDbPath, '--all', '--exists']);

    // Should not show deleted file
    expect(result.stdout).not.toContain('exists_test.mp4');
  });

  test('prints only deleted files', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Add file
    const videoPath = createDummyVideo('deleted_test.mp4');
    await cli.runAndVerify(['add', testDbPath, videoPath]);

    // Delete and check
    fs.unlinkSync(videoPath);
    await cli.runAndVerify(['check', testDbPath, tempDir]);

    // Print only deleted
    const result = await cli.runAndVerify(['print', testDbPath, '--only-deleted']);

    // Should show deleted file
    expect(result.stdout).toContain('deleted_test.mp4');
  });

  test('hides deleted files', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Add file
    const videoPath = createDummyVideo('hide_deleted_test.mp4');
    await cli.runAndVerify(['add', testDbPath, videoPath]);

    // Delete and check
    fs.unlinkSync(videoPath);
    await cli.runAndVerify(['check', testDbPath, tempDir]);

    // Print hiding deleted
    const result = await cli.runAndVerify(['print', testDbPath, '--all', '--hide-deleted']);

    // Should not show deleted file
    expect(result.stdout).not.toContain('hide_deleted_test.mp4');
  });

  test('prints media with where clause', async ({ cli, tempDir, testDbPath, createDummyVideo, createDummyAudio }) => {
    // Add files
    createDummyVideo('video.mp4');
    createDummyAudio('audio.mp3');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Print with where clause
    const result = await cli.runAndVerify(['print', testDbPath, '--all', '-w', "type = 'video'"]);

    // Should only show video
    expect(result.stdout).toContain('video.mp4');
    expect(result.stdout).not.toContain('audio.mp3');
  });

  test('prints media by category', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Add file
    createDummyVideo('movie.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Print by category (should work even without categories set)
    const result = await cli.runAndVerify(['print', testDbPath, '--all', '--category', 'video']);

    // Command should succeed
    expect(result.exitCode).toBe(0);
  });

  test('prints media with created after filter', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Add file
    createDummyVideo('recent.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Print with date filter
    const result = await cli.runAndVerify(['print', testDbPath, '--all', '--created-after', '2020-01-01']);

    // Should show recently added file
    expect(result.stdout).toContain('recent.mp4');
  });

  test('prints related media', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Add files with similar names
    createDummyVideo('movie_part1.mp4');
    createDummyVideo('movie_part2.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Print related to first result
    const result = await cli.runAndVerify(['print', testDbPath, '--search', 'movie', '-R']);

    // Should find related files
    expect(result.stdout).toContain('movie');
  });
});

test.describe('CLI: Search Command', () => {
  test('searches media by title', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Add files
    createDummyVideo('matrix_movie.mp4');
    createDummyVideo('other_movie.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Search for "matrix"
    const result = await cli.runAndVerify(['search', testDbPath, 'matrix']);

    // Should find matrix
    expect(result.stdout).toContain('matrix_movie');
  });

  test('searches with multiple terms', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Add files
    createDummyVideo('action_movie.mp4');
    createDummyVideo('comedy_movie.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Search for multiple terms
    const result = await cli.runAndVerify(['search', testDbPath, 'action movie']);

    // Should find action movie
    expect(result.stdout).toContain('action_movie');
  });

  test('searches with OR terms', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Add files
    createDummyVideo('action_movie.mp4');
    createDummyVideo('comedy_movie.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Search with OR
    const result = await cli.runAndVerify(['search', testDbPath, 'action|comedy']);

    // Should find both
    expect(result.stdout).toContain('action_movie');
    expect(result.stdout).toContain('comedy_movie');
  });

  test('searches with video-only filter', async ({ cli, tempDir, testDbPath, createDummyVideo, createDummyAudio }) => {
    // Add files
    createDummyVideo('video.mp4');
    createDummyAudio('audio.mp3');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Search with video-only filter
    const result = await cli.runAndVerify(['search', '--video-only', testDbPath, '']);

    // Should only show video
    expect(result.stdout).toContain('video.mp4');
    expect(result.stdout).not.toContain('audio.mp3');
  });

  test('searches with limit', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Add multiple files
    for (let i = 0; i < 10; i++) {
      createDummyVideo(`video${i}.mp4`);
    }
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Search with limit
    const result = await cli.runAndVerify(['search', '-L', '5', testDbPath, 'video']);

    // Should limit results
    const lines = result.stdout.split('\n').filter(l => l.includes('.mp4'));
    expect(lines.length).toBeLessThanOrEqual(5);
  });

  test('searches as JSON', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Add file
    createDummyVideo('json_search.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Search as JSON
    const result = await cli.runJson(['search', '-j', testDbPath, 'json']);

    expect(Array.isArray(result)).toBe(true);
    expect(result.length).toBe(1);
  });

  test('searches with exact match', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Add files
    createDummyVideo('exact.mp4');
    createDummyVideo('exact_match.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Search with exact match
    const result = await cli.runAndVerify(['search', '--exact', testDbPath, 'exact']);

    // Should only find exact match
    const lines = result.stdout.split('\n').filter(l => l.includes('.mp4'));
    expect(lines.length).toBe(1);
    expect(lines[0]).toContain('exact.mp4');
  });

  test('searches with flexible search', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Add files
    createDummyVideo('matrix_movie.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Search with flexible/fuzzy match
    const result = await cli.runAndVerify(['search', '--flexible-search', testDbPath, 'mtrix']);

    // Should find fuzzy match
    expect(result.stdout).toContain('matrix_movie');
  });
});
