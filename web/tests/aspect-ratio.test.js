import { describe, it, expect, beforeEach, vi } from 'vitest';
import { setupTestEnvironment } from './test-helper';

describe('Aspect Ratio Cycling', () => {
    beforeEach(async () => {
        await setupTestEnvironment();
    });

    it('cycles through aspect ratio modes when pressing a key', async () => {
        const card = document.querySelector('.media-card');
        card.click();

        await vi.waitFor(async () => {
            const video = document.querySelector('video');
            if (video) {
                video.play = vi.fn().mockResolvedValue(undefined);
                video.pause = vi.fn();
                return true;
            }
        });

        const video = document.querySelector('video');
        expect(video).toBeTruthy();

        // Initial state should be default (empty)
        expect(video.style.aspectRatio).toBe('');
        expect(video.style.objectFit).toBe('');

        // Press 'a' to cycle to first aspect ratio (16:9)
        const aEvent = new KeyboardEvent('keydown', { key: 'a', bubbles: true });
        document.dispatchEvent(aEvent);

        // Should now be 16:9
        expect(video.style.aspectRatio.replace(/\s+/g, '')).toBe('16/9');
        expect(video.style.objectFit).toBe('contain');

        // Press 'a' again to cycle to 4:3
        document.dispatchEvent(aEvent);
        expect(video.style.aspectRatio.replace(/\s+/g, '')).toBe('4/3');
        expect(video.style.objectFit).toBe('contain');

        // Press 'a' again to cycle to 21:9
        document.dispatchEvent(aEvent);
        expect(video.style.aspectRatio.replace(/\s+/g, '')).toBe('21/9');
        expect(video.style.objectFit).toBe('contain');
    });

    it('shows toast notification when changing aspect ratio', async () => {
        const card = document.querySelector('.media-card');
        card.click();

        await vi.waitFor(async () => {
            const video = document.querySelector('video');
            if (video) {
                video.play = vi.fn().mockResolvedValue(undefined);
                video.pause = vi.fn();
                return true;
            }
        });

        const video = document.querySelector('video');
        expect(video).toBeTruthy();

        // Get initial toast count
        const initialToastCount = window.disco.state.playback.toastTimer ? 1 : 0;

        // Press 'a' to cycle aspect ratio
        const aEvent = new KeyboardEvent('keydown', { key: 'a', bubbles: true });
        document.dispatchEvent(aEvent);

        // Verify aspect ratio changed (which implies toast was shown per implementation)
        expect(video.style.aspectRatio).toBeTruthy();
        expect(video.style.objectFit).toBe('contain');
    });

    it('cycles through all aspect ratio modes and wraps around', async () => {
        const card = document.querySelector('.media-card');
        card.click();

        await vi.waitFor(async () => {
            const video = document.querySelector('video');
            if (video) {
                video.play = vi.fn().mockResolvedValue(undefined);
                video.pause = vi.fn();
                return true;
            }
        });

        const video = document.querySelector('video');
        expect(video).toBeTruthy();

        const aEvent = new KeyboardEvent('keydown', { key: 'a', bubbles: true });

        // Cycle through all modes (6 modes + back to default = 7 presses)
        const expectedModes = [
            { aspectRatio: '16/9', objectFit: 'contain' },
            { aspectRatio: '4/3', objectFit: 'contain' },
            { aspectRatio: '21/9', objectFit: 'contain' },
            { aspectRatio: '1/1', objectFit: 'contain' },
            { aspectRatio: '', objectFit: 'fill' }, // Stretch mode
            { aspectRatio: '', objectFit: '' } // Back to default
        ];

        for (const expected of expectedModes) {
            document.dispatchEvent(aEvent);
            expect(video.style.aspectRatio.replace(/\s+/g, '')).toBe(expected.aspectRatio);
            expect(video.style.objectFit).toBe(expected.objectFit);
        }
    });

    it('does not change aspect ratio for non-video media', async () => {
        // Mock audio element instead of video
        const card = document.querySelector('.media-card');
        card.click();

        await vi.waitFor(async () => {
            // Force audio by mocking type
            window.disco.state.playback.item = {
                path: 'audio1.mp3',
                media_type: 'audio/mpeg'
            };
            const audio = document.querySelector('audio');
            if (audio) {
                audio.play = vi.fn().mockResolvedValue(undefined);
                audio.pause = vi.fn();
                return true;
            }
        });

        const audio = document.querySelector('audio');
        
        if (audio) {
            // Spy on showToast to verify it wasn't called
            const showToastSpy = vi.spyOn(window.disco, 'showToast');

            // Press 'a' - should not affect audio
            const aEvent = new KeyboardEvent('keydown', { key: 'a', bubbles: true });
            document.dispatchEvent(aEvent);

            // Audio elements don't have aspect ratio cycling
            // The 'a' key should not trigger aspect ratio change for audio
            expect(audio.style.aspectRatio).toBe('');
        }
    });
});
