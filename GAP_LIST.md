# Gap list

## Covered

All 50 key-based and key-less routes of the public API that a server can call, under
`/api/v1`:

| Group | Routes |
|---|---|
| `Links` | `GET /links`, `POST /links` (with a key), `GET`/`PATCH`/`DELETE /links/{public_id}`, `POST /links/{public_id}/enable`, `POST /links/{public_id}/ab/winner` |
| `Analytics` | `GET /analytics/links/{public_id}` |
| `Workspaces` | `GET /workspaces`, `GET /workspaces/{public_id}` |
| `Tags` | `GET`/`POST /workspaces/{public_id}/tags`, `GET`/`PUT`/`DELETE /tags/{public_id}` |
| `UTMTemplates` | `GET`/`POST /workspaces/{public_id}/utm-templates`, `PUT`/`DELETE /utm-templates/{public_id}` |
| `Destinations` | `GET`/`POST /workspaces/{public_id}/destinations`, `PUT`/`DELETE /workspaces/{public_id}/destinations/{destination_public_id}` |
| `Teams` | `GET`/`POST /workspaces/{public_id}/teams`, `GET`/`PUT`/`DELETE /teams/{slug}` |
| `Domains` | `GET`/`POST /domains`, `POST /domains/{id}/verify`, `DELETE /domains/{id}` |
| `Reports` | `GET`/`POST /reports/rules`, `GET`/`PATCH`/`DELETE /reports/rules/{public_id}` |
| `Conversions` | `POST /conversions/{leads,sales,refunds}` and their `/validations`, `GET /conversions/{summary,attribution,timeline,events}` |
| `Public` | `GET /validate`, `GET /qr`, `GET /plans` |

## Left out on purpose

- `POST /links` without a key and `GET /links/{code}/analytics`: they need a browser human
  check (`X-Turnstile-Token`) passed on shortn.link, so a server can't use them.
- Browser-session (`/api/app`) and internal endpoints.

## Left to do

- Typed models for the alert-only report rule fields (`watch_condition`, `watch_recurring`,
  `watch_reset_increment`) and for report definitions; today they are maps or in `Raw`.
- Typed `move_impact` (UTM template update) and domain verification details (`txt`,
  `routing`); today they are maps.
- Pagination helpers (iterators) for `Links.List` and `Conversions.Events`.
- Per-request retry overrides.

## Behaviour to know

- Test mode and refunds: a `test_mode` refund names a live sale's `invoice_id` and is
  recorded as a test event that leaves the sale's totals alone; a test-mode sale can't be
  refunded, so the smoke test expects `404 conversion_sale_not_found` for that step.
- Test-mode leads and sales are never deduplicated: each request records a new test event.
  The client therefore never retries a test-mode request, even with an idempotency key.
