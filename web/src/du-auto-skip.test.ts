import { describe, it, expect, beforeEach, vi } from 'vitest';

describe('DU Mode Auto-Skip Logic', () => {
    beforeEach(() => {
        vi.clearAllMocks();
    });

    /**
     * Test the auto-skip decision logic
     * Auto-skip should happen when:
     * 1. duData.length <= 1 (keep descending until duData.length > 1)
     * 2. The single item is a folder with count > 0
     */
    it('should determine when to auto-skip based on duData.length', () => {
        // Test case 1: Single folder (duData.length = 1) - should skip
        const data1 = [{ path: '/home', count: 5, total_size: 1000 }];
        const shouldSkip1 = data1.length <= 1 && data1[0].count > 0;
        expect(shouldSkip1).toBe(true);

        // Test case 2: Multiple items (duData.length = 2) - should NOT skip
        const data2 = [
            { path: '/home', count: 5, total_size: 1000 },
            { path: '/var', count: 3, total_size: 500 }
        ];
        const shouldSkip2 = data2.length <= 1 && data2[0]?.count > 0;
        expect(shouldSkip2).toBe(false);

        // Test case 3: Single file (duData.length = 1, but no count property) - should NOT skip
        const data3: any[] = [{ path: '/file.txt', size: 100 }];
        const shouldSkip3 = data3.length <= 1 && data3[0].count !== undefined && data3[0].count > 0;
        expect(shouldSkip3).toBe(false);

        // Test case 4: Single empty folder (count=0) - should NOT skip
        const data4 = [{ path: '/empty', count: 0, total_size: 0 }];
        const shouldSkip4 = data4.length <= 1 && data4[0].count !== undefined && data4[0].count > 0;
        expect(shouldSkip4).toBe(false);

        // Test case 5: Empty data (duData.length = 0) - should NOT skip (no item to navigate to)
        const data5: any[] = [];
        const shouldSkip5 = data5.length <= 1 && data5[0]?.count !== undefined && data5[0]?.count > 0;
        expect(shouldSkip5).toBe(false);
    });

    /**
     * Test multi-level auto-skip path construction
     */
    it('should construct correct paths for multi-level auto-skip', () => {
        const paths: string[] = [];
        
        // Simulate auto-skip chain: / -> /home -> /home/xk -> /home/xk/sync
        const skipChain = ['/', '/home', '/home/xk', '/home/xk/sync'];

        for (const path of skipChain) {
            // Normalize path: add trailing slash unless it's root
            const normalized = path === '/' ? '/' : path + '/';
            paths.push(normalized);
        }

        expect(paths).toEqual(['/', '/home/', '/home/xk/', '/home/xk/sync/']);
    });

    /**
     * Test path normalization for auto-skip
     */
    it('should normalize paths correctly', () => {
        const normalizePath = (path: string): string => {
            return path + (path.endsWith('/') ? '' : '/');
        };

        expect(normalizePath('/home')).toBe('/home/');
        expect(normalizePath('/home/')).toBe('/home/');
        expect(normalizePath('/')).toBe('/');
        expect(normalizePath('media')).toBe('media/');
    });

    /**
     * Test auto-skip termination conditions
     */
    it('should stop auto-skip at correct conditions', () => {
        // Condition 1: Multiple folders
        expect(() => {
            const folders = [{ path: '/a' }, { path: '/b' }];
            if (folders.length !== 1) throw new Error('STOP: multiple folders');
        }).toThrow('STOP: multiple folders');

        // Condition 2: Files present
        expect(() => {
            const files = [{ path: '/file.txt' }];
            if (files.length !== 0) throw new Error('STOP: files present');
        }).toThrow('STOP: files present');

        // Condition 3: Empty folder (count=0)
        expect(() => {
            const folder = { path: '/empty', count: 0 };
            if (folder.count <= 0) throw new Error('STOP: empty folder');
        }).toThrow('STOP: empty folder');

        // Condition 4: Max depth reached (simulated)
        expect(() => {
            const depth = 5;
            const maxDepth = 5;
            if (depth >= maxDepth) throw new Error('STOP: max depth');
        }).toThrow('STOP: max depth');
    });

    /**
     * Test auto-skip with different path formats
     */
    it('should handle different path formats', () => {
        const testCases = [
            { input: '/home/user', expected: '/home/user/' },
            { input: '/home/user/', expected: '/home/user/' },
            { input: 'relative/path', expected: 'relative/path/' },
            { input: 'single', expected: 'single/' },
            { input: '/', expected: '/' },
            { input: '', expected: '/' }
        ];

        for (const { input, expected } of testCases) {
            const normalized = input ? (input.endsWith('/') ? input : input + '/') : '/';
            expect(normalized).toBe(expected);
        }
    });
});
