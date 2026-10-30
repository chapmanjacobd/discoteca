import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import fs from 'fs';
import path from 'path';
import { setupTestEnvironment } from './test-helper';

function trackUrlHistory() {
    vi.spyOn(window.history, 'pushState').mockImplementation((state, title, url) => {
        const hashIndex = url.indexOf('#');
        window.location.hash = hashIndex !== -1 ? url.substring(hashIndex) : '';
    });
    vi.spyOn(window.history, 'replaceState').mockImplementation((state, title, url) => {
        const hashIndex = url.indexOf('#');
        window.location.hash = hashIndex !== -1 ? url.substring(hashIndex) : '';
    });
}

function lastFetchUrl(test) {
    const calls = global.fetch.mock.calls;
    if (calls.length === 0) throw new Error('No fetch calls recorded');
    return String(calls[calls.length - 1][0]);
}

describe('Search input handler', () => {
    beforeEach(async () => {
        await setupTestEnvironment();
    });

    it('normalizes backslashes and requests path suggestions', async () => {
        const input = document.getElementById('search-input');
        global.fetch.mockClear();

        input.value = '\\home\\user\\';
        input.dispatchEvent(new Event('input'));

        expect(input.value).toBe('/home/user/');
        await vi.waitFor(() => {
            expect(global.fetch.mock.calls.some(call => String(call[0]).includes('/api/ls?path='))).toBe(true);
        });
    });

    it('debounces normal search input', async () => {
        const input = document.getElementById('search-input');
        global.fetch.mockClear();

        input.value = 'debounced-query';
        input.dispatchEvent(new Event('input'));

        expect(global.fetch.mock.calls.some(call => String(call[0]).includes('search=debounced-query'))).toBe(false);

        await vi.waitFor(() => {
            expect(global.fetch.mock.calls.some(call => String(call[0]).includes('search=debounced-query'))).toBe(true);
        }, 1500);
    });
});

describe('RSVP routing', () => {
    beforeEach(async () => {
        await setupTestEnvironment();
    });

    it('opens a document in the video player', async () => {
        await window.disco.playRSVP({
            path: 'book.pdf',
            media_type: 'text/pdf',
            duration: 30
        });

        expect(document.getElementById('document-modal').classList.contains('hidden')).toBe(true);
        expect(document.getElementById('pip-player').classList.contains('hidden')).toBe(false);
        expect(document.querySelector('#media-viewer video')).not.toBeNull();
    });
});

describe('URL schema', () => {
    beforeEach(async () => {
        await setupTestEnvironment();
        trackUrlHistory();
    });

    it('round-trips every written parameter', () => {
        const state = window.disco.state;
        state.page = 'search';
        state.databases = ['a.db', 'b.db'];
        state.filters.excludedDbs = [];
        state.filters.search = 'needle';
        state.filters.categories = ['Movies'];
        state.filters.languages = ['en'];
        state.filters.ratings = [5];
        state.filters.media_types = ['video'];
        state.filters.sort = 'size';
        state.filters.reverse = true;
        state.filters.limit = 25;
        state.filters.all = false;
        state.currentPage = 2;
        state.view = 'details';

        window.disco.syncUrl({ navigation: true });

        const hash = decodeURIComponent(window.location.hash);
        expect(hash).toContain('mode=search');
        expect(hash).toContain('search=needle');
        expect(hash).toContain('category=Movies');
        expect(hash).toContain('language=en');
        expect(hash).toContain('rating=5');
        expect(hash).toContain('media_type=video');
        expect(hash).toContain('sort=size');
        expect(hash).toContain('reverse=true');
        expect(hash).toContain('limit=25');
        expect(hash).toContain('page=2');
        expect(hash).toContain('view=details');
        expect(hash).toContain('db=a.db');
        expect(hash).toContain('db=b.db');

        state.filters.search = '';
        state.filters.categories = [];
        state.filters.languages = [];
        state.filters.ratings = [];
        state.filters.media_types = [];
        state.filters.sort = 'default';
        state.filters.reverse = false;
        state.filters.limit = 99;
        state.currentPage = 1;
        state.view = 'grid';

        window.disco.readUrl();

        expect(state.filters.search).toBe('needle');
        expect(state.filters.categories).toEqual(['Movies']);
        expect(state.filters.languages).toEqual(['en']);
        expect(state.filters.ratings).toEqual(['5']);
        expect(state.filters.media_types).toEqual(['video']);
        expect(state.filters.sort).toBe('size');
        expect(state.filters.reverse).toBe(true);
        expect(state.filters.limit).toBe(25);
        expect(state.currentPage).toBe(2);
        expect(state.view).toBe('details');
        expect(state.filters.excludedDbs).toEqual([]);
    });

    it('pushes mode changes and replaces filter changes', () => {
        const state = window.disco.state;
        state.page = 'search';
        state.filters.search = 'one';
        window.disco.syncUrl({ navigation: true });

        window.history.replaceState.mockClear();
        window.history.pushState.mockClear();

        state.filters.search = 'two';
        window.disco.syncUrl();

        expect(window.history.replaceState).toHaveBeenCalled();
        expect(window.history.pushState).not.toHaveBeenCalled();

        window.history.replaceState.mockClear();
        window.history.pushState.mockClear();

        state.page = 'trash';
        window.disco.syncUrl();

        expect(window.history.pushState).toHaveBeenCalled();
    });

    it('restores sort keys that have no visible toolbar option', () => {
        const state = window.disco.state;
        state.page = 'trash';
        state.filters.sort = 'time_deleted';
        state.filters.reverse = true;
        window.disco.syncUrl({ navigation: true });

        state.filters.sort = 'default';
        window.disco.readUrl();

        expect(state.filters.sort).toBe('time_deleted');
        expect(state.filters.reverse).toBe(true);
        expect(document.getElementById('sort-by').value).toBe('time_deleted');
    });

    it('sends the __none__ sentinel when every database is excluded', () => {
        const state = window.disco.state;
        state.page = 'search';
        state.databases = ['a.db', 'b.db'];
        state.filters.excludedDbs = ['a.db', 'b.db'];

        window.disco.syncUrl({ navigation: true });

        expect(window.location.hash).toContain('db=__none__');
    });
});

describe('Database state before first query', () => {
    beforeEach(async () => {
        await setupTestEnvironment();
    });

    it('renders an explicit empty state and skips the query when all databases are excluded', async () => {
        const state = window.disco.state;
        state.databases = ['a.db', 'b.db'];
        state.filters.excludedDbs = ['a.db', 'b.db'];
        state.filters.search = '';
        state.page = 'search';

        global.fetch.mockClear();
        await window.disco.performSearch();

        expect(global.fetch.mock.calls.some(call => String(call[0]).includes('/api/query'))).toBe(false);
        expect(document.getElementById('results-container').textContent).toContain('No databases selected');
        expect(document.getElementById('results-count').textContent).toBe('0 results');
    });
});

describe('Disk usage mode', () => {
    beforeEach(async () => {
        await setupTestEnvironment();
    });

    it('omits limit/offset when all is requested', async () => {
        const state = window.disco.state;
        state.databases = ['test.db'];
        state.filters.excludedDbs = [];
        state.filters.all = true;
        state.filters.limit = 50;
        state.filters.sort = 'size';
        state.filters.reverse = true;

        await window.disco.fetchDU('');

        const url = lastFetchUrl();
        expect(url).toContain('all=true');
        expect(url).not.toContain('limit=');
        expect(url).not.toContain('offset=');
        expect(url).toContain('sort=size');
        expect(url).toContain('reverse=true');
        expect(document.getElementById('pagination-container').classList.contains('hidden')).toBe(true);
        expect(document.getElementById('results-count').textContent).toContain('total');
    });

    it('does not send custom sort fields and preserves reverse for the default sort', async () => {
        const state = window.disco.state;
        state.databases = ['test.db'];
        state.filters.excludedDbs = [];
        state.filters.all = false;
        state.filters.sort = 'custom';
        state.filters.customSortFields = 'size DESC';
        state.filters.reverse = true;

        await window.disco.fetchDU('');

        const url = lastFetchUrl();
        expect(url).not.toContain('sort_fields=');
        expect(url).not.toContain('sort=');
        expect(url).toContain('reverse=true');
    });

    it('only advertises supported sort options in DU mode', () => {
        const state = window.disco.state;
        state.page = 'du';
        window.disco.updateNavActiveStates();

        const sortBy = document.getElementById('sort-by');
        const visible = Array.from(sortBy.options).filter(option => !option.hidden).map(option => option.value);

        expect(visible).toEqual(['default', 'path', 'size', 'duration']);
        expect(document.getElementById('sort-complex-btn').classList.contains('hidden')).toBe(true);
    });
});

describe('Playback state', () => {
    beforeEach(async () => {
        await setupTestEnvironment();
    });

    it('applies the media-type default rate for each new item', async () => {
        const state = window.disco.state;
        state.defaultVideoRate = 1.5;
        state.defaultAudioRate = 0.75;

        await window.disco.openActivePlayer({ path: 'a.mp4', media_type: 'video/mp4', duration: 10 });
        expect(state.playbackRate).toBe(1.5);

        await window.disco.openActivePlayer({ path: 'b.mp3', media_type: 'audio/mpeg', duration: 10 });
        expect(state.playbackRate).toBe(0.75);
    });

    it('preserves theatre mode while switching tracks', async () => {
        const state = window.disco.state;
        state.playerMode = 'theatre';

        await window.disco.openActivePlayer({ path: 'a.mp4', media_type: 'video/mp4', duration: 10 });
        await window.disco.openActivePlayer({ path: 'b.mp4', media_type: 'video/mp4', duration: 10 });

        expect(state.playerMode).toBe('theatre');
        expect(document.getElementById('pip-player').classList.contains('theatre')).toBe(true);
    });

    it('clears seek history and marked position on item change', async () => {
        const state = window.disco.state;
        state.playback.seekHistory = [1, 2, 3];
        state.playback.markedPosition = 12;

        await window.disco.openActivePlayer({ path: 'b.mp4', media_type: 'video/mp4', duration: 10 });

        expect(state.playback.seekHistory).toEqual([]);
        expect(state.playback.markedPosition).toBe(null);
    });

    it('refreshes the unplayed filter using the previous item', async () => {
        const state = window.disco.state;
        state.filters.unplayed = true;
        state.playback.pendingUpdate = null;

        await window.disco.openActivePlayer({ path: 'played.mp4', media_type: 'video/mp4', duration: 10, play_count: 1 });
        global.fetch.mockClear();

        await window.disco.openActivePlayer({ path: 'next.mp4', media_type: 'video/mp4', duration: 10 });

        await vi.waitFor(() => {
            expect(global.fetch.mock.calls.some(call => String(call[0]).includes('/api/query'))).toBe(true);
        });
    });

    it('stores document progress separately from media seconds', () => {
        const state = window.disco.state;
        state.localResume = true;

        window.disco.saveDocumentProgress('book.pdf', 0.42);

        const documentProgress = JSON.parse(localStorage.getItem('disco-document-progress'));
        expect(documentProgress['book.pdf'].scrollPercent).toBeCloseTo(0.42);
        expect(window.disco.getLocalProgress({ path: 'book.pdf', media_type: 'text/pdf' })).toBe(0);
        expect(localStorage.getItem('disco-progress')).toBeNull();

        expect(window.disco.restoreDocumentProgress('book.pdf').scrollPercent).toBeCloseTo(0.42);
    });
});

describe('WebVTT parsing', () => {
    beforeEach(async () => {
        await setupTestEnvironment();
    });

    it('returns every cue in a multi-cue file', () => {
        const cues = window.disco.parseWebVTT(
            'WEBVTT\n\n' +
            '1\n00:00:01.000 --> 00:00:02.000\nFirst line\nsecond line\n\n' +
            '2\n00:00:03.000 --> 00:00:04.000\nSecond cue\n'
        );

        expect(cues).toHaveLength(2);
        expect(cues[0].text).toBe('First line\nsecond line');
        expect(cues[1].text).toBe('Second cue');
    });

    it('supports MM:SS.mmm timestamps and no trailing blank line', () => {
        const cues = window.disco.parseWebVTT(
            'WEBVTT\n\n00:01.000 --> 00:02.500\nShort form\n\nNOTE ignored\n\n00:03.000 --> 00:04.000\nLast cue'
        );

        expect(cues).toHaveLength(2);
        expect(cues[0].start).toBeCloseTo(1);
        expect(cues[0].end).toBeCloseTo(2.5);
        expect(cues[1].text).toBe('Last cue');
    });

    it('seeks to the next cue across multiple cues', async () => {
        const vtt = 'WEBVTT\n\n00:00:00.000 --> 00:00:04.000\nOne\n\n00:00:10.000 --> 00:00:14.000\nTwo\n\n00:00:20.000 --> 00:00:24.000\nThree\n';
        const baseImplementation = global.fetch.getMockImplementation();
        global.fetch.mockImplementation((url, options) => {
            if (String(url).includes('/api/subtitles')) {
                return Promise.resolve({
                    ok: true,
                    status: 200,
                    headers: { get: () => null },
                    text: () => Promise.resolve(vtt),
                    json: () => Promise.resolve([])
                });
            }
            return baseImplementation(url, options);
        });

        await window.disco.openActivePlayer({ path: 'video.mp4', media_type: 'video/mp4', duration: 30 });
        const video = document.querySelector('#media-viewer video');
        video.currentTime = 5;

        window.disco.seekToSubtitleCue();

        await vi.waitFor(() => {
            expect(video.currentTime).toBe(10);
        });
    });
});

describe('Captions rendering', () => {
    beforeEach(async () => {
        await setupTestEnvironment();
    });

    it('uses the captions details renderer in details view', async () => {
        const state = window.disco.state;
        state.page = 'captions';
        state.view = 'details';
        state.filters.search = '';

        await window.disco.performSearch();

        const container = document.getElementById('results-container');
        expect(container.textContent).toContain('First Caption');
        expect(container.textContent).not.toContain('Progress');
    });
});

describe('Playlist and group pagination', () => {
    beforeEach(async () => {
        await setupTestEnvironment();
    });

    it('paginates playlist items on the client', async () => {
        const state = window.disco.state;
        state.playlistItems = [
            { path: 'a.mp4', media_type: 'video/mp4' },
            { path: 'b.mp4', media_type: 'video/mp4' },
            { path: 'c.mp4', media_type: 'video/mp4' }
        ];
        state.filters.limit = 2;
        state.filters.all = false;
        state.currentPage = 1;

        window.disco.filterPlaylistItems();
        expect(state.totalCount).toBe(3);
        expect(window.disco.currentMedia.map(item => item.path)).toEqual(['a.mp4', 'b.mp4']);

        state.currentPage = 2;
        window.disco.filterPlaylistItems();
        expect(window.disco.currentMedia.map(item => item.path)).toEqual(['c.mp4']);
    });

    it('applies search filters to playlist items', async () => {
        const state = window.disco.state;
        state.playlistItems = [
            { path: 'Alpha.mp4', media_type: 'video/mp4' },
            { path: 'Beta.mp4', media_type: 'video/mp4' }
        ];
        state.filters.limit = 99;
        state.filters.all = false;
        state.currentPage = 1;
        state.filters.search = 'beta';

        window.disco.filterPlaylistItems();

        expect(window.disco.currentMedia.map(item => item.path)).toEqual(['Beta.mp4']);
    });

    it('resets to page 1 when switching playlists', async () => {
        const state = window.disco.state;

        await window.disco.fetchPlaylistItems('First');
        state.currentPage = 2;
        await window.disco.fetchPlaylistItems('Second');

        expect(state.currentPage).toBe(1);
    });

    it('requests every group without a limit', async () => {
        const baseImplementation = global.fetch.getMockImplementation();
        global.fetch.mockImplementation((url, options) => {
            if (String(url).includes('/api/episodes')) {
                return Promise.resolve({
                    ok: true,
                    status: 200,
                    headers: { get: () => null },
                    json: () => Promise.resolve([
                        { path: '/one', count: 2, files: [{ path: 'a.mp4' }, { path: 'b.mp4' }] },
                        { path: '/two', count: 1, files: [{ path: 'c.mp4' }] }
                    ]),
                    text: () => Promise.resolve('')
                });
            }
            return baseImplementation(url, options);
        });

        const state = window.disco.state;
        state.page = 'search';
        state.view = 'group';
        state.filters.limit = 99;

        await window.disco.fetchEpisodes();

        const url = lastFetchUrl();
        expect(url).toContain('/api/episodes');
        expect(url).toContain('all=true');
        expect(url).not.toContain('limit=');
        expect(document.getElementById('results-count').textContent).toContain('2 folders found');
    });
});

describe('Empty bin', () => {
    beforeEach(async () => {
        await setupTestEnvironment();
    });

    it('empties the whole bin and refreshes from page 1', async () => {
        const state = window.disco.state;
        state.page = 'trash';
        state.totalCount = 10;
        state.currentPage = 2;
        window.disco.currentMedia = [
            { path: 'a.mp4', media_type: 'video/mp4' },
            { path: 'b.mp4', media_type: 'video/mp4' }
        ];
        window.confirm = vi.fn().mockReturnValue(true);
        global.fetch.mockClear();

        await window.disco.emptyBin();

        const emptyCall = global.fetch.mock.calls.find(call => String(call[0]).includes('/api/empty-bin'));
        expect(emptyCall).toBeTruthy();
        const body = JSON.parse(emptyCall[1].body);
        expect(body.paths).toEqual([]);
        expect(window.confirm).toHaveBeenCalledWith(expect.stringContaining('all 10 files'));
        expect(state.currentPage).toBe(1);

        await vi.waitFor(() => {
            expect(global.fetch.mock.calls.some(call => String(call[0]).includes('trash=true'))).toBe(true);
        });
    });
});

describe('Media type usage', () => {
    beforeEach(async () => {
        await setupTestEnvironment();
    });

    it('preloads the next image based on media_type', async () => {
        const created = [];
        const originalImage = global.Image;
        global.Image = class {
            constructor() {
                created.push(this);
            }
        };

        try {
            const state = window.disco.state;
            window.disco.currentMedia = [
                { path: 'one.png', media_type: 'image/png' },
                { path: 'two.png', media_type: 'image/png' }
            ];
            state.filters.all = true;

            await window.disco.openActivePlayer({ path: 'one.png', media_type: 'image/png' });

            expect(created.some(image => String(image.src).includes('two.png'))).toBe(true);
        } finally {
            global.Image = originalImage;
        }
    });

    it('sorts playlist items by media_type', async () => {
        const state = window.disco.state;
        state.playlistItems = [
            { path: 'b.mp3', media_type: 'audio/mpeg' },
            { path: 'a.mp4', media_type: 'video/mp4' }
        ];
        state.filters.sort = 'media_type';
        state.filters.reverse = false;
        state.filters.limit = 99;
        state.filters.all = false;
        state.currentPage = 1;

        window.disco.filterPlaylistItems();

        expect(window.disco.currentMedia.map(item => item.path)).toEqual(['b.mp3', 'a.mp4']);
    });
});

describe('Listener lifecycle', () => {
    beforeEach(async () => {
        await setupTestEnvironment();
    });

    it('does not add shared viewer listeners for every opened item', async () => {
        const viewer = document.getElementById('media-viewer');
        const viewerSpy = vi.spyOn(viewer, 'addEventListener');
        const documentSpy = vi.spyOn(document, 'addEventListener');

        for (let i = 0; i < 4; i++) {
            await window.disco.openActivePlayer({
                path: `video-${i}.mp4`,
                media_type: 'video/mp4',
                duration: 10
            });
        }

        expect(viewerSpy).not.toHaveBeenCalled();
        expect(documentSpy).not.toHaveBeenCalled();
    });
});

describe('Dead UI IDs', () => {
    // IDs created by render functions rather than present in index.html.
    const dynamicIds = new Set([
        'categorization-link-btn',
        'categorize-full-path',
        'curation-cat-list',
        'disco-reader-style',
        'empty-bin-btn',
        'suggestions-area'
    ]);

    it('every getElementById target exists in index.html or is created dynamically', () => {
        const mainSource = fs.readFileSync(path.resolve(__dirname, '../src/main.ts'), 'utf8');
        const html = fs.readFileSync(path.resolve(__dirname, '../index.html'), 'utf8');
        const ids = [...mainSource.matchAll(/getElementById\(['"]([^'"]+)['"]\)/g)].map(match => match[1]);

        const missing = [...new Set(ids)].filter(id =>
            !dynamicIds.has(id) && !html.includes(`id="${id}"`)
        );

        expect(missing).toEqual([]);
    });
});

describe('Frontend logging', () => {
    it('does not ship console.log statements', () => {
        const source = fs.readFileSync(path.resolve(__dirname, '../src/main.ts'), 'utf8');
        expect(source).not.toContain('console.log');
    });
});
