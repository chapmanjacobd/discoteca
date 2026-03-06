import { describe, it, expect, beforeEach, vi } from 'vitest';
import { setupTestEnvironment } from './test-helper';

describe('Trash Filter Behavior', () => {
    beforeEach(async () => {
        await setupTestEnvironment();
    });

    it('preserves Trash page when clicking history filters', async () => {
        const trashBtn = document.getElementById('trash-btn');
        const inProgressBtn = document.getElementById('history-in-progress-btn');
        const unplayedBtn = document.getElementById('history-unplayed-btn');

        // Go to Trash
        trashBtn.click();
        expect(window.disco.state.page).toBe('trash');

        // Click In Progress
        inProgressBtn.click();
        expect(window.disco.state.page).toBe('trash'); // Should NOT switch to 'search'
        expect(window.disco.state.filters.unfinished).toBe(true);

        // Click Unplayed (toggles unfinished off, unplayed on)
        unplayedBtn.click();
        expect(window.disco.state.page).toBe('trash'); // Should still be 'trash'
        expect(window.disco.state.filters.unplayed).toBe(true);
        expect(window.disco.state.filters.unfinished).toBe(false);
    });

    it('resets currentPage to 1 when filters change', async () => {
        const inProgressBtn = document.getElementById('history-in-progress-btn');
        
        // Manually set page to 2
        window.disco.state.currentPage = 2;
        
        // Click In Progress
        inProgressBtn.click();
        
        // Should reset to page 1
        expect(window.disco.state.currentPage).toBe(1);
    });

    it('preserves current page when history filters clicked outside Trash', async () => {
        const inProgressBtn = document.getElementById('history-in-progress-btn');

        // Start in 'search' page (default)
        expect(window.disco.state.page).toBe('search');

        // Click In Progress (Turn ON)
        inProgressBtn.click();
        expect(window.disco.state.filters.unfinished).toBe(true);

        // Turn OFF
        inProgressBtn.click();
        expect(window.disco.state.filters.unfinished).toBe(false);

        // Now Go to playlist page
        window.disco.state.page = 'playlist';

        // Click In Progress (Turn ON)
        inProgressBtn.click();

        // Should stay on playlist page (history filters work across all modes now)
        expect(window.disco.state.page).toBe('playlist');
        expect(window.disco.state.filters.unfinished).toBe(true);
    });
});
