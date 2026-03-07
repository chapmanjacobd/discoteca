import { describe, it, expect, beforeEach, vi } from 'vitest';
import { setupTestEnvironment } from './test-helper';

describe('Document Fullscreen', () => {
    let currentFullscreenElement = null;

    beforeEach(async () => {
        // Mock Fullscreen API
        currentFullscreenElement = null;
        Object.defineProperty(document, 'fullscreenElement', {
            get: () => currentFullscreenElement,
            configurable: true
        });

        Element.prototype.requestFullscreen = vi.fn().mockImplementation(function() {
            currentFullscreenElement = this;
            document.dispatchEvent(new Event('fullscreenchange'));
            return Promise.resolve();
        });

        document.exitFullscreen = vi.fn().mockImplementation(() => {
            currentFullscreenElement = null;
            document.dispatchEvent(new Event('fullscreenchange'));
            return Promise.resolve();
        });

        await setupTestEnvironment();
    });

    it('toggles fullscreen when the button is clicked', async () => {
        const item = { path: 'test.txt', type: 'text/plain' };

        // 1. Open document viewer
        window.disco.openInDocumentViewer(item);

        const modal = document.getElementById('document-modal');
        expect(modal.classList.contains('hidden')).toBe(false);

        const fsBtn = document.getElementById('doc-fullscreen');
        expect(fsBtn).not.toBeNull();
        expect(fsBtn.classList.contains('hidden')).toBe(false);

        const container = document.getElementById('document-container');

        // 2. Click fullscreen button to enter
        fsBtn.click();
        expect(container.requestFullscreen).toHaveBeenCalled();
        expect(document.fullscreenElement).toBe(container);

        // 3. Verify title change if implemented (I only added title update in global listener)
        // Wait, I haven't added the global listener yet!

        // 4. Click fullscreen button to exit
        fsBtn.click();
        expect(document.exitFullscreen).toHaveBeenCalled();
        expect(document.fullscreenElement).toBeNull();
    });

    it('toggles fullscreen when "f" key is pressed', async () => {
        const item = { path: 'test.txt', type: 'text/plain' };
        window.disco.openInDocumentViewer(item);

        const modal = document.getElementById('document-modal');
        const container = document.getElementById('document-container');

        // Press 'f'
        const event = new KeyboardEvent('keydown', { key: 'f', bubbles: true });
        document.dispatchEvent(event);

        expect(container.requestFullscreen).toHaveBeenCalled();
        expect(document.fullscreenElement).toBe(container);

        // Press 'f' again
        document.dispatchEvent(event);
        expect(document.exitFullscreen).toHaveBeenCalled();
        expect(document.fullscreenElement).toBeNull();
    });
});
