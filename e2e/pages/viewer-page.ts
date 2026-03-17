import { Locator } from '@playwright/test';
import { BasePage } from './base-page';

/**
 * Page Object for media viewer/player
 */
export class ViewerPage extends BasePage {
  // Player-specific locators
  readonly playerContainer: Locator;
  readonly mediaTitle: Locator;
  readonly videoElement: Locator;
  readonly audioElement: Locator;
  readonly imageElement: Locator;
  readonly closeBtn: Locator;
  readonly theatreBtn: Locator;
  readonly speedBtn: Locator;
  readonly speedMenu: Locator;
  readonly nextBtn: Locator;
  readonly prevBtn: Locator;
  readonly fullscreenBtn: Locator;
  readonly pipBtn: Locator;
  readonly streamTypeBtn: Locator;
  readonly slideshowBtn: Locator;
  readonly queueContainer: Locator;
  readonly documentModal: Locator;
  readonly documentContainer: Locator;
  readonly documentTitle: Locator;
  readonly documentFullscreenBtn: Locator;

  constructor(page: any) {
    super(page);
    this.playerContainer = page.locator('#pip-player');
    this.mediaTitle = page.locator('#media-title');
    this.videoElement = page.locator('#pip-player video');
    this.audioElement = page.locator('#pip-player audio');
    this.imageElement = page.locator('#pip-player img');
    this.closeBtn = page.locator('.close-pip, #pip-player .player-close, button:has-text("Close")').first();
    this.theatreBtn = page.locator('#pip-theatre');
    this.speedBtn = page.locator('#pip-speed');
    this.speedMenu = page.locator('#pip-speed-menu');
    this.nextBtn = page.locator('#pip-next, button:has-text("Next")').first();
    this.prevBtn = page.locator('#pip-previous, button:has-text("Previous")').first();
    this.fullscreenBtn = page.locator('#pip-fullscreen, button:has-text("Fullscreen")').first();
    this.pipBtn = page.locator('#pip-native, button:has-text("PiP")').first();
    this.streamTypeBtn = page.locator('#pip-stream-type');
    this.slideshowBtn = page.locator('#pip-slideshow');
    this.queueContainer = page.locator('#queue-container');
    this.documentModal = page.locator('#document-modal');
    this.documentContainer = page.locator('#document-container');
    this.documentTitle = page.locator('#document-title');
    this.documentFullscreenBtn = page.locator('#doc-fullscreen');
  }

  /**
   * Wait for player to be visible
   */
  async waitForPlayer(timeout: number = 10000): Promise<void> {
    await this.playerContainer.waitFor({ state: 'visible', timeout });
    // Wait for video, audio, or image element (images use img tag)
    const media = this.videoElement.or(this.audioElement).or(this.imageElement);
    await media.waitFor({ state: 'visible', timeout });
    // For video/audio, wait for readyState; for images, just wait for load event
    await this.page.waitForFunction(() => {
      const video = document.querySelector('#pip-player video') as HTMLVideoElement;
      const audio = document.querySelector('#pip-player audio') as HTMLAudioElement;
      const img = document.querySelector('#pip-player img') as HTMLImageElement;
      if (img && !video && !audio) {
        return img.complete;
      }
      const media = video || audio;
      return media && media.readyState >= 1;
    }, { timeout });
  }

  /**
   * Check if player is open
   */
  async isOpen(): Promise<boolean> {
    return await this.playerContainer.isVisible();
  }

  /**
   * Check if player is hidden
   */
  async isHidden(): Promise<boolean> {
    return await this.playerContainer.first().evaluate((el: Element) => el.classList.contains('hidden'));
  }

  /**
   * Wait for player to be hidden (for backward compatibility)
   */
  async waitForHidden(timeout: number = 10000): Promise<void> {
    await this.playerContainer.waitFor({ state: 'hidden', timeout });
  }

  /**
   * Wait for player to be hidden (alias)
   */
  async waitForPlayerHidden(timeout: number = 10000): Promise<void> {
    await this.playerContainer.waitFor({ state: 'hidden', timeout });
  }

  /**
   * Check if in theatre mode
   */
  async isInTheatreMode(): Promise<boolean> {
    const classes = await this.playerContainer.getAttribute('class');
    return classes?.includes('theatre') ?? false;
  }

  /**
   * Toggle theatre mode
   */
  async toggleTheatreMode(): Promise<void> {
    await this.theatreBtn.click();
  }

  /**
   * Enter theatre mode
   */
  async enterTheatreMode(): Promise<void> {
    if (!await this.isInTheatreMode()) {
      await this.toggleTheatreMode();
      await this.playerContainer.waitFor({ state: 'visible' });
    }
  }

  /**
   * Exit theatre mode
   */
  async exitTheatreMode(): Promise<void> {
    if (await this.isInTheatreMode()) {
      await this.toggleTheatreMode();
      await this.waitForTimeout(300);
    }
  }

  /**
   * Close player
   */
  async close(): Promise<void> {
    if (await this.closeBtn.isVisible()) {
      await this.closeBtn.click();
      await this.playerContainer.waitFor({ state: 'hidden' });
    }
  }

  /**
   * Get playback speed
   */
  async getPlaybackSpeed(): Promise<string> {
    return await this.speedBtn.textContent() || '1x';
  }

  /**
   * Set playback speed
   */
  async setPlaybackSpeed(speed: string): Promise<void> {
    await this.speedBtn.click();
    await this.speedMenu.waitFor({ state: 'visible' });
    const option = this.speedMenu.locator(`button:has-text("${speed}")`);
    await option.click();
    await this.speedMenu.waitFor({ state: 'hidden' });
  }

  /**
   * Play media
   */
  async play(): Promise<void> {
    const media = this.videoElement.or(this.audioElement);
    if (await media.count() > 0) {
      await media.evaluate((el: HTMLMediaElement) => el.play());
    }
  }

  /**
   * Pause media
   */
  async pause(): Promise<void> {
    const media = this.videoElement.or(this.audioElement);
    if (await media.count() > 0) {
      await media.evaluate((el: HTMLMediaElement) => el.pause());
    }
  }

  /**
   * Check if media is playing
   */
  async isPlaying(): Promise<boolean> {
    const media = this.videoElement.or(this.audioElement);
    if (await media.count() > 0) {
      return await media.evaluate((el: HTMLMediaElement) => !el.paused);
    }
    // For images, consider them "playing" if visible
    return await this.imageElement.isVisible();
  }

  /**
   * Get current playback time
   */
  async getCurrentTime(): Promise<number> {
    const media = this.videoElement.or(this.audioElement);
    if (await media.count() > 0) {
      return await media.evaluate((el: HTMLMediaElement) => el.currentTime);
    }
    return 0;
  }

  /**
   * Get media duration
   */
  async getDuration(): Promise<number> {
    const media = this.videoElement.or(this.audioElement);
    if (await media.count() > 0) {
      return await media.evaluate((el: HTMLMediaElement) => el.duration);
    }
    return 0;
  }

  /**
   * Seek to time
   */
  async seekTo(time: number): Promise<void> {
    const media = this.videoElement.or(this.audioElement);
    await media.evaluate((el: HTMLMediaElement, t: number) => {
      el.currentTime = t;
    }, time);
  }

  /**
   * Go to next media
   */
  async next(): Promise<void> {
    if (await this.nextBtn.isVisible()) {
      const isDisabled = await this.nextBtn.isDisabled();
      if (!isDisabled) await this.nextBtn.click();
    }
  }

  /**
   * Go to previous media
   */
  async previous(): Promise<void> {
    if (await this.prevBtn.isVisible()) {
      const isDisabled = await this.prevBtn.isDisabled();
      if (!isDisabled) await this.prevBtn.click();
    }
  }

  /**
   * Get media title
   */
  async getTitle(): Promise<string> {
    if (await this.documentModal.isVisible()) {
      return await this.documentTitle.textContent() || '';
    }
    return await this.mediaTitle.textContent() || '';
  }

  /**
   * Check if queue is visible
   */
  async isQueueVisible(): Promise<boolean> {
    return await this.queueContainer.isVisible();
  }

  /**
   * Get queue count
   */
  async getQueueCount(): Promise<number> {
    if (!await this.isQueueVisible()) return 0;
    return await this.queueContainer.locator('.queue-item').count();
  }

  /**
   * Check if slideshow is active
   */
  async isSlideshowActive(): Promise<boolean> {
    const classes = await this.slideshowBtn.getAttribute('class') || '';
    const isVisible = await this.slideshowBtn.isVisible();
    return isVisible && !classes.includes('hidden');
  }

  /**
   * Toggle slideshow
   */
  async toggleSlideshow(): Promise<void> {
    await this.slideshowBtn.click();
  }

  /**
   * Get stream type
   */
  async getStreamType(): Promise<string> {
    return await this.streamTypeBtn.textContent() || '';
  }

  /**
   * Wait for media to load
   */
  async waitForMediaLoaded(timeout: number = 10000): Promise<void> {
    // For images, wait for complete state
    const img = this.imageElement;
    if (await img.count() > 0) {
      await this.page.waitForFunction(() => {
        const img = document.querySelector('#pip-player img') as HTMLImageElement;
        return img && img.complete;
      }, { timeout });
      return;
    }
    // For video/audio, wait for readyState
    await this.page.waitForFunction(() => {
      const video = document.querySelector('#pip-player video') as HTMLVideoElement;
      const audio = document.querySelector('#pip-player audio') as HTMLAudioElement;
      const media = video || audio;
      return media && media.readyState >= 3;
    }, { timeout });
  }

  /**
   * Capture frame screenshot
   */
  async captureFrame(): Promise<Buffer> {
    return await this.playerContainer.screenshot();
  }

  /**
   * Check if media has ended
   */
  async hasEnded(): Promise<boolean> {
    const media = this.videoElement.or(this.audioElement);
    return await media.evaluate((el: HTMLMediaElement) => el.ended);
  }

  /**
   * Wait for media to end
   */
  async waitForEnd(timeout: number = 60000): Promise<void> {
    await this.page.waitForFunction(() => {
      const video = document.querySelector('#pip-player video') as HTMLVideoElement;
      const audio = document.querySelector('#pip-player audio') as HTMLAudioElement;
      const media = video || audio;
      return media && media.ended;
    }, { timeout });
  }

  /**
   * Get media element
   */
  getMediaElement(): Locator {
    return this.videoElement.or(this.audioElement).or(this.imageElement);
  }

  /**
   * Get queue items
   */
  getQueueItems(): Locator {
    return this.queueContainer.locator('.queue-item');
  }

  /**
   * Get queue item by index
   */
  getQueueItem(index: number): Locator {
    return this.getQueueItems().nth(index);
  }

  /**
   * Check if speed menu is visible
   */
  async isSpeedMenuVisible(): Promise<boolean> {
    return await this.speedMenu.isVisible();
  }

  /**
   * Get speed options
   */
  getSpeedOptions(): Locator {
    return this.speedMenu.locator('button');
  }

  /**
   * Get speed option by value
   */
  getSpeedOption(speed: string): Locator {
    return this.speedMenu.locator(`button:has-text("${speed}")`);
  }

  /**
   * Check if next button is disabled
   */
  async isNextDisabled(): Promise<boolean> {
    return await this.nextBtn.isDisabled();
  }

  /**
   * Check if previous button is disabled
   */
  async isPrevDisabled(): Promise<boolean> {
    return await this.prevBtn.isDisabled();
  }

  /**
   * Get player classes
   */
  async getPlayerClasses(): Promise<string> {
    return await this.playerContainer.first().getAttribute('class') || '';
  }

  /**
   * Check if player has class
   */
  async hasPlayerClass(className: string): Promise<boolean> {
    const classes = await this.getPlayerClasses();
    return classes.includes(className);
  }

  /**
   * Get duration formatted
   */
  async getDurationFormatted(): Promise<string> {
    const duration = await this.getDuration();
    const mins = Math.floor(duration / 60);
    const secs = Math.floor(duration % 60);
    return `${mins}:${secs.toString().padStart(2, '0')}`;
  }

  /**
   * Get current time formatted
   */
  async getCurrentTimeFormatted(): Promise<string> {
    const time = await this.getCurrentTime();
    const mins = Math.floor(time / 60);
    const secs = Math.floor(time % 60);
    return `${mins}:${secs.toString().padStart(2, '0')}`;
  }

  /**
   * Check if media is muted
   */
  async isMuted(): Promise<boolean> {
    const media = this.getMediaElement();
    return await media.evaluate((el: HTMLMediaElement) => el.muted);
  }

  /**
   * Toggle mute
   */
  async toggleMute(): Promise<void> {
    const media = this.getMediaElement();
    await media.evaluate((el: HTMLMediaElement) => {
      el.muted = !el.muted;
    });
  }

  /**
   * Set volume
   */
  async setVolume(volume: number): Promise<void> {
    const media = this.getMediaElement();
    await media.evaluate((el: HTMLMediaElement, vol: number) => {
      el.volume = Math.max(0, Math.min(1, vol));
    }, volume);
  }

  /**
   * Get volume level
   */
  async getVolume(): Promise<number> {
    const media = this.getMediaElement();
    return await media.evaluate((el: HTMLMediaElement) => el.volume);
  }

  /**
   * Check if media is looping
   */
  async isLooping(): Promise<boolean> {
    const media = this.getMediaElement();
    return await media.evaluate((el: HTMLMediaElement) => el.loop);
  }

  /**
   * Toggle loop
   */
  async toggleLoop(): Promise<void> {
    const media = this.getMediaElement();
    await media.evaluate((el: HTMLMediaElement) => {
      el.loop = !el.loop;
    });
  }

  /**
   * Get ready state
   */
  async getReadyState(): Promise<number> {
    const media = this.getMediaElement();
    return await media.evaluate((el: HTMLMediaElement) => el.readyState);
  }

  /**
   * Wait for media data
   */
  async waitForMediaData(timeout: number = 10000): Promise<void> {
    await this.page.waitForFunction(() => {
      const video = document.querySelector('#pip-player video') as HTMLVideoElement;
      const audio = document.querySelector('#pip-player audio') as HTMLAudioElement;
      const media = video || audio;
      return media && media.readyState >= 3;
    }, { timeout });
  }

  /**
   * Get buffered ranges
   */
  async getBufferedRanges(): Promise<Array<{ start: number; end: number }>> {
    const media = this.getMediaElement();
    return await media.evaluate((el: HTMLMediaElement) => {
      const ranges = [];
      for (let i = 0; i < el.buffered.length; i++) {
        ranges.push({
          start: el.buffered.start(i),
          end: el.buffered.end(i)
        });
      }
      return ranges;
    });
  }

  /**
   * Check if document modal is visible
   */
  async isDocumentModalVisible(): Promise<boolean> {
    return await this.documentModal.first().isVisible();
  }

  /**
   * Check if document modal is hidden
   */
  async isDocumentModalHidden(): Promise<boolean> {
    return await this.documentModal.first().evaluate(el => el.classList.contains('hidden'));
  }

  /**
   * Wait for document modal
   */
  async waitForDocumentModal(timeout: number = 30000): Promise<void> {
    await this.documentModal.first().waitFor({ state: 'visible', timeout });
  }

  /**
   * Close document modal
   */
  async closeDocumentModal(): Promise<void> {
    if (await this.isFullscreenActive()) {
      await this.page.keyboard.press('Escape');
      await this.waitForTimeout(300);
    }
    await this.page.keyboard.press('Escape');
    await this.documentModal.first().waitFor({ state: 'hidden', timeout: 5000 });
  }

  /**
   * Get document iframe
   */
  getDocumentIframe(): Locator {
    return this.documentContainer.locator('iframe');
  }

  /**
   * Check if metadata modal is visible
   */
  async isMetadataModalVisible(): Promise<boolean> {
    return await this.metadataModal.first().isVisible();
  }

  /**
   * Check if metadata modal is hidden
   */
  async isMetadataModalHidden(): Promise<boolean> {
    return await this.metadataModal.first().evaluate(el => el.classList.contains('hidden'));
  }

  /**
   * Check if help modal is visible
   */
  async isHelpModalVisible(): Promise<boolean> {
    return await this.helpModal.first().isVisible();
  }

  /**
   * Check if help modal is hidden
   */
  async isHelpModalHidden(): Promise<boolean> {
    return await this.helpModal.first().evaluate(el => el.classList.contains('hidden'));
  }

  /**
   * Get image element
   */
  getImageElement(): Locator {
    return this.imageElement;
  }

  /**
   * Check if image viewer is open
   */
  async isImageViewerOpen(): Promise<boolean> {
    return await this.imageElement.isVisible();
  }

  /**
   * Wait for image to load
   */
  async waitForImageLoad(timeout: number = 10000): Promise<void> {
    await this.imageElement.waitFor({ state: 'visible', timeout });
  }

  /**
   * Check if fullscreen is active
   */
  async isFullscreenActive(): Promise<boolean> {
    return await this.page.evaluate(() => !!document.fullscreenElement);
  }

  /**
   * Wait for fullscreen change
   */
  async waitForFullscreenChange(timeout: number = 5000): Promise<void> {
    await this.page.waitForFunction(() => !!document.fullscreenElement, { timeout });
  }

  /**
   * Wait for fullscreen exit
   */
  async waitForFullscreenExit(timeout: number = 5000): Promise<void> {
    await this.page.waitForFunction(() => !document.fullscreenElement, { timeout });
  }

  /**
   * Get aspect ratio style
   */
  async getAspectRatio(): Promise<string> {
    const video = this.videoElement;
    return await video.evaluate((el: HTMLVideoElement) => el.style.aspectRatio || '');
  }
}
