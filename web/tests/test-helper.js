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
            data = mocks.databases || { databases: ['test.db'], read_only: false, dev: false };
        } else if (url.includes('/api/categories')) {
            data = mocks.categories || [{ category: 'comedy', count: 5 }, { category: 'music', count: 3 }];
        } else if (url.includes('/api/genres')) {
            data = mocks.genres || [{ genre: 'Rock', count: 10 }, { genre: 'Jazz', count: 2 }];
        } else if (url.includes('/api/ratings')) {
            data = mocks.ratings || [{ rating: 5, count: 1 }, { rating: 0, count: 10 }];
        } else if (url.includes('/api/playlists')) {
            data = mocks.playlists || ['My Playlist'];
        } else if (url.includes('/api/query')) {
            // Return filter counts with results when include_counts=true
            if (url.includes('include_counts=true')) {
                let items;
                if (url.includes('captions=true')) {
                    items = mocks.media_with_captions || [
                        { path: 'video1.mp4', media_type: 'video/mp4', size: 1024, duration: 60, db: 'test.db', caption_text: 'sample caption', caption_time: 10.5 },
                    ];
                } else if (url.includes('media_type=')) {
                    // Filter by media_type if specified
                    const typeMatch = url.match(/media_type=([^&]+)/);
                    if (typeMatch) {
                        const filterType = decodeURIComponent(typeMatch[1]);
                        // Match partial types (e.g., "video" matches "video/mp4")
                        items = (mocks.media || []).filter(m => m.media_type && m.media_type.includes(filterType));
                        if (items.length === 0) {
                            // Return empty array if no matches
                            items = [];
                        }
                    } else {
                        items = mocks.media || [];
                    }
                } else {
                    items = mocks.media || [
                        { path: 'video1.mp4', media_type: 'video/mp4', size: 1024, duration: 60, db: 'test.db' },
                        { path: 'audio1.mp3', media_type: 'audio/mpeg', size: 512, duration: 120, db: 'test.db' }
                    ];
                }
                data = {
                    items: items,
                    counts: mocks.filter_bins || {
                        episodes: [], size: [], duration: [],
                        episodes_min_val: 1, episodes_max_val: 24,
                        size_min_val: 0, size_max_val: 100 * 1024 * 1024,
                        duration_min_val: 0, duration_max_val: 3600,
                        episodes_percentiles: Array.from({length: 101}, (_, i) => Math.floor(i * 0.24)),
                        size_percentiles: Array.from({length: 101}, (_, i) => Math.floor(i * 1024 * 1024)),
                        duration_percentiles: Array.from({length: 101}, (_, i) => Math.floor(i * 36)),
                        media_type: []
                    }
                };
            } else if (url.includes('captions=true')) {
                data = mocks.media_with_captions || [
                    { path: 'video1.mp4', media_type: 'video/mp4', size: 1024, duration: 60, db: 'test.db', caption_text: 'sample caption', caption_time: 10.5 },
                    { path: 'video2.mp4', media_type: 'video/mp4', size: 2048, duration: 120, db: 'test.db', caption_text: 'another caption', caption_time: 20.0 },
                    { path: 'video3.mp4', media_type: 'video/mp4', size: 512, duration: 30, db: 'test.db', caption_text: 'third caption', caption_time: 5.0 }
                ];
            } else {
                data = mocks.media || [
                    { path: 'video1.mp4', media_type: 'video/mp4', size: 1024, duration: 60, db: 'test.db', caption_text: 'sample caption', caption_time: 10.5 },
                    { path: 'audio1.mp3', media_type: 'audio/mpeg', size: 512, duration: 120, db: 'test.db', caption_text: 'another caption', caption_time: 20.0 }
                ];
            }
        } else if (url.includes('/api/filter-bins')) {
            // Legacy endpoint - still supported but not used by default
            data = mocks.filter_bins || {
                episodes: [], size: [], duration: [],
                episodes_min_val: 0, episodes_max_val: 100,
                size_min_val: 0, size_max_val: 100 * 1024 * 1024,
                duration_min_val: 0, duration_max_val: 3600,
                duration_percentiles: Array.from({length: 101}, (_, i) => Math.floor(i * 36)),
                size_percentiles: Array.from({length: 101}, (_, i) => i * 1024 * 1024),
                episodes_percentiles: Array.from({length: 101}, (_, i) => i)
            };
        } else if (url.includes('/api/categorize/keywords')) {
            data = mocks.categorize_keywords || [
                { category: 'Genre', keywords: ['Rock', 'Jazz', 'Pop'] },
                { category: 'Mood', keywords: ['Happy', 'Sad'] }
            ];
        } else if (url.includes('/api/categorize/suggest')) {
            data = mocks.categorize_suggest || ['Concert', 'Live', 'Studio'];
        } else if (url.includes('/api/categorize/apply')) {
            data = { success: true };
        } else if (url.includes('/api/categorize/defaults')) {
            data = { success: true };
        } else if (url.includes('/api/categorize/keyword')) {
            data = { success: true };
        } else if (url.includes('/api/categorize/category')) {
            data = { success: true };
        } else if (url.includes('/api/du')) {
            data = mocks.du_data || [
                { path: '/videos/movies/', total_size: 1073741824, total_duration: 7200, count: 5, files: [] },
                { path: '/videos/music/', total_size: 536870912, total_duration: 3600, count: 10, files: [] },
                { path: '/audio/', total_size: 268435456, total_duration: 1800, count: 20, files: [] }
            ];
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

    // Mock HTMLMediaElement methods (play, pause, etc.) to avoid JSDOM warnings
    HTMLMediaElement.prototype.play = vi.fn().mockResolvedValue(undefined);
    HTMLMediaElement.prototype.pause = vi.fn();

    global.Hls = class {
        static isSupported() { return true; }
        loadSource() { }
        attachMedia() { }
        on() { }
        destroy() { }
        static get Events() { return { MANIFEST_PARSED: 'hlsManifestParsed' }; }
    };

    // Load index.html
    const htmlPath = path.resolve(__dirname, '../index.html');
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

    await import('../src/main.ts');
    document.dispatchEvent(new Event('DOMContentLoaded'));

    // Wait for async init and multiple renders
    await new Promise(resolve => setTimeout(resolve, 300));
}
