export interface MediaItem {
    path: string;
    name: string;
    media_type: string;
    is_dir: boolean;
    size?: number;
    duration?: number;
    play_count?: number;
    time_last_played?: string;
    progress?: number;
    time_created?: string;
    time_modified?: string;
    time_downloaded?: string;
    bitrate?: number;
    extension?: string;
    score?: number;
    categories?: string[];
    languages?: string[];
    rating?: number;
    similarity?: number;
    parent_path?: string;
    transcode?: boolean;
}

export interface FilterBin {
    label: string;
    min?: number;
    max?: number;
    value?: number;
}

export interface FilterBins {
    // Actual min/max values from the full dataset (stable across filtering)
    episodes_min_val: number;
    episodes_max_val: number;
    size_min_val: number;
    size_max_val: number;
    duration_min_val: number;
    duration_max_val: number;
    modified_min_val: number;
    modified_max_val: number;
    created_min_val: number;
    created_max_val: number;
    downloaded_min_val: number;
    downloaded_max_val: number;

    // Percentiles for slider calculations (0-100)
    episodes_percentiles: number[];
    size_percentiles: number[];
    duration_percentiles: number[];
    modified_percentiles: number[];
    created_percentiles: number[];
    downloaded_percentiles: number[];

    // MediaType counts (special case - not a percentile distribution)
    media_type: FilterBin[];
}

export interface PlaybackState {
    item: MediaItem | null;
    queueIndex: number;
    queue: MediaItem[];
    repeatMode: 'off' | 'one' | 'all';
    shuffle: boolean;
    timer: any;
    slideshowTimer: any;
    startTime: number | null;
    lastUpdate: number;
    lastLocalUpdate: number;
    lastPlayedIndex: number;
    hasMarkedComplete: boolean;
    pendingUpdate: any;
    skipTimeout: any;
    lastSkipTime: number;
    hlsInstance: any;
    toastTimer: any;
    muted: boolean;
    consecutiveErrors: number;
    seekHistory: number[];
    markedPosition: number | null;
}

export interface DocumentViewSettings {
    theme: 'light' | 'dark' | 'sepia' | 'solarized' | 'auto';
    fontSize: number;
    fontFamily: string;
    textReflow: boolean;
    columnSpread: 1 | 2;
    lineHeight: number;
    letterSpacing: number;
    maxWidth: number;
    textAlign: 'left' | 'justify';
}

export interface State {
    view: string;
    page: 'search' | 'trash' | 'history' | 'playlist' | 'du' | 'curation' | 'captions' | 'episodes';
    currentPage: number;
    totalCount: number;
    filters: {
        media_types: string[];
        search: string;
        categories: string[];
        genre: string;
        languages: string[];
        ratings: string[];
        playlist: string | null;
        sort: string;
        reverse: boolean;
        limit: number;
        all: boolean;
        excludedDbs: string[];
        sizes: any[];
        durations: any[];
        modified: any[];
        created: any[];
        downloaded: any[];
        min_score: string;
        max_score: string;
        episodes: any[];
        unplayed: boolean;
        unfinished: boolean;
        completed: boolean;
        captions: boolean;
        searchType: 'fts' | 'substring';
        browseCol: string;
        browseVal: string;
        customSortFields?: string;
    };
    activeModal: string | null;
    duPath: string;
    draggedItem: MediaItem | null;
    applicationStartTime: number | null;
    lastActivity: number;
    player: string;
    language: string;
    theme: string;
    postPlaybackAction: string;
    defaultView: 'pip' | 'theatre';
    autoplay: boolean;
    imageAutoplay: boolean;
    localResume: boolean;
    showPipSpeed: boolean;
    showPipSurf: boolean;
    showPipStream: boolean;
    defaultVideoRate: number;
    defaultAudioRate: number;
    playbackRate: number;
    slideshowDelay: number;
    rsvpWpm: number;
    autoLoopMaxDuration: number;
    enableQueue: boolean;
    queueExpanded: boolean;
    queueAddMode: 'end' | 'next';
    playerMode: 'pip' | 'theatre';
    readOnly: boolean;
    dev: boolean;
    databases: string[];
    categories: { category: string; count: number }[];
    genres: { genre: string; count: number }[];
    languages: { category: string; count: number }[];
    ratings: { rating: number; count: number }[];
    filterBins: FilterBins;
    duData?: any[];
    duDataRaw?: any; // Raw API response: {folders?: [], files?: []}
    similarityData?: any[];
    playlists: string[];
    newCategories: string[];
    playlistItems: MediaItem[];
    sidebarState: Record<string, boolean>;
    lastSuggestions: MediaItem[];
    playback: PlaybackState;
    documentView: DocumentViewSettings;
}
