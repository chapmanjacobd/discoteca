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
    const resp = await fetch(url, { ...options, headers });
    if (resp.status === 403) throw new Error('Access Denied');
    if (resp.status === 401) {
        window.location.reload();
        throw new Error('Unauthorized');
    }
    return resp;
}
