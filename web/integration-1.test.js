import { describe, it, expect, beforeEach, vi } from 'vitest';
import { setupTestEnvironment } from './test-helper';

describe('Integration Test', () => {
    beforeEach(async () => {
        await setupTestEnvironment();
    });

    it('initializes correctly', async () => {
        expect(global.fetch).toHaveBeenCalledWith(
            '/api/databases',
            expect.objectContaining({
                headers: expect.objectContaining({
                    'X-Disco-Token': 'mock-test-token'
                })
            })
        );
        expect(document.getElementById('search-input')).toBeDefined();
    });

    it('performs search when enter is pressed', async () => {
        const searchInput = document.getElementById('search-input');
        searchInput.value = 'test query';

        const event = new KeyboardEvent('keypress', { key: 'Enter', bubbles: true });
        searchInput.dispatchEvent(event);

        await vi.waitFor(() => {
            expect(global.fetch).toHaveBeenCalledWith(
                expect.stringContaining('search=test+query'),
                expect.any(Object)
            );
        });
    });

    it('switches to trash view', async () => {
        const trashBtn = document.getElementById('trash-btn');
        trashBtn.click();

        await vi.waitFor(() => {
            expect(global.fetch).toHaveBeenCalledWith(
                expect.stringContaining('trash=true'),
                expect.any(Object)
            );
        });
    });

    it('toggles media type filters in sidebar', async () => {
        const audioBtn = document.querySelector('#media-type-list .category-btn[data-type="audio"]');
        audioBtn.click(); // Toggle off

        await vi.waitFor(() => {
            expect(global.fetch).toHaveBeenCalledWith(
                expect.not.stringContaining('type=audio'),
                expect.any(Object)
            );
        });
    });

    it('opens and closes settings modal', async () => {
        const settingsBtn = document.getElementById('settings-button');
        const modal = document.getElementById('settings-modal');

        expect(modal.classList.contains('hidden')).toBe(true);
        settingsBtn.click();
        expect(modal.classList.contains('hidden')).toBe(false);

        const closeBtn = modal.querySelector('.close-modal');
        closeBtn.click();
        expect(modal.classList.contains('hidden')).toBe(true);
    });

    it('toggles view modes', async () => {
        const viewGroup = document.getElementById('view-group');
        const resultsContainer = document.getElementById('results-container');

        viewGroup.click();
        await vi.waitFor(() => {
            const calls = global.fetch.mock.calls;
            const lastCall = calls[calls.length - 1];
            expect(lastCall[0]).toContain('/api/episodes');
        });

        const viewGrid = document.getElementById('view-grid');
        viewGrid.click();
        await vi.waitFor(() => {
            expect(resultsContainer.classList.contains('grid')).toBe(true);
        });
    });

    it('trashes a media item', async () => {
        await new Promise(r => setTimeout(r, 100));
        const deleteBtn = document.querySelector('.media-action-btn.delete');
        expect(deleteBtn).not.toBeNull();

        window.confirm = vi.fn().mockReturnValue(true);
        deleteBtn.click();

        await vi.waitFor(() => {
            expect(global.fetch).toHaveBeenCalledWith(
                '/api/delete',
                expect.objectContaining({
                    method: 'POST',
                    body: expect.stringContaining('"restore":false')
                })
            );
        });
    });

    it('restores an item from trash', async () => {
        const trashBtn = document.getElementById('trash-btn');
        trashBtn.click();

        await new Promise(r => setTimeout(r, 100));
        const restoreBtn = document.querySelector('.media-action-btn.restore');
        expect(restoreBtn).not.toBeNull();

        restoreBtn.click();

        await vi.waitFor(() => {
            expect(global.fetch).toHaveBeenCalledWith(
                '/api/delete',
                expect.objectContaining({
                    method: 'POST',
                    body: expect.stringContaining('"restore":true')
                })
            );
        });
    });

    it('plays media when card is clicked', async () => {
        window.disco.state.player = 'system';

        await new Promise(r => setTimeout(r, 200));
        const card = document.querySelector('.media-card');
        const title = card.querySelector('.media-title');
        title.click();

        // Should fetch play API
        await vi.waitFor(() => {
            expect(global.fetch).toHaveBeenCalledWith(
                '/api/play',
                expect.objectContaining({
                    method: 'POST',
                    body: expect.stringContaining('video1.mp4')
                })
            );
        });
    });

    it('toggles theatre mode', async () => {
        await new Promise(r => setTimeout(r, 100));
        const card = document.querySelector('.media-card');
        card.click(); // Open PiP

        const theatreBtn = document.getElementById('pip-theatre');
        const pipPlayer = document.getElementById('pip-player');

        expect(pipPlayer.classList.contains('theatre')).toBe(false);
        theatreBtn.click();
        expect(pipPlayer.classList.contains('theatre')).toBe(true);

        theatreBtn.click();
        expect(pipPlayer.classList.contains('theatre')).toBe(false);
    });

    it('changes sort order', async () => {
        const sortBy = document.getElementById('sort-by');
        sortBy.value = 'size';
        sortBy.dispatchEvent(new Event('change'));

        await vi.waitFor(() => {
            expect(global.fetch).toHaveBeenCalledWith(
                expect.stringContaining('sort=size'),
                expect.any(Object)
            );
        });
    });
    it('filters by category', async () => {
        const allMediaBtn = document.getElementById('all-media-btn');
        expect(allMediaBtn.classList.contains('active')).toBe(true);

        await vi.waitFor(() => {
            const comedyBtn = document.querySelector('.category-btn[data-cat="comedy"]');
            expect(comedyBtn).not.toBeNull();
        });

        const comedyBtn = document.querySelector('.category-btn[data-cat="comedy"]');
        comedyBtn.click();

        await vi.waitFor(() => {
            expect(global.fetch).toHaveBeenCalledWith(
                expect.stringContaining('category=comedy'),
                expect.any(Object)
            );
        });
        expect(allMediaBtn.classList.contains('active')).toBe(false);
    });

    it('filters by rating', async () => {
        await vi.waitFor(() => {
            const ratingBtn = document.querySelector('.category-btn[data-rating="5"]');
            expect(ratingBtn).not.toBeNull();
        });

        const ratingBtn = document.querySelector('.category-btn[data-rating="5"]');
        ratingBtn.click();

        await vi.waitFor(() => {
            expect(global.fetch).toHaveBeenCalledWith(
                expect.stringContaining('rating=5'),
                expect.any(Object)
            );
        });
    });

    it('creates a new playlist', async () => {
        window.prompt = vi.fn().mockReturnValue('New Cool List');
        const newPlaylistBtn = document.getElementById('new-playlist-btn');
        newPlaylistBtn.click();

        await vi.waitFor(() => {
            expect(global.fetch).toHaveBeenCalledWith(
                '/api/playlists',
                expect.objectContaining({
                    method: 'POST',
                    body: expect.stringContaining('"title":"New Cool List"')
                })
            );
        });
    });

    it('adds an item to a playlist', async () => {
        await new Promise(r => setTimeout(r, 100));
        const addBtn = document.querySelector('.media-action-btn.add-playlist');
        expect(addBtn).not.toBeNull();

        // Mock prompt for playlist selection (it shows a list and asks for index)
        window.prompt = vi.fn().mockReturnValue('1');

        addBtn.click();

        await vi.waitFor(() => {
            expect(global.fetch).toHaveBeenCalledWith(
                '/api/playlists/items',
                expect.objectContaining({
                    method: 'POST',
                    body: expect.stringContaining('"playlist_title":"My Playlist"')
                })
            );
        });
    });

    it('drags an item into a playlist', async () => {
        await new Promise(r => setTimeout(r, 100));
        const card = document.querySelector('.media-card');
        expect(card).not.toBeNull();

        const playlistZone = document.querySelector('.playlist-drop-zone');
        expect(playlistZone).not.toBeNull();

        // Simulate dragstart
        const dragStartEvent = new DragEvent('dragstart', { bubbles: true });
        card.dispatchEvent(dragStartEvent);
        expect(window.disco.state.draggedItem).not.toBeNull();
        expect(window.disco.state.draggedItem.path).toBe('video1.mp4');
        expect(document.body.classList.contains('is-dragging')).toBe(true);

        // Simulate dragenter
        const dragEnterEvent = new DragEvent('dragenter', { bubbles: true });
        playlistZone.dispatchEvent(dragEnterEvent);
        expect(playlistZone.classList.contains('drag-over')).toBe(true);

        // Simulate dragover
        const dragOverEvent = new DragEvent('dragover', { bubbles: true });
        playlistZone.dispatchEvent(dragOverEvent);

        // Simulate drop
        const dropEvent = new DragEvent('drop', { bubbles: true });
        // Manually set data since we improved the mock
        dropEvent.dataTransfer.setData('text/plain', 'video1.mp4');
        playlistZone.dispatchEvent(dropEvent);

        // Simulate dragend on source
        const dragEndEvent = new DragEvent('dragend', { bubbles: true });
        card.dispatchEvent(dragEndEvent);

        await vi.waitFor(() => {
            expect(global.fetch).toHaveBeenCalledWith(
                '/api/playlists/items',
                expect.objectContaining({
                    method: 'POST',
                    body: expect.stringContaining('"playlist_title":"My Playlist"')
                })
            );
        });
        expect(playlistZone.classList.contains('drag-over')).toBe(false);
        expect(document.body.classList.contains('is-dragging')).toBe(false);
    });

    it('merges local progress into history', async () => {
        // Mock local progress
        const localProgress = {
            'local-video.mp4': { pos: 50, last: Date.now() }
        };
        localStorage.setItem('disco-progress', JSON.stringify(localProgress));

        const historyCompletedBtn = document.getElementById('history-completed-btn');
        historyCompletedBtn.click();

        // Should fetch metadata for missing paths
        await vi.waitFor(() => {
            expect(global.fetch).toHaveBeenCalledWith(
                expect.stringContaining('paths=local-video.mp4'),
                expect.objectContaining({
                    headers: expect.objectContaining({
                        'X-Disco-Token': 'mock-test-token'
                    })
                })
            );
        });
    });

    it('persists sidebar state and resets on logo click', async () => {
        const catDetails = document.getElementById('details-categories');
        const playlistDetails = document.getElementById('details-playlists');

        // Initially categories is closed, playlists is closed
        expect(catDetails.open).toBe(false);
        expect(playlistDetails.open).toBe(false);

        // Set some filters
        playlistDetails.open = true;
        playlistDetails.dispatchEvent(new Event('toggle'));
        window.disco.state.filters.types = ['video', 'audio'];
        expect(window.disco.state.sidebarState['details-playlists']).toBe(true);

        // Click logo
        const logo = document.querySelector('.logo');
        const allMediaBtn = document.getElementById('all-media-btn');
        logo.click();

        expect(catDetails.open).toBe(false);
        expect(playlistDetails.open).toBe(false);
        expect(window.disco.state.sidebarState['details-playlists']).toBe(false);
        expect(window.disco.state.filters.types.length).toBe(0);
        expect(allMediaBtn.classList.contains('active')).toBe(true);
    });

    it('shows unplayable toast on 404', async () => {
        // Mock 404 response for play API
        global.fetch.mockImplementation((url) => {
            if (typeof url !== 'string') url = url.toString();
            if (url.includes('/api/play')) {
                return Promise.resolve({
                    ok: false,
                    status: 404,
                    headers: { get: () => null },
                    text: () => Promise.resolve('Not found')
                });
            }
            return Promise.resolve({
                ok: true,
                status: 200,
                headers: { get: () => '0' },
                json: () => Promise.resolve([])
            });
        });

        window.disco.state.player = 'system';
        await new Promise(r => setTimeout(r, 100));

        const card = document.querySelector('.media-card');
        const title = card.querySelector('.media-title');
        title.click();

        await vi.waitFor(() => {
            const toast = document.getElementById('toast');
            expect(toast.textContent).toContain('File not found');
        });
    });

    it('shows unplayable toast on 415', async () => {
        // Mock 415 response for play API
        global.fetch.mockImplementation((url) => {
            if (typeof url !== 'string') url = url.toString();
            if (url.includes('/api/play')) {
                return Promise.resolve({
                    ok: false,
                    status: 415,
                    headers: { get: () => null },
                    text: () => Promise.resolve('Unsupported Media Type')
                });
            }
            return Promise.resolve({
                ok: true,
                status: 200,
                headers: { get: () => '0' },
                json: () => Promise.resolve([])
            });
        });

        window.disco.state.player = 'system';
        await new Promise(r => setTimeout(r, 100));

        const card = document.querySelector('.media-card');
        const title = card.querySelector('.media-title');
        title.click();

        await vi.waitFor(() => {
            const toast = document.getElementById('toast');
            expect(toast.textContent).toContain('Unplayable (Unsupported)');
        });
    });

    it('paginates results', async () => {
        const nextBtn = document.getElementById('next-page');
        const prevBtn = document.getElementById('prev-page');
        const limitInput = document.getElementById('limit');

        // Mock current media to ensure "Next" is enabled (must be >= limit)
        limitInput.value = '1';
        limitInput.dispatchEvent(new Event('change'));

        const searchInput = document.getElementById('search-input');
        searchInput.value = 'test';
        searchInput.dispatchEvent(new KeyboardEvent('keypress', { key: 'Enter', bubbles: true }));

        await vi.waitFor(() => {
            expect(nextBtn.disabled).toBe(false);
        }, { timeout: 2000 });
        expect(prevBtn.disabled).toBe(true);

        nextBtn.click();
        expect(window.disco.state.currentPage).toBe(2);

        await vi.waitFor(() => {
            expect(global.fetch).toHaveBeenCalledWith(
                expect.stringContaining('offset=1'),
                expect.any(Object)
            );
        });

        // Wait for pagination to re-render and enable prevBtn
        await vi.waitFor(() => {
            expect(prevBtn.disabled).toBe(false);
        });

        prevBtn.click();
        expect(window.disco.state.currentPage).toBe(1);
    });

    it('applies sidebar bin filters', async () => {
        document.getElementById('details-size').open = true;

        await vi.waitFor(() => {
            const minSlider = document.getElementById('size-min-slider');
            const maxSlider = document.getElementById('size-max-slider');
            expect(minSlider).not.toBeNull();
            expect(maxSlider).not.toBeNull();
        });

        const minSlider = document.getElementById('size-min-slider');
        const maxSlider = document.getElementById('size-max-slider');
        minSlider.value = 0;
        maxSlider.value = 10;
        minSlider.dispatchEvent(new Event('change'));

        await vi.waitFor(() => {
            expect(global.fetch).toHaveBeenCalledWith(
                expect.stringContaining('size=p0-10'),
                expect.any(Object)
            );
        });
    });

    it('toggles settings options', async () => {
        const settingsBtn = document.getElementById('settings-button');
        settingsBtn.click();

        const themeSelect = document.getElementById('setting-theme');
        themeSelect.value = 'dark';
        themeSelect.dispatchEvent(new Event('change'));
    });

    it('drags an item into trash', async () => {
        const trashBtn = document.getElementById('trash-btn');
        const card = document.querySelector('.media-card');
        const path = card.dataset.path;

        // Simulate dragenter
        trashBtn.dispatchEvent(new DragEvent('dragenter', {
            dataTransfer: {
                setData: vi.fn(),
                getData: vi.fn(),
                effectAllowed: 'move',
                dropEffect: 'none'
            }
        }));
        expect(trashBtn.classList.contains('drag-over')).toBe(true);

        // Simulate drop
        const dropEvent = new DragEvent('drop', {
            dataTransfer: {
                getData: vi.fn((type) => type === 'text/plain' ? path : ''),
                effectAllowed: 'move',
                dropEffect: 'move'
            }
        });
        Object.defineProperty(dropEvent, 'target', { value: trashBtn });

        trashBtn.dispatchEvent(dropEvent);

        expect(trashBtn.classList.contains('drag-over')).toBe(false);
        await vi.waitFor(() => {
            expect(global.fetch).toHaveBeenCalledWith(
                '/api/delete',
                expect.objectContaining({
                    method: 'POST',
                    body: JSON.stringify({ path, restore: false })
                })
            );
        }, { timeout: 2000 });
    });
});
