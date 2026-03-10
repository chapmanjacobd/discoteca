import { test, expect } from '../fixtures-cli';
import * as fs from 'fs';
import * as path from 'path';

test.describe('CLI: Check Command', () => {
  test('marks missing files as deleted', async ({ cli, tempDir, testDbPath, createValidVideo }) => {
    const videoPath = createValidVideo('to_delete.mp4');
    await cli.runAndVerify(['add', testDbPath, videoPath]);

    fs.unlinkSync(videoPath);
    await cli.runAndVerify(['check', testDbPath]);

    const queryResult = await cli.runJson<any[]>(['print', testDbPath, '--all', '--only-deleted', '--json']);
    expect(queryResult.length).toBe(1);
    expect(queryResult[0].time_deleted).toBeGreaterThan(0);
  });

  test('dry run does not mark files', async ({ cli, tempDir, testDbPath, createValidVideo }) => {
    const videoPath = createValidVideo('dry_run_check.mp4');
    await cli.runAndVerify(['add', testDbPath, videoPath]);

    fs.unlinkSync(videoPath);
    await cli.runAndVerify(['check', '--dry-run', testDbPath]);

    const queryResult = await cli.runJson<any[]>(['print', testDbPath, '--all', '--json']);
    expect(queryResult.length).toBe(1);
    expect(queryResult[0].time_deleted).toBe(0);
  });
});

test.describe('CLI: Print Command', () => {
  test('prints all media', async ({ cli, tempDir, testDbPath, createValidVideo }) => {
    createValidVideo('v1.mp4');
    createValidVideo('v2.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    const result = await cli.runAndVerify(['print', testDbPath, '--all']);
    expect(result.stdout).toContain('v1.mp4');
    expect(result.stdout).toContain('v2.mp4');
  });

  test('prints media sorted by size', async ({ cli, tempDir, testDbPath, createValidVideo }) => {
    const small = path.join(tempDir, 'small.mp4');
    const large = path.join(tempDir, 'large.mp4');
    
    const videoFixture = path.join(__dirname, '../fixtures/media/videos/test_video1.mp4');
    const clipFixture = path.join(__dirname, '../fixtures/media/videos/test_clip1.mp4');
    
    fs.copyFileSync(clipFixture, small);
    fs.copyFileSync(videoFixture, large);
    
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    const result = await cli.runAndVerify(['print', testDbPath, '--all', '--sort-by', 'size']);
    const smallIdx = result.stdout.indexOf('small.mp4');
    const largeIdx = result.stdout.indexOf('large.mp4');
    expect(smallIdx).toBeLessThan(largeIdx);
  });

  test('prints only existing files', async ({ cli, tempDir, testDbPath, createValidVideo }) => {
    const v1 = createValidVideo('v1.mp4');
    const v2 = createValidVideo('v2.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    fs.unlinkSync(v1);
    const result = await cli.runAndVerify(['print', testDbPath, '--all', '--exists']);
    expect(result.stdout).not.toContain('v1.mp4');
    expect(result.stdout).toContain('v2.mp4');
  });
});

test.describe('CLI: Search Command', () => {
  test('searches media by title', async ({ cli, tempDir, testDbPath, createValidVideo }) => {
    createValidVideo('matrix_movie.mp4');
    createValidVideo('other.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    const result = await cli.runAndVerify(['search', testDbPath, '--search', 'matrix']);
    expect(result.stdout).toContain('matrix_movie');
    expect(result.stdout).not.toContain('other.mp4');
  });

  test('searches with exact match', async ({ cli, tempDir, testDbPath, createValidVideo }) => {
    createValidVideo('exact.mp4');
    createValidVideo('exact_match.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    const result = await cli.runAndVerify(['search', '--exact', testDbPath, '--search', 'exact']);
    expect(result.stdout).toContain('exact.mp4');
    expect(result.stdout).not.toContain('exact_match.mp4');
  });
});
