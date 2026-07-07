# HelixTerminator Component Library Specification

**Version:** 1.0.0  
**Date:** 2026-07-05  
**Status:** Draft  
**Authority:** `06_ux_design_system.md` §5, `SERVICE_REGISTRY.md`

---

## Table of Contents

1. [Overview](#1-overview)
2. [Component Catalog](#2-component-catalog)
3. [OpenDesign Mapping Details](#3-opendesign-mapping-details)
4. [Token Reference Matrix](#4-token-reference-matrix)
5. [Platform Adaptations](#5-platform-adaptations)
6. [Accessibility Requirements](#6-accessibility-requirements)

---

## 1. Overview

This document maps all 50+ HelixTerminator Flutter components to OpenDesign patterns. Each component entry includes:

- **Helix Name**: The Flutter widget name
- **OpenDesign Group**: The closest OpenDesign component group
- **OpenDesign Selectors**: CSS selectors from `components.manifest.json`
- **Token References**: All design tokens consumed by this component
- **Platform Notes**: Platform-specific adaptations
- **Accessibility**: WCAG requirements and Flutter Semantics

---

## 2. Component Catalog

### 2.1 Action Components (1-6)

#### 1. HelixButton
- **OpenDesign Group**: `buttons`
- **OpenDesign Selectors**: `.btn`, `.btn-primary`, `.btn-primary:hover`, `.btn-primary:active`, `.btn-secondary`, `.btn-secondary:hover`, `.btn:focus-visible`
- **Variants**: Primary, Secondary, Danger, Ghost, Icon-only
- **Sizes**: Small (32px), Default (40px), Large (48px)
- **Token References**:
  - `component.button.primaryBg` → `--accent`
  - `component.button.primaryBgHover` → `--accent-hover`
  - `component.button.primaryBgPressed` → `--accent-active`
  - `component.button.primaryText` → `--accent-on`
  - `component.button.destructiveBg` → `--danger` (with darkening)
  - `component.button.heightSm/Base/Lg` → `--space-*`
  - `component.button.paddingXSm/Base/Lg` → `--space-*`
  - `component.button.borderRadius` → `--radius-sm`
  - `component.button.fontWeight` → `--font-body` weight
- **States**: Default, Hover, Focus (2px `--border-brand` ring), Active/Pressed, Disabled (38% opacity), Loading (spinner)
- **Accessibility**: Focus ring always visible, minimum 44×44px touch target for icon-only, `Semantics(button: true, label: '...', enabled: !disabled)`
- **Flutter Base**: `FilledButton` (primary), `OutlinedButton` (secondary), `TextButton` (ghost)

#### 2. HelixTextInput
- **OpenDesign Group**: `inputs`
- **OpenDesign Selectors**: `.field`, `.field input`, `.field input::placeholder`, `.field input:focus-visible`, `.field label`, `.field-help`
- **Variants**: Default, Password, Search, Multiline, Code/Command
- **Token References**:
  - `component.input.bg` → `--surface-sunken` (dark) / `--bg` (light)
  - `component.input.border` → `--border`
  - `component.input.borderFocus` → `--accent`
  - `component.input.text` → `--fg`
  - `component.input.placeholder` → `--muted`
  - `component.input.height` → `--space-*`
  - `component.input.borderRadius` → `--radius-sm`
- **States**: Idle, Focused (2px border + glow), Filled, Error (`--danger` border), Warning, Disabled
- **Accessibility**: `Semantics(label: fieldLabel, hint: helperText, enabled: !disabled)`, error announced via `Semantics(label: 'Email, error: required')`
- **Flutter Base**: `TextField` with custom `InputDecoration`

#### 3. HelixSelect
- **OpenDesign Group**: `inputs` (extended)
- **OpenDesign Selectors**: `.field` (dropdown behavior)
- **Token References**:
  - Same as HelixTextInput for trigger
  - Dropdown panel: `--surface` bg, `--border` border, `--shadow-lg`
  - Selected item: `--accent` + checkmark
  - Hover item: `--surface-interactive`
- **Accessibility**: `Semantics(hasPopup: true)`, arrow key navigation, `SemanticsService.announce` on selection change
- **Flutter Base**: Custom overlay with `CompositedTransformFollower`

#### 4. HelixCheckbox
- **OpenDesign Group**: `inputs` (custom extension)
- **Token References**:
  - Unchecked: `--border` border
  - Checked: `--accent` fill + white checkmark
  - Indeterminate: `--accent` fill + white dash
  - Focus: `--shadow-brand`
  - Disabled: `--muted` fill
- **Size**: 18×18px box, 4px radius
- **Accessibility**: `Semantics(checked: value, label: labelText)`, 40×40px touch target
- **Flutter Base**: `Checkbox` with custom theming

#### 5. HelixRadio
- **OpenDesign Group**: `inputs` (custom extension)
- **Token References**: Same as Checkbox
- **Size**: 18px outer ring, 8px inner fill
- **Accessibility**: `Semantics(selected: value, label: labelText)`, always in groups of 2+
- **Flutter Base**: `Radio` with custom theming

#### 6. HelixSwitch
- **OpenDesign Group**: `inputs` (custom extension)
- **Token References**:
  - Off: `--border` track, `--muted` thumb
  - On: `--accent` track, white thumb
  - Transition: `--motion-fast` (150ms)
- **Size**: 44×24px track, 20px thumb
- **Accessibility**: `Semantics(toggled: value, label: labelText)`
- **Flutter Base**: `Switch` with custom theming

### 2.2 Feedback Components (7-12)

#### 7. HelixTooltip
- **OpenDesign Group**: — (custom)
- **Token References**:
  - Background: `--surface-overlay`
  - Text: `--fg`
  - Border: `--border-soft`
  - Shadow: `--shadow-md`
  - Padding: `--space-2` `--space-3`
  - Border radius: `--radius-sm`
- **Timing**: 300ms hover delay, 100ms appear, immediate dismiss
- **Max width**: 280px
- **Accessibility**: `Semantics(label: tooltipText)` on trigger
- **Flutter Base**: `Tooltip` (native) wrapped in `HelixTooltip`

#### 8. HelixDivider
- **OpenDesign Group**: — (custom)
- **Token References**:
  - Hairline: `--border-soft`
  - Default: `--border`
  - Strong: `--border-strong`
- **Variants**: Horizontal, Vertical, Section (with label)
- **Flutter Base**: `Divider` or custom `Container` with border

#### 9. HelixToast
- **OpenDesign Group**: — (custom)
- **Token References**:
  - Background: `--surface-overlay`
  - Shadow: `--shadow-md`
  - Border radius: `--radius-md`
  - Success icon: `--success`
  - Warning icon: `--warn`
  - Error icon: `--danger`
  - Info icon: `--accent`
- **Position**: Bottom-center (mobile), bottom-right (desktop)
- **Duration**: 4000ms (success/info), 6000ms (warning/error)
- **Stack**: Up to 3 visible, additional queued
- **Flutter Base**: Custom `OverlayEntry`

#### 10. HelixAlertBanner
- **OpenDesign Group**: — (custom)
- **Token References**: Same color variants as Toast
- **Structure**: Full-width, 4px left color stripe, 12px vertical padding
- **Flutter Base**: `Container` with `Row` content

#### 11. HelixProgressBar
- **OpenDesign Group**: — (custom)
- **Token References**:
  - Track: `--surface-sunken`
  - Fill: `--accent`
  - Indeterminate shimmer: `--surface-overlay` to `--surface-raised`
- **Variants**: Linear (4/6/8px), Circular (16/24/40/64px)
- **Flutter Base**: `LinearProgressIndicator` / `CircularProgressIndicator` with custom theming

#### 12. HelixSkeleton
- **OpenDesign Group**: — (custom)
- **Token References**:
  - Base: `--surface-raised`
  - Shimmer: `--surface-overlay`
  - Animation: `--duration-loop` (1500ms)
- **Shapes**: Text line, Title line, Avatar, Card body, Table row
- **Flutter Base**: `Shimmer` package or custom `AnimatedContainer`

### 2.3 Navigation Components (13-18)

#### 13. HelixSidebar
- **OpenDesign Group**: `layout` (extended)
- **OpenDesign Selectors**: `.container`, `section`
- **Token References**:
  - Background: `--surface`
  - Width expanded: 240px
  - Width collapsed: 56px
  - Item hover: `--surface-interactive`
  - Active indicator: 3px `--accent` left border
  - Active item: `--surface-selected` + `--accent` text
- **Structure**: Logo, Quick Connect, Nav sections (Hosts, Sessions, Tools), Spacer, Notifications, Settings, User
- **Platform**: Expanded sidebar (desktop), Collapsed (tablet), Hidden + bottom tab (mobile)
- **Flutter Base**: `NavigationRail` (tablet) / custom `Drawer` (desktop) / `BottomNavigationBar` (mobile)

#### 14. HelixTabBar
- **OpenDesign Group**: — (custom)
- **Token References**:
  - Height: 40px
  - Active tab: `--surface-raised` + 2px top `--accent` border
  - Inactive tab: `--bg` + `--muted` text
  - Status dot: `--helix-ssh-connected`
  - Close button: `--muted` → `--fg` on hover
- **Behavior**: Horizontal scroll on overflow, context menu on right-click
- **Flutter Base**: Custom `TabBar` with `TabController`

#### 15. HelixBreadcrumb
- **OpenDesign Group**: — (custom)
- **Token References**:
  - Separator: `--muted`
  - Clickable items: `--accent` (link)
  - Current item: `--fg`
- **Overflow**: Truncate middle items as `...` when > 4 segments
- **Flutter Base**: `Row` of `InkWell` + `Text` widgets

#### 16. HelixModal
- **OpenDesign Group**: — (custom)
- **Token References**:
  - Backdrop: `rgba(0,0,0,0.7)`
  - Background: `--surface-raised`
  - Border radius: `--radius-lg` (12px)
  - Shadow: `--shadow-lg`
  - Width: 360/480/560/720px
- **Animation**: Scale 0.95→1.0 + opacity 0→1, 150ms ease-out
- **Flutter Base**: `showDialog` with custom `transitionBuilder`

#### 17. HelixSheet
- **OpenDesign Group**: — (custom)
- **Token References**: Same as Modal
- **Bottom Sheet**: Slides up, handle bar, drag-to-dismiss, 90% max height
- **Side Sheet**: Slides from right, 320-480px width
- **Flutter Base**: `showModalBottomSheet` / custom `HelixSideSheet`

#### 18. HelixDrawer
- **OpenDesign Group**: — (custom)
- **Token References**:
  - Background: `--surface`
  - Width: 280px or 80% screen
  - Backdrop: 50% black
- **Flutter Base**: `Drawer` with custom theming

### 2.4 Layout Components (19-24)

#### 19. HelixCard
- **OpenDesign Group**: `cards`
- **OpenDesign Selectors**: `.card`
- **Token References**:
  - Background: `--surface`
  - Border: `--border-soft`
  - Border radius: `--radius-md` (8px)
  - Shadow: `--shadow-xs`
  - Padding: `--space-4` (16px)
  - Hover (interactive): `--surface-interactive` + `--shadow-sm`
- **Variants**: Default, Flat, Elevated, Interactive, Host Card
- **Flutter Base**: `Card` with custom `CardTheme`

#### 20. HelixPanel
- **OpenDesign Group**: `layout` (extended)
- **Token References**: Same as Card + resize handle styling
- **Features**: Resize handles (4px drag zone), collapse animation (200ms), snap on double-click
- **Flutter Base**: Custom `Row`/`Column` with `GestureDetector` for resize

#### 21. HelixPopover
- **OpenDesign Group**: — (custom)
- **Token References**:
  - Background: `--surface-overlay`
  - Shadow: `--shadow-lg`
  - Arrow: 8px
  - Max width: 320px
- **Flutter Base**: Custom `Overlay` with `CompositedTransformFollower`

#### 22. HelixContextMenu
- **OpenDesign Group**: — (custom)
- **Token References**:
  - Background: `--surface-overlay`
  - Item height: 32-36px
  - Icon: 16px, `--muted`
  - Label: `--fg`
  - Shortcut: `--muted`
  - Destructive: `--danger`
- **Flutter Base**: `CustomSingleChildLayout` with pointer position

#### 23. HelixEmptyState
- **OpenDesign Group**: — (custom)
- **Token References**:
  - Icon: `--muted` (48px)
  - Title: `--muted` (headingMD)
  - Body: `--muted` (bodyBase)
- **Structure**: Icon + Title + Body + Primary CTA + Secondary link
- **Flutter Base**: `Column` with `Center` alignment

#### 24. HelixErrorState
- **OpenDesign Group**: — (custom)
- **Token References**:
  - Icon: `--danger` (48px)
  - Title: `--fg`
  - Body: `--muted`
- **Structure**: Error icon + Title + Error code + Body + Retry button + Support link
- **Flutter Base**: `Column` with `Center` alignment

### 2.5 Data Display Components (25-31)

#### 25. HelixDataTable
- **OpenDesign Group**: — (custom)
- **Token References**:
  - Header height: 40px
  - Row height: 44px
  - Header text: `--fg` (semiBold)
  - Row hover: `--surface-interactive`
  - Row selected: `--surface-selected`
  - Sort indicator: `--muted` (inactive), `--fg` (active)
- **Features**: Sortable, pagination, column resize, row selection, bulk actions, sticky column
- **Flutter Base**: `DataTable2` package with custom theming

#### 26. HelixList
- **OpenDesign Group**: `layout` (extended)
- **OpenDesign Selectors**: `.features-grid`
- **Token References**: Same as Card for items
- **Features**: Virtual scroll (`ListView.builder`), selection modes (none/single/multi)
- **Flutter Base**: `ListView.builder` wrapped in `HelixList`

#### 27. HelixTreeView
- **OpenDesign Group**: — (custom)
- **Token References**:
  - Indent per level: 16px (`--space-4`)
  - Expand icon: `--muted` → rotates 90°
- **Features**: Drag & drop, context menu, expand/collapse animation (150ms)
- **Flutter Base**: Custom `TreeController` with `AnimatedContainer`

#### 28. HelixSFTPBrowser
- **OpenDesign Group**: — (custom)
- **Token References**:
  - Pane background: `--surface`
  - File row hover: `--surface-interactive`
  - Directory icon: `--accent`
  - File icon: `--muted`
  - Selected: `--surface-selected`
- **Layout**: Dual-pane (desktop), Single pane (mobile)
- **Flutter Base**: Custom virtual scroll file list

#### 29. HelixTransferQueue
- **OpenDesign Group**: — (custom)
- **Token References**:
  - Progress bar: `--accent` fill, `--surface-sunken` track
  - Speed text: `--muted`
  - Completed: `--success`
  - Failed: `--danger`
- **Flutter Base**: `Column` of `ListTile` with `LinearProgressIndicator`

#### 30. HelixTag
- **OpenDesign Group**: `badges`
- **OpenDesign Selectors**: `.badge`, `.badge-muted`, `.badge-success`
- **Token References**:
  - Background: `--surface-overlay`
  - Text: `--muted`
  - Border radius: `--radius-full` (pill)
  - Height: 24px
  - Padding: `--space-2` horizontal
- **Color variants**: Neutral, Blue, Teal, Purple, Amber, Red, Custom
- **Flutter Base**: `Chip` with custom theming

#### 31. HelixBadge
- **OpenDesign Group**: `badges`
- **OpenDesign Selectors**: `.badge`
- **Token References**:
  - Background: `--danger`
  - Text: `--accent-on` (white)
  - Height: 16px, min width: 16px
  - Border radius: `--radius-full`
- **Max display**: "99+" for values ≥ 100
- **Flutter Base**: `Badge` (Flutter 3.19+) or custom `Container`

### 2.6 Terminal Components (32-38)

#### 32. HelixTerminal
- **OpenDesign Group**: — (custom, terminal-specific)
- **Token References**:
  - Background: User-selected terminal scheme bg
  - Foreground: User-selected terminal scheme fg
  - Cursor: `--accent` or user-selected
  - Selection: `--accent` at 30% opacity
- **Features**: GPU-accelerated rendering, configurable cursor, scrollback, selection, copy/paste, find
- **Performance**: 60fps target, 4ms max rasterization
- **Flutter Base**: `xterm.dart` or equivalent with custom `RenderBox`

#### 33. HelixTerminalTab
- **OpenDesign Group**: — (custom)
- **Token References**:
  - Active: `--surface-raised` + 2px top `--accent`
  - Inactive: `--bg` + `--muted`
  - Status dot: `--helix-ssh-connected`
  - Recording badge: `--danger` + "REC"
  - Broadcast: `--accent` icon
- **States**: Active, Inactive, Bell (amber flash), Activity (left border pulse), Recording, Broadcast
- **Flutter Base**: Custom `Tab` with `InkWell`

#### 34. HelixTerminalToolbar
- **OpenDesign Group**: — (custom)
- **Token References**:
  - Background: `--surface-overlay` (semi-transparent)
  - Text: `--fg`
  - Icons: `--muted` → `--fg` on hover
- **Behavior**: Auto-show on mouse hover at top, `⌘⇧T` / `Ctrl+Shift+T` toggle
- **Flutter Base**: `AnimatedContainer` with `Opacity`

#### 35. HelixSplitView
- **OpenDesign Group**: — (custom)
- **Token References**:
  - Divider: 4px `--border`
  - Divider hover: `--accent` cursor
- **Layouts**: 1-pane, 2-pane horizontal, 2-pane vertical, 2×2 grid, custom tiling
- **Flutter Base**: `Row`/`Column` with `GestureDetector` dividers

#### 36. HelixFocusModeOverlay
- **OpenDesign Group**: — (custom)
- **Token References**:
  - Chrome opacity: 0% (hidden), 30% (hover edge), 100% (sustained hover)
  - Status dot: `--helix-ssh-connected` (8px corner)
- **Activation**: `⌘⇧F` / `Ctrl+Shift+F` or double-click terminal
- **Flutter Base**: `Stack` with `AnimatedOpacity` layers

#### 37. HelixBroadcastIndicator
- **OpenDesign Group**: — (custom)
- **Token References**:
  - Banner: `--warn` background
  - Text: `--accent-on`
  - Target border: 4px `--accent` left
- **Flutter Base**: `Positioned` `Overlay` entry

#### 38. HelixSessionBadge
- **OpenDesign Group**: — (custom)
- **Token References**:
  - Connected: `--helix-ssh-connected`
  - Connecting: `--helix-ssh-connecting`
  - Disconnected: `--helix-ssh-disconnected`
  - Error: `--helix-ssh-error`
  - Reconnecting: `--helix-ssh-reconnecting`
  - Recording: `--danger`
  - Suspended: `--muted`
- **Flutter Base**: `Container` with `CircleAvatar` or `Icon`

### 2.7 SSH-Specific Components (39-46)

#### 39. HelixHostCard
- **OpenDesign Group**: `cards` (extended)
- **OpenDesign Selectors**: `.card`
- **Token References**:
  - Grid variant (160px): `--surface` bg, `--border-soft` border
  - List variant (44px): Same + row layout
  - Connected state: 3px left `--helix-ssh-connected`
  - Error state: 3px left `--helix-ssh-error`
- **Flutter Base**: `Card` with `InkWell` + custom `BoxDecoration`

#### 40. HelixConnectionDialog
- **OpenDesign Group**: — (custom)
- **Token References**: Same as Modal
- **Content**: Host info, user/port/protocol, auth method, jump host chain, options, actions
- **Flutter Base**: `HelixModal` with custom content

#### 41. HelixJumpHostChip
- **OpenDesign Group**: — (custom)
- **Token References**:
  - Background: `--surface-overlay`
  - Border: `--border`
  - Height: 24px, pill shape
- **Flutter Base**: `Chip` with custom styling

#### 42. HelixProtocolBadge
- **OpenDesign Group**: `badges` (extended)
- **OpenDesign Selectors**: `.badge`
- **Token References**:
  - SSH: `--purple-800` bg
  - Mosh: `--blue-800` bg
  - SFTP: `--teal-800` bg
  - Telnet: `--amber-800` bg
  - RDP: `--red-800` bg
- **Size**: `labelSM`, 6px horizontal padding, 2px vertical, `radius-xs`
- **Flutter Base**: `Container` with `Text`

#### 43. HelixAuthMethodIcon
- **OpenDesign Group**: `icons`
- **OpenDesign Selectors**: `.icon`
- **Token References**:
  - Icon color: `--muted`
  - Size: 20px
- **Methods**: Password, SSH Key, SSH Agent, Certificate, GSSAPI/Kerberos, 2FA/TOTP, Hardware Key (FIDO2)
- **Flutter Base**: `Icon` with `Tooltip`

#### 44. HelixKeyFingerprint
- **OpenDesign Group**: — (custom)
- **Token References**:
  - Background: `--surface-sunken`
  - Text: `--fg` (monospace)
  - Border radius: `--radius-sm`
  - Padding: `--space-2` `--space-3`
- **Features**: Copy button, randomart visualization toggle
- **Flutter Base**: `Container` with `SelectableText`

#### 45. HelixPortForwardRow
- **OpenDesign Group**: — (custom)
- **Token References**:
  - Active indicator: `--helix-ssh-connected`
  - Inactive: `--muted`
- **Anatomy**: Type badge → local-port → remote-host:remote-port → status → actions
- **Flutter Base**: `ListTile` with custom leading/trailing

#### 46. HelixSnippetCard
- **OpenDesign Group**: `cards` (extended)
- **OpenDesign Selectors**: `.card`
- **Token References**:
  - Background: `--surface`
  - Code preview: `--fg` (monospace)
  - Syntax highlighting: Helix terminal colors
- **Flutter Base**: `Card` with `Column` containing code preview

### 2.8 Utility Components (47-50)

#### 47. HelixDatePicker
- **OpenDesign Group**: — (custom)
- **Token References**:
  - Selected date: `--accent` filled circle
  - Today: `--accent` outline
  - Disabled: `--muted`
  - Range: `--surface-selected` fill
- **Flutter Base**: `HelixModal` with custom calendar grid

#### 48. HelixColorPicker
- **OpenDesign Group**: — (custom)
- **Token References**: Uses primitive color tokens for presets
- **Flutter Base**: Custom canvas with `GestureDetector`

#### 49. HelixCommandPalette
- **OpenDesign Group**: — (custom)
- **Token References**:
  - Backdrop: 50% black
  - Palette: `--surface-raised`
  - Width: 600-800px
  - Selected item: `--surface-selected`
- **Animation**: Slide down 20px + fade, 150ms ease-out
- **Flutter Base**: Custom `Overlay` with `TextField` + `ListView`

#### 50. HelixAppShell
- **OpenDesign Group**: `layout` (extended)
- **OpenDesign Selectors**: `.container`, `section`, `section + section`
- **Token References**:
  - Background: `--bg`
  - Sidebar: `--surface`
  - Content area: `--bg`
  - Status bar: `--surface-raised`
- **Structure**: Sidebar + Main Content + Optional Detail Panel + Status Bar
- **Flutter Base**: `Scaffold` with custom `NavigationRail`/`Drawer`/`BottomNavigationBar`

---

## 3. OpenDesign Mapping Details

### 3.1 Direct Mappings (✅)

These components map cleanly to OpenDesign without extensions:

| Helix Component | OpenDesign Group | Mapping Confidence |
|-----------------|------------------|-------------------|
| HelixButton | `buttons` | 95% — Size/variant tokens extend OD base |
| HelixTextInput | `inputs` | 90% — Password/search variants are Helix-specific |
| HelixTag | `badges` | 85% — Color variants extend OD base |
| HelixBadge | `badges` | 85% — Numeric count is Helix-specific |
| HelixCard | `cards` | 90% — Host/elevated variants are Helix-specific |
| HelixAuthMethodIcon | `icons` | 80% — Icon set is Helix-specific |
| HelixAppShell | `layout` | 75% — Complex nav structure extends OD base |

### 3.2 Extended Mappings (⚠️)

These components map to OpenDesign but require Helix-specific extensions:

| Helix Component | OpenDesign Group | Extensions Needed |
|-----------------|------------------|-------------------|
| HelixSelect | `inputs` | Dropdown panel styling, searchable variant |
| HelixCheckbox | `inputs` | Add to custom manifest |
| HelixRadio | `inputs` | Add to custom manifest |
| HelixSwitch | `inputs` | Add to custom manifest |
| HelixSidebar | `layout` | Nav sections, collapse behavior, platform variants |
| HelixModal | — | Backdrop, sizing, animation tokens |
| HelixSheet | — | Bottom/side variants, drag-to-dismiss |
| HelixDrawer | — | Mobile nav drawer |
| HelixHostCard | `cards` | Status indicators, OS icons, connection states |
| HelixProtocolBadge | `badges` | Protocol color mapping |
| HelixSnippetCard | `cards` | Code preview, syntax highlighting |

### 3.3 Custom Components (🔶)

These have no OpenDesign equivalent and must be added to the custom manifest:

| Helix Component | Category | Priority |
|-----------------|----------|----------|
| HelixTooltip | Feedback | P1 |
| HelixDivider | Layout | P1 |
| HelixToast | Feedback | P0 |
| HelixAlertBanner | Feedback | P1 |
| HelixProgressBar | Feedback | P1 |
| HelixSkeleton | Feedback | P1 |
| HelixEmptyState | Feedback | P1 |
| HelixErrorState | Feedback | P1 |
| HelixTabBar | Navigation | P0 |
| HelixBreadcrumb | Navigation | P1 |
| HelixPopover | Layout | P1 |
| HelixContextMenu | Layout | P1 |
| HelixDataTable | Data Display | P1 |
| HelixList | Data Display | P0 |
| HelixTreeView | Data Display | P1 |
| HelixSFTPBrowser | Data Display | P1 |
| HelixTransferQueue | Data Display | P1 |
| HelixTerminal | Terminal | P0 |
| HelixTerminalTab | Terminal | P0 |
| HelixTerminalToolbar | Terminal | P1 |
| HelixSplitView | Terminal | P1 |
| HelixFocusModeOverlay | Terminal | P2 |
| HelixBroadcastIndicator | Terminal | P2 |
| HelixSessionBadge | SSH | P0 |
| HelixConnectionDialog | SSH | P0 |
| HelixJumpHostChip | SSH | P2 |
| HelixKeyFingerprint | SSH | P1 |
| HelixPortForwardRow | SSH | P1 |
| HelixDatePicker | Utility | P2 |
| HelixColorPicker | Utility | P2 |
| HelixCommandPalette | Utility | P2 |

---

## 4. Token Reference Matrix

### 4.1 Component → Token Cross-Reference

| Component | Colors | Typography | Spacing | Shadows | Motion |
|-----------|--------|-----------|---------|---------|--------|
| HelixButton | 8 | 2 | 4 | 0 | 3 |
| HelixTextInput | 6 | 2 | 3 | 1 | 2 |
| HelixCard | 5 | 0 | 3 | 2 | 2 |
| HelixTag | 3 | 1 | 3 | 0 | 0 |
| HelixBadge | 2 | 0 | 2 | 0 | 0 |
| HelixModal | 3 | 0 | 4 | 2 | 2 |
| HelixToast | 6 | 1 | 3 | 1 | 3 |
| HelixSidebar | 5 | 1 | 4 | 0 | 2 |
| HelixTerminal | 4 | 1 | 0 | 0 | 1 |
| HelixHostCard | 6 | 2 | 3 | 1 | 2 |
| HelixAppShell | 4 | 0 | 2 | 0 | 1 |
| **Total** | **~120** | **~20** | **~50** | **~15** | **~25** |

### 4.2 Token Consumption by Category

| Token Category | Components Using | Total References |
|----------------|-----------------|-----------------|
| `color.semantic.surface*` | 48 | ~180 |
| `color.semantic.text*` | 46 | ~200 |
| `color.semantic.interactive*` | 28 | ~80 |
| `color.semantic.border*` | 35 | ~100 |
| `color.semantic.ssh-*` | 12 | ~30 |
| `component.button.*` | 8 | ~25 |
| `component.input.*` | 6 | ~20 |
| `component.card.*` | 15 | ~40 |
| `shadow.*` | 22 | ~50 |
| `duration.*` | 30 | ~60 |
| `easing.*` | 25 | ~40 |

---

## 5. Platform Adaptations

### 5.1 Web
- All components use CSS custom properties for theming
- Responsive breakpoints: `mobile-sm` (320px) through `desktop-xl` (1920px+)
- Focus rings use `:focus-visible` pseudo-class
- Scrollbars styled with `::-webkit-scrollbar`

### 5.2 macOS
- Sidebar uses `NSVisualEffectView` vibrancy option
- Native menu bar integration
- Touch Bar support for terminal tabs
- `⌘` prefix for all keyboard shortcuts

### 5.3 Windows
- Acrylic material option in sidebar
- Custom title bar with system buttons
- Mica material on Windows 11
- `Ctrl` prefix for keyboard shortcuts

### 5.4 Linux
- GTK theme respect for file dialogs
- xdg-desktop-portal integration
- System font fallback

### 5.5 iOS
- Safe area handling for notch/Dynamic Island
- Bottom sheet for modals
- Haptic feedback on connection events
- Custom keyboard accessory view

### 5.6 Android
- Edge-to-edge display
- Material You dynamic color (optional)
- Bottom navigation on phones
- Native haptic patterns

### 5.7 HarmonyOS
- HarmonyOS Sans font
- HMS service integration
- Distributed UI capabilities

### 5.8 AuroraOS
- Silica UI pulley menu patterns
- Gesture navigation
- System ambience respect

---

## 6. Accessibility Requirements

### 6.1 Per-Component WCAG Requirements

| Component | WCAG 2.1 Requirements | Flutter Implementation |
|-----------|----------------------|----------------------|
| HelixButton | 2.4.7 Focus Visible, 1.4.3 Contrast | Focus ring, `Semantics(button: true)` |
| HelixTextInput | 3.3.1 Error Identification, 1.4.3 Contrast | `errorText`, `Semantics(label:)` |
| HelixCheckbox | 1.4.11 Non-text Contrast, 4.1.2 Name/Role/Value | `Semantics(checked:)`, 40×40px target |
| HelixRadio | Same as Checkbox | `Semantics(selected:)` |
| HelixSwitch | Same as Checkbox | `Semantics(toggled:)` |
| HelixTooltip | 1.4.13 Content on Hover | 300ms delay, dismissible |
| HelixModal | 2.4.3 Focus Order, 2.4.7 Focus Visible | Focus trap, `barrierDismissible` |
| HelixToast | 4.1.3 Status Messages | `SemanticsService.announce` |
| HelixTerminal | 1.3.1 Info and Relationships | Transcript mode, output summary |
| HelixDataTable | 1.3.1 Info and Relationships | Column headers, sort indicators |

### 6.2 Screen Reader Support

All components implement `Semantics` with:
- `label`: Descriptive name
- `hint`: Usage instruction
- `value`: Current state/value
- `enabled`: Interactive state
- `button`, `link`, `header`, `image` roles as appropriate

### 6.3 Keyboard Navigation

| Component | Tab Order | Arrow Keys | Enter/Space | Escape |
|-----------|-----------|-----------|-------------|--------|
| HelixButton | Focusable | — | Activate | — |
| HelixTextInput | Focusable | Cursor move | Submit | — |
| HelixCheckbox | Focusable | — | Toggle | — |
| HelixRadio | Focusable | Next/prev in group | Select | — |
| HelixSelect | Focusable | Open/Navigate | Select | Close |
| HelixModal | Trap focus | — | Primary action | Close |
| HelixContextMenu | — | Navigate items | Activate | Close |
| HelixCommandPalette | Focus input | Navigate results | Execute | Close |
| HelixTerminal | Focusable | Scroll | — | — |

---

*HelixTerminator Component Library Specification v1.0.0*  
*All content conforms to CANONICAL_FACTS.md and SERVICE_REGISTRY.md*
