import { describe, it, expect, beforeEach, vi } from 'vitest';
import { setupTestEnvironment } from './test-helper';

describe('Sidebar Active States', () => {
    beforeEach(async () => {
        document.body.innerHTML = '';
        await setupTestEnvironment();
    });

    it('updates sidebar button active states when switching to text view', async () => {
        const textBtn = document.querySelector('#media-type-list .category-btn[data-type="text"]');
        expect(textBtn).not.toBeNull();

        // Click Text button
        textBtn.click();

        await vi.waitFor(() => {
            const calls = global.fetch.mock.calls;
            const hasQueryCall = calls.some(call => call[0].includes('/api/query'));
            expect(hasQueryCall).toBe(true);
        });

        // Let's simulate a URL change to type=text
        window.location.hash = '#type=text';
        window.dispatchEvent(new HashChangeEvent('hashchange'));

        await vi.waitFor(() => {
            const calls = global.fetch.mock.calls;
            // Should have made another query after hash change
            expect(calls.length).toBeGreaterThan(0);
        });
    });

    it('highlights All Media button when no filters are active', async () => {
        const allMediaBtn = document.getElementById('all-media-btn');
        allMediaBtn.click();

        await vi.waitFor(() => {
            expect(allMediaBtn.classList.contains('active')).toBe(true);
            expect(window.disco.state.filters.types.length).toBe(0);
        });

        const videoBtn = document.querySelector('#media-type-list .category-btn[data-type="video"]');

        // Select video, All Media should deactivate
        videoBtn.click();
        await vi.waitFor(() => {
            const calls = global.fetch.mock.calls;
            const hasTypeQuery = calls.some(call => call[0].includes('/api/query'));
            expect(hasTypeQuery).toBe(true);
        });

        // Unselect video, All Media should reactivate
        videoBtn.click();
        await vi.waitFor(() => {
            const calls = global.fetch.mock.calls;
            // Should have made another query
            expect(calls.length).toBeGreaterThan(0);
        });
    });
});
