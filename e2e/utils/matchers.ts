import { expect, Locator, Page } from '@playwright/test';

/**
 * Custom Playwright matchers for Discoteca E2E tests
 */

/**
 * Assert that a locator has a specific number of visible items
 */
export async function toHaveMediaCount(locator: Locator, expected: number): Promise<{ pass: boolean; message: () => string }> {
  const actual = await locator.count();
  const pass = actual === expected;
  
  return {
    pass,
    message: () => `Expected media count to be ${expected}, but got ${actual}`
  };
}

/**
 * Assert that the page is in a specific mode (based on URL hash)
 */
export async function toBeInMode(page: Page, expectedMode: string): Promise<{ pass: boolean; message: () => string }> {
  const url = page.url();
  const hashIndex = url.indexOf('#');
  const currentMode = hashIndex === -1 ? '' : url.substring(hashIndex + 1);
  const pass = currentMode === expectedMode;
  
  return {
    pass,
    message: () => `Expected mode to be "${expectedMode}", but got "${currentMode}" (URL: ${url})`
  };
}

/**
 * Assert that a JSON response has expected structure
 */
export async function toHaveJsonOutput(response: Response, expectedKeys: string[]): Promise<{ pass: boolean; message: () => string }> {
  try {
    const json = await response.json();
    const missingKeys = expectedKeys.filter(key => !(key in json));
    const pass = missingKeys.length === 0;
    
    return {
      pass,
      message: () => pass 
        ? 'JSON response has expected keys'
        : `JSON response missing keys: ${missingKeys.join(', ')}`
    };
  } catch (e) {
    return {
      pass: false,
      message: () => `Response is not valid JSON: ${e}`
    };
  }
}

/**
 * Assert that a media card has progress indicator
 */
export async function toHaveProgress(locator: Locator, expectedProgress: number, tolerance: number = 5): Promise<{ pass: boolean; message: () => string }> {
  const progressBar = locator.locator('.progress-bar');
  const isVisible = await progressBar.isVisible();
  
  if (!isVisible) {
    return {
      pass: false,
      message: () => 'Progress bar not found'
    };
  }
  
  const width = await progressBar.evaluate((el: HTMLElement) => {
    const style = window.getComputedStyle(el);
    return parseFloat(style.width) || 0;
  });
  
  // Convert percentage to approximate progress
  const actualProgress = width; // Assuming width is percentage
  const pass = Math.abs(actualProgress - expectedProgress) <= tolerance;
  
  return {
    pass,
    message: () => `Expected progress to be ~${expectedProgress}%, but got ${actualProgress}%`
  };
}

/**
 * Assert that player is in a specific state
 */
export async function toBePlaying(locator: Locator): Promise<{ pass: boolean; message: () => string }> {
  const isPlaying = await locator.evaluate((el: HTMLMediaElement) => !el.paused);
  
  return {
    pass: isPlaying,
    message: () => isPlaying 
      ? 'Media is playing' 
      : 'Media is not playing'
  };
}

/**
 * Assert that player is paused
 */
export async function toBePaused(locator: Locator): Promise<{ pass: boolean; message: () => string }> {
  const isPaused = await locator.evaluate((el: HTMLMediaElement) => el.paused);
  
  return {
    pass: isPaused,
    message: () => isPaused 
      ? 'Media is paused' 
      : 'Media is not paused'
  };
}

/**
 * Extend Playwright's expect with custom matchers
 */
export const customExpect = expect.extend({
  toHaveMediaCount,
  toBeInMode,
  toHaveJsonOutput,
  toHaveProgress,
  toBePlaying,
  toBePaused,
});

// Re-export expect with custom matchers
export { expect };

/**
 * Helper: Wait for network response with specific URL pattern
 */
export async function waitForApiResponse(page: Page, urlPattern: string, timeout: number = 5000): Promise<Response> {
  const [response] = await Promise.all([
    page.waitForResponse(resp => resp.url().includes(urlPattern), { timeout }),
    // Trigger the request if needed
  ]);
  return response;
}

/**
 * Helper: Wait for API request with specific URL pattern
 */
export async function waitForApiRequest(page: Page, urlPattern: string, timeout: number = 5000): Promise<void> {
  await page.waitForRequest(req => req.url().includes(urlPattern), { timeout });
}

/**
 * Helper: Assert that element has specific data attribute
 */
export async function toHaveDataAttribute(locator: Locator, attr: string, value: string): Promise<{ pass: boolean; message: () => string }> {
  const actualValue = await locator.getAttribute(attr);
  const pass = actualValue === value;
  
  return {
    pass,
    message: () => `Expected ${attr} to be "${value}", but got "${actualValue}"`
  };
}

/**
 * Helper: Assert that toast notification appears with specific message
 */
export async function toHaveToast(page: Page, expectedText: string, timeout: number = 5000): Promise<{ pass: boolean; message: () => string }> {
  try {
    const toast = page.locator('#toast');
    await toast.waitFor({ state: 'visible', timeout });
    const toastText = await toast.textContent();
    const pass = toastText?.includes(expectedText) ?? false;
    
    return {
      pass,
      message: () => pass 
        ? 'Toast message found' 
        : `Expected toast to contain "${expectedText}", but got "${toastText}"`
    };
  } catch (e) {
    return {
      pass: false,
      message: () => `Toast did not appear within ${timeout}ms`
    };
  }
}

/**
 * Helper: Assert that no error toast appears
 */
export async function toHaveNoErrorToast(page: Page, timeout: number = 2000): Promise<{ pass: boolean; message: () => string }> {
  const toast = page.locator('#toast');
  const isVisible = await toast.isVisible();
  
  if (isVisible) {
    const toastText = await toast.textContent();
    const isError = toastText?.includes('Error') || toastText?.includes('Failed') || toastText?.includes('⚠️');
    
    if (isError) {
      return {
        pass: false,
        message: () => `Error toast appeared: "${toastText}"`
      };
    }
  }
  
  return {
    pass: true,
    message: () => 'No error toast appeared'
  };
}
