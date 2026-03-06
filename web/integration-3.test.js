import { describe, it, expect, beforeEach, vi } from 'vitest';
import { setupTestEnvironment } from './test-helper';

describe('Advanced Integration Tests', () => {
    beforeEach(async () => {
        await setupTestEnvironment();
    });

    it('persists filters when switching between All Media and Disk Usage', async () => {
        // 1. Start in All Media, select a category
        const comedyBtn = Array.from(document.querySelectorAll('.sidebar .category-btn'))
            .find(btn => btn.textContent.includes('comedy'));
        expect(comedyBtn).toBeTruthy();
        comedyBtn.click();

        await vi.waitFor(() => {
            const calls = global.fetch.mock.calls;
            const lastCall = calls[calls.length - 1];
            expect(lastCall[0]).toContain('category=comedy');
        });

        // 2. Select Video type only
        const videoBtn = document.querySelector('#media-type-list .category-btn[data-type="video"]');
        videoBtn.click(); // Initial state is nothing selected (all types). Clicking video selects ONLY video.

        await vi.waitFor(() => {
            const calls = global.fetch.mock.calls;
            const hasQuery = calls.some(call =>
                call[0].includes('category=comedy') &&
                call[0].includes('/api/query')
            );
            expect(hasQuery).toBe(true);
        });

        // 3. Switch to Disk Usage mode
        const duBtn = document.getElementById('du-btn');
        duBtn.click();

        await vi.waitFor(() => {
            const calls = global.fetch.mock.calls;
            const hasDUQuery = calls.some(call =>
                call[0].includes('/api/du') &&
                call[0].includes('category=comedy')
            );
            expect(hasDUQuery).toBe(true);
        });
    });

    it('handles view mode switching (Grid, Group, Details)', async () => {
        const viewGroup = document.getElementById('view-group');
        const viewDetails = document.getElementById('view-details');
        const viewGrid = document.getElementById('view-grid');

        // Switch to Group view
        viewGroup.click();
        await vi.waitFor(() => {
            const calls = global.fetch.mock.calls;
            const lastCall = calls[calls.length - 1];
            expect(lastCall[0]).toContain('/api/episodes');
            expect(viewGroup.classList.contains('active')).toBe(true);
        });

        // Switch to Details (Table) view
        viewDetails.click();
        await vi.waitFor(() => {
            const calls = global.fetch.mock.calls;
            const hasQueryCall = calls.some(call => call[0].includes('/api/query'));
            expect(hasQueryCall).toBe(true);
            expect(viewDetails.classList.contains('active')).toBe(true);
            expect(document.querySelector('.details-table')).toBeTruthy();
        });

        // Switch back to Grid
        viewGrid.click();
        await vi.waitFor(() => {
            expect(viewGrid.classList.contains('active')).toBe(true);
            expect(document.querySelector('.grid')).toBeTruthy();
        });
    });

    it('filters by progress states under History & Progress', async () => {
        const unplayedBtn = document.getElementById('history-unplayed-btn');
        const inProgressBtn = document.getElementById('history-in-progress-btn');
        const completedBtn = document.getElementById('history-completed-btn');

        unplayedBtn.click();
        await vi.waitFor(() => {
            const calls = global.fetch.mock.calls;
            const lastCall = calls[calls.length - 1];
            expect(lastCall[0]).toContain('unplayed=true');
        });

        inProgressBtn.click();
        await vi.waitFor(() => {
            const calls = global.fetch.mock.calls;
            const lastCall = calls[calls.length - 1];
            expect(lastCall[0]).toContain('unfinished=true');
            expect(lastCall[0]).not.toContain('unplayed=true');
        });

        completedBtn.click();
        await vi.waitFor(() => {
            const calls = global.fetch.mock.calls;
            const lastCall = calls[calls.length - 1];
            expect(lastCall[0]).toContain('completed=true');
        });
    });

    it('filters by sidebar bins (Episodes, Size, Duration)', async () => {
        // Open details to ensure visibility
        document.getElementById('details-episodes').open = true;

        const minSlider = document.getElementById('episodes-min-slider');
        const maxSlider = document.getElementById('episodes-max-slider');
        minSlider.value = 10;
        maxSlider.value = 50;
        minSlider.dispatchEvent(new Event('change'));

        await vi.waitFor(() => {
            const calls = global.fetch.mock.calls;
            const lastCall = calls[calls.length - 1];
            expect(lastCall[0]).toContain('episodes=p10-50');
        });
    });

    it('navigates to Captions page and performs keyword search', async () => {
        // 1. Normal search on All Media
        const allMediaBtn = document.getElementById('all-media-btn');
        allMediaBtn.click();

        const searchInput = document.getElementById('search-input');
        searchInput.value = 'normal';
        searchInput.dispatchEvent(new Event('input'));
        await new Promise(r => setTimeout(r, 400)); // wait for debounce

        await vi.waitFor(() => {
            const calls = global.fetch.mock.calls;
            const hasQueryCall = calls.some(call => call[0].includes('/api/query') && call[0].includes('search=normal'));
            expect(hasQueryCall).toBe(true);

            // Check that results container has normal media cards
            expect(document.querySelector('.grid .media-card')).toBeTruthy();
            expect(document.querySelector('.captions-list-view')).toBeNull();
        });

        // 2. Caption search on Captions page
        const captionsBtn = document.getElementById('captions-btn');
        captionsBtn.click();

        await vi.waitFor(() => {
            const calls = global.fetch.mock.calls;
            // The URL might contain either view=captions or captions=true depending on implementation
            const hasCaptionsRequest = calls.some(call => call[0].includes('captions=true') || call[0].includes('view=captions'));
            expect(hasCaptionsRequest).toBe(true);
        });

        searchInput.value = 'findme';
        searchInput.dispatchEvent(new Event('input'));

        // Wait for debounce
        await new Promise(r => setTimeout(r, 400));

        await vi.waitFor(() => {
            const calls = global.fetch.mock.calls;
            const hasCaptionsQuery = calls.some(call =>
                (call[0].includes('captions=true') || call[0].includes('view=captions')) &&
                call[0].includes('search=findme')
            );
            expect(hasCaptionsQuery).toBe(true);

            // Check that results container has caption cards (grouped by media)
            expect(document.querySelector('.captions-list-view .caption-media-card')).toBeTruthy();
            expect(document.querySelector('.grid .media-card')).toBeNull();
        });
    });
});
