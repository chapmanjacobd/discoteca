import { describe, it, expect, beforeEach, vi } from 'vitest';
import { setupTestEnvironment } from './test-helper';

describe('Sidebar Media Options Filtering', () => {
    beforeEach(async () => {
        document.body.innerHTML = '';
        await setupTestEnvironment();
    });

    it('filters media by type on All Media page', async () => {
        const audioBtn = document.querySelector('#media-type-list .category-btn[data-type="audio"]');

        // Initial state: nothing selected (means everything)
        // Select audio
        audioBtn.click();

        await vi.waitFor(() => {
            const calls = global.fetch.mock.calls;
            // Verify type=audio is in the URL
            const hasAudioQuery = calls.some(call =>
                call[0].includes('/api/query') &&
                call[0].includes('type=audio') &&
                call[0].includes('include_counts=true')
            );
            expect(hasAudioQuery).toBe(true);
        });
    });

    it('filters media by type on Completed page', async () => {
        const historyCompletedBtn = document.getElementById('history-completed-btn');
        historyCompletedBtn.click();

        await vi.waitFor(() => {
            expect(global.fetch).toHaveBeenCalledWith(expect.stringContaining('completed=true'), expect.any(Object));
        });

        // Now toggle audio on
        const audioBtn = document.querySelector('#media-type-list .category-btn[data-type="audio"]');
        audioBtn.click();

        await vi.waitFor(() => {
            const calls = global.fetch.mock.calls;
            const hasCompletedAudioQuery = calls.some(call =>
                call[0].includes('/api/query') &&
                call[0].includes('completed=true') &&
                call[0].includes('type=audio') &&
                call[0].includes('include_counts=true')
            );
            expect(hasCompletedAudioQuery).toBe(true);
        });
    });

    it('shows everything when all media types are unselected', async () => {
        // Wait for buttons to be rendered
        await vi.waitFor(() => {
            const buttons = document.querySelectorAll('#media-type-list .category-btn[data-type]');
            expect(buttons.length).toBeGreaterThan(0);
        }, { timeout: 3000 });

        // Toggle some type on
        const videoBtn = document.querySelector('#media-type-list .category-btn[data-type="video"]');
        videoBtn.click();
        
        // Verify type=video is in the URL
        await vi.waitFor(() => {
            const calls = global.fetch.mock.calls;
            const hasVideoQuery = calls.some(call =>
                call[0].includes('/api/query') &&
                call[0].includes('type=video')
            );
            expect(hasVideoQuery).toBe(true);
        }, { timeout: 3000 });

        // Unselect everything
        videoBtn.click();

        await vi.waitFor(() => {
            const calls = global.fetch.mock.calls;
            // If all are unselected, type parameter should NOT be in the URL
            const hasCleanQuery = calls.some(call =>
                call[0].includes('/api/query') &&
                !call[0].includes('type=')
            );
            expect(hasCleanQuery).toBe(true);
        }, { timeout: 3000 });
    });

    it('re-renders episodes results when media type filter is clicked on Group view', async () => {
        const viewGroup = document.getElementById('view-group');
        viewGroup.click();

        await vi.waitFor(() => {
            const calls = global.fetch.mock.calls;
            const hasEpisodesCall = calls.some(call => call[0].includes('/api/episodes'));
            expect(hasEpisodesCall).toBe(true);
        });

        // Wait for buttons to be rendered
        let audioBtn;
        await vi.waitFor(() => {
            audioBtn = document.querySelector('#media-type-list .category-btn[data-type="audio"]');
            expect(audioBtn).not.toBeNull();
        });

        // Toggle audio on (stays in Group view, filters episodes)
        audioBtn.click();

        // It should call /api/episodes again to re-fetch with the type filter
        await vi.waitFor(() => {
            const calls = global.fetch.mock.calls;
            const episodesCalls = calls.filter(call => call[0].includes('/api/episodes'));
            // Expect at least two calls: one for initial Group view, one for filter change
            expect(episodesCalls.length).toBeGreaterThanOrEqual(2);
        });
    });

});
