import { state } from '../state';
import { formatSize, formatDuration } from '../utils';

let episodesMinSlider: HTMLInputElement;
let episodesMaxSlider: HTMLInputElement;
let episodesLabel: HTMLElement | null;
let epMinLabel: HTMLElement | null;
let epMaxLabel: HTMLElement | null;

let sizeMinSlider: HTMLInputElement;
let sizeMaxSlider: HTMLInputElement;
let sizeLabel: HTMLElement | null;
let sizeMinLabel: HTMLElement | null;
let sizeMaxLabel: HTMLElement | null;

let durationMinSlider: HTMLInputElement;
let durationMaxSlider: HTMLInputElement;
let durationLabel: HTMLElement | null;
let durMinLabel: HTMLElement | null;
let durMaxLabel: HTMLElement | null;

let _performSearch: () => void;

export function updateSliderLabels() {
    const updateRange = (minSlider: HTMLInputElement, maxSlider: HTMLInputElement, label: HTMLElement | null, minF: HTMLElement | null, maxF: HTMLElement | null, type: string) => {
        if (!minSlider || !state.filterBins) return;

        const minP = parseInt(minSlider.value);
        const maxP = parseInt(maxSlider.value);

        const percentiles = (state.filterBins as any)[`${type}_percentiles`] || [];
        const getVal = (p: number) => {
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

        const format = (v: number) => {
            if (type === 'size') return formatSize(v);
            if (type === 'duration') return formatDuration(v);
            return Math.round(v).toString();
        };

        if (label) label.textContent = `${format(valMin)} - ${format(valMax)}`;

        if (minF) minF.textContent = format(getVal(0));
        if (maxF) maxF.textContent = format(getVal(100));

        const track = minSlider.parentElement?.querySelector('.range-track') as HTMLElement;
        if (track) {
            track.style.background = `linear-gradient(to right,
                var(--border-color) ${minP}%,
                var(--accent-color) ${minP}%,
                var(--accent-color) ${maxP}%,
                var(--border-color) ${maxP}%)`;
        }
    };

    if (episodesMinSlider) updateRange(episodesMinSlider, episodesMaxSlider, episodesLabel, epMinLabel, epMaxLabel, 'episodes');
    if (sizeMinSlider) updateRange(sizeMinSlider, sizeMaxSlider, sizeLabel, sizeMinLabel, sizeMaxLabel, 'size');
    if (durationMinSlider) updateRange(durationMinSlider, durationMaxSlider, durationLabel, durMinLabel, durMaxLabel, 'duration');
}

function handleSliderChange(type: string, minP: string, maxP: string) {
    let filterKey = '';
    let lsKey = '';

    if (!state.filterBins) return;

    if (type === 'episodes') { filterKey = 'episodes'; lsKey = 'disco-filter-episodes'; }
    else if (type === 'size') { filterKey = 'sizes'; lsKey = 'disco-filter-sizes'; }
    else if (type === 'duration') { filterKey = 'durations'; lsKey = 'disco-filter-durations'; }

    if (!filterKey) return;

    // Use percentiles for population weighting and correct filtering
    (state.filters as any)[filterKey] = [{
        label: `${minP}-${maxP}%`,
        value: `@p`,
        min: parseInt(minP),
        max: parseInt(maxP)
    }];

    localStorage.setItem(lsKey, JSON.stringify((state.filters as any)[filterKey]));
    updateSliderLabels();
    if (_performSearch) _performSearch();
}

export function initSliders(performSearch: () => void) {
    _performSearch = performSearch;

    episodesMinSlider = document.getElementById('episodes-min-slider') as HTMLInputElement;
    episodesMaxSlider = document.getElementById('episodes-max-slider') as HTMLInputElement;
    episodesLabel = document.getElementById('episodes-percentile-label');
    epMinLabel = document.getElementById('episodes-min-label');
    epMaxLabel = document.getElementById('episodes-max-label');

    sizeMinSlider = document.getElementById('size-min-slider') as HTMLInputElement;
    sizeMaxSlider = document.getElementById('size-max-slider') as HTMLInputElement;
    sizeLabel = document.getElementById('size-percentile-label');
    sizeMinLabel = document.getElementById('size-min-label');
    sizeMaxLabel = document.getElementById('size-max-label');

    durationMinSlider = document.getElementById('duration-min-slider') as HTMLInputElement;
    durationMaxSlider = document.getElementById('duration-max-slider') as HTMLInputElement;
    durationLabel = document.getElementById('duration-percentile-label');
    durMinLabel = document.getElementById('duration-min-label');
    durMaxLabel = document.getElementById('duration-max-label');

    const setupSlider = (minSlider: HTMLInputElement, maxSlider: HTMLInputElement, type: string, filterKey: string) => {
        if (!minSlider) return;

        // Restore from state
        const filters = (state.filters as any)[filterKey];
        const filter = filters && filters.find((f: any) => f.value === '@p' || f.value === '@abs');
        if (filter) {
            if (filter.value === '@p') {
                minSlider.value = filter.min;
                maxSlider.value = filter.max;
            }
        }

        const onInput = (e: Event) => {
            let min = parseInt(minSlider.value);
            let max = parseInt(maxSlider.value);

            if (min > max) {
                if (e.target === minSlider) {
                    maxSlider.value = min.toString();
                } else {
                    minSlider.value = max.toString();
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

    setupSlider(episodesMinSlider, episodesMaxSlider, 'episodes', 'episodes');
    setupSlider(sizeMinSlider, sizeMaxSlider, 'size', 'sizes');
    setupSlider(durationMinSlider, durationMaxSlider, 'duration', 'durations');
    updateSliderLabels();
}

export function updateSlidersFromAbsolute(type: string, filterKey: string) {
    const filters = (state.filters as any)[filterKey];
    const filter = filters && filters.find((f: any) => f.value === '@abs');
    if (filter && state.filterBins) {
        let minTotal = 0, maxTotal = 0;
        if (type === 'episodes') { minTotal = state.filterBins.episodes_min; maxTotal = state.filterBins.episodes_max; }
        else if (type === 'size') { minTotal = state.filterBins.size_min; maxTotal = state.filterBins.size_max; }
        else if (type === 'duration') { minTotal = state.filterBins.duration_min; maxTotal = state.filterBins.duration_max; }

        if (maxTotal > minTotal) {
            const minP = Math.max(0, Math.min(100, ((filter.min - minTotal) / (maxTotal - minTotal)) * 100));
            const maxP = Math.max(0, Math.min(100, ((filter.max - minTotal) / (maxTotal - minTotal)) * 100));
            setSliderValues(type, Math.round(minP), Math.round(maxP));
        }
    }
}

export function setSliderValues(type: string, min: number, max: number) {
    if (type === 'episodes' && episodesMinSlider) {
        episodesMinSlider.value = min.toString();
        episodesMaxSlider.value = max.toString();
    } else if (type === 'size' && sizeMinSlider) {
        sizeMinSlider.value = min.toString();
        sizeMaxSlider.value = max.toString();
    } else if (type === 'duration' && durationMinSlider) {
        durationMinSlider.value = min.toString();
        durationMaxSlider.value = max.toString();
    }
    updateSliderLabels();
}

export function resetSliders() {
    if (episodesMinSlider) {
        episodesMinSlider.value = '0';
        episodesMaxSlider.value = '100';
    }
    if (sizeMinSlider) {
        sizeMinSlider.value = '0';
        sizeMaxSlider.value = '100';
    }
    if (durationMinSlider) {
        durationMinSlider.value = '0';
        durationMaxSlider.value = '100';
    }
    updateSliderLabels();
}
