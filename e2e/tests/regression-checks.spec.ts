import { test, expect } from '../fixtures';

test.describe('Regression checks', () => {
  test.use({ readOnly: true });

  test('group view excludes subtitle sidecars', async ({ mediaPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());
    await expect(mediaPage.page.locator('#results-count')).toContainText('14 results');

    await mediaPage.page.locator('#view-group').click();
    await expect(mediaPage.page.locator('#results-container')).toHaveClass(/similarity-view/);
    await expect(mediaPage.page.locator('#results-count')).toContainText('14 files');
    await expect(mediaPage.page.locator('.media-card').filter({ hasText: '.vtt' })).toHaveCount(0);
  });

  test('leaving trash resets the hidden deleted sort', async ({ mediaPage, server }) => {
    await mediaPage.page.goto(server.getBaseUrl() + '/#mode=trash');
    await expect(mediaPage.page.locator('#sort-by')).toHaveValue('time_deleted');

    await mediaPage.page.locator('#all-media-btn').click();
    await expect(mediaPage.page.locator('#sort-by')).toHaveValue('default');
    expect(mediaPage.page.url()).not.toContain('time_deleted');

    await mediaPage.page.goto(server.getBaseUrl() + '/#mode=trash');
    await expect(mediaPage.page.locator('#sort-by')).toHaveValue('time_deleted');
    await mediaPage.page.locator('#history-btn').click();
    await expect(mediaPage.page.locator('#sort-by')).toHaveValue('time_last_played');

    await mediaPage.page.goto(server.getBaseUrl() + '/#mode=trash');
    await expect(mediaPage.page.locator('#sort-by')).toHaveValue('time_deleted');
    await mediaPage.page.locator('#du-btn').click();
    await expect(mediaPage.page.locator('#sort-by')).toHaveValue('default');
    expect(mediaPage.page.url()).not.toContain('time_deleted');
  });

  test('random sort keeps pages disjoint', async ({ mediaPage, server }) => {
    await mediaPage.page.goto(server.getBaseUrl() + '/#sort=random&limit=5');
    await expect(mediaPage.page.locator('.media-card')).toHaveCount(5);

    const seen = new Set<string>();
    for (let page = 0; page < 3; page++) {
      const paths = await mediaPage.page.locator('.media-card').evaluateAll(cards =>
        cards.map(card => (card as HTMLElement).dataset.path || '')
      );
      for (const path of paths) {
        expect(seen.has(path)).toBe(false);
        seen.add(path);
      }
      if (page < 2) {
        await mediaPage.page.locator('#next-page').click();
        await mediaPage.page.waitForTimeout(600);
      }
    }
    expect(seen.size).toBe(14);
  });

  test('rating shortcut is disabled in read-only mode', async ({ mediaPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());
    const rateRequests: string[] = [];
    mediaPage.page.on('request', request => {
      if (request.url().includes('/api/rate')) rateRequests.push(request.url());
    });

    await mediaPage.page.locator('#logo').click();
    await mediaPage.page.locator('.media-card').first().hover();
    await mediaPage.page.keyboard.press('4');

    await expect(mediaPage.page.locator('#toast')).toContainText('Read-only mode');
    expect(rateRequests).toHaveLength(0);
  });

  test('cancelling the token prompt does not reload in a loop', async ({ server, browser }) => {
    const context = await browser.newContext();
    const page = await context.newPage();
    let navigations = 0;
    let dialogs = 0;

    page.on('framenavigated', frame => {
      if (frame === page.mainFrame()) navigations++;
    });
    page.on('dialog', async dialog => {
      dialogs++;
      await dialog.dismiss();
    });

    await page.goto(server.getBaseUrl(), { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(2000);

    expect(navigations).toBeLessThan(5);
    expect(dialogs).toBeLessThanOrEqual(2);
    await expect(page.locator('#results-container')).toContainText('Authentication required');
    await context.close();
  });
});
