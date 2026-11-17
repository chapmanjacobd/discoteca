import { Locator } from '@playwright/test';
import { BasePage } from './base-page';

/**
 * Page Object for sidebar navigation and filters
 */
export class SidebarPage extends BasePage {
  // Sidebar-specific locators
  readonly duButton: Locator;
  readonly captionsButton: Locator;
  readonly curationButton: Locator;
  readonly channelSurfButton: Locator;
  readonly trashButton: Locator;
  readonly historyInProgressButton: Locator;
  readonly historyUnplayedButton: Locator;
  readonly historyCompletedButton: Locator;
  readonly categoryList: Locator;
  readonly allMediaBtn: Locator;
  readonly newPlaylistBtn: Locator;
  readonly duToolbar: Locator;

  constructor(page: any) {
    super(page);
    this.duToolbar = page.locator('#du-toolbar');
    this.allMediaBtn = page.locator('#all-media-btn');
    this.newPlaylistBtn = page.locator('#new-playlist-btn');
    this.duButton = page.locator('#du-btn');
    this.captionsButton = page.locator('#captions-btn');
    this.curationButton = page.locator('#curation-btn');
    this.channelSurfButton = page.locator('#channel-surf-btn');
    this.trashButton = page.locator('#trash-btn');
    this.historyInProgressButton = page.locator('#history-in-progress-btn');
    this.historyUnplayedButton = page.locator('#history-unplayed-btn');
    this.historyCompletedButton = page.locator('#history-completed-btn');
    this.categoryList = page.locator('#category-list');
  }

  /**
   * Navigate to Disk Usage view
   */
  async openDiskUsage(): Promise<void> {
    await this.openSidebar();
    await this.duButton.click();
    await this.duToolbar.waitFor({ state: 'visible' });
  }

  /**
   * Navigate to Captions view
   */
  async openCaptions(): Promise<void> {
    await this.openSidebar();
    await this.captionsButton.click();
    await this.page.locator('.caption-media-card').first().waitFor({ state: 'visible' });
  }

  /**
   * Navigate to Curation view
   */
  async openCuration(): Promise<void> {
    await this.openSidebar();
    await this.curationButton.click();
  }

  /**
   * Navigate to Trash view
   */
  async openTrash(): Promise<void> {
    await this.openSidebar();
    await this.trashButton.click();
  }

  /**
   * Navigate to History - In Progress
   */
  async openHistoryInProgress(): Promise<void> {
    await this.openSidebar();
    await this.ensureHistoryDetailsOpen();
    await this.historyInProgressButton.waitFor({ state: 'visible' });
    await this.historyInProgressButton.click();
  }

  /**
   * Navigate to History - Unplayed
   */
  async openHistoryUnplayed(): Promise<void> {
    await this.openSidebar();
    await this.ensureHistoryDetailsOpen();
    await this.historyUnplayedButton.waitFor({ state: 'visible' });
    await this.historyUnplayedButton.click();
  }

  /**
   * Navigate to History - Completed
   */
  async openHistoryCompleted(): Promise<void> {
    await this.openSidebar();
    await this.ensureHistoryDetailsOpen();
    await this.historyCompletedButton.waitFor({ state: 'visible' });
    await this.historyCompletedButton.click();
  }

  /**
   * Expand the History section when the sidebar starts with it collapsed.
   */
  private async ensureHistoryDetailsOpen(): Promise<void> {
    const details = this.page.locator('#details-history');
    if (await details.count() === 0) return;
    const isOpen = await details.evaluate((el) => (el as HTMLDetailsElement).open);
    if (!isOpen) {
      await details.locator('summary').click();
    }
  }

  /**
   * Navigate to All Media
   */
  async openAllMedia(): Promise<void> {
    await this.openSidebar();
    await this.allMediaBtn.waitFor({ state: 'visible' });
    await this.allMediaBtn.click();
  }

  /**
   * Open settings modal
   */
  async openSettings(): Promise<void> {
    await this.settingsButton.click();
    await this.settingsModal.waitFor({ state: 'visible' });
  }

  /**
   * Close settings modal
   */
  async closeSettings(): Promise<void> {
    await this.page.locator('#settings-modal .close-modal').first().click();
    await this.page.locator('#settings-modal').waitFor({ state: 'hidden' });
  }

  /**
   * Apply category filter
   */
  async applyCategoryFilter(category: string): Promise<void> {
    await this.openSidebar();
    const categoryBtn = this.categoryList.locator(`button:has-text("${category}")`);
    await categoryBtn.waitFor({ state: 'visible' });
    await categoryBtn.click();
  }

  /**
   * Toggle unplayed filter
   */
  async toggleUnplayedFilter(): Promise<void> {
    await this.openSidebar();
    await this.filterUnplayed.waitFor({ state: 'visible' });
    await this.filterUnplayed.click();
  }

  /**
   * Toggle captions filter
   */
  async toggleCaptionsFilter(): Promise<void> {
    await this.openSidebar();
    await this.filterCaptions.waitFor({ state: 'visible' });
    await this.filterCaptions.click();
  }

  /**
   * Set media type filter
   */
  async setMediaTypeFilter(type: 'video' | 'audio' | 'text' | 'image'): Promise<void> {
    await this.openSidebar();
    const typeBtn = this.page.locator(`button[data-media_type="${type}"]`);
    await typeBtn.waitFor({ state: 'visible' });
    await typeBtn.click();
  }

  /**
   * Check if sidebar is visible
   */
  async isVisible(): Promise<boolean> {
    if (await this.menuToggle.isVisible()) {
      return await this.sidebar.isVisible();
    }
    return true;
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
    return await this.allMediaBtn.evaluate((el) => el.classList.contains('active'));
  }

  /**
   * Click history in progress
   */
  async clickHistoryInProgress(): Promise<void> {
    await this.openSidebar();
    await this.historyInProgressButton.click();
  }

  /**
   * Click history unplayed
   */
  async clickHistoryUnplayed(): Promise<void> {
    await this.openSidebar();
    await this.historyUnplayedButton.click();
  }

  /**
   * Click history completed
   */
  async clickHistoryCompleted(): Promise<void> {
    await this.openSidebar();
    await this.historyCompletedButton.click();
  }

  /**
   * Wait for history button
   */
  async waitForHistoryButton(timeout: number = 5000): Promise<void> {
    await this.historyInProgressButton.waitFor({ state: 'visible', timeout });
  }

  /**
   * Expand ratings section
   */
  async expandRatingsSection(): Promise<void> {
    await this.openSidebar();
    await this.expandSection(this.detailsRatings);
  }

  /**
   * Expand media type section
   */
  async expandMediaTypeSection(): Promise<void> {
    await this.openSidebar();
    await this.expandSection(this.detailsMediaType);
  }

  /**
   * Expand history section
   */
  async expandHistorySection(): Promise<void> {
    await this.openSidebar();
    await this.expandSection(this.detailsHistory);
  }

  /**
   * Expand playlists section
   */
  async expandPlaylistsSection(): Promise<void> {
    await this.openSidebar();
    await this.expandSection(this.detailsPlaylists);
  }

  /**
   * Expand episodes section
   */
  async expandEpisodesSection(): Promise<void> {
    await this.openSidebar();
    await this.expandSection(this.detailsEpisodes);
  }

  /**
   * Expand size section
   */
  async expandSizeSection(): Promise<void> {
    await this.openSidebar();
    await this.expandSection(this.detailsSize);
  }

  /**
   * Expand duration section
   */
  async expandDurationSection(): Promise<void> {
    await this.openSidebar();
    await this.expandSection(this.detailsDuration);
  }

  /**
   * Get episodes slider
   */
  getEpisodesSlider(): Locator {
    return this.getSlider(this.episodesSliderContainer);
  }

  /**
   * Get size slider
   */
  getSizeSlider(): Locator {
    return this.getSlider(this.sizeSliderContainer);
  }

  /**
   * Get duration slider
   */
  getDurationSlider(): Locator {
    return this.getSlider(this.durationSliderContainer);
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
   * Check if filter browse is visible
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
   * Open sidebar on mobile (alias for backward compatibility)
   */
  async open(): Promise<void> {
    await this.openSidebar();
  }

  /**
   * Close sidebar on mobile (alias for backward compatibility)
   */
  async close(): Promise<void> {
    await this.closeSidebar();
  }

  /**
   * Get all media button (alias for backward compatibility)
   */
  get allMediaButton(): Locator {
    return this.allMediaBtn;
  }

  /**
   * Get new playlist button
   */
  getNewPlaylistButton(): Locator {
    return this.newPlaylistBtn;
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
    return this.mediaTypeList.locator(`button[data-media_type="${type}"]`);
  }

  /**
   * Check if settings modal is open
   */
  async isSettingsOpen(): Promise<boolean> {
    const modal = this.page.locator('#settings-modal');
    return await modal.first().isVisible();
  }
}
