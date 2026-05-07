# StrataHQ API Documentation

The backend API is implemented in Go with Chi. Most product endpoints are mounted under `/api/v1`; open integration endpoints are mounted under `/api/open/v1`.

## Response Envelope

Success responses use:

```json
{ "data": {}, "meta": {} }
```

## Error Envelope

Error responses use:

```json
{ "error": { "code": "VALIDATION_ERROR", "message": "Human-readable message" } }
```

## Platform

| Method | Path | Auth | Purpose |
| --- | --- | --- | --- |
| `GET` | `/healthz` | No | Liveness check |
| `GET` | `/readyz` | No | Readiness check for database and Redis |
| `GET` | `/metrics` | Token in production when configured | Prometheus metrics |

## Auth And Account

| Method | Path | Auth | Purpose |
| --- | --- | --- | --- |
| `POST` | `/api/v1/auth/register` | No | Register user |
| `POST` | `/api/v1/auth/login` | No | Log in |
| `POST` | `/api/v1/auth/refresh` | No | Refresh access token |
| `POST` | `/api/v1/auth/logout` | No | Log out and revoke refresh token |
| `POST` | `/api/v1/auth/forgot-password` | No | Request password reset |
| `POST` | `/api/v1/auth/reset-password` | No | Complete password reset |
| `POST` | `/api/v1/onboarding/setup` | Yes | Complete org and scheme onboarding |
| `GET` | `/api/v1/auth/me` | Yes | Current user profile |
| `PATCH` | `/api/v1/auth/profile` | Yes | Update user profile |
| `PATCH` | `/api/v1/auth/org` | Yes | Update organization |
| `POST` | `/api/v1/auth/change-password` | Yes | Change password |

## Schemes/Units/Members

| Method | Path | Auth | Purpose |
| --- | --- | --- | --- |
| `GET` | `/api/v1/schemes` | Yes | List schemes visible to the user |
| `POST` | `/api/v1/schemes` | Yes | Create scheme |
| `GET` | `/api/v1/schemes/{id}` | Yes | Get scheme detail |
| `PUT` | `/api/v1/schemes/{id}` | Yes | Update scheme |
| `DELETE` | `/api/v1/schemes/{id}` | Yes | Delete scheme |
| `GET` | `/api/v1/schemes/{id}/units` | Yes | List scheme units |
| `POST` | `/api/v1/schemes/{id}/units` | Yes | Create unit |
| `PUT` | `/api/v1/schemes/{id}/units/{unitId}` | Yes | Update unit |
| `GET` | `/api/v1/schemes/{id}/members` | Yes | List scheme members |
| `PATCH` | `/api/v1/schemes/{id}/members/{userId}` | Yes | Update member role or unit assignment |

## Invitations

| Method | Path | Auth | Purpose |
| --- | --- | --- | --- |
| `GET` | `/api/v1/invitations/verify/{token}` | No | Verify invitation token |
| `POST` | `/api/v1/invitations/verify/{token}/accept` | No | Accept invitation |
| `POST` | `/api/v1/invitations` | Yes | Create invitation |
| `GET` | `/api/v1/invitations` | Yes | List invitations |
| `POST` | `/api/v1/invitations/{id}/resend` | Yes | Resend invitation |
| `DELETE` | `/api/v1/invitations/{id}` | Yes | Revoke invitation |

## Levies And Collections

| Method | Path | Auth | Purpose |
| --- | --- | --- | --- |
| `GET` | `/api/v1/levies/attention` | Yes | Portfolio attention queue |
| `GET` | `/api/v1/levies/{schemeId}` | Yes | Levy dashboard |
| `GET` | `/api/v1/levies/{schemeId}/attention` | Yes | Scheme attention queue |
| `GET` | `/api/v1/levies/{schemeId}/accounts/{accountId}/events` | Yes | Collection event history |
| `POST` | `/api/v1/levies/{schemeId}/accounts/{accountId}/events` | Yes | Record collection event |
| `GET` | `/api/v1/levies/{schemeId}/accounts/{accountId}/reminder-draft` | Yes | Generate reminder draft |
| `POST` | `/api/v1/levies/{schemeId}/accounts/{accountId}/reminders` | Yes | Send reminder |
| `POST` | `/api/v1/levies/{schemeId}/periods` | Yes | Create levy period |
| `POST` | `/api/v1/levies/{schemeId}/reconcile` | Yes | Reconcile payments |
| `POST` | `/api/v1/levies/{schemeId}/reconcile/imports` | Yes | Import bank statement CSV |
| `GET` | `/api/v1/levies/{schemeId}/reconcile/imports/{importId}` | Yes | Get bank statement import |
| `POST` | `/api/v1/levies/{schemeId}/reconcile/imports/{importId}/apply` | Yes | Apply bank statement import |

## Maintenance

| Method | Path | Auth | Purpose |
| --- | --- | --- | --- |
| `GET` | `/api/v1/maintenance/{schemeId}` | Yes | Maintenance dashboard |
| `POST` | `/api/v1/maintenance/{schemeId}` | Yes | Create maintenance request |
| `POST` | `/api/v1/maintenance/{schemeId}/{id}/assign` | Yes | Assign request |
| `POST` | `/api/v1/maintenance/{schemeId}/{id}/resolve` | Yes | Resolve request |

## Governance/Documents/Reporting

| Method | Path | Auth | Purpose |
| --- | --- | --- | --- |
| `GET` | `/api/v1/agm/{schemeId}` | Yes | AGM dashboard |
| `POST` | `/api/v1/agm/{schemeId}/meetings` | Yes | Schedule meeting |
| `POST` | `/api/v1/agm/{schemeId}/meetings/{meetingId}/proxy` | Yes | Assign proxy |
| `POST` | `/api/v1/agm/{schemeId}/resolutions/{resolutionId}/vote` | Yes | Cast vote |
| `GET` | `/api/v1/documents/{schemeId}` | Yes | List documents |
| `POST` | `/api/v1/documents/{schemeId}` | Yes | Create document record |
| `DELETE` | `/api/v1/documents/{schemeId}/{id}` | Yes | Delete document |
| `GET` | `/api/v1/financials/{schemeId}` | Yes | Financial dashboard |
| `PUT` | `/api/v1/financials/{schemeId}/budget-lines` | Yes | Upsert budget line |
| `PUT` | `/api/v1/financials/{schemeId}/reserve-fund` | Yes | Update reserve fund |

## Communications/Compliance/WhatsApp/Contractors

| Method | Path | Auth | Purpose |
| --- | --- | --- | --- |
| `GET` | `/api/v1/communications/{schemeId}` | Yes | List communications |
| `POST` | `/api/v1/communications/{schemeId}` | Yes | Create communication |
| `GET` | `/api/v1/compliance/portfolio` | Yes | Portfolio compliance dashboard |
| `GET` | `/api/v1/compliance/{schemeId}` | Yes | Compliance dashboard |
| `POST` | `/api/v1/compliance/{schemeId}/assess` | Yes | Create compliance assessment |
| `POST` | `/api/v1/compliance/{schemeId}/items` | Yes | Create compliance item |
| `PUT` | `/api/v1/compliance/{schemeId}/items/{itemId}` | Yes | Update compliance item |
| `DELETE` | `/api/v1/compliance/{schemeId}/items/{itemId}` | Yes | Delete compliance item |
| `GET` | `/api/v1/whatsapp/{schemeId}` | Yes | WhatsApp dashboard |
| `POST` | `/api/v1/whatsapp/{schemeId}/broadcasts` | Yes | Create broadcast |
| `POST` | `/api/v1/whatsapp/{schemeId}/messages/{messageId}/maintenance-request` | Yes | Convert WhatsApp message to maintenance request |
| `PATCH` | `/api/v1/whatsapp/{schemeId}/maintenance-intakes/{intakeId}` | Yes | Dismiss maintenance intake |
| `GET` | `/api/v1/contractors` | Yes | List contractors |
| `POST` | `/api/v1/contractors` | Yes | Create contractor |
| `GET` | `/api/v1/contractors/marketplace` | Yes | Search marketplace contractors |
| `PATCH` | `/api/v1/contractors/{contractorId}` | Yes | Update contractor |
| `POST` | `/api/v1/contractors/{contractorId}/reviews` | Yes | Create contractor review |

## Billing/Early Access/AI/Audit/Integrations

| Method | Path | Auth | Purpose |
| --- | --- | --- | --- |
| `POST` | `/api/v1/billing/checkout` | Yes | Create Stripe checkout session |
| `POST` | `/api/v1/billing/portal` | Yes | Create Stripe portal session |
| `GET` | `/api/v1/billing/subscription` | Yes | Get subscription state |
| `POST` | `/api/v1/billing/webhooks/stripe` | No | Stripe webhook |
| `POST` | `/api/v1/early-access` | No | Submit early-access request |
| `GET` | `/api/v1/early-access/{id}/approve` | Signed link | Approve page |
| `POST` | `/api/v1/early-access/{id}/approve` | Signed link | Approve request |
| `GET` | `/api/v1/early-access/{id}/reject` | Signed link | Reject page |
| `POST` | `/api/v1/early-access/{id}/reject` | Signed link | Reject request |
| `GET` | `/api/v1/admin/early-access` | Admin | List early-access requests |
| `POST` | `/api/v1/admin/early-access/{id}/approve` | Admin | Approve request |
| `POST` | `/api/v1/admin/early-access/{id}/reject` | Admin | Reject request |
| `POST` | `/api/v1/ai/copilot` | Yes | AI copilot response |
| `GET` | `/api/v1/audit/schemes/{schemeId}/events` | Yes | List scheme audit events |
| `GET` | `/api/v1/integrations/api-clients` | Yes | List API clients |
| `POST` | `/api/v1/integrations/api-clients` | Yes | Create API client |
| `DELETE` | `/api/v1/integrations/api-clients/{clientId}` | Yes | Revoke API client |
| `GET` | `/api/v1/whatsapp/webhooks` | No | Verify inbound WhatsApp webhook |
| `POST` | `/api/v1/whatsapp/webhooks` | No | Receive inbound WhatsApp webhook payloads |

## Open API

Open API routes are mounted under `/api/open/v1`. The OpenAPI document is public; resource endpoints require API-key authentication.

| Method | Path | Auth | Purpose |
| --- | --- | --- | --- |
| `GET` | `/api/open/v1/openapi.json` | No | OpenAPI document |
| `GET` | `/api/open/v1/schemes` | API key | List permitted schemes |
| `GET` | `/api/open/v1/schemes/{schemeId}` | API key | Get scheme |
| `GET` | `/api/open/v1/schemes/{schemeId}/units` | API key | List units |
| `GET` | `/api/open/v1/schemes/{schemeId}/levy-periods` | API key | List levy periods |
| `GET` | `/api/open/v1/schemes/{schemeId}/levy-accounts` | API key | List levy accounts |
| `GET` | `/api/open/v1/schemes/{schemeId}/levy-payments` | API key | List levy payments |
| `GET` | `/api/open/v1/schemes/{schemeId}/financials` | API key | Financial summary |
