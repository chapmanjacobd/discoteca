import { test, expect } from '../fixtures';

/**
 * E2E tests for the complete categorization workflow
 * Tests creating custom categories, managing keywords, and running categorization
 */
test.describe('Categorization Workflow - Full Process', () => {
  test.use({ readOnly: false });

  test.beforeEach(async ({ page }) => {
    page.on('console', msg => {
      if (msg.type() === 'error') {
        console.error('BROWSER ERROR:', msg.text());
      }
    });
  });

  test('completes full categorization workflow: create category, add keywords, run categorization', async ({ page, server }) => {
    console.log('=== Starting Full Categorization Workflow Test ===');

    // Step 1: Navigate to the site and wait for media to load
    await page.goto(server.getBaseUrl());
    await page.waitForSelector('.media-card', { timeout: 10000 });

    const initialMediaCount = await page.locator('.media-card').count();
    console.log(`Found ${initialMediaCount} media items`);
    expect(initialMediaCount).toBeGreaterThan(0);

    // Step 2: Navigate to Curation Tool
    console.log('Navigating to Curation Tool...');
    await page.evaluate(() => {
      window.location.hash = 'mode=curation';
    });
    await page.waitForTimeout(2000);

    // Wait for curation view to load
    await page.waitForSelector('.curation-view', { timeout: 10000 });
    await page.waitForSelector('.curation-cat-list', { timeout: 5000 });
    console.log('Curation Tool loaded successfully');

    // Step 3: Find potential keywords FIRST (before creating any categories)
    console.log('Finding potential keywords...');
    const findKeywordsBtn = page.locator('#find-keywords-btn');
    let keywordToAdd = '';
    
    if (await findKeywordsBtn.count() > 0) {
      await findKeywordsBtn.click();
      await page.waitForTimeout(3000);

      // Check if suggestions appeared
      const suggestions = page.locator('.suggestion-tag');
      const suggestionCount = await suggestions.count();
      console.log(`Found ${suggestionCount} keyword suggestions`);
      expect(suggestionCount).toBeGreaterThan(0);

      // Get the first suggestion to use later
      const firstSuggestion = suggestions.first();
      keywordToAdd = await firstSuggestion.getAttribute('data-word');
      console.log(`Will use keyword: ${keywordToAdd}`);
    }

    // Step 4: Create a custom category with the suggested keyword
    console.log('Creating custom category...');
    // Handle TWO prompts: first for category name, second for first keyword
    let promptCount = 0;
    page.on('dialog', async dialog => {
      if (dialog.type() === 'prompt') {
        promptCount++;
        if (promptCount === 1) {
          // First prompt: category name
          await dialog.accept('My Custom Category');
        } else if (promptCount === 2) {
          // Second prompt: first keyword (use the suggestion we found)
          await dialog.accept(keywordToAdd || 'clip');
        }
      }
    });
    const newCategoryBtn = page.locator('#new-category-btn');
    await newCategoryBtn.click();
    await page.waitForTimeout(2000);

    // Verify custom category was created
    const customCategory = page.locator('.curation-cat-card[data-category="My Custom Category"]');
    const customCategoryExists = await customCategory.count() > 0;
    console.log(`Custom category created: ${customCategoryExists}`);
    expect(customCategoryExists).toBe(true);

    // Step 5: Find more keyword suggestions (refresh after creating category)
    console.log('Finding more keyword suggestions...');
    if (await findKeywordsBtn.count() > 0) {
      await findKeywordsBtn.click();
      await page.waitForTimeout(3000);

      const suggestions = page.locator('.suggestion-tag');
      const suggestionCount = await suggestions.count();
      console.log(`Found ${suggestionCount} more keyword suggestions`);

      // Add another keyword from suggestions if available
      if (suggestionCount > 0) {
        console.log('Adding keyword to custom category...');
        const firstSuggestion = suggestions.first();
        const anotherKeyword = await firstSuggestion.getAttribute('data-word');
        console.log(`Adding keyword: ${anotherKeyword}`);

        if (anotherKeyword) {
          // Click to add keyword to category (via prompt)
          page.on('dialog', async dialog => {
            if (dialog.type() === 'prompt') {
              await dialog.accept('My Custom Category');
            }
          });
          await firstSuggestion.click();
          await page.waitForTimeout(1000);

          // Verify keyword was added to custom category
          const customCategoryCard = page.locator('.curation-cat-card[data-category="My Custom Category"]');
          const keywordInCategory = customCategoryCard.locator(`.curation-tag[data-keyword="${anotherKeyword}"]`);
          const keywordAdded = await keywordInCategory.count() > 0;
          console.log(`Keyword added to category: ${keywordAdded}`);
          expect(keywordAdded).toBe(true);
        }
      }
    }

    // Step 6: Run categorization
    console.log('Running categorization...');
    const runCategorizeBtn = page.locator('#run-auto-categorize');
    await runCategorizeBtn.click();
    await page.waitForTimeout(5000);

    // Check for success toast or message
    const toast = page.locator('.toast');
    if (await toast.count() > 0) {
      const toastText = await toast.textContent();
      console.log(`Categorization result: ${toastText}`);
      expect(toastText).toContain('categorized');
    }

    console.log('=== Full Categorization Workflow Test Complete ===');
  });

  test('creates and deletes a custom category', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Navigate to Curation Tool
    await page.evaluate(() => {
      window.location.hash = 'mode=curation';
    });
    await page.waitForTimeout(2000);
    await page.waitForSelector('.curation-view', { timeout: 10000 });

    // Create custom category
    const newCategoryBtn = page.locator('#new-category-btn');
    page.on('dialog', async dialog => {
      if (dialog.type() === 'prompt') {
        await dialog.accept('Test Category To Delete');
      }
    });
    await newCategoryBtn.click();
    await page.waitForTimeout(1000);

    // Verify category exists
    const categoryCard = page.locator('.curation-cat-card[data-category="Test Category To Delete"]');
    await expect(categoryCard).toBeVisible();

    // Delete the category
    const deleteBtn = categoryCard.locator('.delete-cat-btn');
    if (await deleteBtn.count() > 0) {
      page.on('dialog', async dialog => {
        if (dialog.type() === 'confirm') {
          await dialog.accept();
        }
      });
      await deleteBtn.click();
      await page.waitForTimeout(1000);

      // Verify category is deleted
      const categoryExists = await categoryCard.count() > 0;
      expect(categoryExists).toBe(false);
    }
  });

  test('adds keyword to existing category', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Navigate to Curation Tool
    await page.evaluate(() => {
      window.location.hash = 'mode=curation';
    });
    await page.waitForTimeout(2000);
    await page.waitForSelector('.curation-view', { timeout: 10000 });

    // Create a custom category first (don't use default categories)
    let promptCount = 0;
    page.on('dialog', async dialog => {
      if (dialog.type() === 'prompt') {
        promptCount++;
        if (promptCount === 1) {
          await dialog.accept('Test Category');
        } else if (promptCount === 2) {
          await dialog.accept('test');
        }
      }
    });
    const newCategoryBtn = page.locator('#new-category-btn');
    await newCategoryBtn.click();
    await page.waitForTimeout(2000);

    // Find keywords
    const findKeywordsBtn = page.locator('#find-keywords-btn');
    if (await findKeywordsBtn.count() > 0) {
      await findKeywordsBtn.click();
      await page.waitForTimeout(3000);

      const suggestions = page.locator('.suggestion-tag');
      const suggestionCount = await suggestions.count();

      if (suggestionCount > 0) {
        const firstSuggestion = suggestions.first();
        const keywordText = await firstSuggestion.getAttribute('data-word');

        if (keywordText) {
          // Add keyword to Test Category
          page.on('dialog', async dialog => {
            if (dialog.type() === 'prompt') {
              await dialog.accept('Test Category');
            }
          });
          await firstSuggestion.click();
          await page.waitForTimeout(1000);

          // Verify keyword was added
          const categoryCard = page.locator('.curation-cat-card[data-category="Test Category"]');
          const keywordTag = categoryCard.locator(`.curation-tag[data-keyword="${keywordText}"]`);
          await expect(keywordTag).toBeVisible();
        }
      }
    }
  });

  test('removes keyword from category', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Navigate to Curation Tool
    await page.evaluate(() => {
      window.location.hash = 'mode=curation';
    });
    await page.waitForTimeout(2000);
    await page.waitForSelector('.curation-view', { timeout: 10000 });

    // Add default categories
    const addDefaultBtn = page.locator('#add-default-cats');
    if (await addDefaultBtn.count() > 0) {
      await addDefaultBtn.click();
      await page.waitForTimeout(2000);
    }

    // Find a category with keywords
    const categories = page.locator('.curation-cat-card');
    const categoryCount = await categories.count();
    
    if (categoryCount > 0) {
      const firstCategory = categories.first();
      const keywords = firstCategory.locator('.curation-tag.existing-keyword');
      const keywordCount = await keywords.count();
      
      if (keywordCount > 0) {
        const firstKeyword = keywords.first();
        const keywordText = await firstKeyword.getAttribute('data-keyword');
        const categoryName = await firstCategory.getAttribute('data-category');
        
        console.log(`Removing keyword "${keywordText}" from "${categoryName}"`);
        
        // Click remove button
        const removeBtn = firstKeyword.locator('.remove-kw');
        await removeBtn.click();
        await page.waitForTimeout(1000);

        // Verify keyword was removed
        const keywordExists = await firstKeyword.count() > 0;
        expect(keywordExists).toBe(false);
      }
    }
  });

  test('adds default categories', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Navigate to Curation Tool
    await page.evaluate(() => {
      window.location.hash = 'mode=curation';
    });
    await page.waitForTimeout(2000);
    await page.waitForSelector('.curation-view', { timeout: 10000 });

    // Count categories before
    const categoriesBefore = page.locator('.curation-cat-card');
    const countBefore = await categoriesBefore.count();
    console.log(`Categories before: ${countBefore}`);

    // Add default categories
    const addDefaultBtn = page.locator('#add-default-cats');
    if (await addDefaultBtn.count() > 0 && await addDefaultBtn.isVisible()) {
      // Handle confirm dialog BEFORE clicking
      page.once('dialog', async dialog => {
        if (dialog.type() === 'confirm') {
          await dialog.accept();
        }
      });
      await addDefaultBtn.click();
      // Wait for categories to be fetched and rendered
      await page.waitForTimeout(3000);

      // Count categories after
      const countAfter = await categoriesBefore.count();
      console.log(`Categories after: ${countAfter}`);

      // Should have more categories
      expect(countAfter).toBeGreaterThan(countBefore);
    }
  });

  test('finds potential keywords from uncategorized files', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Navigate to Curation Tool
    await page.evaluate(() => {
      window.location.hash = 'mode=curation';
    });
    await page.waitForTimeout(2000);
    await page.waitForSelector('.curation-view', { timeout: 10000 });

    // Click find keywords button
    const findKeywordsBtn = page.locator('#find-keywords-btn');
    if (await findKeywordsBtn.count() > 0) {
      await findKeywordsBtn.click();
      await page.waitForTimeout(3000);

      // Check for suggestions
      const suggestionsArea = page.locator('.curation-col').last();
      const suggestions = suggestionsArea.locator('.suggestion-tag');
      const count = await suggestions.count();
      
      console.log(`Found ${count} keyword suggestions`);
      
      // Each suggestion should have data-word attribute
      if (count > 0) {
        const firstSuggestion = suggestions.first();
        const word = await firstSuggestion.getAttribute('data-word');
        expect(word).toBeTruthy();
        console.log(`First suggestion: ${word}`);
      }
    }
  });

  test('navigates back from curation view', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Navigate to Curation Tool
    await page.evaluate(() => {
      window.location.hash = 'mode=curation';
    });
    await page.waitForTimeout(2000);
    await page.waitForSelector('.curation-view', { timeout: 10000 });

    // Click back button
    const backBtn = page.locator('#curation-back-btn');
    await backBtn.click();
    await page.waitForTimeout(1000);

    // Should return to search view
    const hash = await page.evaluate(() => window.location.hash);
    expect(hash).not.toContain('curation');
    
    // Media cards should be visible
    await page.waitForSelector('.media-card', { timeout: 5000 });
    const mediaCount = await page.locator('.media-card').count();
    expect(mediaCount).toBeGreaterThan(0);
  });

  test('handles categorization with no uncategorized files', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Navigate to Curation Tool
    await page.evaluate(() => {
      window.location.hash = 'mode=curation';
    });
    await page.waitForTimeout(2000);
    await page.waitForSelector('.curation-view', { timeout: 10000 });

    // Try to find keywords (may show "no keywords" message)
    const findKeywordsBtn = page.locator('#find-keywords-btn');
    if (await findKeywordsBtn.count() > 0) {
      await findKeywordsBtn.click();
      await page.waitForTimeout(3000);

      // Should not crash, may show "no keywords" message
      const suggestionsArea = page.locator('.curation-col').last();
      await expect(suggestionsArea).toBeVisible();
    }
  });

  test('verifies category structure and UI elements', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Navigate to Curation Tool
    await page.evaluate(() => {
      window.location.hash = 'mode=curation';
    });
    await page.waitForTimeout(2000);
    await page.waitForSelector('.curation-view', { timeout: 10000 });

    // Verify UI structure
    const curationHeader = page.locator('.curation-header');
    await expect(curationHeader).toBeVisible();

    const categoriesCol = page.locator('.curation-col').first();
    await expect(categoriesCol).toBeVisible();

    const runCategorizeBtn = page.locator('#run-auto-categorize');
    await expect(runCategorizeBtn).toBeVisible();

    const newCategoryBtn = page.locator('#new-category-btn');
    await expect(newCategoryBtn).toBeVisible();

    const addDefaultBtn = page.locator('#add-default-cats');
    await expect(addDefaultBtn).toBeVisible();
  });

  test('drag and drop keyword to category (if supported)', async ({ page, server }) => {
    await page.goto(server.getBaseUrl());
    await page.waitForSelector('.media-card', { timeout: 10000 });

    // Navigate to Curation Tool
    await page.evaluate(() => {
      window.location.hash = 'mode=curation';
    });
    await page.waitForTimeout(2000);
    await page.waitForSelector('.curation-view', { timeout: 10000 });

    // Add default categories
    const addDefaultBtn = page.locator('#add-default-cats');
    if (await addDefaultBtn.count() > 0) {
      await addDefaultBtn.click();
      await page.waitForTimeout(2000);
    }

    // Find keywords
    const findKeywordsBtn = page.locator('#find-keywords-btn');
    if (await findKeywordsBtn.count() > 0) {
      await findKeywordsBtn.click();
      await page.waitForTimeout(3000);

      const suggestions = page.locator('.suggestion-tag');
      if (await suggestions.count() > 0) {
        const firstSuggestion = suggestions.first();
        const categories = page.locator('.curation-cat-card');
        
        if (await categories.count() > 0) {
          const firstCategory = categories.first();
          
          // Try drag and drop (may not work in all test environments)
          try {
            await firstSuggestion.dragTo(firstCategory);
            await page.waitForTimeout(1000);
            console.log('Drag and drop completed');
          } catch (e) {
            console.log('Drag and drop not supported in this environment, skipping');
          }
        }
      }
    }
  });
});
