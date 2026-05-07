# StrataHQ Case Study

StrataHQ is a South African sectional-title property management platform. This case study is written as a technical-first portfolio summary and should be read alongside [docs/architecture.md](docs/architecture.md) and [docs/engineering-decisions.md](docs/engineering-decisions.md).

## Problem

Sectional-title operations cut across finance, maintenance, governance, resident communication, and access control. In practice, many teams still stitch these workflows together from spreadsheets, inboxes, chat threads, PDF packs, and separate billing or document tools.

That creates predictable problems:

- managing agents do not have one reliable portfolio view
- trustees need scheme oversight without gaining full operator privileges
- residents need a simpler, narrower view of their own scheme activity
- audit trails, payment follow-up, and governance workflows become fragmented across tools

StrataHQ addresses that by treating property operations as one product with clear role boundaries instead of a loose set of adjacent utilities.

## Users

The current product shape is built around three main user groups:

- Managing agents: organization-level operators who oversee multiple schemes, onboarding, member access, levy collection, billing, and portfolio attention items.
- Trustees: scheme-level decision makers who need visibility into levies, maintenance, AGM workflows, documents, compliance, and audit history.
- Residents: users with a narrower scheme and unit-level view for profile, maintenance, communications, and other resident-facing activity.

These audiences are reflected directly in the data model through org memberships, scheme memberships, and role checks rather than only in frontend navigation.

## What Was Built

The repository contains a full-stack application with a Next.js frontend and a Go backend API. The current implemented scope includes:

- managing-agent portfolio and scheme dashboards
- authentication, onboarding, invitations, and profile management
- scheme, unit, and member administration
- levy periods, levy accounts, payment tracking, collection follow-up, bank statement imports, and reconciliation
- maintenance request intake, assignment, and resolution
- AGM meetings, resolutions, voting, and proxy assignment
- document records, financial views, communications, and compliance workflows
- WhatsApp inbox, broadcasts, and WhatsApp-to-maintenance intake
- Stripe billing, early-access administration, AI copilot, audit history, and an open integration API

This is not a marketing prototype. The repo contains the UI, API, schema migrations, generated query layer, test suites, seed flow, and separate worker process needed to run the platform locally.

## Core Workflows / Demo Walkthrough

The current demo path, also documented in [docs/demo.md](docs/demo.md), is easiest to understand from the managing-agent role.

1. Log in as the managing-agent demo user.
2. Open the portfolio dashboard and review scheme-level attention items.
3. Enter a scheme and inspect the scheme overview, units, members, and role-aware navigation.
4. Open levy management to show periods, account status, reconciliation, bank statement import, and collection events.
5. Open maintenance to show request creation, triage, assignment, and resolution.
6. Open AGM, documents, compliance, or financials to show scheme governance and reporting coverage.
7. Compare the same scheme as a trustee or resident to show narrower access boundaries.

That walkthrough demonstrates the core product claim: StrataHQ is organized around recurring operational workflows, not just static record storage.

## Architecture

At a high level:

- Next.js 16 handles the web app, server components, client components, and browser-facing API routes.
- Next.js route handlers proxy allowed backend calls and manage session-related cookies.
- The Go API, built with Chi, owns domain logic and authorization checks.
- PostgreSQL stores durable business data and the background job queue.
- `sqlc` generates typed query access from hand-written SQL.
- Redis supports rate limiting and runtime support concerns.
- A separate Go worker processes asynchronous jobs such as reminders and bank import work.
- External providers include Stripe, Resend, Twilio WhatsApp, and an OpenAI-compatible AI provider.

The architecture is described in more detail in [docs/architecture.md](docs/architecture.md).

## Engineering Decisions

The main engineering choices were made to keep product complexity explicit instead of hiding it behind a single framework abstraction.

- The frontend is separate from the backend, but browser traffic is funneled through a Next.js proxy so session handling and CSRF checks stay at the frontend boundary.
- The backend is decomposed into domain packages so major product areas can evolve independently.
- PostgreSQL and `sqlc` were chosen over an ORM-heavy stack because the product depends on explicit relational modeling and query control.
- Role and tenancy boundaries are part of the data model and service layer, not just UI state.
- Provider integrations are isolated in backend services.
- Background work runs in a separate worker instead of inside request handlers.
- Security and audit controls are treated as shared platform concerns.

The tradeoffs behind those decisions are documented in [docs/engineering-decisions.md](docs/engineering-decisions.md).

## Current Status

StrataHQ is a working application repo with enough surface area to demonstrate real product and engineering decisions:

- there is a seeded demo flow with managing-agent, trustee, and resident accounts
- the backend exposes a broad `/api/v1` product API plus `/api/open/v1` integration routes
- the database schema covers identity, tenancy, levies, maintenance, AGM, documents, compliance, WhatsApp, billing, audit, and background jobs
- the repo includes frontend tests, backend tests, integration tests, load-test hooks, and local setup commands

At the same time, it should still be read as a product in active development:

- some areas are deeper than others
- provider-backed flows depend on environment configuration
- the docs and demo emphasize fake seeded data rather than a production tenant

## Next Improvements

The next useful improvements are less about adding another module and more about tightening the platform:

- expand end-to-end coverage across the main role-based workflows
- continue hardening auth, session, and provider-integration security controls
- improve operational observability for worker jobs, retries, and provider failures
- deepen scheme-level financial and reporting workflows where the product already has data foundations
- keep the open API and internal product API aligned as integrations mature

## Closing Note

As a portfolio piece, StrataHQ is strongest when presented as a systems project: a multi-role product with explicit domain boundaries, a typed relational backend, asynchronous job handling, and a clear separation between browser concerns and core business logic.
