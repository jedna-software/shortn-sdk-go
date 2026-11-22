package shortn

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
)

// PublicService covers the endpoints that need no API key: URL validation,
// QR codes and the plan catalog. A client without a key (NewPublicClient)
// can call them; with a key, the key is sent too.
//
// The key-less link creation (POST /links without a key) and anonymous link
// preview (GET /links/{code}/analytics) need a human check passed on
// shortn.link, so servers can't use them and this SDK leaves them out: with
// an API key, use Links.Create and Analytics.Link.
type PublicService service

// URLValidation is the answer of Public.ValidateURL. An unusable URL still
// answers 200, with Valid false and Code and Error saying why.
type URLValidation struct {
	Valid      bool           `json:"valid"`
	Normalized string         `json:"normalized"`
	Code       string         `json:"code"`
	Error      string         `json:"error"`
	Raw        map[string]any `json:"-"`
}

func (v *URLValidation) UnmarshalJSON(b []byte) error {
	type plain URLValidation
	return decodeObject(b, (*plain)(v), &v.Raw)
}

// Plan is one entry of the plan catalog. This endpoint's fields are
// capitalised, unlike the rest of the API. Price is in cents; MaxLinks and
// MaxClicks are -1 when unlimited.
type Plan struct {
	ID          string         `json:"ID"`
	Name        string         `json:"Name"`
	Description string         `json:"Description"`
	Price       int64          `json:"Price"`
	Currency    string         `json:"Currency"`
	Interval    string         `json:"Interval"`
	Features    any            `json:"Features"`
	PriceID     string         `json:"PriceID"`
	MaxLinks    int64          `json:"MaxLinks"`
	MaxClicks   int64          `json:"MaxClicks"`
	Raw         map[string]any `json:"-"`
}

func (p *Plan) UnmarshalJSON(b []byte) error {
	type plain Plan
	return decodeObject(b, (*plain)(p), &p.Raw)
}

// PlanCatalog is the answer of Public.Plans, keyed by plan ID (free,
// starter, professional, agency and their _yearly variants).
type PlanCatalog struct {
	Plans map[string]Plan `json:"plans"`
	Raw   map[string]any  `json:"-"`
}

func (c *PlanCatalog) UnmarshalJSON(b []byte) error {
	type plain PlanCatalog
	return decodeObject(b, (*plain)(c), &c.Raw)
}

// QRCodeParams is the query of Public.QRCode. URL is required; the rest is
// sent only when set. Without an account, only the options every plan has
// work (flat colors, the simple border, rectangle or circle modules, the
// scan frame); others answer 403 plan_limit_exceeded.
type QRCodeParams struct {
	URL string
	// Format is png (default), svg or jpg.
	Format string
	// Size is preview (default), preview-hd, preview-final, preview-xl or
	// download.
	Size string
	// ColorMode is flat (default) or gradient.
	ColorMode string
	// Foreground and Background are hex colors (fg, bg); Background may be
	// "transparent".
	Foreground     string
	Background     string
	GradientColors string
	GradientDir    string
	QRShape        string
	FinderShape    string
	CornerStyle    string
	BorderPattern  string
	BorderColor    string
	FrameTemplate  string
	FrameText      string
	FrameTextColor string
	// FrameTextFollowsBackground, when set, is sent as true or false.
	FrameTextFollowsBackground *bool
	// Branding is default or none.
	Branding string
	// Extra holds any other query parameter the endpoint reads.
	Extra url.Values
}

// ValidateURL checks whether rawURL can be a link's destination
// (GET /validate).
func (s *PublicService) ValidateURL(ctx context.Context, rawURL string) (*URLValidation, error) {
	var out URLValidation
	q := url.Values{"url": {rawURL}}
	if err := s.client.do(ctx, request{method: http.MethodGet, path: "/validate", query: q, public: true}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// QRCode draws a QR code (GET /qr) and returns the image bytes.
func (s *PublicService) QRCode(ctx context.Context, params QRCodeParams) ([]byte, error) {
	q := url.Values{}
	for key, values := range params.Extra {
		q[key] = append([]string(nil), values...)
	}
	q.Set("url", params.URL)
	setString(q, "format", params.Format)
	setString(q, "size", params.Size)
	setString(q, "colorMode", params.ColorMode)
	setString(q, "fg", params.Foreground)
	setString(q, "bg", params.Background)
	setString(q, "gradientColors", params.GradientColors)
	setString(q, "gradientDir", params.GradientDir)
	setString(q, "qrShape", params.QRShape)
	setString(q, "finderShape", params.FinderShape)
	setString(q, "cornerStyle", params.CornerStyle)
	setString(q, "borderPattern", params.BorderPattern)
	setString(q, "borderColor", params.BorderColor)
	setString(q, "frameTemplate", params.FrameTemplate)
	setString(q, "frameText", params.FrameText)
	setString(q, "frameTextColor", params.FrameTextColor)
	if params.FrameTextFollowsBackground != nil {
		q.Set("frameTextFollowsBackground", strconv.FormatBool(*params.FrameTextFollowsBackground))
	}
	setString(q, "branding", params.Branding)
	raw, err := s.client.send(ctx, request{method: http.MethodGet, path: "/qr", query: q, public: true, accept: "image/*"})
	if err != nil {
		return nil, err
	}
	return raw.body, nil
}

// Plans reads the plan catalog (GET /plans).
func (s *PublicService) Plans(ctx context.Context) (*PlanCatalog, error) {
	var out PlanCatalog
	if err := s.client.do(ctx, request{method: http.MethodGet, path: "/plans", public: true}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
