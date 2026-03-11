export const state = {
    view: localStorage.getItem('disco-view') || 'grid',
    page: 'search', // 'search', 'trash', 'history', or 'playlist'
    currentPage: 1,
    totalCount: 0,
    filters: {
        types: JSON.parse(localStorage.getItem('disco-types') || '[]'),
        search: '',
        categories: JSON.parse(localStorage.getItem('disco-filter-categories') || '[]'),
        genre: '',
        languages: JSON.parse(localStorage.getItem('disco-filter-languages') || '[]'),
        ratings: JSON.parse(localStorage.getItem('disco-filter-ratings') || '[]'),
        playlist: null, // This will now be the playlist title (string)
        sort: localStorage.getItem('disco-sort') || 'default',
        reverse: localStorage.getItem('disco-reverse') === 'true',
        limit: parseInt(localStorage.getItem('disco-limit')) || 99,
        all: localStorage.getItem('disco-limit-all') === 'true',
        excludedDbs: JSON.parse(localStorage.getItem('disco-excluded-dbs') || '[]'),
        sizes: JSON.parse(localStorage.getItem('disco-filter-sizes') || '[]'),
        durations: JSON.parse(localStorage.getItem('disco-filter-durations') || '[]'),
        min_score: '',
        max_score: '',
        episodes: JSON.parse(localStorage.getItem('disco-filter-episodes') || '[]'),
        unplayed: localStorage.getItem('disco-unplayed') === 'true',
        unfinished: false,
        completed: false,
        captions: false,
        searchType: localStorage.getItem('disco-search-type') || 'fts', // 'fts' or 'substring'
        browseCol: '',
        browseVal: ''
    },
    activeModal: null,
    duPath: '',
    draggedItem: null,
    applicationStartTime: null,
    lastActivity: Date.now() - (4 * 60 * 1000), // 4 mins ago
    player: localStorage.getItem('disco-player') || 'browser',
    language: localStorage.getItem('disco-language') || '',
    theme: localStorage.getItem('disco-theme') || 'auto',
    postPlaybackAction: localStorage.getItem('disco-post-playback') || 'nothing',
    defaultView: localStorage.getItem('disco-default-view') || 'pip',
    autoplay: localStorage.getItem('disco-autoplay') !== 'false',
    imageAutoplay: localStorage.getItem('disco-image-autoplay') === 'true',
    get localResume() {
        return localStorage.getItem('disco-local-resume') !== 'false';
    },
    set localResume(value) {
        localStorage.setItem('disco-local-resume', value);
    },
    showPipSpeed: localStorage.getItem('disco-show-pip-speed') === 'true',
    showPipSurf: localStorage.getItem('disco-show-pip-surf') === 'true',
    showPipStream: localStorage.getItem('disco-show-pip-stream') === 'true',
    defaultVideoRate: parseFloat(localStorage.getItem('disco-default-video-rate')) || 1.0,
    defaultAudioRate: parseFloat(localStorage.getItem('disco-default-audio-rate')) || 1.0,
    playbackRate: parseFloat(localStorage.getItem('disco-playback-rate')) || 1.0,
    slideshowDelay: parseInt(localStorage.getItem('disco-slideshow-delay')) || 5,
    rsvpWpm: parseInt(localStorage.getItem('disco-rsvp-wpm')) || 250,
    trackShuffleDuration: parseInt(localStorage.getItem('disco-track-shuffle-duration')) || 0,
    autoLoopMaxDuration: parseInt(localStorage.getItem('disco-auto-loop-max-duration')) || 30,

    enableQueue: localStorage.getItem('disco-enable-queue') === 'true',
    queueExpanded: localStorage.getItem('disco-queue-expanded') === 'true',
    queueAddMode: localStorage.getItem('disco-queue-add-mode') || 'end', // 'end' or 'next'

    playerMode: localStorage.getItem('disco-default-view') || 'pip', // Initialize with preference
    trashcan: false,
    readOnly: false,
    dev: false,
    databases: [], // Array of database paths from server
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
    playlists: [], // String array of titles
    newCategories: [], // Track categories added in this session to keep them at the top
    playlistItems: [], // Cache for client-side filtering
    sidebarState: JSON.parse(localStorage.getItem('disco-sidebar-state') || '{}'),
    lastSuggestions: [],
    playback: {
        item: null,
        queueIndex: -1,
        queue: JSON.parse(localStorage.getItem('disco-queue') || '[]'), // Queue of upcoming media items
        repeatMode: localStorage.getItem('disco-repeat-mode') || 'off', // 'off' | 'one' | 'all'
        shuffle: localStorage.getItem('disco-shuffle') === 'true',
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
        muted: localStorage.getItem('disco-muted') === 'true',
        consecutiveErrors: 0,
        seekHistory: [], // Stack of previous positions for undo seek
        markedPosition: null // Position marked by Shift+Ctrl+Backspace
    }
};
