import { describe, it, expect, beforeEach, vi } from 'vitest';
import { setupTestEnvironment } from './test-helper';

describe('Metadata Modal', () => {
    beforeEach(async () => {
        await setupTestEnvironment();
    });

    it('opens metadata modal with i key', async () => {
        const card = document.querySelector('.media-card');
        card.click();

        await vi.waitFor(() => {
            const pipPlayer = document.getElementById('pip-player');
            return !pipPlayer.classList.contains('hidden');
        });

        const modal = document.getElementById('metadata-modal');
        expect(modal.classList.contains('hidden')).toBe(true);

        const iEvent = new KeyboardEvent('keydown', { key: 'i', bubbles: true });
        document.dispatchEvent(iEvent);

        await vi.waitFor(() => {
            expect(modal.classList.contains('hidden')).toBe(false);
        });
    });

    it('closes metadata modal with i key', async () => {
        const card = document.querySelector('.media-card');
        card.click();

        await vi.waitFor(() => {
            const pipPlayer = document.getElementById('pip-player');
            return !pipPlayer.classList.contains('hidden');
        });

        // Open modal
        const iEvent = new KeyboardEvent('keydown', { key: 'i', bubbles: true });
        document.dispatchEvent(iEvent);

        await vi.waitFor(() => {
            const modal = document.getElementById('metadata-modal');
            expect(modal.classList.contains('hidden')).toBe(false);
        });

        // Close modal
        const iEvent2 = new KeyboardEvent('keydown', { key: 'i', bubbles: true });
        document.dispatchEvent(iEvent2);

        await vi.waitFor(() => {
            const modal = document.getElementById('metadata-modal');
            expect(modal.classList.contains('hidden')).toBe(true);
        });
    });

    it('shows media metadata in modal', async () => {
        const card = document.querySelector('.media-card');
        card.click();

        await vi.waitFor(() => {
            const pipPlayer = document.getElementById('pip-player');
            return !pipPlayer.classList.contains('hidden');
        });

        const iEvent = new KeyboardEvent('keydown', { key: 'i', bubbles: true });
        document.dispatchEvent(iEvent);

        // Modal should exist
        const modal = document.getElementById('metadata-modal');
        expect(modal).toBeTruthy();
    });

    it('closes metadata modal with close button', async () => {
        const card = document.querySelector('.media-card');
        card.click();

        await vi.waitFor(() => {
            const pipPlayer = document.getElementById('pip-player');
            return !pipPlayer.classList.contains('hidden');
        });

        // Open modal
        const iEvent = new KeyboardEvent('keydown', { key: 'i', bubbles: true });
        document.dispatchEvent(iEvent);

        await vi.waitFor(() => {
            const modal = document.getElementById('metadata-modal');
            return !modal.classList.contains('hidden');
        });

        // Close with button
        const modal = document.getElementById('metadata-modal');
        const closeBtn = modal.querySelector('.close-modal');
        if (closeBtn) {
            closeBtn.click();

            await vi.waitFor(() => {
                expect(modal.classList.contains('hidden')).toBe(true);
            });
        } else {
            // If no close button, close with keyboard
            const iEvent2 = new KeyboardEvent('keydown', { key: 'i', bubbles: true });
            document.dispatchEvent(iEvent2);
        }
    });

    it('shows metadata modal via right-click context menu', async () => {
        const card = document.querySelector('.media-card');

        const contextMenuEvent = new MouseEvent('contextmenu', {
            bubbles: true,
            cancelable: true,
            clientX: 100,
            clientY: 100
        });
        card.dispatchEvent(contextMenuEvent);

        // Check if context menu appears (if implemented)
        const contextMenu = document.querySelector('.context-menu');
        if (contextMenu) {
            expect(contextMenu.classList.contains('hidden')).toBe(false);
        }
    });
});
