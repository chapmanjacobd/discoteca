import { describe, it, expect } from 'vitest';
import { applyBasePath } from './base';

describe('applyBasePath', () => {
    it('leaves URLs unchanged at the root', () => {
        expect(applyBasePath('', '/api/query?limit=10')).toBe('/api/query?limit=10');
        expect(applyBasePath('/', '/api/query')).toBe('/api/query');
    });

    it('prefixes application paths', () => {
        expect(applyBasePath('/disco', '/api/query')).toBe('/disco/api/query');
        expect(applyBasePath('/disco/', '/api/query')).toBe('/disco/api/query');
        expect(applyBasePath('/a/b', '/index.js')).toBe('/a/b/index.js');
    });

    it('is idempotent', () => {
        const once = applyBasePath('/disco', '/api/query');
        expect(applyBasePath('/disco', once)).toBe(once);
    });

    it('does not prefix absolute or relative URLs', () => {
        expect(applyBasePath('/disco', 'https://example.com/x')).toBe('https://example.com/x');
        expect(applyBasePath('/disco', 'blob:http://localhost/123')).toBe('blob:http://localhost/123');
        expect(applyBasePath('/disco', 'data:text/plain,hi')).toBe('data:text/plain,hi');
        expect(applyBasePath('/disco', 'relative/path')).toBe('relative/path');
        expect(applyBasePath('/disco', '')).toBe('');
    });

    it('does not treat a sibling prefix as already prefixed', () => {
        expect(applyBasePath('/disco', '/discovery')).toBe('/disco/discovery');
    });
});
