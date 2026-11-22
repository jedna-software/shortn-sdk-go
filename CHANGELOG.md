# Changelog

## 0.1.0

First release.

- Module `github.com/jedna-software/shortn-sdk-go`, package `shortn`, Go 1.22 or newer.
- Covers the Shortn Link public API (`/api/v1`) in resource groups, every method taking a
  `context.Context` first:
  - `Links`: `List`, `Get`, `Create`, `Update`, `Delete` (suspend), `Enable`, `SetABWinner`.
  - `Analytics`: `Link`.
  - `Workspaces`: `List`, `Get`.
  - `Tags`: `List`, `Create`, `Get`, `Update`, `Delete`.
  - `UTMTemplates`: `List`, `Create`, `Update`, `Delete`.
  - `Destinations`: `List`, `Create`, `Update`, `Delete`.
  - `Teams`: `List`, `Create`, `Get`, `Update`, `Delete` (archive).
  - `Domains`: `List`, `Create`, `Verify`, `Delete`.
  - `Reports`: `ListRules`, `GetRule`, `CreateRule`, `UpdateRule`, `DeleteRule`.
  - `Conversions`: `TrackLead`, `ValidateLead`, `TrackSale`, `ValidateSale`, `TrackRefund`,
    `ValidateRefund`, `Summary`, `Attribution`, `Timeline`, `Events`.
  - `Public` (no key needed): `ValidateURL`, `QRCode` (image bytes), `Plans`.
- Typed request params and results that mirror the API's JSON; every result keeps the whole
  decoded object in `Raw`.
- `APIError` with status, `code`, message, `hint`, `field`, `details`, request id,
  `Retry-After`, the decoded body and the raw body.
- Retries on 429 and 503 (honouring `Retry-After`) and on network errors, for `GET`, `PUT`,
  `DELETE` and for `POST`s carrying an idempotency key; configurable and switchable off.
  A `Retry-After` longer than `RetryOptions.MaxRetryAfter` (default 60 s), such as the
  monthly quota's wait until the next month, is not waited for: the error returns at once.
  A `POST` with `test_mode: true` is never retried, idempotency key or not, because the API
  doesn't deduplicate test events.
- Options for base URL, timeout, retries, HTTP client and User-Agent suffix
  (`shortn-sdk-go/0.1.0`). The default HTTP client does not follow redirects, so a `3xx`
  answer is an `APIError`; a client passed with `WithHTTPClient` keeps its own policy.
- Offline unit tests for every method, and an opt-in smoke test against a live instance.
