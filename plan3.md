# Plan 3: E2E Testing Framework Modernization

## Goal
Transition the Playwright E2E suite from brittle, timeout-heavy tests to a robust Page Object Model (POM) architecture with custom assertions.

## Context
Current tests (e.g., `e2e/tests/navigation.spec.ts`) rely on `page.waitForTimeout(1000)` and hardcoded CSS selectors. This leads to flaky tests and makes UI changes difficult to maintain.

## Tasks

### 1. Implement Page Object Model (POM)
- Create `e2e/pages/` directory.
- **MediaPage:** Methods for `openItem(title)`, `play()`, `setRating(n)`.
- **SidebarPage:** Methods for `switchMode(mode)`, `applyFilter(name, value)`.
- **ViewerPage:** Methods for `next()`, `previous()`, `close()`.

### 2. Create Custom Playwright Matchers
- **File:** `e2e/fixtures.ts` or a new `e2e/utils/matchers.ts`.
- **Implement:**
    - `toHaveMediaCount(count)`: Verifies the number of visible cards.
    - `toBeInMode(mode)`: Verifies URL and UI state (e.g., `#mode=du`).
    - `toHaveJsonOutput()`: Specific to CLI tests.

### 3. Replace Hard Timeouts
- Audit `e2e/tests/` and replace `waitForTimeout` with:
    - `expect(locator).toBeVisible()`
    - `page.waitForResponse(url)`
    - `page.waitForFunction(fn)`

## Verification
- Run `npx playwright test --project=desktop` and ensure zero flakes over 5 consecutive runs.
