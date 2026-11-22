# Shortn Link Go SDK

Go client for the [Shortn Link](https://shortn.link) public API
(`https://app.shortn.link/api/v1`).

Use it from trusted server-side code: web backends, workers, serverless functions. Never
put a Shortn API key in browser, mobile or desktop code.

## Installation

```sh
go get github.com/jedna-software/shortn-sdk-go
```

Requires Go 1.22 or newer. The package name is `shortn`.

## Quick start

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	shortn "github.com/jedna-software/shortn-sdk-go"
)

func main() {
	client, err := shortn.NewClient(os.Getenv("SHORTN_API_KEY"))
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()

	link, err := client.Links.Create(ctx, shortn.CreateLinkParams{
		URL:   "https://example.com/spring",
		Title: "Spring campaign",
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(link.ShortURL, link.PublicID)
}
```

Every method takes a `context.Context` first. Resources hang off the client in groups:
`client.Links`, `Analytics`, `Workspaces`, `Tags`, `UTMTemplates`, `Destinations`, `Teams`,
`Domains`, `Reports`, `Conversions` and `Public`. Each API key needs the scopes of the
endpoints it calls (see [authentication](https://docs.shortn.link/docs/api/authentication/)).

## Configuration

```go
client, err := shortn.NewClient(
	os.Getenv("SHORTN_API_KEY"),
	shortn.WithBaseURL("https://app.shortn.link"), // the SDK appends /api/v1
	shortn.WithTimeout(10*time.Second),             // per HTTP attempt
	shortn.WithRetry(shortn.RetryOptions{MaxRetries: 3}),
	shortn.WithHTTPClient(&http.Client{Transport: myTransport}),
	shortn.WithUserAgent("my-app/1.0"),             // appended to shortn-sdk-go/0.1.0
)
```

Requests send `Authorization: Bearer <key>` and `User-Agent: shortn-sdk-go/0.1.0`.

The SDK's default HTTP client does not follow redirects, so a `301`/`302` can't silently turn
a `POST` into a `GET`: a `3xx` answer is returned as an `*shortn.APIError`. A client passed
with `WithHTTPClient` is used as it is, with its own redirect policy (Go's default follows
redirects); set its `CheckRedirect` to return `http.ErrUseLastResponse` to keep the SDK's
behaviour.

## Responses

Results mirror the API's JSON: an endpoint that answers `{"data": {...}}` returns a struct
with a `Data` field, a list with `pagination` returns `Data` and `Pagination`, and so on.
Times are RFC 3339 strings (`shortn.FormatTime` formats a `time.Time` for requests).

Every response struct also has a `Raw map[string]any` holding the whole decoded object, so
fields the API adds later are readable before the SDK models them:

```go
got, _ := client.Links.Get(ctx, publicID)
fmt.Println(got.Data.Title, got.Data.Raw["some_new_field"])
```

Optional request fields are pointers or `omitempty`, and only what you set is sent. Use
`shortn.String`, `shortn.Bool`, `shortn.Int`, `shortn.Int64` for pointer fields, and
`shortn.Strings(...)` for lists where an empty list matters (`shortn.Strings()` sends `[]`,
for example to make a tag workspace-wide again).

## Examples by group

### Links

Links are addressed by their `public_id`.

```go
opts := &shortn.LinkListOptions{Limit: 100}
for {
	page, err := client.Links.List(ctx, opts)
	if err != nil {
		return err
	}
	for _, l := range page.Data {
		fmt.Println(l.PublicID, l.ShortCode, l.Clicks)
	}
	if !page.Pagination.HasMore || page.Pagination.NextOffset == nil {
		break
	}
	opts.Offset = *page.Pagination.NextOffset
}

got, err := client.Links.Get(ctx, publicID)
_, err = client.Links.Update(ctx, publicID, shortn.UpdateLinkParams{Title: shortn.String("Spring sale")})
_, err = client.Links.Delete(ctx, publicID) // suspends; analytics are kept
_, err = client.Links.Enable(ctx, publicID)
_, err = client.Links.SetABWinner(ctx, publicID, shortn.ABWinnerParams{Variant: "B"})
```

### Analytics

```go
stats, err := client.Analytics.Link(ctx, publicID)
fmt.Println(stats.TotalClicks, stats.UniqueClicks, stats.ClicksByCountry["Germany"])
```

### Workspaces

```go
workspaces, err := client.Workspaces.List(ctx)
workspace, err := client.Workspaces.Get(ctx, workspaces.Data[0].PublicID)
```

### Tags

```go
tag, err := client.Tags.Create(ctx, workspaceID, shortn.CreateTagParams{Name: "Spring", Color: "#3b82f6"})
tags, err := client.Tags.List(ctx, workspaceID, nil)
tag, err = client.Tags.Get(ctx, tag.PublicID)
tag, err = client.Tags.Update(ctx, tag.PublicID, shortn.UpdateTagParams{Color: "#16a34a"})
_, err = client.Tags.Delete(ctx, tag.PublicID)
```

### UTM templates

```go
created, err := client.UTMTemplates.Create(ctx, workspaceID, shortn.CreateUTMTemplateParams{
	Name: "Newsletter", UTMSource: "newsletter", UTMMedium: "email",
})
list, err := client.UTMTemplates.List(ctx, workspaceID, &shortn.UTMTemplateListOptions{IncludeTeamScoped: true})
_, err = client.UTMTemplates.Update(ctx, created.Data.PublicID, shortn.UpdateUTMTemplateParams{
	Name: "Newsletter", UTMSource: "newsletter", UTMMedium: "email", UTMCampaign: "december",
}) // PUT replaces the template
_, err = client.UTMTemplates.Delete(ctx, created.Data.PublicID, &shortn.UTMTemplateDeleteOptions{Links: "keep"})
```

### Destinations

```go
dest, err := client.Destinations.Create(ctx, workspaceID, shortn.CreateDestinationParams{
	Name: "Ops inbox", Type: "email", Config: map[string]any{"to": []string{"ops@example.com"}},
})
all, err := client.Destinations.List(ctx, workspaceID)
_, err = client.Destinations.Update(ctx, workspaceID, dest.PublicID, shortn.UpdateDestinationParams{Name: "Ops", Type: "email"})
_, err = client.Destinations.Delete(ctx, workspaceID, dest.PublicID)
```

### Teams

Teams are read, changed and archived by slug. A slug is unique only within its workspace,
so pass `&shortn.TeamOptions{Workspace: workspaceID}` when a key reaches several workspaces.

```go
team, err := client.Teams.Create(ctx, workspaceID, shortn.CreateTeamParams{Name: "Paid Social"})
opts := &shortn.TeamOptions{Workspace: workspaceID}
got, err := client.Teams.Get(ctx, team.Slug, opts)
_, err = client.Teams.Update(ctx, team.Slug, shortn.UpdateTeamParams{Name: shortn.String("Paid Social EU")}, opts)
teams, err := client.Teams.List(ctx, workspaceID)
_, err = client.Teams.Delete(ctx, team.Slug, opts) // archives
```

### Domains

Adding, verifying and deleting a domain need an account-wide key.

```go
domains, err := client.Domains.List(ctx)
added, err := client.Domains.Create(ctx, shortn.CreateDomainParams{Domain: "go.example.com"})
check, err := client.Domains.Verify(ctx, added.Domain.ID)
_, err = client.Domains.Delete(ctx, added.Domain.ID)
```

### Reports and alerts

```go
rules, err := client.Reports.ListRules(ctx, &shortn.ReportRuleListOptions{WorkspacePublicID: workspaceID})
rule, err := client.Reports.GetRule(ctx, rules.Data[0].PublicID)
created, err := client.Reports.CreateRule(ctx, shortn.ReportRuleParams{
	WorkspacePublicID:    workspaceID,
	ReportType:           "custom_builder_v1",
	IntervalSeconds:      604800,
	ReportName:           "Weekly traffic",
	DestinationPublicIDs: shortn.Strings(destinationID),
	DefinitionJSON:       rule.Data.Definition, // build one in the dashboard and read it back
})
_, err = client.Reports.UpdateRule(ctx, created.Data.PublicID, shortn.ReportRuleParams{Enabled: shortn.Bool(false)})
err = client.Reports.DeleteRule(ctx, created.Data.PublicID)
```

`CreateRule` creates rules enabled; pause one with `UpdateRule`.

### Conversions

Send events from your backend, with the click id the
[capture script](https://docs.shortn.link/docs/conversion-tracking/install-capture-script/)
stored on your site.

```go
lead, err := client.Conversions.TrackLead(ctx, shortn.TrackLeadParams{
	ClickID: clickID, EventName: "Sign up", LeadCategory: "signup",
	IdempotencyKey: "signup_8412",
})
sale, err := client.Conversions.TrackSale(ctx, shortn.TrackSaleParams{
	ClickID: clickID, AmountCents: shortn.Int64(4900), Currency: "EUR",
	InvoiceID: "inv_123", IdempotencyKey: "sale_inv_123",
})
refund, err := client.Conversions.TrackRefund(ctx, shortn.TrackRefundParams{
	InvoiceID: "inv_123", RefundID: "re_456", AmountCents: shortn.Int64(1900),
	IdempotencyKey: "refund_re_456",
})

// Same checks, nothing recorded:
check, err := client.Conversions.ValidateSale(ctx, shortn.TrackSaleParams{ClickID: clickID, Amount: "49.00", Currency: "EUR", InvoiceID: "inv_124"})

summary, err := client.Conversions.Summary(ctx)
byTarget, err := client.Conversions.Attribution(ctx, &shortn.AttributionOptions{From: "2026-11-01", To: "2026-11-30"})
timeline, err := client.Conversions.Timeline(ctx, &shortn.TimelineOptions{Interval: "week"})
events, err := client.Conversions.Events(ctx, &shortn.EventsOptions{EventType: "sale", Limit: 25})
```

Send `TestMode: true` to record test events, and read them back with `Environment: "test"`.
Test events are never deduplicated (each request records a new one), so the client never
retries a test-mode call.

### Public helpers (no key needed)

```go
public, err := shortn.NewPublicClient()
validation, err := public.Public.ValidateURL(ctx, "example.com/menu") // Valid, Normalized
png, err := public.Public.QRCode(ctx, shortn.QRCodeParams{URL: "https://shtnl.ink/abC12x9", Size: "download"})
catalog, err := public.Public.Plans(ctx)
```

A client without a key can call only these three; every other method returns
`shortn.ErrAPIKeyRequired` without sending a request. Creating a link without a key and the
anonymous link preview need a browser human check on shortn.link, so the SDK doesn't offer
them: with a key, use `Links.Create` and `Analytics.Link`.

## Errors

Every non-2xx answer is an `*shortn.APIError`:

```go
var apiErr *shortn.APIError
if errors.As(err, &apiErr) {
	switch apiErr.Code {
	case "plan_limit_exceeded":
		log.Printf("plan limit: %v (needs %v)", apiErr.Raw["feature"], apiErr.Raw["required_plan"])
	case "link_not_found":
		// ...
	}
	log.Printf("status=%d code=%s message=%s hint=%s field=%s", apiErr.StatusCode, apiErr.Code, apiErr.Message, apiErr.Hint, apiErr.Field)
}
```

`APIError` carries `StatusCode`, `Status`, `Code` (stable, safe to branch on), `Message`
(the API's `error`), `Hint`, `Field` (`details.field`), `Details`, `RequestID` (conversion
endpoints), `RetryAfter`, `Raw` (the decoded body) and `Body` (the raw bytes).

Every error the API sends carries a `code`. `Code` is empty only when the answer didn't
come from the API, such as a proxy's `502` page: branch on `StatusCode` then. The UTM
template update's `409` (`lost_access_action_required`) lists the affected links in
`Raw["impact"]`; send `LostAccessAction` to go ahead.

## Retries

The client retries HTTP `429` and `503` answers, waiting for `Retry-After` when the server
sends it, and network errors, with exponential backoff (default: 2 retries, 500 ms doubling
up to 5 s, with jitter). It retries only requests that are safe to repeat: `GET`, `PUT` and
`DELETE`, and `POST`s that carry an `IdempotencyKey` (the conversion endpoints). Other
`POST`s and `PATCH`es are sent once. A conversion call with `TestMode: true` is also sent
once, even with an `IdempotencyKey`: the API never deduplicates test events, so a retry after
a lost answer would record a second test event.

`Retry-After` is honoured up to `RetryOptions.MaxRetryAfter` (default 60 s). When the server
asks for a longer wait, the client neither waits nor retries and returns the `*APIError` at
once. The monthly quota answer (`429 api_monthly_limit_exceeded`) is such a case: its
`Retry-After` is the time until the next month, readable in `apiErr.RetryAfter`.

```go
shortn.WithRetry(shortn.RetryOptions{MaxRetries: 4, InitialDelay: time.Second, MaxDelay: 10 * time.Second, MaxRetryAfter: 2 * time.Minute})
shortn.WithRetry(shortn.RetryOptions{Disabled: true})
```

A cancelled context stops waiting at once.

## Development

```sh
go test ./...   # offline unit tests with a mocked transport
go vet ./...
```

The smoke test in `smoke_test.go` runs against a live non-production instance and is skipped
unless `SHORTN_API_KEY` is set. It also needs `SHORTN_BASE_URL`, `SHORTN_SMOKE_WORKSPACE` and
`SHORTN_SMOKE_CLICK_ID` (see the comment at its top); it refuses `app.shortn.link`.

```sh
SHORTN_API_KEY=... SHORTN_BASE_URL=https://staging.example.com \
SHORTN_SMOKE_WORKSPACE=... SHORTN_SMOKE_CLICK_ID=... go test -run TestSmoke -v .
```

Runnable examples live in [`examples`](./examples).

## License

MIT
