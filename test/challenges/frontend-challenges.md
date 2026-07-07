# Frontend Challenges

> Coding challenges for Flutter developers working on the HelixTerminator client.

## Challenge 1: Offline-First Terminal

**Difficulty:** Hard
**Client:** Flutter
**Time:** 6 hours

Build an offline-first terminal experience:
- Queue commands when disconnected
- Sync history when connection restores
- Visual indicator of sync status
- Handle conflicts (e.g., server-side state changed while offline)

### Acceptance Criteria
- [ ] Commands are queued locally with SQLite
- [ ] Reconnection triggers automatic sync
- [ ] UI clearly shows online/offline/syncing states
- [ ] Conflict resolution UI is accessible and intuitive

---

## Challenge 2: Real-Time Collaboration Cursor

**Difficulty:** Medium
**Client:** Flutter
**Time:** 3 hours

Implement real-time collaborative cursors in a text editor:
- Show other users' cursors and selections
- Smooth cursor movement via interpolation
- Handle users joining and leaving gracefully
- Minimal performance impact with 50+ concurrent users

### Acceptance Criteria
- [ ] Cursors are rendered at correct positions
- [ ] Interpolation is smooth (60fps)
- [ ] Memory usage does not grow with user count
- [ ] Cursor colors are unique and accessible

---

## Challenge 3: Biometric Auth Integration

**Difficulty:** Medium
**Client:** Flutter
**Time:** 2 hours

Integrate biometric authentication (Face ID / Touch ID / Fingerprint):
- Securely store credentials in platform keychain
- Fallback to PIN/password
- Handle biometric hardware unavailability
- Cross-platform: iOS, Android, Web (where supported)

### Acceptance Criteria
- [ ] Biometric prompt appears on app launch (configurable)
- [ ] Credentials are never stored in plain text
- [ ] Fallback works seamlessly
- [ ] Platform differences are handled gracefully

---

## Challenge 4: Responsive SSH Terminal

**Difficulty:** Medium
**Client:** Flutter
**Time:** 3 hours

Build a responsive SSH terminal widget:
- Adapt to mobile, tablet, and desktop layouts
- Support touch gestures (scroll, zoom, selection)
- Handle keyboard shortcuts on desktop
- Proper text rendering with Unicode and colors

### Acceptance Criteria
- [ ] Layout adapts to screen size and orientation
- [ ] Touch gestures are intuitive and responsive
- [ ] Keyboard shortcuts do not conflict with host OS
- [ ] Unicode and ANSI colors render correctly

---

## Challenge 5: Secure File Transfer UI

**Difficulty:** Medium
**Client:** Flutter
**Time:** 3 hours

Design and implement a secure file transfer UI:
- Drag-and-drop support (desktop)
- Progress tracking with cancel option
- Transfer history with search and filter
- Virus scan status indicator

### Acceptance Criteria
- [ ] Drag-and-drop works on macOS, Windows, Linux
- [ ] Progress is accurate and updates in real time
- [ ] Cancel stops the transfer and cleans up partial files
- [ ] History persists across app restarts

---

## Challenge 6: Dark Mode & Accessibility

**Difficulty:** Easy
**Client:** Flutter
**Time:** 2 hours

Implement a comprehensive theming system:
- Dark mode, light mode, and system default
- High contrast mode for accessibility
- Font size scaling (100%, 125%, 150%)
- All colors meet WCAG 2.1 AA contrast ratios

### Acceptance Criteria
- [ ] Theme changes apply instantly without restart
- [ ] High contrast mode is detectable by screen readers
- [ ] Font scaling does not break layouts
- [ ] Contrast ratios verified with automated tests

---

## Challenge 7: Notification Management

**Difficulty:** Medium
**Client:** Flutter
**Time:** 2 hours

Build a notification center:
- Group notifications by service and priority
- Swipe-to-dismiss with undo
- Push notification integration (Firebase/APNs)
- Badge count sync with backend

### Acceptance Criteria
- [ ] Notifications are grouped logically
- [ ] Swipe-to-dismiss is smooth and accessible
- [ ] Push notifications arrive reliably
- [ ] Badge count is accurate and synced

---

## Challenge 8: Performance Dashboard Widget

**Difficulty:** Hard
**Client:** Flutter
**Time:** 4 hours

Create a real-time performance dashboard:
- Charts: CPU, memory, network latency
- Data streams via WebSocket
- Efficient rendering with 60fps target
- Export data as CSV/PNG

### Acceptance Criteria
- [ ] Charts update smoothly without jank
- [ ] WebSocket reconnects automatically
- [ ] Export produces valid files
- [ ] Memory usage is stable over long sessions

---

## Challenge 9: Multi-Factor Authentication Flow

**Difficulty:** Medium
**Client:** Flutter
**Time:** 3 hours

Implement a complete MFA enrollment and verification flow:
- TOTP (QR code scanning)
- WebAuthn / FIDO2 (security keys)
- Backup codes generation and storage
- Step-up authentication for sensitive actions

### Acceptance Criteria
- [ ] QR code scanning works on iOS and Android
- [ ] Security key registration follows WebAuthn spec
- [ ] Backup codes are displayed securely (one-time)
- [ ] Step-up auth triggers appropriately

---

## Challenge 10: Workspace Switcher

**Difficulty:** Easy
**Client:** Flutter
**Time:** 2 hours

Build a workspace switcher:
- Quick switch with keyboard shortcut (Ctrl/Cmd+K)
- Search and filter workspaces
- Recent workspaces list
- Workspace creation wizard

### Acceptance Criteria
- [ ] Keyboard shortcut is responsive
- [ ] Search filters in real time
- [ ] Recent list is persisted locally
- [ ] Wizard validates input before submission

---

## Submission Guidelines

1. Fork the repository and create a branch: `challenge/<your-name>-<challenge-number>`
2. Target Flutter 3.24.0
3. Include widget tests and integration tests
4. Open a draft PR for review
5. Tag `@helix-frontend-reviewers` for feedback

## Scoring

- **Pass:** All acceptance criteria met, tests pass, no analyzer warnings
- **Merit:** Pass + performance exceeds baseline, smooth animations
- **Distinction:** Merit + reusable widget or package contribution
