import { test, expect } from '../fixtures-cli';

test.describe('CLI: MPV Control Commands', () => {
  test.describe('Now Command', () => {
    test('shows current mpv playback status', async ({ cli }) => {
      // Now command (will fail without mpv running)
      const result = await cli.run(['now']);

      // Should fail gracefully (no mpv running)
      expect(result.exitCode).not.toBe(0);
      expect(result.stderr).toBeTruthy();
    });

    test('shows playback with custom socket', async ({ cli }) => {
      // Now with custom socket
      const result = await cli.run(['now', '--mpv-socket', '/tmp/mpv-socket']);

      // Should fail gracefully
      expect(result.exitCode).not.toBe(0);
    });

    test('shows playback as JSON', async ({ cli }) => {
      // Now as JSON
      const result = await cli.run(['now', '--json']);

      // Should fail gracefully
      expect(result.exitCode).not.toBe(0);
    });

    test('shows playback with verbose output', async ({ cli }) => {
      // Now with verbose
      const result = await cli.run(['now', '--verbose']);

      // Should fail gracefully with output
      expect(result.exitCode).not.toBe(0);
      expect(result.stderr).toBeTruthy();
    });
  });

  test.describe('Next Command', () => {
    test('skips to next file in mpv', async ({ cli }) => {
      // Next command (will fail without mpv running)
      const result = await cli.run(['next']);

      // Should fail gracefully
      expect(result.exitCode).not.toBe(0);
    });

    test('skips with custom socket', async ({ cli }) => {
      // Next with custom socket
      const result = await cli.run(['next', '--mpv-socket', '/tmp/mpv-socket']);

      // Should fail gracefully
      expect(result.exitCode).not.toBe(0);
    });

    test('skips with verbose output', async ({ cli }) => {
      // Next with verbose
      const result = await cli.run(['next', '--verbose']);

      // Should fail gracefully
      expect(result.exitCode).not.toBe(0);
    });
  });

  test.describe('Stop Command', () => {
    test('stops mpv playback', async ({ cli }) => {
      // Stop command (will fail without mpv running)
      const result = await cli.run(['stop']);

      // Should fail gracefully
      expect(result.exitCode).not.toBe(0);
    });

    test('stops with custom socket', async ({ cli }) => {
      // Stop with custom socket
      const result = await cli.run(['stop', '--mpv-socket', '/tmp/mpv-socket']);

      // Should fail gracefully
      expect(result.exitCode).not.toBe(0);
    });

    test('stops with verbose output', async ({ cli }) => {
      // Stop with verbose
      const result = await cli.run(['stop', '--verbose']);

      // Should fail gracefully
      expect(result.exitCode).not.toBe(0);
    });
  });

  test.describe('Pause Command', () => {
    test('toggles mpv pause state', async ({ cli }) => {
      // Pause command (will fail without mpv running)
      const result = await cli.run(['pause']);

      // Should fail gracefully
      expect(result.exitCode).not.toBe(0);
    });

    test('toggles pause with custom socket', async ({ cli }) => {
      // Pause with custom socket
      const result = await cli.run(['pause', '--mpv-socket', '/tmp/mpv-socket']);

      // Should fail gracefully
      expect(result.exitCode).not.toBe(0);
    });

    test('toggles pause with verbose output', async ({ cli }) => {
      // Pause with verbose
      const result = await cli.run(['pause', '--verbose']);

      // Should fail gracefully
      expect(result.exitCode).not.toBe(0);
    });

    test('pause alias works', async ({ cli }) => {
      // Play alias (pause command)
      const result = await cli.run(['play']);

      // Should fail gracefully
      expect(result.exitCode).not.toBe(0);
    });
  });

  test.describe('Seek Command', () => {
    test('seeks mpv playback forward', async ({ cli }) => {
      // Seek command (will fail without mpv running)
      const result = await cli.run(['seek', '10']);

      // Should fail gracefully
      expect(result.exitCode).not.toBe(0);
    });

    test('seeks backward', async ({ cli }) => {
      // Seek backward
      const result = await cli.run(['seek', '-10']);

      // Should fail gracefully
      expect(result.exitCode).not.toBe(0);
    });

    test('seeks to absolute position', async ({ cli }) => {
      // Seek absolute
      const result = await cli.run(['seek', '60', '--absolute']);

      // Should fail gracefully
      expect(result.exitCode).not.toBe(0);
    });

    test('seeks with custom socket', async ({ cli }) => {
      // Seek with custom socket
      const result = await cli.run(['seek', '10', '--mpv-socket', '/tmp/mpv-socket']);

      // Should fail gracefully
      expect(result.exitCode).not.toBe(0);
    });

    test('seeks with verbose output', async ({ cli }) => {
      // Seek with verbose
      const result = await cli.run(['seek', '10', '--verbose']);

      // Should fail gracefully
      expect(result.exitCode).not.toBe(0);
    });

    test('ffwd alias works', async ({ cli }) => {
      // Ffwd alias
      const result = await cli.run(['ffwd', '10']);

      // Should fail gracefully
      expect(result.exitCode).not.toBe(0);
    });

    test('rewind alias works', async ({ cli }) => {
      // Rewind alias
      const result = await cli.run(['rewind', '10']);

      // Should fail gracefully
      expect(result.exitCode).not.toBe(0);
    });
  });
});

test.describe('CLI: Regex-Sort Command', () => {
  test('sorts by splitting lines', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create and add files
    createDummyVideo('movie_2023_action.mp4');
    createDummyVideo('movie_2022_comedy.mp4');
    createDummyVideo('show_2023_drama.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Regex sort
    const result = await cli.runAndVerify(['regex-sort', testDbPath]);

    // Should sort successfully
    expect(result.exitCode).toBe(0);
  });

  test('sorts with custom regexs', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create and add files
    createDummyVideo('movie_2023.mp4');
    createDummyVideo('show_2022.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Regex sort with custom patterns
    const result = await cli.runAndVerify(['regex-sort', '--regexs', '_\\d+', testDbPath]);

    // Should sort successfully
    expect(result.exitCode).toBe(0);
  });

  test('sorts with word sorts', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create and add files
    createDummyVideo('movie_2023.mp4');
    createDummyVideo('show_2022.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Regex sort with word sorts
    const result = await cli.runAndVerify(['regex-sort', '--word-sorts', 'natural', testDbPath]);

    // Should sort successfully
    expect(result.exitCode).toBe(0);
  });

  test('sorts with line sorts', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create and add files
    createDummyVideo('movie_2023.mp4');
    createDummyVideo('show_2022.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Regex sort with line sorts
    const result = await cli.runAndVerify(['regex-sort', '--line-sorts', 'length', testDbPath]);

    // Should sort successfully
    expect(result.exitCode).toBe(0);
  });

  test('sorts with preprocess', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create and add files
    createDummyVideo('Movie.Name.2023.mp4');
    createDummyVideo('Show.Name.2022.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Regex sort with preprocess
    const result = await cli.runAndVerify(['regex-sort', '--preprocess', testDbPath]);

    // Should sort successfully
    expect(result.exitCode).toBe(0);
  });

  test('sorts with stop words', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create and add files
    createDummyVideo('the_movie.mp4');
    createDummyVideo('the_show.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Regex sort with stop words
    const result = await cli.runAndVerify(['regex-sort', '--stop-words', 'the,a,an', testDbPath]);

    // Should sort successfully
    expect(result.exitCode).toBe(0);
  });

  test('sorts with compat mode', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create and add files
    createDummyVideo('movie_1.mp4');
    createDummyVideo('movie_10.mp4');
    createDummyVideo('movie_2.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Regex sort with compat mode
    const result = await cli.runAndVerify(['regex-sort', '--compat', testDbPath]);

    // Should sort successfully
    expect(result.exitCode).toBe(0);
  });

  test('sorts with verbose output', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create and add files
    createDummyVideo('movie.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Regex sort with verbose
    const result = await cli.runAndVerify(['regex-sort', '--verbose', testDbPath]);

    // Should have output
    expect(result.stdout).toBeTruthy();
  });

  test('sorts empty database', async ({ cli, tempDir, testDbPath }) => {
    // Regex sort empty database
    const result = await cli.runAndVerify(['regex-sort', testDbPath]);

    // Should succeed
    expect(result.exitCode).toBe(0);
  });
});

test.describe('CLI: Cluster-Sort Command', () => {
  test('groups items by similarity', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create and add files with similar names
    createDummyVideo('movie_part1.mp4');
    createDummyVideo('movie_part2.mp4');
    createDummyVideo('show_ep1.mp4');
    createDummyVideo('show_ep2.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Cluster sort
    const result = await cli.runAndVerify(['cluster-sort', testDbPath]);

    // Should cluster successfully
    expect(result.exitCode).toBe(0);
  });

  test('clusters with threshold', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create and add files
    createDummyVideo('file1.mp4');
    createDummyVideo('file2.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Cluster sort with threshold
    const result = await cli.runAndVerify(['cluster-sort', '--threshold', '0.5', testDbPath]);

    // Should cluster successfully
    expect(result.exitCode).toBe(0);
  });

  test('clusters with verbose output', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create and add files
    createDummyVideo('movie1.mp4');
    createDummyVideo('movie2.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Cluster sort with verbose
    const result = await cli.runAndVerify(['cluster-sort', '--verbose', testDbPath]);

    // Should have output
    expect(result.stdout).toBeTruthy();
  });

  test('clusters as JSON', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create and add files
    createDummyVideo('file1.mp4');
    createDummyVideo('file2.mp4');
    await cli.runAndVerify(['add', testDbPath, tempDir]);

    // Cluster sort as JSON
    const result = await cli.runJson(['cluster-sort', '-j', testDbPath]);

    expect(Array.isArray(result)).toBe(true);
  });

  test('clusters empty database', async ({ cli, tempDir, testDbPath }) => {
    // Cluster sort empty database
    const result = await cli.runAndVerify(['cluster-sort', testDbPath]);

    // Should succeed
    expect(result.exitCode).toBe(0);
  });
});

test.describe('CLI: Sample-Hash Command', () => {
  test('calculates hash for file', async ({ cli, tempDir, testDbPath, createDummyVideo }) => {
    // Create and add a file
    const videoPath = createDummyVideo('hash_test.mp4');
    await cli.runAndVerify(['add', testDbPath, videoPath]);

    // Calculate hash
    const result = await cli.runAndVerify(['sample-hash', videoPath]);

    // Should output hash
    expect(result.stdout).toBeTruthy();
    expect(result.stdout.length).toBeGreaterThan(0);
  });

  test('calculates hash with custom segment size', async ({ cli, tempDir, createDummyVideo }) => {
    // Create a file
    const videoPath = createDummyVideo('hash_segment.mp4');

    // Calculate hash with custom segment
    const result = await cli.runAndVerify(['sample-hash', '--segment-size', '1024', videoPath]);

    // Should output hash
    expect(result.stdout).toBeTruthy();
  });

  test('calculates hash with verbose output', async ({ cli, tempDir, createDummyVideo }) => {
    // Create a file
    const videoPath = createDummyVideo('hash_verbose.mp4');

    // Calculate hash with verbose
    const result = await cli.runAndVerify(['sample-hash', '--verbose', videoPath]);

    // Should have output
    expect(result.stdout).toBeTruthy();
  });

  test('calculates hash as JSON', async ({ cli, tempDir, createDummyVideo }) => {
    // Create a file
    const videoPath = createDummyVideo('hash_json.mp4');

    // Calculate hash as JSON
    const result = await cli.runJson(['sample-hash', '-j', videoPath]);

    expect(typeof result).toBe('object');
    expect(result).toHaveProperty('hash');
  });

  test('fails with non-existent file', async ({ cli, tempDir }) => {
    const nonExistentPath = '/tmp/does_not_exist.mp4';

    // Try to hash non-existent file
    const result = await cli.run(['sample-hash', nonExistentPath]);

    // Should fail
    expect(result.exitCode).not.toBe(0);
  });
});
