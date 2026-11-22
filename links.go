package shortn

import (
	"context"
	"net/http"
	"net/url"
)

// LinksService covers /links. Links are addressed by their public_id, which
// Create and List return.
type LinksService service

// Pagination is the paging block of list responses.
type Pagination struct {
	Count      int  `json:"count"`
	Limit      int  `json:"limit"`
	Offset     int  `json:"offset"`
	HasMore    bool `json:"has_more"`
	NextOffset *int `json:"next_offset"`
}

// MessageResult is an answer that carries only a message, such as
// {"message": "URL disabled successfully"}.
type MessageResult struct {
	Message string         `json:"message"`
	Raw     map[string]any `json:"-"`
}

func (r *MessageResult) UnmarshalJSON(b []byte) error {
	type plain MessageResult
	return decodeObject(b, (*plain)(r), &r.Raw)
}

// SuccessResult is an answer such as {"success": true}.
type SuccessResult struct {
	Success bool           `json:"success"`
	Raw     map[string]any `json:"-"`
}

func (r *SuccessResult) UnmarshalJSON(b []byte) error {
	type plain SuccessResult
	return decodeObject(b, (*plain)(r), &r.Raw)
}

// TeamAssignment is one team a link or destination is limited to.
type TeamAssignment struct {
	TeamID       int64  `json:"team_id"`
	TeamPublicID string `json:"team_public_id"`
	TeamName     string `json:"team_name"`
}

// LinkTag is a tag on a link.
type LinkTag struct {
	PublicID string `json:"public_id"`
	Name     string `json:"name"`
	Color    string `json:"color"`
}

// Link is a short link as List and Get return it. Times are RFC 3339 strings.
type Link struct {
	PublicID          string         `json:"public_id"`
	OriginalURL       string         `json:"original_url"`
	ShortCode         string         `json:"short_code"`
	Domain            string         `json:"domain"`
	Title             string         `json:"title"`
	Description       string         `json:"description"`
	Clicks            int64          `json:"clicks"`
	Active            bool           `json:"active"`
	Status            string         `json:"status"`
	SecurityStatus    string         `json:"security_status"`
	WorkspacePublicID string         `json:"workspace_public_id"`
	WorkspaceName     string         `json:"workspace_name"` // only Links.List sets it
	Tags              []LinkTag      `json:"tags"`
	StartsAt          string         `json:"starts_at"`
	ExpiresAt         string         `json:"expires_at"`
	CreatedAt         string         `json:"created_at"`
	UpdatedAt         string         `json:"updated_at"`
	Raw               map[string]any `json:"-"`
}

func (l *Link) UnmarshalJSON(b []byte) error {
	type plain Link
	return decodeObject(b, (*plain)(l), &l.Raw)
}

// LinkList is the answer of Links.List.
type LinkList struct {
	Data       []Link         `json:"data"`
	Pagination Pagination     `json:"pagination"`
	Raw        map[string]any `json:"-"`
}

func (l *LinkList) UnmarshalJSON(b []byte) error {
	type plain LinkList
	return decodeObject(b, (*plain)(l), &l.Raw)
}

// LinkResponse is the answer of Links.Get.
type LinkResponse struct {
	Data Link           `json:"data"`
	Raw  map[string]any `json:"-"`
}

func (l *LinkResponse) UnmarshalJSON(b []byte) error {
	type plain LinkResponse
	return decodeObject(b, (*plain)(l), &l.Raw)
}

// LinkListOptions filters Links.List. Zero values are not sent.
type LinkListOptions struct {
	// Limit is 1 to 100; the API defaults to 50.
	Limit int
	// Offset is how many links to skip.
	Offset int
}

// LinkAttribution says who gets credit for a link's conversions.
type LinkAttribution struct {
	// Type is workspace, workspace_member, team, external_partner, campaign
	// or custom.
	Type            string `json:"type,omitempty"`
	UserPublicID    string `json:"user_public_id,omitempty"`
	TeamPublicID    string `json:"team_public_id,omitempty"`
	PartnerPublicID string `json:"partner_public_id,omitempty"`
	ExternalID      string `json:"external_id,omitempty"`
	Label           string `json:"label,omitempty"`
}

// ABVariant is one destination of an A/B test.
type ABVariant struct {
	URL          string `json:"url"`
	SplitPercent *int   `json:"split_percent,omitempty"`
}

// GeoOverride sends visitors from a country ("country:de") or region
// ("region:europe") to another URL.
type GeoOverride struct {
	Target string `json:"target"`
	URL    string `json:"url"`
}

// CreateLinkParams is the body of Links.Create. URL and Title are required;
// the rest is optional and left out when zero. Many options need a paid plan
// (the API answers 403 plan_limit_exceeded).
type CreateLinkParams struct {
	URL         string `json:"url"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	CustomCode  string `json:"custom_code,omitempty"`
	ShortDomain string `json:"short_domain,omitempty"`

	WorkspacePublicID string   `json:"workspace_public_id,omitempty"`
	TeamPublicIDs     []string `json:"team_public_ids,omitempty"`
	// TeamPublicID is the older single-team form of TeamPublicIDs.
	TeamPublicID  string   `json:"team_public_id,omitempty"`
	TagPublicIDs  []string `json:"tag_public_ids,omitempty"`
	UTMTemplateID string   `json:"utm_template_id,omitempty"`
	UTMSource     string   `json:"utm_source,omitempty"`
	UTMMedium     string   `json:"utm_medium,omitempty"`
	UTMCampaign   string   `json:"utm_campaign,omitempty"`
	UTMTerm       string   `json:"utm_term,omitempty"`
	UTMContent    string   `json:"utm_content,omitempty"`
	UTMID         string   `json:"utm_id,omitempty"`

	// StartsAt and ExpiresAt are RFC 3339 times (see FormatTime).
	StartsAt  string `json:"starts_at,omitempty"`
	ExpiresAt string `json:"expires_at,omitempty"`
	// ExpirationAction is suspended or archive; required with ExpiresAt.
	ExpirationAction string `json:"expiration_action,omitempty"`

	// AccessMode is password or jit.
	AccessMode          string   `json:"access_mode,omitempty"`
	Password            string   `json:"password,omitempty"`
	AccessDurationHours int      `json:"access_duration_hours,omitempty"`
	JITAllowlist        []string `json:"jit_allowlist,omitempty"`

	TargetingGeoOverrides []GeoOverride     `json:"targeting_geo_overrides,omitempty"`
	LanguageTargets       map[string]string `json:"language_targets,omitempty"`
	DeviceTargets         map[string]string `json:"device_targets,omitempty"`
	OSTargets             map[string]string `json:"os_targets,omitempty"`
	BrowserTargets        map[string]string `json:"browser_targets,omitempty"`
	MobileTargetIOS       string            `json:"mobile_target_ios,omitempty"`
	MobileTargetAndroid   string            `json:"mobile_target_android,omitempty"`
	TargetingPriority     []string          `json:"targeting_priority,omitempty"`

	ABVariants []ABVariant `json:"ab_variants,omitempty"`
	// ABDestination and ABSplitPercent are the older two-way A/B form.
	ABDestination  string `json:"ab_destination,omitempty"`
	ABSplitPercent *int   `json:"ab_split_percent,omitempty"`

	ConversionEnabled               *bool            `json:"conversion_enabled,omitempty"`
	ConversionAttributionModel      string           `json:"conversion_attribution_model,omitempty"`
	ConversionAttributionWindowDays *int             `json:"conversion_attribution_window_days,omitempty"`
	Attribution                     *LinkAttribution `json:"attribution,omitempty"`

	PreviewTitle       string `json:"preview_title,omitempty"`
	PreviewDescription string `json:"preview_description,omitempty"`
	PreviewImage       string `json:"preview_image,omitempty"`
	PreviewCustom      bool   `json:"preview_custom,omitempty"`
	CloakingEnabled    *bool  `json:"cloaking_enabled,omitempty"`
}

// CreatedLink is the answer of Links.Create.
type CreatedLink struct {
	PublicID            string         `json:"public_id"`
	ShortURL            string         `json:"short_url"`
	ShortCode           string         `json:"short_code"`
	OriginalURL         string         `json:"original_url"`
	Title               string         `json:"title"`
	LinkStatus          string         `json:"link_status"`
	SecurityVerdict     string         `json:"security_verdict"`
	SecurityRiskScore   int            `json:"security_risk_score"`
	SecurityReasonCodes []string       `json:"security_reason_codes"`
	ExpiresAt           string         `json:"expires_at"`
	Raw                 map[string]any `json:"-"`
}

func (l *CreatedLink) UnmarshalJSON(b []byte) error {
	type plain CreatedLink
	return decodeObject(b, (*plain)(l), &l.Raw)
}

// UpdateLinkParams is the body of Links.Update. Only the fields that are set
// are sent; other link settings can't be changed through the API.
type UpdateLinkParams struct {
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
	URL         *string `json:"url,omitempty"`
	ShortCode   *string `json:"short_code,omitempty"`
	Domain      *string `json:"domain,omitempty"`
	// Status is active, suspended, archived or scheduled.
	Status *string `json:"status,omitempty"`
	Active *bool   `json:"active,omitempty"`
	// TeamPublicIDs replaces the link's teams; Strings() makes it
	// workspace-wide. Send it in an update of its own.
	TeamPublicIDs *[]string `json:"team_public_ids,omitempty"`
	// TeamPublicID is the older single-team form.
	TeamPublicID *string `json:"team_public_id,omitempty"`
}

// LinkUpdateResult is the answer of Links.Update. The team fields are set
// only when the update changed the link's teams.
type LinkUpdateResult struct {
	Message       string           `json:"message"`
	WorkspaceWide *bool            `json:"workspace_wide"`
	TeamPublicIDs []string         `json:"team_public_ids"`
	TeamNames     []string         `json:"team_names"`
	Teams         []TeamAssignment `json:"teams"`
	Raw           map[string]any   `json:"-"`
}

func (r *LinkUpdateResult) UnmarshalJSON(b []byte) error {
	type plain LinkUpdateResult
	return decodeObject(b, (*plain)(r), &r.Raw)
}

// ABWinnerParams is the body of Links.SetABWinner.
type ABWinnerParams struct {
	// Variant is the winner's letter, such as "B".
	Variant string `json:"variant"`
}

// ABWinnerResult is the answer of Links.SetABWinner.
type ABWinnerResult struct {
	Winner      string         `json:"winner"`
	Test        int            `json:"test"`
	Destination string         `json:"destination"`
	Raw         map[string]any `json:"-"`
}

func (r *ABWinnerResult) UnmarshalJSON(b []byte) error {
	type plain ABWinnerResult
	return decodeObject(b, (*plain)(r), &r.Raw)
}

// List lists links, newest first (GET /links). opts may be nil.
func (s *LinksService) List(ctx context.Context, opts *LinkListOptions) (*LinkList, error) {
	q := url.Values{}
	if opts != nil {
		setInt(q, "limit", opts.Limit)
		setInt(q, "offset", opts.Offset)
	}
	var out LinkList
	if err := s.client.do(ctx, request{method: http.MethodGet, path: "/links", query: q}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Get reads one link by public_id (GET /links/{public_id}).
func (s *LinksService) Get(ctx context.Context, publicID string) (*LinkResponse, error) {
	var out LinkResponse
	if err := s.client.do(ctx, request{method: http.MethodGet, path: pathEscape("links", publicID)}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Create creates a link with the API key (POST /links).
func (s *LinksService) Create(ctx context.Context, params CreateLinkParams) (*CreatedLink, error) {
	var out CreatedLink
	if err := s.client.do(ctx, request{method: http.MethodPost, path: "/links", body: params}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Update changes the fields set in params (PATCH /links/{public_id}).
func (s *LinksService) Update(ctx context.Context, publicID string, params UpdateLinkParams) (*LinkUpdateResult, error) {
	var out LinkUpdateResult
	if err := s.client.do(ctx, request{method: http.MethodPatch, path: pathEscape("links", publicID), body: params}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Delete suspends a link (DELETE /links/{public_id}): it stops redirecting
// and keeps its analytics. Enable makes it active again.
func (s *LinksService) Delete(ctx context.Context, publicID string) (*MessageResult, error) {
	var out MessageResult
	if err := s.client.do(ctx, request{method: http.MethodDelete, path: pathEscape("links", publicID)}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Enable re-enables a suspended link (POST /links/{public_id}/enable).
func (s *LinksService) Enable(ctx context.Context, publicID string) (*MessageResult, error) {
	var out MessageResult
	if err := s.client.do(ctx, request{method: http.MethodPost, path: pathEscape("links", publicID, "enable")}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// SetABWinner ends the link's A/B test with a winner
// (POST /links/{public_id}/ab/winner).
func (s *LinksService) SetABWinner(ctx context.Context, publicID string, params ABWinnerParams) (*ABWinnerResult, error) {
	var out ABWinnerResult
	if err := s.client.do(ctx, request{method: http.MethodPost, path: pathEscape("links", publicID, "ab", "winner"), body: params}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
