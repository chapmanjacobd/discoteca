// Complex Sorting Modal Module
import { state } from './state';

export interface SortField {
    field: string;
    reverse: boolean;
}

// Available sort fields with labels
export const SORT_FIELDS: { value: string; label: string }[] = [
    { value: 'video_count', label: 'Video Count' },
    { value: 'audio_count', label: 'Audio Count' },
    { value: 'subtitle_count', label: 'Subtitle Count' },
    { value: 'play_count', label: 'Play Count' },
    { value: 'playhead', label: 'Playback Position' },
    { value: 'time_last_played', label: 'Last Played' },
    { value: 'time_created', label: 'Created' },
    { value: 'time_modified', label: 'Modified' },
    { value: 'time_downloaded', label: 'Time Scanned' },
    { value: 'size', label: 'Size' },
    { value: 'duration', label: 'Duration' },
    { value: 'path', label: 'Path' },
    { value: 'title', label: 'Title' },
    { value: 'path_is_remote', label: 'Local vs Remote' },
    { value: 'title_is_null', label: 'Has Title' },
    { value: 'score', label: 'Rating' },
    { value: 'extension', label: 'Extension' },
    { value: 'random', label: 'Random' },
];

// Preset configurations mapped to sort-by values
export const SORT_PRESETS: Record<string, SortField[]> = {
    default: [
        { field: 'video_count', reverse: true },
        { field: 'audio_count', reverse: true },
        { field: 'path_is_remote', reverse: false },
        { field: 'subtitle_count', reverse: true },
        { field: 'play_count', reverse: false },
        { field: 'playhead', reverse: true },
        { field: 'time_last_played', reverse: false },
        { field: 'title_is_null', reverse: false },
        { field: 'path', reverse: false },
    ],
    path: [
        { field: 'path', reverse: false },
    ],
    size: [
        { field: 'size', reverse: false },
    ],
    duration: [
        { field: 'duration', reverse: false },
    ],
    play_count: [
        { field: 'play_count', reverse: false },
    ],
    time_last_played: [
        { field: 'time_last_played', reverse: false },
    ],
    progress: [
        { field: 'playhead', reverse: true },
        { field: 'duration', reverse: false },
    ],
    time_created: [
        { field: 'time_created', reverse: false },
    ],
    time_modified: [
        { field: 'time_modified', reverse: false },
    ],
    time_downloaded: [
        { field: 'time_downloaded', reverse: false },
    ],
    bitrate: [
        { field: 'size', reverse: false },
        { field: 'duration', reverse: true },
    ],
    extension: [
        { field: 'extension', reverse: false },
    ],
    random: [
        { field: 'random', reverse: false },
    ],
};

let currentFields: SortField[] = [];
let draggedElement: HTMLElement | null = null;

export function initComplexSorting() {
    const modal = document.getElementById('sort-complex-modal');
    const openBtn = document.getElementById('sort-complex-btn');
    const closeBtn = modal?.querySelector('.close-modal');
    const cancelBtn = document.getElementById('sort-cancel-btn');
    const applyBtn = document.getElementById('sort-apply-btn');
    const resetBtn = document.getElementById('sort-reset-btn');
    const addFieldBtn = document.getElementById('sort-add-field-btn');
    const fieldsList = document.getElementById('sort-fields-list');

    if (!modal || !openBtn || !fieldsList) return;

    // Open modal - load config based on current sort-by
    openBtn.addEventListener('click', () => {
        loadConfigFromCurrentSort();
        modal.classList.remove('hidden');
    });

    // Close modal
    const closeModal = () => {
        modal.classList.add('hidden');
    };

    if (closeBtn) closeBtn.addEventListener('click', closeModal);
    if (cancelBtn) cancelBtn.addEventListener('click', closeModal);

    // Apply sorting
    if (applyBtn) {
        applyBtn.addEventListener('click', () => {
            saveConfig();
            closeModal();
            // Trigger search with new sort config by dispatching custom event
            window.dispatchEvent(new CustomEvent('complex-sort-applied'));
        });
    }

    // Reset to default (xklb)
    if (resetBtn) {
        resetBtn.addEventListener('click', () => {
            currentFields = [...SORT_PRESETS.default];
            state.filters.reverse = false;
            renderFieldsList();
        });
    }

    // Add new field
    if (addFieldBtn) {
        addFieldBtn.addEventListener('click', () => {
            currentFields.push({ field: 'path', reverse: false });
            renderFieldsList();
        });
    }
}

// Load configuration based on current sort-by selection
export function loadConfigFromCurrentSort() {
    const sortBy = state.filters.sort || 'default';
    const reverse = state.filters.reverse;
    
    // Get base config for current sort-by
    if (SORT_PRESETS[sortBy]) {
        currentFields = [...SORT_PRESETS[sortBy]];
    } else if (state.filters.customSortFields) {
        // Parse existing custom sort fields
        try {
            const parts = state.filters.customSortFields.split(',');
            currentFields = parts.map(p => {
                const trimmed = p.trim();
                const reverseMatch = trimmed.match(/^(.+?)\s+(asc|desc)$/i);
                if (reverseMatch) {
                    return {
                        field: reverseMatch[1].trim(),
                        reverse: reverseMatch[2].toLowerCase() === 'desc'
                    };
                }
                return { field: trimmed, reverse: false };
            });
        } catch {
            currentFields = [...SORT_PRESETS.default];
        }
    } else {
        currentFields = [...SORT_PRESETS.default];
    }
    
    // Apply reverse flag to all fields if set
    if (reverse) {
        currentFields = currentFields.map(f => ({ ...f, reverse: !f.reverse }));
    }
    
    renderFieldsList();
}

// Check if current fields match a preset
export function matchesPreset(fields: SortField[]): string | null {
    for (const [presetName, presetFields] of Object.entries(SORT_PRESETS)) {
        if (fields.length !== presetFields.length) continue;
        
        const allMatch = fields.every((f, i) => 
            f.field === presetFields[i].field && f.reverse === presetFields[i].reverse
        );
        
        if (allMatch) return presetName;
    }
    return null;
}

function saveConfig() {
    localStorage.setItem('disco-complex-sort', JSON.stringify(currentFields));

    // Update state filters
    state.filters.reverse = false; // Reverse is now handled per-field
    
    // Convert fields to comma-separated string for API
    const sortFieldsStr = currentFields
        .map(f => `${f.field} ${f.reverse ? 'desc' : 'asc'}`)
        .join(',');
    
    // Check if this matches a preset
    const matchedPreset = matchesPreset(currentFields);
    
    if (matchedPreset) {
        // Set sort-by to match the preset
        state.filters.sort = matchedPreset;
        state.filters.customSortFields = '';
        localStorage.setItem('disco-sort', matchedPreset);
        localStorage.removeItem('disco-custom-sort-fields');
    } else {
        // Custom configuration - set sort to "custom"
        state.filters.sort = 'custom';
        state.filters.customSortFields = sortFieldsStr;
        localStorage.setItem('disco-sort', 'custom');
        localStorage.setItem('disco-custom-sort-fields', sortFieldsStr);
    }
}

function renderFieldsList() {
    const fieldsList = document.getElementById('sort-fields-list');
    if (!fieldsList) return;

    fieldsList.innerHTML = '';

    currentFields.forEach((field, index) => {
        const item = createFieldItem(field, index);
        fieldsList.appendChild(item);
    });
}

function createFieldItem(field: SortField, index: number): HTMLElement {
    const item = document.createElement('div');
    item.className = 'sort-field-item';
    item.draggable = true;
    item.dataset.index = index.toString();

    // Drag handle
    const handle = document.createElement('span');
    handle.className = 'drag-handle';
    handle.textContent = '☰';
    item.appendChild(handle);

    // Field selector
    const select = document.createElement('select');
    SORT_FIELDS.forEach(f => {
        const option = document.createElement('option');
        option.value = f.value;
        option.textContent = f.label;
        if (f.value === field.field) option.selected = true;
        select.appendChild(option);
    });
    select.onchange = (e) => {
        currentFields[index].field = (e.target as HTMLSelectElement).value;
    };
    item.appendChild(select);

    // Direction toggle
    const direction = document.createElement('div');
    direction.className = 'direction-toggle';
    direction.textContent = field.reverse ? '↓ DESC' : '↑ ASC';
    direction.onclick = () => {
        currentFields[index].reverse = !currentFields[index].reverse;
        direction.textContent = currentFields[index].reverse ? '↓ DESC' : '↑ ASC';
    };
    item.appendChild(direction);

    // Remove button
    const removeBtn = document.createElement('button');
    removeBtn.className = 'remove-field';
    removeBtn.textContent = '×';
    removeBtn.title = 'Remove field';
    removeBtn.onclick = () => {
        currentFields.splice(index, 1);
        renderFieldsList();
    };
    item.appendChild(removeBtn);

    // Drag events
    item.ondragstart = handleDragStart;
    item.ondragend = handleDragEnd;
    item.ondragover = handleDragOver;
    item.ondrop = handleDrop;

    return item;
}

function handleDragStart(e: DragEvent) {
    if (!e.dataTransfer) return;
    draggedElement = e.currentTarget as HTMLElement;
    e.dataTransfer.effectAllowed = 'move';
    e.dataTransfer.setData('text/plain', (e.currentTarget as HTMLElement).dataset.index || '');
    (e.currentTarget as HTMLElement).classList.add('dragging');
}

function handleDragEnd(e: DragEvent) {
    (e.currentTarget as HTMLElement).classList.remove('dragging');
    draggedElement = null;
}

function handleDragOver(e: DragEvent) {
    e.preventDefault();
    e.dataTransfer!.dropEffect = 'move';
}

function handleDrop(e: DragEvent) {
    e.preventDefault();
    const target = e.currentTarget as HTMLElement;
    const fromIndex = parseInt(e.dataTransfer!.getData('text/plain'));
    const toIndex = parseInt(target.dataset.index || '0');

    if (isNaN(fromIndex) || isNaN(toIndex) || fromIndex === toIndex) return;

    // Reorder array
    const [removed] = currentFields.splice(fromIndex, 1);
    currentFields.splice(toIndex, 0, removed);

    renderFieldsList();
}
