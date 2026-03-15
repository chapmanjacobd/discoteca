import { describe, it, expect, beforeEach, vi } from 'vitest';
import { setupTestEnvironment } from './test-helper';

describe('Image Slideshow', () => {
    beforeEach(async () => {
        await setupTestEnvironment();
    });

    it('continues slideshow to the next image after delay', async () => {
        // Setup: Multiple images in the media list
        const images = [
            { path: 'images/photo1.png', type: 'image/png', db: 'test.db' },
            { path: 'images/photo2.png', type: 'image/png', db: 'test.db' },
            { path: 'images/photo3.png', type: 'image/png', db: 'test.db' },
        ];

        // Mock the API to return our images
        global.fetch = vi.fn().mockImplementation((url) => {
            if (url.includes('/api/query')) {
                return Promise.resolve({
                    ok: true,
                    status: 200,
                    json: () => Promise.resolve({
                        items: images,
                        counts: {
                            episodes: [], size: [], duration: [],
                            episodes_min: 1, episodes_max: 3,
                            size_min: 0, size_max: 100 * 1024 * 1024,
                            duration_min: 0, duration_max: 3600,
                            episodes_percentiles: [0, 1, 2],
                            size_percentiles: [0, 1024, 2048],
                            duration_percentiles: [0, 60, 120]
                        }
                    })
                });
            }
            return Promise.resolve({
                ok: true,
                status: 200,
                json: () => Promise.resolve({ databases: ['test.db'], read_only: false, dev: false })
            });
        });

        // Enable image autoplay
        localStorage.setItem('disco-image-autoplay', 'true');
        // Set slideshow delay to 1 second for faster test
        localStorage.setItem('disco-slideshow-delay', '1');
        window.disco.state.slideshowDelay = 1; // Also update state directly

        // Set currentMedia so playSibling can navigate through images
        window.disco.currentMedia = images;

        // Open first image in PiP
        await window.disco.openInPiP(images[0], true);

        // Wait for image to load
        await new Promise(resolve => setTimeout(resolve, 50));

        // Verify first image is loaded
        const img = document.querySelector('#media-viewer img');
        expect(img).toBeTruthy();
        expect(window.disco.state.playback.item.path).toContain('photo1.png');

        // Start slideshow
        window.disco.startSlideshow();

        // Verify slideshow timer is set
        expect(window.disco.state.playback.slideshowTimer).toBeTruthy();

        // Wait for first transition (1 second delay + buffer)
        await new Promise(resolve => setTimeout(resolve, 1100));

        // Give extra time for state update after playSibling
        await new Promise(resolve => setTimeout(resolve, 50));

        // After first transition, verify we're on the second image
        expect(window.disco.state.playback.item.path).toContain('photo2.png');

        // BUG DETECTION: The slideshow timer should be set again for the next transition,
        // but it's null - the slideshow stops after one image
        expect(window.disco.state.playback.slideshowTimer).toBeTruthy();

        // Wait for second transition
        await new Promise(resolve => setTimeout(resolve, 1100));
        await new Promise(resolve => setTimeout(resolve, 50));

        // Should be on third image now
        expect(window.disco.state.playback.item.path).toContain('photo3.png');

        // Slideshow should still be running (timer should be set for next image)
        // This fails because the slideshow stops after the first transition
        expect(window.disco.state.playback.slideshowTimer).toBeTruthy();
    });

    it('stops slideshow when user interacts', async () => {
        const images = [
            { path: 'images/photo1.png', type: 'image/png', db: 'test.db' },
            { path: 'images/photo2.png', type: 'image/png', db: 'test.db' },
        ];

        localStorage.setItem('disco-image-autoplay', 'true');
        localStorage.setItem('disco-slideshow-delay', '5');

        await window.disco.openInPiP(images[0], true);
        await new Promise(resolve => setTimeout(resolve, 50));

        window.disco.startSlideshow();
        expect(window.disco.state.playback.slideshowTimer).toBeTruthy();

        // Simulate user interaction (playSibling with isUser=true)
        window.disco.playSibling(1, true);

        // Slideshow should be stopped
        expect(window.disco.state.playback.slideshowTimer).toBeNull();
        expect(window.disco.state.imageAutoplay).toBe(false);
    });
});
