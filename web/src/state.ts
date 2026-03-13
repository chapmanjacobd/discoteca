import { State } from './types';

const getLocalStorageItem = (key: string, defaultValue: string | null = null): string | null => {
    return localStorage.getItem(key) || defaultValue;
};

const getJSONLocalStorageItem = (key: string, defaultValue: any = []): any => {
    const item = localStorage.getItem(key);
    if (!item) return defaultValue;
    try {
        return JSON.parse(item);
    } catch (e) {
        return defaultValue;
    }
};

export const state: State = {
    view: getLocalStorageItem('disco-view', 'grid')!,
    page: 'search',
    currentPage: 1,
    totalCount: 0,
    filters: {
        types: getJSONLocalStorageItem('disco-types', []),
        search: '',
        categories: getJSONLocalStorageItem('disco-filter-categories', []),
        genre: '',
        languages: getJSONLocalStorageItem('disco-filter-languages', []),
        ratings: getJSONLocalStorageItem('disco-filter-ratings', []),
        playlist: null,
        sort: getLocalStorageItem('disco-sort', 'default')!,
        reverse: getLocalStorageItem('disco-reverse') === 'true',
        limit: parseInt(getLocalStorageItem('disco-limit', '99')!),
        all: getLocalStorageItem('disco-limit-all') === 'true',
        excludedDbs: getJSONLocalStorageItem('disco-excluded-dbs', []),
        sizes: getJSONLocalStorageItem('disco-filter-sizes', []),
        durations: getJSONLocalStorageItem('disco-filter-durations', []),
        min_score: '',
        max_score: '',
        episodes: getJSONLocalStorageItem('disco-filter-episodes', []),
        unplayed: getLocalStorageItem('disco-unplayed') === 'true',
        unfinished: false,
        completed: false,
        captions: false,
        searchType: (getLocalStorageItem('disco-search-type', 'fts') as 'fts' | 'substring'),
        browseCol: '',
        browseVal: '',
        customSortFields: getLocalStorageItem('disco-custom-sort-fields', '')
    },
    activeModal: null,
    duPath: '',
    draggedItem: null,
    applicationStartTime: null,
    lastActivity: Date.now() - (4 * 60 * 1000),
    player: getLocalStorageItem('disco-player', 'browser')!,
    language: getLocalStorageItem('disco-language', '')!,
    theme: getLocalStorageItem('disco-theme', 'auto')!,
    postPlaybackAction: getLocalStorageItem('disco-post-playback', 'nothing')!,
    defaultView: (getLocalStorageItem('disco-default-view', 'pip') as 'pip' | 'theatre'),
    autoplay: getLocalStorageItem('disco-autoplay') !== 'false',
    imageAutoplay: getLocalStorageItem('disco-image-autoplay') === 'true',
    get localResume() {
        return localStorage.getItem('disco-local-resume') !== 'false';
    },
    set localResume(value: boolean) {
        localStorage.setItem('disco-local-resume', value.toString());
    },
    showPipSpeed: getLocalStorageItem('disco-show-pip-speed') === 'true',
    showPipSurf: getLocalStorageItem('disco-show-pip-surf') === 'true',
    showPipStream: getLocalStorageItem('disco-show-pip-stream') === 'true',
    defaultVideoRate: parseFloat(getLocalStorageItem('disco-default-video-rate', '1.0')!),
    defaultAudioRate: parseFloat(getLocalStorageItem('disco-default-audio-rate', '1.0')!),
    playbackRate: parseFloat(getLocalStorageItem('disco-playback-rate', '1.0')!),
    slideshowDelay: parseInt(getLocalStorageItem('disco-slideshow-delay', '5')!),
    rsvpWpm: parseInt(getLocalStorageItem('disco-rsvp-wpm', '250')!),
    autoLoopMaxDuration: parseInt(getLocalStorageItem('disco-auto-loop-max-duration', '30')!),

    enableQueue: getLocalStorageItem('disco-enable-queue') === 'true',
    queueExpanded: getLocalStorageItem('disco-queue-expanded') === 'true',
    queueAddMode: (getLocalStorageItem('disco-queue-add-mode', 'end') as 'end' | 'next'),

    playerMode: (getLocalStorageItem('disco-default-view', 'pip') as 'pip' | 'theatre'),
    trashcan: false,
    readOnly: false,
    dev: false,
    databases: [],
    categories: [],
    genres: [],
    languages: [],
    ratings: [],
    filterBins: {
        episodes: [], size: [], duration: [],
        episodes_min: 0, episodes_max: 100,
        size_min: 0, size_max: 100 * 1024 * 1024,
        duration_min: 0, duration_max: 3600
    },
    playlists: [],
    newCategories: [],
    playlistItems: [],
    sidebarState: getJSONLocalStorageItem('disco-sidebar-state', {}),
    lastSuggestions: [],
    playback: {
        item: null,
        queueIndex: -1,
        queue: getJSONLocalStorageItem('disco-queue', []),
        repeatMode: (getLocalStorageItem('disco-repeat-mode', 'off') as 'off' | 'one' | 'all'),
        shuffle: getLocalStorageItem('disco-shuffle') === 'true',
        timer: null,
        slideshowTimer: null,
        startTime: null,
        lastUpdate: 0,
        lastLocalUpdate: 0,
        lastPlayedIndex: -1,
        hasMarkedComplete: false,
        pendingUpdate: null,
        skipTimeout: null,
        lastSkipTime: 0,
        hlsInstance: null,
        toastTimer: null,
        muted: getLocalStorageItem('disco-muted') === 'true',
        consecutiveErrors: 0,
        seekHistory: [],
        markedPosition: null
    }
};
