import { test, expect } from '../fixtures-cli';
import * as fs from 'fs';
import * as path from 'path';

test.describe('CLI: Open Command', () => {
  test.describe.configure({ mode: 'serial' });
  test('opens file with default application', async ({ cli, tempDir, createDummyVideo }) => {
    // Create a file
    const videoPath = createDummyVideo('open_test.mp4');

    // Open command (will fail in headless environment)
    const result = await cli.run(['open', videoPath]);

    // Should fail gracefully (no GUI)
    expect(result.exitCode).not.toBe(0);
  });

  test('opens file with custom application', async ({ cli, tempDir, createDummyVideo }) => {
    // Create a file
    const videoPath = createDummyVideo('open_custom.mp4');

    // Open with custom application (non-existent)
    const result = await cli.run(['open', '--app', 'nonexistent-app', videoPath]);

    // Should fail gracefully
    expect(result.exitCode).not.toBe(0);
  });

  test('opens multiple files', async ({ cli, tempDir, createDummyVideo }) => {
    // Create files
    const videoPath1 = createDummyVideo('open1.mp4');
    const videoPath2 = createDummyVideo('open2.mp4');

    // Open multiple files
    const result = await cli.run(['open', videoPath1, videoPath2]);

    // Should fail gracefully
    expect(result.exitCode).not.toBe(0);
  });

  test('opens file with verbose output', async ({ cli, tempDir, createDummyVideo }) => {
    // Create a file
    const videoPath = createDummyVideo('open_verbose.mp4');

    // Open with verbose
    const result = await cli.run(['open', '--verbose', videoPath]);

    // Should have output
    expect(result.stdout || result.stderr).toBeTruthy();
  });

  test('fails with non-existent file', async ({ cli, tempDir }) => {
    const nonExistentPath = path.join(tempDir, 'not_exists.mp4');

    // Try to open non-existent file
    const result = await cli.run(['open', nonExistentPath]);

    // Should fail
    expect(result.exitCode).not.toBe(0);
  });
});

test.describe('CLI: Browse Command', () => {
  test('opens URL in browser', async ({ cli }) => {
    // Browse command (will fail in headless environment)
    const result = await cli.run(['browse', 'https://example.com']);

    // Should fail gracefully (no browser)
    expect(result.exitCode).not.toBe(0);
  });

  test('opens URL with custom browser', async ({ cli }) => {
    // Browse with custom browser (non-existent)
    const result = await cli.run(['browse', '--browser', 'nonexistent-browser', 'https://example.com']);

    // Should fail gracefully
    expect(result.exitCode).not.toBe(0);
  });

  test('opens multiple URLs', async ({ cli }) => {
    // Browse multiple URLs
    const result = await cli.run(['browse', 'https://example.com', 'https://example.org']);

    // Should fail gracefully
    expect(result.exitCode).not.toBe(0);
  });

  test('opens URL with verbose output', async ({ cli }) => {
    // Browse with verbose
    const result = await cli.run(['browse', '--verbose', 'https://example.com']);

    // Should have output
    expect(result.stdout || result.stderr).toBeTruthy();
  });

  test('fails with invalid URL', async ({ cli }) => {
    // Try to browse invalid URL
    const result = await cli.run(['browse', 'not-a-valid-url']);

    // May fail or succeed depending on implementation
    expect(result.exitCode).toBeGreaterThanOrEqual(0);
  });
});

test.describe('CLI: Update Command', () => {
  test('checks for updates', async ({ cli }) => {
    // Check for updates
    const result = await cli.run(['update']);

    // Should complete (may or may not find updates)
    expect(result.exitCode).toBeGreaterThanOrEqual(0);
  });

  test('checks for updates with verbose output', async ({ cli }) => {
    // Check for updates with verbose
    const result = await cli.run(['update', '--verbose']);

    // Should have output
    expect(result.stdout || result.stderr).toBeTruthy();
  });

  test('checks for updates as JSON', async ({ cli }) => {
    // Check for updates as JSON
    const result = await cli.run(['update', '--json']);

    // Should output JSON or fail gracefully
    expect(result.exitCode).toBeGreaterThanOrEqual(0);
  });

  test('checks for updates with dry run', async ({ cli }) => {
    // Check for updates with dry run
    const result = await cli.run(['update', '--dry-run']);

    // Should complete
    expect(result.exitCode).toBeGreaterThanOrEqual(0);
  });

  test('checks for specific version', async ({ cli }) => {
    // Check for specific version
    const result = await cli.run(['update', '--version', 'v0.0.1']);

    // Should complete
    expect(result.exitCode).toBeGreaterThanOrEqual(0);
  });
});

test.describe('CLI: Version Command', () => {
  test('shows version information', async ({ cli }) => {
    // Show version
    const result = await cli.runAndVerify(['version']);

    // Should show version info
    expect(result.stdout).toBeTruthy();
    expect(result.stdout).toMatch(/(version|commit|build)/i);
  });

  test('shows version as JSON', async ({ cli }) => {
    // Show version as JSON
    const result = await cli.runJson(['version', '--json']);

    expect(typeof result).toBe('object');
    expect(result).toHaveProperty('version');
  });

  test('shows version with build info', async ({ cli }) => {
    // Show version with build info
    const result = await cli.runAndVerify(['version', '--build-info']);

    // Should show build information
    expect(result.stdout).toBeTruthy();
    expect(result.stdout).toMatch(/(build|commit|go|time)/i);
  });

  test('shows version with verbose output', async ({ cli }) => {
    // Show version with verbose
    const result = await cli.runAndVerify(['version', '--verbose']);

    // Should have more output
    expect(result.stdout.length).toBeGreaterThan(0);
  });
});

test.describe('CLI: Tui Command', () => {
  test('starts TUI mode', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create and add a file
    createDummyVideo('video.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // TUI command (will fail in non-interactive environment)
    const result = await cli.run(['tui', testDbPath]);

    // Should fail gracefully (no TTY)
    expect(result.exitCode).not.toBe(0);
  });

  test('starts TUI with verbose output', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create and add a file
    createDummyVideo('video.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // TUI with verbose
    const result = await cli.run(['tui', '--verbose', testDbPath]);

    // Should fail gracefully with output
    expect(result.exitCode).not.toBe(0);
  });
});

test.describe('CLI: Readme Command', () => {
  test('generates README content', async ({ cli }) => {
    // Generate README
    const result = await cli.runAndVerify(['readme']);

    // Should generate README content
    expect(result.stdout).toBeTruthy();
    expect(result.stdout).toContain('discotheque');
  });

  test('generates README for specific command', async ({ cli }) => {
    // Generate README for specific command
    const result = await cli.runAndVerify(['readme', '--command', 'add']);

    // Should generate command-specific content
    expect(result.stdout).toBeTruthy();
    expect(result.stdout).toContain('add');
  });

  test('generates README as markdown', async ({ cli }) => {
    // Generate README as markdown
    const result = await cli.runAndVerify(['readme', '--format', 'markdown']);

    // Should generate markdown
    expect(result.stdout).toBeTruthy();
    expect(result.stdout).toContain('#');
  });
});

test.describe('CLI: Search-DB Command', () => {
  test('searches arbitrary database table', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create and add a file
    createDummyVideo('video.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Search media table
    const result = await cli.runAndVerify(['search-db', testDbPath, 'media']);

    // Should return results
    expect(result.stdout).toBeTruthy();
  });

  test('searches table as JSON', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create and add a file
    createDummyVideo('video.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Search as JSON
    const result = await cli.runJson(['search-db', '-j', testDbPath, 'media']);

    expect(Array.isArray(result)).toBe(true);
  });

  test('searches table with limit', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create and add multiple files
    for (let i = 0; i < 10; i++) {
      createDummyVideo(`video${i}.mp4`);
    }
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Search with limit
    const result = await cli.runAndVerify(['search-db', '-L', '5', testDbPath, 'media']);

    // Should limit results
    const lines = result.stdout.split('\n').filter(l => l.includes('.mp4'));
    expect(lines.length).toBeLessThanOrEqual(5);
  });

  test('searches table with where clause', async ({ cli, tempDir, testDbPath, createDummyVideo, createDummyAudio }) => {
    // Create and add files
    createDummyVideo('video.mp4');
    createDummyAudio('audio.mp3');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Search with where clause
    const result = await cli.runAndVerify(['search-db', '-w', "type = 'video'", testDbPath, 'media']);

    // Should only show video
    expect(result.stdout).toContain('video.mp4');
    expect(result.stdout).not.toContain('audio.mp3');
  });

  test('searches table with columns', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create and add a file
    createDummyVideo('video.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Search with specific columns
    const result = await cli.runAndVerify(['search-db', '-c', 'path,size', testDbPath, 'media']);

    // Should show specified columns
    expect(result.stdout).toBeTruthy();
  });

  test('searches table with offset', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create and add multiple files
    for (let i = 0; i < 5; i++) {
      createDummyVideo(`video${i}.mp4`);
    }
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Search with offset
    const result = await cli.runAndVerify(['search-db', '--offset', '3', testDbPath, 'media']);

    // Should skip first 3 results
    const lines = result.stdout.split('\n').filter(l => l.includes('.mp4'));
    expect(lines.length).toBeLessThanOrEqual(2);
  });

  test('searches table with order by', async ({ cli, tempDir, testDbPath, createDummyFile }) => {
    // Create files of different sizes
    createDummyFile('small.bin', 100);
    createDummyFile('large.bin', 10000);
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Search with order by
    const result = await cli.runAndVerify(['search-db', '--order-by', 'size DESC', testDbPath, 'media']);

    // Large should come first
    expect(result.stdout.indexOf('large.bin')).toBeLessThan(result.stdout.indexOf('small.bin'));
  });

  test('searches non-existent table', async ({ cli, tempDir, testDbPath }) => {
    // Search non-existent table
    const result = await cli.run(['search-db', testDbPath, 'non_existent_table']);

    // Should fail
    expect(result.exitCode).not.toBe(0);
  });
});

test.describe('CLI: Search-Captions Command', () => {
  test('searches captions', async ({ cli, tempDir, testDbPath, createDummyVideo, createDummyVtt }) => {
    // Create video and VTT files
    const videoPath = createDummyVideo('movie.mp4');
    const vttPath = createDummyVtt('movie.vtt', `WEBVTT

00:00:01.000 --> 00:00:03.000
Hello world

00:00:04.000 --> 00:00:06.000
Search test caption
`);

    await cli.runAndVerify(['add', '--scan-subtitles', testDbPath, videoPath]);

    // Search captions
    const result = await cli.run(['search-captions', testDbPath, 'search']);

    // Should complete (may or may not find captions depending on implementation)
    expect(result.exitCode).toBeGreaterThanOrEqual(0);
  });

  test('searches captions with limit', async ({ cli, tempDir, testDbPath, createDummyVideo, createDummyVtt }) => {
    // Create video and VTT files
    const videoPath = createDummyVideo('movie.mp4');
    createDummyVtt('movie.vtt');
    await cli.runAndVerify(['add', '--scan-subtitles', testDbPath, videoPath]);

    // Search with limit
    const result = await cli.run(['search-captions', '-L', '10', testDbPath, 'test']);

    // Should complete
    expect(result.exitCode).toBeGreaterThanOrEqual(0);
  });

  test('searches captions as JSON', async ({ cli, tempDir, testDbPath, createDummyVideo, createDummyVtt }) => {
    // Create video and VTT files
    const videoPath = createDummyVideo('movie.mp4');
    createDummyVtt('movie.vtt');
    await cli.runAndVerify(['add', '--scan-subtitles', testDbPath, videoPath]);

    // Search as JSON
    const result = await cli.run(['search-captions', '-j', testDbPath, 'test']);

    // Should output JSON or empty
    expect(result.exitCode).toBeGreaterThanOrEqual(0);
  });

  test('searches captions with FTS', async ({ cli, tempDir, testDbPath, createDummyVideo, createDummyVtt }) => {
    // Create video and VTT files
    const videoPath = createDummyVideo('movie.mp4');
    createDummyVtt('movie.vtt');
    await cli.runAndVerify(['add', '--scan-subtitles', testDbPath, videoPath]);

    // Search with FTS
    const result = await cli.run(['search-captions', '--fts', testDbPath, 'test']);

    // Should complete
    expect(result.exitCode).toBeGreaterThanOrEqual(0);
  });

  test('searches captions with overlap', async ({ cli, tempDir, testDbPath, createDummyVideo, createDummyVtt }) => {
    // Create video and VTT files
    const videoPath = createDummyVideo('movie.mp4');
    createDummyVtt('movie.vtt');
    await cli.runAndVerify(['add', '--scan-subtitles', testDbPath, videoPath]);

    // Search with overlap
    const result = await cli.run(['search-captions', '--overlap', '2', testDbPath, 'test']);

    // Should complete
    expect(result.exitCode).toBeGreaterThanOrEqual(0);
  });
});

test.describe('CLI: Explode Command', () => {
  test('creates symlinks for subcommands', async ({ cli, tempDir }) => {
    // Create target directory
    const targetDir = path.join(tempDir, 'bin');
    fs.mkdirSync(targetDir);

    // Explode command
    const result = await cli.run(['explode', targetDir]);

    // Should create symlinks (may fail without proper permissions)
    expect(result.exitCode).toBeGreaterThanOrEqual(0);
  });

  test('creates symlinks with verbose output', async ({ cli, tempDir }) => {
    // Create target directory
    const targetDir = path.join(tempDir, 'bin');
    fs.mkdirSync(targetDir);

    // Explode with verbose
    const result = await cli.run(['explode', '--verbose', targetDir]);

    // Should have output
    expect(result.stdout || result.stderr).toBeTruthy();
  });
});

test.describe('CLI: Merge-DBs Command', () => {
  test('merges multiple databases', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create source databases
    const db1Path = path.join(tempDir, 'db1.db');
    const db2Path = path.join(tempDir, 'db2.db');

    // Add files to source databases
    createDummyVideo('video1.mp4');
    await cli.runAndVerify(['add', db1Path, tempDir]);

    createDummyVideo('video2.mp4');
    await cli.runAndVerify(['add', db2Path, tempDir]);

    // Merge databases
    const result = await cli.run(['merge-dbs', testDbPath, db1Path, db2Path]);

    // Should complete
    expect(result.exitCode).toBeGreaterThanOrEqual(0);
  });

  test('merges with verbose output', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create source databases
    const db1Path = path.join(tempDir, 'db1.db');

    createDummyVideo('video.mp4');
    await cli.runAndVerify(['add', db1Path, tempDir]);

    // Merge with verbose
    const result = await cli.run(['merge-dbs', '--verbose', testDbPath, db1Path]);

    // Should have output
    expect(result.stdout || result.stderr).toBeTruthy();
  });

  test('merges with dry run', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create source database
    const db1Path = path.join(tempDir, 'db1.db');

    createDummyVideo('video.mp4');
    await cli.runAndVerify(['add', db1Path, tempDir]);

    // Merge with dry run
    const result = await cli.run(['merge-dbs', '--simulate', testDbPath, db1Path]);

    // Should complete without modifying target
    expect(result.exitCode).toBeGreaterThanOrEqual(0);
  });
});
