import { test, expect } from '../fixtures-cli';
import * as fs from 'fs';
import * as path from 'path';

test.describe('CLI: Open and Browse Commands', () => {
  test('browse opens URL with mock browser', async ({ cli, testDbPath, createValidVideo }) => {
    const v1 = createValidVideo('v1.mp4');
    await cli.runAndVerify(['add', testDbPath, v1]);

    const result = await cli.run(['browse', '--browser', 'echo', testDbPath]);
    expect(result.stderr).toContain('no URLs found');
  });

  test('version command shows information', async ({ cli }) => {
    const result = await cli.runAndVerify(['version']);
    expect(result.stdout).toMatch(/(version|commit|build)/i);
  });

  test('readme command generates content', async ({ cli }) => {
    const result = await cli.runAndVerify(['readme']);
    expect(result.stdout).toContain('Usage');
  });
});

test.describe('CLI: Search-DB Command', () => {
  test('searches media table', async ({ cli, testDbPath, createValidVideo }) => {
    const videoPath = createValidVideo('search_db_test.mp4');
    await cli.runAndVerify(['add', testDbPath, videoPath]);

    const result = await cli.runAndVerify(['search-db', testDbPath, 'media', '.']);
    expect(result.stdout).toContain('search_db_test.mp4');
  });

  test('searches media table with where clause', async ({ cli, testDbPath, createValidVideo }) => {
    const v1 = createValidVideo('v.mp4');
    await cli.runAndVerify(['add', testDbPath, v1]);

    const result = await cli.runAndVerify(['search-db', '-w', "type = 'video'", testDbPath, 'media', '.']);
    expect(result.stdout).toContain('v.mp4');
  });
});

test.describe('CLI: Explode Command', () => {
  test('creates symlinks', async ({ cli, tempDir }) => {
    const binDir = path.join(tempDir, 'bin');
    fs.mkdirSync(binDir);
    
    await cli.runAndVerify(['explode', binDir]);
    expect(fs.existsSync(path.join(binDir, 'add'))).toBe(true);
  });
});
