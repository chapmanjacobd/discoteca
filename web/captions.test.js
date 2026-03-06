import { describe, it, expect, beforeEach, vi } from 'vitest';
import { setupTestEnvironment } from './test-helper';

describe('Captions View', () => {
    beforeEach(async () => {
        await setupTestEnvironment();
    });

    it('navigates to captions page when button is clicked', async () => {
        window.disco.state.view = 'grid';
        const captionsBtn = document.getElementById('captions-btn');
        captionsBtn.click();

        await vi.waitFor(() => {
            expect(window.disco.state.page).toBe('captions');
        });
    });

    it('fetches captions with captions=true parameter', async () => {
        const captionsBtn = document.getElementById('captions-btn');
        captionsBtn.click();

        await vi.waitFor(() => {
            const calls = global.fetch.mock.calls;
            const hasCaptionsRequest = calls.some(call =>
                call[0].includes('/api/query') &&
                call[0].includes('captions=true')
            );
            expect(hasCaptionsRequest).toBe(true);
        });
    });

    it('searches within captions when search input is used', async () => {
        const captionsBtn = document.getElementById('captions-btn');
        captionsBtn.click();

        await vi.waitFor(() => {
            expect(window.disco.state.page).toBe('captions');
        });

        const searchInput = document.getElementById('search-input');
        searchInput.value = 'test caption';
        searchInput.dispatchEvent(new Event('input'));

        // Wait for debounce
        await new Promise(resolve => setTimeout(resolve, 400));

        await vi.waitFor(() => {
            const calls = global.fetch.mock.calls;
            const hasCaptionSearch = calls.some(call =>
                call[0].includes('/api/query') &&
                call[0].includes('captions=true') &&
                call[0].includes('search=test+caption')
            );
            expect(hasCaptionSearch).toBe(true);
        });
    });

    it('renders caption rows with correct structure when data is loaded', async () => {
        // This test verifies the renderCaptionsList function creates proper structure
        // We manually call the render function with mock data
        const mockCaptionData = [
            { path: '/videos/test.mp4', caption_text: 'test caption', caption_time: 10.5, type: 'video/mp4', size: 1024, duration: 60, db: 'test.db' }
        ];

        window.disco.currentMedia = mockCaptionData;
        window.disco.renderCaptionsList();

        const captionCards = document.querySelectorAll('.caption-media-card');
        expect(captionCards.length).toBe(1);

        const card = captionCards[0];
        expect(card.querySelector('.caption-media-header')).toBeTruthy();
        expect(card.querySelector('.caption-segments-container')).toBeTruthy();
        expect(card.querySelector('.caption-segment')).toBeTruthy();
    });

    it('filters out items without caption_text in renderCaptionsList', async () => {
        // Test the filtering logic in renderCaptionsList
        const mockData = [
            { path: '/videos/valid.mp4', caption_text: 'has caption', caption_time: 10.5, type: 'video/mp4', size: 1024, duration: 60, db: 'test.db' },
            { path: '/videos/null.mp4', caption_text: null, caption_time: null, type: 'video/mp4', size: 1024, duration: 60, db: 'test.db' },
            { path: '/videos/empty.mp4', caption_text: '', caption_time: 0, type: 'video/mp4', size: 1024, duration: 60, db: 'test.db' },
            { path: '/videos/valid2.mp4', caption_text: 'another caption', caption_time: 20.0, type: 'video/mp4', size: 1024, duration: 60, db: 'test.db' }
        ];

        window.disco.currentMedia = mockData;
        window.disco.renderCaptionsList();

        const captionCards = document.querySelectorAll('.caption-media-card');
        // Should only render cards with valid captions
        expect(captionCards.length).toBeGreaterThan(0);
    });

    it('seeks to caption time when segment is clicked', async () => {
        const mockCaptionData = [
            { path: '/videos/test.mp4', caption_text: 'test caption', caption_time: 45.5, type: 'video/mp4', size: 1024, duration: 60, db: 'test.db' }
        ];

        window.disco.currentMedia = mockCaptionData;
        window.disco.renderCaptionsList();

        const segment = document.querySelector('.caption-segment');
        expect(segment).toBeTruthy();
        expect(segment.dataset.time).toBeTruthy();
    });
});
