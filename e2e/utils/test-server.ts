import { spawn, ChildProcess } from 'child_process';
import * as path from 'path';
import * as fs from 'fs';
import * as net from 'net';
import * as http from 'http';

export interface TestServerOptions {
  databasePath?: string;
  port?: number;
  verbose?: boolean;
  readOnly?: boolean;
}

// Find a free port dynamically
async function findFreePort(): Promise<number> {
  return new Promise((resolve, reject) => {
    const server = net.createServer();
    server.listen(0, () => {
      const address = server.address();
      if (typeof address === 'object' && address && address.port) {
        server.close(() => resolve(address.port));
      } else {
        reject(new Error('Failed to get port'));
      }
    });
    server.on('error', reject);
  });
}

// Poll the health endpoint until the server is ready
async function waitForHealth(baseUrl: string, timeoutMs: number = 10000): Promise<void> {
  const startTime = Date.now();
  const intervalMs = 100;

  while (Date.now() - startTime < timeoutMs) {
    try {
      await new Promise<void>((resolve, reject) => {
        const req = http.get(`${baseUrl}/health`, (res) => {
          if (res.statusCode === 200) {
            resolve();
          } else {
            reject(new Error(`Health check returned ${res.statusCode}`));
          }
        });
        req.on('error', reject);
        req.setTimeout(1000, () => {
          req.destroy();
          reject(new Error('Health check timeout'));
        });
      });
      return; // Health check passed
    } catch {
      // Health check failed, retry after interval
      await new Promise(resolve => setTimeout(resolve, intervalMs));
    }
  }
  throw new Error(`Server health check failed after ${timeoutMs}ms`);
}

export class TestServer {
  private process: ChildProcess | null = null;
  private baseUrl: string;
  private databasePath: string;
  private port: number;
  private readOnly: boolean;

  constructor(options: TestServerOptions = {}) {
    this.port = options.port || 0; // 0 means find free port dynamically
    this.databasePath = options.databasePath || path.join(__dirname, '../fixtures/test.db');
    this.readOnly = options.readOnly ?? false; // Default to false for backward compatibility
    this.baseUrl = ''; // Will be set after server starts
  }

  async start(): Promise<void> {
    return new Promise(async (resolve, reject) => {
      // Find free port if not specified
      if (this.port === 0) {
        this.port = await findFreePort();
      }

      this.baseUrl = `http://localhost:${this.port}`;
      const binaryPath = process.env.DISCO_BINARY || path.join(__dirname, '../../disco');

      // Check if binary exists
      if (!fs.existsSync(binaryPath)) {
        reject(new Error(`Disco binary not found at ${binaryPath}. Run 'make build' or 'go build -o disco ./cmd/disco' first.`));
        return;
      }

      // Check if database exists
      if (!fs.existsSync(this.databasePath)) {
        reject(new Error(`Test database not found at ${this.databasePath}. Run 'make e2e-init' first.`));
        return;
      }

      const args = [
        'serve',
        this.databasePath,
        '--no-browser',
        '--port', this.port.toString(),
        '--dev',
        '--public-dir', path.resolve(__dirname, '../../web/dist'),
      ];

      // Add --read-only flag if enabled
      if (this.readOnly) {
        args.push('--read-only');
      }

      this.process = spawn(binaryPath, args, {
        stdio: ['pipe', 'pipe', 'pipe'],
        env: {
          ...process.env,
          DISCO_DEV: 'true',
          DISCO_API_TOKEN: 'e2e-test-token'
        },
      });

      let started = false;
      const timeout = setTimeout(() => {
        if (!started) {
          this.stop();
          reject(new Error('Server failed to start within 10 seconds'));
        }
      }, 10000);

      // Wait for health check endpoint to be ready
      waitForHealth(this.baseUrl)
        .then(() => {
          started = true;
          clearTimeout(timeout);
          if (process.env.DEBUG || process.env.VERBOSE) {
            console.log('[disco] Server is healthy and ready');
          }
          resolve();
        })
        .catch((err) => {
          started = true;
          clearTimeout(timeout);
          reject(new Error(`Server health check failed: ${err.message}`));
        });

      this.process.stdout?.on('data', (data) => {
        const output = data.toString();
        if (process.env.DEBUG || process.env.VERBOSE) {
          console.log('[disco]', output.trim());
        }
      });

      this.process.stderr?.on('data', (data) => {
        const output = data.toString();
        if (process.env.DEBUG || process.env.VERBOSE) {
          console.error('[disco error]', output.trim());
        }
      });

      this.process.on('error', (err) => {
        clearTimeout(timeout);
        reject(new Error(`Failed to start server: ${err.message}`));
      });

      this.process.on('exit', (code) => {
        if (!started) {
          clearTimeout(timeout);
          reject(new Error(`Server exited with code ${code} before becoming ready`));
        }
      });
    });
  }

  async stop(): Promise<void> {
    return new Promise((resolve) => {
      if (!this.process) {
        resolve();
        return;
      }

      this.process.on('exit', () => {
        resolve();
      });

      this.process.kill('SIGTERM');

      // Force kill after 5 seconds if still running
      setTimeout(() => {
        if (this.process) {
          this.process.kill('SIGKILL');
        }
        resolve();
      }, 5000);
    });
  }

  getBaseUrl(): string {
    return this.baseUrl;
  }

  getDatabasePath(): string {
    return this.databasePath;
  }
}

// Global server instance for test suite
let globalServer: TestServer | null = null;

export async function startGlobalServer(options?: TestServerOptions): Promise<TestServer> {
  if (globalServer) {
    return globalServer;
  }

  globalServer = new TestServer(options);
  await globalServer.start();
  return globalServer;
}

export async function stopGlobalServer(): Promise<void> {
  if (globalServer) {
    await globalServer.stop();
    globalServer = null;
  }
}
