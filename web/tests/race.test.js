import { describe, it, expect, beforeEach, vi } from 'vitest';
import { setupTestEnvironment } from './test-helper';

describe('Race condition between progress update and search', () => {
    beforeEach(async () => {
        await setupTestEnvironment();
    });

    it('awaits progress update before performing search on item end', async () => {
        const item1 = {
            path: 'video1.mp4',
            type: 'video/mp4',
            duration: 600,
            playhead: 0
        };
        const item2 = {
            path: 'video2.mp4',
            type: 'video/mp4',
            duration: 600,
            playhead: 0
        };

        window.disco.state.filters.unplayed = true;
        window.disco.state.autoplay = true;
        window.disco.state.filters.all = false; // Ensure pagination can happen
        window.disco.state.page = 'search';
        window.disco.state.readOnly = false; // Ensure server sync happens
        window.disco.state.localResume = true; // Ensure local progress is saved

        let progressFinished = false;
        let searchStartedAfterProgress = false;
        let progressResolve;
        let progressPromise = new Promise(resolve => { progressResolve = resolve; });

        // Custom fetch mock to simulate slow progress update
        global.fetch = vi.fn().mockImplementation(async (url) => {
            if (typeof url !== 'string') url = url.toString();

            if (url.includes('/api/progress')) {
                await new Promise(resolve => setTimeout(resolve, 100));
                progressFinished = true;
                progressResolve();
                return { ok: true, status: 200, json: () => Promise.resolve({}) };
            }
            if (url.includes('/api/query')) {
                if (progressFinished) {
                    searchStartedAfterProgress = true;
                }
                return {
                    ok: true,
                    status: 200,
                    headers: { get: () => '2' },
                    json: () => Promise.resolve([item1, item2])
                };
            }
            if (url.includes('/api/play')) {
                return { ok: true, status: 200, json: () => Promise.resolve({}) };
            }
            return { ok: true, status: 200, json: () => Promise.resolve([]) };
        });

        // Trigger a search to populate currentMedia
        const searchInput = document.getElementById('search-input');
        searchInput.value = '';
        searchInput.dispatchEvent(new KeyboardEvent('keypress', { key: 'Enter', bubbles: true }));

        // Wait for search to finish and results to be rendered
        await vi.waitFor(() => {
            const results = document.querySelectorAll('.media-card');
            return results.length > 0;
        }, { timeout: 1000 });

        // Small delay to ensure all async operations complete
        await new Promise(resolve => setTimeout(resolve, 100));

        // Set up state to force pagination on next item
        // Set currentMedia to have only 1 item, but totalCount to be 2
        window.disco.currentMedia = [item1];
        window.disco.state.totalCount = 2;
        window.disco.state.currentPage = 1;
        window.disco.state.filters.limit = 1;

        // Mock startTime to ensure progress sync happens (>90s)
        window.disco.state.playback.startTime = Date.now() - 200000;
        window.disco.state.playback.hasMarkedComplete = false; // Reset to allow marking complete

        // Trigger openInPiP for item1
        await window.disco.openInPiP(item1);

        // Force currentMedia to have only 1 item to trigger pagination on next
        window.disco.currentMedia = [item1];
        window.disco.state.totalCount = 2;
        window.disco.state.currentPage = 1;
        window.disco.state.filters.limit = 1;

        const video = document.querySelector('video');

        // Manually trigger onended
        await video.onended();

        // Wait for progress to finish
        await progressPromise;

        // Wait for the pagination search to be triggered
        await vi.waitFor(() => {
            expect(global.fetch).toHaveBeenCalledWith(
                expect.stringContaining('/api/query'),
                expect.any(Object)
            );
        }, { timeout: 1000 });

        // Verify that progress finished before search started
        expect(progressFinished).toBe(true);
        expect(searchStartedAfterProgress).toBe(true);
    });
});
