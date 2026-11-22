package shortn

import (
	"context"
	"net/http"
	"net/url"
)

// ConversionsService covers /conversions: sending leads, sales and refunds
// from trusted server code, validating them without recording, and reading
// conversion reports.
//
// A track or validate call whose params carry an IdempotencyKey is retried
// on network errors, 429 and 503; without one it is sent once. A call with
// TestMode set is always sent once, IdempotencyKey or not: the API never
// deduplicates test events, so a retry after a lost answer would record a
// second test event.
type ConversionsService service

// TrackLeadParams is the body of Conversions.TrackLead and ValidateLead.
// ClickID and IdempotencyKey are required.
type TrackLeadParams struct {
	// ClickID is the visitor's click id (snt_click_id).
	ClickID   string `json:"click_id"`
	EventName string `json:"event_name,omitempty"`
	// LeadCategory is signup, form_submission, newsletter_signup,
	// trial_started, add_to_cart, checkout_started or custom.
	LeadCategory   string         `json:"lead_category,omitempty"`
	EventQuantity  int            `json:"event_quantity,omitempty"`
	IdempotencyKey string         `json:"idempotency_key,omitempty"`
	Metadata       map[string]any `json:"metadata,omitempty"`
	// OccurredAt is an RFC 3339 time (see FormatTime); now when empty.
	OccurredAt string `json:"occurred_at,omitempty"`
	// TestMode records a test event, kept out of live reports. Test events
	// are never deduplicated, so the client never retries a test-mode call.
	TestMode bool `json:"test_mode,omitempty"`
	// EventType is optional; it must be "lead" when set.
	EventType string `json:"event_type,omitempty"`
}

// TrackSaleParams is the body of Conversions.TrackSale and ValidateSale.
// ClickID and Currency are required, plus InvoiceID or IdempotencyKey.
// Send the amount as AmountCents (hundredths of the main unit) or Amount (a
// decimal string such as "49.00"), not both.
type TrackSaleParams struct {
	ClickID     string `json:"click_id"`
	EventName   string `json:"event_name,omitempty"`
	AmountCents *int64 `json:"amount_cents,omitempty"`
	Amount      string `json:"amount,omitempty"`
	Currency    string `json:"currency,omitempty"`
	// SettlementAmountCents and SettlementCurrency say what the sale settled
	// for in another currency; send both or neither.
	SettlementAmountCents int64          `json:"settlement_amount_cents,omitempty"`
	SettlementCurrency    string         `json:"settlement_currency,omitempty"`
	InvoiceID             string         `json:"invoice_id,omitempty"`
	PaymentProcessor      string         `json:"payment_processor,omitempty"`
	IdempotencyKey        string         `json:"idempotency_key,omitempty"`
	Metadata              map[string]any `json:"metadata,omitempty"`
	OccurredAt            string         `json:"occurred_at,omitempty"`
	TestMode              bool           `json:"test_mode,omitempty"`
	EventType             string         `json:"event_type,omitempty"`
}

// TrackRefundParams is the body of Conversions.TrackRefund and
// ValidateRefund. InvoiceID (the sale's) and RefundID are required, plus
// AmountCents or Amount. Currency and PaymentProcessor default to the sale's.
type TrackRefundParams struct {
	InvoiceID        string         `json:"invoice_id"`
	RefundID         string         `json:"refund_id"`
	EventName        string         `json:"event_name,omitempty"`
	AmountCents      *int64         `json:"amount_cents,omitempty"`
	Amount           string         `json:"amount,omitempty"`
	Currency         string         `json:"currency,omitempty"`
	PaymentProcessor string         `json:"payment_processor,omitempty"`
	Reason           string         `json:"reason,omitempty"`
	IdempotencyKey   string         `json:"idempotency_key,omitempty"`
	Metadata         map[string]any `json:"metadata,omitempty"`
	OccurredAt       string         `json:"occurred_at,omitempty"`
	TestMode         bool           `json:"test_mode,omitempty"`
	EventType        string         `json:"event_type,omitempty"`
}

// ConversionAttribution names who gets credit for a conversion.
type ConversionAttribution struct {
	// Type is workspace, workspace_member, team, external_partner, campaign
	// or custom.
	Type            string `json:"type"`
	UserPublicID    string `json:"user_public_id,omitempty"`
	TeamPublicID    string `json:"team_public_id,omitempty"`
	PartnerPublicID string `json:"partner_public_id,omitempty"`
	ExternalID      string `json:"external_id,omitempty"`
	Label           string `json:"label,omitempty"`
}

// ReportingExclusion is a currency left out of a reporting total for lack of
// an exchange rate.
type ReportingExclusion struct {
	Currency string `json:"currency"`
	Sales    int64  `json:"sales"`
	Refunds  int64  `json:"refunds"`
}

// ConversionReporting is a total in the reporting currency. Events carry
// AmountCents; summaries, rows and points carry the sale, refund and net
// totals.
type ConversionReporting struct {
	Currency          string               `json:"currency"`
	SaleAmountCents   *int64               `json:"sale_amount_cents"`
	RefundAmountCents *int64               `json:"refund_amount_cents"`
	NetRevenueCents   *int64               `json:"net_revenue_cents"`
	AmountCents       *int64               `json:"amount_cents"`
	Approximate       bool                 `json:"approximate"`
	RateSources       []string             `json:"rate_sources"`
	Excluded          []ReportingExclusion `json:"excluded"`
}

// ConversionEvent is a recorded lead, sale or refund.
type ConversionEvent struct {
	PublicID  string `json:"public_id"`
	EventType string `json:"event_type"`
	EventName string `json:"event_name"`
	// LeadCategory is set on leads; purchase_inferred for a lead a sale
	// recorded.
	LeadCategory string `json:"lead_category"`
	// EventOrigin is observed or inferred.
	EventOrigin string `json:"event_origin"`
	// Duplicate is true when a retry matched an existing event.
	Duplicate   bool   `json:"duplicate"`
	IsTest      bool   `json:"is_test"`
	ClickID     string `json:"click_id"`
	URLPublicID string `json:"url_public_id"`
	ShortCode   string `json:"short_code"`
	ShortURL    string `json:"short_url"`

	AmountCents          *int64 `json:"amount_cents"`
	Amount               string `json:"amount"`
	CurrencyDecimals     int    `json:"currency_decimals"`
	Currency             string `json:"currency"`
	InvoiceID            string `json:"invoice_id"`
	RefundID             string `json:"refund_id"`
	Reason               string `json:"reason"`
	PaymentProcessor     string `json:"payment_processor"`
	RefundedAmountCents  *int64 `json:"refunded_amount_cents"`
	RemainingAmountCents *int64 `json:"remaining_amount_cents"`
	RefundStatus         string `json:"refund_status"`

	Attribution       ConversionAttribution `json:"attribution"`
	Reporting         *ConversionReporting  `json:"reporting"`
	WorkspacePublicID string                `json:"workspace_public_id"`
	WorkspaceName     string                `json:"workspace_name"`
	OccurredAt        string                `json:"occurred_at"`
	CreatedAt         string                `json:"created_at"`

	Raw map[string]any `json:"-"`
}

func (e *ConversionEvent) UnmarshalJSON(b []byte) error {
	type plain ConversionEvent
	return decodeObject(b, (*plain)(e), &e.Raw)
}

// ConversionResult is the answer of TrackLead, TrackSale and TrackRefund.
type ConversionResult struct {
	// Type is lead, sale or refund.
	Type  string          `json:"type"`
	Event ConversionEvent `json:"event"`
	Raw   map[string]any  `json:"-"`
}

func (r *ConversionResult) UnmarshalJSON(b []byte) error {
	type plain ConversionResult
	return decodeObject(b, (*plain)(r), &r.Raw)
}

// ConversionValidationResult is what a validation found.
type ConversionValidationResult struct {
	Valid            bool           `json:"valid"`
	DryRun           bool           `json:"dry_run"`
	EventType        string         `json:"event_type"`
	WouldTrack       bool           `json:"would_track"`
	WouldCreateEvent bool           `json:"would_create_event"`
	WouldInferLead   bool           `json:"would_infer_lead"`
	Duplicate        bool           `json:"duplicate"`
	IsTest           bool           `json:"is_test"`
	Raw              map[string]any `json:"-"`
}

func (r *ConversionValidationResult) UnmarshalJSON(b []byte) error {
	type plain ConversionValidationResult
	return decodeObject(b, (*plain)(r), &r.Raw)
}

// ConversionValidation is the answer of ValidateLead, ValidateSale and
// ValidateRefund. Nothing is recorded.
type ConversionValidation struct {
	Valid  bool                       `json:"valid"`
	DryRun bool                       `json:"dry_run"`
	Result ConversionValidationResult `json:"result"`
	Raw    map[string]any             `json:"-"`
}

func (v *ConversionValidation) UnmarshalJSON(b []byte) error {
	type plain ConversionValidation
	return decodeObject(b, (*plain)(v), &v.Raw)
}

// ConversionSummaryData is the all-time live totals. When sales span several
// currencies, MixedCurrencies is true and the per-currency amounts and
// Currency are nil; Reporting still gives the total.
type ConversionSummaryData struct {
	Leads                   int64                `json:"leads"`
	Sales                   int64                `json:"sales"`
	Refunds                 int64                `json:"refunds"`
	SaleAmountCents         *int64               `json:"sale_amount_cents"`
	RefundAmountCents       *int64               `json:"refund_amount_cents"`
	NetRevenueCents         *int64               `json:"net_revenue_cents"`
	Currency                *string              `json:"currency"`
	MixedCurrencies         bool                 `json:"mixed_currencies"`
	MonetaryTotalsAvailable bool                 `json:"monetary_totals_available"`
	LeadToSaleRate          float64              `json:"lead_to_sale_rate"`
	Reporting               *ConversionReporting `json:"reporting"`
	Raw                     map[string]any       `json:"-"`
}

func (d *ConversionSummaryData) UnmarshalJSON(b []byte) error {
	type plain ConversionSummaryData
	return decodeObject(b, (*plain)(d), &d.Raw)
}

// ConversionSummary is the answer of Conversions.Summary.
type ConversionSummary struct {
	Data ConversionSummaryData `json:"data"`
	Raw  map[string]any        `json:"-"`
}

func (s *ConversionSummary) UnmarshalJSON(b []byte) error {
	type plain ConversionSummary
	return decodeObject(b, (*plain)(s), &s.Raw)
}

// ConversionMetrics are the counts and money fields shared by attribution
// rows, attribution totals and timeline points.
type ConversionMetrics struct {
	Clicks                  int64                `json:"clicks"`
	Leads                   int64                `json:"leads"`
	Sales                   int64                `json:"sales"`
	Refunds                 int64                `json:"refunds"`
	SaleAmountCents         int64                `json:"sale_amount_cents"`
	RefundAmountCents       int64                `json:"refund_amount_cents"`
	NetRevenueCents         int64                `json:"net_revenue_cents"`
	Currency                string               `json:"currency"`
	MixedCurrencies         bool                 `json:"mixed_currencies"`
	MonetaryTotalsAvailable bool                 `json:"monetary_totals_available"`
	ClickToLeadRate         float64              `json:"click_to_lead_rate"`
	ClickToSaleRate         float64              `json:"click_to_sale_rate"`
	LeadToSaleRate          float64              `json:"lead_to_sale_rate"`
	Reporting               *ConversionReporting `json:"reporting"`
}

// AttributionRow is one attribution target (the workspace, a member, a
// team, a partner, a campaign) in an attribution report.
type AttributionRow struct {
	ConversionMetrics
	Attribution   ConversionAttribution `json:"attribution"`
	DisplayName   string                `json:"display_name"`
	LatestEventAt string                `json:"latest_event_at"`
	Raw           map[string]any        `json:"-"`
}

func (r *AttributionRow) UnmarshalJSON(b []byte) error {
	type plain AttributionRow
	return decodeObject(b, (*plain)(r), &r.Raw)
}

// AttributionTotals sums an attribution report.
type AttributionTotals struct {
	ConversionMetrics
	Raw map[string]any `json:"-"`
}

func (t *AttributionTotals) UnmarshalJSON(b []byte) error {
	type plain AttributionTotals
	return decodeObject(b, (*plain)(t), &t.Raw)
}

// AttributionReport is the answer of Conversions.Attribution.
type AttributionReport struct {
	Data        []AttributionRow  `json:"data"`
	Totals      AttributionTotals `json:"totals"`
	Pagination  Pagination        `json:"pagination"`
	DateRange   map[string]any    `json:"date_range"`
	Environment string            `json:"environment"`
	Raw         map[string]any    `json:"-"`
}

func (r *AttributionReport) UnmarshalJSON(b []byte) error {
	type plain AttributionReport
	return decodeObject(b, (*plain)(r), &r.Raw)
}

// TimelinePoint is one interval of a conversion timeline.
type TimelinePoint struct {
	ConversionMetrics
	Start string         `json:"start"`
	End   string         `json:"end"`
	Raw   map[string]any `json:"-"`
}

func (p *TimelinePoint) UnmarshalJSON(b []byte) error {
	type plain TimelinePoint
	return decodeObject(b, (*plain)(p), &p.Raw)
}

// ConversionTimeline is the answer of Conversions.Timeline.
type ConversionTimeline struct {
	Data        []TimelinePoint `json:"data"`
	Interval    string          `json:"interval"`
	DateRange   map[string]any  `json:"date_range"`
	Environment string          `json:"environment"`
	Raw         map[string]any  `json:"-"`
}

func (t *ConversionTimeline) UnmarshalJSON(b []byte) error {
	type plain ConversionTimeline
	return decodeObject(b, (*plain)(t), &t.Raw)
}

// ConversionEventList is the answer of Conversions.Events, newest first.
type ConversionEventList struct {
	Data        []ConversionEvent `json:"data"`
	Pagination  Pagination        `json:"pagination"`
	DateRange   map[string]any    `json:"date_range"`
	Environment string            `json:"environment"`
	Raw         map[string]any    `json:"-"`
}

func (l *ConversionEventList) UnmarshalJSON(b []byte) error {
	type plain ConversionEventList
	return decodeObject(b, (*plain)(l), &l.Raw)
}

// AttributionOptions filters Conversions.Attribution. Zero values are not
// sent. From and To are RFC 3339 times or YYYY-MM-DD dates; without them the
// API reads the last 14 days.
type AttributionOptions struct {
	From string
	To   string
	// Environment is live (default) or test.
	Environment string
	// Limit is 1 to 100 (default 50).
	Limit int
}

// TimelineOptions filters Conversions.Timeline. Zero values are not sent.
type TimelineOptions struct {
	From        string
	To          string
	Environment string
	// Interval is day (default), week or month.
	Interval string
}

// EventsOptions filters Conversions.Events. Zero values are not sent.
type EventsOptions struct {
	From        string
	To          string
	Environment string
	Limit       int
	Offset      int
	// EventType is lead, sale or refund.
	EventType string
	// LeadCategory is a lead category or purchase_inferred.
	LeadCategory string
	// EventOrigin is observed or inferred.
	EventOrigin string
	// Link is a link's public ID, short code, short URL or numeric ID.
	Link string
	// Event is an event's public ID.
	Event string
	// Query searches event names and attribution labels (sent as q).
	Query string
}

func (s *ConversionsService) write(ctx context.Context, path, idempotencyKey string, body, out any) error {
	return s.client.do(ctx, request{method: http.MethodPost, path: path, body: body, idempotencyKey: idempotencyKey}, out)
}

// TrackLead records a lead (POST /conversions/leads).
func (s *ConversionsService) TrackLead(ctx context.Context, params TrackLeadParams) (*ConversionResult, error) {
	var out ConversionResult
	if err := s.write(ctx, "/conversions/leads", params.IdempotencyKey, params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ValidateLead checks a lead without recording it
// (POST /conversions/leads/validations).
func (s *ConversionsService) ValidateLead(ctx context.Context, params TrackLeadParams) (*ConversionValidation, error) {
	var out ConversionValidation
	if err := s.write(ctx, "/conversions/leads/validations", params.IdempotencyKey, params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// TrackSale records a sale (POST /conversions/sales).
func (s *ConversionsService) TrackSale(ctx context.Context, params TrackSaleParams) (*ConversionResult, error) {
	var out ConversionResult
	if err := s.write(ctx, "/conversions/sales", params.IdempotencyKey, params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ValidateSale checks a sale without recording it
// (POST /conversions/sales/validations).
func (s *ConversionsService) ValidateSale(ctx context.Context, params TrackSaleParams) (*ConversionValidation, error) {
	var out ConversionValidation
	if err := s.write(ctx, "/conversions/sales/validations", params.IdempotencyKey, params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// TrackRefund records a refund of a sale (POST /conversions/refunds).
func (s *ConversionsService) TrackRefund(ctx context.Context, params TrackRefundParams) (*ConversionResult, error) {
	var out ConversionResult
	if err := s.write(ctx, "/conversions/refunds", params.IdempotencyKey, params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ValidateRefund checks a refund without recording it
// (POST /conversions/refunds/validations).
func (s *ConversionsService) ValidateRefund(ctx context.Context, params TrackRefundParams) (*ConversionValidation, error) {
	var out ConversionValidation
	if err := s.write(ctx, "/conversions/refunds/validations", params.IdempotencyKey, params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Summary reads all-time live totals (GET /conversions/summary). The
// endpoint takes no query parameters.
func (s *ConversionsService) Summary(ctx context.Context) (*ConversionSummary, error) {
	var out ConversionSummary
	if err := s.client.do(ctx, request{method: http.MethodGet, path: "/conversions/summary"}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Attribution reads conversions per attribution target
// (GET /conversions/attribution). opts may be nil.
func (s *ConversionsService) Attribution(ctx context.Context, opts *AttributionOptions) (*AttributionReport, error) {
	q := url.Values{}
	if opts != nil {
		setString(q, "from", opts.From)
		setString(q, "to", opts.To)
		setString(q, "environment", opts.Environment)
		setInt(q, "limit", opts.Limit)
	}
	var out AttributionReport
	if err := s.client.do(ctx, request{method: http.MethodGet, path: "/conversions/attribution", query: q}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Timeline reads conversions per interval (GET /conversions/timeline). opts
// may be nil.
func (s *ConversionsService) Timeline(ctx context.Context, opts *TimelineOptions) (*ConversionTimeline, error) {
	q := url.Values{}
	if opts != nil {
		setString(q, "from", opts.From)
		setString(q, "to", opts.To)
		setString(q, "environment", opts.Environment)
		setString(q, "interval", opts.Interval)
	}
	var out ConversionTimeline
	if err := s.client.do(ctx, request{method: http.MethodGet, path: "/conversions/timeline", query: q}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Events lists conversion events, newest first (GET /conversions/events).
// opts may be nil.
func (s *ConversionsService) Events(ctx context.Context, opts *EventsOptions) (*ConversionEventList, error) {
	q := url.Values{}
	if opts != nil {
		setString(q, "from", opts.From)
		setString(q, "to", opts.To)
		setString(q, "environment", opts.Environment)
		setInt(q, "limit", opts.Limit)
		setInt(q, "offset", opts.Offset)
		setString(q, "event_type", opts.EventType)
		setString(q, "lead_category", opts.LeadCategory)
		setString(q, "event_origin", opts.EventOrigin)
		setString(q, "link", opts.Link)
		setString(q, "event", opts.Event)
		setString(q, "q", opts.Query)
	}
	var out ConversionEventList
	if err := s.client.do(ctx, request{method: http.MethodGet, path: "/conversions/events", query: q}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
