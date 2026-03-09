import { describe, it, expect, beforeEach, vi } from 'vitest';
import { setupTestEnvironment } from './test-helper';

describe('Error Handling', () => {
    beforeEach(async () => {
        await setupTestEnvironment();
        vi.useFakeTimers();
    });

    afterEach(() => {
        vi.useRealTimers();
    });

    it('auto-skips to next item on media error if autoplay is enabled', async () => {
        // Use items from mocks.json or default mock in test-helper.js
        const item1 = { path: 'video1.mp4', type: 'video/mp4' };
        const item2 = { path: 'audio1.mp3', type: 'audio/mpeg' };
        
        window.disco.state.autoplay = true;
        // currentMedia is already populated by setupTestEnvironment via performSearch/fetchDatabases etc.
        // Actually setupTestEnvironment calls readUrl and fetchDatabases but might not call performSearch.
        // Let's ensure currentMedia is what we think it is.
        await window.disco.performSearch();

        await window.disco.openInPiP(item1);
        expect(window.disco.state.playback.item.path).toBe('video1.mp4');

        const video = document.querySelector('video');

        // Trigger error
        video.onerror();

        // Let the async handleMediaError run until it sets the timeout
        await vi.waitFor(() => {
            if (!window.disco.state.playback.skipTimeout) throw new Error('Timeout not set yet');
        });

        // Advance timers
        vi.advanceTimersByTime(1200);

        expect(window.disco.state.playback.item.path).toBe('audio1.mp3');
    });

    it('stops auto-skipping after 30 consecutive errors', async () => {
        window.disco.state.autoplay = true;
        window.disco.state.playback.consecutiveErrors = 30;
        
        const item = { path: 'v30.mp4', type: 'video/mp4' };
        window.disco.state.playback.item = item;

        // Simulate 31st error
        await window.disco.handleMediaError(item);
        
        expect(window.disco.state.playback.item).toBeNull();
        expect(window.disco.state.playback.consecutiveErrors).toBe(0);
    });

    it('resets consecutiveErrors counter when progress is made', async () => {
        const item = { path: 'v1.mp4', type: 'video/mp4' };
        window.disco.state.playback.consecutiveErrors = 50;

        await window.disco.updateProgress(item, 5, 100);
        expect(window.disco.state.playback.consecutiveErrors).toBe(0);
    });
});
