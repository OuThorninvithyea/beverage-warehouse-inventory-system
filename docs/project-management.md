# Project Management Workflow

## Planning layers

The project uses four connected planning layers without duplicating their responsibilities:

1. **Google Gantt chart** — official Week 6–15 roadmap and deadlines.
2. **GitHub Project** — current status, owner, priority, effort and weekly iteration.
3. **GitHub Issues and pull requests** — requirements, implementation and evidence.
4. **Slack** — announcements, blockers, decisions and short progress summaries.

Codex can help turn the weekly plan into issues, implement a selected issue, run checks, prepare pull requests, inspect CI failures and publish Slack-ready summaries.

## Board status

```text
Backlog -> Ready -> In Progress -> Code Review -> QA Testing -> Done
                              \-> Blocked
```

## Required project fields

| Field | Values or purpose |
| --- | --- |
| Status | Backlog, Ready, In Progress, Code Review, QA Testing, Blocked, Done |
| Iteration | Week 6 through Week 15 |
| Owner | Responsible team member |
| Area | Backend, Frontend, Database, QA, Documentation, UI/UX, Barcode |
| Priority | P0 critical, P1 important, P2 optional |
| Effort | 1, 2, 3, 5 or 8 points |
| Target date | Expected completion date |

## Weekly cadence

### Monday: plan

- Confirm the weekly outcome against the Gantt chart.
- Move achievable issues into Ready.
- Confirm owners, dependencies and acceptance criteria.
- Post one weekly kickoff message in Slack.

### Tuesday through Thursday: execute

- Work from Ready issues.
- Link branches and pull requests to their issues.
- Report blockers immediately in Slack.
- Review and test completed work continuously.

### Friday: review

- Demonstrate the working increment.
- Verify completed acceptance criteria.
- Record QA evidence and close accepted issues.
- Replan incomplete work with a written reason.
- Compare actual progress with the Gantt chart.
- Post a concise weekly summary in Slack.

## Slack update format

```text
[W6][Backend] Ou

Done:
- Created the backend project structure

Next:
- Connect PostgreSQL and add migrations

Blocked:
- None

Issue:
- #12
```

Use one parent message per week in `#project_managements`, with routine updates in its thread. Create a new channel message only for an important decision or blocker that requires team visibility.
