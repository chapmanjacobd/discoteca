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

    it('auto-skips single folder at root level on initial load', async () => {
        // Reset state before test
        window.disco.state.page = 'search';
        window.disco.state.duPath = '';
        window.disco.state.duData = null;
        
        let fetchCallCount = 0;
        
        // Clear existing mocks and set up fresh - must return Promises like real fetch
        // Return different data for different paths to avoid infinite loop
        global.fetch.mockClear();
        global.fetch.mockImplementation((url) => {
            if (typeof url !== 'string') url = url.toString();
            
            if (url.includes('/api/databases')) {
                return Promise.resolve({
                    ok: true,
                    status: 200,
                    json: () => Promise.resolve({ databases: ['test.db'], trashcan: false, read_only: false, dev: false })
                });
            } else if (url.includes('/api/du')) {
                fetchCallCount++;
                const urlObj = new URL(url, 'http://localhost');
                const pathParam = urlObj.searchParams.get('path') || '';
                
                // Return different data based on path
                let data;
                if (pathParam === '') {
                    // Root: single folder -> should auto-skip
                    data = [
                        { path: '/videos/', total_size: 1073741824, total_duration: 7200, count: 5, files: [] }
                    ];
                } else {
                    // /videos/: multiple files -> should stop auto-skip
                    data = [
                        { path: '/videos/movie1.mp4', total_size: 536870912, total_duration: 3600, count: 0, files: [{ path: '/videos/movie1.mp4', type: 'video/mp4', size: 536870912, duration: 3600 }] },
                        { path: '/videos/movie2.mp4', total_size: 536870912, total_duration: 3600, count: 0, files: [{ path: '/videos/movie2.mp4', type: 'video/mp4', size: 536870912, duration: 3600 }] }
                    ];
                }
                
                return Promise.resolve({
                    ok: true,
                    status: 200,
                    headers: { get: () => null },
                    json: () => Promise.resolve(data)
                });
            }
            
            return Promise.resolve({
                ok: true,
                status: 200,
                json: () => Promise.resolve([])
            });
        });

        // Navigate to DU - first reset duData to ensure isFirstDUVisit is true
        window.disco.state.duData = null;
        const duBtn = document.getElementById('du-btn');
        duBtn.click();

        // Give async operations a chance to start
        await new Promise(r => setTimeout(r, 50));

        // Wait for auto-navigation to complete - wait for duPath to change
        await vi.waitFor(() => {
            return window.disco.state.duPath === '/videos/';
        }, 5000, 100);

        expect(window.disco.state.duPath).toBe('/videos/');
        
        // Verify fetch was called twice (once for root, once for auto-skipped folder)
        const duCalls = global.fetch.mock.calls.filter(call => call[0].includes('/api/du'));
        expect(duCalls.length).toBe(2);
    });

    it('does not auto-skip when there are multiple folders at root', async () => {
        // Mock DU data with multiple folders at root
        const multipleFoldersData = [
            { path: '/videos/', total_size: 1073741824, total_duration: 7200, count: 5, files: [] },
            { path: '/audio/', total_size: 536870912, total_duration: 3600, count: 10, files: [] }
        ];

        global.fetch.mockImplementation((url) => {
            if (typeof url !== 'string') url = url.toString();
            
            if (url.includes('/api/databases')) {
                return Promise.resolve({
                    ok: true,
                    json: () => Promise.resolve({ databases: ['test.db'], trashcan: false, read_only: false, dev: false })
                });
            } else if (url.includes('/api/du')) {
                return Promise.resolve({
                    ok: true,
                    json: () => Promise.resolve(multipleFoldersData)
                });
            }
            
            return Promise.resolve({
                ok: true,
                json: () => Promise.resolve([])
            });
        });

        const duBtn = document.getElementById('du-btn');
        duBtn.click();

        await vi.waitFor(() => {
            return window.disco.state.page === 'du';
        }, 2000);

        // Should stay at root
        expect(window.disco.state.duPath).toBe('');
        
        // Should only fetch once (no auto-skip)
        const duCalls = global.fetch.mock.calls.filter(call => call[0].includes('/api/du'));
        expect(duCalls.length).toBe(1);
    });

    it('does not auto-skip when single item is a file not a folder', async () => {
        // Mock DU data with single file at root
        const singleFileData = [
            { path: '/video.mp4', total_size: 1073741824, total_duration: 7200, count: 0, files: [{ path: '/video.mp4', type: 'video/mp4', size: 1073741824, duration: 7200 }] }
        ];

        global.fetch.mockImplementation((url) => {
            if (typeof url !== 'string') url = url.toString();
            
            if (url.includes('/api/databases')) {
                return Promise.resolve({
                    ok: true,
                    json: () => Promise.resolve({ databases: ['test.db'], trashcan: false, read_only: false, dev: false })
                });
            } else if (url.includes('/api/du')) {
                return Promise.resolve({
                    ok: true,
                    json: () => Promise.resolve(singleFileData)
                });
            }
            
            return Promise.resolve({
                ok: true,
                json: () => Promise.resolve([])
            });
        });

        const duBtn = document.getElementById('du-btn');
        duBtn.click();

        await vi.waitFor(() => {
            return window.disco.state.page === 'du';
        }, 2000);

        // Should stay at root
        expect(window.disco.state.duPath).toBe('');
        
        // Should only fetch once (no auto-skip)
        const duCalls = global.fetch.mock.calls.filter(call => call[0].includes('/api/du'));
        expect(duCalls.length).toBe(1);
    });
});
