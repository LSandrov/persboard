package eazybi

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"persboard/backend/internal/domain"
)

type Config struct {
	BaseURL      string // e.g. https://jira.example.com
	ExportPrefix string // e.g. /plugins/servlet/eazybi
	AccountID    string
	AuthMode     string // basic, embed, jira_token, token, bearer or none
	Username     string
	Password     string
	EmbedToken   string
	JiraToken    string
	HTTPTimeout  time.Duration
	AllowedHosts []string
	TimeRegex    *regexp.Regexp
}

type Client struct {
	cfg        Config
	httpClient *http.Client
}

func NewClient(cfg Config) *Client {
	timeout := cfg.HTTPTimeout
	if timeout == 0 {
		timeout = 15 * time.Second
	}

	c := &Client{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
	return c
}

// ExportCSV downloads report results in CSV format, optionally using selected_pages.
// It is intentionally strict about URL origin (SSRF protection).
func (c *Client) ExportCSV(ctx context.Context, reportID int, selectedPages []string) (string, error) {
	exportFormat := "csv"
	exportURL, err := c.buildExportURL(reportID, exportFormat, selectedPages)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, exportURL, nil)
	if err != nil {
		return "", err
	}

	switch strings.ToLower(c.cfg.AuthMode) {
	case "basic":
		req.SetBasicAuth(c.cfg.Username, c.cfg.Password)
	case "embed":
		if c.cfg.EmbedToken == "" {
			return "", errors.New("embed token auth requested but EAZYBI_EMBED_TOKEN is empty")
		}
	case "jira_token", "token", "bearer":
		if c.cfg.JiraToken == "" {
			return "", errors.New("jira token auth requested but EAZYBI_JIRA_TOKEN is empty")
		}
		req.Header.Set("Authorization", "Bearer "+c.cfg.JiraToken)
	default:
		// no auth
	}

	res, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return "", fmt.Errorf("eazybi export failed: status %d", res.StatusCode)
	}

	// Cap response size to avoid memory explosions on unexpected reports.
	body, err := io.ReadAll(io.LimitReader(res.Body, 2<<20)) // 2 MiB
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func (c *Client) buildExportURL(reportID int, format string, selectedPages []string) (string, error) {
	base, err := url.Parse(c.cfg.BaseURL)
	if err != nil {
		return "", fmt.Errorf("invalid EAZYBI_BASE_URL: %w", err)
	}
	if base.Scheme != "http" && base.Scheme != "https" {
		return "", fmt.Errorf("invalid scheme in EAZYBI_BASE_URL")
	}
	if base.Host == "" {
		return "", fmt.Errorf("invalid EAZYBI_BASE_URL host")
	}
	if c.cfg.ExportPrefix == "" {
		return "", fmt.Errorf("missing EAZYBI_EXPORT_PREFIX")
	}

	allowed := false
	for _, h := range c.cfg.AllowedHosts {
		if strings.EqualFold(h, base.Hostname()) {
			allowed = true
			break
		}
	}
	if !allowed {
		return "", fmt.Errorf("eazybi host not allowed by config")
	}

	if err := validateResolvedIPNotPrivate(base.Hostname()); err != nil {
		return "", err
	}

	exportPrefix := strings.TrimSuffix(c.cfg.ExportPrefix, "/")
	exportPath := fmt.Sprintf("%s/accounts/%s/export/report/%d.%s", exportPrefix, c.cfg.AccountID, reportID, format)

	u := base.ResolveReference(&url.URL{Path: exportPath})

	if len(selectedPages) > 0 {
		memberList := strings.Join(selectedPages, ",")
		q := u.Query()
		q.Set("selected_pages", memberList)
		u.RawQuery = q.Encode()
	}

	// embed token can be passed as query param (docs mention it exists as request parameter)
	if strings.ToLower(c.cfg.AuthMode) == "embed" && c.cfg.EmbedToken != "" {
		q := u.Query()
		q.Set("embed_token", c.cfg.EmbedToken)
		u.RawQuery = q.Encode()
	}

	return u.String(), nil
}

func validateResolvedIPNotPrivate(host string) error {
	ips, err := net.DefaultResolver.LookupIPAddr(context.Background(), host)
	if err != nil {
		return fmt.Errorf("failed to resolve eazybi host: %w", err)
	}
	for _, ipAddr := range ips {
		parsedIP, parseErr := netip.ParseAddr(ipAddr.IP.String())
		if parseErr != nil {
			continue
		}
		if parsedIP.IsPrivate() || parsedIP.IsLoopback() || parsedIP.IsLinkLocalUnicast() || parsedIP.IsMulticast() {
			return fmt.Errorf("eazybi resolves to a private/unreachable IP; blocked by SSRF policy")
		}
	}

	return nil
}

// ParseDate extracts a YYYY-MM-DD date from a string.
func ParseDate(s string) (string, bool) {
	re := regexp.MustCompile(`\d{4}-\d{2}-\d{2}`)
	m := re.FindString(s)
	if m == "" {
		return "", false
	}
	return m, true
}

func ParseFloatLoose(s string) (float64, bool) {
	// Remove thousands separators and non-numeric clutter.
	clean := strings.TrimSpace(s)
	clean = strings.ReplaceAll(clean, ",", ".")
	clean = strings.ReplaceAll(clean, " ", "")
	// Allow +/- and decimals.
	n, err := strconv.ParseFloat(clean, 64)
	if err != nil {
		return 0, false
	}
	return n, true
}

// ParseCSVMetric expects a CSV with headers.
// If expectedBool is true, it will try to parse values as boolean.
func ParseCSVMetric(csvText string, timeColumnHint string, valueColumnHint string, expectedBool bool) (map[string]*domain.CalendarMetricCellValue, error) {
	reader := csv.NewReader(strings.NewReader(csvText))
	reader.FieldsPerRecord = -1

	headers, err := reader.Read()
	if err != nil {
		return nil, err
	}

	timeIdx := findColumnIndex(headers, timeColumnHint, []string{"time", "date"})
	if timeIdx < 0 {
		timeIdx = 0
	}

	valIdx := findColumnIndex(headers, valueColumnHint, []string{"value", "measure", "metric"})
	if valIdx < 0 {
		valIdx = len(headers) - 1
	}

	out := make(map[string]*domain.CalendarMetricCellValue)
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if timeIdx >= len(row) || valIdx >= len(row) {
			continue
		}

		dateStr, ok := ParseDate(row[timeIdx])
		if !ok {
			continue
		}

		// Prefer numeric parsing for non-bool metrics.
		if !expectedBool {
			v, ok := ParseFloatLoose(row[valIdx])
			if !ok {
				out[dateStr] = nil
				continue
			}
			out[dateStr] = &domain.CalendarMetricCellValue{Number: &v}
			continue
		}

		if b, ok := ParseBoolLoose(row[valIdx]); ok {
			out[dateStr] = &domain.CalendarMetricCellValue{Bool: &b}
		} else if v, ok := ParseFloatLoose(row[valIdx]); ok {
			// Some exports encode booleans as 0/1
			boolV := v != 0
			out[dateStr] = &domain.CalendarMetricCellValue{Bool: &boolV}
		} else {
			out[dateStr] = nil
		}
	}

	return out, nil
}

func findColumnIndex(headers []string, hint string, keywords []string) int {
	if hint != "" {
		for i, h := range headers {
			if strings.EqualFold(strings.TrimSpace(h), hint) {
				return i
			}
		}
	}

	for i, h := range headers {
		lh := strings.ToLower(h)
		for _, kw := range keywords {
			if strings.Contains(lh, kw) {
				return i
			}
		}
	}

	return -1
}

func ParseBoolLoose(s string) (bool, bool) {
	clean := strings.TrimSpace(strings.ToLower(s))
	if clean == "" {
		return false, false
	}

	switch clean {
	case "true", "t", "yes", "y", "1":
		return true, true
	case "false", "f", "no", "n", "0":
		return false, true
	default:
		return false, false
	}
}
