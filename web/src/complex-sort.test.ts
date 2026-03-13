import { describe, it, expect, beforeEach, vi } from 'vitest';
import { SORT_FIELDS, SORT_PRESETS, matchesPreset } from './complex-sort';

// Mock state module
vi.mock('./state', () => ({
    state: {
        filters: {
            sort: 'default',
            reverse: false,
            customSortFields: ''
        }
    }
}));

describe('SORT_FIELDS', () => {
    it('should contain all expected sort fields', () => {
        expect(SORT_FIELDS.length).toBeGreaterThan(10);
        
        const fieldValues = SORT_FIELDS.map(f => f.value);
        expect(fieldValues).toContain('video_count');
        expect(fieldValues).toContain('audio_count');
        expect(fieldValues).toContain('play_count');
        expect(fieldValues).toContain('time_last_played');
        expect(fieldValues).toContain('path');
        expect(fieldValues).toContain('size');
        expect(fieldValues).toContain('duration');
    });

    it('should have labels for all fields', () => {
        SORT_FIELDS.forEach(field => {
            expect(field.label).toBeDefined();
            expect(field.label.length).toBeGreaterThan(0);
        });
    });
});

describe('SORT_PRESETS', () => {
    it('should contain default preset with xklb sorting', () => {
        expect(SORT_PRESETS.default).toBeDefined();
        expect(SORT_PRESETS.default.length).toBeGreaterThan(0);
        
        // First field should be video_count desc
        const firstField = SORT_PRESETS.default[0];
        expect(firstField.field).toBe('video_count');
        expect(firstField.reverse).toBe(true);
    });

    it('should contain presets for all sort options', () => {
        const expectedPresets = [
            'default', 'path', 'size', 'duration', 'play_count',
            'time_last_played', 'progress', 'time_created', 'time_modified',
            'time_downloaded', 'bitrate', 'extension', 'random'
        ];
        
        expectedPresets.forEach(preset => {
            expect(SORT_PRESETS[preset]).toBeDefined();
            expect(SORT_PRESETS[preset].length).toBeGreaterThan(0);
        });
    });

    it('should have valid field configurations for each preset', () => {
        Object.entries(SORT_PRESETS).forEach(([name, fields]) => {
            fields.forEach(field => {
                expect(field.field).toBeDefined();
                expect(typeof field.reverse).toBe('boolean');
                
                // Field should exist in SORT_FIELDS
                const fieldExists = SORT_FIELDS.some(f => f.value === field.field);
                expect(fieldExists).toBe(true);
            });
        });
    });
});

describe('matchesPreset', () => {
    it('should match exact preset configuration', () => {
        const result = matchesPreset(SORT_PRESETS.default);
        expect(result).toBe('default');
    });

    it('should match path preset', () => {
        const result = matchesPreset(SORT_PRESETS.path);
        expect(result).toBe('path');
    });

    it('should return null for non-matching configuration', () => {
        const customFields = [
            { field: 'path', reverse: false },
            { field: 'size', reverse: true }
        ];
        const result = matchesPreset(customFields);
        expect(result).toBeNull();
    });

    it('should return null for empty array', () => {
        const result = matchesPreset([]);
        expect(result).toBeNull();
    });

    it('should not match if reverse is different', () => {
        const reversedPath = [
            { field: 'path', reverse: true }
        ];
        const result = matchesPreset(reversedPath);
        expect(result).toBeNull();
    });

    it('should match size preset', () => {
        const result = matchesPreset(SORT_PRESETS.size);
        expect(result).toBe('size');
    });

    it('should match progress preset', () => {
        const result = matchesPreset(SORT_PRESETS.progress);
        expect(result).toBe('progress');
    });
});

describe('Sort Field Configuration', () => {
    it('should handle multiple fields in custom configuration', () => {
        const customFields = [
            { field: 'video_count', reverse: true },
            { field: 'audio_count', reverse: true },
            { field: 'play_count', reverse: false }
        ];

        const result = matchesPreset(customFields);
        expect(result).toBeNull(); // Should be custom, not a preset
    });

    it('should preserve field order in matching', () => {
        const reversedDefault = [...SORT_PRESETS.default].reverse();
        const result = matchesPreset(reversedDefault);
        expect(result).toBeNull(); // Order matters, should not match
    });

    it('should handle single field presets', () => {
        expect(SORT_PRESETS.path.length).toBe(1);
        expect(SORT_PRESETS.size.length).toBe(1);
        expect(SORT_PRESETS.duration.length).toBe(1);
    });

    it('should handle multi-field presets', () => {
        expect(SORT_PRESETS.default.length).toBeGreaterThan(1);
        expect(SORT_PRESETS.progress.length).toBe(2);
        expect(SORT_PRESETS.bitrate.length).toBe(2);
    });
});

describe('Sort Field Values', () => {
    it('should have correct reverse values for default preset', () => {
        const defaultFields = SORT_PRESETS.default;
        
        // Videos before audio
        expect(defaultFields[0].field).toBe('video_count');
        expect(defaultFields[0].reverse).toBe(true);
        
        // Audio before silent
        expect(defaultFields[1].field).toBe('audio_count');
        expect(defaultFields[1].reverse).toBe(true);
        
        // Unplayed first (play_count asc)
        const playCountField = defaultFields.find(f => f.field === 'play_count');
        expect(playCountField?.reverse).toBe(false);
        
        // Resume progress (playhead desc)
        const playheadField = defaultFields.find(f => f.field === 'playhead');
        expect(playheadField?.reverse).toBe(true);
    });
});

describe('Custom Sort Field Parsing', () => {
    it('should parse custom sort fields string with desc/asc', () => {
        const customFieldsStr = 'video_count desc,audio_count asc,path desc';
        const parts = customFieldsStr.split(',');
        const parsed = parts.map(p => {
            const trimmed = p.trim();
            const reverseMatch = trimmed.match(/^(.+?)\s+(asc|desc)$/i);
            if (reverseMatch) {
                return {
                    field: reverseMatch[1].trim(),
                    reverse: reverseMatch[2].toLowerCase() === 'desc'
                };
            }
            return { field: trimmed, reverse: false };
        });

        expect(parsed.length).toBe(3);
        expect(parsed[0].field).toBe('video_count');
        expect(parsed[0].reverse).toBe(true);
        expect(parsed[1].field).toBe('audio_count');
        expect(parsed[1].reverse).toBe(false);
        expect(parsed[2].field).toBe('path');
        expect(parsed[2].reverse).toBe(true);
    });

    it('should parse fields without direction (default asc)', () => {
        const customFieldsStr = 'path,size,duration';
        const parts = customFieldsStr.split(',');
        const parsed = parts.map(p => {
            const trimmed = p.trim();
            const reverseMatch = trimmed.match(/^(.+?)\s+(asc|desc)$/i);
            if (reverseMatch) {
                return {
                    field: reverseMatch[1].trim(),
                    reverse: reverseMatch[2].toLowerCase() === 'desc'
                };
            }
            return { field: trimmed, reverse: false };
        });

        expect(parsed.length).toBe(3);
        expect(parsed[0].reverse).toBe(false);
        expect(parsed[1].reverse).toBe(false);
        expect(parsed[2].reverse).toBe(false);
    });
});
