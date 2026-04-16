# Queue Outbound Execution Design

**Date:** 2026-04-16
**Status:** Proposed
**Scope:** Turn the collections attention queue into a true execution surface by letting authorized operators review and send debtor-specific reminders through email and WhatsApp directly from each queue item.

---

## Context

StrataHQ now has a unified collections attention queue on the portfolio and scheme overview pages. That closes the triage gap: operators can see which arrears cases matter most.

The next missing layer is execution. Today the queue can identify a case, but it does not yet let the operator complete the most important outbound action from inside the worklist. That means the product still stops short of a full collections operating loop.

The highest-leverage next addition is queue-native outbound execution because it:

- converts the queue from triage into action
- improves collections follow-through without forcing context switching
- compounds the existing attention-queue work instead of starting a separate feature track
- creates a durable audit trail for legally sensitive debtor communication

---

## Product Decision

Add a **Collection Execution Modal** launched from each queue item's recommended action.

V1 supports **actual outbound reminder sending** through:

- email
- WhatsApp

Both channels are reviewed in one combined modal before anything is sent.

This is not an autosend system. The operator must explicitly review and confirm the send. The product promise is:

**See the right levy case, review the right debtor-specific reminder, send it on the right channels, and keep the queue and timeline accurate afterward.**

---

## Recommended Approach

Three approaches were considered:

### 1. Queue-native execution modal

Operators stay inside the attention queue, review debtor-specific drafts, and send directly from the case.

**Pros**

- strongest operational loop
- preserves queue context
- makes the new queue feel like the product's command center

**Cons**

- requires new UI and delivery integration work

### 2. Prefill and hand off to existing communications screens

The queue would only deep-link into Communications or WhatsApp with prefilled content.

**Pros**

- cheaper to ship

**Cons**

- breaks triage flow
- makes the queue feel secondary
- relies on scheme-wide tools that are not debtor-specific

### 3. Semi-automated reminder engine

The system would schedule and send reminders based on rules.

**Pros**

- high automation upside

**Cons**

- too aggressive for current product maturity
- weakens control in a legally and operationally sensitive workflow
- depends on delivery, approval, and template quality not yet proven

### Recommendation

Choose **Approach 1: queue-native execution modal**.

It is the most accretive extension of the shipped attention queue and the most direct route to making StrataHQ feel indispensable in daily collections work.

---

## User Experience

### Entry point

Each attention queue item exposes `Send reminder` as the primary outbound action.

Clicking it opens a combined execution modal.

### Modal layout

The modal includes:

- case summary header with scheme, unit, owner, overdue amount, days overdue, and score drivers
- editable `Email draft` section
- editable `WhatsApp draft` section
- per-channel enable or disable controls
- a single explicit confirmation action for sending the selected channels

The modal keeps the operator in queue context instead of redirecting to other pages.

### Channel behavior

- if both channels are available, both appear enabled by default
- if one channel is unavailable, that channel appears disabled with a precise reason
- if neither channel is available, the send action is unavailable and the queue item explains why

Examples of disabled-state reasons:

- `No email on file`
- `No WhatsApp number or active thread`

### Send behavior

The operator must always review before sending.

V1 does **not** support:

- automatic send on open
- background send sequences
- one-click send without review

### Post-send behavior

After a send attempt:

- the queue refreshes immediately
- the timeline reflects what happened
- successful follow-up cools the case's urgency for a short period instead of making it disappear abruptly
- partial failures are shown clearly channel by channel

---

## Message Generation

Draft generation should be deterministic for V1.

The initial email and WhatsApp drafts are built from structured templates plus live case data:

- owner name
- unit identifier
- scheme name
- outstanding amount
- days overdue
- recommended next step

Email and WhatsApp should not share one identical template.

### Email draft

Email can be more formal and complete. It should include:

- subject line
- debtor context
- overdue amount
- request for action
- next-step guidance

### WhatsApp draft

WhatsApp should be shorter, more direct, and mobile-readable. It should:

- stay concise
- preserve operational clarity
- avoid large formal blocks of text

### AI role

Base drafts should not depend on the model.

An AI assist can later offer optional rewrite or tone-adjustment support inside the modal, but the primary V1 draft path must remain deterministic, explainable, and safe.

---

## Event and Audit Model

Outbound execution must produce a structured record, not a vague note.

Sending from the modal creates a top-level `reminder_sent` collection event tied to the levy account.

That event stores:

- actor id
- actor role
- scheme id
- levy account id
- selected channels
- actual reviewed message content that was sent
- send timestamp
- per-channel delivery outcome
- per-channel failure reason where applicable

Possible per-channel outcomes include:

- `sent`
- `failed`
- `skipped`

This event model supports:

- accurate case timelines
- later retry support
- future delivery receipts
- queue reprioritization based on real follow-up, not assumptions

If the operator edits a draft before sending, the edited version is what gets stored.

---

## Queue and Scoring Effects

The attention queue remains the operational source of truth.

After a successful reminder send:

- the case remains in the queue
- urgency is reduced for a short cooling-off window
- the item displays fresh action metadata such as `last action just now`

After a failed send:

- the case remains urgent
- the timeline records the failed channel
- the queue does not pretend follow-up happened when it did not

This prevents false confidence and keeps ranking aligned with actual operator progress.

---

## Permissions

Permissions must be enforced on the server.

### Managing agents

- can open the execution modal from portfolio and scheme queue items
- can send reminders across schemes they manage

### Trustees

- can open the execution modal only for schemes they are authorized to access
- can send reminders only within those schemes

### Residents

- do not see the attention queue
- cannot access outbound collections execution

UI hiding alone is insufficient. Authorization must be checked when generating drafts and again when executing outbound sends.

---

## Failure Handling

The system should fail clearly and conservatively.

- if draft generation cannot resolve one channel, that channel is disabled with a reason
- if the send partially succeeds, the result is surfaced as a partial success
- if both selected channels fail, no false success state is shown
- timeline entries must reflect actual channel outcomes
- queue reprioritization should only treat successful sends as fresh follow-up

This feature must never collapse delivery ambiguity into a generic `sent` label.

---

## V1 Boundaries

To keep scope sharp, V1 includes:

- queue-native send reminder modal
- deterministic debtor-specific email and WhatsApp drafts
- editable review for both channels in one modal
- explicit user-confirmed outbound send
- structured reminder event logging with per-channel outcomes
- immediate queue refresh after send

V1 excludes:

- one-click send without review
- automatic retry or drip sequences
- attorney escalation workflows
- model-generated base drafts
- route-through reuse of the scheme-wide broadcast tools

V1 also keeps the other queue actions narrower:

- `log follow-up`
- `promise to pay`
- `flag for legal review`

These remain structured non-outbound actions in the queue while `send reminder` becomes the first true execution path.

---

## Why This Is The Right Next Addition

The attention queue solved prioritization. The missing compounding step is execution.

This addition is the strongest next move because it:

- completes the queue's core job
- keeps operators in one workflow instead of bouncing between modules
- creates better collections hygiene and accountability
- produces better data for future scoring and AI assistance
- moves StrataHQ closer to an operating system for scheme collections rather than a set of dashboards

In short, the next strategic win is not another new module. It is making the new queue capable of doing the work it is already smart enough to identify.
