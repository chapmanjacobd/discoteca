import { appUrl } from './base';

export function getCookie(name: string): string | undefined {
    const value = `; ${document.cookie}`;
    const parts = value.split(`; ${name}=`);
    if (parts.length === 2) return parts.pop()?.split(';').shift();
    return undefined;
}

let sessionPromise: Promise<void> | null = null;
let promptedForToken = false;

function tokenFromLocation(): string | undefined {
    const hash = window.location.hash.startsWith('#')
        ? window.location.hash.slice(1)
        : window.location.hash;
    const hashParams = new URLSearchParams(hash);
    const searchParams = new URLSearchParams(window.location.search);
    const token = hashParams.get('token') || searchParams.get('token') || undefined;
    if (!token) return undefined;

    // Remove the token from the address bar before any further navigation.
    hashParams.delete('token');
    searchParams.delete('token');
    const hashText = hashParams.toString();
    const searchText = searchParams.toString();
    const next =
        window.location.pathname +
        (searchText ? `?${searchText}` : '') +
        (hashText ? `#${hashText}` : '');
    history.replaceState(null, '', next);
    return token;
}

async function createSession(token: string): Promise<boolean> {
    const resp = await fetch(appUrl('/api/session'), {
        method: 'POST',
        headers: { Authorization: `Bearer ${token}` },
    });
    return resp.ok;
}

function promptForToken(): string | undefined {
    if (promptedForToken) return undefined;
    promptedForToken = true;
    const token = window.prompt('Enter the discoteca API token');
    return token?.trim() || undefined;
}

// initSessionFromURL consumes a token supplied as ?token= or #token= and
// exchanges it for an HttpOnly session cookie. It is safe to call without a
// token: the app then relies on an existing cookie or prompts on a 401.
export function initSessionFromURL(): void {
    const token = tokenFromLocation();
    if (!token) return;
    sessionPromise = createSession(token)
        .then(() => undefined)
        .catch(() => undefined);
}

async function sessionReady(): Promise<void> {
    if (sessionPromise) await sessionPromise;
}

export async function fetchAPI(url: string, options: RequestInit = {}, retried = false): Promise<Response> {
    if (sessionPromise) await sessionReady();

    const token = getCookie('disco_token');
    const headers: Record<string, string> = {
        ...(options.headers as Record<string, string>),
    };
    if (token && !headers['X-Disco-Token'] && !headers['Authorization']) {
        headers['X-Disco-Token'] = token;
    }

    const resp = await fetch(appUrl(url), { ...options, headers });
    if (resp.status === 403) throw new Error('Access Denied');
    if (resp.status === 401) {
        if (!retried) {
            const urlToken = tokenFromLocation();
            if (urlToken) {
                if (await createSession(urlToken)) {
                    return fetchAPI(url, options, true);
                }
            } else {
                const supplied = promptForToken();
                if (supplied && (await createSession(supplied))) {
                    // Reload so every request that already failed during
                    // initialization runs again with the new session cookie.
                    window.location.reload();
                    return new Promise<Response>(() => {});
                }
            }
        }
        throw new Error('Unauthorized');
    }
    return resp;
}
