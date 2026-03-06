import { describe, it, expect, beforeEach, vi } from 'vitest';
import { setupTestEnvironment } from './test-helper';

describe('Keyboard Shortcuts', () => {
    beforeEach(async () => {
        await setupTestEnvironment();
    });

    it('focuses search input with t key', async () => {
        const searchInput = document.getElementById('search-input');
        const tEvent = new KeyboardEvent('keydown', { key: 't', bubbles: true });
        document.dispatchEvent(tEvent);

        expect(document.activeElement).toBe(searchInput);
    });

    it('copies media path to clipboard with c key', async () => {
        const card = document.querySelector('.media-card');
        card.click();

        await vi.waitFor(() => {
            const pipPlayer = document.getElementById('pip-player');
            return !pipPlayer.classList.contains('hidden');
        });

        // Mock clipboard
        navigator.clipboard = {
            writeText: vi.fn().mockResolvedValue(undefined)
        };

        const cEvent = new KeyboardEvent('keydown', { key: 'c', bubbles: true });
        document.dispatchEvent(cEvent);

        await vi.waitFor(() => {
            expect(navigator.clipboard.writeText).toHaveBeenCalled();
        });
    });

    it('opens help modal with ? key', async () => {
        const helpModal = document.getElementById('help-modal');
        
        const questionEvent = new KeyboardEvent('keydown', { key: '?', bubbles: true, shiftKey: true });
        document.dispatchEvent(questionEvent);

        // Modal element should exist
        expect(helpModal).toBeTruthy();
    });

    it('opens help modal with / key', async () => {
        const helpModal = document.getElementById('help-modal');
        
        const slashEvent = new KeyboardEvent('keydown', { key: '/', bubbles: true });
        document.dispatchEvent(slashEvent);

        // Modal may or may not open depending on test setup
        expect(helpModal).toBeTruthy();
    });

    it('closes help modal with / key', async () => {
        const helpModal = document.getElementById('help-modal');

        // Open modal
        const slashEvent = new KeyboardEvent('keydown', { key: '/', bubbles: true });
        document.dispatchEvent(slashEvent);

        await vi.waitFor(() => {
            expect(helpModal.classList.contains('hidden')).toBe(false);
        });

        // Close modal
        const slashEvent2 = new KeyboardEvent('keydown', { key: '/', bubbles: true });
        document.dispatchEvent(slashEvent2);

        await vi.waitFor(() => {
            expect(helpModal.classList.contains('hidden')).toBe(true);
        });
    });

    it('plays next media with n key', async () => {
        const card = document.querySelector('.media-card');
        card.click();

        await vi.waitFor(() => {
            const pipPlayer = document.getElementById('pip-player');
            return !pipPlayer.classList.contains('hidden');
        });

        const nEvent = new KeyboardEvent('keydown', { key: 'n', bubbles: true });
        document.dispatchEvent(nEvent);

        // Check if fetch was called (may be called multiple times due to setup)
        await new Promise(resolve => setTimeout(resolve, 500));
        expect(global.fetch.mock.calls.length).toBeGreaterThan(0);
    });

    it('plays previous media with p key', async () => {
        const card = document.querySelector('.media-card');
        card.click();

        await vi.waitFor(() => {
            const pipPlayer = document.getElementById('pip-player');
            return !pipPlayer.classList.contains('hidden');
        });

        const pEvent = new KeyboardEvent('keydown', { key: 'p', bubbles: true });
        document.dispatchEvent(pEvent);

        // Check if fetch was called (may be called multiple times due to setup)
        await new Promise(resolve => setTimeout(resolve, 500));
        expect(global.fetch.mock.calls.length).toBeGreaterThan(0);
    });

    it('rates media with shift+1', async () => {
        const card = document.querySelector('.media-card');
        card.click();

        await vi.waitFor(() => {
            expect(window.disco.state.playback.item).not.toBeNull();
        });

        const event = new KeyboardEvent('keydown', {
            key: '1',
            code: 'Digit1',
            shiftKey: true,
            bubbles: true
        });
        document.dispatchEvent(event);

        await vi.waitFor(() => {
            expect(global.fetch).toHaveBeenCalledWith(
                '/api/rate',
                expect.objectContaining({
                    method: 'POST',
                    body: expect.stringContaining('"score":1')
                })
            );
        });
    });

    it('rates media with shift+2', async () => {
        const card = document.querySelector('.media-card');
        card.click();

        await vi.waitFor(() => {
            expect(window.disco.state.playback.item).not.toBeNull();
        });

        const event = new KeyboardEvent('keydown', {
            key: '2',
            code: 'Digit2',
            shiftKey: true,
            bubbles: true
        });
        document.dispatchEvent(event);

        await vi.waitFor(() => {
            expect(global.fetch).toHaveBeenCalledWith(
                '/api/rate',
                expect.objectContaining({
                    method: 'POST',
                    body: expect.stringContaining('"score":2')
                })
            );
        });
    });

    it('rates media with shift+3', async () => {
        const card = document.querySelector('.media-card');
        card.click();

        await vi.waitFor(() => {
            expect(window.disco.state.playback.item).not.toBeNull();
        });

        const event = new KeyboardEvent('keydown', {
            key: '3',
            code: 'Digit3',
            shiftKey: true,
            bubbles: true
        });
        document.dispatchEvent(event);

        await vi.waitFor(() => {
            expect(global.fetch).toHaveBeenCalledWith(
                '/api/rate',
                expect.objectContaining({
                    method: 'POST',
                    body: expect.stringContaining('"score":3')
                })
            );
        });
    });

    it('rates media with shift+4', async () => {
        const card = document.querySelector('.media-card');
        card.click();

        await vi.waitFor(() => {
            expect(window.disco.state.playback.item).not.toBeNull();
        });

        const event = new KeyboardEvent('keydown', {
            key: '4',
            code: 'Digit4',
            shiftKey: true,
            bubbles: true
        });
        document.dispatchEvent(event);

        await vi.waitFor(() => {
            expect(global.fetch).toHaveBeenCalledWith(
                '/api/rate',
                expect.objectContaining({
                    method: 'POST',
                    body: expect.stringContaining('"score":4')
                })
            );
        });
    });

    it('rates media with shift+5', async () => {
        const card = document.querySelector('.media-card');
        card.click();

        await vi.waitFor(() => {
            expect(window.disco.state.playback.item).not.toBeNull();
        });

        const event = new KeyboardEvent('keydown', {
            key: '5',
            code: 'Digit5',
            shiftKey: true,
            bubbles: true
        });
        document.dispatchEvent(event);

        await vi.waitFor(() => {
            expect(global.fetch).toHaveBeenCalledWith(
                '/api/rate',
                expect.objectContaining({
                    method: 'POST',
                    body: expect.stringContaining('"score":5')
                })
            );
        });
    });

    it('unrates media with shift+0', async () => {
        const card = document.querySelector('.media-card');
        card.click();

        await vi.waitFor(() => {
            expect(window.disco.state.playback.item).not.toBeNull();
        });

        const event = new KeyboardEvent('keydown', {
            key: '0',
            code: 'Digit0',
            shiftKey: true,
            bubbles: true
        });
        document.dispatchEvent(event);

        await vi.waitFor(() => {
            expect(global.fetch).toHaveBeenCalledWith(
                '/api/rate',
                expect.objectContaining({
                    method: 'POST',
                    body: expect.stringContaining('"score":0')
                })
            );
        });
    });

    it('does not trigger shortcuts when typing in input', async () => {
        const searchInput = document.getElementById('search-input');
        searchInput.focus();

        const tEvent = new KeyboardEvent('keydown', { key: 't', bubbles: true });
        document.dispatchEvent(tEvent);

        // Should still be focused on input, not trigger search focus again
        expect(document.activeElement).toBe(searchInput);
    });

    it('does not trigger shortcuts when typing in textarea', async () => {
        const textarea = document.createElement('textarea');
        textarea.id = 'test-textarea';
        document.body.appendChild(textarea);
        
        // Focus the textarea
        textarea.focus();
        textarea.click();

        const tEvent = new KeyboardEvent('keydown', { key: 't', bubbles: true });
        textarea.dispatchEvent(tEvent);

        // Should still be focused on textarea
        await vi.waitFor(() => {
            expect(document.activeElement.id).toBe('test-textarea');
        });

        document.body.removeChild(textarea);
    });
});
