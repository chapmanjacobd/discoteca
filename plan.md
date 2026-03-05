# Work Plan

## 1. Testing Priorities
- [ ] **`internal/commands/serve.go` (Handlers)**:
    - Many API handlers are large and handle multiple parameters. More granular tests for input validation and error states are needed.

### Frontend (JavaScript)
The current frontend tests cover basic interactions, but `web/app.js` is quite large and complex.
- [ ] **`performSearch()`**:
    - Test complex filter combinations (e.g., specific category + rating + type + search term).
    - Verify that `AbortController` correctly cancels previous searches.
- [ ] **`openInPiP()` & Player Logic**:
    - Verify HLS fallback logic (when HLS is requested but not supported or fails).
    - Test resume-from-position logic more thoroughly with varying server-side and local storage values.
- [ ] **Routing (`syncUrl` / `onUrlChange`)**:
    - Ensure all state (filters, page, view, search) is correctly persisted and restored from the URL hash.
- [ ] **Error Handling (`handleMediaError`)**:
    - Verify auto-skip behavior on consecutive media errors.
- [ ] **Progress Syncing (`updateProgress`)**:
    - Test the logic that decides when to sync to the server versus just updating local storage (e.g., based on `sessionTime`).

## 3. `web/` (Frontend)
- [ ] `app.js`: Refactor frontend logic into smaller modules or components.
- [ ] Implement better state management (e.g., using a lightweight store instead of global variables).
- [ ] Improve UI responsiveness and mobile experience.
- [ ] Increase test coverage for frontend components (currently 20+ `.test.js` files, but verify depth).
