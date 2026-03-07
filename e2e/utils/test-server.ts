import { spawn, ChildProcess } from 'child_process';
import * as path from 'path';
import * as fs from 'fs';
import * as net from 'net';

export interface TestServerOptions {
  databasePath?: string;
  port?: number;
  verbose?: boolean;
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

export class TestServer {
  private process: ChildProcess | null = null;
  private baseUrl: string;
  private databasePath: string;
  private port: number;

  constructor(options: TestServerOptions = {}) {
    this.port = options.port || 0; // 0 means find free port dynamically
    this.databasePath = options.databasePath || path.join(__dirname, '../fixtures/test.db');
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
        '--port', this.port.toString(),
        '--dev',
      ];

      this.process = spawn(binaryPath, args, {
        stdio: ['pipe', 'pipe', 'pipe'],
        env: { ...process.env, DISCO_DEV: 'true' },
      });

      let started = false;
      const timeout = setTimeout(() => {
        if (!started) {
          this.stop();
          reject(new Error('Server failed to start within 10 seconds'));
        }
      }, 10000);

      this.process.stdout?.on('data', (data) => {
        const output = data.toString();
        if (process.env.DEBUG) {
          console.log('[disco]', output.trim());
        }
        // Check for various startup messages
        if (output.includes('Starting server') ||
            output.includes('listening') ||
            output.includes('addr=') ||
            output.includes('Server starting')) {
          started = true;
          clearTimeout(timeout);
          resolve();
        }
      });

      this.process.stderr?.on('data', (data) => {
        const output = data.toString();
        if (process.env.DEBUG) {
          console.error('[disco error]', output.trim());
        }
        // Also check stderr for startup messages (disco logs startup info to stderr)
        if (output.includes('Starting server') ||
            output.includes('listening') ||
            output.includes('addr=') ||
            output.includes('Server starting')) {
          started = true;
          clearTimeout(timeout);
          resolve();
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
