package shortn

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

// mockResponse is one canned answer of mockTransport. A non-nil err fails
// the round trip as a network error.
type mockResponse struct {
	status int
	body   string
	header http.Header
	err    error
}

type recordedRequest struct {
	method string
	path   string // escaped, as sent
	query  string // encoded, as sent
	header http.Header
	body   []byte
}

// mockTransport answers requests from a queue and records them. When the
// queue runs out it repeats the last answer.
type mockTransport struct {
	mu        sync.Mutex
	responses []mockResponse
	requests  []recordedRequest
}

func (m *mockTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var body []byte
	if r.Body != nil {
		body, _ = io.ReadAll(r.Body)
	}
	m.requests = append(m.requests, recordedRequest{
		method: r.Method,
		path:   r.URL.EscapedPath(),
		query:  r.URL.RawQuery,
		header: r.Header.Clone(),
		body:   body,
	})
	resp := mockResponse{status: http.StatusOK, body: `{}`}
	if len(m.responses) > 0 {
		resp = m.responses[0]
		if len(m.responses) > 1 {
			m.responses = m.responses[1:]
		}
	}
	if resp.err != nil {
		return nil, resp.err
	}
	header := resp.header.Clone()
	if header == nil {
		header = http.Header{}
	}
	if header.Get("Content-Type") == "" {
		header.Set("Content-Type", "application/json")
	}
	status := resp.status
	if status == 0 {
		status = http.StatusOK
	}
	return &http.Response{
		StatusCode: status,
		Status:     http.StatusText(status),
		Header:     header,
		Body:       io.NopCloser(strings.NewReader(resp.body)),
		Request:    r,
	}, nil
}

func newTestClient(t *testing.T, apiKey string, responses ...mockResponse) (*Client, *mockTransport, *[]time.Duration) {
	t.Helper()
	transport := &mockTransport{responses: responses}
	client, err := NewClient(apiKey,
		WithBaseURL("https://api.test"),
		WithHTTPClient(&http.Client{Transport: transport}),
	)
	if err != nil {
		t.Fatal(err)
	}
	var delays []time.Duration
	client.sleep = func(ctx context.Context, d time.Duration) error {
		delays = append(delays, d)
		return ctx.Err()
	}
	return client, transport, &delays
}

func jsonEqual(t *testing.T, got []byte, want string) {
	t.Helper()
	var g, w any
	if err := json.Unmarshal(got, &g); err != nil {
		t.Fatalf("request body is not JSON: %q", got)
	}
	if err := json.Unmarshal([]byte(want), &w); err != nil {
		t.Fatalf("bad expected JSON %q: %v", want, err)
	}
	if !reflect.DeepEqual(g, w) {
		t.Fatalf("request body\n got: %s\nwant: %s", got, want)
	}
}

// methodCase checks one SDK method: the request it sends and how it decodes
// the answer.
type methodCase struct {
	name     string
	call     func(ctx context.Context, c *Client) (any, error)
	method   string
	path     string
	query    string
	body     string // "" means no body
	response string
	status   int
	check    func(t *testing.T, result any)
}

const odd = "id/1 x?" // escapes to id%2F1%20x%3F

func TestEveryMethod(t *testing.T) {
	cases := []methodCase{
		// links
		{
			name: "Links.List",
			call: func(ctx context.Context, c *Client) (any, error) {
				return c.Links.List(ctx, &LinkListOptions{Limit: 10, Offset: 20})
			},
			method:   "GET",
			path:     "/api/v1/links",
			query:    "limit=10&offset=20",
			response: `{"data":[{"public_id":"l1","short_code":"abc","clicks":3,"tags":[{"public_id":"t1","name":"x","color":"#fff"}],"new_field":1}],"pagination":{"count":1,"limit":10,"offset":20,"has_more":true,"next_offset":30}}`,
			check: func(t *testing.T, r any) {
				out := r.(*LinkList)
				if len(out.Data) != 1 || out.Data[0].PublicID != "l1" || out.Data[0].Clicks != 3 || out.Data[0].Tags[0].Name != "x" {
					t.Fatalf("unexpected links: %+v", out.Data)
				}
				if out.Data[0].Raw["new_field"] != float64(1) {
					t.Fatalf("unknown field not kept in Raw: %v", out.Data[0].Raw)
				}
				if !out.Pagination.HasMore || out.Pagination.NextOffset == nil || *out.Pagination.NextOffset != 30 {
					t.Fatalf("unexpected pagination: %+v", out.Pagination)
				}
			},
		},
		{
			name:     "Links.List without options",
			call:     func(ctx context.Context, c *Client) (any, error) { return c.Links.List(ctx, nil) },
			method:   "GET",
			path:     "/api/v1/links",
			response: `{"data":[],"pagination":{"count":0}}`,
		},
		{
			name:     "Links.Get",
			call:     func(ctx context.Context, c *Client) (any, error) { return c.Links.Get(ctx, odd) },
			method:   "GET",
			path:     "/api/v1/links/id%2F1%20x%3F",
			response: `{"data":{"public_id":"l1","title":"T","expires_at":"2026-12-31T23:00:00Z"}}`,
			check: func(t *testing.T, r any) {
				if out := r.(*LinkResponse); out.Data.Title != "T" || out.Data.ExpiresAt == "" {
					t.Fatalf("unexpected link: %+v", out.Data)
				}
			},
		},
		{
			name: "Links.Create",
			call: func(ctx context.Context, c *Client) (any, error) {
				return c.Links.Create(ctx, CreateLinkParams{
					URL: "https://example.com", Title: "Example", WorkspacePublicID: "w1",
					TagPublicIDs: []string{"t1"}, UTMSource: "news", ConversionEnabled: Bool(true),
					ABVariants:  []ABVariant{{URL: "https://a.example", SplitPercent: Int(60)}, {URL: "https://b.example"}},
					Attribution: &LinkAttribution{Type: "campaign", ExternalID: "c1"},
				})
			},
			method: "POST",
			path:   "/api/v1/links",
			body: `{"url":"https://example.com","title":"Example","workspace_public_id":"w1","tag_public_ids":["t1"],"utm_source":"news",
				"conversion_enabled":true,"ab_variants":[{"url":"https://a.example","split_percent":60},{"url":"https://b.example"}],
				"attribution":{"type":"campaign","external_id":"c1"}}`,
			status:   201,
			response: `{"public_id":"l1","short_url":"https://shtnl.ink/abc","short_code":"abc","link_status":"active","security_verdict":"clean"}`,
			check: func(t *testing.T, r any) {
				if out := r.(*CreatedLink); out.ShortCode != "abc" || out.LinkStatus != "active" {
					t.Fatalf("unexpected created link: %+v", out)
				}
			},
		},
		{
			name: "Links.Update",
			call: func(ctx context.Context, c *Client) (any, error) {
				return c.Links.Update(ctx, odd, UpdateLinkParams{Title: String("New"), Active: Bool(false), TeamPublicIDs: Strings()})
			},
			method:   "PATCH",
			path:     "/api/v1/links/id%2F1%20x%3F",
			body:     `{"title":"New","active":false,"team_public_ids":[]}`,
			response: `{"message":"URL updated successfully","workspace_wide":true,"team_public_ids":[],"teams":[]}`,
			check: func(t *testing.T, r any) {
				if out := r.(*LinkUpdateResult); out.Message == "" || out.WorkspaceWide == nil || !*out.WorkspaceWide {
					t.Fatalf("unexpected update result: %+v", out)
				}
			},
		},
		{
			name:     "Links.Delete",
			call:     func(ctx context.Context, c *Client) (any, error) { return c.Links.Delete(ctx, odd) },
			method:   "DELETE",
			path:     "/api/v1/links/id%2F1%20x%3F",
			response: `{"message":"URL disabled successfully"}`,
			check: func(t *testing.T, r any) {
				if r.(*MessageResult).Message != "URL disabled successfully" {
					t.Fatalf("unexpected: %+v", r)
				}
			},
		},
		{
			name:     "Links.Enable",
			call:     func(ctx context.Context, c *Client) (any, error) { return c.Links.Enable(ctx, odd) },
			method:   "POST",
			path:     "/api/v1/links/id%2F1%20x%3F/enable",
			response: `{"message":"URL enabled successfully"}`,
		},
		{
			name: "Links.SetABWinner",
			call: func(ctx context.Context, c *Client) (any, error) {
				return c.Links.SetABWinner(ctx, odd, ABWinnerParams{Variant: "B"})
			},
			method:   "POST",
			path:     "/api/v1/links/id%2F1%20x%3F/ab/winner",
			body:     `{"variant":"B"}`,
			response: `{"winner":"B","test":1,"destination":"https://example.com/b"}`,
			check: func(t *testing.T, r any) {
				if out := r.(*ABWinnerResult); out.Winner != "B" || out.Test != 1 {
					t.Fatalf("unexpected: %+v", out)
				}
			},
		},
		// analytics
		{
			name:     "Analytics.Link",
			call:     func(ctx context.Context, c *Client) (any, error) { return c.Analytics.Link(ctx, odd) },
			method:   "GET",
			path:     "/api/v1/analytics/links/id%2F1%20x%3F",
			response: `{"total_clicks":12,"unique_clicks":9,"clicks_by_country":{"Czechia":12}}`,
			check: func(t *testing.T, r any) {
				if out := r.(*LinkAnalytics); out.TotalClicks != 12 || out.ClicksByCountry["Czechia"] != 12 {
					t.Fatalf("unexpected: %+v", out)
				}
			},
		},
		// workspaces
		{
			name:     "Workspaces.List",
			call:     func(ctx context.Context, c *Client) (any, error) { return c.Workspaces.List(ctx) },
			method:   "GET",
			path:     "/api/v1/workspaces",
			response: `{"data":[{"public_id":"w1","name":"Acme","role":"owner","link_count":4}]}`,
			check: func(t *testing.T, r any) {
				if out := r.(*WorkspaceList); len(out.Data) != 1 || out.Data[0].Role != "owner" || out.Data[0].LinkCount != 4 {
					t.Fatalf("unexpected: %+v", out)
				}
			},
		},
		{
			name:     "Workspaces.Get",
			call:     func(ctx context.Context, c *Client) (any, error) { return c.Workspaces.Get(ctx, odd) },
			method:   "GET",
			path:     "/api/v1/workspaces/id%2F1%20x%3F",
			response: `{"data":{"public_id":"w1","name":"Acme"}}`,
			check: func(t *testing.T, r any) {
				if r.(*WorkspaceResponse).Data.Name != "Acme" {
					t.Fatalf("unexpected: %+v", r)
				}
			},
		},
		// tags
		{
			name: "Tags.List",
			call: func(ctx context.Context, c *Client) (any, error) {
				return c.Tags.List(ctx, odd, &TagListOptions{TeamPublicIDs: []string{"a", "b"}, Team: "none", ReportScope: true})
			},
			method:   "GET",
			path:     "/api/v1/workspaces/id%2F1%20x%3F/tags",
			query:    "report_scope=1&team=none&team_public_ids=a%2Cb",
			response: `{"tags":[{"id":1,"public_id":"t1","name":"Sale","url_count":2}]}`,
			check: func(t *testing.T, r any) {
				if out := r.(*TagList); len(out.Tags) != 1 || out.Tags[0].URLCount != 2 {
					t.Fatalf("unexpected: %+v", out)
				}
			},
		},
		{
			name:     "Tags.List empty",
			call:     func(ctx context.Context, c *Client) (any, error) { return c.Tags.List(ctx, "w1", nil) },
			method:   "GET",
			path:     "/api/v1/workspaces/w1/tags",
			response: `{"tags":null}`,
			check: func(t *testing.T, r any) {
				if r.(*TagList).Tags != nil {
					t.Fatalf("unexpected: %+v", r)
				}
			},
		},
		{
			name: "Tags.Create",
			call: func(ctx context.Context, c *Client) (any, error) {
				return c.Tags.Create(ctx, odd, CreateTagParams{Name: "Sale", Color: "#3b82f6"})
			},
			method:   "POST",
			path:     "/api/v1/workspaces/id%2F1%20x%3F/tags",
			body:     `{"name":"Sale","color":"#3b82f6"}`,
			status:   201,
			response: `{"id":1,"public_id":"t1","name":"Sale","color":"#3b82f6","workspace_wide":true}`,
			check: func(t *testing.T, r any) {
				if out := r.(*Tag); out.PublicID != "t1" || !out.WorkspaceWide {
					t.Fatalf("unexpected: %+v", out)
				}
			},
		},
		{
			name:     "Tags.Get",
			call:     func(ctx context.Context, c *Client) (any, error) { return c.Tags.Get(ctx, odd) },
			method:   "GET",
			path:     "/api/v1/tags/id%2F1%20x%3F",
			response: `{"public_id":"t1","name":"Sale"}`,
		},
		{
			name: "Tags.Update",
			call: func(ctx context.Context, c *Client) (any, error) {
				return c.Tags.Update(ctx, odd, UpdateTagParams{TeamPublicIDs: Strings("team1")})
			},
			method:   "PUT",
			path:     "/api/v1/tags/id%2F1%20x%3F",
			body:     `{"team_public_ids":["team1"]}`,
			response: `{"public_id":"t1","team_public_ids":["team1"]}`,
			check: func(t *testing.T, r any) {
				if out := r.(*Tag); len(out.TeamPublicIDs) != 1 {
					t.Fatalf("unexpected: %+v", out)
				}
			},
		},
		{
			name:     "Tags.Delete",
			call:     func(ctx context.Context, c *Client) (any, error) { return c.Tags.Delete(ctx, odd) },
			method:   "DELETE",
			path:     "/api/v1/tags/id%2F1%20x%3F",
			response: `{"message":"Tag deleted successfully","links_updated":18}`,
			check: func(t *testing.T, r any) {
				if r.(*TagDeleteResult).LinksUpdated != 18 {
					t.Fatalf("unexpected: %+v", r)
				}
			},
		},
		// UTM templates
		{
			name: "UTMTemplates.List",
			call: func(ctx context.Context, c *Client) (any, error) {
				return c.UTMTemplates.List(ctx, odd, &UTMTemplateListOptions{IncludeTeamScoped: true, TeamPublicIDs: []string{"a"}})
			},
			method:   "GET",
			path:     "/api/v1/workspaces/id%2F1%20x%3F/utm-templates",
			query:    "include_team_scoped=1&team_public_ids=a",
			response: `{"data":[{"public_id":"u1","name":"News","utm_source":"newsletter"}]}`,
			check: func(t *testing.T, r any) {
				if out := r.(*UTMTemplateList); out.Data[0].UTMSource != "newsletter" {
					t.Fatalf("unexpected: %+v", out)
				}
			},
		},
		{
			name: "UTMTemplates.Create",
			call: func(ctx context.Context, c *Client) (any, error) {
				return c.UTMTemplates.Create(ctx, odd, CreateUTMTemplateParams{Name: "News", UTMSource: "newsletter", UTMMedium: "email"})
			},
			method:   "POST",
			path:     "/api/v1/workspaces/id%2F1%20x%3F/utm-templates",
			body:     `{"name":"News","utm_source":"newsletter","utm_medium":"email"}`,
			status:   201,
			response: `{"data":{"public_id":"u1","name":"News"}}`,
			check: func(t *testing.T, r any) {
				if r.(*UTMTemplateResponse).Data.PublicID != "u1" {
					t.Fatalf("unexpected: %+v", r)
				}
			},
		},
		{
			name: "UTMTemplates.Update",
			call: func(ctx context.Context, c *Client) (any, error) {
				return c.UTMTemplates.Update(ctx, odd, UpdateUTMTemplateParams{Name: "News 2", UTMCampaign: "dec", LostAccessAction: "copy"})
			},
			method:   "PUT",
			path:     "/api/v1/utm-templates/id%2F1%20x%3F",
			body:     `{"name":"News 2","utm_campaign":"dec","lost_access_action":"copy"}`,
			response: `{"data":{"public_id":"u1","name":"News 2","move_impact":{"count":1,"links":[{"public_id":"l1"}]}}}`,
			check: func(t *testing.T, r any) {
				if out := r.(*UTMTemplateResponse); out.Data.Name != "News 2" || out.MoveImpact["count"] != float64(1) || len(out.MoveImpact["links"].([]any)) != 1 {
					t.Fatalf("unexpected: %+v", out)
				}
			},
		},
		{
			name: "UTMTemplates.Delete",
			call: func(ctx context.Context, c *Client) (any, error) {
				return c.UTMTemplates.Delete(ctx, odd, &UTMTemplateDeleteOptions{Links: "clear"})
			},
			method:   "DELETE",
			path:     "/api/v1/utm-templates/id%2F1%20x%3F",
			query:    "links=clear",
			response: `{"data":{"public_id":"u1","deleted":true,"links":"clear","links_updated":2}}`,
			check: func(t *testing.T, r any) {
				if out := r.(*UTMTemplateDeleteResult); !out.Data.Deleted || out.Data.LinksUpdated != 2 {
					t.Fatalf("unexpected: %+v", out)
				}
			},
		},
		// destinations
		{
			name:     "Destinations.List",
			call:     func(ctx context.Context, c *Client) (any, error) { return c.Destinations.List(ctx, odd) },
			method:   "GET",
			path:     "/api/v1/workspaces/id%2F1%20x%3F/destinations",
			response: `[{"id":9,"public_id":"d1","type":"email","value":"a@example.com"}]`,
			check: func(t *testing.T, r any) {
				if out := r.([]Destination); len(out) != 1 || out[0].Value != "a@example.com" || out[0].Raw["id"] != float64(9) {
					t.Fatalf("unexpected: %+v", out)
				}
			},
		},
		{
			name: "Destinations.Create",
			call: func(ctx context.Context, c *Client) (any, error) {
				return c.Destinations.Create(ctx, odd, CreateDestinationParams{
					Name: "Hook", Type: "webhook", Config: map[string]any{"webhook_url": "https://example.com/h"},
				})
			},
			method:   "POST",
			path:     "/api/v1/workspaces/id%2F1%20x%3F/destinations",
			body:     `{"name":"Hook","type":"webhook","config":{"webhook_url":"https://example.com/h"}}`,
			status:   201,
			response: `{"id":10,"public_id":"d2","name":"Hook","type":"webhook","workspace_wide":true}`,
			check: func(t *testing.T, r any) {
				if r.(*Destination).PublicID != "d2" {
					t.Fatalf("unexpected: %+v", r)
				}
			},
		},
		{
			name: "Destinations.Update",
			call: func(ctx context.Context, c *Client) (any, error) {
				return c.Destinations.Update(ctx, odd, "d/2", UpdateDestinationParams{Name: "Hook 2", Type: "webhook"})
			},
			method:   "PUT",
			path:     "/api/v1/workspaces/id%2F1%20x%3F/destinations/d%2F2",
			body:     `{"name":"Hook 2","type":"webhook"}`,
			response: `{"success":true}`,
			check: func(t *testing.T, r any) {
				if out := r.(*DestinationUpdateResult); !out.Success || out.WorkspaceWide != nil {
					t.Fatalf("unexpected: %+v", out)
				}
			},
		},
		{
			name:     "Destinations.Delete",
			call:     func(ctx context.Context, c *Client) (any, error) { return c.Destinations.Delete(ctx, odd, "d/2") },
			method:   "DELETE",
			path:     "/api/v1/workspaces/id%2F1%20x%3F/destinations/d%2F2",
			response: `{"success":true}`,
			check: func(t *testing.T, r any) {
				if !r.(*SuccessResult).Success {
					t.Fatalf("unexpected: %+v", r)
				}
			},
		},
		// teams
		{
			name:     "Teams.List",
			call:     func(ctx context.Context, c *Client) (any, error) { return c.Teams.List(ctx, odd) },
			method:   "GET",
			path:     "/api/v1/workspaces/id%2F1%20x%3F/teams",
			response: `{"teams":[{"id":31,"public_id":"tm1","slug":"paid-social","member_count":3}]}`,
			check: func(t *testing.T, r any) {
				if out := r.(*TeamList); out.Teams[0].Slug != "paid-social" || out.Teams[0].MemberCount != 3 {
					t.Fatalf("unexpected: %+v", out)
				}
			},
		},
		{
			name: "Teams.Create",
			call: func(ctx context.Context, c *Client) (any, error) {
				return c.Teams.Create(ctx, odd, CreateTeamParams{Name: "Paid Social", Description: "Ads"})
			},
			method:   "POST",
			path:     "/api/v1/workspaces/id%2F1%20x%3F/teams",
			body:     `{"name":"Paid Social","description":"Ads"}`,
			response: `{"id":31,"slug":"paid-social","public_id":"tm1"}`,
			check: func(t *testing.T, r any) {
				if out := r.(*CreatedTeam); out.Slug != "paid-social" || out.ID != 31 {
					t.Fatalf("unexpected: %+v", out)
				}
			},
		},
		{
			name: "Teams.Get",
			call: func(ctx context.Context, c *Client) (any, error) {
				return c.Teams.Get(ctx, "paid social", &TeamOptions{Workspace: "w1"})
			},
			method:   "GET",
			path:     "/api/v1/teams/paid%20social",
			query:    "workspace=w1",
			response: `{"id":31,"slug":"paid-social","workspace_public_id":"w1"}`,
			check: func(t *testing.T, r any) {
				if r.(*Team).WorkspacePublicID != "w1" {
					t.Fatalf("unexpected: %+v", r)
				}
			},
		},
		{
			name: "Teams.Update",
			call: func(ctx context.Context, c *Client) (any, error) {
				return c.Teams.Update(ctx, "paid-social", UpdateTeamParams{Name: String("Paid EU"), Description: String("")}, nil)
			},
			method:   "PUT",
			path:     "/api/v1/teams/paid-social",
			body:     `{"name":"Paid EU","description":""}`,
			response: `{"id":31,"name":"Paid EU","slug":"paid-social"}`,
			check: func(t *testing.T, r any) {
				if r.(*Team).Name != "Paid EU" {
					t.Fatalf("unexpected: %+v", r)
				}
			},
		},
		{
			name: "Teams.Delete",
			call: func(ctx context.Context, c *Client) (any, error) {
				return c.Teams.Delete(ctx, "a/b", &TeamOptions{Workspace: "w1"})
			},
			method:   "DELETE",
			path:     "/api/v1/teams/a%2Fb",
			query:    "workspace=w1",
			response: `{"success":true}`,
		},
		// domains
		{
			name:   "Domains.List",
			call:   func(ctx context.Context, c *Client) (any, error) { return c.Domains.List(ctx) },
			method: "GET",
			path:   "/api/v1/domains",
			response: `{"data":[{"id":14,"domain":"go.example.com","status":"ready","links_using":37,
				"dns":{"verification_txt":{"type":"TXT","name":"_shortn-verify.go.example.com","value":"v","ttl":"300"},"apex_a_records":[{"type":"A","name":"go.example.com","value":"203.0.113.10","ttl":"300"}]}}]}`,
			check: func(t *testing.T, r any) {
				out := r.(*DomainList)
				if out.Data[0].ID != 14 || out.Data[0].DNS == nil || out.Data[0].DNS.VerificationTXT.Type != "TXT" || len(out.Data[0].DNS.ApexARecords) != 1 {
					t.Fatalf("unexpected: %+v", out)
				}
			},
		},
		{
			name: "Domains.Create",
			call: func(ctx context.Context, c *Client) (any, error) {
				return c.Domains.Create(ctx, CreateDomainParams{Domain: "go.example.com"})
			},
			method:   "POST",
			path:     "/api/v1/domains",
			body:     `{"domain":"go.example.com"}`,
			status:   201,
			response: `{"domain":{"id":14,"domain":"go.example.com","provisioning_status":"pending_dns","verification_token":"tok"}}`,
			check: func(t *testing.T, r any) {
				if out := r.(*DomainResponse); out.Domain.VerificationToken != "tok" {
					t.Fatalf("unexpected: %+v", out)
				}
			},
		},
		{
			name:     "Domains.Verify",
			call:     func(ctx context.Context, c *Client) (any, error) { return c.Domains.Verify(ctx, 14) },
			method:   "POST",
			path:     "/api/v1/domains/14/verify",
			status:   202,
			response: `{"domain":{"id":14,"status":"in_progress"},"verification":{"verified":true,"txt":{"match":true},"routing":{"any_route_ok":true}}}`,
			check: func(t *testing.T, r any) {
				if out := r.(*DomainVerifyResult); !out.Verification.Verified || out.Verification.TXT["match"] != true {
					t.Fatalf("unexpected: %+v", out)
				}
			},
		},
		{
			name:     "Domains.Delete",
			call:     func(ctx context.Context, c *Client) (any, error) { return c.Domains.Delete(ctx, 14) },
			method:   "DELETE",
			path:     "/api/v1/domains/14",
			status:   202,
			response: `{"deleted":true,"id":14,"domain":{"id":14,"status":"deleting"}}`,
			check: func(t *testing.T, r any) {
				if out := r.(*DomainDeleteResult); !out.Deleted || out.Domain.Status != "deleting" {
					t.Fatalf("unexpected: %+v", out)
				}
			},
		},
		// reports
		{
			name: "Reports.ListRules",
			call: func(ctx context.Context, c *Client) (any, error) {
				return c.Reports.ListRules(ctx, &ReportRuleListOptions{WorkspacePublicID: "w1", ScheduleTypes: []string{"every_x", "event"}, LinkPublicIDs: []string{"l1"}, TeamPublicIDs: []string{"tm1"}})
			},
			method:   "GET",
			path:     "/api/v1/reports/rules",
			query:    "link_public_ids=l1&schedule_types=every_x%2Cevent&team_public_ids=tm1&workspace_public_id=w1",
			response: `{"data":[{"public_id":"r1","schedule_type":"every_x","schedule_interval_seconds":604800,"next_run_at":null,"destinations":[{"type":"email","data":{"to":["a@example.com"]}}]}]}`,
			check: func(t *testing.T, r any) {
				out := r.(*ReportRuleList)
				if out.Data[0].ScheduleIntervalSeconds == nil || *out.Data[0].ScheduleIntervalSeconds != 604800 || out.Data[0].Destinations[0].Type != "email" {
					t.Fatalf("unexpected: %+v", out)
				}
			},
		},
		{
			name:     "Reports.GetRule",
			call:     func(ctx context.Context, c *Client) (any, error) { return c.Reports.GetRule(ctx, odd) },
			method:   "GET",
			path:     "/api/v1/reports/rules/id%2F1%20x%3F",
			response: `{"data":{"public_id":"r1","definition":{"version":"custom_builder.v1"},"include_unassigned_links":true}}`,
			check: func(t *testing.T, r any) {
				out := r.(*ReportRuleResponse)
				if out.Data.Definition["version"] != "custom_builder.v1" || out.Data.IncludeUnassignedLinks == nil {
					t.Fatalf("unexpected: %+v", out)
				}
			},
		},
		{
			name: "Reports.CreateRule",
			call: func(ctx context.Context, c *Client) (any, error) {
				return c.Reports.CreateRule(ctx, ReportRuleParams{
					WorkspacePublicID: "w1", ReportType: "custom_builder_v1", ScheduleType: "every_x", IntervalSeconds: 604800,
					ReportName: "Weekly", DefinitionJSON: map[string]any{"version": "custom_builder.v1"},
					DestinationPublicIDs: Strings(), Enabled: Bool(false),
				})
			},
			method: "POST",
			path:   "/api/v1/reports/rules",
			body: `{"workspace_public_id":"w1","report_type":"custom_builder_v1","schedule_type":"every_x","interval_seconds":604800,
				"report_name":"Weekly","definition_json":{"version":"custom_builder.v1"},"destination_public_ids":[],"enabled":false}`,
			status:   201,
			response: `{"data":{"public_id":"r1","enabled":false}}`,
		},
		{
			name: "Reports.UpdateRule",
			call: func(ctx context.Context, c *Client) (any, error) {
				return c.Reports.UpdateRule(ctx, odd, ReportRuleParams{Enabled: Bool(true)})
			},
			method:   "PATCH",
			path:     "/api/v1/reports/rules/id%2F1%20x%3F",
			body:     `{"enabled":true}`,
			response: `{"data":{"public_id":"r1","enabled":true}}`,
			check: func(t *testing.T, r any) {
				if !r.(*ReportRuleResponse).Data.Enabled {
					t.Fatalf("unexpected: %+v", r)
				}
			},
		},
		{
			name:   "Reports.DeleteRule",
			call:   func(ctx context.Context, c *Client) (any, error) { return nil, c.Reports.DeleteRule(ctx, odd) },
			method: "DELETE",
			path:   "/api/v1/reports/rules/id%2F1%20x%3F",
			status: 204,
		},
		// conversions
		{
			name: "Conversions.TrackLead",
			call: func(ctx context.Context, c *Client) (any, error) {
				return c.Conversions.TrackLead(ctx, TrackLeadParams{ClickID: "clk_1", EventName: "Sign up", LeadCategory: "signup", IdempotencyKey: "k1", TestMode: true, Metadata: map[string]any{"plan": "pro"}})
			},
			method:   "POST",
			path:     "/api/v1/conversions/leads",
			body:     `{"click_id":"clk_1","event_name":"Sign up","lead_category":"signup","idempotency_key":"k1","test_mode":true,"metadata":{"plan":"pro"}}`,
			response: `{"type":"lead","event":{"public_id":"e1","event_type":"lead","duplicate":false,"is_test":true,"attribution":{"type":"workspace"}}}`,
			check: func(t *testing.T, r any) {
				if out := r.(*ConversionResult); out.Type != "lead" || !out.Event.IsTest || out.Event.Attribution.Type != "workspace" {
					t.Fatalf("unexpected: %+v", out)
				}
			},
		},
		{
			name: "Conversions.ValidateLead",
			call: func(ctx context.Context, c *Client) (any, error) {
				return c.Conversions.ValidateLead(ctx, TrackLeadParams{ClickID: "clk_1", IdempotencyKey: "k1"})
			},
			method:   "POST",
			path:     "/api/v1/conversions/leads/validations",
			body:     `{"click_id":"clk_1","idempotency_key":"k1"}`,
			response: `{"valid":true,"dry_run":true,"result":{"valid":true,"dry_run":true,"event_type":"lead","would_track":true}}`,
			check: func(t *testing.T, r any) {
				if out := r.(*ConversionValidation); !out.Valid || !out.Result.WouldTrack || out.Result.EventType != "lead" {
					t.Fatalf("unexpected: %+v", out)
				}
			},
		},
		{
			name: "Conversions.TrackSale",
			call: func(ctx context.Context, c *Client) (any, error) {
				return c.Conversions.TrackSale(ctx, TrackSaleParams{ClickID: "clk_1", AmountCents: Int64(4900), Currency: "EUR", InvoiceID: "inv_1", SettlementAmountCents: 5300, SettlementCurrency: "USD"})
			},
			method:   "POST",
			path:     "/api/v1/conversions/sales",
			body:     `{"click_id":"clk_1","amount_cents":4900,"currency":"EUR","invoice_id":"inv_1","settlement_amount_cents":5300,"settlement_currency":"USD"}`,
			response: `{"type":"sale","event":{"public_id":"e2","amount_cents":4900,"amount":"49.00","currency_decimals":2,"currency":"EUR"}}`,
			check: func(t *testing.T, r any) {
				if out := r.(*ConversionResult); out.Event.AmountCents == nil || *out.Event.AmountCents != 4900 || out.Event.Amount != "49.00" {
					t.Fatalf("unexpected: %+v", out)
				}
			},
		},
		{
			name: "Conversions.ValidateSale",
			call: func(ctx context.Context, c *Client) (any, error) {
				return c.Conversions.ValidateSale(ctx, TrackSaleParams{ClickID: "clk_1", Amount: "49.00", Currency: "EUR", IdempotencyKey: "k2"})
			},
			method:   "POST",
			path:     "/api/v1/conversions/sales/validations",
			body:     `{"click_id":"clk_1","amount":"49.00","currency":"EUR","idempotency_key":"k2"}`,
			response: `{"valid":true,"dry_run":true,"result":{"would_infer_lead":true}}`,
		},
		{
			name: "Conversions.TrackRefund",
			call: func(ctx context.Context, c *Client) (any, error) {
				return c.Conversions.TrackRefund(ctx, TrackRefundParams{InvoiceID: "inv_1", RefundID: "re_1", AmountCents: Int64(1900), Reason: "Damaged"})
			},
			method:   "POST",
			path:     "/api/v1/conversions/refunds",
			body:     `{"invoice_id":"inv_1","refund_id":"re_1","amount_cents":1900,"reason":"Damaged"}`,
			response: `{"type":"refund","event":{"public_id":"e3","refunded_amount_cents":1900,"remaining_amount_cents":3000,"refund_status":"partial"}}`,
			check: func(t *testing.T, r any) {
				if out := r.(*ConversionResult); out.Event.RemainingAmountCents == nil || *out.Event.RemainingAmountCents != 3000 {
					t.Fatalf("unexpected: %+v", out)
				}
			},
		},
		{
			name: "Conversions.ValidateRefund",
			call: func(ctx context.Context, c *Client) (any, error) {
				return c.Conversions.ValidateRefund(ctx, TrackRefundParams{InvoiceID: "inv_1", RefundID: "re_1", Amount: "19.00", Currency: "EUR"})
			},
			method:   "POST",
			path:     "/api/v1/conversions/refunds/validations",
			body:     `{"invoice_id":"inv_1","refund_id":"re_1","amount":"19.00","currency":"EUR"}`,
			response: `{"valid":true,"dry_run":true,"result":{"event_type":"refund"}}`,
		},
		{
			name:   "Conversions.Summary",
			call:   func(ctx context.Context, c *Client) (any, error) { return c.Conversions.Summary(ctx) },
			method: "GET",
			path:   "/api/v1/conversions/summary",
			response: `{"data":{"leads":120,"sales":31,"sale_amount_cents":null,"currency":null,"mixed_currencies":true,"lead_to_sale_rate":0.25,
				"reporting":{"currency":"EUR","net_revenue_cents":147000,"approximate":true,"excluded":[{"currency":"XYZ","sales":1,"refunds":0}]}}}`,
			check: func(t *testing.T, r any) {
				d := r.(*ConversionSummary).Data
				if d.Leads != 120 || d.SaleAmountCents != nil || d.Currency != nil || !d.MixedCurrencies || d.Reporting == nil || *d.Reporting.NetRevenueCents != 147000 || d.Reporting.Excluded[0].Currency != "XYZ" {
					t.Fatalf("unexpected: %+v", d)
				}
			},
		},
		{
			name: "Conversions.Attribution",
			call: func(ctx context.Context, c *Client) (any, error) {
				return c.Conversions.Attribution(ctx, &AttributionOptions{From: "2026-11-01", To: "2026-11-30", Environment: "test", Limit: 5})
			},
			method: "GET",
			path:   "/api/v1/conversions/attribution",
			query:  "environment=test&from=2026-11-01&limit=5&to=2026-11-30",
			response: `{"data":[{"attribution":{"type":"workspace"},"display_name":"Workspace","clicks":10,"leads":4}],
				"totals":{"clicks":10,"leads":4,"journey":{"clicks":10}},"pagination":{"count":1,"limit":5},"date_range":{"from":"2026-11-01T00:00:00Z","to_exclusive":true},"environment":"test"}`,
			check: func(t *testing.T, r any) {
				out := r.(*AttributionReport)
				if out.Data[0].Clicks != 10 || out.Data[0].DisplayName != "Workspace" || out.Totals.Leads != 4 || out.Totals.Raw["journey"] == nil || out.DateRange["to_exclusive"] != true {
					t.Fatalf("unexpected: %+v", out)
				}
			},
		},
		{
			name: "Conversions.Timeline",
			call: func(ctx context.Context, c *Client) (any, error) {
				return c.Conversions.Timeline(ctx, &TimelineOptions{Environment: "test", Interval: "week"})
			},
			method:   "GET",
			path:     "/api/v1/conversions/timeline",
			query:    "environment=test&interval=week",
			response: `{"data":[{"start":"2026-11-02T00:00:00Z","end":"2026-11-09T00:00:00Z","clicks":3,"sales":1}],"interval":"week"}`,
			check: func(t *testing.T, r any) {
				out := r.(*ConversionTimeline)
				if out.Interval != "week" || out.Data[0].Start == "" || out.Data[0].Sales != 1 {
					t.Fatalf("unexpected: %+v", out)
				}
			},
		},
		{
			name: "Conversions.Events",
			call: func(ctx context.Context, c *Client) (any, error) {
				return c.Conversions.Events(ctx, &EventsOptions{Environment: "test", Limit: 2, Offset: 4, EventType: "sale", LeadCategory: "signup", EventOrigin: "observed", Link: "abc", Event: "e1", Query: "big deal"})
			},
			method:   "GET",
			path:     "/api/v1/conversions/events",
			query:    "environment=test&event=e1&event_origin=observed&event_type=sale&lead_category=signup&limit=2&link=abc&offset=4&q=big+deal",
			response: `{"data":[{"public_id":"e1","event_type":"sale","reporting":{"currency":"EUR","amount_cents":4512,"approximate":true}}],"pagination":{"count":1,"limit":2,"offset":4,"has_more":false,"next_offset":null},"environment":"test"}`,
			check: func(t *testing.T, r any) {
				out := r.(*ConversionEventList)
				if out.Data[0].Reporting == nil || *out.Data[0].Reporting.AmountCents != 4512 || out.Pagination.Offset != 4 || out.Pagination.NextOffset != nil {
					t.Fatalf("unexpected: %+v", out)
				}
			},
		},
		// public
		{
			name:     "Public.ValidateURL",
			call:     func(ctx context.Context, c *Client) (any, error) { return c.Public.ValidateURL(ctx, "example.com/a b") },
			method:   "GET",
			path:     "/api/v1/validate",
			query:    "url=example.com%2Fa+b",
			response: `{"valid":true,"normalized":"https://example.com/a%20b"}`,
			check: func(t *testing.T, r any) {
				if out := r.(*URLValidation); !out.Valid || out.Normalized == "" {
					t.Fatalf("unexpected: %+v", out)
				}
			},
		},
		{
			name: "Public.QRCode",
			call: func(ctx context.Context, c *Client) (any, error) {
				return c.Public.QRCode(ctx, QRCodeParams{URL: "https://shtnl.ink/abc", Format: "svg", Size: "download", Foreground: "1f2937", Background: "transparent", FrameTextFollowsBackground: Bool(false)})
			},
			method:   "GET",
			path:     "/api/v1/qr",
			query:    "bg=transparent&fg=1f2937&format=svg&frameTextFollowsBackground=false&size=download&url=https%3A%2F%2Fshtnl.ink%2Fabc",
			response: "<svg/>",
			check: func(t *testing.T, r any) {
				if string(r.([]byte)) != "<svg/>" {
					t.Fatalf("unexpected: %q", r)
				}
			},
		},
		{
			name:     "Public.Plans",
			call:     func(ctx context.Context, c *Client) (any, error) { return c.Public.Plans(ctx) },
			method:   "GET",
			path:     "/api/v1/plans",
			response: `{"plans":{"free":{"ID":"free","Name":"Always Free","Price":0,"MaxLinks":25,"MaxClicks":1000}}}`,
			check: func(t *testing.T, r any) {
				if out := r.(*PlanCatalog); out.Plans["free"].MaxLinks != 25 || out.Plans["free"].Name != "Always Free" {
					t.Fatalf("unexpected: %+v", out)
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client, transport, _ := newTestClient(t, "sk_test_123", mockResponse{status: tc.status, body: tc.response})
			result, err := tc.call(context.Background(), client)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(transport.requests) != 1 {
				t.Fatalf("expected 1 request, got %d", len(transport.requests))
			}
			req := transport.requests[0]
			if req.method != tc.method {
				t.Errorf("method = %s, want %s", req.method, tc.method)
			}
			if req.path != tc.path {
				t.Errorf("path = %s, want %s", req.path, tc.path)
			}
			if req.query != tc.query {
				t.Errorf("query = %q, want %q", req.query, tc.query)
			}
			if got := req.header.Get("Authorization"); got != "Bearer sk_test_123" {
				t.Errorf("Authorization = %q", got)
			}
			if got := req.header.Get("User-Agent"); got != "shortn-sdk-go/"+Version {
				t.Errorf("User-Agent = %q", got)
			}
			if tc.body == "" {
				if len(req.body) != 0 || req.header.Get("Content-Type") != "" {
					t.Errorf("expected no body, got %q (Content-Type %q)", req.body, req.header.Get("Content-Type"))
				}
			} else {
				if req.header.Get("Content-Type") != "application/json" {
					t.Errorf("Content-Type = %q", req.header.Get("Content-Type"))
				}
				jsonEqual(t, req.body, tc.body)
			}
			if tc.check != nil {
				tc.check(t, result)
			}
		})
	}
	if len(cases) < 50 {
		t.Fatalf("expected a case for each of the 50 methods, have %d cases", len(cases))
	}
}

func TestPublicHelpersWorkWithoutKey(t *testing.T) {
	client, transport, _ := newTestClient(t, "", mockResponse{body: `{"valid":true}`})
	if _, err := client.Public.ValidateURL(context.Background(), "example.com"); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Public.Plans(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Public.QRCode(context.Background(), QRCodeParams{URL: "https://example.com"}); err != nil {
		t.Fatal(err)
	}
	if len(transport.requests) != 3 {
		t.Fatalf("expected 3 requests, got %d", len(transport.requests))
	}
	for _, req := range transport.requests {
		if req.header.Get("Authorization") != "" {
			t.Fatalf("key-less client sent Authorization: %q", req.header.Get("Authorization"))
		}
	}
	if got := transport.requests[2].header.Get("Accept"); got != "image/*" {
		t.Fatalf("QR Accept = %q", got)
	}
}

func TestKeyedMethodsNeedKey(t *testing.T) {
	client, transport, _ := newTestClient(t, "")
	_, err := client.Links.List(context.Background(), nil)
	if !errors.Is(err, ErrAPIKeyRequired) {
		t.Fatalf("expected ErrAPIKeyRequired, got %v", err)
	}
	if err := client.Reports.DeleteRule(context.Background(), "r1"); !errors.Is(err, ErrAPIKeyRequired) {
		t.Fatalf("expected ErrAPIKeyRequired, got %v", err)
	}
	if len(transport.requests) != 0 {
		t.Fatalf("no request should be sent without a key, got %d", len(transport.requests))
	}
	public, err := NewPublicClient()
	if err != nil || public.Public == nil {
		t.Fatalf("NewPublicClient: %v", err)
	}
}

func TestClientOptions(t *testing.T) {
	transport := &mockTransport{}
	client, err := NewClient("sk_test",
		WithHTTPClient(&http.Client{Transport: transport}),
		WithBaseURL("https://dev.example.test/api/v1/"),
		WithTimeout(3*time.Second),
		WithUserAgent("my-app/1.2"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if client.httpClient.Timeout != 3*time.Second || client.httpClient.Transport != transport {
		t.Fatalf("timeout or transport lost: %+v", client.httpClient)
	}
	if _, err := client.Workspaces.List(context.Background()); err != nil {
		t.Fatal(err)
	}
	req := transport.requests[0]
	if req.path != "/api/v1/workspaces" {
		t.Fatalf("path = %s", req.path)
	}
	if got := req.header.Get("User-Agent"); got != "shortn-sdk-go/0.1.0 my-app/1.2" {
		t.Fatalf("User-Agent = %q", got)
	}
	if got := req.header.Get("Accept"); got != "application/json" {
		t.Fatalf("Accept = %q", got)
	}
	if client.baseURL != "https://dev.example.test" {
		t.Fatalf("baseURL = %q", client.baseURL)
	}
	if _, err := NewClient("k", WithBaseURL("not a url")); err == nil {
		t.Fatal("expected an invalid base url error")
	}
	if _, err := NewClient("k", WithTimeout(-1)); err == nil {
		t.Fatal("expected a negative timeout error")
	}
	if _, err := NewClient("k", WithHTTPClient(nil)); err == nil {
		t.Fatal("expected a nil http client error")
	}
	if c, _ := NewClient("k"); c.baseURL != DefaultBaseURL {
		t.Fatalf("default base url = %q", c.baseURL)
	}
}

func TestErrorMapping(t *testing.T) {
	client, _, _ := newTestClient(t, "sk_test", mockResponse{
		status: 400,
		header: http.Header{"Shortn-Request-Id": {"req_header"}},
		body:   `{"code":"conversion_identity_required","error":"idempotency_key is required","hint":"Send a stable key.","details":{"field":"idempotency_key"},"request_id":"req_1"}`,
	})
	_, err := client.Conversions.TrackLead(context.Background(), TrackLeadParams{ClickID: "clk"})
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T %v", err, err)
	}
	if apiErr.StatusCode != 400 || apiErr.Code != "conversion_identity_required" || apiErr.Message != "idempotency_key is required" ||
		apiErr.Hint != "Send a stable key." || apiErr.Field != "idempotency_key" || apiErr.Details["field"] != "idempotency_key" ||
		apiErr.RequestID != "req_1" || !bytes.Contains(apiErr.Body, []byte("conversion_identity_required")) {
		t.Fatalf("unexpected error: %+v", apiErr)
	}
	if apiErr.Error() != "shortn: 400 conversion_identity_required: idempotency_key is required" {
		t.Fatalf("Error() = %q", apiErr.Error())
	}
}

func TestErrorMappingPlanLimitAndOlderShapes(t *testing.T) {
	client, _, _ := newTestClient(t, "sk_test",
		mockResponse{status: 403, body: `{"code":"plan_limit_exceeded","error":"Active links limit reached","feature":"active_links","limit":25,"current":25,"required_plan":null}`},
		mockResponse{status: 404, body: `{"error":"team not found"}`},
		mockResponse{status: 502, header: http.Header{"Content-Type": {"text/html"}}, body: `<html>Bad gateway</html>`},
	)
	ctx := context.Background()
	_, err := client.Links.Create(ctx, CreateLinkParams{URL: "https://example.com", Title: "x"})
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Code != "plan_limit_exceeded" || apiErr.Raw["feature"] != "active_links" || apiErr.Raw["limit"] != float64(25) {
		t.Fatalf("unexpected plan limit error: %+v", err)
	}
	_, err = client.Teams.Get(ctx, "x", nil)
	if !errors.As(err, &apiErr) || apiErr.StatusCode != 404 || apiErr.Code != "" || apiErr.Message != "team not found" {
		t.Fatalf("unexpected message-only error: %+v", err)
	}
	_, err = client.Links.Update(ctx, "l1", UpdateLinkParams{Title: String("x")})
	if !errors.As(err, &apiErr) || apiErr.StatusCode != 502 || apiErr.Raw != nil || string(apiErr.Body) != "<html>Bad gateway</html>" {
		t.Fatalf("unexpected non-JSON error: %+v", err)
	}
}

func TestDecodeErrorIsReported(t *testing.T) {
	client, _, _ := newTestClient(t, "sk_test", mockResponse{body: `{"data": [`})
	_, err := client.Links.List(context.Background(), nil)
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != 200 || !strings.Contains(apiErr.Message, "decode") {
		t.Fatalf("expected a decode error, got %v", err)
	}
}

func TestRetryRules(t *testing.T) {
	ok := mockResponse{body: `{"success":true,"data":[],"type":"lead"}`}
	tooMany := mockResponse{status: 429, header: http.Header{"Retry-After": {"7"}}, body: `{"code":"api_rate_limit_exceeded","error":"API rate limit exceeded"}`}
	unavailable := mockResponse{status: 503, body: `{"error":"Failed to get analytics"}`}
	serverError := mockResponse{status: 500, body: `{"error":"boom"}`}
	network := mockResponse{err: errors.New("connection reset")}

	cases := []struct {
		name      string
		responses []mockResponse
		call      func(ctx context.Context, c *Client) error
		attempts  int
		wantErr   bool
		delays    []time.Duration // nil: don't check
	}{
		{
			name:      "GET retries 429 honouring Retry-After",
			responses: []mockResponse{tooMany, ok},
			call:      func(ctx context.Context, c *Client) error { _, err := c.Workspaces.List(ctx); return err },
			attempts:  2,
			delays:    []time.Duration{7 * time.Second},
		},
		{
			name:      "PUT retries 503",
			responses: []mockResponse{unavailable, ok},
			call: func(ctx context.Context, c *Client) error {
				_, err := c.Tags.Update(ctx, "t1", UpdateTagParams{Name: "x"})
				return err
			},
			attempts: 2,
		},
		{
			name:      "DELETE retries network errors",
			responses: []mockResponse{network, ok},
			call: func(ctx context.Context, c *Client) error {
				_, err := c.Destinations.Delete(ctx, "w1", "d1")
				return err
			},
			attempts: 2,
		},
		{
			name:      "POST with idempotency_key retries",
			responses: []mockResponse{unavailable, network, ok},
			call: func(ctx context.Context, c *Client) error {
				_, err := c.Conversions.TrackLead(ctx, TrackLeadParams{ClickID: "clk", IdempotencyKey: "k1"})
				return err
			},
			attempts: 3,
		},
		{
			name:      "test-mode POST with idempotency_key is sent once on 503",
			responses: []mockResponse{unavailable, ok},
			call: func(ctx context.Context, c *Client) error {
				_, err := c.Conversions.TrackLead(ctx, TrackLeadParams{ClickID: "clk", IdempotencyKey: "k1", TestMode: true})
				return err
			},
			attempts: 1,
			wantErr:  true,
		},
		{
			name:      "test-mode POST with idempotency_key is not retried on network errors",
			responses: []mockResponse{network, ok},
			call: func(ctx context.Context, c *Client) error {
				_, err := c.Conversions.TrackSale(ctx, TrackSaleParams{ClickID: "clk", Currency: "EUR", AmountCents: Int64(100), IdempotencyKey: "k2", TestMode: true})
				return err
			},
			attempts: 1,
			wantErr:  true,
		},
		{
			name:      "test-mode refund with idempotency_key is sent once on 429",
			responses: []mockResponse{tooMany, ok},
			call: func(ctx context.Context, c *Client) error {
				_, err := c.Conversions.TrackRefund(ctx, TrackRefundParams{InvoiceID: "inv", RefundID: "re", AmountCents: Int64(100), IdempotencyKey: "k3", TestMode: true})
				return err
			},
			attempts: 1,
			wantErr:  true,
		},
		{
			name:      "POST without idempotency_key is sent once",
			responses: []mockResponse{tooMany, ok},
			call: func(ctx context.Context, c *Client) error {
				_, err := c.Links.Create(ctx, CreateLinkParams{URL: "https://example.com", Title: "x"})
				return err
			},
			attempts: 1,
			wantErr:  true,
		},
		{
			name:      "POST without idempotency_key is not retried on network errors",
			responses: []mockResponse{network, ok},
			call: func(ctx context.Context, c *Client) error {
				_, err := c.Conversions.TrackSale(ctx, TrackSaleParams{ClickID: "clk", InvoiceID: "inv"})
				return err
			},
			attempts: 1,
			wantErr:  true,
		},
		{
			name:      "PATCH is not retried",
			responses: []mockResponse{unavailable, ok},
			call: func(ctx context.Context, c *Client) error {
				_, err := c.Reports.UpdateRule(ctx, "r1", ReportRuleParams{Enabled: Bool(false)})
				return err
			},
			attempts: 1,
			wantErr:  true,
		},
		{
			name:      "500 is not retried",
			responses: []mockResponse{serverError, ok},
			call:      func(ctx context.Context, c *Client) error { _, err := c.Domains.List(ctx); return err },
			attempts:  1,
			wantErr:   true,
		},
		{
			name:      "attempts are bounded",
			responses: []mockResponse{unavailable},
			call:      func(ctx context.Context, c *Client) error { _, err := c.Links.Get(ctx, "l1"); return err },
			attempts:  3,
			wantErr:   true,
			delays:    []time.Duration{500 * time.Millisecond, time.Second},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client, transport, delays := newTestClient(t, "sk_test", tc.responses...)
			client.retry.Jitter = false
			err := tc.call(context.Background(), client)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tc.wantErr)
			}
			if len(transport.requests) != tc.attempts {
				t.Fatalf("attempts = %d, want %d", len(transport.requests), tc.attempts)
			}
			if tc.delays != nil && !reflect.DeepEqual(*delays, tc.delays) {
				t.Fatalf("delays = %v, want %v", *delays, tc.delays)
			}
		})
	}
}

func TestRetryAfterAboveCapIsNotWaitedFor(t *testing.T) {
	monthly := mockResponse{
		status: 429,
		header: http.Header{"Retry-After": {"2419200"}},
		body:   `{"code":"api_monthly_limit_exceeded","error":"Monthly API limit reached","details":{"reset_after_sec":2419200}}`,
	}
	client, transport, delays := newTestClient(t, "sk_test", monthly, mockResponse{body: `{"data":[]}`})
	_, err := client.Workspaces.List(context.Background())
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != 429 || apiErr.Code != "api_monthly_limit_exceeded" || apiErr.RetryAfter != "2419200" {
		t.Fatalf("expected the monthly quota error, got %v", err)
	}
	if len(transport.requests) != 1 || len(*delays) != 0 {
		t.Fatalf("requests = %d, delays = %v; want 1 request and no wait", len(transport.requests), *delays)
	}

	// Exactly the cap is still waited for.
	atCap := mockResponse{status: 429, header: http.Header{"Retry-After": {"60"}}, body: `{}`}
	client, transport, delays = newTestClient(t, "sk_test", atCap, mockResponse{body: `{"data":[]}`})
	if _, err := client.Workspaces.List(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(transport.requests) != 2 || !reflect.DeepEqual(*delays, []time.Duration{time.Minute}) {
		t.Fatalf("requests = %d, delays = %v; want 2 and [1m0s]", len(transport.requests), *delays)
	}

	// A lower cap applies too.
	short := mockResponse{status: 503, header: http.Header{"Retry-After": {"7"}}, body: `{}`}
	client, transport, delays = newTestClient(t, "sk_test", short, mockResponse{body: `{"data":[]}`})
	client.retry = normalizeRetry(RetryOptions{MaxRetryAfter: 5 * time.Second})
	if _, err := client.Workspaces.List(context.Background()); err == nil {
		t.Fatal("expected the 503 error")
	}
	if len(transport.requests) != 1 || len(*delays) != 0 {
		t.Fatalf("requests = %d, delays = %v; want 1 request and no wait", len(transport.requests), *delays)
	}
}

func TestRetryDefaults(t *testing.T) {
	if got := DefaultRetryOptions().MaxRetryAfter; got != time.Minute {
		t.Fatalf("default MaxRetryAfter = %v", got)
	}
	if got := normalizeRetry(RetryOptions{MaxRetries: 1}).MaxRetryAfter; got != time.Minute {
		t.Fatalf("unset MaxRetryAfter = %v", got)
	}
}

func TestRetryCanBeDisabledAndConfigured(t *testing.T) {
	unavailable := mockResponse{status: 503, body: `{}`}
	client, transport, _ := newTestClient(t, "sk_test", unavailable)
	client.retry = normalizeRetry(RetryOptions{Disabled: true})
	if _, err := client.Workspaces.List(context.Background()); err == nil {
		t.Fatal("expected an error")
	}
	if len(transport.requests) != 1 {
		t.Fatalf("disabled retries sent %d requests", len(transport.requests))
	}

	transport = &mockTransport{responses: []mockResponse{unavailable}}
	client, err := NewClient("sk_test",
		WithBaseURL("https://api.test"),
		WithHTTPClient(&http.Client{Transport: transport}),
		WithRetry(RetryOptions{MaxRetries: 4, InitialDelay: time.Millisecond, MaxDelay: 3 * time.Millisecond}),
	)
	if err != nil {
		t.Fatal(err)
	}
	var delays []time.Duration
	client.sleep = func(ctx context.Context, d time.Duration) error { delays = append(delays, d); return nil }
	if _, err := client.Workspaces.List(context.Background()); err == nil {
		t.Fatal("expected an error")
	}
	want := []time.Duration{time.Millisecond, 2 * time.Millisecond, 3 * time.Millisecond, 3 * time.Millisecond}
	if len(transport.requests) != 5 || !reflect.DeepEqual(delays, want) {
		t.Fatalf("requests = %d, delays = %v, want 5 and %v", len(transport.requests), delays, want)
	}
}

func TestRetryStopsWhenContextEnds(t *testing.T) {
	transport := &mockTransport{responses: []mockResponse{{status: 429, header: http.Header{"Retry-After": {"30"}}, body: `{}`}}}
	client, err := NewClient("sk_test", WithBaseURL("https://api.test"), WithHTTPClient(&http.Client{Transport: transport}))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err = client.Workspaces.List(ctx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context deadline, got %v", err)
	}
	if time.Since(start) > 5*time.Second {
		t.Fatal("waited for Retry-After despite the context ending")
	}
}

func TestParseRetryAfter(t *testing.T) {
	if d, ok := parseRetryAfter("12"); !ok || d != 12*time.Second {
		t.Fatalf("seconds: %v %v", d, ok)
	}
	if d, ok := parseRetryAfter(time.Now().Add(time.Hour).UTC().Format(http.TimeFormat)); !ok || d < 59*time.Minute {
		t.Fatalf("http date: %v %v", d, ok)
	}
	if _, ok := parseRetryAfter("soon"); ok {
		t.Fatal("garbage should not parse")
	}
	if _, ok := parseRetryAfter(""); ok {
		t.Fatal("empty should not parse")
	}
}

func TestOptionalFieldsAreLeftOut(t *testing.T) {
	client, transport, _ := newTestClient(t, "sk_test")
	ctx := context.Background()
	_, _ = client.Links.Update(ctx, "l1", UpdateLinkParams{})
	_, _ = client.Tags.Update(ctx, "t1", UpdateTagParams{Name: "x"})
	_, _ = client.Reports.UpdateRule(ctx, "r1", ReportRuleParams{})
	_, _ = client.UTMTemplates.Delete(ctx, "u1", nil)
	_, _ = client.Teams.Get(ctx, "s", nil)
	_, _ = client.Conversions.Events(ctx, nil)
	jsonEqual(t, transport.requests[0].body, `{}`)
	jsonEqual(t, transport.requests[1].body, `{"name":"x"}`)
	jsonEqual(t, transport.requests[2].body, `{}`)
	for _, i := range []int{3, 4, 5} {
		if transport.requests[i].query != "" {
			t.Fatalf("request %d sent query %q", i, transport.requests[i].query)
		}
	}
}

func TestFormatTime(t *testing.T) {
	ts := time.Date(2026, 7, 9, 14, 30, 0, 0, time.FixedZone("CEST", 2*3600))
	if got := FormatTime(ts); got != "2026-07-09T12:30:00Z" {
		t.Fatalf("FormatTime = %q", got)
	}
}

func TestIsTestModeBody(t *testing.T) {
	for body, want := range map[string]bool{
		`{"test_mode":true}`:           true,
		`{"click_id":"c"}`:             false,
		`{"test_mode":false}`:          false,
		`{"test_mode":"true"}`:         false,
		`[1,2]`:                        false,
		`{"metadata":{"test_mode":1}}`: false,
	} {
		if got := isTestModeBody([]byte(body)); got != want {
			t.Errorf("isTestModeBody(%s) = %v, want %v", body, got, want)
		}
	}
}

func TestBackoffJitterStaysWithinMaxDelay(t *testing.T) {
	retry := normalizeRetry(RetryOptions{MaxRetries: 10, InitialDelay: time.Second, MaxDelay: 5 * time.Second, Jitter: true})
	for i := 0; i < 1000; i++ {
		for attempt := 0; attempt < 6; attempt++ {
			d := backoff(retry, attempt)
			if d > retry.MaxDelay {
				t.Fatalf("attempt %d waited %v, above MaxDelay %v", attempt, d, retry.MaxDelay)
			}
			if attempt == 0 && (d < 800*time.Millisecond || d > 1200*time.Millisecond) {
				t.Fatalf("first delay %v outside ±20%% of 1s", d)
			}
		}
	}
}

func TestDefaultClientDoesNotFollowRedirects(t *testing.T) {
	redirect := mockResponse{status: http.StatusFound, header: http.Header{"Location": {"https://api.test/elsewhere"}}, body: ``}
	created := mockResponse{status: 201, body: `{"public_id":"l1"}`}

	transport := &mockTransport{responses: []mockResponse{redirect, created}}
	client, err := NewClient("sk_test", WithBaseURL("https://api.test"), WithTimeout(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if client.httpClient.CheckRedirect == nil {
		t.Fatal("WithTimeout dropped the redirect policy")
	}
	client.httpClient.Transport = transport
	_, err = client.Links.Create(context.Background(), CreateLinkParams{URL: "https://example.com", Title: "x"})
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusFound {
		t.Fatalf("expected a 302 *APIError, got %v", err)
	}
	if len(transport.requests) != 1 || transport.requests[0].method != http.MethodPost {
		t.Fatalf("requests = %+v, want the one POST", transport.requests)
	}

	// A caller's own client keeps its own policy (Go's default follows).
	transport = &mockTransport{responses: []mockResponse{redirect, created}}
	client, err = NewClient("sk_test", WithBaseURL("https://api.test"), WithHTTPClient(&http.Client{Transport: transport}))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.Links.Create(context.Background(), CreateLinkParams{URL: "https://example.com", Title: "x"}); err != nil {
		t.Fatalf("custom client: %v", err)
	}
	if len(transport.requests) != 2 || transport.requests[1].path != "/elsewhere" {
		t.Fatalf("custom client requests = %+v, want the redirect followed", transport.requests)
	}
}
