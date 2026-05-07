# StrataHQ Demo

## Live Demo

Live app: https://strata-hq-blue.vercel.app

The public demo uses seeded fake data only. Do not connect these accounts to real schemes, residents, owner records, payment data, or contact details.

| Role | Email | Password | What to review |
| --- | --- | --- | --- |
| Managing agent | `agent@demo.stratahq.test` | `StrataDemo!2026` | Portfolio dashboard, scheme operations, invitations, collection workflows |
| Trustee | `trustee@demo.stratahq.test` | `StrataDemo!2026` | Scheme-level oversight, levy status, maintenance, AGM, documents |
| Resident | `resident@demo.stratahq.test` | `StrataDemo!2026` | Resident view, unit-linked scheme activity, profile workflows |

## Demo Data Reset Checklist

Use this checklist before sharing or recording the demo:

1. Confirm the demo backend points at a dedicated demo database.
2. Set `SEED_DEMO_PASSWORD=StrataDemo!2026`.
3. Run the backend seed command against the demo environment.
4. Confirm the three demo users can log in.
5. Confirm all visible scheme, resident, owner, payment, and communication data is fake.
6. Reset or reseed the demo environment after public testing sessions.

Local seed command:

```bash
cd backend
SEED_DEMO_PASSWORD='StrataDemo!2026' make seed
```

## Suggested Demo Walkthrough

1. Open the live app and log in as the managing-agent user.
2. Review the portfolio dashboard and attention queue.
3. Open a scheme and review the scheme overview.
4. Open levy management and show reconciliation/import workflows.
5. Open maintenance and show request triage.
6. Open AGM or documents to show scheme governance workflows.
7. Log out and briefly compare trustee/resident access.

## Screenshot Inventory

Screenshots are stored in `docs/assets/screenshots/` when captured from the seeded app.

Target screenshots:

| File | View | URL |
| --- | --- | --- |
| `landing-page.png` | Public landing page | `/` |
| `login-page.png` | Login page | `/auth/login` |
| `agent-portfolio-dashboard.png` | Managing-agent portfolio dashboard | `/agent` |
| `scheme-overview.png` | Scheme overview | `/app/[schemeId]` |
| `levy-reconciliation.png` | Levy dashboard and reconciliation | `/app/[schemeId]/levy` |
| `maintenance-dashboard.png` | Maintenance dashboard | `/app/[schemeId]/maintenance` |
| `agm-workflow.png` | AGM dashboard | `/app/[schemeId]/agm` |
| `documents-compliance.png` | Documents or compliance workflow | `/app/[schemeId]/documents` or `/app/[schemeId]/compliance` |

## Screenshot Capture Workflow

Start the stack:

```bash
npm ci
cp .env.example .env.local
cd backend
cp .env.example .env
make docker-up
make migrate-up
SEED_DEMO_PASSWORD='StrataDemo!2026' make seed
make run
```

In another terminal:

```bash
npm run dev
```

Save screenshots manually into `docs/assets/screenshots/` using your browser or OS screenshot tool:

1. Open `http://localhost:3000` and capture the public landing page.
2. Open `http://localhost:3000/auth/login` and capture the login page.
3. Use the browser window at a consistent desktop width.
4. Log in as `agent@demo.stratahq.test`.
5. Visit each authenticated route from the inventory table and capture it after the page fully loads.
6. Name each image exactly as listed in the inventory table.

Authenticated screenshots do not require Playwright in this repo. Use a browser profile with only the demo session active so captures stay reproducible and free of personal UI chrome.

## Demo Video Script

Demo video: not yet recorded

Target length: 90-120 seconds.

Script:

1. "StrataHQ is a sectional-title property management platform for South African managing agents, trustees, and residents."
2. "I will use a seeded demo account so no real resident or scheme data is shown."
3. Log in as `agent@demo.stratahq.test`.
4. Show the managing-agent portfolio dashboard and point out scheme attention items.
5. Open a scheme and show the scheme overview.
6. Open the levy dashboard and show payment/reconciliation workflows.
7. Open maintenance and show request triage.
8. Briefly show AGM, documents, or compliance to demonstrate governance coverage.
9. Close on the showcase package materials in this repo: the README entry point plus the architecture, API, database, engineering decisions, and local setup docs once they are added.

Recording checklist:

1. Reset demo data before recording.
2. Use a browser profile with no personal bookmarks or extensions visible.
3. Keep the video under two minutes.
4. Do not show environment variables, private dashboards, real emails, or live provider secrets.
