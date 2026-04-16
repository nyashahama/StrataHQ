# Unified Attention Queue Design

**Date:** 2026-04-16
**Status:** Approved
**Scope:** Add a collections-first operational queue that helps managing agents and trustees identify, prioritize, and act on risky levy arrears cases with clear recommendations and a full audit trail.

---

## Context

StrataHQ already has the underlying surfaces for levy operations, reconciliation, a scheme dashboard, an agent portfolio dashboard, and a basic AI copilot. What it does not yet have is a daily operating layer that turns those signals into an actionable ranked worklist.

Today, the portfolio dashboard shows summary metrics such as collection percentage and open maintenance counts, but it does not tell an operator which debtor needs attention first, what action should happen next, or what has already been done. That leaves a gap between visibility and operational leverage.

The highest-value next addition is a collections-first attention queue because it:

- improves cash flow directly
- makes the existing levy and reconciliation work materially more useful
- gives both agents and trustees a shared, role-safe place to act
- turns StrataHQ from a dashboard product into an operating system for scheme collections

---

## Product Decision

The recommended feature is a **Unified Attention Queue** with a collections-first V1.

Agents see a portfolio-wide ranked queue across all schemes they manage. Trustees see the same queue logic, but only within the schemes they are authorized to access. The queue is not a generic notification feed and not a copilot-first experience. It is an operational worklist of levy accounts that need intervention now.

Each queue item represents one arrears case with:

- scheme and unit identity
- owner or account holder context
- amount outstanding
- arrears age
- risk score
- score drivers
- recommended next action
- last collection action metadata

The primary user promise is simple: **open StrataHQ and know exactly which levy cases need action today.**

---

## Architecture

The attention queue becomes a first-class operating surface rather than a widget inside the existing copilot.

- `app/agent/page.tsx` becomes the portfolio triage entry point for agents, with a queue that spans all managed schemes
- `app/app/[schemeId]/page.tsx` gets a scheme-scoped version for trustees and other authorized scheme operators
- the backend exposes a derived attention-queue endpoint that returns pre-ranked levy cases shaped for the UI
- the UI consumes queue items as operational cases, not raw levy accounts
- the existing copilot remains secondary support for explanation and drafting, but it is not the primary interaction model

This preserves a clear mental model:

- dashboard = triage
- queue item = decision
- action = outcome
- activity log = accountability

---

## Core Components

### `CollectionAttentionItem`

A derived domain model representing one collection case. It should include:

- `scheme_id`
- `scheme_name`
- `unit_id`
- `unit_identifier`
- `levy_account_id`
- `owner_name`
- `outstanding_cents`
- `days_overdue`
- `status`
- `risk_score`
- `score_drivers`
- `recommended_action`
- `last_action`
- `last_action_at`

This is a read-optimized shape for operations. It is not a persistence model by itself.

### `AttentionQueueList`

A reusable list component used in both portfolio and scheme contexts. It supports:

- role-aware columns
- filtering by action type, urgency, and scheme
- deterministic sorting by risk score and age
- quick visual explanation of why an item is ranked highly

### `CollectionActionPanel`

The action surface attached to each queue item. V1 actions:

- `send reminder`
- `log follow-up`
- `mark promise to pay`
- `flag for legal review`

These are explicit user-triggered actions. Nothing runs automatically.

### `CollectionActivityLog`

A per-case timeline showing what happened, who did it, and when. This is required because trustees will be allowed to act and because collection work quickly becomes ambiguous without history.

### `RiskScoringService`

A deterministic ranking layer that converts levy state and collection history into:

- a sortable urgency score
- a human-readable explanation
- a recommended next action

The service must be explainable. V1 should be rules-based rather than model-based.

---

## Ranking Logic

V1 ranking should stay simple, inspectable, and useful. The score should be derived from factors already available or straightforward to capture:

- days overdue
- outstanding amount
- partial-payment pattern
- absence of recent follow-up
- prior promise-to-pay broken or nearing due date

The queue item should also carry explicit score drivers so the UI can say why a case is urgent, for example:

- `90+ days overdue`
- `high balance outstanding`
- `partial payments without clearance`
- `no follow-up in 14 days`

V1 should not attempt predictive ML or opaque probability scoring. The objective is operator trust and actionability, not novelty.

---

## Role Model and Permissions

The queue is shared logic with role-scoped visibility.

### Managing agents

- can see queue items across all schemes they manage
- can filter and sort across the full portfolio
- can perform all V1 actions

### Trustees

- can only see queue items for their own scheme
- can act on collection items within that scheme
- may be restricted from certain escalation actions depending on final policy, but the system must enforce that server-side rather than trusting the client

### Residents

- do not see the queue
- do not see collection activity logs beyond existing resident-safe levy views

Every action must be permission-checked on the server. UI hiding alone is not sufficient.

---

## Data Flow

The attention queue should be backend-derived and client-rendered.

1. The client requests the attention queue for the current role and scope.
2. The backend gathers levy account state and collection-action history.
3. The backend computes a ranked list of `CollectionAttentionItem` objects.
4. The UI renders the ranked queue and its score drivers.
5. A user takes an explicit action on a queue item.
6. The backend writes a structured collection event.
7. The queue and case timeline refresh so ranking and recommendations can update immediately.

This creates a closed loop:

`levy state -> derived case -> user action -> immutable event -> recomputed priority`

The queue should not be ranked in the browser. Server-side derivation keeps the logic consistent across agent and trustee views and makes authorization easier to reason about.

---

## Actions and Event Logging

Each V1 action should write an immutable collection event with:

- actor id
- actor role
- scheme id
- levy account id
- event type
- structured payload
- created at

Examples:

- reminder sent
- follow-up logged with note
- promise to pay recorded with amount and expected date
- legal review flag added

This event log powers the case timeline and provides the operational trace required for shared agent and trustee action.

Free-text notes can exist, but high-value actions should capture structured fields where the information matters. For example, a promise-to-pay record should store amount and due date, not only a note string.

---

## User Experience

### Agent portfolio view

The agent landing page should stop being only a summary board and become a triage console.

V1 layout:

- summary metrics remain at the top
- attention queue becomes the primary working section below
- items are sortable and filterable
- top-ranked items expose one-click actions

The product outcome is that an agent can open the page and immediately work the most consequential levy cases first.

### Trustee scheme view

Trustees see the same queue logic, but scoped to the active scheme. The queue should be prominent on the scheme overview page so it becomes part of normal governance and collections review rather than a hidden secondary screen.

### Copilot relationship

Copilot can later explain the recommendation for a case or draft a reminder message, but it should not be the only way to discover or execute work. The queue remains the source of operational truth.

---

## Error Handling

The feature should fail clearly and conservatively.

- queue load failures show an explicit error state
- missing signals reduce score fidelity but do not block queue generation
- missing signals should be surfaced in the explanation rather than hidden
- no reminder, flag, or promise-to-pay event is created without explicit user confirmation
- invalid or unauthorized actions fail server-side with a clear error message

The main design goal is to avoid fake confidence. If the system cannot explain a ranking or cannot verify permission, it should not pretend otherwise.

---

## Testing Strategy

### Risk scoring tests

Table-driven tests should verify:

- older arrears outrank newer arrears when balances are comparable
- materially larger balances increase urgency
- recent follow-up can reduce urgency
- broken or aging promise-to-pay states increase urgency
- partial-payment patterns can change the recommended action

### Permission tests

Verify that:

- agents receive portfolio-wide results
- trustees only receive scheme-scoped results
- unauthorized actions are rejected server-side

### Workflow tests

Verify that:

- action writes create the correct collection event
- case timelines update as expected
- a new action can change ranking and recommended next action

### UI tests

Cover:

- loading state
- empty state
- error state
- mixed-priority queue rendering
- action submission flows

### Audit/event tests

Verify that each action writes a durable, correctly shaped event payload because traceability is a core part of the feature value.

---

## V1 Boundaries

To keep this feature sharp and shippable, V1 excludes:

- automatic sending or escalation without user approval
- attorney integrations
- assignment workflows
- advanced collections CRM states
- model-based risk prediction
- resident-facing collections actions

The V1 win is not breadth. The win is a trustworthy, ranked queue with immediate actions and a reliable audit trail.

---

## Why This Is The Smartest Next Addition

This feature has the best current leverage because it compounds existing work instead of competing with it.

- it makes levy and reconciliation meaningfully more valuable
- it upgrades the portfolio dashboard from summary to action
- it creates a daily habit loop for both agents and trustees
- it supports a stronger sales story without being a sales-only feature
- it lays clean groundwork for later predictive intelligence and copilot assistance

In short, this is the addition most likely to make StrataHQ feel indispensable rather than merely informative.
