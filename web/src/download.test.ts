import { describe, it, expect } from 'vitest';
import { defaultDownloadOptions, isDownloadURL, normalizeDownloadOptions } from './download';

describe('isDownloadURL', () => {
    it('accepts http and https URLs', () => {
        expect(isDownloadURL('https://example.com/video')).toBe(true);
        expect(isDownloadURL('http://example.com/video?x=1')).toBe(true);
        expect(isDownloadURL('  https://example.com/video  ')).toBe(true);
    });

    it('rejects non-URLs and unsafe schemes', () => {
        expect(isDownloadURL('')).toBe(false);
        expect(isDownloadURL('example.com/video')).toBe(false);
        expect(isDownloadURL('file:///etc/passwd')).toBe(false);
        expect(isDownloadURL('javascript:alert(1)')).toBe(false);
        expect(isDownloadURL('https://')).toBe(false);
    });
});

describe('normalizeDownloadOptions', () => {
    it('returns defaults for missing or malformed values', () => {
        expect(normalizeDownloadOptions(null)).toEqual(defaultDownloadOptions);
        expect(normalizeDownloadOptions({ media_type: 'image', subtitles: 'maybe' })).toEqual(
            defaultDownloadOptions
        );
    });

    it('keeps stored values', () => {
        const options = normalizeDownloadOptions({
            database: '/tmp/library.db',
            media_type: 'audio',
            subtitles: 'auto',
            audio_format: 'mp3',
            cookies_from_browser: 'firefox',
            extra_metadata: true
        });
        expect(options.database).toBe('/tmp/library.db');
        expect(options.media_type).toBe('audio');
        expect(options.subtitles).toBe('auto');
        expect(options.audio_format).toBe('mp3');
        expect(options.cookies_from_browser).toBe('firefox');
        expect(options.extra_metadata).toBe(true);
    });
});
