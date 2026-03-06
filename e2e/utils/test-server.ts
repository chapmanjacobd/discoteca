import { spawn, ChildProcess } from 'child_process';
import * as path from 'path';
import * as fs from 'fs';

export interface TestServerOptions {
  databasePath?: string;
  port?: number;
  verbose?: boolean;
}

export class TestServer {
  private process: ChildProcess | null = null;
  private baseUrl: string;
  private databasePath: string;
  private port: number;

  constructor(options: TestServerOptions = {}) {
    this.port = options.port || 8080;
    this.baseUrl = `http://localhost:${this.port}`;
    this.databasePath = options.databasePath || path.join(__dirname, '../fixtures/test.db');
  }

  async start(): Promise<void> {
    return new Promise((resolve, reject) => {
      const binaryPath = process.env.DISCO_BINARY || path.join(__dirname, '../../disco');
      
      // Check if binary exists
      if (!fs.existsSync(binaryPath)) {
        reject(new Error(`Disco binary not found at ${binaryPath}. Run 'make build' or 'go build -o disco ./cmd/disco' first.`));
        return;
      }

      const args = [
        'serve',
        this.databasePath,
        '--port', this.port.toString(),
      ];

      console.log(`Starting disco server: ${binaryPath} ${args.join(' ')}`);

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
            output.includes('addr=')) {
          started = true;
          clearTimeout(timeout);
          console.log('Disco server started successfully');
          resolve();
        }
      });

      this.process.stderr?.on('data', (data) => {
        console.error('[disco error]', data.toString().trim());
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

      console.log('Stopping disco server...');
      
      this.process.on('exit', () => {
        console.log('Disco server stopped');
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
