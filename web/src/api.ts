export function getCookie(name: string): string | undefined {
    const value = `; ${document.cookie}`;
    const parts = value.split(`; ${name}=`);
    if (parts.length === 2) return parts.pop()?.split(';').shift();
    return undefined;
}

export async function fetchAPI(url: string, options: RequestInit = {}): Promise<Response> {
    const token = getCookie('disco_token');
    const headers = {
        ...(options.headers as Record<string, string>),
        'X-Disco-Token': token || ''
    };
    const resp = await fetch(url, { ...options, headers });
    if (resp.status === 403) throw new Error('Access Denied');
    if (resp.status === 401) {
        window.location.reload();
        throw new Error('Unauthorized');
    }
    return resp;
}
