import { describe, it, expect, beforeEach, vi } from 'vitest';
import { setupTestEnvironment } from './test-helper';

describe('Rapid Playback Clicks', () => {
    let state;
    beforeEach(async () => {
        localStorage.clear();
        await setupTestEnvironment();
        state = window.disco.state;
    });

    it('should handle rapid clicks on the same item without unplayable error', async () => {
        const item = { path: 'video.mkv', type: 'video/x-matroska' };
        
        // Mock showToast to see if unplayable error is shown
        const showToastSpy = vi.spyOn(window.disco, 'showToast');

        // Simulate rapid clicks
        window.disco.openActivePlayer(item, true);
        
        // Find the video element that was just created
        const mediaViewer = document.getElementById('media-viewer');
        const video = mediaViewer.querySelector('video');
        
        if (video) {
            // Simulate another click which would call closeActivePlayer and set src=""
            // In a real browser, this might trigger error on the video element
            window.disco.openActivePlayer(item, true);
            
            // Manually trigger error event on the OLD video element (simulating what browser does)
            const errorEvent = new Event('error');
            video.dispatchEvent(errorEvent);
        }

        // Wait for async operations
        await new Promise(resolve => setTimeout(resolve, 1500));

        // Verify no "Unplayable" toast was shown
        const allToastCalls = showToastSpy.mock.calls.map(args => args[0]);
        const unplayableToasts = allToastCalls.filter(msg => msg && msg.includes('Unplayable'));
        
        expect(unplayableToasts).toEqual([]);
    });
});
