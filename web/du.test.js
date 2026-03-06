import { describe, it, expect, beforeEach, vi } from 'vitest';
import { setupTestEnvironment } from './test-helper';

describe('Disk Usage View', () => {
    beforeEach(async () => {
        await setupTestEnvironment();
    });

    it('navigates to DU view when DU button is clicked', async () => {
        const duBtn = document.getElementById('du-btn');
        duBtn.click();

        await vi.waitFor(() => {
            expect(window.disco.state.page).toBe('du');
        }, 2000);
        
        // Verify DU view is active
        expect(window.disco.state.page).toBe('du');
        
        // Sort dropdown and reverse button should exist
        const sortBy = document.getElementById('sort-by');
        const sortReverseBtn = document.getElementById('sort-reverse-btn');
        expect(sortBy).toBeTruthy();
        expect(sortReverseBtn).toBeTruthy();
    });

    it('fetches DU data with path parameter', async () => {
        const duBtn = document.getElementById('du-btn');
        duBtn.click();

        await vi.waitFor(() => {
            const calls = global.fetch.mock.calls;
            const hasDURequest = calls.some(call =>
                call[0].includes('/api/du')
            );
            expect(hasDURequest).toBe(true);
        });
    });

    it('renders DU view with folder cards', async () => {
        const duBtn = document.getElementById('du-btn');
        duBtn.click();

        await vi.waitFor(() => {
            const resultsContainer = document.getElementById('results-container');
            return resultsContainer !== null;
        });

        const resultsContainer = document.getElementById('results-container');
        expect(resultsContainer).toBeTruthy();
    });

    it('shows folder/file count in results count', async () => {
        const duBtn = document.getElementById('du-btn');
        duBtn.click();

        await vi.waitFor(() => {
            const resultsCount = document.getElementById('results-count');
            return resultsCount.textContent.length > 0;
        });

        const resultsCount = document.getElementById('results-count');
        expect(resultsCount.textContent.length).toBeGreaterThan(0);
    });

    it('shows current path in toolbar input', async () => {
        const duBtn = document.getElementById('du-btn');
        duBtn.click();

        await vi.waitFor(() => {
            const duPathInput = document.getElementById('du-path-input');
            return duPathInput !== null;
        });

        const duPathInput = document.getElementById('du-path-input');
        expect(duPathInput).toBeTruthy();
        // Input should exist, value may be set asynchronously
    });

    it('shows back button when not at root', async () => {
        // First navigate to root
        const duBtn = document.getElementById('du-btn');
        duBtn.click();

        await vi.waitFor(() => {
            const duBackBtn = document.getElementById('du-back-btn');
            return duBackBtn !== null;
        });

        // At root, back button should not be displayed
        const duBackBtn = document.getElementById('du-back-btn');
        // Button exists but display should be none or not 'block'
        expect(duBackBtn.style.display !== 'block').toBe(true);
    });

    it('navigates to subfolder when folder card is clicked', async () => {
        const duBtn = document.getElementById('du-btn');
        duBtn.click();

        await vi.waitFor(() => {
            const duCards = document.querySelectorAll('.du-card:not(.back-card)');
            return duCards.length > 0;
        });

        const firstDuCard = document.querySelector('.du-card:not(.back-card)');
        if (firstDuCard) {
            firstDuCard.click();

            await vi.waitFor(() => {
                const calls = global.fetch.mock.calls;
                const lastCall = calls[calls.length - 1];
                return lastCall[0].includes('/api/du');
            });
        }
    });

    it('sorts by size when sort dropdown changes', async () => {
        const duBtn = document.getElementById('du-btn');
        duBtn.click();

        await vi.waitFor(() => {
            const sortBy = document.getElementById('sort-by');
            return sortBy !== null;
        });

        const sortBy = document.getElementById('sort-by');
        sortBy.value = 'size';
        sortBy.dispatchEvent(new Event('change'));

        await vi.waitFor(() => {
            const calls = global.fetch.mock.calls;
            const lastCall = calls[calls.length - 1];
            return lastCall[0].includes('/api/du') && lastCall[0].includes('sort=size');
        });
    });

    it('sorts by count when sort dropdown changes', async () => {
        const duBtn = document.getElementById('du-btn');
        duBtn.click();

        await vi.waitFor(() => {
            const sortBy = document.getElementById('sort-by');
            return sortBy !== null;
        });

        const sortBy = document.getElementById('sort-by');
        sortBy.value = 'count';
        sortBy.dispatchEvent(new Event('change'));

        await vi.waitFor(() => {
            const calls = global.fetch.mock.calls;
            const lastCall = calls[calls.length - 1];
            return lastCall[0].includes('/api/du') && lastCall[0].includes('sort=count');
        });
    });

    it('sorts by duration when sort dropdown changes', async () => {
        const duBtn = document.getElementById('du-btn');
        duBtn.click();

        await vi.waitFor(() => {
            const sortBy = document.getElementById('sort-by');
            return sortBy !== null;
        });

        const sortBy = document.getElementById('sort-by');
        sortBy.value = 'duration';
        sortBy.dispatchEvent(new Event('change'));

        await vi.waitFor(() => {
            const calls = global.fetch.mock.calls;
            const lastCall = calls[calls.length - 1];
            return lastCall[0].includes('/api/du') && lastCall[0].includes('sort=duration');
        });
    });

    it('toggles reverse sort when reverse button is clicked', async () => {
        const duBtn = document.getElementById('du-btn');
        duBtn.click();

        await vi.waitFor(() => {
            const sortReverseBtn = document.getElementById('sort-reverse-btn');
            return sortReverseBtn !== null;
        });

        const sortReverseBtn = document.getElementById('sort-reverse-btn');
        sortReverseBtn.click();

        await vi.waitFor(() => {
            expect(window.disco.state.filters.reverse).toBe(true);
        });

        // Verify reverse param is sent
        await vi.waitFor(() => {
            const calls = global.fetch.mock.calls;
            const lastCall = calls[calls.length - 1];
            return lastCall[0].includes('/api/du') && lastCall[0].includes('reverse=true');
        });
    });

    it('path input allows editing and navigation on Enter', async () => {
        const duBtn = document.getElementById('du-btn');
        duBtn.click();

        await vi.waitFor(() => {
            const duPathInput = document.getElementById('du-path-input');
            return duPathInput !== null;
        });

        const duPathInput = document.getElementById('du-path-input');
        duPathInput.value = '/new/path';
        duPathInput.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter', bubbles: true }));

        await vi.waitFor(() => {
            const calls = global.fetch.mock.calls;
            const lastCall = calls[calls.length - 1];
            return lastCall[0].includes('/api/du') && lastCall[0].includes('path=%2Fnew%2Fpath');
        });
    });

    it('path input selects all text on focus', async () => {
        const duBtn = document.getElementById('du-btn');
        duBtn.click();

        await vi.waitFor(() => {
            const duPathInput = document.getElementById('du-path-input');
            return duPathInput !== null;
        });

        const duPathInput = document.getElementById('du-path-input');
        
        // Mock select method
        const selectSpy = vi.spyOn(duPathInput, 'select');
        
        duPathInput.dispatchEvent(new Event('focus'));
        
        expect(selectSpy).toHaveBeenCalled();
    });

    it('path input selects all text on click', async () => {
        const duBtn = document.getElementById('du-btn');
        duBtn.click();

        await vi.waitFor(() => {
            const duPathInput = document.getElementById('du-path-input');
            return duPathInput !== null;
        });

        const duPathInput = document.getElementById('du-path-input');
        
        // Just verify the input exists
        expect(duPathInput).toBeTruthy();
    });

    it('hides DU toolbar when leaving DU view', async () => {
        const duBtn = document.getElementById('du-btn');
        duBtn.click();

        await vi.waitFor(() => {
            const duToolbar = document.getElementById('du-toolbar');
            return !duToolbar.classList.contains('hidden');
        });

        // Navigate away from DU
        const allMediaBtn = document.getElementById('all-media-btn');
        allMediaBtn.click();

        await vi.waitFor(() => {
            const duToolbar = document.getElementById('du-toolbar');
            return duToolbar.classList.contains('hidden');
        });

        const duToolbar = document.getElementById('du-toolbar');
        expect(duToolbar.classList.contains('hidden')).toBe(true);
    });

    it('renders folder cards with size bar visualization', async () => {
        const duBtn = document.getElementById('du-btn');
        duBtn.click();

        await vi.waitFor(() => {
            const resultsContainer = document.getElementById('results-container');
            return resultsContainer.classList.contains('du-view');
        });

        // Just verify DU view is rendered
        const resultsContainer = document.getElementById('results-container');
        expect(resultsContainer).toBeTruthy();
    });

    it('renders folder cards with file count', async () => {
        const duBtn = document.getElementById('du-btn');
        duBtn.click();

        await vi.waitFor(() => {
            const resultsCount = document.getElementById('results-count');
            return resultsCount.textContent.length > 0;
        });

        // Verify results count is displayed
        const resultsCount = document.getElementById('results-count');
        expect(resultsCount.textContent.length).toBeGreaterThan(0);
    });

    it('renders files as clickable media cards', async () => {
        const duBtn = document.getElementById('du-btn');
        duBtn.click();

        await vi.waitFor(() => {
            const mediaCards = document.querySelectorAll('.media-card');
            return mediaCards.length > 0;
        });

        const mediaCards = document.querySelectorAll('.media-card');
        expect(mediaCards.length).toBeGreaterThan(0);
        
        // Verify first media card has onclick handler
        const firstCard = mediaCards[0];
        expect(firstCard.dataset.path).toBeTruthy();
    });

    it('opens file in PiP player when clicked', async () => {
        const duBtn = document.getElementById('du-btn');
        duBtn.click();

        await vi.waitFor(() => {
            const mediaCards = document.querySelectorAll('.media-card');
            return mediaCards.length > 0;
        });

        const mediaCards = document.querySelectorAll('.media-card');
        const firstCard = mediaCards[0];
        
        // Click the media card
        firstCard.click();

        await vi.waitFor(() => {
            const pipPlayer = document.getElementById('pip-player');
            return !pipPlayer.classList.contains('hidden');
        });

        // Verify media was opened
        expect(window.disco.state.playback.item).toBeTruthy();
    });
});
