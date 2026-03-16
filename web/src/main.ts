import './style.css';
import Hls from 'hls.js';
import { fetchAPI, getCookie } from './api';
import { state } from './state';
import { initSliders, updateSliderLabels, setSliderValues, resetSliders, updateSlidersFromAbsolute } from './ui/Sliders';
import { initComplexSorting, loadConfigFromCurrentSort } from './complex-sort';
import {
    formatSize,
    formatDuration,
    formatRelativeDate,
    shortDuration,
    truncateString,
    formatParents,
    getIcon
} from './utils';

document.addEventListener('DOMContentLoaded', () => {
    const searchInput = document.getElementById('search-input') as HTMLInputElement;
    const resultsContainer = document.getElementById('results-container');
    const resultsCount = document.getElementById('results-count');
    const sortBy = document.getElementById('sort-by') as HTMLSelectElement;
    const sortReverseBtn = document.getElementById('sort-reverse-btn');
    const limitInput = document.getElementById('limit') as HTMLInputElement;
    const limitAll = document.getElementById('limit-all') as HTMLInputElement;
    const viewGrid = document.getElementById('view-grid');
    const viewDetails = document.getElementById('view-details');
    const categoryList = document.getElementById('category-list');
    const toast = document.getElementById('toast');

    const paginationContainer = document.getElementById('pagination-container');
    const prevPageBtn = document.getElementById('prev-page');
    const nextPageBtn = document.getElementById('next-page');
    const pageInfo = document.getElementById('page-info');

    const menuToggle = document.getElementById('menu-toggle');
    const sidebarOverlay = document.getElementById('sidebar-overlay');
    const sidebar = document.getElementById('sidebar');

    const duBtn = document.getElementById('du-btn');
    const captionsBtn = document.getElementById('captions-btn') as HTMLInputElement;
    const curationBtn = document.getElementById('curation-btn');
    const channelSurfBtn = document.getElementById('channel-surf-btn');
    const filterCaptions = document.getElementById('filter-captions') as HTMLInputElement;

    const pipPlayer = document.getElementById('pip-player') as HTMLVideoElement;
    const pipLoading = document.getElementById('pip-loading');
    const pipViewer = document.getElementById('media-viewer');
    const pipTitle = document.getElementById('media-title');
    if (pipTitle) {
        pipTitle.onclick = () => {
            const range = document.createRange();
            range.selectNodeContents(pipTitle);
            const selection = window.getSelection();
            selection?.removeAllRanges();
            selection?.addRange(range);
        };
    }
    const searchSuggestions = document.getElementById('search-suggestions');

    const viewGroup = document.getElementById('view-group') as HTMLSelectElement;

    const historyInProgressBtn = document.getElementById('history-in-progress-btn');
    const historyUnplayedBtn = document.getElementById('history-unplayed-btn');
    const historyCompletedBtn = document.getElementById('history-completed-btn');

    const allMediaBtn = document.getElementById('all-media-btn');
    const trashBtn = document.getElementById('trash-btn');

    // Percentile Sliders (Initialized via Sliders module)

    const pipSpeedBtn = document.getElementById('pip-speed');
    const pipSpeedMenu = document.getElementById('pip-speed-menu');

    const filterBrowseCol = document.getElementById('filter-browse-col') as HTMLSelectElement;
    const filterBrowseVal = document.getElementById('filter-browse-val') as HTMLSelectElement;
    const filterBrowseValContainer = document.getElementById('filter-browse-val-container');

    const settingSearchType = document.getElementById('setting-search-type') as HTMLInputElement;
    const settingShowLanguages = document.getElementById('setting-show-languages') as HTMLInputElement;

    let currentMedia = [];
    let allDatabases = [];
    let searchAbortController = null;
    let suggestionAbortController = null;
    let selectedSuggestionIndex = -1;

    // --- State Management ---

    initSliders(performSearch);
    initComplexSorting();

    // Listen for complex sort applied event
    window.addEventListener('complex-sort-applied', () => {
        // Update sort-by dropdown to reflect custom or preset selection
        const sortByEl = document.getElementById('sort-by') as HTMLSelectElement;
        if (sortByEl && state.filters.sort) {
            sortByEl.value = state.filters.sort;
        }

        if (state.page === 'du') {
            window.location.reload();
        } else {
            state.currentPage = 1;
            performSearch();
        }
    });

    // Initialize UI from state
    (document.getElementById('setting-player') as HTMLSelectElement).value = state.player;
    (document.getElementById('setting-language') as HTMLSelectElement).value = state.language;
    (document.getElementById('setting-theme') as HTMLSelectElement).value = state.theme;
    (document.getElementById('setting-post-playback') as HTMLSelectElement).value = state.postPlaybackAction;
    (document.getElementById('setting-default-view') as HTMLSelectElement).value = state.defaultView;

    (document.getElementById('setting-autoplay') as HTMLInputElement).checked = state.autoplay;
    const settingEnableQueue = document.getElementById('setting-enable-queue') as HTMLInputElement;
    if (settingEnableQueue) settingEnableQueue.checked = state.enableQueue;

    const settingImageAutoplay = document.getElementById('setting-image-autoplay') as HTMLInputElement;
    if (settingImageAutoplay) settingImageAutoplay.checked = state.imageAutoplay;
    (document.getElementById('setting-local-resume') as HTMLInputElement).checked = state.localResume;
    (document.getElementById('setting-default-video-rate') as HTMLSelectElement).value = state.defaultVideoRate.toString().toString();
    (document.getElementById('setting-default-audio-rate') as HTMLSelectElement).value = state.defaultAudioRate.toString().toString();

    const settingShowPipSpeed = document.getElementById('setting-show-pip-speed') as HTMLInputElement;
    const settingShowPipSurf = document.getElementById('setting-show-pip-surf') as HTMLInputElement;
    const settingShowPipStream = document.getElementById('setting-show-pip-stream') as HTMLInputElement;

    if (settingShowPipSpeed) {
        settingShowPipSpeed.checked = state.showPipSpeed;
        settingShowPipSpeed.onchange = (e) => {
            state.showPipSpeed = (e.target as HTMLInputElement).checked;
            localStorage.setItem('disco-show-pip-speed', String(state.showPipSpeed.toString()));
            updatePipVisibility();
        };
    }
    if (settingShowPipSurf) {
        settingShowPipSurf.checked = state.showPipSurf;
        settingShowPipSurf.onchange = (e) => {
            state.showPipSurf = (e.target as HTMLInputElement).checked;
            localStorage.setItem('disco-show-pip-surf', String(state.showPipSurf.toString()));
            updatePipVisibility();
        };
    }
    if (settingShowPipStream) {
        settingShowPipStream.checked = state.showPipStream;
        settingShowPipStream.onchange = (e) => {
            state.showPipStream = (e.target as HTMLInputElement).checked;
            localStorage.setItem('disco-show-pip-stream', String(state.showPipStream.toString()));
            updatePipVisibility();
        };
    }

    updatePipVisibility();

    (document.getElementById('setting-slideshow-delay') as HTMLInputElement).value = state.slideshowDelay.toString().toString();
    const settingAutoLoopMax = document.getElementById('setting-auto-loop-max') as HTMLInputElement;
    if (settingAutoLoopMax) {
        (settingAutoLoopMax as HTMLInputElement).value = state.autoLoopMaxDuration.toString().toString();
        (settingAutoLoopMax as HTMLElement).onchange = (e) => {
            state.autoLoopMaxDuration = parseInt((e.target as HTMLInputElement).value) || 0;
            localStorage.setItem('disco-auto-loop-max-duration', String(state.autoLoopMaxDuration.toString()));
        };
    }
    if (limitInput) limitInput.value = state.filters.limit.toString();
    if (limitAll) limitAll.checked = state.filters.all;
    const initialUnplayedEl = document.getElementById('filter-unplayed') as HTMLInputElement;
    if (initialUnplayedEl) initialUnplayedEl.checked = state.filters.unplayed;
    if (filterCaptions) {
        filterCaptions.checked = state.filters.captions;
        filterCaptions.onchange = (e) => {
            state.filters.captions = (e.target as HTMLInputElement).checked;
            localStorage.setItem('disco-captions', String(state.filters.captions.toString()));
            performSearch();
        };
    }

    // Search type toggle (FTS vs Substring)
    if (settingSearchType) {
        settingSearchType.checked = state.filters.searchType === 'fts';
        settingSearchType.onchange = (e) => {
            state.filters.searchType = (e.target as HTMLInputElement).checked ? 'fts' : 'substring';
            localStorage.setItem('disco-search-type', String(state.filters.searchType));
            performSearch();
        };
    }

    // Language filter visibility toggle
    if (settingShowLanguages) {
        const showLanguages = localStorage.getItem('disco-show-languages') === 'true';
        settingShowLanguages.checked = showLanguages;
        if (showLanguages) {
            document.body.classList.add('advanced-enabled');
        }
        settingShowLanguages.onchange = (e) => {
            const show = (e.target as HTMLInputElement).checked;
            localStorage.setItem('disco-show-languages', String(show.toString()));
            if (show) {
                document.body.classList.add('advanced-enabled');
            } else {
                document.body.classList.remove('advanced-enabled');
            }
        };
    }

    async function playRandomMedia(maxRetries = 3) {
        let retries = 0;

        while (retries < maxRetries) {
            try {
                // Determine filter type based on current media
                let type = '';
                if (state.playback.item && state.playback.item.type) {
                    const currentType = state.playback.item.type;
                    if (currentType.startsWith('video')) type = 'video';
                    else if (currentType.startsWith('audio')) type = 'audio';
                    else if (currentType.startsWith('image')) type = 'image';
                    else if (currentType.startsWith('text') || currentType.includes('pdf') || currentType.includes('epub') || currentType.includes('mobi')) type = 'text';
                }

                const params = new URLSearchParams();
                if (type) params.append('type', String(type));

                const resp = await fetchAPI(`/api/random-clip?${params.toString()}`);
                if (!resp.ok) {
                    if (resp.status === 403) {
                        showToast('Access Denied', '🚫');
                        return;
                    }
                    if (resp.status === 404) {
                        showToast(`No more ${type || 'media'} found.`, 'ℹ️');
                        return;
                    }
                    // For 415 or other errors, retry with different media
                    retries++;
                    if (retries >= maxRetries) {
                        throw new Error('Failed to fetch playable media after ' + maxRetries + ' attempts');
                    }
                    continue;
                }
                const data = await resp.json();
                if (!data || !data.path) {
                    showToast(`No more ${type || 'media'} found.`, 'ℹ️');
                    return;
                }

                // Open in PiP
                await openActivePlayer(data, true);

                // Seek to the random start time
                const media = pipViewer.querySelector('video, audio');
                if (media && data.start !== undefined) {
                    (media as HTMLMediaElement).currentTime = data.start;
                }
                return; // Success, exit loop
            } catch (err) {
                // Check if it's a media loading error (415) - retry
                if (err.message && err.message.includes('415')) {
                    retries++;
                    if (retries >= maxRetries) {
                        console.error('Failed to play random media after retries:', err);
                        errorToast(err as any, 'Failed to play random media');
                        return;
                    }
                    continue;
                }
                console.error('Failed to play random media:', err);
                errorToast(err as any, 'Failed to play random media');
                return;
            }
        }
    }

    if (channelSurfBtn) {
        channelSurfBtn.onclick = () => {
            playRandomMedia();
        };
    }

    const settingDefaultVideoRate = document.getElementById('setting-default-video-rate');
    if (settingDefaultVideoRate) {
        settingDefaultVideoRate.onchange = (e) => {
            state.defaultVideoRate = parseFloat((e.target as HTMLInputElement).value);
            localStorage.setItem('disco-default-video-rate', String(state.defaultVideoRate.toString()));
            if (state.playback.item && state.playback.item.type.includes('video')) {
                setPlaybackRate(state.defaultVideoRate);
            }
        };
    }

    const settingDefaultAudioRate = document.getElementById('setting-default-audio-rate');
    if (settingDefaultAudioRate) {
        settingDefaultAudioRate.onchange = (e) => {
            state.defaultAudioRate = parseFloat((e.target as HTMLInputElement).value);
            localStorage.setItem('disco-default-audio-rate', String(state.defaultAudioRate.toString()));
            if (state.playback.item && state.playback.item.type.includes('audio')) {
                setPlaybackRate(state.defaultAudioRate);
            }
        };
    }

    const settingSlideshowDelay = document.getElementById('setting-slideshow-delay');
    if (settingSlideshowDelay) {
        settingSlideshowDelay.onchange = (e) => {
            state.slideshowDelay = parseInt((e.target as HTMLInputElement).value);
            localStorage.setItem('disco-slideshow-delay', String(state.slideshowDelay.toString()));
            if (state.playback.slideshowTimer) {
                stopSlideshow();
                startSlideshow();
            }
        };
    }

    if (sortBy) {
        // Set sort-by value, using "custom" if we have custom sort configuration
        if (state.filters.customSortFields && state.filters.sort === 'custom') {
            sortBy.value = 'custom';
        } else {
            sortBy.value = state.filters.sort || 'default';
        }
    }
    if (sortReverseBtn && state.filters.reverse) sortReverseBtn.classList.add('active');

    if (viewGrid && viewDetails) {
        if (state.view === 'details') {
            viewDetails.classList.add('active');
            viewGrid.classList.remove('active');
        } else {
            viewGrid.classList.add('active');
            viewDetails.classList.remove('active');
        }
    }

    // --- Sidebar Persistence ---
    function initSidebarPersistence() {
        const details = document.querySelectorAll('#sidebar details');
        details.forEach(det => {
            const id = (det as HTMLDetailsElement).id;
            if (!id) return;

            // Restore
            if (state.sidebarState[id] !== undefined) {
                (det as HTMLDetailsElement).open = state.sidebarState[id];
            }

            // Listen
            det.addEventListener('toggle', () => {
                state.sidebarState[id] = (det as HTMLDetailsElement).open;
                localStorage.setItem('disco-sidebar-state', String(JSON.stringify(state.sidebarState)));
            });

            // Ctrl+click to toggle all
            const summary = det.querySelector('summary');
            if (summary) {
                (summary as HTMLElement).onclick = (e) => {
                    if (e.ctrlKey || e.metaKey) {
                        e.preventDefault();
                        const newState = !(det as HTMLDetailsElement).open;
                        document.querySelectorAll('#sidebar details').forEach(d => {
                            (d as HTMLDetailsElement).open = newState;
                            if ((d as HTMLDetailsElement).id) {
                                state.sidebarState[(d as HTMLDetailsElement).id] = newState;
                            }
                        });
                        localStorage.setItem('disco-sidebar-state', String(JSON.stringify(state.sidebarState)));
                    }
                };
            }
        });
    }

    async function fetchFilterBins(params) {
        try {
            const url = params ? `/api/filter-bins?${params.toString()}` : '/api/filter-bins';
            const resp = await fetchAPI(url);
            if (!resp.ok) throw new Error('Failed to fetch filter bins');
            const data = await resp.json();
            state.filterBins = {
                episodes: data.episodes || [],
                size: data.size || [],
                duration: data.duration || [],
                modified: data.modified || [],
                created: data.created || [],
                downloaded: data.downloaded || [],
                episodes_min: data.episodes_min !== undefined && data.episodes_min !== null ? data.episodes_min : 0,
                episodes_max: data.episodes_max !== undefined && data.episodes_max !== null ? data.episodes_max : 100,
                size_min: data.size_min !== undefined && data.size_min !== null ? data.size_min : 0,
                size_max: data.size_max !== undefined && data.size_max !== null ? data.size_max : (100 * 1024 * 1024),
                duration_min: data.duration_min !== undefined && data.duration_min !== null ? data.duration_min : 0,
                duration_max: data.duration_max !== undefined && data.duration_max !== null ? data.duration_max : 3600,
                modified_min: data.modified_min !== undefined && data.modified_min !== null ? data.modified_min : 0,
                modified_max: data.modified_max !== undefined && data.modified_max !== null ? data.modified_max : 100,
                created_min: data.created_min !== undefined && data.created_min !== null ? data.created_min : 0,
                created_max: data.created_max !== undefined && data.created_max !== null ? data.created_max : 100,
                downloaded_min: data.downloaded_min !== undefined && data.downloaded_min !== null ? data.downloaded_min : 0,
                downloaded_max: data.downloaded_max !== undefined && data.downloaded_max !== null ? data.downloaded_max : 100,
                episodes_percentiles: data.episodes_percentiles || [],
                size_percentiles: data.size_percentiles || [],
                duration_percentiles: data.duration_percentiles || [],
                modified_percentiles: data.modified_percentiles || [],
                created_percentiles: data.created_percentiles || [],
                downloaded_percentiles: data.downloaded_percentiles || []
            };

            updateSlidersFromAbsolute('episodes', 'episodes');
            updateSlidersFromAbsolute('size', 'sizes');
            updateSlidersFromAbsolute('duration', 'durations');
            updateSlidersFromAbsolute('modified', 'modified');
            updateSlidersFromAbsolute('created', 'created');
            updateSlidersFromAbsolute('downloaded', 'downloaded');

            renderFilterBins();
            updateSliderLabels();
        } catch (err) {
            console.error('Failed to fetch filter bins', err);
        }
    }

    function renderMediaTypeList() {
        const container = document.getElementById('media-type-list');
        if (!container) return;

        const types = [
            { id: 'video', label: 'Video', icon: '🎬' },
            { id: 'audio', label: 'Audio', icon: '🎵' },
            { id: 'text', label: 'Text', icon: '📖' },
            { id: 'image', label: 'Image', icon: '🖼️' }
        ];

        const newHtml = types.map(t => `
            <button class="category-btn ${state.filters.types.includes(t.id) ? 'active' : ''}" data-type="${t.id}">
                ${t.icon} ${t.label}
            </button>
        `).join('');

        container.innerHTML = newHtml;

        container.querySelectorAll('button').forEach(btn => {
            (btn as any).onclick = () => {
                const type = (btn as any).dataset.type;
                if (state.filters.types.includes(type)) {
                    state.filters.types = [];
                } else {
                    state.filters.types = [type];
                }
                localStorage.setItem('disco-types', String(JSON.stringify(state.filters.types)));
                updateNavActiveStates();
                performSearch();
            };
        });
    }

    function renderFilterBins() {
        // Sliders are static in index.html, no need to render bins here.
    }

    function resetSidebar() {
        const details = document.querySelectorAll('#sidebar details');
        state.sidebarState = {};
        state.filters.categories = [];
        state.filters.genre = '';
        state.filters.languages = [];
        state.filters.ratings = [];
        state.filters.playlist = null;
        state.filters.sizes = [];
        state.filters.durations = [];
        state.filters.episodes = [];
        state.filters.modified = [];
        state.filters.created = [];
        state.filters.downloaded = [];
        state.filters.types = [];
        state.filters.search = '';
        state.filters.unplayed = false;
        state.filters.unfinished = false;
        state.filters.completed = false;
        if (searchInput) (searchInput as HTMLInputElement).value = '';

        details.forEach(det => {
            const id = (det as HTMLDetailsElement).id;
            if (!id) return;
            (det as HTMLDetailsElement).open = false;
            state.sidebarState[id] = false;
        });

        localStorage.setItem('disco-sidebar-state', String(JSON.stringify(state.sidebarState)));
        clearAllFilters();
        updateNavActiveStates();
        resetSliders();
    }

    function resetFilters() {
        // Clear all filter-related state
        state.filters.categories = [];
        state.filters.genre = '';
        state.filters.languages = [];
        state.filters.ratings = [];
        state.filters.sizes = [];
        state.filters.durations = [];
        state.filters.episodes = [];
        state.filters.modified = [];
        state.filters.created = [];
        state.filters.downloaded = [];
        state.filters.types = [];
        state.filters.unplayed = false;
        state.filters.unfinished = false;
        state.filters.completed = false;

        // Clear sidebar filter UI
        document.querySelectorAll('#sidebar .category-btn.active').forEach(btn => {
            btn.classList.remove('active');
        });

        // Reset sliders to default
        resetSliders();

        // Save to localStorage
        clearAllFilters();

        updateNavActiveStates();
        performSearch();
    }

    // --- Modal Management ---
    function openModal(id) {
        state.activeModal = id;
        document.getElementById(id).classList.remove('hidden');
        syncUrl();
    }

    function closeModal(id) {
        if (state.activeModal === id) {
            state.activeModal = null;
        }
        document.getElementById(id).classList.add('hidden');
        syncUrl();
    }

    // --- LocalStorage Helpers ---
    function getLocalStorageItem(key, defaultValue = null) {
        const item = localStorage.getItem(key);
        if (item === null || item === undefined) return defaultValue;
        try {
            return JSON.parse(item);
        } catch {
            return defaultValue;
        }
    }

    function setLocalStorageItem(key, value) {
        localStorage.setItem(key, String(JSON.stringify(value)));
    }

    function clearAllFilters() {
        const filterKeys = [
            'disco-filter-categories',
            'disco-filter-ratings',
            'disco-filter-sizes',
            'disco-filter-durations',
            'disco-filter-episodes',
            'disco-filter-modified',
            'disco-filter-created',
            'disco-filter-downloaded',
            'disco-types'
        ];
        filterKeys.forEach(key => localStorage.setItem(key, String('[]')));
    }

    /**
     * Get the element for the currently active viewer (PiP or document container).
     * @returns {HTMLElement|null} The active viewer element or null if none is visible
     */
    function getActiveViewerElement() {
        const docModal = document.getElementById('document-modal');
        if (!docModal.classList.contains('hidden')) {
            return document.getElementById('document-container');
        }
        if (!pipPlayer.classList.contains('hidden')) {
            return pipViewer;
        }
        return null;
    }

    /**
     * Close whichever player is currently active (PiP or Document Modal).
     * This is the unified interface for closing the active player.
     * @param {boolean} skipSync - Whether to skip syncUrl (e.g. if called from readUrl)
     * @param {boolean} keepFullscreen - Whether to stay in fullscreen if active
     * @returns {boolean} true if a player was closed, false if none was open
     */
    async function closeActivePlayer(skipSync = false, keepFullscreen = false) {
        let closed = false;

        // Close PiP if open
        if (!pipPlayer.classList.contains('hidden')) {
            stopSlideshow();

            if (state.playback.skipTimeout) {
                clearTimeout(state.playback.skipTimeout);
                state.playback.skipTimeout = null;
            }

            if (state.playback.hlsInstance) {
                state.playback.hlsInstance.destroy();
                state.playback.hlsInstance = null;
            }
            const media = pipViewer.querySelector('video, audio');
            if (media) {
                (media as HTMLMediaElement).onerror = null;
                (media as HTMLMediaElement).onended = null;
                (media as HTMLMediaElement).onpause = null;
                (media as HTMLMediaElement).ontimeupdate = null;
                (media as HTMLMediaElement).onvolumechange = null;
                (media as HTMLMediaElement).oncanplay = null;
                (media as HTMLMediaElement).pause();
                media.removeAttribute('src');
                (media as HTMLMediaElement).load();
            }
            pipViewer.innerHTML = '';
            pipPlayer.classList.add('hidden');
            document.body.classList.remove('has-pip');

            // Exit fullscreen if active
            if (document.fullscreenElement && !keepFullscreen) {
                document.exitFullscreen().catch(err => {
                    console.error('Failed to exit fullscreen:', err);
                });
            }

            // Reset mode to default preference
            state.playerMode = state.defaultView;
            closed = true;
        }

        // Close Document Modal if open
        const docModal = document.getElementById('document-modal');
        if (!docModal.classList.contains('hidden')) {
            // Save document reading progress before closing
            const docContainer = document.getElementById('document-container');
            const iframe = docContainer?.querySelector('iframe');
            if (iframe && state.playback.item) {
                try {
                    const iframeDoc = iframe.contentDocument || iframe.contentWindow?.document;
                    const iframeWin = iframe.contentWindow;

                    if (iframeDoc && iframeWin) {
                        const scrollTop = iframeWin.scrollY || iframeDoc.documentElement.scrollTop || iframeDoc.body.scrollTop;
                        const scrollHeight = iframeDoc.documentElement.scrollHeight || iframeDoc.body.scrollHeight;
                        const clientHeight = iframeWin.innerHeight || iframeDoc.documentElement.clientHeight;

                        if (scrollHeight > clientHeight) {
                            const scrollPercent = scrollTop / (scrollHeight - clientHeight);
                            saveDocumentProgress(state.playback.item.path, scrollPercent);
                        }
                    }
                } catch (e) {
                    // Cross-origin or other error - silently fail
                }

                // Clear progress timer
                if (documentProgressTimer !== null) {
                    window.clearInterval(documentProgressTimer);
                    documentProgressTimer = null;
                }
            }

            docModal.classList.add('hidden');
            if (state.activeModal === 'document-modal') {
                state.activeModal = null;
            }
            closed = true;
        }

        // Clear playback state
        state.playback.item = null;

        // Update Now Playing button visibility
        updateNowPlayingButton();

        if (closed && !skipSync) {
            syncUrl();
        }

        return closed;
    }

    /**
     * Open the appropriate player for the given media item.
     * Documents open in the document modal, all other media opens in PiP.
     * @param {Object} item - The media item to play
     * @param {boolean} isNewSession - Whether this is a new explicit user request
     * @param {boolean} skipSync - Whether to skip syncUrl
     */
    function openActivePlayer(item, isNewSession = false, skipSync = false, queueIndex = -1, keepFullscreen = false) {
        const type = item.type || "";
        const isDocument = type === 'text' || type.includes('pdf') || type.includes('epub') || type.includes('mobi');

        // Close any existing player before opening new one
        closeActivePlayer(true, keepFullscreen);

        state.playback.item = item;
        state.playback.queueIndex = queueIndex;

        // Open in appropriate viewer
        if (isDocument) {
            state.activeModal = 'document-modal';
            openInDocumentViewer(item);
        } else {
            openInPiP(item, isNewSession);
        }

        if (!skipSync) {
            syncUrl();
        }
    }

    // --- Navigation & URL ---
    function getBinQueryParam(bin) {
        if (bin.value === '@p') return `p${bin.min}-${bin.max}`;
        if (bin.value === '@abs') return `+${bin.min},-${bin.max}`;
        if (bin.value !== undefined) return bin.value.toString();
        if (bin.min !== undefined && bin.max !== undefined) return `${bin.min}-${bin.max}`;
        if (bin.min !== undefined) return `+${bin.min}`;
        if (bin.max !== undefined) return `-${bin.max}`;
        return "";
    }

    function appendFilterParams(params) {
        if (state.filters.search) params.append('search', String(state.filters.search));
        state.filters.categories.forEach(c => params.append('category', String(c)));
        if (state.filters.genre) params.append('genre', String(state.filters.genre));
        state.filters.languages.forEach(l => params.append('language', String(l)));
        state.filters.ratings.forEach(r => params.append('rating', String(r)));
        if (state.filters.unplayed) params.append('unplayed', String('true'));
        if (state.filters.unfinished) params.append('unfinished', String('true'));
        if (state.filters.completed) params.append('completed', String('true'));
        if (state.filters.min_score) params.append('min_score', String(state.filters.min_score));
        if (state.filters.max_score) params.append('max_score', String(state.filters.max_score));

        state.filters.episodes.forEach(b => params.append('episodes', String(getBinQueryParam(b))));
        state.filters.sizes.forEach(b => params.append('size', String(getBinQueryParam(b))));
        state.filters.durations.forEach(b => params.append('duration', String(getBinQueryParam(b))));
        state.filters.modified.forEach(b => params.append('modified', String(getBinQueryParam(b))));
        state.filters.created.forEach(b => params.append('created', String(getBinQueryParam(b))));
        state.filters.downloaded.forEach(b => params.append('downloaded', String(getBinQueryParam(b))));

        state.filters.types.forEach(t => params.append('type', String(t)));

        // Add database filter (send included DBs, not excluded)
        if (state.databases && state.databases.length > 0) {
            const includedDbs = state.databases.filter(db => !state.filters.excludedDbs.includes(db));
            includedDbs.forEach(db => params.append('db', String(db)));
        }

        // Add sort parameters
        if (state.filters.sort && state.filters.sort !== 'default') {
            if (state.filters.sort === 'custom' && state.filters.customSortFields) {
                // Use complex sorting
                params.append('sort_fields', state.filters.customSortFields);
            } else {
                params.append('sort', String(state.filters.sort));
            }
        }
        if (state.filters.reverse) {
            params.append('reverse', String('true'));
        }
    }

    const isMobileOrFullscreen = () => window.innerWidth <= 768 || !!document.fullscreenElement;

    function syncUrl() {
        const params = new URLSearchParams();
        if (state.page === 'trash') {
            params.set('mode', String('trash'));
        } else if (state.page === 'history') {
            params.set('mode', String('history'));
        } else if (state.page === 'playlist' && state.filters.playlist) {
            params.set('mode', String('playlist'));
            params.set('title', String(state.filters.playlist));
        } else if (state.page === 'du') {
            params.set('mode', String('du'));
            if (state.duPath) params.set('path', String(state.duPath));
        } else if (state.page === 'curation') {
            params.set('mode', String('curation'));
        } else if (state.page === 'captions') {
            params.set('mode', String('captions'));
        }

        // History filters apply across all modes (like Media Type filters)
        if (state.filters.unplayed || state.filters.unfinished || state.filters.completed) {
            if (state.filters.unfinished) params.set('history', String('in-progress'));
            else if (state.filters.unplayed) params.set('history', String('unplayed'));
            else if (state.filters.completed) params.set('history', String('completed'));
        }

        if (state.page !== 'curation') {
            // Include search and sidebar filters for non-curation modes
            if (state.page !== 'history') {
                // Categories, genre, ratings don't apply in history mode
                state.filters.categories.forEach(c => params.append('category', String(c)));
                if (state.filters.genre) params.set('genre', String(state.filters.genre));
                state.filters.ratings.forEach(r => params.append('rating', String(r)));
            }
            if (state.filters.search) params.set('search', String(state.filters.search));
            if (state.filters.min_score) params.set('min_score', String(state.filters.min_score));
            if (state.filters.max_score) params.set('max_score', String(state.filters.max_score));
            if (state.filters.searchType && state.filters.searchType !== 'fts') {
                params.set('search_type', String(state.filters.searchType));
            }

            state.filters.episodes.forEach(b => params.append('episodes', String(getBinQueryParam(b))));
            state.filters.sizes.forEach(b => params.append('size', String(getBinQueryParam(b))));
            state.filters.durations.forEach(b => params.append('duration', String(getBinQueryParam(b))));
        }

        // Media Type filters apply across all modes except DU, trash, and playlist
        if (state.page !== 'du' && state.page !== 'trash' && state.page !== 'playlist') {
            state.filters.types.forEach(t => params.append('type', String(t)));
        }

        if (state.currentPage > 1) {
            params.set('p', String(state.currentPage.toString()));
        }

        // Modal and playback states in URL (primarily for mobile back button)
        if (isMobileOrFullscreen()) {
            if (state.activeModal) {
                params.set('modal', String(state.activeModal));
            }
            if (state.playback.item) {
                params.set('playing', String(state.playback.item.path));
            }
        }

        const paramString = params.toString();
        const newHash = paramString ? `#${paramString}` : '';

        if (window.location.hash !== newHash) {
            // Use pushState for DU navigation, page changes, modals, and playback to support back button
            // Use replaceState for filter changes to avoid history spam
            const isDUPathChange = state.page === 'du' && !window.location.hash.includes('path=');
            const isPageChange = !window.location.hash.includes(`mode=${state.page}`);
            const isModalChange = params.has('modal') !== window.location.hash.includes('modal=');
            const isPlayingChange = params.has('playing') !== window.location.hash.includes('playing=');

            if (isPageChange || isDUPathChange || isModalChange || isPlayingChange) {
                window.history.pushState(state.filters, '', window.location.pathname + newHash);
            } else {
                window.history.replaceState(state.filters, '', window.location.pathname + newHash);
            }
            lastHandledHash = window.location.hash;
        }
    }

    function readUrl(openSections = false) {
        // Support both hash and search params, preferring hash for the new system
        const hash = window.location.hash.substring(1);
        const params = hash ? new URLSearchParams(hash) : new URLSearchParams(window.location.search);
        const mode = params.get('mode');

        const pageParam = params.get('p');
        state.currentPage = pageParam ? parseInt(pageParam) : 1;

        // Read history filter from URL - applies across all modes
        const historyFilter = params.get('history');
        state.filters.unfinished = historyFilter === 'in-progress';
        state.filters.unplayed = historyFilter === 'unplayed';
        state.filters.completed = historyFilter === 'completed';

        // Modals and Playback
        const modalParam = params.get('modal');
        const playingParam = params.get('playing');

        // Close sidebar if it's open but not in URL
        if (state.activeModal === 'mobile-sidebar' && modalParam !== 'mobile-sidebar') {
            sidebar.classList.remove('mobile-open');
            sidebarOverlay.classList.add('hidden');
            state.activeModal = null;
        }

        if (modalParam) {
            if (state.activeModal !== modalParam) {
                state.activeModal = modalParam;
                if (modalParam === 'mobile-sidebar') {
                    sidebar.classList.add('mobile-open');
                    sidebarOverlay.classList.remove('hidden');
                } else {
                    const el = document.getElementById(modalParam);
                    if (el) el.classList.remove('hidden');
                }
            }
        } else if (state.activeModal) {
            const el = document.getElementById(state.activeModal);
            if (el) el.classList.add('hidden');
            state.activeModal = null;
        }

        if (playingParam) {
            if (!state.playback.item || state.playback.item.path !== playingParam) {
                // If we're already playing something, close it first
                const mediaItem = currentMedia.find(m => m.path === playingParam) || { path: playingParam };
                openActivePlayer(mediaItem, true, true);
            }
        } else if (state.playback.item) {
            closeActivePlayer(true);
        }

        if (mode === 'trash') {
            state.page = 'trash';
            state.filters.categories = [];
            state.filters.languages = [];
            state.filters.ratings = [];
            // Read filter bins for trash mode
            state.filters.episodes = params.getAll('episodes').map(parseFilterBin);
            state.filters.sizes = params.getAll('size').map(parseFilterBin);
            state.filters.durations = params.getAll('duration').map(parseFilterBin);
        } else if (mode === 'history') {
            state.page = 'history';
            state.filters.categories = [];
            state.filters.languages = [];
            state.filters.ratings = [];
        } else if (mode === 'playlist') {
            state.page = 'playlist';
            state.filters.playlist = params.get('title');
            state.filters.categories = [];
            state.filters.languages = [];
            state.filters.ratings = [];
        } else if (mode === 'du') {
            state.page = 'du';
            state.duPath = params.get('path') || '';
            state.filters.categories = [];
            state.filters.languages = [];
            state.filters.ratings = [];
            // Read filter bins for DU mode
            state.filters.episodes = params.getAll('episodes').map(parseFilterBin);
            state.filters.sizes = params.getAll('size').map(parseFilterBin);
            state.filters.durations = params.getAll('duration').map(parseFilterBin);
        } else if (mode === 'curation') {
            state.page = 'curation';
            state.filters.categories = [];
            state.filters.languages = [];
            state.filters.ratings = [];
        } else if (mode === 'captions') {
            state.page = 'captions';
            state.filters.categories = [];
            state.filters.languages = [];
            state.filters.ratings = [];
            // Read filter bins for captions mode
            state.filters.episodes = params.getAll('episodes').map(parseFilterBin);
            state.filters.sizes = params.getAll('size').map(parseFilterBin);
            state.filters.durations = params.getAll('duration').map(parseFilterBin);
        } else {
            state.page = 'search';
            state.filters.types = params.getAll('type');
            if (state.filters.types.length === 0) {
                // Default fallback if not in URL
                state.filters.types = JSON.parse(localStorage.getItem('disco-types') || '[]');
            }
            state.filters.categories = params.getAll('category');
            state.filters.genre = params.get('genre') || '';
            state.filters.languages = params.getAll('language');
            state.filters.ratings = params.getAll('rating');
            state.filters.search = params.get('search') || '';
            state.filters.all = params.get('all') === 'true';
            state.filters.min_score = params.get('min_score') || '';
            state.filters.max_score = params.get('max_score') || '';
            state.filters.searchType = (params.get('search_type') || 'fts') as 'fts' | 'substring';

            state.filters.episodes = params.getAll('episodes').map(parseFilterBin);
            state.filters.sizes = params.getAll('size').map(parseFilterBin);
            state.filters.durations = params.getAll('duration').map(parseFilterBin);

            if (openSections) {
                if (state.filters.categories.length > 0) state.sidebarState['details-categories'] = true;
                if (state.filters.genre) state.sidebarState['details-browse'] = true;
                if (state.filters.languages.length > 0) state.sidebarState['details-languages'] = true;
                if (state.filters.ratings.length > 0) state.sidebarState['details-ratings'] = true;
                if (state.filters.episodes.length > 0) state.sidebarState['details-episodes'] = true;
                if (state.filters.sizes.length > 0) state.sidebarState['details-size'] = true;
                if (state.filters.durations.length > 0) state.sidebarState['details-duration'] = true;
            }

            // Restoration of complex filters from URL is tricky since we only have labels in bins
            // For now, we rely on state persistence in localStorage which is already happening
            // But we can try to parse them if we want to support sharing URLs

            // Update search type toggle UI
            if (settingSearchType) {
                settingSearchType.checked = state.filters.searchType === 'fts';
            }

            if (searchInput) (searchInput as HTMLInputElement).value = state.filters.search;

            if (state.filters.genre && filterBrowseCol) {
                filterBrowseCol.value = 'genre';
                filterBrowseCol.onchange(new Event('change'));
            } else if (state.filters.categories.length > 0 && filterBrowseCol) {
                filterBrowseCol.value = 'category';
                filterBrowseCol.onchange(new Event('change'));
            } else if (filterBrowseCol) {
                filterBrowseCol.value = '';
                filterBrowseValContainer.classList.add('hidden');
            }
        }

        updateSlidersFromAbsolute('episodes', 'episodes');
        updateSlidersFromAbsolute('size', 'sizes');
        updateSlidersFromAbsolute('duration', 'durations');
    }

    // Helper function to parse filter bin values
    function parseFilterBin(val) {
        if (val.startsWith('p')) {
            const [min, max] = val.substring(1).split('-').map(Number);
            return { label: `${min}-${max}%`, value: '@p', min, max };
        }
        if (val.includes('-')) {
            const [min, max] = val.split('-').map(Number);
            return { label: val, min, max };
        }
        if (val.startsWith('+')) return { label: val, min: Number(val.substring(1)) };
        if (val.startsWith('-')) return { label: val, max: Number(val.substring(1)) };
        return { label: val, value: Number(val) };
    }

    let lastHandledHash = window.location.hash;

    const onUrlChange = () => {
        const currentHash = window.location.hash;
        lastHandledHash = currentHash;

        readUrl();
        updateNavActiveStates();
        if (state.page === 'trash') {
            fetchTrash();
        } else if (state.page === 'history') {
            fetchHistory();
        } else if (state.page === 'playlist' && state.filters.playlist) {
            fetchPlaylistItems(state.filters.playlist);
        } else if (state.page === 'du') {
            fetchDU(state.duPath);
        } else if (state.page === 'episodes') {
            fetchEpisodes();
        } else if (state.page === 'curation') {
            fetchCuration();
        } else {
            performSearch();
        }
        renderCategoryList();
        renderRatingList();
        renderPlaylistList();
    };

    window.onpopstate = () => {
        onUrlChange();
    };

    window.onhashchange = () => {
        if (window.location.hash !== lastHandledHash) {
            onUrlChange();
        }
    };

    // --- API Calls ---
    async function fetchDatabases() {
        try {
            const resp = await fetchAPI('/api/databases');
            if (!resp.ok) throw new Error('Offline');
            const data = await resp.json();
            allDatabases = data.databases;
            state.databases = data.databases; // Store in state for filter params
            state.readOnly = data.read_only;
            state.dev = data.dev;

            renderDbSettingsList(allDatabases);
            if (state.dev) {
                setupAutoReload();
            }
            const trashBtn = document.getElementById('trash-btn');
            const newPlaylistBtn = document.getElementById('new-playlist-btn');
            if (state.readOnly) {
                if (newPlaylistBtn) newPlaylistBtn.classList.add('hidden');
                if (trashBtn) trashBtn.classList.add('hidden');
            } else {
                if (trashBtn) trashBtn.classList.remove('hidden');
            }
        } catch (err) {
            console.error('Failed to fetch databases', err);
        }
    }

    async function fetchSuggestions(path) {
        if (suggestionAbortController) suggestionAbortController.abort();
        suggestionAbortController = new AbortController();

        let apiURL = `/api/ls?path=${encodeURIComponent(path)}`;

        try {
            const resp = await fetchAPI(apiURL, {
                signal: suggestionAbortController.signal
            });
            if (!resp.ok) throw new Error('Failed to fetch suggestions');
            const data = await resp.json();
            state.lastSuggestions = data;
            renderSuggestions(data);
        } catch (err) {
            if (err.name === 'AbortError') return;
            console.error('Failed to fetch suggestions', err);
            searchSuggestions.classList.add('hidden');
        }
    }

    function renderSuggestions(items) {
        if (!items || items.length === 0) {
            searchSuggestions.classList.add('hidden');
            return;
        }

        const inputVal = (searchInput as HTMLInputElement).value;
        const isRelative = inputVal.startsWith('./');

        searchSuggestions.innerHTML = items.map((item, idx) => {
            let displayName = item.name;
            if (isRelative) {
                // If browsing relative paths, bold the part that matches what's after the last slash
                const lastSlash = inputVal.lastIndexOf('/');
                const query = inputVal.substring(lastSlash + 1).toLowerCase();

                // If the item name contains the query, highlight it.
                // We should only do this if we are actively filtering.
                if (query && item.name.toLowerCase().includes(query)) {
                    const qIdx = item.name.toLowerCase().indexOf(query);
                    displayName = `${item.name.substring(0, qIdx)}<b>${item.name.substring(qIdx, qIdx + query.length)}</b>${item.name.substring(qIdx + query.length)}`;
                }
            }
            displayName = truncateString(displayName);
            const displayPath = formatParents(item.path);

            return `
                <div class="suggestion-item" data-path="${item.path}" data-is-dir="${item.is_dir}" data-name="${item.name}" data-index="${idx}">
                    <div class="suggestion-icon">${item.is_dir ? '📁' : getIcon(item.type)}</div>
                    <div class="suggestion-info">
                        <div class="suggestion-name">${displayName}</div>
                        <div class="suggestion-path" title="${item.path}">${displayPath}</div>
                    </div>
                </div>
            `;
        }).join('');

        searchSuggestions.classList.remove('hidden');
        selectedSuggestionIndex = -1;

        searchSuggestions.querySelectorAll('.suggestion-item').forEach(el => {
            (el as HTMLElement).onclick = () => {
                const path = (el as HTMLElement).dataset.path;
                const isDir = (el as HTMLElement).dataset.isDir === 'true';
                if (isDir) {
                    if ((searchInput as HTMLInputElement).value.startsWith('./')) {
                        const newName = (el as HTMLElement).dataset.name;
                        const lastSlash = (searchInput as HTMLInputElement).value.lastIndexOf('/');
                        const newPath = (searchInput as HTMLInputElement).value.substring(0, lastSlash + 1) + newName + '/';
                        (searchInput as HTMLInputElement).value = newPath;
                    } else {
                        const newPath = path.endsWith('/') ? path : path + '/';
                        (searchInput as HTMLInputElement).value = newPath;
                    }
                    searchInput.focus();
                    fetchSuggestions((searchInput as HTMLInputElement).value);
                    performSearch();
                } else {
                    const item = state.lastSuggestions.find(s => s.path === path);
                    if (item) {
                        if (state.player === 'browser') {
                            openActivePlayer(item, true);
                        } else {
                            playMedia(item);
                        }
                        // Set searchbar to parent
                        const parts = path.split('/');
                        parts.pop();
                        const parent = parts.join('/') + '/';
                        (searchInput as HTMLInputElement).value = parent;
                        searchSuggestions.classList.add('hidden');
                        performSearch();
                    }
                }
            };
        });
    }

    async function fetchCategories() {
        try {
            const resp = await fetchAPI('/api/categories');
            if (!resp.ok) throw new Error('Failed to fetch categories');
            state.categories = await resp.json() || [];
            renderCategoryList();
        } catch (err) {
            console.error('Failed to fetch categories', err);
        }
    }

    async function fetchLanguages() {
        try {
            const resp = await fetchAPI('/api/languages');
            if (!resp.ok) throw new Error('Failed to fetch languages');
            state.languages = await resp.json() || [];
            renderLanguageList();
        } catch (err) {
            console.error('Failed to fetch languages', err);
        }
    }

    async function fetchMediaByPaths(paths) {
        if (!paths || paths.length === 0) return [];
        try {
            const p = new URLSearchParams();
            appendFilterParams(p);
            p.delete('unplayed');
            p.delete('unfinished');
            p.delete('completed');

            p.append('all', 'true');
            p.append('paths', paths.join(','));

            const resp = await fetchAPI(`/api/query?${p.toString()}`);
            if (!resp.ok) throw new Error('Failed to fetch media by paths');
            return await resp.json() || [];
        } catch (err) {
            console.error('fetchMediaByPaths failed:', err);
            return [];
        }
    }

    async function fetchGenres() {
        try {
            const resp = await fetchAPI('/api/genres');
            if (!resp.ok) throw new Error('Failed to fetch genres');
            state.genres = await resp.json() || [];
        } catch (err) {
            console.error('Failed to fetch genres', err);
        }
    }

    async function fetchRatings() {
        try {
            const resp = await fetchAPI('/api/ratings');
            if (!resp.ok) throw new Error('Failed to fetch ratings');
            state.ratings = await resp.json() || [];
            renderRatingList();
        } catch (err) {
            console.error('Failed to fetch ratings', err);
        }
    }

    async function fetchPlaylists() {
        try {
            const resp = await fetchAPI('/api/playlists');
            if (!resp.ok) throw new Error('Failed to fetch playlists');
            state.playlists = await resp.json() || [];
            renderPlaylistList();
        } catch (err) {
            console.error('Failed to fetch playlists', err);
        }
    }

    function renderPlaylistList() {
        const playlistList = document.getElementById('playlist-list');
        if (!playlistList) return;

        updateNavActiveStates();

        const playlists = state.playlists || [];
        playlistList.innerHTML = playlists.map(title => `
            <div class="category-btn playlist-drop-zone ${state.page === 'playlist' && state.filters.playlist === title ? 'active' : ''}"
                 data-title="${title}"
                 style="display: flex; justify-content: space-between; align-items: center;">
                <span class="playlist-name" data-title="${title}" style="flex: 1; cursor: pointer;">📁 ${title}</span>
                ${!state.readOnly ? `<button class="delete-playlist-btn" data-title="${title}" style="background: none; border: none; opacity: 0.5; cursor: pointer;">&times;</button>` : ''}
            </div>
        `).join('');

        playlistList.querySelectorAll('.playlist-drop-zone').forEach(zone => {
            (zone as HTMLElement).onclick = (e) => {
                // Ignore clicks on the delete button
                if ((e.target as HTMLElement).closest('.delete-playlist-btn')) return;

                const title = (zone as HTMLElement).dataset.title;
                state.page = 'playlist';
                state.filters.playlist = title;
                state.filters.categories = [];
                state.filters.genre = '';
                state.filters.ratings = [];

                document.querySelectorAll('.category-btn').forEach(b => b.classList.remove('active'));
                zone.classList.add('active');
                updateNavActiveStates();

                fetchPlaylistItems(title);
            };

            zone.addEventListener('dragenter', (e) => {
                e.preventDefault();
                zone.classList.add('drag-over');
            });

            zone.addEventListener('dragover', (e) => {
                e.preventDefault();
                (e as DragEvent).dataTransfer.dropEffect = 'copy';
            });

            zone.addEventListener('dragleave', (e) => {
                // Only remove if we're actually leaving the zone
                if (!zone.contains(((e as MouseEvent).relatedTarget as HTMLElement))) {
                    zone.classList.remove('drag-over');
                }
            });

            zone.addEventListener('drop', async (e) => {
                e.preventDefault();
                e.stopPropagation();

                zone.classList.remove('drag-over');

                const title = (zone as HTMLElement).dataset.title;
                const path = (e as DragEvent).dataTransfer.getData('text/plain');

                if (path && title) {
                    // Find the item if possible
                    const item = (state.draggedItem && state.draggedItem.path === path) ?
                        state.draggedItem : { path };

                    await addToPlaylist(title, item);
                    if (state.page === 'playlist' && state.filters.playlist === title) {
                        fetchPlaylistItems(title);
                    }
                }
                state.draggedItem = null;
                document.body.classList.remove('is-dragging');
            });
        });

        playlistList.querySelectorAll('.delete-playlist-btn').forEach(btn => {
            (btn as any).onclick = (e) => {
                e.stopPropagation();
                if (confirm('Delete this playlist?')) {
                    deletePlaylist((btn as any).dataset.title);
                }
            };
        });

        // Update Now Playing button visibility
        updateNowPlayingButton();
    }

    function updateNowPlayingButton() {
        // Update all media cards to show/hide playing indicator
        const currentPath = state.playback.item ? state.playback.item.path : null;
        document.querySelectorAll('.media-card').forEach(card => {
            const path = (card as HTMLElement).dataset.path;
            if (path && currentPath && path === currentPath) {
                card.classList.add('playing');
            } else {
                card.classList.remove('playing');
            }
        });

        renderQueue();
    }

    async function handlePlaylistReorder(draggedItem, newIndex) {
        if (!state.filters.playlist) return;

        try {
            const resp = await fetchAPI('/api/playlists/reorder', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    playlist_title: state.filters.playlist,
                    media_path: draggedItem.path,
                    new_index: newIndex
                })
            });
            if (!resp.ok) throw new Error('Reorder failed');
            showToast('Playlist reordered');
            fetchPlaylistItems(state.filters.playlist);
        } catch (err) {
            console.error('Reorder failed:', err);
            errorToast(err as any, 'Reorder failed');
        }
    }

    function filterPlaylistItems() {
        if (!state.playlistItems) return;

        // Server returns playlist items - no need for client-side filtering
        // Playlists are typically small, user-curated collections
        currentMedia = state.playlistItems;
        sortPlaylistItems();
        renderResults();
        syncUrl();
    }

    async function fetchPlaylistItems(title) {
        state.page = 'playlist';
        state.filters.playlist = title;
        state.filters.genre = '';
        syncUrl();

        const skeletonTimeout = setTimeout(() => {
            if (state.view === 'grid') showSkeletons();
        }, 150);

        try {
            const resp = await fetchAPI(`/api/playlists/items?title=${encodeURIComponent(title)}`);
            clearTimeout(skeletonTimeout);
            if (!resp.ok) throw new Error('Failed to fetch playlist items');
            state.playlistItems = await resp.json() || [];
            filterPlaylistItems();
        } catch (err) {
            clearTimeout(skeletonTimeout);
            console.error('Playlist items fetch failed:', err);
            errorToast(err as any, 'Failed to load playlist');
        }
    }

    async function deletePlaylist(title) {
        try {
            const resp = await fetchAPI(`/api/playlists?title=${encodeURIComponent(title)}`, { method: 'DELETE' });
            if (!resp.ok) throw new Error('Delete failed');
            showToast('Playlist deleted');
            fetchPlaylists();
            if (state.page === 'playlist' && state.filters.playlist === title) {
                state.page = 'search';
                performSearch();
            }
        } catch (err) {
            console.error('Delete playlist failed:', err);
        }
    }

    async function createPlaylist(title) {
        try {
            const resp = await fetchAPI('/api/playlists', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ title })
            });
            if (!resp.ok) {
                const errorText = await resp.text();
                throw new Error(`Create failed: ${errorText || resp.statusText}`);
            }
            showToast('Playlist created');
            fetchPlaylists();
        } catch (err) {
            console.error('Create playlist failed:', err);
            showToast(err.message);
        }
    }

    async function addToPlaylist(title, item) {
        const payload = {
            playlist_title: title,
            media_path: item.path
        };
        try {
            const resp = await fetchAPI('/api/playlists/items', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(payload)
            });
            if (!resp.ok) {
                const errorText = await resp.text();
                throw new Error(`Add failed: ${errorText || resp.statusText}`);
            }
            const filename = item.path.split('/').pop();
            showToast(`Added to ${title}\n\n${filename}`, '📁');
        } catch (err) {
            console.error('Add to playlist failed:', err, payload);
            showToast(err.message);
        }
    }

    async function removeFromPlaylist(title, item) {
        try {
            const resp = await fetchAPI('/api/playlists/items', {
                method: 'DELETE',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    playlist_title: title,
                    media_path: item.path
                })
            });
            if (!resp.ok) throw new Error('Remove failed');
            showToast('Removed from playlist');
            fetchPlaylistItems(title);
        } catch (err) {
            console.error('Remove from playlist failed:', err);
        }
    }

    async function updateTrackNumber(title, item, num) {
        try {
            const resp = await fetchAPI('/api/playlists/items', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    playlist_title: title,
                    media_path: item.path,
                    track_number: parseInt(num)
                })
            });
            if (!resp.ok) throw new Error('Update failed');
        } catch (err) {
            console.error('Update track number failed:', err);
        }
    }

    function renderRatingList() {
        const ratingList = document.getElementById('rating-list');
        if (!ratingList) return;

        updateNavActiveStates();

        // Always show 5, 4, 3, 2, 1, 0 stars
        const ratingsToShow = [5, 4, 3, 2, 1, 0];

        ratingList.innerHTML = ratingsToShow.map(val => {
            const r = state.ratings.find(x => x.rating === val) || { rating: val, count: 0 };
            const stars = r.rating === 0 ? '☆☆☆☆☆' : '⭐'.repeat(r.rating);
            const isActive = state.filters.ratings.includes(r.rating.toString());
            return `
                <button class="category-btn ${isActive ? 'active' : ''}" data-rating="${r.rating}">
                    ${stars} <small>(${r.count})</small>
                </button>
            `;
        }).join('');

        ratingList.querySelectorAll('.category-btn').forEach(btn => {
            (btn as any).onclick = (e) => {
                const rating = (btn as any).dataset.rating;
                if (state.page !== 'trash') state.page = 'search';

                const idx = state.filters.ratings.indexOf(rating);
                if (idx !== -1) {
                    state.filters.ratings.splice(idx, 1);
                } else {
                    state.filters.ratings.push(rating);
                }

                localStorage.setItem('disco-filter-ratings', String(JSON.stringify(state.filters.ratings)));
                btn.classList.toggle('active');
                state.currentPage = 1;
                updateNavActiveStates();
                performSearch();
            };

            btn.addEventListener('dragenter', (e) => {
                e.preventDefault();
                btn.classList.add('drag-over');
            });

            btn.addEventListener('dragover', (e) => {
                e.preventDefault();
                (e as DragEvent).dataTransfer.dropEffect = 'copy';
            });

            btn.addEventListener('dragleave', (e) => {
                if (!btn.contains(((e as MouseEvent).relatedTarget as HTMLElement))) {
                    btn.classList.remove('drag-over');
                }
            });

            btn.addEventListener('drop', async (e) => {
                e.preventDefault();
                e.stopPropagation();
                btn.classList.remove('drag-over');

                const rating = parseInt((btn as any).dataset.rating);
                const path = (e as DragEvent).dataTransfer.getData('text/plain');

                if (path) {
                    const item = (state.draggedItem && state.draggedItem.path === path) ?
                        state.draggedItem : { path };
                    await rateMedia(item, rating);
                    performSearch();
                }
                state.draggedItem = null;
                document.body.classList.remove('is-dragging');
            });
        });
    }

    async function fetchDU(path = '') {
        const prevPath = state.duPath;
        const isForwardNav = prevPath && path.startsWith(prevPath);
        const isBackwardNav = prevPath && prevPath.startsWith(path);
        // Detect first visit by checking if we haven't loaded any DU data yet
        const hasLoadedDuData = state.duData && state.duData.length > 0;
        const isFirstDUVisit = !hasLoadedDuData;
        // Track if we're auto-skipping to prevent infinite recursion
        const isAutoSkipRecursion = prevPath && path.startsWith(prevPath) && path !== prevPath;

        state.page = 'du';
        state.duPath = path;

        // Set default sort for DU view on first visit: size descending
        if (isFirstDUVisit) {
            state.filters.sort = 'size';
            state.filters.reverse = true;
            state.view = 'grid'; // Default to grid view in DU
            const sortBy = document.getElementById('sort-by');
            const sortReverseBtn = document.getElementById('sort-reverse-btn');
            if (sortBy) (sortBy as HTMLSelectElement).value = 'size';
            if (sortReverseBtn) sortReverseBtn.classList.add('active');

            // Fetch filter bins to populate percentile sliders when entering DU mode
            // This ensures bins are populated even when coming from "No media found" state
            fetchFilterBins(null);
        }

        syncUrl();

        const skeletonTimeout = setTimeout(() => {
            if (state.view === 'grid') showSkeletons();
        }, 150);

        try {
            const params = new URLSearchParams();
            params.append('path', String(path));
            appendFilterParams(params);

            const resp = await fetchAPI(`/api/du?${params.toString()}`);
            clearTimeout(skeletonTimeout);
            if (!resp.ok) throw new Error('Failed to fetch DU');
            let response = await resp.json();

            // New format: {folders: [], files: [], total_count, folder_count, file_count}
            // Store the raw response - no conversion needed
            state.duDataRaw = response;
            
            // Create combined data array for rendering (folders first, then files)
            let data = [];
            if (response.folders) {
                data = data.concat(response.folders);
            }
            if (response.files) {
                data = data.concat(response.files);
            }
            state.duData = data;

            // Auto-skip through single-item folders recursively (including initial load)
            // Auto-skip on first visit OR during auto-skip recursion (when navigating deeper)
            // Only auto-skip if there's exactly 1 folder and 0 files (pure folder navigation)
            let shouldAutoSkip = (isFirstDUVisit || isAutoSkipRecursion) && 
                                 response.folders && response.folders.length === 1 && 
                                 response.files && response.files.length === 0;
            if (shouldAutoSkip) {
                const singleItem = response.folders[0];
                // Check if it's a folder (count > 0 means it has files in subdirectories)
                // A folder with only subfolders (no direct files) will have count > 0
                const isFolder = singleItem.count > 0;
                if (isFolder) {
                    // Auto-navigate into this folder
                    state.duPath = singleItem.path + (singleItem.path.endsWith('/') ? '' : '/');
                    syncUrl();
                    fetchDU(state.duPath);
                    return;
                }
            }

            renderDU(state.duData, state.duDataRaw);
        } catch (err) {
            clearTimeout(skeletonTimeout);
            console.error('DU fetch failed:', err);
            errorToast(err as any, 'Failed to load Disk Usage');
        }
    }

    function renderDU(data, rawResponse?) {
        // Use pre-computed aggregates from backend
        const totalFolders = rawResponse?.folder_count || 0;
        const totalFiles = rawResponse?.file_count || 0;

        // Show current path in toolbar input
        const duToolbar = document.getElementById('du-toolbar');
        const duPathInput = document.getElementById('du-path-input');
        const duBackBtn = document.getElementById('du-back-btn');
        const duBreadcrumbs = document.getElementById('du-breadcrumbs');

        if (duToolbar && duPathInput) {
            duToolbar.classList.remove('hidden');
            const displayPath = state.duPath || '/';
            (duPathInput as HTMLInputElement).value = displayPath;
            duPathInput.title = displayPath;

            // Show/hide back button based on current path
            if (duBackBtn) {
                if (state.duPath && state.duPath !== '/' && state.duPath !== '.') {
                    duBackBtn.style.display = 'block';
                } else {
                    duBackBtn.style.display = 'none';
                }
            }
        }

        // Render breadcrumbs for mobile navigation
        if (duBreadcrumbs) {
            renderDUBreadcrumbs(duBreadcrumbs);
        }

        // Show folder/file count in results-info (from backend aggregates)
        resultsCount.textContent = `${totalFolders} folders, ${totalFiles} files`;

        // Build mediaItems array for navigation (folders with files + direct files)
        let mediaItems = [];
        if (rawResponse?.folders) {
            rawResponse.folders.forEach(folder => {
                if (folder.files && folder.files.length > 0) {
                    mediaItems = mediaItems.concat(folder.files);
                }
            });
        }
        if (rawResponse?.files) {
            mediaItems = mediaItems.concat(rawResponse.files);
        }

        // Set currentMedia for DU mode so that playSibling uses DU siblings
        if (state.page === 'du') {
            currentMedia = mediaItems;
        }

        // Render based on current view mode
        if (state.view === 'details') {
            renderDUDetails(data);
        } else {
            renderDUGrid(data);
        }

        paginationContainer.classList.add('hidden');
        updateNavActiveStates();
    }

    function renderDUBreadcrumbs(container) {
        const path = state.duPath || '/';
        if (!path || path === '/') {
            container.innerHTML = '';
            return;
        }

        const parts = path.split('/').filter(p => p);
        const breadcrumbs = [];

        // Root
        breadcrumbs.push(`<span class="du-breadcrumb-item" data-path="/">/</span>`);

        // Build breadcrumb trail
        let currentPath = '';
        for (let i = 0; i < parts.length; i++) {
            const part = parts[i];
            currentPath += '/' + part;
            const isLast = i === parts.length - 1;

            breadcrumbs.push(`<span class="du-breadcrumb-sep">›</span>`);
            if (isLast) {
                breadcrumbs.push(`<span class="du-breadcrumb-item current">${part}</span>`);
            } else {
                breadcrumbs.push(`<span class="du-breadcrumb-item" data-path="${currentPath}/">${part}</span>`);
            }
        }

        container.innerHTML = breadcrumbs.join('');

        // Add click handlers
        container.querySelectorAll('.du-breadcrumb-item[data-path]').forEach(item => {
            item.onclick = () => {
                const targetPath = (item as HTMLElement).dataset.path;
                if (targetPath !== state.duPath) {
                    fetchDU(targetPath);
                }
            };
        });
    }

    function renderDUDetails(data) {
        // Handle empty state
        if (!data || data.length === 0) {
            resultsContainer.className = 'no-results-container';
            resultsContainer.innerHTML = `
                <div class="no-results" style="
                    display: flex;
                    flex-direction: column;
                    align-items: center;
                    justify-content: center;
                    padding: 2rem;
                    text-align: center;
                    color: var(--text-muted);
                    max-width: 500px;
                    margin: 0 auto;
                ">
                    <div style="font-size: 4rem; margin-bottom: 1rem; opacity: 0.5;">🎒</div>
                    <h2 style="margin: 0 0 0.5rem 0; color: var(--text);">No media found</h2>
                    <p style="margin: 0; max-width: 400px;">
                        Try adjusting your filters or navigate to a different directory.
                    </p>
                    ${state.filters.episodes.length > 0 || state.filters.sizes.length > 0 || state.filters.durations.length > 0 ? `
                        <button class="category-btn" onclick="window.disco.resetFilters()" style="margin-top: 1.5rem;">
                            Clear all filters
                        </button>
                    ` : ''}
                </div>
            `;
            paginationContainer.classList.add('hidden');
            return;
        }

        // Table view for DU data
        resultsContainer.className = 'details-view du-view';
        resultsContainer.innerHTML = '';

        const table = document.createElement('table');
        table.className = 'details-table';
        table.innerHTML = `
            <thead>
                <tr>
                    <th>Name</th>
                    <th>Type</th>
                    <th>Size</th>
                    <th>Duration</th>
                    <th>Files</th>
                </tr>
            </thead>
            <tbody></tbody>
        `;

        const tbody = table.querySelector('tbody');

        data.forEach(item => {
            const tr = document.createElement('tr');

            // Check if this is a direct file (has type field, no count field)
            const isDirectFile = item.type !== undefined && item.count === undefined;

            if (isDirectFile) {
                // Direct file row
                const mediaItem = item;
                const name = mediaItem.title || mediaItem.path.split('/').pop();
                const type = mediaItem.type || 'unknown';
                const size = formatSize(mediaItem.size);
                const duration = formatDuration(mediaItem.duration);

                tr.innerHTML = `
                    <td>📄 ${name}</td>
                    <td>${type}</td>
                    <td>${size}</td>
                    <td>${duration}</td>
                    <td>-</td>
                `;
                tr.onclick = () => playMedia(mediaItem);
            } else if (item.files && item.files.length > 0) {
                // Folder with files - render each file
                item.files.forEach(mediaItem => {
                    const fileTr = document.createElement('tr');
                    const name = mediaItem.title || mediaItem.path.split('/').pop();
                    const type = mediaItem.type || 'unknown';
                    const size = formatSize(mediaItem.size);
                    const duration = formatDuration(mediaItem.duration);

                    fileTr.innerHTML = `
                        <td>📄 ${name}</td>
                        <td>${type}</td>
                        <td>${size}</td>
                        <td>${duration}</td>
                        <td>-</td>
                    `;
                    fileTr.onclick = () => playMedia(mediaItem);
                    tbody.appendChild(fileTr);
                });
                return;
            } else {
                // Folder row
                const name = item.path.split('/').pop() || item.path;
                const size = formatSize(item.total_size);
                const duration = formatDuration(item.total_duration);
                const count = item.count;

                tr.innerHTML = `
                    <td>📁 ${name}</td>
                    <td>folder</td>
                    <td>${size}</td>
                    <td>${duration}</td>
                    <td>${count}</td>
                `;
                tr.onclick = () => fetchDU(item.path + (item.path.endsWith('/') ? '' : '/'));
            }

            tr.style.cursor = 'pointer';
            tr.addEventListener('mouseover', () => tr.style.background = 'var(--sidebar-bg)');
            tr.addEventListener('mouseout', () => tr.style.background = '');
            tbody.appendChild(tr);
        });

        resultsContainer.appendChild(table);
    }

    function renderDUGrid(data) {
        resultsContainer.className = 'grid du-view';
        resultsContainer.innerHTML = '';

        // Handle empty state
        if (!data || data.length === 0) {
            resultsContainer.className = 'no-results-container';
            resultsContainer.innerHTML = `
                <div class="no-results" style="
                    display: flex;
                    flex-direction: column;
                    align-items: center;
                    justify-content: center;
                    padding: 2rem;
                    text-align: center;
                    color: var(--text-muted);
                    max-width: 500px;
                    margin: 0 auto;
                ">
                    <div style="font-size: 4rem; margin-bottom: 1rem; opacity: 0.5;">🎒</div>
                    <h2 style="margin: 0 0 0.5rem 0; color: var(--text);">No media found</h2>
                    <p style="margin: 0; max-width: 400px;">
                        Try adjusting your filters or navigate to a different directory.
                    </p>
                    ${state.filters.episodes.length > 0 || state.filters.sizes.length > 0 || state.filters.durations.length > 0 ? `
                        <button class="category-btn" onclick="window.disco.resetFilters()" style="margin-top: 1.5rem;">
                            Clear all filters
                        </button>
                    ` : ''}
                </div>
            `;
            paginationContainer.classList.add('hidden');
            return;
        }

        // Calculate max size for visualization (only from folders)
        const folders = state.duDataRaw?.folders || [];
        const maxSize = folders.length > 0 ? Math.max(...folders.map(d => d.total_size || 0)) : 0;

        data.forEach(item => {
            const card = document.createElement('div');

            // Check if this is a direct file (has type field, no count field)
            const isDirectFile = item.type !== undefined && item.count === undefined;

            if (isDirectFile) {
                // Render as clickable media card
                const mediaItem = item;
                card.className = 'media-card';
                (card as HTMLElement).dataset.path = mediaItem.path;
                (card as HTMLElement).dataset.type = mediaItem.type || '';
                if (mediaItem.is_dir) (card as HTMLElement).dataset.isDir = 'true';
                (card as HTMLElement).onclick = () => playMedia(mediaItem);

                const title = truncateString(mediaItem.title || mediaItem.path.split('/').pop());
                const thumbUrl = `/api/thumbnail?path=${encodeURIComponent(mediaItem.path)}`;
                const size = formatSize(mediaItem.size);
                const duration = formatDuration(mediaItem.duration);

                card.innerHTML = `
                    <div class="media-thumb">
                        <img src="${thumbUrl}" loading="lazy" onload="this.classList.add('loaded')">
                        <span class="media-duration">${duration}</span>
                    </div>
                    <div class="media-info">
                        <div class="media-title">${title}</div>
                        <div class="media-meta">
                            <span>${size}</span>
                        </div>
                    </div>
                `;
            } else if (item.files && item.files.length > 0) {
                // Render individual files from the files array as clickable media cards
                item.files.forEach(mediaItem => {
                    const fileCard = document.createElement('div');
                    fileCard.className = 'media-card';
                    (fileCard as HTMLElement).dataset.path = mediaItem.path;
                    (fileCard as HTMLElement).dataset.type = mediaItem.type || '';
                    if (mediaItem.is_dir) (fileCard as HTMLElement).dataset.isDir = 'true';
                    fileCard.onclick = () => playMedia(mediaItem);

                    const title = truncateString(mediaItem.title || mediaItem.path.split('/').pop());
                    const thumbUrl = `/api/thumbnail?path=${encodeURIComponent(mediaItem.path)}`;
                    const size = formatSize(mediaItem.size);
                    const duration = formatDuration(mediaItem.duration);

                    fileCard.innerHTML = `
                        <div class="media-thumb">
                            <img src="${thumbUrl}" loading="lazy" onload="this.classList.add('loaded')">
                            <span class="media-duration">${duration}</span>
                        </div>
                        <div class="media-info">
                            <div class="media-title">${title}</div>
                            <div class="media-meta">
                                <span>${size}</span>
                            </div>
                        </div>
                    `;
                    resultsContainer.appendChild(fileCard);
                });
                return; // Skip the rest for this item since we already added the files
            } else {
                // Render as folder card
                card.className = 'media-card du-card';
                (card as HTMLElement).onclick = () => fetchDU(item.path + (item.path.endsWith('/') ? '' : '/'));

                const name = item.path.split('/').pop() || item.path;
                const size = formatSize(item.total_size);
                const duration = formatDuration(item.total_duration);
                const count = item.count;

                const percentage = maxSize > 0 ? Math.round((item.total_size / maxSize) * 100) : 0;

                card.innerHTML = `
                    <div class="media-thumb" style="display: flex; align-items: center; justify-content: center; font-size: 3rem; background: var(--sidebar-bg); position: relative;">
                        📁
                        <div class="du-bar-container" style="position: absolute; bottom: 0; left: 0; right: 0; height: 10px; background: rgba(0,0,0,0.1);">
                            <div class="du-bar" style="width: ${percentage}%; height: 100%; background: var(--accent-color); opacity: 0.6;"></div>
                        </div>
                    </div>
                    <div class="media-info">
                        <div class="media-title" title="${item.path}">${name}</div>
                        <div class="media-meta">
                            <span title="Folder Size">${size}</span>
                            <span>${count} files</span>
                            <span title="Total Duration">${duration}</span>
                        </div>
                    </div>
                `;
            }
            resultsContainer.appendChild(card);
        });

        updateNowPlayingButton();
    }

    function showEpisodesLoading() {
        resultsContainer.className = 'similarity-view';
        resultsContainer.innerHTML = `
            <div class="loading-container" style="text-align: center; padding: 3rem;">
                <div class="spinner"></div>
                <h3>Grouping by Parent Folder...</h3>
                <p>Organizing media into episodic groups.</p>
            </div>
        `;
    }

    async function fetchEpisodes() {
        if (searchAbortController) {
            searchAbortController.abort();
        }
        searchAbortController = new AbortController();

        // Use specific loading screen for episodes
        showEpisodesLoading();

        try {
            const params = new URLSearchParams();
            appendFilterParams(params);

            if (state.filters.all) {
                params.append('all', String('true'));
            } else {
                params.append('limit', String(state.filters.limit.toString()));
            }

            if (state.page === 'trash') {
                params.append('trash', String('true'));
            } else if (state.page === 'history') {
                params.append('watched', String('true'));
            } else if (state.page === 'captions') {
                params.append('captions', String('true'));
            }

            const resp = await fetchAPI(`/api/episodes?${params.toString()}`, {
                signal: searchAbortController.signal
            });
            if (!resp.ok) throw new Error('Failed to fetch episodes');
            let groups = await resp.json();
            if (!groups) groups = [];

            // Merge local progress if enabled
            if (state.localResume) {
                const localProgress = JSON.parse(localStorage.getItem('disco-progress') || '{}');

                if (state.page === 'history' || state.filters.unfinished || state.filters.completed) {
                    const serverFiles = [];
                    groups.forEach(g => { if (g.files) serverFiles.push(...g.files); });
                    const serverPaths = new Set(serverFiles.map(m => m.path));

                    let missingPaths = Object.keys(localProgress).filter(p => !serverPaths.has(p));

                    if (missingPaths.length > 0) {
                        let missingData = await fetchMediaByPaths(missingPaths);

                        // Client-side filtering for merged items
                        if (state.filters.unfinished) {
                            missingData = missingData.filter(item => getPlayCount(item) === 0 && (localProgress[item.path]?.pos > 0));
                        } else if (state.filters.completed) {
                            missingData = missingData.filter(item => getPlayCount(item) > 0);
                        } else if (state.page === 'history') {
                            missingData = missingData.filter(item => (localProgress[item.path]?.last > 0));
                        }

                        if (missingData.length > 0) {
                            missingData.forEach(m => {
                                const parent = m.path.substring(0, m.path.lastIndexOf('/')) || '/';
                                const existing = groups.find(g => g.path === parent);
                                if (existing) {
                                    if (!existing.files.some(f => f.path === m.path)) {
                                        existing.files.push(m);
                                        existing.count++;
                                    }
                                } else {
                                    groups.push({
                                        path: parent,
                                        files: [m],
                                        count: 1
                                    });
                                }
                            });
                            groups.sort((a, b) => a.path.localeCompare(b.path));
                        }
                    }
                }
            }

            // Update playhead and time_last_played from localStorage for all items in groups
            if (state.localResume) {
                const localProgress = JSON.parse(localStorage.getItem('disco-progress') || '{}');
                groups.forEach(group => {
                    if (group.files) {
                        group.files.forEach(item => {
                            const local = localProgress[item.path];
                            if (local) {
                                const localPlayhead = typeof local === 'object' ? local.pos : local;
                                const localTime = typeof local === 'object' ? local.last / 1000 : 0;

                                if (localTime > (item.time_last_played || 0)) {
                                    item.playhead = localPlayhead;
                                    item.time_last_played = localTime;
                                } else if (localPlayhead > (item.playhead || 0)) {
                                    item.playhead = localPlayhead;
                                }
                            }
                        });
                    }
                });
            }

            state.similarityData = groups;
            renderEpisodes(state.similarityData);
        } catch (err) {
            if (err.name === 'AbortError') return;
            console.error('Episodes fetch failed:', err);
            errorToast(err as any, 'Failed to load Episodes');
            resultsContainer.innerHTML = `<div class="error">Failed to load episodes.</div>`;
        }
    }

    function renderEpisodes(data) {
        if (!data) data = [];

        // Server already filters by type, search, and progress via appendFilterParams()
        // Just use the data as-is - no need for redundant client-side filtering
        let filtered = data.map(group => {
            const files = group.files || [];
            return { ...group, files: files, count: files.length };
        }).filter(group => group.count > 0);

        // Update currentMedia so playSibling works in group view
        currentMedia = [];
        filtered.forEach(group => {
            currentMedia.push(...group.files);
        });

        resultsCount.textContent = `${filtered.length} folders found`;
        resultsContainer.className = 'similarity-view';
        resultsContainer.innerHTML = '';

        filtered.forEach((group, gIdx) => {
            const groupEl = document.createElement('div');
            groupEl.className = 'similarity-group';

            // Recalculate group stats
            const totalSize = group.files.reduce((acc, f) => acc + (f.size || 0), 0);
            const totalDuration = group.files.reduce((acc, f) => acc + (f.duration || 0), 0);

            const groupHeader = document.createElement('div');
            groupHeader.className = 'similarity-header';
            groupHeader.innerHTML = `
                <h3>${group.path || 'Common context'}</h3>
                <div class="group-meta">${group.count} files • ${formatSize(totalSize)} • ${formatDuration(totalDuration)}</div>
            `;
            groupEl.appendChild(groupHeader);

            const filesGrid = document.createElement('div');
            filesGrid.className = 'grid';

            group.files.forEach(item => {
                const card = document.createElement('div');
                card.className = 'media-card';
                (card as HTMLElement).dataset.path = item.path;
                (card as HTMLElement).dataset.type = item.type || '';
                if (item.is_dir) (card as HTMLElement).dataset.isDir = 'true';
                (card as HTMLElement).onclick = () => playMedia(item);

                const title = item.title || item.path.split('/').pop();
                const thumbUrl = `/api/thumbnail?path=${encodeURIComponent(item.path)}`;

                card.innerHTML = `
                    <div class="media-thumb">
                        <img src="${thumbUrl}" loading="lazy" onload="this.classList.add('loaded')">
                        <span class="media-duration">${formatDuration(item.duration)}</span>
                    </div>
                    <div class="media-info">
                        <div class="media-title" title="${item.path}">${title}</div>
                        <div class="media-meta">
                            <span>${formatSize(item.size)}</span>
                            <span>${item.video_codecs || ''}</span>
                        </div>
                    </div>
                `;

                filesGrid.appendChild(card);
            });

            groupEl.appendChild(filesGrid);
            resultsContainer.appendChild(groupEl);
        });

        paginationContainer.classList.add('hidden');
        updateNavActiveStates();
        updateNowPlayingButton();
    }

    async function fetchCuration() {
        state.page = 'curation';
        syncUrl();

        document.getElementById('toolbar').classList.add('hidden');
        document.getElementById('search-container').classList.add('hidden');

        // Show loading initially
        resultsContainer.innerHTML = '<div class="loading-container" style="text-align: center; padding: 3rem;"><div class="spinner"></div><h3>Loading Categorization...</h3></div>';

        try {
            const resp = await fetchAPI('/api/categorize/keywords');
            if (!resp.ok) throw new Error('Failed to fetch keywords');
            const data = await resp.json();
            renderCuration(data);
        } catch (err) {
            console.error('Curation fetch failed:', err);
            errorToast(err as any, 'Failed to load Curation Tool');
            resultsContainer.innerHTML = '<div class="error">Failed to load categorization tool.</div>';
        }
    }

    function createCategoryCard(cat) {
        const card = document.createElement('div');
        card.className = 'curation-cat-card';
        (card as HTMLElement).dataset.category = cat.category;
        card.style.background = 'var(--sidebar-bg)';
        card.style.padding = '1rem';
        card.style.borderRadius = '8px';
        card.style.border = '1px solid var(--border-color)';

        let keywordsHtml = (cat.keywords || []).map(kw =>
            `<span class="curation-tag existing-keyword" data-keyword="${kw}" data-category="${cat.category}">
                ${kw} <span class="remove-kw" style="cursor:pointer; margin-left:4px; opacity:0.6;">&times;</span>
            </span>`
        ).join('');

        card.innerHTML = `
            <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 0.5rem;">
                <h4 style="margin: 0;">${cat.category}</h4>
                <button class="delete-cat-btn" title="Delete Category" style="background: none; border: none; cursor: pointer; opacity: 0.5;">🗑️</button>
            </div>
            <div class="cat-keywords" style="display: flex; flex-wrap: wrap; gap: 0.5rem; margin-bottom: 0.5rem;">
                ${keywordsHtml}
            </div>
            <button class="add-kw-btn category-btn" style="font-size: 0.8rem; padding: 2px 8px;">+ Add Keyword</button>
        `;

        // Drag & Drop: Allow dropping tags here
        card.addEventListener('dragover', (e) => {
            e.preventDefault();
            card.style.borderColor = 'var(--accent-color)';
        });
        card.addEventListener('dragleave', () => {
            card.style.borderColor = 'var(--border-color)';
        });
        card.addEventListener('drop', async (e) => {
            e.preventDefault();
            card.style.borderColor = 'var(--border-color)';
            const keyword = (e as DragEvent).dataTransfer.getData('text/plain');
            if (keyword) {
                await addKeyword(cat.category, keyword);
            }
        });

        // Delete Category
        (card.querySelector('.delete-cat-btn') as HTMLElement).onclick = async () => {
            if (confirm(`Delete category "${cat.category}" and all its keywords?`)) {
                await deleteCategory(cat.category);
            }
        };

        // Add Keyword manually
        (card.querySelector('.add-kw-btn') as HTMLElement).onclick = async () => {
            const kw = prompt(`Add keyword to "${cat.category}":`);
            if (kw) {
                await addKeyword(cat.category, kw);
            }
        };

        // Remove Keyword
        card.querySelectorAll('.remove-kw').forEach(btn => {
            (btn as any).onclick = async (e) => {
                e.stopPropagation(); // prevent drag start if any
                const tag = (e.target as HTMLElement).closest('.curation-tag');
                const kw = (tag as HTMLElement).dataset.keyword;
                await deleteKeyword(cat.category, kw);
            };
        });

        return card;
    }

    function renderCuration(keywordsData) {
        if (!keywordsData) keywordsData = [];

        // Reorder data to put new categories at the top
        if (state.newCategories.length > 0) {
            const newOnes = [];
            const oldOnes = [];

            // Map for quick lookup of keywordsData by category name
            const dataMap = new Map();
            keywordsData.forEach(item => dataMap.set(item.category, item));

            // Add new ones in order of state.newCategories
            state.newCategories.forEach(catName => {
                if (dataMap.has(catName)) {
                    newOnes.push(dataMap.get(catName));
                    dataMap.delete(catName);
                }
            });

            // Rest are old ones (already sorted alphabetically by backend)
            dataMap.forEach(item => oldOnes.push(item));

            keywordsData = [...newOnes, ...oldOnes];
        }

        resultsContainer.className = 'curation-view';
        resultsContainer.innerHTML = '';

        const headerEl = document.createElement('div');
        headerEl.id = 'curation-header';
        headerEl.innerHTML = `
            <div style="display: flex; align-items: center; gap: 1rem; margin-bottom: 1rem;">
                <button id="curation-back-btn" class="category-btn">← Back</button>
                <h2 style="margin: 0;">Categorization</h2>
            </div>
            <p>Manage categories and keywords. Drag keywords from the suggestion pool to a category, or add them manually.</p>
            <div style="display: flex; align-items: center; gap: 1.5rem; margin: 1.5rem 0;">
                <button id="run-auto-categorize" class="category-btn" style="background: var(--accent-color); color: white;">Run Categorization Now</button>
                <div style="display: flex; align-items: center; gap: 0.5rem; font-size: 0.9rem;">
                    <input type="checkbox" id="categorize-full-path" ${localStorage.getItem('disco-categorize-full-path') === 'true' ? 'checked' : ''}>
                    <label for="categorize-full-path" title="Include parent directory names in keyword mining and matching">Include folder names</label>
                </div>
            </div>
        `;
        resultsContainer.appendChild(headerEl);

        const fullPathCheck = document.getElementById('categorize-full-path');
        if (fullPathCheck) {
            (fullPathCheck as any).onchange = (e) => {
                localStorage.setItem('disco-categorize-full-path', String((e.target as any).checked));
            };
        }

        const container = document.createElement('div');
        container.id = 'curation-container';
        container.style.display = 'flex';
        container.style.gap = '2rem';
        container.style.height = '100%';
        container.style.padding = '1rem';

        // --- Left Column: Categories ---
        const categoriesCol = document.createElement('div');
        categoriesCol.className = 'curation-col';
        categoriesCol.style.flex = '1';
        categoriesCol.style.display = 'flex';
        categoriesCol.style.flexDirection = 'column';
        categoriesCol.style.borderRight = '1px solid var(--border-color)';
        categoriesCol.style.paddingRight = '1rem';

        // Categories header with buttons
        const categoriesHeader = document.createElement('div');
        categoriesHeader.style.display = 'flex';
        categoriesHeader.style.justifyContent = 'space-between';
        categoriesHeader.style.alignItems = 'center';
        categoriesHeader.style.marginBottom = '1rem';
        categoriesHeader.innerHTML = `
            <h3 style="margin: 0;">Categories</h3>
            <div style="display: flex; gap: 0.5rem;">
                <button id="add-default-cats" class="category-btn" style="font-size: 0.85rem; padding: 4px 8px;">Add Default Categories</button>
                <button id="new-category-btn" class="category-btn" style="font-size: 0.85rem; padding: 4px 8px;">+ New Category</button>
            </div>
        `;
        categoriesCol.appendChild(categoriesHeader);

        // Scrollable categories list
        const categoriesList = document.createElement('div');
        categoriesList.id = 'curation-cat-list';
        categoriesList.style.display = 'flex';
        categoriesList.style.flexDirection = 'column';
        categoriesList.style.gap = '1rem';
        categoriesList.style.overflowY = 'auto';
        categoriesList.style.flex = '1';

        // Render existing categories
        keywordsData.forEach(cat => {
            categoriesList.appendChild(createCategoryCard(cat));
        });

        categoriesCol.appendChild(categoriesList);
        container.appendChild(categoriesCol);

        // Setup New Category button (in header)
        const newCatBtn = categoriesCol.querySelector('#new-category-btn');
        if (newCatBtn) {
            (newCatBtn as HTMLElement).onclick = async () => {
                const name = prompt('New Category Name:');
                if (name) {
                    const kw = prompt(`Add first keyword for "${name}":`);
                    if (kw) {
                        await addKeyword(name, kw);
                    }
                }
            };
        }

        // --- Right Column: Suggestions ---
        const suggestionsCol = document.createElement('div');
        suggestionsCol.className = 'curation-col';
        suggestionsCol.style.flex = '1';
        suggestionsCol.style.overflowY = 'auto';
        suggestionsCol.style.paddingLeft = '1rem';

        suggestionsCol.innerHTML = `
            <h3>Uncategorized / Suggestions</h3>
            <button id="find-keywords-btn" class="category-btn" style="width: 100%; margin-bottom: 1rem;">Find Potential Keywords</button>
            <div id="suggestions-area"></div>
        `;

        const findBtn = suggestionsCol.querySelector('#find-keywords-btn');
        const suggestionsArea = suggestionsCol.querySelector('#suggestions-area');

        (findBtn as HTMLElement).onclick = async () => {
            (findBtn as HTMLButtonElement).disabled = true;
            findBtn.textContent = 'Analyzing...';
            suggestionsArea.innerHTML = '<div class="spinner" style="width: 24px; height: 24px; margin: 1rem auto;"></div>';

            try {
                const isFullPath = fullPathCheck ? (fullPathCheck as any).checked : false;
                const resp = await fetchAPI(`/api/categorize/suggest?full_path=${isFullPath}`);
                if (!resp.ok) throw new Error('Failed');
                const suggestions = await resp.json();
                renderSuggestionsArea(suggestions, suggestionsArea);
            } catch (err) {
                console.error(err);
                suggestionsArea.innerHTML = '<p>Failed to load suggestions.</p>';
            } finally {
                (findBtn as HTMLButtonElement).disabled = false;
                findBtn.textContent = 'Find Potential Keywords';
            }
        };

        container.appendChild(suggestionsCol);
        resultsContainer.appendChild(container);

        // Header Actions
        const backBtn = headerEl.querySelector('#curation-back-btn');
        if (backBtn) {
            (backBtn as HTMLElement).onclick = () => {
                state.page = 'search';
                state.filters.categories = [];
                updateNavActiveStates();
                performSearch();
            };
        }

        const btnRun = headerEl.querySelector('#run-auto-categorize');
        if (btnRun) {
            (btnRun as HTMLElement).onclick = async () => {
                if (state.readOnly) return showToast('Read-only mode');
                (btnRun as HTMLButtonElement).disabled = true;
                btnRun.textContent = 'Running...';
                try {
                    const isFullPath = fullPathCheck ? (fullPathCheck as any).checked : false;
                    const resp = await fetchAPI(`/api/categorize/apply?full_path=${isFullPath}`, { method: 'POST' });
                    if (!resp.ok) throw new Error('Apply failed');
                    const data = await resp.json();
                    showToast(`Successfully categorized ${data.count} files!`, '🏷️');
                    fetchCategories();
                    // Don't refresh curation page necessarily, user might want to keep editing
                } catch (err) {
                    console.error('Apply failed:', err);
                    errorToast(err as any, 'Failed to run categorization');
                } finally {
                    (btnRun as HTMLButtonElement).disabled = false;
                    btnRun.textContent = 'Run Categorization Now';
                }
            };
        }

        const btnDefaults = categoriesCol.querySelector('#add-default-cats');
        if (btnDefaults) {
            (btnDefaults as HTMLElement).onclick = async () => {
                if (confirm('Add default categories and keywords? (Existing ones will be kept)')) {
                    try {
                        const resp = await fetchAPI('/api/categorize/defaults', { method: 'POST' });
                        if (!resp.ok) throw new Error('Failed');
                        showToast('Default categories added');
                        fetchCuration(); // Refresh
                    } catch (err) {
                        console.error(err);
                        errorToast(err as any, 'Failed to add defaults');
                    }
                }
            };
        }

        paginationContainer.classList.add('hidden');
        updateNavActiveStates();
    }

    function renderSuggestionsArea(suggestions, container) {
        if (!suggestions || suggestions.length === 0) {
            container.innerHTML = '<p>No common keywords found in uncategorized files.</p>';
            return;
        }

        container.innerHTML = `
            <p>Drag these keywords to a category on the left.</p>
            <p style="font-size: 0.85rem; color: var(--text-muted); margin-bottom: 1rem;">
                ℹ️ Count shows unique occurrences in uncategorized file names/titles. Each word is counted once per file.
            </p>
            <div class="tags-cloud">
                ${suggestions.map(tag => `
                    <span class="curation-tag suggestion-tag" draggable="true" data-word="${tag.word}" title="${tag.count} uncategorized files contain this word">
                        ${tag.word} <small>${tag.count}</small>
                    </span>
                `).join('')}
            </div>
        `;

        container.querySelectorAll('.suggestion-tag').forEach(tag => {
            tag.addEventListener('dragstart', (e: DragEvent) => { (e as DragEvent).dataTransfer.setData('text/plain', (tag as HTMLElement).dataset.word);
                tag.style.opacity = '0.5';
            });
            tag.addEventListener('dragend', (e) => {
                tag.style.opacity = '1';
            });
            // Click also prompts for category (legacy behavior, still useful)
            tag.onclick = async () => {
                const keyword = (tag as HTMLElement).dataset.word;
                const category = prompt(`Assign keyword "${keyword}" to category:`, keyword);
                if (category) {
                    await addKeyword(category, keyword);
                }
            };
        });
    }

    async function addKeyword(category, keyword) {
        try {
            const resp = await fetchAPI('/api/categorize/keyword', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ category, keyword })
            });
            if (!resp.ok) throw new Error('Failed');
            showToast(`Saved keyword "${keyword}" to "${category}"`);

            // Update state.newCategories to track session additions
            if (!state.newCategories.includes(category)) {
                state.newCategories.unshift(category);
            } else {
                state.newCategories = [category, ...state.newCategories.filter(c => c !== category)];
            }

            // 1. Remove from suggestions pool if it exists
            const suggestionTag = document.querySelector(`.suggestion-tag[data-word="${CSS.escape(keyword)}"]`);
            if (suggestionTag) {
                const parent = suggestionTag.parentElement;
                suggestionTag.remove();

                // If no suggestions left, show message
                if (parent && parent.classList.contains('tags-cloud') && parent.querySelectorAll('.suggestion-tag').length === 0) {
                    const suggestionsArea = document.getElementById('suggestions-area');
                    if (suggestionsArea) {
                        suggestionsArea.innerHTML = '<p>No common keywords found in uncategorized files.</p>';
                    }
                }
            }

            // 2. Add to category card if it exists
            const catCard = document.querySelector(`.curation-cat-card[data-category="${CSS.escape(category)}"]`);
            if (catCard) {
                const kwContainer = catCard.querySelector('.cat-keywords');
                if (kwContainer) {
                    // Check if keyword already exists in this card to avoid duplicates
                    if (!kwContainer.querySelector(`.existing-keyword[data-keyword="${CSS.escape(keyword)}"]`)) {
                        const tag = document.createElement('span');
                        tag.className = 'curation-tag existing-keyword';
                        (tag as HTMLElement).dataset.keyword = keyword;
                        (tag as HTMLElement).dataset.category = category;
                        tag.innerHTML = `${keyword} <span class="remove-kw" style="cursor:pointer; margin-left:4px; opacity:0.6;">&times;</span>`;

                        // Add delete handler to the new tag
                        (tag.querySelector('.remove-kw') as HTMLElement).onclick = async (e) => {
                            e.stopPropagation();
                            await deleteKeyword(category, keyword);
                        };

                        kwContainer.appendChild(tag);
                    }
                }
            } else {
                // Category doesn't exist yet, create it dynamically
                const categoriesList = document.getElementById('curation-cat-list');
                if (categoriesList) {
                    const newCard = createCategoryCard({ category, keywords: [keyword] });
                    // Prepend so it's at the top (matches session pinning logic)
                    categoriesList.insertBefore(newCard, categoriesList.firstChild);
                }
            }
        } catch (err) {
            console.error(err);
            errorToast(err as any, 'Failed to save keyword');
        }
    }

    async function deleteCategory(category) {
        try {
            const resp = await fetchAPI(`/api/categorize/category?category=${encodeURIComponent(category)}`, { method: 'DELETE' });
            if (!resp.ok) throw new Error('Failed');
            showToast(`Deleted category "${category}"`);

            // Remove from session-new list if present
            state.newCategories = state.newCategories.filter(c => c !== category);

            // Remove from UI locally
            const catCard = document.querySelector(`.curation-cat-card[data-category="${CSS.escape(category)}"]`);
            if (catCard) {
                catCard.remove();
            }
        } catch (err) {
            console.error(err);
            errorToast(err as any, 'Failed to delete category');
        }
    }
    async function deleteKeyword(category, keyword) {
        try {
            const resp = await fetchAPI('/api/categorize/keyword', {
                method: 'DELETE',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ category, keyword })
            });
            if (!resp.ok) throw new Error('Failed');
            showToast(`Removed keyword "${keyword}"`);

            // Find and remove the tag from the UI locally
            const catCard = document.querySelector(`.curation-cat-card[data-category="${CSS.escape(category)}"]`);
            if (catCard) {
                const tag = catCard.querySelector(`.existing-keyword[data-keyword="${CSS.escape(keyword)}"]`);
                if (tag) {
                    tag.remove();
                }
            }
        } catch (err) {
            console.error(err);
            errorToast(err as any, 'Failed to delete keyword');
        }
    }

    async function performSearch() {
        if (state.page === 'playlist' && state.filters.playlist) {
            filterPlaylistItems();
            return;
        }

        if (state.page !== 'trash' && state.page !== 'history' && state.page !== 'playlist' && state.page !== 'du' && state.page !== 'curation' && state.page !== 'captions') {
            state.page = 'search';
        }
        state.filters.search = searchInput ? (searchInput as HTMLInputElement).value : '';
        state.filters.sort = sortBy ? sortBy.value : 'default';
        state.filters.limit = limitInput ? (parseInt(limitInput.value) || 100) : 100;
        state.filters.all = limitAll ? limitAll.checked : false;

        syncUrl();

        if (state.page === 'du') {
            fetchDU(state.duPath || '');
            return;
        }

        if (state.view === 'group' && state.page !== 'captions') {
            fetchEpisodes();
            return;
        }

        if (trashBtn && state.page !== 'trash') trashBtn.classList.remove('active');

        if (searchAbortController) {
            searchAbortController.abort();
        }
        searchAbortController = new AbortController();

        localStorage.setItem('disco-limit', String(state.filters.limit));
        localStorage.setItem('disco-limit-all', String(state.filters.all));

        if (limitInput) limitInput.disabled = state.filters.all;

        const skeletonTimeout = setTimeout(() => {
            if (state.page === 'search' || state.page === 'trash' || state.page === 'history' || state.page === 'playlist' || state.page === 'captions') {
                if (state.view === 'grid') showSkeletons();
            }
        }, 150);

        try {
            const params = new URLSearchParams();

            if (state.page === 'trash') {
                params.append('trash', String('true'));
            } else if (state.page === 'history') {
                params.append('watched', String('true'));
            } else if (state.page === 'captions') {
                params.append('captions', String('true'));
                params.append('aggregate', String('true'));
            }

            appendFilterParams(params);

            // Sidebar captions filter (when not in full captions mode)
            if (state.filters.captions && state.page !== 'captions') {
                params.append('captions', String('true'));
                params.append('aggregate', String('true'));
            }

            // Add sort parameters
            if (state.filters.sort === 'custom' && state.filters.customSortFields) {
                params.append('sort_fields', state.filters.customSortFields);
            } else {
                params.append('sort', String(state.filters.sort));
                if (state.filters.reverse) params.append('reverse', String('true'));
            }

            if (state.filters.all) {
                params.append('all', String('true'));
            } else {
                params.append('limit', String(state.filters.limit.toString()));
                params.append('offset', String(((state.currentPage - 1)) * state.filters.limit).toString());
            }

            // Add search type parameter
            if (state.filters.searchType === 'substring') {
                params.append('search_type', String('substring'));
            }

            // Request filter counts for sidebar bins (eliminates separate /api/filter-bins call)
            params.append('include_counts', String('true'));

            const resp = await fetchAPI(`/api/query?${params.toString()}`, {
                signal: searchAbortController.signal
            });

            clearTimeout(skeletonTimeout);

            if (!resp.ok) {
                const text = await resp.text();
                throw new Error(text || `Server returned ${resp.status}`);
            }

            const xTotalCount = resp.headers.get('X-Total-Count');
            if (xTotalCount) {
                state.totalCount = parseInt(xTotalCount);
            }

            let data = await resp.json();
            if (!data) data = [];

            // Extract filter counts if included in response
            if (data && typeof data === 'object' && !Array.isArray(data) && data.items && data.counts) {
                state.filterBins = data.counts;
                data = data.items;
                renderFilterBins();
            }
            if (!Array.isArray(data)) {
                console.error('Expected array of media items but got:', data);
                data = [];
            }

            // Merge local progress if enabled
            if (state.localResume) {
                const localProgress = JSON.parse(localStorage.getItem('disco-progress') || '{}');

                if (state.page === 'history' || state.filters.unfinished || state.filters.completed) {
                    // Find paths that are in localStorage but not in the server results
                    const localPaths = Object.keys(localProgress);
                    const serverPaths = new Set(data.map(item => item.path));
                    const missingPaths = localPaths.filter(p => !serverPaths.has(p));

                    if (missingPaths.length > 0) {
                        const extraMedia = await fetchMediaByPaths(missingPaths);
                        data = [...data, ...extraMedia];
                    }
                }

                // Update playhead and time_last_played from localStorage for all items
                data.forEach(item => {
                    const local = localProgress[item.path];
                    if (local) {
                        const localPlayhead = typeof local === 'object' ? local.pos : local;
                        const localTime = typeof local === 'object' ? local.last / 1000 : 0;

                        // If local progress is newer, trust it even if it is 0
                        if (localTime > (item.time_last_played || 0)) {
                            item.playhead = localPlayhead;
                            item.time_last_played = localTime;
                        } else if (localPlayhead > (item.playhead || 0)) {
                            // Fallback for older localStorage entries that might not have timestamps
                            item.playhead = localPlayhead;
                        }
                    }
                });
            }

            // Set currentMedia from server data
            // Note: DB filtering is now done server-side, progress filtering is also server-side
            // Only client-side filtering remaining is for localStorage merge above
            currentMedia = data;

            // Client-side progress filtering for localStorage items only
            // Server already filters by progress, but localStorage items may not match
            if (state.filters.unplayed) {
                currentMedia = currentMedia.filter(item => getPlayCount(item) === 0);
            } else if (state.filters.unfinished) {
                currentMedia = currentMedia.filter(item => getPlayCount(item) === 0 && (item.playhead || 0) > 0);
            } else if (state.filters.completed) {
                currentMedia = currentMedia.filter(item => getPlayCount(item) > 0);
            } else if (state.page === 'history') {
                currentMedia = currentMedia.filter(item => (item.time_last_played || 0) > 0);
            }

            // Caption search filtering
            // Server already returns filtered results with context (2 before/after matches)
            // No client-side filtering needed - just use the data as-is

            // Local sorting for play_count if global progress is disabled
            if (!!state.readOnly && state.filters.sort === 'play_count') {
                currentMedia.sort((a, b) => {
                    const countA = getPlayCount(a);
                    const countB = getPlayCount(b);
                    if (state.filters.reverse) return countA - countB;
                    return countB - countA;
                });
            } else if (state.filters.sort === 'progress') {
                currentMedia.sort((a, b) => {
                    const progA = (a.duration && a.playhead) ? a.playhead / a.duration : 0;
                    const progB = (b.duration && b.playhead) ? b.playhead / b.duration : 0;

                    if (progA === 0 && progB === 0) return 0;
                    if (progA === 0) return 1;
                    if (progB === 0) return -1;

                    if (state.filters.reverse) return progA - progB;
                    return progB - progA;
                });
            } else if (state.filters.sort === 'extension') {
                currentMedia.sort((a, b) => {
                    const extA = a.path.split('.').pop().toLowerCase();
                    const extB = b.path.split('.').pop().toLowerCase();
                    if (state.filters.reverse) return extB.localeCompare(extA);
                    return extA.localeCompare(extB);
                });
            }

            updateNavActiveStates();
            renderResults();
            // Filter bins are now fetched with search results via include_counts=true
        } catch (err) {
            clearTimeout(skeletonTimeout);
            if (err.name === 'AbortError') return;
            console.error('Search failed:', err);
            resultsContainer.innerHTML = `<div class="error">Search failed: ${err.message}</div>`;
            if (err.message === 'Unauthorized') {
                window.location.reload();
            }
        }
    }
    async function fetchTrash() {
        state.page = 'trash';
        state.filters.sort = 'time_deleted';
        state.filters.reverse = true;
        if (sortBy) sortBy.value = 'time_deleted';
        performSearch();
    }

    async function fetchHistory() {
        state.page = 'history';
        state.filters.genre = '';
        state.filters.sort = 'time_last_played';
        state.filters.reverse = true;
        if (sortBy) sortBy.value = 'time_last_played';
        performSearch();
    }

    async function emptyBin() {
        if (currentMedia.length === 0) return;
        const count = currentMedia.length;
        const unit = count === 1 ? 'file' : 'files';
        if (!confirm(`Are you sure you want to permanently delete these ${count} ${unit}?`)) return;

        try {
            const paths = currentMedia.map(m => m.path);
            const resp = await fetchAPI('/api/empty-bin', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ paths })
            });
            if (!resp.ok) throw new Error('Failed to empty bin');
            const msg = await resp.text();
            showToast(msg, '🔥');
            fetchTrash();
        } catch (err) {
            console.error('Empty bin failed:', err);
            errorToast(err as any, 'Failed to empty bin');
        }
    }

    async function permanentlyDeleteMedia(path) {
        if (!confirm('Are you sure you want to permanently delete this file?')) return;

        try {
            const resp = await fetchAPI('/api/empty-bin', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ paths: [path] })
            });
            if (!resp.ok) throw new Error('Failed to delete');
            const msg = await resp.text();
            showToast(msg, '🔥');
            fetchTrash();
        } catch (err) {
            console.error('Permanent delete failed:', err);
            errorToast(err as any, 'Failed to delete');
        }
    }

    async function deleteMedia(path, restore = false) {
        const itemEl = document.querySelector(`[data-path="${CSS.escape(path)}"]`);
        const content = document.getElementById('content');
        const main = document.querySelector('main');

        if (itemEl && !restore) {
            itemEl.classList.add('fade-out');

            // Wait for animation (matched to 0.2s in CSS)
            await new Promise(r => setTimeout(r, 200));
        }

        try {
            const resp = await fetchAPI('/api/delete', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ path, restore })
            });

            if (!resp.ok) {
                const text = await resp.text();
                throw new Error(text || 'Action failed');
            }

            if (restore) {
                showToast('Item restored');
            } else {
                const filename = path.split('/').pop();
                showToast(`Trashed ${filename}`, '🗑️');
            }

            if (state.page === 'trash') {
                fetchTrash();
            } else {
                performSearch();
            }
        } catch (err) {
            console.error('Delete/Restore failed:', err);
            errorToast(err as any, 'Action failed');
            if (itemEl) itemEl.classList.remove('fade-out');
        } finally {
            if (content) content.style.overflow = '';
            if (main) main.style.overflow = '';
        }
    }

    async function playMedia(item, bypassQueue = false, queueIndex = -1) {
        if (state.enableQueue && !bypassQueue) {
            const itemBasename = item.path.split('/').pop();
            if (state.queueAddMode === 'end') {
                state.playback.queue.push(item);
                showToast(`Added to queue: ${itemBasename}`, '➕');
            } else {
                // Add to next (after current item if it's in the queue, otherwise at start)
                const currentIndex = state.playback.queueIndex;
                state.playback.queue.splice(currentIndex + 1, 0, item);
                showToast(`Added to next: ${itemBasename}`, '⏭️');
            }
            renderQueue();
            return;
        }

        if (state.playback.skipTimeout) {
            clearTimeout(state.playback.skipTimeout);
            state.playback.skipTimeout = null;
        }

        if (state.player === 'browser') {
            openActivePlayer(item, true, false, queueIndex);
            return;
        }

        const prevItem = state.playback.item;
        const wasPlayed = state.playback.hasMarkedComplete || (prevItem && getPlayCount(prevItem) > 0);

        state.playback.item = item;
        state.playback.startTime = Date.now();
        state.playback.hasMarkedComplete = false;
        updateNowPlayingButton();

        if (prevItem && prevItem.path !== item.path && state.filters.unplayed && wasPlayed) {
            if (state.playback.pendingUpdate) await state.playback.pendingUpdate;
            performSearch();
        }

        const path = item.path;
        showToast(`Playing: ${path.split('/').pop()}`);
        try {
            const resp = await fetchAPI('/api/play', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ path })
            });

            if (!resp.ok) {
                if (resp.status === 403) {
                    showToast('Access Denied', '🚫');
                    if (state.autoplay) {
                        playSibling(1);
                    }
                } else if (resp.status === 404 || resp.status === 415) {
                    const basename = path.split('/').pop();
                    const msg = resp.status === 404 ? `File not found: ${basename}` : `Unplayable (Unsupported): ${basename}`;
                    const emoji = resp.status === 404 ? '🗑️' : '⚠️';

                    if (state.page === 'trash') {
                        showToast(msg, '⚠️');
                    } else {
                        showToast(msg, emoji);
                        // Remove from current view if applicable
                        currentMedia = currentMedia.filter(m => m.path !== path);
                        renderResults();
                    }
                    if (state.autoplay) {
                        playSibling(1);
                    }
                } else {
                    errorToast(new Error(resp.statusText), 'Playback failed');
                }
            }
        } catch (err) {
            console.error('Playback failed', err);
            errorToast(err as any, 'Playback failed');
        }
    }

    async function updateProgress(item, playhead, duration, isComplete = false) {
        if (playhead > 1) {
            state.playback.consecutiveErrors = 0;
        }

        const media = pipViewer.querySelector('video, audio');
        if (!isComplete && media && ((media as any).seeking || (media as any).readyState < 3)) {
            return state.playback.pendingUpdate;
        }

        const now = Date.now();

        if (isComplete) {
            if (state.playback.hasMarkedComplete) return state.playback.pendingUpdate;
            state.playback.hasMarkedComplete = true;
        }

        // Local progress is always saved if enabled
        if (state.localResume) {
            // Throttling: only update localStorage once per second
            if (isComplete || (now - state.playback.lastLocalUpdate) >= 1000) {
                const progress = getLocalStorageItem('disco-progress', {});
                if (isComplete) {
                    if (!!state.readOnly) {
                        progress[item.path] = {
                            pos: 0,
                            last: now
                        };
                    } else {
                        delete progress[item.path];
                    }

                    // Increment play count locally if global progress is disabled
                    if (!!state.readOnly) {
                        const counts = getLocalStorageItem('disco-play-counts', {});
                        counts[item.path] = (counts[item.path] || 0) + 1;
                        setLocalStorageItem('disco-play-counts', counts);
                    }
                } else {
                    // Latest wins merging with position preference
                    const existing = progress[item.path];
                    const existingLast = existing && typeof existing === 'object' ? existing.last : 0;
                    const existingPos = existing && typeof existing === 'object' ? existing.pos : 0;
                    const newPos = Math.floor(playhead);

                    // Update if: new timestamp is significantly newer (>5s), OR timestamps are close and new position is higher
                    // This prevents overwriting better progress from another session with worse local progress
                    const timeDiff = now - existingLast;
                    if (timeDiff > 5000 || (timeDiff >= 0 && timeDiff <= 5000 && newPos > existingPos)) {
                        progress[item.path] = {
                            pos: newPos,
                            last: now
                        };
                    }
                }
                setLocalStorageItem('disco-progress', progress);
                state.playback.lastLocalUpdate = now;
            }
        }

        if (!!state.readOnly) return state.playback.pendingUpdate;

        // Server sync logic
        if (item.type.includes('audio') && duration < 420) return state.playback.pendingUpdate; // 7 minutes

        const sessionTime = (now - state.playback.startTime) / 1000;

        if (!isComplete && sessionTime < 90) return state.playback.pendingUpdate; // 90s threshold
        if (!isComplete && (now - state.playback.lastUpdate) < 30000) return state.playback.pendingUpdate; // 30s interval

        state.playback.lastUpdate = now;

        const updatePromise = (async () => {
            try {
                await fetchAPI('/api/progress', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        path: item.path,
                        playhead: isComplete ? 0 : Math.floor(playhead),
                        duration: Math.floor(duration),
                        completed: isComplete
                    })
                });
            } catch (err) {
                console.error('Failed to update progress:', err);
            } finally {
                if (state.playback.pendingUpdate === updatePromise) {
                    state.playback.pendingUpdate = null;
                }
            }
        })();

        state.playback.pendingUpdate = updatePromise;
        return updatePromise;
    }

    function getLocalProgress(item) {
        if (!state.localResume) return 0;
        const progress = JSON.parse(localStorage.getItem('disco-progress') || '{}');
        const entry = progress[item.path];
        if (!entry) return 0;

        let pos, last;
        if (typeof entry === 'object') {
            pos = entry.pos;
            last = entry.last;
        } else {
            // backward compatibility
            pos = entry;
            last = Date.now();
        }

        // Expiration rule: for audio files less than 7 mins (420s) long
        // forget progress if it has been more than 15 minutes (900s) since it was last played.
        if (item.type && item.type.includes('audio') && item.duration < 420) {
            const now = Date.now();
            if ((now - last) > 15 * 60 * 1000) {
                return 0;
            }
        }

        return pos;
    }

    function getPlayCount(item) {
        const localCounts = JSON.parse(localStorage.getItem('disco-play-counts') || '{}');
        const localCount = localCounts[item.path] || 0;
        const serverCount = item.play_count || 0;
        return serverCount + localCount;
    }

    // Predefined playback rates for stepping
    const PLAYBACK_RATES = [
        0.2,
        0.25,
        0.33,
        0.4,
        0.5,
        0.6,
        0.7,
        0.75,
        0.8,
        0.85,
        0.9,
        0.95,
        1,
        1.05,
        1.1,
        1.15,
        1.2,
        1.25,
        1.3,
        1.35,
        1.4,
        1.45,
        1.5,
        1.75,
        2,
        2.5,
        3,
        4,
        6,
        8
    ];

    // Find the nearest predefined playback rate
    function nearestPlaybackRate(rate) {
        let closest = PLAYBACK_RATES[0];
        let minDist = Math.abs(rate - closest);

        for (const r of PLAYBACK_RATES) {
            const d = Math.abs(rate - r);
            if (d < minDist) {
                minDist = d;
                closest = r;
            }
        }

        return closest;
    }

    // Step playback rate up or down through predefined rates
    function stepPlaybackRate(currentRate, direction) {
        const nearest = nearestPlaybackRate(currentRate);
        let i = PLAYBACK_RATES.indexOf(nearest);

        i += direction; // -1 slower, +1 faster

        i = Math.max(0, Math.min(PLAYBACK_RATES.length - 1, i));

        return PLAYBACK_RATES[i];
    }

    function setPlaybackRate(rate) {
        state.playbackRate = rate;
        localStorage.setItem('disco-playback-rate', String(rate));
        const speedBtn = document.getElementById('pip-speed');
        if (speedBtn) speedBtn.textContent = `${rate}x`;

        const media = pipViewer.querySelector('video, audio');
        if (media) {
            (media as any).playbackRate = rate;
        }
    }

    function playSibling(offset, isUser = false, isDelete = false) {
        const keepFullscreen = !!document.fullscreenElement;

        if (state.enableQueue) {
            const queue = state.playback.queue || [];
            if (queue.length > 0) {
                let currentIndex = state.playback.queueIndex;
                if (currentIndex === -1 && state.playback.item) {
                    currentIndex = queue.findIndex(m => m.path === state.playback.item.path);
                }

                let nextIndex;
                if (currentIndex === -1) {
                    nextIndex = offset > 0 ? 0 : queue.length - 1;
                } else {
                    if (state.playback.repeatMode === 'one' && !isUser) {
                        nextIndex = currentIndex;
                    } else {
                        nextIndex = currentIndex + offset;
                        if (state.playback.repeatMode === 'all') {
                            nextIndex = (nextIndex + queue.length) % queue.length;
                        }
                    }
                }

                if (nextIndex >= 0 && nextIndex < queue.length) {
                    openActivePlayer(queue[nextIndex], false, false, nextIndex, keepFullscreen);
                    renderQueue();
                    return;
                } else {
                    // End of queue reached
                    if (!isUser) {
                        closeActivePlayer();
                        return;
                    }
                    return;
                }
            }
            // Fallback to currentMedia if queue is empty
        }

        if (currentMedia.length === 0) return;

        if (isUser && !isDelete) {
            stopSlideshow();
        }

        // Prevent rapid skipping for automated actions
        const now = Date.now();
        if (!isUser && state.playback.lastSkipTime && (now - state.playback.lastSkipTime < 400)) {
            return;
        }
        state.playback.lastSkipTime = now;

        // Clear any pending auto-skips (errors, slideshow, etc.)
        if (state.playback.skipTimeout) {
            clearTimeout(state.playback.skipTimeout);
            state.playback.skipTimeout = null;
        }

        let currentIndex = -1;
        let itemGone = false;
        if (state.playback.item) {
            currentIndex = currentMedia.findIndex(m => m.path === state.playback.item.path);
            if (currentIndex === -1) itemGone = true;
        }

        if (currentIndex === -1) {
            currentIndex = state.playback.lastPlayedIndex;
        }

        let nextIndex;
        if (currentIndex === -1) {
            // Nothing ever played, n -> 0, p -> last
            nextIndex = offset > 0 ? 0 : currentMedia.length - 1;
        } else {
            // If the item is gone (e.g. deleted), the list shifted.
            // The next item is now at the same index.
            if (itemGone && offset > 0) {
                nextIndex = currentIndex + (offset - 1);
            } else {
                nextIndex = currentIndex + offset;
            }
        }

        const isNewSession = pipPlayer.classList.contains('hidden');

        // Helper to play a media item
        const playItem = (index) => {
            if (index >= 0 && index < currentMedia.length) {
                if (state.player === 'browser') {
                    openActivePlayer(currentMedia[index], isNewSession, false, -1, keepFullscreen);
                } else {
                    playMedia(currentMedia[index]);
                }
                return true;
            }
            return false;
        };

        // Try to play the requested index
        if (nextIndex >= 0 && nextIndex < currentMedia.length) {
            playItem(nextIndex);
            return;
        }

        // Handle pagination for both delete and non-delete operations
        if (nextIndex >= currentMedia.length && !state.filters.all && state.page === 'search') {
            const totalPages = Math.ceil(state.totalCount / state.filters.limit);
            if (state.currentPage < totalPages) {
                // End of current page, fetch next
                state.currentPage++;
                performSearch().then(() => {
                    if (currentMedia.length > 0) {
                        playItem(0);
                    }
                });
                return;
            }
        } else if (nextIndex < 0 && state.currentPage > 1 && !state.filters.all && state.page === 'search') {
            // Beginning of current page, fetch previous
            state.currentPage--;
            performSearch().then(() => {
                if (currentMedia.length > 0) {
                    playItem(currentMedia.length - 1);
                }
            });
            return;
        }

        // For delete operation: if pagination didn't apply (e.g., last page), try sibling (and vice versa)
        if (isDelete) {
            if (offset > 0 && nextIndex >= currentMedia.length) {
                // Tried to go next but hit end, try previous
                if (playItem(currentIndex - 1)) return;
            } else if (offset < 0 && nextIndex < 0) {
                // Tried to go previous but hit start, try next
                if (playItem(currentIndex + 1)) return;
            }
        }
    }

    async function rateMedia(item, score) {
        try {
            await fetchAPI('/api/rate', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ path: item.path, score: score })
            });
            if (score === 0) {
                showToast(`Unrated`, '⭐️');
            } else {
                showToast(`Rated: ${'⭐'.repeat(score)}`);
            }
            fetchRatings();
        } catch (err) {
            console.error('Failed to rate media:', err);
        }
    }

    async function markMediaPlayed(item) {
        if (state.readOnly) {
            // Local update for read-only mode
            const progress = getLocalStorageItem('disco-progress', {});
            const now = Date.now();
            // Latest wins merging with position preference
            const existing = progress[item.path];
            const existingLast = existing && typeof existing === 'object' ? existing.last : 0;
            const existingPos = existing && typeof existing === 'object' ? existing.pos : 0;
            // Marking as played (pos: 0) always takes precedence as it represents completion
            const timeDiff = now - existingLast;
            if (timeDiff > 5000 || (timeDiff >= 0 && timeDiff <= 5000)) {
                progress[item.path] = { pos: 0, last: now };
            }
            setLocalStorageItem('disco-progress', progress);

            const counts = getLocalStorageItem('disco-play-counts', {});
            counts[item.path] = (counts[item.path] || 0) + 1;
            setLocalStorageItem('disco-play-counts', counts);

            showToast('Marked as seen (Local)', '✅');
        } else {
            try {
                const resp = await fetchAPI('/api/mark-played', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ path: item.path })
                });
                if (!resp.ok) throw new Error('Action failed');
                showToast('Marked as played', '✅');
            } catch (err) {
                console.error('Failed to mark as played:', err);
                errorToast(err as any, 'Action failed');
                return;
            }
        }

        // Update current state and re-render
        const updated = (m) => {
            if (m.path === item.path) {
                if (!state.readOnly) {
                    m.play_count = (m.play_count || 0) + 1;
                }
                m.playhead = 0;
                m.time_last_played = Math.floor(Date.now() / 1000);
            }
            return m;
        };
        currentMedia = currentMedia.map(updated);
        if (state.playlistItems) state.playlistItems = state.playlistItems.map(updated);

        if (state.filters.unplayed) {
            performSearch();
        } else {
            renderResults();
        }
    }

    async function markMediaUnplayed(item) {
        if (state.readOnly) {
            // Local update for read-only mode
            const counts = getLocalStorageItem('disco-play-counts', {});
            counts[item.path] = 0;
            setLocalStorageItem('disco-play-counts', counts);

            const progress = getLocalStorageItem('disco-progress', {});
            delete progress[item.path];
            setLocalStorageItem('disco-progress', progress);

            showToast('Marked as unplayed (Local)', '⭕');
        } else {
            try {
                const resp = await fetchAPI('/api/mark-unplayed', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ path: item.path })
                });
                if (!resp.ok) throw new Error('Action failed');
                showToast('Marked as unplayed', '⭕');
            } catch (err) {
                console.error('Failed to mark as unplayed:', err);
                errorToast(err as any, 'Action failed');
                return;
            }
        }

        // Update current state and re-render
        const updated = (m) => {
            if (m.path === item.path) {
                m.play_count = 0;
                m.playhead = 0;
                m.time_last_played = 0;
            }
            return m;
        };
        currentMedia = currentMedia.map(updated);
        if (state.playlistItems) state.playlistItems = state.playlistItems.map(updated);

        if (state.filters.completed) {
            performSearch();
        } else {
            renderResults();
        }
    }

    function seekToProgress(el, targetPos, retryCount = 0) {
        if (!el || !targetPos || targetPos <= 0) return;

        // Mute during seek to prevent stuttering artifacts
        if (retryCount === 0) {
            el._systemMute = true;
            el.muted = true;
        }

        const restoreMute = () => {
            el._systemMute = false;
            el.muted = state.playback.muted;
        };

        if (retryCount > 60) {
            restoreMute();
            return;
        }

        const duration = el.duration;

        if (!isNaN(duration) && duration >= targetPos) {
            el.currentTime = targetPos;
            restoreMute();
            return;
        }

        if (!isNaN(duration) && duration > 0) {
            el.currentTime = duration;
        } else if (retryCount === 0) {
            el.currentTime = targetPos;
        }

        setTimeout(() => seekToProgress(el, targetPos, retryCount + 1), 333);
    }

    function cycleSubtitles(reverse = false) {
        const media = pipViewer.querySelector('video');
        if (!media || !(media as HTMLMediaElement).textTracks) return;

        const tracks = Array.from((media as HTMLMediaElement).textTracks).filter(t => t.kind === 'subtitles');
        if (tracks.length === 0) return;

        // Find current active track index
        let activeIndex = tracks.findIndex(t => (t as TextTrack).mode === 'showing');

        // Disable all
        tracks.forEach(t => (t as TextTrack).mode = 'disabled');

        if (reverse) {
            // Cycle: showing (0) -> none (-1) -> last (N-1) -> ...
            if (activeIndex === -1) {
                activeIndex = tracks.length - 1;
            } else {
                activeIndex--;
            }
        } else {
            // Cycle: showing (0) -> showing (1) -> ... -> last (N-1) -> none (-1) -> 0
            if (activeIndex >= tracks.length - 1) {
                activeIndex = -1;
            } else {
                activeIndex++;
            }
        }

        if (activeIndex !== -1) {
            tracks[activeIndex].mode = 'showing';
            showToast(`Subtitles: ${tracks[activeIndex].label || 'Track ' + (activeIndex + 1)}`, '💬');
        } else {
            showToast('Subtitles: Off', '💬');
        }
    }

    // Cache for subtitle cues (loaded on demand)
    let subtitleCuesCache = null;
    let subtitleCachesPath = null;

    async function fetchSubtitleCues() {
        const media = pipViewer.querySelector('video');
        if (!media || !state.playback.item) return null;

        const path = state.playback.item.path;

        // Return cached cues if same media
        if (subtitleCuesCache && subtitleCachesPath === path) {
            return subtitleCuesCache;
        }

        try {
            // Fetch subtitle track from API
            const resp = await fetchAPI(`/api/subtitles?path=${encodeURIComponent(path)}`);
            if (!resp.ok) return null;

            const text = await resp.text();
            const cues = parseWebVTT(text);

            subtitleCuesCache = cues;
            subtitleCachesPath = path;
            return cues;
        } catch (err) {
            console.error('Failed to fetch subtitle cues:', err);
            return null;
        }
    }

    function parseWebVTT(vttText) {
        const cues = [];
        const lines = vttText.split(/\r?\n/);
        let currentTimeRange = null;

        for (const line of lines) {
            const trimmed = line.trim();

            // Skip WEBVTT header and empty lines
            if (trimmed.startsWith('WEBVTT') || trimmed === '' || trimmed.startsWith('NOTE')) {
                continue;
            }

            // Parse timestamp line (e.g., "00:00:01.000 --> 00:00:04.000")
            const timestampMatch = trimmed.match(/(\d{2}:\d{2}:\d{2}\.\d{3})\s*-->\s*(\d{2}:\d{2}:\d{2}\.\d{3})/);
            if (timestampMatch) {
                currentTimeRange = {
                    start: parseVTTTime(timestampMatch[1]),
                    end: parseVTTTime(timestampMatch[2]),
                    text: ''
                };
                continue;
            }

            // Collect text for current cue
            if (currentTimeRange && trimmed) {
                if (currentTimeRange.text) {
                    currentTimeRange.text += ' ' + trimmed;
                } else {
                    currentTimeRange.text = trimmed;
                }
            }

            // Empty line marks end of cue
            if (trimmed === '' && currentTimeRange) {
                cues.push(currentTimeRange);
                currentTimeRange = null;
            }
        }

        // Don't forget the last cue if file doesn't end with empty line
        if (currentTimeRange) {
            cues.push(currentTimeRange);
        }

        return cues;
    }

    function parseVTTTime(timeStr) {
        const parts = timeStr.split(':');
        const seconds = parseFloat(parts[2]);
        const minutes = parseInt(parts[1], 10);
        const hours = parseInt(parts[0], 10);
        return hours * 3600 + minutes * 60 + seconds;
    }

    function seekToSubtitleCue(reverse = false) {
        const media = pipViewer.querySelector('video, audio');
        if (!media || !state.playback.item) return;

        fetchSubtitleCues().then(cues => {
            if (!cues || cues.length === 0) {
                showToast('No subtitles available', '💬');
                return;
            }

            const current = (media as HTMLMediaElement).currentTime || 0;
            let targetCue = null;

            if (reverse) {
                // Find the previous cue (last cue that ends before current time)
                for (let i = cues.length - 1; i >= 0; i--) {
                    if (cues[i].end <= current) {
                        targetCue = cues[i];
                        break;
                    }
                }
                // If no previous cue, go to first cue
                if (!targetCue && cues.length > 0) {
                    targetCue = cues[0];
                }
            } else {
                // Find the next cue (first cue that starts after current time)
                for (const cue of cues) {
                    if (cue.start > current) {
                        targetCue = cue;
                        break;
                    }
                }
                // If no next cue, go to last cue
                if (!targetCue && cues.length > 0) {
                    targetCue = cues[cues.length - 1];
                }
            }

            if (targetCue) {
                (media as HTMLMediaElement).currentTime = targetCue.start;
                showToast(`Subtitle: ${targetCue.text.substring(0, 50)}${targetCue.text.length > 50 ? '...' : ''}`, '💬');
            }
        });
    }

    function takeScreenshot(video, withoutSubs = false) {
        try {
            // Create canvas and draw video frame
            const canvas = document.createElement('canvas');
            canvas.width = video.videoWidth || 1920;
            canvas.height = video.videoHeight || 1080;
            const ctx = canvas.getContext('2d');

            if (withoutSubs && video.textTracks) {
                // Temporarily disable all subtitle tracks
                const tracks = Array.from(video.textTracks);
                const showingTracks = tracks.filter(t => (t as TextTrack).mode === 'showing');
                tracks.forEach(t => (t as TextTrack).mode = 'disabled');

                // Draw video without subtitles
                ctx.drawImage(video, 0, 0, canvas.width, canvas.height);

                // Restore subtitle tracks
                showingTracks.forEach(t => (t as TextTrack).mode = 'showing');
            } else {
                ctx.drawImage(video, 0, 0, canvas.width, canvas.height);
            }

            // Download the screenshot
            canvas.toBlob(blob => {
                const url = URL.createObjectURL(blob);
                const a = document.createElement('a');
                a.href = url;
                const timestamp = new Date().toISOString().replace(/[:.]/g, '-').slice(0, -5);
                a.download = `screenshot-${timestamp}.png`;
                document.body.appendChild(a);
                a.click();
                document.body.removeChild(a);
                URL.revokeObjectURL(url);
                showToast('Screenshot saved', '📸');
            }, 'image/png');
        } catch (err) {
            console.error('Screenshot failed:', err);
            errorToast(err as any, 'Screenshot failed');
        }
    }

    // Aspect ratio modes for video
    const aspectRatioModes = [
        { name: 'Default', value: '' },
        { name: '16:9', value: '16/9' },
        { name: '4:3', value: '4/3' },
        { name: '21:9', value: '21/9' },
        { name: '1:1', value: '1/1' },
        { name: 'Stretch', value: 'stretch' }
    ];
    let currentAspectRatioIndex = 0;

    function cycleAspectRatio(video) {
        currentAspectRatioIndex = (currentAspectRatioIndex + 1) % aspectRatioModes.length;
        const mode = aspectRatioModes[currentAspectRatioIndex];

        if (mode.value === 'stretch') {
            video.style.objectFit = 'fill';
            video.style.aspectRatio = '';
        } else if (mode.value) {
            video.style.objectFit = 'contain';
            video.style.aspectRatio = mode.value;
        } else {
            video.style.objectFit = '';
            video.style.aspectRatio = '';
        }

        showToast(`Aspect Ratio: ${mode.name}`, '📐');
    }

    async function playRSVP(item) {
        const originalPlayer = state.player;
        state.player = 'browser';
        const rsvpItem = { ...item, rsvp: true, type: 'video/webm' };
        await openActivePlayer(rsvpItem, true);
        state.player = originalPlayer;
    }

    async function handleMediaError(item, el) {
        if (pipLoading) pipLoading.classList.add('hidden');

        if (!state.playback.item || state.playback.item.path !== item.path) {
            return;
        }

        // Clear handlers to prevent other events (like onended) firing after error
        const media = pipViewer.querySelector('video, audio, img');
        if (media) {
            (media as HTMLMediaElement).onerror = null;
            (media as HTMLMediaElement).onended = null;
            (media as any).onload = null;
        }

        const basename = item.path.split('/').pop();
        let msg = `Playback failed: ${basename}`;
        let emoji = '⚠️';
        let removeFile = false;
        let shouldVerify = true;

        // Try to get more detailed error info from the media element
        if (el && el.error) {
            const code = el.error.code;
            if (code === 1) {
                msg = `Playback aborted: ${basename}`;
                shouldVerify = false;
            } else if (code === 2) {
                msg = `Network error: ${basename}`;
                emoji = '🌐';
                shouldVerify = false;
            } else if (code === 3) {
                msg = `Decoding failed: ${basename}`;
                emoji = '🚫';
                shouldVerify = false;
            } else if (code === 4) {
                msg = `Format not supported: ${basename}`;
                shouldVerify = true; // Ambiguous: could be 404, 403, or actual codec issue
            }
        }

        if (shouldVerify) {
            try {
                // Verify status on server
                const resp = await fetchAPI(`/api/raw?path=${encodeURIComponent(item.path)}`, { method: 'HEAD' });
                if (resp.status === 404) {
                    msg = `File not found: ${basename}`;
                    emoji = '🗑️';
                    removeFile = true;
                } else if (resp.status === 403) {
                    msg = `Access denied: ${basename}`;
                    emoji = '🚫';
                } else if (resp.status === 415) {
                    msg = `Codec/Transcode failed: ${basename}`;
                    emoji = '🚫';
                } else if (resp.status >= 500) {
                    msg = `Server error (${resp.status}): ${basename}`;
                    emoji = '❌';
                } else if (resp.status === 200) {
                    // File exists but browser failed to play it
                    if (!msg.includes('Decoding') && !msg.includes('Format') && !msg.includes('Network')) {
                        msg = `Playback failed (browser/codec error): ${basename}`;
                    }
                }
            } catch (e) {
                console.error('Failed to verify media status:', e);
                msg = `Network error: ${basename}`;
                emoji = '🌐';
            }
        }

        if (state.page === 'trash') {
            showToast(msg, '⚠️');
        } else {
            showToast(msg, emoji);
            if (removeFile) {
                // Remove from current view if applicable
                currentMedia = currentMedia.filter(m => m.path !== item.path);

                if (state.view === 'group' && state.similarityData) {
                    // Also remove from similarityData groups
                    state.similarityData.forEach(group => {
                        if (group.files) {
                            group.files = group.files.filter(m => m.path !== item.path);
                            group.count = group.files.length;
                        }
                    });
                    state.similarityData = state.similarityData.filter(group => group.count > 0);
                    renderEpisodes(state.similarityData);
                } else {
                    renderResults();
                }
            }
        }

        // Auto-skip to next (up to 30 consecutive errors)
        state.playback.consecutiveErrors = (state.playback.consecutiveErrors || 0) + 1;

        if (state.autoplay && state.playback.consecutiveErrors <= 30) {
            if (state.playback.skipTimeout) {
                clearTimeout(state.playback.skipTimeout);
            }
            state.playback.skipTimeout = setTimeout(() => {
                if (state.playback.skipTimeout) {
                    state.playback.skipTimeout = null;
                    playSibling(1);
                }
            }, 1200);
        } else {
            if (state.playback.consecutiveErrors > 30) {
                showToast('Stopped auto-skip after 30 errors', '🛑');
                state.playback.consecutiveErrors = 0;
            }
            closeActivePlayer();
        }
    }

    function updatePipVisibility() {
        const speedBtn = document.getElementById('pip-speed');
        const surfBtn = document.getElementById('channel-surf-btn');
        const streamBtn = document.getElementById('pip-stream-type');
        const theatreBtn = document.getElementById('pip-theatre');

        if (speedBtn) speedBtn.classList.toggle('hidden', !state.showPipSpeed);
        if (surfBtn) surfBtn.classList.toggle('hidden', !state.showPipSurf);
        if (streamBtn) streamBtn.classList.toggle('hidden', !state.showPipStream);

        if (theatreBtn) {
            const isMobile = window.innerWidth <= 768;
            if (isMobile && !pipPlayer.classList.contains('theatre')) {
                theatreBtn.classList.add('hidden');
            } else {
                theatreBtn.classList.remove('hidden');
            }
        }
    }

    function updateQueueVisibility() {
        const queueContainer = document.getElementById('queue-container');
        const queueList = document.getElementById('queue-list');
        if (!queueContainer || !queueList) return;

        if (state.enableQueue) {
            queueContainer.classList.remove('hidden');
            queueList.classList.toggle('expanded', state.queueExpanded);
            renderQueue();
        } else {
            queueContainer.classList.add('hidden');
        }
    }

    function saveQueue() {
        localStorage.setItem('disco-queue', String(JSON.stringify(state.playback.queue || [])));
    }

    function renderQueue() {
        const queueList = document.getElementById('queue-list');
        const queueCountBadge = document.getElementById('queue-count-badge');
        if (!queueList) return;

        queueList.innerHTML = '';
        const queue = state.playback.queue || [];
        queueCountBadge.textContent = queue.length.toString();

        queue.forEach((item, index) => {
            const queueItem = document.createElement('div');
            queueItem.className = 'queue-item';
            if (state.playback.queueIndex === index) {
                queueItem.classList.add('playing');
            }

            const type = item.type || '';
            let icon = '📄';
            if (type.includes('video')) icon = '🎬';
            else if (type.includes('audio')) icon = '🎵';
            else if (type.includes('image')) icon = '🖼️';

            const itemBasename = item.path.split('/').pop();

            queueItem.innerHTML = `
                <div class="queue-item-handle" draggable="true">☰</div>
                <div class="queue-item-thumb">${icon}</div>
                <div class="queue-item-info">
                    <div class="queue-item-title" title="${item.path}">${itemBasename}</div>
                    <div class="queue-item-meta">${formatDuration(item.duration)}</div>
                </div>
                <div class="queue-item-actions">
                    <button class="queue-action-btn remove" title="Remove from Queue">❌</button>
                </div>
            `;

            queueItem.onclick = (e) => {
                if ((e.target as HTMLElement).closest('.queue-item-title') ||
                    (e.target as HTMLElement).closest('.queue-item-handle') ||
                    (e.target as HTMLElement).closest('.queue-item-actions')) {
                    return;
                }
                playMedia(item, true, index);
            };

            (queueItem.querySelector('.remove') as HTMLElement).onclick = (e) => {
                e.stopPropagation();
                state.playback.queue.splice(index, 1);
                // Adjust queueIndex if needed
                if (state.playback.queueIndex === index) {
                    state.playback.queueIndex = -1;
                } else if (state.playback.queueIndex > index) {
                    state.playback.queueIndex--;
                }
                renderQueue();
            };

            // Drag and drop for reordering and adding new items
            const handle = queueItem.querySelector('.queue-item-handle');
            (handle as HTMLElement).ondragstart = (e: DragEvent) => { (e as DragEvent).dataTransfer.setData('application/x-disco-queue-index', index.toString());
                queueItem.style.opacity = '0.5';
            };
            (handle as HTMLElement).ondragend = () => {
                queueItem.style.opacity = '1';
            };

            queueItem.ondragover = (e) => {
                e.preventDefault();
                const rect = queueItem.getBoundingClientRect();
                const midY = rect.top + rect.height / 2;
                if (e.clientY < midY) {
                    queueItem.classList.add('drop-before');
                    queueItem.classList.remove('drop-after');
                } else {
                    queueItem.classList.add('drop-after');
                    queueItem.classList.remove('drop-before');
                }
            };
            queueItem.ondragleave = () => {
                queueItem.classList.remove('drop-before', 'drop-after');
            };
            queueItem.ondrop = (e) => {
                e.preventDefault();
                e.stopPropagation();
                const rect = queueItem.getBoundingClientRect();
                const midY = rect.top + rect.height / 2;
                const isBefore = e.clientY < midY;
                queueItem.classList.remove('drop-before', 'drop-after');

                const fromIndexStr = (e as DragEvent).dataTransfer.getData('application/x-disco-queue-index');
                const targetIndex = isBefore ? index : index + 1;

                if (fromIndexStr !== '') {
                    // Reordering existing item
                    const fromIndex = parseInt(fromIndexStr);
                    if (fromIndex !== index) {
                        const movedItem = state.playback.queue.splice(fromIndex, 1)[0];
                        // If we moved from before target, index shifted
                        const adjustedTarget = (fromIndex < targetIndex) ? targetIndex - 1 : targetIndex;
                        state.playback.queue.splice(adjustedTarget, 0, movedItem);
                        renderQueue();
                    }
                } else if (state.draggedItem) {
                    // Adding new item from outside
                    const item = state.draggedItem;
                    state.playback.queue.splice(targetIndex, 0, item);
                    showToast(`Added to queue: ${item.path.split('/').pop()}`, '➕');
                    renderQueue();
                }
            };

            queueList.appendChild(queueItem);
        });

        // Update control button states
        const playPauseBtn = document.getElementById('queue-play-pause-btn');
        const shuffleBtn = document.getElementById('queue-shuffle-btn');
        const repeatBtn = document.getElementById('queue-repeat-btn');
        const addModeBtn = document.getElementById('queue-add-mode-btn');

        if (playPauseBtn) {
            const media = pipViewer.querySelector('video, audio');
            if (media && !(media as HTMLMediaElement).paused) {
                playPauseBtn.textContent = '⏸️';
            } else {
                playPauseBtn.textContent = '▶️';
            }
            playPauseBtn.classList.toggle('hidden', !state.playback.item);
        }

        if (shuffleBtn) {
            shuffleBtn.classList.toggle('active', state.playback.shuffle);
        }

        if (repeatBtn) {
            repeatBtn.textContent = `🔁 ${state.playback.repeatMode.charAt(0).toUpperCase() + state.playback.repeatMode.slice(1)}`;
            repeatBtn.classList.toggle('active', state.playback.repeatMode !== 'off');
        }

        if (addModeBtn) {
            addModeBtn.textContent = `➕ ${state.queueAddMode.charAt(0).toUpperCase() + state.queueAddMode.slice(1)}`;
            addModeBtn.classList.toggle('active', state.queueAddMode === 'next');
        }

        const expandBtn = document.getElementById('queue-expand-btn');
        if (expandBtn) {
            expandBtn.classList.toggle('active', state.queueExpanded);
        }

        saveQueue();
    }

    function initQueueControls() {
        const queueContainer = document.getElementById('queue-container');
        const playPauseBtn = document.getElementById('queue-play-pause-btn');
        const expandBtn = document.getElementById('queue-expand-btn');
        const shuffleBtn = document.getElementById('queue-shuffle-btn');
        const repeatBtn = document.getElementById('queue-repeat-btn');
        const addModeBtn = document.getElementById('queue-add-mode-btn');
        const clearBtn = document.getElementById('queue-clear-btn');

        if (playPauseBtn) {
            playPauseBtn.onclick = () => {
                const media = pipViewer.querySelector('video, audio');
                if (media) {
                    if ((media as HTMLMediaElement).paused) (media as HTMLMediaElement).play();
                    else (media as HTMLMediaElement).pause();
                    renderQueue();
                }
            };
        }

        if (queueContainer) {
            queueContainer.addEventListener('dragover', (e) => {
                if (state.draggedItem) {
                    e.preventDefault();
                    (e as DragEvent).dataTransfer.dropEffect = 'copy';
                }
            });

            queueContainer.addEventListener('drop', (e) => {
                if (!state.draggedItem) return;

                e.preventDefault();
                const item = state.draggedItem;
                const filename = item.path.split('/').pop();

                if (state.queueAddMode === 'end') {
                    state.playback.queue.push(item);
                    showToast(`Added to queue: ${filename}`, '➕');
                } else {
                    const currentPath = state.playback.item ? state.playback.item.path : null;
                    const currentIndex = currentPath ? state.playback.queue.findIndex(m => m.path === currentPath) : -1;
                    state.playback.queue.splice(currentIndex + 1, 0, item);
                    showToast(`Added to next: ${filename}`, '⏭️');
                }
                renderQueue();
            });
        }

        if (expandBtn) {
            expandBtn.onclick = () => {
                state.queueExpanded = !state.queueExpanded;
                localStorage.setItem('disco-queue-expanded', String(state.queueExpanded));
                updateQueueVisibility();
            };
        }

        if (shuffleBtn) {
            shuffleBtn.onclick = () => {
                state.playback.shuffle = !state.playback.shuffle;
                localStorage.setItem('disco-shuffle', String(state.playback.shuffle));
                if (state.playback.shuffle) {
                    // Simple shuffle
                    for (let i = state.playback.queue.length - 1; i > 0; i--) {
                        const j = Math.floor(Math.random() * (i + 1));
                        [state.playback.queue[i], state.playback.queue[j]] = [state.playback.queue[j], state.playback.queue[i]];
                    }
                }
                renderQueue();
            };
        }

        if (repeatBtn) {
            repeatBtn.onclick = () => {
                const modes = ['off', 'all', 'one'];
                const currentIndex = modes.indexOf(state.playback.repeatMode);
                state.playback.repeatMode = modes[(currentIndex + 1) % modes.length] as any;
                localStorage.setItem('disco-repeat-mode', String(state.playback.repeatMode));
                renderQueue();
            };
        }

        if (addModeBtn) {
            addModeBtn.onclick = () => {
                state.queueAddMode = state.queueAddMode === 'end' ? 'next' : 'end';
                localStorage.setItem('disco-queue-add-mode', String(state.queueAddMode));
                renderQueue();
            };
        }

        if (clearBtn) {
            clearBtn.onclick = () => {
                if (confirm('Clear entire queue?')) {
                    state.playback.queue = [];
                    renderQueue();
                }
            };
        }
    }

    async function openInPiP(item, isNewSession = false) {
        updatePipVisibility();

        if (state.playback.slideshowTimer) {
            clearTimeout(state.playback.slideshowTimer);
            state.playback.slideshowTimer = null;
        }

        if (isNewSession) {
            // New explicit request: reset state.imageAutoplay to user preference
            state.imageAutoplay = localStorage.getItem('disco-image-autoplay') === 'true';
        }

        const type = item.type || "";

        // Reset playback rate to default for new media if not currently playing something
        if (!state.playback.item) {
            if (type.includes('video')) {
                state.playbackRate = state.defaultVideoRate;
            } else if (type.includes('audio')) {
                state.playbackRate = state.defaultAudioRate;
            } else {
                state.playbackRate = 1.0;
            }
            const speedBtn = document.getElementById('pip-speed');
            if (speedBtn) speedBtn.textContent = `${state.playbackRate}x`;
        }

        const prevItem = state.playback.item;
        const wasPlayed = state.playback.hasMarkedComplete || (prevItem && getPlayCount(prevItem) > 0);

        state.playback.item = item;
        state.playback.startTime = Date.now();
        state.playback.lastUpdate = 0;
        state.playback.hasMarkedComplete = false;
        state.playback.lastPlayedIndex = currentMedia.findIndex(m => m.path === item.path);

        // Build queue from current media list (next items to play)
        if (!state.enableQueue) {
            if (state.playback.lastPlayedIndex !== -1) {
                // Queue up the next 120 items (or fewer if at end of list)
                const queueEnd = Math.min(state.playback.lastPlayedIndex + 120, currentMedia.length);
                state.playback.queue = currentMedia.slice(state.playback.lastPlayedIndex + 1, queueEnd);
            } else {
                state.playback.queue = [];
            }
        }

        // Update Now Playing button visibility
        updateNowPlayingButton();

        // Preload next 2 images
        if (state.playback.lastPlayedIndex !== -1) {
            let count = 0;
            for (let i = state.playback.lastPlayedIndex + 1; i < currentMedia.length && count < 2; i++) {
                const nextItem = currentMedia[i];
                if (nextItem.type && nextItem.type.includes('image')) {
                    const img = new Image();
                    img.src = `/api/raw?path=${encodeURIComponent(nextItem.path)}`;
                    count++;
                }
            }
        }

        if (prevItem && prevItem.path !== item.path && state.filters.unplayed && wasPlayed) {
            if (state.playback.pendingUpdate) await state.playback.pendingUpdate;
            performSearch();
        }

        const path = item.path;
        pipTitle.textContent = path.split('/').pop();
        pipTitle.title = path;
        pipViewer.innerHTML = '';

        // Apply mode
        const theatreAnchor = document.getElementById('theatre-anchor');
        const btn = document.getElementById('pip-theatre');

        if (state.playerMode === 'theatre') {
            pipPlayer.classList.add('theatre');
            document.body.classList.remove('has-pip');
            if (pipPlayer.parentElement !== theatreAnchor) {
                theatreAnchor.appendChild(pipPlayer);
            }
            if (btn) {
                btn.textContent = '❐';
                btn.title = 'Restore to PiP';
            }
        } else {
            pipPlayer.classList.remove('theatre');
            document.body.classList.add('has-pip');
            if (pipPlayer.parentElement !== document.body) {
                document.body.appendChild(pipPlayer);
            }
            if (btn) {
                btn.textContent = '□';
                btn.title = 'Theatre Mode';
            }
        }

        pipPlayer.classList.remove('hidden');

        const slideshowBtn = document.getElementById('pip-slideshow');
        const speedBtn = document.getElementById('pip-speed');
        if (speedBtn) {
            if (type.includes('image') || !state.showPipSpeed) {
                speedBtn.classList.add('hidden');
                if (pipSpeedMenu) pipSpeedMenu.classList.add('hidden');
            } else {
                speedBtn.classList.remove('hidden');
            }
        }

        const surfBtn = document.getElementById('channel-surf-btn');
        if (surfBtn) {
            surfBtn.classList.toggle('hidden', !state.showPipSurf);
        }

        if (slideshowBtn) {
            if (type.includes('image')) {
                slideshowBtn.classList.remove('hidden');
            } else {
                slideshowBtn.classList.add('hidden');
                stopSlideshow();
            }
        }

        // Check if item needs transcoding (provided by backend)
        let needsTranscode = item.transcode === true;

        const streamBtn = document.getElementById('pip-stream-type');
        if (streamBtn) {
            streamBtn.textContent = needsTranscode ? '🔄 HLS' : '⚡ Direct';
            streamBtn.title = `Currently using ${needsTranscode ? 'Transcoding (HLS)' : 'Direct Stream'}. Click to switch.`;
            streamBtn.classList.toggle('hidden', !state.showPipStream || type.includes('image'));
        }

        const nativePipBtn = document.getElementById('pip-native');
        if (nativePipBtn) {
            if (type.includes('video') && document.pictureInPictureEnabled) {
                nativePipBtn.classList.remove('hidden');
            } else {
                nativePipBtn.classList.add('hidden');
            }
        }

        let localPos = getLocalProgress(item);
        if (!localPos && item.playhead > 0) {
            localPos = item.playhead;
        }

        // Standard raw URL (possibly sliced if using fallback)
        let url = item.rsvp ? `/api/rsvp?path=${encodeURIComponent(path)}&wpm=${state.rsvpWpm}` : `/api/raw?path=${encodeURIComponent(path)}`;

        if (state.playback.hlsInstance) {
            state.playback.hlsInstance.destroy();
            state.playback.hlsInstance = null;
        }

        if (pipLoading) {
            if (item.rsvp) {
                pipLoading.classList.remove('hidden');
            } else {
                pipLoading.classList.add('hidden');
            }
        }

        let el;

        if (type.includes('video')) {
            el = document.createElement('video');
            el.controls = true;
            el.autoplay = true;
            el.preload = 'auto';
            el.playsInline = true;
            el.muted = state.playback.muted;

            if (item.rsvp) {
                el.oncanplay = () => {
                    if (pipLoading) pipLoading.classList.add('hidden');
                };
            }

            el.onvolumechange = () => {
                if (el._systemMute) return;
                state.playback.muted = el.muted;
                localStorage.setItem('disco-muted', String(el.muted));
            };

            // Auto-Loop short media
            if (state.autoLoopMaxDuration > 0 && item.duration > 0 && item.duration <= state.autoLoopMaxDuration) {
                el.loop = true;
            }

            el.onerror = () => {
                // If this element is no longer the active one, ignore the error
                if (el !== pipViewer.querySelector('video, audio')) return;

                // Check if this is a decode error (error code 3 = MEDIA_ERR_DECODE)
                // This can happen for animated GIFs that ffprobe detected as video
                // but the browser's video element can't decode
                if (el.error && el.error.code === (typeof MediaError !== 'undefined' ? MediaError.MEDIA_ERR_DECODE : 3)) {
                    console.warn("Video decode failed, falling back to image element for:", item.path);
                    fallbackToImageElement(item, url);
                    return;
                }

                const currentSrc = el.src || '';
                if (needsTranscode && (currentSrc.includes('/api/hls/playlist') || (state.playback.hlsInstance && state.playback.hlsInstance.url === currentSrc))) {
                    console.warn("HLS failed, trying direct stream fallback...");
                    if (state.playback.hlsInstance) {
                        state.playback.hlsInstance.destroy();
                        state.playback.hlsInstance = null;
                    }
                    el.src = url;
                    el.playbackRate = state.playbackRate;
                    seekToProgress(el, localPos);
                } else {
                    handleMediaError(item, el);
                }
            };

            if (needsTranscode) {
                const hlsUrl = `/api/hls/playlist?path=${encodeURIComponent(path)}`;

                if (el.canPlayType('application/vnd.apple.mpegurl')) {
                    // Native HLS (Safari)
                    el.src = hlsUrl;
                    el.playbackRate = state.playbackRate;
                    el.addEventListener('loadedmetadata', () => {
                        seekToProgress(el, localPos);
                    }, { once: true });
                } else if (Hls.isSupported()) {
                    // hls.js
                    const hls = new (Hls as any)({
                        maxBufferLength: 60,
                        maxMaxBufferLength: 180,
                        maxBufferSize: 120 * 1000 * 1000, // 120MB
                    });
                    hls.loadSource(hlsUrl);
                    hls.attachMedia(el);
                    hls.on(Hls.Events.MANIFEST_PARSED, () => {
                        seekToProgress(el, localPos);
                        el.playbackRate = state.playbackRate;
                        el.play().catch(e => console.log("Auto-play blocked:", e));
                    });
                    hls.on(Hls.Events.ERROR, (event, data) => {
                        if (data.fatal) {
                            console.warn("HLS.js fatal error, trying direct stream fallback:", data.type);
                            hls.destroy();
                            state.playback.hlsInstance = null;
                            el.src = url;
                            el.playbackRate = state.playbackRate;
                            seekToProgress(el, localPos);
                        }
                    });
                    state.playback.hlsInstance = hls;
                } else {
                    // Fallback to sliced stream
                    el.src = url;
                }
            } else {
                el.src = url;
                el.playbackRate = state.playbackRate;
                seekToProgress(el, localPos);
            }

            el.ontimeupdate = () => {
                const isComplete = (el.duration > 90) && (el.duration - el.currentTime < 90) && (el.currentTime / el.duration > 0.95);
                updateProgress(item, el.currentTime, el.duration, isComplete);
            };

            el.onended = async () => {
                await updateProgress(item, el.duration, el.duration, true);
                handlePostPlayback(item);
            };

            const addTrack = (trackUrl, label, index) => {
                const track = document.createElement('track');
                track.kind = 'subtitles';
                track.label = label;
                (track as any).srclang = state.language || 'en';
                track.src = trackUrl; // Append start param

                track.onload = () => {
                    // Try to auto-enable
                    if (el.textTracks.length <= 1) {
                        track.track.mode = 'showing';
                    } else {
                        // If we have a language preference and this matches, switch to it
                        if (state.language && label.toLowerCase().includes(state.language.toLowerCase())) {
                            for (let i = 0; i < el.textTracks.length; i++) {
                                el.textTracks[i].mode = 'disabled';
                            }
                            track.track.mode = 'showing';
                        }
                    }
                };

                el.appendChild(track);
                // Hint to browser to load it
                if (el.textTracks.length <= 1) track.default = true;

                return track;
            };

            // 1. Add embedded tracks from metadata
            if (item.subtitle_codecs) {
                const codecs = item.subtitle_codecs.split(';');
                codecs.forEach((codec, index) => {
                    // Check if this is an external subtitle (stored with format "Language (ext)" or just "ext")
                    // External subtitles are identified by having a file extension pattern
                    const extMatch = codec.match(/\((srt|vtt|ass|ssa|lrc|idx|sub)\)$/i);
                    const isExt = extMatch !== null;
                    const fileExt = extMatch ? extMatch[1].toLowerCase() : null;

                    // Use the codec name as-is (it's already in "Language (codec)" format from backend)
                    const label = codec || `Track ${index + 1}`;

                    let trackUrl;
                    if (isExt && fileExt) {
                        // External subtitle: need to find the actual file
                        // The backend stores the display name, but we need to construct the URL
                        // Try to get the subtitle via the API which will find the external file
                        trackUrl = `/api/subtitles?path=${encodeURIComponent(path)}&ext=${fileExt}`;
                    } else {
                        // Embedded subtitle
                        trackUrl = `/api/subtitles?path=${encodeURIComponent(path)}&index=${index}`;
                    }

                    addTrack(trackUrl, label, isExt ? 'auto' : index);
                });
            }

            // 2. Always check for external subtitle file (sibling with same name)
            // This is a fallback for when external subtitles weren't scanned during import
            if (!type.includes('image')) {
                addTrack(`/api/subtitles?path=${encodeURIComponent(path)}`, 'External', 'auto');
            }

        } else if (type.includes('audio')) {
            el = document.createElement('audio');
            el.controls = true;
            el.autoplay = true;
            el.src = url;
            el.playbackRate = state.playbackRate;
            el.muted = state.playback.muted;

            el.onvolumechange = () => {
                if (el._systemMute) return;
                state.playback.muted = el.muted;
                localStorage.setItem('disco-muted', String(el.muted));
            };

            el.onerror = () => handleMediaError(item, el);

            // Auto-Loop short media
            if (state.autoLoopMaxDuration > 0 && item.duration > 0 && item.duration <= state.autoLoopMaxDuration) {
                el.loop = true;
            }

            seekToProgress(el, localPos);

            el.ontimeupdate = () => {
                const isComplete = (el.duration > 90) && (el.duration - el.currentTime < 90) && (el.currentTime / el.duration > 0.95);
                updateProgress(item, el.currentTime, el.duration, isComplete);
            };

            el.onended = async () => {
                await updateProgress(item, el.duration, el.duration, true);
                handlePostPlayback(item);
            };

            // Try to fetch lyrics (server will look for siblings)
            if (!type.includes('image')) {
                const track = document.createElement('track');
                track.kind = 'subtitles';
                track.src = `/api/subtitles?path=${encodeURIComponent(path)}`;
                (track as any).srclang = state.language || 'en';
                el.appendChild(track);

                track.onload = () => {
                    const textTrack = el.textTracks[0];
                    if (textTrack && textTrack.cues && textTrack.cues.length > 0) {
                        textTrack.mode = 'hidden';
                    }
                };
            }
        } else if (type.includes('image')) {
            el = document.createElement('img');
            el.onload = () => {
                if (state.imageAutoplay) {
                    startSlideshow();
                }
            };
            el.onerror = () => handleMediaError(item, el);
            el.src = url;
            // Handle cached images where load fires synchronously before handler is attached
            if (el.complete) {
                el.onload();
            }
            (el as HTMLElement).ondblclick = () => toggleFullscreen(pipViewer as HTMLElement);
            setupImageZoomPan(el);
        } else {
            showToast('Unsupported media format');
            return;
        }

        pipViewer.appendChild(el);
    }

    // Document reading progress tracking
    let documentProgressTimer: number | null = null;
    const DOCUMENT_PROGRESS_INTERVAL = 5000; // Save progress every 5 seconds

    function saveDocumentProgress(path: string, scrollPercent: number, chapter?: string) {
        if (!state.localResume) return;

        const progress = getLocalStorageItem('disco-progress', {});
        const now = Date.now();

        // Store document-specific progress with scroll position and optional chapter
        progress[path] = {
            pos: Math.floor(scrollPercent * 100), // Store as percentage (0-100)
            chapter: chapter || '',
            last: now
        };

        setLocalStorageItem('disco-progress', progress);
    }

    function restoreDocumentProgress(path: string): { scrollPercent: number; chapter: string } | null {
        if (!state.localResume) return null;

        const progress = getLocalStorageItem('disco-progress', {});
        const entry = progress[path];

        if (!entry || typeof entry !== 'object') return null;

        const scrollPercent = (entry.pos || 0) / 100;
        const chapter = entry.chapter || '';

        return { scrollPercent, chapter };
    }

    function trackDocumentProgress(iframe: HTMLIFrameElement, path: string) {
        // Clear existing timer
        if (documentProgressTimer !== null) {
            window.clearInterval(documentProgressTimer);
        }

        // Try to access iframe content (works for same-origin or calibre-converted content)
        const tryTrackScroll = () => {
            try {
                const iframeDoc = iframe.contentDocument || iframe.contentWindow?.document;
                if (!iframeDoc) return;

                const iframeWin = iframe.contentWindow;
                if (!iframeWin) return;

                // Get scroll position from iframe
                const scrollTop = iframeWin.scrollY || iframeDoc.documentElement.scrollTop || iframeDoc.body.scrollTop;
                const scrollHeight = iframeDoc.documentElement.scrollHeight || iframeDoc.body.scrollHeight;
                const clientHeight = iframeWin.innerHeight || iframeDoc.documentElement.clientHeight;

                if (scrollHeight > clientHeight) {
                    const scrollPercent = scrollTop / (scrollHeight - clientHeight);
                    saveDocumentProgress(path, scrollPercent);
                }
            } catch (e) {
                // Cross-origin restriction - can't access iframe content
                // Progress will only be saved when document is closed
            }
        };

        // Track periodically while reading
        documentProgressTimer = window.setInterval(tryTrackScroll, DOCUMENT_PROGRESS_INTERVAL);

        // Also save on unload
        iframe.addEventListener('beforeunload', () => {
            tryTrackScroll();
        });
    }

    function applyDocumentProgress(iframe: HTMLIFrameElement, path: string) {
        const saved = restoreDocumentProgress(path);
        if (!saved) return;

        const { scrollPercent, chapter } = saved;

        // Wait for iframe to load, then restore position
        iframe.addEventListener('load', () => {
            // Give browser 1 second to stabilize layout
            setTimeout(() => {
                try {
                    const iframeDoc = iframe.contentDocument || iframe.contentWindow?.document;
                    const iframeWin = iframe.contentWindow;

                    if (!iframeDoc || !iframeWin) return;

                    const scrollHeight = iframeDoc.documentElement.scrollHeight || iframeDoc.body.scrollHeight;
                    const clientHeight = iframeWin.innerHeight || iframeDoc.documentElement.clientHeight;

                    if (scrollHeight > clientHeight) {
                        const targetScroll = scrollPercent * (scrollHeight - clientHeight);
                        iframeWin.scrollTo({ top: targetScroll, behavior: 'smooth' });

                        // Show toast with resume position
                        if (scrollPercent > 0.05) { // Only show if > 5% progress
                            const percent = Math.round(scrollPercent * 100);
                            const msg = chapter
                                ? `Resumed at chapter: ${chapter} (${percent}%)`
                                : `Resumed at ${percent}%`;
                            showToast(msg, '📖');
                        }
                    }
                } catch (e) {
                    // Cross-origin or other error - silently fail
                }
            }, 1000); // 1 second delay for browser to adjust
        });
    }

    function openInDocumentViewer(item) {
        const modal = document.getElementById('document-modal');
        const title = document.getElementById('document-title');
        const container = document.getElementById('document-container');

        openModal('document-modal');

        title.textContent = item.path.split('/').pop(); // Use the full filename directly
        title.title = item.path;

        // Add explicit text selection on click for the title
        title.onclick = () => {
            const selection = window.getSelection();
            const range = document.createRange();
            range.selectNodeContents(title);
            selection.removeAllRanges();
            selection.addRange(range);
        };
        // Set playback state for keyboard shortcuts (delete, etc.)
        state.playback.item = item;
        state.playback.lastPlayedIndex = currentMedia.findIndex(m => m.path === item.path);

        // Clear previous viewer content
        container.innerHTML = '';
        // Setup fullscreen button
        const fsBtn = document.getElementById('doc-fullscreen');
        if (fsBtn) {
            fsBtn.classList.remove('hidden');
            fsBtn.onclick = () => toggleFullscreen(container);
        }

        // Setup RSVP button
        const rsvpBtn = document.getElementById('doc-rsvp');
        if (rsvpBtn) {
            rsvpBtn.onclick = () => {
                closeActivePlayer();
                playRSVP(item);
            };
        }

        // Check if this is a calibre-supported format that should be converted
        const ext = item.path.split('.').pop().toLowerCase();
        const calibreFormats = ['epub', 'mobi', 'azw', 'azw3', 'fb2', 'djvu', 'cbz', 'cbr', 'docx', 'odt', 'rtf', 'txt', 'md', 'html', 'htm', 'pdf'];

        if (calibreFormats.includes(ext)) {
            // Set default transcode state for text files if not set
            // PDF defaults to Direct (doc_transcode: false)
            // Other formats default to HTML (doc_transcode: true)
            if (item.doc_transcode === undefined) {
                item.doc_transcode = ext !== 'pdf';
            }

            const streamBtn = document.getElementById('doc-stream-type');
            if (streamBtn) {
                streamBtn.classList.toggle('hidden', !state.showPipStream);
                streamBtn.textContent = item.doc_transcode ? '🔄 HTML' : '⚡ Direct';
                streamBtn.title = `Currently using ${item.doc_transcode ? 'Calibre (HTML)' : 'Direct (Raw)'}. Click to switch.`;
                streamBtn.onclick = () => {
                    item.doc_transcode = !item.doc_transcode;
                    openInDocumentViewer(item);
                };
            }

            if (item.doc_transcode) {
                // Use calibre conversion endpoint - serves index.html from extracted HTML
                const pathWithoutLeadingSlash = item.path.replace(/^\/+/, '');
                const encodedPath = encodeURIComponent(pathWithoutLeadingSlash);
                const htmlUrl = `/api/epub/${encodedPath}`;

                const iframe = document.createElement('iframe');
                iframe.src = htmlUrl;
                iframe.style.width = '100%';
                iframe.style.height = '100%';
                iframe.style.border = 'none';

                iframe.onerror = () => {
                    console.error('Failed to load converted document, falling back to raw');
                    iframe.src = `/api/raw?path=${encodeURIComponent(item.path)}`;
                };

                container.appendChild(iframe);

                // Track and restore reading progress
                trackDocumentProgress(iframe, item.path);
                applyDocumentProgress(iframe, item.path);
            } else {
                // Serve raw file directly (e.g. PDF or raw text/markdown)
                const rawUrl = `/api/raw?path=${encodeURIComponent(item.path)}`;
                const iframe = document.createElement('iframe');
                iframe.src = rawUrl;
                iframe.style.width = '100%';
                iframe.style.height = '100%';
                iframe.style.border = 'none';
                container.appendChild(iframe);

                // For raw files, try to restore progress (works for same-origin)
                applyDocumentProgress(iframe, item.path);
            }
        }
        else {
            const streamBtn = document.getElementById('doc-stream-type');
            if (streamBtn) streamBtn.classList.add('hidden');

            // Use raw endpoint for unknown formats
            const url = `/api/raw?path=${encodeURIComponent(item.path)}`;

            // Use iframe for all document types - modern browsers have built-in PDF viewers
            // and browser extensions handle EPUB files better than any JS library
            const iframe = document.createElement('iframe');
            iframe.src = url;
            iframe.style.width = '100%';
            iframe.style.height = '100%';
            iframe.style.border = 'none';
            container.appendChild(iframe);

            // Try to restore progress
            applyDocumentProgress(iframe, item.path);
        }
    }

    function showMetadata(item) {
        if (!item) return;
        const content = document.getElementById('metadata-content');
        if (!content) return;

        const formatValue = (key, val) => {
            if (val === null || val === undefined || val === '') return '-';
            if (key.startsWith('time_') || key.endsWith('_played') || key.includes('_uploaded') || key.includes('_downloaded')) {
                return new Date(val * 1000).toLocaleString();
            }
            if (key === 'size') return formatSize(val);
            if (key === 'duration') return formatDuration(val);
            if (typeof val === 'number') return val.toLocaleString();
            return val;
        };

        const keys = Object.keys(item).sort().filter(k => {
            const val = item[k];
            if (val === null || val === undefined || val === '') return false;
            if (k === 'db' || k === 'transcode' || k === 'track_number') return false;

            // Hide 0 values for timestamps and other numeric fields where 0 means "unset"
            if (val === 0 || val === 0.0 || val === '0') {
                if (k.startsWith('time_') || k === 'playhead' || k === 'play_count' || k === 'score' || k === 'upvote_ratio') return false;
            }
            return true;
        });

        content.innerHTML = keys.map(k => {
            const label = k.replace(/_/g, ' ').replace(/\b\w/g, c => c.toUpperCase());
            return `<div>${label}</div><div>${formatValue(k, item[k])}</div>`;
        }).join('');

        openModal('metadata-modal');
    }

    function startSlideshow() {
        if (!state.playback.item) return;
        state.imageAutoplay = true;

        if (state.playback.slideshowTimer) {
            clearTimeout(state.playback.slideshowTimer);
            clearInterval(state.playback.slideshowTimer);
        }

        const btn = document.getElementById('pip-slideshow');
        if (btn) {
            btn.textContent = '⏸️';
            btn.classList.add('active');
        }

        state.playback.slideshowTimer = setTimeout(() => {
            state.playback.slideshowTimer = null;
            playSibling(1);
            startSlideshow();
        }, state.slideshowDelay * 1000);
    }

    function stopSlideshow() {
        state.imageAutoplay = false;
        if (state.playback.slideshowTimer) {
            clearTimeout(state.playback.slideshowTimer);
            clearInterval(state.playback.slideshowTimer);
            state.playback.slideshowTimer = null;
        }
        const btn = document.getElementById('pip-slideshow');
        if (btn) {
            btn.textContent = '▶️';
            btn.classList.remove('active');
        }
    }

    function toggleFullscreen(el) {
        if (!el) return;

        if (document.fullscreenElement) {
            document.exitFullscreen().catch(err => {
                console.error(`Error attempting to exit full-screen mode: ${err.message}`);
            });
        } else {
            el.requestFullscreen().catch(err => {
                console.error(`Error attempting to enable full-screen mode: ${err.message}`);
            });
        }
    }

    // Setup zoom/pan functionality for image elements in fullscreen
    function setupImageZoomPan(el) {
        if (!el) return;

        let scale = 1;
        let translateX = 0;
        let translateY = 0;
        let isDragging = false;
        let lastX, lastY;

        el.addEventListener('wheel', (e) => {
            if (!document.fullscreenElement) return;
            e.preventDefault();
            const delta = e.deltaY > 0 ? 0.9 : 1.1;
            const newScale = Math.min(Math.max(1, scale * delta), 10);

            if (newScale !== scale) {
                scale = newScale;
                if (scale === 1) {
                    translateX = 0;
                    translateY = 0;
                }
                el.style.transform = `scale(${scale}) translate(${translateX}px, ${translateY}px)`;
            }
        }, { passive: false });

        el.addEventListener('mousedown', (e) => {
            if (!document.fullscreenElement || scale <= 1) return;
            isDragging = true;
            lastX = e.clientX;
            lastY = e.clientY;
            el.style.cursor = 'grabbing';
        });

        window.addEventListener('mousemove', (e) => {
            if (!isDragging || !document.fullscreenElement) return;
            const dx = (e.clientX - lastX) / scale;
            const dy = (e.clientY - lastY) / scale;
            translateX += dx;
            translateY += dy;
            lastX = e.clientX;
            lastY = e.clientY;
            el.style.transform = `scale(${scale}) translate(${translateX}px, ${translateY}px)`;
        });

        window.addEventListener('mouseup', () => {
            isDragging = false;
            if (el) el.style.cursor = '';
        });

        document.addEventListener('fullscreenchange', () => {
            if (!document.fullscreenElement) {
                scale = 1;
                translateX = 0;
                translateY = 0;
                if (el) {
                    el.style.transform = '';
                    el.style.cursor = '';
                }
            }
        });
    }

    // Fallback function for when video element can't decode a file (e.g., animated GIFs)
    // Replaces the video element with an img element
    function fallbackToImageElement(item, url) {
        const imgEl = document.createElement('img');
        imgEl.onload = () => {
            if (state.imageAutoplay) {
                startSlideshow();
            }
        };
        imgEl.onerror = () => handleMediaError(item, imgEl);
        imgEl.src = url;
        if (imgEl.complete && imgEl.onload) {
            (imgEl as any).onload(new Event("load"));
        }
        imgEl.ondblclick = () => toggleFullscreen(pipViewer as HTMLElement);
        (imgEl as any).controls = false;

        // Replace video with img in the viewer
        pipViewer.innerHTML = '';
        pipViewer.appendChild(imgEl);
        setupImageZoomPan(imgEl);
    }

    async function closePiP() {
        await closeActivePlayer();
    }

    function renderPagination() {
        if (state.filters.all || state.page === 'curation') {
            paginationContainer.classList.add('hidden');
            return;
        }

        paginationContainer.classList.remove('hidden');

        const totalPages = Math.ceil(state.totalCount / state.filters.limit);
        if (totalPages > 0) {
            pageInfo.textContent = `Page ${state.currentPage} of ${totalPages}`;
        } else {
            pageInfo.textContent = `Page ${state.currentPage}`;
        }

        (prevPageBtn as HTMLButtonElement).disabled = state.currentPage === 1;
        (nextPageBtn as HTMLButtonElement).disabled = state.currentPage >= totalPages;
    }

    function showSkeletons() {
        const count = state.filters.all ? 20 : Math.min(state.filters.limit, 20);
        resultsContainer.innerHTML = '';
        resultsContainer.className = 'grid';
        for (let i = 0; i < count; i++) {
            const skeleton = document.createElement('div');
            skeleton.className = 'media-card skeleton';
            skeleton.innerHTML = `
                <div class="media-thumb"></div>
                <div class="media-info">
                    <div class="media-title">&nbsp;</div>
                    <div class="media-meta">
                        <span>&nbsp;</span>
                        <span>&nbsp;</span>
                    </div>
                </div>
            `;
            resultsContainer.appendChild(skeleton);
        }
    }

    // --- Rendering ---
    function renderResults() {
        if (!currentMedia) currentMedia = [];

        // Hide DU toolbar when not in DU view
        const duToolbar = document.getElementById('du-toolbar');
        if (duToolbar && state.page !== 'du') {
            duToolbar.classList.add('hidden');
        }

        // Prevent scroll jump by keeping current height temporarily
        const currentHeight = resultsContainer.offsetHeight;
        if (currentHeight > 0) {
            resultsContainer.style.minHeight = `${currentHeight}px`;
        }

        if (state.page === 'trash') {
            const unit = currentMedia.length === 1 ? 'file' : 'files';
            resultsCount.innerHTML = `<span>${currentMedia.length} ${unit} in trash</span> <button id="empty-bin-btn" class="category-btn" style="margin-left: 1rem; background: #e74c3c; color: white;">Empty Bin</button>`;
            const emptyBtn = document.getElementById('empty-bin-btn');
            if (emptyBtn) emptyBtn.onclick = emptyBin;
        } else if (state.page === 'history') {
            // Show filtered count for history filters, otherwise show server total
            const displayCount = (state.filters.unplayed || state.filters.unfinished || state.filters.completed) ? currentMedia.length : state.totalCount;
            const unit = displayCount === 1 ? 'result' : 'results';
            resultsCount.textContent = `${displayCount} recently played ${unit}`;
        } else if (state.page === 'playlist') {
            const unit = currentMedia.length === 1 ? 'result' : 'results';
            resultsCount.textContent = `${currentMedia.length} ${unit} in ${state.filters.playlist || 'playlist'}`;
        } else {
            // Show filtered count for client-side filters (unplayed/unfinished/completed), otherwise show server total
            // Database filtering is now done server-side, so excludedDbs doesn't count as client filter
            const hasClientFilter = state.filters.unplayed || state.filters.unfinished || state.filters.completed;
            const displayCount = hasClientFilter ? currentMedia.length : state.totalCount;
            const unit = displayCount === 1 ? 'result' : 'results';
            resultsCount.textContent = `${displayCount} ${unit}`;
        }

        if (currentMedia.length === 0) {
            // Remove grid class so empty state can be full width
            resultsContainer.className = 'no-results-container';
            resultsContainer.innerHTML = `
                <div class="no-results" style="
                    display: flex;
                    flex-direction: column;
                    align-items: center;
                    justify-content: center;
                    padding: 2rem;
                    text-align: center;
                    color: var(--text-muted);
                    max-width: 500px;
                    margin: 0 auto;
                ">
                    <div style="font-size: 4rem; margin-bottom: 1rem; opacity: 0.5;">🎒</div>
                    <h2 style="margin: 0 0 0.5rem 0; color: var(--text);">No media found</h2>
                    <p style="margin: 0; max-width: 400px;">
                        ${state.filters.search ?
                    `No results for "${state.filters.search}". Try adjusting your search or filters.` :
                    'Try adjusting your filters or add some media to your library.'}
                    </p>
                    ${state.filters.types.length > 0 || state.filters.categories.length > 0 || state.filters.sizes.length > 0 || state.filters.durations.length > 0 ? `
                        <button class="category-btn" onclick="window.disco.resetFilters()" style="margin-top: 1.5rem;">
                            Clear all filters
                        </button>
                    ` : ''}
                </div>
            `;
            resultsContainer.style.minHeight = '';
            paginationContainer.classList.add('hidden');
            return;
        }

        if (state.view === 'details') {
            renderDetailsTable();
            renderPagination();
            resultsContainer.style.minHeight = '';
            return;
        }

        if (state.page === 'captions') {
            renderCaptionsList();
            resultsContainer.style.minHeight = '';
            return;
        }

        const fragment = document.createDocumentFragment();
        resultsContainer.className = 'grid';

        currentMedia.forEach((item, index) => {
            const card = document.createElement('div');
            card.className = 'media-card';
            (card as HTMLElement).dataset.path = item.path;
            (card as HTMLElement).dataset.type = item.type || '';
            if (item.is_dir) (card as HTMLElement).dataset.isDir = 'true';
            (card as any)._item = item;
            card.draggable = true;

            card.addEventListener('dragstart', (e) => {
                state.draggedItem = item;
                (e as DragEvent).dataTransfer.effectAllowed = 'all';
                (e as DragEvent).dataTransfer.setData('text/plain', item.path);
                card.classList.add('dragging');
                document.body.classList.add('is-dragging');
            });

            card.addEventListener('dragend', () => {
                card.classList.remove('dragging');
                document.body.classList.remove('is-dragging');
                state.draggedItem = null;
                clearAllDragOver();
            });

            (card as HTMLElement).onclick = (e) => {
                if ((e.target as HTMLElement).closest('.media-actions') || (e.target as HTMLElement).closest('.media-action-btn')) return;

                if (item.is_dir) {
                    (searchInput as HTMLInputElement).value = item.path.endsWith('/') ? item.path : item.path + '/';
                    performSearch();
                    return;
                }

                if (item.path.toLowerCase().endsWith('.zim')) {
                    window.open(`/api/zim/view?path=${encodeURIComponent(item.path)}`, '_blank');
                    return;
                }

                const isCaptionClick = (e.target as HTMLElement).closest('.caption-highlight');
                if (isCaptionClick && item.caption_time) {
                    playMedia(item).then(() => {
                        const media = pipViewer.querySelector('video, audio');
                        if (media) (media as HTMLMediaElement).currentTime = item.caption_time;
                    });
                } else {
                    playMedia(item);
                }
            };

            const title = item.title || item.path.split('/').pop();
            const displayPath = formatParents(item.path);
            const size = formatSize(item.size);
            const duration = formatDuration(item.duration);
            const plays = getPlayCount(item);
            const thumbUrl = `/api/thumbnail?path=${encodeURIComponent(item.path)}`;

            const localPos = getLocalProgress(item);
            const playhead = (localPos > 0) ? localPos : (item.playhead || 0);
            const progress = (item.duration && playhead) ? Math.round((playhead / item.duration) * 100) : 0;
            const progressHtml = progress > 0 ? `
                <div class="progress-container" title="${progress}% completed">
                    <div class="progress-bar" style="width: ${progress}%"></div>
                </div>
            ` : '';

            const captionHtml = item.caption_text ? `
                <div class="caption-highlight" title="Click to play at this time">
                    "…${item.caption_text}…"
                    <span class="caption-time">${formatDuration(item.caption_time)}</span>
                </div>
            ` : '';

            const isTrash = state.page === 'trash';
            const isPlaylist = state.page === 'playlist';

            let actionBtns = '';
            if (isTrash) {
                actionBtns = `
                    <button class="media-action-btn restore" title="Restore">↺</button>
                    <button class="media-action-btn delete-permanent" title="Permanently Delete">🔥</button>
                `;
            } else if (isPlaylist) {
                actionBtns = `
                    ${!state.readOnly ? `<button class="media-action-btn remove-playlist" title="Remove from Playlist">&times;</button>` : ''}
                `;
            } else {
                actionBtns = `
                    ${!state.readOnly ? `<button class="media-action-btn add-playlist" title="Add to Playlist">+</button>` : ''}
                    ${plays > 0 ?
                        `<button class="media-action-btn mark-unplayed" title="Mark as Unplayed">⭕</button>` :
                        `<button class="media-action-btn mark-played" title="Mark as Played">✅</button>`
                    }
                    ${!state.readOnly ? `<button class="media-action-btn delete" title="Delete">🗑️</button>` : ''}
                `;
            }

            let thumbHtml = `
                <img src="${thumbUrl}" loading="lazy" onload="this.classList.add('loaded')" onerror="this.style.display='none'; this.nextElementSibling.style.display='block'">
                <i style="display: none">${getIcon(item.type)}</i>
            `;

            if (item.is_dir) {
                thumbHtml = `<div style="width:100%; height:100%; display:flex; align-items:center; justify-content:center; background:var(--sidebar-bg); font-size:4rem;">📂</div>`;
            }

            card.innerHTML = `
                <div class="media-thumb">
                    ${thumbHtml}
                    ${duration ? `<span class="media-duration">${duration}</span>` : ''}
                    <div class="media-actions">
                        ${actionBtns}
                    </div>
                </div>
                <div class="media-info">
                    <div class="media-title" title="${item.path}">${title}</div>
                    <div class="media-meta">
                        <span>${size}</span>

                        <span title="${item.path}">${displayPath}</span>
                        ${plays > 0 ? `<span title="Play count">▶️ ${plays}</span>` : ''}
                    </div>
                    ${progressHtml}
                    ${captionHtml}
                </div>
            `;

            // Reordering logic within a playlist
            if (isPlaylist) {
                card.addEventListener('dragover', (e) => {
                    e.preventDefault();
                    (e as DragEvent).dataTransfer.dropEffect = 'move';

                    const rect = card.getBoundingClientRect();
                    const x = e.clientX - rect.left;
                    if (x < rect.width / 2) {
                        card.style.borderLeft = '4px solid var(--accent-color)';
                        card.style.borderRight = '';
                    } else {
                        card.style.borderLeft = '';
                        card.style.borderRight = '4px solid var(--accent-color)';
                    }
                });

                card.addEventListener('dragleave', () => {
                    card.style.borderLeft = '';
                    card.style.borderRight = '';
                });

                card.addEventListener('drop', (e) => {
                    e.preventDefault();
                    card.style.borderLeft = '';
                    card.style.borderRight = '';

                    if (state.draggedItem && state.draggedItem !== item) {
                        const rect = card.getBoundingClientRect();
                        const x = e.clientX - rect.left;
                        let dropIndex = index;

                        // Calculate if we drop before (left) or after (right)
                        if (x > rect.width / 2) {
                            dropIndex = index + 1;
                        }

                        // Calculate dragged index
                        const draggedIndex = currentMedia.findIndex(m => m.path === state.draggedItem.path);
                        if (draggedIndex !== -1 && draggedIndex < dropIndex) {
                            dropIndex--;
                        }

                        handlePlaylistReorder(state.draggedItem, dropIndex);
                    }
                });
            }

            const btnDelete = card.querySelector('.media-action-btn.delete');
            if (btnDelete) (btnDelete as HTMLElement).onclick = (e) => {
                e.stopPropagation();
                deleteMedia(item.path, false);
            };

            const btnRSVP = card.querySelector('.media-action-btn.rsvp');
            if (btnRSVP) (btnRSVP as HTMLElement).onclick = (e) => {
                e.stopPropagation();
                playRSVP(item);
            };

            const btnRestore = card.querySelector('.media-action-btn.restore');
            if (btnRestore) (btnRestore as HTMLElement).onclick = (e) => {
                e.stopPropagation();
                deleteMedia(item.path, true);
            };

            const btnDeletePermanent = card.querySelector('.media-action-btn.delete-permanent');
            if (btnDeletePermanent) (btnDeletePermanent as HTMLElement).onclick = (e) => {
                e.stopPropagation();
                permanentlyDeleteMedia(item.path);
            };

            const btnAddPlaylist = card.querySelector('.media-action-btn.add-playlist');
            if (btnAddPlaylist) (btnAddPlaylist as HTMLElement).onclick = (e) => {
                e.stopPropagation();
                if (state.playlists.length === 0) {
                    showToast('Create a playlist first');
                    return;
                }
                // For simplicity, just add to the first playlist if only one, or prompt
                if (state.playlists.length === 1) {
                    addToPlaylist(state.playlists[0], item);
                } else {
                    const names = state.playlists.map((title, i) => `${i + 1}: ${title}`).join('\n');
                    const choice = prompt(`Add to which playlist?\n${names}`);
                    const idx = parseInt(choice) - 1;
                    if (state.playlists[idx]) {
                        addToPlaylist(state.playlists[idx], item);
                    }
                }
            };

            const btnMarkPlayed = card.querySelector('.media-action-btn.mark-played');
            if (btnMarkPlayed) (btnMarkPlayed as HTMLElement).onclick = (e) => {
                e.stopPropagation();
                markMediaPlayed(item);
            };

            const btnMarkUnplayed = card.querySelector('.media-action-btn.mark-unplayed');
            if (btnMarkUnplayed) (btnMarkUnplayed as HTMLElement).onclick = (e) => {
                e.stopPropagation();
                markMediaUnplayed(item);
            };

            const btnRemovePlaylist = card.querySelector('.media-action-btn.remove-playlist');
            if (btnRemovePlaylist) (btnRemovePlaylist as HTMLElement).onclick = (e) => {
                e.stopPropagation();
                removeFromPlaylist(state.filters.playlist, item);
            };

            fragment.appendChild(card);
        });

        resultsContainer.innerHTML = '';
        resultsContainer.appendChild(fragment);
        renderPagination();
        updateNowPlayingButton();

        // Reset min-height after content is loaded
        resultsContainer.style.minHeight = '';
    }

    function renderCaptionsList() {
        resultsContainer.innerHTML = '';

        const fragment = document.createDocumentFragment();

        // Filter out items without captions
        const itemsWithCaptions = currentMedia.filter(item =>
            item.caption_text && item.caption_text.trim() !== '' &&
            item.caption_time !== null && item.caption_time !== undefined
        );

        if (itemsWithCaptions.length === 0) {
            resultsContainer.className = 'no-results-container';
            resultsContainer.innerHTML = `
                <div class="no-results" style="display: flex; flex-direction: column; align-items: center; justify-content: center; padding: 2rem; text-align: center; color: var(--text-muted); max-width: 500px; margin: 0 auto;">
                    <div style="font-size: 4rem; margin-bottom: 1rem; opacity: 0.5;">💬</div>
                    <h2 style="margin: 0 0 0.5rem 0; color: var(--text);">No captions found</h2>
                    <p style="margin: 0; max-width: 400px;">Try adjusting your search or filters.</p>
                </div>
            `;
            return;
        }

        // Group captions by media path
        const captionsByPath = {};
        itemsWithCaptions.forEach(item => {
            if (!captionsByPath[item.path]) {
                captionsByPath[item.path] = [];
            }
            captionsByPath[item.path].push(item);
        });

        // Sort captions within each group by time
        Object.keys(captionsByPath).forEach(path => {
            captionsByPath[path].sort((a, b) => {
                const timeA = a.caption_time || 0;
                const timeB = b.caption_time || 0;
                return timeA - timeB;
            });
        });

        // Handle different view modes
        if (state.view === 'details') {
            // Details mode: show table with aggregated info per path
            renderCaptionsDetails(captionsByPath, fragment);
        } else if (state.view === 'group') {
            // Group mode: custom grouped view
            renderCaptionsGroup(captionsByPath, fragment);
        } else {
            // Default grid view
            renderCaptionsGrid(captionsByPath, fragment);
        }

        resultsContainer.appendChild(fragment);
        if (state.view === 'details') {
            resultsContainer.className = 'details-view';
        } else if (state.view === 'group') {
            resultsContainer.className = 'captions-group-view';
        } else {
            resultsContainer.className = 'captions-list-view';
        }
        updateNowPlayingButton();
    }

    function renderCaptionsGroup(captionsByPath, fragment) {
        Object.keys(captionsByPath).forEach(path => {
            const captions = captionsByPath[path];
            const group = document.createElement('div');
            group.className = 'caption-group';

            const basename = path.split('/').pop();
            const thumbUrl = `/api/thumbnail?path=${encodeURIComponent(path)}`;
            const firstCap = captions[0];

            let segmentsHtml = '';
            captions.forEach(cap => {
                const timeStr = formatDuration(cap.caption_time);
                segmentsHtml += `
                    <div class="caption-group-segment" data-time="${cap.caption_time}">
                        <span class="caption-group-time">${timeStr}</span>
                        <span class="caption-group-text">${cap.caption_text}</span>
                    </div>
                `;
            });

            group.innerHTML = `
                <div class="caption-group-header">
                    <img class="caption-group-thumb" src="${thumbUrl}" loading="lazy" onload="this.classList.add('loaded')">
                    <div class="caption-group-info">
                        <div class="caption-group-title" title="${path}">${basename}</div>
                        <div class="caption-group-meta">${captions.length} captions found • ${formatSize(firstCap.size)}</div>
                    </div>
                    <button class="queue-control-btn play-group" title="Play Media">▶️ Play</button>
                </div>
                <div class="caption-group-segments">
                    ${segmentsHtml}
                </div>
            `;

            (group.querySelector('.play-group') as HTMLElement).onclick = (e) => {
                e.stopPropagation();
                playMedia(firstCap);
            };

            group.querySelectorAll('.caption-group-segment').forEach(seg => {
                (seg as HTMLElement).onclick = (e) => {
                    e.stopPropagation();
                    const time = parseFloat((seg as HTMLElement).dataset.time);
                    playMedia(firstCap).then(() => {
                        const media = pipViewer.querySelector('video, audio');
                        if (media) {
                            (media as HTMLMediaElement).currentTime = time;
                            // Highlight
                            seg.classList.add('playing');
                            setTimeout(() => seg.classList.remove('playing'), 3000);
                        }
                    });
                };
            });

            fragment.appendChild(group);
        });
    }

    function renderCaptionsGrid(captionsByPath, fragment) {
        // Render grouped captions as cards
        Object.keys(captionsByPath).forEach(path => {
            const captions = captionsByPath[path];
            const card = document.createElement('div');
            card.className = 'media-card caption-media-card';
            (card as HTMLElement).dataset.path = path;

            const basename = path.split('/').pop();
            const thumbUrl = `/api/thumbnail?path=${encodeURIComponent(path)}`;
            const firstCap = captions[0];

            // Get caption count from aggregated data or count manually
            const captionCount = firstCap.caption_count || captions.length;

            // Build caption segments HTML - show all segments
            let captionsHtml = '';
            for (let i = 0; i < captions.length; i++) {
                const cap = captions[i];
                const timeStr = formatDuration(cap.caption_time);
                const isMatch = cap._isMatch;
                captionsHtml += `
                    <div class="caption-segment ${isMatch ? 'caption-match' : ''}" data-time="${cap.caption_time}">
                        <span class="caption-time-link" title="Jump to ${timeStr}">${timeStr}</span>
                        <span class="caption-text">${cap.caption_text}</span>
                    </div>
                `;
            }

            card.innerHTML = `
                <div class="caption-media-header">
                    <div class="media-thumb">
                        <img src="${thumbUrl}" loading="lazy" onload="this.classList.add('loaded')">
                        <span class="caption-count-badge">${captionCount}</span>
                    </div>
                    <div class="media-info">
                        <div class="media-title" title="${path}">${basename}</div>
                    </div>
                </div>
                <div class="caption-segments-container">
                    ${captionsHtml}
                </div>
            `;

            // Click on thumb plays media
            const thumb = card.querySelector('.media-thumb');
            (thumb as HTMLElement).onclick = (e) => {
                e.stopPropagation();
                playMedia(captions[0]);
            };

            // Click on caption segment jumps to that time
            card.querySelectorAll('.caption-segment').forEach(seg => {
                (seg as HTMLElement).onclick = (e) => {
                    e.stopPropagation();
                    const time = parseFloat((seg as HTMLElement).dataset.time);
                    playMedia(captions[0]).then(() => {
                        const media = pipViewer.querySelector('video, audio');
                        if (media) {
                            (media as HTMLMediaElement).currentTime = time;
                            // Highlight the segment briefly
                            seg.classList.add('caption-playing');
                            setTimeout(() => seg.classList.remove('caption-playing'), 2000);
                        }
                    });
                };
            });

            fragment.appendChild(card);
        });
    }

    function renderCaptionsDetails(captionsByPath, fragment) {
        // Create details table for captions
        const table = document.createElement('table');
        table.className = 'details-table';
        table.innerHTML = `
            <thead>
                <tr>
                    <th>Path</th>
                    <th>Type</th>
                    <th>Size</th>
                    <th>Duration</th>
                    <th>Captions</th>
                    <th>First Caption</th>
                </tr>
            </thead>
            <tbody></tbody>
        `;

        const tbody = table.querySelector('tbody');

        Object.keys(captionsByPath).forEach(path => {
            const captions = captionsByPath[path];
            const firstCap = captions[0];
            const tr = document.createElement('tr');
            (tr as HTMLElement).dataset.path = path;

            const size = formatSize(firstCap.size || 0);
            const duration = formatDuration(firstCap.duration || 0);

            tr.innerHTML = `
                <td>${path}</td>
                <td>${firstCap.type || 'unknown'}</td>
                <td>${size}</td>
                <td>${duration}</td>
                <td>${captions.length}</td>
                <td>${firstCap.caption_text ? firstCap.caption_text.substring(0, 50) + (firstCap.caption_text.length > 50 ? '...' : '') : ''}</td>
            `;

            tr.onclick = () => {
                playMedia(firstCap).then(() => {
                    const media = pipViewer.querySelector('video, audio');
                    if (media) (media as HTMLMediaElement).currentTime = captions[0].caption_time;
                });
            };

            tbody.appendChild(tr);
        });

        fragment.appendChild(table);
    }

    function renderDetailsTable() {
        const table = document.createElement('table');
        table.className = 'details-table';

        const isTrash = state.page === 'trash';
        const isPlaylist = state.page === 'playlist';

        const sortIcon = (field) => {
            if (state.filters.sort !== field) return '↕️';
            return state.filters.reverse ? '🔽' : '🔼';
        };

        let headers = `
            <th data-sort="path">Name ${sortIcon('path')}</th>
            <th data-sort="size">Size ${sortIcon('size')}</th>
            <th data-sort="duration">Duration ${sortIcon('duration')}</th>
            <th data-sort="progress">Progress ${sortIcon('progress')}</th>
            <th data-sort="type">Type ${sortIcon('type')}</th>
            ${isTrash ? `<th data-sort="time_deleted">Deleted ${sortIcon('time_deleted')}</th>` : `<th data-sort="play_count">Plays ${sortIcon('play_count')}</th>`}
        `;

        if (isPlaylist) {
            headers = `<th>#</th>` + headers;
        }

        table.innerHTML = `
            <thead>
                <tr>
                    ${headers}
                    <th>Action</th>
                </tr>
            </thead>
            <tbody></tbody>
        `;

        const tbody = table.querySelector('tbody');
        currentMedia.forEach((item, index) => {
            const tr = document.createElement('tr');
            tr.onclick = () => playMedia(item);
            (tr as HTMLElement).dataset.path = item.path;
            tr.draggable = true;

            tr.addEventListener('dragstart', (e) => {
                state.draggedItem = item;
                (e as DragEvent).dataTransfer.effectAllowed = 'all';
                (e as DragEvent).dataTransfer.setData('text/plain', item.path);
                tr.classList.add('dragging');
                document.body.classList.add('is-dragging');
            });

            tr.addEventListener('dragend', () => {
                tr.classList.remove('dragging');
                document.body.classList.remove('is-dragging');
                state.draggedItem = null;
                clearAllDragOver();
            });

            if (isPlaylist) {
                tr.addEventListener('dragover', (e) => {
                    e.preventDefault();
                    (e as DragEvent).dataTransfer.dropEffect = 'move';

                    const rect = tr.getBoundingClientRect();
                    const y = e.clientY - rect.top;
                    if (y < rect.height / 2) {
                        tr.style.borderTop = '2px solid var(--accent-color)';
                        tr.style.borderBottom = '';
                    } else {
                        tr.style.borderTop = '';
                        tr.style.borderBottom = '2px solid var(--accent-color)';
                    }
                });

                tr.addEventListener('dragleave', () => {
                    tr.style.borderTop = '';
                    tr.style.borderBottom = '';
                });

                tr.addEventListener('drop', (e) => {
                    e.preventDefault();
                    tr.style.borderTop = '';
                    tr.style.borderBottom = '';

                    if (state.draggedItem && state.draggedItem !== item) {
                        const rect = tr.getBoundingClientRect();
                        const y = e.clientY - rect.top;
                        let dropIndex = index;

                        // Calculate if we drop before (top) or after (bottom)
                        if (y > rect.height / 2) {
                            dropIndex = index + 1;
                        }

                        // Calculate dragged index
                        const draggedIndex = currentMedia.findIndex(m => m.path === state.draggedItem.path);
                        if (draggedIndex !== -1 && draggedIndex < dropIndex) {
                            dropIndex--;
                        }

                        handlePlaylistReorder(state.draggedItem, dropIndex);
                    }
                });
            }

            const title = item.title || item.path.split('/').pop();
            let actions = '';
            if (isTrash) {
                actions = `
                    <div class="playlist-item-actions">
                        <button class="table-action-btn restore-btn" title="Restore">↺</button>
                        <button class="table-action-btn delete-permanent-btn" title="Permanently Delete">🔥</button>
                    </div>
                `;
            } else if (isPlaylist) {
                actions = !state.readOnly ? `<div class="playlist-item-actions"><button class="table-action-btn remove-btn" title="Remove from Playlist">&times;</button></div>` : '';
            } else {
                const plays = getPlayCount(item);
                actions = `
                    <div class="playlist-item-actions">
                        ${!state.readOnly ? `<button class="table-action-btn add-btn" title="Add to Playlist">+</button>` : ''}
                        ${plays > 0 ?
                        `<button class="table-action-btn mark-unplayed-btn" title="Unmark as Played">⭕</button>` :
                        `<button class="table-action-btn mark-played-btn" title="Mark as Played">✅</button>`
                    }
                        ${!state.readOnly ? `<button class="table-action-btn delete-btn" title="Delete">🗑️</button>` : ''}
                    </div>
                `;
            }

            const localPos = getLocalProgress(item);
            const playhead = (localPos > 0) ? localPos : (item.playhead || 0);
            const progress = (item.duration && playhead) ? Math.round((playhead / item.duration) * 100) : 0;
            const progressHtml = `
                <div style="display: flex; align-items: center; gap: 8px;">
                    <div class="progress-container" style="margin-top: 0; flex: 1; background: rgba(0,0,0,0.2);">
                        <div class="progress-bar" style="width: ${progress}%"></div>
                    </div>
                    <span style="font-size: 0.75rem; min-width: 30px;">${progress}%</span>
                </div>
            `;

            let cells = `
                <td>
                    <div class="table-cell-title" title="${item.path}">
                        <span class="table-icon">${getIcon(item.type)}</span>
                        <div style="display: flex; flex-direction: column;">
                            <span class="media-title-span">${title}</span>

                        </div>
                    </div>
                </td>
                <td>${formatSize(item.size)}</td>
                <td>${formatDuration(item.duration)}</td>
                <td>${progressHtml}</td>
                <td>${item.type || ''}</td>
                <td>${isTrash ? formatRelativeDate(item.time_deleted) : (getPlayCount(item) || '')}</td>
            `;

            if (isPlaylist) {
                const trackDisplay = !state.readOnly ? `<input type="number" class="track-number-input" value="${item.track_number || ''}" min="1">` : `<span>${item.track_number || ''}</span>`;
                cells = `<td>${trackDisplay}</td>` + cells;
            }

            tr.innerHTML = `
                ${cells}
                <td>${actions}</td>
            `;

            const btnMarkPlayed = tr.querySelector('.mark-played-btn');
            if (btnMarkPlayed) (btnMarkPlayed as HTMLElement).onclick = (e) => {
                e.stopPropagation();
                markMediaPlayed(item);
            };

            const btnMarkUnplayed = tr.querySelector('.mark-unplayed-btn');
            if (btnMarkUnplayed) (btnMarkUnplayed as HTMLElement).onclick = (e) => {
                e.stopPropagation();
                markMediaUnplayed(item);
            };

            const btnDelete = tr.querySelector('.delete-btn');
            if (btnDelete) (btnDelete as HTMLElement).onclick = (e) => {
                e.stopPropagation();
                deleteMedia(item.path, false);
            };

            const btnRestore = tr.querySelector('.restore-btn');
            if (btnRestore) (btnRestore as HTMLElement).onclick = (e) => {
                e.stopPropagation();
                deleteMedia(item.path, true);
            };

            const btnDeletePermanent = tr.querySelector('.delete-permanent-btn');
            if (btnDeletePermanent) (btnDeletePermanent as HTMLElement).onclick = (e) => {
                e.stopPropagation();
                permanentlyDeleteMedia(item.path);
            };

            const btnAdd = tr.querySelector('.add-btn');
            if (btnAdd) (btnAdd as HTMLElement).onclick = (e) => {
                e.stopPropagation();
                if (state.playlists.length === 0) {
                    showToast('Create a playlist first');
                    return;
                }
                if (state.playlists.length === 1) {
                    addToPlaylist(state.playlists[0], item);
                } else {
                    const names = state.playlists.map((title, i) => `${i + 1}: ${title}`).join('\n');
                    const choice = prompt(`Add to which playlist?\n${names}`);
                    const idx = parseInt(choice) - 1;
                    if (state.playlists[idx]) {
                        addToPlaylist(state.playlists[idx], item);
                    }
                }
            };

            const btnRemove = tr.querySelector('.remove-btn');
            if (btnRemove) (btnRemove as HTMLElement).onclick = (e) => {
                e.stopPropagation();
                removeFromPlaylist(state.filters.playlist, item);
            };

            const trackInput = tr.querySelector('.track-number-input');
            if (trackInput) {
                (trackInput as HTMLInputElement).onclick = (e) => e.stopPropagation();
                (trackInput as HTMLInputElement).onchange = (e) => {
                    updateTrackNumber(state.filters.playlist, item, (e.target as any).value);
                };
            }

            tbody.appendChild(tr);
        });

        table.querySelectorAll('th[data-sort]').forEach(th => {
            (th as HTMLElement).onclick = () => {
                const field = (th as HTMLElement).dataset.sort;
                if (state.filters.sort === field) {
                    state.filters.reverse = !state.filters.reverse;
                } else {
                    state.filters.sort = field;
                    state.filters.reverse = false;
                }
                // Sync with toolbar
                sortBy.value = state.filters.sort;
                if (state.filters.reverse) {
                    sortReverseBtn.classList.add('active');
                } else {
                    sortReverseBtn.classList.remove('active');
                }
                performSearch();
            };
        });

        resultsContainer.innerHTML = '';
        resultsContainer.className = 'details-view';
        resultsContainer.appendChild(table);
    }

    function renderDbSettingsList(dbs) {
        const list = document.getElementById('db-checkbox-list');
        if (!list) return;

        list.innerHTML = dbs.map(db => `
            <label class="db-checkbox-item">
                <input type="checkbox" value="${db}" ${!state.filters.excludedDbs.includes(db) ? 'checked' : ''}>
                <span>${db.split('/').pop()}</span>
                <small style="color: #666; margin-left: auto;">${db}</small>
            </label>
        `).join('');

        list.querySelectorAll('input').forEach(input => {
            input.onchange = (e) => {
                const val = (e.target as any).value;
                if ((e.target as any).checked) {
                    state.filters.excludedDbs = state.filters.excludedDbs.filter(d => d !== val);
                } else {
                    state.filters.excludedDbs.push(val);
                }
                localStorage.setItem('disco-excluded-dbs', String(JSON.stringify(state.filters.excludedDbs)));
                state.currentPage = 1;
                performSearch();
            };
        });
    }

    function renderCategoryList() {
        if (!categoryList) return;

        updateNavActiveStates();

        const sortedCategories = [...state.categories].sort((a, b) => {
            if (a.category === 'Uncategorized') return 1;
            if (b.category === 'Uncategorized') return -1;
            return b.count - a.count;
        });

        categoryList.innerHTML = sortedCategories.map(c => `
            <button class="category-btn ${state.filters.categories.includes(c.category) ? 'active' : ''}" data-cat="${c.category}">
                ${c.category} <small>(${c.count})</small>
            </button>
        `).join('') + `
            <button id="categorization-link-btn" class="category-btn ${state.page === 'curation' ? 'active' : ''}" style="width: 100%; text-align: left;">
                Categorization <small>🏷️</small>
            </button>
        `;

        const curationLinkBtn = document.getElementById('categorization-link-btn');
        if (curationLinkBtn) {
            curationLinkBtn.onclick = () => {
                if (state.page === 'curation') {
                    state.page = 'search';
                    updateNavActiveStates();
                    performSearch();
                } else {
                    state.page = 'curation';
                    updateNavActiveStates();
                    fetchCuration();
                }
            };
        }

        categoryList.querySelectorAll('.category-btn').forEach(btn => {
            if (btn.id === 'categorization-link-btn') return;
            (btn as any).onclick = (e) => {
                const cat = (btn as any).dataset.cat;
                if (state.page !== 'trash') state.page = 'search';

                if (state.filters.categories.includes(cat)) {
                    state.filters.categories = [];
                } else {
                    state.filters.categories = [cat];
                }

                localStorage.setItem('disco-filter-categories', String(JSON.stringify(state.filters.categories)));
                state.currentPage = 1;
                updateNavActiveStates();
                performSearch();
            };
        });
    }

    function renderLanguageList() {
        const languageList = document.getElementById('language-list');
        if (!languageList) return;

        const sortedLanguages = [...state.languages].sort((a, b) => {
            return b.count - a.count;
        });

        languageList.innerHTML = sortedLanguages.map(l => `
            <button class="category-btn ${state.filters.languages.includes(l.category) ? 'active' : ''}" data-lang="${l.category}">
                ${l.category} <small>(${l.count})</small>
            </button>
        `).join('');

        languageList.querySelectorAll('.category-btn').forEach(btn => {
            (btn as any).onclick = (e) => {
                const lang = (btn as any).dataset.lang;
                if (state.page !== 'trash') state.page = 'search';

                if (state.filters.languages.includes(lang)) {
                    state.filters.languages = [];
                } else {
                    state.filters.languages = [lang];
                }

                localStorage.setItem('disco-filter-languages', String(JSON.stringify(state.filters.languages)));
                state.currentPage = 1;
                updateNavActiveStates();
                performSearch();
            };
        });
    }

    // --- Helpers ---
    function errorToast(err: any, fallbackMsg) {
        console.error('errorToast:', fallbackMsg, err);
        if (err.message === 'Access Denied' || err.message === 'Unauthorized') {
            showToast(err.message);
            return true;
        }
        if (fallbackMsg) {
            showToast(fallbackMsg);
        }
        return false;
    }

    function showToast(msg, customEmoji = null) {
        console.log('showToast:', msg, customEmoji);
        if (state.playback.toastTimer) {
            clearTimeout(state.playback.toastTimer);
        }

        let icon = customEmoji;
        const lowerMsg = (msg || '').toLowerCase();
        if (lowerMsg === 'access denied' || lowerMsg.includes(': forbidden')) {
            msg = 'Access Denied';
            icon = '🚫';
        } else if (lowerMsg === 'unauthorized' || lowerMsg.includes(': unauthorized')) {
            msg = 'Unauthorized';
            icon = '🚫';
        }

        if (!icon) {
            icon = lowerMsg.includes('fail') || lowerMsg.includes('error') ? '❌' : 'ℹ️';
        }

        toast.innerHTML = `<span>${icon}</span> <span>${msg}</span>`;

        // Move toast into fullscreen element if active
        if (document.fullscreenElement) {
            if (toast.parentElement !== document.fullscreenElement) {
                document.fullscreenElement.appendChild(toast);
            }
        } else if (toast.parentElement !== document.body) {
            document.body.appendChild(toast);
        }

        toast.classList.remove('hidden');

        state.playback.toastTimer = setTimeout(() => {
            toast.classList.add('hidden');
            state.playback.toastTimer = null;
        }, 3000);
    }

    // --- Helpers ---
    function debounce(func, wait) {
        let timeout;
        return function executedFunction(...args) {
            const later = () => {
                clearTimeout(timeout);
                func(...args);
            };
            clearTimeout(timeout);
            timeout = setTimeout(later, wait);
        };
    }

    // --- Keyboard Shortcuts ---
    window.addEventListener('keydown', (e) => {
        // Don't trigger shortcuts if user is typing in an input
        if ((e.target as HTMLElement).tagName === 'INPUT' || (e.target as HTMLElement).tagName === 'TEXTAREA' || (e.target as HTMLElement).tagName === 'SELECT') {
            return;
        }

        const pipPlayer = document.getElementById('pip-player');
        const docModal = document.getElementById('document-modal');
        const isPipVisible = pipPlayer && !pipPlayer.classList.contains('hidden');
        const isDocModalVisible = docModal && !docModal.classList.contains('hidden');
        const hasActiveViewer = isPipVisible || isDocModalVisible;

        // Helper to get hovered media card
        const getHoveredMediaCard = () => {
            const hovered = document.querySelector('.media-card:hover');
            if (hovered) {
                const pathEl = hovered.querySelector('[data-path]');
                if (pathEl) return (pathEl as HTMLElement).dataset.path;
                // Try to get path from data attribute on card itself
                if ((hovered as HTMLElement).dataset.path) return (hovered as HTMLElement).dataset.path;
            }
            return null;
        };

        // 1. Rating shortcuts (1-5) - work anytime, prefer active viewer, fallback to hovered card
        if (!e.ctrlKey && !e.metaKey && !e.altKey && ['1', '2', '3', '4', '5'].includes(e.key)) {
            const score = parseInt(e.key);
            // Rate current playing item or hovered card
            if (state.playback.item) {
                rateMedia(state.playback.item, score);
            } else {
                const hoveredPath = getHoveredMediaCard();
                if (hoveredPath) {
                    const hoveredItem = currentMedia.find(m => m.path === hoveredPath);
                    if (hoveredItem) {
                        rateMedia(hoveredItem, score);
                    }
                }
            }
            return;
        }

        // 0: Unrate (set rating to 0)
        if (!e.ctrlKey && !e.metaKey && !e.altKey && e.key === '`') {
            if (state.playback.item) {
                rateMedia(state.playback.item, 0);
            } else {
                const hoveredPath = getHoveredMediaCard();
                if (hoveredPath) {
                    const hoveredItem = currentMedia.find(m => m.path === hoveredPath);
                    if (hoveredItem) {
                        rateMedia(hoveredItem, 0);
                    }
                }
            }
            return;
        }

        // 2. Seek shortcuts (Shift+0-9) - seek to 0%, 10%, 20%, ... 90%, 100%
        if (!e.ctrlKey && !e.metaKey && !e.altKey && e.shiftKey &&
            ['0', '1', '2', '3', '4', '5', '6', '7', '8', '9'].includes(e.key)) {
            if (isPipVisible) {
                const media = pipViewer.querySelector('video, audio');
                if (media && !isNaN((media as HTMLMediaElement).duration)) {
                    const percent = e.key === '0' ? 1.0 : parseInt(e.key) / 10;
                    (media as HTMLMediaElement).currentTime = (media as HTMLMediaElement).duration * percent;
                    showToast(`Seek to ${e.key === '0' ? 100 : parseInt(e.key) * 10}%`, '⏩');
                }
            }
            return;
        }

        // 2. Independent shortcuts (don't require active viewer)
        if (!e.ctrlKey && !e.metaKey && !e.altKey) {
            // Escape key closes the topmost visible modal (but not document-modal, handled below)
            if (e.key === 'Escape') {
                const allModals = document.querySelectorAll('.modal:not(.hidden)');
                if (allModals.length > 0) {
                    const topmostModal = allModals[allModals.length - 1];
                    if (topmostModal.id !== 'document-modal') {
                        closeModal(topmostModal.id);
                        e.preventDefault();
                        return;
                    }
                }
                // Let Escape fall through to close PiP or document-modal below
            }

            switch (e.key.toLowerCase()) {
                case 'n':
                    playSibling(1, true);
                    return;
                case 'p':
                    playSibling(-1, true);
                    return;
                case 'i':
                    if (state.playback.item) {
                        const modal = document.getElementById('metadata-modal');
                        if (modal.classList.contains('hidden')) {
                            showMetadata(state.playback.item);
                        } else {
                            closeModal('metadata-modal');
                        }
                    }
                    return;
                case '?':
                case '/':
                    const helpModal = document.getElementById('help-modal');
                    if (helpModal.classList.contains('hidden')) {
                        openModal('help-modal');
                    } else {
                        closeModal('help-modal');
                    }
                    return;
                case 't':
                    e.preventDefault();
                    if (searchInput) {
                        searchInput.focus();
                        searchInput.select();
                    }
                    return;
                case 'c':
                    if (state.playback.item) {
                        const path = state.playback.item.path;
                        navigator.clipboard.writeText(path).then(() => {
                            showToast(`Copied path to clipboard`, '📋');
                        }).catch(err => {
                            console.error('Failed to copy path:', err);
                            errorToast(err as any, 'Failed to copy path');
                        });
                    }
                    return;
                case 'r':
                    playRandomMedia();
                    return;
                case 'f':
                    // 'f' toggles fullscreen for active viewer, or exits fullscreen if no viewer
                    if (document.fullscreenElement && !hasActiveViewer) {
                        // If in fullscreen but no viewer is visible, exit fullscreen
                        document.exitFullscreen().catch(err => {
                            console.error('Failed to exit fullscreen:', err);
                        });
                    } else {
                        toggleFullscreen(getActiveViewerElement());
                    }
                    return;
            }
        }

        // 3. Global shortcuts that require an active viewer (PiP or document modal)
        if (!hasActiveViewer) {
            return;
        }

        // Close shortcuts (work for both PiP and document modal)
        switch (e.key.toLowerCase()) {
            case 'q':
            case 'w':
                closeActivePlayer();
                e.preventDefault();
                return;
            case 'escape':
                // Escape exits fullscreen first if active, otherwise closes the viewer
                if (document.fullscreenElement) {
                    document.exitFullscreen().catch(err => {
                        console.error('Failed to exit fullscreen:', err);
                    });
                    e.preventDefault();
                    return;
                }
                closeActivePlayer();
                e.preventDefault();
                return;
            case 'delete':
                if (state.playback.item) {
                    const itemToDelete = state.playback.item;

                    if (e.shiftKey) {
                        closeActivePlayer();
                    } else {
                        playSibling(1, true, true);
                    }

                    deleteMedia(itemToDelete.path);
                }
                e.preventDefault();
                return;
        }

        // Only process media-specific shortcuts if PiP is visible
        if (!isPipVisible) {
            // Navigation shortcuts (seamlessly switch between PiP and document modal)
            if (e.key === 'ArrowLeft' || e.key === 'ArrowRight') {
                if (!e.ctrlKey && !e.metaKey && !e.altKey) {
                    playSibling(e.key === 'ArrowLeft' ? -1 : 1, true);
                    e.preventDefault();
                }
                return;
            }
            return;
        }

        // Navigation shortcuts when PiP is visible (seek within media)
        if (e.key === 'ArrowLeft' || e.key === 'ArrowRight') {
            if (!e.ctrlKey && !e.metaKey && !e.altKey) {
                // Let it fall through to the switch statement for seeking
            } else {
                playSibling(e.key === 'ArrowLeft' ? -1 : 1, true);
                e.preventDefault();
                return;
            }
        }

        const media = pipViewer.querySelector('video, audio, img');
        if (!media) {
            return;
        }

        const isPlaying = ((media as HTMLMediaElement).paused === false);
        const duration = (media as HTMLMediaElement).duration;
        const currentTime = (media as HTMLMediaElement).currentTime || 0;

        const setTime = (t) => {
            if ((media as HTMLMediaElement).currentTime !== undefined && !isNaN(t) && isFinite(t)) {
                (media as HTMLMediaElement).currentTime = t;
            }
        };

        const playPause = () => {
            if (media.tagName === 'IMG') {
                if (state.playback.slideshowTimer) stopSlideshow();
                else startSlideshow();
            } else {
                if ((media as HTMLMediaElement).paused) (media as HTMLMediaElement).play();
                else (media as HTMLMediaElement).pause();
            }
        };

        const toggleSubtitleVisibility = () => {
            if (!(media as HTMLMediaElement).textTracks) return;
            const tracks = Array.from((media as HTMLMediaElement).textTracks).filter(t => t.kind === 'subtitles');
            if (tracks.length === 0) return;

            // Check if any track is showing
            const hasShowingTrack = tracks.some(t => (t as TextTrack).mode === 'showing');

            if (hasShowingTrack) {
                // Disable all tracks
                tracks.forEach(t => (t as TextTrack).mode = 'disabled');
                showToast('Subtitles: Off', '💬');
            } else {
                // Enable first track
                tracks[0].mode = 'showing';
                showToast(`Subtitles: ${tracks[0].label || 'Track 1'}`, '💬');
            }
        };

        switch (e.key.toLowerCase()) {
            case ' ':
            case 'k':
                e.preventDefault();
                playPause();
                break;
            case 'm':
                (media as HTMLMediaElement).muted = !(media as HTMLMediaElement).muted;
                break;
            case 'j':
                if (media.tagName === 'VIDEO') {
                    cycleSubtitles(e.shiftKey);
                } else {
                    setTime(Math.max(0, currentTime - 10));
                }
                break;
            case 'v':
                toggleSubtitleVisibility();
                break;
            case 'l':
                if (e.shiftKey) {
                    // Toggle automatic looping preference
                    if (state.autoLoopMaxDuration > 0) {
                        state.autoLoopMaxDuration = 0;
                    } else {
                        state.autoLoopMaxDuration = 30;
                    }
                    localStorage.setItem('disco-auto-loop-max-duration', String(state.autoLoopMaxDuration));
                    const settingAutoLoopMax = document.getElementById('setting-auto-loop-max');
                    if (settingAutoLoopMax) (settingAutoLoopMax as HTMLInputElement).value = state.autoLoopMaxDuration.toString();
                    showToast(`Auto-Loop: ${state.autoLoopMaxDuration > 0 ? state.autoLoopMaxDuration + 's' : 'OFF'}`, '🔁');
                    return;
                }
                if (media.tagName === 'VIDEO' || media.tagName === 'AUDIO') {
                    (media as HTMLMediaElement).loop = !(media as HTMLMediaElement).loop;
                    showToast((media as HTMLMediaElement).loop ? 'Loop: ON' : 'Loop: OFF', '🔁');
                }
                break;
            case 'o':
                // Show progression bar, elapsed time and total duration on OSD
                if (media.tagName === 'VIDEO' || media.tagName === 'AUDIO') {
                    const elapsed = formatDuration(currentTime);
                    const total = formatDuration(duration);
                    const progress = duration ? Math.round((currentTime / duration) * 100) : 0;
                    showToast(`${elapsed} / ${total} (${progress}%)`, '📊');
                }
                break;
            // Seek shortcuts - arrow keys with modifiers
            case 'arrowup':
                if (e.shiftKey) {
                    // Shift+Up: Seek forward 5s (exact)
                    state.playback.seekHistory.push(currentTime);
                    if (state.playback.seekHistory.length > 10) state.playback.seekHistory.shift();
                    setTime(Math.min(duration, currentTime + 5));
                } else {
                    // Up: Seek forward 1 minute
                    state.playback.seekHistory.push(currentTime);
                    if (state.playback.seekHistory.length > 10) state.playback.seekHistory.shift();
                    setTime(Math.min(duration, currentTime + 60));
                }
                break;
            case 'arrowdown':
                if (e.shiftKey) {
                    // Shift+Down: Seek backward 5s (exact)
                    state.playback.seekHistory.push(currentTime);
                    if (state.playback.seekHistory.length > 10) state.playback.seekHistory.shift();
                    setTime(Math.max(0, currentTime - 5));
                } else {
                    // Down: Seek backward 1 minute
                    state.playback.seekHistory.push(currentTime);
                    if (state.playback.seekHistory.length > 10) state.playback.seekHistory.shift();
                    setTime(Math.max(0, currentTime - 60));
                }
                break;
            case 'arrowleft':
                if (e.ctrlKey || e.metaKey) {
                    // Ctrl+Left: Seek to previous subtitle
                    if (media.tagName === 'VIDEO') {
                        seekToSubtitleCue(true);
                    }
                } else if (e.shiftKey) {
                    // Shift+Left: Seek backward 1s (exact)
                    state.playback.seekHistory.push(currentTime);
                    if (state.playback.seekHistory.length > 10) state.playback.seekHistory.shift();
                    setTime(Math.max(0, currentTime - 1));
                } else {
                    // Left: Seek backward 5s
                    if (currentTime < 1) {
                        playSibling(-1, true);
                    } else {
                        state.playback.seekHistory.push(currentTime);
                        if (state.playback.seekHistory.length > 10) state.playback.seekHistory.shift();
                        setTime(Math.max(0, currentTime - 5));
                    }
                }
                break;
            case 'arrowright':
                if (e.ctrlKey || e.metaKey) {
                    // Ctrl+Right: Seek to next subtitle
                    if (media.tagName === 'VIDEO') {
                        seekToSubtitleCue(false);
                    }
                } else if (e.shiftKey) {
                    // Shift+Right: Seek forward 1s (exact)
                    state.playback.seekHistory.push(currentTime);
                    if (state.playback.seekHistory.length > 10) state.playback.seekHistory.shift();
                    setTime(Math.min(duration, currentTime + 1));
                } else {
                    // Right: Seek forward 5s
                    if (isNaN(duration) || duration - currentTime < 1) {
                        playSibling(1, true);
                    } else if (!isNaN(duration)) {
                        state.playback.seekHistory.push(currentTime);
                        if (state.playback.seekHistory.length > 10) state.playback.seekHistory.shift();
                        setTime(Math.min(duration, currentTime + 5));
                    }
                }
                break;
            // Playback speed shortcuts
            case '[':
                // Decrease speed to next lower predefined rate
                if (media.tagName === 'VIDEO' || media.tagName === 'AUDIO') {
                    const newRate = stepPlaybackRate(state.playbackRate, -1);
                    setPlaybackRate(newRate);
                    showToast(`Speed: ${newRate}x`, '⚡');
                }
                break;
            case ']':
                // Increase speed to next higher predefined rate
                if (media.tagName === 'VIDEO' || media.tagName === 'AUDIO') {
                    const newRate = stepPlaybackRate(state.playbackRate, 1);
                    setPlaybackRate(newRate);
                    showToast(`Speed: ${newRate}x`, '⚡');
                }
                break;
            case '{':
                // Halve speed
                if (media.tagName === 'VIDEO' || media.tagName === 'AUDIO') {
                    const newRate = Math.max(0.2, state.playbackRate / 2);
                    setPlaybackRate(newRate);
                    showToast(`Speed: ${newRate}x`, '⚡');
                }
                break;
            case '}':
                // Double speed
                if (media.tagName === 'VIDEO' || media.tagName === 'AUDIO') {
                    const newRate = Math.min(8, state.playbackRate * 2);
                    setPlaybackRate(newRate);
                    showToast(`Speed: ${newRate}x`, '⚡');
                }
                break;
            // Seek undo/redo shortcuts
            case 'backspace':
                if (e.shiftKey && e.ctrlKey) {
                    // Shift+Ctrl+Backspace: Mark current position
                    state.playback.markedPosition = currentTime;
                    showToast(`Position marked: ${formatDuration(currentTime)}`, '📍');
                    e.preventDefault();
                } else if (e.shiftKey) {
                    // Shift+Backspace: Undo last seek
                    if (state.playback.seekHistory.length > 0) {
                        const prevTime = state.playback.seekHistory.pop();
                        setTime(prevTime);
                        showToast('Seek undone', '↩️');
                    } else if (state.playback.markedPosition !== null) {
                        // If no seek history, use marked position
                        setTime(state.playback.markedPosition);
                        state.playback.markedPosition = null;
                        showToast('Returned to marked position', '📍');
                    } else {
                        showToast('No seek to undo', 'ℹ️');
                    }
                    e.preventDefault();
                } else {
                    // Backspace: Reset speed to normal
                    if (media.tagName === 'VIDEO' || media.tagName === 'AUDIO') {
                        setPlaybackRate(1.0);
                        showToast('Speed: Normal', '⚡');
                    }
                }
                break;
            // Playlist navigation shortcuts
            case '<':
                // Previous in playlist
                playSibling(-1, true);
                break;
            case '>':
                // Next in playlist
                playSibling(1, true);
                break;
            // Home key - seek to start
            case 'home':
                setTime(0);
                showToast('Seek to start', '⏮️');
                break;
            case 'pageup':
                state.playback.seekHistory.push(currentTime);
                if (state.playback.seekHistory.length > 10) state.playback.seekHistory.shift();
                setTime(Math.max(0, currentTime - 600));
                showToast('Seek -10 min', '⏪');
                break;
            case 'pagedown':
                state.playback.seekHistory.push(currentTime);
                if (state.playback.seekHistory.length > 10) state.playback.seekHistory.shift();
                setTime(Math.min(duration, currentTime + 600));
                showToast('Seek +10 min', '⏩');
                break;
            // Volume control shortcuts (numpad)
            case '/':
                if (e.code === 'NumpadDivide' || e.key === 'NumpadDivide') {
                    (media as HTMLMediaElement).volume = Math.max(0, (media as HTMLMediaElement).volume - 0.1);
                    showToast(`Volume: ${Math.round((media as HTMLMediaElement).volume * 100)}%`, '🔊');
                }
                break;
            case '*':
                if (e.code === 'NumpadMultiply' || e.key === 'NumpadMultiply') {
                    (media as HTMLMediaElement).volume = Math.min(1, (media as HTMLMediaElement).volume + 0.1);
                    showToast(`Volume: ${Math.round((media as HTMLMediaElement).volume * 100)}%`, '🔊');
                }
                break;
            case '9':
                // Numpad 9 for volume down (when not seeking)
                (media as HTMLMediaElement).volume = Math.max(0, (media as HTMLMediaElement).volume - 0.1);
                showToast(`Volume: ${Math.round((media as HTMLMediaElement).volume * 100)}%`, '🔊');
                break;
            case '0':
                // Numpad 0 for volume up (when not seeking)
                (media as HTMLMediaElement).volume = Math.min(1, (media as HTMLMediaElement).volume + 0.1);
                showToast(`Volume: ${Math.round((media as HTMLMediaElement).volume * 100)}%`, '🔊');
                break;
            // Screenshot shortcuts
            case 's':
                if (media.tagName === 'VIDEO') {
                    e.preventDefault();
                    takeScreenshot(media, e.shiftKey);
                }
                break;
            // Aspect ratio toggle
            case 'a':
                if (media.tagName === 'VIDEO') {
                    cycleAspectRatio(media);
                }
                break;
            // Media keys (Previous/Next) - same behavior as arrow keys for seeking
            case 'mediaprev':
            case 'medianext':
            case 'MediaTrackPrevious':
            case 'MediaTrackNext':
                if (e.key === 'MediaTrackNext' || e.key === 'MediaPlayNext') {
                    // Next: seek forward 1 min or go to next media at end
                    if (!isNaN(duration) && duration - currentTime < 1) {
                        playSibling(1, true);
                    } else if (!isNaN(duration)) {
                        state.playback.seekHistory.push(currentTime);
                        if (state.playback.seekHistory.length > 10) state.playback.seekHistory.shift();
                        setTime(Math.min(duration, currentTime + 60));
                    }
                } else if (e.key === 'MediaTrackPrevious' || e.key === 'MediaPlayPrevious') {
                    // Previous: seek backward 1 min or go to prev media at start
                    if (currentTime < 1) {
                        playSibling(-1, true);
                    } else {
                        state.playback.seekHistory.push(currentTime);
                        if (state.playback.seekHistory.length > 10) state.playback.seekHistory.shift();
                        setTime(Math.max(0, currentTime - 60));
                    }
                }
                break;
            case '.':
                // Step forward one frame
                if (media.tagName === 'VIDEO' || media.tagName === 'AUDIO') {
                    e.preventDefault();
                    if (isPlaying) {
                        (media as HTMLMediaElement).pause();
                    }
                    // Try to advance by 1 frame at a time
                    // Start with 1/60s and increase until we see a change
                    const fps = (media as any).webkitDecodedFrameCount && (media as any).webkitDecodedFrameCount > 0
                        ? 60
                        : (duration && duration <= 10 ? 30 : 24); // Estimate based on duration
                    let step = 1 / fps;
                    const originalTime = currentTime;
                    let newTime = Math.min(duration, originalTime + step);

                    // If we're very close to the end, just go to the end
                    if (duration - originalTime < step) {
                        setTime(duration);
                    } else {
                        // Try up to 3 frames if 1 frame doesn't advance
                        for (let i = 1; i <= 3 && newTime === originalTime; i++) {
                            step = i / fps;
                            newTime = Math.min(duration, originalTime + step);
                        }
                        setTime(newTime);
                    }
                }
                break;
            case ',':
                // Step backward one frame
                if (media.tagName === 'VIDEO' || media.tagName === 'AUDIO') {
                    e.preventDefault();
                    if (isPlaying) {
                        (media as HTMLMediaElement).pause();
                    }
                    // Try to go back by 1 frame at a time
                    const fps = (media as any).webkitDecodedFrameCount && (media as any).webkitDecodedFrameCount > 0
                        ? 60
                        : (duration && duration <= 10 ? 30 : 24);
                    let step = 1 / fps;
                    const originalTime = currentTime;
                    let newTime = Math.max(0, originalTime - step);

                    // Try up to 3 frames if 1 frame doesn't move
                    for (let i = 1; i <= 3 && newTime === originalTime; i++) {
                        step = i / fps;
                        newTime = Math.max(0, originalTime - step);
                    }
                    setTime(newTime);
                }
                break;
        }
    });

    // --- Wheel Shortcuts ---
    pipViewer.addEventListener('wheel', (e) => {
        // Only handle wheel events when PiP is visible
        if (pipPlayer.classList.contains('hidden')) return;

        const media = pipViewer.querySelector('video, audio');
        if (!media) return;

        e.preventDefault();

        const duration = (media as HTMLMediaElement).duration;
        const currentTime = (media as HTMLMediaElement).currentTime || 0;

        // Handle horizontal scroll (wheel left/right)
        if (e.deltaX !== 0) {
            // Wheel left/right: Seek backward/forward 10 seconds
            const delta = e.deltaX > 0 ? 10 : -10;
            const newTime = Math.max(0, Math.min(duration, currentTime + delta));
            state.playback.seekHistory.push(currentTime);
            if (state.playback.seekHistory.length > 10) state.playback.seekHistory.shift();
            (media as HTMLMediaElement).currentTime = newTime;
            const direction = delta > 0 ? 'forward' : 'backward';
            showToast(`Seek ${direction} 10s`, '⏩');
        }
        // Handle vertical scroll (wheel up/down)
        else if (e.deltaY !== 0) {
            // Wheel up/down: Increase/decrease volume
            const delta = e.deltaY > 0 ? -0.05 : 0.05;
            const newVolume = Math.max(0, Math.min(1, (media as HTMLMediaElement).volume + delta));
            (media as HTMLMediaElement).volume = newVolume;
            const volumePercent = Math.round(newVolume * 100);
            showToast(`Volume: ${volumePercent}%`, '🔊');
        }
    }, { passive: false });

    // --- Dev Mode Auto-Reload ---
    async function setupAutoReload() {
        const url = '/api/events';
        try {
            const res = await fetch(url, { credentials: 'include' });
            if (res.status === 401) {
                location.reload();
                return;
            }
        } catch {
            // Network error, will retry
        }

        const events = new EventSource(url);
        events.onmessage = (event) => {
            const startTime = event.data;
            if (state.applicationStartTime && state.applicationStartTime !== startTime) {
                console.log('Server restarted, reloading...');
                location.reload();
            }
            state.applicationStartTime = startTime;
        };
        events.onerror = () => {
            events.close();
            setTimeout(setupAutoReload, 2000);
        };
    }

    function handlePostPlayback(item) {
        // Only proceed if the player is still active and playing this item
        if (!state.playback.item || state.playback.item.path !== item.path) return;

        // If we just had an error, don't trigger post-playback skip
        if (state.playback.skipTimeout) return;

        if (state.postPlaybackAction === 'delete') {
            deleteMedia(item.path);
            if (state.autoplay) playSibling(1, false, true);
        } else if (state.postPlaybackAction === 'ask') {
            openModal('confirm-modal');
            document.getElementById('confirm-yes').onclick = () => {
                closeModal('confirm-modal');
                deleteMedia(item.path);
                if (state.autoplay) playSibling(1, false, true);
            };
            document.getElementById('confirm-no').onclick = () => {
                closeModal('confirm-modal');
                if (state.autoplay) playSibling(1);
            };
        } else {
            if (state.autoplay) playSibling(1);
        }
    }

    function applyTheme() {
        if (state.theme === 'auto') {
            document.documentElement.removeAttribute('data-theme');
        } else {
            document.documentElement.setAttribute('data-theme', state.theme);
        }
    }

    // Watch for system theme changes if set to auto
    window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => {
        if (state.theme === 'auto') {
            applyTheme();
        }
    });

    // --- Event Listeners ---
    const debouncedSearch = debounce(performSearch, 300);

    document.addEventListener('click', (e) => {
        if (!searchInput.contains(e.target as Node) && !searchSuggestions.contains(e.target as Node)) {
            searchSuggestions.classList.add('hidden');
        }
    });

    searchInput.oninput = (e) => {
        let val = (e.target as any).value;
        if (val.includes('\\')) {
            val = val.replace(/\\/g, '/');
            (e.target as any).value = val;
        }

        if (val.startsWith('/') || val.startsWith('./')) {
            // Path browsing
            fetchSuggestions(val);
        } else {
            searchSuggestions.classList.add('hidden');
            debouncedSearch();
        }
    };

    searchInput.onfocus = () => {
        let val = (searchInput as HTMLInputElement).value;
        if (val.includes('\\')) {
            val = val.replace(/\\/g, '/');
            (searchInput as HTMLInputElement).value = val;
        }

        if (val.startsWith('/') || val.startsWith('./')) {
            fetchSuggestions(val);
        }
    };

    searchInput.onkeydown = (e) => {
        if (e.key === 'Tab' && e.shiftKey) {
            e.preventDefault();
            const val = (searchInput as HTMLInputElement).value;
            if (val.startsWith('/') || val.startsWith('./')) {
                const parts = val.split('/');
                if (val.endsWith('/')) {
                    parts.pop(); // remove empty trailing
                    parts.pop(); // remove last folder
                } else {
                    parts.pop(); // remove partial segment
                }
                const newVal = parts.join('/') + (parts.length > 0 ? '/' : '');
                (searchInput as HTMLInputElement).value = newVal || (val.startsWith('/') ? '/' : './');
                fetchSuggestions((searchInput as HTMLInputElement).value);
                performSearch();
            }
            return;
        }

        const items = searchSuggestions.querySelectorAll('.suggestion-item');
        if (searchSuggestions.classList.contains('hidden') || items.length === 0) return;

        if (e.key === 'Tab') {
            e.preventDefault();
            if (selectedSuggestionIndex === -1) {
                selectedSuggestionIndex = 0;
            }
            const el = items[selectedSuggestionIndex];
            const path = (el as HTMLElement).dataset.path;
            const isDir = (el as HTMLElement).dataset.isDir === 'true';
            if (isDir) {
                if ((searchInput as HTMLInputElement).value.startsWith('./')) {
                    const newName = (el as HTMLElement).dataset.name;
                    const lastSlash = (searchInput as HTMLInputElement).value.lastIndexOf('/');
                    const newPath = (searchInput as HTMLInputElement).value.substring(0, lastSlash + 1) + newName + '/';
                    (searchInput as HTMLInputElement).value = newPath;
                } else {
                    const newPath = path.endsWith('/') ? path : path + '/';
                    (searchInput as HTMLInputElement).value = newPath;
                }
                fetchSuggestions((searchInput as HTMLInputElement).value);
                performSearch();
            } else {
                (searchInput as HTMLInputElement).value = path;
                searchSuggestions.classList.add('hidden');
                performSearch();
            }
        } else if (e.key === 'ArrowDown') {
            e.preventDefault();
            selectedSuggestionIndex = (selectedSuggestionIndex + 1) % items.length;
            updateSelectedSuggestion(items);
        } else if (e.key === 'ArrowUp') {
            e.preventDefault();
            selectedSuggestionIndex = (selectedSuggestionIndex - 1 + items.length) % items.length;
            updateSelectedSuggestion(items);
        } else if (e.key === 'Enter' && selectedSuggestionIndex >= 0) {
            e.preventDefault();
            const el = items[selectedSuggestionIndex];
            const path = (el as HTMLElement).dataset.path;
            const isDir = (el as HTMLElement).dataset.isDir === 'true';
            if (isDir) {
                if ((searchInput as HTMLInputElement).value.startsWith('./')) {
                    const newName = (el as HTMLElement).dataset.name;
                    const lastSlash = (searchInput as HTMLInputElement).value.lastIndexOf('/');
                    const newPath = (searchInput as HTMLInputElement).value.substring(0, lastSlash + 1) + newName + '/';
                    (searchInput as HTMLInputElement).value = newPath;
                } else {
                    const newPath = path.endsWith('/') ? path : path + '/';
                    (searchInput as HTMLInputElement).value = newPath;
                }
                fetchSuggestions((searchInput as HTMLInputElement).value);
                performSearch();
            } else {
                (searchInput as HTMLInputElement).value = path;
                searchSuggestions.classList.add('hidden');
                performSearch();
            }
        } else if (e.key === 'Escape') {
            searchSuggestions.classList.add('hidden');
        }
    };

    function updateSelectedSuggestion(items) {
        items.forEach((item, idx) => {
            if (idx === selectedSuggestionIndex) {
                item.classList.add('selected');
                item.scrollIntoView({ block: 'nearest' });
            } else {
                item.classList.remove('selected');
            }
        });
    }

    const settingsBtn = document.getElementById('settings-button');
    if (settingsBtn) settingsBtn.onclick = () => {
        calculateStorageSize();
        openModal('settings-modal');
    };

    document.querySelectorAll('.close-modal').forEach(btn => {
        (btn as any).onclick = (e) => {
            const modal = (e.target as HTMLElement).closest('.modal');
            if (modal) {
                if (modal.id === 'document-modal') {
                    closeActivePlayer();
                } else {
                    closeModal(modal.id);
                }
            }
        };
    });

    const closePipBtn = document.querySelector('.close-pip');
    if (closePipBtn) (closePipBtn as HTMLElement).onclick = closePiP;



    if (filterBrowseCol) {
        filterBrowseCol.onchange = async () => {
            const col = filterBrowseCol.value;
            if (!col) {
                filterBrowseValContainer.classList.add('hidden');
                return;
            }

            filterBrowseValContainer.classList.remove('hidden');
            filterBrowseVal.innerHTML = '<option value="">Loading...</option>';

            let options = [];
            if (col === 'genre') {
                if (state.genres.length === 0) await fetchGenres();
                options = state.genres.map(g => ({ val: g.genre, label: `${g.genre} (${g.count})` }));
            } else if (col === 'category') {
                if (state.categories.length === 0) await fetchCategories();
                options = state.categories.map(c => ({ val: c.category, label: `${c.category} (${c.count})` }));
            } else if (col === 'db') {
                options = allDatabases.map(db => ({ val: db, label: db }));
            }

            filterBrowseVal.innerHTML = '<option value="">All</option>' +
                options.map(o => `<option value="${o.val}">${o.label}</option>`).join('');

            // Try to restore selection from state
            if (col === 'genre') filterBrowseVal.value = state.filters.genre;
            else if (col === 'category') filterBrowseVal.value = state.filters.categories[0] || '';
        };
    }

    const pipSlideshowBtn = document.getElementById('pip-slideshow');
    if (pipSlideshowBtn) {
        pipSlideshowBtn.onclick = (e) => {
            e.stopPropagation();
            if (state.playback.slideshowTimer) {
                stopSlideshow();
            } else {
                startSlideshow();
            }
        };
    }

    if (pipSpeedBtn) {
        (pipSpeedBtn as HTMLElement).onclick = (e) => {
            e.stopPropagation();
            pipSpeedMenu.classList.toggle('hidden');
        };
    }

    document.querySelectorAll('.speed-opt').forEach(btn => {
        (btn as any).onclick = (e) => {
            e.stopPropagation();
            const rate = parseFloat((btn as any).dataset.speed);
            setPlaybackRate(rate);
            pipSpeedMenu.classList.add('hidden');
        };
    });

    // Close speed menu when clicking elsewhere
    window.addEventListener('click', () => {
        if (pipSpeedMenu) pipSpeedMenu.classList.add('hidden');
    });

    const pipStreamTypeBtn = document.getElementById('pip-stream-type');
    if (pipStreamTypeBtn) pipStreamTypeBtn.onclick = () => {
        if (!state.playback.item) return;
        state.playback.item.transcode = !state.playback.item.transcode;
        const currentPos = (pipViewer.querySelector('video') as HTMLMediaElement)?.currentTime || 0;
        openActivePlayer(state.playback.item);

        const media = pipViewer.querySelector('video, audio');
        if (media) {
            (media as HTMLMediaElement).onloadedmetadata = () => {
                (media as HTMLMediaElement).currentTime = currentPos;
            };
        }
    };

    const pipNativeBtn = document.getElementById('pip-native');
    if (pipNativeBtn) {
        pipNativeBtn.onclick = async () => {
            const video = pipViewer.querySelector('video');
            if (!video) return;
            try {
                if (video !== document.pictureInPictureElement) {
                    await video.requestPictureInPicture();
                } else {
                    await document.exitPictureInPicture();
                }
            } catch (error) {
                console.error('PiP Error:', error);
                showToast('PiP failed: ' + error.message);
            }
        };
    }

    const pipTheatreBtn = document.getElementById('pip-theatre');
    if (pipTheatreBtn) pipTheatreBtn.onclick = toggleTheatreMode;

    function toggleTheatreMode() {
        const theatreAnchor = document.getElementById('theatre-anchor');
        const btn = document.getElementById('pip-theatre');
        if (state.playerMode === 'pip') {
            state.playerMode = 'theatre';
            pipPlayer.classList.add('theatre');
            if (pipPlayer.parentElement !== theatreAnchor) {
                theatreAnchor.appendChild(pipPlayer);
            }
            if (btn) {
                btn.textContent = '❐';
                btn.title = 'Restore to PiP';
            }
        } else {
            state.playerMode = 'pip';
            pipPlayer.classList.remove('theatre');
            if (pipPlayer.parentElement !== document.body) {
                document.body.appendChild(pipPlayer);
            }
            if (btn) {
                btn.textContent = '□';
                btn.title = 'Theatre Mode';
            }
        }
    }

    // --- Gesture Support ---
    if ('ontouchstart' in window || navigator.maxTouchPoints > 0) {
        let touchStartX = 0;
        let touchStartY = 0;
        let touchStartTime = 0;
        let lastTapTime = 0;
        let initialSeekTime = 0;
        const seekIndicator = document.getElementById('seek-indicator');
        let seekTimer = null;

        function showIndicator(text) {
            if (!seekIndicator) return;
            seekIndicator.textContent = text;
            seekIndicator.classList.remove('hidden');
            if (seekTimer) clearTimeout(seekTimer);
            seekTimer = setTimeout(() => seekIndicator.classList.add('hidden'), 1000);
        }

        pipPlayer.addEventListener('touchstart', (e) => {
            if ((e.target as HTMLElement).closest('#pip-controls') || (e.target as HTMLElement).closest('button') || (e.target as HTMLElement).closest('select')) return;

            const touch = e.changedTouches[0];
            touchStartX = touch.screenX;
            touchStartY = touch.screenY;
            touchStartTime = Date.now();

            const media = pipViewer.querySelector('video, audio');

            // Double tap detection
            const now = Date.now();
            if (now - lastTapTime < 300 && media && media.tagName === 'VIDEO') {
                const rect = pipPlayer.getBoundingClientRect();
                const relativeX = (touch.clientX - rect.left) / rect.width;

                if (relativeX < 0.33) {
                    (media as HTMLMediaElement).currentTime = Math.max(0, (media as HTMLMediaElement).currentTime - 10);
                    showIndicator('⏪ -10s');
                } else if (relativeX > 0.66) {
                    (media as HTMLMediaElement).currentTime = Math.min((media as HTMLMediaElement).duration, (media as HTMLMediaElement).currentTime + 10);
                    showIndicator('⏩ +10s');
                } else {
                    if ((media as HTMLMediaElement).paused) (media as HTMLMediaElement).play();
                    else (media as HTMLMediaElement).pause();
                    showIndicator((media as HTMLMediaElement).paused ? '⏸️' : '▶️');
                }
                lastTapTime = 0;
                touchStartTime = 0; // Prevent swipe after double tap
                if (e.cancelable) e.preventDefault();
                return;
            }
            lastTapTime = now;

            // Initialize seek time on touch
            if (media) initialSeekTime = (media as HTMLMediaElement).currentTime;

        }, { passive: false });

        pipPlayer.addEventListener('touchmove', (e) => {
            if (touchStartTime === 0) return;
            const touch = e.changedTouches[0];
            const diffX = touch.screenX - touchStartX;
            const diffY = touch.screenY - touchStartY;

            const media = pipViewer.querySelector('video, audio');
            if (media && !isNaN((media as HTMLMediaElement).duration)) {
                // MX Player style: movement relative to screen width
                const screenWidth = window.innerWidth;
                const sensitivity = 0.5; // Swiping full screen = half video duration
                const timeDiff = (diffX / screenWidth) * (media as HTMLMediaElement).duration * sensitivity;
                const targetTime = Math.max(0, Math.min((media as HTMLMediaElement).duration, initialSeekTime + timeDiff));
                (media as HTMLMediaElement).currentTime = targetTime;

                const timeStr = formatDuration(targetTime) + ' / ' + formatDuration((media as HTMLMediaElement).duration);
                showIndicator(timeStr);
                if (e.cancelable) e.preventDefault();
            }

            // If it's clearly a gesture for the player, prevent page scroll
            if (Math.abs(diffX) > 10 || Math.abs(diffY) > 10) {
                if (e.cancelable) e.preventDefault();
            }
        }, { passive: false });

        pipPlayer.addEventListener('touchend', (e) => {
            if (touchStartTime === 0) return;

            if ((e.target as HTMLElement).closest('#pip-controls') || (e.target as HTMLElement).closest('button') || (e.target as HTMLElement).closest('select')) {
                touchStartTime = 0;
                return;
            }

            const touch = e.changedTouches[0];
            const diffX = touch.screenX - touchStartX;
            const diffY = touch.screenY - touchStartY;
            const duration = Date.now() - touchStartTime;

            // Thresholds: < 500ms duration
            if (touchStartTime !== 0 && duration < 500) {
                if (Math.abs(diffX) > 60 && Math.abs(diffY) < 80) {
                    if (diffX > 60) {
                        // Swipe Right -> Previous
                        playSibling(-1, true);
                    } else if (diffX < -60) {
                        // Swipe Left -> Next
                        playSibling(1, true);
                    }
                } else if (diffY > 80 && Math.abs(diffX) < 60) {
                    // Swipe Down -> Close
                    closeActivePlayer();
                }
            }
            touchStartTime = 0;
        }, { passive: true });
    }

    const settingPlayer = document.getElementById('setting-player');
    if (settingPlayer) settingPlayer.onchange = (e) => {
        state.player = (e.target as any).value;
        localStorage.setItem('disco-player', String(state.player));
    };

    const settingLanguage = document.getElementById('setting-language');
    if (settingLanguage) settingLanguage.oninput = (e) => {
        state.language = (e.target as any).value;
        localStorage.setItem('disco-language', String(state.language));

        // Update current tracks
        const media = pipViewer.querySelector('video, audio');
        if (media) {
            for (let i = 0; i < (media as HTMLMediaElement).textTracks.length; i++) {
                ((media as HTMLMediaElement).textTracks[i] as any).srclang = state.language;
            }
        }
    };

    const settingTheme = document.getElementById('setting-theme');
    if (settingTheme) settingTheme.onchange = (e) => {
        state.theme = (e.target as any).value;
        localStorage.setItem('disco-theme', String(state.theme));
        applyTheme();
    };

    const settingPostPlayback = document.getElementById('setting-post-playback');
    if (settingPostPlayback) settingPostPlayback.onchange = (e) => {
        state.postPlaybackAction = (e.target as any).value;
        localStorage.setItem('disco-post-playback', String(state.postPlaybackAction));
    };

    const settingDefaultView = document.getElementById('setting-default-view');
    if (settingDefaultView) settingDefaultView.onchange = (e) => {
        state.defaultView = (e.target as any).value;
        localStorage.setItem('disco-default-view', String(state.defaultView));

        if (pipPlayer.classList.contains('hidden')) {
            state.playerMode = state.defaultView;
        }
    };



    const settingAutoplay = document.getElementById('setting-autoplay');
    if (settingAutoplay) {
        (settingAutoplay as HTMLInputElement).checked = state.autoplay;
        (settingAutoplay as HTMLElement).onchange = (e) => {
            state.autoplay = (e.target as any).checked;
            localStorage.setItem('disco-autoplay', String(state.autoplay.toString()));
        };
    }

    const settingEnableQueueOnchange = document.getElementById('setting-enable-queue');
    if (settingEnableQueueOnchange) {
        settingEnableQueueOnchange.addEventListener('change', (e) => {
            state.enableQueue = (e.target as any).checked;
            localStorage.setItem('disco-enable-queue', String(state.enableQueue.toString()));
            updateQueueVisibility();
        });
    }

    const settingRsvpWpm = document.getElementById('setting-rsvp-wpm');
    if (settingRsvpWpm) {
        (settingRsvpWpm as HTMLInputElement).value = state.rsvpWpm.toString();
        (settingRsvpWpm as HTMLElement).onchange = (e) => {
            state.rsvpWpm = parseInt((e.target as any).value) || 250;
            localStorage.setItem('disco-rsvp-wpm', String(state.rsvpWpm.toString()));
        };
    }

    if (settingImageAutoplay) settingImageAutoplay.onchange = (e) => {
        state.imageAutoplay = (e.target as any).checked;
        localStorage.setItem('disco-image-autoplay', String(state.imageAutoplay.toString()));
    };

    const settingLocalResume = document.getElementById('setting-local-resume');
    if (settingLocalResume) settingLocalResume.onchange = (e) => {
        state.localResume = (e.target as any).checked;
        localStorage.setItem('disco-local-resume', String(state.localResume.toString()));
    };

    function calculateStorageSize() {
        const display = document.getElementById('storage-size');
        if (!display) return;

        let total = 0;
        for (let i = 0; i < localStorage.length; i++) {
            const key = localStorage.key(i);
            const val = localStorage.getItem(key);
            total += (key.length + val.length) * 2; // Rough estimate in bytes (UTF-16)
        }

        display.textContent = formatSize(total);
    }

    const clearStorageBtn = document.getElementById('clear-storage-btn');
    if (clearStorageBtn) {
        clearStorageBtn.onclick = () => {
            const keysToKeep = [
                'disco-player', 'disco-language', 'disco-theme',
                'disco-post-playback', 'disco-autoplay', 'disco-local-resume',
                'disco-limit', 'disco-limit-all', 'disco-excluded-dbs',
                'disco-play-counts'
            ];

            const keys = Object.keys(localStorage);
            keys.forEach(key => {
                if (key.startsWith('disco-') && !keysToKeep.includes(key)) {
                    localStorage.removeItem(key);
                }
            });

            showToast(`Cleared temporary items`, '🧹');
            calculateStorageSize();
        };
    }

    // Close modal on outside click
    window.onclick = (event) => {
        if (event.target.classList.contains('modal')) {
            event.target.classList.add('hidden');
        }
    };

    if (searchInput) {
        searchInput.onkeypress = (e) => { if (e.key === 'Enter') performSearch(); };
    }

    function updateNavActiveStates() {
        const toolbar = document.getElementById('toolbar');
        const searchContainer = document.getElementById('search-container');
        if (state.page === 'curation') {
            if (toolbar) toolbar.classList.add('hidden');
            if (searchContainer) searchContainer.classList.add('hidden');
        } else {
            if (toolbar) toolbar.classList.remove('hidden');
            if (searchContainer) searchContainer.classList.remove('hidden');
        }

        // Update Media Type buttons
        document.querySelectorAll('#media-type-list .category-btn').forEach(btn => {
            const isActive = state.filters.types.includes((btn as any).dataset.type);
            btn.classList.toggle('active', isActive);
        });

        // Update Sliders
        const epFilter = state.filters.episodes.find(f => f.value === '@p');
        if (epFilter) {
            setSliderValues('episodes', epFilter.min, epFilter.max);
        } else {
            setSliderValues('episodes', 0, 100);
        }

        const sizeFilter = state.filters.sizes.find(f => f.value === '@p');
        if (sizeFilter) {
            setSliderValues('size', sizeFilter.min, sizeFilter.max);
        } else {
            setSliderValues('size', 0, 100);
        }

        const durFilter = state.filters.durations.find(f => f.value === '@p');
        if (durFilter) {
            setSliderValues('duration', durFilter.min, durFilter.max);
        } else {
            setSliderValues('duration', 0, 100);
        }
        updateSliderLabels();

        // Update Bins
        if (state.filterBins) {
            document.querySelectorAll('#episodes-list .category-btn').forEach(btn => {
                const bin = state.filterBins.episodes[(btn as any).dataset.index];
                if (bin) btn.classList.toggle('active', state.filters.episodes.some(b => b.label === bin.label));
            });
            document.querySelectorAll('#size-list .category-btn').forEach(btn => {
                const bin = state.filterBins.size[(btn as any).dataset.index];
                if (bin) btn.classList.toggle('active', state.filters.sizes.some(b => b.label === bin.label));
            });
            document.querySelectorAll('#duration-list .category-btn').forEach(btn => {
                const bin = state.filterBins.duration[(btn as any).dataset.index];
                if (bin) btn.classList.toggle('active', state.filters.durations.some(b => b.label === bin.label));
            });
        }

        if (allMediaBtn) allMediaBtn.classList.toggle('active', state.page === 'search' && state.filters.categories.length === 0 && state.filters.genre === '' && state.filters.languages.length === 0 && state.filters.ratings.length === 0 && !state.filters.playlist && !state.filters.unplayed && !state.filters.unfinished && !state.filters.completed && state.filters.sizes.length === 0 && state.filters.durations.length === 0 && state.filters.episodes.length === 0 && state.filters.types.length === 0);
        if (trashBtn) trashBtn.classList.toggle('active', state.page === 'trash');
        if (duBtn) duBtn.classList.toggle('active', state.page === 'du');
        if (captionsBtn) captionsBtn.classList.toggle('active', state.page === 'captions');

        if (historyInProgressBtn) historyInProgressBtn.classList.toggle('active', state.filters.unfinished);
        if (historyUnplayedBtn) historyUnplayedBtn.classList.toggle('active', state.filters.unplayed);
        if (historyCompletedBtn) historyCompletedBtn.classList.toggle('active', state.filters.completed);

        // View Toggles
        if (viewGrid) viewGrid.classList.toggle('active', state.view === 'grid');
        if (viewGroup) {
            // Hide Group view button in DU mode (doesn't make sense for disk usage)
            viewGroup.style.display = state.page === 'du' ? 'none' : '';
            viewGroup.classList.toggle('active', state.view === 'group');
        }
        if (viewDetails) viewDetails.classList.toggle('active', state.view === 'details');

        // Handle playlists and categories in the sidebar lists
        document.querySelectorAll('#sidebar .category-btn').forEach(btn => {
            if (btn === allMediaBtn || btn === trashBtn || btn === duBtn || btn === captionsBtn || btn === historyInProgressBtn || btn === historyUnplayedBtn || btn === historyCompletedBtn) return;
            if (btn.closest('#media-type-list')) return;
            if (btn.closest('#episodes-list') || btn.closest('#size-list') || btn.closest('#duration-list')) return;

            const cat = (btn as any).dataset.cat;
            const genre = (btn as any).dataset.genre;
            const lang = (btn as any).dataset.lang;
            const rating = (btn as any).dataset.rating;
            const type = (btn as any).dataset.type;
            // For playlists, we check both the button itself and if it's a wrapper for a drop zone
            const playlist = (btn as any).dataset.title || (btn.querySelector('.playlist-name') as HTMLElement)?.dataset.title;

            let isActive = false;
            if (cat !== undefined) isActive = state.page === 'search' && state.filters.categories.includes(cat);
            else if (genre !== undefined) isActive = state.page === 'search' && state.filters.genre === genre;
            else if (lang !== undefined) isActive = state.page === 'search' && state.filters.languages.includes(lang);
            else if (rating !== undefined) isActive = state.page === 'search' && state.filters.ratings.includes(rating);
            else if (playlist !== undefined) isActive = state.page === 'playlist' && state.filters.playlist === playlist;
            else if (type !== undefined) isActive = state.page === 'search' && state.filters.types.length === 1 && state.filters.types[0] === type;

            btn.classList.toggle('active', isActive);
        });
    }

    function clearAllDragOver() {
        document.querySelectorAll('.drag-over').forEach(el => el.classList.remove('drag-over'));
    }

    if (allMediaBtn) {
        allMediaBtn.onclick = () => {
            state.page = 'search';
            state.currentPage = 1;

            updateNavActiveStates();
            performSearch();
        };
    }

    if (trashBtn) {
        trashBtn.onclick = () => {
            if (state.page === 'trash') {
                // Toggle off trash mode if already active
                state.page = 'search';
                updateNavActiveStates();
                performSearch();
                return;
            }

            // Reset ALL filters when entering trash mode for safety
            // This prevents dangerous situations where filters could show untrashed files in trash view
            resetFilters();

            // Save to localStorage
            clearAllFilters();

            state.page = 'trash';
            updateNavActiveStates();
            fetchTrash();
        };

        trashBtn.addEventListener('dragenter', (e) => {
            e.preventDefault();
            trashBtn.classList.add('drag-over');
        });

        trashBtn.addEventListener('dragover', (e) => {
            e.preventDefault();
            (e as DragEvent).dataTransfer.dropEffect = 'move';
        });

        trashBtn.addEventListener('dragleave', (e) => {
            if (!trashBtn.contains(((e as MouseEvent).relatedTarget as HTMLElement))) {
                trashBtn.classList.remove('drag-over');
            }
        });

        trashBtn.addEventListener('drop', async (e) => {
            e.preventDefault();
            e.stopPropagation();
            trashBtn.classList.remove('drag-over');

            const path = (e as DragEvent).dataTransfer.getData('text/plain');
            if (path) {
                deleteMedia(path);
            }
            state.draggedItem = null;
            document.body.classList.remove('is-dragging');
        });
    }

    if (historyInProgressBtn) {
        historyInProgressBtn.onclick = () => {
            if (state.filters.unfinished) {
                // Toggle off
                state.filters.unfinished = false;
            } else {
                // Toggle on - mutually exclusive with other history filters
                state.filters.unfinished = true;
                state.filters.completed = false;
                state.filters.unplayed = false;
            }
            state.currentPage = 1;
            updateNavActiveStates();
            performSearch();
        };
    }

    if (historyUnplayedBtn) {
        historyUnplayedBtn.onclick = () => {
            if (state.filters.unplayed) {
                // Toggle off
                state.filters.unplayed = false;
            } else {
                // Toggle on - mutually exclusive with other history filters
                state.filters.unplayed = true;
                state.filters.unfinished = false;
                state.filters.completed = false;
            }
            state.currentPage = 1;
            updateNavActiveStates();
            performSearch();
        };
    }

    if (historyCompletedBtn) {
        historyCompletedBtn.onclick = () => {
            if (state.filters.completed) {
                // Toggle off
                state.filters.completed = false;
            } else {
                // Toggle on - mutually exclusive with other history filters
                state.filters.completed = true;
                state.filters.unfinished = false;
                state.filters.unplayed = false;
            }
            state.currentPage = 1;
            updateNavActiveStates();
            performSearch();
        };
    }

    if (duBtn) {
        duBtn.onclick = () => {
            if (state.page === 'du') {
                // Toggle off DU mode if already active
                state.page = 'search';
                updateNavActiveStates();
                performSearch();
            } else {
                state.page = 'du';
                updateNavActiveStates();
                fetchDU(state.duPath);
            }
        };
    }

    // DU back button
    const duBackBtn = document.getElementById('du-back-btn');
    if (duBackBtn) {
        duBackBtn.onclick = () => {
            if (state.duPath && state.duPath !== '/' && state.duPath !== '.') {
                let p = state.duPath;
                if (p.endsWith('/') && p.length > 1) p = p.slice(0, -1);
                const lastSlash = p.lastIndexOf('/');
                if (lastSlash === -1) {
                    fetchDU('');
                } else {
                    let parent = p.substring(0, lastSlash + 1);
                    fetchDU(parent);
                }
            }
        };
    }

    // DU path input - allows editing and navigation
    const duPathInput = document.getElementById('du-path-input');
    if (duPathInput) {
        // Navigate to path on Enter
        duPathInput.addEventListener('keydown', (e) => {
            if (e.key === 'Enter') {
                e.preventDefault();
                const newPath = (duPathInput as HTMLInputElement).value.trim();
                if (newPath && newPath !== state.duPath) {
                    fetchDU(newPath);
                }
            }
        });

        // Also select on focus
        duPathInput.addEventListener('focus', () => {
            (duPathInput as HTMLInputElement).select();
        });
    }

    if (captionsBtn) {
        captionsBtn.onclick = () => {
            state.page = 'captions';
            updateNavActiveStates();
            performSearch();
        };
    }

    const newPlaylistBtn = document.getElementById('new-playlist-btn');
    if (newPlaylistBtn) {
        newPlaylistBtn.onclick = () => {
            const title = prompt('Playlist Title:');
            if (title) createPlaylist(title);
        };
    }

    if (sortBy) sortBy.onchange = async () => {
        // If "Custom" is selected, open the custom sorting modal instead of querying
        if (sortBy.value === 'custom') {
            const modal = document.getElementById('sort-complex-modal');
            if (modal) {
                loadConfigFromCurrentSort();
                modal.classList.remove('hidden');
            }
            return;
        }

        state.filters.sort = sortBy.value;
        localStorage.setItem('disco-sort', String(state.filters.sort));

        // Clear custom sort fields when selecting a preset
        if (sortBy.value !== 'custom') {
            state.filters.customSortFields = '';
            localStorage.removeItem('disco-custom-sort-fields');
        }

        if (state.page === 'playlist') {
            sortPlaylistItems();
            renderResults();
        } else if (state.page === 'du') {
            state.currentPage = 1;
            fetchDU(state.duPath);
        } else {
            state.currentPage = 1;
            performSearch();
        }
    };

    if (sortReverseBtn) sortReverseBtn.onclick = () => {
        state.filters.reverse = !state.filters.reverse;
        localStorage.setItem('disco-reverse', String(state.filters.reverse));
        sortReverseBtn.classList.toggle('active');
        if (state.page === 'playlist') {
            sortPlaylistItems();
            renderResults();
        } else if (state.page === 'du') {
            state.currentPage = 1;
            fetchDU(state.duPath);
        } else {
            state.currentPage = 1;
            performSearch();
        }
    };

    function sortPlaylistItems() {
        const field = state.filters.sort;
        const reverse = state.filters.reverse;

        if (field === 'default') {
            // Default sort is by track_number (asc), then time_added (asc) which is not available in frontend yet, so we use original order or path
            // Since backend sends it sorted, we might just want to re-fetch or rely on current order if we haven't messed it up.
            // But if we want to be safe, we can sort by track_number.
            currentMedia.sort((a, b) => {
                const trackA = a.track_number || Number.MAX_SAFE_INTEGER;
                const trackB = b.track_number || Number.MAX_SAFE_INTEGER;
                if (trackA !== trackB) return trackA - trackB;
                return a.path.localeCompare(b.path);
            });
            if (reverse) currentMedia.reverse();
            return;
        }

        currentMedia.sort((a, b) => {
            let valA, valB;

            switch (field) {
                case 'path': valA = a.path; valB = b.path; break;
                case 'size': valA = a.size || 0; valB = b.size || 0; break;
                case 'duration': valA = a.duration || 0; valB = b.duration || 0; break;
                case 'play_count': valA = getPlayCount(a); valB = getPlayCount(b); break;
                case 'time_last_played': valA = a.time_last_played || 0; valB = b.time_last_played || 0; break;
                case 'progress':
                    valA = (a.duration && a.playhead) ? a.playhead / a.duration : 0;
                    valB = (b.duration && b.playhead) ? b.playhead / b.duration : 0;
                    break;
                case 'time_created': valA = a.time_created || 0; valB = b.time_created || 0; break;
                case 'time_modified': valA = a.time_modified || 0; valB = b.time_modified || 0; break;
                case 'bitrate':
                    // Estimate bitrate
                    valA = (a.size && a.duration) ? a.size / a.duration : 0;
                    valB = (b.size && b.duration) ? b.size / b.duration : 0;
                    break;
                case 'extension':
                    valA = a.path.split('.').pop().toLowerCase();
                    valB = b.path.split('.').pop().toLowerCase();
                    break;
                case 'type':
                    valA = a.type || '';
                    valB = b.type || '';
                    break;
                case 'random': return Math.random() - 0.5;
                default: return 0;
            }

            if (typeof valA === 'string') {
                return reverse ? valB.localeCompare(valA) : valA.localeCompare(valB);
            }
            return reverse ? valB - valA : valA - valB;
        });
    }

    if (limitInput) limitInput.oninput = debounce(() => {
        state.currentPage = 1;
        performSearch();
    }, 500);
    if (limitAll) limitAll.onchange = () => {
        state.currentPage = 1;
        performSearch();
    };

    if (viewGrid) {
        viewGrid.onclick = () => {
            state.view = 'grid';
            localStorage.setItem('disco-view', String('grid'));
            updateNavActiveStates();
            performSearch();
        };
    }

    if (viewGroup) {
        viewGroup.onclick = () => {
            state.view = 'group';
            localStorage.setItem('disco-view', String('group'));
            updateNavActiveStates();
            performSearch();
        };
    }

    if (viewDetails) {
        viewDetails.onclick = () => {
            state.view = 'details';
            localStorage.setItem('disco-view', String('details'));
            updateNavActiveStates();
            performSearch();
        };
    }

    if (prevPageBtn) (prevPageBtn as HTMLElement).onclick = () => {
        if (state.currentPage > 1) {
            state.currentPage--;
            performSearch();
            resultsContainer.scrollTo(0, 0);
        }
    };

    if (nextPageBtn) (nextPageBtn as HTMLElement).onclick = () => {
        const totalPages = Math.ceil(state.totalCount / state.filters.limit);
        if (state.currentPage < totalPages) {
            state.currentPage++;
            performSearch();
            resultsContainer.scrollTo(0, 0);
        }
    };

    // --- Inactivity Tracking ---
    const logoText = document.getElementById('logo-text');
    const activityEvents = ['mousedown', 'mousemove', 'keydown', 'scroll', 'touchstart'];

    activityEvents.forEach(name => {
        window.addEventListener(name, () => {
            const now = Date.now();
            const inactiveTime = now - state.lastActivity;

            if (inactiveTime > 3 * 60 * 1000) { // 3 minutes
                if (logoText) {
                    logoText.classList.remove('shimmering');
                    void logoText.offsetWidth; // Trigger reflow
                    logoText.classList.add('shimmering');

                    // Remove class when done so it's clean
                    logoText.onanimationend = () => {
                        logoText.classList.remove('shimmering');
                    };
                }
            }

            state.lastActivity = now;
        }, { passive: true });
    });

    const logoReset = document.getElementById('logo');
    if (logoReset) {
        logoReset.style.cursor = 'pointer';
        logoReset.onclick = () => {
            state.page = 'search';
            state.currentPage = 1;
            resetSidebar();
            performSearch();
        };
    }

    // --- Mobile Sidebar Controls ---
    function toggleMobileSidebar() {
        const isOpen = sidebar.classList.contains('mobile-open');
        if (!isOpen) {
            state.activeModal = 'mobile-sidebar';
        } else if (state.activeModal === 'mobile-sidebar') {
            state.activeModal = null;
        }
        sidebar.classList.toggle('mobile-open');
        sidebarOverlay.classList.toggle('hidden');
        syncUrl();
    }

    function closeMobileSidebar() {
        if (sidebar.classList.contains('mobile-open')) {
            sidebar.classList.remove('mobile-open');
            sidebarOverlay.classList.add('hidden');
            if (state.activeModal === 'mobile-sidebar') {
                state.activeModal = null;
            }
            syncUrl();
        }
    }

    if (menuToggle) menuToggle.onclick = toggleMobileSidebar;
    if (sidebarOverlay) sidebarOverlay.onclick = closeMobileSidebar;

    // Close sidebar when clicking on a category, genre, rating or playlist on mobile
    // Also close on media selection to show the player/content
    document.addEventListener('click', (e) => {
        const target = e.target as HTMLElement;
        const isClickable = target.closest('.category-btn') ||
            target.closest('.playlist-name') ||
            target.closest('#trash-btn') ||
            target.closest('#history-btn') ||
            target.closest('.media-card');

        if (isClickable && window.innerWidth <= 768) {
            closeMobileSidebar();
        }
    });

    // Initial load
    readUrl(true);
    fetchDatabases();

    fetchCategories();
    fetchLanguages();
    fetchGenres();
    fetchRatings();
    fetchPlaylists();
    // Filter bins are now fetched with search results via include_counts=true
    renderMediaTypeList(); // Render media type buttons on initial load
    renderCategoryList();
    initSidebarPersistence();
    initQueueControls();
    updateQueueVisibility();
    onUrlChange();
    applyTheme();

    document.addEventListener('fullscreenchange', () => {
        const fsBtn = document.getElementById('doc-fullscreen');
        if (fsBtn) {
            fsBtn.title = document.fullscreenElement ? 'Exit Fullscreen' : 'Toggle Fullscreen';
        }

        // If exiting fullscreen, move toast back to body
        if (!document.fullscreenElement && toast.parentElement !== document.body) {
            document.body.appendChild(toast);
        }
    });

    // Expose for testing
    (window as any).disco = {
        get currentMedia() { return currentMedia; },
        set currentMedia(v) { currentMedia = v; },
        formatSize,
        formatDuration,
        shortDuration,
        getIcon,
        truncateString,
        formatRelativeDate,
        formatParents,
        openInPiP,
        openInDocumentViewer,
        openActivePlayer,
        closeActivePlayer,
        openModal,
        closeModal,
        toggleMobileSidebar,
        closeMobileSidebar,
        performSearch,
        updateProgress,
        handleMediaError,
        seekToProgress,
        closePiP,
        getPlayCount,
        markMediaPlayed,
        updateNavActiveStates,
        playSibling,
        renderPagination,
        renderCaptionsList,
        readUrl,
        syncUrl,
        showToast,
        resetFilters,
        updateSliderLabels,
        updateQueueVisibility,
        renderQueue,
        startSlideshow,
        stopSlideshow,
        state
    };
});
