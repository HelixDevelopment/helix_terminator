# Security Challenges

> Security-focused challenges for HelixTerminator security engineers and developers.

## Challenge 1: Auth Bypass Detection

**Difficulty:** Hard
**Time:** 4 hours

Write a system that detects and prevents authentication bypass attempts:
- Detect JWT manipulation (alg=none, key confusion)
- Detect session fixation and hijacking
- Rate-limit and lock out suspicious IPs
- Alert security team with full request context

### Acceptance Criteria
- [ ] `alg=none` is rejected immediately
- [ ] Key confusion attacks are prevented
- [ ] Session anomalies trigger alerts
- [ ] False positive rate < 0.1%

---

## Challenge 2: SQL Injection Fuzzing Harness

**Difficulty:** Medium
**Time:** 3 hours

Build a fuzzing harness that tests all service APIs for SQL injection:
- Generate payloads based on SQLi patterns (boolean-based, time-based, union)
- Integrate with CI/CD to run on every PR
- Report vulnerable endpoints with proof-of-concept
- Track remediation status

### Acceptance Criteria
- [ ] All API endpoints are tested
- [ ] CI fails if new vulnerabilities are introduced
- [ ] Reports include curl commands to reproduce
- [ ] Historical trends are tracked

---

## Challenge 3: XSS Prevention Pipeline

**Difficulty:** Medium
**Time:** 2 hours

Implement a comprehensive XSS prevention pipeline:
- Content Security Policy (CSP) generation and validation
- Input sanitization library for all user-facing outputs
- DOM-based XSS detection in frontend code
- Automated CSP violation reporting

### Acceptance Criteria
- [ ] CSP is strict and blocks inline scripts
- [ ] All user input is sanitized before rendering
- [ ] DOM-based XSS vectors are detected in CI
- [ ] Violation reports are triaged within 24 hours

---

## Challenge 4: CSRF Protection Audit

**Difficulty:** Easy
**Time:** 2 hours

Audit and harden CSRF protection:
- Verify SameSite cookie attributes
- Validate anti-CSRF tokens on all state-changing requests
- Ensure tokens are cryptographically random
- Test bypass scenarios

### Acceptance Criteria
- [ ] All cookies use `SameSite=Lax` or `SameSite=Strict`
- [ ] State-changing endpoints require valid tokens
- [ ] Tokens are unpredictable and bound to session
- [ ] Bypass attempts are logged and alerted

---

## Challenge 5: Privilege Escalation Detection

**Difficulty:** Hard
**Time:** 5 hours

Build a system that detects privilege escalation attempts:
- Monitor for role changes outside normal workflows
- Detect horizontal privilege escalation (accessing other users' data)
- Detect vertical privilege escalation (gaining admin rights)
- Real-time alerting with forensic context

### Acceptance Criteria
- [ ] Unauthorized role changes trigger immediate alerts
- [ ] Horizontal escalation is detected within 5 seconds
- [ ] Vertical escalation attempts are blocked and logged
- [ ] Forensic data includes full request chain

---

## Challenge 6: Secret Scanning Automation

**Difficulty:** Medium
**Time:** 2 hours

Automate secret scanning across the entire codebase:
- Pre-commit hooks with TruffleHog or Gitleaks
- CI scanning for historical leaks
- Automatic rotation of leaked secrets
- Integration with secret management system

### Acceptance Criteria
- [ ] No secrets can be committed to Git
- [ ] Historical scans run quarterly
- [ ] Leaked secrets are rotated within 1 hour of detection
- [ ] Scan results are tracked to resolution

---

## Challenge 7: Penetration Test Automation

**Difficulty:** Hard
**Time:** 6 hours

Automate penetration testing:
- OWASP ZAP baseline and full scans
- Nuclei template-based scanning
- Custom HelixTerminator-specific test cases
- Scheduled scans with trend analysis

### Acceptance Criteria
- [ ] Scans cover all exposed endpoints
- [ ] Findings are triaged and assigned automatically
- [ ] Trend analysis shows security posture over time
- [ ] Critical findings trigger immediate alerts

---

## Challenge 8: Supply Chain Security

**Difficulty:** Medium
**Time:** 3 hours

Harden the software supply chain:
- SLSA Level 3 compliance for all artifacts
- Signed container images with cosign
- SBOM generation and attestation
- Dependency vulnerability scanning

### Acceptance Criteria
- [ ] All artifacts have provenance attestations
- [ ] Images are signed and signatures verified on deploy
- [ ] SBOMs are generated for every build
- [ ] Vulnerable dependencies are flagged in CI

---

## Challenge 9: Zero Trust Network Architecture

**Difficulty:** Hard
**Time:** 6 hours

Implement a zero trust network architecture:
- Mutual TLS for all service-to-service communication
- Identity-aware proxy (IAP) for all ingress
- Device trust scoring
- Continuous authentication with step-up

### Acceptance Criteria
- [ ] No service trusts another based on network position alone
- [ ] All traffic is encrypted and authenticated
- [ ] Device trust is evaluated per-request
- [ ] Step-up auth triggers on risk score changes

---

## Challenge 10: Incident Response Automation

**Difficulty:** Hard
**Time:** 5 hours

Automate security incident response:
- Auto-containment of compromised accounts
- Forensic snapshot preservation
- Communication templates for stakeholders
- Post-incident report generation

### Acceptance Criteria
- [ ] Compromised accounts are disabled within 60 seconds
- [ ] Evidence is preserved without alerting the attacker
- [ ] Stakeholders are notified with appropriate urgency
- [ ] Reports include timeline, impact, and recommendations

---

## Submission Guidelines

1. Fork the repository and create a branch: `challenge/<your-name>-<challenge-number>`
2. Include proof-of-concept code, tests, and documentation
3. Open a draft PR for review
4. Tag `@helix-security-reviewers` for feedback

## Scoring

- **Pass:** All acceptance criteria met, no false positives, docs complete
- **Merit:** Pass + automation is fully integrated into CI/CD
- **Distinction:** Merit + novel detection technique or reusable tool
