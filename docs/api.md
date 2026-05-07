# StrataHQ API Documentation

The backend API is implemented in Go with Chi. Most product endpoints are mounted under `/api/v1`; open integration endpoints are mounted under `/api/open/v1`.

## Response Envelope

The standard JSON API success envelope uses a `data` wrapper. Example `200 OK` body:

```json
{ "data": {} }
```

List and paginated responses include `meta` only when the handler returns them through `response.JSONList`:

```json
{ "data": [], "meta": { "page": 1, "per_page": 50, "total": 123 } }
```

Documented exceptions outside the standard JSON API success envelope:
- Some successful endpoints return `204 No Content`.
- `GET /api/open/v1/openapi.json` serves the OpenAPI document directly instead of the standard JSON envelope.
- The signed-link early-access flows render HTML pages instead of the standard JSON envelope:
  - `GET /api/v1/early-access/{id}/approve`
  - `POST /api/v1/early-access/{id}/approve`
  - `GET /api/v1/early-access/{id}/reject`
  - `POST /api/v1/early-access/{id}/reject`
- `GET /metrics` serves Prometheus text output.
- `GET /api/v1/whatsapp/webhooks` can return plain-text verification output or a bare `200 OK`.
- `POST /api/v1/whatsapp/webhooks` returns a bare `200 OK` on successful inbound webhook handling.

## Error Envelope

Standard JSON API error responses use this shape. Example `400 BAD_REQUEST` body:

```json
{ "error": { "code": "BAD_REQUEST", "message": "Human-readable message", "requestId": "req_123" } }
```

Exceptions outside the standard JSON API error envelope:
- The signed-link early-access flows return HTML error pages instead of JSON error bodies.
- `GET /metrics` can return a plain-text `401 unauthorized` response when a metrics token is configured and the request omits or mismatches it.
- `GET /api/v1/whatsapp/webhooks` can return plain-text or bare non-JSON responses during provider verification flows.

## Platform

| Method | Path | Auth | Purpose |
| --- | --- | --- | --- |
| `GET` | `/healthz` | No | Liveness check |
| `GET` | `/readyz` | No | Readiness check for database and Redis |
| `GET` | `/metrics` | Token when configured | Prometheus metrics |

## Auth/Account

| Method | Path | Auth | Purpose |
| --- | --- | --- | --- |
| `POST` | `/api/v1/auth/register` | No | Register user |
| `POST` | `/api/v1/auth/login` | No | Log in |
| `POST` | `/api/v1/auth/refresh` | No | Refresh access token |
| `POST` | `/api/v1/auth/logout` | No | Log out and revoke refresh token |
| `POST` | `/api/v1/auth/forgot-password` | No | Request password reset |
| `POST` | `/api/v1/auth/reset-password` | No | Complete password reset |
| `POST` | `/api/v1/onboarding/setup` | Org admin | Complete org and scheme onboarding |
| `GET` | `/api/v1/auth/me` | Authenticated user | Current user profile |
| `PATCH` | `/api/v1/auth/profile` | Authenticated user | Update user profile |
| `PATCH` | `/api/v1/auth/org` | Org admin | Update organization |
| `POST` | `/api/v1/auth/change-password` | Authenticated user | Change password |

## Schemes/Units/Members

| Method | Path | Auth | Purpose |
| --- | --- | --- | --- |
| `GET` | `/api/v1/schemes` | Authenticated user | List schemes visible to the user |
| `POST` | `/api/v1/schemes` | Org admin | Create scheme |
| `GET` | `/api/v1/schemes/{id}` | Scheme member or org admin | Get scheme detail |
| `PUT` | `/api/v1/schemes/{id}` | Org admin | Update scheme |
| `DELETE` | `/api/v1/schemes/{id}` | Org admin | Delete scheme |
| `GET` | `/api/v1/schemes/{id}/units` | Scheme member or org admin | List scheme units |
| `POST` | `/api/v1/schemes/{id}/units` | Org admin | Create unit |
| `PUT` | `/api/v1/schemes/{id}/units/{unitId}` | Org admin | Update unit |
| `GET` | `/api/v1/schemes/{id}/members` | Scheme member or org admin | List scheme members |
| `PATCH` | `/api/v1/schemes/{id}/members/{userId}` | Org admin | Update member role or unit assignment |

## Invitations

| Method | Path | Auth | Purpose |
| --- | --- | --- | --- |
| `GET` | `/api/v1/invitations/verify/{token}` | No | Verify invitation token |
| `POST` | `/api/v1/invitations/verify/{token}/accept` | No | Accept invitation |
| `POST` | `/api/v1/invitations` | Org admin | Create invitation |
| `GET` | `/api/v1/invitations` | Org admin | List invitations |
| `POST` | `/api/v1/invitations/{id}/resend` | Org admin | Resend invitation |
| `DELETE` | `/api/v1/invitations/{id}` | Org admin | Revoke invitation |

## Levies/Collections

| Method | Path | Auth | Purpose |
| --- | --- | --- | --- |
| `GET` | `/api/v1/levies/attention` | Org admin | Portfolio attention queue |
| `GET` | `/api/v1/levies/{schemeId}` | Scheme member or org admin | Levy dashboard |
| `GET` | `/api/v1/levies/{schemeId}/attention` | Non-resident scheme member or org admin | Scheme attention queue |
| `GET` | `/api/v1/levies/{schemeId}/accounts/{accountId}/events` | Non-resident scheme member or org admin | Collection event history |
| `POST` | `/api/v1/levies/{schemeId}/accounts/{accountId}/events` | Non-resident scheme member or org admin | Record collection event |
| `GET` | `/api/v1/levies/{schemeId}/accounts/{accountId}/reminder-draft` | Authenticated user (current implementation) | Generate reminder draft; current implementation does not enforce scheme membership |
| `POST` | `/api/v1/levies/{schemeId}/accounts/{accountId}/reminders` | Non-resident scheme member or org admin | Send reminder |
| `POST` | `/api/v1/levies/{schemeId}/periods` | Org admin | Create levy period |
| `POST` | `/api/v1/levies/{schemeId}/reconcile` | Org admin | Reconcile payments |
| `POST` | `/api/v1/levies/{schemeId}/reconcile/imports` | Org admin | Import bank statement CSV |
| `GET` | `/api/v1/levies/{schemeId}/reconcile/imports/{importId}` | Org admin | Get bank statement import |
| `POST` | `/api/v1/levies/{schemeId}/reconcile/imports/{importId}/apply` | Org admin | Apply bank statement import |

## Maintenance

| Method | Path | Auth | Purpose |
| --- | --- | --- | --- |
| `GET` | `/api/v1/maintenance/{schemeId}` | Scheme member or org admin | Maintenance dashboard |
| `POST` | `/api/v1/maintenance/{schemeId}` | Scheme member or org admin | Create maintenance request |
| `POST` | `/api/v1/maintenance/{schemeId}/{id}/assign` | Non-resident scheme member or org admin | Assign request |
| `POST` | `/api/v1/maintenance/{schemeId}/{id}/resolve` | Non-resident scheme member or org admin | Resolve request |

## Governance/Documents/Reporting

| Method | Path | Auth | Purpose |
| --- | --- | --- | --- |
| `GET` | `/api/v1/agm/{schemeId}` | Scheme member or org admin | AGM dashboard |
| `POST` | `/api/v1/agm/{schemeId}/meetings` | Org admin | Schedule meeting |
| `POST` | `/api/v1/agm/{schemeId}/meetings/{meetingId}/proxy` | Non-admin scheme member | Assign proxy |
| `POST` | `/api/v1/agm/{schemeId}/resolutions/{resolutionId}/vote` | Non-admin scheme member | Cast vote |
| `GET` | `/api/v1/documents/{schemeId}` | Scheme member or org admin | List documents |
| `POST` | `/api/v1/documents/{schemeId}` | Org admin | Create document record |
| `DELETE` | `/api/v1/documents/{schemeId}/{id}` | Org admin | Delete document |
| `GET` | `/api/v1/financials/{schemeId}` | Scheme member or org admin | Financial dashboard |
| `PUT` | `/api/v1/financials/{schemeId}/budget-lines` | Non-resident scheme member or org admin | Upsert budget line |
| `PUT` | `/api/v1/financials/{schemeId}/reserve-fund` | Non-resident scheme member or org admin | Update reserve fund |

## Communications/Compliance/WhatsApp/Contractors

| Method | Path | Auth | Purpose |
| --- | --- | --- | --- |
| `GET` | `/api/v1/communications/{schemeId}` | Scheme member or org admin | List communications |
| `POST` | `/api/v1/communications/{schemeId}` | Non-resident scheme member or org admin | Create communication |
| `GET` | `/api/v1/compliance/portfolio` | Org admin | Portfolio compliance dashboard |
| `GET` | `/api/v1/compliance/{schemeId}` | Non-resident scheme member or org admin | Compliance dashboard |
| `POST` | `/api/v1/compliance/{schemeId}/assess` | Non-resident scheme member or org admin | Create compliance assessment |
| `POST` | `/api/v1/compliance/{schemeId}/items` | Non-resident scheme member or org admin | Create compliance item |
| `PUT` | `/api/v1/compliance/{schemeId}/items/{itemId}` | Non-resident scheme member or org admin | Update compliance item |
| `DELETE` | `/api/v1/compliance/{schemeId}/items/{itemId}` | Non-resident scheme member or org admin | Delete compliance item |
| `GET` | `/api/v1/whatsapp/{schemeId}` | Scheme member or org admin | WhatsApp dashboard |
| `POST` | `/api/v1/whatsapp/{schemeId}/broadcasts` | Non-resident scheme member or org admin | Create broadcast |
| `POST` | `/api/v1/whatsapp/{schemeId}/messages/{messageId}/maintenance-request` | Non-resident scheme member or org admin | Convert WhatsApp message to maintenance request |
| `PATCH` | `/api/v1/whatsapp/{schemeId}/maintenance-intakes/{intakeId}` | Non-resident scheme member or org admin | Dismiss maintenance intake |
| `GET` | `/api/v1/contractors` | Org admin, or non-resident with scheme access | List contractors; non-admin callers must pass `scheme_id` |
| `POST` | `/api/v1/contractors` | Org admin | Create contractor |
| `GET` | `/api/v1/contractors/marketplace` | Org admin, or non-resident with scheme access | Search marketplace contractors for `scheme_id` |
| `PATCH` | `/api/v1/contractors/{contractorId}` | Org admin | Update contractor |
| `POST` | `/api/v1/contractors/{contractorId}/reviews` | Org admin, or non-resident with scheme access | Create contractor review |

## Billing/Early Access/AI/Audit/Integrations

| Method | Path | Auth | Purpose |
| --- | --- | --- | --- |
| `POST` | `/api/v1/billing/checkout` | Org admin | Create Stripe checkout session |
| `POST` | `/api/v1/billing/portal` | Org admin | Create Stripe portal session |
| `GET` | `/api/v1/billing/subscription` | Org admin | Get subscription state |
| `POST` | `/api/v1/billing/webhooks/stripe` | No | Stripe webhook |
| `POST` | `/api/v1/early-access` | No | Submit early-access request |
| `GET` | `/api/v1/early-access/{id}/approve` | Signed link | Approve page |
| `POST` | `/api/v1/early-access/{id}/approve` | Signed link | Approve request |
| `GET` | `/api/v1/early-access/{id}/reject` | Signed link | Reject page |
| `POST` | `/api/v1/early-access/{id}/reject` | Signed link | Reject request |
| `GET` | `/api/v1/admin/early-access` | Org admin with configured admin email | List early-access requests |
| `POST` | `/api/v1/admin/early-access/{id}/approve` | Org admin with configured admin email | Approve request |
| `POST` | `/api/v1/admin/early-access/{id}/reject` | Org admin with configured admin email | Reject request |
| `POST` | `/api/v1/ai/copilot` | Org admin for portfolio scope; non-resident scheme member or org admin with scheme access for scheme scope | AI copilot response |
| `GET` | `/api/v1/audit/schemes/{schemeId}/events` | Admin or trustee | List scheme audit events |
| `GET` | `/api/v1/integrations/api-clients` | Org admin | List API clients |
| `POST` | `/api/v1/integrations/api-clients` | Org admin | Create API client |
| `DELETE` | `/api/v1/integrations/api-clients/{clientId}` | Org admin | Revoke API client |
| `GET` | `/api/v1/whatsapp/webhooks` | No | Verify inbound WhatsApp webhook |
| `POST` | `/api/v1/whatsapp/webhooks` | No | Receive inbound WhatsApp webhook payloads |

## Open API

Open API routes are mounted under `/api/open/v1`. The OpenAPI document is public; resource endpoints require API-key authentication.

| Method | Path | Auth | Purpose |
| --- | --- | --- | --- |
| `GET` | `/api/open/v1/openapi.json` | No | OpenAPI document |
| `GET` | `/api/open/v1/schemes` | API key (read:schemes) | List permitted schemes |
| `GET` | `/api/open/v1/schemes/{schemeId}` | API key (read:schemes + scheme grant) | Get scheme |
| `GET` | `/api/open/v1/schemes/{schemeId}/units` | API key (read:schemes + scheme grant) | List units |
| `GET` | `/api/open/v1/schemes/{schemeId}/levy-periods` | API key (read:schemes + scheme grant) | List levy periods |
| `GET` | `/api/open/v1/schemes/{schemeId}/levy-accounts` | API key (read:levies + scheme grant) | List levy accounts |
| `GET` | `/api/open/v1/schemes/{schemeId}/levy-payments` | API key (read:levies + scheme grant) | List levy payments |
| `GET` | `/api/open/v1/schemes/{schemeId}/financials` | API key (read:financials + scheme grant) | Financial summary |
