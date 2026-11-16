import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { setupTestEnvironment } from './test-helper';

describe('Random Media', () => {
    beforeEach(async () => {
        await setupTestEnvironment();
    });

    it('fetches a random clip and plays it', async () => {
        const channelSurfBtn = document.getElementById('channel-surf-btn');
        expect(channelSurfBtn).toBeTruthy();

        // Mock fetch for random-clip
        const originalFetch = global.fetch;
        global.fetch = vi.fn().mockImplementation((url) => {
            if (url.includes('/api/random-clip')) {
                return Promise.resolve({
                    ok: true,
                    json: () => Promise.resolve({
                        path: '/path/to/video.mp4',
                        start: 10,
                        end: 25,
                        media_type: 'video/mp4'
                    })
                });
            }
            return originalFetch(url);
        });

        // Wait for app to initialize
        await new Promise(r => setTimeout(r, 100));

        channelSurfBtn.click();

        await vi.waitFor(() => {
            expect(global.fetch).toHaveBeenCalledWith(
                expect.stringContaining('/api/random-clip'),
                expect.anything()
            );

            const pipPlayer = document.getElementById('pip-player');
            expect(pipPlayer.classList.contains('hidden')).toBe(false);

            const title = document.getElementById('media-title');
            expect(title.textContent).toContain('video.mp4');
        });
    });

    it('restricts random media to the current media type', async () => {
        const channelSurfBtn = document.getElementById('channel-surf-btn');
        const state = window.disco.state;

        // Set current media to an image
        state.playback.item = { path: '/path/to/image.jpg', media_type: 'image/jpeg' };

        const originalFetch = global.fetch;
        global.fetch = vi.fn().mockImplementation((url) => {
            if (url.includes('/api/random-clip')) {
                // Check if media_type=image is in the URL
                expect(url).toContain('media_type=image');
                return Promise.resolve({
                    ok: true,
                    json: () => Promise.resolve({
                        path: '/path/to/another-image.jpg',
                        media_type: 'image/jpeg'
                    })
                });
            }
            return originalFetch(url);
        });

        channelSurfBtn.click();

        await vi.waitFor(() => {
            expect(global.fetch).toHaveBeenCalledWith(
                expect.stringContaining('media_type=image'),
                expect.anything()
            );
        });
    });

    it('stops random media if no more media of the same type is found', async () => {
        const channelSurfBtn = document.getElementById('channel-surf-btn');
        const toast = document.getElementById('toast');
        const state = window.disco.state;

        // Set current media to video
        state.playback.item = { path: '/path/to/video.mp4', media_type: 'video/mp4' };

        global.fetch = vi.fn().mockImplementation((url) => {
            if (url.includes('/api/random-clip')) {
                return Promise.resolve({
                    ok: false,
                    status: 404
                });
            }
            return Promise.resolve({ ok: true });
        });

        channelSurfBtn.click();

        await vi.waitFor(() => {
            expect(toast.textContent).toContain('No more video found');
            expect(toast.classList.contains('hidden')).toBe(false);
        });
    });
});
