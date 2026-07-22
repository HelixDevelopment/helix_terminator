# FEATURE Directive Queue — helix_terminator

**Revision:** 1
**Last modified:** 2026-07-22T14:25:08Z

Constitution §11.4.213 durable queue of scheduled `FEATURE` directive
requests. A `FEATURE` request — recognized via any of the six §11.4.140
grammar forms (`FEATURE :: x`, `DEFAULT::FEATURE :: x`, `/FEATURE x`,
`/DEFAULT::FEATURE x`, `FEATURE ---> x`, or the single-colon `FEATURE: x`,
registered-action-only so ordinary prose is never mistaken for one) —
**SCHEDULES** a deep, enterprise-grade research + implementation-planning
effort rather than executing it synchronously. Each scheduled request is
appended below **the moment it is accepted**, and separately lands as a
Type=Task / Status=Queued workable item via the shared §11.4.202
`report_item.sh` engine (never a duplicate tracker-push implementation).
This queue exists so a scheduled `FEATURE` request is **never dropped**
between acceptance and the autonomous loop (§11.4.87/.94/.97/.103/.126)
claiming and driving it to a genuinely completed-and-wired, or explicitly
evidence-backed closed, terminal state (§11.4.197).

| id | accepted | request summary | status | tracked item |
|----|----------|------------------|--------|--------------|
| — | — | (no FEATURE requests scheduled yet) | — | — |
