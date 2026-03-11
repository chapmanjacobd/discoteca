import { describe, it, expect, beforeEach, vi } from 'vitest';
import { setupTestEnvironment } from './test-helper';
import { state } from '../state';

describe('Queue Management', () => {
    let state;
    beforeEach(async () => {
        // Reset localStorage and setup environment
        localStorage.clear();
        await setupTestEnvironment();
        state = window.disco.state;
    });

    it('should show queue container when enabled', async () => {
        const queueContainer = document.getElementById('queue-container');
        const settingEnableQueue = document.getElementById('setting-enable-queue');
        
        // Initially hidden (default state)
        expect(queueContainer.classList.contains('hidden')).toBe(true);
        
        // Enable queue
        settingEnableQueue.checked = true;
        settingEnableQueue.dispatchEvent(new Event('change'));
        
        expect(state.enableQueue).toBe(true);
        expect(queueContainer.classList.contains('hidden')).toBe(false);
    });

    it('should add items to queue when enabled', async () => {
        // Enable queue
        state.enableQueue = true;
        const updateQueueVisibility = window.disco.updateQueueVisibility;
        updateQueueVisibility();

        const mediaCard = document.querySelector('.media-card');
        expect(mediaCard).toBeTruthy();

        // Click media card
        mediaCard.click();

        expect(state.playback.queue.length).toBe(1);
        expect(state.playback.queue[0].path).toBe('video1.mp4');

        const queueItems = document.querySelectorAll('.queue-item');
        expect(queueItems.length).toBe(1);
        expect(queueItems[0].querySelector('.queue-item-title').textContent).toBe('video1.mp4');
    });

    it('should toggle expansion when expand button clicked', async () => {
        // Enable queue
        state.enableQueue = true;
        window.disco.updateQueueVisibility();

        const queueContainer = document.getElementById('queue-container');
        const queueList = document.getElementById('queue-list');
        const expandBtn = document.getElementById('queue-expand-btn');
        
        expect(expandBtn).toBeTruthy();
        expect(state.queueExpanded).toBe(false);
        expect(queueList.classList.contains('expanded')).toBe(false);

        // Click expand
        expandBtn.click();

        expect(state.queueExpanded).toBe(true);
        expect(queueList.classList.contains('expanded')).toBe(true);
        expect(localStorage.getItem('disco-queue-expanded')).toBe('true');

        // Click again to collapse
        expandBtn.click();

        expect(state.queueExpanded).toBe(false);
        expect(queueList.classList.contains('expanded')).toBe(false);
        expect(localStorage.getItem('disco-queue-expanded')).toBe('false');
    });

    it('should add item to queue via drag and drop', async () => {
        state.enableQueue = true;
        window.disco.updateQueueVisibility();

        const queueContainer = document.getElementById('queue-container');
        const mediaCard = document.querySelector('.media-card');
        const item = mediaCard._item;

        // Simulate dragstart
        const dragStartEvent = new DragEvent('dragstart');
        mediaCard.dispatchEvent(dragStartEvent);
        state.draggedItem = item;

        // Simulate drop on queue container
        const dropEvent = new DragEvent('drop', {
            dataTransfer: {
                getData: () => item.path
            }
        });
        queueContainer.dispatchEvent(dropEvent);

        expect(state.playback.queue.length).toBe(1);
        expect(state.playback.queue[0].path).toBe(item.path);
    });

    it('should clear queue when clear button clicked', async () => {
        // Mock confirm
        global.confirm = vi.fn(() => true);

        state.enableQueue = true;
        state.playback.queue = [{ path: 'test.mp4' }];
        window.disco.updateQueueVisibility();

        const clearBtn = document.getElementById('queue-clear-btn');
        expect(document.querySelectorAll('.queue-item').length).toBe(1);

        clearBtn.click();

        expect(global.confirm).toHaveBeenCalled();
        expect(state.playback.queue.length).toBe(0);
        expect(document.querySelectorAll('.queue-item').length).toBe(0);
    });

    it('should persist queue to localStorage', async () => {
        state.enableQueue = true;
        state.playback.queue = [{ path: 'persist.mp4', duration: 10 }];
        
        // Trigger save
        window.disco.renderQueue();
        
        const saved = JSON.parse(localStorage.getItem('disco-queue'));
        expect(saved).toBeTruthy();
        expect(saved.length).toBe(1);
        expect(saved[0].path).toBe('persist.mp4');
    });

    it('should advance correctly through duplicate items in queue', async () => {
        state.enableQueue = true;
        const item = { path: 'duplicate.mp4', duration: 10, type: 'video/mp4' };
        state.playback.queue = [item, item, { path: 'third.mp4', duration: 10, type: 'video/mp4' }];
        
        // Play first item (explicitly passing index 0)
        window.disco.openActivePlayer(state.playback.queue[0], true, false, 0);
        expect(state.playback.item.path).toBe('duplicate.mp4');
        expect(state.playback.queueIndex).toBe(0);
        
        // Advance to next (should be the second 'duplicate.mp4' at index 1)
        window.disco.playSibling(1);
        expect(state.playback.item.path).toBe('duplicate.mp4');
        expect(state.playback.queueIndex).toBe(1);

        // Advance again
        window.disco.playSibling(1);
        expect(state.playback.item.path).toBe('third.mp4');
        expect(state.playback.queueIndex).toBe(2);
    });
});
