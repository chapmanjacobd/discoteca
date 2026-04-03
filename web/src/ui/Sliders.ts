import { state } from '../state';
import { formatSize, formatDuration, formatRelativeDate } from '../utils';

let sliders: Record<string, {
    min: HTMLInputElement;
    max: HTMLInputElement;
    label: HTMLElement | null;
    minLabel: HTMLElement | null;
    maxLabel: HTMLElement | null;
}> = {};

let _performSearch: () => void;

export function updateSliderLabels() {
    const updateRange = (type: string) => {
        const s = sliders[type];
        if (!s || !s.min || !state.filterBins) return;

        const minP = parseInt(s.min.value);
        const maxP = parseInt(s.max.value);

        const percentiles = (state.filterBins as any)[`${type}_percentiles`] || [];
        const getVal = (p: number) => {
            if (percentiles.length === 0) return 0;
            // percentiles array has 101 values (0-100), use percentage directly as index
            const idx = Math.round(p);
            if (idx < 0) return percentiles[0];
            if (idx >= percentiles.length) return percentiles[percentiles.length - 1];
            return percentiles[idx];
        };

        const valMin = getVal(minP);
        const valMax = getVal(maxP);

        const format = (v: number) => {
            if (type === 'size') return formatSize(v);
            if (type === 'duration') return formatDuration(v);
            if (['modified', 'created', 'downloaded'].includes(type)) return formatRelativeDate(v);
            if (type === 'episodes') return `${Math.round(v)} files`;
            return Math.round(v).toString();
        };

        if (s.label) s.label.textContent = `${format(valMin)} - ${format(valMax)}`;

        // Use fixed min/max values from the API for footer labels (stable across filtering)
        const minVal = (state.filterBins as any)[`${type}_min_val`];
        const maxVal = (state.filterBins as any)[`${type}_max_val`];
        if (s.minLabel) s.minLabel.textContent = minVal !== undefined ? format(minVal) : format(getVal(0));
        if (s.maxLabel) s.maxLabel.textContent = maxVal !== undefined ? format(maxVal) : format(getVal(100));

        const track = s.min.parentElement?.querySelector('.range-track') as HTMLElement;
        if (track) {
            track.style.background = `linear-gradient(to right,
                var(--border-color) ${minP}%,
                var(--accent-color) ${minP}%,
                var(--accent-color) ${maxP}%,
                var(--border-color) ${maxP}%)`;
        }
    };

    Object.keys(sliders).forEach(updateRange);
}

function handleSliderChange(type: string, minP: string, maxP: string) {
    if (!state.filterBins) return;

    let filterKey = type;
    if (type === 'size') filterKey = 'sizes';
    if (type === 'duration') filterKey = 'durations';

    const lsKey = `disco-filter-${filterKey}`;

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

    const types = ['episodes', 'size', 'duration', 'modified', 'created', 'downloaded'];
    types.forEach(type => {
        const min = document.getElementById(`${type}-min-slider`) as HTMLInputElement;
        const max = document.getElementById(`${type}-max-slider`) as HTMLInputElement;
        if (min && max) {
            sliders[type] = {
                min, max,
                label: document.getElementById(`${type}-percentile-label`),
                minLabel: document.getElementById(`${type}-min-label`),
                maxLabel: document.getElementById(`${type}-max-label`)
            };

            let filterKey = type;
            if (type === 'size') filterKey = 'sizes';
            if (type === 'duration') filterKey = 'durations';

            setupSlider(type, filterKey);
        }
    });

    updateSliderLabels();
}

function setupSlider(type: string, filterKey: string) {
    const s = sliders[type];
    if (!s) return;

    // Restore from state
    const filters = (state.filters as any)[filterKey];
    const filter = filters && filters.find((f: any) => f.value === '@p' || f.value === '@abs');
    if (filter) {
        if (filter.value === '@p') {
            s.min.value = filter.min.toString();
            s.max.value = filter.max.toString();
        }
    }

    const onInput = (e: Event) => {
        let min = parseInt(s.min.value);
        let max = parseInt(s.max.value);

        if (min > max) {
            if (e.target === s.min) {
                s.max.value = min.toString();
            } else {
                s.min.value = max.toString();
            }
        }
        updateSliderLabels();
    };

    s.min.oninput = onInput;
    s.max.oninput = onInput;
    s.min.onchange = () => handleSliderChange(type, s.min.value, s.max.value);
    s.max.onchange = () => handleSliderChange(type, s.min.value, s.max.value);
}

export function updateSlidersFromAbsolute(type: string, filterKey: string) {
    const filters = (state.filters as any)[filterKey];
    const filter = filters && filters.find((f: any) => f.value === '@abs');
    if (filter && state.filterBins) {
        const percentiles = (state.filterBins as any)[`${type}_percentiles`] || [];
        if (percentiles.length > 1) {
            const minTotal = percentiles[0];
            const maxTotal = percentiles[percentiles.length - 1];

            if (maxTotal > minTotal) {
                const minP = Math.max(0, Math.min(100, ((filter.min - minTotal) / (maxTotal - minTotal)) * 100));
                const maxP = Math.max(0, Math.min(100, ((filter.max - minTotal) / (maxTotal - minTotal)) * 100));
                setSliderValues(type, Math.round(minP), Math.round(maxP));
            }
        }
    }
}

export function setSliderValues(type: string, min: number, max: number) {
    const s = sliders[type];
    if (s) {
        s.min.value = min.toString();
        s.max.value = max.toString();
        updateSliderLabels();
    }
}

export function resetSliders() {
    Object.values(sliders).forEach(s => {
        s.min.value = '0';
        s.max.value = '100';
    });
    updateSliderLabels();
}
