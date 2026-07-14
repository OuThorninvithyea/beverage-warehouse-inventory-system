# Contributing

## Working agreement

- GitHub Issues are the source of truth for implementation work.
- The Google Gantt chart controls the Week 6–15 deadline and phase alignment.
- Slack is used for announcements, blockers and concise progress updates.
- Do not begin implementation without acceptance criteria.
- Keep issues small enough to finish in approximately half a day to two days.

## Branch names

Use a short category followed by the issue number and description:

```text
feat/12-health-endpoint
fix/27-barcode-lookup
docs/41-qa-evidence
test/52-fefo-picking
```

## Commit messages

Use an imperative, focused message:

```text
Add backend health endpoint
Validate transfer destination
Document Week 6 acceptance tests
```

## Pull requests

Every pull request must:

- Link its GitHub issue using `Closes #123` when appropriate.
- Explain what changed and why.
- Include validation evidence.
- Avoid unrelated file changes.
- Receive review before merge.
- Pass QA when the issue requires QA.

## Definition of Done

An issue is Done only when:

- Its acceptance criteria are satisfied.
- Relevant automated and manual tests pass.
- The pull request is reviewed and merged.
- Documentation is updated when behavior or setup changes.
- QA evidence is recorded when required.
- No unresolved blocker remains.

Finishing code moves work to `Code Review`; it does not move directly to `Done`.
