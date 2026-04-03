import { spawn, ChildProcess } from 'child_process';
import * as path from 'path';
import * as fs from 'fs';

export interface CLIResult {
  stdout: string;
  stderr: string;
  exitCode: number | null;
  command: string;
  duration: number;
}

export interface CLIOptions {
  timeout?: number;
  cwd?: string;
  env?: Record<string, string>;
  binaryPath?: string;
  verbose?: boolean;
}

/**
 * CLI test runner for executing disco commands
 */
export class CLIRunner {
  private binaryPath: string;
  private defaultTimeout: number;
  private verbose: boolean;

  constructor(options?: CLIOptions) {
    this.binaryPath = options?.binaryPath || this.findBinary();
    this.defaultTimeout = options?.timeout || 30000;
    this.verbose = options?.verbose || false;
  }

  /**
   * Find the disco binary in common locations
   */
  private findBinary(): string {
    const binaryPath = path.join(__dirname, '../../disco');
    
    if (fs.existsSync(binaryPath)) {
      return binaryPath;
    }
    
    // Fallback to PATH
    return 'disco';
  }

  /**
   * Execute a disco command
   */
  async run(args: string[], options?: CLIOptions): Promise<CLIResult> {
    const startTime = Date.now();
    const command = `${this.binaryPath} ${args.join(' ')}`;

    return new Promise((resolve, reject) => {
      const timeout = options?.timeout || this.defaultTimeout;
      const env = {
        ...process.env,
        ...options?.env,
      };

      const childProcess = spawn(this.binaryPath, args, {
        cwd: options?.cwd,
        env,
        stdio: ['pipe', 'pipe', 'pipe'],
      });

      let stdout = '';
      let stderr = '';
      let timedOut = false;

      const timeoutId = setTimeout(() => {
        timedOut = true;
        childProcess.kill('SIGKILL');
      }, timeout);

      childProcess.stdout.on('data', (data) => {
        stdout += data.toString();
        if (this.verbose) {
          process.stdout.write(data);
        }
      });

      childProcess.stderr.on('data', (data) => {
        stderr += data.toString();
        if (this.verbose) {
          process.stderr.write(data);
        }
      });

      childProcess.on('error', (err) => {
        clearTimeout(timeoutId);
        reject(new Error(`Failed to execute command: ${err.message}`));
      });

      childProcess.on('close', (code) => {
        clearTimeout(timeoutId);
        const duration = Date.now() - startTime;

        if (timedOut) {
          resolve({
            stdout,
            stderr: stderr + '\n[TIMEOUT] Command exceeded timeout limit',
            exitCode: null,
            command,
            duration,
          });
        } else {
          resolve({
            stdout,
            stderr,
            exitCode: code,
            command,
            duration,
          });
        }
      });
    });
  }

  /**
   * Execute a command and assert it succeeds
   */
  async runAndVerify(args: string[], options?: CLIOptions): Promise<CLIResult> {
    const result = await this.run(args, options);

    if (result.exitCode !== 0) {
      throw new Error(
        `Command failed with exit code ${result.exitCode}:\n${result.stderr}`
      );
    }

    return result;
  }

  /**
   * Execute a command and return JSON output
   */
  async runJson<T>(args: string[], options?: CLIOptions): Promise<T> {
    const result = await this.runAndVerify([...args, '--json'], options);

    try {
      return JSON.parse(result.stdout) as T;
    } catch (err) {
      throw new Error(
        `Failed to parse JSON output: ${err}\nOutput: ${result.stdout}`
      );
    }
  }

  /**
   * Execute a command with a database argument
   */
  async runWithDb(
    dbPath: string,
    command: string,
    args: string[],
    options?: CLIOptions
  ): Promise<CLIResult> {
    return this.run([command, dbPath, ...args], options);
  }
}

/**
 * Create a temporary test database
 */
export async function createTestDatabase(
  tempDir: string,
  schema?: string
): Promise<string> {
  const dbPath = path.join(tempDir, `test-${Date.now()}.db`);

  // Copy schema if provided
  if (schema) {
    const fs = await import('fs');
    const { exec } = await import('child_process');

    return new Promise((resolve, reject) => {
      exec(`sqlite3 "${dbPath}" "${schema}"`, (err) => {
        if (err) {
          reject(err);
        } else {
          resolve(dbPath);
        }
      });
    });
  }

  return dbPath;
}

/**
 * Create temporary directory for tests
 */
export function createTempDir(prefix = 'disco-test-'): string {
  const fs = require('fs');
  const os = require('os');
  const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), prefix));
  return tempDir;
}

/**
 * Cleanup temporary directory
 */
export function cleanupTempDir(dir: string): void {
  const fs = require('fs');
  if (fs.existsSync(dir)) {
    fs.rmSync(dir, { recursive: true, force: true });
  }
}
