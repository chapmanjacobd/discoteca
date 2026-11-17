import { Page, Locator, expect } from '@playwright/test';

/**
 * Base page object with common functionality
 * All page objects should extend this class
 */
export class BasePage {
  readonly page: Page;

  // Common locators shared across multiple pages
  readonly menuToggle: Locator;
  readonly sidebar: Locator;
  readonly toast: Locator;
  readonly queueCountBadge: Locator;
  readonly settingsButton: Locator;
  readonly settingsModal: Locator;
  readonly metadataModal: Locator;
  readonly helpModal: Locator;

  // Filter sections
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
  readonly episodesSliderContainer: Locator;
  readonly sizeSliderContainer: Locator;
  readonly durationSliderContainer: Locator;
  readonly filterUnplayed: Locator;
  readonly filterCaptions: Locator;
  readonly filterBrowseContainer: Locator;

  constructor(page: Page) {
    this.page = page;
    this.menuToggle = page.locator('#menu-toggle');
    this.sidebar = page.locator('#sidebar');
    this.toast = page.locator('#toast');
    this.queueCountBadge = page.locator('#queue-count-badge');
    this.settingsButton = page.locator('#settings-button');
    this.settingsModal = page.locator('#settings-modal');
    this.metadataModal = page.locator('#metadata-modal');
    this.helpModal = page.locator('#help-modal');

    // Filter sections
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
    this.episodesSliderContainer = page.locator('#episodes-slider-container');
    this.sizeSliderContainer = page.locator('#size-slider-container');
    this.durationSliderContainer = page.locator('#duration-slider-container');
    this.filterUnplayed = page.locator('#filter-unplayed');
    this.filterCaptions = page.locator('#filter-captions');
    this.filterBrowseContainer = page.locator('#filter-browse-col');
  }

  /**
   * Get current mode from URL hash
   */
  async getCurrentMode(): Promise<string> {
    const url = this.page.url();
    const hashIndex = url.indexOf('#');
    return hashIndex === -1 ? '' : url.substring(hashIndex + 1);
  }

  /**
   * Wait for specific mode in URL
   */
  async waitForMode(mode: string, timeout: number = 5000): Promise<void> {
    await this.page.waitForURL(`#${mode}`, { timeout });
  }

  /**
   * Wait for toast notification
   */
  async waitForToast(timeout: number = 5000): Promise<void> {
    await this.toast.waitFor({ state: 'visible', timeout });
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
   * Open sidebar on mobile
   */
  async openSidebar(): Promise<void> {
    if (!(await this.menuToggle.isVisible())) return;
    if (await this.sidebar.isVisible()) return;
    await this.menuToggle.click();
    await this.sidebar.waitFor({ state: 'visible' });
  }

  /**
   * Close sidebar on mobile
   */
  async closeSidebar(): Promise<void> {
    if (!(await this.menuToggle.isVisible())) return;
    if (!(await this.sidebar.isVisible())) return;
    // The open sidebar covers the menu toggle, so close it through the
    // overlay instead (which is what a user taps outside the drawer).
    await this.page.evaluate(() => {
      (document.getElementById('sidebar-overlay') as HTMLElement | null)?.click();
    });
    await this.sidebar.waitFor({ state: 'hidden' });
  }

  /**
   * Expand a details section
   */
  async expandSection(section: Locator): Promise<void> {
    const isOpen = await section.getAttribute('open');
    if (!isOpen) {
      await section.locator('summary').click();
      await section.waitFor({ state: 'visible' });
    }
  }

  /**
   * Collapse a details section
   */
  async collapseSection(section: Locator): Promise<void> {
    const isOpen = await section.getAttribute('open');
    if (isOpen) {
      await section.locator('summary').click();
    }
  }

  /**
   * Get slider from container
   */
  getSlider(container: Locator): Locator {
    return container.locator('input[type="range"]');
  }

  /**
   * Check if element is visible
   */
  async isVisible(locator: Locator): Promise<boolean> {
    return await locator.isVisible();
  }

  /**
   * Wait for element to be visible
   */
  async waitForVisible(locator: Locator, timeout: number = 5000): Promise<void> {
    await locator.waitFor({ state: 'visible', timeout });
  }

  /**
   * Click element if visible
   */
  async clickIfVisible(locator: Locator): Promise<void> {
    if (await locator.isVisible()) {
      await locator.click();
    }
  }

  /**
   * Get element text content
   */
  async getText(locator: Locator): Promise<string> {
    return await locator.textContent() || '';
  }

  /**
   * Get element attribute
   */
  async getAttribute(locator: Locator, attr: string): Promise<string | null> {
    return await locator.getAttribute(attr);
  }

  /**
   * Get element count
   */
  async getCount(locator: Locator): Promise<number> {
    return await locator.count();
  }

  /**
   * Assert element is visible
   */
  async expectVisible(locator: Locator): Promise<void> {
    await expect(locator).toBeVisible();
  }

  /**
   * Assert element is hidden
   */
  async expectHidden(locator: Locator): Promise<void> {
    await expect(locator).toBeHidden();
  }

  /**
   * Assert element has text
   */
  async expectHasText(locator: Locator, text: string): Promise<void> {
    await expect(locator).toContainText(text);
  }

  /**
   * Assert element has attribute
   */
  async expectAttribute(locator: Locator, attr: string, value: string): Promise<void> {
    await expect(locator).toHaveAttribute(attr, value);
  }

  /**
   * Assert element has class
   */
  async expectHasClass(locator: Locator, className: string): Promise<void> {
    await expect(locator).toHaveClass(new RegExp(className));
  }

  /**
   * Assert element does not have class
   */
  async expectNotHasClass(locator: Locator, className: string): Promise<void> {
    await expect(locator).not.toHaveClass(new RegExp(className));
  }

  /**
   * Wait for page load
   */
  async waitForPageLoad(timeout: number = 10000): Promise<void> {
    await this.page.waitForLoadState('networkidle', { timeout });
  }

  /**
   * Wait for timeout (use sparingly)
   */
  async waitForTimeout(ms: number): Promise<void> {
    await this.page.waitForTimeout(ms);
  }

  /**
   * Wait for function to return true
   */
  async waitForFunction<T>(fn: () => T, options?: { timeout?: number }): Promise<void> {
    await this.page.waitForFunction(fn, undefined, options);
  }

  /**
   * Check if sidebar section exists
   */
  async sectionExists(sectionId: string): Promise<boolean> {
    return await this.page.locator(`#${sectionId}`).count() > 0;
  }

  /**
   * Get category button by text
   */
  getCategoryButtonByText(text: string): Locator {
    return this.page.locator(`button:has-text("${text}")`);
  }

  /**
   * Get playlist button by name
   */
  getPlaylistButtonByName(name: string): Locator {
    return this.playlistList.locator(`.category-btn:has-text("${name}")`);
  }
}
