# Project tracking templates

## Weekly progress

```text
[WEEK][AREA] Owner

Done:
- Completed outcome with evidence

Next:
- Next concrete task

Blocked:
- None, or blocker plus owner and decision date

Issue / PR:
- Link
```

## Blocker

```text
Blocker ID:
Date raised:
Affected issue:
Owner:
Impact:
What was attempted:
Decision needed:
Decision owner:
Required by:
Current status: Open | Resolved
Resolution evidence:
```

## Decision

```text
Decision ID:
Date:
Context:
Options considered:
Decision:
Reason:
Consequences:
Owner:
Related issues / documents:
```

## Requirement evidence

```text
Requirement ID:
Issue:
Acceptance criterion:
Implementation file or URL:
Automated test:
Manual evidence:
QA reviewer:
Result: Pass | Fail | Blocked
```

## Current decisions

| ID | Decision | Reason |
| --- | --- | --- |
| DEC-001 | Gantt Week 6–15 is the schedule source of truth | It is the approved live roadmap |
| DEC-002 | FEFO physical selection and FIFO costing remain separate | They solve different inventory/accounting requirements |
| DEC-003 | Barcode scan never mutates inventory | Every movement requires validation and explicit confirmation |
| DEC-004 | Use PostgreSQL transactions for every stock movement | Prevents partial balance, cost and audit updates |
| DEC-005 | Use hard-coded Go RBAC for four roles | The small stable permission matrix does not require an external policy engine |
