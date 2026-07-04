# HelixTerminator MVP Spec — MASTER REMEDIATION REGISTER

Synthesized from six independent audits (A1–A6) covering all 12 source docs
(`01_core_architecture` … `12_mermaid_diagrams` + `README`).
Source docs live in `docs/research/mvp/output/docs/markdown/`. Evidence pointers
(`doc:line` / `§`) are preserved verbatim from the source reports.

Audit → doc coverage map:
- **A1** → `06_ux_design_system`
- **A2** → `07_api_and_database`, `05_security_zero_trust`
- **A3** → `08_product_roadmap_features`, `03_testing_strategy`
- **A4** → `01_core_architecture`, `10_submodule_integration`
- **A5** → `09_performance_analysis`, `02_client_specification`
- **A6** → `11_constitution_compliance`, `04_devops_infrastructure`, `12_mermaid_diagrams`, `README` + full cross-doc sweep

---

## 1. EXECUTIVE SUMMARY

**Raw findings across the six reports:** 276 (A1 33, A2 103, A3 28, A4 32, A5 23, A6 57).
**After cross-report de-duplication:** **≈253 distinct findings** (~15–18 cross-cutting duplicates collapsed — see themes).

**Severity roll-up (normalized to Critical / Important / Minor; A5's High folded into Important, A2/A5 Improvement+Diagram counted separately):**

| Bucket | Count |
|---|---|
| Critical (C) | ~55 |
| Important / High (I/H) | ~100 |
| Minor / Low (M/L) | ~98 |
| Constructive (Improvement + Diagram-Need, not severity-graded) | ~45 |

> Note: exact severity totals are approximate because the six reports used two scales (C/I/M vs C/H/M/L). The register is organized by *action class* (Canonical Decision / Fix-Now / Deep-Work), which is what drives the fix waves — not by severity.

### Dominant systemic themes (the register is really ~10 root causes)

1. **Verbatim-duplication document corruption.** `02_client_specification` repeats a ~1,340-line block (§9→Appendix C) twice with a spliced/unparseable Dart fragment and a double "End of document" marker (A5, 02:L6884-8225, L5351). `06_ux_design_system` is two unrelated documents interleaved under duplicate section numbers 2–10 (A1, 06:L197 vs L812 etc.). `01_core_architecture` has §4.15-4.20 stranded after §10 (A4, 01:L5341/L6142).
2. **≥4 mutually incompatible "25-microservice" enumerations.** Distinct sets in `01`, `10` (three internal variants), `04`, and `11` (two variants). No two agree on the full list (A4, A6).
3. **Product-identity schism.** `11_constitution_compliance` describes a generic **WireGuard/VPN/network-termination** product (`pb.Protocol_WIREGUARD`, 11:L1253+; "network termination and session management platform", 11:L645) while 9 of 12 docs describe an **SSH/vault/terminal/collaboration** platform (A6).
4. **Pervasive version drift — across AND within docs.** PostgreSQL (16.2 / 16.3 / 16 / 17.0 / 17-alpine / 17.2 — five docs, self-contradiction inside `04`), Go (1.23 vs 1.25), Kafka (3.7 vs 3.9), Kubernetes (1.30 vs 1.31), Flutter (3.22 vs 3.24), Istio (1.21+ vs 1.22) (A3, A4, A6).
5. **Multiple conflicting "canonical" `helix-deps.yaml` / submodule catalogues.** 2–3 files with different schemas, submodule membership, and every version pin; submodule versions in `01` (v0.x) vs `10` (v1.x–v2.x); dot-path vs slash-path import conventions that cannot co-compile (A4, A6).
6. **Org / repo / domain identity stated 4–5 ways.** `vasic-digital` vs `digital-vasic`; `HelixDevelopment` vs `helixterm` vs `HelixTerminator`; `helixterm.io` vs `helixterminator.io` (A4, A6).
7. **"Zero-knowledge" claim is not achieved.** Server-side SSH key generation (07:L3285), `auth_method=password` plaintext handling, plaintext `TeamVault.VaultKey` (05:L2669), Shamir share transport unspecified — all contradict "server never sees plaintext" (07:L1674 / 05:L2348) (A1, A2).
8. **Multi-tenant isolation & audit tamper-evidence are theatrical.** Zero Row-Level Security anywhere in `07` §17–20; audit hash chain omits PII columns, is superuser-bypassable (RULE/RLS only), has no external WORM anchor, and collides with the GDPR-erasure UPDATE that silently no-ops (A2).
9. **Governance mandates contradicted by the actual pipeline.** Rootless-Podman mandate (§11.4.161) never implemented (CI uses Docker Buildx); `:latest` used in prod manifests (violates the doc's own hard-forbidden rule); K8s names lack the mandated `helixterm-` prefix; Kafka topics lack the mandated `helix.terminator.` prefix; a `--check` CI dispatch flag is a bluff gate (never read) (A6).
10. **Rigor cliff after Phase 1 / thin acceptance criteria.** Roadmap Phases 2–5 drop Acceptance-Criteria/Test/DoD; empty-body stub test violates the Anti-Bluff Covenant; fabricated image digests; contradictory 12-vs-17 test-type mandate (A3, A6).
11. **Real-time collaboration under-specified everywhere.** No perf/latency budget (09), no BLoC/transport/CRDT client spec (02), no wireframe (06) — for a product marketed as a collaboration platform (A1, A5).
12. **Enterprise resilience gaps.** No PostgreSQL DR/HA/failover/PITR, no RPO/RTO, no backup cadence for 22 per-service DBs (01); no cost/FinOps section (04); no client auto-update or mobile background-execution spec (02).

---

## 2. CANONICAL DECISIONS REQUIRED

High-blast-radius conflicts that CANNOT be fixed until a single source of truth is chosen. Format: **Decision — distinct values (asserting doc) — blast radius — RECOMMENDED default + one-line justification.**

**CD-1 — Product identity.**
Values: *SSH/vault/terminal/collaboration client platform* (README:L3, 12:L3, and 01/02/05/07/08/09/10) vs *WireGuard/VPN network-termination & session-management platform* (11:L645, WireGuard test code 11:L1253/1285/1900/1930/2113).
Blast radius: rewrite `11_constitution_compliance` §1.2.2, §2.11, Appendix A.2, and all §6 WireGuard test examples; align AGENTS.md deployment block.
**RECOMMENDED: SSH/vault/terminal/collaboration platform.** Justification: 9 of 12 docs + the README + all diagrams already center this; doc 11 is the lone outlier and reads as boilerplate from a different project.

**CD-2 — Canonical org / repo / domain name.**
Values: `vasic-digital` (10 §1.1 origin), `digital-vasic` (04 Appendix A helix-deps, reversed), `HelixDevelopment` (11:L45 `HelixDevelopment/HelixConstitution`), `helixterm`/`helixterm.io` (04:L2070, 11:L252), `HelixTerminator` (04:L7935/8143), domain `helixterminator.io` (08:L1862/2703/2822).
Blast radius: 04, 05, 08, 10, 11, README.
**RECOMMENDED: GitHub org `HelixDevelopment`; module/domain `helixterm.io`.** Justification: prompt confirms the real git remote is `github.com/HelixDevelopment`; `helixterm.io` is the pervasive module path (08's `helixterminator.io` is the outlier).

**CD-3 — Canonical 25-microservice list.**
Values: `01` SSH/vault set (`gateway,auth,user,vault,host,ssh-proxy,terminal,sftp,port-forward,snippet,keychain,workspace,collab,notification,audit,analytics,ai,recording,pki,org,billing,config,health,container-bridge,helixtrack-bridge`); `10` §1.2 / go.work / Appendix-D (three internal variants adding `scheduler,file-manager,identity,team,secret,webhook,search,onboarding,rbac,session,challenge`); `04` repo tree (fourth set: `credential-manager,rbac-service,key-rotation-service,compliance-service,tunnel-service,policy-engine,file-transfer-service,…`); `11` §2.11 vs Appendix A.2 (two more, VPN-flavored).
Blast radius: 01, 04, 10, 11, 12 (and every doc that names a service).
**RECOMMENDED: adopt `01_core_architecture`'s SSH-domain 25-service set as canonical; publish ONE registry table (name, port, DB, owning team, submodule deps) transcluded everywhere.** Justification: `01` is the architecture-of-record and its set is the most externally-referenced; `10` §13.2's own docs_chain check was meant to enforce exactly this.

**CD-4 — Version pins (PostgreSQL / Go / Kafka / Redis / Kubernetes / Flutter / Istio).**
Values — PostgreSQL: 16.2 (04 §4.4/§6.2), 17-alpine (04 §9.4 self-contradiction), 17.0 (01:L1697, 06:L2827), 16.3 (03:L1903), 16-alpine (10:L4621), 17.2 (04 Appendix A), "16" (README:L97). Go: 1.23 (08:L417) vs 1.25 (11:L132, all Dockerfiles). Kafka: 3.7 (04 terraform, README) vs 3.9.0 (04 Appendix A). Redis: 8.0 (04 Appendix A) vs unstated. Kubernetes: 1.30 (04:L4485 terraform) vs 1.31 (04 Appendix A / PSS). Flutter: 3.22 (03, 11) vs 3.24 (08). Istio: 1.21+ (11) vs 1.22 (08).
Blast radius: 01, 03, 04, 06, 08, 10, 11, README.
**RECOMMENDED: PostgreSQL 17.0, Go 1.25, Kafka 3.7, Redis 7.x→confirm, Kubernetes 1.30, Flutter 3.24, Istio 1.22.** Justification: use each stack's *majority/terraform-authoritative* value; `08`'s Go 1.23 and `04` Appendix A's bumped pins are the isolated outliers. (Redis needs an explicit owner decision — only `04` Appendix A states one.)

**CD-5 — API-gateway port.**
Values: `:443` external edge (12 diag 2, 12:L114/211), `:8000` internal node (12 diag 3/27, 12:L236/2387), `:8080` container port (04 ConfigMap 04:L736; 11 smoke-test 11:L2822-2833).
Blast radius: 04, 11, 12.
**RECOMMENDED: container/internal port `8080`, external ingress `443`; delete the `:8000` node.** Justification: `8080` is what the runnable artifacts (ConfigMap + smoke test) actually use; `:443` is the correct edge; `:8000` appears only in an orphaned diagram node.

**CD-6 — Primary / DR regions.**
Values: primary `us-east-1`, DR `eu-west-1` (04:L6061/6082 — has the failover runbook) vs primary `eu-west-1`, DR `us-east-1` (12:L2478/2506, reversed).
Blast radius: 12 (align diagram 28 to 04).
**RECOMMENDED: primary `us-east-1`, DR `eu-west-1`.** Justification: `04` owns the concrete DR runbook, RTO/RPO, and failover commands; the mermaid diagram is the derivative artifact.

**CD-7 — JWT signing algorithm + issuer.**
Values: EdDSA/Ed25519, `iss: api.helixterm.io` (07:L368-383) vs RS256/4096-bit RSA, `iss: auth.helixterm.io` (05:L4291).
Blast radius: 05, 07 (and any client token verifier).
**RECOMMENDED: EdDSA (Ed25519), `iss: auth.helixterm.io`.** Justification: modern, smaller tokens, RFC 8037; issuer should be the dedicated auth host per the security doc's boundary model. (If an HSM only supports RSA signing, fall back to RS256 — resolve against CD's HSM story.)

**CD-8 — RBAC role vocabulary.**
Values: `org_members.role` CHECK = `owner,admin,member,viewer,billing` (07:L6638); API responses = `admin,member,viewer`; `org_db.roles` = free-form JSONB; security doc canonical = `super_admin,org_admin,team_admin,member,guest,api_user` (05 §6.1).
Blast radius: 05, 07 (schema + every API response + Istio AuthorizationPolicy).
**RECOMMENDED: security doc's 6-role set (`super_admin,org_admin,team_admin,member,guest,api_user`) as canonical, with a normalized `roles/permissions/role_permissions/role_assignments/resource_policies` schema.** Justification: it is the only fully-specified, scope-aware model; billing becomes a permission, not a role.

**CD-9 — HelixConstitution version.**
Values: v1.1.0 (01:L8, 04 Appendix A), v2.0 (10 throughout + footer 10:L6782), v1.0.0 (11 §5 helix-deps), v4.1.0 (04 §1.6 helix-deps).
Blast radius: 01, 04, 10, 11.
**RECOMMENDED: v2.0.** Justification: `10_submodule_integration` (the doc README designates authoritative for submodules/governance) cites v2.0 consistently including its footer; pick it and back-fill the clause-numbering scheme it uses. (Requires reconciling clause anchors — `01` uses §11.4.73-78, `10` uses §2.1-§10.1/§11.4.31.)

**CD-10 — Is zero-knowledge / true E2E vault encryption a HARD requirement?**
Values: asserted as guarantee ("server cannot decrypt", 05:L2348; "server never sees plaintext", 07:L1674) but contradicted by server-side key generation (07:L3285), `auth_method=password` plaintext (07:L5710), plaintext `TeamVault.VaultKey []byte` (05:L2669), server-side item re-wrap on rotation (05:L2707), and unspecified Shamir share transport (05:L2627).
Blast radius: 02, 05, 06 (arch half), 07.
**RECOMMENDED: HARD for *vault items* (client-side crypto only — remove server-side vault key generation and server-side re-wrap; specify client-side Shamir combination), and EXPLICITLY CARVE OUT SSH `auth_method=password` and server-side SSH key generation as a separate, non-zero-knowledge credential class that the marketing must stop describing as zero-knowledge.** Justification: the enterprise value prop depends on vault zero-knowledge being real; conflating it with SSH password auth is what makes the claim false today. If instead the org accepts server-side key custody, delete every "zero-knowledge / server cannot decrypt" claim.

**CD-11 (secondary) — Single canonical `helix-deps.yaml` + submodule import-path convention.**
Values: schema `helix/v1 DependencyManifest` (04 §1.6) vs `schema_version:1.0` (11 §5) vs a third in 04 Appendix A; submodule versions `01` v0.x vs `10`/manifests v1.x–v2.x; dot-path (`digital.vasic.security/...`) vs slash-path (`digital.vasic/security`) imports — the two cannot co-compile.
Blast radius: 01, 04, 10, 11.
**RECOMMENDED: one `helix-deps.yaml` (adopt `10` Appendix H compatibility-matrix pins as truth), slash-path import convention (`digital.vasic/<module>`), delete the other two manifests.** Justification: slash-path is the dominant form in `10`'s code body; a single manifest is the precondition for the CI catalogue-compliance gate to mean anything.

**CD-12 (secondary) — Mandated test-type count: 12 or 17?**
Values: "12 mandatory test types" (README:L112; 11 §6.1-6.12 enumerates 12) vs "seventeen test types" (03:L220-222 title + philosophy).
Blast radius: 03, 11, README.
**RECOMMENDED: reconcile to one number in the constitution, then regenerate 03's compliance matrix from it.** Justification: this is a governance-authored count; `11` enumerates concretely (12), so default to 12 unless the constitution repo says otherwise.

---

## 3. FIX-NOW (NO DECISION NEEDED)

Objectively-correct fixes requiring no product judgment. Grouped by doc.

### 02_client_specification (document-integrity — treat as pipeline bug, highest urgency)
- **Delete the duplicated block** §9→Appendix C repeated at L6884-8225 (diff = 1 line vs L5446-6788). — A5 (02:L6884-8225).
- **Repair the spliced/corrupted code block** in `host_list_page_test.dart`: cut off mid-statement at `sortDirection` (L5351), then unrelated `AuthBloc` mock code appended in the same unclosed fence. — A5 (02:L5351-5352).
- **Remove the second "End of document" marker** (L8224) and de-duplicate colliding `### 9.3` (L5219 vs L5446) and triple `### 9.4` (L5284/5505/6943). — A5.
- **Fix the ToC** (L11-23) to reflect actual sections + appendices. — A5.

### 06_ux_design_system
- **Split the interleaved document**: sections 2–10 exist twice (UX vs backend-architecture) under identical numbers — separate into `06_ux_design_system` (UX only) + a new platform-arch doc. — A1 (06:L197 vs L812, L1275 vs L1517, … L9268 vs L9940).
- **Recompute the WCAG contrast tables** (§2.6 L797-811, §9.2 L8280-8309) from actual token hex — every ratio is wrong; two are real failures presented as passes: `text-disabled` 2.69:1 (claimed 3.1:1, L334/L8294) and button-label-on-primary 4.32:1 (claimed 5.2:1, fails AA 4.5:1, L805/L8291). Add a CI check computing contrast from token JSON. — A1.
- **Resolve keyboard-shortcut collisions**: `⌘K` = Command Palette AND Clear terminal (L7606/L7697/L7844/L7851); `⌘⇧Z` = Redo AND Suspend-to-background (L7618/L7724). — A1.
- **Fix the touch-target self-contradiction** §9.9: claims "44×44 min" then lists Checkbox/Radio 40×40 "Slightly below min" (L8573/L8578-8579). — A1.
- **Complete the light-theme token tables** (§2.3 / JSON L9438-9466) to match the "full-fidelity" claim (L114), or drop the claim; add missing `helixLight` terminal-scheme JSON. — A1.

### 07_api_and_database
- **Fix the migration-tool Go sample** — uses `strings.ToUpper` without importing `strings`; won't compile (L7558-7576). — A2.
- **Fix SSH-config export** emitting identical `IdentityFile` for two hosts despite per-host `key_id` (L2311/L2318). — A2.
- **Reconcile `reason` field**: prose "required by org policy" vs field table `Required: No` (L2520). — A2.
- **Collapse the doubly-defined `api_keys` table** to one schema (L1257/L5402 vs 05 `prefix VARCHAR(8)`). — A2.

### 05_security_zero_trust
- **Fix retention mismatch**: `sshkey.exported` seeded `severity='critical'` but `retention_class='compliance'` (7y) not critical (10y) (L3503 vs L3686). — A2.
- **Root-CA ceremony** produces both RSA-4096 and Ed25519 with no statement of which is authoritative — pick one (L1924-1931). — A2.
- **Fix the GDPR-erasure no-op**: `AnonymizeUser` UPDATE silently no-ops under `no_audit_update … DO INSTEAD NOTHING` (L3446 vs L4011) — the erasure "succeeds" while PII remains. (Mechanism; may need Deep-Work, but the contradiction itself must be removed.) — A2.

### 08_product_roadmap_features
- **Fix the "(Complete Specifications)" heading** for UC-040–050 (L1838) — content is condensed, contradicting the label. — A3.
- **Correct version drift** to CD-4 values (Go 1.23→1.25, Flutter 3.24, Kafka, Istio 1.22, PostgreSQL) (08:L417). — A3/A6.
- **Fix domain drift** `helixterminator.io` → `helixterm.io` per CD-2 (08:L1862/2703/2822). — A6.

### 03_testing_strategy
- **Replace the empty-body stub test** `TestAuthService_RateLimit_Login` — table-driven shell with no assertions, violates the Anti-Bluff Covenant (L626-628). — A3.
- **Fix flaky-tolerance contradiction**: DoD "<1% over 100 runs" (L141) vs config `flaky_threshold: 0.05` (L5144). — A3.
- **Replace fabricated image digests** (`sha256:abcdef…`, `sha256:fedcba…`, L1905-1906) with real pinned digests, per the doc's own supply-chain claim. — A3.
- **Align Go/Flutter CI versions** to CD-4 (L5429/5929/6007, L5430/6051). — A3.

### 01_core_architecture
- **Restore TOC order**: §4.15-4.20 stranded after §9/§10 (L6142 after L5341). — A4.
- **Reconcile go.mod vs submodule tree**: `digital.vasic.middleware v0.2.4` required (L600) but absent from `submodules/` tree (L566-579). — A4.

### 10_submodule_integration
- **Unify the three internal service enumerations** (§1.2 L76-100 vs go.work L5639-5695 vs Appendix D L6153-6177) to the CD-3 registry. — A4.
- **Unify import paths** (slash vs dot) per CD-11 — Appendix G / go.work replace directives currently don't match §2-§14 code (L154 vs L6498 vs L5700). — A4.
- **Fix `helix-deps.yaml` `required_by`** entries pointing at non-existent services (`ssh-key` L5077, etc.). — A4.
- **Fix embedded governance filenames**: `AGENTS.MD`/`docs-chain.yaml` reference `docs/01_architecture.md` (L5409/6288/4097) — actual file is `01_core_architecture.md`; also normalize `AGENTS.MD`/`CLAUDE.MD` casing (L5317/5413). — A4.
- **Replace unpinned `v1.x.x` in-body placeholders** (§2-§14, 12 occurrences, e.g. L130) with the pinned versions per CD-11. — A4.

### 11_constitution_compliance
- **Remove `:latest` tags** from Appendix A.2's 25-row example table (L5198-5222) — violates the doc's own HARD-FORBIDDEN rule #15 (L843-844). — A6.
- **Implement or remove the `--check <name>` dispatch flag** — declared (L3448) but never read in `RunAll()` (L3486-3512); CI steps invoking `--check context-propagation` etc. (L3208-3214) are a bluff gate. — A6.
- **Complete Appendix A** rule-reference table — missing §9.2, §11.4.38, §11.4.65, §2.18 that the body cites (L5148-5192). — A6.
- **Rename §7 CI jobs** to stop colliding "§N" job ordinals with constitution-clause "§N" (L2898 vs L873). — A6.
- **Fix HT-NAME-002 violations in the doc's own examples** (`auth/`, `gateway/` bare names vs mandated hyphenated multi-word) (L261 vs L4305/4327) — resolve alongside CD-3. — A6.

### 04_devops_infrastructure
- **Remove `:latest` tags** from prod Deployments (`helixterm-prod`: gateway/auth/vault/ssh-proxy/session-recorder, L577/809/991/1183/1372). — A6.
- **Fix container UID mismatch**: Deployments+`_helpers.tpl` use `runAsUser: 65532` but §10.1 baseline mandates `65534` (undefined in distroless) (L548 vs L7310). — A6.
- **De-duplicate PrometheusRule groups**: two `ssh.alerts` (L5389-5436 vs L5572-5607) with conflicting metric names/thresholds; two `vault.alerts` (L5439 vs L5609). — A6.
- **Fix the `/healthz` grep gate** — greps literal `"/healthz"` but real routes are `/healthz/live` + `/healthz/ready`; false-fails every service (L8028 vs L601/609). — A6.
- **Fix DR-runbook drift**: chart/release named `helix-terminator` (L6432) vs pervasive `helixterm`; failover curls `/healthz` without `/live` (L6470). — A6.
- **Reconcile PSS enforce-version** v1.30 (L443) vs v1.31 (L7286); and PostgreSQL 16.2 (§4.4/§6.2) vs 17 (§9.4) within this same doc, per CD-4. — A6.

### 12_mermaid_diagrams
- **Remove/​wire orphaned nodes**: `P1` API-Gateway node (L236), `CONFIG_POD`/`HEALTH_POD` (L2277-2278) declared but never edged. — A6.
- **Declare `GW` participant** in sequence diagram 13 (used L926, never declared L917-925). — A6.
- **Collapse the double API-Gateway declaration** with conflicting ports per CD-5. — A6.
- **Add the 6 missing pods** to the K8s layout (Keychain/Snippet/Workspace/Analytics/HelixTrack-Bridge/Container-Bridge). — A6.

### README
- **Reconcile endpoint counts**: "221 REST API endpoints" (L106) vs doc-07 row "126 REST endpoints" (L22) vs doc-01 "221" (L16). — A6.
- **Reconcile test-type count** 12 vs 17 per CD-12 (L112). — A6.
- **Fix diagram categorization**: #04-05 (Kafka/RabbitMQ flow) mis-filed under "C4 L1/L2" (L52). — A6.

---

## 4. DEEP-WORK ITEMS

Substantial authoring gaps needing real content. Size S/M/L. Mapped to landing doc(s).

### Security hardening (closes the audit's critical security cluster)
- **[L] Add Row-Level Security to every multi-tenant table** in `07` §17-20 (currently zero RLS; isolation is app-code `WHERE org_id` only — one missed clause = cross-tenant IDOR). Mirror the `audit_events` RLS pattern. → **07** (+ 01 schema).
- **[L] Real audit tamper-evidence**: external WORM anchoring (S3 Object Lock compliance mode / notarization), include all PII columns in the hash chain, add fail-closed durability for privileged-op audit writes. → **05, 07**.
- **[M] Resolve GDPR-erasure vs audit-immutability** end-to-end: define anonymization method (hash/tokenize/null) that survives the append-only RULE and the hash chain. → **05, 07**.
- **[S] Encrypt SSO IdP OAuth tokens** — `sso_identities.access_token/refresh_token` are plaintext `TEXT` (07:L5507-5508). → **07**.
- **[L] Item-level vault endpoints + key-rotation/re-wrap** — `07` §4 exposes only container CRUD + opaque bulk sync; no fetch/rotate/audit of individual secrets, no re-key after member removal (07:L1668-1954, L5581). → **07** (+ 05 crypto).
- **[M] Secret redaction** for session logs/recordings and the AI-context path (07 §14 ships raw `sudo …`/output to model) — plus PII/data-residency policy. → **07, 05**.
- **[M] Injection & blast-radius gating**: escape/quote snippet `{{param}}` shell substitution (07:L3473); add elevated-permission/org-policy gates to port-forward (SOCKS5/remote) and multi-host broadcast/execute (07:L3053/L2641). → **07**.
- **[M] Break-glass / JIT / SoD controls**: emergency *grant* path, two-person control for `super_admin`/`org_admin`, time-boxed elevation (05 has only emergency *lockout*). → **05**.

### Resilience / operations
- **[L] PostgreSQL DR + HA**: RPO/RTO targets, cross-region replication, Patroni/pg_auto_failover for the write path, PITR/backup cadence for all 22 per-service DBs (absent in `01`; `04` has strong DR for the app tier but PG HA is under-specified). → **01, 04**.
- **[M] Redis persistence + cluster hash-tags** for load-bearing non-expiring keys (`vault:{id}:version`), and automated partition management (pg_partman) for the hand-created 2026/2027 partitions. → **07, 01**.
- **[M] WebSocket reconnect/resume semantics** (resume token / last-event-id / output replay) for the terminal channel. → **07**.
- **[S] RabbitMQ production path** — disabled in prod with no external host/Terraform resource, yet services depend on it (04:L2516); either provision (Amazon MQ) or remove (ties to CD on whether RabbitMQ is real — 03 has a full RabbitMQ test section 08 never mentions). → **04, 03**.
- **[M] Cost / FinOps section** for `04` (zero cost content across 8,220 lines despite EKS/multi-AZ RDS/MSK/DR). → **04**.

### Product / roadmap completeness
- **[L] Complete Phases 2–5** with Acceptance-Criteria / Test-Requirements / Definition-of-Done (rigor currently collapses after Phase 1). → **08**.
- **[M] Schedule the orphaned features** into phases: Terminal Multiplayer/Collab, Break-Glass, JIT, SSH Playground, Config Diff, Collaborative Debugging (spec'd but never scheduled). → **08**.
- **[M] Add a risk register + task owners + effort estimates** (none exist below whole-doc level). → **08**.
- **[M] Phase-level exit KPIs + self-hosted support runbook** to operationalize the 24h-P0 SLA. → **08**.

### Real-time collaboration (cross-doc — must land together)
- **[L] Full collaboration spec**: latency/presence perf budget (**09** — no collab service in any SLO table), client BLoC/events/states/transport + control-contention arbitration + channel encryption (**02** §1.16 has entities only), and wireframes/IA (**06** — no collab or vault wireframe). Also reconcile client `lastWriteWins` sync vs server-side CRDT/vector-clock (02:L3177 vs 09:L404). → **09, 02, 06**.

### Client platform completeness
- **[M] Auto-update mechanism** for all 6 platforms (channels, staged rollout, signing/notarization, forced security-patch) — zero coverage. → **02**.
- **[M] Mobile background-execution** for SSH/SFTP (iOS BGTaskScheduler/VoIP-push, Android foreground service) — zero coverage. → **02**.
- **[M] Conflict-resolution UI + connection-error taxonomy**: `manualMerge` has no screen; single generic `TerminalError` for all failure modes. → **02**.
- **[M] Missing wireframes**: Vault/Credential Manager, Org/Team, Billing (tokens reserved, screens never designed). → **06**.

### Testing depth
- **[M] Device/topology matrix** (iOS/Android/desktop OS + browser versions) — none exists. → **03**.
- **[M] Terminal-rendering performance test methodology** (60fps, <16ms keystroke, vttest, Impeller GPU mem, cold-start) — the product's "contractual SLOs" are untested. → **03**.
- **[M] Native accessibility testing** (VoiceOver/TalkBack/desktop SR) — a11y is web-only today. → **03**.
- **[S] Missing Pact contracts** (Vault, Host/Group, Recording, SCIM/SAML) + Flutter/Dart SBOM/vuln gate. → **03**.

### Diagrams (DIAGRAM-NEED, mostly S each)
- Convert ASCII architecture/flow art to Mermaid across **09, 02, 01** (drift risk).
- Add: C4 System-Context diagram (**01**); canary-promotion + DR-runbook sequence diagrams (**04/12**); RBAC/permission model diagram (**12**); token-lifecycle, key-hierarchy, trust-boundary, vault key-rotation, audit-pipeline sequences (**05**); WS-auth handshake + cross-service ER + SSH-session-lifecycle (**07**); screen-navigation flowchart, shortcut-scope, token-hierarchy, SSH-status state machine (**06**); roadmap Gantt/dependency graph (**08**); CI/CD gating flowchart (**03**).

---

## 5. PER-DOC WORK PACKAGES

Each doc can be owned by one implementer. **Cross-doc reconciliation items (must be done together / after the CD is made) are flagged separately at the end — do NOT let a single-doc owner touch these unilaterally.**

**01_core_architecture** — Fix-Now: TOC reordering (§4.15-4.20); go.mod↔submodule-tree mismatch. Deep-Work: PostgreSQL DR/HA (with 04); C4 context diagram; circuit-breaker/failure-mode coverage for all 25 services; API-versioning strategy. *Cross-doc: CD-3 (service list), CD-4 (PG version), CD-9 (constitution version), CD-11 (submodule versions/import paths).*

**02_client_specification** — Fix-Now: delete 1,340-line duplicate block, repair spliced test code, remove double end-marker, fix ToC, de-dupe §9.3/§9.4. Deep-Work: collaboration module spec, auto-update, mobile background-exec, conflict-resolution UI, error taxonomy, per-platform startup targets, feature-parity matrix. *Cross-doc: client-vs-server sync/CRDT (with 09).*

**03_testing_strategy** — Fix-Now: replace empty stub test, flaky-threshold, fabricated digests, CI versions. Deep-Work: device matrix, terminal-render perf tests, native a11y, missing Pact contracts, Dart SBOM gate, HelixTerminator-specific security tests. *Cross-doc: CD-4 (versions), CD-12 (12-vs-17 test types), RabbitMQ decision.*

**04_devops_infrastructure** — Fix-Now: remove prod `:latest`; UID 65532↔65534; de-dupe ssh.alerts/vault.alerts groups; `/healthz` grep gate; DR chart-name + healthz path; PSS + PG in-doc version contradictions. Deep-Work: cost/FinOps section; RabbitMQ prod path; cloud dev environment; provision Harbor; blue-green (if intended); canary-pipeline diagram. *Cross-doc: CD-1 (Podman/naming mandates), CD-2 (org name), CD-3 (service list), CD-4/5/6, CD-11 (helix-deps).*

**05_security_zero_trust** — Fix-Now: sshkey.exported retention; root-CA single algo; GDPR-erasure no-op contradiction. Deep-Work: external WORM audit anchor + full-PII hash chain + fail-closed durability; break-glass/JIT/SoD; vault crypto (client-side per CD-10); SAML endpoints + multi-tenant ACS routing; SCIM auth; SIEM data-residency; SOC2/ISO/PCI scoping caveats; security diagrams. *Cross-doc: CD-7 (JWT), CD-8 (RBAC), CD-10 (zero-knowledge), naming split (05:L707 mixes conventions).*

**06_ux_design_system** — Fix-Now: **split the two interleaved docs**; recompute WCAG tables (+CI check); shortcut collisions (⌘K, ⌘⇧Z); touch-target contradiction; complete light-theme tokens. Deep-Work: Vault/Org/Billing wireframes; Button component spec; collaboration wireframes; TUI/RTL/colorblind a11y; OpenDesign §11.4.162 alignment; UX diagrams. *Cross-doc: CD-4 (PG 17 assertion in arch half), CD-3 (service catalog in arch half).*

**07_api_and_database** — Fix-Now: `strings` import; SSH-config duplicate IdentityFile; `reason` required mismatch; single `api_keys` schema. Deep-Work: RLS everywhere; item-level vault + key-rotation endpoints; secret redaction (logs/recordings/AI); injection escaping + blast-radius gating; WS reconnect/resume; Redis persistence + hash-tags; partition automation; backup/PITR; access-approval endpoints. *Cross-doc: CD-7 (JWT), CD-8 (RBAC), CD-10 (zero-knowledge), audit anchoring (with 05).*

**08_product_roadmap_features** — Fix-Now: "(Complete Specifications)" heading; version drift; domain drift. Deep-Work: complete Phases 2-5 AC/Test/DoD; schedule orphaned features; risk register + owners + estimates; phase KPIs; self-hosted support; roadmap Gantt; competitor-claim citations. *Cross-doc: CD-1 (identity), CD-2 (domain), CD-4 (versions).*

**09_performance_analysis** — Fix-Now: (none pure — mostly gaps). Deep-Work: real load-test results (§13 has tooling only); collaboration perf model; stress data to 2-3× (currently 150%); soak/chaos results; cost/perf tradeoffs; audit-sink throughput reconciliation; NIC/egress sizing; multi-region SLO tiering; Mermaid conversion. *Cross-doc: client sync/CRDT (with 02).*

**10_submodule_integration** — Fix-Now: unify 3 internal service lists; unify import paths; fix `required_by` phantom services; governance filenames (`docs/01_architecture.md`→actual, AGENTS.MD casing); replace `v1.x.x` placeholders; add Owner column. Deep-Work: submodule import-graph cycle check; tag-mirroring/propagation-order strategy; canonical service-registry transclusion. *Cross-doc: CD-3, CD-9, CD-11 (the big one — all three enumerations + versions + import paths resolve here).*

**11_constitution_compliance** — Fix-Now: remove Appendix A.2 `:latest`; implement/remove `--check` bluff flag; complete Appendix A; rename §7 "§N" jobs; fix HT-NAME self-violations. Deep-Work: **rewrite the WireGuard/VPN product model to the SSH/vault model (CD-1)**; reconcile CI-gate scoring weights; make the compliance gate exercisable against the real service set. *Cross-doc: CD-1 (identity — root cause), CD-3 (service list ×2 internal), CD-9 (constitution version), CD-11 (helix-deps ×2), CD-12 (test count).*

**12_mermaid_diagrams** — Fix-Now: orphaned nodes (P1/CONFIG_POD/HEALTH_POD); declare GW participant; single API-Gateway/port; add 6 missing pods. Deep-Work: DR-runbook + canary + RBAC + backup diagrams. *Cross-doc: CD-3 (name the 25th/26th service), CD-5 (port), CD-6 (regions — diagram is reversed).*

**README** — Fix-Now: endpoint counts (221 vs 126); test-type count; diagram categorization; PG version headline. *Cross-doc: CD-1, CD-3, CD-4, CD-12.*

### Cross-doc reconciliation locks (owned by the architect, not single-doc implementers)
1. **CD-3 service registry** → touches 01, 04, 10, 11, 12, README simultaneously.
2. **CD-4 version pins** → 01, 03, 04, 06, 08, 10, 11, README.
3. **CD-11 helix-deps + import paths + submodule versions** → 01, 04, 10, 11.
4. **CD-1 product identity** → 11 (rewrite) + AGENTS.md, cross-checked against README/12.
5. **CD-7/CD-8/CD-10 security trio** → 05 + 07 must change together.
6. **CD-2 org/domain, CD-5 port, CD-6 regions, CD-9 constitution version, CD-12 test count** → each spans 2-4 docs listed above.

---
*End of master register.*
