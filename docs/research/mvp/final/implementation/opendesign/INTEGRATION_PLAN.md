# HelixTerminator × OpenDesign Integration Plan

**Version:** 1.0.0  
**Date:** 2026-07-05  
**Status:** Draft — Awaiting implementation  
**Authority:** `CANONICAL_FACTS.md` CD-1..CD-12, `SERVICE_REGISTRY.md`, `06_ux_design_system.md`

---

## Table of Contents

1. [OpenDesign Architecture Overview](#1-opendesign-architecture-overview)
2. [Design Token Mapping](#2-design-token-mapping)
3. [Component Library Mapping](#3-component-library-mapping)
4. [Multi-Platform Design Strategy](#4-multi-platform-design-strategy)
5. [Wireframe-to-Mockup Conversion Plan](#5-wireframe-to-mockup-conversion-plan)
6. [Visual Regression Test Strategy](#6-visual-regression-test-strategy)
7. [OpenDesign Compliance Gate Remediation](#7-opendesign-compliance-gate-remediation)
8. [Implementation Roadmap](#8-implementation-roadmap)

---

## 1. OpenDesign Architecture Overview

### 1.1 What is OpenDesign?

OpenDesign (https://github.com/nexu-io/open-design) is an open-source design system framework and AI-assisted design tool. It provides:

- **Design System Engine**: A structured approach to defining brand visual languages via 9-section `DESIGN.md` files
- **Token System**: CSS custom properties (`--bg`, `--surface`, `--fg`, `--accent`, etc.) with layered architecture (A1-identity, A1-structure, A2, B-slot)
- **Craft Knowledge**: Universal UI craft rules (typography, color, motion, accessibility) that apply regardless of brand
- **Component Manifests**: Machine-readable component specifications with token references, selectors, and validation
- **Linter**: Automated artifact linting against design system rules (anti-AI-slop, token consumption, etc.)
- **Plugin Ecosystem**: MCP-based plugins for design system source context and skill registration

### 1.2 OpenDesign Repository Structure (Shallow Clone)

```
open-design/
├── craft/                    # Universal craft knowledge (color.md, typography.md, etc.)
├── design-systems/           # 150+ brand design systems (default/, apple/, linear-app/, etc.)
│   └── <brand>/
│       ├── DESIGN.md         # 9-section brand visual language spec
│       ├── source/
│       │   ├── tokens.source.json   # Machine-readable token catalog
│       │   └── evidence.md          # Token provenance
│       ├── system/
│       │   ├── tokens.default.json  # Runtime token values
│       │   ├── kit.html / kit.dark.html  # Component reference kits
│       │   └── artifacts/           # Pre-built artifact templates
│       ├── preview/          # Visual preview HTML files
│       ├── components.manifest.json   # Component token reference manifest
│       └── manifest.json     # Design system metadata
├── plugins/                  # Plugin specifications and community registry
├── tools/                    # Build, pack, serve, release tooling
└── apps/                     # Desktop, web, daemon, landing-page applications
```

### 1.3 Key OpenDesign Concepts for HelixTerminator

| Concept | OpenDesign Term | HelixTerminator Equivalent |
|---------|----------------|---------------------------|
| Brand visual language | `DESIGN.md` | `06_ux_design_system.md` §1-5 |
| Design tokens | `tokens.source.json` + `tokens.css` | `design-tokens.json` (this plan) |
| Component specs | `components.manifest.json` | `component-library-spec.md` (this plan) |
| Craft rules | `craft/*.md` | §3 Typography, §6 Motion, §9 Accessibility |
| Token layers | A1-identity, A1-structure, A2, B-slot | Primitive → Semantic → Component → Theme |
| Lint gates | `lint-artifact.ts` | `cm_opendesign_ui_system.sh` |

### 1.4 OpenDesign Token Contract

OpenDesign's token system uses CSS custom properties with these standard names:

```css
/* Core surface tokens */
--bg, --surface, --surface-warm, --fg, --fg-2, --muted, --meta, --border, --border-soft

/* Accent and semantic */
--accent, --accent-on, --accent-hover, --accent-active
--success, --warn, --danger

/* Typography */
--font-display, --font-body, --font-mono
--text-xs, --text-sm, --text-base, --text-lg, --text-xl, --text-2xl, --text-3xl, --text-4xl
--leading-body, --leading-tight, --tracking-display

/* Spacing */
--space-1 through --space-20, --section-y-*

/* Shape */
--radius-sm, --radius-md, --radius-lg, --radius-pill

/* Elevation */
--elev-flat, --elev-ring, --elev-raised
--focus-ring

/* Motion */
--motion-fast, --motion-base, --ease-standard

/* Layout */
--container-max, --container-gutter-*
```

---

## 2. Design Token Mapping

### 2.1 Mapping Philosophy

HelixTerminator has **750+ design tokens** organized into a 4-tier hierarchy:
1. **Primitives** (`color.primitive.*`, `typography.primitive.*`) — Raw values
2. **Semantic** (`color.semantic.*`, `typography.semantic.*`) — Meaning-based aliases
3. **Component** (`component.*`) — Component-specific values
4. **Theme** (`theme.*`) — Theme-level overrides

OpenDesign uses a flatter 3-layer system (A1-identity, A1-structure, A2). The mapping bridges these by:
- **Tier 1 → OpenDesign A1-identity**: Direct value mapping for core brand colors, fonts
- **Tier 2 → OpenDesign A1-structure**: Semantic tokens map to OpenDesign's standard semantic names
- **Tier 3 → OpenDesign A2**: Component tokens extend OpenDesign's base with Helix-specific additions
- **Tier 4 → OpenDesign B-slots**: Theme overrides use OpenDesign's slot/alias mechanism

### 2.2 Core Color Token Mapping (Dark Theme)

| Helix Token | Helix Value | OpenDesign Token | OpenDesign Value | Notes |
|-------------|-------------|------------------|------------------|-------|
| `surface` | `#0E0E14` | `--bg` | `#0E0E14` | Direct match |
| `surface-raised` | `#16161E` | `--surface` | `#16161E` | Direct match |
| `surface-overlay` | `#1E1E2A` | `--surface-warm` | `#1E1E2A` | Mapped |
| `surface-sunken` | `#0A0A10` | — | `#0A0A10` | Helix-specific, added as custom |
| `text-primary` | `#FFFFFF` | `--fg` | `#FFFFFF` | Direct match |
| `text-secondary` | `#A0A0B8` | `--muted` | `#A0A0B8` | Direct match |
| `text-tertiary` | `#6B6B80` | `--fg-2` | `#6B6B80` | Mapped |
| `text-disabled` | `#56566A` | `--meta` | `#56566A` | Mapped |
| `text-link` | `#9590FF` | — | `#9590FF` | Helix-specific, added as custom |
| `text-error` | `#FF7A7A` | `--danger` | `#FF7A7A` | Mapped |
| `text-warning` | `#F59E0B` | `--warn` | `#F59E0B` | Mapped |
| `text-success` | `#00D4B1` | `--success` | `#00D4B1` | Mapped |
| `interactive-default` | `#6C63FF` | `--accent` | `#6C63FF` | Direct match |
| `interactive-hover` | `#7B74FF` | `--accent-hover` | `color-mix(in oklab, var(--accent), white 8%)` | Computed |
| `interactive-pressed` | `#5A52E0` | `--accent-active` | `color-mix(in oklab, var(--accent), black 8%)` | Computed |
| `border-default` | `#2E2E3E` | `--border` | `#2E2E3E` | Direct match |
| `border-subtle` | `#1E1E2A` | `--border-soft` | `#1E1E2A` | Mapped |
| `border-strong` | `#3A3A4A` | — | `#3A3A4A` | Helix-specific |
| `border-brand` | `#6C63FF` | — | `#6C63FF` | Same as accent |
| `border-error` | `#FF6B6B` | — | `#FF6B6B` | Same as danger |

### 2.3 Core Color Token Mapping (Light Theme)

| Helix Token | Helix Value | OpenDesign Token | OpenDesign Value | Notes |
|-------------|-------------|------------------|------------------|-------|
| `surface` | `#FFFFFF` | `--bg` | `#FFFFFF` | Direct match |
| `surface-raised` | `#F8F8FA` | `--surface` | `#F8F8FA` | Direct match |
| `surface-overlay` | `#FFFFFF` | `--surface-warm` | `#FFFFFF` | Mapped |
| `text-primary` | `#1A1A2E` | `--fg` | `#1A1A2E` | Direct match |
| `text-secondary` | `#4A4A5E` | `--muted` | `#4A4A5E` | Direct match |
| `text-tertiary` | `#7A7A8E` | `--fg-2` | `#7A7A8E` | Mapped |
| `text-disabled` | `#B0B0BE` | `--meta` | `#B0B0BE` | Mapped |
| `text-link` | `#4640AA` | — | `#4640AA` | Helix-specific |
| `interactive-default` | `#5952D4` | `--accent` | `#5952D4` | Direct match |
| `border-default` | `#E0E0E8` | `--border` | `#E0E0E8` | Direct match |
| `border-subtle` | `#F0F0F4` | `--border-soft` | `#F0F0F4` | Mapped |

### 2.4 Typography Token Mapping

| Helix Token | Value | OpenDesign Token | Value | Notes |
|-------------|-------|------------------|-------|-------|
| `font-ui` | Inter | `--font-body` | `"Inter", -apple-system, sans-serif` | Direct match |
| `font-mono` | JetBrains Mono | `--font-mono` | `"JetBrains Mono", ui-monospace, monospace` | Direct match |
| `font-display` | Inter 700 | `--font-display` | `"Inter", -apple-system, sans-serif` | Same stack, weight via token |
| `text-xs` | 11px | `--text-xs` | `11px` | Direct match |
| `text-sm` | 13px | `--text-sm` | `13px` | Direct match |
| `text-base` | 15px | `--text-base` | `15px` | Direct match |
| `text-md` | 17px | `--text-lg` | `17px` | Mapped (OD has fewer steps) |
| `text-lg` | 20px | — | `20px` | Custom |
| `text-xl` | 24px | `--text-xl` | `24px` | Direct match |
| `text-2xl` | 30px | `--text-2xl` | `30px` | Direct match |
| `text-3xl` | 38px | `--text-3xl` | `38px` | Direct match |
| `leading-body` | 1.5 | `--leading-body` | `1.5` | Direct match |
| `leading-tight` | 1.2 | `--leading-tight` | `1.2` | Direct match |

### 2.5 Spacing Token Mapping

| Helix Token | Value | OpenDesign Token | Value | Notes |
|-------------|-------|------------------|-------|-------|
| `space-0.5` | 2px | — | `2px` | Custom |
| `space-1` | 4px | `--space-1` | `4px` | Direct match |
| `space-2` | 8px | `--space-2` | `8px` | Direct match |
| `space-3` | 12px | `--space-3` | `12px` | Direct match |
| `space-4` | 16px | `--space-4` | `16px` | Direct match |
| `space-5` | 20px | `--space-5` | `20px` | Direct match |
| `space-6` | 24px | `--space-6` | `24px` | Direct match |
| `space-8` | 32px | `--space-8` | `32px` | Direct match |
| `space-10` | 40px | — | `40px` | Custom |
| `space-12` | 48px | `--space-12` | `48px` | Direct match |
| `space-16` | 64px | — | `64px` | Custom |
| `space-20` | 80px | `--space-20` | `80px` | Direct match |
| `space-24` | 96px | — | `96px` | Custom |

### 2.6 Motion Token Mapping

| Helix Token | Value | OpenDesign Token | Value | Notes |
|-------------|-------|------------------|-------|-------|
| `duration-instant` | 0ms | — | `0ms` | Custom |
| `duration-fast` | 100ms | `--motion-fast` | `100ms` | Direct match |
| `duration-normal` | 200ms | `--motion-base` | `200ms` | Direct match |
| `duration-slow` | 300ms | — | `300ms` | Custom |
| `duration-slower` | 500ms | — | `500ms` | Custom |
| `duration-relaxed` | 800ms | — | `800ms` | Custom |
| `duration-loop` | 1500ms | — | `1500ms` | Custom |
| `ease-out` | `cubic-bezier(0.0,0.0,0.2,1)` | `--ease-standard` | `cubic-bezier(0.2, 0, 0, 1)` | Close match |
| `ease-in` | `cubic-bezier(0.4,0.0,1,1)` | — | Custom | |
| `ease-in-out` | `cubic-bezier(0.4,0.0,0.2,1)` | — | Custom | |
| `ease-spring` | Spring | — | Custom | |
| `ease-linear` | Linear | — | Custom | |

### 2.7 Shadow/Elevation Token Mapping

| Helix Token | Dark Value | Light Value | OpenDesign Token | Notes |
|-------------|-----------|-------------|------------------|-------|
| `shadow-xs` | `0 1px 2px rgba(0,0,0,0.3)` | `0 1px 2px rgba(0,0,0,0.05)` | `--elev-flat` | Mapped |
| `shadow-sm` | `0 2px 4px rgba(0,0,0,0.4)` | `0 2px 4px rgba(0,0,0,0.08)` | `--elev-ring` | Mapped |
| `shadow-md` | `0 4px 8px rgba(0,0,0,0.5)` | `0 4px 8px rgba(0,0,0,0.12)` | `--elev-raised` | Mapped |
| `shadow-lg` | `0 8px 16px rgba(0,0,0,0.6)` | `0 8px 16px rgba(0,0,0,0.16)` | — | Custom |
| `shadow-xl` | `0 16px 32px rgba(0,0,0,0.7)` | `0 16px 32px rgba(0,0,0,0.20)` | — | Custom |
| `shadow-brand` | `0 0 0 3px rgba(108,99,255,0.40)` | `0 0 0 3px rgba(89,82,212,0.30)` | `--focus-ring` | Mapped |

### 2.8 SSH Status Color Mapping (Helix-Specific)

These tokens have no OpenDesign equivalent and are added as custom extensions:

| Helix Token | Dark Value | Light Value | Extension Name |
|-------------|-----------|-------------|----------------|
| `ssh-connected` | `#00D4B1` | `#00B894` | `--helix-ssh-connected` |
| `ssh-connecting` | `#F59E0B` | `#E8950A` | `--helix-ssh-connecting` |
| `ssh-disconnected` | `#6B6B80` | `#9A9AA8` | `--helix-ssh-disconnected` |
| `ssh-error` | `#FF6B6B` | `#E05555` | `--helix-ssh-error` |
| `ssh-reconnecting` | `#F59E0B` | `#E8950A` | `--helix-ssh-reconnecting` |

### 2.9 Terminal Theme Color Mapping

Terminal colors are user-selectable and map to OpenDesign as theme overrides:

| ANSI Role | Helix Dark | Dracula | Nord | Gruvbox | Solarized | One Dark |
|-----------|-----------|---------|------|---------|-----------|----------|
| Black | `#1E1E2A` | `#44475A` | `#3B4252` | `#282828` | `#073642` | `#282C34` |
| Red | `#FF6B6B` | `#FF5555` | `#BF616A` | `#CC241D` | `#DC322F` | `#E06C75` |
| Green | `#00D4B1` | `#50FA7B` | `#A3BE8C` | `#98971A` | `#859900` | `#98C379` |
| Yellow | `#F59E0B` | `#F1FA8C` | `#EBCB8B` | `#D79921` | `#B58900` | `#E5C07B` |
| Blue | `#6C63FF` | `#BD93F9` | `#81A1C1` | `#458588` | `#268BD2` | `#61AFEF` |
| Magenta | `#FF79C6` | `#FF79C6` | `#B48EAD` | `#B16286` | `#D33682` | `#C678DD` |
| Cyan | `#8BE9FD` | `#8BE9FD` | `#88C0D0` | `#689D6A` | `#2AA198` | `#56B6C2` |
| White | `#F8F8F2` | `#F8F8F2` | `#E5E9F0` | `#A89984` | `#EEE8D5` | `#ABB2BF` |

---

## 3. Component Library Mapping

### 3.1 Mapping Philosophy

HelixTerminator has **50+ custom Flutter components** (`Helix*` widgets). Each maps to an OpenDesign pattern by:
1. Identifying the closest OpenDesign component group from `components.manifest.json`
2. Mapping Helix tokens to OpenDesign tokens for that component
3. Documenting Helix-specific extensions (terminal-specific, SSH-specific)
4. Providing a migration path from ad-hoc hex to token-driven styling

### 3.2 Component Mapping Table

| # | Helix Component | OpenDesign Group | OpenDesign Selectors | Status | Notes |
|---|-----------------|------------------|----------------------|--------|-------|
| 1 | `HelixButton` | buttons | `.btn`, `.btn-primary`, `.btn-secondary` | ✅ Direct | Primary uses `component.button.primaryBg*` not raw `interactive-default` |
| 2 | `HelixTextInput` | inputs | `.field`, `.field input` | ✅ Direct | Add password/search/multiline variants |
| 3 | `HelixSelect` | inputs | `.field` (dropdown behavior) | ⚠️ Extended | Custom dropdown panel styling |
| 4 | `HelixCheckbox` | inputs | (checkbox not in default manifest) | ⚠️ Extended | Add to custom manifest |
| 5 | `HelixRadio` | inputs | (radio not in default manifest) | ⚠️ Extended | Add to custom manifest |
| 6 | `HelixSwitch` | inputs | (toggle not in default manifest) | ⚠️ Extended | Add to custom manifest |
| 7 | `HelixSlider` | — | — | 🔶 Custom | New component group |
| 8 | `HelixCard` | cards | `.card` | ✅ Direct | Add interactive/elevated/host variants |
| 9 | `HelixTag` | badges | `.badge`, `.badge-muted`, `.badge-success` | ✅ Direct | Add color variants |
| 10 | `HelixBadge` | badges | `.badge` | ✅ Direct | Numeric count variant |
| 11 | `HelixTooltip` | — | — | ⚠️ Extended | OpenDesign has `.icon` slot, not tooltip |
| 12 | `HelixDivider` | — | — | 🔶 Custom | Horizontal/vertical/section variants |
| 13 | `HelixSidebar` | layout | `.container`, section | ⚠️ Extended | Complex navigation structure |
| 14 | `HelixTabBar` | — | — | 🔶 Custom | Terminal-specific tab behavior |
| 15 | `HelixBreadcrumb` | — | — | 🔶 Custom | Navigation path display |
| 16 | `HelixModal` | — | — | ⚠️ Extended | Modal/Dialog/Sheet variants |
| 17 | `HelixSheet` | — | — | ⚠️ Extended | Bottom sheet + side sheet |
| 18 | `HelixDrawer` | — | — | ⚠️ Extended | Mobile navigation drawer |
| 19 | `HelixPopover` | — | — | 🔶 Custom | Contextual info popover |
| 20 | `HelixContextMenu` | — | — | 🔶 Custom | Right-click menu |
| 21 | `HelixToast` | — | — | 🔶 Custom | Toast/notification system |
| 22 | `HelixAlertBanner` | — | — | 🔶 Custom | Persistent alert |
| 23 | `HelixProgressBar` | — | — | 🔶 Custom | Linear + circular progress |
| 24 | `HelixSkeleton` | — | — | 🔶 Custom | Loading placeholder |
| 25 | `HelixEmptyState` | — | — | 🔶 Custom | Empty content state |
| 26 | `HelixErrorState` | — | — | 🔶 Custom | Error display state |
| 27 | `HelixDataTable` | — | — | 🔶 Custom | Sortable, paginated table |
| 28 | `HelixList` | layout | `.features-grid` | ⚠️ Extended | Virtual scroll list |
| 29 | `HelixTreeView` | — | — | 🔶 Custom | Host group tree |
| 30 | `HelixSFTPBrowser` | — | — | 🔶 Custom | Dual-pane file browser |
| 31 | `HelixTransferQueue` | — | — | 🔶 Custom | File transfer monitor |
| 32 | `HelixTerminal` | — | — | 🔶 Custom | Terminal emulator viewport |
| 33 | `HelixTerminalTab` | — | — | 🔶 Custom | Terminal session tab |
| 34 | `HelixTerminalToolbar` | — | — | 🔶 Custom | Terminal action toolbar |
| 35 | `HelixSplitView` | — | — | 🔶 Custom | Tiled terminal layout |
| 36 | `HelixFocusModeOverlay` | — | — | 🔶 Custom | Focus mode chrome |
| 37 | `HelixBroadcastIndicator` | — | — | 🔶 Custom | Multi-session broadcast |
| 38 | `HelixSessionBadge` | — | — | 🔶 Custom | Connection status badge |
| 39 | `HelixHostCard` | cards | `.card` | ⚠️ Extended | Host-specific card variant |
| 40 | `HelixConnectionDialog` | — | — | 🔶 Custom | SSH connection confirmation |
| 41 | `HelixJumpHostChip` | — | — | 🔶 Custom | Jump host chain display |
| 42 | `HelixProtocolBadge` | badges | `.badge` | ⚠️ Extended | Protocol label badge |
| 43 | `HelixAuthMethodIcon` | icons | `.icon` | ✅ Direct | Icon + tooltip |
| 44 | `HelixKeyFingerprint` | — | — | 🔶 Custom | SSH key display |
| 45 | `HelixPortForwardRow` | — | — | 🔶 Custom | Port forwarding rule |
| 46 | `HelixSnippetCard` | cards | `.card` | ⚠️ Extended | Snippet with code preview |
| 47 | `HelixDatePicker` | — | — | 🔶 Custom | Calendar date picker |
| 48 | `HelixColorPicker` | — | — | 🔶 Custom | Theme color picker |
| 49 | `HelixCommandPalette` | — | — | 🔶 Custom | Quick action palette |
| 50 | `HelixAppShell` | layout | `.container`, section | ⚠️ Extended | Main app scaffold |

**Legend:**
- ✅ Direct — Maps cleanly to existing OpenDesign component
- ⚠️ Extended — Maps to OpenDesign but requires Helix-specific extensions
- 🔶 Custom — No OpenDesign equivalent; must be added to custom manifest

### 3.3 Custom Component Manifest Strategy

For the 30+ custom components, we create a `helix-components.manifest.json` that extends OpenDesign's default manifest with:

```json
{
  "schemaVersion": 1,
  "brandId": "helix-terminator",
  "extends": "design-systems/default/components.manifest.json",
  "customGroups": [
    {
      "id": "terminal",
      "label": "Terminal emulator components",
      "selectors": [".helix-terminal", ".helix-terminal-tab", ".helix-terminal-toolbar"],
      "tokenReferences": ["--helix-terminal-bg", "--helix-terminal-fg", "--helix-cursor"]
    },
    {
      "id": "ssh-status",
      "label": "SSH connection status indicators",
      "selectors": [".helix-session-badge", ".helix-connection-dot"],
      "tokenReferences": ["--helix-ssh-connected", "--helix-ssh-connecting", "--helix-ssh-error"]
    },
    {
      "id": "file-browser",
      "label": "SFTP file browser components",
      "selectors": [".helix-sftp-pane", ".helix-file-row", ".helix-transfer-item"],
      "tokenReferences": ["--helix-file-dir", "--helix-file-executable", "--helix-transfer-progress"]
    }
  ]
}
```

---

## 4. Multi-Platform Design Strategy

### 4.1 Platform Matrix

HelixTerminator targets **8 platforms** via Flutter. Each platform has specific design token adaptations:

| Platform | Framework | Design System Base | Key Adaptations |
|----------|-----------|-------------------|-----------------|
| **Web** | Flutter Web (WASM) | OpenDesign default + CSS | CSS custom properties, responsive breakpoints |
| **macOS** | Flutter Desktop | Apple HIG + OpenDesign | Native menu bar, SF Pro font fallback, vibrancy |
| **Windows** | Flutter Desktop | Fluent Design + OpenDesign | Segoe UI font, acrylic material, title bar |
| **Linux** | Flutter Desktop | GTK/Adwaita + OpenDesign | System font, native file picker, portal integration |
| **iOS** | Flutter Mobile | Apple HIG + OpenDesign | Safe areas, bottom sheet, Dynamic Island |
| **Android** | Flutter Mobile | Material 3 + OpenDesign | Material You dynamic color, edge-to-edge |
| **HarmonyOS** | Flutter Mobile | HarmonyOS Design + OpenDesign | System font, native services |
| **AuroraOS** | Flutter Mobile | Sailfish OS + OpenDesign | Silica UI patterns, gesture navigation |

### 4.2 Platform-Specific Token Overrides

Each platform receives a token override file that adapts the base tokens:

```
design-tokens/
├── base/                    # Core tokens (platform-agnostic)
│   ├── tokens.dark.json
│   └── tokens.light.json
├── platforms/
│   ├── web.tokens.json      # Web-specific overrides
│   ├── macos.tokens.json    # macOS-specific overrides
│   ├── windows.tokens.json  # Windows-specific overrides
│   ├── linux.tokens.json    # Linux-specific overrides
│   ├── ios.tokens.json      # iOS-specific overrides
│   ├── android.tokens.json  # Android-specific overrides
│   ├── harmonyos.tokens.json # HarmonyOS-specific overrides
│   └── auroraos.tokens.json # AuroraOS-specific overrides
```

### 4.3 Platform-Specific Adaptations

#### Web
- **Font stack**: `Inter, -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif`
- **Scrollbars**: Custom styled webkit scrollbars matching Helix theme
- **Focus rings**: CSS `:focus-visible` with `box-shadow` token
- **Responsive**: Full breakpoint system (mobile-sm through desktop-xl)
- **CSS output**: Design tokens as CSS custom properties in `:root`

#### macOS
- **Font fallback**: `SF Pro Text` before `Inter` for native feel
- **Window chrome**: Native title bar with traffic lights (hidden in focus mode)
- **Vibrancy**: Optional `NSVisualEffectView` behind sidebar (`--helix-macos-vibrancy`)
- **Menu bar**: Native menu with HelixTerminator menu structure
- **Touch Bar**: Terminal session tabs, quick actions
- **Keyboard shortcuts**: `⌘` prefix for all shortcuts

#### Windows
- **Font fallback**: `Segoe UI Variable` before `Inter`
- **Acrylic**: Optional acrylic material in sidebar (`--helix-windows-acrylic`)
- **Title bar**: Custom title bar with minimize/maximize/close
- **Mica material**: Windows 11 Mica backdrop option
- **Keyboard shortcuts**: `Ctrl` prefix for all shortcuts

#### Linux
- **Font fallback**: `Noto Sans` before `Inter`
- **GTK integration**: Respect system GTK theme for file picker
- **Portal integration**: Use xdg-desktop-portal for file operations
- **Wayland/X11**: Proper surface handling under both
- **Keyboard shortcuts**: `Ctrl` prefix, respect i3/sway mod key

#### iOS
- **Safe areas**: Respect notch, Dynamic Island, home indicator
- **Bottom sheet**: Use native iOS bottom sheet for modals
- **Haptic feedback**: Light impact on connection success, error on failure
- **Dynamic Island**: Show active session status
- **Keyboard**: Custom accessory view with ESC/CTRL/TAB/arrow keys
- **Context menu**: Native iOS context menu with haptic

#### Android
- **Edge-to-edge**: Full screen with system bar theming
- **Material You**: Optional dynamic color from wallpaper
- **Bottom navigation**: Native bottom nav on phones
- **Haptic feedback**: Standard Android haptic patterns
- **Keyboard**: Custom input method with terminal keys

#### HarmonyOS
- **System font**: HarmonyOS Sans as primary font
- **Service integration**: Native HMS services where available
- **Distributed UI**: Cross-device session continuity

#### AuroraOS (Sailfish OS)
- **Silica patterns**: Pulley menu, page stack navigation
- **Gesture navigation**: Swipe-from-edge patterns
- **Ambience**: Respect system ambience (theme)

---

## 5. Wireframe-to-Mockup Conversion Plan

### 5.1 Screen Inventory (28 Screens)

| # | Screen | Section | Priority | OpenDesign Skill | Complexity |
|---|--------|---------|----------|------------------|------------|
| 1 | Splash / Launch | §7.1 | P0 | `splash-screen` | Low |
| 2 | Onboarding Step 1 | §7.2 | P0 | `onboarding-welcome` | Low |
| 3 | Onboarding Step 2 | §7.3 | P0 | `onboarding-import` | Medium |
| 4 | Onboarding Step 3 | §7.4 | P0 | `onboarding-appearance` | Medium |
| 5 | Login Desktop | §7.5 | P0 | `auth-login` | Low |
| 6 | Login Mobile | §7.6 | P0 | `auth-login-mobile` | Low |
| 7 | MFA TOTP | §7.7 | P0 | `auth-mfa-totp` | Low |
| 8 | MFA FIDO2 | §7.8 | P0 | `auth-mfa-fido2` | Low |
| 9 | Host List Grid | §7.9 | P0 | `host-list-grid` | High |
| 10 | Host List List | §7.10 | P0 | `host-list-list` | High |
| 11 | Host Detail / Edit | §7.11 | P0 | `host-detail-form` | High |
| 12 | Quick Connect | §7.12 | P0 | `quick-connect-dialog` | Medium |
| 13 | Terminal Single Desktop | §7.13 | P0 | `terminal-single-desktop` | High |
| 14 | Terminal Single Mobile | §7.14 | P0 | `terminal-single-mobile` | High |
| 15 | Terminal Split 2×1 | §7.15 | P1 | `terminal-split-horizontal` | High |
| 16 | Terminal Split 1×2 | §7.16 | P1 | `terminal-split-vertical` | High |
| 17 | Terminal Split 2×2 | §7.17 | P1 | `terminal-split-grid` | High |
| 18 | SFTP Browser | §7.18 | P1 | `sftp-browser` | High |
| 19 | Port Forwarding | §7.19 | P1 | `port-forwarding` | Medium |
| 20 | Snippets Library | §7.20 | P1 | `snippet-library` | Medium |
| 21 | Key Manager | §7.21 | P1 | `key-manager` | Medium |
| 22 | Settings | §7.22 | P1 | `settings-page` | High |
| 23 | Vault | §7.23 | P1 | `vault-manager` | High |
| 24 | Session History | §7.24 | P1 | `session-history` | Medium |
| 25 | Audit Log | §7.25 | P2 | `audit-log` | Medium |
| 26 | Collaboration | §7.26 | P2 | `collaboration-panel` | High |
| 27 | Command Palette | §7.27 | P2 | `command-palette` | Medium |
| 28 | Workspace Manager | §7.28 | P2 | `workspace-manager` | Medium |

### 5.2 Conversion Methodology

For each screen, the wireframe-to-mockup conversion follows this pipeline:

```
Wireframe (ASCII) → Token Assignment → Component Selection → Layout Specification → OpenDesign Skill Prompt → Generated Mockup → Visual Regression Capture
```

**Step 1: Token Assignment**
- Map every color in the wireframe to a design token
- Map every dimension to a spacing token
- Map every font to a typography token
- Verify no ad-hoc hex values remain

**Step 2: Component Selection**
- Identify which OpenDesign components are used
- Identify which Helix custom components are needed
- Document component hierarchy

**Step 3: Layout Specification**
- Define responsive behavior per breakpoint
- Define platform-specific adaptations
- Specify animation/motion tokens

**Step 4: OpenDesign Skill Prompt**
- Generate an OpenDesign-compatible skill prompt
- Include `od.craft.requires` for needed craft sections
- Reference the active `DESIGN.md` (HelixTerminator brand)

**Step 5: Generated Mockup**
- Use OpenDesign's generation pipeline
- Validate output against component manifest
- Run lint-artifact for token compliance

**Step 6: Visual Regression Capture**
- Capture screenshot at multiple breakpoints
- Store as baseline in `__image_snapshots__`
- Tag with platform and theme

### 5.3 Example: Screen 9 (Host List Grid) Conversion

**Wireframe elements → Tokens:**
- Background: `surface` → `--bg`
- Sidebar: `surface-raised` → `--surface`
- Host cards: `surface-raised` + `border-subtle` → `--surface` + `--border-soft`
- Connected dot: `ssh-connected` → `--helix-ssh-connected`
- Button: `interactive-default` → `--accent`
- Text: `text-primary` → `--fg`, `text-secondary` → `--muted`

**Components used:**
- `HelixSidebar` → OpenDesign layout + custom nav
- `HelixHostCard` → OpenDesign `.card` + custom extensions
- `HelixButton` → OpenDesign `.btn-primary`
- `HelixTag` → OpenDesign `.badge`
- `HelixSearchInput` → OpenDesign `.field input`
- `HelixIconButton` → OpenDesign `.btn` (icon-only)

**OpenDesign skill prompt:**
```yaml
od:
  craft:
    requires: [typography, color, state-coverage, accessibility-baseline]
  design-system: helix-terminator
  artifact: host-list-grid
  platforms: [web, macos, windows, linux, ios, android]
```

---

## 6. Visual Regression Test Strategy

### 6.1 Test Architecture

Visual regression testing ensures that UI changes do not introduce unintended visual differences. The strategy uses:

- **Baseline capture**: Screenshots of all 28 screens at multiple breakpoints and themes
- **Pixel diff comparison**: Perceptual diff with threshold tolerance
- **CI integration**: Automated capture and comparison on every PR
- **Platform coverage**: Web (primary), with spot checks on desktop/mobile

### 6.2 Test File Structure

```
tests/visual-regression/
├── baselines/                    # Golden master screenshots
│   ├── web/
│   │   ├── dark/
│   │   │   ├── 01-splash.png
│   │   │   ├── 05-login-desktop.png
│   │   │   └── ... (28 screens)
│   │   └── light/
│   │       ├── 01-splash.png
│   │       └── ...
│   ├── macos/
│   │   └── ...
│   └── ios/
│       └── ...
├── snapshots/                    # Current test run screenshots
├── diffs/                        # Generated diff images
├── config/
│   ├── playwright.config.ts      # Playwright test configuration
│   └── viewport-sizes.ts         # Breakpoint definitions
├── specs/
│   ├── 01-splash.spec.ts
│   ├── 05-login.spec.ts
│   └── ... (28 screen specs)
└── helpers/
    ├── theme-switcher.ts         # Programmatic theme switching
    ├── screenshot-utils.ts       # Capture utilities
    └── diff-reporter.ts          # Diff report generation
```

### 6.3 Test Specifications

Each screen test:

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
          { threshold: 0.2 } // 0.2% pixel difference tolerance
        );
      });
    }
  }
  
  test('interaction: host card hover', async ({ page }) => {
    await page.goto('/hosts?view=grid');
    const card = page.locator('[data-testid="host-card"]').first();
    await card.hover();
    await page.waitForTimeout(200); // Allow hover transition
    const screenshot = await page.screenshot();
    expect(screenshot).toMatchSnapshot('09-host-list-grid-hover.png');
  });
  
  test('interaction: connection status change', async ({ page }) => {
    await page.goto('/hosts?view=grid');
    // Simulate connection state change
    await page.evaluate(() => {
      window.dispatchEvent(new CustomEvent('mock-connection', { detail: 'connected' }));
    });
    await page.waitForTimeout(300); // Allow status animation
    const screenshot = await page.screenshot();
    expect(screenshot).toMatchSnapshot('09-host-list-grid-connected.png');
  });
});
```

### 6.4 CI Integration

```yaml
# .github/workflows/visual-regression.yml
name: Visual Regression Tests
on: [pull_request]
jobs:
  visual-regression:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Run visual regression tests
        run: |
          npm install
          npx playwright install
          npm run test:visual
      - name: Upload diff report
        if: failure()
        uses: actions/upload-artifact@v4
        with:
          name: visual-diff-report
          path: tests/visual-regression/diffs/
```

### 6.5 Baseline Update Process

1. Developer makes UI change
2. CI runs visual regression, detects diffs
3. If diffs are intentional: developer runs `npm run test:visual:update`
4. Updated baselines are committed as part of the PR
5. Reviewer approves both code and visual changes

---

## 7. OpenDesign Compliance Gate Remediation

### 7.1 Gate Analysis

The `cm_opendesign_ui_system.sh` gate checks four sub-invariants:

| Sub-check | Current State | Required Action | Owner |
|-----------|--------------|-------------------|-------|
| (a) OpenDesign declared | ❌ FAIL | Add `.mcp.json` referencing open-design | This plan |
| (b) Token artifact + no ad-hoc hex | ❌ FAIL | Create `design-tokens.json`, remove hardcoded hex | This plan |
| (c) Light + dark variants | ⚠️ PARTIAL | Complete light theme tokens | This plan |
| (d) Visual regression tests | ❌ FAIL | Create visual regression test suite | This plan |

### 7.2 Remediation Plan

#### 7.2.1 Sub-check (a): Declare OpenDesign Dependency

**Action:** Create `.mcp.json` at project root:

```json
{
  "mcpServers": {
    "open-design": {
      "command": "npx",
      "args": ["-y", "@open-design/mcp@latest"],
      "env": {
        "OPEN_DESIGN_API_KEY": "${OPEN_DESIGN_API_KEY}"
      }
    }
  },
  "dependencies": {
    "design-systems": [
      {
        "name": "helix-terminator",
        "source": "./design-systems/helix-terminator",
        "extends": "default"
      }
    ]
  }
}
```

Also add to `pubspec.yaml` (Flutter) and `go.mod` (Go backend) metadata:

```yaml
# pubspec.yaml
open_design:
  design_system: helix-terminator
  token_file: assets/design-tokens.json
```

#### 7.2.2 Sub-check (b): Token Artifact + No Ad-Hoc Hex

**Action:** 
1. Create `docs/research/mvp/final/implementation/opendesign/design-tokens.json` (W3C Style Dictionary format)
2. Create platform-specific token variants
3. Audit all existing theme sources for hardcoded hex values
4. Replace all `#RRGGBB` literals with token references

**Files to create:**
- `design-tokens.json` — Base token file
- `design-tokens.web.json` — Web platform overrides
- `design-tokens.macos.json` — macOS overrides
- `design-tokens.windows.json` — Windows overrides
- `design-tokens.linux.json` — Linux overrides
- `design-tokens.ios.json` — iOS overrides
- `design-tokens.android.json` — Android overrides
- `design-tokens.harmonyos.json` — HarmonyOS overrides
- `design-tokens.auroraos.json` — AuroraOS overrides

#### 7.2.3 Sub-check (c): Light + Dark Variants

**Action:**
1. Complete the light theme token set (currently deferred in §2.3.2)
2. Ensure every screen/component has both light and dark implementations
3. Add theme toggle to settings
4. Respect OS theme preference (`prefers-color-scheme`)

**Files to update:**
- `design-tokens.json` — Add complete light theme section
- All component specs — Verify light theme support

#### 7.2.4 Sub-check (d): Visual Regression Tests

**Action:**
1. Create `tests/visual-regression/` directory structure
2. Implement Playwright-based screenshot capture
3. Create baseline screenshots for all 28 screens
4. Add CI workflow for automated visual regression
5. Document baseline update process

### 7.3 Gate Pass Verification

After implementing the remediation plan, run:

```bash
# From project root
bash constitution/scripts/gates/cm_opendesign_ui_system.sh --root .
```

Expected output:
```
CM-OPENDESIGN-UI-SYSTEM (§11.4.162) — auditing /home/milos/Factory/projects/tools_and_research/helix_terminator
======================================================================
✅ (a) OpenDesign declared dependency — declared in .mcp.json
✅ (b) design-token artifact present AND theme sources free of ad-hoc hex
      tokens: docs/research/mvp/final/implementation/opendesign/design-tokens.json
✅ (c) light + dark variants both present in theme/token sources
✅ (d) visual-regression tests present
      tests/visual-regression/specs/
======================================================================
✅ CM-OPENDESIGN-UI-SYSTEM: PASS — all 4 applicable sub-checks passed
```

---

## 8. Implementation Roadmap

### Phase 1: Foundation (Week 1-2)
- [ ] Create `design-tokens.json` with dark + light themes
- [ ] Create platform-specific token variants (8 platforms)
- [ ] Create `.mcp.json` OpenDesign dependency declaration
- [ ] Create custom component manifest (`helix-components.manifest.json`)
- [ ] Run gate script, verify (a) and (b) pass

### Phase 2: Component Integration (Week 3-4)
- [ ] Map all 50+ Helix components to OpenDesign patterns
- [ ] Create component library specification document
- [ ] Implement token-driven theming in Flutter (`HelixTheme`)
- [ ] Replace ad-hoc hex values with token references
- [ ] Run gate script, verify (c) passes

### Phase 3: Screen Implementation (Week 5-8)
- [ ] Convert wireframes to mockups for all 28 screens
- [ ] Implement screens in Flutter using token-driven components
- [ ] Platform-specific adaptations (8 platforms)
- [ ] Accessibility audit per screen

### Phase 4: Visual Regression (Week 9-10)
- [ ] Set up Playwright visual regression framework
- [ ] Capture baseline screenshots for all screens
- [ ] Implement CI workflow
- [ ] Run gate script, verify (d) passes

### Phase 5: Polish & Compliance (Week 11-12)
- [ ] Full gate pass verification
- [ ] Performance audit (60fps targets)
- [ ] Accessibility audit (WCAG 2.1 AA)
- [ ] Documentation finalization
- [ ] Team handoff

---

*HelixTerminator × OpenDesign Integration Plan v1.0.0*  
*All content conforms to CANONICAL_FACTS.md and SERVICE_REGISTRY.md*
