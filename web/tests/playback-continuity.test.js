import { describe, it, expect, beforeEach, vi } from 'vitest';
import { setupTestEnvironment } from './test-helper';

describe('Playback Continuity', () => {
    let state;
    beforeEach(async () => {
        localStorage.clear();
        await setupTestEnvironment();
        state = window.disco.state;
        
        // Mock fullscreen API
        document.exitFullscreen = vi.fn().mockResolvedValue(undefined);
        Object.defineProperty(document, 'fullscreenElement', {
            get: () => global.mockFullscreenElement,
            configurable: true
        });
        global.mockFullscreenElement = null;
    });

    it('should maintain fullscreen when playing next sibling', async () => {
        const item1 = { path: 'video1.mp4', type: 'video/mp4' };
        const item2 = { path: 'video2.mp4', type: 'video/mp4' };
        window.disco.currentMedia = [item1, item2];

        // 1. Play first item
        window.disco.openActivePlayer(item1, true);
        const pipPlayer = document.getElementById('pip-player');
        expect(pipPlayer.classList.contains('hidden')).toBe(false);

        // 2. Go fullscreen
        const pipViewer = document.getElementById('media-viewer');
        pipViewer.requestFullscreen = vi.fn().mockImplementation(function() {
            global.mockFullscreenElement = this;
            return Promise.resolve();
        });
        
        await pipViewer.requestFullscreen();
        expect(document.fullscreenElement).toBe(pipViewer);

        // 3. Play next sibling (ArrowRight / 'n')
        // This calls playSibling(1, true)
        window.disco.playSibling(1, true);

        // VERIFY: Fullscreen should STILL be active for the next video
        // If it failed, closeActivePlayer would have called exitFullscreen
        expect(document.exitFullscreen).not.toHaveBeenCalled();
        expect(document.fullscreenElement).toBe(pipViewer);
        expect(state.playback.item.path).toBe('video2.mp4');
    });
});
