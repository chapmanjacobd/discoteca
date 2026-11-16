import { appUrl } from './base';

export async function fetchAPI(url: string, options: RequestInit = {}): Promise<Response> {
    const resp = await fetch(appUrl(url), options);
    if (resp.status === 403) throw new Error('Access Denied');
    if (resp.status === 401) throw new Error('Unauthorized');
    return resp;
}
