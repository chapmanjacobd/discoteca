export function getCookie(name) {
    const value = `; ${document.cookie}`;
    const parts = value.split(`; ${name}=`);
    if (parts.length === 2) return parts.pop().split(';').shift();
}

export async function fetchAPI(url, options = {}) {
    const token = getCookie('disco_token');
    const headers = {
        ...options.headers,
        'X-Disco-Token': token
    };
    return fetch(url, { ...options, headers });
}
