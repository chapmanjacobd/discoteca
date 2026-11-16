export type DownloadMediaType = 'auto' | 'video' | 'audio';
export type DownloadSubtitleMode = 'none' | 'download' | 'auto';

export interface DownloadOptions {
    database: string;
    media_type: DownloadMediaType;
    subtitles: DownloadSubtitleMode;
    audio_format: string;
    cookies_from_browser: string;
    cookies_file: string;
    download_archive: string;
    extra_metadata: boolean;
    safe: boolean;
}

export const defaultDownloadOptions: DownloadOptions = {
    database: '',
    media_type: 'auto',
    subtitles: 'download',
    audio_format: 'opus',
    cookies_from_browser: '',
    cookies_file: '',
    download_archive: '',
    extra_metadata: false,
    safe: false
};

export function isDownloadURL(value: string): boolean {
    const trimmed = value.trim();
    if (!trimmed) return false;
    try {
        const url = new URL(trimmed);
        return (url.protocol === 'http:' || url.protocol === 'https:') && url.hostname.length > 0;
    } catch {
        return false;
    }
}

export function normalizeDownloadOptions(value: unknown): DownloadOptions {
    const raw = value && typeof value === 'object' ? (value as Partial<DownloadOptions>) : {};
    const mediaType: DownloadMediaType =
        raw.media_type === 'video' || raw.media_type === 'audio' ? raw.media_type : 'auto';
    const subtitles: DownloadSubtitleMode =
        raw.subtitles === 'none' || raw.subtitles === 'auto' ? raw.subtitles : 'download';

    return {
        database: typeof raw.database === 'string' ? raw.database : '',
        media_type: mediaType,
        subtitles,
        audio_format:
            typeof raw.audio_format === 'string' && raw.audio_format
                ? raw.audio_format
                : defaultDownloadOptions.audio_format,
        cookies_from_browser:
            typeof raw.cookies_from_browser === 'string' ? raw.cookies_from_browser : '',
        cookies_file: typeof raw.cookies_file === 'string' ? raw.cookies_file : '',
        download_archive: typeof raw.download_archive === 'string' ? raw.download_archive : '',
        extra_metadata: raw.extra_metadata === true,
        safe: raw.safe === true
    };
}
