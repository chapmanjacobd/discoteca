export interface MediaItem {
    path: string;
    name: string;
    type: string;
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
    // Percentiles for slider calculations (0%, 16.6%, 33.3%, 50%, 66.6%, 83.3%, 100%)
    // Use percentiles[0] as min and percentiles[len-1] as max
    episodes_percentiles: number[];
    size_percentiles: number[];
    duration_percentiles: number[];
    modified_percentiles: number[];
    created_percentiles: number[];
    downloaded_percentiles: number[];

    // Type counts (special case - not a percentile distribution)
    type: FilterBin[];
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

export interface State {
    view: string;
    page: 'search' | 'trash' | 'history' | 'playlist' | 'du' | 'curation' | 'captions' | 'episodes';
    currentPage: number;
    totalCount: number;
    filters: {
        types: string[];
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
}
