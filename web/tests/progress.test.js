import { describe, it, expect, beforeEach, vi } from 'vitest';
import { setupTestEnvironment } from './test-helper';

describe('Progress Resuming', () => {
    beforeEach(async () => {
        await setupTestEnvironment();
    });

    it('resumes from item.playhead when read-only is FALSE and local storage is empty', async () => {
        const item = {
            path: 'video1.mp4',
            type: 'video/mp4',
            playhead: 42,
            duration: 100
        };

        window.disco.state.readOnly = false;
        localStorage.clear();

        // Trigger openInPiP
        await window.disco.openInPiP(item);

        const video = document.querySelector('video');
        expect(video.currentTime).toBe(42);
    });

    it('resumes from item.playhead when read-only is TRUE and local storage is empty', async () => {
        const item = {
            path: 'video1.mp4',
            type: 'video/mp4',
            playhead: 42,
            duration: 100
        };

        window.disco.state.readOnly = true;
        localStorage.clear();

        // Trigger openInPiP
        await window.disco.openInPiP(item);

        const video = document.querySelector('video');
        // Should use item.playhead when local storage is empty (works in both read-only and normal mode)
        expect(video.currentTime).toBe(42);
    });

    it('prefers local storage over item.playhead', async () => {
        const item = {
            path: 'video1.mp4',
            type: 'video/mp4',
            playhead: 42,
            duration: 100
        };

        window.disco.state.readOnly = false;
        localStorage.setItem('disco-progress', JSON.stringify({
            'video1.mp4': { pos: 60, last: Date.now() }
        }));

        await window.disco.openInPiP(item);

        const video = document.querySelector('video');
        expect(video.currentTime).toBe(60);
    });

    it('resumes audio from item.playhead when read-only is TRUE', async () => {
        const item = {
            path: 'audio1.mp3',
            type: 'audio/mpeg',
            playhead: 30,
            duration: 300
        };

        window.disco.state.readOnly = true;
        localStorage.clear();

        await window.disco.openInPiP(item);

        const audio = document.querySelector('audio');
        expect(audio.currentTime).toBe(30);
    });

    it('expires local audio progress after 15 minutes if duration < 7 mins', async () => {
        const item = {
            path: 'audio1.mp3',
            type: 'audio/mpeg',
            duration: 300 // 5 mins
        };

        const oldTime = Date.now() - (20 * 60 * 1000); // 20 mins ago
        localStorage.setItem('disco-progress', JSON.stringify({
            'audio1.mp3': { pos: 40, last: oldTime }
        }));

        await window.disco.openInPiP(item);

        const audio = document.querySelector('audio');
        expect(audio.currentTime).toBe(0);
    });

    it('does NOT expire local audio progress if duration > 7 mins', async () => {
        const item = {
            path: 'long-audio.mp3',
            type: 'audio/mpeg',
            duration: 600 // 10 mins
        };

        const oldTime = Date.now() - (20 * 60 * 1000); // 20 mins ago
        localStorage.setItem('disco-progress', JSON.stringify({
            'long-audio.mp3': { pos: 40, last: oldTime }
        }));

        await window.disco.openInPiP(item);

        const audio = document.querySelector('audio');
        expect(audio.currentTime).toBe(40);
    });

    it('skips server sync if sessionTime < 90s and not complete', async () => {
        const item = {
            path: 'video1.mp4',
            type: 'video/mp4',
            duration: 600
        };

        // Mock startTime to be just now
        window.disco.state.playback.startTime = Date.now();
        window.disco.state.playback.lastUpdate = 0;

        global.fetch.mockClear();
        await window.disco.updateProgress(item, 45, 600, false);

        expect(global.fetch).not.toHaveBeenCalledWith(
            '/api/progress',
            expect.any(Object)
        );
    });

    it('syncs to server if sessionTime > 90s', async () => {
        const item = {
            path: 'video1.mp4',
            type: 'video/mp4',
            duration: 600
        };

        // Mock startTime to be 100s ago
        window.disco.state.playback.startTime = Date.now() - 100000;
        window.disco.state.playback.lastUpdate = 0;

        global.fetch.mockClear();
        await window.disco.updateProgress(item, 100, 600, false);

        expect(global.fetch).toHaveBeenCalledWith(
            '/api/progress',
            expect.objectContaining({
                method: 'POST',
                body: expect.stringContaining('"playhead":100')
            })
        );
    });

    it('merges local progress into search results', async () => {
        localStorage.setItem('disco-progress', JSON.stringify({
            'video1.mp4': { pos: 45, last: Date.now() }
        }));

        const searchInput = document.getElementById('search-input');
        searchInput.value = 'video1.mp4';
        searchInput.dispatchEvent(new KeyboardEvent('keypress', { key: 'Enter', bubbles: true }));

        await vi.waitFor(() => {
            const card = document.querySelector('[data-path="video1.mp4"]');
            expect(card).not.toBeNull();
            // The progress bar should be visible and represent 45/60 (from mocks.json duration)
            const progressBar = card.querySelector('.progress-bar');
            expect(progressBar).not.toBeNull();
        });
    });

    it('sums local and server play counts in read-only mode', () => {
        const item = {
            path: 'video1.mp4',
            play_count: 5
        };

        window.disco.state.readOnly = true;
        localStorage.setItem('disco-play-counts', JSON.stringify({
            'video1.mp4': 2
        }));

        const count = window.disco.getPlayCount(item);
        expect(count).toBe(7);
    });

    it('marks media as played locally in read-only mode', async () => {
        const item = {
            path: 'video1.mp4',
            play_count: 5,
            playhead: 100
        };

        window.disco.state.readOnly = true;
        localStorage.clear();

        await window.disco.markMediaPlayed(item);

        const progress = JSON.parse(localStorage.getItem('disco-progress'));
        expect(progress['video1.mp4'].pos).toBe(0);

        const counts = JSON.parse(localStorage.getItem('disco-play-counts'));
        expect(counts['video1.mp4']).toBe(1);

        // Verify it sums correctly now
        const count = window.disco.getPlayCount(item);
        expect(count).toBe(6); // 5 server + 1 local
    });

    it('syncs to server if isComplete is true even if sessionTime < 90s', async () => {
        const item = {
            path: 'video1.mp4',
            type: 'video/mp4',
            duration: 600
        };

        // Mock startTime to be just now
        window.disco.state.playback.startTime = Date.now();
        window.disco.state.playback.lastUpdate = 0;

        global.fetch.mockClear();
        await window.disco.updateProgress(item, 600, 600, true);

        expect(global.fetch).toHaveBeenCalledWith(
            '/api/progress',
            expect.objectContaining({
                method: 'POST',
                body: expect.stringContaining('"completed":true')
            })
        );
    });

    it('throttles local storage updates to once per second', async () => {
        const item = { path: 'video1.mp4', type: 'video/mp4' };
        localStorage.clear();
        
        // Use fake timers to control time
        vi.useFakeTimers();
        
        await window.disco.updateProgress(item, 10, 100);
        let progress = JSON.parse(localStorage.getItem('disco-progress'));
        expect(progress['video1.mp4'].pos).toBe(10);
        
        // Call again immediately with new pos
        await window.disco.updateProgress(item, 11, 100);
        progress = JSON.parse(localStorage.getItem('disco-progress'));
        expect(progress['video1.mp4'].pos).toBe(10); // Should still be 10 due to throttle
        
        // Advance time by 1.1s
        vi.advanceTimersByTime(1100);
        
        await window.disco.updateProgress(item, 12, 100);
        progress = JSON.parse(localStorage.getItem('disco-progress'));
        expect(progress['video1.mp4'].pos).toBe(12); // Should now be updated to 12
        
        vi.useRealTimers();
    });

    it('syncs to server at 30s intervals after initial 90s', async () => {
        const item = { path: 'video1.mp4', type: 'video/mp4', duration: 1000 };
        
        // Mock startTime to be 100s ago
        window.disco.state.playback.startTime = Date.now() - 100000;
        window.disco.state.playback.lastUpdate = Date.now();
        
        global.fetch.mockClear();
        
        // Advance 20s (less than 30s interval)
        vi.useFakeTimers();
        vi.advanceTimersByTime(20000);
        
        await window.disco.updateProgress(item, 120, 1000);
        expect(global.fetch).not.toHaveBeenCalled();
        
        // Advance another 15s (total 35s)
        vi.advanceTimersByTime(15000);
        
        await window.disco.updateProgress(item, 135, 1000);
        expect(global.fetch).toHaveBeenCalled();
        
        vi.useRealTimers();
    });
});
