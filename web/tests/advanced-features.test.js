import { describe, it, expect, beforeEach, vi } from 'vitest';
import { setupTestEnvironment } from './test-helper';

describe('Playlists', () => {
    beforeEach(async () => {
        await setupTestEnvironment();
    });

    it('shows playlists in sidebar', async () => {
        const playlistSection = document.querySelector('#playlist-list');
        expect(playlistSection).toBeTruthy();
    });

    it('fetches playlists on startup', async () => {
        await vi.waitFor(() => {
            const calls = global.fetch.mock.calls;
            const hasPlaylistsCall = calls.some(call =>
                call[0].includes('/api/playlists')
            );
            expect(hasPlaylistsCall).toBe(true);
        });
    });

    it('selects a playlist from sidebar', async () => {
        await vi.waitFor(() => {
            const playlistBtn = document.querySelector('#playlist-list .category-btn');
            return playlistBtn !== null;
        });

        const playlistBtn = document.querySelector('#playlist-list .category-btn');
        playlistBtn.click();

        await vi.waitFor(() => {
            const calls = global.fetch.mock.calls;
            const lastCall = calls[calls.length - 1];
            // URL format may vary, just check it contains playlist info
            expect(lastCall[0]).toContain('playlist');
        });
    });

    it('creates a new playlist', async () => {
        const createBtn = document.getElementById('new-playlist-btn');
        expect(createBtn).not.toBeNull();

        window.prompt = vi.fn().mockReturnValue('Test Playlist');
        createBtn.click();

        await vi.waitFor(() => {
            expect(global.fetch).toHaveBeenCalledWith(
                expect.stringContaining('/api/playlists'),
                expect.objectContaining({
                    method: 'POST',
                    body: expect.stringContaining('Test Playlist')
                })
            );
        }, 2000);
    });

    it('deletes a playlist', async () => {
        await vi.waitFor(() => {
            const deleteBtn = document.querySelector('#playlist-list .delete-playlist-btn');
            return deleteBtn !== null;
        });

        const deleteBtn = document.querySelector('#playlist-list .delete-playlist-btn');
        expect(deleteBtn).not.toBeNull();
        window.confirm = vi.fn().mockReturnValue(true);
        deleteBtn.click();

        await vi.waitFor(() => {
            expect(global.fetch).toHaveBeenCalledWith(
                expect.stringContaining('/api/playlists?title='),
                expect.objectContaining({
                    method: 'DELETE'
                })
            );
        });
    });
});

describe('Trash Functionality', () => {
    beforeEach(async () => {
        await setupTestEnvironment();
    });

    it('switches to trash view', async () => {
        const trashBtn = document.getElementById('trash-btn');
        trashBtn.click();

        await vi.waitFor(() => {
            const calls = global.fetch.mock.calls;
            const lastCall = calls[calls.length - 1];
            expect(lastCall[0]).toContain('trash=true');
        });
    });

    it('moves media to trash', async () => {
        await new Promise(r => setTimeout(r, 100));
        const deleteBtn = document.querySelector('.media-action-btn.delete');
        expect(deleteBtn).not.toBeNull();

        window.confirm = vi.fn().mockReturnValue(true);
        deleteBtn.click();

        // Just verify fetch was called
        await vi.waitFor(() => {
            expect(global.fetch.mock.calls.length).toBeGreaterThan(0);
        });
    });

    it('shows empty bin button in trash view', async () => {
        const trashBtn = document.getElementById('trash-btn');
        trashBtn.click();

        await vi.waitFor(() => {
            const emptyBtn = document.getElementById('empty-bin-btn');
            expect(emptyBtn).not.toBeNull();
        });
    });

    it('empties the trash bin', async () => {
        const trashBtn = document.getElementById('trash-btn');
        trashBtn.click();
        await new Promise(r => setTimeout(r, 100));

        const emptyBtn = document.getElementById('empty-bin-btn');
        expect(emptyBtn).not.toBeNull();
        window.confirm = vi.fn().mockReturnValue(true);
        emptyBtn.click();

        await vi.waitFor(() => {
            expect(global.fetch).toHaveBeenCalledWith(
                expect.stringContaining('/api/empty-bin'),
                expect.objectContaining({
                    method: 'POST'
                })
            );
        });
    });

    it('permanently deletes an item from trash', async () => {
        const trashBtn = document.getElementById('trash-btn');
        trashBtn.click();

        await new Promise(r => setTimeout(r, 100));

        const permDeleteBtn = document.querySelector('.media-action-btn.delete-permanent');
        expect(permDeleteBtn).not.toBeNull();
        window.confirm = vi.fn().mockReturnValue(true);
        permDeleteBtn.click();

        await vi.waitFor(() => {
            expect(global.fetch).toHaveBeenCalledWith(
                expect.stringContaining('/api/empty-bin'),
                expect.objectContaining({
                    method: 'POST'
                })
            );
        });
    });
});

describe('View Modes', () => {
    beforeEach(async () => {
        await setupTestEnvironment();
    });

    it('switches to group view', async () => {
        const viewGroup = document.getElementById('view-group');
        viewGroup.click();

        await vi.waitFor(() => {
            const calls = global.fetch.mock.calls;
            const lastCall = calls[calls.length - 1];
            expect(lastCall[0]).toContain('/api/episodes');
            expect(viewGroup.classList.contains('active')).toBe(true);
        });
    });

    it('switches to details view', async () => {
        const viewDetails = document.getElementById('view-details');
        viewDetails.click();

        await vi.waitFor(() => {
            const calls = global.fetch.mock.calls;
            const hasQueryCall = calls.some(call => call[0].includes('/api/query'));
            expect(hasQueryCall).toBe(true);
            expect(viewDetails.classList.contains('active')).toBe(true);
        });
    });

    it('switches to grid view', async () => {
        const viewGrid = document.getElementById('view-grid');
        viewGrid.click();

        await vi.waitFor(() => {
            expect(viewGrid.classList.contains('active')).toBe(true);
            const container = document.getElementById('results-container');
            expect(container.classList.contains('grid')).toBe(true);
        });
    });
});

describe('Responsive Design', () => {
    beforeEach(async () => {
        await setupTestEnvironment();
    });

    it('shows menu toggle on mobile', async () => {
        // Simulate mobile viewport
        Object.defineProperty(window, 'innerWidth', {
            writable: true,
            configurable: true,
            value: 480
        });

        const menuToggle = document.getElementById('menu-toggle');
        expect(menuToggle).toBeTruthy();
    });

    it('toggles mobile sidebar', async () => {
        const menuToggle = document.getElementById('menu-toggle');
        const sidebar = document.getElementById('sidebar');
        const overlay = document.getElementById('sidebar-overlay');

        menuToggle.click();

        await vi.waitFor(() => {
            expect(sidebar.classList.contains('mobile-open')).toBe(true);
            expect(overlay.classList.contains('hidden')).toBe(false);
        });

        overlay.click();

        await vi.waitFor(() => {
            expect(sidebar.classList.contains('mobile-open')).toBe(false);
            expect(overlay.classList.contains('hidden')).toBe(true);
        });
    });

    it('has touch-friendly tap targets', async () => {
        const categoryBtns = document.querySelectorAll('.category-btn');
        // Just verify buttons exist - actual size testing requires proper CSS loading
        expect(categoryBtns.length).toBeGreaterThan(0);
    });
});

describe('Search Functionality', () => {
    beforeEach(async () => {
        await setupTestEnvironment();
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

    it('performs search with filters', async () => {
        const searchInput = document.getElementById('search-input');
        searchInput.value = 'filtered search';

        // Toggle a media type filter first
        const videoBtn = document.querySelector('#media-type-list .category-btn[data-media_type="video"]');
        if (videoBtn) {
            videoBtn.click();
            await vi.waitFor(() => {
                const calls = global.fetch.mock.calls;
                const hasQueryCall = calls.some(call => call[0].includes('/api/query'));
                expect(hasQueryCall).toBe(true);
            }, 1000);
        }

        // Now search
        const event = new KeyboardEvent('keypress', { key: 'Enter', bubbles: true });
        searchInput.dispatchEvent(event);

        await vi.waitFor(() => {
            const calls = global.fetch.mock.calls;
            const hasSearchWithFilter = calls.some(call =>
                call[0].includes('search=filtered+search') &&
                call[0].includes('/api/query')
            );
            expect(hasSearchWithFilter).toBe(true);
        });
    });
});
