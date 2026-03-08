import { describe, it, expect, beforeEach, vi } from 'vitest';
import { setupTestEnvironment } from './test-helper';

describe('History Navigation (Mobile/Fullscreen)', () => {
    beforeEach(async () => {
        await setupTestEnvironment();

        // Mock pushState/replaceState to update window.location.hash
        vi.spyOn(window.history, 'pushState').mockImplementation((state, title, url) => {
            const hashIndex = url.indexOf('#');
            window.location.hash = hashIndex !== -1 ? url.substring(hashIndex) : '';
        });
        vi.spyOn(window.history, 'replaceState').mockImplementation((state, title, url) => {
            const hashIndex = url.indexOf('#');
            window.location.hash = hashIndex !== -1 ? url.substring(hashIndex) : '';
        });
    });

    it('updates URL with modal param on mobile when opening a modal', async () => {
        global.innerWidth = 375; // Mobile width
        
        window.disco.openModal('settings-modal');
        
        expect(window.location.hash).toContain('modal=settings-modal');
        expect(window.disco.state.activeModal).toBe('settings-modal');
    });

    it('does NOT update URL with modal param on desktop', async () => {
        global.innerWidth = 1024; // Desktop width
        
        window.disco.openModal('settings-modal');
        
        expect(window.location.hash).not.toContain('modal=settings-modal');
        expect(window.disco.state.activeModal).toBe('settings-modal');
    });

    it('closes modal when modal param is removed from URL (simulating back button)', async () => {
        global.innerWidth = 375;
        window.location.hash = '#modal=settings-modal';
        
        // Initial state
        window.disco.readUrl();
        expect(window.disco.state.activeModal).toBe('settings-modal');
        expect(document.getElementById('settings-modal').classList.contains('hidden')).toBe(false);
        
        // Simulating back button: hash changed to empty
        window.location.hash = '';
        window.disco.readUrl();
        
        expect(window.disco.state.activeModal).toBe(null);
        expect(document.getElementById('settings-modal').classList.contains('hidden')).toBe(true);
    });

    it('updates URL with playing param on mobile when opening player', async () => {
        global.innerWidth = 375;
        const testItem = { path: 'test-video.mp4', type: 'video/mp4' };
        
        window.disco.openActivePlayer(testItem);
        
        expect(window.location.hash).toContain('playing=' + encodeURIComponent('test-video.mp4'));
    });

    it('closes player when playing param is removed from URL', async () => {
        global.innerWidth = 375;
        window.location.hash = '#playing=test-video.mp4';
        
        window.disco.readUrl();
        expect(window.disco.state.playback.item.path).toBe('test-video.mp4');
        expect(document.getElementById('pip-player').classList.contains('hidden')).toBe(false);
        
        window.location.hash = '';
        window.disco.readUrl();
        
        expect(window.disco.state.playback.item).toBe(null);
        expect(document.getElementById('pip-player').classList.contains('hidden')).toBe(true);
    });

    it('handles mobile sidebar via URL', async () => {
        global.innerWidth = 375;
        
        window.disco.toggleMobileSidebar();
        expect(window.location.hash).toContain('modal=mobile-sidebar');
        expect(document.getElementById('sidebar').classList.contains('mobile-open')).toBe(true);
        
        window.location.hash = '';
        window.disco.readUrl();
        expect(document.getElementById('sidebar').classList.contains('mobile-open')).toBe(false);
    });

    it('uses URL parameters on desktop when in fullscreen', async () => {
        global.innerWidth = 1024; // Desktop
        
        // Mock fullscreen
        Object.defineProperty(document, 'fullscreenElement', {
            get: () => document.body,
            configurable: true
        });
        
        window.disco.openModal('settings-modal');
        expect(window.location.hash).toContain('modal=settings-modal');
        
        // Clean up mock
        Object.defineProperty(document, 'fullscreenElement', {
            get: () => null,
            configurable: true
        });
    });
});
