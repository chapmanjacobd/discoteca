# Plan 5: Performance & Mobile UX Refinement

## Goal
Improve system efficiency and fix critical UI issues on mobile devices.

## Context
Subtitles are currently converted using FFmpeg on every request, regardless of whether the file has them. Additionally, Disk Usage (DU) mode is unusable on small screens due to overlapping UI elements.

## Tasks

### 1. Optimized Subtitle Conversion
- **File:** `internal/commands/serve_streaming.go`
- **Logic:** Before spawning `ffmpeg -i ... -f webvtt`, query the `media` table for the `subtitle_count` of the specific file.
- **Action:** If `subtitle_count == 0`, return a `404` or empty response immediately, saving CPU cycles.

### 2. Mobile DU Mode Fix
- **File:** `web/style.css`, `web/app.js`
- **Issue:** The toolbar and DU tree overlap on mobile.
- **Fix:** 
    - Use media queries to stack the DU toolbar vertically.
    - Implement horizontal scrolling or "drill-down" navigation for the DU path list on screens < 600px.

### 3. Search Enhancements
- **Task:** Add a toggle in the UI to switch between FTS (Fast) and Substring (Thorough) search.
- **Implementation:** Update `HandleSearch` in `serve_handlers.go` to support `LIKE %query%` when a specific flag is passed.

## Verification
- **Performance:** Profile CPU usage during a page load with 50+ media items.
- **Mobile:** Use Chrome DevTools mobile emulation (iPhone SE/Pixel 5) to verify DU mode layout.
