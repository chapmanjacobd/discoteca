import { vi } from 'vitest';
import fs from 'fs';
import path from 'path';

export async function setupTestEnvironment(initialLocalStorage) {
    // Mock CSS.escape (missing in JSDOM)
    global.CSS = {
        escape: (s) => s.replace(/([!"#$%&'()*+,.\/:;<=>?@\[\\\]^`{|}~])/g, "\\$1")
    };

    // Load mocks from mocks.json
    const mocksPath = path.resolve(__dirname, 'mocks.json');
    let mocks = {};
    if (fs.existsSync(mocksPath)) {
        mocks = JSON.parse(fs.readFileSync(mocksPath, 'utf8'));
    }

    // Mock fetch
    global.fetch = vi.fn().mockImplementation((url) => {
        if (typeof url !== 'string') url = url.toString();

        let data = [];
        if (url.includes('/api/databases')) {
            data = mocks.databases || { databases: ['test.db'], trashcan: true, read_only: false, dev: false };
        } else if (url.includes('/api/categories')) {
            data = mocks.categories || [{ category: 'comedy', count: 5 }, { category: 'music', count: 3 }];
        } else if (url.includes('/api/genres')) {
            data = mocks.genres || [{ genre: 'Rock', count: 10 }, { genre: 'Jazz', count: 2 }];
        } else if (url.includes('/api/ratings')) {
            data = mocks.ratings || [{ rating: 5, count: 1 }, { rating: 0, count: 10 }];
        } else if (url.includes('/api/playlists')) {
            data = mocks.playlists || ['My Playlist'];
        } else if (url.includes('/api/filter-bins')) {
            data = mocks.filter_bins || { 
                episodes: [], size: [], duration: [],
                episodes_min: 0, episodes_max: 100,
                size_min: 0, size_max: 100 * 1024 * 1024,
                duration_min: 0, duration_max: 3600
            };
        } else if (url.includes('/api/query')) {
            data = mocks.media || [
                { path: 'video1.mp4', type: 'video/mp4', size: 1024, duration: 60, db: 'test.db', caption_text: 'sample caption', caption_time: 10.5 },
                { path: 'audio1.mp3', type: 'audio/mpeg', size: 512, duration: 120, db: 'test.db', caption_text: 'another caption', caption_time: 20.0 }
            ];
        } else if (url.includes('/api/syncweb/folders')) {
            data = mocks.syncweb_folders || [{ id: 'mysync' }];
        } else if (url.includes('/api/syncweb/ls')) {
            data = mocks.syncweb_ls || [{ path: 'syncweb://mysync/file.mp4', type: 'video/mp4', size: 1024 }];
        }

        return Promise.resolve({
            ok: true,
            status: 200,
            headers: {
                get: (name) => {
                    if (name === 'X-Total-Count') return '2';
                    return null;
                }
            },
            json: () => Promise.resolve(data),
            text: () => Promise.resolve(typeof data === 'string' ? data : JSON.stringify(data))
        });
    });

    // Mock disco_token cookie
    document.cookie = 'disco_token=mock-test-token';

    // Mock window.innerWidth
    global.innerWidth = 1024;

    // Mock window.location
    delete window.location;
    window.location = {
        hash: '',
        search: '',
        href: 'http://localhost/',
        pathname: '/',
        reload: vi.fn(),
        replace: vi.fn(),
        assign: vi.fn(),
        toString: () => 'http://localhost/'
    };

    // Mock matchMedia
    window.matchMedia = vi.fn().mockImplementation(query => ({
        matches: false,
        media: query,
        onchange: null,
        addListener: vi.fn(),
        removeListener: vi.fn(),
        addEventListener: vi.fn(),
        removeEventListener: vi.fn(),
        dispatchEvent: vi.fn(),
    }));

    if (typeof global.DragEvent === 'undefined') {
        let sharedData = {};
        global.DragEvent = class DragEvent extends Event {
            constructor(type, options = {}) {
                super(type, options);
                this.dataTransfer = options.dataTransfer || {
                    setData: vi.fn((format, data) => { sharedData[format] = data; }),
                    getData: vi.fn((format) => sharedData[format] || ''),
                    effectAllowed: 'none',
                    dropEffect: 'none'
                };
            }
        };
    }

    // Mock APIs
    document.pictureInPictureEnabled = true;
    HTMLElement.prototype.scrollTo = vi.fn();
    HTMLElement.prototype.scrollIntoView = vi.fn();
    global.IntersectionObserver = class { constructor() { } observe() { } unobserve() { } disconnect() { } };
    global.Hls = class {
        static isSupported() { return true; }
        loadSource() { }
        attachMedia() { }
        on() { }
        destroy() { }
        static get Events() { return { MANIFEST_PARSED: 'hlsManifestParsed' }; }
    };

    // Load index.html
    const htmlPath = path.resolve(__dirname, 'index.html');
    const html = fs.readFileSync(htmlPath, 'utf8');
    document.body.innerHTML = html;

    window.location.hash = '';
    if (typeof initialLocalStorage === 'object') {
        Object.keys(initialLocalStorage).forEach(key => {
            localStorage.setItem(key, initialLocalStorage[key]);
        });
    } else {
        localStorage.clear();
    }
    vi.resetModules();

    await import('./app.js');
    document.dispatchEvent(new Event('DOMContentLoaded'));

    // Wait for async init and multiple renders
    await new Promise(resolve => setTimeout(resolve, 300));
}
