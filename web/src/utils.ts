/**
 * Escape a value for safe interpolation into HTML text or a quoted attribute.
 * Use this anywhere user-controlled data (paths, titles, search terms, tags)
 * is placed into an innerHTML template.
 */
export function escapeHtml(value: unknown): string {
    return String(value ?? '')
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#39;');
}

export function formatRelativeDate(timestamp: number | null): string {
    if (!timestamp || timestamp === 0) return '-';
    const now = Math.floor(Date.now() / 1000);
    const diff = now - timestamp;

    if (diff < 60) return 'just now';
    if (diff < 3600) return `${Math.floor(diff / 60)}m ago`;
    if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`;
    if (diff < 2592000) return `${Math.floor(diff / 86400)}d ago`;
    if (diff < 31536000) return `${Math.floor(diff / 2592000)}mo ago`;
    return `${Math.floor(diff / 31536000)}y ago`;
}

export function formatSize(bytes: number | undefined | null): string {
    if (bytes === undefined || bytes === null || bytes === 0) return '-';
    let b = bytes;
    const units = ['B', 'KB', 'MB', 'GB', 'TB'];
    let i = 0;
    while (b >= 1000 && i < units.length - 1) {
        b /= 1000;
        i++;
    }
    return `${b.toFixed(1)} ${units[i]}`;
}

export function formatDuration(seconds: number | undefined): string {
    if (seconds === undefined || seconds === null) return '';

    const totalSeconds = Math.floor(seconds);

    const d = Math.floor(totalSeconds / 86400);
    const h = Math.floor((totalSeconds % 86400) / 3600);
    const m = Math.floor((totalSeconds % 3600) / 60);
    const s = totalSeconds % 60;

    if (d > 0) {
        // Show days for durations > 24 hours
        return `${d}d ${h < 10 ? '0' + h : h}:${m < 10 ? '0' + m : m}`;
    }
    if (h > 0) {
        return `${h}:${m < 10 ? '0' + m : m}:${s < 10 ? '0' + s : s}`;
    }
    return `${m}:${s < 10 ? '0' + s : s}`;
}

export function shortDuration(seconds: number | undefined): string {
    if (!seconds) return '0s';
    const d = Math.floor(seconds / 86400);
    const h = Math.floor((seconds % 86400) / 3600);
    const m = Math.floor((seconds % 3600) / 60);
    const s = Math.floor(seconds % 60);

    const parts = [];
    if (d > 0) parts.push(`${d}d`);
    if (h > 0) parts.push(`${h}h`);
    if (m > 0) parts.push(`${m}m`);
    if (s > 0 && d === 0) parts.push(`${s}s`);
    return parts.join(' ') || '0s';
}

export function truncateString(str: string | undefined): string {
    if (!str) return '';
    const limit = window.innerWidth <= 768 ? 35 : 55;
    if (str.length <= limit) return str;
    return str.substring(0, limit - 3) + '...';
}

export function formatParents(path: string | undefined): string {
    if (!path) return '';
    const parts = path.split('/');
    if (parts.length > 1) {
        // Remove filename
        parts.pop();
        if (parts.length === 0) return '';
        // Show up to two parent folders
        const display = parts.slice(-2).join('/');
        return truncateString(display);
    }
    return '';
}

export function getIcon(type: string | undefined): string {
    if (!type) return '📄';
    if (type.includes('video')) return '🎬';
    if (type.includes('audio')) return '🎵';
    if (type.includes('image')) return '🖼️';
    if (type.includes('epub') || type.includes('pdf') || type.includes('mobi')) return '📚';
    return '📄';
}

/**
 * Generate a client-side thumbnail using canvas when server thumbnail fails
 * Creates a colored placeholder with file extension
 */
export function generateClientThumbnail(canvas: HTMLCanvasElement, filename: string, type: string | undefined): string {
    const ctx = canvas.getContext('2d');
    if (!ctx) return '';

    const width = canvas.width || 320;
    const height = canvas.height || 240;

    // Get color based on file type
    const colors: Record<string, string> = {
        'video': '#8b5cf6',
        'audio': '#ec4899',
        'image': '#10b981',
        'epub': '#f59e0b',
        'pdf': '#ef4444',
        'default': '#3b82f6'
    };

    let color = colors['default'];
    if (type) {
        for (const [key, value] of Object.entries(colors)) {
            if (type.includes(key)) {
                color = value;
                break;
            }
        }
    }

    // Get file extension
    const ext = filename.split('.').pop()?.toUpperCase() || 'FILE';

    // Draw background gradient
    const gradient = ctx.createLinearGradient(0, 0, width, height);
    gradient.addColorStop(0, color);
    gradient.addColorStop(1, adjustColor(color, 20));
    ctx.fillStyle = gradient;
    ctx.fillRect(0, 0, width, height);

    // Draw extension text
    ctx.fillStyle = 'white';
    ctx.font = 'bold 48px system-ui, sans-serif';
    ctx.textAlign = 'center';
    ctx.textBaseline = 'middle';
    ctx.fillText(ext, width / 2, height / 2 - 20);

    // Draw filename
    ctx.font = '14px system-ui, sans-serif';
    ctx.fillStyle = 'rgba(255, 255, 255, 0.7)';
    const displayName = filename.split('/').pop() || filename;
    const truncatedName = displayName.length > 30 ? displayName.substring(0, 27) + '...' : displayName;
    ctx.fillText(truncatedName, width / 2, height - 30);

    return canvas.toDataURL('image/jpeg');
}

/**
 * Adjust color brightness
 */
function adjustColor(hex: string, percent: number): string {
    const num = parseInt(hex.replace('#', ''), 16);
    const amt = Math.round(2.55 * percent);
    const R = Math.max(0, Math.min(255, (num >> 16) + amt));
    const G = Math.max(0, Math.min(255, ((num >> 8) & 0x00FF) + amt));
    const B = Math.max(0, Math.min(255, (num & 0x0000FF) + amt));
    return '#' + (0x1000000 + R * 0x10000 + G * 0x100 + B).toString(16).slice(1);
}
