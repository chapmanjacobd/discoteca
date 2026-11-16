// Node.js 25+ defines global Web Storage objects that stay unavailable unless
// the process is started with --localstorage-file. Vitest's jsdom environment
// keeps those globals because jsdom cannot overwrite existing keys, which
// leaves bare `localStorage` undefined inside tests. Bind the globals to the
// jsdom storage instances instead.
for (const key of ['localStorage', 'sessionStorage']) {
    const storage = globalThis.jsdom?.window?.[key];
    if (storage) {
        Object.defineProperty(globalThis, key, {
            configurable: true,
            writable: true,
            value: storage,
        });
    }
}
