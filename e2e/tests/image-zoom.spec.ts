import { test, expect } from '../fixtures';

test.describe('Image Zoom and Pan', () => {
  test.use({ readOnly: true });

  test('zooms in on image with mouse wheel in fullscreen', async ({ mediaPage, viewerPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Open an image
    const imageCard = mediaPage.getFirstMediaCardByType('image');
    await imageCard.click();
    await viewerPage.waitForImageLoad();

    const img = viewerPage.getImageElement();
    await expect(img).toBeVisible();

    // Enter fullscreen first (double-click)
    await img.dblclick();
    await mediaPage.page.waitForTimeout(800);

    // Get initial transform
    const initialTransform = await img.evaluate(el => el.style.transform);

    // Simulate mouse wheel (zoom in) - only works in fullscreen
    await img.evaluate(el => {
        const event = new WheelEvent('wheel', {
            deltaY: -100, // Zoom in
            bubbles: true,
            cancelable: true
        });
        el.dispatchEvent(event);
    });

    // Check if transform changed to include scale > 1
    const zoomedTransform = await img.evaluate(el => el.style.transform);
    expect(zoomedTransform).toContain('scale(');

    // Extract scale value
    const match = zoomedTransform.match(/scale\(([\d.]+)\)/);
    if (match) {
        const scale = parseFloat(match[1]);
        expect(scale).toBeGreaterThan(1);
    }
  });

  test('double click toggles fullscreen, not zoom', async ({ mediaPage, viewerPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Open an image
    const imageCard = mediaPage.getFirstMediaCardByType('image');
    await imageCard.click();
    await viewerPage.waitForImageLoad();

    const img = viewerPage.getImageElement();

    // Double click should toggle fullscreen (not zoom)
    // Check that we're in fullscreen after double-click
    await img.dblclick();
    await mediaPage.page.waitForTimeout(800);

    const isFullscreen = await viewerPage.page.evaluate(() => !!document.fullscreenElement);
    expect(isFullscreen).toBe(true);

    // Double click again to exit fullscreen
    await img.dblclick();
    await mediaPage.page.waitForTimeout(800);

    // Should exit fullscreen
    const isNotFullscreen = await viewerPage.page.evaluate(() => !!document.fullscreenElement);
    expect(isNotFullscreen).toBe(false);
  });

  test('zoom only works in fullscreen mode', async ({ mediaPage, viewerPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Open an image
    const imageCard = mediaPage.getFirstMediaCardByType('image');
    await imageCard.click();
    await viewerPage.waitForImageLoad();

    const img = viewerPage.getImageElement();

    // Get initial transform (not in fullscreen)
    const initialTransform = await img.evaluate(el => el.style.transform);

    // Try to zoom with wheel (should NOT work outside fullscreen)
    await img.evaluate(el => {
        const event = new WheelEvent('wheel', {
            deltaY: -100, // Zoom in
            bubbles: true,
            cancelable: true
        });
        el.dispatchEvent(event);
    });

    // Transform should NOT have changed (zoom only works in fullscreen)
    const afterWheelTransform = await img.evaluate(el => el.style.transform);
    expect(afterWheelTransform).toBe(initialTransform);
  });

  test('click to zoom out when zoomed in', async ({ mediaPage, viewerPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Open an image
    const imageCard = mediaPage.getFirstMediaCardByType('image');
    await imageCard.click();
    await viewerPage.waitForImageLoad();

    const img = viewerPage.getImageElement();

    // Enter fullscreen first
    await img.dblclick();
    await mediaPage.page.waitForTimeout(800);

    // Zoom in with wheel
    await img.evaluate(el => {
        const event = new WheelEvent('wheel', {
            deltaY: -100,
            bubbles: true,
            cancelable: true
        });
        el.dispatchEvent(event);
    });
    await mediaPage.page.waitForTimeout(300);

    // Verify zoomed in
    const zoomedTransform = await img.evaluate(el => el.style.transform);
    expect(zoomedTransform).toContain('scale(');

    // Click to zoom out (has 250ms delay to distinguish from double-click)
    await img.click();
    await mediaPage.page.waitForTimeout(400);

    // Should be reset
    const resetTransform = await img.evaluate(el => el.style.transform);
    expect(resetTransform).toBe('');
  });

  test('image has drag and selection disabled', async ({ mediaPage, viewerPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Open an image
    const imageCard = mediaPage.getFirstMediaCardByType('image');
    await imageCard.click();
    await viewerPage.waitForImageLoad();

    const img = viewerPage.getImageElement();

    // Check draggable attribute is false
    const draggable = await img.evaluate(el => (el as HTMLImageElement).draggable);
    expect(draggable).toBe(false);

    // Check user-select style
    const userSelect = await img.evaluate(el => el.style.userSelect);
    expect(userSelect).toBe('none');
  });
});
