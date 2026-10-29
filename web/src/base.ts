// The server injects window.__DISCO_BASE_PATH__ into index.html when discoteca
// is mounted below a URL prefix (for example /disco). It is empty when the app
// is served at the root.
declare global {
    interface Window {
        __DISCO_BASE_PATH__?: string;
    }
}

function readBasePath(): string {
    const injected = typeof window !== 'undefined' ? window.__DISCO_BASE_PATH__ : '';
    if (!injected || injected === '/') return '';
    return injected.replace(/\/+$/, '');
}

export const BASE_PATH = readBasePath();

export function applyBasePath(base: string, path: string): string {
    const prefix = base === '/' ? '' : base.replace(/\/+$/, '');
    if (!path || !path.startsWith('/')) return path;
    if (prefix && (path === prefix || path.startsWith(prefix + '/'))) {
        return path;
    }
    return prefix + path;
}

// appUrl prefixes root-relative application URLs with the configured base
// path. Absolute URLs (http://, blob:, data:) and relative paths are returned
// unchanged. The function is idempotent, so calling it twice is safe.
export function appUrl(path: string): string {
    return applyBasePath(BASE_PATH, path);
}
