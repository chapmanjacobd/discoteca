import { fetchAPI } from './api.js';
import { state } from './state.js';
import {
    formatSize,
    formatDuration,
    formatRelativeDate,
    shortDuration,
    truncateString,
    formatParents,
    getIcon
} from './utils.js';

document.addEventListener('DOMContentLoaded', () => {
    const searchInput = document.getElementById('search-input');
    const resultsContainer = document.getElementById('results-container');
    const resultsCount = document.getElementById('results-count');
    const sortBy = document.getElementById('sort-by');
    const sortReverseBtn = document.getElementById('sort-reverse-btn');
    const limitInput = document.getElementById('limit');
    const limitAll = document.getElementById('limit-all');
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
    const sidebar = document.querySelector('.sidebar');

    const duBtn = document.getElementById('du-btn');
    const captionsBtn = document.getElementById('captions-btn');
    const curationBtn = document.getElementById('curation-btn');
    const channelSurfBtn = document.getElementById('channel-surf-btn');
    const filterCaptions = document.getElementById('filter-captions');

    const pipPlayer = document.getElementById('pip-player');
    const pipLoading = document.getElementById('pip-loading');
    const pipViewer = document.getElementById('media-viewer');
    const pipTitle = document.getElementById('media-title');
    if (pipTitle) {
        pipTitle.onclick = () => {
            const range = document.createRange();
            range.selectNodeContents(pipTitle);
            const selection = window.getSelection();
            selection.removeAllRanges();
            selection.addRange(range);
        };
    }
    const lyricsDisplay = document.getElementById('lyrics-display');
    const secondarySubtitle = document.getElementById('secondary-subtitle');
    const searchSuggestions = document.getElementById('search-suggestions');

    const viewGroup = document.getElementById('view-group');

    const historyInProgressBtn = document.getElementById('history-in-progress-btn');
    const historyUnplayedBtn = document.getElementById('history-unplayed-btn');
    const historyCompletedBtn = document.getElementById('history-completed-btn');

    const allMediaBtn = document.getElementById('all-media-btn');
    const trashBtn = document.getElementById('trash-btn');

    // Percentile Sliders
    const episodesMinSlider = document.getElementById('episodes-min-slider');
    const episodesMaxSlider = document.getElementById('episodes-max-slider');
    const episodesLabel = document.getElementById('episodes-percentile-label');

    const sizeMinSlider = document.getElementById('size-min-slider');
    const sizeMaxSlider = document.getElementById('size-max-slider');
    const sizeLabel = document.getElementById('size-percentile-label');

    const durationMinSlider = document.getElementById('duration-min-slider');
    const durationMaxSlider = document.getElementById('duration-max-slider');
    const durationLabel = document.getElementById('duration-percentile-label');

    const epMinLabel = document.getElementById('episodes-min-label');
    const epMaxLabel = document.getElementById('episodes-max-label');
    const sizeMinLabel = document.getElementById('size-min-label');
    const sizeMaxLabel = document.getElementById('size-max-label');
    const durMinLabel = document.getElementById('duration-min-label');
    const durMaxLabel = document.getElementById('duration-max-label');

    const settingTrackShuffleDuration = document.getElementById('setting-track-shuffle-duration');

    const pipSpeedBtn = document.getElementById('pip-speed');
    const pipSpeedMenu = document.getElementById('pip-speed-menu');
    const pipSubtitlesBtn = document.getElementById('pip-subtitles');

    if (pipSubtitlesBtn) {
        pipSubtitlesBtn.onclick = () => {
            renderSubtitleList();
            openModal('subtitle-modal');
        };
    }

    function renderSubtitleList() {
        const list = document.getElementById('subtitle-list');
        if (!list) return;
        list.innerHTML = '';

        const media = pipViewer.querySelector('video, audio');
        if (!media || !media.textTracks || media.textTracks.length === 0) {
            list.innerHTML = '<p style="text-align: center; padding: 1rem; color: var(--text-muted);">No subtitle tracks found.</p>';
            return;
        }

        const tracks = Array.from(media.textTracks);

        // Add "None" option
        const noneBtn = document.createElement('button');
        noneBtn.className = 'category-btn';
        noneBtn.style.width = '100%';
        noneBtn.style.marginBottom = '0.5rem';
        noneBtn.textContent = 'Disable All';
        noneBtn.onclick = () => {
            tracks.forEach(t => t.mode = 'disabled');
            renderSubtitleList();
        };
        list.appendChild(noneBtn);

        tracks.forEach((track, index) => {
            const row = document.createElement('div');
            row.style.display = 'flex';
            row.style.gap = '0.5rem';
            row.style.alignItems = 'center';

            const primaryBtn = document.createElement('button');
            primaryBtn.className = 'category-btn' + (track.mode === 'showing' ? ' active' : '');
            primaryBtn.style.flex = '1';
            primaryBtn.style.textAlign = 'left';
            primaryBtn.textContent = track.label || `Track ${index + 1}`;
            primaryBtn.onclick = () => {
                const currentMode = track.mode;
                tracks.forEach(t => { if (t.mode === 'showing') t.mode = 'disabled'; });
                track.mode = (currentMode === 'showing') ? 'disabled' : 'showing';
                renderSubtitleList();
            };

            const secondaryBtn = document.createElement('button');
            secondaryBtn.className = 'category-btn' + (track.mode === 'hidden' ? ' active' : '');
            secondaryBtn.textContent = '2nd';
            secondaryBtn.title = 'Show as secondary subtitle (at top of player)';
            secondaryBtn.style.padding = '0.5rem';
            secondaryBtn.onclick = () => {
                const currentMode = track.mode;
                tracks.forEach(t => { if (t.mode === 'hidden') t.mode = 'disabled'; });
                track.mode = (currentMode === 'hidden') ? 'disabled' : 'hidden';
                renderSubtitleList();
            };

            row.appendChild(primaryBtn);
            row.appendChild(secondaryBtn);
            list.appendChild(row);
        });
    }

    const filterBrowseCol = document.getElementById('filter-browse-col');
    const filterBrowseVal = document.getElementById('filter-browse-val');
    const filterBrowseValContainer = document.getElementById('filter-browse-val-container');

    let currentMedia = [];
    let allDatabases = [];
    let searchAbortController = null;
    let suggestionAbortController = null;
    let selectedSuggestionIndex = -1;

    // --- State Management ---


    function formatSliderValue(type, val) {
        return Math.round(val).toString() + '%';
    }

    function updateSliderLabels() {
        const updateRange = (minSlider, maxSlider, label, minF, maxF, type) => {
            if (!minSlider || !state.filterBins) return;

            const minP = parseInt(minSlider.value);
            const maxP = parseInt(maxSlider.value);

            const percentiles = state.filterBins[`${type}_percentiles`] || [];
            const getVal = (p) => {
                if (percentiles.length > p) return percentiles[p];

                // Fallback to linear if percentiles missing
                let minTotal = 0, maxTotal = 0;
                if (type === 'episodes') { minTotal = state.filterBins.episodes_min; maxTotal = state.filterBins.episodes_max; }
                else if (type === 'size') { minTotal = state.filterBins.size_min; maxTotal = state.filterBins.size_max; }
                else if (type === 'duration') { minTotal = state.filterBins.duration_min; maxTotal = state.filterBins.duration_max; }
                return minTotal + (maxTotal - minTotal) * (p / 100);
            };

            const valMin = getVal(minP);
            const valMax = getVal(maxP);

            const format = (v) => {
                if (type === 'size') return formatSize(v);
                if (type === 'duration') return formatDuration(v);
                return Math.round(v).toString();
            };

            if (label) label.textContent = `${format(valMin)} - ${format(valMax)}`;

            if (minF) minF.textContent = format(getVal(0));
            if (maxF) maxF.textContent = format(getVal(100));

            const track = minSlider.parentElement.querySelector('.range-track');
            if (track) {
                track.style.background = `linear-gradient(to right,
                    var(--border-color) ${minP}%,
                    var(--accent-color) ${minP}%,
                    var(--accent-color) ${maxP}%,
                    var(--border-color) ${maxP}%)`;
            }
        };

        updateRange(episodesMinSlider, episodesMaxSlider, episodesLabel, epMinLabel, epMaxLabel, 'episodes');
        updateRange(sizeMinSlider, sizeMaxSlider, sizeLabel, sizeMinLabel, sizeMaxLabel, 'size');
        updateRange(durationMinSlider, durationMaxSlider, durationLabel, durMinLabel, durMaxLabel, 'duration');
    }

    function handleSliderChange(type, minP, maxP) {
        let filterKey = '';
        let lsKey = '';

        if (!state.filterBins) return;

        if (type === 'episodes') { filterKey = 'episodes'; lsKey = 'disco-filter-episodes'; }
        else if (type === 'size') { filterKey = 'sizes'; lsKey = 'disco-filter-sizes'; }
        else if (type === 'duration') { filterKey = 'durations'; lsKey = 'disco-filter-durations'; }

        if (!filterKey) return;

        // Use percentiles for population weighting and correct filtering
        state.filters[filterKey] = [{
            label: `${minP}-${maxP}%`,
            value: `@p`,
            min: parseInt(minP),
            max: parseInt(maxP)
        }];

        localStorage.setItem(lsKey, JSON.stringify(state.filters[filterKey]));
        updateSliderLabels();
        performSearch();
    }

    const initSlider = (minSlider, maxSlider, type, filterKey, isEpisodes = false) => {
        if (!minSlider) return;

        // Restore from state
        const filter = state.filters[filterKey] && state.filters[filterKey].find(f => f.value === '@p' || f.value === '@abs');
        if (filter) {
            if (filter.value === '@p') {
                minSlider.value = filter.min;
                maxSlider.value = filter.max;
            }
            // @abs is handled in fetchFilterBins because we need the current distribution
        }

        const onInput = (e) => {
            let min = parseInt(minSlider.value);
            let max = parseInt(maxSlider.value);

            if (min > max) {
                if (e.target === minSlider) {
                    maxSlider.value = min;
                } else {
                    minSlider.value = max;
                }
            }
            updateSliderLabels();
        };

        minSlider.oninput = onInput;
        maxSlider.oninput = onInput;
        minSlider.onchange = () => {
            handleSliderChange(type, minSlider.value, maxSlider.value);
        };
        maxSlider.onchange = () => {
            handleSliderChange(type, minSlider.value, maxSlider.value);
        };
    };

    initSlider(episodesMinSlider, episodesMaxSlider, 'episodes', 'episodes', true);
    initSlider(sizeMinSlider, sizeMaxSlider, 'size', 'sizes');
    initSlider(durationMinSlider, durationMaxSlider, 'duration', 'durations');
    updateSliderLabels();

    // Initialize UI from state
    document.getElementById('setting-player').value = state.player;
    document.getElementById('setting-language').value = state.language;
    document.getElementById('setting-theme').value = state.theme;
    document.getElementById('setting-post-playback').value = state.postPlaybackAction;
    document.getElementById('setting-default-view').value = state.defaultView;

    document.getElementById('setting-autoplay').checked = state.autoplay;
    const settingImageAutoplay = document.getElementById('setting-image-autoplay');
    if (settingImageAutoplay) settingImageAutoplay.checked = state.imageAutoplay;
    document.getElementById('setting-local-resume').checked = state.localResume;
    document.getElementById('setting-default-video-rate').value = state.defaultVideoRate;
    document.getElementById('setting-default-audio-rate').value = state.defaultAudioRate;
    document.getElementById('setting-slideshow-delay').value = state.slideshowDelay;
    if (settingTrackShuffleDuration) settingTrackShuffleDuration.value = state.trackShuffleDuration;
    if (limitInput) limitInput.value = state.filters.limit;
    if (limitAll) limitAll.checked = state.filters.all;
    const initialUnplayedEl = document.getElementById('filter-unplayed');
    if (initialUnplayedEl) initialUnplayedEl.checked = state.filters.unplayed;
    if (filterCaptions) {
        filterCaptions.checked = state.filters.captions;
        filterCaptions.onchange = (e) => {
            state.filters.captions = e.target.checked;
            localStorage.setItem('disco-captions', state.filters.captions);
            performSearch();
        };
    }

    if (channelSurfBtn) {
        channelSurfBtn.onclick = async (e) => {
            const isAutomated = e && e.detail && e.detail.isAutomated;
            const isManual = e && (e.isTrusted || e.detail?.isManual);

            if (state.playback.isSurfing) {
                if (isManual) {
                    state.playback.isSurfing = false;
                    if (state.playback.surfTimer) {
                        clearTimeout(state.playback.surfTimer);
                        state.playback.surfTimer = null;
                    }
                    showToast('Channel Surf Stopped', 'ℹ️');
                    channelSurfBtn.classList.remove('active');
                    return;
                }
            } else {
                if (isAutomated) return;
                state.playback.isSurfing = true;
            }

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
                if (type) params.append('type', type);

                // Add duration param
                const duration = state.trackShuffleDuration || 0;
                params.append('duration', duration);

                const resp = await fetchAPI(`/api/random-clip?${params.toString()}`);
                if (!resp.ok) {
                    if (resp.status === 403) {
                        showToast('Access Denied', '🚫');
                        return;
                    }
                    if (resp.status === 404) {                        showToast(`No more ${type || 'media'} found to surf.`, 'ℹ️');
                        return;
                    }
                    throw new Error('Failed to fetch random clip');
                }
                const data = await resp.json();
                if (!data || !data.path) {
                    showToast(`No more ${type || 'media'} found to surf.`, 'ℹ️');
                    return;
                }

                // Show toast about what's playing
                const filename = data.path.split('/').pop();
                showToast(`Channel Surf: ${filename} (${formatDuration(data.start)})`, '🔀');

                channelSurfBtn.classList.add('active');

                // Open in PiP
                await openInPiP(data, true, true);

                // Seek to the random start time
                const media = pipViewer.querySelector('video, audio');
                if (media) {
                    media.currentTime = data.start;

                    // Add listener for end of clip
                    if (data.end && data.end > data.start) {
                        const checkTime = () => {
                            if (!state.playback.isSurfing) {
                                media.removeEventListener('timeupdate', checkTime);
                                return;
                            }
                            if (media.currentTime >= data.end) {
                                media.removeEventListener('timeupdate', checkTime);
                                // Trigger next surf
                                if (channelSurfBtn) channelSurfBtn.dispatchEvent(new CustomEvent('click', { detail: { isAutomated: true } }));
                            }
                        };
                        media.addEventListener('timeupdate', checkTime);
                    }
                } else {
                    // Handle images
                    const img = pipViewer.querySelector('img');
                    if (img) {
                        // Use slideshow delay for images when surfing
                        const delay = state.slideshowDelay || 5;
                        if (state.playback.surfTimer) clearTimeout(state.playback.surfTimer);
                        state.playback.surfTimer = setTimeout(() => {
                            state.playback.surfTimer = null;
                            if (!state.playback.isSurfing) return;
                            if (channelSurfBtn) channelSurfBtn.dispatchEvent(new CustomEvent('click', { detail: { isAutomated: true } }));
                        }, delay * 1000);
                    }
                }
            } catch (err) {
                console.error('Channel surf failed:', err);
                errorToast(err, 'Channel surf failed');
                state.playback.isSurfing = false;
                channelSurfBtn.classList.remove('active');
            }
        };
    }

    const settingDefaultVideoRate = document.getElementById('setting-default-video-rate');
    if (settingDefaultVideoRate) {
        settingDefaultVideoRate.onchange = (e) => {
            state.defaultVideoRate = parseFloat(e.target.value);
            localStorage.setItem('disco-default-video-rate', state.defaultVideoRate);
            if (state.playback.item && state.playback.item.type.includes('video')) {
                setPlaybackRate(state.defaultVideoRate);
            }
        };
    }

    const settingDefaultAudioRate = document.getElementById('setting-default-audio-rate');
    if (settingDefaultAudioRate) {
        settingDefaultAudioRate.onchange = (e) => {
            state.defaultAudioRate = parseFloat(e.target.value);
            localStorage.setItem('disco-default-audio-rate', state.defaultAudioRate);
            if (state.playback.item && state.playback.item.type.includes('audio')) {
                setPlaybackRate(state.defaultAudioRate);
            }
        };
    }

    const settingSlideshowDelay = document.getElementById('setting-slideshow-delay');
    if (settingSlideshowDelay) {
        settingSlideshowDelay.onchange = (e) => {
            state.slideshowDelay = parseInt(e.target.value);
            localStorage.setItem('disco-slideshow-delay', state.slideshowDelay);
            if (state.playback.slideshowTimer) {
                stopSlideshow();
                startSlideshow();
            }
        };
    }

    if (settingTrackShuffleDuration) {
        settingTrackShuffleDuration.onchange = (e) => {
            state.trackShuffleDuration = parseInt(e.target.value);
            localStorage.setItem('disco-track-shuffle-duration', state.trackShuffleDuration);
        };
    }

    if (sortBy) sortBy.value = state.filters.sort;
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
        const details = document.querySelectorAll('.sidebar details');
        details.forEach(det => {
            const id = det.id;
            if (!id) return;

            // Restore
            if (state.sidebarState[id] !== undefined) {
                det.open = state.sidebarState[id];
            }

            // Listen
            det.addEventListener('toggle', () => {
                state.sidebarState[id] = det.open;
                localStorage.setItem('disco-sidebar-state', JSON.stringify(state.sidebarState));
            });

            // Ctrl+click to toggle all
            const summary = det.querySelector('summary');
            if (summary) {
                summary.onclick = (e) => {
                    if (e.ctrlKey || e.metaKey) {
                        e.preventDefault();
                        const newState = !det.open;
                        document.querySelectorAll('.sidebar details').forEach(d => {
                            d.open = newState;
                            if (d.id) {
                                state.sidebarState[d.id] = newState;
                            }
                        });
                        localStorage.setItem('disco-sidebar-state', JSON.stringify(state.sidebarState));
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
                episodes_min: data.episodes_min !== undefined && data.episodes_min !== null ? data.episodes_min : 0,
                episodes_max: data.episodes_max !== undefined && data.episodes_max !== null ? data.episodes_max : 100,
                size_min: data.size_min !== undefined && data.size_min !== null ? data.size_min : 0,
                size_max: data.size_max !== undefined && data.size_max !== null ? data.size_max : (100 * 1024 * 1024),
                duration_min: data.duration_min !== undefined && data.duration_min !== null ? data.duration_min : 0,
                duration_max: data.duration_max !== undefined && data.duration_max !== null ? data.duration_max : 3600,
                episodes_percentiles: data.episodes_percentiles || [],
                size_percentiles: data.size_percentiles || [],
                duration_percentiles: data.duration_percentiles || []
            };

            // Recalculate slider positions if we have absolute filters
            const updateSliderPos = (type, filterKey, minSlider, maxSlider) => {
                const filter = state.filters[filterKey] && state.filters[filterKey].find(f => f.value === '@abs');
                if (filter && minSlider && state.filterBins) {
                    let minTotal = 0, maxTotal = 0;
                    if (type === 'episodes') { minTotal = state.filterBins.episodes_min; maxTotal = state.filterBins.episodes_max; }
                    else if (type === 'size') { minTotal = state.filterBins.size_min; maxTotal = state.filterBins.size_max; }
                    else if (type === 'duration') { minTotal = state.filterBins.duration_min; maxTotal = state.filterBins.duration_max; }

                    if (maxTotal > minTotal) {
                        const minP = Math.max(0, Math.min(100, ((filter.min - minTotal) / (maxTotal - minTotal)) * 100));
                        const maxP = Math.max(0, Math.min(100, ((filter.max - minTotal) / (maxTotal - minTotal)) * 100));
                        minSlider.value = Math.round(minP);
                        maxSlider.value = Math.round(maxP);
                    }
                }
            };

            updateSliderPos('episodes', 'episodes', episodesMinSlider, episodesMaxSlider);
            updateSliderPos('size', 'sizes', sizeMinSlider, sizeMaxSlider);
            updateSliderPos('duration', 'durations', durationMinSlider, durationMaxSlider);

            renderMediaTypeList();
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
            { id: 'image', label: 'Image', icon: '🖼️' },
            { id: 'app', label: 'App', icon: '📱' }
        ];

        const newHtml = types.map(t => `
            <button class="category-btn ${state.filters.types.includes(t.id) ? 'active' : ''}" data-type="${t.id}">
                ${t.icon} ${t.label}
            </button>
        `).join('');

        container.innerHTML = newHtml;

        container.querySelectorAll('button').forEach(btn => {
            btn.onclick = () => {
                const type = btn.dataset.type;
                if (state.filters.types.includes(type)) {
                    state.filters.types = [];
                } else {
                    state.filters.types = [type];
                }
                localStorage.setItem('disco-types', JSON.stringify(state.filters.types));
                updateNavActiveStates();
                performSearch();
            };
        });
    }

    function renderFilterBins() {
        // Sliders are static in index.html, no need to render bins here.
    }

    function resetSidebar() {
        const details = document.querySelectorAll('.sidebar details');
        state.sidebarState = {};
        state.filters.categories = [];
        state.filters.genre = '';
        state.filters.ratings = [];
        state.filters.playlist = null;
        state.filters.sizes = [];
        state.filters.durations = [];
        state.filters.episodes = [];
        state.filters.types = [];
        state.filters.search = '';
        state.filters.unplayed = false;
        state.filters.unfinished = false;
        state.filters.completed = false;
        if (searchInput) searchInput.value = '';

        details.forEach(det => {
            const id = det.id;
            if (!id) return;
            det.open = false;
            state.sidebarState[id] = false;
        });

        localStorage.setItem('disco-sidebar-state', JSON.stringify(state.sidebarState));
        localStorage.setItem('disco-filter-categories', '[]');
        localStorage.setItem('disco-filter-ratings', '[]');
        localStorage.setItem('disco-filter-sizes', '[]');
        localStorage.setItem('disco-filter-durations', '[]');
        localStorage.setItem('disco-filter-episodes', '[]');
        localStorage.setItem('disco-types', '[]');
        updateNavActiveStates();
    }

    // --- Modal Management ---
    function openModal(id) {
        document.getElementById(id).classList.remove('hidden');
    }

    function closeModal(id) {
        document.getElementById(id).classList.add('hidden');
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
        if (state.filters.search) params.append('search', state.filters.search);
        state.filters.categories.forEach(c => params.append('category', c));
        if (state.filters.genre) params.append('genre', state.filters.genre);
        state.filters.ratings.forEach(r => params.append('rating', r));
        if (state.filters.unplayed) params.append('unplayed', 'true');
        if (state.filters.unfinished) params.append('unfinished', 'true');
        if (state.filters.completed) params.append('completed', 'true');
        if (state.filters.min_score) params.append('min_score', state.filters.min_score);
        if (state.filters.max_score) params.append('max_score', state.filters.max_score);

        state.filters.episodes.forEach(b => params.append('episodes', getBinQueryParam(b)));
        state.filters.sizes.forEach(b => params.append('size', getBinQueryParam(b)));
        state.filters.durations.forEach(b => params.append('duration', getBinQueryParam(b)));

        state.filters.types.forEach(t => params.append('type', t));
    }

    function syncUrl() {
        const params = new URLSearchParams();
        if (state.page === 'trash') {
            params.set('view', 'trash');
        } else if (state.page === 'history') {
            params.set('view', 'history');
        } else if (state.page === 'playlist' && state.filters.playlist) {
            params.set('view', 'playlist');
            params.set('title', state.filters.playlist);
        } else if (state.page === 'du') {
            params.set('view', 'du');
            if (state.duPath) params.set('path', state.duPath);
        } else if (state.page === 'curation') {
            params.set('view', 'curation');
        } else if (state.page === 'captions') {
            params.set('view', 'captions');
        } else {
            state.filters.categories.forEach(c => params.append('category', c));
            if (state.filters.genre) params.set('genre', state.filters.genre);
            state.filters.ratings.forEach(r => params.append('rating', r));
            if (state.filters.search) params.set('search', state.filters.search);
            if (state.filters.min_score) params.set('min_score', state.filters.min_score);
            if (state.filters.max_score) params.set('max_score', state.filters.max_score);
            if (state.filters.unplayed) params.set('unplayed', 'true');
            if (state.filters.unfinished) params.set('unfinished', 'true');
            if (state.filters.completed) params.set('completed', 'true');

            state.filters.episodes.forEach(b => params.append('episodes', getBinQueryParam(b)));
            state.filters.sizes.forEach(b => params.append('size', getBinQueryParam(b)));
            state.filters.durations.forEach(b => params.append('duration', getBinQueryParam(b)));
        }

        if (state.page !== 'du' && state.page !== 'trash' && state.page !== 'playlist') {
            state.filters.types.forEach(t => params.append('type', t));
        }

        if (state.currentPage > 1) {
            params.set('p', state.currentPage);
        }

        const paramString = params.toString();
        const newHash = paramString ? `#${paramString}` : '';

        if (window.location.hash !== newHash) {
            // Use pushState for DU navigation and page changes to support back button
            // Use replaceState for filter changes to avoid history spam
            const isDUPathChange = state.page === 'du' && !window.location.hash.includes('view=du');
            const isPageChange = !window.location.hash.includes(`view=${state.page}`);

            if (isPageChange || state.page === 'du') {
                window.history.pushState(state.filters, '', window.location.pathname + newHash);
            } else {
                window.history.replaceState(state.filters, '', window.location.pathname + newHash);
            }
        }
    }

    function readUrl(openSections = false) {
        // Support both hash and search params, preferring hash for the new system
        const hash = window.location.hash.substring(1);
        const params = hash ? new URLSearchParams(hash) : new URLSearchParams(window.location.search);
        const view = params.get('view');

        const pageParam = params.get('p');
        state.currentPage = pageParam ? parseInt(pageParam) : 1;

        if (view === 'trash') {
            state.page = 'trash';
            state.filters.categories = [];
            state.filters.ratings = [];
        } else if (view === 'history') {
            state.page = 'history';
            state.filters.categories = [];
            state.filters.ratings = [];
        } else if (view === 'playlist') {
            state.page = 'playlist';
            state.filters.playlist = params.get('title');
            state.filters.categories = [];
            state.filters.ratings = [];
        } else if (view === 'du') {
            state.page = 'du';
            state.duPath = params.get('path') || '';
            state.filters.categories = [];
            state.filters.ratings = [];
        } else if (view === 'curation') {
            state.page = 'curation';
            state.filters.categories = [];
            state.filters.ratings = [];
        } else if (view === 'captions') {
            state.page = 'captions';
            state.filters.categories = [];
            state.filters.ratings = [];
        } else {
            state.page = 'search';
            state.filters.types = params.getAll('type');
            if (state.filters.types.length === 0) {
                // Default fallback if not in URL
                state.filters.types = JSON.parse(localStorage.getItem('disco-types') || '[]');
            }
            state.filters.categories = params.getAll('category');
            state.filters.genre = params.get('genre') || '';
            state.filters.ratings = params.getAll('rating');
            state.filters.all = params.get('all') === 'true';
            state.filters.min_score = params.get('min_score') || '';

            state.filters.max_score = params.get('max_score') || '';
            state.filters.unplayed = params.get('unplayed') === 'true';
            state.filters.unfinished = params.get('unfinished') === 'true';
            state.filters.completed = params.get('completed') === 'true';

            state.filters.episodes = params.getAll('episodes').map(val => {
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
            });
            state.filters.sizes = params.getAll('size').map(val => {
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
            });
            state.filters.durations = params.getAll('duration').map(val => {
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
            });

            if (openSections) {
                if (state.filters.categories.length > 0) state.sidebarState['details-categories'] = true;
                if (state.filters.genre) state.sidebarState['details-browse'] = true;
                if (state.filters.ratings.length > 0) state.sidebarState['details-ratings'] = true;
                if (state.filters.episodes.length > 0) state.sidebarState['details-episodes'] = true;
                if (state.filters.sizes.length > 0) state.sidebarState['details-size'] = true;
                if (state.filters.durations.length > 0) state.sidebarState['details-duration'] = true;
            }

            // Restoration of complex filters from URL is tricky since we only have labels in bins
            // For now, we rely on state persistence in localStorage which is already happening
            // But we can try to parse them if we want to support sharing URLs

            if (searchInput) searchInput.value = state.filters.search;

            if (state.filters.genre && filterBrowseCol) {
                filterBrowseCol.value = 'genre';
                filterBrowseCol.onchange();
            } else if (state.filters.categories.length > 0 && filterBrowseCol) {
                filterBrowseCol.value = 'category';
                filterBrowseCol.onchange();
            } else if (filterBrowseCol) {
                filterBrowseCol.value = '';
                filterBrowseValContainer.classList.add('hidden');
            }
        }
    }

    const onUrlChange = () => {
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

    window.onpopstate = onUrlChange;
    window.onhashchange = onUrlChange;

    // --- API Calls ---
    async function fetchDatabases() {
        try {
            const resp = await fetchAPI('/api/databases');
            if (!resp.ok) throw new Error('Offline');
            const data = await resp.json();
            allDatabases = data.databases;
            state.trashcan = data.trashcan;
            state.readOnly = data.read_only;
            state.dev = data.dev;

            renderDbSettingsList(allDatabases);
            if (state.trashcan) {
                if (trashBtn) trashBtn.classList.remove('hidden');
            }
            if (state.dev) {
                setupAutoReload();
            }
            if (state.readOnly) {
                const newPlaylistBtn = document.getElementById('new-playlist-btn');
                if (newPlaylistBtn) newPlaylistBtn.classList.add('hidden');
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

        const inputVal = searchInput.value;
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
            el.onclick = () => {
                const path = el.dataset.path;
                const isDir = el.dataset.isDir === 'true';
                if (isDir) {
                    if (searchInput.value.startsWith('./')) {
                        const newName = el.dataset.name;
                        const lastSlash = searchInput.value.lastIndexOf('/');
                        const newPath = searchInput.value.substring(0, lastSlash + 1) + newName + '/';
                        searchInput.value = newPath;
                    } else {
                        const newPath = path.endsWith('/') ? path : path + '/';
                        searchInput.value = newPath;
                    }
                    searchInput.focus();
                    fetchSuggestions(searchInput.value);
                    performSearch();
                } else {
                    const item = state.lastSuggestions.find(s => s.path === path);
                    if (item) {
                        if (state.player === 'browser') {
                            openInPiP(item, true);
                        } else {
                            playMedia(item);
                        }
                        // Set searchbar to parent
                        const parts = path.split('/');
                        parts.pop();
                        const parent = parts.join('/') + '/';
                        searchInput.value = parent;
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
            zone.onclick = (e) => {
                // Ignore clicks on the delete button
                if (e.target.closest('.delete-playlist-btn')) return;

                const title = zone.dataset.title;
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
                e.dataTransfer.dropEffect = 'copy';
            });

            zone.addEventListener('dragleave', (e) => {
                // Only remove if we're actually leaving the zone
                if (!zone.contains(e.relatedTarget)) {
                    zone.classList.remove('drag-over');
                }
            });

            zone.addEventListener('drop', async (e) => {
                e.preventDefault();
                e.stopPropagation();

                zone.classList.remove('drag-over');

                const title = zone.dataset.title;
                const path = e.dataTransfer.getData('text/plain');
                console.log('Drop detected:', { title, path });

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
            btn.onclick = (e) => {
                e.stopPropagation();
                if (confirm('Delete this playlist?')) {
                    deletePlaylist(btn.dataset.title);
                }
            };
        });
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
            errorToast(err, 'Reorder failed');
        }
    }

    function filterPlaylistItems() {
        if (!state.playlistItems) return;

        let filtered = [...state.playlistItems];

        // Filter by type
        const types = state.filters.types || [];
        const hasVideo = types.includes('video');
        const hasAudio = types.includes('audio');
        const hasImage = types.includes('image');
        const hasText = types.includes('text');

        if (hasVideo || hasAudio || hasImage || hasText) {
            filtered = filtered.filter(item => {
                const mime = item.type || '';
                if (hasVideo && mime.startsWith('video')) return true;
                if (hasAudio && mime.startsWith('audio')) return true;
                if (hasImage && mime.startsWith('image')) return true;
                if (hasText && (mime.startsWith('text') || mime === '')) return true;
                return false;
            });
        }

        // Filter by search text (client-side)
        if (state.filters.search) {
            const query = state.filters.search.toLowerCase();
            filtered = filtered.filter(item => {
                const title = (item.title || '').toLowerCase();
                const path = (item.path || '').toLowerCase();
                return title.includes(query) || path.includes(query);
            });
        }

        currentMedia = filtered;
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
            errorToast(err, 'Failed to load playlist');
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
            btn.onclick = (e) => {
                const rating = btn.dataset.rating;
                if (state.page !== 'trash') state.page = 'search';

                const idx = state.filters.ratings.indexOf(rating);
                if (idx !== -1) {
                    state.filters.ratings.splice(idx, 1);
                } else {
                    state.filters.ratings.push(rating);
                }

                localStorage.setItem('disco-filter-ratings', JSON.stringify(state.filters.ratings));
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
                e.dataTransfer.dropEffect = 'copy';
            });

            btn.addEventListener('dragleave', (e) => {
                if (!btn.contains(e.relatedTarget)) {
                    btn.classList.remove('drag-over');
                }
            });

            btn.addEventListener('drop', async (e) => {
                e.preventDefault();
                e.stopPropagation();
                btn.classList.remove('drag-over');

                const rating = parseInt(btn.dataset.rating);
                const path = e.dataTransfer.getData('text/plain');

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
        state.page = 'du';
        state.duPath = path;
        syncUrl();

        const skeletonTimeout = setTimeout(() => {
            if (state.view === 'grid') showSkeletons();
        }, 150);

        try {
            const params = new URLSearchParams();
            params.append('path', path);
            appendFilterParams(params);

            const resp = await fetchAPI(`/api/du?${params.toString()}`);
            clearTimeout(skeletonTimeout);
            if (!resp.ok) throw new Error('Failed to fetch DU');
            state.duData = await resp.json();
            renderDU(state.duData);
        } catch (err) {
            clearTimeout(skeletonTimeout);
            console.error('DU fetch failed:', err);
            errorToast(err, 'Failed to load Disk Usage');
        }
    }

    function renderDU(data) {
        if (!data) data = [];

        resultsCount.textContent = `Disk Usage: ${state.duPath || 'Root'}`;
        resultsContainer.className = 'grid du-view';
        resultsContainer.innerHTML = '';

        // Add "Back" item if not at root
        if (state.duPath && state.duPath !== '/' && state.duPath !== '.') {
            const backCard = document.createElement('div');
            backCard.className = 'media-card du-card back-card';
            backCard.onclick = () => {
                let p = state.duPath;
                if (p.endsWith('/') && p.length > 1) p = p.slice(0, -1);
                const lastSlash = p.lastIndexOf('/');
                if (lastSlash === -1) {
                    fetchDU('');
                } else {
                    let parent = p.substring(0, lastSlash + 1);
                    fetchDU(parent);
                }
            };
            backCard.innerHTML = `
                <div class="media-thumb" style="display: flex; align-items: center; justify-content: center; font-size: 3rem; background: var(--sidebar-bg);">
                    🔙
                </div>
                <div class="media-info">
                    <div class="media-title">Go Back</div>
                    <div class="media-meta">To parent directory</div>
                </div>
            `;
            resultsContainer.appendChild(backCard);
        }

        const maxSize = Math.max(...data.map(d => d.total_size || 0));

        data.forEach(item => {
            const isFile = item.count === 0 && item.files && item.files.length === 1 && item.files[0].path === item.path;
            const card = document.createElement('div');

            if (isFile) {
                const mediaItem = item.files[0];
                card.className = 'media-card';
                card.dataset.path = mediaItem.path;
                card.onclick = () => playMedia(mediaItem);

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
            } else {
                card.className = 'media-card du-card';
                card.onclick = () => fetchDU(item.path + (item.path.endsWith('/') ? '' : '/'));

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
                            <span>${size}</span>
                            <span>${count} files</span>
                            <span>${duration}</span>
                        </div>
                    </div>
                `;
            }
            resultsContainer.appendChild(card);
        });

        paginationContainer.classList.add('hidden');
        updateNavActiveStates();
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
                params.append('all', 'true');
            } else {
                params.append('limit', state.filters.limit);
            }

            if (state.page === 'trash') {
                params.append('trash', 'true');
            } else if (state.page === 'history') {
                params.append('watched', 'true');
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
            errorToast(err, 'Failed to load Episodes');
            resultsContainer.innerHTML = `<div class="error">Failed to load episodes.</div>`;
        }
    }

    function renderEpisodes(data) {
        if (!data) data = [];

        let filtered = data.map(group => {
            const files = group.files || [];
            const filteredFiles = files.filter(f => {
                // Filter by types
                const selectedTypes = state.filters.types || [];
                if (selectedTypes.length > 0) {
                    const type = (f.type || '').split('/')[0];
                    let match = selectedTypes.includes(type);
                    if (!match && selectedTypes.includes('audio') && f.type === 'audiobook') match = true;
                    if (!match) return false;
                }

                // Filter by search
                if (state.filters.search) {
                    const query = state.filters.search.toLowerCase();
                    const path = (f.path || '').toLowerCase();
                    const title = (f.title || '').toLowerCase();
                    if (!path.includes(query) && !title.includes(query)) return false;
                }

                // Client-side progress filtering
                if (state.filters.unplayed) {
                    if (getPlayCount(f) > 0 || (f.playhead || 0) > 0) return false;
                } else if (state.filters.unfinished) {
                    if (getPlayCount(f) > 0 || (f.playhead || 0) === 0) return false;
                } else if (state.filters.completed) {
                    if (getPlayCount(f) === 0) return false;
                } else if (state.page === 'history') {
                    if ((f.time_last_played || 0) === 0) return false;
                }

                return true;
            });

            return { ...group, files: filteredFiles, count: filteredFiles.length };
        }).filter(group => group.count > 0);

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
                card.onclick = () => playMedia(item);

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
    }

    async function fetchCuration() {
        state.page = 'curation';
        syncUrl();

        document.getElementById('toolbar').classList.add('hidden');
        document.querySelector('.search-container').classList.add('hidden');

        // Show loading initially
        resultsContainer.innerHTML = '<div class="loading-container" style="text-align: center; padding: 3rem;"><div class="spinner"></div><h3>Loading Categorization...</h3></div>';

        try {
            const resp = await fetchAPI('/api/categorize/keywords');
            if (!resp.ok) throw new Error('Failed to fetch keywords');
            const data = await resp.json();
            renderCuration(data);
        } catch (err) {
            console.error('Curation fetch failed:', err);
            errorToast(err, 'Failed to load Curation Tool');
            resultsContainer.innerHTML = '<div class="error">Failed to load categorization tool.</div>';
        }
    }

    function renderCuration(keywordsData) {
        if (!keywordsData) keywordsData = [];

        resultsCount.textContent = ``;
        resultsContainer.className = 'curation-view';
        resultsContainer.innerHTML = '';

        const headerEl = document.createElement('div');
        headerEl.className = 'curation-header';
        headerEl.innerHTML = `
            <div style="display: flex; align-items: center; gap: 1rem; margin-bottom: 1rem;">
                <button id="curation-back-btn" class="category-btn">← Back</button>
                <h2 style="margin: 0;">Categorization</h2>
            </div>
            <p>Manage categories and keywords. Drag keywords from the suggestion pool to a category, or add them manually.</p>
            <div style="display: flex; gap: 1rem; margin: 1.5rem 0;">
                <button id="run-auto-categorize" class="category-btn" style="background: var(--accent-color); color: white;">Run Categorization Now</button>
                <button id="add-default-cats" class="category-btn">Add Default Categories</button>
            </div>
        `;
        resultsContainer.appendChild(headerEl);

        const container = document.createElement('div');
        container.className = 'curation-container';
        container.style.display = 'flex';
        container.style.gap = '2rem';
        container.style.height = 'calc(100vh - 250px)'; // Approx height

        // --- Left Column: Categories ---
        const categoriesCol = document.createElement('div');
        categoriesCol.className = 'curation-col';
        categoriesCol.style.flex = '1';
        categoriesCol.style.overflowY = 'auto';
        categoriesCol.style.borderRight = '1px solid var(--border-color)';
        categoriesCol.style.paddingRight = '1rem';

        categoriesCol.innerHTML = `<h3>Categories</h3>`;
        const categoriesList = document.createElement('div');
        categoriesList.className = 'curation-cat-list';
        categoriesList.style.display = 'flex';
        categoriesList.style.flexDirection = 'column';
        categoriesList.style.gap = '1rem';

        // Render existing categories
        keywordsData.forEach(cat => {
            const card = document.createElement('div');
            card.className = 'curation-cat-card';
            card.dataset.category = cat.category;
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
                const keyword = e.dataTransfer.getData('text/plain');
                if (keyword) {
                    await addKeyword(cat.category, keyword);
                }
            });

            // Delete Category
            card.querySelector('.delete-cat-btn').onclick = async () => {
                if (confirm(`Delete category "${cat.category}" and all its keywords?`)) {
                    await deleteCategory(cat.category);
                }
            };

            // Add Keyword manually
            card.querySelector('.add-kw-btn').onclick = async () => {
                const kw = prompt(`Add keyword to "${cat.category}":`);
                if (kw) {
                    await addKeyword(cat.category, kw);
                }
            };

            // Remove Keyword
            card.querySelectorAll('.remove-kw').forEach(btn => {
                btn.onclick = async (e) => {
                    e.stopPropagation(); // prevent drag start if any
                    const tag = e.target.closest('.curation-tag');
                    const kw = tag.dataset.keyword;
                    await deleteKeyword(cat.category, kw);
                };
            });

            categoriesList.appendChild(card);
        });

        // Add New Category Button at bottom of list
        const newCatBtn = document.createElement('button');
        newCatBtn.className = 'category-btn';
        newCatBtn.style.marginTop = '1rem';
        newCatBtn.style.width = '100%';
        newCatBtn.style.border = '1px dashed var(--border-color)';
        newCatBtn.textContent = '+ New Category';
        newCatBtn.onclick = async () => {
            const name = prompt('New Category Name:');
            if (name) {
                // To create a category, we need at least one keyword.
                // Or we can just refresh, but the backend only stores keywords.
                // So we'll prompt for a keyword too or add a placeholder?
                // Let's prompt for keyword immediately.
                const kw = prompt(`Add first keyword for "${name}":`);
                if (kw) {
                    await addKeyword(name, kw);
                }
            }
        };
        categoriesList.appendChild(newCatBtn);

        categoriesCol.appendChild(categoriesList);
        container.appendChild(categoriesCol);

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

        findBtn.onclick = async () => {
            findBtn.disabled = true;
            findBtn.textContent = 'Analyzing...';
            suggestionsArea.innerHTML = '<div class="spinner" style="width: 24px; height: 24px; margin: 1rem auto;"></div>';

            try {
                const resp = await fetchAPI('/api/categorize/suggest');
                if (!resp.ok) throw new Error('Failed');
                const suggestions = await resp.json();
                renderSuggestionsArea(suggestions, suggestionsArea);
            } catch (err) {
                console.error(err);
                suggestionsArea.innerHTML = '<p>Failed to load suggestions.</p>';
            } finally {
                findBtn.disabled = false;
                findBtn.textContent = 'Find Potential Keywords';
            }
        };

        container.appendChild(suggestionsCol);
        resultsContainer.appendChild(container);

        // Header Actions
        const backBtn = headerEl.querySelector('#curation-back-btn');
        if (backBtn) {
            backBtn.onclick = () => {
                state.page = 'search';
                state.filters.category = '';
                updateNavActiveStates();
                performSearch();
            };
        }

        const btnRun = headerEl.querySelector('#run-auto-categorize');
        if (btnRun) {
            btnRun.onclick = async () => {
                if (state.readOnly) return showToast('Read-only mode');
                btnRun.disabled = true;
                btnRun.textContent = 'Running...';
                try {
                    const resp = await fetchAPI('/api/categorize/apply', { method: 'POST' });
                    if (!resp.ok) throw new Error('Apply failed');
                    const data = await resp.json();
                    showToast(`Successfully categorized ${data.count} files!`, '🏷️');
                    fetchCategories();
                    // Don't refresh curation page necessarily, user might want to keep editing
                } catch (err) {
                    console.error('Apply failed:', err);
                    errorToast(err, 'Failed to run categorization');
                } finally {
                    btnRun.disabled = false;
                    btnRun.textContent = 'Run Categorization Now';
                }
            };
        }

        const btnDefaults = headerEl.querySelector('#add-default-cats');
        if (btnDefaults) {
            btnDefaults.onclick = async () => {
                if (confirm('Add default categories and keywords? (Existing ones will be kept)')) {
                    try {
                        const resp = await fetchAPI('/api/categorize/defaults', { method: 'POST' });
                        if (!resp.ok) throw new Error('Failed');
                        showToast('Default categories added');
                        fetchCuration(); // Refresh
                    } catch (err) {
                        console.error(err);
                        errorToast(err, 'Failed to add defaults');
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
            <div class="tags-cloud">
                ${suggestions.map(tag => `
                    <span class="curation-tag suggestion-tag" draggable="true" data-word="${tag.word}" title="${tag.count} occurrences">
                        ${tag.word} <small>${tag.count}</small>
                    </span>
                `).join('')}
            </div>
        `;

        container.querySelectorAll('.suggestion-tag').forEach(tag => {
            tag.addEventListener('dragstart', (e) => {
                e.dataTransfer.setData('text/plain', tag.dataset.word);
                tag.style.opacity = '0.5';
            });
            tag.addEventListener('dragend', (e) => {
                tag.style.opacity = '1';
            });
            // Click also prompts for category (legacy behavior, still useful)
            tag.onclick = async () => {
                const keyword = tag.dataset.word;
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
            fetchCuration(); // Refresh UI
        } catch (err) {
            console.error(err);
            errorToast(err, 'Failed to save keyword');
        }
    }

    async function deleteCategory(category) {
        try {
            const resp = await fetchAPI(`/api/categorize/category?category=${encodeURIComponent(category)}`, { method: 'DELETE' });
            if (!resp.ok) throw new Error('Failed');
            showToast(`Deleted category "${category}"`);
            fetchCuration();
        } catch (err) {
            console.error(err);
            errorToast(err, 'Failed to delete category');
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
            fetchCuration();
        } catch (err) {
            console.error(err);
            errorToast(err, 'Failed to delete keyword');
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
        state.filters.search = searchInput.value;
        state.filters.sort = sortBy.value;
        state.filters.limit = parseInt(limitInput.value) || 100;
        state.filters.all = limitAll ? limitAll.checked : false;

        syncUrl();

        if (state.page === 'du') {
            fetchDU(state.duPath || '');
            return;
        }

        if (state.view === 'group') {
            fetchEpisodes();
            return;
        }

        if (trashBtn && state.page !== 'trash') trashBtn.classList.remove('active');

        if (searchAbortController) {
            searchAbortController.abort();
        }
        searchAbortController = new AbortController();

        localStorage.setItem('disco-limit', state.filters.limit);
        localStorage.setItem('disco-limit-all', state.filters.all);

        if (limitInput) limitInput.disabled = state.filters.all;

        const skeletonTimeout = setTimeout(() => {
            if (state.page === 'search' || state.page === 'trash' || state.page === 'history' || state.page === 'playlist' || state.page === 'captions') {
                if (state.view === 'grid') showSkeletons();
            }
        }, 150);

        try {
            const params = new URLSearchParams();

            appendFilterParams(params);
            params.append('sort', state.filters.sort);

            if (state.filters.reverse) params.append('reverse', 'true');

            if (state.filters.all) {
                params.append('all', 'true');
            } else {
                params.append('limit', state.filters.limit);
                params.append('offset', (state.currentPage - 1) * state.filters.limit);
            }

            if (state.page === 'captions' || state.filters.captions) params.append('captions', 'true');

            if (state.page === 'trash') {
                params.append('trash', 'true');
            } else if (state.page === 'history') {
                params.append('watched', 'true');
            }

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

            // Client-side DB filtering
            currentMedia = data.filter(item => !state.filters.excludedDbs.includes(item.db));

            // Client-side progress filtering (in case server is slightly behind or for local counts)
            if (state.filters.unplayed) {
                currentMedia = currentMedia.filter(item => getPlayCount(item) === 0);
            } else if (state.filters.unfinished) {
                currentMedia = currentMedia.filter(item => getPlayCount(item) === 0 && (item.playhead || 0) > 0);
            } else if (state.filters.completed) {
                currentMedia = currentMedia.filter(item => getPlayCount(item) > 0);
            } else if (state.page === 'history') {
                currentMedia = currentMedia.filter(item => (item.time_last_played || 0) > 0);
            }

            // Update total count after client-side filtering
            if (state.filters.unplayed || state.filters.unfinished || state.filters.completed || state.page === 'history' || state.filters.excludedDbs.length > 0) {
                state.totalCount = currentMedia.length;
            }

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
            fetchFilterBins(params);
        } catch (err) {
            clearTimeout(skeletonTimeout);
            if (err.name === 'AbortError') return;
            console.error('Search failed:', err);
            resultsContainer.innerHTML = `<div class="error">Search failed: ${err.message}</div>`;
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
            errorToast(err, 'Failed to empty bin');
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
            errorToast(err, 'Failed to delete');
        }
    }

    async function deleteMedia(path, restore = false) {
        const itemEl = document.querySelector(`[data-path="${CSS.escape(path)}"]`);
        const content = document.querySelector('.content');
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
            errorToast(err, 'Action failed');
            if (itemEl) itemEl.classList.remove('fade-out');
        } finally {
            if (content) content.style.overflow = '';
            if (main) main.style.overflow = '';
        }
    }

    async function playMedia(item) {
        if (state.playback.skipTimeout) {
            clearTimeout(state.playback.skipTimeout);
            state.playback.skipTimeout = null;
        }

        if (state.player === 'browser') {
            openInPiP(item, true); // True means this was an explicit user request / new session
            return;
        }

        const prevItem = state.playback.item;
        const wasPlayed = state.playback.hasMarkedComplete || (prevItem && getPlayCount(prevItem) > 0);

        state.playback.item = item;
        state.playback.startTime = Date.now();
        state.playback.hasMarkedComplete = false;

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
                    errorToast(err, 'Playback failed');
                }
            }
        } catch (err) {
            console.error('Playback failed', err);
            errorToast(err, 'Playback failed');
        }
    }

    async function updateProgress(item, playhead, duration, isComplete = false) {
        if (state.playback.isSurfing) return state.playback.pendingUpdate;

        const media = pipViewer.querySelector('video, audio');
        if (!isComplete && media && (media.seeking || media.readyState < 3)) {
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
                const progress = JSON.parse(localStorage.getItem('disco-progress') || '{}');
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
                        const counts = JSON.parse(localStorage.getItem('disco-play-counts') || '{}');
                        counts[item.path] = (counts[item.path] || 0) + 1;
                        localStorage.setItem('disco-play-counts', JSON.stringify(counts));
                    }
                } else {
                    progress[item.path] = {
                        pos: Math.floor(playhead),
                        last: now
                    };
                }
                localStorage.setItem('disco-progress', JSON.stringify(progress));
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

    function setPlaybackRate(rate) {
        state.playbackRate = rate;
        localStorage.setItem('disco-playback-rate', rate);
        const speedBtn = document.getElementById('pip-speed');
        if (speedBtn) speedBtn.textContent = `${rate}x`;

        const media = pipViewer.querySelector('video, audio');
        if (media) {
            media.playbackRate = rate;
        }
    }

    function playSibling(offset, isUser = false, isDelete = false) {
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

        if (nextIndex >= 0 && nextIndex < currentMedia.length) {
            if (state.player === 'browser') {
                openInPiP(currentMedia[nextIndex], isNewSession);
            } else {
                playMedia(currentMedia[nextIndex]);
            }
        } else if (nextIndex >= currentMedia.length && !state.filters.all && state.page === 'search') {
            // End of current page, fetch next
            state.currentPage++;
            performSearch().then(() => {
                if (currentMedia.length > 0) {
                    if (state.player === 'browser') {
                        openInPiP(currentMedia[0], isNewSession);
                    } else {
                        playMedia(currentMedia[0]);
                    }
                }
            });
        } else if (nextIndex < 0 && state.currentPage > 1 && !state.filters.all && state.page === 'search') {
            // Beginning of current page, fetch previous
            state.currentPage--;
            performSearch().then(() => {
                if (currentMedia.length > 0) {
                    if (state.player === 'browser') {
                        openInPiP(currentMedia[currentMedia.length - 1], isNewSession);
                    } else {
                        playMedia(currentMedia[currentMedia.length - 1]);
                    }
                }
            });
        }
    }

    async function rateMedia(item, score) {
        try {
            await fetchAPI('/api/rate', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ path: item.path, score: score })
            });
            showToast(`Rated: ${'⭐'.repeat(score)}`);
            fetchRatings();
        } catch (err) {
            console.error('Failed to rate media:', err);
        }
    }

    async function markMediaPlayed(item) {
        if (state.readOnly) {
            // Local update for read-only mode
            const progress = JSON.parse(localStorage.getItem('disco-progress') || '{}');
            progress[item.path] = { pos: 0, last: Date.now() };
            localStorage.setItem('disco-progress', JSON.stringify(progress));

            const counts = JSON.parse(localStorage.getItem('disco-play-counts') || '{}');
            counts[item.path] = (counts[item.path] || 0) + 1;
            localStorage.setItem('disco-play-counts', JSON.stringify(counts));

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
                errorToast(err, 'Action failed');
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
            const counts = JSON.parse(localStorage.getItem('disco-play-counts') || '{}');
            counts[item.path] = 0;
            localStorage.setItem('disco-play-counts', JSON.stringify(counts));

            const progress = JSON.parse(localStorage.getItem('disco-progress') || '{}');
            delete progress[item.path];
            localStorage.setItem('disco-progress', JSON.stringify(progress));

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
                errorToast(err, 'Action failed');
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
        if (!media || !media.textTracks) return;

        const tracks = Array.from(media.textTracks).filter(t => t.kind === 'subtitles');
        if (tracks.length === 0) return;

        // Find current active track index
        let activeIndex = tracks.findIndex(t => t.mode === 'showing');

        // Disable all
        tracks.forEach(t => t.mode = 'disabled');

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

    async function playRSVP(item) {
        const originalPlayer = state.player;
        state.player = 'browser';
        const rsvpItem = { ...item, rsvp: true, type: 'video/webm' };
        await openInPiP(rsvpItem, true);
        state.player = originalPlayer;
    }

    async function handleMediaError(item) {
        if (pipLoading) pipLoading.classList.add('hidden');
        // Only handle error for the currently active item.
        // If state.playback.item is null, the player was likely closed manually.
        if (!state.playback.item || state.playback.item.path !== item.path) return;

        // Clear handlers to prevent other events (like onended) from firing after error
        const media = pipViewer.querySelector('video, audio, img');
        if (media) {
            media.onerror = null;
            media.onended = null;
            media.onload = null;
        }

        const basename = item.path.split('/').pop();
        let is404 = false;
        try {
            const resp = await fetchAPI(`/api/raw?path=${encodeURIComponent(item.path)}`, { method: 'HEAD' });
            is404 = resp.status === 404;
        } catch (err) {
            console.error('Failed to check media status:', err);
        }

        const msg = is404 ? `File not found: ${basename}` : `Unplayable: ${basename}`;
        const emoji = is404 ? '🗑️' : '⚠️';

        if (state.page === 'trash') {
            showToast(msg, '⚠️');
        } else {
            showToast(msg, emoji);
            // Remove from current view if applicable
            currentMedia = currentMedia.filter(m => m.path !== item.path);
            renderResults();
        }

        // Auto-skip to next
        if (state.autoplay) {
            if (state.playback.skipTimeout) {
                clearTimeout(state.playback.skipTimeout);
            }
            state.playback.skipTimeout = setTimeout(() => {
                if (state.playback.skipTimeout) { // Check if it was cleared in the meantime
                    state.playback.skipTimeout = null;
                    playSibling(1);
                }
            }, 1200);
        } else {
            closePiP();
        }
    }

    async function openInPiP(item, isNewSession = false, isSurfing = false) {
        state.playback.isSurfing = isSurfing;

        if (state.playback.slideshowTimer) {
            clearTimeout(state.playback.slideshowTimer);
            state.playback.slideshowTimer = null;
        }
        if (state.playback.surfTimer) {
            clearTimeout(state.playback.surfTimer);
            state.playback.surfTimer = null;
        }

        const wasFullscreen = !!document.fullscreenElement;

        if (isNewSession) {
            // New explicit request: reset state.imageAutoplay to user preference
            state.imageAutoplay = localStorage.getItem('disco-image-autoplay') === 'true';
        }

        const type = item.type || "";
        // Handle Documents separately
        if (type === 'text' || type.includes('pdf') || type.includes('epub') || type.includes('mobi')) {
            openInDocumentViewer(item);
            return;
        }

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
        lyricsDisplay.classList.add('hidden');
        lyricsDisplay.textContent = '';
        secondarySubtitle.classList.add('hidden');
        secondarySubtitle.textContent = '';

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
            if (type.includes('image')) {
                speedBtn.classList.add('hidden');
                if (pipSpeedMenu) pipSpeedMenu.classList.add('hidden');
            } else {
                speedBtn.classList.remove('hidden');
            }
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
        let url = item.rsvp ? `/api/rsvp?path=${encodeURIComponent(path)}` : `/api/raw?path=${encodeURIComponent(path)}`;

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
                localStorage.setItem('disco-muted', el.muted);
            };

            // Loop short videos/GIFs (under 8s)
            if (item.duration > 0 && item.duration < 8) {
                el.loop = true;
            }

            el.onerror = () => {
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
                    handleMediaError(item);
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
                    const hls = new Hls({
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

                // Handle secondary subtitles
                const tracks = Array.from(el.textTracks);
                const secondary = tracks.find(t => t.mode === 'hidden');
                if (secondary && secondary.activeCues && secondary.activeCues.length > 0) {
                    const cue = Array.from(secondary.activeCues).pop();
                    if (cue) {
                        secondarySubtitle.classList.remove('hidden');
                        secondarySubtitle.textContent = cue.text;
                    }
                } else {
                    secondarySubtitle.classList.add('hidden');
                }
            };

            el.onended = async () => {
                await updateProgress(item, el.duration, el.duration, true);
                handlePostPlayback(item);
            };

            const addTrack = (trackUrl, label, index) => {
                const track = document.createElement('track');
                track.kind = 'subtitles';
                track.label = label;
                track.srclang = state.language || 'en';
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
                    const isExt = codec.startsWith('.');
                    const label = isExt ? `External (${codec})` : (codec || `Embedded #${index + 1}`);
                    const trackUrl = isExt ?
                        `/api/subtitles?path=${encodeURIComponent(item.path.substring(0, item.path.lastIndexOf('.')) + codec)}` :
                        `/api/subtitles?path=${encodeURIComponent(path)}&index=${index}`;

                    addTrack(trackUrl, label, index);
                });
            }

            // 2. Always check for external subtitle file (sibling with same name)
            if (!type.includes('image')) {
                addTrack(`/api/subtitles?path=${encodeURIComponent(path)}`, 'External/Auto', 'auto');
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
                localStorage.setItem('disco-muted', el.muted);
            };

            el.onerror = () => handleMediaError(item);

            seekToProgress(el, localPos);

            el.ontimeupdate = () => {
                const isComplete = (el.duration > 90) && (el.duration - el.currentTime < 90) && (el.currentTime / el.duration > 0.95);
                updateProgress(item, el.currentTime, el.duration, isComplete);

                // Handle lyrics/subtitles
                const tracks = Array.from(el.textTracks);
                const primary = tracks.find(t => t.mode === 'showing') || tracks[0];
                const secondary = tracks.find(t => t.mode === 'hidden');

                if (primary && primary.activeCues && primary.activeCues.length > 0) {
                    const cue = Array.from(primary.activeCues).pop();
                    if (cue) {
                        lyricsDisplay.classList.remove('hidden');
                        lyricsDisplay.textContent = cue.text;
                    }
                } else {
                    lyricsDisplay.classList.add('hidden');
                }

                if (secondary && secondary.activeCues && secondary.activeCues.length > 0) {
                    const cue = Array.from(secondary.activeCues).pop();
                    if (cue) {
                        secondarySubtitle.classList.remove('hidden');
                        secondarySubtitle.textContent = cue.text;
                    }
                } else {
                    secondarySubtitle.classList.add('hidden');
                }
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
                track.srclang = state.language || 'en';
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
            el.src = url;
            el.onerror = () => handleMediaError(item);
            el.onload = () => {
                if (state.imageAutoplay && !state.playback.isSurfing) {
                    startSlideshow();
                }
            };
            el.ondblclick = () => toggleFullscreen(pipViewer, pipViewer);

            // Zoom/Pan logic for fullscreen
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
        } else {
            showToast('Unsupported media format');
            return;
        }

        pipViewer.appendChild(el);

        // Maintain/Switch fullscreen state if active
        if (wasFullscreen) {
            const preferred = (type.includes('video')) ? el : pipViewer;
            preferred.requestFullscreen().catch(e => console.error("Fullscreen switch failed:", e));
        }
    }

    function openInDocumentViewer(item) {
        const modal = document.getElementById('document-modal');
        const title = document.getElementById('document-title');
        const container = document.getElementById('document-container');
        const epubViewer = document.getElementById('epub-viewer');
        const pdfCanvas = document.getElementById('pdf-canvas');
        const pageInfo = document.getElementById('doc-page-info');
        const zoomInfo = document.getElementById('doc-zoom-info');

        title.textContent = truncateString(item.path.split('/').pop());
        title.title = item.path;
        epubViewer.innerHTML = '';
        epubViewer.tabIndex = 0; // Make focusable for keyboard shortcuts
        pdfCanvas.classList.add('hidden');
        epubViewer.classList.add('hidden');

        const url = `/api/raw?path=${encodeURIComponent(item.path)}`;
        const type = item.type || '';

        // Helper to show/hide EPUB-only controls
        const toggleEpubControls = (show) => {
            const controls = ['doc-prev', 'doc-next', 'doc-zoom-in', 'doc-zoom-out', 'doc-zoom-info'];
            controls.forEach(id => {
                const el = document.getElementById(id);
                if (el) {
                    if (show) el.classList.remove('hidden');
                    else el.classList.add('hidden');
                }
            });
        };

        const fsBtn = document.getElementById('doc-fullscreen');
        const modalContent = modal.querySelector('.modal-content');
        if (fsBtn) {
            fsBtn.classList.remove('hidden');
            fsBtn.onclick = () => toggleFullscreen(modalContent);
        }

        const rsvpBtn = document.getElementById('doc-rsvp');
        if (rsvpBtn) {
            rsvpBtn.onclick = () => {
                closeModal('document-modal');
                playRSVP(item);
            };
        }

        if (type.includes('epub')) {
            toggleEpubControls(true);
            epubViewer.classList.remove('hidden');
            const book = ePub(url);
            const rendition = book.renderTo("epub-viewer", {
                width: "100%",
                height: "100%",
                flow: "scrolled",
                manager: "continuous"
            });
            rendition.display();

            document.getElementById('doc-prev').onclick = () => rendition.prev();
            document.getElementById('doc-next').onclick = () => rendition.next();

            let zoom = 100;
            const zoomInfo = document.getElementById('doc-zoom-info');
            document.getElementById('doc-zoom-in').onclick = () => {
                zoom += 10;
                epubViewer.style.fontSize = `${zoom}%`;
                zoomInfo.textContent = `${zoom}%`;
            };
            document.getElementById('doc-zoom-out').onclick = () => {
                zoom = Math.max(50, zoom - 10);
                epubViewer.style.fontSize = `${zoom}%`;
                zoomInfo.textContent = `${zoom}%`;
            };
            pageInfo.textContent = "EPUB Mode";
        } else if (type.includes('pdf')) {
            toggleEpubControls(false);
            // Browsers have great built-in PDF viewers, let's use iframe but in the large modal
            const iframe = document.createElement('iframe');
            iframe.src = url;
            iframe.style.width = '100%';
            iframe.style.height = '100%';
            iframe.style.border = 'none';
            epubViewer.classList.remove('hidden');
            epubViewer.appendChild(iframe);
            pageInfo.textContent = "PDF Mode";
            document.getElementById('doc-prev').onclick = null;
            document.getElementById('doc-next').onclick = null;
        } else {
            toggleEpubControls(false);
            // Fallback for other text
            const iframe = document.createElement('iframe');
            iframe.src = url;
            iframe.style.width = '100%';
            iframe.style.height = '100%';
            iframe.style.border = 'none';
            epubViewer.classList.remove('hidden');
            epubViewer.appendChild(iframe);
            pageInfo.textContent = "Text Mode";
        }

        epubViewer.ondblclick = (e) => {
            e.stopPropagation();
            toggleFullscreen(modalContent);
        };
        container.ondblclick = () => toggleFullscreen(modalContent);

        openModal('document-modal');
        epubViewer.focus();
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

    function toggleFullscreen(defaultEl, preferredEl) {
        const el = preferredEl || defaultEl;
        if (!el) return;

        if (document.fullscreenElement) {
            // If already fullscreen on a different element, switch to preferred
            if (document.fullscreenElement !== el) {
                el.requestFullscreen().catch(err => {
                    console.error(`Error attempting to switch full-screen mode: ${err.message}`);
                });
            } else {
                document.exitFullscreen();
            }
        } else {
            el.requestFullscreen().catch(err => {
                console.error(`Error attempting to enable full-screen mode: ${err.message}`);
            });
        }
    }

    async function closePiP() {
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
            media.pause();
            media.src = "";
        }
        pipViewer.innerHTML = '';
        lyricsDisplay.classList.add('hidden');
        lyricsDisplay.textContent = '';
        pipPlayer.classList.add('hidden');
        document.body.classList.remove('has-pip');

        // Reset mode to default preference
        state.playerMode = state.defaultView;
        state.playback.item = null;
    }

    function renderPagination() {
        if (state.filters.all || state.page === 'trash' || state.page === 'playlist' || state.page === 'history') {
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

        prevPageBtn.disabled = state.currentPage === 1;
        nextPageBtn.disabled = state.currentPage >= totalPages;
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
            const unit = state.totalCount === 1 ? 'result' : 'results';
            resultsCount.textContent = `${state.totalCount} recently played ${unit}`;
        } else if (state.page === 'playlist') {
            const unit = currentMedia.length === 1 ? 'result' : 'results';
            resultsCount.textContent = `${currentMedia.length} ${unit} in ${state.filters.playlist || 'playlist'}`;
        } else {
            const unit = state.totalCount === 1 ? 'result' : 'results';
            resultsCount.textContent = `${state.totalCount} ${unit}`;
        }

        if (currentMedia.length === 0) {
            resultsContainer.innerHTML = '<div class="no-results">No media found</div>';
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
            card.dataset.path = item.path;
            card.draggable = true;

            card.addEventListener('dragstart', (e) => {
                state.draggedItem = item;
                e.dataTransfer.effectAllowed = 'all';
                e.dataTransfer.setData('text/plain', item.path);
                card.classList.add('dragging');
                document.body.classList.add('is-dragging');
            });

            card.addEventListener('dragend', () => {
                card.classList.remove('dragging');
                document.body.classList.remove('is-dragging');
                state.draggedItem = null;
                clearAllDragOver();
            });

            card.onclick = (e) => {
                if (e.target.closest('.media-actions') || e.target.closest('.media-action-btn')) return;

                if (item.is_dir) {
                    searchInput.value = item.path.endsWith('/') ? item.path : item.path + '/';
                    performSearch();
                    return;
                }

                if (item.path.toLowerCase().endsWith('.zim')) {
                    window.open(`/api/zim/view?path=${encodeURIComponent(item.path)}`, '_blank');
                    return;
                }

                const isCaptionClick = e.target.closest('.caption-highlight');
                if (isCaptionClick && item.caption_time) {
                    playMedia(item).then(() => {
                        const media = pipViewer.querySelector('video, audio');
                        if (media) media.currentTime = item.caption_time;
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

            const progress = (item.duration && item.playhead) ? Math.round((item.playhead / item.duration) * 100) : 0;
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
                const isText = item.type && (item.type.includes('text') || item.type.includes('epub') || item.type.includes('pdf'));
                actionBtns = `
                    ${isText ? `<button class="media-action-btn rsvp" title="Play as RSVP Video">👁️</button>` : ''}
                    ${!state.readOnly ? `<button class="media-action-btn add-playlist" title="Add to Playlist">+</button>` : ''}
                    ${plays > 0 ?
                        `<button class="media-action-btn mark-unplayed" title="Mark as Unplayed">⭕</button>` :
                        `<button class="media-action-btn mark-played" title="Mark as Played">✅</button>`
                    }
                    ${!state.readOnly ? `<button class="media-action-btn delete" title="Move to Trash">🗑️</button>` : ''}
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
                        ${item.type === 'app' && item.artist ? `<span>v${item.artist}</span>` : ''}
                        ${item.type === 'app' && item.language ? `<span>SDK ${item.language}</span>` : ''}
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
                    e.dataTransfer.dropEffect = 'move';

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
            if (btnDelete) btnDelete.onclick = (e) => {
                e.stopPropagation();
                deleteMedia(item.path, false);
            };

            const btnRSVP = card.querySelector('.media-action-btn.rsvp');
            if (btnRSVP) btnRSVP.onclick = (e) => {
                e.stopPropagation();
                playRSVP(item);
            };

            const btnRestore = card.querySelector('.media-action-btn.restore');
            if (btnRestore) btnRestore.onclick = (e) => {
                e.stopPropagation();
                deleteMedia(item.path, true);
            };

            const btnDeletePermanent = card.querySelector('.media-action-btn.delete-permanent');
            if (btnDeletePermanent) btnDeletePermanent.onclick = (e) => {
                e.stopPropagation();
                permanentlyDeleteMedia(item.path);
            };

            const btnAddPlaylist = card.querySelector('.media-action-btn.add-playlist');
            if (btnAddPlaylist) btnAddPlaylist.onclick = (e) => {
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
            if (btnMarkPlayed) btnMarkPlayed.onclick = (e) => {
                e.stopPropagation();
                markMediaPlayed(item);
            };

            const btnMarkUnplayed = card.querySelector('.media-action-btn.mark-unplayed');
            if (btnMarkUnplayed) btnMarkUnplayed.onclick = (e) => {
                e.stopPropagation();
                markMediaUnplayed(item);
            };

            const btnRemovePlaylist = card.querySelector('.media-action-btn.remove-playlist');
            if (btnRemovePlaylist) btnRemovePlaylist.onclick = (e) => {
                e.stopPropagation();
                removeFromPlaylist(state.filters.playlist, item);
            };

            fragment.appendChild(card);
        });

        resultsContainer.innerHTML = '';
        resultsContainer.appendChild(fragment);
        renderPagination();

        // Reset min-height after content is loaded
        resultsContainer.style.minHeight = '';
    }

    function renderCaptionsList() {
        resultsContainer.className = 'captions-list-view';
        resultsContainer.innerHTML = '';

        const fragment = document.createDocumentFragment();

        currentMedia.forEach(item => {
            const row = document.createElement('div');
            row.className = 'caption-row';

            const basename = item.path.split('/').pop();
            const captionTime = item.caption_time || 0;
            const timeStr = formatDuration(captionTime);

            row.innerHTML = `
                <div class="caption-header">
                    <span class="caption-basename" title="${item.path}">${basename}</span>
                    <span class="caption-timestamp">${timeStr}</span>
                </div>
                <div class="caption-text">${item.caption_text || '(no text)'}</div>
            `;

            row.onclick = () => {
                playMedia(item).then(() => {
                    const media = pipViewer.querySelector('video, audio');
                    if (media) media.currentTime = captionTime;
                });
            };

            fragment.appendChild(row);
        });

        resultsContainer.appendChild(fragment);
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
            tr.dataset.path = item.path;
            tr.draggable = true;

            tr.addEventListener('dragstart', (e) => {
                state.draggedItem = item;
                e.dataTransfer.effectAllowed = 'all';
                e.dataTransfer.setData('text/plain', item.path);
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
                    e.dataTransfer.dropEffect = 'move';

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
                    <button class="table-action-btn restore-btn" title="Restore">↺</button>
                    <button class="table-action-btn delete-permanent-btn" title="Permanently Delete">🔥</button>
                `;
            } else if (isPlaylist) {
                actions = !state.readOnly ? `<button class="table-action-btn remove-btn" title="Remove from Playlist">&times;</button>` : '';
            } else {
                const plays = getPlayCount(item);
                actions = `
                    <div class="playlist-item-actions">
                        ${!state.readOnly ? `<button class="table-action-btn add-btn" title="Add to Playlist">+</button>` : ''}
                        ${plays > 0 ?
                        `<button class="table-action-btn mark-unplayed-btn" title="Unmark as Played">⭕</button>` :
                        `<button class="table-action-btn mark-played-btn" title="Mark as Played">✅</button>`
                    }
                        ${!state.readOnly ? `<button class="table-action-btn delete-btn" title="Move to Trash">🗑️</button>` : ''}
                    </div>
                `;
            }

            const progress = (item.duration && item.playhead) ? Math.round((item.playhead / item.duration) * 100) : 0;
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
                            ${item.type === 'app' ? `<span style="font-size: 0.7rem; color: var(--text-muted);">${item.artist ? `v${item.artist}` : ''} ${item.language ? `(SDK ${item.language})` : ''}</span>` : ''}
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
            if (btnMarkPlayed) btnMarkPlayed.onclick = (e) => {
                e.stopPropagation();
                markMediaPlayed(item);
            };

            const btnMarkUnplayed = tr.querySelector('.mark-unplayed-btn');
            if (btnMarkUnplayed) btnMarkUnplayed.onclick = (e) => {
                e.stopPropagation();
                markMediaUnplayed(item);
            };

            const btnDelete = tr.querySelector('.delete-btn');
            if (btnDelete) btnDelete.onclick = (e) => {
                e.stopPropagation();
                deleteMedia(item.path, false);
            };

            const btnRestore = tr.querySelector('.restore-btn');
            if (btnRestore) btnRestore.onclick = (e) => {
                e.stopPropagation();
                deleteMedia(item.path, true);
            };

            const btnDeletePermanent = tr.querySelector('.delete-permanent-btn');
            if (btnDeletePermanent) btnDeletePermanent.onclick = (e) => {
                e.stopPropagation();
                permanentlyDeleteMedia(item.path);
            };

            const btnAdd = tr.querySelector('.add-btn');
            if (btnAdd) btnAdd.onclick = (e) => {
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
            if (btnRemove) btnRemove.onclick = (e) => {
                e.stopPropagation();
                removeFromPlaylist(state.filters.playlist, item);
            };

            const trackInput = tr.querySelector('.track-number-input');
            if (trackInput) {
                trackInput.onclick = (e) => e.stopPropagation();
                trackInput.onchange = (e) => {
                    updateTrackNumber(state.filters.playlist, item, e.target.value);
                };
            }

            tbody.appendChild(tr);
        });

        table.querySelectorAll('th[data-sort]').forEach(th => {
            th.onclick = () => {
                const field = th.dataset.sort;
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
                const val = e.target.value;
                if (e.target.checked) {
                    state.filters.excludedDbs = state.filters.excludedDbs.filter(d => d !== val);
                } else {
                    state.filters.excludedDbs.push(val);
                }
                localStorage.setItem('disco-excluded-dbs', JSON.stringify(state.filters.excludedDbs));
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
                state.page = 'curation';
                updateNavActiveStates();
                fetchCuration();
            };
        }

        categoryList.querySelectorAll('.category-btn').forEach(btn => {
            if (btn.id === 'categorization-link-btn') return;
            btn.onclick = (e) => {
                const cat = btn.dataset.cat;
                if (state.page !== 'trash') state.page = 'search';

                if (state.filters.categories.includes(cat)) {
                    state.filters.categories = [];
                } else {
                    state.filters.categories = [cat];
                }

                localStorage.setItem('disco-filter-categories', JSON.stringify(state.filters.categories));
                state.currentPage = 1;
                updateNavActiveStates();
                performSearch();
            };
        });
    }

    // --- Helpers ---
    function errorToast(err, fallbackMsg) {
        if (err.message === 'Access Denied' || err.message === 'Unauthorized') {
            showToast(err.message);
            return true;
        }
        if (fallbackMsg) {
            showToast(fallbackMsg);
        }
        return false;
    }

    function showToast(msg, customEmoji) {
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
        if (e.target.tagName === 'INPUT' || e.target.tagName === 'TEXTAREA' || e.target.tagName === 'SELECT') {
            return;
        }

        // 1. Independent shortcuts (don't require active PiP)
        if (!e.ctrlKey && !e.metaKey && !e.altKey) {
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
                            errorToast(err, 'Failed to copy path');
                        });
                    }
                    return;
                case 'f':
                    const docModal = document.getElementById('document-modal');
                    const pipPlayer = document.getElementById('pip-player');
                    if (!docModal.classList.contains('hidden')) {
                        toggleFullscreen(docModal.querySelector('.modal-content'));
                    } else if (!pipPlayer.classList.contains('hidden')) {
                        const media = pipViewer.querySelector('video, audio, img');
                        if (media) {
                            toggleFullscreen(pipViewer, media.tagName === 'VIDEO' ? media : pipViewer);
                        }
                    }
                    return;
            }
        }

        switch (e.key.toLowerCase()) {
            case 'delete':
                if (state.playback.item && !pipPlayer.classList.contains('hidden')) {
                    const itemToDelete = state.playback.item;
                    if (e.shiftKey) {
                        closePiP();
                    } else {
                        playSibling(1, true, true);
                    }
                    deleteMedia(itemToDelete.path);
                    return;
                }
                break;
        }

        // 2. Rating shortcuts (require active PiP item but not necessarily visible/unpaused)
        if (e.shiftKey && ['Digit1', 'Digit2', 'Digit3', 'Digit4', 'Digit5'].includes(e.code)) {
            if (state.playback.item) {
                const score = parseInt(e.code.replace('Digit', ''));
                rateMedia(state.playback.item, score);
            }
            return;
        }

        // 3. Playback shortcuts (require active & visible PiP)
        const media = pipViewer.querySelector('video, audio, img');
        const isPipVisible = !pipPlayer.classList.contains('hidden');

        if (!e.ctrlKey && !e.metaKey && !e.altKey) {
            if (e.key === 'ArrowLeft' || e.key === 'ArrowRight') {
                const isImage = media && media.tagName === 'IMG';
                if (!isPipVisible || isImage) {
                    playSibling(e.key === 'ArrowLeft' ? -1 : 1, true);
                    return;
                }
            }
        }

        if (!media || !isPipVisible) {
            return;
        }

        const isPlaying = (media.paused === false);
        const duration = media.duration;
        const currentTime = media.currentTime;

        const setTime = (t) => {
            if (media.currentTime !== undefined) media.currentTime = t;
        };

        const playPause = () => {
            if (media.tagName === 'IMG') {
                if (state.playback.slideshowTimer) stopSlideshow();
                else startSlideshow();
            }
            else {
                if (media.paused) media.play();
                else media.pause();
            }
        };

        switch (e.key.toLowerCase()) {
            case 'q':
            case 'w':
            case 's':
            case 'escape':
                closePiP();
                break;
            case ' ':
            case 'k':
                e.preventDefault();
                playPause();
                break;
            case 'm':
                media.muted = !media.muted;
                break;
            case 'j':
                if (media.tagName === 'VIDEO') {
                    cycleSubtitles(e.shiftKey);
                } else {
                    setTime(Math.max(0, currentTime - 10));
                }
                break;
            case 'l':
                setTime(Math.min(duration, currentTime + 10));
                break;
            case 'arrowleft':
                setTime(Math.max(0, currentTime - 5));
                break;
            case 'arrowright':
                setTime(Math.min(duration, currentTime + 5));
                break;
            case '0': case '1': case '2': case '3': case '4':
            case '5': case '6': case '7': case '8': case '9':
                if (e.code.startsWith('Digit')) {
                    const percent = parseInt(e.code.replace('Digit', '')) / 10;
                    if (!isNaN(duration)) {
                        setTime(duration * percent);
                    }
                }
                break;
        }
    });

    // --- Dev Mode Auto-Reload ---
    function setupAutoReload() {
        const events = new EventSource('/api/events');
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
            // Retry connection after a delay
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
            if (state.autoplay) playSibling(1);
        } else if (state.postPlaybackAction === 'ask') {
            openModal('confirm-modal');
            document.getElementById('confirm-yes').onclick = () => {
                closeModal('confirm-modal');
                deleteMedia(item.path);
                if (state.autoplay) playSibling(1);
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
        if (!searchInput.contains(e.target) && !searchSuggestions.contains(e.target)) {
            searchSuggestions.classList.add('hidden');
        }
    });

    searchInput.oninput = (e) => {
        let val = e.target.value;
        if (val.includes('\\')) {
            val = val.replace(/\\/g, '/');
            e.target.value = val;
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
        let val = searchInput.value;
        if (val.includes('\\')) {
            val = val.replace(/\\/g, '/');
            searchInput.value = val;
        }

        if (val.startsWith('/') || val.startsWith('./')) {
            fetchSuggestions(val);
        }
    };

    searchInput.onkeydown = (e) => {
        if (e.key === 'Tab' && e.shiftKey) {
            e.preventDefault();
            const val = searchInput.value;
            if (val.startsWith('/') || val.startsWith('./')) {
                const parts = val.split('/');
                if (val.endsWith('/')) {
                    parts.pop(); // remove empty trailing
                    parts.pop(); // remove last folder
                } else {
                    parts.pop(); // remove partial segment
                }
                const newVal = parts.join('/') + (parts.length > 0 ? '/' : '');
                searchInput.value = newVal || (val.startsWith('/') ? '/' : './');
                fetchSuggestions(searchInput.value);
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
            const path = el.dataset.path;
            const isDir = el.dataset.isDir === 'true';
            if (isDir) {
                if (searchInput.value.startsWith('./')) {
                    const newName = el.dataset.name;
                    const lastSlash = searchInput.value.lastIndexOf('/');
                    const newPath = searchInput.value.substring(0, lastSlash + 1) + newName + '/';
                    searchInput.value = newPath;
                } else {
                    const newPath = path.endsWith('/') ? path : path + '/';
                    searchInput.value = newPath;
                }
                fetchSuggestions(searchInput.value);
                performSearch();
            } else {
                searchInput.value = path;
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
            const path = el.dataset.path;
            const isDir = el.dataset.isDir === 'true';
            if (isDir) {
                if (searchInput.value.startsWith('./')) {
                    const newName = el.dataset.name;
                    const lastSlash = searchInput.value.lastIndexOf('/');
                    const newPath = searchInput.value.substring(0, lastSlash + 1) + newName + '/';
                    searchInput.value = newPath;
                } else {
                    const newPath = path.endsWith('/') ? path : path + '/';
                    searchInput.value = newPath;
                }
                fetchSuggestions(searchInput.value);
                performSearch();
            } else {
                searchInput.value = path;
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
        btn.onclick = (e) => {
            const modal = e.target.closest('.modal');
            modal.classList.add('hidden');
        };
    });

    const closePipBtn = document.querySelector('.close-pip');
    if (closePipBtn) closePipBtn.onclick = closePiP;



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
        pipSpeedBtn.onclick = (e) => {
            e.stopPropagation();
            pipSpeedMenu.classList.toggle('hidden');
        };
    }

    document.querySelectorAll('.speed-opt').forEach(btn => {
        btn.onclick = (e) => {
            e.stopPropagation();
            const rate = parseFloat(btn.dataset.speed);
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
        const currentPos = pipViewer.querySelector('video, audio')?.currentTime || 0;
        openInPiP(state.playback.item);

        const media = pipViewer.querySelector('video, audio');
        if (media) {
            media.onloadedmetadata = () => {
                media.currentTime = currentPos;
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
        let isLongPress = false;
        let longPressTimeout = null;
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
            if (e.target.closest('.pip-controls') || e.target.closest('button') || e.target.closest('select')) return;

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
                    media.currentTime = Math.max(0, media.currentTime - 10);
                    showIndicator('⏪ -10s');
                } else if (relativeX > 0.66) {
                    media.currentTime = Math.min(media.duration, media.currentTime + 10);
                    showIndicator('⏩ +10s');
                } else {
                    if (media.paused) media.play();
                    else media.pause();
                    showIndicator(media.paused ? '⏸️' : '▶️');
                }
                lastTapTime = 0;
                touchStartTime = 0; // Prevent swipe after double tap
                if (e.cancelable) e.preventDefault();
                return;
            }
            lastTapTime = now;

            // Long press for seeking
            isLongPress = false;
            if (longPressTimeout) clearTimeout(longPressTimeout);
            longPressTimeout = setTimeout(() => {
                if (touchStartTime === 0) return;
                isLongPress = true;
                if (media) initialSeekTime = media.currentTime;
            }, 500);

        }, { passive: false });

        pipPlayer.addEventListener('touchmove', (e) => {
            if (touchStartTime === 0) return;
            const touch = e.changedTouches[0];
            const diffX = touch.screenX - touchStartX;
            const diffY = touch.screenY - touchStartY;

            if (isLongPress) {
                const media = pipViewer.querySelector('video, audio');
                if (media && !isNaN(media.duration)) {
                    // MX Player style: movement relative to screen width
                    const screenWidth = window.innerWidth;
                    const sensitivity = 0.5; // Swiping full screen = half video duration
                    const timeDiff = (diffX / screenWidth) * media.duration * sensitivity;
                    const targetTime = Math.max(0, Math.min(media.duration, initialSeekTime + timeDiff));
                    media.currentTime = targetTime;

                    const timeStr = formatDuration(targetTime) + ' / ' + formatDuration(media.duration);
                    showIndicator(timeStr);
                    if (e.cancelable) e.preventDefault();
                }
                return;
            }

            // If it's clearly a gesture for the player, prevent page scroll
            if (Math.abs(diffX) > 10 || Math.abs(diffY) > 10) {
                if (e.cancelable) e.preventDefault();
                // If moving significantly before long press, cancel it
                if (!isLongPress && (Math.abs(diffX) > 30 || Math.abs(diffY) > 30)) {
                    clearTimeout(longPressTimeout);
                }
            }
        }, { passive: false });

        pipPlayer.addEventListener('touchend', (e) => {
            clearTimeout(longPressTimeout);
            if (touchStartTime === 0) return;

            if (isLongPress) {
                isLongPress = false;
                touchStartTime = 0;
                return;
            }

            if (e.target.closest('.pip-controls') || e.target.closest('button') || e.target.closest('select')) {
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
                    closePiP();
                }
            }
            touchStartTime = 0;
        }, { passive: true });
    }

    const settingPlayer = document.getElementById('setting-player');
    if (settingPlayer) settingPlayer.onchange = (e) => {
        state.player = e.target.value;
        localStorage.setItem('disco-player', state.player);
    };

    const settingLanguage = document.getElementById('setting-language');
    if (settingLanguage) settingLanguage.oninput = (e) => {
        state.language = e.target.value;
        localStorage.setItem('disco-language', state.language);

        // Update current tracks
        const media = pipViewer.querySelector('video, audio');
        if (media) {
            for (let i = 0; i < media.textTracks.length; i++) {
                media.textTracks[i].srclang = state.language;
            }
        }
    };

    const settingTheme = document.getElementById('setting-theme');
    if (settingTheme) settingTheme.onchange = (e) => {
        state.theme = e.target.value;
        localStorage.setItem('disco-theme', state.theme);
        applyTheme();
    };

    const settingPostPlayback = document.getElementById('setting-post-playback');
    if (settingPostPlayback) settingPostPlayback.onchange = (e) => {
        state.postPlaybackAction = e.target.value;
        localStorage.setItem('disco-post-playback', state.postPlaybackAction);
    };

    const settingDefaultView = document.getElementById('setting-default-view');
    if (settingDefaultView) settingDefaultView.onchange = (e) => {
        state.defaultView = e.target.value;
        localStorage.setItem('disco-default-view', state.defaultView);

        if (pipPlayer.classList.contains('hidden')) {
            state.playerMode = state.defaultView;
        }
    };



    const settingAutoplay = document.getElementById('setting-autoplay');
    if (settingAutoplay) settingAutoplay.onchange = (e) => {
        state.autoplay = e.target.checked;
        localStorage.setItem('disco-autoplay', state.autoplay);
    };

    if (settingImageAutoplay) settingImageAutoplay.onchange = (e) => {
        state.imageAutoplay = e.target.checked;
        localStorage.setItem('disco-image-autoplay', state.imageAutoplay);
    };

    const settingLocalResume = document.getElementById('setting-local-resume');
    if (settingLocalResume) settingLocalResume.onchange = (e) => {
        state.localResume = e.target.checked;
        localStorage.setItem('disco-local-resume', state.localResume);
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
        const searchContainer = document.querySelector('.search-container');
        if (state.page === 'curation') {
            if (toolbar) toolbar.classList.add('hidden');
            if (searchContainer) searchContainer.classList.add('hidden');
        } else {
            if (toolbar) toolbar.classList.remove('hidden');
            if (searchContainer) searchContainer.classList.remove('hidden');
        }

        // Update Media Type buttons
        document.querySelectorAll('#media-type-list .category-btn').forEach(btn => {
            const isActive = state.filters.types.includes(btn.dataset.type);
            btn.classList.toggle('active', isActive);
        });

        // Update Sliders
        const epFilter = state.filters.episodes.find(f => f.value === '@p');
        if (epFilter && episodesMinSlider) {
            episodesMinSlider.value = epFilter.min;
            episodesMaxSlider.value = epFilter.max;
        } else if (episodesMinSlider) {
            episodesMinSlider.value = 0;
            episodesMaxSlider.value = 100;
        }

        const sizeFilter = state.filters.sizes.find(f => f.value === '@p');
        if (sizeFilter && sizeMinSlider) {
            sizeMinSlider.value = sizeFilter.min;
            sizeMaxSlider.value = sizeFilter.max;
        } else if (sizeMinSlider) {
            sizeMinSlider.value = 0;
            sizeMaxSlider.value = 100;
        }

        const durFilter = state.filters.durations.find(f => f.value === '@p');
        if (durFilter && durationMinSlider) {
            durationMinSlider.value = durFilter.min;
            durationMaxSlider.value = durFilter.max;
        } else if (durationMinSlider) {
            durationMinSlider.value = 0;
            durationMaxSlider.value = 100;
        }
        updateSliderLabels();

        // Update Bins
        if (state.filterBins) {
            document.querySelectorAll('#episodes-list .category-btn').forEach(btn => {
                const bin = state.filterBins.episodes[btn.dataset.index];
                if (bin) btn.classList.toggle('active', state.filters.episodes.some(b => b.label === bin.label));
            });
            document.querySelectorAll('#size-list .category-btn').forEach(btn => {
                const bin = state.filterBins.size[btn.dataset.index];
                if (bin) btn.classList.toggle('active', state.filters.sizes.some(b => b.label === bin.label));
            });
            document.querySelectorAll('#duration-list .category-btn').forEach(btn => {
                const bin = state.filterBins.duration[btn.dataset.index];
                if (bin) btn.classList.toggle('active', state.filters.durations.some(b => b.label === bin.label));
            });
        }

        if (allMediaBtn) allMediaBtn.classList.toggle('active', state.page === 'search' && state.filters.categories.length === 0 && state.filters.genre === '' && state.filters.ratings.length === 0 && !state.filters.playlist && !state.filters.unplayed && !state.filters.unfinished && !state.filters.completed && state.filters.sizes.length === 0 && state.filters.durations.length === 0 && state.filters.episodes.length === 0 && state.filters.types.length === 0);
        if (trashBtn) trashBtn.classList.toggle('active', state.page === 'trash');
        if (duBtn) duBtn.classList.toggle('active', state.page === 'du');
        if (captionsBtn) captionsBtn.classList.toggle('active', state.page === 'captions');

        if (historyInProgressBtn) historyInProgressBtn.classList.toggle('active', state.filters.unfinished);
        if (historyUnplayedBtn) historyUnplayedBtn.classList.toggle('active', state.filters.unplayed);
        if (historyCompletedBtn) historyCompletedBtn.classList.toggle('active', state.filters.completed);

        // View Toggles
        if (viewGrid) viewGrid.classList.toggle('active', state.view === 'grid');
        if (viewGroup) viewGroup.classList.toggle('active', state.view === 'group');
        if (viewDetails) viewDetails.classList.toggle('active', state.view === 'details');

        // Handle playlists and categories in the sidebar lists
        document.querySelectorAll('.sidebar .category-btn').forEach(btn => {
            if (btn === allMediaBtn || btn === trashBtn || btn === duBtn || btn === captionsBtn || btn === historyInProgressBtn || btn === historyUnplayedBtn || btn === historyCompletedBtn) return;
            if (btn.closest('#media-type-list')) return;
            if (btn.closest('#episodes-list') || btn.closest('#size-list') || btn.closest('#duration-list')) return;

            const cat = btn.dataset.cat;
            const genre = btn.dataset.genre;
            const rating = btn.dataset.rating;
            const type = btn.dataset.type;
            // For playlists, we check both the button itself and if it's a wrapper for a drop zone
            const playlist = btn.dataset.title || btn.querySelector('.playlist-name')?.dataset.title;

            let isActive = false;
            if (cat !== undefined) isActive = state.page === 'search' && state.filters.categories.includes(cat);
            else if (genre !== undefined) isActive = state.page === 'search' && state.filters.genre === genre;
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
            // Remove active from other categories
            state.filters.categories = [];
            state.filters.ratings = [];
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
            e.dataTransfer.dropEffect = 'move';
        });

        trashBtn.addEventListener('dragleave', (e) => {
            if (!trashBtn.contains(e.relatedTarget)) {
                trashBtn.classList.remove('drag-over');
            }
        });

        trashBtn.addEventListener('drop', async (e) => {
            e.preventDefault();
            e.stopPropagation();
            trashBtn.classList.remove('drag-over');

            const path = e.dataTransfer.getData('text/plain');
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
                state.filters.unfinished = false;
            } else {
                if (state.page !== 'trash') state.page = 'search';
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
                state.filters.unplayed = false;
            } else {
                if (state.page !== 'trash') state.page = 'search';
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
                state.filters.completed = false;
            } else {
                if (state.page !== 'trash') state.page = 'search';
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
            state.page = 'du';
            updateNavActiveStates();
            fetchDU(state.duPath);
        };
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

    if (sortBy) sortBy.onchange = () => {
        state.filters.sort = sortBy.value;
        localStorage.setItem('disco-sort', state.filters.sort);
        if (state.page === 'playlist') {
            sortPlaylistItems();
            renderResults();
        } else {
            state.currentPage = 1;
            performSearch();
        }
    };

    if (sortReverseBtn) sortReverseBtn.onclick = () => {
        state.filters.reverse = !state.filters.reverse;
        localStorage.setItem('disco-reverse', state.filters.reverse);
        sortReverseBtn.classList.toggle('active');
        if (state.page === 'playlist') {
            sortPlaylistItems();
            renderResults();
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
            localStorage.setItem('disco-view', 'grid');
            updateNavActiveStates();
            performSearch();
        };
    }

    if (viewGroup) {
        viewGroup.onclick = () => {
            state.view = 'group';
            localStorage.setItem('disco-view', 'group');
            updateNavActiveStates();
            performSearch();
        };
    }

    if (viewDetails) {
        viewDetails.onclick = () => {
            state.view = 'details';
            localStorage.setItem('disco-view', 'details');
            updateNavActiveStates();
            performSearch();
        };
    }

    if (prevPageBtn) prevPageBtn.onclick = () => {
        if (state.currentPage > 1) {
            state.currentPage--;
            performSearch();
            resultsContainer.scrollTo(0, 0);
        }
    };

    if (nextPageBtn) nextPageBtn.onclick = () => {
        state.currentPage++;
        performSearch();
        resultsContainer.scrollTo(0, 0);
    };

    // --- Inactivity Tracking ---
    const logoText = document.querySelector('.logo-text');
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

    const logoReset = document.querySelector('.logo');
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
        sidebar.classList.toggle('mobile-open');
        sidebarOverlay.classList.toggle('hidden');
    }

    function closeMobileSidebar() {
        sidebar.classList.remove('mobile-open');
        sidebarOverlay.classList.add('hidden');
    }

    if (menuToggle) menuToggle.onclick = toggleMobileSidebar;
    if (sidebarOverlay) sidebarOverlay.onclick = closeMobileSidebar;

    // Close sidebar when clicking on a category, genre, rating or playlist on mobile
    sidebar.addEventListener('click', (e) => {
        const target = e.target;
        if (target.closest('.category-btn') || target.closest('.playlist-name') || target.closest('#trash-btn') || target.closest('#history-btn')) {
            if (window.innerWidth <= 768) {
                closeMobileSidebar();
            }
        }
    });

    // Initial load
    readUrl(true);
    fetchDatabases();

    fetchCategories();
    fetchGenres();
    fetchRatings();
    fetchPlaylists();
    fetchFilterBins();
    renderCategoryList();
    initSidebarPersistence();
    onUrlChange();
    applyTheme();

    document.addEventListener('fullscreenchange', () => {
        const fsBtn = document.getElementById('doc-fullscreen');
        if (fsBtn) {
            fsBtn.title = document.fullscreenElement ? 'Exit Fullscreen' : 'Toggle Fullscreen';
        }
    });

    // Expose for testing
    window.disco = {
        formatSize,
        formatDuration,
        shortDuration,
        getIcon,
        truncateString,
        formatRelativeDate,
        formatParents,
        openInPiP,
        openInDocumentViewer,
        performSearch,
        updateProgress,
        seekToProgress,
        closePiP,
        getPlayCount,
        markMediaPlayed,
        updateNavActiveStates,
        showToast,
        state
    };
});


