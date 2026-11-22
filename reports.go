package shortn

import (
	"context"
	"net/http"
	"net/url"
)

// ReportsService covers /reports/rules: scheduled reports and alerts.
type ReportsService service

// ReportRuleDestination is a destination as a rule returns it. Secrets
// (webhook URLs, tokens) come back shortened to scheme and host.
type ReportRuleDestination struct {
	Type string         `json:"type"`
	Data map[string]any `json:"data"`
}

// ReportRule is a scheduled report or an alert. Definition,
// IncludeUnassignedLinks and the watch_* fields are set only by GetRule (and
// CreateRule); read alert-only fields from Raw.
type ReportRule struct {
	PublicID          string   `json:"public_id"`
	Name              string   `json:"name"` // empty from GetRule and CreateRule
	WorkspacePublicID string   `json:"workspace_public_id"`
	WorkspaceName     string   `json:"workspace_name"`
	TeamPublicID      string   `json:"team_public_id"`
	TeamName          string   `json:"team_name"`
	TeamPublicIDs     []string `json:"team_public_ids"`
	TeamNames         []string `json:"team_names"`
	WorkspaceWide     bool     `json:"workspace_wide"`
	// ScheduleType is every_x (a report) or event (an alert).
	ScheduleType            string `json:"schedule_type"`
	ScheduleIntervalSeconds *int64 `json:"schedule_interval_seconds"`
	NextRunAt               string `json:"next_run_at"`
	Enabled                 bool   `json:"enabled"`
	TriggerCount            int64  `json:"trigger_count"`
	ReportName              string `json:"report_name"`
	ReportQueryType         string `json:"report_query_type"`
	CreatedAt               string `json:"created_at"` // empty from GetRule and CreateRule

	Destinations           []ReportRuleDestination `json:"destinations"`
	Definition             map[string]any          `json:"definition"`
	IncludeUnassignedLinks *bool                   `json:"include_unassigned_links"`

	Raw map[string]any `json:"-"`
}

func (r *ReportRule) UnmarshalJSON(b []byte) error {
	type plain ReportRule
	return decodeObject(b, (*plain)(r), &r.Raw)
}

// ReportRuleList is the answer of Reports.ListRules.
type ReportRuleList struct {
	Data []ReportRule   `json:"data"`
	Raw  map[string]any `json:"-"`
}

func (l *ReportRuleList) UnmarshalJSON(b []byte) error {
	type plain ReportRuleList
	return decodeObject(b, (*plain)(l), &l.Raw)
}

// ReportRuleResponse is the answer of Reports.GetRule, CreateRule and
// UpdateRule.
type ReportRuleResponse struct {
	Data ReportRule     `json:"data"`
	Raw  map[string]any `json:"-"`
}

func (r *ReportRuleResponse) UnmarshalJSON(b []byte) error {
	type plain ReportRuleResponse
	return decodeObject(b, (*plain)(r), &r.Raw)
}

// ReportRuleListOptions filters Reports.ListRules. Zero values are not sent.
type ReportRuleListOptions struct {
	WorkspacePublicID string
	// ScheduleTypes is every_x (reports) and/or event (alerts).
	ScheduleTypes []string
	LinkPublicIDs []string
	TeamPublicIDs []string
}

// ReportRuleParams is the body of Reports.CreateRule and Reports.UpdateRule.
// Only set fields are sent; on update, what you leave out is kept.
type ReportRuleParams struct {
	// WorkspacePublicID is required on create unless the key is restricted
	// to a workspace.
	WorkspacePublicID string `json:"workspace_public_id,omitempty"`
	// ReportType is custom_builder_v1 for new rules.
	ReportType string `json:"report_type,omitempty"`
	// ScheduleType is every_x (default) or event.
	ScheduleType    string `json:"schedule_type,omitempty"`
	IntervalSeconds int    `json:"interval_seconds,omitempty"`
	// InitialRunAt is an RFC 3339 time.
	InitialRunAt string `json:"initial_run_at,omitempty"`
	ReportName   string `json:"report_name,omitempty"`
	// DefinitionJSON is the rule's sections or conditions: a map, a struct
	// or a json.RawMessage. Read one with GetRule to get a valid shape.
	DefinitionJSON         any       `json:"definition_json,omitempty"`
	DestinationPublicIDs   *[]string `json:"destination_public_ids,omitempty"`
	TeamPublicIDs          *[]string `json:"team_public_ids,omitempty"`
	IncludeUnassignedLinks *bool     `json:"include_unassigned_links,omitempty"`
	LinkPublicIDs          *[]string `json:"link_public_ids,omitempty"`
	// Enabled pauses (false) or resumes (true) the rule.
	Enabled *bool `json:"enabled,omitempty"`
}

// ListRules lists the report and alert rules you can see
// (GET /reports/rules). opts may be nil.
func (s *ReportsService) ListRules(ctx context.Context, opts *ReportRuleListOptions) (*ReportRuleList, error) {
	q := url.Values{}
	if opts != nil {
		setString(q, "workspace_public_id", opts.WorkspacePublicID)
		setList(q, "schedule_types", opts.ScheduleTypes)
		setList(q, "link_public_ids", opts.LinkPublicIDs)
		setList(q, "team_public_ids", opts.TeamPublicIDs)
	}
	var out ReportRuleList
	if err := s.client.do(ctx, request{method: http.MethodGet, path: "/reports/rules", query: q}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetRule reads a rule by public_id (GET /reports/rules/{public_id}).
func (s *ReportsService) GetRule(ctx context.Context, publicID string) (*ReportRuleResponse, error) {
	var out ReportRuleResponse
	if err := s.client.do(ctx, request{method: http.MethodGet, path: pathEscape("reports", "rules", publicID)}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CreateRule creates a report or alert rule (POST /reports/rules).
func (s *ReportsService) CreateRule(ctx context.Context, params ReportRuleParams) (*ReportRuleResponse, error) {
	var out ReportRuleResponse
	if err := s.client.do(ctx, request{method: http.MethodPost, path: "/reports/rules", body: params}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateRule changes the fields set in params
// (PATCH /reports/rules/{public_id}).
func (s *ReportsService) UpdateRule(ctx context.Context, publicID string, params ReportRuleParams) (*ReportRuleResponse, error) {
	var out ReportRuleResponse
	if err := s.client.do(ctx, request{method: http.MethodPatch, path: pathEscape("reports", "rules", publicID), body: params}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteRule deletes a rule (DELETE /reports/rules/{public_id}); the API
// answers 204 with no body.
func (s *ReportsService) DeleteRule(ctx context.Context, publicID string) error {
	return s.client.do(ctx, request{method: http.MethodDelete, path: pathEscape("reports", "rules", publicID)}, nil)
}
