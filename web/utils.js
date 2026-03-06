export function formatRelativeDate(timestamp) {
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

export function formatSize(bytes) {
    if (!bytes) return '-';
    const units = ['B', 'KB', 'MB', 'GB', 'TB'];
    let i = 0;
    while (bytes >= 1024 && i < units.length - 1) {
        bytes /= 1024;
        i++;
    }
    return `${bytes.toFixed(1)} ${units[i]}`;
}

export function formatDuration(seconds) {
    if (!seconds && seconds !== 0) return '';
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

export function shortDuration(seconds) {
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

export function truncateString(str) {
    if (!str) return '';
    const limit = window.innerWidth <= 768 ? 35 : 55;
    if (str.length <= limit) return str;
    return str.substring(0, limit - 3) + '...';
}

export function formatParents(path) {
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

export function getIcon(type) {
    if (!type) return '📄';
    if (type.includes('video')) return '🎬';
    if (type.includes('audio')) return '🎵';
    if (type.includes('image')) return '🖼️';
    if (type.includes('epub') || type.includes('pdf') || type.includes('mobi')) return '📚';
    return '📄';
}
