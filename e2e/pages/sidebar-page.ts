import { Page, Locator } from '@playwright/test';

/**
 * Page Object Model for sidebar navigation and filters
 * Handles sidebar interactions, mode switching, and filter application
 */
export class SidebarPage {
  readonly page: Page;
  readonly menuToggle: Locator;
  readonly sidebar: Locator;
  readonly duButton: Locator;
  readonly captionsButton: Locator;
  readonly curationButton: Locator;
  readonly channelSurfButton: Locator;
  readonly allMediaButton: Locator;
  readonly trashButton: Locator;
  readonly settingsButton: Locator;
  readonly historyInProgressButton: Locator;
  readonly historyUnplayedButton: Locator;
  readonly historyCompletedButton: Locator;
  readonly categoryList: Locator;
  readonly filterBrowseContainer: Locator;
  readonly detailsRatings: Locator;
  readonly detailsMediaType: Locator;
  readonly detailsHistory: Locator;
  readonly detailsPlaylists: Locator;
  readonly detailsEpisodes: Locator;
  readonly detailsSize: Locator;
  readonly detailsDuration: Locator;
  readonly detailsFilterBrowse: Locator;
  readonly mediaTypeList: Locator;
  readonly playlistList: Locator;
  readonly newPlaylistBtn: Locator;
  readonly episodesSliderContainer: Locator;
  readonly sizeSliderContainer: Locator;
  readonly durationSliderContainer: Locator;
  readonly filterUnplayed: Locator;
  readonly filterCaptions: Locator;

  constructor(page: Page) {
    this.page = page;
    this.menuToggle = page.locator('#menu-toggle');
    this.sidebar = page.locator('#sidebar');
    this.duButton = page.locator('#du-btn');
    this.captionsButton = page.locator('#captions-btn');
    this.curationButton = page.locator('#curation-btn');
    this.channelSurfButton = page.locator('#channel-surf-btn');
    this.allMediaButton = page.locator('#all-media-btn');
    this.trashButton = page.locator('#trash-btn');
    this.settingsButton = page.locator('#settings-button');
    this.historyInProgressButton = page.locator('#history-in-progress-btn');
    this.historyUnplayedButton = page.locator('#history-unplayed-btn');
    this.historyCompletedButton = page.locator('#history-completed-btn');
    this.categoryList = page.locator('#category-list');
    this.filterBrowseContainer = page.locator('#filter-browse-col');
    this.detailsRatings = page.locator('#details-ratings');
    this.detailsMediaType = page.locator('#details-media-type');
    this.detailsHistory = page.locator('#details-history');
    this.detailsPlaylists = page.locator('#details-playlists');
    this.detailsEpisodes = page.locator('#details-episodes');
    this.detailsSize = page.locator('#details-size');
    this.detailsDuration = page.locator('#details-duration');
    this.detailsFilterBrowse = page.locator('#details-filter-browse');
    this.mediaTypeList = page.locator('#media-type-list');
    this.playlistList = page.locator('#playlist-list');
    this.newPlaylistBtn = page.locator('#new-playlist-btn');
    this.episodesSliderContainer = page.locator('#episodes-slider-container');
    this.sizeSliderContainer = page.locator('#size-slider-container');
    this.durationSliderContainer = page.locator('#duration-slider-container');
    this.filterUnplayed = page.locator('#filter-unplayed');
    this.filterCaptions = page.locator('#filter-captions');
  }

  /**
   * Open sidebar on mobile (if visible)
   */
  async open(): Promise<void> {
    if (await this.menuToggle.isVisible()) {
      await this.menuToggle.click();
      await this.sidebar.waitFor({ state: 'visible' });
    }
  }

  /**
   * Close sidebar on mobile
   */
  async close(): Promise<void> {
    if (await this.menuToggle.isVisible()) {
      await this.menuToggle.click();
      await this.sidebar.waitFor({ state: 'hidden' });
    }
  }

  /**
   * Navigate to Disk Usage view
   */
  async openDiskUsage(): Promise<void> {
    await this.open();
    await this.duButton.click();
    await this.page.locator('#du-toolbar').waitFor({ state: 'visible' });
  }

  /**
   * Navigate to Captions view
   */
  async openCaptions(): Promise<void> {
    await this.open();
    await this.captionsButton.click();
    await this.page.locator('.caption-media-card').first().waitFor({ state: 'visible' });
  }

  /**
   * Navigate to Curation view
   */
  async openCuration(): Promise<void> {
    await this.open();
    await this.curationButton.click();
  }

  /**
   * Navigate to Trash view
   */
  async openTrash(): Promise<void> {
    await this.open();
    await this.trashButton.click();
  }

  /**
   * Navigate to History - In Progress
   */
  async openHistoryInProgress(): Promise<void> {
    await this.open();
    await this.historyInProgressButton.waitFor({ state: 'visible' });
    await this.historyInProgressButton.click();
  }

  /**
   * Navigate to History - Unplayed
   */
  async openHistoryUnplayed(): Promise<void> {
    await this.open();
    await this.historyUnplayedButton.waitFor({ state: 'visible' });
    await this.historyUnplayedButton.click();
  }

  /**
   * Navigate to History - Completed
   */
  async openHistoryCompleted(): Promise<void> {
    await this.open();
    await this.historyCompletedButton.waitFor({ state: 'visible' });
    await this.historyCompletedButton.click();
  }

  /**
   * Navigate to All Media (reset filters)
   */
  async openAllMedia(): Promise<void> {
    await this.open();
    await this.allMediaButton.waitFor({ state: 'visible' });
    await this.allMediaButton.click();
  }

  /**
   * Open settings modal
   */
  async openSettings(): Promise<void> {
    await this.settingsButton.click();
    await this.page.locator('#settings-modal').waitFor({ state: 'visible' });
  }

  /**
   * Close settings modal
   */
  async closeSettings(): Promise<void> {
    await this.page.locator('#settings-modal .close-modal').first().click();
    await this.page.locator('#settings-modal').waitFor({ state: 'hidden' });
  }

  /**
   * Apply a category filter
   */
  async applyCategoryFilter(category: string): Promise<void> {
    await this.open();
    const categoryBtn = this.categoryList.locator(`button:has-text("${category}")`);
    await categoryBtn.waitFor({ state: 'visible' });
    await categoryBtn.click();
  }

  /**
   * Toggle unplayed filter
   */
  async toggleUnplayedFilter(): Promise<void> {
    await this.open();
    const unplayedCheckbox = this.page.locator('#filter-unplayed');
    await unplayedCheckbox.waitFor({ state: 'visible' });
    await unplayedCheckbox.click();
  }

  /**
   * Toggle captions filter
   */
  async toggleCaptionsFilter(): Promise<void> {
    await this.open();
    const captionsCheckbox = this.page.locator('#filter-captions');
    await captionsCheckbox.waitFor({ state: 'visible' });
    await captionsCheckbox.click();
  }

  /**
   * Set media type filter
   */
  async setMediaTypeFilter(type: 'video' | 'audio' | 'text' | 'image'): Promise<void> {
    await this.open();
    const typeBtn = this.page.locator(`button[data-type="${type}"]`);
    await typeBtn.waitFor({ state: 'visible' });
    await typeBtn.click();
  }

  /**
   * Check if sidebar is visible
   */
  async isVisible(): Promise<boolean> {
    if (await this.menuToggle.isVisible()) {
      // Mobile - check if sidebar is visible
      return await this.sidebar.isVisible();
    }
    // Desktop - sidebar is always visible
    return true;
  }

  /**
   * Get current active page/mode from URL hash
   */
  async getCurrentMode(): Promise<string> {
    const url = this.page.url();
    const hashIndex = url.indexOf('#');
    if (hashIndex === -1) return '';
    return url.substring(hashIndex + 1);
  }

  /**
   * Wait for URL to contain specific mode
   */
  async waitForMode(mode: string, timeout: number = 5000): Promise<void> {
    await this.page.waitForURL(`#${mode}`, { timeout });
  }

  /**
   * Expand a sidebar section (details/summary)
   */
  async expandSection(sectionId: string): Promise<void> {
    const section = this.page.locator(`#${sectionId}`);
    const isOpen = await section.getAttribute('open');
    if (!isOpen) {
      await section.locator('summary').click();
      await section.waitFor({ state: 'visible' });
    }
  }

  /**
   * Collapse a sidebar section
   */
  async collapseSection(sectionId: string): Promise<void> {
    const section = this.page.locator(`#${sectionId}`);
    const isOpen = await section.getAttribute('open');
    if (isOpen) {
      await section.locator('summary').click();
    }
  }

  /**
   * Wait for history button to be visible
   */
  async waitForHistoryButton(timeout: number = 5000): Promise<void> {
    await this.historyInProgressButton.waitFor({ state: 'visible', timeout });
  }

  /**
   * Click history in progress button
   */
  async clickHistoryInProgress(): Promise<void> {
    await this.open();
    await this.historyInProgressButton.click();
  }

  /**
   * Click history unplayed button
   */
  async clickHistoryUnplayed(): Promise<void> {
    await this.open();
    await this.historyUnplayedButton.click();
  }

  /**
   * Click history completed button
   */
  async clickHistoryCompleted(): Promise<void> {
    await this.open();
    await this.historyCompletedButton.click();
  }

  /**
   * Check if history button is active
   */
  async isHistoryButtonActive(button: 'inProgress' | 'unplayed' | 'completed'): Promise<boolean> {
    let btn: Locator;
    switch (button) {
      case 'inProgress':
        btn = this.historyInProgressButton;
        break;
      case 'unplayed':
        btn = this.historyUnplayedButton;
        break;
      case 'completed':
        btn = this.historyCompletedButton;
        break;
    }
    return await btn.evaluate((el) => el.classList.contains('active'));
  }

  /**
   * Check if all media button is active
   */
  async isAllMediaActive(): Promise<boolean> {
    return await this.allMediaButton.evaluate((el) => el.classList.contains('active'));
  }

  /**
   * Wait for mode in URL
   */
  async waitForModeInUrl(mode: string, timeout: number = 5000): Promise<void> {
    await this.page.waitForURL(`#${mode}`, { timeout });
  }

  /**
   * Get sidebar button by ID
   */
  getSidebarButton(id: string): Locator {
    return this.page.locator(`#${id}`);
  }

  /**
   * Get category button by text
   */
  getCategoryButtonByText(text: string): Locator {
    return this.categoryList.locator(`button:has-text("${text}")`);
  }

  /**
   * Get media type button
   */
  getMediaTypeButton(type: string): Locator {
    return this.mediaTypeList.locator(`button[data-type="${type}"]`);
  }

  /**
   * Get playlist button by name
   */
  getPlaylistButtonByName(name: string): Locator {
    return this.playlistList.locator(`.category-btn:has-text("${name}")`);
  }

  /**
   * Expand ratings section
   */
  async expandRatingsSection(): Promise<void> {
    await this.open();
    const isOpen = await this.detailsRatings.getAttribute('open');
    if (!isOpen) {
      await this.detailsRatings.locator('summary').click();
    }
  }

  /**
   * Expand media type section
   */
  async expandMediaTypeSection(): Promise<void> {
    await this.open();
    const isOpen = await this.detailsMediaType.getAttribute('open');
    if (!isOpen) {
      await this.detailsMediaType.locator('summary').click();
    }
  }

  /**
   * Expand history section
   */
  async expandHistorySection(): Promise<void> {
    await this.open();
    const isOpen = await this.detailsHistory.getAttribute('open');
    if (!isOpen) {
      await this.detailsHistory.locator('summary').click();
    }
  }

  /**
   * Expand playlists section
   */
  async expandPlaylistsSection(): Promise<void> {
    await this.open();
    const isOpen = await this.detailsPlaylists.getAttribute('open');
    if (!isOpen) {
      await this.detailsPlaylists.locator('summary').click();
    }
  }

  /**
   * Expand episodes section
   */
  async expandEpisodesSection(): Promise<void> {
    await this.open();
    const isOpen = await this.detailsEpisodes.getAttribute('open');
    if (!isOpen) {
      await this.detailsEpisodes.locator('summary').click();
    }
  }

  /**
   * Expand size section
   */
  async expandSizeSection(): Promise<void> {
    await this.open();
    const isOpen = await this.detailsSize.getAttribute('open');
    if (!isOpen) {
      await this.detailsSize.locator('summary').click();
    }
  }

  /**
   * Expand duration section
   */
  async expandDurationSection(): Promise<void> {
    await this.open();
    const isOpen = await this.detailsDuration.getAttribute('open');
    if (!isOpen) {
      await this.detailsDuration.locator('summary').click();
    }
  }

  /**
   * Get episodes slider
   */
  getEpisodesSlider(): Locator {
    return this.episodesSliderContainer.locator('input[type="range"]');
  }

  /**
   * Get size slider
   */
  getSizeSlider(): Locator {
    return this.sizeSliderContainer.locator('input[type="range"]');
  }

  /**
   * Get duration slider
   */
  getDurationSlider(): Locator {
    return this.durationSliderContainer.locator('input[type="range"]');
  }

  /**
   * Check if slider container is visible
   */
  async isSliderContainerVisible(type: 'episodes' | 'size' | 'duration'): Promise<boolean> {
    switch (type) {
      case 'episodes':
        return await this.episodesSliderContainer.isVisible();
      case 'size':
        return await this.sizeSliderContainer.isVisible();
      case 'duration':
        return await this.durationSliderContainer.isVisible();
    }
  }

  /**
   * Get filter unplayed checkbox
   */
  getFilterUnplayed(): Locator {
    return this.filterUnplayed;
  }

  /**
   * Get filter captions checkbox
   */
  getFilterCaptions(): Locator {
    return this.filterCaptions;
  }

  /**
   * Check if filter browse container is visible
   */
  async isFilterBrowseVisible(): Promise<boolean> {
    return await this.filterBrowseContainer.isVisible();
  }

  /**
   * Get channel surf button
   */
  getChannelSurfButton(): Locator {
    return this.channelSurfButton;
  }

  /**
   * Get curation button
   */
  getCurationButton(): Locator {
    return this.curationButton;
  }

  /**
   * Get trash button
   */
  getTrashButton(): Locator {
    return this.trashButton;
  }

  /**
   * Check if sidebar section exists
   */
  async sectionExists(sectionId: string): Promise<boolean> {
    return await this.page.locator(`#${sectionId}`).count() > 0;
  }

  /**
   * Get new playlist button
   */
  getNewPlaylistButton(): Locator {
    return this.newPlaylistBtn;
  }

  /**
   * Check if settings modal is open (helper method)
   */
  async isSettingsOpen(): Promise<boolean> {
    const modal = this.page.locator('#settings-modal');
    return await modal.first().isVisible();
  }
}
