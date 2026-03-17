import { Locator } from '@playwright/test';
import { BasePage } from './base-page';

/**
 * Page Object for media grid/list view
 * Handles media card interactions, search, and view modes
 */
export class MediaPage extends BasePage {
  // Media-specific locators
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
  readonly menuToggle: Locator;
  readonly newPlaylistBtn: Locator;
  readonly allMediaBtn: Locator;
  readonly documentModal: Locator;
  readonly documentTitle: Locator;
  readonly documentContainer: Locator;
  readonly documentFullscreenBtn: Locator;
  readonly duToolbar: Locator;
  readonly duPathInput: Locator;
  readonly duBackBtn: Locator;

  constructor(page: any) {
    super(page);
    this.searchInput = page.locator('#search-input');
    this.resultsContainer = page.locator('#results-container');
    this.mediaCards = page.locator('.media-card:not(.skeleton), .details-table tbody tr');
    this.sortBySelect = page.locator('#sort-by');
    this.viewGridButton = page.locator('#view-grid');
    this.viewDetailsButton = page.locator('#view-details');
    this.viewGroupButton = page.locator('#view-group');
    this.sortReverseBtn = page.locator('#sort-reverse-btn');
    this.paginationContainer = page.locator('#pagination-container');
    this.pageInfo = page.locator('#page-info');
    this.menuToggle = page.locator('#menu-toggle');
    this.newPlaylistBtn = page.locator('#new-playlist-btn');
    this.allMediaBtn = page.locator('#all-media-btn');
    this.documentModal = page.locator('#document-modal');
    this.documentTitle = page.locator('#document-title');
    this.documentContainer = page.locator('#document-container');
    this.documentFullscreenBtn = page.locator('#doc-fullscreen');
    this.duToolbar = page.locator('#du-toolbar');
    this.duPathInput = page.locator('#du-path-input');
    this.duBackBtn = page.locator('#du-back-btn');
  }

  /**
   * Navigate to home page and wait for media to load
   */
  async goto(baseUrl: string, timeout: number = 10000): Promise<void> {
    await this.page.goto(baseUrl);
    await this.waitForMediaToLoad(timeout);
    await this.waitForPageState(timeout);
  }

  /**
   * Wait for media cards to be visible
   */
  async waitForMediaToLoad(timeout: number = 10000): Promise<void> {
    const selectors = [
      '.media-card:not(.skeleton)',
      '.details-table tbody tr',
      '.caption-group',
      '.details-table',
      '.no-results',
      '.captions-group-view',
      '.caption-media-card',
      '.is-folder',
    ];
    await this.page.locator(selectors.join(', ')).first().waitFor({ state: 'visible', timeout });
  }

  /**
   * Wait for page state to be populated
   */
  async waitForPageState(timeout: number = 10000): Promise<void> {
    await this.page.waitForFunction((timeout) => {
      const disco = (window as any).disco;
      if (!disco) return false;
      const isDUMode = window.location.hash.includes('mode=du') || disco.state?.page === 'du';
      if (isDUMode) {
        if (disco.duData && disco.duData.length > 0) return true;
        const duToolbar = document.getElementById('du-toolbar');
        if (duToolbar && !duToolbar.classList.contains('hidden')) return true;
        return false;
      }
      return disco.currentMedia && disco.currentMedia.length > 0;
    }, { timeout });
  }

  /**
   * Get count of visible media cards
   */
  async getMediaCount(): Promise<number> {
    return await this.mediaCards.count();
  }

  /**
   * Get media card by index
   */
  getMediaCard(index: number): Locator {
    return this.mediaCards.nth(index);
  }

  /**
   * Get clickable area of media card
   */
  getMediaCardClickable(index: number): Locator {
    return this.mediaCards.nth(index).locator('.media-title, .media-info').first();
  }

  /**
   * Open media item by title or path
   */
  async openItem(titleOrPath: string): Promise<void> {
    const card = this.mediaCards.filter({ hasText: titleOrPath }).first();
    await this._clickCard(card);
  }

  /**
   * Open first media item matching type
   */
  async openFirstMediaByType(type: 'video' | 'audio' | 'image' | 'text'): Promise<void> {
    const card = this.page.locator(`.media-card[data-type*="${type}"]`).first();
    await this._clickCard(card);
  }

  /**
   * Click first media by type
   */
  async clickFirstMediaByType(type: 'video' | 'audio' | 'image' | 'text'): Promise<void> {
    await this.openFirstMediaByType(type);
  }

  /**
   * Click nth media by type
   */
  async clickNthMediaByType(
    type: 'video' | 'audio' | 'image' | 'text' | 'document',
    index: number = 0,
    timeout: number = 500
  ): Promise<void> {
    const searchType = type === 'document' ? 'text' : type;
    const card = this.page.locator(`.media-card[data-type*="${searchType}"]`).nth(index);
    await this._clickCard(card);
    if (timeout > 0) await this.waitForTimeout(timeout);
  }

  /**
   * Click first video or audio (fallback)
   */
  async clickFirstVideoOrAudio(): Promise<void> {
    const videoCard = this.page.locator('.media-card[data-type*="video"]').first();
    if (await videoCard.count() > 0) {
      await this.clickFirstMediaByType('video');
    } else {
      await this.clickFirstMediaByType('audio');
    }
  }

  /**
   * Click first image or video (fallback)
   */
  async clickFirstImageOrVideo(): Promise<void> {
    const imageCard = this.page.locator('.media-card[data-type*="image"]').first();
    if (await imageCard.count() > 0) {
      await this.clickFirstMediaByType('image');
    } else {
      await this.clickFirstVideoOrAudio();
    }
  }

  /**
   * Click first document
   */
  async clickFirstDocument(): Promise<void> {
    await this.clickFirstMediaByType('text');
  }

  /**
   * Search for media
   */
  async search(query: string): Promise<void> {
    await this.searchInput.fill(query);
    await this.searchInput.press('Enter');
    await this.page.locator('.media-card.skeleton').first().waitFor({ state: 'visible', timeout: 1000 }).catch(() => {});
    await this.waitForMediaToLoad();
  }

  /**
   * Clear search
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
    return classes.includes('active') ? 'grid' : 'details';
  }

  /**
   * Set sort order
   */
  async setSortBy(value: string): Promise<void> {
    await this.sortBySelect.selectOption(value);
  }

  /**
   * Get media card title
   */
  async getMediaCardTitle(index: number): Promise<string> {
    return await this.getMediaCard(index).textContent() || '';
  }

  /**
   * Get media title text
   */
  async getMediaTitle(index: number): Promise<string> {
    const card = this.getMediaCard(index);
    return await card.locator('.media-title').textContent() || '';
  }

  /**
   * Get play count badge
   */
  async getPlayCountBadge(index: number): Promise<string> {
    const card = this.getMediaCard(index);
    return await card.locator('.play-count-badge').textContent() || '';
  }

  /**
   * Check if media card has progress
   */
  async hasProgress(index: number): Promise<boolean> {
    const card = this.getMediaCard(index);
    const progressBar = card.locator('.progress-bar');
    const playheadIndicator = card.locator('.playhead-indicator');
    return await progressBar.isVisible() || await playheadIndicator.isVisible();
  }

  /**
   * Hover over media card
   */
  async hoverOverMediaCard(index: number): Promise<void> {
    await this.getMediaCard(index).hover();
  }

  /**
   * Right click media card
   */
  async rightClickMediaCard(index: number): Promise<void> {
    await this.getMediaCard(index).click({ button: 'right' });
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
    return await this.getCaptionSegments().nth(index).locator('.caption-text').textContent() || '';
  }

  /**
   * Get caption time
   */
  async getCaptionTime(index: number): Promise<number> {
    const timeAttr = await this.getCaptionSegments().nth(index).getAttribute('data-time');
    return parseFloat(timeAttr || '0');
  }

  /**
   * Get caption count badge
   */
  async getCaptionCountBadge(index: number): Promise<string> {
    const card = this.getCaptionCards().nth(index);
    return await card.locator('.caption-count-badge').textContent() || '';
  }

  /**
   * Get media cards by type
   */
  getMediaCardsByType(type: 'video' | 'audio' | 'image' | 'text' | 'document'): Locator {
    const searchType = type === 'document' ? 'text' : type;
    return this.page.locator(`.media-card[data-type*="${searchType}"]`);
  }

  /**
   * Get media card by text pattern and type
   */
  getMediaCardByText(type: 'video' | 'audio' | 'image' | 'text' | 'document', textPattern: string | RegExp): Locator {
    const searchType = type === 'document' ? 'text' : type;
    const cards = this.page.locator(`.media-card[data-type*="${searchType}"]`);
    // Use filter for both string and RegExp to get the card element, not child elements
    return cards.filter({ hasText: textPattern });
  }

  /**
   * Get media card by path
   */
  getMediaCardByPath(path: string): Locator {
    return this.page.locator(`.media-card[data-path="${path}"]`);
  }

  /**
   * Get first media card by type
   */
  getFirstMediaCardByType(type: 'video' | 'audio' | 'image' | 'text'): Locator {
    return this.page.locator(`.media-card[data-type*="${type}"]`).first();
  }

  /**
   * Get media card path
   */
  async getMediaCardPath(index: number): Promise<string> {
    return await this.getMediaCard(index).getAttribute('data-path') || '';
  }

  /**
   * Get media card type
   */
  async getMediaCardType(index: number): Promise<string> {
    return await this.getMediaCard(index).getAttribute('data-type') || '';
  }

  /**
   * Get all media card types
   */
  async getAllMediaCardTypes(): Promise<string[]> {
    return await this.mediaCards.evaluateAll(els => els.map(el => el.getAttribute('data-type') || ''));
  }

  /**
   * Get all media card paths
   */
  async getAllMediaCardPaths(): Promise<string[]> {
    return await this.mediaCards.evaluateAll(els => els.map(el => el.getAttribute('data-path') || ''));
  }

  /**
   * Get rating buttons
   */
  getRatingButtons(): Locator {
    return this.detailsRatings.locator('.category-btn[data-rating]');
  }

  /**
   * Get playlist buttons
   */
  getPlaylistButtons(): Locator {
    return this.playlistList.locator('.category-btn');
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
   * Get DU cards
   */
  getDUCards(): Locator {
    return this.page.locator('.media-card.is-folder');
  }

  /**
   * Get folder cards
   */
  getFolderCards(): Locator {
    return this.page.locator('.media-card.is-folder');
  }

  /**
   * Get DU file cards
   */
  getDUFileCards(): Locator {
    return this.page.locator('#results-container .media-card[data-path]');
  }

  /**
   * Get all DU files as array
   */
  async getDUFiles(): Promise<string[]> {
    const fileCards = this.getDUFileCards();
    const count = await fileCards.count();
    const paths: string[] = [];
    for (let i = 0; i < count; i++) {
      const path = await fileCards.nth(i).getAttribute('data-path');
      if (path) paths.push(path);
    }
    return paths;
  }

  /**
   * Find and click folder by text
   */
  async findAndClickFolderByText(searchText: string | RegExp, timeout: number = 1500): Promise<boolean> {
    const folderCards = this.getFolderCards();
    const folderCount = await folderCards.count();

    for (let i = 0; i < folderCount; i++) {
      const folderText = await folderCards.nth(i).textContent();
      if (folderText && (
        typeof searchText === 'string' ? folderText.includes(searchText) : searchText.test(folderText)
      )) {
        await folderCards.nth(i).click();
        await this.waitForTimeout(timeout);
        return true;
      }
    }
    return false;
  }

  /**
   * Click media by text pattern
   */
  async clickMediaByText(
    type: 'video' | 'audio' | 'image' | 'text' | 'document',
    textPattern: string | RegExp,
    timeout: number = 500
  ): Promise<void> {
    const searchType = type === 'document' ? 'text' : type;
    const cards = this.page.locator(`.media-card[data-type*="${searchType}"]`);
    const count = await cards.count();

    for (let i = 0; i < count; i++) {
      const text = await cards.nth(i).textContent();
      if (text && (
        typeof textPattern === 'string' ? text.includes(textPattern) : textPattern.test(text)
      )) {
        await this._clickCard(cards.nth(i));
        if (timeout > 0) await this.waitForTimeout(timeout);
        return;
      }
    }
    throw new Error(`No media found matching: ${textPattern}`);
  }

  /**
   * Click media card (for backward compatibility)
   */
  async clickMediaCard(index: number): Promise<void> {
    const card = this.getMediaCard(index);
    await this._clickCard(card);
  }

  /**
   * Get DU toolbar (getter for backward compatibility)
   */
  getDUTToolbar(): Locator {
    return this.duToolbar;
  }

  /**
   * Get DU path input (getter for backward compatibility)
   */
  getDUPathInput(): Locator {
    return this.duPathInput;
  }

  /**
   * Get DU back button (getter for backward compatibility)
   */
  getDUBackBtn(): Locator {
    return this.duBackBtn;
  }

  /**
   * Expand details section (for backward compatibility)
   */
  async expandDetailSection(section: string): Promise<void> {
    const sectionEl = this.page.locator(`#${section}`);
    await sectionEl.evaluate((el: HTMLDetailsElement) => el.open = true);
  }

  /**
   * Expand details section (alias for backward compatibility - typo version)
   */
  async expandDetailsSection(section: string): Promise<void> {
    await this.expandDetailSection(section);
  }

  /**
   * Collapse details section (for backward compatibility)
   */
  async collapseDetailSection(section: string): Promise<void> {
    const sectionEl = this.page.locator(`#${section}`);
    await sectionEl.evaluate((el: HTMLDetailsElement) => el.open = false);
  }

  /**
   * Click category button (for backward compatibility)
   */
  async clickCategoryButton(selector: string): Promise<void> {
    const btn = this.page.locator(selector);
    await btn.click();
  }

  /**
   * Get settings modal (for backward compatibility)
   */
  getSettingsModal(): Locator {
    return this.page.locator('#settings-modal');
  }

  /**
   * Get viewport size (for backward compatibility)
   */
  getViewportSize(): { width: number; height: number } {
    return this.page.viewportSize() || { width: 1280, height: 720 };
  }

  /**
   * Check if search input is focused (for backward compatibility)
   */
  async isSearchFocused(): Promise<boolean> {
    return await this.searchInput.evaluate((el: HTMLInputElement) => el === document.activeElement);
  }

  /**
   * Get advanced settings summary (for backward compatibility)
   */
  getAdvancedSettingsSummary(): Locator {
    // Find the details/summary element containing "Advanced Settings" text
    return this.page.locator('#settings-modal details').filter({ hasText: 'Advanced Settings' }).first();
  }

  /**
   * Get setting toggle slider (for backward compatibility)
   */
  getSettingToggleSlider(settingId: string): Locator {
    return this.page.locator(`#${settingId}`).locator('xpath=..').locator('.slider');
  }

  /**
   * Navigate to DU files with fallback (for backward compatibility)
   */
  async navigateToDUFolderWithFallback(
    folderName: string | RegExp,
    minFiles: number = 2,
    maxDepth: number = 5,
    folderTimeout: number = 1500
  ): Promise<{
    fileCards: Locator;
    folderCards: Locator;
    fileCount: number;
    folderCount: number;
    depth: number;
  }> {
    // Try to find and click the specific folder first
    const folderFound = await this.findAndClickFolderByText(folderName, folderTimeout);

    // Wait for files to load
    await this.waitForTimeout(500);

    // Get file and folder counts
    const fileCards = this.getDUFileCards();
    const folderCards = this.getFolderCards();
    const fileCount = await fileCards.count();
    const folderCount = await folderCards.count();

    return {
      fileCards,
      folderCards,
      fileCount,
      folderCount,
      depth: folderFound ? 1 : 0
    };
  }

  /**
   * Get media count by type (for backward compatibility)
   */
  async getMediaCountByType(type: 'video' | 'audio' | 'image' | 'text'): Promise<number> {
    const cards = this.getMediaCardsByType(type);
    return await cards.count();
  }

  /**
   * Get progress (for backward compatibility - returns progress data object)
   */
  async getProgress(): Promise<Record<string, { pos: number; last: number }>> {
    return await this.page.evaluate(() => {
      const data = localStorage.getItem('disco-progress');
      return data ? JSON.parse(data) : {};
    });
  }

  /**
   * Set progress in localStorage (for backward compatibility)
   */
  async setProgress(path: string, position: number, timestamp: number): Promise<void> {
    await this.page.evaluate(({ path, pos, ts }) => {
      const data = localStorage.getItem('disco-progress');
      const progress = data ? JSON.parse(data) : {};
      progress[path] = { pos, last: ts };
      localStorage.setItem('disco-progress', JSON.stringify(progress));
    }, { path, pos: position, ts: timestamp });
  }

  /**
   * Set play count in localStorage
   */
  async setPlayCount(mediaPath: string, count: number): Promise<void> {
    await this.page.evaluate(({ path, cnt }) => {
      const key = 'disco-play-counts';
      const counts = JSON.parse(localStorage.getItem(key) || '{}');
      counts[path] = cnt;
      localStorage.setItem(key, JSON.stringify(counts));
    }, { path: mediaPath, cnt: count });
  }

  /**
   * Get setting (for backward compatibility)
   */
  getSetting(key: string): Locator {
    // Handle both id-based and data-setting-based selectors
    if (key.startsWith('setting-')) {
      return this.page.locator(`#${key}`);
    }
    return this.page.locator(`[data-setting="${key}"]`);
  }

  /**
   * Scroll settings modal (for backward compatibility)
   */
  async scrollSettingsModal(direction: 'up' | 'down' | number, amount: number = 100): Promise<void> {
    const modal = this.page.locator('#settings-modal .modal-content');
    const scrollAmount = typeof direction === 'number' ? direction : amount;
    const scrollDir = direction === 'up' ? -1 : 1;
    await modal.evaluate((el: HTMLElement, amt: number) => el.scrollTop += amt, scrollAmount * scrollDir);
  }

  /**
   * Get current hash from URL (for backward compatibility)
   */
  async getCurrentHash(): Promise<string> {
    return await this.page.evaluate(() => window.location.hash);
  }

  /**
   * Remove localStorage item (for backward compatibility)
   */
  async removeLocalStorageItem(key: string): Promise<void> {
    await this.page.evaluate((k: string) => localStorage.removeItem(k), key);
  }

  /**
   * Internal: Click card safely
   */
  private async _clickCard(card: Locator): Promise<void> {
    const title = card.locator('.media-title').first();
    if (await title.count() > 0) {
      await title.click();
    } else {
      const info = card.locator('.media-info').first();
      if (await info.count() > 0) {
        await info.click();
      } else {
        await card.click({ position: { x: 100, y: 50 } });
      }
    }
  }
}
