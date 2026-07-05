# 10 — UX Design System

**Status:** `Draft`  
**Module:** A (Secure Terminal Platform)  
**Authority:** `CANONICAL_FACTS.md` (CD-4: Flutter 3.24)  

---

## Overview

HelixTerminator's design system provides a unified, accessible, and performant user experience across all 6 platforms. The system is built on 750+ design tokens, 35 reusable components, and 25 screen wireframes.

| Statistic | Count |
|-----------|-------|
| Design tokens | 750+ |
| Reusable components | 35 |
| Screen wireframes | 25 |
| Keyboard shortcuts | 130+ |
| Terminal themes | 6 (Dracula, Gruvbox, Nord, One Dark, Solarized, Helix Dark) |
| Platform token sets | 9 (Web, macOS, Windows, Linux, iOS, Android, AuroraOS, HarmonyOS) |

---

## Design Tokens

Tokens are stored as JSON and consumed by the Flutter `ThemeExtension` system. Platform-specific overrides allow native look-and-feel while maintaining brand consistency.

### Token Categories

| Category | Examples |
|----------|----------|
| Color | `helixPrimary`, `helixSurface`, `helixError`, `terminalBackground` |
| Typography | `headingLarge`, `bodyRegular`, `monospaceSmall` |
| Spacing | `spaceXS`, `spaceMD`, `spaceXL` |
| Elevation | `shadowCard`, `shadowModal`, `shadowToast` |
| Border | `radiusSM`, `radiusMD`, `radiusFull` |
| Motion | `durationFast`, `durationNormal`, `easeInOut` |
| Breakpoint | `mobile`, `tablet`, `desktop`, `wide` |

### Terminal Themes

| Theme | Background | Foreground | Accent |
|-------|-----------|------------|--------|
| Helix Dark | `#0d1117` | `#c9d1d9` | `#58a6ff` |
| Dracula | `#282a36` | `#f8f8f2` | `#bd93f9` |
| Gruvbox | `#282828` | `#ebdbb2` | `#b8bb26` |
| Nord | `#2e3440` | `#d8dee9` | `#88c0d0` |
| One Dark | `#282c34` | `#abb2bf` | `#61afef` |
| Solarized | `#002b36` | `#839496` | `#268bd2` |

---

## Component Library

### Core Components (35 total)

| Category | Components |
|----------|------------|
| Navigation | `NavRail`, `NavBar`, `Breadcrumb`, `CommandPalette` |
| Data Display | `HostCard`, `VaultItemCard`, `SessionTile`, `SnippetCard`, `DataTable` |
| Input | `TerminalInput`, `SearchField`, `FormField`, `PasswordField`, `TagInput` |
| Feedback | `Toast`, `Banner`, `ProgressBar`, `SkeletonLoader`, `EmptyState` |
| Overlay | `Modal`, `Drawer`, `Popover`, `Tooltip`, `ContextMenu` |
| Terminal | `TerminalPane`, `SplitPane`, `TabBar`, `ScrollbackBuffer` |
| Collaboration | `ParticipantList`, `CursorOverlay`, `ChatPanel`, `PermissionBadge` |

### Accessibility

- WCAG 2.1 AA compliance target
- Minimum contrast ratio 4.5:1 for normal text, 3:1 for large text
- Touch target minimum 44×44 dp (mobile)
- Keyboard navigation for all interactive elements
- Screen reader support (VoiceOver, TalkBack, NVDA, JAWS)

---

## Keyboard Shortcuts

### Global (130+ shortcuts)

| Shortcut | Action |
|----------|--------|
| `⌘K` | Command Palette |
| `⌘T` | New terminal tab |
| `⌘W` | Close current tab |
| `⌘N` | New window |
| `⌘Shift+F` | Find in terminal |
| `⌘Shift+K` | Clear terminal |
| `⌘Shift+Z` | Redo |
| `⌘1..9` | Switch to tab N |
| `⌘+` | Zoom in |
| `⌘-` | Zoom out |
| `⌘0` | Reset zoom |
| `⌘Shift+S` | Save session snippet |
| `⌘Shift+R` | Start recording |
| `⌘Shift+C` | Copy selection |
| `⌘Shift+V` | Paste |
| `Ctrl+Space` | AI autocomplete |
| `F11` | Fullscreen |

> **Note:** Shortcut collisions identified in source doc (⌘K = Command Palette AND Clear terminal; ⌘ShiftZ = Redo AND Suspend-to-background) — flagged for resolution.

---

## Wireframes

25 screen wireframes covering:

1. Login / MFA
2. Dashboard / Host list
3. Terminal session (single + split-pane)
4. SFTP file manager
5. Vault (list + item detail)
6. Snippet editor
7. Workspace manager
8. Organization / Team settings
9. User profile / Preferences
10. Collaboration panel
11. AI suggestion overlay
12. Session recording player
13. Port-forward manager
14. Settings (general, terminal, security)
15. Billing / Subscription
16. Audit log viewer
17. Compliance dashboard
18. Kubernetes pod explorer
19. Onboarding flow
20. Invite / Member management
21. Role / Permission editor
22. Notification center
23. Command palette
24. Search / Global find
25. Help / Documentation

> **DEFERRED:** Vault/Credential Manager, Org/Team, and Billing wireframes are partially specified or missing in source doc 06.

---

## OpenDesign Integration

The `opendesign/` directory contains platform-specific design token exports and icon assets for the OpenDesign system (§11.4.162).

| File | Description |
|------|-------------|
| `design-tokens.json` | Base token set |
| `design-tokens.<platform>.json` | Platform overrides (9 platforms) |
| `terminal-themes/*.json` | 6 terminal color schemes |
| `icons/*.svg` | 50+ icon assets |
| `component-library-spec.md` | Component API specification |
| `INTEGRATION_PLAN.md` | OpenDesign integration roadmap |

---

## Cross-References

- [06 — Client Specification](../06-client-specification/) — Flutter architecture, BLoC pattern
- [09 — Security — Zero Trust](../09-security-zero-trust/) — Security UX requirements
- [16 — References](../16-references/) — Canonical facts

---

*Section 10 — UX Design System*  
*Consolidated from: 06_ux_design_system.md, CANONICAL_FACTS.md (CD-4)*
