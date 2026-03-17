import { test, expect } from '../fixtures';

test.describe('Image Zoom and Pan', () => {
  test.use({ readOnly: true });

  test('zooms in on image with mouse wheel', async ({ mediaPage, viewerPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Open an image
    const imageCard = mediaPage.getFirstMediaCardByType('image');
    await imageCard.click();
    await viewerPage.waitForImageLoad();

    const img = viewerPage.getImageElement();
    await expect(img).toBeVisible();

    // Get initial transform
    const initialTransform = await img.evaluate(el => el.style.transform);
    
    // Simulate mouse wheel (zoom in)
    // Dispatching a wheel event directly on the element
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

  test('double click toggles zoom', async ({ mediaPage, viewerPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Open an image
    const imageCard = mediaPage.getFirstMediaCardByType('image');
    await imageCard.click();
    await viewerPage.waitForImageLoad();

    const img = viewerPage.getImageElement();
    
    // Double click to zoom in
    await img.dblclick();
    
    const zoomedTransform = await img.evaluate(el => el.style.transform);
    expect(zoomedTransform).toContain('scale(3)'); // My impl uses 3 for dblclick

    // Double click to zoom out
    await img.dblclick();
    
    const resetTransform = await img.evaluate(el => el.style.transform);
    expect(resetTransform).toContain('scale(1)');
  });

  test('panning works when zoomed in', async ({ mediaPage, viewerPage, server }) => {
    await mediaPage.goto(server.getBaseUrl());

    // Open an image
    const imageCard = mediaPage.getFirstMediaCardByType('image');
    await imageCard.click();
    await viewerPage.waitForImageLoad();

    const img = viewerPage.getImageElement();
    
    // Zoom in first
    await img.dblclick();
    
    // Get position before pan
    const beforePanTransform = await img.evaluate(el => el.style.transform);

    // Drag to pan
    const box = await img.boundingBox();
    if (box) {
        await mediaPage.page.mouse.move(box.x + box.width / 2, box.y + box.height / 2);
        await mediaPage.page.mouse.down();
        await mediaPage.page.mouse.move(box.x + box.width / 2 + 50, box.y + box.height / 2 + 50);
        await mediaPage.page.mouse.up();
    }

    // Check if translate changed
    const afterPanTransform = await img.evaluate(el => el.style.transform);
    expect(afterPanTransform).toContain('translate(');
    expect(afterPanTransform).not.toBe(beforePanTransform);
  });
});
