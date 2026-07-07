# HelixTerminator Visual Regression Test Strategy

**Version:** 1.0.0  
**Date:** 2026-07-05  
**Status:** Draft  
**Authority:** `06_ux_design_system.md` §5, `cm_opendesign_ui_system.sh` §11.4.162

---

## 1. Overview

This document defines the visual regression testing strategy for HelixTerminator's OpenDesign-compliant UI system. The goal is to ensure that every UI change is visually verified across all 28 screens, 2 themes (dark/light), and 8 platforms.

## 2. Test Scope

### 2.1 Screens Under Test (28 Screens)

| # | Screen | Priority | Theme Coverage | Platform Coverage |
|---|--------|----------|---------------|-------------------|
| 1 | Splash / Launch | P0 | Dark, Light | Web, Desktop |
| 2 | Onboarding Step 1 | P0 | Dark, Light | All |
| 3 | Onboarding Step 2 | P0 | Dark, Light | All |
| 4 | Onboarding Step 3 | P0 | Dark, Light | All |
| 5 | Login Desktop | P0 | Dark, Light | Web, macOS, Windows, Linux |
| 6 | Login Mobile | P0 | Dark, Light | iOS, Android |
| 7 | MFA TOTP | P0 | Dark, Light | All |
| 8 | MFA FIDO2 | P0 | Dark, Light | All |
| 9 | Host List Grid | P0 | Dark, Light | All |
| 10 | Host List List | P0 | Dark, Light | All |
| 11 | Host Detail / Edit | P0 | Dark, Light | All |
| 12 | Quick Connect | P0 | Dark, Light | All |
| 13 | Terminal Single Desktop | P0 | Dark, Light | Web, Desktop |
| 14 | Terminal Single Mobile | P0 | Dark, Light | Mobile |
| 15 | Terminal Split 2×1 | P1 | Dark, Light | Desktop |
| 16 | Terminal Split 1×2 | P1 | Dark, Light | Desktop |
| 17 | Terminal Split 2×2 | P1 | Dark, Light | Desktop |
| 18 | SFTP Browser | P1 | Dark, Light | All |
| 19 | Port Forwarding | P1 | Dark, Light | All |
| 20 | Snippets Library | P1 | Dark, Light | All |
| 21 | Key Manager | P1 | Dark, Light | All |
| 22 | Settings | P1 | Dark, Light | All |
| 23 | Vault | P1 | Dark, Light | All |
| 24 | Session History | P1 | Dark, Light | All |
| 25 | Audit Log | P2 | Dark, Light | Desktop |
| 26 | Collaboration | P2 | Dark, Light | Desktop |
| 27 | Command Palette | P2 | Dark, Light | All |
| 28 | Workspace Manager | P2 | Dark, Light | All |

### 2.2 Viewport Breakpoints

| Breakpoint | Width | Target |
|-----------|-------|--------|
| mobile-sm | 320px | Small phones |
| mobile | 480px | Standard phones |
| tablet-sm | 600px | Small tablets |
| tablet | 768px | iPad, Android tablets |
| desktop-sm | 1024px | Small laptops |
| desktop | 1280px | Standard laptops |
| desktop-lg | 1536px | Large monitors |
| desktop-xl | 1920px | 4K monitors |

### 2.3 Theme Coverage

- **Dark theme**: Helix Dark (default)
- **Light theme**: Helix Light
- **High Contrast**: Both dark and light variants

## 3. Test Architecture

### 3.1 Technology Stack

| Layer | Tool | Purpose |
|-------|------|---------|
| Test Runner | Playwright | Cross-browser screenshot capture |
| Baseline Storage | Git LFS | Version-controlled golden masters |
| Diff Engine | pixelmatch | Perceptual diff with threshold |
| CI Integration | GitHub Actions | Automated PR checks |
| Reporting | HTML report | Visual diff review |

### 3.2 Directory Structure

```
tests/visual-regression/
├── baselines/                    # Golden master screenshots
│   ├── web/
│   │   ├── dark/
│   │   │   ├── 01-splash@desktop.png
│   │   │   ├── 01-splash@mobile.png
│   │   │   └── ...
│   │   └── light/
│   │       └── ...
│   ├── macos/
│   │   └── ...
│   └── ios/
│       └── ...
├── snapshots/                    # Current test run screenshots
├── diffs/                        # Generated diff images
├── config/
│   ├── playwright.config.ts
│   └── viewport-sizes.ts
├── specs/
│   ├── 01-splash.spec.ts
│   ├── 05-login.spec.ts
│   └── ... (28 screen specs)
├── helpers/
│   ├── theme-switcher.ts
│   ├── screenshot-utils.ts
│   └── diff-reporter.ts
└── package.json
```

## 4. Test Specifications

### 4.1 Baseline Capture Test

```typescript
// tests/visual-regression/specs/09-host-list-grid.spec.ts
import { test, expect } from '@playwright/test';
import { setTheme, captureAtViewport } from '../helpers';

test.describe('Screen 9: Host List Grid', () => {
  const screenName = '09-host-list-grid';
  
  for (const theme of ['dark', 'light']) {
    for (const viewport of ['mobile', 'tablet', 'desktop', 'desktop-lg']) {
      test(`${theme} / ${viewport}`, async ({ page }) => {
        await page.goto('/hosts?view=grid');
        await setTheme(page, theme);
        await captureAtViewport(page, viewport);
        
        const screenshot = await page.screenshot({ fullPage: false });
        expect(screenshot).toMatchSnapshot(
          `${screenName}-${theme}-${viewport}.png`,
          { threshold: 0.2 }
        );
      });
    }
  }
});
```

### 4.2 Interaction Tests

```typescript
test.describe('Interactions', () => {
  test('host card hover', async ({ page }) => {
    await page.goto('/hosts?view=grid');
    const card = page.locator('[data-testid="host-card"]').first();
    await card.hover();
    await page.waitForTimeout(200);
    const screenshot = await page.screenshot();
    expect(screenshot).toMatchSnapshot('09-host-list-grid-hover.png');
  });

  test('connection status change', async ({ page }) => {
    await page.goto('/hosts?view=grid');
    await page.evaluate(() => {
      window.dispatchEvent(new CustomEvent('mock-connection', { 
        detail: 'connected' 
      }));
    });
    await page.waitForTimeout(300);
    const screenshot = await page.screenshot();
    expect(screenshot).toMatchSnapshot('09-host-list-grid-connected.png');
  });
});
```

## 5. CI Integration

### 5.1 GitHub Actions Workflow

```yaml
# .github/workflows/visual-regression.yml
name: Visual Regression Tests
on: [pull_request]
jobs:
  visual-regression:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Setup Node.js
        uses: actions/setup-node@v4
        with:
          node-version: '20'
      - name: Install dependencies
        run: |
          npm install
          npx playwright install
      - name: Run visual regression tests
        run: npm run test:visual
      - name: Upload diff report
        if: failure()
        uses: actions/upload-artifact@v4
        with:
          name: visual-diff-report
          path: tests/visual-regression/diffs/
```

### 5.2 Baseline Update Process

1. Developer makes UI change
2. CI runs visual regression, detects diffs
3. If diffs are intentional: `npm run test:visual:update`
4. Updated baselines committed as part of PR
5. Reviewer approves both code and visual changes

## 6. Acceptance Criteria

- [ ] All 28 screens have baseline screenshots for dark theme
- [ ] All 28 screens have baseline screenshots for light theme
- [ ] All P0 screens tested on all 8 platforms
- [ ] CI workflow runs on every PR
- [ ] Diff report generated on failure
- [ ] Baseline update process documented
- [ ] `cm_opendesign_ui_system.sh` sub-check (d) passes

## 7. Performance Budgets

| Context | Target FPS | Max Rasterization Time |
|---------|-----------|----------------------|
| Terminal text rendering | 60fps | 4ms |
| UI transitions | 60fps | 8ms |
| Scrolling | 60fps | 6ms |
| Modal/panel animation | 60fps | 5ms |

---

*HelixTerminator Visual Regression Test Strategy v1.0.0*
