import { describe, it, expect, beforeEach, vi } from 'vitest';
import { setupTestEnvironment } from './test-helper';

describe('Navigation Shortcuts', () => {
    let playSiblingSpy;

    beforeEach(async () => {
        await setupTestEnvironment();
        const item = { path: 'v2.mp4', type: 'video/mp4', duration: 100 };
        window.disco.currentMedia = [{ path: 'v1.mp4' }, item, { path: 'v3.mp4' }];
        
        // Mock playSibling on the window object so the code calls our spy
        playSiblingSpy = vi.spyOn(window.disco, 'playSibling');
        
        await window.disco.openInPiP(item);
    });

    it('goes to previous item if ArrowLeft is pressed at 0:00', async () => {
        const video = document.querySelector('video');
        // Mock currentTime and duration for JSDOM
        Object.defineProperty(video, 'currentTime', { value: 0, writable: true });
        
        document.dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowLeft', bubbles: true }));
        expect(playSiblingSpy).toHaveBeenCalledWith(-1, true);
    });

    it('goes to next item if ArrowRight is pressed at the end', async () => {
        const video = document.querySelector('video');
        // Mock currentTime and duration for JSDOM
        Object.defineProperty(video, 'currentTime', { value: 99.5, writable: true });
        Object.defineProperty(video, 'duration', { value: 100, writable: true });
        
        document.dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowRight', bubbles: true }));
        expect(playSiblingSpy).toHaveBeenCalledWith(1, true);
    });
});
