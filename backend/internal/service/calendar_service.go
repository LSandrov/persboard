package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"persboard/backend/internal/domain"
	"persboard/backend/internal/integrations/eazybi"
)

type CalendarService struct {
	repo domain.Repository
	eazy *eazybi.Client

	metrics []domain.CalendarMetricDefinition
}

func NewCalendarService(repo domain.Repository, eazybiClient *eazybi.Client, metrics []domain.CalendarMetricDefinition) *CalendarService {
	return &CalendarService{
		repo:    repo,
		eazy:    eazybiClient,
		metrics: metrics,
	}
}

func (s *CalendarService) EnsureWeights(ctx context.Context) error {
	if len(s.metrics) == 0 {
		return nil
	}
	return s.repo.UpsertMetricWeights(ctx, s.metrics)
}

func (s *CalendarService) CalendarMetrics(ctx context.Context, from, to time.Time) (domain.CalendarMetricsResponse, error) {
	if err := s.EnsureWeights(ctx); err != nil {
		return domain.CalendarMetricsResponse{}, err
	}

	if from.After(to) {
		return domain.CalendarMetricsResponse{}, ValidationError{Message: "from must be <= to"}
	}

	days := make([]string, 0)
	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		days = append(days, domain.DateToYMD(d))
	}

	weightsMap, err := s.repo.GetMetricWeights(ctx, metricKeys(s.metrics))
	if err != nil {
		return domain.CalendarMetricsResponse{}, err
	}

	metricsOut := make([]domain.CalendarMetric, 0, len(s.metrics))

	for _, def := range s.metrics {
		weight := def.DefaultWeight
		if w, ok := weightsMap[def.Key]; ok {
			weight = w.Weight
		}

		values := make(map[string]*domain.CalendarMetricCellValue, len(days))
		for _, day := range days {
			values[day] = nil
		}

		if s.eazy == nil || def.EazyBIReportID <= 0 {
			fillMockMetricValues(def, days, values)
		} else {
			// request eazybi data once per metric for the whole date range
			selectedPages, err := s.buildSelectedPages(def, days)
			if err != nil {
				slog.WarnContext(ctx, "eazybi: falling back to mock metric values", "metric_key", def.Key, "reason", "build_pages", "err", err)
				fillMockMetricValues(def, days, values)
				continue
			}

			csvText, err := s.eazy.ExportCSV(ctx, def.EazyBIReportID, selectedPages)
			if err != nil {
				slog.WarnContext(ctx, "eazybi: export failed, using mock values", "metric_key", def.Key, "report_id", def.EazyBIReportID, "err", err)
				fillMockMetricValues(def, days, values)
				continue
			}

			parsed, err := eazybi.ParseCSVMetric(csvText, "", "", def.TargetValue.Bool != nil)
			if err != nil {
				slog.WarnContext(ctx, "eazybi: csv parse failed, using mock values", "metric_key", def.Key, "err", err)
				fillMockMetricValues(def, days, values)
				continue
			}

			for _, day := range days {
				if v, ok := parsed[day]; ok {
					values[day] = v
				}
			}
		}

		metricsOut = append(metricsOut, domain.CalendarMetric{
			Key:            def.Key,
			Title:          def.Title,
			Weight:         weight,
			MetricType:     def.MetricType,
			TargetValue:    def.TargetValue,
			TargetOperator: def.TargetOperator,
			ValuesByDate:   values,
		})
	}

	return domain.CalendarMetricsResponse{
		From:    domain.DateToYMD(from),
		To:      domain.DateToYMD(to),
		Days:    days,
		Metrics: metricsOut,
	}, nil
}

func (s *CalendarService) UpdateMetricWeight(ctx context.Context, input domain.UpdateMetricWeightInput) (domain.MetricWeight, error) {
	var def *domain.CalendarMetricDefinition
	for i := range s.metrics {
		if s.metrics[i].Key == input.MetricKey {
			def = &s.metrics[i]
			break
		}
	}
	if def == nil {
		return domain.MetricWeight{}, ValidationError{Message: "unknown metricKey"}
	}

	if input.Weight < 0 || input.Weight > 1000 || math.IsNaN(input.Weight) || math.IsInf(input.Weight, 0) {
		return domain.MetricWeight{}, ValidationError{Message: "weight must be between 0 and 1000"}
	}

	if err := s.repo.SetMetricWeight(ctx, input, def.Title); err != nil {
		return domain.MetricWeight{}, err
	}

	return domain.MetricWeight{
		Key:    def.Key,
		Title:  def.Title,
		Weight: input.Weight,
	}, nil
}

func (s *CalendarService) buildSelectedPages(def domain.CalendarMetricDefinition, days []string) ([]string, error) {
	tf := def.TimeMemberFormat
	if strings.TrimSpace(tf) == "" {
		tf = "[Time].[%s]"
	}
	selected := make([]string, 0, len(days))
	for _, day := range days {
		// def.TimeMemberFormat must be something like "[Time].[%s]"
		selected = append(selected, fmt.Sprintf(tf, day))
	}
	// Keep deterministic ordering to reduce diffs and caching misses.
	sort.Strings(selected)
	return selected, nil
}

func metricKeys(defs []domain.CalendarMetricDefinition) []string {
	out := make([]string, 0, len(defs))
	for _, d := range defs {
		out = append(out, d.Key)
	}
	return out
}

func mockMetricValue(metricKey string, i int) float64 {
	seed := 0
	for _, r := range metricKey {
		seed += int(r)
	}
	// deterministic mock: value grows with i
	v := float64((seed%7)+1) * float64(i+1)
	return math.Round(v*100) / 100
}

func mockMetricBoolValue(metricKey string, i int) bool {
	seed := 0
	for _, r := range metricKey {
		seed += int(r)
	}
	// deterministic mock: alternates by day index parity
	return (seed+i)%2 == 0
}

func fillMockMetricValues(def domain.CalendarMetricDefinition, days []string, values map[string]*domain.CalendarMetricCellValue) {
	for i, day := range days {
		if def.TargetValue.Bool != nil {
			v := mockMetricBoolValue(def.Key, i)
			values[day] = &domain.CalendarMetricCellValue{Bool: &v}
		} else {
			v := mockMetricValue(def.Key, i)
			values[day] = &domain.CalendarMetricCellValue{Number: &v}
		}
	}
}

// LoadCalendarMetricsFromEnv parses CALENDAR_METRICS_JSON from env.
func LoadCalendarMetricsFromEnv() ([]domain.CalendarMetricDefinition, error) {
	raw := os.Getenv("CALENDAR_METRICS_JSON")
	if strings.TrimSpace(raw) == "" {
		// default placeholders to make UI functional even without integration credentials.
		return []domain.CalendarMetricDefinition{
			{
				Key:              "metric-1",
				Title:            "Metric 1",
				DefaultWeight:    1,
				MetricType:       domain.MetricTypeNeutral,
				TargetValue:      domain.TargetValue{Number: ptrFloat64(10)},
				TargetOperator:   domain.TargetOperatorEq,
				EazyBIReportID:   0,
				EazyBIFormat:     "csv",
				TimeMemberFormat: "[Time].[%s]",
			},
			{
				Key:              "metric-2",
				Title:            "Metric 2",
				DefaultWeight:    1,
				MetricType:       domain.MetricTypePositive,
				TargetValue:      domain.TargetValue{Number: ptrFloat64(20)},
				TargetOperator:   domain.TargetOperatorEq,
				EazyBIReportID:   0,
				EazyBIFormat:     "csv",
				TimeMemberFormat: "[Time].[%s]",
			},
			{
				// Negative metric example (lower is better).
				Key:              "defectsPerSprint",
				Title:            "Defects / sprint",
				DefaultWeight:    1,
				MetricType:       domain.MetricTypeNegative,
				TargetValue:      domain.TargetValue{Number: ptrFloat64(5)},
				TargetOperator:   domain.TargetOperatorLt,
				EazyBIReportID:   0,
				EazyBIFormat:     "csv",
				TimeMemberFormat: "[Time].[%s]",
			},
		}, nil
	}

	var defs []domain.CalendarMetricDefinition
	if err := json.Unmarshal([]byte(raw), &defs); err != nil {
		return nil, err
	}

	// basic normalization
	for i := range defs {
		if strings.TrimSpace(defs[i].Key) == "" || strings.TrimSpace(defs[i].Title) == "" {
			return nil, ValidationError{Message: "CALENDAR_METRICS_JSON has empty key/title"}
		}
		defs[i].MetricType = normalizeMetricType(defs[i].MetricType)
		if defs[i].TargetOperator == "" {
			defs[i].TargetOperator = domain.TargetOperatorEq
		}
		defs[i].EazyBIFormat = strings.ToLower(strings.TrimSpace(defs[i].EazyBIFormat))
		if defs[i].EazyBIFormat == "" {
			defs[i].EazyBIFormat = "csv"
		}
		if defs[i].TimeMemberFormat == "" {
			defs[i].TimeMemberFormat = "[Time].[%s]"
		}
		if defs[i].DefaultWeight <= 0 {
			defs[i].DefaultWeight = 1
		}
		if defs[i].TargetValue.Number == nil && defs[i].TargetValue.Bool == nil {
			// default target to 0 for number metrics and false for boolean metrics
			// If caller didn't provide a targetValue, keep it neutral at 0.
			z := 0.0
			defs[i].TargetValue = domain.TargetValue{Number: &z}
		}
		if defs[i].EazyBIReportID < 0 {
			return nil, ValidationError{Message: "eazybiReportId must be >= 0"}
		}
	}

	return defs, nil
}

func normalizeMetricType(mt domain.MetricType) domain.MetricType {
	switch strings.ToLower(string(mt)) {
	case "positive":
		return domain.MetricTypePositive
	case "negative":
		return domain.MetricTypeNegative
	default:
		return domain.MetricTypeNeutral
	}
}

// BuildEazyBIClientFromEnv builds a client; returns nil if env not configured.
func BuildEazyBIClientFromEnv() (*eazybi.Client, error) {
	baseURL := strings.TrimSpace(os.Getenv("EAZYBI_BASE_URL"))
	accountID := strings.TrimSpace(os.Getenv("EAZYBI_ACCOUNT_ID"))
	exportPrefix := strings.TrimSpace(os.Getenv("EAZYBI_EXPORT_PREFIX"))
	authMode := strings.TrimSpace(os.Getenv("EAZYBI_AUTH_MODE"))
	allowedHostsRaw := strings.TrimSpace(os.Getenv("EAZYBI_ALLOWED_HOSTS"))

	if baseURL == "" || accountID == "" || exportPrefix == "" {
		return nil, nil
	}

	if authMode == "" {
		authMode = "basic"
	}

	allowedHosts := []string{}
	if allowedHostsRaw != "" {
		for _, h := range strings.Split(allowedHostsRaw, ",") {
			trim := strings.TrimSpace(h)
			if trim != "" {
				allowedHosts = append(allowedHosts, trim)
			}
		}
	}

	// SSRF policy requires explicit allowlist.
	if len(allowedHosts) == 0 {
		return nil, ValidationError{Message: "EAZYBI_ALLOWED_HOSTS is required (SSRf policy)"}
	}

	cfg := eazybi.Config{
		BaseURL:      baseURL,
		ExportPrefix: exportPrefix,
		AccountID:    accountID,
		AuthMode:     authMode,
		Username:     os.Getenv("EAZYBI_USERNAME"),
		Password:     os.Getenv("EAZYBI_PASSWORD"),
		EmbedToken:   os.Getenv("EAZYBI_EMBED_TOKEN"),
		JiraToken:    os.Getenv("EAZYBI_JIRA_TOKEN"),
		AllowedHosts: allowedHosts,
	}

	// If basic is selected, credentials must exist.
	if strings.ToLower(authMode) == "basic" {
		if strings.TrimSpace(cfg.Username) == "" || strings.TrimSpace(cfg.Password) == "" {
			return nil, ValidationError{Message: "EAZYBI_USERNAME/EAZYBI_PASSWORD are required for basic auth"}
		}
	}
	if strings.EqualFold(authMode, "jira_token") || strings.EqualFold(authMode, "token") || strings.EqualFold(authMode, "bearer") {
		if strings.TrimSpace(cfg.JiraToken) == "" {
			return nil, ValidationError{Message: "EAZYBI_JIRA_TOKEN is required for jira token auth"}
		}
	}

	return eazybi.NewClient(cfg), nil
}

// Helper for parse number from string, used nowhere yet but kept for future.
func parseFloatPtr(s *string) (*float64, error) {
	if s == nil {
		return nil, nil
	}
	v, err := strconv.ParseFloat(*s, 64)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func ptrFloat64(v float64) *float64 {
	return &v
}
