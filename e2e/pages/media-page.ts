import { Page, Locator } from '@playwright/test';

/**
 * Page Object Model for media grid/list view
 * Handles interactions with media cards and search functionality
 */
export class MediaPage {
  readonly page: Page;
  readonly searchInput: Locator;
  readonly resultsContainer: Locator;
  readonly mediaCards: Locator;
  readonly sortBySelect: Locator;
  readonly viewGridButton: Locator;
  readonly viewDetailsButton: Locator;
  readonly viewGroupButton: Locator;
  readonly sortReverseBtn: Locator;
  readonly paginationContainer: Locator;
  readonly pageInfo: Locator;
  readonly toast: Locator;
  readonly queueCountBadge: Locator;
  readonly menuToggle: Locator;
  readonly detailsRatings: Locator;
  readonly detailsMediaType: Locator;
  readonly detailsHistory: Locator;
  readonly detailsPlaylists: Locator;
  readonly detailsEpisodes: Locator;
  readonly detailsSize: Locator;
  readonly detailsDuration: Locator;
  readonly mediaTypeList: Locator;
  readonly playlistList: Locator;
  readonly newPlaylistBtn: Locator;
  readonly allMediaBtn: Locator;
  readonly episodesSliderContainer: Locator;
  readonly sizeSliderContainer: Locator;
  readonly durationSliderContainer: Locator;
  readonly filterUnplayed: Locator;
  readonly filterCaptions: Locator;
  readonly filterBrowseContainer: Locator;

  constructor(page: Page) {
    this.page = page;
    this.searchInput = page.locator('#search-input');
    this.resultsContainer = page.locator('#results-container');
    this.mediaCards = page.locator('.media-card');
    this.sortBySelect = page.locator('#sort-by');
    this.viewGridButton = page.locator('#view-grid');
    this.viewDetailsButton = page.locator('#view-details');
    this.viewGroupButton = page.locator('#view-group');
    this.sortReverseBtn = page.locator('#sort-reverse-btn');
    this.paginationContainer = page.locator('#pagination-container');
    this.pageInfo = page.locator('#page-info');
    this.toast = page.locator('#toast');
    this.queueCountBadge = page.locator('#queue-count-badge');
    this.menuToggle = page.locator('#menu-toggle');
    this.detailsRatings = page.locator('#details-ratings');
    this.detailsMediaType = page.locator('#details-media-type');
    this.detailsHistory = page.locator('#details-history');
    this.detailsPlaylists = page.locator('#details-playlists');
    this.detailsEpisodes = page.locator('#details-episodes');
    this.detailsSize = page.locator('#details-size');
    this.detailsDuration = page.locator('#details-duration');
    this.mediaTypeList = page.locator('#media-type-list');
    this.playlistList = page.locator('#playlist-list');
    this.newPlaylistBtn = page.locator('#new-playlist-btn');
    this.allMediaBtn = page.locator('#all-media-btn');
    this.episodesSliderContainer = page.locator('#episodes-slider-container');
    this.sizeSliderContainer = page.locator('#size-slider-container');
    this.durationSliderContainer = page.locator('#duration-slider-container');
    this.filterUnplayed = page.locator('#filter-unplayed');
    this.filterCaptions = page.locator('#filter-captions');
    this.filterBrowseContainer = page.locator('#filter-browse-col');
  }

  /**
   * Navigate to the home page and wait for media to load
   */
  async goto(baseUrl: string): Promise<void> {
    await this.page.goto(baseUrl);
    await this.waitForMediaToLoad();
  }

  /**
   * Wait for media cards to be visible
   */
  async waitForMediaToLoad(timeout: number = 10000): Promise<void> {
    const selectors = [
      '.media-card',
      '.caption-group',
      '.details-table',
      '.no-results',
      '.captions-group-view',
      '.caption-media-card'
    ];
    await this.page.locator(selectors.join(', ')).first().waitFor({ state: 'visible', timeout });
  }

  /**
   * Get count of visible media cards
   */
  async getMediaCount(): Promise<number> {
    return await this.mediaCards.count();
  }

  /**
   * Find and click a media item by title or path
   */
  async openItem(titleOrPath: string): Promise<void> {
    const card = this.mediaCards.filter({
      hasText: titleOrPath
    }).first();
    await card.click();
  }

  /**
   * Open first media item matching type filter
   */
  async openFirstMediaByType(type: 'video' | 'audio' | 'image' | 'text'): Promise<void> {
    const card = this.page.locator(`.media-card[data-type*="${type}"]`).first();
    await card.click();
  }

  /**
   * Search for media by query
   */
  async search(query: string): Promise<void> {
    await this.searchInput.fill(query);
    await this.searchInput.press('Enter');
    await this.waitForMediaToLoad();
  }

  /**
   * Clear search input
   */
  async clearSearch(): Promise<void> {
    await this.searchInput.clear();
    await this.searchInput.press('Enter');
    await this.waitForMediaToLoad();
  }

  /**
   * Switch to grid view
   */
  async switchToGridView(): Promise<void> {
    await this.viewGridButton.click();
    await this.viewGridButton.waitFor({ state: 'visible' });
  }

  /**
   * Switch to details view
   */
  async switchToDetailsView(): Promise<void> {
    await this.viewDetailsButton.click();
    await this.viewDetailsButton.waitFor({ state: 'visible' });
  }

  /**
   * Get current view mode
   */
  async getCurrentViewMode(): Promise<'grid' | 'details'> {
    const classes = await this.viewGridButton.getAttribute('class') || '';
    const gridActive = classes.includes('active');
    return gridActive ? 'grid' : 'details';
  }

  /**
   * Change sort order
   */
  async setSortBy(value: string): Promise<void> {
    await this.sortBySelect.selectOption(value);
  }

  /**
   * Get media card by index
   */
  getMediaCard(index: number): Locator {
    return this.mediaCards.nth(index);
  }

  /**
   * Get media card title
   */
  async getMediaCardTitle(index: number): Promise<string> {
    const card = this.getMediaCard(index);
    return await card.textContent() || '';
  }

  /**
   * Hover over media card to reveal actions
   */
  async hoverOverMediaCard(index: number): Promise<void> {
    const card = this.getMediaCard(index);
    await card.hover();
  }

  /**
   * Wait for specific number of media cards
   */
  async waitForMediaCount(count: number, timeout: number = 5000): Promise<void> {
    await this.page.waitForFunction(
      (expectedCount) => {
        const cards = document.querySelectorAll('.media-card');
        return cards.length === expectedCount;
      },
      count,
      { timeout }
    );
  }

  /**
   * Get media card by path attribute
   */
  getMediaCardByPath(path: string): Locator {
    return this.page.locator(`.media-card[data-path="${path}"]`);
  }

  /**
   * Get first media card matching type
   */
  getFirstMediaCardByType(type: 'video' | 'audio' | 'image' | 'text'): Locator {
    return this.page.locator(`.media-card[data-type*="${type}"]`).first();
  }

  /**
   * Get media card path attribute
   */
  async getMediaCardPath(index: number): Promise<string> {
    const card = this.getMediaCard(index);
    return await card.getAttribute('data-path') || '';
  }

  /**
   * Get media card type attribute
   */
  async getMediaCardType(index: number): Promise<string> {
    const card = this.getMediaCard(index);
    return await card.getAttribute('data-type') || '';
  }

  /**
   * Get all media card types
   */
  async getAllMediaCardTypes(): Promise<string[]> {
    return await this.mediaCards.evaluateAll((els) =>
      els.map(el => el.getAttribute('data-type') || '')
    );
  }

  /**
   * Get all media card paths
   */
  async getAllMediaCardPaths(): Promise<string[]> {
    return await this.mediaCards.evaluateAll((els) =>
      els.map(el => el.getAttribute('data-path') || '')
    );
  }

  /**
   * Get media title text
   */
  async getMediaTitle(index: number): Promise<string> {
    const card = this.getMediaCard(index);
    const title = card.locator('.media-title');
    return await title.textContent() || '';
  }

  /**
   * Get play count badge text
   */
  async getPlayCountBadge(index: number): Promise<string> {
    const card = this.getMediaCard(index);
    const badge = card.locator('.play-count-badge');
    return await badge.textContent() || '';
  }

  /**
   * Get progress bar element
   */
  getProgressBar(index: number): Locator {
    const card = this.getMediaCard(index);
    return card.locator('.progress-bar');
  }

  /**
   * Check if progress bar is visible
   */
  async hasProgress(index: number): Promise<boolean> {
    const card = this.getMediaCard(index);
    const progressBar = card.locator('.progress-bar');
    const playheadIndicator = card.locator('.playhead-indicator');
    return await progressBar.isVisible() || await playheadIndicator.isVisible();
  }

  /**
   * Right click on media card
   */
  async rightClickMediaCard(index: number): Promise<void> {
    const card = this.getMediaCard(index);
    await card.click({ button: 'right' });
  }

  /**
   * Get context menu
   */
  getContextMenu(): Locator {
    return this.page.locator('.context-menu, [role="menu"]');
  }

  /**
   * Get similarity groups
   */
  getSimilarityGroups(): Locator {
    return this.page.locator('.similarity-group');
  }

  /**
   * Get caption cards
   */
  getCaptionCards(): Locator {
    return this.page.locator('.media-card.caption-media-card');
  }

  /**
   * Get caption segments
   */
  getCaptionSegments(): Locator {
    return this.page.locator('.caption-segment');
  }

  /**
   * Get caption text
   */
  async getCaptionText(index: number): Promise<string> {
    const segment = this.getCaptionSegments().nth(index);
    const text = segment.locator('.caption-text');
    return await text.textContent() || '';
  }

  /**
   * Get caption time attribute
   */
  async getCaptionTime(index: number): Promise<number> {
    const segment = this.getCaptionSegments().nth(index);
    const timeAttr = await segment.getAttribute('data-time');
    return parseFloat(timeAttr || '0');
  }

  /**
   * Get caption count badge
   */
  async getCaptionCountBadge(index: number): Promise<string> {
    const card = this.getCaptionCards().nth(index);
    const badge = card.locator('.caption-count-badge');
    return await badge.textContent() || '';
  }

  /**
   * Expand details section
   */
  async expandDetailsSection(section: string): Promise<void> {
    const sectionEl = this.page.locator(`#${section}`);
    await sectionEl.evaluate((el: HTMLDetailsElement) => el.open = true);
  }

  /**
   * Collapse details section
   */
  async collapseDetailSection(section: string): Promise<void> {
    const sectionEl = this.page.locator(`#${section}`);
    await sectionEl.evaluate((el: HTMLDetailsElement) => el.open = false);
  }

  /**
   * Check if details section is open
   */
  async isDetailSectionOpen(section: string): Promise<boolean> {
    const sectionEl = this.page.locator(`#${section}`);
    return await sectionEl.evaluate((el: HTMLDetailsElement) => el.open);
  }

  /**
   * Click category button
   */
  async clickCategoryButton(selector: string): Promise<void> {
    const btn = this.page.locator(selector);
    await btn.click();
  }

  /**
   * Get category button by data attribute
   */
  getCategoryButton(dataAttr: string, value: string): Locator {
    return this.page.locator(`.category-btn[data-${dataAttr}="${value}"]`);
  }

  /**
   * Get rating buttons
   */
  getRatingButtons(): Locator {
    return this.page.locator('.category-btn[data-rating]');
  }

  /**
   * Get playlist buttons
   */
  getPlaylistButtons(): Locator {
    return this.page.locator('#playlist-list .category-btn');
  }

  /**
   * Get favorites playlist button
   */
  getFavoritesPlaylistBtn(): Locator {
    return this.playlistList.locator('.category-btn').filter({ hasText: 'Favorites' });
  }

  /**
   * Get mark played buttons
   */
  getMarkPlayedButtons(): Locator {
    return this.page.locator('.media-action-btn.mark-played');
  }

  /**
   * Get toast message
   */
  async getToastMessage(): Promise<string> {
    return await this.toast.textContent() || '';
  }

  /**
   * Check if toast contains text
   */
  async toastContainsText(text: string): Promise<boolean> {
    const toastText = await this.getToastMessage();
    return toastText.includes(text);
  }

  /**
   * Wait for toast to be visible
   */
  async waitForToast(timeout: number = 5000): Promise<void> {
    await this.toast.waitFor({ state: 'visible', timeout });
  }

  /**
   * Get document modal
   */
  getDocumentModal(): Locator {
    return this.page.locator('#document-modal');
  }

  /**
   * Get document title
   */
  getDocumentTitle(): Locator {
    return this.page.locator('#document-title');
  }

  /**
   * Get document container iframe
   */
  getDocumentIframe(): Locator {
    return this.page.locator('#document-container iframe');
  }

  /**
   * Get document fullscreen button
   */
  getDocumentFullscreenBtn(): Locator {
    return this.page.locator('#doc-fullscreen');
  }

  /**
   * Check if document modal is hidden
   */
  async isDocumentModalHidden(): Promise<boolean> {
    const modal = this.getDocumentModal();
    return await modal.first().evaluate(el => el.classList.contains('hidden'));
  }

  /**
   * Get metadata modal
   */
  getMetadataModal(): Locator {
    return this.page.locator('#metadata-modal');
  }

  /**
   * Get help modal
   */
  getHelpModal(): Locator {
    return this.page.locator('#help-modal');
  }

  /**
   * Get DU toolbar
   */
  getDUTToolbar(): Locator {
    return this.page.locator('#du-toolbar');
  }

  /**
   * Get DU path input
   */
  getDUPathInput(): Locator {
    return this.page.locator('#du-path-input');
  }

  /**
   * Get DU back button
   */
  getDUBackBtn(): Locator {
    return this.page.locator('#du-back-btn');
  }

  /**
   * Get DU cards
   */
  getDUCards(): Locator {
    return this.page.locator('.media-card.du-card');
  }

  /**
   * Get folder cards in DU mode
   */
  getFolderCards(): Locator {
    return this.page.locator('.media-card.du-card');
  }

  /**
   * Get settings modal
   */
  getSettingsModal(): Locator {
    return this.page.locator('#settings-modal');
  }

  /**
   * Get settings modal body
   */
  getSettingsModalBody(): Locator {
    return this.page.locator('#settings-modal .modal-body');
  }

  /**
   * Get setting by ID
   */
  getSetting(settingId: string): Locator {
    return this.page.locator(`#${settingId}`);
  }

  /**
   * Get setting toggle slider
   */
  getSettingToggleSlider(settingId: string): Locator {
    return this.page.locator(`#${settingId}`).locator('xpath=..').locator('.slider');
  }

  /**
   * Get advanced settings summary
   */
  getAdvancedSettingsSummary(): Locator {
    return this.page.locator('summary:has-text("Advanced Settings")');
  }

  /**
   * Scroll settings modal to position
   */
  async scrollSettingsModal(scrollTop: number): Promise<void> {
    await this.page.evaluate((top) => {
      const modalBody = document.querySelector('#settings-modal .modal-body');
      if (modalBody) modalBody.scrollTop = top;
    }, scrollTop);
  }

  /**
   * Get current URL hash
   */
  async getCurrentHash(): Promise<string> {
    return await this.page.evaluate(() => window.location.hash);
  }

  /**
   * Check if element has active class
   */
  async hasActiveClass(selector: string): Promise<boolean> {
    const el = this.page.locator(selector);
    return await el.evaluate((el) => el.classList.contains('active'));
  }

  /**
   * Get local storage item
   */
  async getLocalStorageItem(key: string): Promise<any> {
    return await this.page.evaluate((k) => {
      const item = localStorage.getItem(k);
      return item ? JSON.parse(item) : null;
    }, key);
  }

  /**
   * Set local storage item
   */
  async setLocalStorageItem(key: string, value: any): Promise<void> {
    await this.page.evaluate((args) => {
      localStorage.setItem(args.key, JSON.stringify(args.value));
    }, { key, value });
  }

  /**
   * Remove local storage item
   */
  async removeLocalStorageItem(key: string): Promise<void> {
    await this.page.evaluate((k) => {
      localStorage.removeItem(k);
    }, key);
  }

  /**
   * Get progress from local storage
   */
  async getProgress(): Promise<Record<string, any>> {
    return await this.getLocalStorageItem('disco-progress') || {};
  }

  /**
   * Get play counts from local storage
   */
  async getPlayCounts(): Promise<Record<string, number>> {
    return await this.getLocalStorageItem('disco-play-counts') || {};
  }

  /**
   * Set play count for path
   */
  async setPlayCount(path: string, count: number): Promise<void> {
    const counts = await this.getPlayCounts();
    counts[path] = count;
    await this.setLocalStorageItem('disco-play-counts', counts);
  }

  /**
   * Set progress for path
   */
  async setProgress(path: string, pos: number, last?: number): Promise<void> {
    const progress = await this.getProgress();
    progress[path] = { pos, last: last || Date.now() };
    await this.setLocalStorageItem('disco-progress', progress);
  }

  /**
   * Get video element position
   */
  async getVideoPosition(): Promise<{ x: number; y: number; width: number; height: number } | null> {
    return await this.page.evaluate(() => {
      const video = document.querySelector('video, audio');
      if (!video) return null;
      const rect = video.getBoundingClientRect();
      return { x: rect.x, y: rect.y, width: rect.width, height: rect.height };
    });
  }

  /**
   * Get viewport size
   */
  async getViewportSize(): Promise<{ width: number; height: number } | null> {
    return await this.page.evaluate(() => {
      return { width: window.innerWidth, height: window.innerHeight };
    });
  }

  /**
   * Check if search input is focused
   */
  async isSearchFocused(): Promise<boolean> {
    return await this.page.evaluate(() => {
      return document.activeElement === document.getElementById('search-input');
    });
  }

  /**
   * Get bounding box of element
   */
  async getBoundingBox(selector: string): Promise<{ x: number; y: number; width: number; height: number } | null> {
    const el = this.page.locator(selector);
    const box = await el.first().boundingBox();
    return box;
  }
}
