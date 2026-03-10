# Plan 2: CI/CD & Infrastructure Recovery

## Goal
Restore the automated pipeline and implement server health checks to ensure reliable deployment and testing.

## Context
The GitHub Action workflows in `.github/workflows/` have been renamed to `.bak0`, effectively disabling CI. Additionally, E2E tests often fail because they attempt to connect to the server before it is fully initialized.

## Tasks

### 1. Restore GitHub Actions
- Rename and repair:
    - `ci.yml.bak0` -> `ci.yml`: Standard Go build, lint, and unit tests.
    - `e2e.yml.bak0` -> `e2e.yml`: Playwright integration tests.
    - `release.yml.bak0` -> `release.yml`: Automated binary tagging and uploading.
- **Update Dependencies:** Ensure Node.js and Go versions match the current development environment.

### 2. Implement Health Check Endpoint
- **File:** `internal/commands/serve.go`
- **Action:** Add an `/api/ping` or `/health` endpoint that returns a `200 OK`.
- **Purpose:** Allow the E2E test runner to "wait-for-it" before starting Playwright suites.

### 3. Test Runner Integration
- **File:** `e2e/utils/test-server.ts`
- **Action:** Update the server startup logic to poll the health check endpoint before resolving the `start()` promise.

## Verification
- Push a branch and verify that GitHub Actions trigger and pass.
- Run `cd e2e && npx playwright test` to ensure tests wait for the server correctly.
